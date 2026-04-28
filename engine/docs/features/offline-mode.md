# Offline Mode

## Overview

Offline mode enables BitCode applications to work without internet connectivity. Data is stored locally in SQLite, synced to the server when connectivity is restored. The UI layer (Stencil components) runs inside a Tauri native shell that provides access to device capabilities (camera, GPS, biometrics, etc.).

**Key principle:** One toggle (`mode: "offline"`) enables everything. All complexity is handled by the engine automatically.

## Configuration

### Basic (one line)

```toml
# bitcode.toml
[app]
mode = "offline"    # "online" (default) | "offline"
```

### Per-Module Override

```json
// modules/pos/module.json
{
  "app": {
    "mode": "offline"
  }
}
```

### Per-Model Override

```json
// modules/crm/models/lead.json — lead is offline
{
  "name": "lead",
  "app": { "mode": "offline" },
  "fields": { ... }
}

// modules/crm/models/contact.json — contact inherits module default (online)
{
  "name": "contact",
  "fields": { ... }
}
```

### Per-View Override

```json
// modules/pos/views/settings.json
{
  "app": {
    "mode": "online"
  }
}
```

**Resolution chain:** `model.app.mode → module.app.mode → project.app.mode → default("online")`

### Advanced Options (all optional, sensible defaults)

```toml
[app.offline]
max_offline_hours = 72               # Force re-auth after N hours offline. Default: 72
sync_batch_size = 100                # Operations per sync batch. Default: 100
inventory_oversell = "allow"         # "allow" (default) | "block" | "warn"
conflict_on_same_field = "latest"    # "latest" (default) | "ask_user" | "server_wins"
```

## Architecture

```
ONLINE MODE:
  User → Stencil component → fetch() → Go Engine API → PostgreSQL

OFFLINE MODE:
  User → Stencil component → BcNative → Local SQLite
                                           │ (background, when online)
                                           └→ Sync engine → Go Engine API → PostgreSQL
```

Offline mode uses Tauri as the native shell. Stencil components are unchanged — they run inside Tauri's native WebView. The bridge abstraction layer (`bc-native.ts`) detects the environment and routes to Tauri IPC or Web API fallback.

### Client-Side CRUD Routing (`offline-store.ts`)

`OfflineStore` intercepts CRUD operations and routes based on model config:

```
OfflineStore.find('lead', params)
  → BcSetup.isModelOffline('lead') → true
  → BcNative.dbSelect() → Local SQLite

OfflineStore.find('contact', params)
  → BcSetup.isModelOffline('contact') → false
  → fetch() → Server API
```

On app init, `OfflineStore.initFromServer()` calls `GET /api/v1/sync/schema` to discover which models are offline and registers them via `BcSetup.registerOfflineModels()`.

Write operations on offline models (create/update/delete) automatically record in `_off_outbox` for later sync.

### Schema Sync Endpoint

`GET /api/v1/sync/schema` returns the list of offline models with their field definitions:

```json
{
  "models": [
    {
      "name": "lead",
      "module": "crm",
      "table_name": "crm_lead",
      "fields": [
        { "name": "name", "type": "string", "required": true },
        { "name": "email", "type": "string" }
      ],
      "primary_key": { "strategy": "uuid", "version": "v7" }
    }
  ]
}
```

## Primary Key Handling

### Supported PK Strategies

| Strategy | Offline Support | Extra Column Needed |
|----------|----------------|-------------------|
| `uuid` (v4, v7, format) | ✅ Full — recommended | No — UUID is the PK everywhere |
| `auto_increment` | ⚠️ With warning | Yes — `_off_uuid` added as temporary identity |
| `natural_key` | ⚠️ With warning | Yes — `_off_uuid` added as backup identity |
| `naming_series` | ⚠️ With warning | Yes — `_off_uuid` added as temporary identity |
| `composite` | ❌ Not supported | Engine rejects with error |
| `manual` | ⚠️ With warning | Yes — `_off_uuid` added, collision risk |

### Recommended: `uuid` with `v7`

