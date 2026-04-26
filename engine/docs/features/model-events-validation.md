# Model Lifecycle Events & Validation

## Overview

Models support declarative lifecycle events, field validation, and sanitization — all defined in JSON. These fire at the repository layer, ensuring consistency regardless of entry point (API, process, script).

## Lifecycle Events

Add `events` to model JSON:

```json
{
  "events": {
    "before_create": [
      { "process": "set_defaults" },
      { "script": { "lang": "typescript", "file": "scripts/on_create.ts" } }
    ],
    "after_create": [
      { "process": "notify", "sync": false, "on_error": "log" }
    ],
    "before_update": [
      { "process": "check_transition", "condition": "old.status != status" }
    ],
    "on_change": {
      "customer_id": [{ "process": "fill_defaults" }]
    }
  }
}
```

### Event Types

| Event | Modify data? | Abort? | Old data? |
|-------|-------------|--------|-----------|
| `before_validate` | Yes | Yes | update: Yes |
| `after_validate` | Yes | Yes | update: Yes |
| `before_create` | Yes | Yes | No |
| `after_create` | No (copy) | sync: Yes | No |
| `before_update` | Yes | Yes | Yes |
| `after_update` | No (copy) | sync: Yes | Yes |
| `before_delete` | No | Yes | Yes |
| `after_delete` | No (copy) | No | Yes |
| `before_save` | Yes | Yes | update: Yes |
| `after_save` | No (copy) | sync: Yes | update: Yes |
| `before_soft_delete` | No | Yes | Yes |
| `after_soft_delete` | No (copy) | No | Yes |
| `before_hard_delete` | No | Yes | Yes |
| `after_hard_delete` | No (copy) | No | Yes |
| `before_restore` | No | Yes | Yes |
| `after_restore` | No (copy) | No | Yes |
| `on_change` | Yes | Yes | Yes |

### Handler Options

- `process` — process name to call
- `script` — `{ "lang": "typescript", "file": "path" }`
- `condition` — expression (supports `old.field`, `session.user_id`)
- `sync` — true/false (default: true for before_*, false for after_*)
- `on_error` — `"fail"` / `"log"` / `"ignore"`
- `retry` — `{ "max": 3, "delay": "5s", "backoff": "exponential" }`
- `timeout` — duration string (default: 30s before, 60s after)
- `priority` — int (lower = first, default 50)
- `bulk_mode` — `"each"` / `"batch"` / `"skip"`

### Data Modification

Before-event handlers modify data via return value merge:

```typescript
export default function(ctx) {
    return { phone: normalize(ctx.data.phone) };
}
```

After-event handlers receive an immutable copy.

### Auto Event Bus

Every CRUD operation publishes to the event bus:
- `model.{name}.created`
- `model.{name}.updated`
- `model.{name}.deleted`
- `model.{name}.restored`

## Field Validation

Add `validation` to field definitions:

```json
{
  "fields": {
    "email": {
      "type": "email",
      "sanitize": ["trim", "lowercase"],
      "validation": {
        "required": true,
        "email": true,
        "unique": { "scope": ["tenant_id"] },
        "max_length": 255
      }
    },
    "end_date": {
      "type": "date",
      "validation": {
        "required": true,
        "date_after": "start_date"
      }
    },
    "cancel_reason": {
      "type": "text",
      "validation": {
        "required_if": { "status": "cancelled" },
        "min_length": 10
      }
    }
  }
}
```

### Built-in Validators

**Presence**: required, email, url, phone, ip, ipv4, ipv6, uuid, json
**Format**: alpha, alpha_num, alpha_dash, numeric, regex, starts_with, ends_with, contains, not_contains, lowercase, uppercase
**Size**: min, max, min_length, max_length, between, length_between, size
**Comparison**: in, not_in, confirmed, different, gt, gte, lt, lte
**Date**: date_before, date_after, date_before_or_equal, date_after_or_equal
**Unique**: unique (simple or scoped with case_insensitive)
**Relational**: exists, exists_where, min_items, max_items
**File**: file_size, file_type
**Immutability**: immutable, immutable_after

### Conditional Validators

- `required_if` / `required_unless`
- `required_with` / `required_with_all`
- `required_without` / `required_without_all`
- `exclude_if` / `exclude_unless`
- `when` — apply validators conditionally
- `on` — `"create"` / `"update"` / `"always"`

### Model-Level Validators

```json
{
  "validators": [
    {
      "name": "date_range",
      "expression": "end_date >= start_date",
      "message": "End date must be after start date"
    }
  ]
}
```

### Error Response (422)

```json
{
  "error": "Validation failed",
  "code": "VALIDATION_ERROR",
  "errors": {
    "email": [{ "rule": "required", "message": "Email is required" }],
    "_model": [{ "rule": "date_range", "message": "End date must be after start date" }]
  }
}
```

## Sanitization

```json
{
  "fields": {
    "email": { "type": "email", "sanitize": ["trim", "lowercase"] },
    "name": { "type": "string", "sanitize": ["trim", "title_case"] }
  },
  "sanitize": { "_all_strings": ["trim"] }
}
```

Built-in: trim, trim_left, trim_right, lowercase, uppercase, title_case, strip_tags, strip_whitespace, slugify, normalize_email, normalize_phone, truncate:N, escape_html.

## Backward Compatibility

- All new fields (`events`, `validation`, `validators`, `sanitize`) are optional
- Existing `required`, `max`, `min` auto-map to validators
- Existing `mandatory_if` continues to work for UI

## Files

- `engine/internal/compiler/parser/model.go` — types
- `engine/internal/runtime/validation/` — validator + sanitizer
- `engine/internal/runtime/hook/` — event dispatcher
- `engine/internal/infrastructure/persistence/repository.go` — integration
- `engine/internal/presentation/api/crud_handler.go` — 422 errors
