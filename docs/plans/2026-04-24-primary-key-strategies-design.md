# Primary Key Strategies & Sequence Engine — Design Document

**Date**: 2026-04-24
**Status**: Approved
**Scope**: Engine-wide primary key strategy system with format template engine and race-condition-safe sequence generation

---

## 1. Overview

Add configurable primary key strategies to the BitCode model system. Today, all dynamic models use an implicit UUID v4 `id` column. This design introduces 6 PK strategies, a format template engine with 30+ built-in functions, and an atomic sequence number system.

**Backward compatible**: no `primary_key` section in model JSON = implicit UUID v4 (current behavior preserved).

---

## 2. Model JSON Schema

### 2.1 Primary Key Configuration (Model-Level)

New optional top-level `primary_key` object in `ModelDefinition`:

```json
{
  "name": "invoice",
  "primary_key": {
    "strategy": "naming_series",
    "field": "code",
    "format": "INV/{time.year}/{sequence(6)}",
    "sequence": {
      "reset": "yearly",
      "step": 1
    }
  },
  "fields": {
    "code": { "type": "string", "max": 50, "label": "Invoice Code" },
    "customer_id": { "type": "many2one", "model": "customer" }
  }
}
```

### 2.2 Non-PK Field Auto-Format

Any field can use the format engine via `auto_format`:

```json
{
  "name": "purchase_order",
  "fields": {
    "po_number": {
      "type": "string",
      "max": 50,
      "auto_format": {
        "format": "PO/{upper(data.department)}/{time.year}-{time.month}/{sequence(5)}",
        "sequence": { "reset": "monthly", "step": 1 }
      }
    },
    "department": { "type": "string", "max": 50 }
  }
}
```

---

## 3. The 6 PK Strategies

### 3.1 Auto Increment (`auto_increment`)

DB-native auto-increment. No application-side ID generation.

```json
"primary_key": { "strategy": "auto_increment" }
```

- PK column: `id INTEGER PRIMARY KEY AUTOINCREMENT` (SQLite), `id BIGSERIAL PRIMARY KEY` (Postgres), `id BIGINT AUTO_INCREMENT PRIMARY KEY` (MySQL)
- Form: hidden on create and update
- FK references from other models use INTEGER type

### 3.2 Composite Key (`composite`)

Multiple fields form the primary key or unique constraint.

**Mode A — With surrogate (default):**
```json
"primary_key": {
  "strategy": "composite",
  "fields": ["order_id", "product_id", "variant"],
  "surrogate": true
}
```
- Keeps auto-generated UUID `id` as PK
- Composite fields become a UNIQUE constraint
- URL uses `id` as usual
- FK references use UUID `id`

**Mode B — Without surrogate (legacy support):**
```json
"primary_key": {
  "strategy": "composite",
  "fields": ["product_code", "region_code", "effective_date"],
  "surrogate": false
}
```
- No `id` column; composite fields ARE the PK: `PRIMARY KEY (product_code, region_code, effective_date)`
- URL uses base64-encoded JSON of key values
- Repository detects composite model and decodes: `eyJ0ZW5hbnRfaWQiOiJ0ZW5hbnQtMDAxIn0=`
- FK references require all composite key fields

### 3.3 UUID (`uuid`)

Three sub-variants controlled by `version`:

**v4 (default, random):**
```json
"primary_key": { "strategy": "uuid", "version": "v4" }
```
Or simply omit `primary_key` entirely (backward compatible).

**v7 (time-ordered):**
```json
"primary_key": { "strategy": "uuid", "version": "v7" }
```
- Sortable by creation time
- Better index performance than v4

**format (UUID v5 from template):**
```json
"primary_key": {
  "strategy": "uuid",
  "version": "format",
  "format": "{data.nik}-{data.customer_type}-{time.year}",
  "namespace": "customers"
}
```
- UUID v5 = SHA-1 deterministic: same input always produces same UUID
- `namespace` defaults to model name if omitted
- Algorithm: `uuid.NewSHA1(namespace_uuid, []byte(resolved_format))`
- Format can include `{sequence(N)}` and all other template functions

### 3.4 Natural Key (`natural_key`)

An existing field becomes the PK.

```json
"primary_key": {
  "strategy": "natural_key",
  "field": "iso_code"
}
```
- Referenced field must exist in `fields`, be `required: true`, and `unique: true`
- PK column uses the field's native SQL type (no separate `id` column)
- Form: shown on create (user inputs), read-only on update

### 3.5 Naming Series (`naming_series`)

Auto-generated formatted string as PK.

