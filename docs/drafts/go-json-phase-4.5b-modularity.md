# Phase 4.5b — go-json Modularity: Struct + Import (Draft)

**Status**: Draft
**Depends on**: Phase 4.5a (Core Language)
**Blocks**: Phase 4.5c

---

## §1. Struct Definition

### 1.1 Basic Struct

```json
{
  "structs": {
    "Address": {
      "fields": {
        "street": "string",
        "city": "string",
        "zip": "string"
      }
    }
  }
}
```

### 1.2 Fields with Defaults

```json
{
  "structs": {
    "Address": {
      "fields": {
        "street": "string",
        "city": "string",
        "zip": "string",
        "country": {"type": "string", "default": "'ID'"}
      }
    }
  }
}
```

Field definition formats:
- Short: `"field_name": "type"` — no default, required on construction
- Long: `"field_name": {"type": "T", "default": "expr"}` — has default, optional on construction

### 1.3 Nested Structs

```json
{
  "structs": {
    "Address": {
      "fields": {
        "street": "string",
        "city": "string"
      }
    },
    "Person": {
      "fields": {
        "name": "string",
        "age": "int",
        "address": "Address",
        "tags": "[]string"
      }
    }
  }
}
```

`"address": "Address"` references the `Address` struct defined in the same file or imported.

### 1.4 Nullable Struct Fields

```json
{
  "fields": {
    "name": "string",
    "nickname": "?string",
    "address": "?Address"
  }
}
```

`?T` means the field can be `nil`. Non-nullable fields MUST be provided on construction.

### 1.5 Array of Structs

```json
{
  "fields": {
    "addresses": "[]Address",
    "scores": "[]int"
  }
}
```

---

## §2. Struct Methods

### 2.1 Method Definition

```json
{
  "structs": {
    "Person": {
      "fields": {
        "name": "string",
        "age": "int"
      },
      "methods": {
        "fullInfo": {
          "returns": "string",
          "steps": [
            {"return": "self.name + ' (' + string(self.age) + ')'"}
          ]
        },
        "isAdult": {
          "returns": "bool",
          "steps": [
            {"return": "self.age >= 18"}
          ]
        }
      }
    }
  }
}
```

- `self` is an implicit variable referencing the struct instance.
- Methods can access all fields via `self.fieldName`.
- Methods can call other methods on the same struct via `self.methodName()`.

### 2.2 Methods with Parameters

```json
{
  "methods": {
    "greet": {
      "params": {"greeting": "string"},
      "returns": "string",
      "steps": [
        {"return": "greeting + ', ' + self.name + '!'"}
      ]
    }
  }
}
```

### 2.3 Mutation Methods

**[OPEN: OQ-2]** Current design: mutable by default (like Go).

```json
{
  "methods": {
    "birthday": {
      "steps": [
        {"set": "self.age", "expr": "self.age + 1"}
      ]
    }
  }
}
```

Calling `person.birthday()` modifies the `person` instance in-place.

### 2.4 Method Returning New Instance

For immutable patterns, methods can return new struct:

```json
{
  "methods": {
    "withAge": {
      "params": {"newAge": "int"},
      "returns": "Person",
      "steps": [
        {"return": {"new": "Person", "with": {
          "name": "self.name",
          "age": "newAge"
        }}}
      ]
    }
  }
}
```

### 2.5 No Inheritance

go-json structs do NOT support inheritance. No `extends`, no `super`.

Composition is the pattern:

```json
{
  "structs": {
    "Employee": {
      "fields": {
        "person": "Person",
        "department": "string",
        "salary": "float"
      },
      "methods": {
        "fullInfo": {
          "returns": "string",
          "steps": [
            {"return": "self.person.fullInfo() + ' - ' + self.department"}
          ]
        }
      }
    }
  }
}
```

---

## §3. Struct Construction — `new` + `with`

### 3.1 Basic Construction

```json
{"let": "addr", "new": "Address", "with": {
  "street": "'Jl. Sudirman No. 1'",
  "city": "'Jakarta'",
  "zip": "'10110'"
}}
```

- `new` — struct type name
- `with` — computed object, each field value is an expression
- Fields not in `with` must have defaults, otherwise → compile error

### 3.2 Construction with Defaults

