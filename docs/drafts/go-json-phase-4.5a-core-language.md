# Phase 4.5a — go-json Core Language + Stdlib (Draft)

**Status**: Draft
**Depends on**: None (foundational)
**Blocked by**: Phase 4.5b, 4.5c, Phase 7

---

## §1. Package Structure

```
packages/go-json/
├── go.mod                    # module github.com/bitcode-framework/go-json
├── lang/
│   ├── ast.go                # AST node types (all step types)
│   ├── parser.go             # JSON → AST (validate structure, resolve types)
│   ├── compiler.go           # AST → validated Program (type check, limit check)
│   ├── vm.go                 # Tree-walk interpreter
│   ├── scope.go              # Variable scoping (block scope per if/loop/function)
│   ├── types.go              # Type system (gradual: untyped → schema → full)
│   ├── errors.go             # Error types with step position info
│   └── program.go            # Program struct (compiled, ready to run)
├── stdlib/
│   ├── registry.go           # Function registry + All() helper
│   ├── math.go               # 14 math functions
│   ├── strings.go            # 20 string functions
│   ├── arrays.go             # 20 array functions
│   └── types.go              # 6 type conversion functions
├── runtime/
│   ├── limits.go             # Resource limits struct + defaults
│   ├── context.go            # Execution context (session, metadata)
│   └── runtime.go            # NewRuntime(), Execute(), options
└── testdata/
    ├── hello.json
    ├── variables.json
    ├── control_flow.json
    ├── loops.json
    ├── functions.json
    ├── recursion.json
    ├── error_handling.json
    └── stdlib_test.json
```

---

## §2. Expression Engine — expr-lang/expr