```json
"primary_key": {
  "strategy": "naming_series",
  "field": "code",
  "format": "INV/{time.year}/{sequence(6)}",
  "sequence": { "reset": "yearly", "step": 1 }
}
```
- `field` references a declared string field that stores the generated value
- The generated value IS the PK (no separate `id` column)
- Form: hidden on create and update
- PK column type: VARCHAR based on field's `max` attribute

### 3.6 Manual Input (`manual`)

User provides the PK value.

```json
"primary_key": {
  "strategy": "manual",
  "field": "sku"
}
```
- Referenced field must exist in `fields`
- Form: shown on create (user inputs), read-only on update
- No auto-generation; validation ensures non-empty on create

---

## 4. Format Template Engine

### 4.1 Built-in Functions

| Token | Description | Example Output |
|-------|-------------|----------------|
| `{data.<field>}` | Value from record field | `{data.nik}` → `3201234567` |
| `{time.now}` | Full ISO datetime | `2025-04-24T10:30:00` |
| `{time.year}` | 4-digit year | `2025` |
| `{time.month}` | 2-digit month | `04` |
| `{time.day}` | 2-digit day | `24` |
| `{time.hour}` | 2-digit hour | `10` |
| `{time.minute}` | 2-digit minute | `30` |
| `{time.date}` | Date only | `2025-04-24` |
| `{time.unix}` | Unix timestamp (seconds) | `1745467200` |
| `{session.user_id}` | Current user ID | `abc-123-...` |
| `{session.username}` | Current username | `admin` |
| `{session.tenant_id}` | Current tenant ID | `tenant-001` |
| `{session.group_id}` | Primary group ID | `grp-001` |
| `{session.group_code}` | Primary group code | `sales_team` |
| `{setting.<key>}` | Value from settings table | `{setting.company_code}` → `ACME` |
| `{sequence(N)}` | Zero-padded auto-increment, N digits | `{sequence(6)}` → `000001` |
| `{substring(src,start,len)}` | Substring extraction | `{substring(data.nik,0,3)}` → `320` |
| `{upper(src)}` | Uppercase | `{upper(data.dept)}` → `SALES` |
| `{lower(src)}` | Lowercase | `{lower(data.name)}` → `budi` |
| `{uuid4}` | Random UUID v4 | `550e8400-e29b-41d4-...` |
| `{uuid7}` | Time-ordered UUID v7 | `018f3e1c-...` |
| `{model.name}` | Model name | `invoice` |
| `{model.module}` | Module name | `sales` |
| `{random(N)}` | Random alphanumeric, N chars | `{random(8)}` → `aB3kZ9mQ` |
| `{random_fixed(len,min,max)}` | Random number, fixed length, bounded | `{random_fixed(3,200,999)}` → `547` |
| `{hash(src)}` | Short hash (8 hex chars) | `{hash(data.email)}` → `a1b2c3d4` |

### 4.2 Sequence Position

`{sequence(N)}` can appear anywhere in the format string:

- Prefix: `{sequence(6)}/INV/{time.year}` → `000001/INV/2025`
- Middle: `INV/{sequence(6)}/{time.year}` → `INV/000001/2025`
- Suffix: `INV/{time.year}/{sequence(6)}` → `INV/2025/000001`

### 4.3 Nested Function Resolution

Functions that take `src` parameters resolve inner references first:
- `{upper(data.dept)}` → resolve `data.dept` → `"sales"` → apply `upper` → `"SALES"`
- `{substring(data.nik,0,3)}` → resolve `data.nik` → `"3201234567"` → apply `substring(0,3)` → `"320"`
- `{hash(data.email)}` → resolve `data.email` → `"budi@test.com"` → apply `hash` → `"a1b2c3d4"`

---

## 5. Sequence Engine

### 5.1 Database Table: `sequences`

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID PK | Row identifier |
| `model_name` | VARCHAR(100) NOT NULL | e.g. `invoice` |
| `field_name` | VARCHAR(100) NOT NULL | e.g. `code`, `id`, `po_number` |
| `sequence_key` | VARCHAR(500) NOT NULL | Resolved key e.g. `INV/2025/` |
| `next_value` | BIGINT NOT NULL DEFAULT 1 | Next number to issue |
| `step` | INT NOT NULL DEFAULT 1 | Increment step |
| `created_at` | TIMESTAMP | |
| `updated_at` | TIMESTAMP | |
| | UNIQUE constraint | `(model_name, field_name, sequence_key)` |

This table is auto-created during engine startup (alongside other system tables).

### 5.2 Reset Modes