```json
{"let": "addr", "new": "Address", "with": {
  "street": "'Jl. Sudirman'",
  "city": "'Jakarta'",
  "zip": "'10110'"
}}
// addr.country == "ID" (from default)
```

### 3.3 Construction from Variables

```json
{"let": "person", "new": "Person", "with": {
  "name": "input.name",
  "age": "input.age",
  "address": "existingAddress",
  "tags": "input.tags ?? []"
}}
```

### 3.4 Nested Construction

```json
{"let": "person", "new": "Person", "with": {
  "name": "'Alice'",
  "age": 30,
  "address": {"new": "Address", "with": {
    "street": "'Jl. Sudirman'",
    "city": "'Jakarta'",
    "zip": "'10110'"
  }},
  "tags": "['developer']"
}}
```

Wait — `"age": 30` inside `with` is a problem. `with` values are expressions, but `30` is a JSON number, not a string expression.

**Rule clarification**: Inside `with`, values can be:
- String → treated as expression: `"age + 1"`, `"'hello'"`, `"input.name"`
- Number/bool/null → treated as literal: `30`, `true`, `null`
- Array → treated as literal: `[1, 2, 3]`
- Object with `new` → nested struct construction
- Object without `new` → treated as literal object

This is pragmatic — forcing `"age": "30"` (string containing number) would be confusing.

### 3.5 Type Validation on Construction

```json
// Person.age is int
{"let": "p", "new": "Person", "with": {
  "name": "'Alice'",
  "age": "'thirty'"    // ← compile error: expected int, got string expression
}}
```

---

## §4. Nested Property Access + Mutation

### 4.1 Reading Nested Properties

```json
{"let": "city", "expr": "person.address.city"}
{"let": "first_tag", "expr": "person.tags[0]"}
{"let": "zip", "expr": "person.address?.zip"}
```

- Dot notation: `a.b.c`
- Bracket notation: `a[0]`, `a["key"]`
- Optional chaining: `a?.b` (returns nil if `a` is nil, no error)

### 4.2 Mutating Nested Properties

```json
{"set": "person.address.city", "expr": "'Bandung'"}
{"set": "person.tags[0]", "expr": "'senior-developer'"}
```

`set` with dot/bracket notation traverses the object and mutates the leaf.

### 4.3 Edge Cases

| Case | Behavior |
|---|---|
| `set "a.b.c"` where `a.b` is nil | Runtime error: "cannot set property 'c' on nil" |
| `set "a[5]"` where array has 3 elements | Runtime error: "index 5 out of bounds (len 3)" |
| `set "a.b"` where `a` is not struct/map | Runtime error: "cannot set property on int" |
| `expr "a?.b.c"` where `a` is nil | Returns nil (optional chaining stops at nil) |

---

## §5. Import System

### 5.1 Import Syntax

```json
{
  "import": {
    "alias": "path"
  }
}
```

- `alias` — local name used to reference imported items
- `path` — where to find the file

### 5.2 Path Types

| Path Format | Resolves To | Example |
|---|---|---|
| `"./file.json"` | Relative to current file | `"./validators.json"` |
| `"../dir/file.json"` | Relative parent | `"../shared/types.json"` |
| `"stdlib:name"` | Built-in stdlib module | `"stdlib:math"` |
| `"ext:name"` | Host-injected extension | `"ext:bitcode"` |

### 5.3 Import Resolution Rules

1. Parse import path
2. If `stdlib:` → load from built-in registry
3. If `ext:` → load from host-injected extensions
4. If relative path → resolve against current file's directory
5. Read and parse the target JSON file
6. Extract exportable items (structs + functions, NOT steps)
7. Register under the alias in current scope

### 5.4 What Gets Exported

| File contains | Exported | Not exported |
|---|---|---|
| `structs` | ✅ All struct definitions | |
| `functions` | ✅ All function definitions | |
| `steps` | | ❌ Entry point, not exportable |
| `import` | | ❌ Transitive imports not re-exported |
| `input` | | ❌ Program-specific |
| `limits` | | ❌ Program-specific |

### 5.5 Using Imported Items

```json
{
  "import": {
    "models": "../types/person.json",
    "v": "../utils/validators.json"
  },
  "steps": [
    {"let": "person", "new": "models.Person", "with": {
      "name": "input.name",
      "age": "input.age"
    }},
    {"let": "valid", "call": "v.isEmail", "with": {
      "value": "input.email"
    }},
    {"let": "info", "expr": "person.fullInfo()"}
  ]
}
```

