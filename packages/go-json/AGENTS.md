# AGENTS.md — go-json

## Overview

Standalone JSON/JSONC programming language engine. Embeddable in Go applications. Part of the BitCode platform but independently usable.

**Pipeline:** JSONC pre-process → JSON parse → import resolution → AST → compile (struct registration, structural validation, limit resolution) → immutable Program → VM execution with debug hooks.

**Expression evaluation** delegated to [expr-lang/expr](https://github.com/expr-lang/expr) via `ExprEngine` abstraction layer. The VM never calls expr-lang directly.

**Phase 4.5a design:** `docs/plans/2026-07-14-runtime-engine-phase-4.5a-go-json-core-language.md`
**Phase 4.5b design:** `docs/plans/2026-07-14-runtime-engine-phase-4.5b-go-json-modularity.md`
**Phase 4.5b plan:** `docs/plans/2026-07-14-runtime-engine-phase-4.5b-go-json-modularity-plan.md`
**Decisions:** `docs/plans/2026-04-28-go-json-brainstorming-design.md`

## Package Structure

```
packages/go-json/
├── lang/           Core language engine (AST, parser, compiler, VM, scope, types, errors, expr engine, debugger, import resolver)
├── stdlib/         Layer 2 stdlib (34 functions + crypto namespace). Layer 1 = expr-lang built-ins (~68 functions, zero work)
├── runtime/        Runtime API: NewRuntime(), Execute(), CompileFile(), program cache, limits, logger, session
├── cmd/go-json/    CLI placeholder
├── io/             Reserved for Phase 4.5c (I/O modules)
└── testdata/       Test fixture programs (.json, .jsonc)
```

## Key Architecture Decisions

1. **ExprEngine abstraction** — VM never calls expr-lang directly. All expression work goes through `ExprEngine` interface for testability and swappability.
2. **Compile-once, run-many** — `CompiledProgram` is immutable after compilation. Each execution gets fresh VM + scope. Multiple goroutines can run the same program concurrently.
3. **Structural validation at compile time, expression validation at runtime** — expr-lang's compile-time type checking requires a fully-typed environment which we don't have with gradual typing. Runtime catches expression errors.
4. **Scope isolation for functions and methods** — `IsolatedChild()` creates scope WITHOUT parent link. Functions and methods cannot access caller variables. `self` is injected into method scope. Block scope (`NewChild()`) for if/for/while allows reading and mutating outer variables.
5. **Sentinel types for control flow** — `returnValue`, `breakSignal`, `continueSignal` are unexported struct types that propagate through `executeSteps()` return values.
6. **Resource limits at every step** — step count, call depth, loop iterations, timeout checked before each step execution. MaxVariables, MaxVariableSize checked after every `Declare()`. MaxOutputSize checked on program return.
7. **JSON param ordering** — Go maps don't preserve insertion order. `extractOrderedKeys()` uses `json.Decoder` tokenization to recover function param order from raw JSON.
8. **Built-in name protection** — `let` blocks variable names that shadow critical built-in functions (len, abs, min, max, etc.). Curated list excludes common-word functions (count, filter, sort) that are also natural variable names.
9. **Implicit scope variables** — `session.*` (user_id, locale, tenant_id, groups) and `execution.*` (id, program, started_at, depth, step_count) injected automatically into every execution.
10. **Trace enrichment** — `TraceEntry` captures Var/Value for let/set, Condition/Result for if/while/switch, per step type via `enrichTraceEntry()`.

## Step Types (16)

`let`, `set`, `if`/`elif`/`else`, `switch`, `for` (each + range), `while`, `break`, `continue`, `return`, `call`, `try`/`catch`/`finally`, `error`, `log`, `parallel`, `_c` (comment)

## Struct System (Phase 4.5b)

- Structs defined in `structs` block with fields, methods, optional `frozen: true`
- Construction via `{"let": "x", "new": "StructName", "with": {...}}`
- Nested construction: `"field": {"new": "Other", "with": {...}}`
- Methods with implicit `self` binding, callable at expression and step level
- Frozen structs: compile-time rejection of `set "self.*"` in methods
- Forward references resolved via two-pass compilation
- Circular non-nullable struct references detected at compile time

## Import System (Phase 4.5b)

- Import key: `"import"` (preferred) or `"imports"` (compat)
- Path types: relative (`./`), stdlib (`stdlib:`), extension (`ext:`), I/O (`io:`)
- Imported items namespaced via alias: `alias.StructName`, `alias.functionName`
- Circular import detection via import stack
- Barrel file re-export via `{"alias": "imported.Type"}` in structs block
- Diamond imports handled correctly (loaded once, cached)
- Wired via `Runtime.CompileFile(path)` — import resolution between parse and compile

## Parallel Execution (Phase 4.5b)

- `{"parallel": {"branch1": [...], "branch2": [...]}, "into": "results"}`
- Each branch gets own VM + scope (read parent, cannot write parent)
- Compile-time check: `set` targeting parent variable in parallel branch = error
- Error modes: `cancel_all` (default), `continue`, `collect`
- Goroutine leak prevention: drain channel after cancel

## Stdlib Layers

| Layer | Contents | Ownership |
|-------|----------|-----------|
| Layer 1 | expr-lang built-ins (~68 functions: abs, ceil, floor, round, min, max, len, upper, lower, trim, split, filter, map, reduce, find, sort, int, float, string, type, etc.) | expr-lang — DO NOT reimplement |
| Layer 2 | go-json additions (34 functions + crypto namespace). Phase 4.5a: clamp, sign, randomInt, randomFloat, pow, sqrt, mod, padLeft, padRight, substring, format, matches, append, prepend, slice, chunk, zip, bool, isNil. Phase 4.5b: has, get, merge, pick, omit, formatDate, addDuration, diffDates, urlEncode, urlDecode, sprintf, crypto.sha256, crypto.md5, crypto.uuid, crypto.hmac | `stdlib/` package |
| Layer 3 | I/O / host modules (DB, HTTP, file, etc.) | Phase 4.5c — not yet implemented |

## Conventions

- Follow root `AGENTS.md` conventions (no unnecessary comments, tests required)
- All exported types and functions need Go doc comments
- `go build ./...` and `go vet ./...` must pass
- Tests: `go test ./... -v`

## Testing

```bash
cd packages/go-json
go test ./... -v          # All tests (131)
go test ./lang/ -v        # Language engine tests
go test ./lang/ -run TestStruct -v       # Struct tests
go test ./lang/ -run TestMethod -v       # Method tests
go test ./lang/ -run TestParallel -v     # Parallel tests
go test ./lang/ -run TestImport -v       # Import tests
go test ./lang/ -run TestIntegration -v  # Integration tests
go test ./stdlib/ -v                     # Stdlib tests
```

## What's Done (Phase 4.5b)

- Struct definitions with fields, defaults, frozen, methods
- Struct construction (`new` + `with`), nested construction, return with new
- Method system with `self` binding, mutation, frozen compile-time check
- Import system with relative file resolution, circular detection, barrel files
- Parallel execution with 3 error modes, scope isolation, compile-time parent write check
- Stdlib Layer 2 extensions: maps, datetime, encoding, crypto (namespaced), format
- Nullable type support (`?T`), optional chaining via expr-lang

## What's NOT Done (Phase 4.5c)

- I/O modules (DB, HTTP, file, email) — `io:` imports parsed but not resolved
- stdlib/ext module resolution — `stdlib:`/`ext:` imports parsed but not resolved
- BitCode engine integration (process engine routing `.json` scripts to go-json)
- Expression-level compile-time type validation (deferred to runtime)
