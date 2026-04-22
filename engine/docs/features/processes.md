# Processes

Processes define business logic as a sequence of steps. Each step has a type and parameters.

## Example

```json
{
  "name": "confirm_order",
  "steps": [
    { "type": "validate", "rules": { "status": { "eq": "draft" } }, "error": "Only draft orders can be confirmed" },
    { "type": "update", "model": "order", "set": { "status": "confirmed" } },
    { "type": "if", "condition": "{{input.total > 10000}}", "then": "notify_manager", "else": "skip" },
    { "type": "log", "message": "Order {{input.id}} confirmed" },
    { "type": "emit", "event": "order.confirmed" }
  ]
}
```

## Step Types

### Data Steps

| Type | Description | Key Fields |
|------|-------------|------------|
| `validate` | Check conditions, fail with error | `rules`, `error` |
| `query` | Read from database | `model`, `domain`, `into` |
| `create` | Create a record | `model`, `set` |
| `update` | Update current record | `model`, `set` |
| `delete` | Delete a record | `model` |

### Control Flow Steps

| Type | Description | Key Fields |
|------|-------------|------------|
| `if` | Conditional branch | `condition`, `then_steps`, `else_steps` |
| `switch` | Multi-way branch | `field`, `case_steps` |
| `loop` | Iterate over list | `over`, `steps` |

### Integration Steps

| Type | Description | Key Fields |
|------|-------------|------------|
| `emit` | Publish domain event | `event`, `data` |
| `call` | Invoke another process | `process` |
| `script` | Run TypeScript/Python | `runtime`, `script` |
| `http` | Call external API | `url`, `method`, `headers`, `body` |

### Utility Steps

| Type | Description | Key Fields |
|------|-------------|------------|
| `assign` | Set a variable | `variable`, `value` |
| `log` | Write audit log | `message` |

## Variable Interpolation

Use `{{...}}` to reference values:

- `{{input.field}}` — from request input
- `{{user_id}}` — current user ID
- `{{variable_name}}` — from assigned variables
- `{{result.field}}` — from previous step result

## Control Flow Examples

### if with nested steps

```json
{
  "type": "if",
  "condition": "{{input.expected_revenue > 50000}}",
  "then_steps": [
    { "type": "assign", "variable": "deal_type", "value": "enterprise" },
    { "type": "log", "message": "High-value lead detected" }
  ],
  "else_steps": [
    { "type": "assign", "variable": "deal_type", "value": "standard" }
  ]
}
```

### switch with case_steps

```json
{
  "type": "switch",
  "field": "{{priority}}",
  "case_steps": {
    "high": [
      { "type": "assign", "variable": "handler", "value": "senior" }
    ],
    "low": [
      { "type": "assign", "variable": "handler", "value": "junior" }
    ],
    "default": [
      { "type": "assign", "variable": "handler", "value": "auto" }
    ]
  }
}
```

### loop with sub-steps

```json
{
  "type": "loop",
  "over": "{{order_lines}}",
  "steps": [
    { "type": "update", "model": "stock", "set": { "quantity": "{{_item.qty}}" } },
    { "type": "log", "message": "Updated stock for item {{_index}}" }
  ]
}
```

Inside loops, `{{_index}}` is the 0-based iteration index and `{{_item}}` is the current element.

### call (sub-process)

```json
{ "type": "call", "process": "validate_inventory" }
```

Loads and executes another process by name. Variables and events merge back into the parent context.

## Nesting Depth

Control flow steps can be nested up to 10 levels deep. Exceeding this limit returns an error.

## Backward Compatibility

The old `then`/`else` string format and `cases` map format are still parsed but only set a `_goto` variable. Use `then_steps`/`else_steps` and `case_steps` for actual branching execution.

## Validate Rules

```json
{ "type": "validate", "rules": { "status": { "eq": "draft" } }, "error": "Must be draft" }
{ "type": "validate", "rules": { "email": { "required": true } }, "error": "Email required" }
{ "type": "validate", "rules": { "status": { "neq": "cancelled" } }, "error": "Cannot be cancelled" }
```