Imported items are accessed via `alias.ItemName`:
- `models.Person` — struct from models
- `models.Address` — another struct from models
- `v.isEmail` — function from validators
- `v.ValidationResult` — struct from validators

### 5.6 Circular Import Detection

```
a.json imports b.json
b.json imports a.json
→ Compile error: "circular import detected: a.json → b.json → a.json"
```

Detection: maintain import stack during compilation. If file appears twice → error.

### 5.7 Re-export (Barrel Files)

```json
{
  "name": "types_index",
  "import": {
    "_addr": "./address.json",
    "_person": "./person.json"
  },
  "structs": {
    "Address": {"alias": "_addr.Address"},
    "Person": {"alias": "_person.Person"}
  }
}
```

This allows:
```json
{"import": {"types": "./types/index.json"}}
// Use: types.Address, types.Person
```

Instead of importing each file separately.

### 5.8 Import Edge Cases

| Case | Behavior |
|---|---|
| File not found | Compile error: "import './foo.json' not found" |
| File has JSON syntax error | Compile error: "import './foo.json' parse error: ..." |
| Imported file has compile error | Compile error propagated with import chain |
| Alias collision | Compile error: "import alias 'models' already defined" |
| Imported struct references unknown type | Compile error in imported file |
| Diamond import (A→B, A→C, B→D, C→D) | D loaded once, shared. No duplication. |

---

## §6. Stdlib — Tier 2 (30 Functions)

### 6.1 Map/Object (8 functions)

| Function | Signature | Description |
|---|---|---|
| `keys(obj)` | `map → []string` | Get all keys |
| `values(obj)` | `map → []any` | Get all values |
| `entries(obj)` | `map → [][]any` | Get [key, value] pairs |
| `fromEntries(arr)` | `[][]any → map` | Create map from pairs |
| `has(obj, key)` | `map, string → bool` | Check key exists |
| `get(obj, path)` | `map, string → any` | Get by dot path: `get(obj, "a.b.c")` |
| `merge(a, b)` | `map, map → map` | Shallow merge (b overrides a) |
| `pick(obj, keys)` | `map, []string → map` | Pick subset of keys |

### 6.2 DateTime (10 functions)

| Function | Signature | Description |
|---|---|---|
| `now()` | `→ datetime` | Current datetime |
| `date(str, format?)` | `string, string? → datetime` | Parse date string |
| `formatDate(dt, format)` | `datetime, string → string` | Format datetime |
| `year(dt)` | `datetime → int` | Extract year |
| `month(dt)` | `datetime → int` | Extract month (1-12) |
| `day(dt)` | `datetime → int` | Extract day |
| `hour(dt)` | `datetime → int` | Extract hour |
| `minute(dt)` | `datetime → int` | Extract minute |
| `addDuration(dt, dur)` | `datetime, string → datetime` | Add duration: `"2h30m"`, `"7d"` |
| `diffDates(a, b)` | `datetime, datetime → duration` | Difference between dates |

### 6.3 Encoding (6 functions)

| Function | Signature | Description |
|---|---|---|
| `toJSON(val)` | `any → string` | Serialize to JSON string |
| `fromJSON(str)` | `string → any` | Parse JSON string |
| `toBase64(str)` | `string → string` | Base64 encode |
| `fromBase64(str)` | `string → string` | Base64 decode |
| `urlEncode(str)` | `string → string` | URL encode |
| `urlDecode(str)` | `string → string` | URL decode |

### 6.4 Crypto (4 functions)

| Function | Signature | Description |
|---|---|---|
| `md5(str)` | `string → string` | MD5 hash (hex) |
| `sha256(str)` | `string → string` | SHA-256 hash (hex) |
| `uuid()` | `→ string` | Generate UUID v4 |
| `hmac(str, key, algo?)` | `string, string, string? → string` | HMAC (default SHA-256) |

### 6.5 Format (2 functions)

| Function | Signature | Description |
|---|---|---|
| `sprintf(fmt, args...)` | `string, ...any → string` | Printf-style formatting |
| `printf(fmt, args...)` | `string, ...any → void` | Print formatted (debug) |

