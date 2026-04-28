# Handoff: Offline Mode — Phase 3 (Sync Engine) Complete

**Date:** 2026-04-30
**Previous:** Phase 1, 2, 2.5 (see `2026-04-29-offline-mode-phase-2.5-complete.md`)
**Next:** Phase 4 (Conflict Resolution & Edge Cases)

## What Was Done

### Bug Fixes (5 bugs from Phase 2.5 handoff)

1. **SQL injection risk in `offline-store.ts`** — Added two-layer defense: regex validation (`/^[a-zA-Z_][a-zA-Z0-9_]*$/`) for all identifiers + schema registry lookup that validates column names against known fields from server schema. `assertSafeIdentifier()`, `assertValidColumn()`, `assertSafeTable()` guard every SQL interpolation point.

2. **`_off_outbox` missing `device_id`** — Added `device_id` column to outbox schema in both Go (`offline_schema.go`) and Rust (`main.rs`). `recordOutbox()` now writes `_deviceId` to every outbox record. `OfflineStore.setDeviceId()` sets the device identity after registration.

3. **Non-transactional create/update/delete** — All write operations now wrapped in `BEGIN TRANSACTION` / `COMMIT` with `ROLLBACK` on error. Data write and outbox recording are atomic — no orphaned data or missing outbox entries on crash.

4. **`GetSchema` field order non-deterministic** — Added `sort.Slice(fields, ...)` to sort fields by name before returning. Client-side caching/diffing is now reliable.

5. **Tauri fresh clone fails** — Added `beforeDevCommand` to `tauri.conf.json` so `cargo tauri dev` automatically builds Stencil components first.

### Additional Fix (from Known Issues #14)

- **`_off_version` not incremented** — `create()` now sets `_off_version: 1`, `update()` uses raw SQL `_off_version = _off_version + 1`, `delete()` also increments version. Ready for optimistic locking in Phase 4.

### Phase 3: Sync Engine

#### Task 3.1: Device Registration Flow
- **Server:** `POST /api/v1/sync/register` fully implemented in `sync_handler.go`
  - Accepts `{ platform, app_version, store_id }`
  - Generates unique `device_id` (format: `DEV-{random_hex}`) and `device_prefix` (format: `NNN-X`)
  - Inserts into `_sync_devices` table
  - Returns `{ device_id, device_prefix, registered_at }`
- **Client:** `OfflineStore.registerDevice()` in `offline-store.ts`
  - Checks local `_off_sync_state` first (skip if already registered)
  - Calls server endpoint, persists result to `_off_sync_state`
  - Sets `_deviceId` for all subsequent outbox records
  - Auto-detects platform from user agent

#### Task 3.2: Outbox Recording
- `recordOutbox()` now includes `envelope_id`, `device_id`, and `created_at`
- `_off_version` auto-incremented on every write
- `_off_device_id` set on create operations
- Outbox schema updated in both Go and Rust to include `device_id` column

#### Task 3.3: Sync Push
- **Client:** `OfflineStore.syncPush()` reads PENDING outbox entries, groups by `envelope_id`, sends each envelope to server
  - On success: marks outbox entries as SYNCED
  - On failure: increments `retry_count`, marks as DEAD after 5 retries
  - Network errors: keeps as PENDING for next retry
- **Server:** `POST /api/v1/sync/push` processes envelope atomically
  - Idempotency via `_sync_log` — duplicate envelope returns cached response
  - Begins DB transaction, applies each operation (CREATE/UPDATE/DELETE)
  - Records version in `_sync_versions` for each operation
  - Commits or rolls back entire envelope
  - Logs to `_sync_log` with status, timing, error details
  - Updates `_sync_devices.last_sync_at` and `last_sync_version`
  - Table name and column name validation prevents SQL injection on server side

#### Task 3.4: Sync Pull
- **Client:** `OfflineStore.syncPull()` reads `last_pull_version` from `_off_sync_state`, fetches changes from server
  - Applies changes in a transaction (CREATE via INSERT OR REPLACE, UPDATE, DELETE as soft-delete)
  - Updates `last_pull_version` and `last_sync_at` after successful apply
  - Validates all table/column names from server response
- **Server:** `GET /api/v1/sync/pull?since_version=N&device_id=X`
  - Queries `_sync_versions` for changes since version N
  - Excludes changes made by the requesting device (`changed_by != device_id`)
  - Fetches current record data for each change
  - Returns `{ changes: [...], max_version, count }`
  - Supports pagination via `limit` parameter (max 5000)

#### Task 3.5: Envelope Grouping
- `OfflineStore.beginTransaction()` returns an `envelope_id`
- All operations until `commitTransaction()` share that envelope_id
- Server processes entire envelope in one DB transaction (all-or-nothing)
- Operations outside a transaction get individual auto-generated envelope IDs

## Verification Results

| Check | Result |
|-------|--------|
| Go build | ✅ |
| Go tests | ✅ 588 passed, 42 packages |
| TypeScript LSP | ✅ 0 errors |
| Stencil tests | ✅ 80 passed, 6 suites |

## Key Files Changed

