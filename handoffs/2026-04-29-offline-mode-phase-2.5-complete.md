# Handoff: Offline Mode — Phase 1, 2, 2.5 Complete

**Date:** 2026-04-29
**Next:** Phase 3 (Sync Engine)

## What Was Done

### Phase 1: Foundation (previous session)
- Engine understands `mode:"offline"` in module.json and bitcode.toml
- `_off_*` columns auto-appended to offline model tables (device_id, status, version, deleted, hlc, envelope_id)
- `_off_uuid` added when PK is not UUID strategy
- 4 client-side infrastructure tables: `_off_outbox`, `_off_sync_state`, `_off_conflict_log`, `_off_number_sequence`
- 4 server-side sync tables: `_sync_log`, `_sync_devices`, `_sync_conflicts`, `_sync_versions`
- PK validation: composite PK rejected for offline models
- 5 sync API stubs returning 501

### Phase 2: Tauri Shell (this session)
- Tauri 2.0 project at `packages/tauri/` — Cargo.toml, main.rs, tauri.conf.json, capabilities
- Plugins: tauri-plugin-sql (SQLite), tauri-plugin-fs, tauri-plugin-notification
- Mobile plugins (barcode-scanner, biometric) behind `mobile-plugins` feature flag
- `bc-native.ts` bridge — 13 methods with Tauri IPC / Web API fallback
- Build pipeline: `npm run dev:desktop`, `build:desktop`, `dev:android`, `build:android`, `dev:ios`, `build:ios`
- Icons generated for all platforms

### Phase 2.5: Per-Model Offline & Client Routing (this session)
- `model.json` supports `"app": {"mode": "offline"}` per-model
- Resolution chain: model → module → project → default("online")
- `GET /api/v1/sync/schema` endpoint returns offline models + field definitions
- `offline-store.ts` routes CRUD to SQLite (offline) or fetch() (online)
- `BcSetup.registerOfflineModels()`, `isModelOffline()`, `getOfflineModels()`
- Outbox recording on create/update/delete for offline models

## Verification Results

| Check | Result |
|-------|--------|
| Go build | ✅ |
| Go tests | ✅ 570 passed, 41 packages |
| Tauri Rust build | ✅ cargo check passes |
| TypeScript LSP | ✅ 0 errors |
| Stencil tests | ✅ 74 passed, 6 suites |

## Commits Created (7 atomic commits, not pushed)

```
038c953 docs: update project docs for offline mode Phase 1, 2, 2.5 completion
c6c5c75 docs: add offline mode design doc, implementation plan, and feature reference
48a90ee feat(components): add offline-store.ts — CRUD routing to SQLite/fetch() with outbox
e738b79 feat(components): add bc-native.ts bridge — 13 methods with Tauri IPC/Web API fallback
7c93e43 feat(tauri): scaffold Tauri 2.0 native shell — plugins, SQLite migrations, build pipeline
469bf9d feat(engine): add per-model offline mode with resolution chain
ec9ce8e feat(engine): add offline mode server infrastructure — _off_* schema, _sync_* tables, sync API endpoints
```

## Key Files (for quick navigation)

### Go Engine
| File | What |
|------|------|
| `engine/internal/compiler/parser/model.go` | `ModelAppConfig`, `IsOffline()` on ModelDefinition |
| `engine/internal/compiler/parser/module.go` | `ModuleAppConfig`, `OfflineConfig`, `IsOffline()` on ModuleDefinition |
| `engine/internal/domain/model/registry.go` | `RegisterWithModule` resolution chain, `ProjectAppMode` |
| `engine/internal/infrastructure/persistence/offline_schema.go` | `OfflineColumns()`, `OfflineUUIDColumn()`, client `_off_*` DDLs |
| `engine/internal/infrastructure/persistence/sync_schema.go` | Server `_sync_*` DDLs (PostgreSQL/MySQL/SQLite) |
| `engine/internal/presentation/api/sync_handler.go` | 6 endpoints: 5 stubs + `GetSchema` |
| `engine/internal/app.go` | `initSyncInfrastructure()`, sync route registration |

### TypeScript (packages/components/src/core/)
| File | What |
|------|------|
| `bc-native.ts` | Bridge: 13 methods, Tauri detection via `window.__TAURI__` |
| `bc-setup.ts` | `registerOfflineModels()`, `isModelOffline()`, `getOfflineModels()` |
| `offline-store.ts` | `find()`, `findById()`, `create()`, `update()`, `delete()`, `initFromServer()` |

