# Primary Key Strategies

## Overview

BitCode supports 6 primary key strategies configured at the model level via `primary_key` in model JSON. Default (no config) = UUID v4 for backward compatibility.

## Strategies

### 1. Auto Increment (`auto_increment`)

DB-native auto-increment integer PK. Hidden in forms.

```json
"primary_key": { "strategy": "auto_increment" }
```

### 2. Composite Key (`composite`)

Multiple fields as PK or unique constraint. Two modes:

**With surrogate (default):** UUID `id` as PK + composite fields as UNIQUE constraint.
```json
"primary_key": {
  "strategy": "composite",
  "fields": ["order_id", "product_id"],
  "surrogate": true
}
```

**Without surrogate:** No `id` column, composite fields ARE the PK. URL uses base64-encoded JSON.
```json
"primary_key": {
  "strategy": "composite",
  "fields": ["product_code", "region_code"],
  "surrogate": false
}
```

### 3. UUID (`uuid`)

Three variants: `v4` (random, default), `v7` (time-ordered), `format` (UUID v5 from template).

```json
"primary_key": { "strategy": "uuid", "version": "v7" }
```

```json
"primary_key": {
  "strategy": "uuid",
  "version": "format",
  "format": "{data.nik}-{data.type}-{time.year}",
  "namespace": "customers"
}
```

### 4. Natural Key (`natural_key`)

Existing field becomes PK. Shown in create form, read-only on update.

```json
"primary_key": {
  "strategy": "natural_key",
  "field": "iso_code"
}
```

### 5. Naming Series (`naming_series`)

Auto-generated formatted string as PK. Hidden in forms.

```json
"primary_key": {
  "strategy": "naming_series",
  "field": "code",
  "format": "INV/{time.year}/{sequence(6)}",
  "sequence": { "reset": "yearly", "step": 1 }
}
```

### 6. Manual Input (`manual`)

User provides PK value. Shown in create form, read-only on update.

```json
"primary_key": {
  "strategy": "manual",
  "field": "sku"
}
```

## Non-PK Auto-Format Fields

Any field can use the format engine via `auto_format`:

```json
"po_number": {
  "type": "string",
  "max": 50,
  "auto_format": {
    "format": "PO/{time.year}-{time.month}/{sequence(5)}",
    "sequence": { "reset": "monthly", "step": 1 }
  }
}
```

## Format Template Functions

| Token | Description | Example |
|-------|-------------|---------|
| `{data.<field>}` | Record field value | `{data.nik}` |
| `{time.year}` | 4-digit year | `2026` |
| `{time.month}` | 2-digit month | `04` |
| `{time.day}` | 2-digit day | `24` |
| `{time.hour}` | 2-digit hour | `10` |
| `{time.minute}` | 2-digit minute | `30` |
| `{time.now}` | ISO datetime | `2026-04-24T10:30:00Z` |
| `{time.date}` | Date only | `2026-04-24` |
| `{time.unix}` | Unix timestamp | `1745487000` |
| `{session.user_id}` | Current user ID | |
| `{session.username}` | Current username | |
| `{session.tenant_id}` | Current tenant ID | |
| `{session.group_id}` | Primary group ID | |
| `{session.group_code}` | Primary group code | |
| `{setting.<key>}` | Settings table value | |
| `{sequence(N)}` | Zero-padded sequence, N digits | `000001` |
| `{substring(src,start,len)}` | Substring | `{substring(data.nik,0,3)}` |
| `{upper(src)}` | Uppercase | `{upper(data.dept)}` |
| `{lower(src)}` | Lowercase | `{lower(data.name)}` |
| `{hash(src)}` | SHA-256 first 8 hex chars | |
| `{uuid4}` | Random UUID v4 | |
| `{uuid7}` | Time-ordered UUID v7 | |
| `{model.name}` | Model name | |
| `{model.module}` | Module name | |
| `{random(N)}` | N random alphanumeric chars | |
| `{random_fixed(len,min,max)}` | Random number, fixed length | `{random_fixed(3,200,999)}` |

## Sequence Reset Modes

| Mode | Behavior |
|------|----------|
| `never` | Global counter, never resets |
| `minutely` | Resets every minute |
| `hourly` | Resets every hour |
| `daily` | Resets daily |
| `monthly` | Resets monthly |
| `yearly` | Resets yearly |
| `key` | Resets when resolved template key changes |

## Implementation Files

| File | Purpose |
|------|---------|
| `engine/internal/compiler/parser/model.go` | PK config structs and validation |
| `engine/internal/infrastructure/persistence/sequence.go` | Atomic sequence engine |
| `engine/internal/runtime/format/engine.go` | Format template engine |
| `engine/internal/runtime/pkgen/generator.go` | PK strategy dispatcher |
| `engine/internal/infrastructure/persistence/dynamic_model.go` | Strategy-aware DDL |
| `engine/internal/infrastructure/persistence/repository.go` | PK-aware CRUD |
| `engine/internal/presentation/api/crud_handler.go` | API PK generation |
| `engine/internal/presentation/view/component_compiler.go` | Form field visibility |
