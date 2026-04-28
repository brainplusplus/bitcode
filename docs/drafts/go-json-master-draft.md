# go-json — Standalone JSON Programming Language (Master Draft)

**Status**: Draft
**Package**: `packages/go-json`
**Module**: `github.com/bitcode-framework/go-json`
**Phases**: 4.5a, 4.5b, 4.5c

---

## 1. Vision

go-json is a standalone, general-purpose programming language written in JSON format, embeddable in Go applications.

**Analogy:**
```
Stencil : Web Components : Bitcode View Engine
go-json : JSON Language  : Bitcode Process Engine
```

Stencil can be used without bitcode — anyone can build web components with Stencil. But bitcode uses Stencil as its core view engine.

go-json can be used without bitcode — anyone can build automation, workflows, business rules with go-json. But bitcode uses go-json as its core process/scripting engine.

### 1.1 What Makes go-json Different

| Existing Tool | Focus | Limitation |
|---|---|---|
| jsonnet | Data templating | No side effects (no I/O) |
| jq | JSON transformation | Single-purpose, no general programming |
| AWS Step Functions | Cloud workflow | AWS-locked, no local execution |
| Node-RED | IoT/integration | Node.js-dependent, visual-only |
| n8n | Automation | SaaS-focused, not embeddable |
| CEL | Policy/rules | Expression only, no control flow |
| expr-lang | Expression evaluation | Expression only, no statements |

**go-json's niche**: General-purpose JSON programming language embeddable in Go applications. No existing tool fills this niche.

### 1.2 Design Principles

1. **JSON-native** — Programs are valid JSON. No custom syntax to learn beyond JSON.
2. **Standalone** — Zero dependency on bitcode. Usable by anyone.
3. **Embeddable** — Host applications extend go-json via extension hooks.
4. **Gradually typed** — Untyped for quick scripts, fully typed for production.
5. **Safe by default** — Resource limits, no infinite loops, memory-safe.
6. **Code-generation ready** — Well-defined AST enables transpilation to Go/JS/Python.

---

## 2. Topic Index — Where to Find What

### Phase 4.5a — Core Language + Stdlib
**Document**: [go-json-phase-4.5a-core-language.md](./go-json-phase-4.5a-core-language.md)

| Topic | Section |
|---|---|
| Package structure (`packages/go-json/`) | §1 |
| Expression engine (expr-lang/expr) | §2 |
| Value assignment: `value` / `expr` / `with` | §3 |
| Variable declaration: `let` / `set` | §3 |
| Type system: inference, gradual typing, input schema | §4 |
| Control flow: `if`/`elif`/`else`, `switch` | §5 |
| Loops: `for`/`while`/`range`, `break`/`continue` | §6 |
| Functions: definition, params, returns | §7 |
| Recursion: depth limit, isolated scope, tail-call | §7.4 |
| Error handling: `try`/`catch`/`finally`, `error` | §8 |
| Return values: `return`, computed objects | §9 |
| Resource limits: depth, steps, timeout, memory proxy | §10 |
| Config ordering: engine > project > module > program > step | §10.3 |
| Stdlib Tier 1: math (14), string (20), array (20), type (6) | §11 |
| Context: session, execution metadata | §12 |
| Variable scoping: block scope, isolation | §13 |

### Phase 4.5b — Modularity (Struct + Import)
**Document**: [go-json-phase-4.5b-modularity.md](./go-json-phase-4.5b-modularity.md)

| Topic | Section |
|---|---|
| Struct definition: fields, defaults, nested structs | §1 |
| Struct methods: `self`, mutation vs immutable | §2 |
| Struct construction: `new` + `with` | §3 |
| Nested property access + mutation: `set "a.b.c"` | §4 |
| Import system: relative, stdlib, extension | §5 |
| Import resolution rules | §5.3 |
| Export rules: structs + functions exportable, steps not | §5.4 |
| Circular import detection | §5.5 |
| Re-export / barrel files | §5.6 |
| Stdlib Tier 2: map (8), datetime (10), encoding (6), crypto (4), format (2) | §6 |
| Parallel execution: `parallel`/`join`, isolated branches | §7 |
| Nullable types: `?string`, `?Person` | §8 |

### Phase 4.5c — I/O + Integration + Code Generation
**Document**: [go-json-phase-4.5c-io-integration.md](./go-json-phase-4.5c-io-integration.md)