| File | What Changed |
|------|-------------|
| `engine/internal/presentation/api/sync_handler.go` | Full implementation: `RegisterDevice`, `PushEnvelope`, `PullChanges`, `DeviceStatus` + helpers |
| `engine/internal/infrastructure/persistence/offline_schema.go` | Added `device_id` column to `_off_outbox` DDL |
| `packages/components/src/core/offline-store.ts` | Major rewrite: SQL injection prevention, transactions, device registration, syncPush, syncPull, envelope grouping |
| `packages/components/src/core/offline-store.spec.ts` | 17 tests (up from 11): transactions, rollback, SQL injection, envelope grouping |
| `packages/tauri/src-tauri/src/main.rs` | Added `device_id` column to outbox migration |
| `packages/tauri/src-tauri/tauri.conf.json` | Added `beforeDevCommand` |
| `docs/plans/impl/offline-mode-implementation.md` | Phase 3 status → COMPLETE |
| `engine/docs/features/offline-mode.md` | Updated implementation files table |

## What's Next: Phase 4 (Conflict Resolution & Edge Cases)

### Task 4.1: Hybrid Logical Clock (HLC)
- TypeScript HLC at `packages/components/src/core/hlc.ts`
- `now()`, `receive(remote)`, `compare(a, b)` operations
- Monotonically increasing, handles clock skew up to 1 minute
- Tie-breaking by device_id

### Task 4.2: Field-Level Conflict Merge
- During pull, detect if local record was also modified
- Compare field by field: remote-only → accept, local-only → keep, both → HLC wins
- Log conflicts in `_off_conflict_log` (client) and `_sync_conflicts` (server)

### Task 4.3: Device-Prefixed Receipt Numbering
- `getNextReceiptNumber(tableName)` using `_off_number_sequence`
- Format: `{store_code}-{device_letter}-{zero_padded_sequence}`

### Task 4.4: Inventory Delta-Based Tracking
- Sync inventory as deltas (`qty_delta: -5`), not absolutes
- Allow negative stock, create oversell alerts

## Critical Context

1. **`_off_outbox` now has `device_id` column** — schema changed in both Go and Rust. Existing databases from Phase 2.5 will need migration (the column has `DEFAULT ''` so it's backward compatible with `CREATE TABLE IF NOT EXISTS`).

2. **SQL injection prevention is two-layered** — regex + schema lookup. Schema is populated from server's `GET /api/v1/sync/schema` response. If schema isn't loaded yet (first launch before `initFromServer()`), only regex validation applies.

3. **Server-side `PushEnvelope` uses raw SQL** — not GORM models, because sync tables are infrastructure tables (same pattern as audit_log). All table/column names are validated via `isValidTableName()` before interpolation.

4. **`_off_version` is now incremented** — `create()` sets to 1, `update()`/`delete()` use `_off_version + 1` in raw SQL. This is ready for optimistic locking in Phase 4.

5. **Do NOT touch** `engine/internal/runtime/bridge/`, `engine/internal/runtime/embedded/`, `engine/internal/runtime/goja/` — separate work in progress by another agent.

6. **`sprints/` folder** is owner's personal notes — never commit generated content there.

## Known Issues (remaining from Phase 2.5 + new)

### From Phase 2.5 (still applicable)

6. `offline-store.ts` search is naive — only checks `id` and `_off_uuid` with LIKE
7. No local table creation from schema — tables still hardcoded in Rust migrations
8. `takePhoto()` and `getLocation()` Tauri plugin commands unverified
9. `generateUUIDv7()` is custom implementation — not battle-tested library
10. No retry logic for `OfflineStore.initFromServer()`
11. `offline-store.ts` doesn't handle module-qualified model names
12. Tauri CSP is permissive (fine for dev, tighten for production)
13. No error handling in CRUD — errors propagate unhandled to component

### Resolved (post Phase 3)

19. ~~`syncPush` per-envelope retry~~ — **Fixed:** after 3 failed retries, envelope is split into individual operations. Each operation is pushed separately; successful ones are marked SYNCED, failed ones get ERROR/DEAD independently.

20. ~~`PullChanges` fetches full record data~~ — **Fixed:** deduplicates per record (only latest version entry per table+record_id), and for UPDATE operations with `changed_fields`, returns only the delta (changed fields + id) instead of the full record.

21. ~~`recordSyncVersion` race condition~~ — **Fixed:** PostgreSQL uses `INSERT ... RETURNING version` (BIGSERIAL handles concurrency). SQLite/MySQL uses atomic `INSERT ... SELECT COALESCE(MAX(version),0)+1 FROM _sync_versions` in a single statement.

22. ~~`DeviceStatus` read-only~~ — **Fixed:** added `PATCH /api/v1/sync/devices/:device_id` endpoint for updating device name, activating/deactivating with reason. `PushEnvelope` now rejects deactivated devices with 403.

## How to Verify Current State

```bash
# Go
cd engine && go build ./... && go test ./...
# Expected: 588 tests pass, 42 packages

# Stencil
cd packages/components && npm test
# Expected: 80 tests pass, 6 suites

# Tauri
cd packages/tauri/src-tauri && cargo check
# Expected: compiles without errors
```