UUIDv7 is generated on the client device. The same UUID becomes the permanent PK on the server. No ID remapping needed. No collision possible.

```json
"primary_key": { "strategy": "uuid", "version": "v7" }
```

### Why `auto_increment` Needs Warning

Auto-increment IDs are assigned by the server. Client cannot generate valid IDs offline. Engine adds `_off_uuid` as temporary identity and remaps all FK references after sync. This adds complexity.

```
Engine warning when auto_increment + offline:
  ⚠️ WARNING: Module "pos" uses pk:"auto_increment" with mode:"offline".
     Auto-increment PK requires additional sync complexity (ID remapping).
     Consider using pk:"uuid" for better offline support.
```

## Auto-Generated Columns (`_off_*` prefix)

When `mode = "offline"`, the engine automatically adds these columns to every table in the module. Developers do NOT define these — they are invisible in model JSON.

### Per-Record Columns

| Column | Type | Default | Purpose |
|--------|------|---------|---------|
| `_off_device_id` | TEXT NOT NULL | (from device registration) | ID of the device that created/modified this record. Used for receipt numbering prefix, conflict attribution, audit trail. Example: `"DEV-A"` |
| `_off_status` | INTEGER NOT NULL | `0` | Sync status. `0` = PENDING (not yet synced), `1` = SYNCED (confirmed on server), `2` = CONFLICT (needs review). UI shows ⏳/✅/⚠️ based on this |
| `_off_version` | INTEGER NOT NULL | `1` | Version counter. Increments on every edit. Used for optimistic concurrency: if client version < server version, someone else edited while offline. Server compares versions to detect conflicts |
| `_off_deleted` | INTEGER NOT NULL | `0` | Soft delete flag. `0` = active, `1` = deleted. Records are never hard-deleted from local SQLite so the sync engine can propagate deletions to the server |
| `_off_created_at` | TEXT NOT NULL | `now()` | ISO 8601 timestamp when record was created. Used for sorting, audit trail, tiered sync (recent data syncs first) |
| `_off_updated_at` | TEXT NOT NULL | `now()` | ISO 8601 timestamp of last modification. Updated on every business field change. Used for delta sync (only sync records changed since last sync) |
| `_off_hlc` | TEXT NOT NULL | (generated) | Hybrid Logical Clock. Format: `"{wall_time}:{counter}:{device_id}"`. Handles clock skew between devices. Used for conflict resolution ordering — determines "who edited last" even when device clocks disagree |
| `_off_envelope_id` | TEXT | `NULL` | Sync batch ID. Groups related operations into one atomic envelope. NULL until sync engine batches the record. All records in one envelope are processed as a single server transaction |

### Conditional Column

| Column | Type | When Added | Purpose |
|--------|------|-----------|---------|
| `_off_uuid` | TEXT NOT NULL UNIQUE | Only when PK is NOT `uuid` (i.e., `auto_increment`, `natural_key`, `naming_series`, `manual`) | Temporary/backup identity for sync. Client-generated UUIDv7 used as record identity while offline. After sync, server assigns the real PK (auto-increment ID or validates natural key) |

## Auto-Generated Tables (`_off_*` prefix)

### `_off_outbox` — Sync Queue

Every CREATE, UPDATE, DELETE operation is recorded here before being sent to the server.

| Column | Type | Purpose |
|--------|------|---------|
| `id` | INTEGER PK AUTOINCREMENT | Local ordering of operations |
| `envelope_id` | TEXT NOT NULL | Groups related operations into one atomic batch. Example: 1 sale header + 3 items + 1 payment = 5 operations sharing one envelope_id |
| `table_name` | TEXT NOT NULL | Which table this operation affects. Example: `"sales"`, `"sale_items"` |
| `record_id` | TEXT NOT NULL | UUID of the affected record |
| `operation` | TEXT NOT NULL | `"CREATE"`, `"UPDATE"`, or `"DELETE"` |
| `payload` | TEXT NOT NULL | JSON data. CREATE: full record. UPDATE: only changed fields (delta). DELETE: empty `{}` |
| `status` | TEXT NOT NULL DEFAULT 'PENDING' | Lifecycle: `PENDING` → `IN_FLIGHT` → `SYNCED` or `ERROR` or `DEAD` |
| `idempotency_key` | TEXT NOT NULL UNIQUE | Prevents duplicate sync. Format: `"{record_id}:{operation}:{version}"`. If sync succeeds but response is lost, retry with same key → server returns cached response |
| `created_at` | TEXT NOT NULL | When this operation was recorded |
| `retry_count` | INTEGER NOT NULL DEFAULT 0 | Retry attempts. After 5 retries → status becomes `DEAD` |

