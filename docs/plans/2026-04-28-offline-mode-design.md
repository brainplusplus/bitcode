# Offline Mode — Design Document

> **Date:** 2026-04-28
> **Status:** Implementing — Phase 1 ✅, Phase 2 ✅, Phase 2.5 ✅, Phase 3 (Sync Engine) next
> **Approach:** D+ (Stencil on Tauri) — Stencil components unchanged, Tauri as native shell
> **Principle:** Convention over Configuration — one toggle enables everything

---

## Table of Contents

1. [Background & Motivation](#1-background--motivation)
2. [Decision Journey](#2-decision-journey)
3. [Architecture Overview](#3-architecture-overview)
4. [Bridge Abstraction Layer](#4-bridge-abstraction-layer)
5. [Offline-First Data Architecture](#5-offline-first-data-architecture)
6. [Primary Key Strategy](#6-primary-key-strategy)
7. [Sync Engine](#7-sync-engine)
8. [Edge Cases & Solutions](#8-edge-cases--solutions)
9. [Developer Experience](#9-developer-experience)
10. [Flutter Feasibility Report](#10-flutter-feasibility-report)
11. [Effort Estimates](#11-effort-estimates)

---

## 1. Background & Motivation

### Problem Statement

BitCode has 103 production-ready Stencil web components. However, several use cases require native capabilities that browsers cannot fully provide:

1. **Native performance** — Stencil in mobile WebView (via Capacitor) is not smooth enough for heavy UI
2. **Single codebase for mobile + desktop** — no current native solution
3. **Developer preference** — some teams prefer native app development
4. **App Store distribution** — need native apps, not just browser
5. **Native capabilities** — offline-first, camera (audit/inspection), location detection (attendance)

### Specific Use Cases

| Use Case | Native Capabilities Needed |
|----------|--------------------------|
| **POS (Point of Sale)** | Offline-first, camera (barcode scan), receipt printing |
| **Audit/Inspection** | Camera (photo evidence), GPS (location), offline data collection |
| **Attendance (Absensi)** | GPS (check-in location), biometrics, offline capability |
| **Field Work** | Offline-first, camera, GPS, background sync |

---

## 2. Decision Journey

### Approaches Evaluated

| Approach | Concept | Effort | Risk | Verdict |
|----------|---------|--------|------|---------|
| **A: Stencil + Capacitor** | Keep 100% Stencil, wrap with Capacitor | 2-4 wk | Offline unreliable | Quick but doesn't solve core problems |
| **B: Cascading Renderer** | Stencil default + Flutter components for native-heavy | 16-22 wk | Medium | Good balance but high effort |
| **C: Flutter-First** | Flutter replaces Stencil for all platforms | 24-32 wk | High | Too risky, wastes 103 components |
| **D: Flutter Shell** | Flutter as native shell, Stencil via WebView | 5-8 wk | Low (desktop weak) | Good but desktop WebView weak |
| **D+: Tauri Shell ⭐** | Tauri as native shell for ALL platforms | 6-10 wk | Low | **CHOSEN** — best balance |

### Why Tauri Won Over Flutter Shell

| Criteria | Flutter Shell | Tauri Shell ⭐ |
|----------|-------------|---------------|
| WebView maturity | ⚠️ Less mature on desktop | ✅ Native OS WebView — excellent everywhere |
| App size | ~15-20MB | ~600KB-2MB |
| Desktop support | ⚠️ Weak | ✅ Tauri's primary strength |
| Language | Dart (new skill) | TypeScript + Rust (web stack) |
| JS Bridge | Custom via JavascriptChannel | Built-in `invoke()` IPC |
| Local web server needed? | Yes (shelf) | No — Tauri natively serves web assets |
| Stencil integration | Via localhost + WebView | Direct — Tauri IS a WebView framework |

### Why Tauri Won Over Capacitor

| Capability | Capacitor | Tauri |
|-----------|-----------|-------|
| Local SQLite reliability | ⚠️ Plugin history rocky, browser can clear storage | ✅ tauri-plugin-sql, native filesystem |
| Background sync | ❌ WebView throttled after 5 min on Android | ✅ Rust backend, no throttling |
| Data persistence | ⚠️ Browser can clear WebView storage on low space | ✅ Native filesystem, persistent |
| Desktop | ❌ Not supported | ✅ Excellent (Windows, macOS, Linux) |
| App size | ~20MB | ~600KB-2MB |

---

## 3. Architecture Overview

```
                ┌────────────────────────────────┐
                │        BitCode Application     │
                └──────────┬─────────────────────┘
                           │
              ┌────────────┼────────────────┐
              │            │                │
        ┌─────▼──────┐ ┌──▼───────┐ ┌──────▼───────┐
        │  Browser   │ │  Tauri   │ │    Tauri     │
        │  (Web)     │ │  Desktop │ │    Mobile    │
        │            │ │ Win/Mac/ │ │  iOS/Android │
        │            │ │  Linux   │ │              │
        └─────┬──────┘ └────┬─────┘ └──────┬───────┘
              │              │              │
              └──────────────┼──────────────┘
                             │
                  ┌──────────▼──────────────┐
                  │   Stencil Components    │
                  │   (103 components)      │
                  │   UNCHANGED             │
                  └──────────┬──────────────┘
                             │
                  ┌──────────▼──────────────┐
                  │   bc-native.ts          │
                  │   (Bridge Abstraction)  │
                  │                         │
                  │   if (inTauri)           │
                  │     → Tauri invoke()    │
                  │   else                  │
                  │     → Web APIs / noop   │
                  └──────────┬──────────────┘
                             │
              ┌──────────────┼──────────────┐
              │                             │
        ┌─────▼──────────┐      ┌───────────▼───────┐
        │  Web APIs      │      │  Tauri Backend    │
        │  (browser)     │      │  (Rust)           │
        │  • navigator.* │      │  • SQLite         │
        │  • fetch()     │      │  • Camera         │
        │  • IndexedDB   │      │  • GPS            │
        │                │      │  • Filesystem     │
        └────────────────┘      │  • Background     │
                                │    sync           │
                                │  • Biometrics     │
                                │  • Push notif     │
                                └───────────────────┘
```

### Key Principles

1. **Stencil components do NOT change** — zero modification to 103 components
2. **`bc-native.ts`** — single abstraction layer that detects environment and routes to Tauri IPC or Web APIs
3. **Tauri = transparent shell** — user doesn't know if they're in browser or Tauri app
4. **Standalone capable** — Stencil + Tauri works with or without BitCode Engine
5. **Convention over configuration** — one toggle enables everything

### Tauri Platform Support

| Platform | Status | WebView Engine |
|----------|--------|---------------|
| Windows | ✅ Stable | WebView2 (Chromium) |
| macOS | ✅ Stable | WKWebView |
| Linux | ✅ Stable | WebKitGTK |
| iOS | ✅ Supported | WKWebView |
| Android | ✅ Supported | Android System WebView |

### iOS App Store Compliance (Guideline 4.2)

BitCode Tauri app will NOT be rejected because it provides genuine native value beyond a WebView wrapper:

1. ✅ Offline-first — works without internet
2. ✅ Native camera — photo audit, barcode scan
3. ✅ Native GPS — attendance, location tracking
4. ✅ Push notifications — native APNs
5. ✅ Biometric auth — Face ID / Touch ID
6. ✅ Background sync — data sync when app is backgrounded
7. ✅ Local data persistence — native filesystem, not browser storage

---

## 4. Bridge Abstraction Layer

### `bc-native.ts` — Universal Native API

A single file in `packages/components/src/core/bc-native.ts` that provides a unified API for native capabilities. Components call `BcNative`, it routes to Tauri IPC or Web API fallback automatically.

```typescript
// packages/components/src/core/bc-native.ts

// Detect if running inside Tauri shell
const isTauri = () => '__TAURI__' in window;

export const BcNative = {

  // ===== CAMERA =====
  // Take a photo using device camera
  // Returns: base64 encoded image string
  // Browser fallback: <input type="file" capture="environment">
  async takePhoto(options?: { quality?: number }): Promise<string> {
    if (isTauri()) {
      return invoke('plugin:camera|take_photo', options);
    }
    return webCameraFallback(options);
  },

  // ===== GPS / GEOLOCATION =====
  // Get current device location
  // Returns: { lat: number, lng: number }
  // Browser fallback: navigator.geolocation.getCurrentPosition()
  async getLocation(): Promise<{ lat: number; lng: number }> {
    if (isTauri()) {
      return invoke('plugin:geolocation|get_position');
    }
    return webGeolocationFallback();
  },

  // ===== OFFLINE DATABASE =====
  // Execute SQL query on local SQLite database
  // Returns: query result rows
  // Browser fallback: IndexedDB or sql.js (WASM SQLite)
  async dbExecute(sql: string, params?: any[]): Promise<any> {
    if (isTauri()) {
      return invoke('plugin:sql|execute', { sql, params });
    }
    return webDbFallback(sql, params);
  },

  // ===== BARCODE / QR SCANNER =====
  // Scan barcode or QR code using device camera
  // Returns: decoded barcode string
  // Browser fallback: JS barcode library via camera stream
  async scanBarcode(): Promise<string> {
    if (isTauri()) {
      return invoke('plugin:barcode-scanner|scan');
    }
    return webBarcodeFallback();
  },

  // ===== BIOMETRIC AUTHENTICATION =====
  // Authenticate user via fingerprint or face recognition
  // Returns: true if authenticated, false if failed/cancelled
  // Browser fallback: WebAuthn API or password prompt
  async authenticate(): Promise<boolean> {
    if (isTauri()) {
      return invoke('plugin:biometric|authenticate');
    }
    return webAuthFallback();
  },

  // ===== FILE SYSTEM =====
  // Save file to device filesystem
  // Browser fallback: download via blob URL
  async saveFile(path: string, data: Uint8Array): Promise<void> {
    if (isTauri()) {
      return invoke('plugin:fs|write_file', { path, data });
    }
    return webFileSaveFallback(path, data);
  },

  // ===== PUSH NOTIFICATIONS =====
  // Request permission to send push notifications
  // Returns: true if permission granted
  // Browser fallback: Notification.requestPermission()
  async requestNotificationPermission(): Promise<boolean> {
    if (isTauri()) {
      return invoke('plugin:notification|request_permission');
    }
    return Notification.requestPermission() === 'granted';
  },

  // ===== DATA SYNC =====
  // Trigger sync of local data to server
  // Returns: sync result with counts of synced/failed/conflicted records
  // Browser fallback: standard fetch() to API
  async syncData(endpoint: string): Promise<SyncResult> {
    if (isTauri()) {
      return invoke('cmd_sync_data', { endpoint });
    }
    return webSyncFallback(endpoint);
  },

  // ===== ENVIRONMENT DETECTION =====
  // Detect which environment the app is running in
  // Returns: 'browser' | 'tauri-desktop' | 'tauri-mobile'
  getEnvironment(): 'browser' | 'tauri-desktop' | 'tauri-mobile' {
    if (!isTauri()) return 'browser';
    return window.__TAURI__.os.type === 'ios' ||
           window.__TAURI__.os.type === 'android'
      ? 'tauri-mobile' : 'tauri-desktop';
  }
};
```

### How Components Use It

Components that need native capabilities import `BcNative`. The change is minimal and opt-in:

```typescript
// Example: bc-field-barcode.tsx — adding scan capability
import { BcNative } from '../../core/bc-native';

// Inside component method:
async handleScan() {
  const code = await BcNative.scanBarcode();  // Works in Tauri AND browser
  this.value = code;
}
```

### Key Properties

- **Graceful degradation** — in browser, falls back to Web API or no-op. Never throws errors
- **Transparent** — components don't need to know whether they're in Tauri or browser
- **Opt-in** — only components that need native features import `bc-native`
- **Centralized** — all native capabilities in one file, easy to extend

---

## 5. Offline-First Data Architecture

### Configuration: One Toggle

**Developer-facing config — just ONE line:**

```toml
# bitcode.toml
[app]
mode = "online"          # online | offline
```

That's it. Setting `mode = "offline"` automatically enables everything:

| What Engine Does Automatically | Detail |
|-------------------------------|--------|
| Switch PK generation to client-side | UUIDv7 generated on device, no server round-trip needed |
| Create local SQLite database | One database per module, encrypted at rest |
| Enable sync engine | Operation-based outbox pattern with idempotency |
| Enable field-level conflict merge | Auto-merge different fields, Last-Write-Wins for same field |
| Setup offline auth cache | Cached credentials + offline PIN, max 72 hours |
| Assign device receipt prefix | Device-prefixed sequential numbering for receipts |
| Enable encryption at rest | SQLite encryption via tauri-plugin-sql |
| Add sync status UI indicator | Shows online/offline status + pending sync count |
| Generate `_off_*` columns | Offline tracking columns added to every table (see Section 6) |
| Create `_off_*` tables | Outbox, sync state, conflict log, number sequence tables |

### Cascading Override

Config can be overridden at module or view level:

```toml
# bitcode.toml — project level
[app]
mode = "online"          # default: all modules online
```

```json
// modules/pos/module.json — module level override
{
  "app": {
    "mode": "offline"    // POS module = offline, overrides project setting
  }
}
```

```json
// modules/crm/views/contact_form.json — view level override
{
  "app": {
    "mode": "online"     // this specific view = online, overrides module setting
  }
}
```

**Resolution chain:** `view.app.mode → module.app.mode → project.app.mode → default("online")`

### Advanced Config (Optional)

99% of users never need these. All have sensible defaults:

```toml
# bitcode.toml — advanced (all optional)
[app]
mode = "offline"

[app.offline]
max_offline_hours = 72               # default: 72. Force re-auth after this many hours offline
sync_batch_size = 100                # default: 100. Operations per sync batch
inventory_oversell = "allow"         # default: "allow". Options: "allow" | "block" | "warn"
conflict_on_same_field = "latest"    # default: "latest". Options: "latest" | "ask_user" | "server_wins"
```

### Data Flow Comparison

```
ONLINE MODE:
  User action → Stencil component → fetch() → Go Engine API → PostgreSQL
  
  Simple. Direct. No local storage.

OFFLINE MODE:
  User action → Stencil component → BcNative.dbExecute() → Local SQLite
                                                              │
                                                    (background, when online)
                                                              │
                                                    Sync engine push/pull
                                                    to Go Engine API → PostgreSQL
```

---

## 6. Primary Key Strategy

### Supported PK Types for Offline Mode

BitCode supports 6 PK strategies (see `engine/docs/features/primary-keys.md`). For offline mode, 3 are fully supported:

| PK Strategy | Offline Support | Auto-Generated Columns |
|-------------|----------------|----------------------|
| `uuid` (v4, v7, format) | ✅ Full — recommended | `_off_*` tracking columns only |
| `auto_increment` | ⚠️ With warning — needs `_off_uuid` | `_off_uuid` + `_off_*` tracking columns |
| `natural_key` | ⚠️ With warning — needs `_off_uuid` | `_off_uuid` + `_off_*` tracking columns |
| `naming_series` | ⚠️ With warning — needs `_off_uuid` | `_off_uuid` + `_off_*` tracking columns |
| `composite` | ❌ Not supported offline | Engine rejects with error |
| `manual` | ⚠️ With warning — collision risk | `_off_uuid` + `_off_*` tracking columns |

### How Each PK Type Works Offline

#### `pk: "uuid"` — Recommended, Simplest

UUID is generated on the client device. The same UUID becomes the permanent PK on the server. No remapping needed.

```
Client generates: id = "019abc12-3def-7000-8000-abcdef123456" (UUIDv7)
                       ↓
Stored locally:   INSERT INTO sales (id, ...) VALUES ("019abc12-...", ...)
                       ↓
Synced to server: Server accepts UUID as-is → INSERT INTO sales (id, ...) VALUES ("019abc12-...", ...)
                       ↓
Done. No ID remapping. No collision. No extra columns needed.
```

#### `pk: "auto_increment"` — Needs Extra Column

Auto-increment IDs are assigned by the server database. Client cannot generate valid auto-increment IDs offline. Engine adds `_off_uuid` as temporary identity.

```
Client creates:   _off_uuid = "019abc12-..." (temporary identity)
                  id = NULL (server will assign)
                       ↓
Stored locally:   INSERT INTO sales (_off_uuid, id, ...) VALUES ("019abc12-...", NULL, ...)
                  FK references use _off_uuid, not id
                       ↓
Synced to server: Server assigns id = 5432
                  Server returns mapping: { _off_uuid: "019abc12-...", id: 5432 }
                       ↓
Client updates:   UPDATE sales SET id = 5432 WHERE _off_uuid = "019abc12-..."
                  UPDATE sale_items SET sale_id = 5432 WHERE sale__off_uuid = "019abc12-..."
                       ↓
More complex. Engine shows warning:
  ⚠️ WARNING: Module "pos" uses pk:"auto_increment" with mode:"offline".
     Auto-increment PK requires additional sync complexity (ID remapping).
     Consider using pk:"uuid" for better offline support.
```

#### `pk: "natural_key"` — Collision Risk

Natural keys (e.g., `code = "kg"`) can collide if two devices create records with the same value offline.

```
Device A creates: code = "box", name = "Box"
Device B creates: code = "box", name = "Kardus"
                       ↓
Both sync → COLLISION on natural key "box"
                       ↓
Server detects conflict → returns error to later device
                       ↓
User must resolve: rename to "box-2" or merge with existing
```

Engine adds `_off_uuid` as backup identity for sync tracking.

### All PK Types Compared

| | `uuid` | `auto_increment` | `natural_key` |
|---|---|---|---|
| **PK column** | `id UUID` (client-generated) | `id SERIAL` (server-generated) | `{field} TEXT` (user-defined) |
| **Needs `_off_uuid`?** | ❌ No — `id` is already UUID | ✅ Yes — temporary identity while offline | ✅ Yes — backup identity for sync |
| **FK references offline** | Use `id` directly | Use `_off_uuid` while offline, remap to `id` after sync | Use natural key + `_off_uuid` as fallback |
| **Sync complexity** | ⭐ Simple | ⭐⭐⭐ Complex (ID remapping for all FKs) | ⭐⭐ Medium (collision handling) |
| **Collision risk** | ❌ None | ❌ None (server assigns) | ⚠️ Possible (same natural key on 2 devices) |
| **Recommended for offline?** | ✅ Yes — default | ⚠️ With warning | ⚠️ Master data only |

### Auto-Generated Columns (`_off_*` prefix)

When `mode = "offline"`, the engine automatically adds tracking columns to every table. All columns use the `_off_` prefix to clearly distinguish them from business fields.

**Example: What the engine generates for a `sales` table**

Developer defines this model:

```json
{
  "name": "Sale",
  "primary_key": { "strategy": "uuid", "version": "v7" },
  "fields": [
    { "name": "customer_id", "type": "link", "ref": "Customer" },
    { "name": "sale_date", "type": "datetime", "default": "now" },
    { "name": "total", "type": "currency" },
    { "name": "payment_method", "type": "select", "options": ["cash", "card", "qris"] },
    { "name": "status", "type": "select", "options": ["draft", "completed", "cancelled"] }
  ]
}
```

Engine generates this SQLite schema:

```sql
CREATE TABLE sales (
  -- ================================================================
  -- BUSINESS FIELDS (from model definition — developer defines these)
  -- ================================================================
  id              TEXT PRIMARY KEY,      -- UUIDv7, generated on client device
  customer_id     TEXT,                  -- FK to customers.id (also UUID)
  sale_date       TEXT NOT NULL,         -- ISO 8601 datetime
  total           REAL DEFAULT 0,        -- currency amount
  payment_method  TEXT,                  -- "cash" | "card" | "qris"
  status          TEXT DEFAULT 'draft',  -- "draft" | "completed" | "cancelled"

  -- ================================================================
  -- OFFLINE ENGINE FIELDS (auto-generated, developer does NOT define these)
  -- All prefixed with _off_ to distinguish from business fields
  -- ================================================================

  _off_device_id    TEXT NOT NULL,
    -- ID of the device that created or last modified this record.
    -- Assigned during device registration (e.g., "DEV-A", "DEV-B").
    -- Used for: receipt number prefix, conflict attribution, audit trail.
    -- Example: "DEV-A"

  _off_status       INTEGER NOT NULL DEFAULT 0,
    -- Sync status of this record.
    -- 0 = PENDING  → record created/modified locally, not yet synced to server
    -- 1 = SYNCED   → record successfully synced to server
    -- 2 = CONFLICT → sync detected a conflict, needs review
    -- UI uses this to show: ⏳ (pending), ✅ (synced), ⚠️ (conflict)

  _off_version      INTEGER NOT NULL DEFAULT 1,
    -- Version counter. Increments by 1 every time the record is edited.
    -- Used for optimistic concurrency control during sync:
    --   If client version < server version → someone else edited while you were offline
    --   Server compares versions to detect conflicts.
    -- Example: created=1, first edit=2, second edit=3

  _off_deleted      INTEGER NOT NULL DEFAULT 0,
    -- Soft delete flag.
    -- 0 = active (normal record)
    -- 1 = deleted (marked for deletion, hidden from UI)
    -- Records are NEVER hard-deleted from local SQLite.
    -- Why? So the sync engine knows "this record was deleted" and can
    -- propagate the deletion to the server. Hard delete = sync engine
    -- doesn't know the record ever existed.

  _off_created_at   TEXT NOT NULL,
    -- ISO 8601 timestamp when the record was first created.
    -- Example: "2026-04-28T10:30:00.000Z"
    -- Used for: sorting, audit trail, tiered sync (sync recent data first),
    -- initial sync prioritization (download recent records before old ones).

  _off_updated_at   TEXT NOT NULL,
    -- ISO 8601 timestamp when the record was last modified.
    -- Updated every time any business field changes.
    -- Example: "2026-04-28T14:15:30.000Z"
    -- Used for: delta sync (only sync records changed since last sync),
    -- conflict detection (compare with server's updated_at).

  _off_hlc          TEXT NOT NULL,
    -- Hybrid Logical Clock timestamp.
    -- Format: "{wall_time_base36}:{logical_counter_base36}:{device_id}"
    -- Example: "01jk5p9q:0001:DEV-A"
    -- More reliable than plain timestamps because it handles clock skew
    -- (device clock set incorrectly or drifting).
    -- Combines physical time + logical counter + device ID for
    -- deterministic ordering even when clocks disagree.
    -- Used for: conflict resolution — determines "who edited last" accurately.

  _off_envelope_id  TEXT
    -- Sync batch ID. Groups related operations into one atomic envelope.
    -- NULL when record is first created (not yet batched for sync).
    -- Set by sync engine when it groups operations for sending.
    -- Example: "019def45-6789-7000-8000-abcdef654321"
    -- Used for: atomic sync (all records in one envelope = one server
    -- transaction — all succeed or all fail), retry tracking,
    -- idempotency (if sync response is lost, resend same envelope,
    -- server recognizes it and returns cached response).
);
```

**Visualisasi: Contoh data di SQLite**

```
┌─────────────────────────────────────────────────────────────────────────────────────────────────────────────┐
│                                          sales (SQLite)                                                    │
├──────────────────┬─────────────┬────────────┬────────┬─────────┬────────┬──────────┬──────────┬───────────┤
│ id               │ customer_id │ sale_date  │ total  │ payment │ status │_off_     │_off_     │_off_      │
│ (PK, UUIDv7)     │             │            │        │ _method │        │device_id │status    │version    │
├──────────────────┼─────────────┼────────────┼────────┼─────────┼────────┼──────────┼──────────┼───────────┤
│ 019abc12-3def... │ 019aaa11... │ 2026-04-28 │ 150000 │ cash    │ done   │ DEV-A    │ 1 (sync) │ 1         │
│ 019abc13-4ef0... │ NULL        │ 2026-04-28 │  85000 │ qris    │ done   │ DEV-A    │ 1 (sync) │ 2         │
│ 019abc14-5f01... │ 019aaa22... │ 2026-04-28 │ 220000 │ card    │ done   │ DEV-A    │ 0 (pend) │ 1         │
│ 019abc15-6012... │ NULL        │ 2026-04-28 │  45000 │ cash    │ draft  │ DEV-A    │ 0 (pend) │ 1         │
└──────────────────┴─────────────┴────────────┴────────┴─────────┴────────┴──────────┴──────────┴───────────┘

Continued columns:
┌───────────┬──────────────────────┬──────────────────────┬──────────────────┬──────────────────┐
│_off_      │_off_created_at       │_off_updated_at       │_off_hlc          │_off_envelope_id  │
│deleted    │                      │                      │                  │                  │
├───────────┼──────────────────────┼──────────────────────┼──────────────────┼──────────────────┤
│ 0         │ 2026-04-28T08:00:00Z │ 2026-04-28T08:00:00Z │ 01jk5p:0000:DA  │ 019env01-...     │
│ 0         │ 2026-04-28T09:15:00Z │ 2026-04-28T09:20:00Z │ 01jk5q:0001:DA  │ 019env01-...     │
│ 0         │ 2026-04-28T10:30:00Z │ 2026-04-28T10:30:00Z │ 01jk5r:0000:DA  │ NULL (not yet)   │
│ 0         │ 2026-04-28T11:00:00Z │ 2026-04-28T11:00:00Z │ 01jk5s:0000:DA  │ NULL (not yet)   │
└───────────┴──────────────────────┴──────────────────────┴──────────────────┴──────────────────┘
```

### Auto-Generated Tables (`_off_*` prefix)

The engine also creates these infrastructure tables automatically:

#### `_off_outbox` — Sync Queue

Every CREATE, UPDATE, DELETE operation is recorded here before being sent to the server.

```sql
CREATE TABLE _off_outbox (
  id              INTEGER PRIMARY KEY AUTOINCREMENT,
    -- Local auto-increment ID for ordering operations within the outbox.

  envelope_id     TEXT NOT NULL,
    -- Groups related operations into one atomic batch.
    -- Example: a POS sale creates 1 header + 3 items + 1 payment = 5 operations,
    -- all sharing the same envelope_id. Server processes them as one transaction.

  table_name      TEXT NOT NULL,
    -- Which table this operation affects.
    -- Example: "sales", "sale_items", "payments"

  record_id       TEXT NOT NULL,
    -- UUID of the record being created/updated/deleted.
    -- Example: "019abc12-3def-7000-8000-abcdef123456"

  operation       TEXT NOT NULL,
    -- What happened: "CREATE", "UPDATE", or "DELETE"

  payload         TEXT NOT NULL,
    -- JSON-encoded data for the operation.
    -- For CREATE: full record data
    -- For UPDATE: only changed fields (delta), e.g., {"price": 15000, "status": "completed"}
    -- For DELETE: empty {} (record ID is enough)

  status          TEXT NOT NULL DEFAULT 'PENDING',
    -- Lifecycle of this operation:
    -- PENDING    → waiting to be synced
    -- IN_FLIGHT  → currently being sent to server
    -- SYNCED     → server confirmed receipt
    -- ERROR      → sync failed, will retry
    -- DEAD       → failed too many times, needs manual intervention

  idempotency_key TEXT NOT NULL UNIQUE,
    -- Unique key for this specific operation.
    -- If sync succeeds but response is lost (network timeout), client retries
    -- with the same key. Server recognizes it and returns cached response
    -- instead of creating a duplicate record.
    -- Format: "{record_id}:{operation}:{version}"

  created_at      TEXT NOT NULL,
    -- When this operation was recorded in the outbox.

  retry_count     INTEGER NOT NULL DEFAULT 0
    -- How many times sync has been attempted for this operation.
    -- After 5 retries, status changes to DEAD.
);
```

**Visualisasi: Contoh data di `_off_outbox`**

```
┌────┬──────────────┬────────────┬──────────────┬───────────┬──────────────────────────────────┬─────────┬──────────────────┐
│ id │ envelope_id  │ table_name │ record_id    │ operation │ payload                          │ status  │ idempotency_key  │
├────┼──────────────┼────────────┼──────────────┼───────────┼──────────────────────────────────┼─────────┼──────────────────┤
│  1 │ ENV-001      │ sales      │ 019abc14-... │ CREATE    │ {"total":220000,"method":"card"} │ PENDING │ 019abc14:C:1     │
│  2 │ ENV-001      │ sale_items │ 019abc16-... │ CREATE    │ {"product":"Widget","qty":2}     │ PENDING │ 019abc16:C:1     │
│  3 │ ENV-001      │ sale_items │ 019abc17-... │ CREATE    │ {"product":"Gadget","qty":1}     │ PENDING │ 019abc17:C:1     │
│  4 │ ENV-001      │ payments   │ 019abc18-... │ CREATE    │ {"method":"card","amount":220K}  │ PENDING │ 019abc18:C:1     │
│  5 │ ENV-002      │ sales      │ 019abc15-... │ CREATE    │ {"total":45000,"method":"cash"}  │ PENDING │ 019abc15:C:1     │
│  6 │ ENV-003      │ products   │ 019aaa33-... │ UPDATE    │ {"price":15000}                  │ SYNCED  │ 019aaa33:U:3     │
└────┴──────────────┴────────────┴──────────────┴───────────┴──────────────────────────────────┴─────────┴──────────────────┘

Note: rows 1-4 share envelope_id "ENV-001" → they are ONE POS sale transaction.
      Server will process all 4 in a single database transaction (all or nothing).
```

#### `_off_sync_state` — Sync Metadata

Tracks the sync state for this device. One row per device.

```sql
CREATE TABLE _off_sync_state (
  device_id       TEXT PRIMARY KEY,
    -- This device's unique identifier. Example: "DEV-A"

  device_prefix   TEXT NOT NULL,
    -- Prefix for receipt numbering. Example: "001-A"
    -- Format: "{store_code}-{device_letter}"

  last_sync_at    TEXT,
    -- ISO 8601 timestamp of last successful sync.
    -- NULL if never synced. Example: "2026-04-28T10:30:00Z"

  last_pull_version INTEGER NOT NULL DEFAULT 0,
    -- Server's version counter at last pull.
    -- Used for delta sync: "give me all changes since version 12345"

  registered_at   TEXT NOT NULL,
    -- When this device was first registered with the server.

  auth_cached_at  TEXT,
    -- When auth credentials were last cached for offline use.
    -- If (now - auth_cached_at) > max_offline_hours → force re-login.

  user_id         TEXT,
    -- Currently logged-in user ID (cached for offline auth).

  user_hash       TEXT
    -- Bcrypt hash of user's password or PIN (for offline authentication).
);
```

#### `_off_conflict_log` — Conflict History

Records every conflict that occurred during sync and how it was resolved.

```sql
CREATE TABLE _off_conflict_log (
  id              INTEGER PRIMARY KEY AUTOINCREMENT,

  table_name      TEXT NOT NULL,
    -- Which table had the conflict. Example: "products"

  record_id       TEXT NOT NULL,
    -- UUID of the conflicting record. Example: "019aaa33-..."

  field_name      TEXT NOT NULL,
    -- Which field conflicted. Example: "price"

  local_value     TEXT,
    -- Value on this device. Example: "15000"

  remote_value    TEXT,
    -- Value on the server (from another device). Example: "12000"

  resolved_value  TEXT,
    -- Final value after resolution. Example: "15000" (local won)

  resolution      TEXT NOT NULL,
    -- How it was resolved:
    -- "auto_merge"   → different fields edited, both kept
    -- "local_wins"   → same field, local timestamp was newer
    -- "remote_wins"  → same field, remote timestamp was newer
    -- "user_resolved"→ user manually chose a value

  resolved_at     TEXT NOT NULL,
    -- When the conflict was resolved.

  device_id       TEXT NOT NULL
    -- Which device this conflict was detected on.
);
```

#### `_off_number_sequence` — Receipt Numbering

Tracks sequential numbering per device for receipt numbers, invoice numbers, etc.

```sql
CREATE TABLE _off_number_sequence (
  id              INTEGER PRIMARY KEY AUTOINCREMENT,

  table_name      TEXT NOT NULL,
    -- Which table this sequence is for. Example: "sales"

  prefix          TEXT NOT NULL,
    -- Device-specific prefix. Example: "001-A"
    -- Format: "{store_code}-{device_letter}"

  last_sequence   INTEGER NOT NULL DEFAULT 0,
    -- Last used sequence number. Next receipt = last_sequence + 1.
    -- Example: 15 → next receipt = "001-A-0016"

  UNIQUE(table_name, prefix)
);
```

**Visualisasi: Contoh data di `_off_number_sequence`**

```
┌────┬────────────┬────────┬───────────────┐
│ id │ table_name │ prefix │ last_sequence │
├────┼────────────┼────────┼───────────────┤
│  1 │ sales      │ 001-A  │ 15            │  → next receipt: 001-A-0016
│  2 │ invoices   │ 001-A  │ 8             │  → next invoice: 001-A-0009
└────┴────────────┴────────┴───────────────┘
```

### Server-Side Tables

The server (Go engine + PostgreSQL) does NOT need `_off_*` columns on business tables. Business tables stay clean:

```sql
-- Server business tables remain unchanged. Standard columns only:
CREATE TABLE sales (
  id          UUID PRIMARY KEY,       -- same UUID from client
  customer_id UUID,
  sale_date   TIMESTAMPTZ,
  total       DECIMAL(15,2),
  payment_method TEXT,
  status      TEXT,
  created_at  TIMESTAMPTZ DEFAULT NOW(),
  updated_at  TIMESTAMPTZ DEFAULT NOW(),
  created_by  TEXT,                   -- user who created
  updated_by  TEXT                    -- user who last edited
);
```

Server adds 4 infrastructure tables for sync:

#### `_sync_log` — Idempotency Tracking

Prevents duplicate sync processing. If client retries an envelope (because response was lost), server returns cached response.

| Column | Type | Default | Purpose |
|--------|------|---------|---------|
| `envelope_id` | UUID PK | | Envelope ID from client's `_off_outbox`. Idempotency key — same ID twice = return cached response |
| `device_id` | TEXT NOT NULL | | Which device sent this. Example: `"DEV-A"` |
| `received_at` | TIMESTAMPTZ NOT NULL | `NOW()` | When server received this envelope (server timestamp, authoritative) |
| `status` | TEXT NOT NULL | | Result: `"applied"` (success), `"rejected"` (validation failed), `"conflict"` (fields conflicted) |
| `operations_count` | INTEGER NOT NULL | | How many operations in this envelope. Example: 1 sale + 3 items + 1 payment = 5 |
| `response` | JSONB | | Cached response payload. Returned as-is on retry |
| `error_message` | TEXT | `NULL` | If rejected, why. Example: `"FK violation: customer_id '019xxx' not found"` |
| `processing_time_ms` | INTEGER | | Processing duration in ms. For performance monitoring |

#### `_sync_devices` — Device Registry

All registered devices. Device must register (requires internet) before first sync.

| Column | Type | Default | Purpose |
|--------|------|---------|---------|
| `device_id` | TEXT PK | | Unique device ID. Example: `"DEV-A"` |
| `device_prefix` | TEXT NOT NULL UNIQUE | | Receipt numbering prefix. Example: `"001-A"`. Must be unique across all devices |
| `device_name` | TEXT | `NULL` | Human-readable name. Example: `"Kasir 1 - Toko Utama"` |
| `platform` | TEXT NOT NULL | | `"android"`, `"ios"`, `"windows"`, `"macos"`, `"linux"` |
| `app_version` | TEXT | `NULL` | Tauri app version. Example: `"1.2.0"`. Detect outdated clients |
| `user_id` | UUID | `NULL` | Currently logged-in user. Updated on every sync |
| `tenant_id` | UUID | `NULL` | Which tenant (multi-tenant). Devices isolated per tenant |
| `store_id` | UUID | `NULL` | Which store/branch. Used for receipt prefix and data scoping |
| `registered_at` | TIMESTAMPTZ NOT NULL | `NOW()` | First registration timestamp. Immutable |
| `last_sync_at` | TIMESTAMPTZ | `NULL` | Last successful sync. NULL if never synced |
| `last_sync_version` | INTEGER NOT NULL | `0` | Server change version at last sync. For delta sync |
| `is_active` | BOOLEAN NOT NULL | `true` | `false` = device revoked (lost/stolen). Sync rejected |
| `deactivated_at` | TIMESTAMPTZ | `NULL` | When deactivated. NULL if active |
| `deactivated_reason` | TEXT | `NULL` | Why deactivated. Example: `"Device lost"`, `"Employee terminated"` |

#### `_sync_conflicts` — Conflict Audit Trail

Every conflict across all devices. Admin can review and audit.

| Column | Type | Default | Purpose |
|--------|------|---------|---------|
| `id` | SERIAL PK | | Auto-increment ID |
| `envelope_id` | UUID NOT NULL | | Which sync envelope triggered this. FK to `_sync_log` |
| `device_id` | TEXT NOT NULL | | Which device caused the conflict (the "losing" side) |
| `other_device_id` | TEXT | `NULL` | The "winning" side. NULL if conflict with server/admin edit |
| `table_name` | TEXT NOT NULL | | Which table. Example: `"products"` |
| `record_id` | UUID NOT NULL | | Which record |
| `field_name` | TEXT NOT NULL | | Which field. Each conflicting field = separate row |
| `device_value` | TEXT | `NULL` | Value from the losing device. JSON-encoded |
| `server_value` | TEXT | `NULL` | Value already on server |
| `resolved_value` | TEXT | `NULL` | Final value after resolution |
| `resolution` | TEXT NOT NULL | | `"auto_merge"`, `"device_wins"`, `"server_wins"`, `"user_resolved"` |
| `auto_resolved` | BOOLEAN NOT NULL | | `true` = automatic, `false` = needs manual review |
| `reviewed_by` | UUID | `NULL` | Admin who reviewed (if manual). NULL if auto |
| `reviewed_at` | TIMESTAMPTZ | `NULL` | When reviewed. NULL if auto or not yet reviewed |
| `created_at` | TIMESTAMPTZ NOT NULL | `NOW()` | When conflict was recorded |
| `device_hlc` | TEXT | `NULL` | Device's HLC value. For debugging ordering |
| `server_hlc` | TEXT | `NULL` | Server's HLC value. For debugging ordering |

#### `_sync_versions` — Change Tracking for Delta Sync

Every write operation (from any source) gets a version number. Devices use this for delta sync.

| Column | Type | Default | Purpose |
|--------|------|---------|---------|
| `id` | SERIAL PK | | Auto-increment ID |
| `table_name` | TEXT NOT NULL | | Which table changed. Example: `"sales"` |
| `record_id` | UUID NOT NULL | | Which record changed |
| `operation` | TEXT NOT NULL | | `"INSERT"`, `"UPDATE"`, `"DELETE"` |
| `version` | BIGINT NOT NULL | | Global monotonic version. Devices say "give me changes > my last version" |
| `changed_fields` | JSONB | `NULL` | For UPDATE: which fields changed. Example: `["price", "status"]`. NULL for INSERT/DELETE |
| `changed_by` | TEXT | `NULL` | Who made this change: device ID or user ID. Used to skip sending device's own changes back |
| `created_at` | TIMESTAMPTZ NOT NULL | `NOW()` | When recorded |

---

## 7. Sync Engine

### Operation-Based Sync (Not State-Based)

The sync engine sends **operations** (what changed), not **full records** (current state). This enables field-level merge and prevents data loss.

```
❌ State-based (bad):
  Send: { id: "019abc", price: 15000, stock: 50, name: "Widget" }
  Problem: overwrites ALL fields, even ones not changed

✅ Operation-based (good):
  Send: { id: "019abc", operation: "UPDATE", changes: { price: 15000 } }
  Only the changed field is sent. Other fields untouched.
```

### Sync Flow

```
1. User creates/edits/deletes record
   └→ Record saved to local SQLite
   └→ Operation recorded in _off_outbox (status: PENDING)

2. Sync engine detects connectivity (periodic check or manual trigger)
   └→ Groups PENDING operations into envelopes by transaction
   └→ Sorts envelopes by dependency order (customers before orders)

3. For each envelope:
   └→ Set outbox status: IN_FLIGHT
   └→ POST /api/v1/sync/batch { envelope_id, operations: [...] }
   └→ Server checks _sync_log: already processed?
       ├→ Yes: return cached response (idempotent)
       └→ No: process in DB transaction (all or nothing)
           ├→ Success: INSERT into _sync_log, return "applied"
           └→ Failure: INSERT into _sync_log, return "rejected" + reason

4. Client receives response:
   ├→ "applied": set outbox status SYNCED, set record _off_status = 1
   ├→ "rejected": set outbox status ERROR, increment retry_count
   └→ Network error: leave as IN_FLIGHT, will retry next cycle

5. Pull phase: GET /api/v1/sync/pull?since_version={last_pull_version}
   └→ Server returns all changes since that version
   └→ Client applies changes to local SQLite
   └→ For each incoming change, check for conflicts (see Section 8)
```

---

## 8. Edge Cases & Solutions

### 8.1 Conflict Resolution: Field-Level Merge

| Scenario | Strategy | Example |
|----------|----------|---------|
| Different fields edited on 2 devices | **Auto-merge** — both changes kept | Device A: price=15000. Device B: stock=50. Result: price=15000 AND stock=50 ✅ |
| Same field, different values | **Last-Write-Wins** (by HLC) — latest wins, user notified | Device A: price=15000. Device B: price=12000. HLC(A) > HLC(B) → price=15000 |
| Edit vs Delete | **Edit wins** (default) — deleted record resurrected, edit applied | Device A deletes product. Device B edits price. Result: product restored with new price |
| Parent deleted, child created | **Resurrect parent** — parent restored to maintain referential integrity | Device A deletes customer. Device B creates order for that customer. Result: customer restored |

### 8.2 Sequential Numbering (POS Receipts)

Each device gets a unique prefix. Sequence is sequential per device.

```
Device A receipts: 001-A-0001, 001-A-0002, 001-A-0003
Device B receipts: 001-B-0001, 001-B-0002

Format: {store_code}-{device_letter}-{sequence_zero_padded}
```

No collision possible. Legal compliance met (sequential per terminal). This is industry standard (Square POS, Shopify POS, Dynamics 365).

### 8.3 Inventory Overselling

```
Stock: 10 items
Device A (offline): sells 7 → local stock = 3
Device B (offline): sells 5 → local stock = 5
Sync: total sold = 12, actual stock = 10 → NEGATIVE STOCK
```

**Solution: Accept-then-reconcile (industry standard)**

1. ALLOW the sale offline — never block a sale (better to oversell than lose a customer)
2. Use delta-based inventory (store movements, not absolute quantities)
3. Server detects negative stock during sync → flag as "stock discrepancy"
4. Notify manager via push notification
5. Manager decides: backorder, cancel, or adjust

### 8.4 Sync Mechanics

| Edge Case | Solution |
|-----------|----------|
| **Sync order** | Dependency graph: sync customers → products → sales → sale_items → payments. Engine auto-detects from FK relationships |
| **First sync (large dataset)** | Tiered: Tier 1 (critical, blocks app) → Tier 2 (important, background) → Tier 3 (historical, on-demand) |
| **Bandwidth (2G/3G)** | Delta sync only (changed records since last sync). Compress payload (gzip). Prioritize outgoing mutations over incoming data |
| **Interrupted sync** | Resume from last acknowledged envelope. Idempotency keys prevent duplicates |
| **Clock skew** | Hybrid Logical Clock (HLC) — combines physical timestamp + logical counter. Tolerates clock drift. Deterministic ordering |

### 8.5 Security

| Edge Case | Solution |
|-----------|----------|
| **Offline auth** | Cache encrypted auth token + user profile locally. Validate via cached credentials. Max offline: 72 hours (configurable) |
| **Token expiry** | Auto-refresh when back online before sync. If revoked, force re-login but preserve local data for sync after login |
| **Multi-tenant isolation** | Separate SQLite database per tenant. Switch tenant = switch DB file. Never mix data |
| **Data at rest** | SQLite encryption via tauri-plugin-sql encryption feature |

---

## 9. Developer Experience

### Building a POS Module with Offline Support

#### Step 1: Define Model (same as online — no changes)

```json
{
  "name": "Sale",
  "primary_key": { "strategy": "uuid", "version": "v7" },
  "fields": [
    { "name": "customer_id", "type": "link", "ref": "Customer", "required": false },
    { "name": "sale_date", "type": "datetime", "default": "now" },
    { "name": "total", "type": "currency", "computed": "sum(items.subtotal)" },
    { "name": "payment_method", "type": "select", "options": ["cash", "card", "qris"] },
    { "name": "status", "type": "select", "options": ["draft", "completed", "cancelled"], "default": "draft" },
    { "name": "receipt_number", "type": "string", "auto": true }
  ],
  "children": [
    {
      "name": "items",
      "model": "SaleItem",
      "fields": [
        { "name": "product_id", "type": "link", "ref": "Product", "required": true },
        { "name": "quantity", "type": "integer", "required": true, "min": 1 },
        { "name": "price", "type": "currency", "fetch_from": "product.price" },
        { "name": "subtotal", "type": "currency", "computed": "quantity * price" }
      ]
    }
  ]
}
```

No `_off_*` fields defined. Engine handles all of that automatically.

#### Step 2: Module Config (add ONE line)

```json
{
  "name": "Point of Sale",
  "icon": "shopping-cart",
  "app": {
    "mode": "offline"
  }
}
```

#### Step 3: Define View (same as online — no changes)

```json
{
  "name": "Sales",
  "type": "list",
  "model": "Sale",
  "fields": ["receipt_number", "customer_id", "sale_date", "total", "payment_method", "status"],
  "filters": [{ "field": "sale_date", "operator": ">=", "value": "today" }],
  "sort": [{ "field": "sale_date", "order": "desc" }],
  "actions": [{ "label": "New Sale", "action": "create", "view": "sale_form" }]
}
```

#### What the User Sees

**Online:**
```
┌──────────────────────────────────────┐
│  📱 Point of Sale              🔄   │
│──────────────────────────────────────│
│  🟢 Online  |  Last sync: 2 min ago │
│──────────────────────────────────────│
│  Receipt   Customer   Date    Total  │
│  A-0015    John D.    Today   150K   │
│  A-0014    -          Today    85K   │
│  A-0013    Sarah L.   Today   220K   │
│──────────────────────────────────────│
│  [+ New Sale]                        │
└──────────────────────────────────────┘
```

**Offline:**
```
┌──────────────────────────────────────┐
│  📱 Point of Sale              ⏸️   │
│──────────────────────────────────────│
│  🔴 Offline  |  3 pending sync      │
│──────────────────────────────────────│
│  Receipt   Customer   Date    Total  │
│  ⏳ A-0015  John D.   Today   150K  │
│  ⏳ A-0014  -         Today    85K  │
│  ⏳ A-0013  Sarah L.  Today   220K  │
│──────────────────────────────────────│
│  [+ New Sale]  ← STILL WORKS!       │
└──────────────────────────────────────┘
```

#### Developer Effort Summary

| What Developer Does | Same as Online? |
|---------------------|-----------------|
| Define model (sale.json) | ✅ Exactly the same |
| Define view (sale_list.json) | ✅ Exactly the same |
| Define API (apis/*.json) | ✅ Exactly the same |
| Module config | ➕ Add 1 line: `"mode": "offline"` |
| Sync logic | ❌ Not needed — engine handles |
| SQLite schema | ❌ Not needed — auto-generated |
| Conflict resolution | ❌ Not needed — engine handles |
| Receipt numbering | ❌ Not needed — auto-generated |
| `_off_*` columns | ❌ Not needed — auto-generated |

**Total extra effort for developer: 1 line of config.**

---

## 10. Flutter Feasibility Report

> This section documents the Flutter analysis that led to choosing Tauri instead.

### Component-by-Component Analysis (103 components)

| Category | Total | ✅ Easy | ⚠️ Medium | 🔴 Hard | ❌ Very Hard |
|----------|-------|---------|-----------|---------|-------------|
| Fields | 34 | 24 | 6 | 3 | 1 |
| Layout | 10 | 10 | 0 | 0 | 0 |
| Views | 10 | 4 | 3 | 2 | 1 |
| Charts | 11 | 9 | 2 | 0 | 0 |
| Widgets | 18 | 16 | 2 | 0 | 0 |
| Dialogs | 5 | 5 | 0 | 0 | 0 |
| DataTable | 1 | 0 | 0 | 0 | 1 |
| Search | 4 | 4 | 0 | 0 | 0 |
| Social | 3 | 3 | 0 | 0 | 0 |
| Print | 3 | 1 | 2 | 0 | 0 |
| Table | 1 | 1 | 0 | 0 | 0 |
| Placeholder | 1 | 1 | 0 | 0 | 0 |
| **TOTAL** | **103** | **78 (76%)** | **15 (15%)** | **5 (5%)** | **2 (2%)** |

### Hardest Components for Flutter

1. **bc-datable** (❌) — Custom DataTable with virtual scrolling, column resize, drag-reorder, inline editing, export. 3-4 weeks custom build
2. **bc-view-editor** (❌) — JSON editor + live preview. 100% custom build. 3-4 weeks
3. **bc-field-richtext** (🔴) — Tiptap → flutter_quill. Format mismatch (Delta vs HTML). 1-2 weeks
4. **bc-field-code** (🔴) — CodeMirror → flutter_code_editor. Functional but UX inferior. 1 week
5. **bc-field-geo + bc-view-map** (🔴) — Leaflet → flutter_map. Platform complexity. 1-2 weeks
6. **bc-view-gantt** (🔴) — Frappe Gantt → young Flutter packages. 2-3 weeks

### Why Flutter Was Not Chosen

With Tauri (Approach D+), none of these components need to be rebuilt. All 103 Stencil components work as-is in Tauri's WebView. This saved 16-30 weeks of development effort.

---

## 11. Effort Estimates

### Implementation Phases

| Phase | Scope | Effort |
|-------|-------|--------|
| Tauri shell setup | Project scaffold, Stencil build integration, basic IPC | 1-2 weeks |
| Native bridge (`bc-native.ts`) | Camera, GPS, barcode, biometrics via Tauri plugins | 1-2 weeks |
| Stencil bridge client | Detect shell, route to Tauri IPC or Web API | 1 week |
| Offline infrastructure | SQLite, sync engine, outbox pattern, conflict resolution, `_off_*` columns | 2-3 weeks |
| Testing + polish | Cross-platform testing (Win/Mac/Linux/iOS/Android) | 1-2 weeks |
| **TOTAL** | | **6-10 weeks** |

### What Does NOT Need to Be Built

- ❌ Flutter components (0 of 103)
- ❌ New UI framework
- ❌ Component porting
- ❌ New design system
- ❌ `_off_*` column definitions by developer (auto-generated)