### Tauri (packages/tauri/src-tauri/)
| File | What |
|------|------|
| `Cargo.toml` | Tauri 2.10 + plugins, `mobile-plugins` feature flag |
| `src/main.rs` | Plugin registration, 4 SQLite migrations for `_off_*` tables |
| `tauri.conf.json` | `frontendDist: ../../components/www`, `withGlobalTauri: true` |
| `capabilities/default.json` | Permissions: core, sql, fs, notification |

### Docs
| File | What |
|------|------|
| `docs/plans/2026-04-28-offline-mode-design.md` | Architecture decisions, edge cases, all trade-offs |
| `docs/plans/impl/offline-mode-implementation.md` | All 5 phases with tasks, acceptance criteria |
| `engine/docs/features/offline-mode.md` | Feature reference — config, API, bridge, files |

## What's Next: Phase 3 (Sync Engine)

Phase 3 is the hardest phase — involves both Go (server) and TypeScript (client) code.

### Task 3.1: Device Registration Flow
- **Server:** Implement `POST /api/v1/sync/register` in `sync_handler.go` (currently returns 501)
  - Accept: `{ platform, app_version, store_id }`
  - Create record in `_sync_devices` table
  - Return: `{ device_id, device_prefix, registered_at }`
- **Client:** Call registration from `offline-store.ts` on first launch
  - Store device_id in `_off_sync_state` via `BcNative.dbExecute()`

### Task 3.2: Outbox Recording
- `offline-store.ts` already records to `_off_outbox` on create/update/delete — this is done
- Need to add `_off_device_id` to outbox records (requires device registration first)
- Need to add `envelope_id` grouping for related operations

### Task 3.3: Sync Push
- **Client:** Read PENDING records from `_off_outbox`, send to server
- **Server:** Implement `POST /api/v1/sync/push` — process envelope, apply to PostgreSQL
  - Idempotency via `_sync_log` (check `idempotency_key` before processing)
  - Update `_sync_versions` with new version number
  - Return: `{ envelope_id, status: "applied"|"conflict", version }`
- **Client:** Mark outbox records as SYNCED on success

### Task 3.4: Sync Pull
- **Server:** Implement `GET /api/v1/sync/pull?since_version=N&device_id=X`
  - Query `_sync_versions` for changes since version N
  - Return: `{ changes: [...], max_version }`
- **Client:** Apply changes to local SQLite, update `last_pull_version` in `_off_sync_state`

### Task 3.5: Envelope Grouping
- Group related operations (e.g., POS sale = header + items + payment) into atomic envelopes
- All operations in an envelope share the same `envelope_id`
- Server processes entire envelope atomically (all-or-nothing)

## Critical Context

1. **`_sync_*` tables are Go-generated, NOT JSON models.** Same pattern as `audit_log`, `ir_migration`. Future task: evaluate migration to JSON models when base module is refactored.

2. **`_off_*` tables defined in TWO places** (known duplication):
   - `offline_schema.go` (Go) — server-side schema generation
   - `main.rs` (Rust) — client-side SQLite migrations
   Once Phase 3 schema sync is done, client should receive schema from server, eliminating Rust-side duplication.

3. **`bc-native.ts` uses `window.__TAURI__` global** — requires `withGlobalTauri: true` in tauri.conf.json. No npm dependency on `@tauri-apps/api`.

4. **Mobile plugins behind feature flag** — `cargo build --features mobile-plugins` for barcode/biometric.

5. **`offline-store.ts` intercepts at data layer** — components don't need modification. `BcSetup.isModelOffline()` determines routing.

6. **Do NOT touch** `engine/internal/runtime/bridge/`, `engine/internal/runtime/goja/`, `engine/internal/runtime/embedded/` — separate work in progress.

7. **`sprints/` folder** is owner's personal notes — never commit generated content there.

## Known Issues, Potential Bugs & Improvements

### Bugs / Must Fix (before Phase 3)

1. **SQL injection risk in `offline-store.ts`** — `buildSelectSQL` interpolates `table` and field names directly into SQL strings (e.g., `SELECT * FROM ${table}`). While table names come from server schema (not user input), filter keys from `params.filters` are also interpolated as column names without sanitization. If a component passes user-controlled filter keys, this is exploitable. **Fix:** validate column names against known schema fields before interpolation.

2. **`_off_outbox` missing `device_id` column** — `recordOutbox()` in `offline-store.ts` doesn't write `device_id` because device registration (Phase 3.1) isn't done yet. Once device registration is implemented, every outbox record MUST include the device_id. Without it, the server can't attribute operations to devices.