### `_off_sync_state` — Device Sync Metadata

One row per device. Tracks sync progress and offline auth.

| Column | Type | Purpose |
|--------|------|---------|
| `device_id` | TEXT PK | This device's unique ID. Example: `"DEV-A"` |
| `device_prefix` | TEXT NOT NULL | Receipt numbering prefix. Example: `"001-A"` (store 001, device A) |
| `last_sync_at` | TEXT | ISO 8601 timestamp of last successful sync. NULL if never synced |
| `last_pull_version` | INTEGER NOT NULL DEFAULT 0 | Server's version counter at last pull. Used for delta sync: "give me changes since version X" |
| `registered_at` | TEXT NOT NULL | When this device was first registered |
| `auth_cached_at` | TEXT | When auth credentials were cached. If `(now - auth_cached_at) > max_offline_hours` → force re-login |
| `user_id` | TEXT | Currently logged-in user ID (cached for offline auth) |
| `user_hash` | TEXT | Bcrypt hash of password/PIN for offline authentication |

### `_off_conflict_log` — Conflict History

Records every conflict detected during sync and how it was resolved.

| Column | Type | Purpose |
|--------|------|---------|
| `id` | INTEGER PK AUTOINCREMENT | Auto-increment ID |
| `table_name` | TEXT NOT NULL | Which table had the conflict |
| `record_id` | TEXT NOT NULL | UUID of the conflicting record |
| `field_name` | TEXT NOT NULL | Which field conflicted. Example: `"price"` |
| `local_value` | TEXT | Value on this device |
| `remote_value` | TEXT | Value on the server (from another device) |
| `resolved_value` | TEXT | Final value after resolution |
| `resolution` | TEXT NOT NULL | How resolved: `"auto_merge"`, `"local_wins"`, `"remote_wins"`, `"user_resolved"` |
| `resolved_at` | TEXT NOT NULL | When the conflict was resolved |
| `device_id` | TEXT NOT NULL | Which device detected this conflict |

### `_off_number_sequence` — Receipt Numbering

Tracks sequential numbering per device for receipts, invoices, etc.

| Column | Type | Purpose |
|--------|------|---------|
| `id` | INTEGER PK AUTOINCREMENT | Auto-increment ID |
| `table_name` | TEXT NOT NULL | Which table this sequence is for. Example: `"sales"` |
| `prefix` | TEXT NOT NULL | Device-specific prefix. Example: `"001-A"` |
| `last_sequence` | INTEGER NOT NULL DEFAULT 0 | Last used number. Next = last_sequence + 1 |
| UNIQUE | | `(table_name, prefix)` |

## Server-Side Infrastructure

Server (Go engine + PostgreSQL) does NOT need `_off_*` columns on business tables. The `sales` table on the server stays clean — just business fields + standard `created_at`/`updated_at`/`created_by`/`updated_by`.

Server adds these infrastructure tables for sync.

**These tables are Go-generated, NOT JSON models.** They are infrastructure tables (like `audit_log`, `data_revisions`, `ir_migration`) created directly via `sync_schema.go` DDL at startup. They do NOT appear in admin panel, do NOT get auto-CRUD, and developers never define them. This is consistent with how all other infrastructure tables work in BitCode. Future consideration: when the base/core module is migrated from Go code to JSON model definitions, these sync tables should be evaluated for migration too.

### `_sync_log` — Idempotency Tracking

