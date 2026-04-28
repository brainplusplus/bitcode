# Offline Mode — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Enable BitCode applications to work offline using Tauri as native shell, with automatic sync when connectivity is restored.

**Architecture:** Stencil components (unchanged) run inside Tauri's native WebView. A bridge abstraction (`bc-native.ts`) routes native capability calls to Tauri IPC or Web API fallback. The Go engine generates `_off_*` columns/tables for offline tracking and provides sync API endpoints. A Rust-based client sync engine handles outbox, conflict resolution, and background sync.

**Tech Stack:** Tauri 2.0 (Rust), Stencil (TypeScript), Go (engine), SQLite (local), PostgreSQL (server), UUIDv7

**Design Doc:** `docs/plans/2026-04-28-offline-mode-design.md`
**Feature Ref:** `engine/docs/features/offline-mode.md`

---

## Phase Overview

| Phase | Name | Scope | Effort | Status | Deliverable |
|-------|------|-------|--------|--------|-------------|
| **1** | Foundation | Config parsing + `_off_*` schema generation + sync API stubs | 2-3 weeks | ✅ COMPLETE | Engine understands `mode: "offline"` and generates correct schemas |
| **2** | Tauri Shell | Tauri project setup + Stencil integration + `bc-native.ts` bridge | 1-2 weeks | ✅ COMPLETE | Stencil components run inside Tauri on desktop + mobile |
| **2.5** | Per-Model Offline & Client Routing | Per-model `app.mode`, resolution chain, `offline-store.ts`, schema sync endpoint | 3-5 days | ✅ COMPLETE | Granular per-model offline + client CRUD routing |
| **3** | Sync Engine | Client outbox + server sync endpoints + idempotency + delta sync | 2-3 weeks | ✅ COMPLETE | Data syncs between local SQLite and server PostgreSQL |
| **4** | Conflict Resolution & Edge Cases | Field-level merge + HLC + receipt numbering + inventory handling | 1-2 weeks | ✅ COMPLETE | All edge cases from design doc handled |
| **5** | Polish & Cross-Platform Testing | Encryption, offline auth, sync UI, performance, CSP hardening | 1-2 weeks | ✅ COMPLETE | Production-ready infrastructure |

**Total: 8-13 weeks** (honest estimate for 1 developer)

---

## Phase 1: Foundation (2-3 weeks) — ✅ COMPLETE

**Goal:** Engine understands `mode: "offline"`, generates `_off_*` columns on local SQLite schemas, creates `_off_*` infrastructure tables, and provides sync API stubs.

**Why this first:** Everything else depends on the engine knowing about offline mode. Without this, Tauri has nothing to sync, and the bridge has no local database to talk to.

### Task 1.1: Add `app` config to Module and Project

**What:** Add `app.mode` field to `ModuleDefinition` and `AppConfig` so the engine can detect offline modules.

**Files:**
- Modify: `engine/internal/compiler/parser/module.go` — add `App` field to `ModuleDefinition` struct
- Modify: `engine/internal/config.go` — add `app.mode` default to viper config
- Test: `engine/internal/compiler/parser/module_test.go` — test parsing `app.mode` from module JSON

**Detail:**

In `module.go`, add to `ModuleDefinition` struct:
```go
type AppConfig struct {
    Mode string `json:"mode,omitempty"` // "online" (default) | "offline"
}

type ModuleDefinition struct {
    // ... existing fields ...
    App *AppConfig `json:"app,omitempty"`
}

func (m *ModuleDefinition) IsOffline() bool {
    if m.App == nil {
        return false
    }
    return m.App.Mode == "offline"
}
```

In `config.go`, add defaults:
```go
v.SetDefault("app.mode", "online")
```

**Acceptance criteria:**
- `module.json` with `"app": {"mode": "offline"}` parses correctly
- `module.IsOffline()` returns `true` for offline modules, `false` for online
- `bitcode.toml` with `[app] mode = "offline"` parses correctly
- Missing `app` config defaults to `"online"`

**Cascading resolution logic:**
```go
func ResolveAppMode(projectMode string, moduleMode string, viewMode string) string {
    if viewMode != "" {
        return viewMode
    }
    if moduleMode != "" {
        return moduleMode
    }
    if projectMode != "" {
        return projectMode
    }
    return "online"
}
```

---

### Task 1.2: Add `_off_*` columns to schema generation

**What:** When a module is offline, the engine must add `_off_*` tracking columns to every table in that module during schema migration.

**Files:**
- Modify: `engine/internal/infrastructure/persistence/dynamic_model.go` — add offline columns in `buildColumns()`
- Create: `engine/internal/infrastructure/persistence/offline_schema.go` — offline-specific schema logic
- Test: `engine/internal/infrastructure/persistence/offline_schema_test.go`

**Detail:**

Create `offline_schema.go`:
```go
package persistence

import "fmt"

// OfflineColumns returns the _off_* columns to add to a table when module is offline.
// These columns are auto-generated — developers do NOT define them.
func OfflineColumns(dialect DBDialect) string {
    switch dialect {
    case DialectSQLite:
        return `
    _off_device_id    TEXT NOT NULL DEFAULT '',
    _off_status       INTEGER NOT NULL DEFAULT 0,
    _off_version      INTEGER NOT NULL DEFAULT 1,
    _off_deleted      INTEGER NOT NULL DEFAULT 0,
    _off_created_at   TEXT NOT NULL DEFAULT '',
    _off_updated_at   TEXT NOT NULL DEFAULT '',
    _off_hlc          TEXT NOT NULL DEFAULT '',
    _off_envelope_id  TEXT`
    case DialectPostgres:
        // Server does NOT get _off_* columns — return empty
        return ""
    default:
        return ""
    }
}

// OfflineUUIDColumn returns the _off_uuid column needed when PK is NOT uuid strategy.
// This provides a client-generated identity for sync when the real PK is server-assigned.
func OfflineUUIDColumn(pkStrategy PKStrategy, dialect DBDialect) string {
    if pkStrategy == PKUUID {
        return "" // UUID PK already serves as sync identity
    }
    if dialect != DialectSQLite {
        return "" // Server doesn't need _off_uuid
    }
    return "_off_uuid TEXT NOT NULL UNIQUE"
}
```

