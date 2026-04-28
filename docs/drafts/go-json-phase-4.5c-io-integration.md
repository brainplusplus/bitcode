# Phase 4.5c — go-json I/O, Bitcode Integration, Code Generation (Draft)

**Status**: Draft
**Depends on**: Phase 4.5a (Core Language), Phase 4.5b (Modularity)
**Blocks**: Phase 7 (Module "setting")

---

## §1. I/O Modules (Standalone)

These modules provide side-effect capabilities for standalone go-json usage. Bitcode replaces these with its bridge API.

### 1.1 Module Overview

| Module | Functions | Purpose |
|---|---|---|
| `io.http` | 5 | HTTP client |
| `io.fs` | 6 | File system |
| `io.sql` | 2 | SQL database |
| `io.exec` | 1 | Command execution |
| `io.regex` | 3 | Regex operations |

### 1.2 HTTP Module

```json
// Available when io.http is enabled
{"let": "resp", "call": "http.get", "with": {
  "url": "'https://api.example.com/users'",
  "headers": "{'Authorization': 'Bearer ' + token}"
}}

{"let": "resp", "call": "http.post", "with": {
  "url": "'https://api.example.com/users'",
  "body": "{'name': name, 'email': email}",
  "headers": "{'Content-Type': 'application/json'}"
}}
```

**Functions:**

| Function | Input | Returns |
|---|---|---|
| `http.get(url, headers?, timeout?)` | URL + optional headers/timeout | `{status, body, headers}` |
| `http.post(url, body?, headers?, timeout?)` | URL + body + optional headers | `{status, body, headers}` |
| `http.put(url, body?, headers?, timeout?)` | Same as post | `{status, body, headers}` |
| `http.patch(url, body?, headers?, timeout?)` | Same as post | `{status, body, headers}` |
| `http.delete(url, headers?, timeout?)` | Same as get | `{status, body, headers}` |

**Response shape:**
```json
{
  "status": 200,
  "body": {"any": "parsed JSON or string"},
  "headers": {"Content-Type": "application/json"}
}
```

### 1.3 File System Module

```json
{"let": "content", "call": "fs.read", "with": {"path": "'data.json'"}}
{"call": "fs.write", "with": {"path": "'output.txt'", "content": "result"}}
{"let": "exists", "call": "fs.exists", "with": {"path": "'config.json'"}}
```

| Function | Input | Returns |
|---|---|---|
| `fs.read(path)` | File path | `string` (file content) |
| `fs.write(path, content)` | Path + content | `void` |
| `fs.exists(path)` | Path | `bool` |
| `fs.list(path)` | Directory path | `[]string` (file names) |
| `fs.mkdir(path)` | Directory path | `void` |
| `fs.remove(path)` | Path | `void` |

### 1.4 SQL Module

```json
{"let": "users", "call": "sql.query", "with": {
  "dsn": "'postgres://localhost/mydb'",
  "query": "'SELECT * FROM users WHERE age > ?'",
  "args": "[18]"
}}

{"let": "result", "call": "sql.execute", "with": {
  "dsn": "'postgres://localhost/mydb'",
  "query": "'UPDATE users SET active = true WHERE id = ?'",
  "args": "[userId]"
}}
```

| Function | Input | Returns |
|---|---|---|
| `sql.query(dsn, query, args?)` | DSN + SQL + params | `[]map` (rows) |
| `sql.execute(dsn, query, args?)` | DSN + SQL + params | `{rows_affected, last_insert_id}` |

### 1.5 Exec Module

```json
{"let": "result", "call": "exec.run", "with": {
  "cmd": "'pandoc'",
  "args": "['input.md', '-o', 'output.pdf']",
  "cwd": "'/tmp'",
  "timeout": "30000"
}}
```

| Function | Input | Returns |
|---|---|---|
| `exec.run(cmd, args?, cwd?, timeout?, env?)` | Command + options | `{exit_code, stdout, stderr}` |

### 1.6 Regex Module

```json
{"let": "valid", "expr": "regex.match(email, '^[a-z]+@[a-z]+\\.[a-z]+$')"}
{"let": "numbers", "expr": "regex.findAll(text, '\\d+')"}
{"let": "cleaned", "expr": "regex.replace(text, '\\s+', ' ')"}
```

| Function | Input | Returns |
|---|---|---|
| `regex.match(str, pattern)` | String + regex | `bool` |
| `regex.findAll(str, pattern)` | String + regex | `[]string` |
| `regex.replace(str, pattern, replacement)` | String + regex + replacement | `string` |

### 1.7 I/O Security — Enable/Disable