Prevents duplicate sync processing. When a client sends an envelope, the server records it here. If the same envelope is sent again (e.g., client retried because response was lost), server returns the cached response instead of processing again.

| Column | Type | Default | Purpose |
|--------|------|---------|---------|
| `envelope_id` | UUID PRIMARY KEY | | The envelope ID from the client's `_off_outbox.envelope_id`. This is the idempotency key — if the server sees the same `envelope_id` twice, it knows it's a retry and returns the cached response |
| `device_id` | TEXT NOT NULL | | Which device sent this envelope. Example: `"DEV-A"`. Used for audit trail and debugging sync issues |
| `received_at` | TIMESTAMPTZ NOT NULL | `NOW()` | When the server received this envelope. Server timestamp, not client timestamp — authoritative for ordering |
| `status` | TEXT NOT NULL | | Result of processing: `"applied"` (all operations succeeded), `"rejected"` (validation/business rule failed), `"conflict"` (one or more fields conflicted, resolved automatically or flagged for review) |
| `operations_count` | INTEGER NOT NULL | | How many operations were in this envelope. Example: a POS sale with 1 header + 3 items + 1 payment = 5 operations. Used for monitoring and debugging |
| `response` | JSONB | | Cached response payload. Returned as-is on idempotent retry. Contains: operation results, assigned server IDs (for auto_increment PK), conflict details if any |
| `error_message` | TEXT | `NULL` | If status is `"rejected"`, the error message explaining why. Example: `"Foreign key violation: customer_id '019xxx' does not exist"` |
| `processing_time_ms` | INTEGER | | How long the server took to process this envelope in milliseconds. Used for performance monitoring |

**Example data:**

```
┌──────────────────┬───────────┬──────────────────────┬──────────┬─────┬──────────────────────────────┐
│ envelope_id      │ device_id │ received_at          │ status   │ ops │ response                     │
├──────────────────┼───────────┼──────────────────────┼──────────┼─────┼──────────────────────────────┤
│ 019env01-...     │ DEV-A     │ 2026-04-28T10:35:00Z │ applied  │  5  │ {"synced_ids": [...]}        │
│ 019env02-...     │ DEV-A     │ 2026-04-28T11:00:00Z │ applied  │  1  │ {"synced_ids": [...]}        │
│ 019env03-...     │ DEV-B     │ 2026-04-28T11:05:00Z │ conflict │  2  │ {"conflicts": [{...}]}       │
│ 019env04-...     │ DEV-A     │ 2026-04-28T11:10:00Z │ rejected │  3  │ {"error": "FK violation..."}│
└──────────────────┴───────────┴──────────────────────┴──────────┴─────┴──────────────────────────────┘
```

### `_sync_devices` — Device Registry

Tracks all registered devices. A device must register before it can sync. Registration happens on first app launch (requires internet).

| Column | Type | Default | Purpose |
|--------|------|---------|---------|
| `device_id` | TEXT PRIMARY KEY | | Unique device identifier. Generated during first registration. Example: `"DEV-A"`. Format is configurable but must be unique across all devices in the tenant |
| `device_prefix` | TEXT NOT NULL UNIQUE | | Prefix for receipt/invoice numbering. Example: `"001-A"` (store 001, device A). Must be unique — two devices cannot share a prefix, otherwise receipt numbers would collide |
| `device_name` | TEXT | `NULL` | Human-readable name for the device. Example: `"Kasir 1 - Toko Utama"`. Optional, for admin dashboard display |
| `platform` | TEXT NOT NULL | | Device platform: `"android"`, `"ios"`, `"windows"`, `"macos"`, `"linux"`. Detected automatically during registration. Used for platform-specific sync optimizations |
| `app_version` | TEXT | `NULL` | Version of the Tauri app installed on this device. Example: `"1.2.0"`. Used to detect outdated clients that might need forced update |
| `user_id` | UUID | `NULL` | Which user is currently logged in on this device. Updated on every sync. NULL if no user logged in. A user can be logged in on multiple devices |
| `tenant_id` | UUID | `NULL` | Which tenant this device belongs to (for multi-tenant deployments). Devices are isolated per tenant — a device registered for tenant A cannot sync data from tenant B |
| `store_id` | UUID | `NULL` | Which store/branch this device is assigned to. Used for receipt prefix generation and data scoping (device only syncs data relevant to its store) |
| `registered_at` | TIMESTAMPTZ NOT NULL | `NOW()` | When this device was first registered. Immutable after creation |
| `last_sync_at` | TIMESTAMPTZ | `NULL` | When this device last successfully synced (push or pull). NULL if never synced after registration. Used for monitoring — admin can see which devices haven't synced recently |
| `last_sync_version` | INTEGER NOT NULL | `0` | The server's change version at the time of last sync. Used for delta sync: server sends only changes with version > this value |
| `is_active` | BOOLEAN NOT NULL | `true` | Whether this device is active. Set to `false` to revoke a device (e.g., lost/stolen phone). Inactive devices are rejected during sync with error `"Device deactivated"` |
| `deactivated_at` | TIMESTAMPTZ | `NULL` | When the device was deactivated. NULL if still active |
| `deactivated_reason` | TEXT | `NULL` | Why the device was deactivated. Example: `"Device lost"`, `"Employee terminated"`, `"Replaced by new device"` |