Modify `buildColumns()` in `dynamic_model.go` to call `OfflineColumns()` when module is offline:
```go
func buildColumns(model *parser.ModelDefinition, dialect DBDialect) string {
    // ... existing column building logic ...

    // If module is offline, append _off_* columns (SQLite only)
    if model.IsOfflineModule && dialect == DialectSQLite {
        cols = append(cols, OfflineColumns(dialect))
        if extraCol := OfflineUUIDColumn(model.PrimaryKey.Strategy, dialect); extraCol != "" {
            cols = append(cols, extraCol)
        }
    }

    return strings.Join(cols, ",\n")
}
```

**Acceptance criteria:**
- Offline module + SQLite → table has all 8 `_off_*` columns
- Offline module + non-UUID PK + SQLite → table also has `_off_uuid` column
- Offline module + PostgreSQL → table has NO `_off_*` columns (server stays clean)
- Online module → table has NO `_off_*` columns regardless of dialect
- All `_off_*` columns have correct types and defaults

---

### Task 1.3: Create `_off_*` infrastructure tables

**What:** When a module is offline, create the 4 infrastructure tables: `_off_outbox`, `_off_sync_state`, `_off_conflict_log`, `_off_number_sequence`.

**Files:**
- Modify: `engine/internal/infrastructure/persistence/offline_schema.go` — add infrastructure table creation
- Test: `engine/internal/infrastructure/persistence/offline_schema_test.go`

**Detail:**

Add to `offline_schema.go`:
```go
// CreateOfflineInfrastructureTables creates the 4 _off_* tables needed for sync.
// Called once per offline module during migration.
func CreateOfflineInfrastructureTables(db *gorm.DB) error {
    tables := []string{
        createOfflineOutboxSQL(),
        createOfflineSyncStateSQL(),
        createOfflineConflictLogSQL(),
        createOfflineNumberSequenceSQL(),
    }
    for _, sql := range tables {
        if err := db.Exec(sql).Error; err != nil {
            return fmt.Errorf("failed to create offline infrastructure table: %w", err)
        }
    }
    return nil
}
```

Each function returns the full CREATE TABLE SQL as documented in `engine/docs/features/offline-mode.md`:
- `_off_outbox` — 10 columns (id, envelope_id, table_name, record_id, operation, payload, status, idempotency_key, created_at, retry_count)
- `_off_sync_state` — 8 columns (device_id, device_prefix, last_sync_at, last_pull_version, registered_at, auth_cached_at, user_id, user_hash)
- `_off_conflict_log` — 10 columns (id, table_name, record_id, field_name, local_value, remote_value, resolved_value, resolution, resolved_at, device_id)
- `_off_number_sequence` — 4 columns (id, table_name, prefix, last_sequence) + UNIQUE(table_name, prefix)

**Acceptance criteria:**
- All 4 tables created with correct columns, types, defaults, and constraints
- Tables are idempotent (IF NOT EXISTS) — safe to run multiple times
- Tables are created in SQLite dialect (TEXT for dates, INTEGER for booleans)

---

### Task 1.4: Create server-side sync tables

**What:** Create the 4 server-side sync infrastructure tables: `_sync_log`, `_sync_devices`, `_sync_conflicts`, `_sync_versions`.

**Files:**
- Create: `engine/internal/infrastructure/persistence/sync_schema.go` — server sync table creation
- Test: `engine/internal/infrastructure/persistence/sync_schema_test.go`

**Detail:**

Create `sync_schema.go`:
```go
// CreateSyncInfrastructureTables creates server-side sync tables in PostgreSQL.
// Called once during engine startup if any module has mode: "offline".
func CreateSyncInfrastructureTables(db *gorm.DB) error {
    tables := []string{
        createSyncLogSQL(),
        createSyncDevicesSQL(),
        createSyncConflictsSQL(),
        createSyncVersionsSQL(),
    }
    for _, sql := range tables {
        if err := db.Exec(sql).Error; err != nil {
            return fmt.Errorf("failed to create sync table: %w", err)
        }
    }
    return nil
}
```

Each function returns full CREATE TABLE SQL as documented in `engine/docs/features/offline-mode.md`:
- `_sync_log` — 8 columns (envelope_id UUID PK, device_id, received_at, status, operations_count, response JSONB, error_message, processing_time_ms)
- `_sync_devices` — 14 columns (device_id PK, device_prefix UNIQUE, device_name, platform, app_version, user_id, tenant_id, store_id, registered_at, last_sync_at, last_sync_version, is_active, deactivated_at, deactivated_reason)
- `_sync_conflicts` — 16 columns (id SERIAL PK, envelope_id, device_id, other_device_id, table_name, record_id, field_name, device_value, server_value, resolved_value, resolution, auto_resolved, reviewed_by, reviewed_at, created_at, device_hlc, server_hlc)
- `_sync_versions` — 8 columns (id SERIAL PK, table_name, record_id, operation, version BIGINT, changed_fields JSONB, changed_by, created_at)

**Acceptance criteria:**
- All 4 tables created with correct PostgreSQL types (UUID, TIMESTAMPTZ, JSONB, SERIAL, BIGINT)
- Tables are idempotent (IF NOT EXISTS)
- `_sync_versions.version` has a sequence or trigger for auto-increment
- Indexes on frequently queried columns: `_sync_versions(version)`, `_sync_devices(tenant_id)`, `_sync_conflicts(record_id)`

