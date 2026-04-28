HANDOFF CONTEXT
===============

USER REQUESTS (AS-IS)
---------------------
- "(OFFLINE MODE FEATURE) Lanjutkan pekerjaan offline mode dari handoff: handoffs/2026-04-29-offline-mode-phase-2.5-complete.md"
- "Kerjakan Phase 3 (Sync Engine) — Task 3.1 sampai 3.5."
- "Sebelum mulai Phase 3, perbaiki dulu bugs #1-5 dari section 'Known Issues' di handoff."
- "WAJIB berpikir kritis, detail, mateng, lengkap dan jujur."
- "Setelah selesai per phase, update semua docs terkait lalu commit."
- "ada orang/agent lain yang sedang mengerjakan fitur berbeda, jadi jangan sembarangan untuk meremove atau melakukan hal-hal yang bisa merusak kerjaan orang/agent lain"
- "selesaikan semua itu ya" (referring to 4 known issues from Phase 3)

GOAL
----
Implement Phase 4 (Conflict Resolution & Edge Cases) of the offline mode feature: HLC, field-level conflict merge, device-prefixed receipt numbering, and inventory delta tracking. Then update docs and commit.

WORK COMPLETED
--------------
- I fixed 5 bugs from Phase 2.5 known issues: SQL injection prevention (two-layer: regex + schema registry), device_id in outbox, transactional CRUD (BEGIN/COMMIT/ROLLBACK), GetSchema field sort, Tauri beforeDevCommand
- I also fixed _off_version not being incremented (create sets 1, update/delete uses _off_version + 1 in raw SQL)
- I implemented all 5 Phase 3 tasks:
  - Task 3.1: Device registration — server POST /api/v1/sync/register + client OfflineStore.registerDevice()
  - Task 3.2: Outbox recording with device_id, envelope_id, _off_version increment
  - Task 3.3: Sync push — client groups by envelope, server processes atomically with idempotency via _sync_log
  - Task 3.4: Sync pull — server returns delta since version (deduped per record), client applies in transaction
  - Task 3.5: Envelope grouping — beginTransaction()/commitTransaction() for atomic multi-table ops
- I resolved 4 Phase 3 known issues:
  - recordSyncVersion race: PostgreSQL uses INSERT RETURNING (BIGSERIAL), SQLite uses atomic INSERT SELECT MAX+1
  - PullChanges dedup: only latest version per record, UPDATE ops send changed_fields delta
  - syncPush envelope split: after 3 failed retries, multi-op envelopes split into individual operations
  - Device management: PATCH /api/v1/sync/devices/:device_id + PushEnvelope rejects deactivated devices (403)
- Commits: a127bd2 (Phase 3) and 8463b1c (known issues fix), both on master, not pushed

CURRENT STATE
-------------
- Branch: master (not pushed)
- Go build: packages I touch compile clean (presentation, infrastructure, domain, compiler). Note: engine/internal/runtime/embedded/yaegi/ has compile errors from ANOTHER AGENT's WIP — do not touch
- Go tests: 370 passed in 20 packages (my packages). Full ./... fails due to yaegi
- Stencil tests: 80 passed, 6 suites
- TypeScript LSP: 0 errors on offline-store.ts
- Uncommitted files: AGENTS.md, sprints/notes.md, sprints/offline_mode*.md — these are from other agents or owner's personal notes, do NOT commit them

PENDING TASKS
-------------
Phase 4 has 4 tasks:

- Task 4.1: Hybrid Logical Clock (HLC) implementation
  - Create packages/components/src/core/hlc.ts
  - Create packages/components/src/core/hlc.spec.ts
  - HLC format from design doc: "{wall_time_base36}:{logical_counter_base36}:{device_id}"
  - Example: "01jk5p9q:0001:DEV-A"
  - Three operations: now(), receive(remote), compare(a, b)
  - Must be monotonically increasing, handle clock skew up to 1 minute
  - Tie-breaking by device_id (lexicographic)
  - Wire HLC into offline-store.ts create/update to populate _off_hlc column

- Task 4.2: Field-level conflict merge
  - Create engine/internal/runtime/sync/conflict.go + conflict_test.go
  - Modify syncPull in offline-store.ts to detect conflicts during pull
  - Logic: for each incoming change, check if local record was also modified (_off_version > 1 AND _off_status = 'PENDING')
  - If no local changes: apply directly
  - If local changes: compare field by field
    - Field changed only remotely: accept remote
    - Field changed only locally: keep local
    - Field changed on BOTH sides: HLC determines winner, log conflict in _off_conflict_log
  - Edit vs Delete: edit wins, deleted record resurrected
  - Server-side: populate _sync_conflicts table for admin review

- Task 4.3: Device-prefixed receipt numbering
  - Modify offline-store.ts: add getNextReceiptNumber(tableName)
  - Uses _off_number_sequence table (already exists in SQLite migrations)
  - Format: {store_code}-{device_letter}-{zero_padded_sequence}
  - Example: "001-A-0016"
  - Sequential per device, no collision, survives app restart

