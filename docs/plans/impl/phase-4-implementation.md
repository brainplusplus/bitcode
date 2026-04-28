# Phase 4 Implementation Plan: Embedded JS (goja + QuickJS)

**Estimated effort**: 8-10 days
**Prerequisites**: Phase 1 (bridge interfaces)
**Test command**: `go test ./internal/runtime/embedded/...`

---

## Implementation Order

```
Stream 1: Shared Interfaces & Executor (Day 1-2)
  ↓
Stream 2: goja Runtime (Day 2-5)
  ↓
Stream 3: QuickJS Runtime (Day 5-8)
  ↓
Stream 4: Integration & Registry (Day 8-9)
  ↓
Stream 5: Tests (Day 9-10)
```

---

## Stream 1: Shared Layer

**Directory**: `internal/runtime/embedded/`

- `runtime.go` — `EmbeddedRuntime` interface (CreateVM, Shutdown) + `VM` interface (InjectBridge, Execute, Interrupt)
- `executor.go` — `ExecuteEmbedded()` with timeout (context.WithTimeout), panic recovery, VM pool acquire/release
- `script_loader.go` — Load script file, detect signature (module.exports.execute vs direct return), hot reload via mtime
- `bridge_helper.go` — Shared conversion helpers: `parseSearchOpts()`, `parseHTTPOpts()`, `mapToFieldDomain()`, etc.
- `registry.go` — Engine registry: map runtime string → EmbeddedRuntime. Resolution: step → module → project → default

## Stream 2: goja

**Directory**: `internal/runtime/embedded/goja/`

- `runtime.go` — GojaRuntime: create VM pool, require.Registry for CommonJS
- `vm.go` — GojaVM: InjectBridge injects `bitcode` object with all 20 namespaces as Go functions. Promise support via goja event loop.
- `proxy.go` — Model proxy: `bitcode.model("contact")` returns proxy object with `.get()`, `.search()`, `.create()`, `.write()`, `.delete()`, `.sudo()`, etc.

**Key challenge**: goja Promises. Use `goja.EventLoop` for async/await support. All bridge calls return Promises.

## Stream 3: QuickJS (via Wazero)

**Directory**: `internal/runtime/embedded/qjs/`

- `runtime.go` — QJSRuntime: create Wazero instances
- `vm.go` — QJSVM: Register host functions (`__bitcode_model_get`, `__bitcode_db_query`, etc.)
- `proxy.go` — Host function registration for all 20 namespaces
- `bitcode_init.js` — JS wrapper that creates `bitcode.*` API from flat `__bitcode_*` host functions

**Key challenge**: QJS ↔ Go communication is via host functions (flat, not objects). `bitcode_init.js` wraps them into the same `bitcode.*` API that goja exposes.

## Stream 4: Integration

- Update `steps/script.go` — route `"javascript"`, `"javascript:goja"`, `"javascript:quickjs"` to embedded executor
- Update `plugin/manager.go` — `detectRuntime()` for embedded JS
- Add `goja` + `qjs` to `go.mod`
- Wire `EngineRegistry` in `app.go`

## Definition of Done

- [ ] goja executes JS scripts with all 20 bridge namespaces
- [ ] QuickJS executes JS scripts with all 20 bridge namespaces
- [ ] Both support async/await
- [ ] Timeout/interrupt works for both
- [ ] VM pool with configurable size
- [ ] Compilation cache with mtime invalidation (goja)
- [ ] Memory limit enforceable (QuickJS via Wazero pages)
- [ ] Runtime resolution: step → module → project → default
- [ ] All 12 sample scripts (6 TS transpiled to JS) execute successfully