---

### Task 1.5: Sync API stubs

**What:** Create the sync API endpoints (stubs that return 501 Not Implemented for now). This establishes the API contract early.

**Files:**
- Create: `engine/internal/presentation/api/sync_handler.go` — sync API handler stubs
- Modify: `engine/internal/presentation/api/router.go` — register sync routes

**Detail:**

Create `sync_handler.go` with these endpoints:

| Method | Path | Purpose |
|--------|------|---------|
| `POST` | `/api/v1/sync/register` | Register a new device. Returns device_id and device_prefix |
| `POST` | `/api/v1/sync/push` | Push an envelope of operations from client to server |
| `GET` | `/api/v1/sync/pull` | Pull changes from server since a given version |
| `POST` | `/api/v1/sync/auth/cache` | Get cached auth credentials for offline use |
| `GET` | `/api/v1/sync/status` | Get sync status for a device (last sync, pending conflicts) |

In `router.go`, add sync route group:
```go
func (r *Router) RegisterSyncRoutes() {
    sync := r.app.Group("/api/v1/sync")
    handler := NewSyncHandler(r.db)
    
    sync.Post("/register", handler.RegisterDevice)
    sync.Post("/push", handler.PushEnvelope)
    sync.Get("/pull", handler.PullChanges)
    sync.Post("/auth/cache", handler.CacheAuth)
    sync.Get("/status", handler.DeviceStatus)
}
```

**Acceptance criteria:**
- All 5 endpoints registered and reachable
- Each returns 501 with `{"error": "not implemented", "message": "sync endpoint coming in Phase 3"}`
- Endpoints are behind auth middleware (device must be authenticated)
- API contract (request/response shapes) documented in handler comments

---

### Task 1.6: Validate offline + PK compatibility

**What:** When engine loads a module with `mode: "offline"`, validate that the PK strategy is compatible. Warn for `auto_increment`/`natural_key`, error for `composite`.

**Files:**
- Modify: `engine/internal/compiler/parser/model.go` — add offline validation in model parsing
- Test: `engine/internal/compiler/parser/model_test.go`

**Detail:**

Add validation during model parsing:
```go
func validateOfflinePK(model *ModelDefinition, isOfflineModule bool) error {
    if !isOfflineModule {
        return nil
    }
    
    strategy := model.PrimaryKey.Strategy
    
    switch strategy {
    case PKUUID:
        // Best choice for offline. No warning.
        return nil
    case PKAutoIncrement, PKNaturalKey, PKNamingSeries, PKManual:
        // Works but with extra complexity. Log warning.
        log.Printf("⚠️  WARNING: Model %q uses pk:%q with offline mode. "+
            "This requires _off_uuid column for sync identity. "+
            "Consider using pk:\"uuid\" for simpler offline support.", 
            model.Name, strategy)
        return nil
    case PKComposite:
        // Not supported offline.
        return fmt.Errorf("model %q uses pk:\"composite\" which is not supported in offline mode. "+
            "Use pk:\"uuid\" or pk:\"natural_key\" instead", model.Name)
    default:
        return nil
    }
}
```

**Acceptance criteria:**
- `uuid` PK + offline → no warning, no error
- `auto_increment` PK + offline → warning logged, no error, `_off_uuid` column added
- `natural_key` PK + offline → warning logged, no error, `_off_uuid` column added
- `composite` PK + offline → error, engine refuses to start
- Online module → no validation regardless of PK strategy

---

### Phase 1 Checkpoint

After Phase 1, you can verify:
1. `bitcode.toml` with `[app] mode = "offline"` is parsed correctly
2. `module.json` with `"app": {"mode": "offline"}` is parsed correctly
3. SQLite schema for offline modules includes all `_off_*` columns
4. PostgreSQL schema for offline modules does NOT include `_off_*` columns
5. 4 client-side `_off_*` tables are created in SQLite
6. 4 server-side `_sync_*` tables are created in PostgreSQL
7. Sync API endpoints are reachable (returning 501)
8. PK compatibility validation works (warn/error as expected)

**This phase produces NO user-visible changes.** It's pure infrastructure. But it's the foundation everything else builds on.

---

## Phase 2: Tauri Shell (1-2 weeks) — ✅ COMPLETE

**Goal:** Stencil components run inside Tauri on desktop and mobile. `bc-native.ts` bridge detects environment and routes to Tauri IPC or Web API fallback.

**Why this second:** With the engine foundation in place, we need the native shell to actually run the app offline. This phase makes the app "installable" on desktop/mobile.

### Task 2.1: Create Tauri project

**What:** Initialize a Tauri 2.0 project that serves the Stencil component build output.

**Files:**
- Create: `packages/tauri/` — new Tauri project directory
- Create: `packages/tauri/src-tauri/` — Rust backend
- Create: `packages/tauri/src-tauri/Cargo.toml` — Rust dependencies
- Create: `packages/tauri/src-tauri/src/main.rs` — Tauri entry point
- Create: `packages/tauri/src-tauri/tauri.conf.json` — Tauri configuration
- Create: `packages/tauri/src-tauri/capabilities/default.json` — permissions

**Detail:**

`tauri.conf.json` key settings:
```json
{
  "build": {
    "frontendDist": "../../components/dist"
  },
  "app": {
    "withGlobalTauri": true,
    "windows": [
      {
        "title": "BitCode",
        "width": 1280,
        "height": 800,
        "resizable": true
      }
    ]
  },
  "bundle": {
    "active": true,
    "targets": "all",
    "identifier": "com.bitcode.app"
  }
}
```

