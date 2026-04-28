# Handoff: Offline Mode — Phase 5 (Polish & Cross-Platform) Complete

**Date:** 2026-05-01
**Previous:** Phase 4 (see `2026-05-01-offline-mode-phase-4-complete.md`)
**Status:** ALL 5 PHASES COMPLETE — Offline mode is production-ready infrastructure.

## What Was Done

### Task 5.1: SQLite Encryption at Rest

- **Modified:** `packages/tauri/src-tauri/Cargo.toml` — added `encryption` feature flag
- **Modified:** `packages/tauri/src-tauri/src/main.rs` — `db_connection_string()` reads `BITCODE_DB_KEY` env var when `encryption` feature is enabled, passes key to SQLite connection string
- **How to enable:** `cargo build --features encryption` + set `BITCODE_DB_KEY` environment variable
- **Requires:** SQLCipher library installed on build system (documented in Cargo.toml)

### Task 5.2: Offline Authentication with 72-Hour Expiry

- **Modified:** `engine/internal/presentation/api/sync_handler.go` — implemented `CacheAuth` endpoint (was 501 stub)
  - Validates credentials against server `users` table
  - Returns SHA-256 hash of password (not bcrypt — client can verify SHA-256 natively)
  - Returns user groups for offline permission checks
  - Returns `expires_at` (72 hours from cache time)
  - Updates `_sync_devices.user_id` on successful cache
- **Modified:** `packages/components/src/core/offline-store.ts` — added:
  - `cacheAuth(baseUrl, username, password)` — calls server, stores credentials in `_off_auth_cache` table
  - `authenticateOffline(username, password)` — verifies against cached SHA-256 hash
  - Brute-force protection: 5 failed attempts → 15-minute lockout (stored in `_off_sync_state`)
  - Expiry check: rejects if `expires_at` has passed
- **Modified:** `packages/tauri/src-tauri/src/main.rs` — added migration v5 for `_off_auth_cache` table + `failed_auth_attempts`/`locked_until` columns to `_off_sync_state`

### Task 5.3: Sync Status UI Component

- **Created:** `packages/components/src/components/widgets/bc-sync-status/bc-sync-status.tsx`
  - Shows online/offline status with colored dot indicator
  - Shows pending sync count, error count, conflict count
  - Shows last sync time in human-readable format ("2m ago", "3h ago")
  - Manual "Sync Now" button (disabled when offline or syncing)
  - Compact mode (`compact` prop) for toolbar/header placement
  - Configurable poll interval (default 30s)
  - Emits `bcSyncTriggered` and `bcSyncCompleted` events
- **Created:** `packages/components/src/components/widgets/bc-sync-status/bc-sync-status.css`
- **Added to `offline-store.ts`:** `getSyncStatus()` and `syncAll()` methods

### Task 5.4: Performance Optimization

- **Modified:** `packages/components/src/core/offline-store.ts`:
  - `configureSyncOptions({ batchSize, enableCompression, pullPageSize, maxRetries })` — configurable sync parameters
  - `syncPush()` now uses `LIMIT` based on configured batch size (default 100)
  - `syncPush()` now compresses payloads > 1KB using `CompressionStream` API (gzip) when available
  - `syncPull()` now passes `limit` parameter to server for paginated pulls
  - `syncPull()` sends `Accept-Encoding: gzip` header

### Task 5.5: CSP Hardening

- **Modified:** `packages/tauri/src-tauri/tauri.conf.json`:
  - Removed `'unsafe-eval'` from `script-src` — replaced with `'wasm-unsafe-eval'`
  - Removed `http:` and `ws:` from `connect-src` — production uses HTTPS/WSS only
  - Added `object-src 'none'` — blocks plugins
  - Added `base-uri 'self'` — prevents base tag injection
  - Added `form-action 'self'` — prevents form submission to external domains
  - Added `frame-ancestors 'none'` — prevents clickjacking

### Task 5.6: Cross-Platform Guards

