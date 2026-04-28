# Phase 5 Implementation Plan: yaegi (Embedded Go Runtime)

**Estimated effort**: 6-8 days
**Prerequisites**: Phase 1 (bridge interfaces), Phase 4 (shared EmbeddedRuntime interface)
**Test command**: `go test ./internal/runtime/embedded/yaegi/...`

---

## Implementation Order

```
Stream 1: Core yaegi Runtime (Day 1-3)
  ↓
Stream 2: Bridge Injection as Go Package (Day 3-5)
  ↓
Stream 3: Custom Bridges Folder + go.mod (Day 5-6)
  ↓
Stream 4: os/exec Whitelist & Stdlib Filter (Day 6-7)
  ↓
Stream 5: Integration & Tests (Day 7-8)
```

---

## Stream 1: Core Runtime

**Directory**: `internal/runtime/embedded/yaegi/`

- `runtime.go` — YaegiRuntime implements EmbeddedRuntime. Creates `interp.New()` with filtered stdlib.
- `vm.go` — YaegiVM implements VM. InjectBridge loads `bitcode` package as `interp.Exports`. Execute runs `main.Run(ctx)` or `main.Execute(params)`.

**Key challenge**: Timeout. yaegi has no built-in interrupt. Use `context.Context` cooperative cancellation. Scripts must check `ctx.Done()` in loops. Non-cooperative scripts = goroutine leak (documented limitation).

## Stream 2: Bridge Injection

Inject `bitcode` package as yaegi symbols:
```go
interp.Exports{
    "bitcode/bitcode": {
        "Model":   reflect.ValueOf(bridgeCtx.Model),
        "DB":      reflect.ValueOf(bridgeCtx.DB),
        "Session": reflect.ValueOf(bridgeCtx.Session),
        // ... all 20 namespaces
    },
}
```

Go scripts import: `import "bitcode"` → `bitcode.Model("contact").Get(id)`

## Stream 3: Custom Bridges + go.mod

- `bridge_loader.go` — Scan `bridges/` folder (project-level + module-level), load `.go` files as yaegi symbols
- go.mod per module: `bitcode module extract-deps` CLI command → `yaegi extract` pipeline → inject symbols

## Stream 4: Security

- `stdlib_filter.go` — Remove `os/exec`, `unsafe`, `syscall` from yaegi stdlib. Allow `os.ReadFile`, `os.Stat`, etc.
- `bitcode.Exec()` bridge is the ONLY way to run external commands (with whitelist)

## Definition of Done

- [ ] yaegi executes Go scripts with all 20 bridge namespaces
- [ ] Context-based timeout works (cooperative cancellation)
- [ ] Custom bridges/ folder loaded (project + module level)
- [ ] go.mod per module works (extract-deps + yaegi extract)
- [ ] os/exec blocked in stdlib, only via bitcode.Exec() with whitelist
- [ ] unsafe, syscall blocked
- [ ] Goroutine support works (WaitGroup, channels)
- [ ] Panic recovery works
