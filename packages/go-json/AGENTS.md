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
├── lang/           Core language engine (AST, parser, compiler, VM, scope, types, errors, expr engine, debugger)
├── stdlib/         Layer 2 stdlib (19 functions). Layer 1 = expr-lang built-ins (~68 functions, zero work)
├── runtime/        Runtime API: NewRuntime(), Execute(), program cache, limits, logger, session
├── cmd/go-json/    CLI placeholder
├── codegen/        Reserved for Phase 4.5b (struct codegen)
├── io/             Reserved for Phase 4.5c (I/O modules)
└── testdata/       Test fixture programs (.json, .jsonc)
```

## Key Architecture Decisions

1. **ExprEngine abstraction** — VM never calls expr-lang directly. All expression work goes through `ExprEngine` interface for testability and swappability.
2. **Compile-once, run-many** — `CompiledProgram` is immutable after compilation. Each execution gets fresh VM + scope. Multiple goroutines can run the same program concurrently.
3. **Structural validation at compile time, expression validation at runtime** — expr-lang's compile-time type checking requires a fully-typed environment which we don't have with gradual typing. Runtime catches expression errors.
4. **Scope isolation for functions** — `IsolatedChild()` creates scope WITHOUT parent link. Functions cannot access caller variables. Block scope (`NewChild()`) for if/for/while allows reading and mutating outer variables.
5. **Sentinel types for control flow** — `returnValue`, `breakSignal`, `continueSignal` are unexported struct types that propagate through `executeSteps()` return values.
6. **Resource limits at every step** — step count, call depth, loop iterations, timeout checked before each step execution. MaxVariables, MaxVariableSize checked after every `Declare()`. MaxOutputSize checked on program return.
7. **JSON param ordering** — Go maps don't preserve insertion order. `extractOrderedKeys()` uses `json.Decoder` tokenization to recover function param order from raw JSON.
8. **Built-in name protection** — `let` blocks variable names that shadow critical built-in functions (len, abs, min, max, etc.). Curated list excludes common-word functions (count, filter, sort) that are also natural variable names.
9. **Implicit scope variables** — `session.*` (user_id, locale, tenant_id, groups) and `execution.*` (id, program, started_at, depth, step_count) injected automatically into every execution.
10. **Trace enrichment** — `TraceEntry` captures Var/Value for let/set, Condition/Result for if/while/switch, per step type via `enrichTraceEntry()`.

## Step Types (15)

`let`, `set`, `if`/`elif`/`else`, `switch`, `for` (each + range), `while`, `break`, `continue`, `return`, `call`, `try`/`catch`/`finally`, `error`, `log`, `_c` (comment)

## Stdlib Layers

| Layer | Contents | Ownership |
|-------|----------|-----------|
| Layer 1 | expr-lang built-ins (~68 functions: abs, ceil, floor, round, min, max, len, upper, lower, trim, split, filter, map, reduce, find, sort, int, float, string, type, etc.) | expr-lang — DO NOT reimplement |
| Layer 2 | go-json additions (19 functions: clamp, sign, randomInt, randomFloat, pow, sqrt, mod, padLeft, padRight, substring, format, matches, append, prepend, slice, chunk, zip, bool, isNil) | `stdlib/` package |
| Layer 3 | I/O / host modules (DB, HTTP, file, etc.) | Phase 4.5c — not yet implemented |

## Conventions

- Follow root `AGENTS.md` conventions (no unnecessary comments, tests required)
- All exported types and functions need Go doc comments
- `go build ./...` and `go vet ./...` must pass
- Tests: `go test ./... -v`

## Testing

```bash
cd packages/go-json
go test ./... -v          # All tests (82)
go test ./lang/ -v        # Language engine tests
go test ./lang/ -run TestIntegration -v  # Integration tests only
go test ./lang/ -run TestEdge -v         # Edge case tests only
```

## What's NOT Done (Phase 4.5b, 4.5c)

- Struct definitions and codegen
- Import/module system
- I/O modules (DB, HTTP, file, email)
- BitCode engine integration (process engine routing `.json` scripts to go-json)
- Expression-level compile-time type validation (deferred to runtime — gradual typing and compile-time validation are fundamentally in tension, see compiler.go)
- `Timezone` expr-lang config (no date/time stdlib functions in Phase 4.5a)
- `WithContext` expr-lang config (no long-running custom functions in Phase 4.5a)
- Stdlib deprecation support (nothing to deprecate yet — add when stdlib evolves)
