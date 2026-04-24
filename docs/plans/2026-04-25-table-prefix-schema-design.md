# Table Prefix & Postgres Schema Support — Design

**Date**: 2026-04-25
**Status**: Approved
**Scope**: Task 1 (Table Prefix) + Task 2 (Postgres Schema)

## Overview

Add per-module table name prefix support and Postgres schema support. This centralizes table name derivation (currently hardcoded as `model + "s"` in 16+ places) into a single resolver, removes naive pluralization, and adds configurable prefixes.

## Design Decisions

1. **No pluralization** — table name = model name (not `model + "s"`). `contact` → `contact`, not `contacts`.
2. **API paths also no pluralization** — `/api/contact`, not `/api/contacts`. Consistent with model name.
3. **Per-module prefix** via `module.json` `"table"` config.
4. **Per-model override** via model JSON `"table"` field.
5. **System tables** (audit_log, sequence, view_revision, data_revision, attachment) stay unprefixed — they are engine internals, not module-defined models.
6. **Postgres schema** via `DB_SCHEMA` config, applied via `search_path` at connection time.
7. **Centralized resolver** in `domain/model/registry.go`.

## JSON Config

### module.json — `"table"` field

```json
// With prefix
{
  "name": "crm",
  "table": { "prefix": "crm" },
  ...
}

// No prefix (default)
{
  "name": "auth",
  ...
}
```

### model JSON — `"table"` override

```json
// Direct table name override
{
  "name": "contact",
  "table": "custom_contact",
  ...
}

// Override prefix (different from module)
{
  "name": "log",
  "table": { "prefix": "sys" },
  ...
}

// Clear module prefix for this model
{
  "name": "setting",
  "table": { "prefix": "" },
  ...
}
```

## Table Name Resolution Order

```
ResolveTableName(modelName) → string:

1. Model has "table" as string → return that string directly
   e.g. "table": "custom_contact" → "custom_contact"

2. Model has "table" as object with "prefix" → model prefix + "_" + model name
   e.g. "table": {"prefix": "sys"}, name: "log" → "sys_log"
   e.g. "table": {"prefix": ""}, name: "setting" → "setting"

3. Module has "table.prefix" → module prefix + "_" + model name
   e.g. module prefix "crm", name: "contact" → "crm_contact"

4. Default → model name as-is
   e.g. name: "contact" → "contact"
```

## Example Results

| Module | Module Prefix | Model | Model Override | → Table Name |
|--------|--------------|-------|----------------|--------------|
| base | `res` | user | - | `res_user` |
| base | `res` | setting | `{"prefix": ""}` | `setting` |
| crm | `crm` | contact | - | `crm_contact` |
| crm | `crm` | lead | - | `crm_lead` |
| sales | `sale` | order | - | `sale_order` |
| hrm | `hr` | employee | - | `hr_employee` |
| hrm | `hr` | leave_request | - | `hr_leave_request` |

Junction tables follow prefix: `crm_contact_tag` (prefix + model1 + "_" + fieldName)

## Postgres Schema Support

### Config

```toml
[database]
driver = "postgres"
schema = "myapp"   # default: "public"
```

Environment: `DB_SCHEMA=myapp`

### Behavior

- On Postgres connection open: `CREATE SCHEMA IF NOT EXISTS {schema}` then `SET search_path TO {schema}`
- MySQL/SQLite: ignore schema config completely
- No per-query schema prefixing needed — `search_path` handles it globally

## Files Changed

### New/Modified Structs

| File | Change |
|------|--------|
| `parser/module.go` | Add `TableConfig` struct, `Table` field on `ModuleDefinition` |
| `parser/model.go` | Add `Table` field on `ModelDefinition` (string or object) |
| `domain/model/registry.go` | Refactor `TableName()` → `ResolveTableName()` with prefix lookup |
| `persistence/database.go` | Add `Schema` to `DatabaseConfig`, set `search_path` for Postgres |
| `config.go` | Add `database.schema` binding |

### Patched Call Sites (remove `+ "s"`)

| File | Lines | What |
|------|-------|------|
| `persistence/dynamic_model.go` | 33, 278 | MigrateModel, createJunctionTable |
| `presentation/api/router.go` | 56, 58 | RegisterAPI |
| `presentation/view/renderer.go` | 74, 167, 218, 264, 314, 343, 369 | All view renderers |
| `runtime/executor/steps/data.go` | 22 | Process data handler |
| `runtime/expression/hydrator.go` | 97 | Child loading |
| `app.go` | 975, 977 | Form POST handling |
| `presentation/admin/admin_api.go` | 272 | Model data listing |
| `compiler/parser/api.go` | 46 | GetBasePath |
| `presentation/admin/admin.go` | 1534, 1609 | User queries |
| `cmd/bitcode/main.go` | 335 | CLI user listing |

### Seeder Update

| File | Change |
|------|--------|
| `module/seeder.go` | Accept module prefix, use resolver for table names |

### Module JSON Updates

| File | Prefix |
|------|--------|
| `engine/embedded/modules/base/module.json` | `"table": {"prefix": "res"}` |
| `engine/modules/crm/module.json` | `"table": {"prefix": "crm"}` |
| `engine/modules/sales/module.json` | `"table": {"prefix": "sale"}` |
| `samples/erp/modules/crm/module.json` | `"table": {"prefix": "crm"}` |
| `samples/erp/modules/hrm/module.json` | `"table": {"prefix": "hr"}` |

### Seed Data File Updates

All seed data JSON files that use table name keys need updating to match new table names (no 's' suffix, with prefix where applicable).

## Breaking Changes

1. **Table names** — all tables lose 's' suffix, gain module prefix. Existing databases need table renames.
2. **API paths** — `/api/contacts` → `/api/contact`. All API consumers must update.
3. **Seed data keys** — JSON keys in data files must match new table names.

## Testing

- Unit tests for `ResolveTableName()` — all 4 resolution paths
- Unit tests for schema config loading
- Integration: `go test ./...` must pass
- Verify migration creates tables with correct names
