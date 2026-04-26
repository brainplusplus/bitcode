# Model Lifecycle Events & Validation — Design Document

**Date**: 2026-04-26
**Status**: Approved
**Scope**: Model JSON `events`, `validation`, `validators`, `sanitize` fields + runtime engine

---

## 1. Overview

Add two major capabilities to model JSON definitions:

1. **Lifecycle Events** — before/after hooks for CRUD operations that can call processes or scripts
2. **Field & Model Validation** — declarative validation rules enforced server-side at the repository layer

Both features are injected at the **repository layer** (not HTTP handler), so they fire consistently regardless of entry point (API, process step, script, internal).

### Design Principles

- **Declarative** — define in JSON, no Go code needed
- **Familiar** — naming follows Laravel/Rails/Frappe conventions
- **Powerful** — supports conditional logic, cross-field validation, custom handlers
- **Safe** — clear transaction boundaries, explicit error handling
- **Backward compatible** — all new fields optional, existing models unchanged

### Framework References

| Concept | Laravel | JPA/Spring | Odoo | Frappe | Rails | Django |
|---------|---------|------------|------|--------|-------|--------|
| Before create | `creating` | `@PrePersist` | `create()` override | `before_insert` | `before_create` | `pre_save` |
| After create | `created` | `@PostPersist` | `create()` after super | `after_insert` | `after_create` | `post_save` |
| Before update | `updating` | `@PreUpdate` | `write()` override | `before_save` | `before_update` | `pre_save` |
| After update | `updated` | `@PostUpdate` | `write()` after super | `on_update` | `after_update` | `post_save` |
| Before delete | `deleting` | `@PreRemove` | `unlink()` override | `before_cancel` | `before_destroy` | `pre_delete` |
| After delete | `deleted` | `@PostRemove` | `unlink()` after super | `after_delete` | `after_destroy` | `post_delete` |
| Validation | Form Request | Bean Validation | `@api.constrains` | `validate` | ActiveModel | `clean()` |
| Onchange | — | — | `@api.onchange` | client script | — | — |

---

## 2. Lifecycle Events

### 2.1 Event Names

```
before_validate    → Before field validation (can modify data)
after_validate     → After validation passes (can abort, data still mutable)

before_create      → Before INSERT (can modify data, can abort)
after_create       → After INSERT committed (side effects)

before_update      → Before UPDATE (can modify data, can abort)
after_update       → After UPDATE committed (side effects)

before_delete      → Before DELETE (can abort)
after_delete       → After DELETE committed (cleanup)

before_save        → Before INSERT or UPDATE (shared logic, can modify)
after_save         → After INSERT or UPDATE committed

before_soft_delete → Before soft delete only (can abort)
after_soft_delete  → After soft delete only
before_hard_delete → Before hard delete only (can abort)
after_hard_delete  → After hard delete only
before_restore     → Before restoring soft-deleted record (can abort)
after_restore      → After restore committed

on_change          → Field-level: when specific field value changes
```

### 2.2 Event Capabilities Matrix

| Event | Can modify data? | Can abort? | Has OldData? | Has Changes? | Transaction |
|-------|-----------------|------------|--------------|--------------|-------------|
| `before_validate` | YES | YES (return error) | create: NO / update: YES | create: NO / update: YES | Inside |
| `after_validate` | YES | YES (return error) | create: NO / update: YES | create: NO / update: YES | Inside |
| `before_create` | YES | YES | NO | NO | Inside |
| `after_create` | NO (receives copy) | sync: YES / async: NO | NO | NO | sync: Inside / async: Post-commit |
| `before_update` | YES | YES | YES | YES | Inside |
| `after_update` | NO (receives copy) | sync: YES / async: NO | YES | YES | sync: Inside / async: Post-commit |
| `before_delete` | NO | YES | YES (full record) | NO | Inside |
| `after_delete` | NO (receives copy) | sync: YES / async: NO | YES (full record) | NO | sync: Inside / async: Post-commit |
| `before_save` | YES | YES | update: YES | update: YES | Inside |
| `after_save` | NO (receives copy) | sync: YES / async: NO | update: YES | update: YES | sync: Inside / async: Post-commit |
| `before_soft_delete` | NO | YES | YES | NO | Inside |
| `after_soft_delete` | NO (receives copy) | NO | YES | NO | Post-commit |
| `before_hard_delete` | NO | YES | YES | NO | Inside |
| `after_hard_delete` | NO (receives copy) | NO | YES | NO | Post-commit |
| `before_restore` | NO | YES | YES | NO | Inside |
| `after_restore` | NO (receives copy) | NO | YES | NO | Post-commit |
| `on_change` | YES | YES | YES (per field) | YES | Inside (before validation) |

**Key rule**: `before_*` handlers receive a **mutable reference** to data. `after_*` handlers receive an **immutable copy** (snapshot of committed state). This prevents after-hooks from creating inconsistency between memory and DB.

### 2.3 Model JSON Schema — Events