| Topic | Section |
|---|---|
| I/O modules: HTTP, FS, SQL, exec | §1 |
| I/O security: enable/disable per module | §1.2 |
| Bitcode bridge integration via extension hooks | §2 |
| Extension API: `WithExtension()`, `WithoutIO()` | §2.2 |
| How bitcode replaces raw I/O with bridge | §2.3 |
| `scripts/*.json` support in bitcode process engine | §3 |
| Migration path: current process engine → go-json | §3.2 |
| AST export for code generation | §4 |
| Code generation targets: Go, JavaScript, Python | §4.2 |
| Standalone CLI runner: `go-json run program.json` | §5 |
| REPL / playground | §5.2 |

---

## 3. Open Questions

These are unresolved design decisions that need further discussion. Each is marked `[OPEN]` in the relevant phase document.

### OQ-1: Return Computed Objects
**Phase**: 4.5a §9
**Problem**: Two keywords for return (`return` + `return_object`) is ugly.
**Options**:
- A: `{"return": "expr"}` + `{"return": {"with": {...}}}` — overloaded return
- B: `{"return": "expr"}` + `{"return_object": {...}}` — separate keyword
- C: `{"return": "{'key': value}"}` — object literal in expression (expr-lang supports this)
**Current lean**: Option A — `return` with optional `with` sub-key.

### OQ-2: Struct Mutability
**Phase**: 4.5b §2
**Problem**: Mutable structs (`set: "self.age"`) complicate code generation to functional languages.
**Options**:
- A: Mutable by default (like Go) — simple, familiar
- B: Immutable by default, explicit `mut` — safer, code-gen friendly
- C: Hybrid — methods can return new struct or mutate, user chooses
**Current lean**: Option A — mutable by default. Code generation to functional languages can auto-transform to copy-on-write.

### OQ-3: Function Call Duality
**Phase**: 4.5a §7
**Problem**: Two ways to call functions — step-level `call` and expression-level `factorial(n)`.
**Rule needed**: When to use which?
**Current lean**: Expression-level for pure functions (no side effects). Step-level `call` for functions that have steps/side effects or need `with` (computed input object).

### OQ-4: Error Types
**Phase**: 4.5a §8
**Problem**: Are errors just strings, or structured objects?
**Options**:
- A: String only — `{"error": "'something went wrong'"}`
- B: Structured — `{"error": {"code": "'VALIDATION'", "message": "'invalid email'", "details": "errors"}}`
- C: Both — string shorthand + structured form
**Current lean**: Option C — string shorthand for simple cases, structured for complex.

### OQ-5: Parallel Error Handling
**Phase**: 4.5b §7
**Problem**: In parallel execution, if one branch fails, what happens to others?
**Options**:
- A: Cancel all on first failure
- B: Wait for all, collect errors
- C: Configurable via `on_error` field: `"cancel_all"` | `"continue"` | `"ignore"`
**Current lean**: Option C — configurable, default `"cancel_all"`.

### OQ-6: Namespace / Module Namespace for Struct Disambiguation
**Phase**: 4.5b §5
**Problem**: Two different modules define struct with same name. E.g. `crm.Contact` vs `hrm.Contact`. How to disambiguate?
**Context**: In Java it's `com.company.crm.Contact`. In Go it's `crm.Contact` (package-scoped). In Python it's `from crm.models import Contact`.
**Sub-questions**:
- Is the import alias sufficient? (`{"import": {"crm": "./crm/types.json", "hrm": "./hrm/types.json"}}` → `crm.Contact` vs `hrm.Contact`)
- Or do we need explicit namespace declaration inside the JSON file itself? (`"namespace": "crm"`)
- What about deeply nested namespaces? `company.division.module.Type`?
- How does this interact with extensions? `ext:bitcode` already acts as a namespace.
**Current lean**: Import alias IS the namespace. `crm.Contact` and `hrm.Contact` are disambiguated by their import alias. No need for explicit namespace declaration inside files. But needs further thought for large projects with deep module hierarchies.