---

## §7. Parallel Execution

### 7.1 Parallel Step

```json
{
  "parallel": {
    "api_data": [
      {"let": "resp", "call": "http.get", "with": {"url": "'https://api1.com'"}},
      {"return": "resp.body"}
    ],
    "db_data": [
      {"let": "rows", "call": "sql.query", "with": {"query": "'SELECT * FROM users'"}},
      {"return": "rows"}
    ],
    "cache_data": [
      {"let": "cached", "call": "cache.get", "with": {"key": "'user_list'"}},
      {"return": "cached"}
    ]
  },
  "join": "all",
  "into": "results"
}
```

- `parallel` — object where each key is a branch name, value is array of steps
- `join` — how to wait: `"all"` (default), `"any"` (first to complete), `"settled"` (all, ignore errors)
- `into` — variable name for results map: `{"api_data": ..., "db_data": ..., "cache_data": ...}`

### 7.2 Branch Scope Isolation

Each branch gets its own scope. Branches CANNOT access each other's variables. After join, only the return values are available via `into`.

### 7.3 Error Handling in Parallel

**[OPEN: OQ-5]** Current design: configurable via `on_error`.

```json
{
  "parallel": { ... },
  "join": "all",
  "on_error": "cancel_all",
  "into": "results"
}
```

| `on_error` | Behavior |
|---|---|
| `"cancel_all"` (default) | First branch error cancels all others. Error propagated. |
| `"continue"` | Other branches continue. Failed branch result = error object. |
| `"ignore"` | Errors silently ignored. Failed branch result = nil. |

### 7.4 Timeout in Parallel

All branches share the parent's remaining timeout. If parent has 30s remaining, all branches must complete within 30s total.

---

## §8. Nullable Types

### 8.1 Declaration

```json
{"let": "name", "value": null}              // type: ?any
{"let": "name", "expr": "input.name"}       // type: ?string (if input.name could be nil)
```

### 8.2 Nil Checking

```json
{"if": "name != nil", "then": [
  {"log": "'Name: ' + name"}
]}

{"let": "display", "expr": "name ?? 'Anonymous'"}
```

### 8.3 Optional Chaining

```json
{"let": "city", "expr": "person?.address?.city ?? 'Unknown'"}
```

If `person` is nil → returns nil (no error).
If `person.address` is nil → returns nil (no error).
`?? 'Unknown'` provides fallback.

---

## §9. Implementation Tasks

| # | Task | Effort | Priority |
|---|---|---|---|
| 1 | Struct definition parser (fields, defaults, nested) | Medium | Must |
| 2 | Struct type registration in type system | Medium | Must |
| 3 | Struct construction (`new` + `with`) | Medium | Must |
| 4 | Struct field access (dot notation) | Medium | Must |
| 5 | Struct method definition + `self` binding | Large | Must |
| 6 | Struct method invocation (expression + step level) | Medium | Must |
| 7 | Nested property mutation (`set "a.b.c"`) | Medium | Must |
| 8 | Import system — file resolution | Medium | Must |
| 9 | Import system — struct/function extraction | Medium | Must |
| 10 | Import system — alias scoping | Medium | Must |
| 11 | Circular import detection | Small | Must |
| 12 | Re-export / barrel files | Small | Should |
| 13 | Stdlib: map functions (8) | Medium | Must |
| 14 | Stdlib: datetime functions (10) | Medium | Must |
| 15 | Stdlib: encoding functions (6) | Small | Must |
| 16 | Stdlib: crypto functions (4) | Small | Must |
| 17 | Stdlib: format functions (2) | Small | Must |
| 18 | Parallel execution engine | Large | Must |
| 19 | Parallel scope isolation | Medium | Must |
| 20 | Parallel error handling (on_error modes) | Medium | Must |
| 21 | Nullable type support + optional chaining | Medium | Must |
| 22 | Tests: struct CRUD (create, read, update fields) | Medium | Must |
| 23 | Tests: struct methods + self | Medium | Must |
| 24 | Tests: import system (relative, stdlib, circular) | Large | Must |
| 25 | Tests: parallel execution | Medium | Must |
| 26 | Tests: nullable + optional chaining | Medium | Must |
| 27 | Tests: stdlib tier 2 functions | Large | Must |