```json
{
  "name": "order",
  "fields": {},
  "events": {
    "before_validate": [
      { "process": "normalize_order_data" }
    ],
    "before_create": [
      { "process": "set_default_warehouse" },
      { "script": { "lang": "typescript", "file": "scripts/on_create.ts" } }
    ],
    "after_create": [
      {
        "process": "notify_warehouse",
        "sync": false,
        "on_error": "log",
        "retry": { "max": 3, "delay": "5s", "backoff": "exponential" }
      },
      {
        "process": "critical_audit",
        "sync": true,
        "on_error": "fail"
      }
    ],
    "before_update": [
      {
        "process": "validate_status_transition",
        "condition": "old.status != status"
      }
    ],
    "after_update": [
      {
        "process": "notify_customer",
        "condition": "old.status != status"
      }
    ],
    "before_delete": [
      {
        "process": "check_dependencies",
        "condition": "status != 'draft'"
      }
    ],
    "before_soft_delete": [
      { "process": "deactivate_related" }
    ],
    "before_restore": [
      { "process": "check_restore_allowed" }
    ],
    "on_change": {
      "customer_id": [
        { "process": "fill_customer_defaults" }
      ],
      "quantity": [
        { "process": "recalculate_totals", "server_only": true }
      ]
    }
  }
}
```

### 2.4 Event Handler Types

#### Process handler
```json
{ "process": "process_name" }
```
Calls a registered process by name. Process receives EventContext as input.

#### Script handler
```json
{ "script": { "lang": "typescript", "file": "scripts/handler.ts" } }
```
```json
{ "script": { "lang": "python", "file": "scripts/handler.py" } }
```
Calls a script file via the plugin runtime (JSON-RPC). Script receives EventContext as JSON.

#### Conditional handler
```json
{
  "process": "notify_manager",
  "condition": "status == 'won' && expected_revenue > 100000"
}
```
Handler only executes if condition evaluates to true. Condition has access to all context variables.

### 2.5 Handler Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `process` | `string` | — | Process name to call |
| `script` | `object` | — | Script file to call (`lang` + `file`) |
| `condition` | `string` | — | Expression that must be true for handler to run |
| `sync` | `bool` | `true` for before_*, varies for after_* | Synchronous (in transaction) or async (post-commit) |
| `on_error` | `string` | `"fail"` for before_*, `"log"` for after_* | Error strategy: `"fail"`, `"log"`, `"ignore"` |
| `retry` | `object` | — | Retry config for async handlers: `max`, `delay`, `backoff` |
| `timeout` | `string` | `"30s"` before, `"60s"` after | Max execution time |
| `server_only` | `bool` | `false` | For on_change: skip client-side onchange API |
| `priority` | `int` | `50` | Execution order (lower = first). For inheritance merge. |

### 2.6 Handler Data Modification — Return Value Merge

Handlers that can modify data (before_* events) do so via **return value merge**:

```go
// Engine merges handler result into Data
result, err := executor.Execute(ctx, process, eventCtx)
if err != nil {
    return err // abort operation
}
if modifications, ok := result.(map[string]any); ok && len(modifications) > 0 {
    for k, v := range modifications {
        eventCtx.Data[k] = v
    }
}
```

For scripts:
```typescript
// scripts/on_create.ts — return fields to modify
export default function(ctx) {
    return {
        phone: normalizePhone(ctx.data.phone),
        score: calculateScore(ctx.data)
    };
}
```

Return `null`/empty = no modifications. Return error = abort (before_* only).

### 2.7 Event Context

```go
type EventContext struct {
    Model     string            // model name (e.g., "lead")
    Module    string            // module name (e.g., "crm")
    Event     string            // event name (e.g., "before_create")
    Operation string            // "create", "update", "delete", "restore"
    Data      map[string]any    // current record data (mutable for before_*, copy for after_*)
    OldData   map[string]any    // previous data (update/delete only, nil for create)
    Changes   map[string]any    // changed fields only (update only, nil for create/delete)
    UserID    string            // current user ID
    TenantID  string            // current tenant ID
    Session   map[string]any    // full session data
    IsBulk    bool              // true if part of bulk operation
    BulkIndex int               // index in bulk (0-based)
    BulkTotal int               // total records in bulk
}
```

### 2.8 Condition Expression Variables

Available in `condition` expressions and `when` clauses:

```
{field_name}        → current value from Data
old.{field_name}    → previous value from OldData (update/delete only)
session.user_id     → current user ID
session.tenant_id   → current tenant ID
session.username    → current username
session.group_code  → current group code
is_create           → true if create operation
is_update           → true if update operation
is_delete           → true if delete operation
is_bulk             → true if bulk operation
```

### 2.9 on_change Cascade Behavior

`on_change` handlers can modify data, which may trigger other `on_change` handlers:

1. Detect changed fields (diff old vs incoming)
2. Run `on_change` handlers for changed fields
3. If handlers return modifications that change OTHER fields with `on_change` handlers, cascade
4. Max cascade depth: **5** (configurable)
5. Circular detection: if field A already processed in this cascade chain, skip
6. Log warning at depth > 3