- **Modified:** `packages/components/src/core/bc-native.ts` — added 3 new methods:
  - `isOnline()` — checks `navigator.onLine`
  - `onConnectivityChange(callback)` — subscribes to online/offline events, returns unsubscribe function
  - `getPlatformInfo()` — returns `{ platform, isMobile, hasCamera, hasGeo }`

## Verification Results

| Check | Result |
|-------|--------|
| Go build (full project) | ✅ |
| Go tests (sync package) | ✅ 20 passed |
| TypeScript LSP | ✅ 0 errors |
| Stencil tests | ✅ 120 passed, 7 suites |

## Key Files Changed

| File | What Changed |
|------|-------------|
| `packages/tauri/src-tauri/Cargo.toml` | Added `encryption` feature flag + build instructions |
| `packages/tauri/src-tauri/src/main.rs` | `db_connection_string()` for encrypted DB, migration v5 (`_off_auth_cache`), `failed_auth_attempts`/`locked_until` in `_off_sync_state` |
| `packages/tauri/src-tauri/tauri.conf.json` | Hardened CSP |
| `engine/internal/presentation/api/sync_handler.go` | `CacheAuth` fully implemented (was stub), added `sha256Hex()`, added `crypto/sha256` import |
| `packages/components/src/core/offline-store.ts` | `cacheAuth()`, `authenticateOffline()`, `getSyncStatus()`, `syncAll()`, `configureSyncOptions()`, batch sync with compression, brute-force lockout |
| `packages/components/src/core/offline-store.spec.ts` | 34 tests (up from 24): auth caching, offline auth, sync status, batch config |
| `packages/components/src/core/bc-native.ts` | `isOnline()`, `onConnectivityChange()`, `getPlatformInfo()` |
| `packages/components/src/components/widgets/bc-sync-status/bc-sync-status.tsx` | **NEW** — Sync status UI component |
| `packages/components/src/components/widgets/bc-sync-status/bc-sync-status.css` | **NEW** — Sync status styles |
| `engine/docs/features/offline-mode.md` | Updated implementation files table to Phase 1-5 |
| `docs/features.md` | Offline Mode → ✅ COMPLETE, updated counts |
| `docs/plans/impl/offline-mode-implementation.md` | Phase 4+5 → ✅ COMPLETE |

## Known Limitations

See `engine/docs/features/offline-mode.md` § Known Limitations for the canonical list. Summary:

1. `takePhoto()` / `getLocation()` unverified on mobile (requires physical device)
2. `generateUUIDv7()` is custom implementation (works, not battle-tested at scale)
3. Module-qualified model names (`crm.lead`) not handled — only bare names (`lead`)
4. Local table creation hardcoded in Rust migrations (not dynamic from schema)
5. `'unsafe-inline'` still required in CSP (Stencil limitation)
6. Offline auth uses SHA-256 not bcrypt (deliberate tradeoff, mitigated by expiry + lockout)

## Critical Context

1. **`CacheAuth` is now fully implemented** — no longer returns 501. Callers should call it after successful online login to cache credentials for offline use.

2. **Offline auth flow:** `cacheAuth()` (online) → stores SHA-256 hash in `_off_auth_cache` → `authenticateOffline()` (offline) → verifies against cached hash. After 72 hours, forces re-login.

3. **Brute-force protection:** 5 failed offline auth attempts → 15-minute lockout. Stored in `_off_sync_state.failed_auth_attempts` and `locked_until`.

4. **Encryption is opt-in:** Requires `--features encryption` at build time AND `BITCODE_DB_KEY` env var at runtime. Without both, database is unencrypted (same as before).

5. **CSP change may break dev workflows** — removed `http:` and `ws:` from `connect-src`. Dev servers using HTTP/WS will need to temporarily relax CSP or use HTTPS/WSS.

6. **Do NOT touch** `engine/internal/runtime/bridge/`, `engine/internal/runtime/embedded/`, `engine/internal/runtime/goja/` — separate work in progress by another agent.

7. **`sprints/` folder** is owner's personal notes — never commit generated content there.