`Cargo.toml` dependencies:
```toml
[dependencies]
tauri = { version = "2", features = [] }
tauri-plugin-sql = { version = "2", features = ["sqlite"] }
tauri-plugin-notification = "2"
tauri-plugin-fs = "2"
serde = { version = "1", features = ["derive"] }
serde_json = "1"
```

**Acceptance criteria:**
- `cargo tauri dev` launches a window showing Stencil components
- Stencil components render correctly in Tauri WebView
- Window title, size, and resizing work
- Build produces installable binaries for current platform

---

### Task 2.2: Create `bc-native.ts` bridge

**What:** Create the bridge abstraction layer that detects Tauri and routes native calls.

**Files:**
- Create: `packages/components/src/core/bc-native.ts` — bridge abstraction
- Test: `packages/components/src/core/bc-native.spec.ts`

**Detail:**

Full implementation as documented in `docs/plans/2026-04-28-offline-mode-design.md` Section 4. All 8 methods:
- `takePhoto()` — camera via Tauri plugin or `<input capture>` fallback
- `getLocation()` — GPS via Tauri plugin or `navigator.geolocation` fallback
- `dbExecute()` — SQLite via Tauri plugin or IndexedDB fallback
- `scanBarcode()` — barcode scanner via Tauri plugin or JS library fallback
- `authenticate()` — biometrics via Tauri plugin or WebAuthn fallback
- `saveFile()` — filesystem via Tauri plugin or blob download fallback
- `requestNotificationPermission()` — notifications via Tauri plugin or Web Notification API
- `syncData()` — sync trigger via Tauri command or fetch() fallback
- `getEnvironment()` — returns `'browser'` | `'tauri-desktop'` | `'tauri-mobile'`

**Acceptance criteria:**
- In browser: all methods use Web API fallbacks, no errors
- In Tauri: all methods route to Tauri `invoke()` calls
- `getEnvironment()` correctly detects browser vs Tauri desktop vs Tauri mobile
- TypeScript types are correct and exported
- No runtime errors when `__TAURI__` is not available (browser)

---

### Task 2.3: Add Tauri plugins for native capabilities

**What:** Configure Tauri plugins for camera, GPS, barcode, biometrics, filesystem, notifications.

**Files:**
- Modify: `packages/tauri/src-tauri/Cargo.toml` — add plugin dependencies
- Modify: `packages/tauri/src-tauri/src/main.rs` — register plugins
- Modify: `packages/tauri/src-tauri/capabilities/default.json` — grant permissions

**Detail:**

Plugins to add:
```toml
# Cargo.toml
tauri-plugin-sql = { version = "2", features = ["sqlite"] }
tauri-plugin-fs = "2"
tauri-plugin-notification = "2"
tauri-plugin-barcode-scanner = "2"
tauri-plugin-biometric = "2"
# Camera and geolocation may need community plugins or custom Rust commands
```

Register in `main.rs`:
```rust
fn main() {
    tauri::Builder::default()
        .plugin(tauri_plugin_sql::Builder::new().build())
        .plugin(tauri_plugin_fs::init())
        .plugin(tauri_plugin_notification::init())
        .plugin(tauri_plugin_barcode_scanner::init())
        .plugin(tauri_plugin_biometric::init())
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
```

**Acceptance criteria:**
- SQLite plugin works: can create database, execute queries, read results
- Filesystem plugin works: can write and read files
- Notification plugin works: can request permission and send notification
- All plugins have correct permissions in `capabilities/default.json`

**Honest caveat:** Camera and geolocation plugins for Tauri 2.0 mobile are less mature than desktop. May need custom Rust commands wrapping platform-specific APIs. This is a known risk — budget extra time for mobile-specific issues.

---

### Task 2.4: Build pipeline integration

**What:** Set up build pipeline so Stencil build output is automatically available to Tauri.

**Files:**
- Modify: `packages/tauri/src-tauri/tauri.conf.json` — point to Stencil dist
- Create: `packages/tauri/package.json` — npm scripts for dev/build
- Modify: root `package.json` — add tauri workspace scripts

**Detail:**

`packages/tauri/package.json`:
```json
{
  "name": "@bitcode/tauri",
  "scripts": {
    "dev": "cargo tauri dev",
    "build": "cargo tauri build",
    "build:android": "cargo tauri android build",
    "build:ios": "cargo tauri ios build"
  }
}
```

Build flow:
```
npm run build:components  →  packages/components/dist/
npm run build:tauri       →  reads from packages/components/dist/
                          →  produces native binary
```

**Acceptance criteria:**
- `npm run dev` in tauri package starts dev server with hot reload
- `npm run build` produces installable binary
- Stencil component changes reflect in Tauri app after rebuild
- Build works on Windows, macOS, and Linux

---

### Phase 2 Checkpoint

After Phase 2, you can verify:
1. Tauri app launches and shows Stencil components correctly
2. `BcNative.getEnvironment()` returns `'tauri-desktop'` inside Tauri
3. `BcNative.getEnvironment()` returns `'browser'` in normal browser
4. SQLite database can be created and queried via `BcNative.dbExecute()`
5. Native notifications work via `BcNative.requestNotificationPermission()`
6. Build produces installable binaries

**This phase produces a VISIBLE deliverable:** the app runs natively on desktop. But it's still online-only — no sync yet.

---

## Phase 2.5: Per-Model Offline & Client Routing (added post-Phase 2) — ✅ COMPLETE

**Goal:** Enable per-model offline mode (not just per-module), and create the client-side routing layer that directs CRUD to local SQLite or server API based on model config.

**Why this before Phase 3:** Phase 3 (Sync Engine) needs to know WHICH models are offline at a per-model granularity. Without this, the sync engine can't selectively sync only offline models. The client also needs `offline-store.ts` to intercept CRUD before sync can record operations in the outbox.

### Task 2.5.1: Per-model `app.mode` in model.json