### 2.10 Inherited Model Event Merge

When a child model inherits from a parent (`"inherit": "contact"`):

- **Default**: child events are **appended** after parent events (parent runs first)
- Child can disable parent events per event type:

```json
{
  "inherit": "contact",
  "events": {
    "before_create": {
      "inherit": false,
      "handlers": [
        { "process": "vip_only_validation" }
      ]
    }
  }
}
```

`"inherit": false` = skip parent handlers for this event. Default = `true` (append).

Priority field controls ordering when merging: parent handlers default priority 50, child handlers default 50. Lower priority runs first.

### 2.11 Bulk Operation Modes

Each handler can specify behavior during bulk operations:

```json
{
  "before_create": [
    { "process": "enrich_record", "bulk_mode": "each" },
    { "process": "send_notification", "bulk_mode": "batch" },
    { "process": "expensive_check", "bulk_mode": "skip" }
  ]
}
```

| Mode | Description |
|------|-------------|
| `"each"` (default) | Handler called per record |
| `"batch"` | Handler called once with array of all records |
| `"skip"` | Handler skipped during bulk operations |

### 2.12 Auto-publish to Event Bus

After every CRUD operation, automatically publish to the existing event bus:

```
model.{name}.created    → data: { record }
model.{name}.updated    → data: { record, old_data, changes }
model.{name}.deleted    → data: { record }
model.{name}.restored   → data: { record }
```

This means existing agents and WebSocket continue working without changes.

---

## 3. Field-Level Validation

### 3.1 Validation in Field Definition

```json
{
  "fields": {
    "email": {
      "type": "email",
      "validation": {
        "required": true,
        "email": true,
        "unique": true,
        "max_length": 255
      }
    }
  }
}
```

### 3.2 Complete Built-in Validators

#### Presence & Type

| Validator | Value Type | Description | Example |
|-----------|-----------|-------------|---------|
| `required` | `bool` or `{"on": "create"}` | Must be present and non-empty | `"required": true` |
| `email` | `bool` | Valid email format (RFC 5322) | `"email": true` |
| `url` | `bool` | Valid URL (http/https) | `"url": true` |
| `phone` | `bool` | Valid phone (E.164 or common formats) | `"phone": true` |
| `ip` | `bool` | Valid IPv4 or IPv6 | `"ip": true` |
| `ipv4` | `bool` | Valid IPv4 only | `"ipv4": true` |
| `ipv6` | `bool` | Valid IPv6 only | `"ipv6": true` |
| `uuid` | `bool` | Valid UUID format | `"uuid": true` |
| `json` | `bool` | Valid JSON string | `"json": true` |

#### String Format

| Validator | Value Type | Description | Example |
|-----------|-----------|-------------|---------|
| `alpha` | `bool` | Letters only (Unicode-aware) | `"alpha": true` |
| `alpha_num` | `bool` | Letters and numbers | `"alpha_num": true` |
| `alpha_dash` | `bool` | Letters, numbers, dashes, underscores | `"alpha_dash": true` |
| `numeric` | `bool` | Numeric string (including decimals) | `"numeric": true` |
| `regex` | `string` | Match regex pattern | `"regex": "^[A-Z]{3}$"` |
| `regex_message` | `string` | Custom error for regex failure | `"regex_message": "Must be 3 uppercase letters"` |
| `starts_with` | `string` or `[strings]` | Must start with prefix(es) | `"starts_with": "INV-"` |
| `ends_with` | `string` or `[strings]` | Must end with suffix(es) | `"ends_with": [".com", ".org"]` |
| `contains` | `string` | Must contain substring | `"contains": "@"` |
| `not_contains` | `string` | Must NOT contain substring | `"not_contains": "admin"` |
| `lowercase` | `bool` | Must be all lowercase | `"lowercase": true` |
| `uppercase` | `bool` | Must be all uppercase | `"uppercase": true` |

#### Size & Range

| Validator | Value Type | Description | Example |
|-----------|-----------|-------------|---------|
| `min` | `number` | Minimum value (numeric fields) | `"min": 0` |
| `max` | `number` | Maximum value (numeric fields) | `"max": 999` |
| `min_length` | `int` | Minimum string length | `"min_length": 3` |
| `max_length` | `int` | Maximum string length | `"max_length": 255` |
| `between` | `[min, max]` | Value between range (inclusive) | `"between": [1, 100]` |
| `length_between` | `[min, max]` | String length between range | `"length_between": [3, 50]` |
| `size` | `int` | Exact string length | `"size": 10` |

#### Comparison & Enumeration