| Reset Mode | Sequence Key Derivation |
|------------|------------------------|
| `never` | `"{model_name}:{field_name}"` (static, never resets) |
| `yearly` | `"{model_name}:{field_name}:{year}"` |
| `monthly` | `"{model_name}:{field_name}:{year}-{month}"` |
| `daily` | `"{model_name}:{field_name}:{year}-{month}-{day}"` |
| `hourly` | `"{model_name}:{field_name}:{year}-{month}-{day}T{hour}"` |
| `minutely` | `"{model_name}:{field_name}:{year}-{month}-{day}T{hour}:{minute}"` |
| `key` | Strip `{sequence(N)}` from format, resolve remaining tokens, use result as key |

When the key changes (new year, new month, etc.), a new row is inserted in `sequences` with `next_value = 1`. Old rows remain for audit/reference.

### 5.3 Race Condition Handling

Atomic DB operations — no application-level locks needed:

**PostgreSQL:**
```sql
INSERT INTO sequences (id, model_name, field_name, sequence_key, next_value, step)
VALUES (gen_random_uuid(), $1, $2, $3, 2, $4)
ON CONFLICT (model_name, field_name, sequence_key)
DO UPDATE SET next_value = sequences.next_value + sequences.step,
             updated_at = NOW()
RETURNING next_value - step;
```

**SQLite:**
```sql
INSERT INTO sequences (id, model_name, field_name, sequence_key, next_value, step)
VALUES ($1, $2, $3, $4, 2, $5)
ON CONFLICT (model_name, field_name, sequence_key)
DO UPDATE SET next_value = next_value + step,
             updated_at = CURRENT_TIMESTAMP
RETURNING next_value - step;
```

**MySQL:**
```sql
INSERT INTO sequences (id, model_name, field_name, sequence_key, next_value, step)
VALUES (?, ?, ?, ?, 2, ?)
ON DUPLICATE KEY UPDATE next_value = next_value + step,
                        updated_at = NOW();
-- Then: SELECT next_value - step FROM sequences WHERE model_name = ? AND field_name = ? AND sequence_key = ?;
```

The `RETURNING` clause (Postgres/SQLite) or follow-up `SELECT` (MySQL) returns the allocated value atomically. Concurrent requests get different values without conflicts.

---

## 6. Form Visibility Rules

| Strategy | Create Form | Update Form |
|----------|-------------|-------------|
| `auto_increment` | Hidden | Hidden (read-only display optional) |
| `uuid` (v4/v7) | Hidden | Hidden |
| `uuid` (format) | Hidden | Hidden |
| `naming_series` | Hidden | Hidden (read-only display optional) |
| `natural_key` | Shown (user inputs) | Read-only |
| `composite` (surrogate) | Per-field (shown if in layout) | Per-field |
| `composite` (no surrogate) | Per-field (shown if in layout) | Read-only on PK fields |
| `manual` | Shown (user inputs) | Read-only |
| Non-PK `auto_format` field | Hidden | Read-only |

The component compiler and form engine enforce these rules automatically based on the model's PK strategy.

---

## 7. Affected Layers (Change Map)

| Layer | File | Change |
|-------|------|--------|
| **Parser** | `engine/internal/compiler/parser/model.go` | Add `PrimaryKeyConfig`, `SequenceConfig`, `AutoFormatConfig` structs; add `PrimaryKey *PrimaryKeyConfig` to `ModelDefinition`; add `AutoFormat *AutoFormatConfig` to `FieldDefinition`; validation for PK config |
| **Migration** | `engine/internal/infrastructure/persistence/dynamic_model.go` | Generate PK column based on strategy; create `sequences` system table; handle composite PK DDL; handle auto-increment DDL per dialect; handle natural key / naming series PK column |
| **Sequence Engine** | `engine/internal/infrastructure/persistence/sequence.go` (**NEW**) | `NextValue(db, modelName, fieldName, sequenceKey, step)` — atomic sequence generation per dialect |
| **Format Engine** | `engine/internal/runtime/format/engine.go` (**NEW**) | Parse & resolve format templates; all 30+ built-in functions; nested function resolution |
| **PK Generator** | `engine/internal/runtime/pkgen/generator.go` (**NEW**) | Strategy dispatcher — routes to format engine + sequence engine + UUID libs based on model's PK config |
| **Repository** | `engine/internal/infrastructure/persistence/repository.go` | Accept PK generator in constructor; call PK generator in `Create`; support composite key `FindByPK` with base64 decode for no-surrogate models; update `FindByID`/`Update`/`Delete` to use correct PK column name |
| **CRUD Handler** | `engine/internal/presentation/api/crud_handler.go` | Remove hardcoded `uuid.New()`; delegate to PK generator; handle base64 composite IDs in URL params |
| **Form/View Compiler** | `engine/internal/presentation/view/component_compiler.go` | Auto-hide PK fields for auto-generated strategies; auto-readonly for `auto_format` fields on update |
| **Seeder** | `engine/internal/infrastructure/module/seder.go` | Use PK generator instead of hardcoded UUID fallback |
| **Process Steps** | `engine/internal/runtime/executor/steps/data.go` | Use PK generator for create steps |
| **App Wiring** | `engine/internal/app.go` | Initialize PK generator; wire into repository creation; SSR form POST uses PK generator; create `sequences` table on startup |
| **Admin UI** | `engine/internal/presentation/admin/admin.go` | Display PK strategy info in model detail view |