**Example data:**

```
┌───────────┬────────┬──────────────────┬──────────┬─────────┬──────────┬──────────────────────┬────────────┬───────────┐
│ device_id │ prefix │ device_name      │ platform │ version │ user_id  │ last_sync_at         │ sync_ver   │ is_active │
├───────────┼────────┼──────────────────┼──────────┼─────────┼──────────┼──────────────────────┼────────────┼───────────┤
│ DEV-A     │ 001-A  │ Kasir 1 - Utama  │ android  │ 1.2.0   │ usr-001  │ 2026-04-28T10:35:00Z │ 12345      │ true      │
│ DEV-B     │ 001-B  │ Kasir 2 - Utama  │ android  │ 1.2.0   │ usr-002  │ 2026-04-28T11:05:00Z │ 12340      │ true      │
│ DEV-C     │ 002-A  │ Kasir 1 - Cabang │ ios      │ 1.1.0   │ usr-003  │ 2026-04-27T16:00:00Z │ 12200      │ true      │
│ DEV-D     │ 001-C  │ Tablet Gudang    │ android  │ 1.0.0   │ NULL     │ NULL                 │ 0          │ false     │
└───────────┴────────┴──────────────────┴──────────┴─────────┴──────────┴──────────────────────┴────────────┴───────────┘
```

### `_sync_conflicts` — Conflict Audit Trail

Records every conflict that occurred during sync across all devices. This is the server-side equivalent of the client's `_off_conflict_log`, but aggregated from all devices. Used by admin to review and audit conflict resolutions.

| Column | Type | Default | Purpose |
|--------|------|---------|---------|
| `id` | SERIAL PRIMARY KEY | | Auto-increment ID for this conflict record |
| `envelope_id` | UUID NOT NULL | | Which sync envelope triggered this conflict. FK to `_sync_log.envelope_id`. Used to trace back to the full sync batch |
| `device_id` | TEXT NOT NULL | | Which device's data caused the conflict. Example: `"DEV-A"`. The "losing" side of the conflict |
| `other_device_id` | TEXT | `NULL` | Which other device's data was the "winning" side. NULL if conflict was with server-originated data (e.g., admin edit via web) |
| `table_name` | TEXT NOT NULL | | Which table had the conflict. Example: `"products"` |
| `record_id` | UUID NOT NULL | | UUID of the conflicting record. Example: `"019aaa33-..."` |
| `field_name` | TEXT NOT NULL | | Which specific field conflicted. Example: `"price"`. Each conflicting field gets its own row — if 3 fields conflict on the same record, there are 3 rows |
| `device_value` | TEXT | `NULL` | The value from the device that lost the conflict. Stored as JSON-encoded string. Example: `"15000"` |
| `server_value` | TEXT | `NULL` | The value that was already on the server (from another device or web edit). Example: `"12000"` |
| `resolved_value` | TEXT | `NULL` | The final value after conflict resolution. Example: `"15000"` (device won because its HLC was newer) |
| `resolution` | TEXT NOT NULL | | How the conflict was resolved: `"auto_merge"` (different fields, both kept), `"device_wins"` (device's HLC was newer), `"server_wins"` (server's HLC was newer), `"user_resolved"` (admin manually chose a value) |
| `auto_resolved` | BOOLEAN NOT NULL | | Whether the conflict was resolved automatically (`true`) or requires manual review (`false`). Business-critical fields (e.g., price) may be flagged for manual review even if auto-resolved |
| `reviewed_by` | UUID | `NULL` | If manually reviewed, which admin user reviewed it. NULL if auto-resolved or not yet reviewed |
| `reviewed_at` | TIMESTAMPTZ | `NULL` | When the manual review happened. NULL if auto-resolved or not yet reviewed |
| `created_at` | TIMESTAMPTZ NOT NULL | `NOW()` | When this conflict was recorded |
| `device_hlc` | TEXT | `NULL` | The Hybrid Logical Clock value from the device. Used for debugging ordering issues |
| `server_hlc` | TEXT | `NULL` | The HLC value from the server side. Used for debugging ordering issues |