| Validator | Value Type | Description | Example |
|-----------|-----------|-------------|---------|
| `in` | `[values]` | Value must be in list | `"in": ["A", "B", "C"]` |
| `not_in` | `[values]` | Value must NOT be in list | `"not_in": ["X", "Y"]` |
| `confirmed` | `string` | Must match another field's value | `"confirmed": "password_confirm"` |
| `different` | `string` | Must differ from another field | `"different": "old_password"` |
| `gt` | `string` | Greater than another field (numeric) | `"gt": "min_price"` |
| `gte` | `string` | Greater than or equal another field | `"gte": "min_price"` |
| `lt` | `string` | Less than another field | `"lt": "max_price"` |
| `lte` | `string` | Less than or equal another field | `"lte": "budget"` |

#### Date Validators

| Validator | Value Type | Description | Example |
|-----------|-----------|-------------|---------|
| `date_before` | `string` | Date before field or value | `"date_before": "end_date"` |
| `date_after` | `string` | Date after field or value | `"date_after": "start_date"` |
| `date_before_or_equal` | `string` | Date <= field or value | `"date_before_or_equal": "2030-12-31"` |
| `date_after_or_equal` | `string` | Date >= field or value | `"date_after_or_equal": "today"` |

Date values can be: field name, `"today"`, `"now"`, or ISO date string.

#### Uniqueness

| Validator | Value Type | Description | Example |
|-----------|-----------|-------------|---------|
| `unique` | `bool` or `object` | Unique in table | `"unique": true` |

Advanced unique:
```json
"unique": {
  "scope": ["tenant_id"],
  "case_insensitive": true
}
```

- `scope` — additional columns for composite uniqueness
- `case_insensitive` — case-insensitive comparison (default: false)
- On update, automatically excludes current record from check

#### Relational Validators

| Validator | Value Type | Description | Example |
|-----------|-----------|-------------|---------|
| `exists` | `bool` | Referenced record must exist | `"exists": true` |
| `exists_where` | `object` | Referenced record must exist with conditions | `"exists_where": {"active": true}` |
| `min_items` | `int` | Minimum related records (one2many/many2many) | `"min_items": 1` |
| `max_items` | `int` | Maximum related records | `"max_items": 100` |

#### File Validators

| Validator | Value Type | Description | Example |
|-----------|-----------|-------------|---------|
| `file_size` | `string` | Max file size | `"file_size": "5MB"` |
| `file_type` | `[strings]` | Allowed MIME types | `"file_type": ["image/png", "image/jpeg"]` |

#### Immutability

| Validator | Value Type | Description | Example |
|-----------|-----------|-------------|---------|
| `immutable` | `bool` | Cannot change after create | `"immutable": true` |
| `immutable_after` | `object` | Cannot change after condition | `"immutable_after": {"status": ["confirmed", "shipped"]}` |

### 3.3 Conditional / Dependent Validators

| Validator | Value Type | Description | Example |
|-----------|-----------|-------------|---------|
| `required_if` | `object` | Required when field = value(s) | `"required_if": {"status": "rejected"}` |
| `required_unless` | `object` | Required unless field = value | `"required_unless": {"type": "internal"}` |
| `required_with` | `[fields]` | Required when ANY of fields present | `"required_with": ["phone"]` |
| `required_with_all` | `[fields]` | Required when ALL fields present | `"required_with_all": ["city", "state"]` |
| `required_without` | `[fields]` | Required when ANY of fields absent | `"required_without": ["email"]` |
| `required_without_all` | `[fields]` | Required when ALL fields absent | `"required_without_all": ["phone", "email"]` |
| `exclude_if` | `object` | Skip validation when condition met | `"exclude_if": {"type": "draft"}` |
| `exclude_unless` | `object` | Skip validation unless condition | `"exclude_unless": {"active": true}` |

`required_if` value can be single value or array:
```json
"required_if": { "status": "rejected" }
"required_if": { "status": ["rejected", "cancelled"] }
"required_if": { "delivery_method": ["shipping", "express"] }
```

### 3.4 `on` — Create vs Update Mode

Any validator can be scoped to create or update only:

```json
{
  "password": {
    "type": "password",
    "validation": {
      "required": { "on": "create" },
      "min_length": 8
    }
  },
  "username": {
    "type": "string",
    "validation": {
      "required": true,
      "immutable": true
    }
  }
}
```

`on` values: `"create"`, `"update"`, `"always"` (default).

When a validator value is an object with `on` key, the actual validator value is in `value`:
```json
"min_length": { "on": "create", "value": 8 }
```

Shorthand: when value is primitive, `on` defaults to `"always"`:
```json
"min_length": 8
```
is equivalent to:
```json
"min_length": { "on": "always", "value": 8 }
```

### 3.5 `when` — Conditional Application

Any validator (or group of validators) can have a `when` condition:

```json
{
  "tax_id": {
    "type": "string",
    "validation": {
      "required_if": { "is_company": true },
      "regex": "^\\d{2}\\.\\d{3}\\.\\d{1}-\\d{3}\\.\\d{3}$",
      "when": { "country": "ID" }
    }
  }
}
```

`when` applies to ALL validators in the same validation block (except `required_if` and other conditional validators which have their own conditions).

