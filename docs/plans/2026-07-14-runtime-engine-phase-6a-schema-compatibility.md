# Phase 6A: Schema Compatibility — Field Types, Storage Hints, Modifiers, Display & Table Naming

**Date**: 14 July 2026
**Status**: Draft
**Depends on**: None (parser-level, can be implemented independently)
**Unlocks**: Phase 6B (polymorphic relations), Phase 6C (engine enhancements), Phase 7 (module setting)
**Master doc**: `2026-07-14-runtime-engine-redesign-master.md`

---

## Table of Contents

1. [Goal](#1-goal)
2. [Current State Audit](#2-current-state-audit)
3. [Design Philosophy: Two-Layer Type System](#3-design-philosophy-two-layer-type-system)
4. [New Semantic Types](#4-new-semantic-types)
5. [Storage Hint System](#5-storage-hint-system)
6. [Field Modifiers](#6-field-modifiers)
7. [Title Field Format Engine Integration](#7-title-field-format-engine-integration)
8. [Many2One Display: `display_field` Override](#8-many2one-display-display_field-override)
9. [Currency Field Enhancement](#9-currency-field-enhancement)
10. [JSON Variants: `json`, `json:object`, `json:array`](#10-json-variants-json-jsonobject-jsonarray)
11. [Precision/Scale Backward Compatibility Fix](#11-precisionscale-backward-compatibility-fix)
12. [Complete 4-Database SQL Mapping](#12-complete-4-database-sql-mapping)
13. [MongoDB Mapping](#13-mongodb-mapping)
14. [Validation Rules per Type](#14-validation-rules-per-type)
15. [Default Behavior per Type](#15-default-behavior-per-type)
16. [Table Naming: Plural Convention](#16-table-naming-plural-convention)
17. [Duplicate Model Detection](#17-duplicate-model-detection)
18. [Migration & ALTER COLUMN](#18-migration--alter-column)
19. [Parser Changes Summary](#19-parser-changes-summary)
20. [Implementation Tasks](#20-implementation-tasks)

---

## 1. Goal

Fix and extend the engine's field type system to be **complete, unambiguous, and cross-database compatible** while maintaining full backward compatibility.

### 1.1 Problems Being Solved

1. **20 of 33 field types fall to `default: return "TEXT"`** in `fieldTypeToSQL` — no proper SQL mapping for float, currency, percent, time, duration, password, richtext, markdown, html, code, smalltext, toggle, radio, dynamic_link, image, signature, barcode, color, geolocation, rating
2. **Missing common types**: uuid, ip, ipv6, year, vector, binary — types that real applications need
3. **`precision` naming is wrong** — current `precision` field actually means `scale` (decimal places), not total digits
4. **No storage hint** — developer cannot request BIGINT, NUMERIC unlimited, LONGTEXT without a new type
5. **No `hidden` modifier** — types like vector/binary should be hidden by default but there's no mechanism
6. **`title_field` cannot format** — dropdown labels limited to single field, no "{data.code} - {data.name}" support
7. **No `display_field` override** — every many2one uses target model's `title_field`, no per-field override
8. **`currency_field` missing** — no way to reference dynamic currency from another field in the same record
9. **JSON has no variants** — `json` cannot distinguish object vs array for default values and UI widgets
10. **No plural table naming** — Laravel developers expect `contacts` table, engine produces `contact`. No config to switch.
11. **Duplicate model name = silent override** — two files with same `"name"` in one module silently overwrites, no error

### 1.2 Success Criteria

- All 33+ field types have explicit SQL mapping for PostgreSQL, MySQL, SQLite
- All field types have explicit MongoDB mapping
- New types (uuid, ip, ipv6, year, vector, binary) fully functional
- `storage` hint works for integer, decimal, text, binary, string
- `title_field` supports format engine templates
- `display_field` works on many2one fields
- `json:object` and `json:array` variants parsed and handled
- Precision/scale backward compatible
- Plural table naming configurable (project + per-model)
- Duplicate model name in same module = startup error
- Zero breaking changes to existing JSON definitions

### 1.3 What This Phase Does NOT Do

- Does not add polymorphic relations (Phase 6B)
- Does not add `visible_if`, `disabled_if` to LayoutRow (Phase 6C)
- Does not change runtime/bridge API (Phase 1-5)
- Does not change view system beyond `display_field` (Phase 6C)

---

## 2. Current State Audit

### 2.1 Existing Field Types (33 total)

```
Core:        string, text, integer, decimal, boolean, date, datetime, selection, json, file, computed
Relation:    many2one, one2many, many2many, dynamic_link
String-UI:   email, password, barcode, color
Text-UI:     smalltext, richtext, markdown, html, code
Number-UI:   float, currency, percent, rating
Boolean-UI:  toggle, radio
Date-UI:     time, duration
File-UI:     image, signature
Special:     geolocation
```

### 2.2 Types with Explicit SQL Mapping (13)

These types have proper `case` in `fieldTypeToSQL()` (`dynamic_model.go:206-282`):

| Type | PostgreSQL | MySQL | SQLite |
|------|-----------|-------|--------|
| `string` | VARCHAR(255) / VARCHAR(max) | VARCHAR(255) / VARCHAR(max) | TEXT |
| `text` | TEXT | TEXT | TEXT |
| `integer` | INTEGER | INTEGER | INTEGER |
| `decimal` | DECIMAL(18,2) / DECIMAL(18,p) | DECIMAL(18,2) / DECIMAL(18,p) | REAL |
| `boolean` | BOOLEAN | BOOLEAN | INTEGER |
| `date` | DATE | DATE | TEXT |
| `datetime` | TIMESTAMPTZ | DATETIME | TEXT |
| `selection` | VARCHAR(50) | VARCHAR(50) | TEXT |
| `email` | VARCHAR(255) | VARCHAR(255) | TEXT |
| `many2one` | UUID | CHAR(36) | TEXT |
| `json` | JSONB | JSON | TEXT |
| `file` | VARCHAR(500) | VARCHAR(500) | TEXT |
| `one2many/many2many/computed` | *(no column)* | *(no column)* | *(no column)* |

### 2.3 Types Falling to `default: TEXT` (20)

**These all become TEXT regardless of database — this is wrong:**

| Type | Should Be | Current |
|------|----------|---------|
| `float` | DOUBLE PRECISION / DOUBLE / REAL | TEXT ❌ |
| `currency` | DECIMAL(18,2) | TEXT ❌ |
| `percent` | DECIMAL(5,2) | TEXT ❌ |
| `rating` | SMALLINT | TEXT ❌ |
| `time` | TIME | TEXT ❌ |
| `duration` | INTEGER (seconds) | TEXT ❌ |
| `toggle` | BOOLEAN | TEXT ❌ |
| `radio` | VARCHAR(50) | TEXT ❌ |
| `password` | VARCHAR(255) | TEXT ❌ |
| `smalltext` | VARCHAR(500) | TEXT ❌ |
| `richtext` | TEXT | TEXT ✅ (accidental) |
| `markdown` | TEXT | TEXT ✅ (accidental) |
| `html` | TEXT | TEXT ✅ (accidental) |
| `code` | TEXT | TEXT ✅ (accidental) |
| `image` | VARCHAR(500) | TEXT ❌ |
| `signature` | TEXT | TEXT ✅ (accidental) |
| `barcode` | VARCHAR(255) | TEXT ❌ |
| `color` | VARCHAR(7) | TEXT ❌ |
| `geolocation` | JSONB / JSON | TEXT ❌ |
| `dynamic_link` | VARCHAR(255) | TEXT ❌ |

**13 types have wrong storage. 7 are accidentally correct because TEXT happens to work.**

### 2.4 FieldDefinition Struct (current)

```go
type FieldDefinition struct {
    Type         FieldType         // ✅
    Label        string            // ✅
    Required     bool              // ✅
    Unique       bool              // ✅
    Default      any               // ✅
    Max          int               // ✅
    Min          int               // ✅
    Precision    int               // ⚠️ actually means scale
    MaxSize      string            // ✅
    Options      []string          // ✅
    Model        string            // ✅
    Inverse      string            // ✅
    Computed     string            // ✅
    Auto         bool              // ✅
    Widget       string            // ✅
    DependsOn    string            // ✅
    ReadOnlyIf   string            // ✅
    MandatoryIf  string            // ✅
    FetchFrom    string            // ✅
    Formula      string            // ✅
    Language     string            // ✅
    Toolbar      string            // ✅
    CurrencyCode string            // ✅
    Format       string            // ✅
    DrawMode     string            // ✅
    MaxStars     int               // ✅
    HalfStars    bool              // ✅
    Rows         int               // ✅
    Accept       string            // ✅
    Multiple     bool              // ✅
    PathFormat   string            // ✅
    NameFormat   string            // ✅
    AutoFormat   *AutoFormatConfig // ✅
    Encrypted    bool              // ✅
    Validation   *FieldValidation  // ✅
    Sanitize     []string          // ✅
    Mask         bool              // ✅
    MaskLength   int               // ✅
    Groups       []string          // ✅
    // MISSING: Hidden, Storage, Scale, CurrencyField, DisplayField, Dimensions
}
```

---

## 3. Design Philosophy: Two-Layer Type System

### 3.1 The Two Layers

```
Layer 1: SEMANTIC TYPE (what the developer writes)
  → Answers: "What is this field FOR?"
  → Examples: email, currency, rating, vector
  → Determines: validation, UI widget, default behavior

Layer 2: STORAGE (what goes into the database)
  → Answers: "How is this stored?"
  → Determined by: semantic type + optional storage hint
  → Examples: VARCHAR(255), DECIMAL(18,2), JSONB, BYTEA
```

### 3.2 Rules

1. **Semantic type is mandatory** — every field must have a `type`
2. **Storage is automatic** — engine picks the best storage for each type + database
3. **Storage hint is optional** — power users can override: `"storage": "bigint"`
4. **Semantic types that are well-known get flat names** — `email`, `uuid`, `ip`, `vector`
5. **Variants only for genuine sub-types** — `json:object`, `json:array`
6. **Storage concerns never become types** — no `bigint` type, no `longtext` type, no `numeric` type

### 3.3 Why Not `type:variant` for Everything?

We considered `integer:big`, `text:long`, `string:uuid` etc. Rejected because:

- `integer:big` is a **storage concern**, not semantic — developer shouldn't think about INT vs BIGINT
- `text:long` is a **storage concern** — engine should auto-handle based on content
- `string:uuid` — UUID is its own concept (format, generation, PK strategy), not a string variant
- `string:ip` — IP has its own validation, format, and meaning

**Exception**: `json:object` and `json:array` ARE genuine variants because they affect:
- Default value (`{}` vs `[]` vs `null`)
- UI widget (key-value editor vs list editor vs code editor)
- Validation (must be object vs must be array vs any)

---

## 4. New Semantic Types

### 4.1 New Types to Add

| Type | Semantic Meaning | Default Storage (PG) | Default Storage (MySQL) | Default Storage (SQLite) |
|------|-----------------|---------------------|------------------------|-------------------------|
| `uuid` | Universally unique identifier | UUID | CHAR(36) | TEXT |
| `ip` | IP address (IPv4 or IPv6) | VARCHAR(45) | VARCHAR(45) | TEXT |
| `ipv6` | Strict IPv6 only | VARCHAR(45) | VARCHAR(45) | TEXT |
| `year` | Calendar year (1900-2300) | SMALLINT | SMALLINT | INTEGER |
| `vector` | Embedding vector for similarity search | vector(N) | JSON | TEXT |
| `binary` | Raw binary data | BYTEA | LONGBLOB | BLOB |

### 4.2 New FieldType Constants

```go
const (
    // ... existing constants ...

    // New types
    FieldUUID       FieldType = "uuid"
    FieldIP         FieldType = "ip"
    FieldIPv6       FieldType = "ipv6"
    FieldYear       FieldType = "year"
    FieldVector     FieldType = "vector"
    FieldBinary     FieldType = "binary"

    // JSON variants (parsed from "json:object", "json:array")
    FieldJSONObject FieldType = "json:object"
    FieldJSONArray  FieldType = "json:array"
)
```

### 4.3 Type Details

#### `uuid`

```json
{ "token": { "type": "uuid" } }
```

- **Storage**: Native UUID (PostgreSQL), CHAR(36) (MySQL), TEXT (SQLite)
- **Validation**: Must match UUID format (8-4-4-4-12 hex)
- **Default**: Can be auto-generated (`"default": "uuid4"` or `"default": "uuid7"`)
- **Index**: Not automatic — follows normal index rules. PK auto-indexes. FK auto-indexes via relation.
- **Use cases**: Tokens, external IDs, API keys, non-PK unique identifiers
- **Note**: For UUID as primary key, use `"primary_key": {"strategy": "uuid"}` — that's already implemented

#### `ip`

```json
{ "client_ip": { "type": "ip" } }
```

- **Storage**: VARCHAR(45) — enough for both IPv4 ("192.168.1.1") and IPv6 ("2001:0db8:85a3::8a2e:0370:7334")
- **Validation**: Accept both IPv4 and IPv6 format. Use `ip:v4` for strict IPv4, `ip:v6` for strict IPv6.
- **Parsed variants**: `ip` → accept both, `ip:v4` → IPv4 only (VARCHAR(15)), `ip:v6` → IPv6 only (VARCHAR(45))
- **UI widget**: Text input with IP format hint

#### `ipv6`

```json
{ "server_ip": { "type": "ipv6" } }
```

- **Alias**: Equivalent to `ip:v6`. Both are accepted by parser, both resolve to `FieldIPv6`.
- **Storage**: VARCHAR(45)
- **Validation**: Strict IPv6 format only

#### `year`

```json
{ "fiscal_year": { "type": "year" } }
```

- **Storage**: SMALLINT (PostgreSQL/MySQL), INTEGER (SQLite)
- **Validation**: Range 1900-2300
- **UI widget**: Year picker (number input with min/max)
- **Use cases**: Fiscal year, birth year, graduation year

#### `vector`

```json
{
  "embedding": {
    "type": "vector",
    "dimensions": 1536
  }
}
```

- **`dimensions` is REQUIRED** — parser error if missing
- **Storage**: `vector(N)` (PostgreSQL with pgvector), JSON (MySQL), TEXT (SQLite)
- **Index**: Not automatic (default: no index). Explicit opt-in:
  - `"index": true` → HNSW (default, best for read-heavy production)
  - `"index": "hnsw"` → explicit HNSW
  - `"index": "ivfflat"` → explicit IVFFlat (faster build, slower query)
  - `"index": false` or omitted → no index (full scan)
- **PostgreSQL index SQL**: `CREATE INDEX ... USING hnsw (field vector_cosine_ops)`
- **MySQL/SQLite**: Index ignored (no vector index support), WARNING logged
- **MongoDB**: Not stored as special type — stored as array of numbers. Index via Atlas Vector Search (external).
- **Default hidden**: `true` — vector fields are infrastructure, not user-facing
- **UI widget**: None (hidden). If explicitly shown: code editor displaying JSON array.
- **Validation**: Must be array of numbers with exactly `dimensions` elements

#### `binary`

```json
{ "thumbnail_data": { "type": "binary" } }
```

- **Storage**: BYTEA (PostgreSQL), LONGBLOB (MySQL), BLOB (SQLite)
- **Default hidden**: `true` — raw bytes are not user-facing
- **UI widget**: None (hidden). If explicitly shown: file download link.
- **Use cases**: Cached thumbnails, serialized data, encryption blobs
- **Note**: For user-uploaded files, use `file` or `image` type instead — those use file storage (local/S3), not database BLOB.

---

## 5. Storage Hint System

### 5.1 New Field in FieldDefinition

```go
type FieldDefinition struct {
    // ... existing fields ...
    Storage string `json:"storage,omitempty"`
}
```

### 5.2 Valid Storage Hints per Type

| Semantic Type | Valid `storage` Values | Default |
|--------------|----------------------|---------|
| `integer` | `"smallint"`, `"bigint"` | `"int"` |
| `decimal` | `"numeric"`, `"double"` | `"decimal"` |
| `float` | `"double"`, `"real"` | `"double"` |
| `text` | `"mediumtext"`, `"longtext"` | `"text"` |
| `string` | `"char"` | `"varchar"` |
| `binary` | `"mediumblob"`, `"longblob"` | `"bytea"` / `"longblob"` |
| `datetime` | `"naive"` | `"timestamptz"` |

**Any other type**: `storage` hint is **ignored with WARNING**. Types like `email`, `currency`, `rating` etc. have fixed storage — overriding makes no sense.

**Invalid storage value**: Parser error. `"storage": "varchar"` on an `integer` field → error.

### 5.3 Storage Mapping Table

#### `integer` + storage hint

| `storage` | PostgreSQL | MySQL | SQLite | MongoDB |
|-----------|-----------|-------|--------|---------|
| *(default)* | INTEGER | INT | INTEGER | NumberInt |
| `"smallint"` | SMALLINT | SMALLINT | INTEGER | NumberInt |
| `"bigint"` | BIGINT | BIGINT | INTEGER | NumberLong |

#### `decimal` + storage hint

| `storage` | PostgreSQL | MySQL | SQLite | MongoDB |
|-----------|-----------|-------|--------|---------|
| *(default)* | NUMERIC(p,s) | DECIMAL(p,s) | REAL | Decimal128 |
| `"numeric"` | NUMERIC *(unlimited)* | DECIMAL(65,30) | TEXT | Decimal128 |
| `"double"` | DOUBLE PRECISION | DOUBLE | REAL | Double |

#### `text` + storage hint

| `storage` | PostgreSQL | MySQL | SQLite | MongoDB |
|-----------|-----------|-------|--------|---------|
| *(default)* | TEXT | TEXT | TEXT | String |
| `"mediumtext"` | TEXT *(PG has no limit)* | MEDIUMTEXT | TEXT | String |
| `"longtext"` | TEXT *(PG has no limit)* | LONGTEXT | TEXT | String |

#### `datetime` + storage hint

| `storage` | PostgreSQL | MySQL | SQLite | MongoDB |
|-----------|-----------|-------|--------|---------|
| *(default)* | TIMESTAMPTZ | DATETIME | TEXT | Date |
| `"naive"` | TIMESTAMP *(without tz)* | DATETIME | TEXT | Date |

### 5.4 Examples

```json
// Large counter (>2 billion)
{ "type": "integer", "storage": "bigint" }

// PostgreSQL unlimited precision for financial calc
{ "type": "decimal", "storage": "numeric" }

// Currency with unlimited precision
{ "type": "currency", "storage": "numeric" }

// MySQL LONGTEXT for huge content
{ "type": "text", "storage": "longtext" }

// Timezone-naive datetime (rare, explicit opt-out)
{ "type": "datetime", "storage": "naive" }
```

### 5.5 Auto-Detection (Engine Optimization)

Engine MAY auto-optimize storage based on field constraints, but MUST NOT change developer-specified storage hints:

```
rating + max_stars: 5          → engine CAN use SMALLINT internally
integer + min: 0, max: 255     → engine CAN use SMALLINT internally
integer + storage: "bigint"    → engine MUST use BIGINT (explicit)
```

Auto-detection is an internal optimization — it does not change the `storage` field value and is not visible to the developer.

---

## 6. Field Modifiers

### 6.1 New Fields in FieldDefinition

```go
type FieldDefinition struct {
    // ... existing fields ...

    // New modifiers
    Hidden        bool              `json:"hidden,omitempty"`         // Field exists in DB but never shown in UI by default
    Storage       string            `json:"storage,omitempty"`        // Storage hint (see §5)
    Scale         int               `json:"scale,omitempty"`          // Decimal places (new, proper name)
    DisplayField  string            `json:"display_field,omitempty"`  // Override title_field for many2one (see §8)
    CurrencyField string            `json:"currency_field,omitempty"` // Dynamic currency reference (see §9)
    Dimensions    int               `json:"dimensions,omitempty"`     // Vector dimensions (required for vector type)
    Index         any               `json:"index,omitempty"`          // true/false/"hnsw"/"ivfflat" (for vector)
    Mask          *FieldMaskConfig  `json:"mask,omitempty"`           // Number/currency input mask override (see §9.5)
}
```

### 6.2 `hidden` Modifier

**Model level** — field exists in database but is not shown in UI by default.

```json
{
  "embedding": {
    "type": "vector",
    "dimensions": 1536,
    "hidden": true
  }
}
```

- `hidden: true` → field is excluded from auto-generated views (list, form)
- `hidden: true` → field is still accessible via API, bridge, scripts
- `hidden: true` → can be overridden in view layout by explicitly including the field
- Some types have `hidden: true` as default (see §15)

### 6.3 Modifier Summary (Model Level)

| Modifier | Type | Existing? | Description |
|----------|------|:---------:|-------------|
| `required` | `bool` | ✅ | Field must have value |
| `unique` | `bool` | ✅ | Value must be unique |
| `hidden` | `bool` | **NEW** | Not shown in UI by default |
| `readonly_if` | `string` | ✅ | Conditional readonly (expression) |
| `mandatory_if` | `string` | ✅ | Conditional required (expression) |
| `depends_on` | `string` | ✅ | Field visibility depends on condition |
| `groups` | `[]string` | ✅ | Field visible only to these groups |
| `encrypted` | `bool` | ✅ | AES-256 encrypted at rest |
| `storage` | `string` | **NEW** | Storage hint override |
| `display_field` | `string` | **NEW** | Override title_field for many2one |
| `currency_field` | `string` | **NEW** | Dynamic currency from another field |
| `dimensions` | `int` | **NEW** | Vector dimensions (required for vector) |
| `index` | `any` | **NEW** | Vector index type |
| `mask` | `*FieldMaskConfig` | **NEW** | Number/currency input mask override |

---

## 7. Title Field Format Engine Integration

### 7.1 Current Behavior

```go
// model.go — title_field is a single field name
TitleField string `json:"title_field,omitempty"`

// Auto-resolve: name → label → title → code → username → description → first string field → "id"
```

### 7.2 New Behavior

`title_field` now supports **two modes**:

**Mode 1: Simple field name (existing, no change)**
```json
{ "title_field": "name" }
```
→ Display label = `record["name"]`

**Mode 2: Format template (new)**
```json
{ "title_field": "{data.code} - {data.name}" }
```
→ Display label = `"IDR - Indonesian Rupiah"` (resolved via format engine)

**Detection**: If `title_field` contains `{`, treat as format template. Otherwise, treat as field name.

### 7.3 Format Engine Capabilities (already implemented in `runtime/format/engine.go`)

```
{data.field_name}              → value from record
{session.user_name}            → value from session
{setting.key}                  → value from settings
{time.year}                    → current year
{upper(data.code)}             → uppercase
{lower(data.name)}             → lowercase
{substring(data.name, 0, 3)}   → first 3 chars
{hash(data.email)}             → SHA256 hash (first 8 hex)
```

### 7.4 `search_field` Auto-Extraction

When `search_field` is not explicitly set:

```
title_field is simple ("name")
  → search_field = ["name"]

title_field is format ("{data.code} - {data.name}")
  → extract all {data.xxx} tokens (including inside functions like {upper(data.code)})
  → filter: only fields with string/text storage family
  → search_field = ["code", "name"]
```

**Extraction rules:**
- `{data.code}` → extract `"code"`
- `{upper(data.code)}` → unwrap function, extract `"code"`
- `{substring(data.name, 0, 3)}` → unwrap function, extract `"name"`
- `{time.year}` → skip (not a data field)
- `{session.user}` → skip (not a data field)
- `{sequence(5)}` → skip (not a data field)

**Filter rules:**
- Field type is string, text, email, password, barcode, color, code, smalltext, richtext, markdown, html, ip, ipv6, uuid → **include** (searchable text)
- Field type is integer, decimal, float, currency, percent, boolean, date, json, vector, binary → **exclude** (not text-searchable in LIKE query)

**No cap on count** — if format has 5 data fields that are all string/text, search_field gets all 5.

**Fallback**: If extraction yields zero searchable fields → fall back to `resolveTitleField()` auto-detect.

### 7.5 Limitation: No Dot-Notation for Relations

```json
// ❌ NOT SUPPORTED
{ "title_field": "{data.currency_id.code} - {data.name}" }
```

`{data.currency_id}` resolves to the FK value (integer/UUID), not the related record's fields. To display related data in title, use `fetch_from` to denormalize:

```json
{
  "currency_code": {
    "type": "string",
    "fetch_from": "currency_id.code"
  }
}
```

Then: `{ "title_field": "{data.currency_code} - {data.name}" }`

This is a documented limitation. Dot-notation in title_field format may be added in a future phase if there is demand.

### 7.6 Warning Log

If `title_field` auto-resolves to `"id"` (no string/text field found), engine logs:

```
WARN: model 'xxx' has no title_field and no string/text field — dropdown will show record ID
```

---

## 8. Many2One Display: `display_field` Override

### 8.1 Problem

Every many2one field uses the target model's `title_field` for dropdown label. But sometimes the same model is referenced from different places with different display needs:

```json
// In invoice form: want to show "IDR - Indonesian Rupiah"
// In compact widget: want to show just "Rp" (symbol)
```

### 8.2 Solution: `display_field` on FieldDefinition

```json
{
  "currency_id": {
    "type": "many2one",
    "model": "base.currency",
    "display_field": "symbol"
  }
}
```

### 8.3 Behavior

```
Dropdown label resolution for many2one:
1. Field has display_field?     → use that field/format from target record
2. Target model has title_field? → use title_field (simple or format)
3. Auto-detect                   → name → label → title → code → ... → id
```

`display_field` supports the same two modes as `title_field`:

```json
// Simple field name
{ "display_field": "symbol" }
// → label = related_record["symbol"]

// Format template
{ "display_field": "{data.code} ({data.symbol})" }
// → label = "IDR (Rp)"
```

### 8.4 Validation

- `display_field` is only valid on `many2one` fields. Parser warning if set on other types.
- `display_field` references fields on the **target model**, not the current model.
- If `display_field` references a non-existent field on target model → runtime warning, fallback to `title_field`.

---

## 9. Currency Field Enhancement

### 9.1 New Field: `currency_field`

```go
type FieldDefinition struct {
    CurrencyCode  string `json:"currency,omitempty"`        // existing: fixed currency
    CurrencyField string `json:"currency_field,omitempty"`  // NEW: dynamic currency from another field
}
```

### 9.2 Three Modes

**Mode 1: Fixed currency**
```json
{
  "price": {
    "type": "currency",
    "currency": "IDR"
  }
}
```
→ Always formatted as IDR. UI shows "Rp" symbol.

**Mode 2: Dynamic currency (references another field)**
```json
{
  "currency_id": {
    "type": "many2one",
    "model": "base.currency"
  },
  "amount": {
    "type": "currency",
    "currency_field": "currency_id"
  }
}
```
→ Currency determined per-record from `currency_id` field. Engine resolves via relation to get symbol, decimal places, etc.

**Mode 3: Default (no currency specified)**
```json
{
  "amount": {
    "type": "currency"
  }
}
```
→ Fallback resolution hierarchy:
1. `session.currency_id` — user's preferred currency (from user profile/session)
2. Project config: `locale.currency` in `bitcode.toml`
3. Hardcoded: `"USD"`

**Session currency**: If the `user` model has a `currency_id` field (many2one to currency table), the engine reads it from the session on login. This enables per-user currency preference in multi-country SaaS apps:

```
User A (Indonesia) → session.currency_id = "IDR" → sees Rp 1.500.000
User B (US)        → session.currency_id = "USD" → sees $1,500.00
Same page, same field, different formatting.
```

This pattern applies generically to all locale preferences:

```
Resolution hierarchy (generic):
  Field-level     → explicit config on the field definition
  Record-level    → currency_field (from another field in same record)
  Session-level   → user preference (session.currency_id, session.timezone, etc.)
  Project-level   → bitcode.toml config (locale.currency, locale.timezone)
  Hardcoded       → engine default ("USD", "UTC")
```

### 9.3 Storage

Currency type storage is **always DECIMAL-family**, regardless of mode:

| `storage` hint | PostgreSQL | MySQL | SQLite | MongoDB |
|---------------|-----------|-------|--------|---------|
| *(default)* | NUMERIC(18,2) | DECIMAL(18,2) | REAL | Decimal128 |
| `"numeric"` | NUMERIC *(unlimited)* | DECIMAL(65,30) | TEXT | Decimal128 |

### 9.4 Validation

- `currency` and `currency_field` are mutually exclusive on the same field. Parser error if both set.
- `currency_field` must reference a field that exists in the same model. Parser warning if not found (may be added by inheritance).
- `currency_field` typically references a `many2one` to a currency model, but can also reference a `selection` or `string` field containing a currency code.

### 9.5 Number/Currency Formatting

#### Three-Layer Formatting System

**Layer 1: Currency table** (for `currency` type fields)

The `base.currency` model should include formatting fields:

```json
{
  "name": "currency",
  "source": "array",
  "primary_key": { "strategy": "natural_key", "field": "code" },
  "fields": {
    "code":               { "type": "string", "max": 3 },
    "name":               { "type": "string" },
    "symbol":             { "type": "string", "max": 5 },
    "decimals":           { "type": "integer", "default": 2 },
    "thousand_separator": { "type": "string", "max": 1, "default": "," },
    "decimal_separator":  { "type": "string", "max": 1, "default": "." },
    "symbol_position":    { "type": "selection", "options": ["before", "after"], "default": "before" }
  },
  "rows": [
    { "code": "IDR", "name": "Indonesian Rupiah", "symbol": "Rp", "decimals": 0, "thousand_separator": ".", "decimal_separator": ",", "symbol_position": "before" },
    { "code": "USD", "name": "US Dollar", "symbol": "$", "decimals": 2, "thousand_separator": ",", "decimal_separator": ".", "symbol_position": "before" },
    { "code": "EUR", "name": "Euro", "symbol": "€", "decimals": 2, "thousand_separator": ".", "decimal_separator": ",", "symbol_position": "before" }
  ]
}
```

Currency fields resolve formatting from this table automatically. Developer does not need to specify separators per field.

**Layer 2: Project locale config** (for non-currency numeric fields)

```toml
# bitcode.toml
[locale]
currency = "IDR"
number_format = "id-ID"          # BCP 47 locale tag — auto-resolve separators
# OR explicit:
thousand_separator = "."
decimal_separator = ","
```

If `number_format` is set, `thousand_separator` and `decimal_separator` are auto-resolved from the locale. Explicit values override auto-resolution.

Applies to: `integer`, `decimal`, `float`, `percent`, `rating` — any numeric field that is NOT `currency`.

**Layer 3: Per-field mask override** (rare, for special cases)

```go
type FieldMaskConfig struct {
    ThousandSeparator string `json:"thousand_separator,omitempty"`
    DecimalSeparator  string `json:"decimal_separator,omitempty"`
    Precision         int    `json:"precision,omitempty"`
    Prefix            string `json:"prefix,omitempty"`   // e.g., "$", "Rp "
    Suffix            string `json:"suffix,omitempty"`   // e.g., "%", " kg"
}
```

```json
{
  "weight": {
    "type": "decimal",
    "scale": 3,
    "mask": {
      "thousand_separator": ",",
      "precision": 3,
      "suffix": " kg"
    }
  }
}
```

#### Formatting Resolution

For `currency` type:
```
1. field.mask (per-field override)         → if set, use this
2. currency table (from resolved currency) → symbol, decimals, separators
3. locale config (project-level)           → fallback
```

For other numeric types (`decimal`, `float`, `integer`, `percent`):
```
1. field.mask (per-field override)         → if set, use this
2. locale config (project-level)           → thousand_separator, decimal_separator
3. hardcoded                               → thousand: ",", decimal: "."
```

#### Input Masking (Client-Side)

The `mask` config is also used for **live input masking** in form fields. The `bc-field-currency` and `bc-field-number` web components read the mask config and apply formatting as the user types:

```
User types: 1500000
Display:    1.500.000 (with thousand separator)
Stored:     1500000 (raw number in DB)
```

---

## 10. JSON Variants: `json`, `json:object`, `json:array`

### 10.1 Parsing

The parser splits `type` on `:` to detect variants:

```go
func parseFieldType(raw string) (FieldType, error) {
    switch raw {
    case "json":        return FieldJSON, nil
    case "json:object": return FieldJSONObject, nil
    case "json:array":  return FieldJSONArray, nil
    case "ip:v4":       return FieldIP, nil       // ip:v4 is alias for ip (default)
    case "ip:v6":       return FieldIPv6, nil      // ip:v6 is alias for ipv6
    default:
        // Check if it contains ":" — only json and ip support variants
        if strings.Contains(raw, ":") {
            base := strings.SplitN(raw, ":", 2)[0]
            return "", fmt.Errorf("type %q does not support variants (only json and ip do)", base)
        }
        return FieldType(raw), nil
    }
}
```

### 10.2 Behavior Differences

| Aspect | `json` | `json:object` | `json:array` |
|--------|--------|--------------|-------------|
| Default value | `null` | `{}` | `[]` |
| Validation | Any valid JSON | Must be object | Must be array |
| UI widget | Code editor | Key-value editor | List editor |
| Storage | JSONB / JSON / TEXT | Same | Same |

### 10.3 Storage

All three variants have identical storage — the difference is only in validation, default value, and UI widget.

---

## 11. Precision/Scale Backward Compatibility Fix

### 11.1 The Problem

Current `precision` field in FieldDefinition actually means **scale** (number of decimal places):

```go
// Current behavior (WRONG naming):
if field.Precision > 0 {
    return fmt.Sprintf("DECIMAL(18,%d)", field.Precision)  // precision used as scale
}
```

### 11.2 The Fix

Add proper `scale` field. Keep `precision` for backward compatibility:

```go
type FieldDefinition struct {
    Precision int `json:"precision,omitempty"` // Total digits (NEW meaning) OR legacy scale (backward compat)
    Scale     int `json:"scale,omitempty"`     // NEW: decimal places
}
```

### 11.3 Resolution Logic

```go
func resolveDecimalPrecision(field FieldDefinition) (totalDigits int, decimalPlaces int) {
    if field.Scale > 0 {
        // New style: both precision and scale specified properly
        p := field.Precision
        if p == 0 {
            p = 18 // default total digits
        }
        return p, field.Scale
    }
    if field.Precision > 0 {
        // Legacy: precision actually means scale (backward compat)
        return 18, field.Precision
    }
    // Default
    return 18, 2
}
```

### 11.4 Examples

```json
// Legacy (still works, backward compat)
{ "type": "decimal", "precision": 4 }
// → DECIMAL(18,4) — precision treated as scale

// New style (recommended)
{ "type": "decimal", "precision": 10, "scale": 4 }
// → DECIMAL(10,4)

// Default
{ "type": "decimal" }
// → DECIMAL(18,2)

// Unlimited (PostgreSQL)
{ "type": "decimal", "storage": "numeric" }
// → NUMERIC (no precision/scale)
```

---

## 12. Complete 4-Database SQL Mapping

### 12.1 Full Mapping Table

This is the **complete, authoritative** mapping for all field types.

#### Core Types

| Type | PostgreSQL | MySQL | SQLite | Notes |
|------|-----------|-------|--------|-------|
| `string` | VARCHAR(max\|255) | VARCHAR(max\|255) | TEXT | `max` sets length |
| `text` | TEXT | TEXT | TEXT | |
| `integer` | INTEGER | INT | INTEGER | |
| `decimal` | NUMERIC(p,s) | DECIMAL(p,s) | REAL | Default (18,2) |
| `float` | DOUBLE PRECISION | DOUBLE | REAL | IEEE 754 approximate |
| `boolean` | BOOLEAN | BOOLEAN | INTEGER | |
| `date` | DATE | DATE | TEXT | |
| `datetime` | TIMESTAMPTZ | DATETIME | TEXT | TZ-aware by default |
| `time` | TIME | TIME | TEXT | |
| `json` | JSONB | JSON | TEXT | |
| `json:object` | JSONB | JSON | TEXT | Default `{}` |
| `json:array` | JSONB | JSON | TEXT | Default `[]` |

#### String-Family Semantic Types

| Type | PostgreSQL | MySQL | SQLite | Notes |
|------|-----------|-------|--------|-------|
| `email` | VARCHAR(255) | VARCHAR(255) | TEXT | Email validation |
| `password` | VARCHAR(255) | VARCHAR(255) | TEXT | Masked UI |
| `barcode` | VARCHAR(255) | VARCHAR(255) | TEXT | |
| `color` | VARCHAR(7) | VARCHAR(7) | TEXT | `#RRGGBB` |
| `uuid` | UUID | CHAR(36) | TEXT | Native UUID on PG |
| `ip` | VARCHAR(45) | VARCHAR(45) | TEXT | IPv4 + IPv6 |
| `ipv6` | VARCHAR(45) | VARCHAR(45) | TEXT | Strict IPv6 |
| `smalltext` | VARCHAR(500) | VARCHAR(500) | TEXT | |
| `dynamic_link` | VARCHAR(255) | VARCHAR(255) | TEXT | |

#### Text-Family Semantic Types

| Type | PostgreSQL | MySQL | SQLite | Notes |
|------|-----------|-------|--------|-------|
| `richtext` | TEXT | TEXT | TEXT | |
| `markdown` | TEXT | TEXT | TEXT | |
| `html` | TEXT | TEXT | TEXT | |
| `code` | TEXT | TEXT | TEXT | |

#### Number-Family Semantic Types

| Type | PostgreSQL | MySQL | SQLite | Notes |
|------|-----------|-------|--------|-------|
| `currency` | NUMERIC(18,2) | DECIMAL(18,2) | REAL | Respects precision/scale |
| `percent` | NUMERIC(5,2) | DECIMAL(5,2) | REAL | 0.00 - 100.00 |
| `rating` | SMALLINT | SMALLINT | INTEGER | 1-N stars |
| `year` | SMALLINT | SMALLINT | INTEGER | 1900-2300 |
| `duration` | INTEGER | INT | INTEGER | Stored as seconds |

#### Boolean-Family Semantic Types

| Type | PostgreSQL | MySQL | SQLite | Notes |
|------|-----------|-------|--------|-------|
| `toggle` | BOOLEAN | BOOLEAN | INTEGER | Same as boolean, different UI |
| `radio` | VARCHAR(50) | VARCHAR(50) | TEXT | Same as selection, different UI |

#### Relation Types

| Type | PostgreSQL | MySQL | SQLite | Notes |
|------|-----------|-------|--------|-------|
| `many2one` | UUID | CHAR(36) | TEXT | FK to related table |
| `one2many` | *(no column)* | *(no column)* | *(no column)* | Virtual |
| `many2many` | *(no column)* | *(no column)* | *(no column)* | Junction table |
| `selection` | VARCHAR(50) | VARCHAR(50) | TEXT | |

#### File Types

| Type | PostgreSQL | MySQL | SQLite | Notes |
|------|-----------|-------|--------|-------|
| `file` | VARCHAR(500) | VARCHAR(500) | TEXT | Path to file |
| `image` | VARCHAR(500) | VARCHAR(500) | TEXT | Path to image |
| `signature` | TEXT | TEXT | TEXT | Base64 or path |

#### Special Types

| Type | PostgreSQL | MySQL | SQLite | Notes |
|------|-----------|-------|--------|-------|
| `geolocation` | JSONB | JSON | TEXT | `{lat, lng}` |
| `vector` | vector(N) | JSON | TEXT | Requires pgvector extension |
| `binary` | BYTEA | LONGBLOB | BLOB | Raw bytes |
| `computed` | *(no column)* | *(no column)* | *(no column)* | Virtual |

---

## 13. MongoDB Mapping

MongoDB is schemaless — types are enforced at application level, not database level. However, the engine uses appropriate BSON types for optimal storage and querying.

### 13.1 MongoDB Type Mapping

| Semantic Type | MongoDB/BSON Type | Notes |
|--------------|------------------|-------|
| `string` | String | |
| `text` | String | |
| `integer` | NumberInt / NumberLong | NumberLong if `storage: "bigint"` |
| `decimal` | Decimal128 | Always Decimal128 for precision |
| `float` | Double | IEEE 754 |
| `boolean` | Boolean | |
| `date` | Date | Stored as ISODate |
| `datetime` | Date | Stored as ISODate (always UTC) |
| `time` | String | "HH:MM:SS" format |
| `json` | Object / Array | Native BSON, no special handling |
| `json:object` | Object | Validated as object |
| `json:array` | Array | Validated as array |
| `email` | String | |
| `password` | String | |
| `uuid` | String | Stored as string representation |
| `ip` | String | |
| `ipv6` | String | |
| `year` | NumberInt | |
| `vector` | Array | Array of Double |
| `binary` | BinData | BSON binary type |
| `currency` | Decimal128 | |
| `percent` | Decimal128 | |
| `rating` | NumberInt | |
| `duration` | NumberInt | Seconds |
| `geolocation` | Object | `{type: "Point", coordinates: [lng, lat]}` for GeoJSON |
| `color` | String | |
| `barcode` | String | |
| `selection` | String | |
| `many2one` | String / ObjectId | Depends on PK strategy of target |
| `file` | String | Path |
| `image` | String | Path |
| `signature` | String | |
| `computed` | *(not stored)* | |

### 13.2 MongoDB Index for Vector

MongoDB Atlas supports vector search via Atlas Vector Search index. This is **not created automatically** by the engine — it requires Atlas configuration. Engine logs INFO when vector field is detected on MongoDB:

```
INFO: model 'xxx' has vector field 'embedding' — create Atlas Vector Search index manually for similarity search
```

---

## 14. Validation Rules per Type

### 14.1 Built-in Validation (automatic, no config needed)

| Type | Validation | Error Message |
|------|-----------|---------------|
| `email` | RFC 5322 email format | "invalid email format" |
| `uuid` | UUID format (8-4-4-4-12 hex) | "invalid UUID format" |
| `ip` | IPv4 or IPv6 format | "invalid IP address" |
| `ip:v4` | IPv4 only | "invalid IPv4 address" |
| `ipv6` / `ip:v6` | IPv6 only | "invalid IPv6 address" |
| `year` | 1900 ≤ value ≤ 2300 | "year must be between 1900 and 2300" |
| `color` | `#RRGGBB` hex format | "invalid color format (expected #RRGGBB)" |
| `rating` | 1 ≤ value ≤ max_stars | "rating must be between 1 and {max_stars}" |
| `percent` | 0 ≤ value ≤ 100 (unless min/max override) | "percent must be between 0 and 100" |
| `vector` | Array of numbers, length = dimensions | "vector must have exactly {dimensions} dimensions" |
| `json:object` | Must be JSON object | "value must be a JSON object" |
| `json:array` | Must be JSON array | "value must be a JSON array" |
| `selection` / `radio` | Value must be in `options` | "invalid option: {value}" |

### 14.2 Validation + `storage` Interaction

Storage hints do NOT change validation rules. `{ "type": "integer", "storage": "bigint" }` still validates as integer — just stored in a bigger column.

### 14.3 Migration Validation

When a field type changes (e.g., `string` → `ip`), **existing data is NOT validated**. Validation applies only to new/updated records. This is by design — the developer is responsible for data migration.

---

## 15. Default Behavior per Type

### 15.1 Default Hidden

| Type | Default `hidden` | Reason |
|------|:----------------:|--------|
| `vector` | `true` | Infrastructure field, not user-facing |
| `binary` | `true` | Raw bytes, not user-facing |
| All others | `false` | User-facing by default |

### 15.2 Default Readonly

| Type | Default readonly | Reason |
|------|:----------------:|--------|
| `computed` | `true` | Computed fields are always readonly |
| All others | `false` | Editable by default |

### 15.3 Default Widget Mapping

| Type | Default Widget | Notes |
|------|---------------|-------|
| `string` | `bc-field-string` | Text input |
| `text` | `bc-field-text` | Textarea |
| `integer` | `bc-field-number` | Number input |
| `decimal` | `bc-field-number` | Number input with decimals |
| `float` | `bc-field-number` | Number input with decimals |
| `boolean` | `bc-field-checkbox` | Checkbox |
| `toggle` | `bc-field-toggle` | Toggle switch |
| `date` | `bc-field-date` | Date picker |
| `datetime` | `bc-field-datetime` | Datetime picker |
| `time` | `bc-field-time` | Time picker |
| `selection` | `bc-field-select` | Dropdown |
| `radio` | `bc-field-radio` | Radio buttons |
| `many2one` | `bc-field-link` | Search dropdown |
| `many2many` | `bc-field-tags` | Tag input |
| `email` | `bc-field-string` | Email input |
| `password` | `bc-field-string` | Password input (masked) |
| `currency` | `bc-field-currency` | Number + currency symbol |
| `percent` | `bc-field-percent` | Number + % |
| `rating` | `bc-field-rating` | Star rating |
| `color` | `bc-field-color` | Color picker |
| `file` | `bc-field-file` | File upload |
| `image` | `bc-field-image` | Image upload |
| `signature` | `bc-field-signature` | Signature pad |
| `barcode` | `bc-field-barcode` | Barcode scanner |
| `geolocation` | `bc-field-geo` | Map picker |
| `json` | `bc-field-json` | Code editor |
| `json:object` | `bc-field-json` | Code editor (could be key-value editor) |
| `json:array` | `bc-field-json` | Code editor (could be list editor) |
| `richtext` | `bc-field-richtext` | Rich text editor (Tiptap) |
| `markdown` | `bc-field-markdown` | Markdown editor |
| `html` | `bc-field-html` | HTML editor |
| `code` | `bc-field-code` | Code editor (CodeMirror) |
| `smalltext` | `bc-field-text` | Small textarea |
| `duration` | `bc-field-duration` | Duration input |
| `uuid` | `bc-field-string` | Text input (readonly if auto-generated) |
| `ip` | `bc-field-string` | Text input with IP hint |
| `ipv6` | `bc-field-string` | Text input with IPv6 hint |
| `year` | `bc-field-number` | Number input (min=1900, max=2300) |
| `vector` | *(hidden)* | Not rendered by default |
| `binary` | *(hidden)* | Not rendered by default |
| `dynamic_link` | `bc-field-dynlink` | Dynamic link selector |
| `computed` | *(varies)* | Based on computed result type |

---

## 16. Table Naming: Plural Convention

### 16.1 Problem

Laravel uses plural table names by default (`users`, `contacts`, `order_items`). Many developers coming from Laravel expect this convention. Current engine always uses singular (`user`, `contact`, `order_item`).

### 16.2 Naming Principle

**Model name (`"name"` in JSON) is ALWAYS singular** — it is the identity of the entity, not the table name. The engine handles pluralization automatically when configured.

```
"name": "contact"     ← identity (singular, always)
table:  "contacts"    ← derived (plural, when configured)
API:    /api/v1/crm/contacts  ← derived (plural, when configured)
```

This matches Laravel's own convention: `class Contact extends Model` (singular class) → `contacts` table (plural, auto-derived).

### 16.3 Configuration

#### Project-level (`bitcode.toml`)

```toml
[database]
table_naming = "plural"    # "singular" (default) | "plural"
```

#### Per-model override (`model.json`)

```json
{
  "name": "contact",
  "table": {
    "prefix": "crm",
    "plural": true
  }
}
```

Per-model `plural` overrides project-level `table_naming`. This allows mixing conventions:

```json
// Most models follow project default (plural)
// But this specific model stays singular:
{
  "name": "settings",
  "table": { "plural": false }
}
```

### 16.4 What Gets Pluralized

| Aspect | Singular mode | Plural mode | Notes |
|--------|:------------:|:-----------:|-------|
| Table name | `contact` | `contacts` | Core change |
| API path (auto CRUD) | `/api/v1/crm/contact` | `/api/v1/crm/contacts` | Follows table naming |
| GraphQL query | `contacts` | `contacts` | Already plural, no change |
| WebSocket event | `contact.created` | `contact.created` | Always singular (entity identity) |
| OpenAPI/Swagger | `/api/v1/crm/contact` | `/api/v1/crm/contacts` | Follows API path |
| Pages (auto) | `/crm/contact` | `/crm/contacts` | Follows table naming |
| Model registry key | `contact` | `contact` | Always singular (identity) |
| Bridge API | `bitcode.db.find("contact")` | `bitcode.db.find("contact")` | Always singular (identity) |
| many2one reference | `"model": "contact"` | `"model": "contact"` | Always singular (identity) |
| Process step model | `"model": "contact"` | `"model": "contact"` | Always singular (identity) |
| Junction table (m2m) | `contact_tag` | `contact_tag` | Always singular (not entity) |

**Rule**: Plural applies to **external-facing names** (table, URL, API). Internal references (registry, bridge, relations) are **always singular**.

### 16.5 Resolution Logic

```go
func ResolveTableName(model *ModelDefinition, moduleDef *ModuleDefinition, projectConfig *ProjectConfig) string {
    // 1. Explicit table name → as-is, no pluralization
    if model.TableName != "" {
        return model.TableName
    }

    name := model.Name

    // 2. Determine prefix
    prefix := ""
    if model.TablePrefix != nil {
        prefix = *model.TablePrefix
    } else if moduleDef != nil && moduleDef.Table != nil && moduleDef.Table.Prefix != "" {
        prefix = moduleDef.Table.Prefix
    }

    // 3. Determine plural
    shouldPlural := false
    if model.TablePlural != nil {
        shouldPlural = *model.TablePlural                      // per-model override
    } else if projectConfig != nil && projectConfig.TableNaming == "plural" {
        shouldPlural = true                                     // project default
    }

    // 4. Pluralize name (not prefix)
    if shouldPlural {
        name = inflection.Plural(name)
    }

    // 5. Apply prefix
    if prefix != "" {
        return prefix + "_" + name
    }
    return name
}
```

### 16.6 API Path Resolution

```go
func resolveAPIBasePath(model *ModelDefinition, moduleName string, projectConfig *ProjectConfig) string {
    // Explicit base_path → as-is
    if model.API != nil && model.API.BasePath != "" {
        return model.API.BasePath
    }

    name := model.Name

    // Plural follows same logic as table naming
    shouldPlural := false
    if model.TablePlural != nil {
        shouldPlural = *model.TablePlural
    } else if projectConfig != nil && projectConfig.TableNaming == "plural" {
        shouldPlural = true
    }

    if shouldPlural {
        name = inflection.Plural(name)
    }

    if moduleName != "" {
        return "/api/v1/" + moduleName + "/" + name
    }
    return "/api/" + name
}
```

### 16.7 Pluralization Library

Replace hand-rolled `pluralize()` (in `auto_api.go` and `graphql/schema.go`) with `jinzhu/inflection`:

```go
import "github.com/jinzhu/inflection"

inflection.Plural("contact")    // → "contacts"
inflection.Plural("person")     // → "people"
inflection.Plural("child")      // → "children"
inflection.Plural("category")   // → "categories"
inflection.Plural("status")     // → "statuses"
inflection.Plural("address")    // → "addresses"
inflection.Plural("order_item") // → "order_items"
```

This library is already a transitive dependency (GORM uses it), so no new dependency added.

### 16.8 GraphQL Double-Plural Fix

Current bug: `pluralizeModel()` in `graphql/schema.go` always pluralizes. If model name is already plural (shouldn't happen, but defensive), it double-pluralizes.

Fix: GraphQL always uses `inflection.Plural(model.Name)`. Since model name is always singular, this is always correct.

### 16.9 Examples

#### Project: `table_naming = "plural"`, module CRM with prefix "crm"

| Model name | Table | API | GraphQL |
|-----------|-------|-----|---------|
| `contact` | `crm_contacts` | `/api/v1/crm/contacts` | `contacts` |
| `lead` | `crm_leads` | `/api/v1/crm/leads` | `leads` |
| `order_item` | `crm_order_items` | `/api/v1/crm/order_items` | `orderItems` |
| `person` | `crm_people` | `/api/v1/crm/people` | `people` |
| `status` | `crm_statuses` | `/api/v1/crm/statuses` | `statuses` |

#### Project: `table_naming = "singular"` (default, backward compat)

| Model name | Table | API | GraphQL |
|-----------|-------|-----|---------|
| `contact` | `crm_contact` | `/api/v1/crm/contact` | `contacts` |
| `lead` | `crm_lead` | `/api/v1/crm/lead` | `leads` |

No change from current behavior. 100% backward compatible.

### 16.10 Backward Compatibility

- Default is `"singular"` — existing projects unchanged
- No config needed for current behavior
- `table_naming = "plural"` is opt-in
- Existing explicit `"table"` names are never modified

---

## 17. Duplicate Model Detection

### 17.1 Problem

Current `Registry.Register()` silently overwrites when two files in the same module have the same `"name"` property:

```go
// Current: silent overwrite
r.models[model.Name] = model  // map assignment, no duplicate check
```

If `models/contact.json` and `models/kontak.json` both have `"name": "contact"`, the last file loaded wins. No error, no warning. Developer loses data silently.

### 17.2 Solution: Error on Same-Module Duplicate

```go
func (r *Registry) Register(model *parser.ModelDefinition) error {
    r.mu.Lock()
    defer r.mu.Unlock()

    if model.Name == "" {
        return fmt.Errorf("model name is required")
    }

    key := model.Name
    if model.Module != "" {
        key = model.Module + "." + model.Name
    }

    if existing, ok := r.models[key]; ok {
        // Same module, not inheritance → ERROR
        if existing.Module == model.Module && model.Inherit == "" {
            return fmt.Errorf(
                "duplicate model name %q in module %q (check files in models/ directory for duplicate \"name\" property)",
                model.Name, model.Module,
            )
        }
    }

    // Cross-module same name → OK (qualified names disambiguate)
    if model.Module != "" {
        qualifiedKey := model.Module + "." + model.Name
        r.models[qualifiedKey] = model
        r.moduleNames[model.Name] = appendUnique(r.moduleNames[model.Name], model.Module)
    }

    r.models[model.Name] = model
    return nil
}
```

### 17.3 Behavior Matrix

| Scenario | Result |
|----------|--------|
| `crm/contact.json` (name: "contact") + `crm/kontak.json` (name: "contact") | **ERROR**: duplicate model name "contact" in module "crm" |
| `crm/contact.json` (name: "contact") + `hrm/contact.json` (name: "contact") | **OK**: `crm.contact` and `hrm.contact` coexist, `IsAmbiguous("contact")` returns true |
| `crm/contact.json` (name: "contact") + `crm/contact_ext.json` (name: "contact", inherit: "crm.contact") | **OK**: inheritance, not duplicate — extends existing model |
| `crm/contact.json` (name: "contact") + `crm/contact.json` updated (same file reloaded in dev mode) | **OK**: same file, same key — overwrite is expected during hot reload |

### 17.4 Error Message

The error message must be actionable — developer should immediately know what to fix:

```
ERROR: duplicate model name "contact" in module "crm"
  (check files in models/ directory for duplicate "name" property)
```

### 17.5 Hot Reload Consideration

In dev mode, files are reloaded on change. The registry is rebuilt from scratch on each reload cycle, so duplicate detection runs fresh each time. This is correct — if a developer renames a file but forgets to change the `"name"` property, they get an immediate error.

---

## 18. Migration & ALTER COLUMN

### 16.1 When ALTER COLUMN is Needed

Adding `storage` hint to an existing field requires column type change:

```json
// Before
{ "type": "integer" }           // → INT

// After
{ "type": "integer", "storage": "bigint" }  // → BIGINT
```

### 16.2 PostgreSQL

```sql
ALTER TABLE tablename ALTER COLUMN fieldname TYPE BIGINT;
ALTER TABLE tablename ALTER COLUMN fieldname TYPE NUMERIC;
ALTER TABLE tablename ALTER COLUMN fieldname TYPE TEXT;  -- for longtext
```

PostgreSQL handles most type changes safely. INT → BIGINT is always safe. DECIMAL(18,2) → NUMERIC is safe.

### 16.3 MySQL

```sql
ALTER TABLE tablename MODIFY fieldname BIGINT;
ALTER TABLE tablename MODIFY fieldname DECIMAL(65,30);
ALTER TABLE tablename MODIFY fieldname LONGTEXT;
```

MySQL handles most type changes safely. Similar to PostgreSQL.

### 16.4 SQLite

SQLite does **not support ALTER COLUMN**. When storage hint changes are detected on SQLite:

```
WARN: SQLite does not support ALTER COLUMN — storage hint change for field 'xxx' on model 'yyy' will be applied on next table recreation. Current data is preserved.
```

Engine skips the ALTER and logs warning. The change takes effect only if the table is recreated (e.g., during a fresh migration).

### 16.5 MongoDB

MongoDB is schemaless — storage hints have no effect on existing data. New documents use the new type. Existing documents retain their original type.

```
INFO: MongoDB is schemaless — storage hint change for field 'xxx' on model 'yyy' applies to new documents only.
```

### 16.6 New Type Fields on Existing Tables

When a new field type (uuid, ip, vector, etc.) is added to an existing model, the engine adds the column:

```sql
-- PostgreSQL
ALTER TABLE tablename ADD COLUMN fieldname UUID;
ALTER TABLE tablename ADD COLUMN fieldname vector(1536);

-- MySQL
ALTER TABLE tablename ADD COLUMN fieldname CHAR(36);
ALTER TABLE tablename ADD COLUMN fieldname JSON;

-- SQLite
ALTER TABLE tablename ADD COLUMN fieldname TEXT;
```

Adding columns is supported by all databases including SQLite.

### 16.7 Vector Extension Requirement

For PostgreSQL vector type, the pgvector extension must be installed:

```sql
CREATE EXTENSION IF NOT EXISTS vector;
```

Engine attempts this automatically before creating vector columns. If it fails (no permission), engine logs:

```
ERROR: pgvector extension required for vector fields — run 'CREATE EXTENSION vector' as superuser
```

---

## 19. Parser Changes Summary

### 19.1 New FieldType Constants

```go
FieldUUID       FieldType = "uuid"
FieldIP         FieldType = "ip"
FieldIPv6       FieldType = "ipv6"
FieldYear       FieldType = "year"
FieldVector     FieldType = "vector"
FieldBinary     FieldType = "binary"
FieldJSONObject FieldType = "json:object"
FieldJSONArray  FieldType = "json:array"
```

### 19.2 New FieldDefinition Fields

```go
Hidden        bool   `json:"hidden,omitempty"`
Storage       string `json:"storage,omitempty"`
Scale         int    `json:"scale,omitempty"`
DisplayField  string `json:"display_field,omitempty"`
CurrencyField string `json:"currency_field,omitempty"`
Dimensions    int    `json:"dimensions,omitempty"`
Index         any    `json:"index,omitempty"`
```

### 19.3 New/Changed ModelDefinition & Related Structs

```go
// ModelTableConfig — add Plural field
type ModelTableConfig struct {
    Prefix string `json:"prefix"`
    Plural *bool  `json:"plural,omitempty"`  // NEW: per-model plural override
    Name   string `json:"name,omitempty"`    // NEW: explicit table name (alternative to top-level TableName)
}
```

`title_field`, `search_field` already exist in ModelDefinition. Behavior changes only (format engine support, auto-extraction).

### 19.4 New Validation Rules in Parser

```go
// Vector must have dimensions
if field.Type == FieldVector && field.Dimensions == 0 {
    return error("vector field %q must specify dimensions")
}

// currency and currency_field are mutually exclusive
if field.CurrencyCode != "" && field.CurrencyField != "" {
    return error("field %q cannot have both currency and currency_field")
}

// display_field only valid on many2one
if field.DisplayField != "" && field.Type != FieldMany2One {
    warn("display_field on non-many2one field %q will be ignored")
}

// storage hint validation
if field.Storage != "" {
    if !isValidStorageHint(field.Type, field.Storage) {
        return error("invalid storage hint %q for type %q — valid: %v", ...)
    }
}

// JSON variant parsing
if strings.HasPrefix(string(field.Type), "json:") {
    // parse variant
}

// IP variant parsing
if field.Type == "ip:v4" → FieldIP
if field.Type == "ip:v6" → FieldIPv6

// Duplicate model detection (in Registry, not parser)
if existing.Module == model.Module && model.Inherit == "" {
    return error("duplicate model name %q in module %q", ...)
}
```

### 19.5 `fieldTypeToSQL` Rewrite

The current `fieldTypeToSQL` function needs to be rewritten to handle all 40+ types explicitly. No more `default: return "TEXT"`. Every type must have an explicit case.

---

## 20. Implementation Tasks

### 20.1 Parser Layer (`compiler/parser/`)

- [ ] Add new FieldType constants (uuid, ip, ipv6, year, vector, binary, json:object, json:array)
- [ ] Add new FieldDefinition fields (hidden, storage, scale, display_field, currency_field, dimensions, index)
- [ ] Implement type variant parsing (json:object, json:array, ip:v4, ip:v6)
- [ ] Add storage hint validation (whitelist per type)
- [ ] Add vector dimensions validation
- [ ] Add currency/currency_field mutual exclusion validation
- [ ] Add display_field type validation (many2one only)
- [ ] Update precision/scale resolution logic (backward compat)
- [ ] Update search_field auto-extraction (format template support)
- [ ] Add title_field format detection (contains `{`)
- [ ] Add parser warning for title_field fallback to "id"
- [ ] Write tests for all new parsing rules

### 20.2 Migration Layer (`infrastructure/persistence/`)

- [ ] Rewrite `fieldTypeToSQL` — explicit case for ALL types, no default TEXT
- [ ] Add storage hint resolution in `fieldTypeToSQL`
- [ ] Add vector column support (pgvector CREATE EXTENSION + vector(N) type)
- [ ] Add vector index support (HNSW/IVFFlat)
- [ ] Add binary column support (BYTEA/LONGBLOB/BLOB)
- [ ] Add uuid column support (native UUID on PG)
- [ ] Add year column support (SMALLINT)
- [ ] Add ip/ipv6 column support (VARCHAR(45)/VARCHAR(15))
- [ ] Add ALTER COLUMN support for storage hint changes (PG + MySQL)
- [ ] Add SQLite ALTER COLUMN skip + WARNING
- [ ] Add MongoDB WARNING for storage hint changes
- [ ] Update `mongo_migration.go` for new types
- [ ] Write tests for all SQL generation

### 20.3 Validation Layer

- [ ] Add built-in validators: email (existing), uuid, ip, ipv4, ipv6, year, color, vector dimensions
- [ ] Add json:object / json:array validation
- [ ] Ensure validation only applies to new/updated records (not migration)
- [ ] Write tests for all validators

### 20.4 Repository Layer (`infrastructure/persistence/repository.go`)

- [ ] Update `loadMany2OneRelation` to use `display_field` or `title_field` for label resolution
- [ ] Add format engine integration for title_field templates
- [ ] Update search query to use `search_field` (already partially implemented)
- [ ] Write tests for many2one label resolution

### 20.5 View/UI Layer (`presentation/view/`)

- [ ] Update `component_compiler.go` — add widget mapping for new types
- [ ] Update `component_compiler.go` — respect `hidden` field
- [ ] Update `auto_page_generator.go` — exclude hidden fields from auto-generated views
- [ ] Update currency rendering to support `currency_field`
- [ ] Write tests for component rendering

### 20.6 Table Naming & Duplicate Detection

- [ ] Add `TablePlural *bool` to `ModelTableConfig` struct
- [ ] Add `table_naming` config to project config (Viper binding)
- [ ] Update `ResolveTableName()` — add plural logic with `jinzhu/inflection`
- [ ] Update `auto_api.go` — API path follows table naming convention
- [ ] Replace hand-rolled `pluralize()` in `auto_api.go` with `inflection.Plural()`
- [ ] Replace hand-rolled `pluralizeModel()` in `graphql/schema.go` with `inflection.Plural()`
- [ ] Fix GraphQL double-plural bug
- [ ] Update auto page path generation to follow table naming
- [ ] Add duplicate model detection in `Registry.Register()` — error on same-module duplicate
- [ ] Skip duplicate error for inheritance (`model.Inherit != ""`)
- [ ] Write tests: plural table name resolution (with prefix, without prefix, per-model override)
- [ ] Write tests: API path pluralization
- [ ] Write tests: duplicate model detection (same module error, cross-module OK, inheritance OK)
- [ ] Write tests: junction table naming (always singular)

### 20.7 Documentation

- [ ] Update `engine/docs/features/models.md` with new types and storage hints
- [ ] Add examples for each new type
- [ ] Document migration behavior (ALTER COLUMN, SQLite limitation)
- [ ] Document title_field format engine support
