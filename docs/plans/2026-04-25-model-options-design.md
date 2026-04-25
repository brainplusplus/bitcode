# Model Options: version, timestamps, timestamps_by, soft_deletes, soft_deletes_by

## Summary

Add model-level options to control automatic column generation and behavior for versioning (optimistic locking), timestamps, audit tracking, and soft deletes.

## Model JSON Schema

```json
{
  "name": "contact",
  "version": false,
  "timestamps": true,
  "timestamps_by": true,
  "soft_deletes": false,
  "soft_deletes_by": false,
  "fields": { ... }
}
```

## Options

| Option | Type | Default | Columns Generated |
|--------|------|---------|-------------------|
| `version` | bool | `false` | `version INTEGER NOT NULL DEFAULT 1` |
| `timestamps` | bool | `true` | `created_at DATETIME`, `updated_at DATETIME` |
| `timestamps_by` | bool | `true` | `created_by UUID`, `updated_by UUID` |
| `soft_deletes` | bool | `false` | `deleted_at DATETIME` (nullable) |
| `soft_deletes_by` | bool | `false` | `deleted_by UUID` |

Note: `active BOOLEAN DEFAULT TRUE` is always generated — it is a business field, not a delete mechanism.

## Key Concept: active vs soft_deletes

These are two **separate** concepts:

- **`active`** — A business field. An inactive record still appears in lists but cannot be used in selections/dropdowns. User-controlled toggle. Always present.
- **`soft_deletes`** — Recycle bin. A soft-deleted record is hidden from all default queries. Can be restored. Only present when `soft_deletes: true`.

## Behavior

### `version: true` — Optimistic Locking

- Adds `version INTEGER NOT NULL DEFAULT 1` column to the table.
- On **create**: `version` is set to `1`.
- On **update**: The update query includes `WHERE version = ?` with the current version, and increments `version` by 1. If 0 rows affected, returns HTTP 409 Conflict with error `"record has been modified by another user"`.
- Client must send `version` in the update body to participate in optimistic locking.
- Applies to both SQL (GenericRepository) and MongoDB (MongoRepository).

### `timestamps: true` (default)

- Adds `created_at` and `updated_at` columns.
- `created_at` is set on create, never updated.
- `updated_at` is set on create and updated on every update.
- When `false`, these columns are not generated and not auto-populated.

### `timestamps_by: true` (default)

- Adds `created_by` and `updated_by` columns (UUID FK to users).
- `created_by` is set on create, never updated.
- `updated_by` is set on create and updated on every update.
- When `false`, these columns are not generated and not auto-populated.

### `soft_deletes: true`

- Adds `deleted_at` column (nullable datetime).
- On delete: sets `deleted_at = NOW()` AND `active = false`.
- `FindAll` filters `WHERE deleted_at IS NULL` to exclude soft-deleted records.
- When `false` (default), `deleted_at` column is not generated. `FindAll` has no auto-filter.

### `soft_deletes_by: true`

- Adds `deleted_by` column (UUID FK to users).
- On soft delete: sets `deleted_by` to the current user ID.
- When `false` (default), `deleted_by` column is not generated.

## Query Methods

Two sets of query methods exist on every repository:

### Base methods (exclude soft-deleted only)

| Method | Filter Applied |
|--------|---------------|
| `FindByID` / `Get` / `Find` | No filter |
| `FindAll` / `GetAll` | `deleted_at IS NULL` (when soft_deletes enabled) |
| `Paginate` | Same as FindAll + total_pages |
| `Count` | Same filter as FindAll |
| `Sum` | Same filter as FindAll |

### Active variants (exclude soft-deleted + inactive)

| Method | Filter Applied |
|--------|---------------|
| `FindActive` | `active = true` + `deleted_at IS NULL` |
| `FindAllActive` | `active = true` + `deleted_at IS NULL` |
| `PaginateActive` | Same as FindAllActive + total_pages |
| `CountActive` | Same filter as FindAllActive |
| `SumActive` | Same filter as FindAllActive |

All registered in `models.{name}.{Op}` process registry.

## Files Changed

### Parser (`engine/internal/compiler/parser/model.go`)
- Add `Version`, `Timestamps`, `TimestampsBy`, `SoftDeletes` fields to `ModelDefinition`
- Helper methods: `IsVersion()`, `IsTimestamps()`, `IsTimestampsBy()`, `IsSoftDeletes()`

### Migration (`engine/internal/infrastructure/persistence/dynamic_model.go`)
- `buildColumns()` conditionally generates columns based on model options
- `MergeInheritedFields()` carries over model options from parent to child

### Repository Interface (`engine/internal/infrastructure/persistence/repository_interface.go`)
- Add `FindActive`, `FindAllActive`, `CountActive`, `SumActive` to `Repository` interface

### SQL Repository (`engine/internal/infrastructure/persistence/repository.go`)
- `applyNotDeleted()` — filters `deleted_at IS NULL` when soft_deletes enabled
- `applyActiveFilter()` — filters `active = true` + soft delete filter
- `FindAllActive()`, `FindActive()`, `CountActive()`, `SumActive()` — Active variants
- `UpdateWithVersion()` — optimistic locking (WHERE version = ? + increment)
- `SoftDeleteWithTimestamp()` — sets deleted_at + active = false

### MongoDB Repository (`engine/internal/infrastructure/persistence/mongo_repository.go`)
- Same Active variants and new methods as SQL repository

### CRUD Handler (`engine/internal/presentation/api/crud_handler.go`)
- `Create()`: set `version = 1` when model has version enabled
- `Update()`: optimistic locking with version check, returns 409 on conflict
- `Delete()`: uses `SoftDeleteWithTimestamp` when model has soft_deletes
- Respects `timestamps_by` flag for created_by/updated_by

### Model Process Registry (`engine/internal/runtime/model_process.go`)
- Register: `FindActive`, `FindAllActive`, `PaginateActive`, `CountActive`, `SumActive`

### Tests (`engine/internal/compiler/parser/model_test.go`)
- 3 new tests: defaults, explicit values, partial specification

## Backward Compatibility

- All defaults match current behavior (`timestamps: true`, `timestamps_by: true`)
- `active` column always generated (existing queries unaffected)
- `version` defaults to `false` (no change for existing models)
- `soft_deletes` defaults to `false` (no change for existing models)
- `FindAll` without soft_deletes has no auto-filter (changed from previous `WHERE active = true`)
- Existing model JSON files without these options work identically to before