## Internationalization (i18n)

Process steps support translation via three mechanisms:

### 1. `{{t('key')}}` function in any string

Use `{{t('key')}}` inside any interpolated string (error messages, log messages, etc.):

```json
{ "type": "log", "message": "{{t('greeting')}}, {{input.name}}!" }
{ "type": "validate", "rules": { "status": { "eq": "draft" } }, "error": "{{t('process.error.draft_only')}}" }
```

The key is looked up in the module's `i18n/*.json` files using the current locale.

### 2. Auto-translate error messages

Validate step error messages are automatically looked up as translation keys. If a translation exists, it's used:

```json
{ "type": "validate", "rules": { "status": { "eq": "draft" } }, "error": "Only draft orders can be confirmed" }
```

In `i18n/id.json`:
```json
{
  "locale": "id",
  "translations": {
    "Only draft orders can be confirmed": "Hanya pesanan draf yang bisa dikonfirmasi"
  }
}
```

When locale is `id`, the error returns `"Hanya pesanan draf yang bisa dikonfirmasi"`.

### 3. Step labels for UI display

Each step can have a `label` field for display in admin UI or process visualization:

```json
{
  "type": "validate",
  "label": "Check Status",
  "rules": { "status": { "eq": "draft" } },
  "error": "Must be draft"
}
```

Labels can be translated via the same i18n mechanism by adding them as translation keys.

### Locale in Context

The process execution context includes `Locale` and `Translator`. When a process is triggered via API, the locale can be set from the request (e.g., `Accept-Language` header).

## DAG Mode (Graph-based Processes)

Processes can be defined as a directed acyclic graph (DAG) instead of a sequential list. The engine auto-detects the format: if `nodes` is present, it runs in DAG mode; otherwise, sequential.

### DAG Format

```json
{
  "name": "order_fulfillment",
  "nodes": {
    "validate": { "type": "validate", "label": "Validate", "rules": { "status": { "eq": "confirmed" } }, "error": "Must be confirmed" },
    "check_stock": { "type": "http", "label": "Check Stock", "method": "GET", "url": "https://api/stock/{{input.product_id}}" },
    "check_payment": { "type": "http", "label": "Check Payment", "method": "GET", "url": "https://api/payment/{{input.payment_id}}" },
    "reserve": { "type": "update", "label": "Reserve Stock", "model": "stock", "set": { "reserved": true } },
    "notify": { "type": "emit", "label": "Notify", "event": "order.ready" }
  },
  "edges": [
    { "from": "validate", "to": "check_stock" },
    { "from": "validate", "to": "check_payment" },
    { "from": "check_stock", "to": "reserve" },
    { "from": "check_payment", "to": "reserve" },
    { "from": "reserve", "to": "notify" }
  ]
}
```

### Parallel Execution (Fan-out)

When a node has multiple outgoing edges, all target nodes execute in parallel:

```
validate → check_stock  (parallel)
         → check_payment (parallel)
```

### Merge / Wait-All (Fan-in)

When a node has multiple incoming edges, it waits for ALL predecessors to complete:

```
check_stock   → reserve (waits for both)
check_payment → reserve
```

### Conditional Edges

Edges can have a `condition` field. The edge is only followed if the condition evaluates to true:

```json
{ "from": "update", "to": "notify_vip", "condition": "{{input.is_vip}}" }
```

If all incoming edges to a node are conditional and none evaluate to true, the node is skipped.

### Cycle Detection

The engine validates the graph before execution. Cycles are rejected with an error.

### Dual Mode

Both formats work side by side. Existing sequential processes (`steps` array) continue to work unchanged. New processes can use either format.

## Execution Context

Each process runs with a context containing:
- `Input` — request body / trigger data
- `Variables` — assigned during execution
- `Result` — output of last data step
- `UserID` — authenticated user
- `Locale` — current locale for translations (e.g., `id`, `en`)
- `Translator` — translation service for `{{t()}}` and auto-translate
- `Events` — collected domain events (published after process completes)