go-json uses [expr-lang/expr](https://github.com/expr-lang/expr) as its expression evaluation engine. This is NOT a custom parser — it's a proven, production-grade library.

### 2.1 Why expr-lang

| Requirement | expr-lang |
|---|---|
| Arithmetic: `+`, `-`, `*`, `/`, `%`, `**` | ✅ |
| Comparison: `==`, `!=`, `<`, `>`, `<=`, `>=` | ✅ |
| Logical: `&&`, `\|\|`, `!`, ternary `?:` | ✅ |
| Nil coalescing: `??` | ✅ |
| String: `contains`, `startsWith`, `endsWith`, `matches` | ✅ |
| Array: `filter`, `map`, `reduce`, `find`, `all`, `any` | ✅ |
| Member access: `a.b.c`, `a[0]`, `a?.b` (optional chaining) | ✅ |
| Pipe: `value \| function` | ✅ |
| Type-safe | ✅ Compile-time type checking |
| Memory-safe | ✅ No buffer overflows |
| Terminating | ✅ No infinite loops in expressions |
| Bytecode VM | ✅ Compiled, fast |
| Custom functions | ✅ `expr.Function()` |
| Go-native | ✅ Pure Go, no CGO |

### 2.2 What expr-lang Does NOT Do (go-json fills the gap)

- Statements (let, set, if/else blocks, loops) → go-json VM
- Functions with multiple steps → go-json function system
- Side effects (I/O, DB) → go-json I/O modules
- Struct definitions → go-json type system
- Import/module system → go-json import system

### 2.3 Integration Point

Every `"expr"` field in a step is evaluated by expr-lang:

```go
// In go-json VM
program, err := expr.Compile(exprString, expr.Env(currentScope))
result, err := expr.Run(program, currentScope)
```

The scope (variables, functions) is passed as expr-lang environment.

---

## §3. Variable Declaration — `let` / `set`

### 3.1 `let` — Declare New Variable

```json
{"let": "name", "value": "Alice"}           // literal string
{"let": "age", "value": 30}                  // literal int
{"let": "scores", "value": [90, 85, 92]}     // literal array
{"let": "active", "value": true}             // literal bool
{"let": "config", "value": {"k": "v"}}       // literal object

{"let": "next_age", "expr": "age + 1"}       // expression
{"let": "greeting", "expr": "'Hello ' + name"}// expression with string concat
{"let": "first", "expr": "scores[0]"}        // expression with array access
{"let": "adult", "expr": "age >= 18"}        // expression with comparison

{"let": "profile", "with": {                  // computed object
  "name": "name",                             //   each value = expression
  "age": "age",
  "is_adult": "age >= 18"
}}
```

**Rules:**
- `let` declares a NEW variable. Error if variable already exists in current scope.
- Exactly one of `value`/`expr`/`with` required. Zero or multiple = compile error.
- Type is inferred from the assigned value.

### 3.2 `set` — Update Existing Variable

```json
{"set": "age", "value": 31}                  // literal
{"set": "age", "expr": "age + 1"}            // expression
{"set": "name", "expr": "upper(name)"}       // expression with function
```

**Rules:**
- `set` updates an EXISTING variable. Error if variable does not exist.
- Type must be compatible with original declaration. `let age = 30` then `set age = "hello"` → type error.

### 3.3 Nested Property Mutation

```json
{"set": "person.address.city", "expr": "'Bandung'"}
{"set": "items[0].name", "expr": "'Updated'"}
```

Dot notation and bracket notation supported for nested mutation.

### 3.4 Value Modes — Complete Rules

| Mode | Key | Semantics | Example |
|---|---|---|---|
| Literal | `value` | JSON value stored as-is. No evaluation. | `"value": 42`, `"value": "hello"`, `"value": [1,2]` |
| Expression | `expr` | String evaluated by expr-lang. Result stored. | `"expr": "age + 1"`, `"expr": "len(items)"` |
| Computed object | `with` | Object where each field's value is an expression string. | `"with": {"name": "input.name", "age": "input.age + 1"}` |

**Why three modes?**
- `value` is unambiguous — `"value": "Alice"` is always the string "Alice", never a variable lookup.
- `expr` is unambiguous — `"expr": "Alice"` is always a variable lookup for `Alice`.
- `with` is unambiguous — every field is an expression, no need for per-field prefix.

This eliminates the `"= "` prefix problem entirely.

---

## §4. Type System — Gradual Typing

### 4.1 Level 1: Untyped (Default)

No type annotations. Types inferred at runtime.

```json
{
  "name": "quick_script",
  "steps": [
    {"let": "x", "value": 42},
    {"let": "y", "expr": "x + 1"},
    {"return": "y"}
  ]
}
```

Type errors caught at runtime with clear messages:
```
TypeError at step 2: cannot add int(42) + string("hello")
```

### 4.2 Level 2: Input Schema

Input types declared. Compiler validates expressions against schema.

```json
{
  "name": "typed_script",
  "input": {
    "name": "string",
    "age": "int",
    "tags": "[]string",
    "metadata": "any"
  },
  "steps": [
    {"let": "greeting", "expr": "'Hello ' + input.name"},
    {"let": "next_year", "expr": "input.age + 1"},
    {"return": "greeting"}
  ]
}
```

Compiler knows `input.name` is `string`, `input.age` is `int`. Expression `input.name + 1` → compile error.

### 4.3 Level 3: Full Typing (with structs, Phase 4.5b)

All variables, function params, and return types declared.

```json
{
  "input": {"data": "Person"},
  "functions": {
    "greet": {
      "params": {"p": "Person"},
      "returns": "string",
      "steps": [{"return": "p.name + ' (' + string(p.age) + ')'"}]
    }
  }
}
```

### 4.4 Type Vocabulary

| Type | JSON representation | Go equivalent |
|---|---|---|
| `string` | `"string"` | `string` |
| `int` | `"int"` | `int64` |
| `float` | `"float"` | `float64` |
| `bool` | `"bool"` | `bool` |
| `[]T` | `"[]string"`, `"[]int"` | `[]T` |
| `[]any` | `"[]any"` | `[]any` |
| `map` | `"map"` | `map[string]any` |
| `any` | `"any"` | `any` |
| `?T` | `"?string"`, `"?int"` | `*T` (pointer, nullable) |
| Struct | `"Person"` | Generated struct |

### 4.5 Type Inference Rules

```
literal 42          → int
literal 3.14        → float
literal "hello"     → string
literal true        → bool
literal [1, 2, 3]   → []int
literal [1, "a"]    → []any
literal {"k": "v"}  → map
expr "age + 1"      → typeof(age)  (if age is int → int)
expr "name + ' hi'" → string
expr "len(items)"   → int
expr "items[0]"     → element type of items
```

---

## §5. Control Flow — `if` / `elif` / `else` / `switch`

### 5.1 If / Elif / Else

```json
{
  "if": "score >= 90",
  "then": [
    {"let": "grade", "value": "A"}
  ],
  "elif": [
    {
      "condition": "score >= 80",
      "then": [{"let": "grade", "value": "B"}]
    },
    {
      "condition": "score >= 70",
      "then": [{"let": "grade", "value": "C"}]
    }
  ],
  "else": [
    {"let": "grade", "value": "F"}
  ]
}
```

**Rules:**
- `if` value is an expression string evaluated to bool.
- `then` is required.
- `elif` is optional, array of `{condition, then}` objects.
- `else` is optional.
- Variables declared inside `then`/`else` are scoped to that block (block scope).

### 5.2 Switch

```json
{
  "switch": "status",
  "cases": {
    "active": [
      {"call": "process_active", "with": {"id": "id"}}
    ],
    "pending": [
      {"call": "process_pending", "with": {"id": "id"}}
    ],
    "default": [
      {"log": "'Unknown status: ' + status"}
    ]
  }
}
```

**Rules:**
- `switch` value is an expression evaluated to a value.
- `cases` keys are matched against the switch value (string comparison after `string()` coercion).
- `default` case is optional, executed if no match.
- No fallthrough (unlike C/Go). Each case is independent.

---

## §6. Loops — `for` / `while` / `range` / `break` / `continue`

### 6.1 For-each Loop

```json
{
  "for": "item",
  "in": "items",
  "index": "i",
  "steps": [
    {"log": "string(i) + ': ' + string(item)"}
  ]
}
```

- `for` — variable name for current element
- `in` — expression evaluating to array
- `index` — optional, variable name for current index (0-based)
- `steps` — loop body
- `item` and `i` are block-scoped to the loop

### 6.2 While Loop

```json
{
  "while": "count < 100",
  "steps": [
    {"set": "count", "expr": "count * 2"}
  ]
}
```

- `while` — expression evaluated to bool before each iteration
- Protected by `max_loop_iterations` limit (default 10000)

### 6.3 Range Loop

```json
{
  "for": "i",
  "range": [0, 10],
  "steps": [
    {"log": "'Iteration: ' + string(i)"}
  ]
}
```

- `range` — `[start, end]` or `[start, end, step]`
- `range: [0, 10]` → i = 0, 1, 2, ..., 9 (exclusive end)
- `range: [0, 10, 2]` → i = 0, 2, 4, 6, 8

### 6.4 Break / Continue

```json
{
  "for": "item", "in": "items",
  "steps": [
    {"if": "item.invalid", "then": [{"continue": true}]},
    {"if": "item.stop", "then": [{"break": true}]},
    {"call": "process_item", "with": {"data": "item"}}
  ]
}
```

- `break` — exit the innermost loop
- `continue` — skip to next iteration of innermost loop
- Using `break`/`continue` outside a loop → compile error

---

## §7. Functions

### 7.1 Function Definition

```json
{
  "functions": {
    "calculateDiscount": {
      "params": {
        "price": "float",
        "quantity": "int",
        "tier": "string"
      },
      "returns": "float",
      "steps": [
        {"let": "rate", "value": 0.0},
        {"if": "tier == 'gold'", "then": [
          {"set": "rate", "value": 0.15}
        ], "elif": [
          {"condition": "tier == 'silver'", "then": [
            {"set": "rate", "value": 0.10}
          ]}
        ], "else": [
          {"set": "rate", "value": 0.05}
        ]},
        {"return": "price * quantity * rate"}
      ]
    }
  }
}
```

### 7.2 Function Call — Step Level

```json
{"let": "discount", "call": "calculateDiscount", "with": {
  "price": "item.price",
  "quantity": "item.qty",
  "tier": "customer.tier"
}}
```

- `call` — function name (string)
- `with` — computed input object (each value = expression)
- `let` + `call` — store return value in variable
- `call` without `let` — execute for side effects, discard return

### 7.3 Function Call — Expression Level

For simple pure functions, callable directly in expressions:

```json
{"let": "total", "expr": "calculateDiscount(100.0, 5, 'gold')"}
{"if": "isValid(email)", "then": [...]}
```

**[OPEN: OQ-3]** Rule for when to use step-level vs expression-level:
- Expression-level: pure functions, simple args, no `with` needed
- Step-level: functions with complex computed inputs, need `with`

### 7.4 Recursion

```json
{
  "functions": {
    "factorial": {
      "params": {"n": "int"},
      "returns": "int",
      "steps": [
        {"if": "n <= 1", "then": [{"return": 1}]},
        {"let": "sub", "call": "factorial", "with": {"n": "n - 1"}},
        {"return": "n * sub"}
      ]
    }
  },
  "steps": [
    {"let": "result", "call": "factorial", "with": {"n": 10}},
    {"return": "result"}
  ]
}
```

**Recursion rules:**
- Each function call creates a new scope (isolated from parent).
- Depth counter is GLOBAL per execution (not per function). A→B→A counts as depth 3.
- Default max depth: 1000. Configurable per program. Hard limit: 10000.
- Infinite recursion → depth limit error with clear stack trace.

### 7.5 Function Scope Isolation

```json
{"let": "x", "value": 10},
{"call": "myFunc", "with": {"a": "x"}},
// x is still 10 here — myFunc cannot modify parent's x
```

Functions receive input via `with`. They CANNOT access parent scope variables directly. This makes functions:
- Testable independently
- Safe from side effects
- Predictable

---

## §8. Error Handling — `try` / `catch` / `finally` / `error`

### 8.1 Try / Catch / Finally

```json
{
  "try": [
    {"let": "data", "call": "fetchData", "with": {"url": "api_url"}},
    {"let": "parsed", "expr": "fromJSON(data)"}
  ],
  "catch": {
    "as": "err",
    "steps": [
      {"log": "'Failed: ' + err.message"},
      {"let": "parsed", "value": null}
    ]
  },
  "finally": [
    {"log": "'Fetch attempt completed'"}
  ]
}
```

**Rules:**
- `try` — array of steps. If any step throws, execution jumps to `catch`.
- `catch.as` — variable name for the error object.
- `catch.steps` — error handling steps.
- `finally` — optional, always executed (success or error).
- Error in `catch` → propagates to parent try/catch or program level.
- Error in `finally` → replaces original error.

### 8.2 Throw Error

```json
{"error": "'Invalid input: name is required'"}
```

**[OPEN: OQ-4]** Structured errors:

```json
// Simple (string)
{"error": "'something went wrong'"}

// Structured
{"error": {
  "code": "'VALIDATION_ERROR'",
  "message": "'Invalid email format'",
  "details": "validationErrors"
}}
```

### 8.3 Error Object Shape

```json
{
  "message": "string — human-readable error message",
  "code": "string — error code (optional, 'ERROR' if not set)",
  "step": "int — step index where error occurred",
  "function": "string — function name (if inside function)",
  "stack": ["string — call stack trace"]
}
```

---

## §9. Return Values

### 9.1 Return Expression

```json
{"return": "result"}
{"return": "age + 1"}
{"return": "nil"}
{"return": "true"}
{"return": "'hello'"}
{"return": "calculateTotal(items)"}
```

`return` value is always an expression string.

### 9.2 Return Computed Object

**[OPEN: OQ-1]** Current design — overloaded `return`:

```json
// Return expression
{"return": "candidate"}

// Return computed object — use "with" sub-key
{"return": {"with": {
  "status": "'eligible'",
  "person": "candidate",
  "count": "len(items)"
}}}

// Return literal object
{"return": {"value": {"status": "ok", "code": 200}}}
```

This follows the same `value`/`expr`/`with` pattern as `let`/`set`:
- `{"return": "expr"}` — shorthand for expression (most common)
- `{"return": {"expr": "..."}}` — explicit expression
- `{"return": {"value": ...}}` — literal value
- `{"return": {"with": {...}}}` — computed object

### 9.3 Return from Function vs Program

- Return inside function → returns to caller, function scope destroyed.
- Return inside top-level steps → ends program, value becomes program output.
- Return inside loop → exits loop AND function/program (like Go).
- Return inside if/else → exits function/program.

---

## §10. Resource Limits

### 10.1 Limit Types

```go
type Limits struct {
    MaxDepth          int           // max recursion/call depth. Default: 1000, hard: 10000
    MaxSteps          int           // max total step executions. Default: 10000, hard: 100000
    MaxLoopIterations int           // max iterations per single loop. Default: 10000, hard: 100000
    MaxVariables      int           // max variables in scope. Default: 1000
    MaxVariableSize   int           // max single variable size in bytes. Default: 10MB
    MaxOutputSize     int           // max program output size in bytes. Default: 50MB
    Timeout           time.Duration // max execution time. Default: 30s
}
```

### 10.2 Why These Limits (Not Raw Memory)

Per-execution memory limiting is not possible in Go (shared heap, no per-goroutine memory tracking). These limits are **observable proxies** for memory:

| Limit | What it prevents |
|---|---|
| `MaxSteps` | Runaway execution (each step allocates memory) |
| `MaxLoopIterations` | Infinite/huge loops |
| `MaxVariables` | Variable accumulation |
| `MaxVariableSize` | Single huge variable (e.g. 1GB query result) |
| `MaxDepth` | Stack overflow from recursion |
| `Timeout` | Wall-clock safety net |

### 10.3 Config Resolution Order

```
Engine hard limit (non-overridable)
  ↓
Project config (bitcode.toml / go-json.toml)
  ↓
Module config (module.json)
  ↓
Program config (in JSON program)
  ↓
Step config (per-step timeout)
```

**Resolution rule: always take the MOST RESTRICTIVE (minimum).**

If project says `max_depth: 1000` and program says `max_depth: 5000`, effective limit is `1000`. A more specific level can be MORE restrictive but NEVER less restrictive than its parent.

### 10.4 Program-Level Limits

```json
{
  "name": "heavy_process",
  "limits": {
    "max_depth": 100,
    "max_steps": 50000,
    "timeout": "120s"
  },
  "steps": [...]
}
```

### 10.5 Timeout Inheritance for Sub-calls

When process A (timeout 60s) calls function B at t=20s:
- B gets remaining timeout: 40s
- B cannot exceed parent's remaining time
- B's own timeout (if set) is capped to min(B.timeout, remaining)

---

## §11. Stdlib — Tier 1 (60 Functions)

All stdlib functions are pure (no side effects) and available in expressions.

### 11.1 Math (14 functions)

| Function | Signature | Description |
|---|---|---|
| `abs(x)` | `number → number` | Absolute value |
| `ceil(x)` | `number → int` | Round up |
| `floor(x)` | `number → int` | Round down |
| `round(x, precision?)` | `number, int? → number` | Round to precision |
| `min(args...)` | `...number → number` | Minimum value. Also accepts array. |
| `max(args...)` | `...number → number` | Maximum value. Also accepts array. |
| `sum(arr)` | `[]number → number` | Sum of array |
| `avg(arr)` | `[]number → float` | Average of array |
| `pow(base, exp)` | `number, number → number` | Power |
| `sqrt(x)` | `number → float` | Square root |
| `mod(a, b)` | `number, number → number` | Modulo |
| `clamp(x, min, max)` | `number, number, number → number` | Clamp to range |
| `randomInt(min, max)` | `int, int → int` | Random integer in range |
| `sign(x)` | `number → int` | -1, 0, or 1 |

### 11.2 String (20 functions)

| Function | Signature | Description |
|---|---|---|
| `len(s)` | `string → int` | String length (also works on arrays) |
| `upper(s)` | `string → string` | Uppercase |
| `lower(s)` | `string → string` | Lowercase |
| `trim(s)` | `string → string` | Trim whitespace |
| `trimLeft(s, chars?)` | `string, string? → string` | Trim left |
| `trimRight(s, chars?)` | `string, string? → string` | Trim right |
| `contains(s, sub)` | `string, string → bool` | Contains substring |
| `startsWith(s, prefix)` | `string, string → bool` | Starts with |
| `endsWith(s, suffix)` | `string, string → bool` | Ends with |
| `indexOf(s, sub)` | `string, string → int` | First index of substring (-1 if not found) |
| `lastIndexOf(s, sub)` | `string, string → int` | Last index of substring |
| `replace(s, old, new, n?)` | `string, string, string, int? → string` | Replace first n (default all) |
| `split(s, sep)` | `string, string → []string` | Split string |
| `join(arr, sep)` | `[]string, string → string` | Join array to string |
| `substring(s, start, end?)` | `string, int, int? → string` | Substring |
| `repeat(s, n)` | `string, int → string` | Repeat string |
| `padLeft(s, n, char?)` | `string, int, string? → string` | Pad left |
| `padRight(s, n, char?)` | `string, int, string? → string` | Pad right |
| `matches(s, pattern)` | `string, string → bool` | Regex match |
| `format(template, args...)` | `string, ...any → string` | String formatting (`%s`, `%d`, etc.) |

### 11.3 Array (20 functions)

| Function | Signature | Description |
|---|---|---|
| `len(arr)` | `[]any → int` | Array length |
| `first(arr)` | `[]any → any` | First element (nil if empty) |
| `last(arr)` | `[]any → any` | Last element (nil if empty) |
| `get(arr, i)` | `[]any, int → any` | Get by index (nil if out of bounds) |
| `append(arr, item)` | `[]any, any → []any` | Append (returns new array) |
| `prepend(arr, item)` | `[]any, any → []any` | Prepend (returns new array) |
| `concat(a, b)` | `[]any, []any → []any` | Concatenate arrays |
| `slice(arr, start, end?)` | `[]any, int, int? → []any` | Slice |
| `reverse(arr)` | `[]any → []any` | Reverse (returns new array) |
| `sort(arr)` | `[]any → []any` | Sort ascending |
| `sortBy(arr, field)` | `[]any, string → []any` | Sort by field |
| `unique(arr)` | `[]any → []any` | Remove duplicates |
| `flatten(arr)` | `[][]any → []any` | Flatten one level |
| `contains(arr, item)` | `[]any, any → bool` | Contains element |
| `indexOf(arr, item)` | `[]any, any → int` | Index of element (-1 if not found) |
| `filter(arr, pred)` | `[]any, predicate → []any` | Filter by predicate |
| `map(arr, pred)` | `[]any, predicate → []any` | Transform each element |
| `reduce(arr, pred, init)` | `[]any, predicate, any → any` | Reduce to single value |
| `find(arr, pred)` | `[]any, predicate → any` | Find first matching |
| `groupBy(arr, pred)` | `[]any, predicate → map` | Group by key |

Predicates use expr-lang lambda syntax: `filter(items, .age > 18)` or `map(items, .name)`.

### 11.4 Type Conversion (6 functions)

| Function | Signature | Description |
|---|---|---|
| `int(x)` | `any → int` | Convert to int |
| `float(x)` | `any → float` | Convert to float |
| `string(x)` | `any → string` | Convert to string |
| `bool(x)` | `any → bool` | Convert to bool |
| `type(x)` | `any → string` | Get type name |
| `isNil(x)` | `any → bool` | Check if nil |

---

## §12. Execution Context

### 12.1 Session (Implicit, Always Available)

```json
// Accessible via session.* in expressions
{
  "session": {
    "user_id": "string",
    "locale": "string",
    "tenant_id": "string",
    "groups": "[]string"
  }
}
```

Usage: `{"if": "session.user_id != nil", "then": [...]}`

Session is provided by the host application (bitcode passes user session, standalone apps pass custom session or empty).

### 12.2 Execution Metadata (Implicit)

```json
{
  "execution": {
    "id": "string — unique execution ID",
    "program": "string — program name",
    "started_at": "datetime",
    "depth": "int — current call depth",
    "step_count": "int — steps executed so far"
  }
}
```

Usage: `{"if": "execution.depth > 50", "then": [{"log": "'Deep recursion warning'"}]}`

### 12.3 Session is Immutable

Session cannot be modified during execution. It is read-only context from the host.

---

## §13. Variable Scoping

### 13.1 Block Scope

Variables declared inside `if`/`else`/`loop`/`function` are scoped to that block.

```json
[
  {"let": "x", "value": 10},
  {
    "if": "true",
    "then": [
      {"let": "y", "value": 20},
      {"log": "string(x + y)"}
    ]
  },
  {"log": "string(y)"}
]
```

Last step → **runtime error**: `variable 'y' not defined` (y was scoped to if-then block).

### 13.2 Outer Variable Access

Inner blocks CAN READ outer variables:

```json
[
  {"let": "x", "value": 10},
  {"if": "true", "then": [
    {"let": "y", "expr": "x + 5"}
  ]}
]
```

This works — `x` is accessible from inner block.

### 13.3 Outer Variable Mutation

Inner blocks CAN MUTATE outer variables via `set`:

```json
[
  {"let": "total", "value": 0},
  {"for": "item", "in": "items", "steps": [
    {"set": "total", "expr": "total + item.price"}
  ]},
  {"return": "total"}
]
```

This works — `set` looks up the scope chain to find `total`.

### 13.4 Function Scope Isolation

Functions do NOT have access to caller's scope:

```json
[
  {"let": "x", "value": 10},
  {"let": "result", "call": "myFunc", "with": {"a": "x"}}
]
```

Inside `myFunc`, only `a` (from input) is available. `x` is NOT accessible. This is intentional — functions are isolated units.

### 13.5 Loop Variable Scoping

Each loop iteration gets a fresh scope for `item`/`index`:

```json
{
  "for": "item", "in": "items",
  "steps": [
    {"let": "processed", "expr": "transform(item)"}
  ]
}
```

`processed` is re-declared each iteration (block scope). No pollution between iterations.

---

## §14. Edge Cases

### 14.1 Execution

| Edge Case | Behavior |
|---|---|
| Empty steps array | Program returns nil |
| Step with unknown type | Compile error: "unknown step type 'xyz'" |
| Expression syntax error | Compile error with position info |
| Division by zero | Runtime error: "division by zero at step N" |
| Array index out of bounds | `get()` returns nil. Direct `arr[i]` → runtime error. |
| Nil property access | `a.b` where a is nil → runtime error. Use `a?.b` for safe access. |
| Variable name collision with stdlib | Compile error: "variable 'len' shadows built-in function" |

### 14.2 Limits

| Edge Case | Behavior |
|---|---|
| MaxSteps exceeded | Runtime error: "step limit (10000) exceeded" |
| MaxDepth exceeded | Runtime error: "call depth limit (1000) exceeded at function 'X'" |
| MaxLoopIterations exceeded | Runtime error: "loop iteration limit (10000) exceeded" |
| Timeout exceeded | Runtime error: "execution timeout (30s) exceeded at step N" |
| MaxVariableSize exceeded | Runtime error: "variable 'X' exceeds size limit (10MB)" |

### 14.3 Type Errors

| Edge Case | Behavior |
|---|---|
| `let` variable already exists | Compile/runtime error: "variable 'x' already declared" |
| `set` variable doesn't exist | Runtime error: "variable 'x' not defined" |
| Type mismatch on `set` | Runtime error: "cannot assign string to variable 'age' (type int)" |
| Wrong argument type to function | Compile error (if typed) or runtime error |
| Wrong number of arguments | Compile error: "function 'X' expects 3 arguments, got 2" |

---

## §15. Implementation Tasks

| # | Task | Effort | Priority |
|---|---|---|---|
| 1 | Create `packages/go-json/` with go.mod | Small | Must |
| 2 | Define AST node types (`lang/ast.go`) | Medium | Must |
| 3 | JSON parser → AST (`lang/parser.go`) | Large | Must |
| 4 | Expression integration with expr-lang (`lang/compiler.go`) | Large | Must |
| 5 | Tree-walk VM (`lang/vm.go`) | Large | Must |
| 6 | Variable scoping (`lang/scope.go`) | Medium | Must |
| 7 | Type system — inference + gradual (`lang/types.go`) | Medium | Must |
| 8 | Error types with position info (`lang/errors.go`) | Small | Must |
| 9 | Resource limits (`runtime/limits.go`) | Medium | Must |
| 10 | Runtime API — NewRuntime, Execute, options (`runtime/runtime.go`) | Medium | Must |
| 11 | Execution context — session, metadata (`runtime/context.go`) | Small | Must |
| 12 | Stdlib: math (14 functions) | Medium | Must |
| 13 | Stdlib: strings (20 functions) | Medium | Must |
| 14 | Stdlib: arrays (20 functions) | Large | Must |
| 15 | Stdlib: type conversion (6 functions) | Small | Must |
| 16 | Tests: variable let/set/scoping | Medium | Must |
| 17 | Tests: control flow (if/elif/else/switch) | Medium | Must |
| 18 | Tests: loops (for/while/range/break/continue) | Medium | Must |
| 19 | Tests: functions + recursion | Medium | Must |
| 20 | Tests: error handling (try/catch/finally) | Medium | Must |
| 21 | Tests: resource limits | Medium | Must |
| 22 | Tests: stdlib functions | Large | Must |
| 23 | Tests: edge cases (type errors, nil, overflow) | Medium | Must |