**Example data:**

```
┌────┬──────────────┬────────┬────────┬────────────┬──────────────┬────────┬──────────────┬──────────────┬──────────────┬──────────────┬───────────────┐
│ id │ envelope_id  │ device │ other  │ table_name │ record_id    │ field  │ device_value │ server_value │ resolved     │ resolution   │ auto_resolved │
├────┼──────────────┼────────┼────────┼────────────┼──────────────┼────────┼──────────────┼──────────────┼──────────────┼──────────────┼───────────────┤
│  1 │ 019env03-... │ DEV-B  │ DEV-A  │ products   │ 019aaa33-... │ price  │ "12000"      │ "15000"      │ "15000"      │ server_wins  │ true          │
│  2 │ 019env03-... │ DEV-B  │ DEV-A  │ products   │ 019aaa33-... │ stock  │ "50"         │ "45"         │ "50"         │ device_wins  │ true          │
│  3 │ 019env05-... │ DEV-A  │ NULL   │ customers  │ 019bbb44-... │ phone  │ "08123456"   │ "08198765"   │ NULL         │ user_resolved│ false         │
└────┴──────────────┴────────┴────────┴────────────┴──────────────┴────────┴──────────────┴──────────────┴──────────────┴──────────────┴───────────────┘

Row 1: DEV-B sent price=12000, but DEV-A already synced price=15000 (newer HLC). Server value wins.
Row 2: DEV-B sent stock=50, DEV-A had stock=45 (older HLC). Device value wins. Auto-merged with row 1.
Row 3: DEV-A sent phone="08123456", server had "08198765" (from web admin edit). Flagged for manual review.
```

### `_sync_versions` — Change Tracking

Tracks the global version counter for delta sync. Every write operation (from any source — API, sync, admin) increments the version.

| Column | Type | Default | Purpose |
|--------|------|---------|---------|
| `id` | SERIAL PRIMARY KEY | | Auto-increment ID |
| `table_name` | TEXT NOT NULL | | Which table was changed. Example: `"sales"` |
| `record_id` | UUID NOT NULL | | Which record was changed |
| `operation` | TEXT NOT NULL | | What happened: `"INSERT"`, `"UPDATE"`, `"DELETE"` |
| `version` | BIGINT NOT NULL | | Global monotonically increasing version number. Every write gets the next version. Devices use this for delta sync: "give me all changes with version > my last_sync_version" |
| `changed_fields` | JSONB | `NULL` | For UPDATE operations: which fields changed. Example: `["price", "status"]`. NULL for INSERT/DELETE. Used by sync engine to send only changed fields to devices |
| `changed_by` | TEXT | `NULL` | Who made this change: device ID (e.g., `"DEV-A"`) or user ID (e.g., `"usr-001"` for web admin). Used to avoid sending a device's own changes back to it |
| `created_at` | TIMESTAMPTZ NOT NULL | `NOW()` | When this change was recorded |