### OQ-7: Stdlib Function Call in Expressions — Chained / Namespaced
**Phase**: 4.5a §11, 4.5b §6
**Problem**: Can stdlib functions be called with namespace and chaining? E.g. `string.random(input.min, 20)` or `math.clamp(x, 0, 100)`.
**Sub-questions**:
- Are stdlib functions flat (`random(10, 20)`) or namespaced (`string.random(10, 20)`)?
- Can they be chained? `"hello".upper().trim()` or `upper(trim("hello"))`?
- If namespaced, does `string` conflict with the type conversion function `string(x)`?
- How does expr-lang handle this? (expr-lang supports method calls on types and custom functions, but not arbitrary namespacing)
**Current lean**: Flat by default (like Python built-ins: `len()`, `upper()`, `abs()`). Namespaced for disambiguation when needed (like `crypto.sha256()` vs potential user function `sha256()`). Method-style chaining NOT supported in Phase 4.5a — too complex for expr-lang integration. Revisit in later phase.

### OQ-8: Undefined/Untyped Variables — Dynamic Type or Compile Error?
**Phase**: 4.5a §4
**Problem**: What happens when a variable has no type annotation and receives values of different types at different points?
**Example**:
```json
{"let": "x", "value": 42},
{"set": "x", "value": "hello"}
```
**Sub-questions**:
- Is this allowed? (dynamic typing like `interface{}` in Go / `Object` in Java / `any` in TypeScript)
- Or is this a type error? (once `x` is `int`, it stays `int`)
- What about variables that receive external/unknown data? (`{"let": "data", "expr": "input.payload"}` where payload could be anything)
- How does `any` type interact with this? Is `any` explicit opt-in, or implicit default?
**Options**:
- A: **Strict after first assignment** — `let x = 42` locks `x` to `int`. `set x = "hello"` → type error. Use `any` explicitly for dynamic: `{"let": "x", "type": "any", "value": 42}`.
- B: **Always dynamic** — variables can hold any type at any time (like Python/JS). Type annotations are hints, not enforced.
- C: **Gradual** — untyped mode = dynamic (Option B). Typed mode (with input schema) = strict (Option A).
**Current lean**: Option C — gradual. In untyped mode, variables are dynamic. In typed mode (input schema declared), variables are strict after first assignment. This matches the gradual typing philosophy already in the design.