---

## 8. JSON Examples (Complete)

### 8.1 Auto Increment
```json
{
  "name": "log_entry",
  "primary_key": { "strategy": "auto_increment" },
  "fields": {
    "message": { "type": "text" },
    "level": { "type": "selection", "options": ["info", "warn", "error"] }
  }
}
```

### 8.2 Composite Key (with surrogate)
```json
{
  "name": "order_line",
  "primary_key": {
    "strategy": "composite",
    "fields": ["order_id", "product_id", "variant"],
    "surrogate": true
  },
  "fields": {
    "order_id": { "type": "many2one", "model": "order" },
    "product_id": { "type": "many2one", "model": "product" },
    "variant": { "type": "string", "max": 50 },
    "quantity": { "type": "integer" },
    "price": { "type": "currency", "currency": "IDR" }
  }
}
```

### 8.3 Composite Key (no surrogate — legacy)
```json
{
  "name": "legacy_price",
  "primary_key": {
    "strategy": "composite",
    "fields": ["product_code", "region_code", "effective_date"],
    "surrogate": false
  },
  "fields": {
    "product_code": { "type": "string", "max": 20 },
    "region_code": { "type": "string", "max": 10 },
    "effective_date": { "type": "date" },
    "price": { "type": "currency", "currency": "IDR" }
  }
}
```

### 8.4 UUID v4 (default — implicit)
```json
{
  "name": "contact",
  "fields": {
    "name": { "type": "string", "required": true },
    "email": { "type": "email" }
  }
}
```

### 8.5 UUID v7
```json
{
  "name": "event_log",
  "primary_key": { "strategy": "uuid", "version": "v7" },
  "fields": {
    "event": { "type": "string" },
    "timestamp": { "type": "datetime" }
  }
}
```

### 8.6 UUID from Format (v5)
```json
{
  "name": "customer",
  "primary_key": {
    "strategy": "uuid",
    "version": "format",
    "format": "{data.nik}-{data.customer_type}-{time.year}",
    "namespace": "customers"
  },
  "fields": {
    "nik": { "type": "string", "required": true, "max": 20 },
    "customer_type": { "type": "selection", "options": ["individual", "corporate"] },
    "name": { "type": "string", "required": true }
  }
}
```

### 8.7 UUID from Format with Sequence
```json
{
  "name": "transaction",
  "primary_key": {
    "strategy": "uuid",
    "version": "format",
    "format": "{data.branch_code}-{time.year}-{sequence(8)}",
    "namespace": "transactions",
    "sequence": { "reset": "yearly", "step": 1 }
  },
  "fields": {
    "branch_code": { "type": "string", "required": true, "max": 10 },
    "amount": { "type": "currency", "currency": "IDR" }
  }
}
```

### 8.8 Natural Key
```json
{
  "name": "country",
  "primary_key": {
    "strategy": "natural_key",
    "field": "iso_code"
  },
  "fields": {
    "iso_code": { "type": "string", "required": true, "unique": true, "max": 3 },
    "name": { "type": "string", "required": true }
  }
}
```

### 8.9 Naming Series
```json
{
  "name": "invoice",
  "primary_key": {
    "strategy": "naming_series",
    "field": "code",
    "format": "INV/{time.year}/{sequence(6)}",
    "sequence": { "reset": "yearly", "step": 1 }
  },
  "fields": {
    "code": { "type": "string", "max": 50, "label": "Invoice Code" },
    "customer_id": { "type": "many2one", "model": "customer" },
    "total": { "type": "currency", "currency": "IDR" }
  }
}
```