**Example data:**

```
┌─────┬────────────┬──────────────┬───────────┬─────────┬──────────────────┬────────────┐
│ id  │ table_name │ record_id    │ operation │ version │ changed_fields   │ changed_by │
├─────┼────────────┼──────────────┼───────────┼─────────┼──────────────────┼────────────┤
│ 100 │ sales      │ 019abc12-... │ INSERT    │ 12345   │ NULL             │ DEV-A      │
│ 101 │ sale_items │ 019abc16-... │ INSERT    │ 12346   │ NULL             │ DEV-A      │
│ 102 │ sale_items │ 019abc17-... │ INSERT    │ 12347   │ NULL             │ DEV-A      │
│ 103 │ products   │ 019aaa33-... │ UPDATE    │ 12348   │ ["price"]        │ DEV-A      │
│ 104 │ products   │ 019aaa33-... │ UPDATE    │ 12349   │ ["stock"]        │ DEV-B      │
│ 105 │ customers  │ 019bbb44-... │ UPDATE    │ 12350   │ ["phone"]        │ usr-001    │
└─────┴────────────┴──────────────┴───────────┴─────────┴──────────────────┴────────────┘

Delta sync example:
  DEV-B says: "give me changes since version 12346"
  Server returns: rows 103, 105 (skips 104 because changed_by = DEV-B itself)
  DEV-B now knows: product price changed (by DEV-A), customer phone changed (by admin)
```

## Sync Engine

### Operation-Based (Not State-Based)

Sync sends **operations** (what changed), not full records. This enables field-level merge.

```
✅ Operation-based: { id: "019abc", op: "UPDATE", changes: { price: 15000 } }
   → Only changed field sent. Other fields untouched.

❌ State-based:     { id: "019abc", price: 15000, stock: 50, name: "Widget" }
   → Overwrites ALL fields, even unchanged ones. Data loss risk.
```

### Conflict Resolution

| Scenario | Strategy |
|----------|----------|
| Different fields edited on 2 devices | **Auto-merge** — both changes kept |
| Same field, different values | **Last-Write-Wins** (by HLC timestamp) — latest wins, user notified |
| Edit vs Delete | **Edit wins** — deleted record resurrected, edit applied |
| Parent deleted, child created | **Resurrect parent** — parent restored for referential integrity |

### Receipt Numbering

Device-prefixed sequential numbers. No collision possible.

```
Device A: 001-A-0001, 001-A-0002, 001-A-0003
Device B: 001-B-0001, 001-B-0002
Format: {store_code}-{device_letter}-{sequence_zero_padded}
```

### Inventory Overselling

Default: allow overselling offline (industry standard — Square, Shopify, Lightspeed all do this). Server detects negative stock during sync and alerts manager.

## Bridge Abstraction (`bc-native.ts`)

Single file at `packages/components/src/core/bc-native.ts`. Detects environment via `window.__TAURI__` (requires `withGlobalTauri: true` in `tauri.conf.json`) and routes to Tauri IPC or Web API fallback.

| Method | Tauri | Browser Fallback |
|--------|-------|-----------------|
| `getEnvironment()` | Returns `'tauri-desktop'` or `'tauri-mobile'` | Returns `'browser'` |
| `isTauri()` | Returns `true` | Returns `false` |
| `takePhoto(options?)` | `invoke('plugin:camera\|take_photo')` | `<input type="file" capture>` + canvas compression |
| `getLocation()` | `invoke('plugin:geolocation\|get_position')` | `navigator.geolocation` |
| `dbExecute(sql, params?)` | `invoke('plugin:sql\|execute')` | IndexedDB fallback (returns `{ rowsAffected: 0 }`) |
| `dbSelect(sql, params?)` | `invoke('plugin:sql\|select')` | IndexedDB fallback (returns `[]`) |
| `setDbPath(path)` | Sets SQLite connection string | No-op |
| `scanBarcode()` | `invoke('plugin:barcode-scanner\|scan')` | Throws error (requires Tauri) |
| `authenticate()` | `invoke('plugin:biometric\|authenticate')` | Returns `false` (WebAuthn planned) |
| `saveFile(path, data)` | `invoke('plugin:fs\|write_file')` | Blob download via `<a>` tag |
| `notify(options)` | `invoke('plugin:notification\|notify')` | `new Notification()` Web API |
| `requestNotificationPermission()` | `invoke('plugin:notification\|request_permission')` | `Notification.requestPermission()` |
| `syncData()` | `invoke('sync_data')` | Returns `{ success: true, synced: 0, errors: 0 }` |

