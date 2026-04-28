# Phase 6C: Engine Enhancements — Array-Backed Models, View Modifiers, Metadata API, Eager Loading

**Date**: 14 July 2026
**Status**: Draft
**Depends on**: Phase 6A (field modifiers, display_field), Phase 1 (bridge API for process data_source)
**Unlocks**: Phase 7 (module setting)
**Master doc**: `2026-07-14-runtime-engine-redesign-master.md`

---

## Table of Contents

1. [Goal](#1-goal)
2. [Array-Backed Models (Sushi-Style)](#2-array-backed-models-sushi-style)
3. [View Layout Modifiers](#3-view-layout-modifiers)
4. [Metadata API](#4-metadata-api)
5. [Embedded View Improvements](#5-embedded-view-improvements)
6. [Process-Based Data Source](#6-process-based-data-source)
7. [Eager Loading Gaps (WithClause)](#7-eager-loading-gaps-withclause)
8. [Project Config Additions](#8-project-config-additions)
9. [Implementation Tasks](#9-implementation-tasks)

---

## 1. Goal

Close the remaining engine gaps that block Phase 7 (module "setting"). Module "setting" is an admin panel built entirely as a JSON module — it needs to introspect the engine's own schema, render conditional UI, and use process-based data sources.

### 1.1 Problems Being Solved

1. **No array-backed models** — fixture data (currencies, countries, timezones) must be seeded to database manually. No way to define static/config data inline in model JSON.
2. **No `visible_if` / `disabled_if` in view layout** — cannot conditionally show/hide or disable fields in forms based on record state
2. **No metadata API** — no way for scripts or views to introspect model definitions, field types, view definitions at runtime. Module "setting" needs this to render model editors, field inspectors, etc.
3. **Embedded views have no filtering** — embedded list views in form tabs always show ALL records, no way to filter by parent record
4. **Data source only supports model queries** — `data_sources` in views can only query a model, cannot execute a process to get data
5. **`WithClause.Conditions` and `WithClause.Nested` parsed but not used** — eager loading ignores conditions and nested relations

### 1.2 Success Criteria

- `source: "array"` models work — data loaded from JSON into main DB, queryable like normal models
- Array models support all PK strategies, relations, permissions, views, API
- `visible_if` and `disabled_if` work in LayoutRow and ChildTableColumn
- Metadata API exposes model list, model detail, field definitions, view definitions
- Embedded views support `filter_by` to scope records by parent
- Data sources support `process` execution (not just model query)
- `WithClause.Conditions` applied during relation loading
- `WithClause.Nested` triggers recursive relation loading (at least 1 level deep)

### 1.3 What This Phase Does NOT Do

- Does not build module "setting" (Phase 7)
- Does not add new field types (Phase 6A)
- Does not add polymorphic relations (Phase 6B)
- Does not change runtime/bridge API (Phase 1-5)

---

## 2. Array-Backed Models (Sushi-Style)

### 2.1 Concept

Inspired by Laravel Sushi — models whose data comes from JSON/CSV/XLSX/XML instead of user input. Data is loaded into the **main database** on startup, so all existing features (query, relations, permissions, views, API) work without any changes.

### 2.2 Model JSON Schema

```json
{
  "name": "currency",
  "source": "array",
  "primary_key": { "strategy": "natural_key", "field": "code" },
  "fields": {
    "code": { "type": "string", "max": 3 },
    "name": { "type": "string" },
    "symbol": { "type": "string", "max": 5 },
    "decimals": { "type": "integer", "default": 2 }
  },
  "rows": [
    { "code": "IDR", "name": "Indonesian Rupiah", "symbol": "Rp", "decimals": 0 },
    { "code": "USD", "name": "US Dollar", "symbol": "$", "decimals": 2 },
    { "code": "EUR", "name": "Euro", "symbol": "€", "decimals": 2 }
  ]
}
```

With external file:

```json
{
  "name": "country",
  "source": "array",
  "primary_key": { "strategy": "natural_key", "field": "iso_code" },
  "rows_file": "data/countries.csv",
  "fields": {
    "iso_code": { "type": "string", "max": 2 },
    "name": { "type": "string" },
    "dial_code": { "type": "string", "max": 5 }
  }
}
```

### 2.3 New ModelDefinition Fields

```go
type ModelDefinition struct {
    // ... existing ...
    Source    string            `json:"source,omitempty"`     // "db" (default) | "array"
    Writable *bool             `json:"writable,omitempty"`   // array source: allow CRUD? default false
    Rows     []map[string]any  `json:"rows,omitempty"`       // inline array data
    RowsFile string            `json:"rows_file,omitempty"`  // external file path
}
```

### 2.4 `source` Values

| Value | Description | Default |
|-------|-------------|:-------:|
| `"db"` | Normal database model — data from user/API/seed | ✅ |
| `"array"` | Array-backed — data from JSON rows or external file | |

### 2.5 Primary Key

Array models follow the **exact same PK rules** as DB models:

- No PK config → default auto_increment (same as DB model)
- All PK strategies supported: auto_increment, uuid, natural_key, naming_series, composite, manual

**Recommendation for array models**: Use `natural_key` or explicit `id` in rows for stable FK references. Auto_increment IDs are assigned by insertion order — if rows are reordered or inserted in the middle, IDs shift.

> **WARNING** (documented): "Array models with auto_increment PK: row IDs depend on insertion order. For models referenced by other models (many2one/FK), use `primary_key: { strategy: "natural_key" }` or include explicit `id` in rows."

### 2.6 `writable` Mode

| Mode | `writable` | API | Sync on restart | Use case |
|------|:----------:|-----|-----------------|----------|
| Read-only | `false` (default) | GET only, POST/PUT/DELETE → 405 | DELETE all + re-INSERT from source | Fixture: currencies, countries, timezones |
| Editable | `true` | Full CRUD | Seed only if table empty | Settings, feature flags, editable config |

### 2.7 `rows_file` — Supported Formats

Format auto-detected from file extension:

| Extension | Format | Parser |
|-----------|--------|--------|
| `.json` | JSON array of objects | `encoding/json` |
| `.csv` | CSV with header row | `encoding/csv` |
| `.xlsx` | Excel — first sheet, first row = header | `excelize` library |
| `.xml` | XML collection — `<rows><row>...</row></rows>` | `encoding/xml` |

This reuses the same parsing infrastructure as the seed/migration engine.

```json
{ "rows_file": "data/countries.json" }   // → JSON
{ "rows_file": "data/countries.csv" }    // → CSV
{ "rows_file": "data/countries.xlsx" }   // → Excel
{ "rows_file": "data/countries.xml" }    // → XML
```

`rows_file` path is relative to the module directory.

### 2.8 Sync Strategy

#### Read-only (`writable: false`)

```
Startup / hot reload:
1. CREATE TABLE IF NOT EXISTS (normal migration)
2. DELETE all existing rows
3. INSERT all rows from source (JSON inline or file)
4. Result: table always matches source exactly
```

#### Editable (`writable: true`)

```
Startup:
1. CREATE TABLE IF NOT EXISTS (normal migration)
2. COUNT existing rows
3. If count == 0 → INSERT all rows from source (initial seed)
4. If count > 0 → SKIP (preserve user edits)
```

### 2.9 Auto-Disabled Features for Read-Only Array

| Feature | Read-only array | Writable array | DB model |
|---------|:--------------:|:--------------:|:--------:|
| `timestamps` | Auto-disabled | Normal | Normal |
| `soft_deletes` | Auto-disabled | Normal | Normal |
| Audit log during sync | Skipped | Normal | Normal |
| API write endpoints | 405 Method Not Allowed | Normal | Normal |
| Bridge write methods | Error: "model is read-only" | Normal | Normal |
| Permissions (read) | Normal | Normal | Normal |
| Relations | Normal | Normal | Normal |
| Views | Normal | Normal | Normal |

### 2.10 Validation

| Check | Severity | Message |
|-------|----------|---------|
| `rows` and `rows_file` both set | ERROR | "model 'xxx' cannot have both rows and rows_file" |
| `source: "array"` without `rows` or `rows_file` | ERROR | "array model 'xxx' must have rows or rows_file" |
| `rows_file` extension not supported | ERROR | "unsupported file format '.xyz' for rows_file" |
| `rows_file` file not found | ERROR | "rows_file 'data/xxx.csv' not found in module path" |
| Extra fields in rows not in `fields` | WARNING | "array model 'xxx': row has field 'yyy' not defined in fields" |
| `writable: true` without `source: "array"` | WARNING | "writable is only meaningful for array source models" |

### 2.11 Examples

#### Fixture data (read-only)

```json
{
  "name": "timezone",
  "source": "array",
  "primary_key": { "strategy": "natural_key", "field": "name" },
  "fields": {
    "name": { "type": "string" },
    "offset": { "type": "string", "max": 6 },
    "label": { "type": "string" }
  },
  "rows_file": "data/timezones.json"
}
```

#### Editable settings

```json
{
  "name": "setting",
  "source": "array",
  "writable": true,
  "primary_key": { "strategy": "natural_key", "field": "key" },
  "fields": {
    "key": { "type": "string", "unique": true },
    "value": { "type": "text" },
    "type": { "type": "selection", "options": ["string", "integer", "boolean", "json"] },
    "group": { "type": "string" }
  },
  "rows": [
    { "key": "app.name", "value": "My App", "type": "string", "group": "general" },
    { "key": "app.debug", "value": "false", "type": "boolean", "group": "general" },
    { "key": "mail.from", "value": "noreply@example.com", "type": "string", "group": "mail" }
  ]
}
```

#### Large dataset from CSV

```json
{
  "name": "country",
  "source": "array",
  "primary_key": { "strategy": "natural_key", "field": "iso_code" },
  "rows_file": "data/countries.csv",
  "fields": {
    "iso_code": { "type": "string", "max": 2 },
    "iso3": { "type": "string", "max": 3 },
    "name": { "type": "string" },
    "dial_code": { "type": "string", "max": 5 },
    "currency_code": { "type": "string", "max": 3 }
  }
}
```

#### Metadata as array model

```json
{
  "name": "_field_type",
  "source": "array",
  "module": "base",
  "primary_key": { "strategy": "natural_key", "field": "name" },
  "title_field": "name",
  "fields": {
    "name": { "type": "string" },
    "category": { "type": "string" },
    "storage_default": { "type": "string" },
    "widget": { "type": "string" },
    "hidden_default": { "type": "boolean", "default": false }
  },
  "rows": [
    { "name": "string", "category": "core", "storage_default": "VARCHAR(255)", "widget": "bc-field-string", "hidden_default": false },
    { "name": "currency", "category": "number", "storage_default": "NUMERIC(18,2)", "widget": "bc-field-currency", "hidden_default": false },
    { "name": "vector", "category": "special", "storage_default": "vector(N)", "widget": null, "hidden_default": true }
  ]
}
```

### 2.12 Process Source — Dynamic Data

For data that comes from computation, API calls, or scripts:

```json
{
  "name": "exchange_rate",
  "source": "process",
  "process": "fetch_exchange_rates",
  "refresh": "1h",
  "fields": {
    "from": { "type": "string", "max": 3 },
    "to": { "type": "string", "max": 3 },
    "rate": { "type": "decimal" },
    "updated_at": { "type": "datetime" }
  }
}
```

Or with inline script:

```json
{
  "name": "system_info",
  "source": "process",
  "script": { "lang": "javascript", "file": "scripts/system_info.js" },
  "refresh": "5m",
  "fields": { ... }
}
```

Process source follows the same pattern as model events — `process` (named process) or `script` (inline script reference). The process/script must return an array of objects matching the field definitions.

### 2.13 Refresh Strategy

Applies to all source types. Manual refresh is **always available** regardless of config — via API (`POST /api/v1/_meta/models/:name/refresh`) or bridge (`bitcode.model.refresh("name")`).

```json
{ "refresh": "startup" }   // only on startup (default for "array")
{ "refresh": "1h" }        // every hour + manual
{ "refresh": "30m" }       // every 30 minutes + manual
{ "refresh": "5m" }        // every 5 minutes + manual
{ "refresh": "never" }     // only manual (default for "process")
```

| `refresh` | `source: "array"` | `source: "process"` |
|-----------|:-:|:-:|
| `"startup"` | ✅ Default — sync from file on startup | Re-execute process on startup |
| `"1h"` / `"30m"` / etc. | Re-read file on interval | Re-execute process on interval |
| `"never"` | Only manual trigger | ✅ Default — only manual trigger |

### 2.14 Write-Back Sync (`sync_source`)

For array models with `writable: true`, changes made via UI/API can be written back to the source file:

```json
{
  "name": "currency",
  "source": "array",
  "writable": true,
  "sync_source": true,
  "rows_file": "data/currencies.json",
  "fields": { ... }
}
```

**Behavior**: When a record is created/updated/deleted via API or bridge, the engine writes the current DB state back to the source file in the same format (JSON/CSV/XLSX/XML).

**Source of truth**: DB is always source of truth when `writable: true`. If the file is changed externally (e.g., git pull), engine logs a WARNING and does NOT auto-reload (to prevent overwriting user edits). Use CLI to force:

```bash
bitcode model sync --to-file currency    # DB → File (export)
bitcode model sync --from-file currency  # File → DB (reimport, overwrites DB)
bitcode model diff currency              # Show differences
```

**Validation constraints**:

| Constraint | Error |
|-----------|-------|
| `sync_source: true` + `writable: false` | "sync_source requires writable: true" |
| `sync_source: true` + inline `rows` (no `rows_file`) | "sync_source requires rows_file (cannot write back to inline rows)" |
| `sync_source: true` + `source: "process"` | "sync_source only works with array source" |

### 2.15 Complete Source Behavior Matrix

| `source` | `writable` | `sync_source` | File→DB | DB→File | Refresh | Source of Truth |
|----------|:----------:|:-------------:|:-------:|:-------:|---------|:---------------:|
| `"db"` | — | — | — | — | — | DB |
| `"array"` | `false` | `false` | ✅ Always | ❌ | startup | File |
| `"array"` | `true` | `false` | ✅ Seed | ❌ | startup | DB |
| `"array"` | `true` | `true` | ✅ Seed | ✅ On write | startup | DB |
| `"process"` | `false` | — | ✅ Execute | ❌ | never | Process |
| `"process"` | `true` | — | ✅ Execute | ❌ | never | DB (after seed) |

---

## 3. View Layout Modifiers

### 3.1 Current State

LayoutRow currently has:

```go
type LayoutRow struct {
    Field    string `json:"field,omitempty"`
    Width    int    `json:"width,omitempty"`
    Readonly bool   `json:"readonly,omitempty"`
    Widget   string `json:"widget,omitempty"`
    Formula  string `json:"formula,omitempty"`
}
```

Only `readonly` (static boolean). No conditional visibility or disable.

### 3.2 New Fields

```go
type LayoutRow struct {
    // ... existing fields ...
    VisibleIf  string `json:"visible_if,omitempty"`   // Expression: show field when true
    DisabledIf string `json:"disabled_if,omitempty"`  // Expression: disable field when true
    ReadonlyIf string `json:"readonly_if,omitempty"`  // Expression: readonly when true (view-level override)
    CSSClass   string `json:"css_class,omitempty"`    // Custom CSS class
    HelpText   string `json:"help_text,omitempty"`    // Tooltip / help text
}
```

Same additions for ChildTableColumn:

```go
type ChildTableColumn struct {
    // ... existing fields ...
    VisibleIf  string `json:"visible_if,omitempty"`
    DisabledIf string `json:"disabled_if,omitempty"`
    ReadonlyIf string `json:"readonly_if,omitempty"`
}
```

### 3.3 Expression Syntax

Expressions use the same syntax as existing `readonly_if`, `mandatory_if`, `depends_on` in FieldDefinition:

```json
{
  "row": [
    { "field": "name", "width": 6 },
    { "field": "company", "width": 6, "visible_if": "type == 'business'" },
    { "field": "tax_id", "width": 6, "visible_if": "type == 'business'", "readonly_if": "status != 'draft'" },
    { "field": "notes", "width": 12, "disabled_if": "status == 'closed'" }
  ]
}
```

### 3.4 Evaluation

Expressions are evaluated **client-side** (in the web component) for real-time reactivity. The server renders all fields but marks them with `data-visible-if`, `data-disabled-if` attributes:

```html
<div class="bc-field" data-visible-if="type == 'business'">
  <label>Company</label>
  <input name="company" ...>
</div>
```

The `bc-form` web component evaluates expressions on form data change and toggles visibility/disabled state.

### 3.5 Server-Side Rendering Fallback

For SSR (non-JS environments), the server evaluates expressions against the current record data and omits hidden fields / adds disabled attribute:

```go
func shouldRender(expr string, record map[string]any) bool {
    if expr == "" {
        return true
    }
    return expression.Evaluate(expr, record)
}
```

### 3.6 Precedence: Model vs View

| Modifier | Model Level | View Level | Precedence |
|----------|:-----------:|:----------:|------------|
| `readonly_if` | FieldDefinition | LayoutRow | View overrides model |
| `depends_on` | FieldDefinition | — | Model only (visibility) |
| `visible_if` | — | LayoutRow | View only |
| `disabled_if` | — | LayoutRow | View only |
| `hidden` | FieldDefinition | — | Model only (excluded from auto-views) |

**Rule**: View-level modifiers override model-level for the same concern. If model says `readonly_if: "status != 'draft'"` but view says `readonly_if: "false"` (always editable), view wins.

---

## 4. Metadata API

### 4.1 Why Needed

Module "setting" (Phase 7) needs to:
- List all registered models
- Show field definitions for a model
- Show view definitions
- Show module definitions
- Allow editing model/view JSON (with validation)

Currently, this information is only accessible via `admin.go` (hardcoded HTML). Module "setting" needs it as **JSON API**.

### 4.2 Endpoints

```
GET /api/v1/_meta/models                    → list all models
GET /api/v1/_meta/models/:name              → model definition (fields, indexes, etc.)
GET /api/v1/_meta/models/:name/fields       → field definitions only
GET /api/v1/_meta/views                     → list all views
GET /api/v1/_meta/views/:name               → view definition
GET /api/v1/_meta/modules                   → list all modules
GET /api/v1/_meta/modules/:name             → module definition
GET /api/v1/_meta/processes                 → list all processes
GET /api/v1/_meta/processes/:name           → process definition
GET /api/v1/_meta/field-types               → list all supported field types with metadata
```

### 4.3 Response Format

#### `GET /api/v1/_meta/models`

```json
{
  "models": [
    {
      "name": "contact",
      "module": "crm",
      "label": "Contact",
      "title_field": "name",
      "field_count": 12,
      "table_name": "crm_contacts",
      "has_api": true,
      "has_views": true
    }
  ]
}
```

#### `GET /api/v1/_meta/models/contact`

```json
{
  "name": "contact",
  "module": "crm",
  "label": "Contact",
  "title_field": "name",
  "search_field": ["name", "email"],
  "table_name": "crm_contacts",
  "fields": {
    "name": { "type": "string", "required": true, "max": 255 },
    "email": { "type": "email", "unique": true },
    "tags": { "type": "many2many", "model": "tag" }
  },
  "indexes": [["email"]],
  "record_rules": [],
  "api": { "auto_crud": true, "auth": true },
  "timestamps": true,
  "soft_deletes": false
}
```

#### `GET /api/v1/_meta/field-types`

```json
{
  "types": [
    {
      "name": "string",
      "category": "core",
      "storage": "VARCHAR(255)",
      "widget": "bc-field-string",
      "supports_storage_hint": true,
      "valid_storage_hints": ["char"]
    },
    {
      "name": "currency",
      "category": "number",
      "storage": "NUMERIC(18,2)",
      "widget": "bc-field-currency",
      "supports_storage_hint": true,
      "valid_storage_hints": ["numeric"]
    },
    {
      "name": "vector",
      "category": "special",
      "storage": "vector(N)",
      "widget": null,
      "default_hidden": true,
      "requires": ["dimensions"]
    }
  ]
}
```

### 4.4 Authentication

Metadata API requires **admin authentication** by default. Configurable:

```toml
# bitcode.toml
[meta_api]
enabled = true          # default: true
auth = true             # default: true (require auth)
admin_only = true       # default: true (only admin group)
```

### 4.5 Bridge API Access

Scripts can also access metadata via bridge:

```javascript
const models = await bitcode.meta.models();
const contact = await bitcode.meta.model("contact");
const fields = await bitcode.meta.fields("contact");
const fieldTypes = await bitcode.meta.fieldTypes();
```

This is added to the `bitcode.meta` namespace in the bridge API (Phase 1 already reserved this namespace).

---

## 5. Embedded View Improvements

### 5.1 Problem

Embedded views in form tabs currently load ALL records:

```go
// renderer.go:87 — no filtering!
records, total, err := repo.FindAll(context.Background(), nil, 1, 10)
```

When a form shows an embedded list of "orders for this customer", it shows ALL orders, not just this customer's orders.

### 5.2 Solution: `filter_by` on TabDefinition

```go
type TabDefinition struct {
    Label    string   `json:"label"`
    View     string   `json:"view,omitempty"`
    Fields   []string `json:"fields,omitempty"`
    Visible  string   `json:"visible,omitempty"`
    FilterBy string   `json:"filter_by,omitempty"`  // NEW: filter embedded view by parent record
}
```

### 5.3 Usage

```json
{
  "tabs": [
    {
      "label": "Orders",
      "view": "order_list",
      "filter_by": "customer_id"
    }
  ]
}
```

**Behavior**: When rendering the embedded `order_list` view inside a `customer` form, the engine adds a filter:

```sql
WHERE customer_id = {current_record_id}
```

### 5.4 Implementation

```go
func (r *Renderer) renderEmbeddedView(viewDef *parser.ViewDefinition, parentRecord map[string]any, filterBy string) string {
    repo := persistence.NewGenericRepository(r.db, r.resolveTable(viewDef.Model))

    var query *persistence.Query
    if filterBy != "" && parentRecord != nil {
        parentID := parentRecord["id"]
        query = &persistence.Query{
            Conditions: []persistence.Condition{
                {Field: filterBy, Operator: "=", Value: parentID},
            },
        }
    }

    records, total, err := repo.FindAll(context.Background(), query, 1, 10)
    if err != nil {
        return errorHTML(err)
    }
    return r.renderEmbeddedList(viewDef, records, total)
}
```

### 5.5 Multiple Filters

For complex cases, `filter_by` can be an object:

```json
{
  "tabs": [
    {
      "label": "Active Orders",
      "view": "order_list",
      "filter_by": {
        "customer_id": "{record.id}",
        "status": "active"
      }
    }
  ]
}
```

This requires `filter_by` to accept both `string` (simple FK filter) and `map[string]any` (complex filter). Parser handles both:

```go
type TabDefinition struct {
    // ... existing ...
    FilterByRaw json.RawMessage `json:"filter_by,omitempty"`
    FilterBy    string          `json:"-"` // simple mode
    FilterByMap map[string]any  `json:"-"` // complex mode
}
```

---

## 6. Process-Based Data Source

### 6.1 Problem

View `data_sources` currently only support model queries:

```json
{
  "data_sources": {
    "stats": {
      "model": "order",
      "domain": [["status", "=", "completed"]]
    }
  }
}
```

But module "setting" needs data from **processes** — e.g., "get system stats", "get recent activity", "get module health". These are not simple model queries.

### 6.2 Solution: Process Data Source

```json
{
  "data_sources": {
    "stats": {
      "process": "get_dashboard_stats"
    },
    "recent_orders": {
      "model": "order",
      "domain": [["status", "=", "completed"]]
    }
  }
}
```

### 6.3 DataSourceDefinition Update

```go
type DataSourceDefinition struct {
    Model   string  `json:"model,omitempty"`
    Domain  [][]any `json:"domain,omitempty"`
    Process string  `json:"process,omitempty"`  // NEW: execute process to get data
}
```

**Validation**: `model` and `process` are mutually exclusive. Parser error if both set.

### 6.4 Implementation

```go
func (r *Renderer) resolveDataSource(ctx context.Context, name string, ds parser.DataSourceDefinition) (any, error) {
    if ds.Model != "" {
        // Existing: query model
        repo := persistence.NewGenericRepository(r.db, r.resolveTable(ds.Model))
        records, _, err := repo.FindAll(ctx, persistence.QueryFromDomain(ds.Domain), 1, 1000)
        return records, err
    }
    if ds.Process != "" {
        // NEW: execute process
        result, err := r.processExecutor.Execute(ctx, ds.Process, nil)
        return result, err
    }
    return nil, fmt.Errorf("data source %q has neither model nor process", name)
}
```

### 6.5 Security

Process data sources execute with the **current user's permissions**. The process itself handles authorization via its own permission checks.

---

## 7. Eager Loading Gaps (WithClause)

### 7.1 Problem

`WithClause` has `Conditions` and `Nested` fields that are parsed from JSON but **ignored** by relation loaders:

```go
type WithClause struct {
    Relation   string       // ✅ used
    Conditions []Condition  // ❌ parsed but IGNORED
    Select     []string     // ⚠️ partially used (many2one only)
    OrderBy    []OrderClause // ⚠️ partially used (one2many only)
    Limit      int          // ⚠️ partially used (one2many only)
    Nested     []WithClause // ❌ parsed but IGNORED
}
```

### 7.2 Fix: Apply Conditions

```javascript
// Bridge API usage:
const post = await bitcode.db.findOne("post", postId, {
  with: [{
    relation: "comments",
    conditions: [["status", "=", "approved"]],
    order: [{ field: "created_at", direction: "desc" }],
    limit: 10
  }]
});
```

**Implementation**: Apply conditions as additional WHERE clauses in relation loading:

```go
func (r *GenericRepository) loadOne2ManyRelation(ctx context.Context, w WithClause, fieldDef parser.FieldDefinition, results []map[string]any) {
    // ... existing code ...

    relQ := r.db.WithContext(ctx).Table(relatedTable).
        Where(fmt.Sprintf("%s IN ?", inverseField), parentIDs)

    // NEW: apply WithClause conditions
    for _, cond := range w.Conditions {
        relQ = applyCondition(relQ, cond)
    }

    // Existing: apply order and limit
    // ...
}
```

Apply conditions to ALL relation types (many2one, one2many, many2many, morph_*).

### 7.3 Fix: Apply Select, OrderBy, Limit Consistently

Currently `Select` only works for many2one, `OrderBy`/`Limit` only for one2many. Fix: apply all three to ALL relation types.

### 7.4 Fix: Nested Eager Loading

```javascript
// Load post → comments → author (nested)
const post = await bitcode.db.findOne("post", postId, {
  with: [{
    relation: "comments",
    nested: [{
      relation: "author"
    }]
  }]
});
```

**Implementation**: After loading the primary relation, recursively load nested relations on the loaded records:

```go
func (r *GenericRepository) loadWithRelations(ctx context.Context, query *Query, results []map[string]any) {
    for _, w := range query.With {
        // Load primary relation
        r.loadRelation(ctx, w, results)

        // NEW: load nested relations
        if len(w.Nested) > 0 {
            // Get the loaded related records
            for _, rec := range results {
                if related, ok := rec["_"+w.Relation]; ok {
                    // Create sub-repository for the related model
                    // Load nested relations on related records
                    r.loadNestedRelations(ctx, w, related)
                }
            }
        }
    }
}
```

**Depth limit**: Maximum 3 levels of nesting to prevent infinite recursion and performance issues. Configurable per-query.

### 7.5 Backward Compatibility

All changes are additive. Existing queries without `conditions` or `nested` work exactly as before.

---

## 8. Project Config Additions

### 8.1 New Config Keys

```toml
# bitcode.toml

[database]
table_naming = "plural"         # "singular" (default) | "plural" — from Phase 6A

[locale]
currency = "IDR"                # Default currency code — from Phase 6A
timezone = "Asia/Jakarta"       # Default timezone
number_format = "id-ID"         # BCP 47 locale — auto-resolve number separators
thousand_separator = "."        # Explicit override (takes precedence over number_format)
decimal_separator = ","         # Explicit override (takes precedence over number_format)

[meta_api]
enabled = true                  # Enable metadata API
auth = true                     # Require authentication
admin_only = true               # Restrict to admin group

[eager_loading]
max_depth = 3                   # Maximum nested eager loading depth
```

### 8.2 Viper Bindings

```go
v.SetDefault("database.table_naming", "singular")
v.SetDefault("locale.currency", "USD")
v.SetDefault("locale.timezone", "UTC")
v.SetDefault("locale.number_format", "en-US")
v.SetDefault("locale.thousand_separator", "")  // empty = auto from number_format
v.SetDefault("locale.decimal_separator", "")   // empty = auto from number_format
v.SetDefault("meta_api.enabled", true)
v.SetDefault("meta_api.auth", true)
v.SetDefault("meta_api.admin_only", true)
v.SetDefault("eager_loading.max_depth", 3)

v.BindEnv("database.table_naming", "DB_TABLE_NAMING")
v.BindEnv("locale.currency", "DEFAULT_CURRENCY")
v.BindEnv("locale.timezone", "DEFAULT_TIMEZONE")
v.BindEnv("locale.number_format", "NUMBER_FORMAT")
v.BindEnv("locale.thousand_separator", "THOUSAND_SEPARATOR")
v.BindEnv("locale.decimal_separator", "DECIMAL_SEPARATOR")
v.BindEnv("meta_api.enabled", "META_API_ENABLED")
v.BindEnv("meta_api.auth", "META_API_AUTH")
v.BindEnv("meta_api.admin_only", "META_API_ADMIN_ONLY")
v.BindEnv("eager_loading.max_depth", "EAGER_LOADING_MAX_DEPTH")
```

---

## 9. Implementation Tasks

### 9.1 Array-Backed Models

- [ ] Add `Source`, `Writable`, `Rows`, `RowsFile` fields to ModelDefinition
- [ ] Implement array source sync: DELETE all + re-INSERT for read-only
- [ ] Implement array source seed: INSERT only if table empty for writable
- [ ] Implement `rows_file` parser: JSON format
- [ ] Implement `rows_file` parser: CSV format (header row)
- [ ] Implement `rows_file` parser: XLSX format (first sheet, header row)
- [ ] Implement `rows_file` parser: XML format (`<rows><row>...</row></rows>`)
- [ ] Auto-detect format from file extension
- [ ] Implement process source: execute process/script, populate table from result
- [ ] Implement refresh scheduler (interval-based re-sync)
- [ ] Implement manual refresh endpoint: `POST /api/v1/_meta/models/:name/refresh`
- [ ] Implement manual refresh bridge: `bitcode.model.refresh("name")`
- [ ] Implement `sync_source`: write-back to file on DB change (JSON/CSV/XLSX/XML)
- [ ] Implement CLI: `bitcode model sync --to-file`, `--from-file`, `bitcode model diff`
- [ ] Auto-disable timestamps/soft_deletes/audit for read-only array
- [ ] Block write API endpoints (405) for read-only array
- [ ] Validation: rows + rows_file mutual exclusion
- [ ] Validation: source "array" without rows/rows_file = error
- [ ] Validation: sync_source constraints (writable + rows_file required)
- [ ] Warning: extra fields in rows not in fields definition
- [ ] Write tests: array source lifecycle (create, sync, restart)
- [ ] Write tests: writable mode (seed, CRUD, no re-sync)
- [ ] Write tests: process source (execute, refresh)
- [ ] Write tests: all file formats (JSON, CSV, XLSX, XML)
- [ ] Write tests: sync_source write-back
- [ ] Write tests: relations between array and DB models

### 9.2 View Modifiers (`presentation/view/`)

- [ ] Add `visible_if`, `disabled_if`, `readonly_if`, `css_class`, `help_text` to LayoutRow
- [ ] Add `visible_if`, `disabled_if`, `readonly_if` to ChildTableColumn
- [ ] Update `component_compiler.go` — render `data-visible-if`, `data-disabled-if` attributes
- [ ] Update SSR rendering — evaluate expressions server-side for non-JS fallback
- [ ] Document precedence rules (view overrides model)
- [ ] Write tests for conditional rendering

### 9.3 Metadata API (`presentation/api/`)

- [ ] Create `meta_handler.go` — new handler for metadata endpoints
- [ ] Implement `GET /api/v1/_meta/models` — list all models with summary
- [ ] Implement `GET /api/v1/_meta/models/:name` — full model definition
- [ ] Implement `GET /api/v1/_meta/models/:name/fields` — field definitions only
- [ ] Implement `GET /api/v1/_meta/views` — list all views
- [ ] Implement `GET /api/v1/_meta/views/:name` — view definition
- [ ] Implement `GET /api/v1/_meta/modules` — list all modules
- [ ] Implement `GET /api/v1/_meta/modules/:name` — module definition
- [ ] Implement `GET /api/v1/_meta/processes` — list all processes
- [ ] Implement `GET /api/v1/_meta/processes/:name` — process definition
- [ ] Implement `GET /api/v1/_meta/field-types` — supported field types with metadata
- [ ] Add auth middleware for metadata endpoints
- [ ] Add `meta_api` config to Viper
- [ ] Add `bitcode.meta.*` bridge methods
- [ ] Write tests for all metadata endpoints

### 9.4 Embedded View Improvements (`presentation/view/`)

- [ ] Add `filter_by` to TabDefinition (string and map modes)
- [ ] Update `embeddedViewRenderer` — apply filter_by when rendering embedded views
- [ ] Parse `filter_by` in tab definition (handle both string and object)
- [ ] Write tests for filtered embedded views

### 9.5 Process Data Source (`presentation/view/`)

- [ ] Add `process` field to DataSourceDefinition
- [ ] Add validation: `model` and `process` mutually exclusive
- [ ] Update `renderChart` and `renderCustom` — resolve process data sources
- [ ] Wire process executor into Renderer
- [ ] Write tests for process-based data sources

### 9.6 Eager Loading Fixes (`infrastructure/persistence/`)

- [ ] Apply `WithClause.Conditions` in `loadMany2OneRelation`
- [ ] Apply `WithClause.Conditions` in `loadOne2ManyRelation`
- [ ] Apply `WithClause.Conditions` in `loadMany2ManyRelation`
- [ ] Apply `WithClause.Conditions` in all morph relation loaders (Phase 6B)
- [ ] Apply `WithClause.Select` consistently across all relation types
- [ ] Apply `WithClause.OrderBy` consistently across all relation types
- [ ] Apply `WithClause.Limit` consistently across all relation types
- [ ] Implement `WithClause.Nested` — recursive relation loading (max depth configurable)
- [ ] Add `eager_loading.max_depth` config
- [ ] Write tests for conditional eager loading
- [ ] Write tests for nested eager loading (1, 2, 3 levels)
- [ ] Write tests for depth limit enforcement

### 9.7 Project Config

- [ ] Add `database.table_naming` config + Viper binding
- [ ] Add `locale.currency` config + Viper binding
- [ ] Add `locale.timezone` config + Viper binding
- [ ] Add `meta_api.*` config + Viper bindings
- [ ] Add `eager_loading.max_depth` config + Viper binding
- [ ] Write tests for config defaults and env var overrides

### 9.8 Documentation

- [ ] Update `engine/docs/features/views.md` with view modifiers
- [ ] Document metadata API endpoints and response format
- [ ] Document embedded view filter_by
- [ ] Document process data source
- [ ] Document eager loading conditions and nesting
- [ ] Update configuration docs with new config keys