For per-validator conditions, use `rules` array:
```json
{
  "zip_code": {
    "type": "string",
    "validation": {
      "required": true,
      "rules": [
        { "regex": "^\\d{5}$", "when": { "country": "ID" }, "regex_message": "Must be 5 digits" },
        { "regex": "^\\d{5}(-\\d{4})?$", "when": { "country": "US" }, "regex_message": "US format: 12345 or 12345-6789" },
        { "regex": "^[A-Z]\\d[A-Z] \\d[A-Z]\\d$", "when": { "country": "CA" }, "regex_message": "CA format: A1A 1A1" }
      ]
    }
  }
}
```

`when` value types:
- `{ "field": "value" }` — equality check
- `{ "field": ["val1", "val2"] }` — in-list check
- `string` — expression: `"status != 'draft' && total > 0"`

### 3.6 Custom Validators (Process / Script)

```json
{
  "tax_id": {
    "type": "string",
    "validation": {
      "required": true,
      "custom": [
        {
          "process": "validate_tax_id",
          "message": "Invalid tax ID format for selected country"
        },
        {
          "script": { "lang": "typescript", "file": "scripts/check_tax_id.ts" },
          "message": "Tax ID already registered in external system"
        }
      ]
    }
  }
}
```

Custom validators:
- Receive the full record data + field name + field value
- Return error (string) to fail, or nil/empty to pass
- Only run if all built-in validators pass (optimization)

### 3.7 Validation Messages — i18n

```json
{
  "email": {
    "type": "email",
    "validation": {
      "required": true,
      "email": true,
      "messages": {
        "required": "validation.email.required",
        "email": "validation.email.invalid"
      }
    }
  }
}
```

- If `messages` not set → auto-generated English defaults with interpolation
- If `messages` value looks like i18n key (contains `.`) → lookup from translator
- If `messages` value is plain string → use as-is

Default message templates (interpolated):
```
"{label} is required"
"{label} must be a valid email address"
"{label} must be at least {min_length} characters"
"{label} must not exceed {max_length} characters"
"{label} must be between {min} and {max}"
"{label} must match pattern {regex}"
"{label} must be unique"
"{label} is required when {other_field} is {other_value}"
"{label} cannot be changed"
"{label} cannot be changed after {condition}"
```

### 3.8 Validation Short-Circuit Strategy

Per field:
1. Run `required` / `required_if` / `required_with` first
2. If required fails AND field is empty → **skip remaining validators for this field**
3. If required passes → run all remaining validators, accumulate errors
4. `unique` validator (DB query) → only run if all format validators pass
5. `custom` validators (process/script) → only run if all built-in validators pass

