# Phase 5: Embedded Go Runtime (yaegi) + Custom Bridges + go.mod + exec Whitelist

**Date**: 14 July 2026
**Status**: Done
**Depends on**: Phase 1 (bridge API), Phase 4 (shared embedded runtime interface)
**Unlocks**: Phase 6C (engine enhancements), Phase 7 (module setting)
**Master doc**: `2026-07-14-runtime-engine-redesign-master.md`

---

## Table of Contents

1. [Goal](#1-goal)
2. [Why Go Scripts](#2-why-go-scripts)
3. [yaegi Overview](#3-yaegi-overview)
4. [Implementation — Reuse Phase 4 Interface](#4-implementation--reuse-phase-4-interface)
5. [Bridge Injection — `bitcode` as Go Package](#5-bridge-injection--bitcode-as-go-package)
6. [Custom Bridges (`bridges/` Folder)](#6-custom-bridges-bridges-folder)
7. [go.mod Per Module](#7-gomod-per-module)
8. [`bitcode.exec()` — os/exec Whitelist](#8-bitcodeexec--osexec-whitelist)
9. [Timeout — The Hard Problem](#9-timeout--the-hard-problem)
10. [Security — What's Blocked vs Allowed](#10-security--whats-blocked-vs-allowed)
11. [Edge Cases](#11-edge-cases)
12. [Implementation Tasks](#12-implementation-tasks)

---

## 1. Goal

Add embedded Go runtime via **yaegi** (Go interpreter by Traefik). Scripts written in Go, interpreted at runtime, with full goroutine support, custom bridges, and per-module `go.mod` dependencies.

### Prerequisites

- Phase 1 complete: `bridge.Context` interfaces
- Phase 4 complete: shared `EmbeddedRuntime` + `VM` interface (reused)

### 1.1 Success Criteria

- `runtime: "go"` executes Go scripts via yaegi
- Scripts can `import "bitcode"` to access all 20 bridge methods
- Scripts can use goroutines (`go func(){}()`) for true concurrency
- Custom bridges in `bridges/` folder extend `bitcode.*` namespace
- Per-module `go.mod` for external Go package dependencies
- `bitcode.Exec()` works with whitelist from `module.json`
- `os.Exit()`, `unsafe`, `syscall` blocked by default
- `os.ReadFile()`, `os.WriteFile()`, `net/http` allowed

### 1.2 What This Phase Does NOT Do

- Does not change JS runtimes (Phase 2-4)
- Does not change Python runtime (Phase 3)
- Does not build module setting (Phase 7)

---

## 2. Why Go Scripts

Go scripts fill a unique niche that JS and Python cannot:

| Capability | JS (goja/qjs) | Python | **Go (yaegi)** |
|-----------|---------------|--------|----------------|
| True concurrency | ❌ Single-threaded | ❌ GIL | ✅ **Goroutines** |
| Type safety | ❌ Dynamic | ❌ Dynamic | ✅ **Static types** |
| Go stdlib access | ❌ | ❌ | ✅ **Full stdlib** |
| Custom bridges | ❌ | ❌ | ✅ **Write in Go** |
| os/exec (whitelisted) | Via bridge only | Via bridge only | ✅ **Native + bridge** |
| Performance | Interpreted JS | Interpreted Python | **Interpreted Go** (faster than both) |
| Ecosystem | npm (child only) | pip (child only) | **Go modules** |

**Use cases unique to Go scripts:**
- Parallel HTTP crawling with goroutines + WaitGroup
- Custom bridge development (extend `bitcode.*`)
- Performance-critical batch processing
- System integration (file processing, network tools)
- Scripts that need Go packages (excelize, chromedp, etc.)

---

## 3. yaegi Overview

```
Engine:     traefik/yaegi
Language:   Pure Go (zero CGO)
Go support: Full Go spec (latest two major releases)
Sandbox:    unsafe + syscall blocked by default
Goroutines: ✅ Native support (real goroutines)
Stars:      7,000+ | Maintained by Traefik team | Production: Traefik plugins
```

### Key API

```go
i := interp.New(interp.Options{
    GoPath:       gopath,
    Unrestricted: false,  // sandbox: no unsafe, no syscall
})
i.Use(stdlib.Symbols)           // load Go stdlib
i.Use(customExports)            // load custom packages (bitcode, bridges)
v, err := i.Eval(code)          // interpret Go code
```

### What yaegi Can Do

- Interpret any valid Go code (functions, structs, interfaces, goroutines, channels, defer, etc.)
- Import Go stdlib packages (`fmt`, `os`, `net/http`, `sync`, `encoding/json`, etc.)
- Import custom packages exposed via `i.Use()`
- Import external packages from GOPATH/go.mod (via `yaegi extract`)
- Full goroutine support — `go func(){}()` creates real goroutines

### What yaegi Cannot Do

- No built-in timeout/interrupt mechanism
- Cannot import `unsafe` or `syscall` in sandbox mode
- Cannot import packages not pre-loaded via `i.Use()` or available in GOPATH
- Slightly slower than compiled Go (~5-10x for CPU-heavy code)

---

## 4. Implementation — Reuse Phase 4 Interface

yaegi implements the same `EmbeddedRuntime` + `VM` interface from Phase 4:

### 4.1 YaegiRuntime

```go
// engine/internal/runtime/embedded/yaegi/runtime.go
package yaegi_runtime

import (
    "github.com/traefik/yaegi/interp"
    "github.com/traefik/yaegi/stdlib"
)

type YaegiRuntime struct {
    bridgeSymbols interp.Exports  // bitcode package symbols
    customBridges interp.Exports  // from bridges/ folder
}

func New(bridgeSymbols, customBridges interp.Exports) *YaegiRuntime {
    return &YaegiRuntime{
        bridgeSymbols: bridgeSymbols,
        customBridges: customBridges,
    }
}

func (r *YaegiRuntime) Name() string { return "yaegi" }

func (r *YaegiRuntime) NewVM(opts embedded.VMOptions) (embedded.VM, error) {
    i := interp.New(interp.Options{
        Unrestricted: false,  // sandbox mode
    })
    
    // Load selective stdlib (not everything — see §10)
    if err := i.Use(stdlib.Symbols); err != nil {
        return nil, err
    }
    
    // Load bitcode package
    if err := i.Use(r.bridgeSymbols); err != nil {
        return nil, err
    }
    
    // Load custom bridges
    if r.customBridges != nil {
        if err := i.Use(r.customBridges); err != nil {
            return nil, err
        }
    }
    
    return &YaegiVM{interp: i, opts: opts}, nil
}
```

### 4.2 YaegiVM

```go
// engine/internal/runtime/embedded/yaegi/vm.go

type YaegiVM struct {
    interp    *interp.Interpreter
    opts      embedded.VMOptions
    cancelled atomic.Bool
}

func (v *YaegiVM) InjectBridge(bc *bridge.Context) error {
    // Bridge is injected as a Go package "bitcode"
    // Scripts do: import "bitcode"
    // Then: bitcode.Model("lead").Search(...)
    
    symbols := interp.Exports{
        "bitcode/bitcode": map[string]reflect.Value{
            "Model":     reflect.ValueOf(bc.Model),
            "DB":        reflect.ValueOf(bc.DB),
            "HTTP":      reflect.ValueOf(bc.HTTP),
            "Cache":     reflect.ValueOf(bc.Cache),
            "Env":       reflect.ValueOf(bc.Env),
            "Session":   reflect.ValueOf(bc.Session),
            "Config":    reflect.ValueOf(bc.Config),
            "Log":       reflect.ValueOf(bc.Log),
            "Emit":      reflect.ValueOf(bc.Emit),
            "Call":      reflect.ValueOf(bc.Call),
            "Exec":      reflect.ValueOf(bc.Exec),
            "T":         reflect.ValueOf(bc.T),
            "FS":        reflect.ValueOf(bc.FS),
            "Email":     reflect.ValueOf(bc.Email),
            "Notify":    reflect.ValueOf(bc.Notify),
            "Storage":   reflect.ValueOf(bc.Storage),
            "Security":  reflect.ValueOf(bc.Security),
            "Audit":     reflect.ValueOf(bc.Audit),
            "Crypto":    reflect.ValueOf(bc.Crypto),
            "Execution": reflect.ValueOf(bc.Execution),
            "Tx":        reflect.ValueOf(bc.Tx),
        },
    }
    return v.interp.Use(symbols)
}

func (v *YaegiVM) InjectParams(params map[string]any) error {
    symbols := interp.Exports{
        "params/params": map[string]reflect.Value{
            "Input":     reflect.ValueOf(params["input"]),
            "Variables": reflect.ValueOf(params["variables"]),
            "UserID":    reflect.ValueOf(params["user_id"]),
        },
    }
    return v.interp.Use(symbols)
}

func (v *YaegiVM) Execute(code string, filename string) (any, error) {
    // Eval the script
    val, err := v.interp.Eval(code)
    if err != nil {
        return nil, &bridge.BridgeError{Code: "SCRIPT_ERROR", Message: err.Error()}
    }
    
    // Look for Execute function
    execVal, err := v.interp.Eval("Execute")
    if err == nil && execVal.IsValid() && execVal.Kind() == reflect.Func {
        // Call Execute(params)
        results := execVal.Call([]reflect.Value{
            reflect.ValueOf(params),
        })
        if len(results) == 2 && !results[1].IsNil() {
            return nil, results[1].Interface().(error)
        }
        if len(results) >= 1 {
            return results[0].Interface(), nil
        }
    }
    
    // No Execute function — return eval result
    if val.IsValid() {
        return val.Interface(), nil
    }
    return nil, nil
}

func (v *YaegiVM) Interrupt(reason string) {
    v.cancelled.Store(true)
    // yaegi has no built-in interrupt — see §9 for timeout strategy
}

func (v *YaegiVM) Close() {
    // yaegi interpreter doesn't need explicit cleanup
}
```

---

## 5. Bridge Injection — `bitcode` as Go Package

### 5.1 Script File Format

Go scripts **must** use `package main` and export an `Execute` function:

```go
// modules/crm/scripts/parallel_enrich.go
package main  // MUST be package main
```

**Why `package main`**:
- Most familiar to Go developers
- yaegi retrieves function via `i.Eval("main.Execute")`
- Consistent with `go run` convention

**Why not `package scripts`**: yaegi uses package name for symbol lookup. `package scripts` would require `i.Eval("scripts.Execute")` — works but less intuitive.

**Function signature** — two variants supported:

```go
// Recommended: with context (supports timeout cancellation)
func Execute(ctx context.Context, params map[string]any) (any, error)

// Simple: without context (no cooperative timeout)
func Execute(params map[string]any) (any, error)
```

Engine detects signature via reflection and calls accordingly.

### 5.2 How Scripts Use the Bridge

```go
// modules/crm/scripts/parallel_enrich.go
package main

import (
    "bitcode"   // bridge API — registered via i.Use(), not a real Go module
    "context"
    "sync"
)

func Execute(ctx context.Context, params map[string]any) (any, error) {
    leads, err := bitcode.Model("lead").Search(bridge.SearchOptions{
        Domain: [][]any{{"status", "=", "new"}},
        Limit:  100,
    })
    if err != nil {
        return nil, err
    }
    
    var wg sync.WaitGroup
    results := make([]map[string]any, len(leads))
    
    for i, lead := range leads {
        wg.Add(1)
        go func(idx int, l map[string]any) {
            defer wg.Done()
            // Concurrent HTTP calls — real goroutines!
            resp, _ := bitcode.HTTP().Get(
                "https://api.enrichment.com/company?domain=" + l["email"].(string),
                nil,
            )
            results[idx] = map[string]any{
                "lead_id": l["id"],
                "company": resp.Body,
            }
        }(i, lead)
    }
    
    wg.Wait()
    
    // Bulk update
    for _, r := range results {
        bitcode.Model("lead").Write(r["lead_id"].(string), map[string]any{
            "company_data": r["company"],
        })
    }
    
    bitcode.Log("info", "Enriched leads", map[string]any{"count": len(results)})
    return map[string]any{"enriched": len(results)}, nil
}
```

### 5.3 Goroutine Lifecycle — Important

**Scripts must wait for all goroutines before returning.** Use `sync.WaitGroup`:

```go
func Execute(ctx context.Context, params map[string]any) (any, error) {
    var wg sync.WaitGroup
    
    for _, item := range items {
        wg.Add(1)
        go func(it map[string]any) {
            defer wg.Done()
            // ... work
        }(item)
    }
    
    wg.Wait()  // MUST wait — don't return while goroutines are running
    return result, nil
}
```

**What happens if script returns before goroutines finish:**
1. Bridge context may be cleaned up → goroutine's bridge calls fail with stale context
2. Execution log records script as "completed" but work is still running
3. Goroutines become orphaned — no way to track or cancel them

**Engine mitigation**: After `Execute()` returns, engine waits a short grace period (1 second) for any pending bridge calls to complete, then invalidates the bridge context. Any bridge call after invalidation returns `BridgeError{Code: "CONTEXT_EXPIRED"}`.

```go
func (v *YaegiVM) Execute(code string, filename string) (any, error) {
    // ... call Execute function ...
    result, err := fn(ctx, params)
    
    // Grace period for orphaned goroutines
    time.Sleep(1 * time.Second)
    
    // Invalidate bridge context — orphaned goroutines will get errors
    v.bridgeCtx.Invalidate()
    
    return result, err
}
```

### 5.4 Naming Convention: PascalCase

Go convention is PascalCase for exported functions. Bridge API in Go scripts uses PascalCase:

```go
// Go script (PascalCase — Go convention)
bitcode.Model("lead").Search(opts)
bitcode.HTTP().Get(url, opts)
bitcode.Log("info", "message")
bitcode.Emit("event", data)

// vs JavaScript (camelCase — JS convention)
bitcode.model("lead").search(opts)
bitcode.http.get(url, opts)
bitcode.log("info", "message")
bitcode.emit("event", data)
```

Same bridge.Context underneath. Different casing per language convention.

### 5.5 Thread Safety of Bridge Calls

Go scripts can call bridge methods from goroutines. Bridge methods **must be thread-safe**:

```go
// This is valid — two goroutines calling bridge simultaneously
go func() { bitcode.Model("lead").Create(data1) }()
go func() { bitcode.Model("lead").Create(data2) }()
```

Bridge implementation (Phase 1) must use mutex or be inherently thread-safe:
- `ModelHandle.Create()` → GORM is goroutine-safe ✅
- `HTTPClient.Get()` → tls-client is goroutine-safe ✅
- `Cache.Set()` → Redis client is goroutine-safe ✅
- `Logger.Log()` → structured logger is goroutine-safe ✅
- `EventEmitter.Emit()` → needs mutex (append to slice) ⚠️

**Phase 1 bridge implementation must document thread-safety guarantees for each method.**

---

## 6. Custom Bridges (`bridges/` Folder)

### 6.1 Concept

Developers can extend `bitcode.*` with custom namespaces by writing Go files in `bridges/` folder:

```
# Project-level bridges (available to ALL modules)
project/
├── bridges/
│   ├── whatsapp.go       → bitcode.Whatsapp.Send(phone, msg)
│   ├── telegram.go       → bitcode.Telegram.Send(chatId, msg)
│   └── s3.go             → bitcode.S3.Upload(bucket, key, data)

# Module-level bridges (available ONLY to this module)
modules/crm/
├── bridges/
│   ├── salesforce.go     → bitcode.Salesforce.GetLeads()
│   └── hubspot.go        → bitcode.Hubspot.SyncContacts()
```

### 6.2 Bridge File Format

```go
// bridges/whatsapp.go
package bridges

import (
    "fmt"
    "net/http"
    "encoding/json"
)

// WhatsappBridge — exported struct becomes bitcode.Whatsapp
type WhatsappBridge struct {
    APIKey  string
    BaseURL string
}

func (w *WhatsappBridge) Send(phone string, message string) error {
    body := map[string]string{"phone": phone, "message": message}
    jsonBody, _ := json.Marshal(body)
    
    req, _ := http.NewRequest("POST", w.BaseURL+"/send", bytes.NewReader(jsonBody))
    req.Header.Set("Authorization", "Bearer "+w.APIKey)
    
    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != 200 {
        return fmt.Errorf("whatsapp API error: %d", resp.StatusCode)
    }
    return nil
}

func (w *WhatsappBridge) SendTemplate(phone string, template string, data map[string]any) error {
    // ...
    return nil
}
```

### 6.3 How Engine Loads Bridges

At startup, engine scans `bridges/` folders and loads them via yaegi:

```go
func loadBridges(projectDir string, modules []ModuleDefinition) (interp.Exports, error) {
    exports := interp.Exports{}
    
    // 1. Load project-level bridges
    projectBridges := filepath.Join(projectDir, "bridges")
    if dirExists(projectBridges) {
        symbols, err := extractBridgeSymbols(projectBridges)
        if err != nil { return nil, err }
        mergeExports(exports, symbols)
    }
    
    // 2. Load module-level bridges
    for _, mod := range modules {
        modBridges := filepath.Join(mod.Path, "bridges")
        if dirExists(modBridges) {
            symbols, err := extractBridgeSymbols(modBridges)
            if err != nil { return nil, err }
            // Namespace: bitcode_crm/bridges for module "crm"
            mergeExports(exports, symbols)
        }
    }
    
    return exports, nil
}
```

### 6.4 How Scripts Use Custom Bridges

```go
// modules/crm/scripts/notify_deal.go
package scripts

import (
    "bitcode"
    "bridges"  // project-level bridges
)

func Execute(params map[string]any) (any, error) {
    lead := params["input"].(map[string]any)
    
    // Use custom bridge
    wa := &bridges.WhatsappBridge{
        APIKey:  bitcode.Env("WHATSAPP_API_KEY"),
        BaseURL: "https://api.whatsapp.com/v1",
    }
    
    err := wa.Send(lead["phone"].(string), "Deal won: "+lead["name"].(string))
    if err != nil {
        bitcode.Log("error", "WhatsApp send failed", map[string]any{"error": err.Error()})
    }
    
    return map[string]any{"notified": true}, nil
}
```

### 6.5 Bridge Configuration

Bridges often need config (API keys, URLs). These come from `bitcode.Env()` or `bitcode.Config()`:

```go
// Bridge reads config at runtime, not at load time
func NewWhatsappBridge() *WhatsappBridge {
    return &WhatsappBridge{
        APIKey:  bitcode.Env("WHATSAPP_API_KEY"),
        BaseURL: bitcode.Config("whatsapp.base_url").(string),
    }
}
```

### 6.6 Edge Cases

| Edge Case | Behavior |
|-----------|----------|
| Bridge file has syntax error | Engine startup fails with clear error: "bridge 'whatsapp.go' syntax error: ..." |
| Bridge imports `unsafe` | Blocked — yaegi sandbox mode |
| Bridge imports external package not in go.mod | Error: "package not found" |
| Project bridge and module bridge have same name | Module bridge takes precedence (more specific) |
| Bridge file changed while engine running | Not hot-reloaded. Requires engine restart. (Bridges loaded at startup.) |

---

## 7. go.mod Per Module

### 7.1 Module Structure

```
modules/crm/
├── module.json
├── go.mod              ← Go dependencies for this module
├── go.sum
├── scripts/
│   └── parallel_enrich.go
└── bridges/
    └── hubspot.go
```

### 7.2 go.mod Example

```
module bitcode-module-crm

go 1.22

require (
    github.com/chromedp/chromedp v0.9.5
    github.com/xuri/excelize/v2 v2.8.1
)
```

### 7.3 How yaegi Loads External Packages

yaegi cannot directly import from `go.mod`. External packages must be **extracted** (pre-compiled to symbols) using `yaegi extract`:

```bash
# CLI command to extract Go packages for a module
bitcode module extract-deps crm

# What it does:
# 1. cd modules/crm
# 2. go mod download
# 3. yaegi extract github.com/chromedp/chromedp
# 4. yaegi extract github.com/xuri/excelize/v2
# 5. Generated files: modules/crm/_extracted/*.go
# 6. These are loaded via i.Use() at runtime
```

### 7.4 CLI Commands

```bash
# Extract Go dependencies for a module (generates yaegi symbols)
bitcode module extract-deps crm

# Extract all modules
bitcode module extract-deps --all

# Add a Go package to a module
bitcode module add-go-package crm github.com/xuri/excelize/v2
# → updates go.mod + runs extract

# Remove a Go package
bitcode module remove-go-package crm github.com/xuri/excelize/v2
```

### 7.5 Limitations

| Limitation | Why | Workaround |
|-----------|-----|------------|
| Not all Go packages work with yaegi | yaegi interprets Go — some packages use `unsafe`, CGO, or assembly | Check yaegi compatibility before adding |
| `yaegi extract` can be slow | Generates reflection wrappers for all exported symbols | Run once, cache result |
| Large packages generate large symbol files | excelize: ~500KB of generated code | Acceptable — loaded once at startup |
| CGO packages don't work | yaegi is pure Go interpreter | Use child process (Node.js/Python) or native Go plugin |

### 7.6 Edge Cases

| Edge Case | Behavior |
|-----------|----------|
| Module has `go.mod` but `extract-deps` not run | Script fails: "package not found". Clear error message with fix instruction. |
| Package incompatible with yaegi | `yaegi extract` fails with error. Document known incompatible packages. |
| Two modules need different versions of same package | Each has own `go.mod` + extracted symbols. No conflict. |
| `go.mod` references private repo | Needs `GOPRIVATE` + auth. Same as normal Go development. |

---

## 8. `bitcode.exec()` — os/exec Whitelist

### 8.1 How It Works in Go Scripts

```go
// modules/crm/scripts/generate_pdf.go
package scripts

import "bitcode"

func Execute(params map[string]any) (any, error) {
    input := params["input"].(map[string]any)
    
    // Convert markdown to PDF using pandoc
    result, err := bitcode.Exec("pandoc", []string{
        "input.md", "-o", "output.pdf",
    }, &bridge.ExecOptions{
        Cwd:     "/tmp",
        Timeout: 30000,
    })
    
    if err != nil {
        return nil, err
    }
    
    bitcode.Log("info", "PDF generated", map[string]any{
        "exitCode": result.ExitCode,
        "stdout":   result.Stdout,
    })
    
    return map[string]any{"pdf": "/tmp/output.pdf"}, nil
}
```

### 8.2 Why Not Direct `os/exec`?

Go scripts could technically `import "os/exec"` and run commands directly. But:

1. **No whitelist enforcement** — script could run `rm -rf /`
2. **No audit logging** — engine doesn't know what commands were run
3. **No timeout enforcement** — script could spawn process that runs forever

`bitcode.Exec()` goes through the bridge which enforces:
- `exec_allow` whitelist from `module.json`
- `exec_deny` blacklist (engine-level)
- Timeout
- Audit logging

### 8.3 Blocking Direct os/exec

By default, yaegi sandbox (`Unrestricted: false`) does NOT block `os/exec`. We need to selectively block it:

```go
// Instead of loading ALL stdlib:
i.Use(stdlib.Symbols)

// Load stdlib MINUS os/exec:
filteredSymbols := filterStdlib(stdlib.Symbols, []string{
    "os/exec",     // blocked — use bitcode.Exec() instead
    "unsafe",      // blocked by yaegi sandbox
    "syscall",     // blocked by yaegi sandbox
})
i.Use(filteredSymbols)
```

```go
func filterStdlib(symbols interp.Exports, blocked []string) interp.Exports {
    filtered := make(interp.Exports)
    blockedSet := make(map[string]bool)
    for _, b := range blocked {
        blockedSet[b+"/"+filepath.Base(b)] = true
    }
    for key, val := range symbols {
        if !blockedSet[key] {
            filtered[key] = val
        }
    }
    return filtered
}
```

Script that tries `import "os/exec"` gets: `error: package os/exec not available. Use bitcode.Exec() instead.`

---

## 9. Timeout — The Hard Problem

### 9.1 The Problem

yaegi has **no built-in interrupt mechanism**. Unlike:
- goja: `vm.Interrupt()` stops at next JS instruction
- qjs: `rt.Close()` kills WASM runtime
- Node.js/Python: kill process

yaegi interprets Go code as real Go execution. `for {}` becomes a real infinite loop in the engine's goroutine. There's no way to stop it from outside without killing the entire engine process.

### 9.2 Solution: Context-Based Timeout

**Scripts must accept `context.Context` and respect cancellation.** This is idiomatic Go — every Go developer knows this pattern.

```go
// Script signature with context
package scripts

import (
    "bitcode"
    "context"
)

func Execute(ctx context.Context, params map[string]any) (any, error) {
    leads, _ := bitcode.Model("lead").Search(bridge.SearchOptions{
        Domain: [][]any{{"status", "=", "new"}},
    })
    
    for _, lead := range leads {
        // Check context cancellation
        select {
        case <-ctx.Done():
            return nil, ctx.Err()  // timeout or cancelled
        default:
        }
        
        // Process lead...
        bitcode.Model("lead").Write(lead["id"].(string), map[string]any{"processed": true})
    }
    
    return map[string]any{"processed": len(leads)}, nil
}
```

### 9.3 Engine Injects Context with Timeout

```go
func (v *YaegiVM) Execute(code string, filename string) (any, error) {
    // Create context with timeout
    ctx, cancel := context.WithTimeout(context.Background(), v.opts.Timeout)
    defer cancel()
    
    // Eval script
    v.interp.Eval(code)
    
    // Get Execute function
    execVal, err := v.interp.Eval("Execute")
    if err != nil {
        return nil, err
    }
    
    // Call with context
    fn := execVal.Interface().(func(context.Context, map[string]any) (any, error))
    return fn(ctx, params)
}
```

### 9.4 What If Script Ignores Context?

If script doesn't check `ctx.Done()` and runs `for {}`:

```go
// BAD script — ignores context
func Execute(ctx context.Context, params map[string]any) (any, error) {
    for {
        // infinite loop, never checks ctx.Done()
    }
}
```

**This will hang.** The goroutine running this script will be stuck forever.

**Mitigation**: Run script in a separate goroutine with timeout:

```go
func (v *YaegiVM) Execute(code string, filename string) (any, error) {
    ctx, cancel := context.WithTimeout(context.Background(), v.opts.Timeout)
    defer cancel()
    
    type result struct {
        value any
        err   error
    }
    done := make(chan result, 1)
    
    go func() {
        defer func() {
            if r := recover(); r != nil {
                done <- result{nil, fmt.Errorf("panic: %v", r)}
            }
        }()
        // ... eval and call Execute(ctx, params)
        val, err := fn(ctx, params)
        done <- result{val, err}
    }()
    
    select {
    case res := <-done:
        return res.value, res.err
    case <-ctx.Done():
        // Timeout — goroutine is leaked but engine continues
        // Log warning about leaked goroutine
        return nil, &bridge.BridgeError{
            Code:    "STEP_TIMEOUT",
            Message: fmt.Sprintf("Go script timed out after %s. Script should check ctx.Done() for cooperative cancellation.", v.opts.Timeout),
        }
    }
}
```

**Honest assessment**: If script ignores context, the goroutine **leaks**. It continues running in background, consuming CPU. This is a known limitation of Go — you cannot kill a goroutine from outside.

**Mitigations**:
1. **Documentation**: Clearly document that Go scripts MUST check `ctx.Done()` in loops
2. **Linting**: Engine can scan script for `for {` or `for true {` without `ctx.Done()` and warn
3. **Bridge calls check context**: Every `bitcode.*` call internally checks context and returns error if cancelled. So even if script doesn't check, bridge calls will fail after timeout.
4. **Leaked goroutine counter**: Engine tracks leaked goroutines. If count exceeds threshold, log error and suggest restart.

### 9.5 Bridge Calls Auto-Check Context

This is the safety net. Even if script doesn't check `ctx.Done()`, bridge calls do:

```go
// In bridge implementation:
func (m *modelHandle) Search(opts SearchOptions) ([]map[string]any, error) {
    // Check context before executing
    if err := m.ctx.Err(); err != nil {
        return nil, &BridgeError{Code: "STEP_TIMEOUT", Message: "execution cancelled"}
    }
    // ... execute query
}
```

So a script like:
```go
for {
    bitcode.Model("lead").Search(opts)  // this will fail after timeout
}
```

Will eventually stop because `Search()` returns error after context cancellation. The loop continues but every bridge call fails immediately.

**Only pure CPU loops without bridge calls are truly unstoppable:**
```go
for { x++ }  // this is the only case that truly leaks
```

---

## 10. Security — What's Blocked vs Allowed

### 10.1 Default Security

```
BLOCKED (by yaegi sandbox + engine filtering):
  ❌ unsafe.*                    — memory corruption risk
  ❌ syscall.*                   — direct kernel calls
  ❌ os/exec.*                   — use bitcode.Exec() instead (whitelisted)
  ❌ os.Exit()                   — would kill engine process
  ❌ plugin.*                    — native Go plugins, security risk

ALLOWED (useful for scripts):
  ✅ os.ReadFile, os.WriteFile   — file I/O (also available via bitcode.FS)
  ✅ os.MkdirAll, os.Remove      — file management
  ✅ os.Getenv                    — env vars (also via bitcode.Env with whitelist)
  ✅ net/http                     — HTTP client (also via bitcode.HTTP with tls-client)
  ✅ encoding/json, encoding/csv  — data formats
  ✅ fmt, strings, strconv        — string manipulation
  ✅ sync, sync/atomic            — concurrency primitives
  ✅ time                         — time operations
  ✅ math, math/rand              — math operations
  ✅ sort, slices                  — collection operations
  ✅ regexp                        — regex
  ✅ crypto/sha256, crypto/md5     — hashing (also via bitcode.Crypto)
  ✅ context                       — cancellation (required for timeout)
  ✅ errors                        — error handling
  ✅ io, bufio, bytes              — I/O utilities
  ✅ path, path/filepath           — path manipulation
  ✅ log                           — logging (also via bitcode.Log)
```

### 10.2 Why Allow Direct os/net/http When Bridge Has Them?

Developer choice:
- `bitcode.FS().Read("file.txt")` — sandboxed, respects `fs_allow`/`fs_deny`, audited
- `os.ReadFile("file.txt")` — direct, no sandbox, no audit

Both work. Bridge version is safer. Direct version is more flexible. Go developers expect stdlib access — blocking it would feel restrictive and un-Go-like.

**Recommendation**: Use `bitcode.*` for operations that need security/audit. Use stdlib for utility operations where security isn't a concern.

---

## 11. Edge Cases

### 11.1 Script Execution

| Edge Case | Behavior |
|-----------|----------|
| `.go` file without `runtime` field | Auto-detected as `runtime: "go"` (yaegi). See Phase 2 §10.10 for auto-detection rules. |
| Script missing `package main` | Error: "Go scripts must start with 'package main'. Found: 'package scripts'" |
| Script has syntax error | `i.Eval()` returns error. `BridgeError{Code: "SYNTAX_ERROR"}` |
| Script panics | Recovered by `defer recover()` in executor. `BridgeError{Code: "RUNTIME_PANIC"}` |
| Script spawns goroutine that panics | Goroutine crashes silently (Go default). Engine unaffected. |
| Script infinite loop (no ctx check) | Goroutine leaks. Bridge calls fail after timeout. Warning logged. |
| Script calls `os.Exit(0)` | Blocked — `os.Exit` not available (filtered from stdlib). |
| Script imports `os/exec` | Blocked — filtered from stdlib. Error: "use bitcode.Exec() instead" |

### 11.2 Goroutines

| Edge Case | Behavior |
|-----------|----------|
| Script spawns 1000 goroutines | All run as real goroutines. Go scheduler handles them. |
| Goroutine calls bridge method | Thread-safe — bridge methods use mutex where needed. |
| Script returns before goroutines finish | 1s grace period, then bridge context invalidated. Orphaned goroutines get `CONTEXT_EXPIRED` error. |
| Goroutine writes to shared variable | Race condition — same as normal Go. Developer responsibility. |
| Goroutine panics | Goroutine dies silently (Go default). Other goroutines + engine unaffected. |
| Goroutine spawns more goroutines | Works — no depth limit. But all must finish before script returns. |

### 11.3 Custom Bridges

| Edge Case | Behavior |
|-----------|----------|
| Bridge file has compile error | Engine startup fails with clear error. |
| Bridge imports blocked package | Error at startup: "package os/exec not available in bridge" |
| Bridge panics at runtime | Recovered by script's defer/recover or executor's panic recovery. |
| Two bridges export same symbol | Last loaded wins (module overrides project). Log warning. |

### 11.4 go.mod Dependencies

| Edge Case | Behavior |
|-----------|----------|
| Package not extracted | Script fails: "package not found". Error suggests running `bitcode module extract-deps`. |
| Package incompatible with yaegi | `yaegi extract` fails. Document known incompatible packages. |
| Package uses CGO | `yaegi extract` fails. Use child process runtime instead. |
| Package uses `unsafe` | `yaegi extract` may succeed but runtime fails. Document limitation. |

### 11.5 Memory & Performance

| Edge Case | Behavior |
|-----------|----------|
| `max_memory_mb` for yaegi | NOT enforceable (same as goja — Go heap). Document limitation. |
| Script allocates 2GB | Go process grows. No per-VM limit. |
| yaegi performance vs compiled Go | ~5-10x slower for CPU-heavy code. Acceptable for scripting. |
| Many concurrent Go scripts | Each in own goroutine. Go scheduler handles. No process pool needed. |

---

## 12. Implementation Tasks

### Files to Create

```
engine/internal/runtime/embedded/yaegi/
├── runtime.go              # YaegiRuntime implements EmbeddedRuntime
├── vm.go                   # YaegiVM implements VM
├── stdlib_filter.go        # Filter stdlib: remove os/exec, unsafe, syscall
├── bridge_loader.go        # Load bridges/ folder into yaegi symbols

engine/cmd/bitcode/
├── extract_deps.go         # CLI: bitcode module extract-deps
```

### Files to Modify

```
engine/internal/runtime/embedded/registry.go
  → Register "yaegi" engine

engine/internal/app.go
  → Load bridges/ at startup
  → Pass bridge symbols to YaegiRuntime

engine/internal/compiler/parser/module.go
  → Parse go.mod presence for modules

engine/go.mod
  → Add: github.com/traefik/yaegi
```

### Task Breakdown

| # | Task | Effort | Priority |
|---|------|--------|----------|
| **Core** | | | |
| 1 | Create `yaegi/runtime.go` — YaegiRuntime with stdlib + bridge loading | Medium | Must |
| 2 | Create `yaegi/vm.go` — YaegiVM with InjectBridge, Execute, context-based timeout | Large | Must |
| 3 | Create `yaegi/stdlib_filter.go` — filter os/exec, unsafe, syscall from stdlib | Small | Must |
| 4 | Register "yaegi" in embedded registry | Small | Must |
| **Bridge Injection** | | | |
| 5 | Implement `bitcode` package as `interp.Exports` (all 20 namespaces) | Large | Must |
| 6 | Implement `params` package injection | Small | Must |
| 7 | Ensure all bridge methods are thread-safe (for goroutine access) | Medium | Must |
| **Custom Bridges** | | | |
| 8 | Create `yaegi/bridge_loader.go` — scan and load `bridges/` folders | Medium | Must |
| 9 | Load project-level bridges at startup | Small | Must |
| 10 | Load module-level bridges per module | Small | Must |
| 11 | Handle bridge naming conflicts (module overrides project) | Small | Must |
| **go.mod** | | | |
| 12 | Implement `bitcode module extract-deps` CLI command | Large | Must |
| 13 | Integrate extracted symbols into yaegi at startup | Medium | Must |
| 14 | Handle `go.mod` download + `yaegi extract` pipeline | Medium | Must |
| **os/exec Whitelist** | | | |
| 15 | Verify `bitcode.Exec()` works through yaegi bridge (already in Phase 1) | Small | Must |
| 16 | Verify os/exec is blocked in filtered stdlib | Small | Must |
| **Integration** | | | |
| 17 | Wire YaegiRuntime in app.go with bridge symbols | Small | Must |
| 18 | Add yaegi to go.mod | Small | Must |
| **Tests** | | | |
| 19 | Tests: basic Go script execution via yaegi | Medium | Must |
| 20 | Tests: all 20 bridge namespaces from Go script | Large | Must |
| 21 | Tests: goroutine support (WaitGroup, channels, concurrent bridge calls) | Medium | Must |
| 22 | Tests: context-based timeout (cooperative cancellation) | Medium | Must |
| 23 | Tests: timeout with non-cooperative script (goroutine leak detection) | Medium | Must |
| 24 | Tests: custom bridges loading (project + module level) | Medium | Must |
| 25 | Tests: go.mod extract + import external package | Medium | Must |
| 26 | Tests: stdlib filtering (os/exec blocked, os.ReadFile allowed) | Small | Must |
| 27 | Tests: security (unsafe, syscall, os.Exit blocked) | Small | Must |
| 28 | Tests: bridge thread-safety (concurrent goroutine bridge calls) | Medium | Must |
| 29 | Tests: script signature detection (Execute with context vs without) | Small | Must |
| 30 | Tests: panic recovery (script panic, goroutine panic) | Small | Must |
| 31 | Tests: bridge naming conflicts | Small | Should |
| 32 | Tests: yaegi extract failure (incompatible package) | Small | Should |