**What:** Add `app` field to `ModelDefinition` so individual models can opt into offline mode independently of their module.

**Files:**
- Modify: `engine/internal/compiler/parser/model.go` — add `App *ModelAppConfig` field
- Modify: `engine/internal/domain/model/registry.go` — resolution chain: model → module → project
- Modify: `engine/internal/compiler/parser/model_test.go` — test per-model offline parsing

**Resolution chain:**
```
model.app.mode → module.app.mode → project app.mode → "online"
```

**Example — lead offline, contact online:**
```json
// modules/crm/module.json — module is online by default
{ "name": "crm", "app": { "mode": "online" } }

// modules/crm/models/lead.json — lead overrides to offline
{ "name": "lead", "app": { "mode": "offline" }, "fields": [...] }

// modules/crm/models/contact.json — no app field, inherits module's "online"
{ "name": "contact", "fields": [...] }
```

**Acceptance criteria:**
- Model with `"app": {"mode": "offline"}` is marked `OfflineModule = true` regardless of module setting
- Model without `app` field inherits from module, then project
- `go test ./...` passes
- Composite PK validation still rejects offline models with composite PK

---

### Task 2.5.2: Wire per-model offline to schema generation

**What:** Ensure `dynamic_model.go` uses the resolved per-model offline flag (not just module-level).

**Files:**
- Modify: `engine/internal/domain/model/registry.go` — `RegisterWithModule` checks model-level `app.mode` too
- Verify: `engine/internal/infrastructure/persistence/dynamic_model.go` — already uses `model.OfflineModule`, no change needed

**Acceptance criteria:**
- Lead (offline) gets `_off_*` columns in its table
- Contact (online) does NOT get `_off_*` columns
- Both in the same CRM module

---

### Task 2.5.3: Schema sync endpoint

**What:** Server endpoint that returns the list of offline models and their field definitions, so the Tauri client knows which tables to create locally.

**Files:**
- Modify: `engine/internal/presentation/api/sync_handler.go` — add `GetSchema` endpoint
- Modify: `engine/internal/app.go` — register the new route

**Endpoint:** `GET /api/v1/sync/schema`

**Response:**
```json
{
  "models": [
    {
      "name": "lead",
      "module": "crm",
      "table_name": "crm_lead",
      "fields": [
        { "name": "name", "type": "string", "required": true },
        { "name": "email", "type": "string" },
        { "name": "status", "type": "selection", "options": ["new","contacted","qualified"] }
      ],
      "primary_key": { "strategy": "uuid", "version": "v7" }
    }
  ],
  "offline_config": {
    "max_offline_hours": 72,
    "sync_batch_size": 100
  }
}
```

**Acceptance criteria:**
- Only offline models are returned
- Online models are excluded
- Field definitions match the model JSON
- Endpoint is accessible without device registration (needed for initial setup)

---

### Task 2.5.4: `offline-store.ts` — client-side CRUD routing

**What:** Create an intercept layer that routes CRUD operations to local SQLite (for offline models) or normal fetch() (for online models).

**Files:**
- Create: `packages/components/src/core/offline-store.ts`
- Modify: `packages/components/src/core/bc-setup.ts` — add `registerOfflineModels()` and `isModelOffline()` methods
- Create: `packages/components/src/core/offline-store.spec.ts`

**How it works:**
1. On app init, call `GET /api/v1/sync/schema` to get offline model list
2. Register offline models via `BcSetup.registerOfflineModels(models)`
3. `OfflineStore.find(model, params)` checks `BcSetup.isModelOffline(model)`:
   - If offline → `BcNative.dbSelect()` with SQL built from params
   - If online → normal `fetch()` to server API
4. `OfflineStore.save(model, data)` checks same flag:
   - If offline → `BcNative.dbExecute()` INSERT/UPDATE + record in `_off_outbox`
   - If online → normal `fetch()` POST/PUT to server API

**Acceptance criteria:**
- `OfflineStore.find('lead', ...)` routes to SQLite when lead is offline
- `OfflineStore.find('contact', ...)` routes to fetch() when contact is online
- Save operations on offline models record in `_off_outbox`
- All existing Stencil tests still pass (no breaking changes)

---

### Phase 2.5 Checkpoint

After Phase 2.5, you can verify:
1. `lead.json` with `"app": {"mode": "offline"}` gets `_off_*` columns, `contact.json` without it does not
2. `GET /api/v1/sync/schema` returns only offline models
3. `OfflineStore.find('lead', ...)` queries local SQLite
4. `OfflineStore.find('contact', ...)` calls server API
5. All Go tests pass, all Stencil tests pass

---

## Phase 3: Sync Engine (2-3 weeks)

**Goal:** Data syncs bidirectionally between local SQLite (Tauri) and server PostgreSQL. Outbox pattern, idempotency, delta sync all working.

**Why this third:** With the foundation (Phase 1) and shell (Phase 2) in place, we can now build the actual sync logic. This is the hardest phase.

### Task 3.1: Device registration flow

**What:** Implement the device registration endpoint and client-side registration logic.

**Files:**
- Modify: `engine/internal/presentation/api/sync_handler.go` — implement `RegisterDevice`
- Create: `packages/tauri/src-tauri/src/device.rs` — device registration logic
- Create: `packages/tauri/src-tauri/src/config.rs` — device config persistence

**Detail:**

Server endpoint `POST /api/v1/sync/register`:
```
Request:  { platform: "android", app_version: "1.0.0", store_id: "uuid" }
Response: { device_id: "DEV-A", device_prefix: "001-A", registered_at: "..." }
```

Server logic:
1. Generate unique `device_id` (format: `DEV-{letter}` or `DEV-{random}`)
2. Generate unique `device_prefix` (format: `{store_code}-{device_letter}`)
3. INSERT into `_sync_devices`
4. Return device credentials