## Implementation Files

### Implemented (Phase 1 + 2 + 2.5 + 3)

| File | Purpose |
|------|---------|
| `engine/internal/compiler/parser/module.go` | `ModuleAppConfig`, `OfflineConfig` structs, `IsOffline()`, `GetOfflineConfig()` |
| `engine/internal/compiler/parser/model.go` | `ModelAppConfig` struct, `IsOffline()`, `OfflineModule` bool field on `ModelDefinition` |
| `engine/internal/config.go` | `AppMode` field in `AppConfig`, `app.mode` viper default |
| `engine/internal/app.go` | `initSyncInfrastructure()`, sync route registration, `modelReg.ProjectAppMode` wiring |
| `engine/internal/domain/model/registry.go` | `RegisterWithModule` resolution chain (model→module→project), `ProjectAppMode`, `validateOfflinePK()` |
| `engine/internal/infrastructure/persistence/offline_schema.go` | `OfflineColumns()`, `OfflineUUIDColumn()`, 4 client `_off_*` table DDLs (with `device_id` column) |
| `engine/internal/infrastructure/persistence/sync_schema.go` | 4 server `_sync_*` table DDLs (PostgreSQL/MySQL/SQLite) |
| `engine/internal/infrastructure/persistence/dynamic_model.go` | `buildColumns()` appends `_off_*` columns when `OfflineModule=true` |
| `engine/internal/presentation/api/sync_handler.go` | 6 sync API endpoints: `RegisterDevice`, `PushEnvelope`, `PullChanges`, `DeviceStatus`, `GetSchema` + `CacheAuth` stub |
| `packages/components/src/core/bc-native.ts` | Bridge abstraction layer (13 methods, Tauri/Web fallback) |
| `packages/components/src/core/bc-native.spec.ts` | Bridge unit tests (10 tests) |
| `packages/components/src/core/bc-setup.ts` | `registerOfflineModels()`, `isModelOffline()`, `getOfflineModels()` |
| `packages/components/src/core/offline-store.ts` | Full sync client: CRUD routing, SQL injection prevention, transactions, outbox with device_id/envelope_id, `registerDevice()`, `syncPush()`, `syncPull()`, `beginTransaction()`/`commitTransaction()` |
| `packages/components/src/core/offline-store.spec.ts` | Offline store tests (17 tests — routing, CRUD, transactions, rollback, SQL injection, envelope grouping) |
| `packages/tauri/src-tauri/Cargo.toml` | Tauri 2.10 + plugins (sql, fs, notification, barcode, biometric) |
| `packages/tauri/src-tauri/src/main.rs` | Tauri entry point, plugin registration, SQLite migrations (with `device_id` in outbox) |
| `packages/tauri/src-tauri/tauri.conf.json` | Tauri config — `frontendDist`, `beforeDevCommand`, `withGlobalTauri`, CSP, window |
| `packages/tauri/src-tauri/capabilities/default.json` | Permissions for core, sql, fs, notification |
| `packages/tauri/src-tauri/build.rs` | Tauri build script |
| `packages/tauri/package.json` | npm scripts for dev/build (desktop, android, ios) |

### Planned (Phase 4-5)

| File | Purpose |
|------|---------|
| `engine/internal/runtime/sync/conflict.go` | Field-level conflict resolution logic |
| `engine/internal/runtime/sync/inventory.go` | Inventory delta reconciliation |
| `packages/components/src/core/hlc.ts` | Hybrid Logical Clock implementation |