- Task 4.4: Inventory delta-based tracking
  - Create engine/internal/runtime/sync/inventory.go + inventory_test.go
  - During sync push, inventory changes sync as deltas (qty_delta: -5), not absolutes
  - Server detects negative stock, creates oversell alert, notifies manager
  - Never block a sale — accept then reconcile

After Phase 4: update docs/plans/impl/offline-mode-implementation.md (Phase 4 status), engine/docs/features/offline-mode.md (implementation files table), docs/features.md (feature row), create handoff doc

KEY FILES
---------
- packages/components/src/core/offline-store.ts — main client-side sync engine (636 lines). Has CRUD routing, SQL injection prevention, transactions, registerDevice, syncPush, syncPull, beginTransaction/commitTransaction. Phase 4 adds HLC wiring, conflict detection in syncPull, getNextReceiptNumber
- engine/internal/presentation/api/sync_handler.go — server-side sync API (all 7 endpoints). Phase 4 may need conflict resolution logic here or in a new sync/conflict.go
- packages/components/src/core/bc-native.ts — Tauri/Web bridge (13 methods). dbExecute and dbSelect are the SQLite interface
- packages/tauri/src-tauri/src/main.rs — SQLite migrations. _off_outbox, _off_sync_state, _off_conflict_log, _off_number_sequence tables already exist
- engine/internal/infrastructure/persistence/sync_schema.go — server-side _sync_* table DDLs (4 tables: _sync_log, _sync_devices, _sync_conflicts, _sync_versions)
- engine/internal/infrastructure/persistence/offline_schema.go — _off_* column definitions and client-side table DDLs
- docs/plans/impl/offline-mode-implementation.md — full implementation plan with Phase 4 tasks at line 931
- docs/plans/2026-04-28-offline-mode-design.md — design doc with HLC format (line 561), conflict resolution rules (line 948), receipt numbering (line 957), inventory overselling (line 970)
- engine/docs/features/offline-mode.md — feature reference doc (478 lines)
- packages/components/src/core/offline-store.spec.ts — 17 tests covering CRUD, transactions, SQL injection, envelope grouping

IMPORTANT DECISIONS
-------------------
- SQL injection prevention uses two layers: regex validation (/^[a-zA-Z_][a-zA-Z0-9_]*$/) for all identifiers + schema registry lookup from server's GET /api/v1/sync/schema response
- All offline CRUD operations are wrapped in SQLite transactions (BEGIN/COMMIT/ROLLBACK) — data write and outbox recording are atomic
- _off_version is incremented on every write (create=1, update/delete=+1) — ready for optimistic locking in Phase 4 conflict detection
- Outbox records include device_id and envelope_id — envelope groups related operations for atomic server processing
- Server-side sync uses raw SQL (not GORM models) because sync tables are infrastructure tables (same pattern as audit_log)
- PullChanges deduplicates per record (only latest version entry) and sends changed_fields delta for UPDATE operations
- syncPush splits failed envelopes into individual operations after 3 retries — trade-off between atomicity and progress
- PushEnvelope checks device is_active before processing — deactivated devices get 403
- recordSyncVersion is dialect-aware: PostgreSQL uses BIGSERIAL RETURNING, SQLite/MySQL uses atomic INSERT SELECT MAX+1

EXPLICIT CONSTRAINTS
--------------------
- "ada orang/agent lain yang sedang mengerjakan fitur berbeda, jangan sembarangan meremove atau melakukan hal-hal yang bisa merusak kerjaan orang/agent lain" — only touch offline mode files, never touch engine/internal/runtime/bridge/, engine/internal/runtime/embedded/, engine/internal/runtime/goja/, yaegi/
- "sprints/ folder" is owner's personal notes — never commit generated content there
- "WAJIB berpikir kritis, detail, mateng, lengkap dan jujur"
- "Setelah selesai per phase, update semua docs terkait lalu commit"
- Only stage and commit files you actually changed — check git status carefully before committing

CONTEXT FOR CONTINUATION
------------------------
- Read the handoff at handoffs/2026-04-30-offline-mode-phase-3-complete.md for full Phase 3 details
- Phase 4 implementation plan is at docs/plans/impl/offline-mode-implementation.md line 931-1064
- HLC format spec is at docs/plans/2026-04-28-offline-mode-design.md line 561-569: "{wall_time_base36}:{logical_counter_base36}:{device_id}"
- Conflict resolution rules are at docs/plans/2026-04-28-offline-mode-design.md line 948-955
- The _off_conflict_log table already exists in SQLite (main.rs migration version 3) and _sync_conflicts exists on server (sync_schema.go)
- The _off_number_sequence table already exists in SQLite (main.rs migration version 4)
- go build ./... will fail due to yaegi/ package from another agent — use go build ./internal/presentation/... ./internal/infrastructure/... ./internal/domain/... ./internal/compiler/... to test your packages
- npm test in packages/components runs all 80 Stencil tests
- Known issues still open (from Phase 2.5, low priority): naive search (#6), no local table creation from schema (#7), unverified Tauri plugins (#8), custom UUIDv7 (#9), no initFromServer retry (#10), no module-qualified model names (#11), permissive CSP (#12), unhandled CRUD errors (#13)