Client logic (Rust):
1. On first app launch, check if device is registered (check local config file)
2. If not registered, call `/api/v1/sync/register`
3. Store `device_id` and `device_prefix` in local config file
4. Subsequent launches skip registration

**Acceptance criteria:**
- First launch → device registers successfully
- Second launch → skips registration, uses cached credentials
- Device prefix is unique across all devices in the tenant
- Registration requires internet (cannot register offline)

---

### Task 3.2: Outbox — record operations locally

**What:** When a record is created/updated/deleted in offline mode, record the operation in `_off_outbox`.

**Files:**
- Create: `packages/tauri/src-tauri/src/outbox.rs` — outbox management
- Create: `packages/components/src/core/offline-store.ts` — TypeScript wrapper for offline CRUD

**Detail:**

`offline-store.ts` wraps all CRUD operations:
```typescript
export class OfflineStore {
  async create(table: string, data: Record<string, any>): Promise<string> {
    const id = generateUUIDv7();
    const now = new Date().toISOString();
    const hlc = this.clock.now();
    
    // 1. Insert record into local SQLite
    await BcNative.dbExecute(
      `INSERT INTO ${table} (id, ..., _off_device_id, _off_status, _off_version, 
       _off_deleted, _off_created_at, _off_updated_at, _off_hlc) 
       VALUES (?, ..., ?, 0, 1, 0, ?, ?, ?)`,
      [id, ..., this.deviceId, now, now, hlc]
    );
    
    // 2. Record operation in outbox
    await BcNative.dbExecute(
      `INSERT INTO _off_outbox (envelope_id, table_name, record_id, operation, 
       payload, status, idempotency_key, created_at) 
       VALUES (?, ?, ?, 'CREATE', ?, 'PENDING', ?, ?)`,
      [null, table, id, JSON.stringify(data), `${id}:C:1`, now]
    );
    
    return id;
  }
  
  async update(table: string, id: string, changes: Record<string, any>): Promise<void> {
    // Similar: update record + record UPDATE operation in outbox
    // Only changed fields in payload (delta, not full record)
  }
  
  async delete(table: string, id: string): Promise<void> {
    // Soft delete: SET _off_deleted = 1 + record DELETE operation in outbox
  }
}
```

**Acceptance criteria:**
- CREATE → record in table + operation in `_off_outbox` with status PENDING
- UPDATE → record updated + operation in outbox with only changed fields
- DELETE → record soft-deleted (`_off_deleted = 1`) + operation in outbox
- `_off_version` increments on every update
- `_off_updated_at` and `_off_hlc` update on every change
- Idempotency key format: `{record_id}:{operation_letter}:{version}`

---

### Task 3.3: Sync push — send outbox to server

**What:** Client groups pending outbox operations into envelopes and sends them to the server.

**Files:**
- Create: `packages/tauri/src-tauri/src/sync_push.rs` — push logic
- Modify: `engine/internal/presentation/api/sync_handler.go` — implement `PushEnvelope`
- Create: `engine/internal/runtime/sync/processor.go` — server-side envelope processing

**Detail:**

Client push flow:
1. Query `_off_outbox` for PENDING operations
2. Group by `envelope_id` (operations that belong to same transaction)
3. For operations without `envelope_id`, auto-group by table dependency order
4. For each envelope: POST to `/api/v1/sync/push`
5. On success: mark outbox entries as SYNCED
6. On failure: increment `retry_count`, keep as PENDING (or ERROR after 5 retries)

Server `POST /api/v1/sync/push` logic:
1. Check `_sync_log` for `envelope_id` → if exists, return cached response (idempotent)
2. Begin database transaction
3. For each operation in envelope:
   - CREATE → INSERT into server table
   - UPDATE → UPDATE only changed fields
   - DELETE → soft delete or hard delete based on config
4. Record in `_sync_versions` (one entry per operation with incrementing version)
5. Commit transaction
6. Record in `_sync_log` with status and response
7. Return response to client

**Acceptance criteria:**
- Client sends envelope → server processes atomically (all or nothing)
- Same envelope sent twice → server returns cached response (idempotent)
- Network error mid-sync → client retries from where it left off
- After 5 failed retries → operation marked as DEAD
- `_sync_versions` has entries for every synced operation
- Server response includes any assigned server IDs (for auto_increment PK)

---

### Task 3.4: Sync pull — get server changes

**What:** Client pulls changes from server that happened since last sync.

**Files:**
- Modify: `engine/internal/presentation/api/sync_handler.go` — implement `PullChanges`
- Create: `packages/tauri/src-tauri/src/sync_pull.rs` — pull logic

**Detail:**

Client pull flow:
1. Read `last_pull_version` from `_off_sync_state`
2. GET `/api/v1/sync/pull?since_version={last_pull_version}&device_id={device_id}`
3. Server returns all `_sync_versions` entries with `version > since_version` AND `changed_by != device_id`
4. For each change: apply to local SQLite
5. Update `_off_sync_state.last_pull_version` to max version received

Server `GET /api/v1/sync/pull` logic:
```sql
SELECT sv.*, t.*
FROM _sync_versions sv
JOIN {table_name} t ON t.id = sv.record_id
WHERE sv.version > $since_version
  AND sv.changed_by != $device_id
ORDER BY sv.version ASC
LIMIT 1000
```

**Acceptance criteria:**
- Client pulls only changes since last sync (delta, not full dump)
- Client does NOT receive its own changes back (filtered by `changed_by`)
- Changes applied to local SQLite correctly
- `last_pull_version` updated after successful pull
- Pagination works for large change sets (LIMIT + cursor)

---

### Task 3.5: Envelope grouping for transactions