```go
// Standalone: enable all I/O
rt := gojson.NewRuntime(
    gojson.WithStdlib(stdlib.All()),
    gojson.WithIO(goio.All()),
)

// Selective: only HTTP, no FS/SQL/exec
rt := gojson.NewRuntime(
    gojson.WithStdlib(stdlib.All()),
    gojson.WithIO(goio.HTTP()),
)

// Locked down: no I/O at all
rt := gojson.NewRuntime(
    gojson.WithStdlib(stdlib.All()),
    gojson.WithoutIO(),
)
```

Host application controls which I/O modules are available. Programs that call disabled I/O → compile error: `"function 'http.get' not available (I/O disabled)"`.

---

## §2. Bitcode Bridge Integration

### 2.1 How Bitcode Uses go-json

Bitcode disables raw I/O and injects its bridge API as an extension:

```go
import (
    gojson "github.com/bitcode-framework/go-json/lang"
    "github.com/bitcode-framework/go-json/stdlib"
    "github.com/bitcode-framework/bitcode/internal/runtime/bridge"
)

func createGoJSONRuntime(bc *bridge.Context, limits gojson.Limits) *gojson.Runtime {
    return gojson.NewRuntime(
        gojson.WithStdlib(stdlib.All()),
        gojson.WithoutIO(),
        gojson.WithExtension("bitcode", buildBitcodeExtension(bc)),
        gojson.WithLimits(limits),
        gojson.WithSession(gojson.Session{
            UserID:   bc.Session().UserID,
            Locale:   bc.Session().Locale,
            TenantID: bc.Session().TenantID,
            Groups:   bc.Session().Groups,
        }),
    )
}

func buildBitcodeExtension(bc *bridge.Context) gojson.Extension {
    return gojson.Extension{
        Functions: map[string]any{
            "model":     func(name string) any { ... },
            "http.get":  func(url string, opts ...map[string]any) (any, error) { ... },
            "http.post": func(url string, opts ...map[string]any) (any, error) { ... },
            "cache.get": func(key string) (any, error) { ... },
            "cache.set": func(key string, val any, opts ...map[string]any) error { ... },
            "db.query":  func(sql string, args ...any) ([]map[string]any, error) { ... },
            "fs.read":   func(path string) (string, error) { ... },
            "env":       func(key string) (string, error) { ... },
            "config":    func(key string) any { ... },
            "log":       func(level, msg string, data ...map[string]any) { ... },
            "emit":      func(event string, data map[string]any) error { ... },
            "call":      func(process string, input map[string]any) (any, error) { ... },
            "t":         func(key string) string { ... },
            "exec":      func(cmd string, args []string, opts ...map[string]any) (any, error) { ... },
            "email.send":    func(opts map[string]any) error { ... },
            "notify.send":   func(opts map[string]any) error { ... },
            "storage.upload": func(opts map[string]any) (any, error) { ... },
            "security.permissions": func(model string) (any, error) { ... },
            "audit.log":     func(opts map[string]any) error { ... },
            "crypto.encrypt": func(plaintext string) (string, error) { ... },
            "execution.current": func() any { ... },
            "tx":            func(fn func() error) error { ... },
        },
    }
}
```

### 2.2 Extension API

```go
type Extension struct {
    Name      string
    Functions map[string]any    // function name → Go function
    Structs   map[string]any    // struct name → struct definition (future)
}

// Runtime option
func WithExtension(name string, ext Extension) Option
```

Extensions are accessed in programs via `ext:name`:

```json
{
  "import": {
    "bc": "ext:bitcode"
  },
  "steps": [
    {"let": "leads", "call": "bc.model('lead').search", "with": {
      "domain": "[['status', '=', 'new']]",
      "limit": 100
    }},
    {"call": "bc.log", "with": {
      "level": "'info'",
      "msg": "'Found ' + string(len(leads)) + ' leads'"
    }}
  ]
}
```

### 2.3 How Bitcode Replaces Raw I/O with Bridge

| Standalone I/O | Bitcode Bridge | Difference |
|---|---|---|
| `http.get(url)` | `bc.http.get(url)` | Bridge uses tls-client, has rate limiting |
| `fs.read(path)` | `bc.fs.read(path)` | Bridge enforces fs_allow/fs_deny |
| `sql.query(dsn, sql)` | `bc.db.query(sql)` | Bridge uses connection pool, tenant-scoped |
| `exec.run(cmd)` | `bc.exec(cmd)` | Bridge enforces exec_allow whitelist |

Programs written for bitcode use `ext:bitcode`. Programs written for standalone use raw I/O. The language is the same — only the I/O layer differs.

### 2.4 Bitcode Model Access Pattern