3. **`offline-store.ts` create/update are NOT transactional** — `BcNative.dbExecute()` for the data write and `recordOutbox()` are two separate calls. If the app crashes between them, data is written but not recorded in outbox (will never sync). **Fix:** wrap in a SQLite transaction (`BEGIN; ... COMMIT;`) — requires adding transaction support to `BcNative`.

4. **`GetSchema` endpoint field order is non-deterministic** — Go's `map[string]FieldDefinition` iteration order is random. The schema endpoint returns fields in arbitrary order each time. Not a bug per se, but makes client-side caching/diffing unreliable. **Fix:** sort fields by name before returning.

5. **`www/index.html` is in `.gitignore`** — The entry HTML for Tauri WebView was created but lives in `packages/components/www/` which is gitignored (build output). Tauri's `beforeBuildCommand` regenerates it, but if someone clones fresh and runs `cargo tauri dev` without building Stencil first, it will fail with a missing file error. **Fix:** either add `www/index.html` to git (exclude from .gitignore), or add a check in Tauri's build script.

### Improvements (nice to have)

6. **`offline-store.ts` search is naive** — `buildSelectSQL` search only checks `id` and `_off_uuid` columns with LIKE. Real search should query the model's `search_field` config (same fields the server searches). Requires schema endpoint to include `search_field` in response.

7. **No local table creation from schema** — `OfflineStore.initFromServer()` fetches the schema but doesn't CREATE TABLE locally. It only registers model names. The actual SQLite tables are created by Tauri migrations in `main.rs` (hardcoded). Phase 3 should add dynamic table creation from schema response.

8. **`BcNative.takePhoto()` and `getLocation()` Tauri plugin commands are unverified** — The invoke strings (`plugin:camera|take_photo`, `plugin:geolocation|get_position`) are based on docs but haven't been tested against actual Tauri plugins. Camera and geolocation are NOT in our Cargo.toml (only sql, fs, notification are). These will fail at runtime until the plugins are added.

9. **`generateUUIDv7()` in offline-store.ts is a custom implementation** — It follows the UUIDv7 spec (timestamp + random) but isn't using a battle-tested library. Edge cases: clock rollback, sub-millisecond collisions on fast devices. For production, consider using a proper UUID library or validating against RFC 9562.

10. **No retry logic for `OfflineStore.initFromServer()`** — If the server is unreachable on first launch, offline models are never registered. The app silently falls back to online mode for everything. Should retry periodically or cache the last known schema in SQLite.

11. **`offline-store.ts` doesn't handle module-qualified model names** — The engine supports `crm.lead` (module.model) syntax, but `OfflineStore.find('crm.lead')` won't match `BcSetup.isModelOffline('crm.lead')` if only `'lead'` was registered. Need to handle both qualified and unqualified names.

12. **Tauri CSP is permissive** — `tauri.conf.json` has `unsafe-inline` and `unsafe-eval` in script-src, and `connect-src` allows `http:`. This is fine for development but should be tightened for production builds.

13. **No error handling in `offline-store.ts` CRUD** — If `BcNative.dbExecute()` fails (e.g., constraint violation, disk full), the error propagates unhandled. Should catch, log to `_off_conflict_log`, and return a meaningful error to the component.

14. **`_off_version` column not incremented** — `offline-store.ts` create/update don't set `_off_version`. This column is needed for optimistic locking and conflict detection in Phase 4. Should auto-increment on every write.

### Technical Debt

15. **`_off_*` table schema duplication** — Defined in both `offline_schema.go` (Go) and `main.rs` (Rust). If one is updated without the other, schema drift occurs. Tracked as known issue, will be resolved when Phase 3 implements schema sync from server.

16. **`sync_handler.go` has no tests** — The `GetSchema` endpoint works but has zero test coverage. Should add unit tests that verify only offline models are returned, field mapping is correct, and online models are excluded.

17. **`offline-store.ts` `fetchFromServer` duplicates `data-fetcher.ts` logic** — The server fallback functions rebuild URL params and headers manually instead of reusing `doFetch()` from `data-fetcher.ts`. Should refactor to share the fetch logic.

18. **Per-view `app.mode` override not implemented** — The design doc mentions view-level override (`view.app.mode → model.app.mode → ...`) but only model and module levels are implemented. View-level is lower priority but should be tracked.

## How to Verify Current State

```bash
# Go
cd engine && go build ./... && go test ./...
# Expected: 570 tests pass, 41 packages

# Stencil
cd packages/components && npm test
# Expected: 74 tests pass, 6 suites

# Tauri
cd packages/tauri/src-tauri && cargo check
# Expected: compiles without errors
```