**What:** When a user creates a POS sale (header + items + payment), all operations must be grouped into one envelope for atomic sync.

**Files:**
- Modify: `packages/components/src/core/offline-store.ts` — add transaction grouping
- Test: `packages/components/src/core/offline-store.spec.ts`

**Detail:**

```typescript
export class OfflineStore {
  // Start a transaction group — all operations until commit share one envelope_id
  beginTransaction(): string {
    const envelopeId = generateUUIDv7();
    this.currentEnvelopeId = envelopeId;
    return envelopeId;
  }
  
  // End transaction group
  commitTransaction(): void {
    this.currentEnvelopeId = null;
  }
  
  // Usage:
  // const env = store.beginTransaction();
  // await store.create('sales', saleData);        // envelope_id = env
  // await store.create('sale_items', item1Data);   // envelope_id = env
  // await store.create('sale_items', item2Data);   // envelope_id = env
  // await store.create('payments', paymentData);   // envelope_id = env
  // store.commitTransaction();
}
```

**Acceptance criteria:**
- Operations within a transaction share the same `envelope_id`
- Server processes all operations in one DB transaction
- If any operation fails, entire envelope is rejected
- Operations outside a transaction get individual envelope IDs

---

### Phase 3 Checkpoint

After Phase 3, you can verify:
1. Device registers on first launch
2. Creating a record offline → appears in `_off_outbox` as PENDING
3. Triggering sync → outbox operations sent to server
4. Server processes envelope atomically
5. Sending same envelope twice → idempotent (no duplicates)
6. Changes made on server (or another device) → pulled to local SQLite
7. POS sale (header + items + payment) syncs as one atomic transaction

**This is the first phase where offline actually WORKS end-to-end.** You can create data offline, go online, and see it sync to the server.

---

## Phase 4: Conflict Resolution & Edge Cases (1-2 weeks) ✅ COMPLETE

**Status:** All 4 tasks implemented and tested.

**Goal:** Handle all edge cases from the design doc: field-level merge, HLC, receipt numbering, inventory overselling.

### Task 4.1: Hybrid Logical Clock (HLC) implementation

**What:** Implement HLC for reliable event ordering even with clock skew.

**Files:**
- Create: `packages/components/src/core/hlc.ts` — TypeScript HLC
- Create: `packages/tauri/src-tauri/src/hlc.rs` — Rust HLC (for server-side comparison)
- Test: `packages/components/src/core/hlc.spec.ts`

**Detail:**

Full HLC implementation as documented in design doc. Three operations:
- `now()` — generate timestamp for local event
- `receive(remote)` — update clock when receiving remote event
- `compare(a, b)` — deterministic comparison of two HLC values

**Acceptance criteria:**
- HLC values are monotonically increasing on same device
- HLC handles clock skew up to 1 minute
- `compare()` is deterministic — same inputs always produce same ordering
- Tie-breaking uses device_id (lexicographic)

---

### Task 4.2: Field-level conflict merge

**What:** When pulling changes, detect and resolve conflicts at field level.

**Files:**
- Create: `engine/internal/runtime/sync/conflict.go` — server-side conflict resolution
- Modify: `packages/tauri/src-tauri/src/sync_pull.rs` — client-side conflict detection
- Test: `engine/internal/runtime/sync/conflict_test.go`

**Detail:**

Conflict detection during pull:
```
For each incoming change:
  1. Check if local record was also modified since last sync (_off_version > 1 AND _off_status = 0)
  2. If no local changes → apply remote change directly (no conflict)
  3. If local changes exist → compare field by field:
     a. Field changed only remotely → accept remote value
     b. Field changed only locally → keep local value
     c. Field changed on BOTH sides:
        - Compare HLC → newer wins
        - Log conflict in _off_conflict_log
        - If business-critical field → flag for user review
```

**Acceptance criteria:**
- Different fields edited → auto-merge, both changes preserved
- Same field edited → HLC determines winner, conflict logged
- Edit vs Delete → edit wins, deleted record resurrected
- Conflict log has complete audit trail
- Server `_sync_conflicts` table populated for admin review

---

### Task 4.3: Device-prefixed receipt numbering

**What:** Implement offline-safe sequential numbering for receipts/invoices.

**Files:**
- Modify: `packages/components/src/core/offline-store.ts` — add number generation
- Modify: `engine/internal/runtime/format/engine.go` — add offline number format support

**Detail:**

Client-side number generation:
```typescript
async getNextReceiptNumber(tableName: string): Promise<string> {
  const state = await BcNative.dbExecute(
    `SELECT prefix, last_sequence FROM _off_number_sequence WHERE table_name = ?`,
    [tableName]
  );
  
  const nextSeq = (state?.last_sequence || 0) + 1;
  const prefix = state?.prefix || this.devicePrefix;
  
  await BcNative.dbExecute(
    `INSERT OR REPLACE INTO _off_number_sequence (table_name, prefix, last_sequence) VALUES (?, ?, ?)`,
    [tableName, prefix, nextSeq]
  );
  
  return `${prefix}-${String(nextSeq).padStart(4, '0')}`;
  // Example: "001-A-0016"
}
```

**Acceptance criteria:**
- Numbers are sequential per device (no gaps within a device)
- Different devices have different prefixes (no collision)
- Numbers survive app restart (persisted in `_off_number_sequence`)
- Format: `{store_code}-{device_letter}-{zero_padded_sequence}`

---

### Task 4.4: Inventory delta-based tracking

**What:** Implement delta-based inventory to handle overselling gracefully.

**Files:**
- Create: `engine/internal/runtime/sync/inventory.go` — inventory reconciliation
- Test: `engine/internal/runtime/sync/inventory_test.go`

**Detail:**