### OQ-9: Eval / Inline Code Execution in Other Languages
**Phase**: 4.5c
**Problem**: Should go-json support `eval` that executes code in Go, Python, or JavaScript?
**Example**:
```json
{"let": "result", "eval": "go", "code": "return fmt.Sprintf(\"%d items\", len(items))"}
{"let": "result", "eval": "js", "code": "return items.map(x => x.name).join(', ')"}
{"let": "result", "eval": "python", "code": "return sum(x['price'] for x in items)"}
```
**Sub-questions**:
- Does this break the "standalone" principle? (go-json would need yaegi/goja/python runtime as dependencies)
- Is this a core feature or an extension? (host can inject eval capability via `WithExtension`)
- Security implications? (eval is dangerous — code injection, sandbox escape)
- Performance? (spinning up a JS/Python VM per eval is expensive)
- Does this make code generation impossible? (eval blocks can't be transpiled)
**Options**:
- A: **No eval** — go-json is self-contained. If you need Go/JS/Python, use those runtimes directly via bitcode's script step.
- B: **Eval as extension** — not built-in, but host can inject: `gojson.WithExtension("eval", evalExtension)`. Bitcode could provide this since it already has yaegi/goja.
- C: **Built-in eval** — core feature with language selector.
**Current lean**: Option B — eval as extension, NOT built-in. Keeps go-json standalone and clean. Bitcode can provide eval capability by injecting yaegi/goja/python as extensions. This way go-json itself has zero dependency on any script runtime.

---

## 4. Architecture Overview

```
packages/go-json/
├── go.mod                    # github.com/bitcode-framework/go-json
├── lang/                     # Core language (Phase 4.5a)
│   ├── ast.go                # AST node types
│   ├── parser.go             # JSON → AST
│   ├── compiler.go           # AST → validated program
│   ├── vm.go                 # Tree-walk interpreter
│   ├── scope.go              # Variable scoping (block scope)
│   ├── types.go              # Type system (gradual)
│   └── errors.go             # Error types with position info
├── stdlib/                   # Built-in functions (Phase 4.5a + 4.5b)
│   ├── math.go               # 14 functions
│   ├── strings.go            # 20 functions
│   ├── arrays.go             # 20 functions
│   ├── types.go              # 6 type conversion functions
│   ├── maps.go               # 8 functions (Phase 4.5b)
│   ├── datetime.go           # 10 functions (Phase 4.5b)
│   ├── encoding.go           # 6 functions (Phase 4.5b)
│   ├── crypto.go             # 4 functions (Phase 4.5b)
│   └── fmt.go                # 2 functions (Phase 4.5b)
├── io/                       # I/O extensions (Phase 4.5c)
│   ├── http.go
│   ├── fs.go
│   ├── sql.go
│   └── exec.go
├── runtime/                  # Runtime configuration
│   ├── limits.go             # Resource limits
│   ├── context.go            # Execution context
│   └── hooks.go              # Extension hooks
├── cmd/                      # CLI (Phase 4.5c)
│   └── go-json/
│       └── main.go           # `go-json run program.json`
└── testdata/                 # Test programs
    ├── hello.json
    ├── factorial.json
    └── ...
```

### 4.1 How Bitcode Consumes go-json

```go
// In bitcode engine
import (
    gojson "github.com/bitcode-framework/go-json/lang"
    "github.com/bitcode-framework/go-json/stdlib"
)

rt := gojson.NewRuntime(
    gojson.WithStdlib(stdlib.All()),
    gojson.WithoutIO(),                              // disable raw I/O
    gojson.WithExtension("bitcode", bitcodebridge),   // inject bridge
    gojson.WithLimits(gojson.Limits{...}),
)

result, err := rt.Execute(programJSON, input)
```

### 4.2 How Others Use go-json Standalone

```go
import (
    gojson "github.com/bitcode-framework/go-json/lang"
    "github.com/bitcode-framework/go-json/stdlib"
    goio "github.com/bitcode-framework/go-json/io"
)

rt := gojson.NewRuntime(
    gojson.WithStdlib(stdlib.All()),
    gojson.WithIO(goio.All()),    // enable HTTP, FS, SQL, exec
)

result, err := rt.Execute(programJSON, input)
```

---

## 5. Phase Dependencies

```
Phase 4.5a (Core Language)
  │
  ├──► Phase 4.5b (Modularity) ──► Phase 4.5c (I/O + Integration)
  │                                       │
  └───────────────────────────────────────►│
                                           │
                                           ▼
                                    Phase 7 (Module "setting")
```

Phase 4.5a MUST complete before 4.5b.
Phase 4.5b MUST complete before 4.5c.
Phase 4.5c MUST complete before Phase 7 (go-json replaces current process engine).

---

## 6. Language Quick Reference

### 6.1 Program Structure

```json
{
  "name": "program_name",
  "import": { ... },
  "structs": { ... },
  "functions": { ... },
  "input": { ... },
  "steps": [ ... ]
}
```

All top-level keys are optional except `name`.
- File with `steps` = executable program
- File without `steps` = library (only structs + functions, importable)

### 6.2 Step Types

| Step | Phase | Purpose |
|---|---|---|
| `let` | 4.5a | Declare new variable |
| `set` | 4.5a | Update existing variable |
| `if`/`elif`/`else` | 4.5a | Conditional branching |
| `switch`/`cases` | 4.5a | Multi-way branching |
| `for`/`in` | 4.5a | Iterate over array |
| `for`/`range` | 4.5a | Iterate over number range |
| `while` | 4.5a | Conditional loop |
| `break` | 4.5a | Exit loop |
| `continue` | 4.5a | Skip to next iteration |
| `return` | 4.5a | Return value from function/program |
| `call` | 4.5a | Call function with input |
| `try`/`catch`/`finally` | 4.5a | Error handling |
| `error` | 4.5a | Throw error |
| `log` | 4.5a | Log message |
| `new` | 4.5b | Construct struct instance |
| `parallel` | 4.5b | Parallel execution |

### 6.3 Value Modes

| Mode | Syntax | Semantics |
|---|---|---|
| Literal | `"value": 42` | JSON value as-is, no evaluation |
| Expression | `"expr": "age + 1"` | Evaluated by expr-lang |
| Computed object | `"with": {"k": "expr"}` | Each field value is expression |

Only one of `value`/`expr`/`with` allowed per step. Multiple = compile error.

### 6.4 Type Vocabulary

| Type | Example | Note |
|---|---|---|
| `string` | `"hello"` | |
| `int` | `42` | |
| `float` | `3.14` | |
| `bool` | `true` | |
| `[]T` | `[]string`, `[]int` | Typed array |
| `[]any` | `[1, "two", true]` | Mixed array |
| `map` | `{"k": "v"}` | String-keyed map |
| `StructName` | `Person`, `Address` | User-defined struct |
| `?T` | `?string`, `?Person` | Nullable |
| `any` | anything | Opt-out of type checking |