Across fields:
- Always validate ALL fields (don't stop at first field error)
- Accumulate errors per field
- Return all errors at once

### 3.9 Update Validation — Partial Data Merge

When validating an update (partial data):

1. Fetch old record from DB
2. Merge: `merged = old_record + incoming_changes`
3. Run validators against `merged` data
4. Report errors for:
   - Fields present in `incoming_changes`
   - Fields whose conditional dependency changed (e.g., `required_if: {status: "rejected"}` and `status` is in `incoming_changes`)
5. Do NOT report errors for unchanged fields that were already invalid (existing data concern)

---

## 4. Model-Level Validators (Cross-Field)

For validation that spans multiple fields — similar to Django `clean()`, Rails `validate`, Odoo `@api.constrains`:

```json
{
  "name": "leave_request",
  "fields": {},
  "validators": [
    {
      "name": "date_range_valid",
      "expression": "end_date >= start_date",
      "message": "End date must be after start date"
    },
    {
      "name": "leave_balance_check",
      "process": "check_leave_balance",
      "message": "Insufficient leave balance"
    },
    {
      "name": "max_consecutive_days",
      "expression": "date_diff(end_date, start_date) <= 30",
      "message": "Maximum 30 consecutive days allowed"
    },
    {
      "name": "weekend_check",
      "script": { "lang": "typescript", "file": "scripts/validate_no_weekends.ts" },
      "message": "Leave dates cannot fall on weekends"
    },
    {
      "name": "budget_check",
      "expression": "quantity * unit_price <= budget_limit",
      "message": "Total exceeds budget limit",
      "condition": "status != 'draft'",
      "on": "update"
    }
  ]
}
```

Model-level validator options:
- `name` — identifier (required, for error reporting)
- `expression` — expression that must evaluate to true
- `process` — process to call (returns error string or nil)
- `script` — script to call
- `message` — error message (supports i18n key)
- `condition` — only run when condition is true
- `on` — `"create"`, `"update"`, `"always"` (default)

Model-level errors are reported under `_model` key in error response.

---

## 5. Field Sanitization

Sanitizers run **before** validation, **before** `before_validate` events.

```json
{
  "email": {
    "type": "email",
    "sanitize": ["trim", "lowercase"],
    "validation": { "required": true, "email": true }
  },
  "name": {
    "type": "string",
    "sanitize": ["trim", "title_case"]
  },
  "slug": {
    "type": "string",
    "sanitize": ["trim", "lowercase", "slugify"]
  }
}
```

### Built-in Sanitizers

| Sanitizer | Description |
|-----------|-------------|
| `trim` | Remove leading/trailing whitespace |
| `trim_left` | Remove leading whitespace |
| `trim_right` | Remove trailing whitespace |
| `lowercase` | Convert to lowercase |
| `uppercase` | Convert to uppercase |
| `title_case` | Capitalize first letter of each word |
| `strip_tags` | Remove HTML tags |
| `strip_whitespace` | Collapse multiple whitespace to single space |
| `slugify` | Convert to URL-safe slug (lowercase, hyphens) |
| `normalize_email` | Lowercase + trim + remove dots in gmail-style |
| `normalize_phone` | Normalize to E.164 format |
| `truncate:N` | Truncate to N characters |
| `escape_html` | Escape HTML entities |

Sanitizers are applied in array order. Only applied to non-nil string values.

---

## 6. Validation Error Response Format

```json
{
  "error": "Validation failed",
  "code": "VALIDATION_ERROR",
  "errors": {
    "email": [
      { "rule": "required", "message": "Email is required" },
      { "rule": "email", "message": "Email must be a valid email address" }
    ],
    "phone": [
      { "rule": "min_length", "message": "Phone must be at least 8 characters", "params": { "min_length": 8 } }
    ],
    "end_date": [
      { "rule": "date_after", "message": "End date must be after Start date", "params": { "other": "start_date" } }
    ],
    "_model": [
      { "rule": "leave_balance_check", "message": "Insufficient leave balance" }
    ]
  }
}
```

HTTP status: **422 Unprocessable Entity** (not 400).

Each error includes:
- `rule` — which validator failed
- `message` — human-readable message (i18n-resolved)
- `params` — validator parameters (for client-side formatting)

---

## 7. Complete Execution Flow

### 7.1 CREATE Flow

```
 1. Receive raw data
 2. Apply sanitizers (per field, in order)
 3. Apply dynamic defaults (for fields not in data)
 4. Compute formula fields
 5. Dispatch before_validate → handlers can MODIFY data via return merge
 6. Run field-level built-in validators (with short-circuit per field)
 7. Run field-level conditional validators (required_if, when, etc.)
 8. Run field-level custom validators (process/script)
 9. Run model-level validators (cross-field)
10. If any validation errors → return 422 with accumulated errors, STOP
11. Dispatch after_validate → can abort (return error), data still mutable
12. Dispatch before_save → can modify via return merge, can abort
13. Dispatch before_create → can modify via return merge, can abort
14. BEGIN TRANSACTION
15.   INSERT to DB
16.   Dispatch after_create (sync handlers, on_error="fail") → can rollback
17.   Dispatch after_save (sync handlers) → can rollback
18. COMMIT TRANSACTION
19. Dispatch after_create (async handlers) → fire-and-forget, retryable
20. Dispatch after_save (async handlers) → fire-and-forget
21. Publish "model.{name}.created" to event bus
```

### 7.2 UPDATE Flow

```
 1. Receive partial data
 2. Fetch old record from DB → OldData
 3. Compute changes: Changes = diff(OldData, incoming)
 4. Merge: MergedData = OldData + incoming
 5. Apply sanitizers to changed fields only
 6. Compute formula fields (if dependencies changed)
 7. Dispatch on_change per changed field (cascade max depth 5, circular detection)
 8. Dispatch before_validate → can modify MergedData
 9. Run field-level validators against MergedData
    (report errors only for changed fields + conditionally affected fields)
10. Run field-level conditional validators
11. Run field-level custom validators
12. Run model-level validators
13. If any validation errors → return 422, STOP
14. Dispatch after_validate → can abort
15. Dispatch before_save → can modify, receives OldData + Changes
16. Dispatch before_update → can modify, receives OldData + Changes
17. Compute final changes: FinalChanges = diff(OldData, MergedData)
18. BEGIN TRANSACTION
19.   UPDATE in DB (only FinalChanges, not full record)
20.   Dispatch after_update (sync) → receives OldData, Changes, CommittedData (copy)
21.   Dispatch after_save (sync)
22. COMMIT TRANSACTION
23. Dispatch after_update (async) → receives OldData, Changes, CommittedData (copy)
24. Dispatch after_save (async)
25. Publish "model.{name}.updated" to event bus
```

### 7.3 DELETE Flow

```
 1. Fetch record from DB → RecordData
 2. Dispatch before_delete → can abort
 3. (if soft delete) Dispatch before_soft_delete → can abort
 4. (if hard delete) Dispatch before_hard_delete → can abort
 5. BEGIN TRANSACTION
 6.   DELETE / SOFT DELETE in DB
 7.   Dispatch after_delete (sync) → receives RecordData (copy)
 8.   (if soft) Dispatch after_soft_delete (sync)
 9.   (if hard) Dispatch after_hard_delete (sync)
10. COMMIT TRANSACTION
11. Dispatch after_delete (async)
12. (if soft) Dispatch after_soft_delete (async)
13. (if hard) Dispatch after_hard_delete (async)
14. Publish "model.{name}.deleted" to event bus
```

### 7.4 RESTORE Flow (soft-deleted records)

```
 1. Fetch soft-deleted record → RecordData
 2. Dispatch before_restore → can abort
 3. BEGIN TRANSACTION
 4.   RESTORE (set deleted_at = NULL, deleted_by = NULL)
 5.   Dispatch after_restore (sync)
 6. COMMIT TRANSACTION
 7. Dispatch after_restore (async)
 8. Publish "model.{name}.restored" to event bus
```

---

## 8. Transaction Boundaries

```
┌─────────────────────────────────────────────────┐
│ TRANSACTION                                      │
│                                                   │
│  sanitize → compute → before_validate →           │
│  validation → after_validate →                    │
│  before_save → before_create/update/delete →      │
│  DB OPERATION →                                   │
│  after_* (sync, on_error="fail")                  │
│                                                   │
│  Any error in this block → ROLLBACK entire TX     │
│                                                   │
└─────────────────────────────────────────────────┘
                      │
                      ▼ COMMIT
┌─────────────────────────────────────────────────┐
│ POST-COMMIT (async, no rollback possible)        │
│                                                   │
│  after_* (async, on_error="log"/"ignore")         │
│  event bus publish                                │
│  retry-able handlers                              │
│                                                   │
└─────────────────────────────────────────────────┘
```

**Rule**: Anything that modifies data or can abort MUST be inside the transaction. Side effects (notifications, external API calls) SHOULD be post-commit async.

**Warning for before_* handlers**: If a `before_create` handler calls an external API (HTTP step), and then validation fails later, the external API call **cannot be rolled back**. This is documented behavior — users should put external calls in `after_*` async handlers instead.

---

## 9. Backward Compatibility

### Existing field-level properties auto-mapped to validators

| Existing Property | Auto-mapped Validator |
|-------------------|----------------------|
| `required: true` | `validation.required: true` |
| `max: N` | `validation.max: N` (numeric) or `validation.max_length: N` (string) |
| `min: N` | `validation.min: N` (numeric) or `validation.min_length: N` (string) |
| `mandatory_if: "expr"` | `validation.required_if` (parsed from expression) |

These auto-mappings ensure existing models get server-side validation without changes. Explicit `validation` block takes precedence over auto-mapped values.

### No breaking changes

- `events`, `validation`, `validators`, `sanitize` are all new optional fields
- Existing models without these fields behave exactly as before
- Existing `required`, `max`, `min` continue to work for DDL
- Existing `mandatory_if`, `depends_on`, `readonly_if` continue to work for UI

---

## 10. Complete Example Model

```json
{
  "name": "order",
  "module": "sales",
  "label": "Sales Order",
  "primary_key": {
    "strategy": "naming_series",
    "field": "order_number",
    "format": "SO-{YYYY}{MM}-{####}",
    "sequence": { "reset": "monthly" }
  },
  "fields": {
    "order_number": {
      "type": "string",
      "required": true,
      "label": "Order Number",
      "validation": { "immutable": true }
    },
    "customer_id": {
      "type": "many2one",
      "model": "customer",
      "label": "Customer",
      "validation": {
        "required": true,
        "exists_where": { "active": true }
      }
    },
    "order_date": {
      "type": "date",
      "label": "Order Date",
      "default": "today",
      "validation": { "required": true }
    },
    "delivery_date": {
      "type": "date",
      "label": "Delivery Date",
      "validation": {
        "required": true,
        "date_after": "order_date"
      }
    },
    "status": {
      "type": "selection",
      "options": ["draft", "confirmed", "shipped", "delivered", "cancelled"],
      "default": "draft",
      "label": "Status"
    },
    "email": {
      "type": "email",
      "label": "Contact Email",
      "sanitize": ["trim", "lowercase"],
      "validation": {
        "required": true,
        "email": true,
        "max_length": 255
      }
    },
    "phone": {
      "type": "string",
      "label": "Phone",
      "sanitize": ["trim"],
      "validation": {
        "phone": true,
        "required_without": ["email"]
      }
    },
    "shipping_address": {
      "type": "text",
      "label": "Shipping Address",
      "sanitize": ["trim"],
      "validation": {
        "required_if": { "delivery_method": ["shipping", "express"] },
        "min_length": 10,
        "max_length": 500
      }
    },
    "delivery_method": {
      "type": "selection",
      "options": ["pickup", "shipping", "express"],
      "default": "pickup",
      "label": "Delivery Method"
    },
    "cancel_reason": {
      "type": "text",
      "label": "Cancellation Reason",
      "depends_on": "status == 'cancelled'",
      "validation": {
        "required_if": { "status": "cancelled" },
        "min_length": 10
      }
    },
    "discount_code": {
      "type": "string",
      "label": "Discount Code",
      "sanitize": ["trim", "uppercase"],
      "validation": {
        "regex": "^[A-Z0-9]{4,12}$",
        "regex_message": "Must be 4-12 uppercase alphanumeric characters",
        "custom": [
          {
            "process": "validate_discount_code",
            "message": "Invalid or expired discount code"
          }
        ]
      }
    },
    "total": {
      "type": "currency",
      "currency": "IDR",
      "computed": "sum(lines.subtotal)",
      "label": "Total"
    },
    "lines": {
      "type": "one2many",
      "model": "order_line",
      "inverse": "order_id",
      "label": "Order Lines",
      "validation": {
        "min_items": { "on": "update", "value": 1, "when": "status != 'draft'" }
      }
    }
  },
  "validators": [
    {
      "name": "delivery_after_order",
      "expression": "delivery_date >= order_date",
      "message": "Delivery date must be on or after order date"
    },
    {
      "name": "has_lines_when_confirmed",
      "process": "check_order_has_lines",
      "message": "Order must have at least one line item",
      "condition": "status != 'draft'"
    }
  ],
  "events": {
    "before_validate": [
      { "process": "normalize_order_data" }
    ],
    "before_create": [
      { "process": "set_default_warehouse" }
    ],
    "after_create": [
      {
        "process": "notify_warehouse",
        "sync": false,
        "on_error": "log",
        "retry": { "max": 3, "delay": "5s", "backoff": "exponential" }
      }
    ],
    "before_update": [
      {
        "process": "validate_status_transition",
        "condition": "old.status != status"
      }
    ],
    "after_update": [
      {
        "process": "notify_customer_status_change",
        "condition": "old.status != status",
        "sync": false
      }
    ],
    "before_delete": [
      {
        "process": "check_order_deletable",
        "condition": "status != 'draft'"
      }
    ],
    "on_change": {
      "customer_id": [
        { "process": "fill_customer_defaults" }
      ],
      "delivery_method": [
        { "process": "recalculate_shipping" }
      ]
    }
  },
  "sanitize": {
    "_all_strings": ["trim"]
  },
  "timestamps": true,
  "timestamps_by": true,
  "soft_deletes": true,
  "version": true
}
```

### Model-level sanitize shorthand

```json
{
  "sanitize": {
    "_all_strings": ["trim"]
  }
}
```

`_all_strings` applies sanitizers to ALL string-type fields that don't have their own `sanitize` defined. This avoids repeating `"sanitize": ["trim"]` on every field.

---

## 11. Architecture — File Map

### New Files

```
engine/internal/
├── runtime/
│   ├── validation/
│   │   ├── validator.go          # Main validation engine (ValidateCreate, ValidateUpdate)
│   │   ├── rules.go              # Built-in validation rule implementations
│   │   ├── conditional.go        # Conditional validators (required_if, when, etc.)
│   │   ├── sanitizer.go          # Sanitization engine
│   │   └── validator_test.go     # Tests
│   └── hook/
│       ├── dispatcher.go         # Event dispatcher (Dispatch, manages handler execution)
│       ├── context.go            # EventContext struct
│       ├── registry.go           # HookRegistry (per-model event handler registration)
│       └── dispatcher_test.go    # Tests
```

### Modified Files

```
engine/internal/
├── compiler/parser/
│   └── model.go                  # Add: EventDefinition, ValidationRule, ModelValidator,
│                                 #   SanitizeConfig structs to ModelDefinition + FieldDefinition
├── infrastructure/persistence/
│   ├── repository.go             # Inject hook dispatch + validation around Create/Update/Delete
│   ├── repository_interface.go   # Add hook/validation dependencies to Repository interface
│   └── mongo_repository.go       # Same hooks for MongoDB
├── presentation/api/
│   └── crud_handler.go           # Handle 422 validation errors, pass context
└── app.go                        # Wire HookRegistry, ValidationEngine, load model events
```

---

## 12. Migration Concerns

- Validation only runs on CREATE and UPDATE operations going forward
- Existing data that doesn't comply with new validation rules is NOT retroactively rejected
- When editing an existing non-compliant record, validation will fire
- Recommendation: provide a `bitcode validate-data` CLI command (future) to check existing data

---

## 13. Phase Plan

### Phase 1 (This Implementation)

Everything in this document:
- Parser: new structs (events, validation, validators, sanitize)
- Validation engine: all built-in validators, conditional, custom, model-level
- Sanitization engine: all built-in sanitizers
- Event/Hook system: dispatcher, context, handler execution (process + script)
- Repository integration: hook injection in Create/Update/Delete
- CRUD handler: 422 error format
- Auto-publish to event bus
- Backward compatibility (auto-map existing required/max/min)
- Tests
- Documentation

### Phase 2 (Future)

- Client-side onchange API endpoint (`POST /api/{model}/onchange`)
- `bitcode validate-data` CLI command
- Event priority for complex inheritance scenarios
- Bulk operation batch mode handlers
- Dynamic defaults (`"default": "{{today + 7d}}"`)