During sync push, if operation involves inventory:
1. Server applies delta (not absolute value)
2. Server checks resulting stock
3. If negative → log alert, notify manager, but accept the sale
4. Manager can review oversell alerts in admin panel

**Acceptance criteria:**
- Inventory changes sync as deltas (`qty_delta: -5`), not absolutes (`qty: 3`)
- Negative stock is allowed (not blocked)
- Oversell alert created when stock goes negative
- Admin can see oversell alerts with details (which device, which sale, how much short)

---

### Phase 4 Checkpoint

After Phase 4, you can verify:
1. Two devices edit same record offline → changes merge correctly
2. Same field edited on 2 devices → HLC determines winner, conflict logged
3. Receipt numbers are unique across devices
4. Inventory overselling is handled gracefully (sale accepted, alert created)
5. Conflict log shows complete audit trail

---

## Phase 5: Polish & Cross-Platform Testing (1-2 weeks)

**Goal:** Production-ready on all 5 platforms. Encryption, performance, App Store compliance.

### Task 5.1: SQLite encryption at rest

**What:** Enable encryption for local SQLite database.

**Files:**
- Modify: `packages/tauri/src-tauri/Cargo.toml` — add encryption feature
- Modify: `packages/tauri/src-tauri/src/main.rs` — configure encrypted database

**Acceptance criteria:**
- SQLite database file is encrypted on disk
- App can read/write encrypted database transparently
- Database is unreadable without the encryption key

---

### Task 5.2: Offline authentication

**What:** Implement cached auth for offline use.

**Files:**
- Modify: `engine/internal/presentation/api/sync_handler.go` — implement `CacheAuth`
- Create: `packages/tauri/src-tauri/src/auth.rs` — offline auth logic

**Acceptance criteria:**
- User can authenticate offline using cached credentials
- After `max_offline_hours` (default 72), force re-login
- Cached credentials are encrypted
- Failed offline auth attempts are limited (prevent brute force)

---

### Task 5.3: Sync status UI indicator

**What:** Add a sync status indicator to the Stencil UI showing online/offline status and pending sync count.

**Files:**
- Create: `packages/components/src/components/widgets/bc-sync-status/` — new Stencil component
- Modify: `packages/components/src/core/bc-native.ts` — add connectivity detection

**Acceptance criteria:**
- Shows 🟢 Online / 🔴 Offline status
- Shows pending sync count (e.g., "3 pending sync")
- Shows last sync time (e.g., "Last sync: 2 min ago")
- Manual sync trigger button
- Conflict indicator when unresolved conflicts exist

---

### Task 5.4: Cross-platform testing

**What:** Test on all 5 target platforms.

| Platform | Test Focus |
|----------|-----------|
| Windows | WebView2 rendering, SQLite path, file permissions |
| macOS | WKWebView rendering, app sandbox, notarization |
| Linux | WebKitGTK rendering, various distros |
| Android | System WebView, camera/GPS plugins, background sync, Play Store |
| iOS | WKWebView, App Store Guideline 4.2, camera/GPS permissions, TestFlight |

**Acceptance criteria:**
- App launches and renders correctly on all 5 platforms
- Offline CRUD works on all platforms
- Sync works on all platforms
- Native capabilities (camera, GPS) work on mobile
- App Store / Play Store submission requirements met

---

### Task 5.5: Performance testing

**What:** Verify sync performance with realistic data volumes.

| Scenario | Target |
|----------|--------|
| Initial sync: 10,000 products | < 30 seconds on 4G |
| Delta sync: 100 changes | < 3 seconds on 4G |
| Offline CRUD: 1,000 operations | < 1 second per operation |
| Outbox with 500 pending operations | Sync completes in < 60 seconds |
| SQLite with 100,000 records | Query response < 100ms |

**Acceptance criteria:**
- All performance targets met
- No memory leaks during extended offline use
- App remains responsive during background sync

---

### Phase 5 Checkpoint (Final)

After Phase 5, the offline mode is **production-ready:**
1. ✅ Works on all 5 platforms (Windows, macOS, Linux, iOS, Android)
2. ✅ Data encrypted at rest
3. ✅ Offline auth with expiry
4. ✅ Sync status visible to user
5. ✅ Performance targets met
6. ✅ App Store / Play Store compliant

---

## Honest Risks & Unknowns

| Risk | Severity | Mitigation |
|------|----------|-----------|
| **Tauri mobile plugins immature** | 🔴 High | Camera/GPS may need custom Rust commands. Budget extra 1-2 weeks for mobile-specific issues |
| **iOS App Store review** | ⚠️ Medium | Ensure 3+ native features (offline, camera, GPS, push). Prepare appeal documentation |
| **Sync conflict edge cases** | ⚠️ Medium | Start with simple LWW, add field-level merge incrementally. Don't over-engineer conflict resolution before real usage data |
| **SQLite performance at scale** | ⚠️ Low | SQLite handles millions of rows well. Index `_off_status` and `_off_updated_at` columns |
| **Clock skew in production** | ⚠️ Low | HLC handles this, but test with deliberately skewed clocks |
| **Multi-tenant data isolation** | ⚠️ Medium | Separate SQLite files per tenant. Test tenant switching thoroughly |

## Dependencies

| Dependency | Version | Purpose | Risk |
|-----------|---------|---------|------|
| Tauri | 2.x | Native shell | Stable for desktop, maturing for mobile |
| tauri-plugin-sql | 2.x | SQLite access | Stable |
| tauri-plugin-fs | 2.x | File system | Stable |
| tauri-plugin-notification | 2.x | Push notifications | Stable |
| tauri-plugin-barcode-scanner | 2.x | Barcode scanning | Community plugin, less tested |
| tauri-plugin-biometric | 2.x | Fingerprint/Face ID | Community plugin, less tested |
| uuid (npm) | 9.x | UUIDv7 generation | Stable |
