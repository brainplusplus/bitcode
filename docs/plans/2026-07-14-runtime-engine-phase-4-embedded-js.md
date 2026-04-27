# Phase 4: Embedded JavaScript Runtimes (goja + QuickJS)

**Date**: 14 July 2026
**Status**: Draft
**Depends on**: Phase 1 (bridge API interfaces)
**Unlocks**: Phase 6 (engine enhancements), Phase 7 (module setting)
**Master doc**: `2026-07-14-runtime-engine-redesign-master.md`

---

## Table of Contents

1. [Goal](#1-goal)
2. [Why Two Embedded JS Engines](#2-why-two-embedded-js-engines)
3. [Shared Architecture](#3-shared-architecture)
4. [goja Implementation](#4-goja-implementation)
5. [QuickJS Implementation](#5-quickjs-implementation)
6. [Bridge Proxy Patterns](#6-bridge-proxy-patterns)
7. [Timeout & Interrupt](#7-timeout--interrupt)
8. [Performance Optimization](#8-performance-optimization)
9. [Runtime Selection & Resolution](#9-runtime-selection--resolution)
10. [Important Limitations & Differences from Child Process](#10-important-limitations--differences-from-child-process)
11. [Edge Cases](#11-edge-cases)
12. [Implementation Tasks](#12-implementation-tasks)

---

## 1. Goal

Add two embedded JavaScript runtimes to the engine — **goja** (lightweight, ES6+) and **QuickJS** via fastschema/qjs (ES2023, async/await). Both run in-process, zero external dependency, single binary.

### Prerequisites

- Phase 1 complete: `bridge.Context` interfaces implemented
- No external dependencies: goja is pure Go, qjs is pure Go (via Wazero WASM)

### 1.1 Success Criteria

- `runtime: "javascript"` executes scripts via default engine (goja)
- `runtime: "javascript:goja"` explicitly uses goja
- `runtime: "javascript:quickjs"` explicitly uses QuickJS
- All 20 `bitcode.*` bridge methods work (real, not stub)
- Bridge calls are **synchronous** (direct Go function call, no IPC)
- Timeout/interrupt works for infinite loops
- 80% code shared between goja and qjs via interface
- Scripts hot-reload (re-read file every execution)

### 1.2 What This Phase Does NOT Do

- Does not add yaegi/Go runtime (Phase 5)
- Does not add `bridges/` folder support (Phase 5)
- Does not change child process runtimes (Phase 2-3)

### 1.3 Key Difference from Phase 2-3

| Aspect | Phase 2-3 (child process) | Phase 4 (embedded) |
|--------|--------------------------|-------------------|
| Communication | Bidirectional JSON-RPC over stdin/stdout | Direct Go function call |
| Process model | Separate OS process, process pool | In-process, VM instance per execution |
| Bridge call latency | ~0.1-0.5ms (IPC) | ~0.001ms (function call) |
| npm/pip packages | ✅ Full ecosystem | ❌ Not available |
| Memory isolation | ✅ Separate process | ⚠️ Shares Go process memory |
| Crash isolation | ✅ Process crash doesn't affect engine | ⚠️ Panic could affect engine (must recover) |
| Timeout | Go-side context + vm.timeout/process kill | `vm.Interrupt()` / WASM interrupt |
| Async/await | ✅ Native (Node.js event loop) | goja: ❌ / qjs: ✅ (but no true concurrency) |

---

## 2. Why Two Embedded JS Engines

### goja — The Lightweight Choice

```
Engine:     dop251/goja
Language:   Pure Go (zero CGO)
ES support: ES6+ (let, const, arrow, class, destructuring, Promises, Symbol, Map, Set)
Missing:    async/await, ES modules (import/export), generators, BigInt
Stars:      6,776 | Last commit: 9 days ago | Production: PocketBase, Grafana k6, Nakama
```

**Best for**: Hot-path scripts where every microsecond matters. Validation, onchange, computed fields. Go interop is excellent — auto struct mapping via reflection.

### QuickJS (fastschema/qjs) — The Modern Choice

```
Engine:     fastschema/qjs (QuickJS-NG via Wazero WASM)
Language:   Pure Go (zero CGO, QuickJS compiled to WASM)
ES support: ES2023 (async/await, modules, generators, Proxy, BigInt)
Missing:    Nothing significant — full ES2023
Stars:      New (2025) | Engine: QuickJS by Fabrice Bellard (creator of FFmpeg, QEMU)
```

**Best for**: Complex business logic where developer wants modern JS syntax. async/await makes code more readable even though bridge calls are synchronous.

### When to Use Which

| Use Case | Engine | Why |
|----------|--------|-----|
| Onchange handler (< 10 lines) | goja | Fastest startup, lowest overhead |
| Field validation | goja | Simple logic, no async needed |
| Computed fields | goja | Hot-path, called frequently |
| Business workflow (50+ lines) | quickjs | Modern syntax, more readable |
| Complex data transformation | quickjs | Destructuring, spread, generators |
| Script using async/await syntax | quickjs | goja doesn't support it |
| Don't care, just works | goja (default) | Lighter, proven, sufficient for most |

---

## 3. Shared Architecture

### 3.1 Interface (80% shared code)

```go
// engine/internal/runtime/embedded/runtime.go

// EmbeddedRuntime — interface for all embedded JS/Go engines
type EmbeddedRuntime interface {
    Name() string                              // "goja", "quickjs", "yaegi"
    NewVM(opts VMOptions) (VM, error)          // create VM instance
}

type VM interface {
    InjectBridge(bc *bridge.Context) error      // expose bitcode.* to VM
    InjectParams(params map[string]any) error   // expose params to VM
    Execute(code string, filename string) (any, error)  // run script
    Interrupt(reason string)                    // timeout/cancel
    Close()                                     // cleanup (free WASM memory, etc.)
}

type VMOptions struct {
    Timeout     time.Duration   // per-execution timeout
    MaxMemoryMB int             // memory limit (0 = unlimited)
}
```

### 3.2 Shared Executor (handles everything except VM-specific code)

```go
// engine/internal/runtime/embedded/executor.go

func ExecuteEmbedded(
    ctx context.Context,
    runtime EmbeddedRuntime,
    scriptPath string,
    params map[string]any,
    bridgeCtx *bridge.Context,
    timeout time.Duration,
) (any, error) {

    // 1. Load script — SHARED
    code, err := loadScript(scriptPath)
    if err != nil {
        return nil, &bridge.BridgeError{Code: "FS_NOT_FOUND", Message: "script not found: " + scriptPath}
    }

    // 2. Create VM — per-runtime via interface
    vm, err := runtime.NewVM(VMOptions{Timeout: timeout})
    if err != nil {
        return nil, &bridge.BridgeError{Code: "RUNTIME_ERROR", Message: "failed to create VM: " + err.Error()}
    }
    defer vm.Close()

    // 3. Inject bridge (bitcode.*) — per-runtime via interface
    if err := vm.InjectBridge(bridgeCtx); err != nil {
        return nil, err
    }

    // 4. Inject params — per-runtime via interface
    if err := vm.InjectParams(params); err != nil {
        return nil, err
    }

    // 5. Execute with timeout — SHARED pattern
    type result struct {
        value any
        err   error
    }
    done := make(chan result, 1)

    go func() {
        // Recover from panic (embedded VM could panic)
        defer func() {
            if r := recover(); r != nil {
                done <- result{nil, &bridge.BridgeError{
                    Code: "RUNTIME_PANIC", Message: fmt.Sprintf("VM panic: %v", r),
                }}
            }
        }()
        val, err := vm.Execute(code, scriptPath)
        done <- result{val, err}
    }()

    // Timeout interrupt
    var timer *time.Timer
    if timeout > 0 {
        timer = time.AfterFunc(timeout, func() {
            vm.Interrupt("execution timeout after " + timeout.String())
        })
        defer timer.Stop()
    }

    select {
    case res := <-done:
        return res.value, res.err
    case <-ctx.Done():
        vm.Interrupt("context cancelled")
        return nil, ctx.Err()
    }
}
```

### 3.3 Script Loader (shared)

```go
// engine/internal/runtime/embedded/script_loader.go

// scriptCache — compiled scripts for reuse (goja.Program, etc.)
// Key: scriptPath, Value: compiled program (runtime-specific)
var scriptCache sync.Map

func loadScript(scriptPath string) (string, error) {
    // Always re-read file (hot reload)
    data, err := os.ReadFile(scriptPath)
    if err != nil {
        return "", err
    }
    return string(data), nil
}
```

### 3.4 File Structure

```
engine/internal/runtime/embedded/
├── runtime.go              # SHARED: EmbeddedRuntime + VM interfaces
├── executor.go             # SHARED: ExecuteEmbedded() — load, inject, timeout, panic recovery
├── script_loader.go        # SHARED: file loading, hot reload
├── registry.go             # SHARED: register engines, resolve "javascript" → engine
│
├── goja/
│   ├── runtime.go          # GojaRuntime implements EmbeddedRuntime
│   ├── vm.go               # GojaVM implements VM
│   └── proxy.go            # Bridge proxy: bridge.Context → goja values
│
└── qjs/
    ├── runtime.go          # QJSRuntime implements EmbeddedRuntime
    ├── vm.go               # QJSVM implements VM
    └── proxy.go            # Bridge proxy: bridge.Context → qjs values
```

---

## 4. goja Implementation

### 4.1 GojaRuntime

```go
// engine/internal/runtime/embedded/goja/runtime.go
package goja_runtime

import (
    "github.com/dop251/goja"
    "github.com/dop251/goja_nodejs/require"
    "github.com/dop251/goja_nodejs/console"
)

type GojaRuntime struct {
    registry *require.Registry  // shared across all VMs (module caching)
}

func New() *GojaRuntime {
    reg := require.NewRegistry(
        require.WithLoader(func(path string) ([]byte, error) {
            return os.ReadFile(path)
        }),
    )
    return &GojaRuntime{registry: reg}
}

func (r *GojaRuntime) Name() string { return "goja" }

func (r *GojaRuntime) NewVM(opts VMOptions) (embedded.VM, error) {
    rt := goja.New()
    r.registry.Enable(rt)
    
    // Custom console that routes to bridge logger
    // (will be connected to bridge.Context in InjectBridge)
    
    return &GojaVM{rt: rt, opts: opts}, nil
}
```

### 4.2 GojaVM

```go
// engine/internal/runtime/embedded/goja/vm.go

type GojaVM struct {
    rt   *goja.Runtime
    opts VMOptions
}

func (v *GojaVM) InjectBridge(bc *bridge.Context) error {
    // bitcode.model(name) → returns proxy object with search/get/create/etc
    v.rt.Set("bitcode", map[string]any{
        "model": func(name string) any {
            return createModelProxy(v.rt, bc, name, false)
        },
        "db": map[string]any{
            "query":   func(sql string, args ...any) (any, error) { return bc.DB().Query(sql, args...) },
            "execute": func(sql string, args ...any) (any, error) { return bc.DB().Execute(sql, args...) },
        },
        "http": map[string]any{
            "get":    func(url string, opts map[string]any) (any, error) { return bc.HTTP().Get(url, parseHTTPOpts(opts)) },
            "post":   func(url string, opts map[string]any) (any, error) { return bc.HTTP().Post(url, parseHTTPOpts(opts)) },
            "put":    func(url string, opts map[string]any) (any, error) { return bc.HTTP().Put(url, parseHTTPOpts(opts)) },
            "patch":  func(url string, opts map[string]any) (any, error) { return bc.HTTP().Patch(url, parseHTTPOpts(opts)) },
            "delete": func(url string, opts map[string]any) (any, error) { return bc.HTTP().Delete(url, parseHTTPOpts(opts)) },
        },
        "cache": map[string]any{
            "get": func(key string) (any, error) { return bc.Cache().Get(key) },
            "set": func(key string, val any, opts ...map[string]any) error { return bc.Cache().Set(key, val, parseCacheOpts(opts)) },
            "del": func(key string) error { return bc.Cache().Del(key) },
        },
        "env":    func(key string) string { return bc.Env(key) },
        "config": func(key string) any { return bc.Config(key) },
        "log":    func(level, msg string, data ...map[string]any) { bc.Log(level, msg, data...) },
        "emit":   func(event string, data map[string]any) error { return bc.Emit(event, data) },
        "call":   func(process string, input map[string]any) (any, error) { return bc.Call(process, input) },
        "t":      func(key string) string { return bc.T(key) },
        "session": bc.Session(),
        "fs": map[string]any{
            "read":   func(p string) (string, error) { return bc.FS().Read(p) },
            "write":  func(p, content string) error { return bc.FS().Write(p, content) },
            "exists": func(p string) (bool, error) { return bc.FS().Exists(p) },
            "list":   func(p string) ([]string, error) { return bc.FS().List(p) },
            "mkdir":  func(p string) error { return bc.FS().Mkdir(p) },
            "remove": func(p string) error { return bc.FS().Remove(p) },
        },
        "exec": func(cmd string, args []string, opts ...map[string]any) (any, error) {
            return bc.Exec(cmd, args, parseExecOpts(opts))
        },
        "email":     createEmailProxy(bc),
        "notify":    createNotifyProxy(bc),
        "storage":   createStorageProxy(bc),
        "security":  createSecurityProxy(bc),
        "audit":     createAuditProxy(bc),
        "crypto":    createCryptoProxy(bc),
        "execution": createExecutionProxy(bc),
        "tx": func(fn goja.Callable) (any, error) {
            return bc.Tx(func(txCtx *bridge.Context) error {
                // Re-inject bridge with tx context
                // Call fn with tx-scoped bitcode
                _, err := fn(goja.Undefined())
                return err
            })
        },
    })
    
    // Custom console → bridge logger
    v.rt.Set("console", map[string]any{
        "log":   func(args ...any) { bc.Log("info", fmt.Sprint(args...)) },
        "warn":  func(args ...any) { bc.Log("warn", fmt.Sprint(args...)) },
        "error": func(args ...any) { bc.Log("error", fmt.Sprint(args...)) },
        "debug": func(args ...any) { bc.Log("debug", fmt.Sprint(args...)) },
    })
    
    return nil
}

func (v *GojaVM) InjectParams(params map[string]any) error {
    v.rt.Set("params", params)
    return nil
}

func (v *GojaVM) Execute(code string, filename string) (any, error) {
    // Try to get compiled program from cache
    program, err := goja.Compile(filename, code, true)
    if err != nil {
        return nil, &bridge.BridgeError{Code: "SYNTAX_ERROR", Message: err.Error()}
    }
    
    val, err := v.rt.RunProgram(program)
    if err != nil {
        // Check if it's an interrupt
        if interrupted, ok := err.(*goja.InterruptedError); ok {
            return nil, &bridge.BridgeError{Code: "STEP_TIMEOUT", Message: interrupted.Value().(string)}
        }
        return nil, &bridge.BridgeError{Code: "SCRIPT_ERROR", Message: err.Error()}
    }
    
    if val == nil || val == goja.Undefined() || val == goja.Null() {
        return nil, nil
    }
    
    // Check if result is a module with execute function
    exported := val.Export()
    if m, ok := exported.(map[string]any); ok {
        if execFn, ok := m["execute"]; ok {
            if callable, ok := goja.AssertFunction(v.rt.ToValue(execFn)); ok {
                bitcode := v.rt.Get("bitcode")
                params := v.rt.Get("params")
                result, err := callable(goja.Undefined(), bitcode, params)
                if err != nil {
                    if interrupted, ok := err.(*goja.InterruptedError); ok {
                        return nil, &bridge.BridgeError{Code: "STEP_TIMEOUT", Message: interrupted.Value().(string)}
                    }
                    return nil, &bridge.BridgeError{Code: "SCRIPT_ERROR", Message: err.Error()}
                }
                if result == nil || result == goja.Undefined() {
                    return nil, nil
                }
                return result.Export(), nil
            }
        }
    }
    
    return exported, nil
}

func (v *GojaVM) Interrupt(reason string) {
    v.rt.Interrupt(reason)
}

func (v *GojaVM) Close() {
    v.rt.ClearInterrupt()
}
```

### 4.3 Key goja Characteristics

- **Bridge calls are synchronous Go function calls** — no IPC, no serialization. `bitcode.model("lead").search({})` directly calls `bc.Model("lead").Search(opts)` in the same goroutine.
- **Auto type conversion** — goja automatically converts Go maps/slices/structs to JS objects and vice versa. No manual marshaling.
- **`goja.Compile()` + `RunProgram()`** — pre-compile script to bytecode. Can be cached for repeated execution.
- **`vm.Interrupt()`** — called from timeout goroutine. Interrupts execution at next JS instruction boundary. Script gets `InterruptedError`.
- **Not goroutine-safe** — one VM per goroutine. But we create a new VM per execution, so no sharing.

---

## 5. QuickJS Implementation

### 5.1 QJSRuntime

```go
// engine/internal/runtime/embedded/qjs/runtime.go
package qjs_runtime

import "github.com/fastschema/qjs"

type QJSRuntime struct{}

func New() *QJSRuntime {
    return &QJSRuntime{}
}

func (r *QJSRuntime) Name() string { return "quickjs" }

func (r *QJSRuntime) NewVM(opts VMOptions) (embedded.VM, error) {
    rt, err := qjs.NewRuntime()
    if err != nil {
        return nil, err
    }
    ctx, err := rt.NewContext()
    if err != nil {
        rt.Close()
        return nil, err
    }
    return &QJSVM{rt: rt, ctx: ctx, opts: opts}, nil
}
```

### 5.2 QJSVM

```go
// engine/internal/runtime/embedded/qjs/vm.go

type QJSVM struct {
    rt   *qjs.Runtime
    ctx  *qjs.Context
    opts VMOptions
}

func (v *QJSVM) InjectBridge(bc *bridge.Context) error {
    // bitcode.model(name) — using SetFunc for Go function binding
    v.ctx.SetFunc("__bitcode_model_search", func(this *qjs.This, args ...*qjs.Value) (*qjs.Value, error) {
        model := args[0].String()
        opts := qjs.JsValueToGo[map[string]any](args[1])
        result, err := bc.Model(model).Search(bridge.SearchOptions{/* parse opts */})
        if err != nil {
            return nil, err
        }
        return qjs.ToJSValue(this.Context(), result), nil
    })
    
    // ... similar for all 20 bridge namespaces
    
    // Inject JavaScript wrapper that creates the nice bitcode.model("lead").search({}) API
    v.ctx.Eval("__bitcode_init.js", qjs.Code(`
        const bitcode = {
            model: (name) => ({
                search: (opts) => __bitcode_model_search(name, opts || {}),
                get: (id, opts) => __bitcode_model_get(name, id, opts),
                create: (data) => __bitcode_model_create(name, data),
                write: (id, data) => __bitcode_model_write(name, id, data),
                delete: (id) => __bitcode_model_delete(name, id),
                count: (opts) => __bitcode_model_count(name, opts || {}),
                sum: (field, opts) => __bitcode_model_sum(name, field, opts || {}),
                upsert: (data, unique) => __bitcode_model_upsert(name, data, unique),
                createMany: (records) => __bitcode_model_createMany(name, records),
                writeMany: (ids, data) => __bitcode_model_writeMany(name, ids, data),
                deleteMany: (ids) => __bitcode_model_deleteMany(name, ids),
                upsertMany: (records, unique) => __bitcode_model_upsertMany(name, records, unique),
                addRelation: (id, field, ids) => __bitcode_model_addRelation(name, id, field, ids),
                removeRelation: (id, field, ids) => __bitcode_model_removeRelation(name, id, field, ids),
                setRelation: (id, field, ids) => __bitcode_model_setRelation(name, id, field, ids),
                loadRelation: (id, field) => __bitcode_model_loadRelation(name, id, field),
                sudo: () => ({
                    search: (opts) => __bitcode_model_search_sudo(name, opts || {}),
                    // ... all sudo methods
                    hardDelete: (id) => __bitcode_model_hardDelete(name, id),
                    withTenant: (tid) => ({ /* ... */ }),
                    skipValidation: () => ({ /* ... */ }),
                }),
            }),
            db: {
                query: (sql, ...args) => __bitcode_db_query(sql, args),
                execute: (sql, ...args) => __bitcode_db_execute(sql, args),
            },
            http: {
                get: (url, opts) => __bitcode_http_request('GET', url, opts),
                post: (url, opts) => __bitcode_http_request('POST', url, opts),
                put: (url, opts) => __bitcode_http_request('PUT', url, opts),
                patch: (url, opts) => __bitcode_http_request('PATCH', url, opts),
                delete: (url, opts) => __bitcode_http_request('DELETE', url, opts),
            },
            // ... all 20 namespaces
            env: (key) => __bitcode_env(key),
            config: (key) => __bitcode_config(key),
            log: (level, msg, data) => __bitcode_log(level, msg, data),
            emit: (event, data) => __bitcode_emit(event, data),
            call: (process, input) => __bitcode_call(process, input),
            t: (key) => __bitcode_t(key),
        };
    `))
    
    return nil
}

func (v *QJSVM) InjectParams(params map[string]any) error {
    paramsVal := qjs.ToJSValue(v.ctx, params)
    v.ctx.Globals().Set("params", paramsVal)
    return nil
}

func (v *QJSVM) Execute(code string, filename string) (any, error) {
    result, err := v.ctx.Eval(filename, qjs.Code(code))
    if err != nil {
        return nil, &bridge.BridgeError{Code: "SCRIPT_ERROR", Message: err.Error()}
    }
    defer result.Free()  // WASM memory management
    
    // Check if result is module with execute function
    if result.IsObject() {
        execProp := result.GetPropertyStr("execute")
        if execProp != nil && !execProp.IsUndefined() {
            defer execProp.Free()
            execFn, err := qjs.JsFuncToGoFunc[func(any, any) (any, error)](execProp)
            if err == nil {
                bitcode := v.ctx.Globals().GetPropertyStr("bitcode")
                params := v.ctx.Globals().GetPropertyStr("params")
                defer bitcode.Free()
                defer params.Free()
                return execFn(bitcode, params)
            }
        }
    }
    
    return qjs.JsValueToGo[any](result), nil
}

func (v *QJSVM) Interrupt(reason string) {
    v.rt.Close()  // Force close runtime — interrupts execution
}

func (v *QJSVM) Close() {
    v.ctx.Close()
    v.rt.Close()
}
```

### 5.3 Key QuickJS Characteristics

- **`ctx.SetFunc()` / `ctx.SetAsyncFunc()`** — expose Go functions. SetAsyncFunc returns Promise to JS.
- **Manual memory management** — `result.Free()`, `value.Free()`. WASM memory is not Go GC'd. Must defer Free() on every JS value.
- **`qjs.ToJSValue()` / `qjs.JsValueToGo[T]()`** — type conversion between Go and JS. Supports ProxyValue for zero-copy.
- **async/await works** — `ctx.SetAsyncFunc()` creates functions that return Promises. Script can `await` them.
- **Interrupt via `rt.Close()`** — less graceful than goja's `Interrupt()`. Kills the entire runtime.

### 5.4 async/await in QuickJS — Important Nuance

```javascript
// This works in QuickJS:
const leads = await bitcode.model("lead").search({});
const scores = await bitcode.http.post("https://api.com/score", { body: leads });

// But it's NOT truly concurrent — each await resolves immediately
// because bridge calls are synchronous Go function calls.
// The "await" is syntactic sugar, not parallelism.

// For true parallelism, use runtime: "node" (event loop) or runtime: "go" (goroutines)
```

If bridge functions are exposed via `ctx.SetFunc()` (synchronous), `await` on them resolves immediately. If exposed via `ctx.SetAsyncFunc()`, they return Promises but still execute synchronously in Go.

**Decision**: Use `ctx.SetFunc()` (synchronous) for all bridge methods. `await` is optional — works but doesn't add concurrency. This is simpler and matches goja behavior.

---

## 6. Bridge Proxy Patterns

### 6.1 goja — Direct Go Object Mapping

goja's strength is **automatic type conversion**. Go maps, slices, structs are directly usable in JS:

```go
// Go side:
vm.Set("bitcode", map[string]any{
    "model": func(name string) any { return createModelProxy(bc, name) },
})

// JS side — just works:
const leads = bitcode.model("lead").search({ domain: [["status", "=", "new"]] });
// Go receives: map[string]any{"domain": []any{[]any{"status", "=", "new"}}}
// Go returns: []map[string]any{{...}, {...}}
// JS receives: [{...}, {...}]
```

No serialization. No JSON.parse/stringify. Direct memory sharing via reflection.

### 6.2 QuickJS — JS Wrapper + Go Host Functions

QuickJS needs a JS wrapper because `ctx.SetFunc()` only registers flat functions, not nested objects:

```go
// Go side: register flat functions
ctx.SetFunc("__bitcode_model_search", func(...) { ... })
ctx.SetFunc("__bitcode_model_create", func(...) { ... })

// JS side: wrapper creates the nice API
// (injected via ctx.Eval() in InjectBridge)
const bitcode = {
    model: (name) => ({
        search: (opts) => __bitcode_model_search(name, opts),
        create: (data) => __bitcode_model_create(name, data),
    }),
};
```

This JS wrapper is ~200 lines and is **embedded as a Go string constant** — loaded once per VM creation.

### 6.3 Shared Bridge Proxy Helper

Both engines need to convert between `bridge.Context` methods and runtime-specific values. The conversion logic is shared:

```go
// engine/internal/runtime/embedded/bridge_helper.go

// parseSearchOpts — converts JS object to bridge.SearchOptions
// Used by both goja and qjs proxy code
func parseSearchOpts(raw map[string]any) bridge.SearchOptions {
    opts := bridge.SearchOptions{}
    if domain, ok := raw["domain"]; ok {
        opts.Domain = domain.([][]any)
    }
    if fields, ok := raw["fields"]; ok {
        opts.Fields = toStringSlice(fields)
    }
    if order, ok := raw["order"]; ok {
        opts.Order = order.(string)
    }
    if limit, ok := raw["limit"]; ok {
        opts.Limit = toInt(limit)
    }
    if offset, ok := raw["offset"]; ok {
        opts.Offset = toInt(offset)
    }
    if include, ok := raw["include"]; ok {
        opts.Include = toStringSlice(include)
    }
    return opts
}

// Similar helpers for parseHTTPOpts, parseCacheOpts, parseExecOpts, etc.
```

---

## 7. Timeout & Interrupt

### 7.1 goja — Graceful Interrupt

```go
// Timeout goroutine calls vm.Interrupt()
// Script receives InterruptedError at next JS instruction boundary
// VM is still usable after ClearInterrupt()

timer := time.AfterFunc(timeout, func() {
    vm.rt.Interrupt("timeout")
})
defer timer.Stop()

result, err := vm.rt.RunProgram(program)
if err != nil {
    if interrupted, ok := err.(*goja.InterruptedError); ok {
        return nil, &bridge.BridgeError{Code: "STEP_TIMEOUT", Message: "..."}
    }
}
```

**Graceful**: Script stops, error returned, VM can be reused (after ClearInterrupt). No resource leak.

### 7.2 QuickJS — Runtime Close

```go
// Timeout goroutine closes the runtime
// This is less graceful — runtime is destroyed

timer := time.AfterFunc(timeout, func() {
    vm.rt.Close()  // force close
})
defer timer.Stop()

result, err := vm.ctx.Eval(...)
// err will be non-nil if runtime was closed
```

**Less graceful**: Runtime is destroyed. Must create new VM for next execution. But since we create new VM per execution anyway, this is acceptable.

### 7.3 Unified Timeout (in shared executor)

Both engines use the same timeout pattern in `ExecuteEmbedded()` (§3.2). The `vm.Interrupt()` call is polymorphic — goja interrupts gracefully, qjs closes runtime. Same interface, different behavior.

---

## 8. Performance Optimization

### 8.1 goja — Script Compilation Cache

```go
// Compile script once, reuse across executions
var compiledScripts sync.Map  // path → *goja.Program

func getCompiledScript(path, code string) (*goja.Program, error) {
    if cached, ok := compiledScripts.Load(path); ok {
        return cached.(*goja.Program), nil
    }
    program, err := goja.Compile(path, code, true)
    if err != nil {
        return nil, err
    }
    compiledScripts.Store(path, program)
    return program, nil
}
```

**Why this works**: `goja.Program` is immutable bytecode. Safe to share across goroutines. Each VM runs its own copy of the state but shares the compiled bytecode.

**Cache invalidation**: On file change (hot reload), delete from cache. Simple timestamp check:

```go
func loadAndCompile(path string) (*goja.Program, string, error) {
    info, _ := os.Stat(path)
    cacheKey := path + ":" + info.ModTime().String()
    
    if cached, ok := compiledScripts.Load(cacheKey); ok {
        return cached.(*goja.Program), "", nil
    }
    // ... compile and cache
}
```

### 8.2 QuickJS — No Compilation Cache (Yet)

QuickJS via Wazero doesn't expose bytecode caching in the same way. Each `ctx.Eval()` compiles and runs. This is acceptable because:
- QuickJS compilation is fast (~1-5ms for typical scripts)
- The WASM runtime itself is cached by Wazero
- Can be optimized later if profiling shows it's a bottleneck

### 8.3 VM Instance Lifecycle

**Create per execution, not pooled.** Why:

- goja VM is lightweight (~1-2MB). Creation takes ~0.1ms.
- qjs VM is lightweight too (WASM context).
- No state leakage between executions (clean sandbox every time).
- No goroutine-safety concerns (each execution gets its own VM).
- Pooling adds complexity (reset state, clear globals) for minimal gain.

**Exception**: If profiling shows VM creation is a bottleneck (unlikely), can add `sync.Pool` later:

```go
var gojaPool = sync.Pool{
    New: func() any { return goja.New() },
}
```

But this is premature optimization. Start simple.

---

## 9. Runtime Selection & Resolution

### 9.1 Format

```
runtime: "javascript"              → default engine (from config)
runtime: "javascript:goja"         → explicit goja
runtime: "javascript:quickjs"      → explicit quickjs
```

### 9.2 Resolution Order

```
Step level:    step.runtime = "javascript:quickjs"  → quickjs
Process level: process.runtime = "javascript"       → resolve from module/project
Module level:  module.runtime_defaults.javascript = "quickjs"  → quickjs
Project level: runtime.javascript.default_engine = "goja"      → goja
Hardcoded:     "goja"
```

### 9.3 Engine Registry

```go
// engine/internal/runtime/embedded/registry.go

type EngineRegistry struct {
    engines map[string]EmbeddedRuntime  // "goja" → GojaRuntime, "quickjs" → QJSRuntime
    mu      sync.RWMutex
}

func NewRegistry() *EngineRegistry {
    r := &EngineRegistry{engines: make(map[string]EmbeddedRuntime)}
    r.Register("goja", goja_runtime.New())
    r.Register("quickjs", qjs_runtime.New())
    return r
}

func (r *EngineRegistry) Get(name string) (EmbeddedRuntime, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    engine, ok := r.engines[name]
    if !ok {
        return nil, &bridge.BridgeError{
            Code: "RUNTIME_NOT_FOUND",
            Message: fmt.Sprintf("embedded JS engine '%s' not found. Available: %v", name, r.Names()),
        }
    }
    return engine, nil
}

func (r *EngineRegistry) Resolve(runtimeField string, module ModuleDefinition, config RuntimeConfig) (EmbeddedRuntime, error) {
    engine := parseEngine(runtimeField)  // "javascript:quickjs" → "quickjs"
    if engine == "" {
        // Resolution: module → project → hardcoded
        engine = module.RuntimeDefaults["javascript"]
        if engine == "" { engine = config.JavaScript.DefaultEngine }
        if engine == "" { engine = "goja" }
    }
    return r.Get(engine)
}
```

---

## 10. Important Limitations & Differences from Child Process

### 10.1 Pool Config for Embedded Runtimes

Embedded runtimes (goja, qjs) do **not** use process pools — there's no persistent process to pool. A new VM is created per execution and destroyed after.

**But** the `pool` field in process JSON still matters — it determines **which timeout config** to use:

```json
{ "pool": "background", "steps": [{ "runtime": "javascript", "script": "heavy.js" }] }
```

→ No background process pool used (embedded, in-process)
→ BUT timeout comes from `background` config: `default_step_timeout: "5m"`, `max_step_timeout: "0"` (unlimited)
→ Instead of worker's `default_step_timeout: "30s"`, `max_step_timeout: "5m"`

This is consistent — developer thinks "this is slow work" → sets `pool: "background"` → gets relaxed timeout, regardless of whether runtime is embedded or child process.

### 10.2 max_memory_mb for Embedded Runtimes

**goja**: `max_memory_mb` **cannot be enforced**. goja allocates on Go heap. There's no per-VM memory tracking in Go. If a goja script creates a 2GB array, the Go process grows by 2GB. The only protection is the OS OOM killer.

**QuickJS (qjs)**: `max_memory_mb` **can be enforced** via Wazero's memory limit:

```go
func (r *QJSRuntime) NewVM(opts VMOptions) (embedded.VM, error) {
    config := wazero.NewModuleConfig()
    if opts.MaxMemoryMB > 0 {
        // Wazero memory is in pages (64KB each)
        pages := uint32(opts.MaxMemoryMB * 1024 / 64)
        config = config.WithMemoryLimitPages(pages)
    }
    // ... create runtime with config
}
```

**Implication**: If memory limits are important for a script, use `runtime: "javascript:quickjs"` (enforceable) instead of `runtime: "javascript:goja"` (not enforceable). Or use child process runtimes (Phase 2-3) where memory is enforced via process RSS monitoring.

| Runtime | max_memory_mb enforceable? | How |
|---------|--------------------------|-----|
| goja | ❌ No | Go heap, no per-VM tracking |
| quickjs | ✅ Yes | Wazero WASM memory pages |
| Node.js | ✅ Yes | Process RSS monitoring + kill |
| Python | ✅ Yes | Process RSS monitoring + kill |
| yaegi | ❌ No | Go heap, same as goja |

### 10.3 `bitcode.tx()` in Embedded Runtimes

In child process (Phase 2-3), transactions work via `txId` in JSON-RPC — each bridge call within tx includes txId, Go routes to transaction DB.

In embedded runtimes, bridge calls are direct Go function calls. Transaction is handled by **re-injecting bridge context**:

```go
// Inside tx(), the bridge context is swapped to tx-scoped version
err := bc.Tx(func(txCtx *bridge.Context) error {
    // Temporarily replace global "bitcode" with tx-scoped version
    vm.InjectBridge(txCtx)
    defer vm.InjectBridge(bc)  // restore original after tx
    
    // Now all bitcode.model() calls inside callback use tx DB
    _, err := callable(goja.Undefined())
    return err
})
```

This means:
- All `bitcode.model()` calls inside `tx()` callback use transaction DB ✅
- `bitcode.http()`, `bitcode.cache()`, etc. are NOT transactional (same as Phase 2-3) ✅
- If callback throws, transaction is rolled back ✅
- After tx completes (commit or rollback), original bridge is restored ✅

### 10.4 Script Signature Detection

Scripts can use multiple patterns. Engine detects and handles all:

```javascript
// Pattern 1: Object with execute method (recommended)
module.exports = {
    execute(bitcode, params) {
        return { success: true };
    }
};

// Pattern 2: Default export (ES module style — qjs only)
export default {
    execute(bitcode, params) {
        return { success: true };
    }
};

// Pattern 3: Direct function export
module.exports = function(bitcode, params) {
    return { success: true };
};

// Pattern 4: Legacy (no bitcode arg)
module.exports = {
    execute(params) {
        return { total: params.leads.length };
    }
};

// Pattern 5: Simple expression (no exports)
// Script just runs and the last expression is the result
const leads = bitcode.model("lead").search({});
({ count: leads.length });
```

Detection logic:

```go
func (v *GojaVM) Execute(code string, filename string) (any, error) {
    val, err := v.rt.RunProgram(program)
    // ...
    exported := val.Export()
    
    // Pattern 1/2/4: object with execute function
    if m, ok := exported.(map[string]any); ok {
        if execFn, exists := m["execute"]; exists {
            callable, _ := goja.AssertFunction(v.rt.ToValue(execFn))
            // Detect param count for backward compat
            return callWithSignatureDetection(callable, v.rt)
        }
    }
    
    // Pattern 3: direct function
    if callable, ok := goja.AssertFunction(val); ok {
        return callWithSignatureDetection(callable, v.rt)
    }
    
    // Pattern 5: expression result
    return exported, nil
}

func callWithSignatureDetection(fn goja.Callable, rt *goja.Runtime) (any, error) {
    bitcode := rt.Get("bitcode")
    params := rt.Get("params")
    
    // Try new style: execute(bitcode, params)
    result, err := fn(goja.Undefined(), bitcode, params)
    if err != nil {
        // If function only accepts 1 param, try legacy: execute(params)
        result, err = fn(goja.Undefined(), params)
    }
    // ...
}
```

### 10.6 Script File Auto-Detection

File extension determines default runtime when `runtime` field is not set in process JSON:

```
.js  → default: "javascript" (embedded goja — lightest, fastest)
.ts  → default: "node" (TypeScript needs Bun native or Node.js + esbuild)
.py  → default: "python"
.go  → default: "go" (yaegi)
```

**Important**: `.js` files default to **embedded** goja, NOT Node.js child process. This is intentional — embedded is faster and has zero external dependency. Developer must explicitly set `runtime: "node"` if they need npm packages.

**Override**: `runtime` field in process JSON always wins over auto-detection.

```json
// .js file → auto: "javascript" (goja). Override to Node.js:
{ "type": "script", "script": "scripts/crawl.js", "runtime": "node" }

// .js file → auto: "javascript" (goja). Override to QuickJS:
{ "type": "script", "script": "scripts/complex.js", "runtime": "javascript:quickjs" }
```

**Validation**: `.ts` + `runtime: "javascript"` → error. TypeScript requires `runtime: "node"` (Bun native or Node.js + esbuild). See Phase 2 §10.10 for full validation rules.

### 10.7 ES Module Syntax (import/export)

| Syntax | goja | quickjs |
|--------|------|---------|
| `module.exports = { ... }` | ✅ (via goja_nodejs require) | ✅ |
| `exports.execute = function() {}` | ✅ | ✅ |
| `export default { ... }` | ❌ Not supported | ✅ |
| `import { x } from './other.js'` | ❌ Not supported | ✅ |

**Implication**: Scripts using ES module syntax (`import`/`export`) must use `runtime: "javascript:quickjs"`. goja only supports CommonJS (`module.exports`/`require`).

---

## 11. Edge Cases

### 11.1 Script Execution

| Edge Case | Behavior |
|-----------|----------|
| Script has syntax error | `goja.Compile()` / `ctx.Eval()` returns error. `BridgeError{Code: "SYNTAX_ERROR"}` |
| Script throws unhandled error | Caught by executor. `BridgeError{Code: "SCRIPT_ERROR"}` |
| Script infinite loop | `vm.Interrupt()` after timeout. `BridgeError{Code: "STEP_TIMEOUT"}` |
| Script calls `process.exit()` | Not available in sandbox (not exposed). No effect. |
| Script modifies `bitcode` object | Allowed but only affects current execution. New VM next time. |
| Script file not found | `BridgeError{Code: "FS_NOT_FOUND"}` |
| Script file changed between executions | Hot reload — re-read every time. goja cache invalidated by mtime. |

### 11.2 Bridge Calls

| Edge Case | Behavior |
|-----------|----------|
| `bitcode.model("nonexistent").search({})` | `BridgeError{Code: "MODEL_NOT_FOUND"}` — same as Phase 2-3 |
| Bridge call during timeout interrupt | goja: interrupt happens at next JS instruction, not mid-bridge-call. Bridge call completes, then interrupt. |
| Bridge call throws error | Error propagated to script as exception. Script can try/catch. |
| `bitcode.tx()` in goja (no async) | Works — `tx()` takes a callback function, not async. `bitcode.tx(function(tx) { tx.model(...) })` |
| `await bitcode.model().search()` in goja | Error — goja doesn't support await. Use without await (synchronous). |
| `await bitcode.model().search()` in quickjs | Works — await on synchronous value resolves immediately. |

### 11.3 Memory

| Edge Case | Behavior |
|-----------|----------|
| Script creates huge array (1M elements) | goja: Go heap grows. qjs: WASM memory grows. Both bounded by Go process memory. |
| QuickJS value not Free()'d | WASM memory leak within execution. Cleaned up when VM is closed (per execution). |
| goja VM not garbage collected | Go GC handles it. No manual cleanup needed. |
| Many concurrent executions (100 VMs) | Each VM ~1-2MB. 100 VMs = ~100-200MB. Acceptable. |

### 11.4 Type Conversion

| Edge Case | Behavior |
|-----------|----------|
| JS `undefined` returned | Go receives `nil` |
| JS `null` returned | Go receives `nil` |
| JS `BigInt` in goja | Not supported — error. Use quickjs for BigInt. |
| JS `Date` object | goja: converts to Go `time.Time`. qjs: converts to string (ISO 8601). |
| Go `[]byte` passed to JS | goja: becomes JS ArrayBuffer. qjs: becomes JS Uint8Array. |
| Circular reference in JS object | goja: `Export()` panics (recovered by executor). qjs: `JsValueToGo` returns error. |

### 11.5 Concurrency

| Edge Case | Behavior |
|-----------|----------|
| Two scripts execute simultaneously | Two separate VMs in two goroutines. No shared state. Safe. |
| Script calls `bitcode.call()` which triggers another script | Nested execution. New VM created for inner script. Same goroutine (sequential). |
| goja VM used from two goroutines | NOT SAFE. But we create per-execution, so this doesn't happen. |

---

## 12. Implementation Tasks

### Files to Create

```
engine/internal/runtime/embedded/
├── runtime.go              # EmbeddedRuntime + VM interfaces
├── executor.go             # ExecuteEmbedded() — shared execution logic
├── script_loader.go        # File loading + hot reload
├── bridge_helper.go        # Shared: parseSearchOpts, parseHTTPOpts, etc.
├── registry.go             # Engine registry + resolution
│
├── goja/
│   ├── runtime.go          # GojaRuntime
│   ├── vm.go               # GojaVM (InjectBridge, Execute, Interrupt)
│   └── proxy.go            # Model proxy, sub-proxies for all 20 namespaces
│
└── qjs/
    ├── runtime.go          # QJSRuntime
    ├── vm.go               # QJSVM (InjectBridge, Execute, Interrupt)
    ├── proxy.go            # Host function registration for all 20 namespaces
    └── bitcode_init.js     # JS wrapper that creates bitcode.* API from flat __bitcode_* functions
```

### Files to Modify

```
engine/go.mod
  → Add: github.com/dop251/goja
  → Add: github.com/dop251/goja_nodejs
  → Add: github.com/fastschema/qjs

engine/internal/runtime/executor/steps/script.go
  → Route runtime "javascript" / "javascript:goja" / "javascript:quickjs" to embedded executor

engine/internal/runtime/plugin/manager.go
  → Add detectRuntime() case for "javascript" → embedded (not child process)

engine/internal/compiler/parser/module.go
  → Add RuntimeDefaults field to ModuleDefinition

engine/internal/config.go
  → Add JavaScript.DefaultEngine to RuntimeConfig

engine/internal/app.go
  → Initialize EngineRegistry
  → Wire into ScriptHandler
```

### Task Breakdown

| # | Task | Effort | Priority |
|---|------|--------|----------|
| **Shared** | | | |
| 1 | Create `embedded/runtime.go` — EmbeddedRuntime + VM interfaces | Small | Must |
| 2 | Create `embedded/executor.go` — ExecuteEmbedded with timeout + panic recovery | Medium | Must |
| 3 | Create `embedded/script_loader.go` — file loading + hot reload | Small | Must |
| 4 | Create `embedded/bridge_helper.go` — parseSearchOpts, parseHTTPOpts, etc. | Medium | Must |
| 5 | Create `embedded/registry.go` — engine registry + resolution | Small | Must |
| **goja** | | | |
| 6 | Create `goja/runtime.go` — GojaRuntime with require.Registry | Small | Must |
| 7 | Create `goja/vm.go` — GojaVM with InjectBridge (all 20 namespaces) | Large | Must |
| 8 | Create `goja/proxy.go` — model proxy with sudo/relations/bulk | Medium | Must |
| 9 | Implement goja compilation cache with mtime invalidation | Small | Should |
| **QuickJS** | | | |
| 10 | Create `qjs/runtime.go` — QJSRuntime | Small | Must |
| 11 | Create `qjs/vm.go` — QJSVM with InjectBridge (host functions) | Large | Must |
| 12 | Create `qjs/proxy.go` — host function registration for all 20 namespaces | Medium | Must |
| 13 | Create `qjs/bitcode_init.js` — JS wrapper for bitcode.* API | Medium | Must |
| **Integration** | | | |
| 14 | Update `script.go` — route "javascript" to embedded executor | Small | Must |
| 15 | Update `manager.go` — detectRuntime for "javascript" | Small | Must |
| 16 | Add RuntimeDefaults to ModuleDefinition | Small | Must |
| 17 | Add JavaScript.DefaultEngine to RuntimeConfig | Small | Must |
| 18 | Wire EngineRegistry in app.go | Small | Must |
| 19 | Add goja + qjs to go.mod | Small | Must |
| **Tests** | | | |
| 20 | Tests: shared executor (timeout, panic recovery, hot reload) | Medium | Must |
| 21 | Tests: goja — all 20 bridge namespaces | Large | Must |
| 22 | Tests: qjs — all 20 bridge namespaces | Large | Must |
| 23 | Tests: goja — model proxy (CRUD + bulk + relations + sudo) | Medium | Must |
| 24 | Tests: qjs — model proxy (same) | Medium | Must |
| 25 | Tests: goja — timeout/interrupt | Small | Must |
| 26 | Tests: qjs — timeout/interrupt | Small | Must |
| 27 | Tests: engine registry + resolution (step → module → project) | Medium | Must |
| 28 | Tests: goja compilation cache + invalidation | Small | Should |
| 29 | Tests: type conversion edge cases (undefined, null, BigInt, Date, circular) | Medium | Must |
| 30 | Tests: concurrent executions (multiple VMs in parallel goroutines) | Medium | Should |
| 31 | Tests: script signature detection (module.exports.execute vs direct return) | Small | Must |
| 32 | Tests: console.log interception → bridge logger | Small | Should |

### Effort Comparison

| Component | Lines (estimate) | Shared? |
|-----------|-----------------|---------|
| Interfaces + executor + loader + helper + registry | ~400 | ✅ Shared |
| goja/ (runtime + vm + proxy) | ~500 | ❌ goja only |
| qjs/ (runtime + vm + proxy + bitcode_init.js) | ~600 | ❌ qjs only |
| Integration (script.go, manager.go, app.go, config) | ~100 | ✅ Shared |
| **Total** | **~1600** | **500 shared (31%)** |

The shared code percentage is lower than the 80% I estimated earlier because the proxy code (mapping 20 bridge namespaces to runtime-specific APIs) is the bulk of the work and is per-engine. But the **logic** is not duplicated — both proxies call the same `bridge.Context` methods. The difference is only in how values are converted.