The `bc.model()` pattern needs special handling because it returns a proxy object with methods:

```json
// In bitcode context
{"let": "leads", "expr": "bc.model('lead').search({'domain': [['status', '=', 'new']]})"}
{"let": "lead", "expr": "bc.model('lead').get(leadId)"}
{"call": "bc.model('lead').write", "with": {"id": "leadId", "data": "{'status': 'processed'}"}}
```

This works because `bc.model('lead')` returns a Go object with methods, and expr-lang can call methods on Go objects.

---

## §3. Bitcode Process Engine Migration

### 3.1 Current Process Engine → go-json

Current process engine (executor/) will be replaced by go-json. Migration path:

| Current Step Type | go-json Equivalent |
|---|---|
| `validate` | `if` + `error` steps |
| `query` | `bc.model(name).search(opts)` |
| `create` | `bc.model(name).create(data)` |
| `update` | `bc.model(name).write(id, data)` |
| `delete` | `bc.model(name).delete(id)` |
| `if` | `if`/`elif`/`else` (same, but with real expression evaluator) |
| `switch` | `switch`/`cases` (same) |
| `loop` | `for`/`in` (same, but with break/continue) |
| `emit` | `bc.emit(event, data)` |
| `call` | `call` with isolated scope (improved) |
| `script` | `call` to external script (or inline go-json function) |
| `http` | `bc.http.get/post/...` (through bridge, not raw) |
| `assign` | `let`/`set` (improved with expressions) |
| `log` | `bc.log(level, msg)` |
| `upsert` | `bc.model(name).upsert(data, unique)` |
| `count` | `bc.model(name).count(opts)` |
| `sum` | `bc.model(name).sum(field, opts)` |

### 3.2 JSON Script Support in Bitcode

Bitcode's script step handler detects `.json` files and routes to go-json:

```json
// In bitcode process definition
{
  "type": "script",
  "script": "scripts/process_data.json",
  "runtime": "go-json"
}
```

Or auto-detected by extension:
```go
func detectRuntimeFromExtension(script string) string {
    switch {
    case strings.HasSuffix(script, ".js"):   return "javascript"
    case strings.HasSuffix(script, ".go"):   return "go"
    case strings.HasSuffix(script, ".ts"):   return "node"
    case strings.HasSuffix(script, ".py"):   return "python"
    case strings.HasSuffix(script, ".json"): return "go-json"  // NEW
    }
    return ""
}
```

### 3.3 Backward Compatibility

Current process JSON format (with `type: "query"`, `type: "create"`, etc.) continues to work. go-json is an ADDITIONAL runtime, not a replacement of the existing format.

Over time, new features will only be added to go-json format. Old format enters maintenance mode.

---

## §4. Code Generation Foundation

### 4.1 AST Export

go-json programs are parsed into a well-defined AST. This AST can be exported for code generation:

```go
program, err := gojson.Parse(jsonBytes)
ast := program.AST()

// Export as JSON (for external tools)
astJSON, _ := json.Marshal(ast)

// Export as Go code
goCode := codegen.ToGo(ast)

// Export as JavaScript
jsCode := codegen.ToJS(ast)
```

### 4.2 Code Generation Targets

| Target | Feasibility | Notes |
|---|---|---|
| Go | High | Direct mapping — structs, functions, control flow all map 1:1 |
| JavaScript | High | Most constructs map directly. Structs → classes. |
| Python | High | Most constructs map directly. Structs → dataclasses. |
| SQL | Medium | Only data-heavy programs (queries, transforms) |
| BPMN XML | Medium | Only workflow-style programs (steps, conditions, parallel) |

### 4.3 AST Node Types

```go
type NodeType string

const (
    NodeProgram    NodeType = "program"
    NodeLet        NodeType = "let"
    NodeSet        NodeType = "set"
    NodeIf         NodeType = "if"
    NodeSwitch     NodeType = "switch"
    NodeForIn      NodeType = "for_in"
    NodeForRange   NodeType = "for_range"
    NodeWhile      NodeType = "while"
    NodeBreak      NodeType = "break"
    NodeContinue   NodeType = "continue"
    NodeReturn     NodeType = "return"
    NodeCall       NodeType = "call"
    NodeTry        NodeType = "try"
    NodeError      NodeType = "error"
    NodeLog        NodeType = "log"
    NodeNew        NodeType = "new"
    NodeParallel   NodeType = "parallel"
    NodeFunction   NodeType = "function"
    NodeStruct     NodeType = "struct"
    NodeImport     NodeType = "import"
    NodeExpression NodeType = "expression"
)
```

### 4.4 Code Generation Example