### 8.10 Naming Series with Complex Format
```json
{
  "name": "surat_keluar",
  "primary_key": {
    "strategy": "naming_series",
    "field": "nomor_surat",
    "format": "{sequence(4)}/{upper(data.jenis)}/{setting.kode_instansi}/{time.month}/{time.year}",
    "sequence": { "reset": "monthly", "step": 1 }
  },
  "fields": {
    "nomor_surat": { "type": "string", "max": 100 },
    "jenis": { "type": "selection", "options": ["undangan", "edaran", "keputusan"] },
    "perihal": { "type": "string", "required": true }
  }
}
```

### 8.11 Manual Input
```json
{
  "name": "product",
  "primary_key": {
    "strategy": "manual",
    "field": "sku"
  },
  "fields": {
    "sku": { "type": "string", "required": true, "max": 50, "label": "SKU" },
    "name": { "type": "string", "required": true },
    "price": { "type": "currency", "currency": "IDR" }
  }
}
```

### 8.12 Non-PK Auto-Format Field
```json
{
  "name": "purchase_order",
  "primary_key": { "strategy": "uuid", "version": "v4" },
  "fields": {
    "po_number": {
      "type": "string",
      "max": 50,
      "auto_format": {
        "format": "PO/{upper(data.department)}/{time.year}-{time.month}/{sequence(5)}",
        "sequence": { "reset": "monthly", "step": 1 }
      }
    },
    "department": { "type": "string", "max": 50 },
    "total": { "type": "currency", "currency": "IDR" }
  }
}
```

---

## 9. Go Struct Additions

### 9.1 Parser Structs (model.go)

```go
type PKStrategy string

const (
    PKAutoIncrement PKStrategy = "auto_increment"
    PKComposite     PKStrategy = "composite"
    PKUUID          PKStrategy = "uuid"
    PKNaturalKey    PKStrategy = "natural_key"
    PKNamingSeries  PKStrategy = "naming_series"
    PKManual        PKStrategy = "manual"
)

type SequenceConfig struct {
    Reset string `json:"reset,omitempty"` // never, minutely, hourly, daily, monthly, yearly, key
    Step  int    `json:"step,omitempty"`  // default 1
}

type PrimaryKeyConfig struct {
    Strategy  PKStrategy      `json:"strategy"`
    Field     string          `json:"field,omitempty"`      // for natural_key, naming_series, manual
    Fields    []string        `json:"fields,omitempty"`     // for composite
    Surrogate *bool           `json:"surrogate,omitempty"`  // for composite, default true
    Version   string          `json:"version,omitempty"`    // for uuid: v4, v7, format
    Format    string          `json:"format,omitempty"`     // for naming_series, uuid format
    Namespace string          `json:"namespace,omitempty"`  // for uuid format
    Sequence  *SequenceConfig `json:"sequence,omitempty"`   // for naming_series, uuid format
}

type AutoFormatConfig struct {
    Format   string          `json:"format"`
    Sequence *SequenceConfig `json:"sequence,omitempty"`
}
```

### 9.2 ModelDefinition Addition

```go
type ModelDefinition struct {
    Name        string                     `json:"name"`
    Module      string                     `json:"module,omitempty"`
    Label       string                     `json:"label,omitempty"`
    Inherit     string                     `json:"inherit,omitempty"`
    PrimaryKey  *PrimaryKeyConfig          `json:"primary_key,omitempty"`
    Fields      map[string]FieldDefinition `json:"fields"`
    RecordRules []RecordRuleDefinition     `json:"record_rules,omitempty"`
    Indexes     [][]string                 `json:"indexes,omitempty"`
    FileConfig  *FileConfig                `json:"file_config,omitempty"`
}
```

### 9.3 FieldDefinition Addition

```go
type FieldDefinition struct {
    // ... existing fields ...
    AutoFormat *AutoFormatConfig `json:"auto_format,omitempty"`
}
```

---

## 10. Design Decisions & Rationale

| Decision | Rationale |
|----------|-----------|
| Model-level PK config (not field-level) | PK strategy is a model concern; composite keys span multiple fields; auto-increment/UUID have no user-visible field |
| Backward compatible (no config = UUID v4) | Zero migration needed for existing models |
| UUID v5 for format-based UUID | Deterministic (idempotent), standard RFC 4122, retry-safe |
| Atomic DB operations for sequences | No app-level mutex; works across multiple engine instances; DB handles concurrency |
| Composite key dual mode (surrogate/no-surrogate) | Surrogate mode preserves existing FK/API patterns; no-surrogate supports legacy databases |
| Base64 encoding for composite no-surrogate URLs | Single URL param; works with existing route structure; no delimiter escaping issues |
| Format template engine as shared infrastructure | Reused by PK generation AND non-PK auto_format fields |
| `sequences` table with unique key | Supports multiple sequence counters per model (different fields, different reset periods) |
