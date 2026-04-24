# Sprint 1.11 — Computed Fields, Data Versioning, Schema Builder

**Date**: 24 April 2026
**Status**: Complete

## Objectives

1. **Feature #5: Computed/Formula Fields** — Runtime expression evaluator for computed fields
2. **Feature #6: Data Versioning** — Full before/after snapshots with rollback/restore
3. **Feature #1: Schema Builder** — Admin Schema tab with visual + JSON editor

## Deliverables

### Feature #5: Computed/Formula Fields ✅

**New files:**
- `engine/internal/runtime/expression/evaluator.go` — Full expression evaluator with lexer, parser, AST
- `engine/internal/runtime/expression/hydrator.go` — Computed field hydrator, loads one2many children
- `engine/internal/runtime/expression/evaluator_test.go` — 17 tests

**What it supports:**
- Arithmetic: `+`, `-`, `*`, `/`, `%`
- Comparisons: `==`, `!=`, `<`, `<=`, `>`, `>=`
- Boolean: `&&`, `||`, `!`
- Functions: `sum`, `count`, `avg`, `min`, `max`, `abs`, `round`, `ceil`, `floor`, `if`
- Field references: `quantity`, `unit_price`
- Dot-path access: `record.field`
- One2many aggregates: `sum(lines.subtotal)`, `count(items.qty)`
- String concatenation: `first + ' ' + last`
- Nested child computed fields (e.g. `order_line.subtotal = quantity * unit_price`, then `order.total = sum(lines.subtotal)`)

**Integration points:**
- `GenericRepository.FindByID()` — hydrates computed fields after fetch
- `GenericRepository.FindAll()` — hydrates computed fields for all records
- `GenericRepository.FindByCompositePK()` — hydrates computed fields
- View renderer `renderList()` and `renderForm()` — hydrates for SSR views

### Feature #6: Data Versioning ✅

**New files:**
- `engine/internal/infrastructure/persistence/data_revision.go` — DataRevision model + repository
- `engine/internal/infrastructure/persistence/data_revision_test.go` — 7 tests

**What it does:**
- `data_revisions` table with `(model_name, record_id, version, action, snapshot, changes, user_id, created_at)`
- Automatic snapshot on every Create/Update/Delete via GenericRepository hooks
- Before/after change diff computed automatically (`ComputeChanges()`)
- Monotonic per-record version numbers
- Cleanup support (keep N most recent revisions)
- Restore creates new head revision (immutable history)

**Admin API endpoints:**
- `GET /admin/api/data/:model/:id/revisions` — list revision history
- `GET /admin/api/data/:model/:id/revisions/:version` — get specific revision with snapshot
- `POST /admin/api/data/:model/:id/restore/:version` — restore record from snapshot

**Modified files:**
- `repository.go` — Added `revisionRepo`, `modelName`, `currentUser` fields; snapshot hooks in Create/Update/Delete/HardDelete
- `app.go` — Auto-migrate `data_revisions` table, wire revision repo into router
- `router.go` — Pass revision repo to CRUD handlers
- `crud_handler.go` — Set current user for revision tracking

### Feature #1: Schema Builder ⚠️ (Partial)

**What was added:**
- New "Schema" tab on `/admin/models/:name?tab=schema`
- Visual field table showing all fields with type, label, required, options, relations, computed expressions
- JSON editor mode with raw model JSON editing
- Visual/JSON toggle
- Save endpoint: `POST /admin/api/models/:name` — validates JSON, writes to disk
- Load endpoint: `GET /admin/api/models/:name/json` — returns model JSON

**What's still missing:**
- Drag-and-drop field reordering
- Add/remove field UI (currently edit JSON directly)
- Stencil `bc-schema-builder` component (currently inline HTML)
- Auto-migration on save (currently requires server restart)

## Test Results

- **Before**: 230 tests, 0 failures
- **After**: 247 tests, 0 failures (+17 expression, +7 data revision, -7 moved to new naming)
- **Build**: Clean, no errors

## Files Changed

| File | Change |
|------|--------|
| `engine/internal/runtime/expression/evaluator.go` | NEW — Expression evaluator engine |
| `engine/internal/runtime/expression/hydrator.go` | NEW — Computed field hydrator |
| `engine/internal/runtime/expression/evaluator_test.go` | NEW — 17 tests |
| `engine/internal/infrastructure/persistence/data_revision.go` | NEW — Data revision model + repository |
| `engine/internal/infrastructure/persistence/data_revision_test.go` | NEW — 7 tests |
| `engine/internal/infrastructure/persistence/repository.go` | MODIFIED — Added hydrator, revision hooks |
| `engine/internal/presentation/api/router.go` | MODIFIED — Added hydrator + revision repo passing |
| `engine/internal/presentation/api/crud_handler.go` | MODIFIED — Set current user for revisions |
| `engine/internal/presentation/admin/admin.go` | MODIFIED — Added Schema tab, data revision repo |
| `engine/internal/presentation/admin/admin_api.go` | MODIFIED — Added model + data revision API endpoints |
| `engine/internal/presentation/view/renderer.go` | MODIFIED — Added hydrator for computed fields in views |
| `engine/internal/app.go` | MODIFIED — Wire hydrator, data revisions, full router |
| `docs/features.md` | UPDATED — Features #1, #5, #6 status |
| `docs/codebase.md` | UPDATED — New file entries |
| `AGENTS.md` | UPDATED — Computed field marked complete |