**go-json source:**
```json
{
  "name": "factorial",
  "functions": {
    "factorial": {
      "params": {"n": "int"},
      "returns": "int",
      "steps": [
        {"if": "n <= 1", "then": [{"return": "1"}]},
        {"let": "sub", "call": "factorial", "with": {"n": "n - 1"}},
        {"return": "n * sub"}
      ]
    }
  },
  "steps": [
    {"let": "result", "call": "factorial", "with": {"n": "10"}},
    {"return": "result"}
  ]
}
```

**Generated Go:**
```go
package main

func factorial(n int) int {
    if n <= 1 {
        return 1
    }
    sub := factorial(n - 1)
    return n * sub
}

func main() {
    result := factorial(10)
    fmt.Println(result)
}
```

**Generated JavaScript:**
```javascript
function factorial(n) {
    if (n <= 1) return 1;
    const sub = factorial(n - 1);
    return n * sub;
}

const result = factorial(10);
console.log(result);
```

**Generated Python:**
```python
def factorial(n: int) -> int:
    if n <= 1:
        return 1
    sub = factorial(n - 1)
    return n * sub

result = factorial(10)
print(result)
```

### 4.5 Code Generation Limitations

| Limitation | Why |
|---|---|
| Dynamic types → generated code may need type assertions | go-json allows `any`, target languages may not |
| Extension calls (`ext:bitcode`) → not portable | Host-specific, cannot generate standalone code |
| Parallel → different concurrency models per language | Go: goroutines, JS: Promise.all, Python: asyncio |
| I/O calls → different libraries per language | HTTP client, FS API differ per language |

Code generation works best for **pure logic** (functions, control flow, data transformation). I/O-heavy programs need manual adaptation.

---

## §5. Standalone CLI

### 5.1 CLI Runner

```bash
# Run a program
go-json run program.json --input '{"name": "Alice"}'

# Run with input from file
go-json run program.json --input-file input.json

# Run with limits
go-json run program.json --timeout 60s --max-depth 500

# Validate (compile check, no execution)
go-json check program.json

# Export AST
go-json ast program.json --output ast.json

# Generate code
go-json codegen program.json --target go --output program.go
go-json codegen program.json --target js --output program.js
go-json codegen program.json --target python --output program.py
```

### 5.2 REPL (Future)

Interactive mode for experimentation:

```bash
go-json repl

> let x = 42
> x + 1
43
> let items = [1, 2, 3, 4, 5]
> filter(items, # > 3)
[4, 5]
> sum(items)
15
```

This is a nice-to-have, not a must for Phase 4.5c.

---

## §6. Implementation Tasks

| # | Task | Effort | Priority |
|---|---|---|---|
| **I/O Modules** | | | |
| 1 | HTTP module (get/post/put/patch/delete) | Medium | Must |
| 2 | FS module (read/write/exists/list/mkdir/remove) | Medium | Must |
| 3 | SQL module (query/execute) | Medium | Must |
| 4 | Exec module (run) | Small | Must |
| 5 | Regex module (match/findAll/replace) | Small | Must |
| 6 | I/O enable/disable mechanism | Small | Must |
| **Bitcode Integration** | | | |
| 7 | Extension API (WithExtension, Extension struct) | Medium | Must |
| 8 | Bitcode bridge adapter (bridge.Context → Extension) | Large | Must |
| 9 | Script handler: detect .json → route to go-json | Small | Must |
| 10 | Replace current process engine data steps with bridge calls | Large | Must |
| 11 | Backward compatibility layer for old process format | Medium | Must |
| **Code Generation** | | | |
| 12 | AST export (JSON serialization) | Medium | Must |
| 13 | Go code generator | Large | Should |
| 14 | JavaScript code generator | Large | Should |
| 15 | Python code generator | Large | Should |
| **CLI** | | | |
| 16 | `go-json run` command | Medium | Must |
| 17 | `go-json check` command (validate) | Small | Must |
| 18 | `go-json ast` command (export AST) | Small | Should |
| 19 | `go-json codegen` command | Medium | Should |
| **Tests** | | | |
| 20 | Tests: HTTP module | Medium | Must |
| 21 | Tests: FS module | Medium | Must |
| 22 | Tests: SQL module | Medium | Must |
| 23 | Tests: Extension API | Medium | Must |
| 24 | Tests: Bitcode bridge integration | Large | Must |
| 25 | Tests: .json script detection in bitcode | Small | Must |
| 26 | Tests: AST export | Medium | Must |
| 27 | Tests: Code generation (Go/JS/Python) | Large | Should |
| 28 | Tests: CLI commands | Medium | Must |
