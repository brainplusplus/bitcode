# Handoff: Offline Mode — Phase 4 (Conflict Resolution & Edge Cases) Complete

**Date:** 2026-05-01
**Previous:** Phase 3 (see `2026-04-30-offline-mode-phase-3-complete.md`)
**Next:** Phase 5 (Polish & Cross-Platform Testing)

## What Was Done

### Task 4.1: Hybrid Logical Clock (HLC)

- **Created:** `packages/components/src/core/hlc.ts`
  - Format: `"{wall_time_base36}:{logical_counter_base36}:{device_id}"` (e.g. `"01jk5p9q:0001:DEV-A"`)
  - `hlcNow()` — generate timestamp for local event, monotonically increasing
  - `hlcReceive(remoteHlc)` — Lamport-style merge when receiving remote timestamp
  - `hlcCompare(a, b)` — deterministic comparison (wall time → logical → device_id)
  - `parseHLC()` / `formatHLC()` — serialization
  - Clock skew guard: rejects if skew exceeds 60 seconds
- **Created:** `packages/components/src/core/hlc.spec.ts` — 18 tests
- **Wired into `offline-store.ts`:** `create()` and `update()` now populate `_off_hlc` column
- `setDeviceId()` also sets HLC device ID via `hlcSetDeviceId()`

### Task 4.2: Field-Level Conflict Merge

- **Created:** `engine/internal/runtime/sync/conflict.go`
  - `ResolveFieldConflicts(base, local, remote, localHLC, remoteHLC)` — field-by-field merge
    - Different fields edited → auto-merge, both changes preserved
    - Same field edited → HLC determines winner, conflict logged
    - Same value on both sides → no conflict
    - System fields (`_off_*`, `_sync_*`, `id`) skipped
  - `ResolveEditVsDelete(editData, isLocalEdit)` — edit wins, record resurrected
  - `RecordConflictsToServer(db, ...)` — inserts into `_sync_conflicts` for admin review
  - `resolveByHLC()` — Go-side HLC comparison (base36 parsing)
- **Created:** `engine/internal/runtime/sync/conflict_test.go` — 12 tests
- **Modified `offline-store.ts` `syncPull()`:**
  - For each incoming change, checks if local record has pending modifications (`_off_status = 'PENDING'` AND `_off_version > 1`)
  - If conflict detected on UPDATE: calls `mergeFieldLevel()` which compares field-by-field using HLC
  - If conflict detected on DELETE (edit vs delete): edit wins, delete is ignored
  - Conflicts logged to `_off_conflict_log` table
  - `syncPull()` now returns `{ applied, conflicts }` instead of just `{ applied }`

### Task 4.3: Device-Prefixed Receipt Numbering

- **Added `getNextReceiptNumber(tableName)` to `OfflineStore`**
  - Reads from `_off_number_sequence` table (already exists in SQLite migrations)
  - Falls back to `device_prefix` from `_off_sync_state` if no sequence exists
  - Format: `{store_code}-{device_letter}-{zero_padded_sequence}` (e.g. `"001-A-0016"`)
  - Uses `INSERT ... ON CONFLICT DO UPDATE` for atomic upsert
  - Sequential per device, survives app restart, no collision across devices

### Task 4.4: Inventory Delta-Based Tracking

- **Created:** `engine/internal/runtime/sync/inventory.go`
  - `ApplyInventoryDelta(tx, delta, envelopeID)` — applies qty delta to record field
    - If resulting stock < 0: creates oversell alert in `_sync_oversell_alerts`, but accepts the sale
    - Never blocks a sale — accept then reconcile (industry standard)
  - `DetectInventoryFields(payload)` — detects `*_delta` fields (convention: `qty_delta`, `stock_delta`)
  - `CreateOversellAlertsTable(db)` — DDL for `_sync_oversell_alerts` (PostgreSQL/MySQL/SQLite)
- **Created:** `engine/internal/runtime/sync/inventory_test.go` — 8 tests

## Verification Results

| Check | Result |
|-------|--------|
| Go build (my packages) | ✅ |
| Go tests (sync package) | ✅ 20 passed |
| TypeScript LSP | ✅ 0 errors |
| Stencil tests | ✅ 110 passed, 7 suites |

## Key Files Changed

| File | What Changed |
|------|-------------|
| `packages/components/src/core/hlc.ts` | **NEW** — HLC implementation |
| `packages/components/src/core/hlc.spec.ts` | **NEW** — 18 HLC tests |
| `packages/components/src/core/offline-store.ts` | HLC wiring in create/update, conflict detection in syncPull, `getNextReceiptNumber()`, `mergeFieldLevel()`, `logConflict()` |
| `packages/components/src/core/offline-store.spec.ts` | 24 tests (up from 17): HLC wiring, conflict detection, edit-vs-delete, receipt numbering |
| `engine/internal/runtime/sync/conflict.go` | **NEW** — Field-level conflict resolution + HLC comparison |
| `engine/internal/runtime/sync/conflict_test.go` | **NEW** — 12 conflict tests |
| `engine/internal/runtime/sync/inventory.go` | **NEW** — Inventory delta tracking + oversell alerts |
| `engine/internal/runtime/sync/inventory_test.go` | **NEW** — 8 inventory tests |
| `docs/plans/impl/offline-mode-implementation.md` | Phase 4 status → COMPLETE |
| `engine/docs/features/offline-mode.md` | Updated implementation files table |
| `docs/features.md` | Updated feature row to Phase 1-4 |

## What's Next: Phase 5 (Polish & Cross-Platform Testing)

1. SQLite encryption at rest
2. Offline auth caching (72-hour token)
3. Cross-platform testing (desktop + mobile)
4. Performance optimization (batch sync, compression)
5. Production CSP hardening

## Critical Context

1. **`syncPull()` return type changed** — now returns `{ applied: number; conflicts: number }` instead of `{ applied: number }`. Callers need to handle the new `conflicts` field.

2. **HLC is wired into create/update** — every offline write now populates `_off_hlc`. This is used during conflict resolution in `syncPull()`.

3. **`_sync_oversell_alerts` table** — new server-side table. Call `CreateOversellAlertsTable(db)` during server init to create it. Not yet wired into `PushEnvelope` — the `DetectInventoryFields()` and `ApplyInventoryDelta()` functions are ready but need to be called from the push handler when inventory fields are detected.

4. **Conflict resolution is client-side during pull** — the `mergeFieldLevel()` function in `offline-store.ts` handles field-level merge. Server-side `RecordConflictsToServer()` is available but needs to be called from the push handler for server-side conflict logging.

5. **Do NOT touch** `engine/internal/runtime/bridge/`, `engine/internal/runtime/embedded/`, `engine/internal/runtime/goja/` — separate work in progress by another agent.

6. **`sprints/` folder** is owner's personal notes — never commit generated content there.

## Known Issues (remaining)

6. `offline-store.ts` search is naive — only checks `id` and `_off_uuid` with LIKE
7. No local table creation from schema — tables still hardcoded in Rust migrations
8. `takePhoto()` and `getLocation()` Tauri plugin commands unverified
9. `generateUUIDv7()` is custom implementation — not battle-tested library
10. No retry logic for `OfflineStore.initFromServer()`
11. `offline-store.ts` doesn't handle module-qualified model names
12. Tauri CSP is permissive (fine for dev, tighten for production)
13. No error handling in CRUD — errors propagate unhandled to component
14. `_sync_oversell_alerts` table not yet auto-created during server init
15. `DetectInventoryFields()` / `ApplyInventoryDelta()` not yet wired into `PushEnvelope`
16. Server-side `RecordConflictsToServer()` not yet called from push handler
