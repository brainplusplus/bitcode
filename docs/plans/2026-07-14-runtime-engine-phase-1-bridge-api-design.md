# Phase 1: Bridge API Design (`bitcode.*`)

**Date**: 14 July 2026
**Status**: Draft (v3 — sudo, session.context, 7 new bridges, tls-client, bulk ops, relations, tx, error contract, execution log)
**Depends on**: —
**Unlocks**: Phase 1.5, 2, 3, 4, 5 (all phases depend on this)
**Master doc**: `2026-07-14-runtime-engine-redesign-master.md`

---

## Table of Contents

1. [Goal](#1-goal)
2. [Current State Analysis](#2-current-state-analysis)
3. [Design Principles](#3-design-principles)
4. [Bridge API Specification — Core](#4-bridge-api-specification--core)
5. [Bridge API Specification — Extended (7)](#5-bridge-api-specification--extended-7)
6. [Execution Log & `bitcode.execution`](#6-execution-log--bitcodeexecution)
7. [sudo() — System Mode](#7-sudo--system-mode)
8. [Error Contract](#8-error-contract)
9. [Security Rules](#9-security-rules)
10. [Go Interface Design](#10-go-interface-design)
11. [Edge Cases](#11-edge-cases)
12. [Tech Debt — Future Additions](#12-tech-debt--future-additions)
13. [Migration Path from Current API](#13-migration-path-from-current-api)
14. [Implementation Tasks](#14-implementation-tasks)

---

## 1. Goal

Design the `bitcode.*` bridge API — the unified interface that all 4 runtimes (Node.js, Python, goja, yaegi) will expose to scripts. This is the **contract** between engine and scripts.

**This phase produces Go interfaces and types only — no runtime implementation yet.** The interfaces will be implemented per-runtime in Phase 2-5.

### Success Criteria

- One Go struct (`bridge.Context`) that covers all script needs
- 19 namespaces: simple enough to learn, powerful enough to not need escape hatches
- `sudo()` pattern for system-level scripts (bypass permission, hard delete, cross-tenant)
- Security rules (env whitelist/blacklist, exec whitelist, fs sandbox) built into design
- HTTP client uses `bogdanfinn/tls-client` for anti-bot/proxy support
- Existing 12 scripts in `samples/erp` can be migrated with minimal changes
- API is identical across all runtimes

---

## 2. Current State Analysis

### What Scripts Receive Today

**TypeScript** (via `plugins/typescript/index.js`):
```javascript
const ctx = {
  db: { query: async (sql) => ({ rows: [] }), create: async (table, data) => data },  // STUB
  http: { get: async (url) => ({ status: 200, data: {} }), post: async (url, body) => ({ status: 200, data: {} }) },  // STUB
  log: (...args) => console.error('[plugin]', ...args),
};
```

**Python** (via `plugins/python/runtime.py`):
```python
def execute_script(script_path, params):
    mod = load_script(script_path)
    if hasattr(mod, "execute"):
        return mod.execute(params)  # only params, no ctx
```

### What Scripts Actually Need (from samples/erp comments)

Every script has `// In production: ...` comments showing what they WANT to do:
- `on_deal_won.ts`: "create follow-up task" → needs `bitcode.model().create()`
- `on_leave_approved.ts`: "update employee.leave_balance" → needs `bitcode.model().write()`
- `notify_manager.ts`: "send notification" → needs `bitcode.email.send()` or `bitcode.notify.send()`

### What the Engine Already Has (in Go)

| Engine Capability | Go Location | Bridge Method |
|-------------------|-------------|---------------|
| Model CRUD | `steps/data.go` → `GenericRepository` | `bitcode.model()` |
| Raw SQL | `GenericRepository` → GORM | `bitcode.db` |
| HTTP client | `steps/http.go` → `net/http` | `bitcode.http` |
| Cache | `infrastructure/cache/` | `bitcode.cache` |
| Event bus | `domain/event/bus.go` | `bitcode.emit()` |
| Process executor | `steps/call.go` → `Executor` | `bitcode.call()` |
| Email | `pkg/email/sender.go` | `bitcode.email` |
| WebSocket | `websocket/hub.go` → `Broadcast()` | `bitcode.notify` |
| Storage | `domain/storage/` → Local + S3 | `bitcode.storage` |
| i18n | `infrastructure/i18n/loader.go` | `bitcode.t()` |
| Permissions | `persistence/permission_checker.go` | `bitcode.security` |
| Audit log | `persistence/audit_log.go` | `bitcode.audit` |
| Encryption | `pkg/security/encryption.go` | `bitcode.crypto` |
| Config | `config.go` → viper | `bitcode.env()`, `bitcode.config()` |

**Key insight**: The bridge API is NOT new functionality. It's exposing what the engine already does internally, to scripts.

---

## 3. Design Principles

Motto: **simplicity, flexible, powerful**.

### Simplicity
- Flat namespace: `bitcode.model()`, `bitcode.http`, `bitcode.log()`
- Consistent patterns: all model operations return the same shape
- One method per concept, not five variants

### Flexible
- Same API, all runtimes: learn once, use everywhere
- Extensible via `bridges/` folder: custom `bitcode.xxx` namespaces
- `sudo()` for system-level access without separate API

### Powerful
- Full CRUD with permission-aware default + sudo bypass
- Stealth HTTP client (TLS fingerprint, proxy, cookie jar)
- Email, notifications, storage, crypto — all built-in
- Cross-tenant access for platform admin scripts

---

## 4. Bridge API Specification — Core (12)

### 4.1 `bitcode.model(name)` — Model CRUD

Default mode: respects permissions, record rules, tenant filter, soft delete.

```javascript
// Search
const leads = await bitcode.model("lead").search({
  domain: [["status", "=", "new"]],
  fields: ["name", "email", "company"],
  order: "created_at desc",
  limit: 50,
  offset: 0
});

// Get by ID
const lead = await bitcode.model("lead").get(id);

// Create
const newLead = await bitcode.model("lead").create({
  name: "John Doe", email: "john@example.com"
});

// Update
await bitcode.model("lead").write(id, { status: "qualified", score: 85 });

// Delete (soft delete if model has soft_deletes)
await bitcode.model("lead").delete(id);

// Count
const count = await bitcode.model("lead").count({ domain: [["status", "=", "new"]] });

// Sum
const total = await bitcode.model("lead").sum("expected_revenue", { domain: [["status", "=", "won"]] });

// Upsert
const record = await bitcode.model("lead").upsert(
  { email: "john@example.com", name: "John Doe", score: 90 },
  ["email"]
);
```

For sudo mode (bypass permissions, hard delete, cross-tenant), see [§6](#6-sudo--system-mode).

### 4.1b `bitcode.model(name)` — Bulk Operations

```javascript
// Bulk Create — single multi-row INSERT (not loop)
const leads = await bitcode.model("lead").createMany([
  { name: "Alice", email: "alice@example.com" },
  { name: "Bob", email: "bob@example.com" },
  { name: "Charlie", email: "charlie@example.com" }
]);
// Returns: [{ id, name, email, ... }, ...] (all created records)

// Bulk Update — UPDATE WHERE id IN (...)
await bitcode.model("lead").writeMany(["id1", "id2", "id3"], {
  status: "qualified"
});
// Returns: { updated: 3 }

// Bulk Delete — DELETE WHERE id IN (...) (soft delete if model has soft_deletes)
await bitcode.model("lead").deleteMany(["id1", "id2", "id3"]);
// Returns: { deleted: 3 }

// Bulk Upsert — INSERT ON CONFLICT UPDATE for each record
const results = await bitcode.model("lead").upsertMany(
  [
    { email: "alice@example.com", score: 90 },
    { email: "bob@example.com", score: 85 }
  ],
  ["email"]  // unique fields to match on
);
// Returns: [{ id, email, score, ... }, ...]
```

**Why not just loop `create()` / `write()`?**
- `createMany([1000 records])` = 1 SQL statement. Loop = 1000 SQL statements. 10-100x faster.
- Bulk ops still run validation and hooks per record, but use batch DB operations.
- `writeMany()` uses `UPDATE ... WHERE id IN (...)` — single query.

**sudo() works with bulk too:**
```javascript
await bitcode.model("lead").sudo().createMany([...]);
await bitcode.model("lead").sudo().deleteMany(ids);  // soft delete
await bitcode.model("lead").sudo().hardDeleteMany(ids);  // permanent
```

### 4.1c `bitcode.model(name)` — Relation Operations

```javascript
// Many2Many: add relations
await bitcode.model("group").addRelation(groupId, "users", [userId1, userId2]);

// Many2Many: remove relations
await bitcode.model("group").removeRelation(groupId, "users", [userId3]);

// Many2Many: load related records
const users = await bitcode.model("group").loadRelation(groupId, "users");
// Returns: [{ id, username, email, ... }, ...]

// Many2Many: set relations (replace all)
await bitcode.model("group").setRelation(groupId, "users", [userId1, userId2]);
// Removes all existing, adds these

// Eager loading: get record with relations populated
const group = await bitcode.model("group").get(groupId, {
  include: ["users", "menus", "pages", "access_rights"]
});
// Returns: { id, name, users: [...], menus: [...], pages: [...], access_rights: [...] }

// Search with eager loading
const groups = await bitcode.model("group").search({
  domain: [["category", "=", "security"]],
  include: ["users"]
});
// Returns: [{ id, name, users: [...] }, ...]
```

**Maps to existing Go code:**
- `addRelation()` → `GenericRepository.AddMany2Many()`
- `removeRelation()` → `GenericRepository.RemoveMany2Many()`
- `loadRelation()` → `GenericRepository.LoadMany2Many()`
- `include` → `loadMany2OneRelation()`, `loadOne2ManyRelation()`, `loadMany2ManyRelation()`

**Why this matters for module setting:**
- Group form 7-tab: each tab is a relation (users, menus, pages, access_rights, record_rules)
- Security sync: bulk add/remove permissions across groups
- Without this, every tab requires manual queries with JOINs

### 4.1d `bitcode.tx()` — Transactions

```javascript
// Atomic multi-model operation
await bitcode.tx(async (tx) => {
  const order = await tx.model("order").create({ customer_id: 42, total: 500 });
  
  await tx.model("order_line").createMany([
    { order_id: order.id, product_id: 1, qty: 5, price: 50 },
    { order_id: order.id, product_id: 2, qty: 3, price: 100 }
  ]);
  
  await tx.model("inventory").write(itemId, { qty: qty - 8 });
  
  // If any operation fails → ALL rolled back automatically
});
// If callback completes without error → committed
// If callback throws → rolled back
```

**`tx` is a scoped bridge context** — same API as `bitcode`, but all operations share one DB transaction:

```javascript
await bitcode.tx(async (tx) => {
  // tx.model() — same as bitcode.model() but transactional
  // tx.db.query() — same as bitcode.db.query() but transactional
  // tx.db.execute() — same as bitcode.db.execute() but transactional
  
  // These are NOT transactional (doesn't make sense):
  // tx.http, tx.cache, tx.email, tx.notify, tx.fs, tx.exec — same as bitcode.*
});
```

**sudo() inside transaction:**
```javascript
await bitcode.tx(async (tx) => {
  await tx.model("salary").sudo().create({ employee_id: 1, amount: 5000 });
  await tx.model("audit_log").sudo().create({ action: "salary_created" });
});
```

**Edge cases:**

| Edge Case | Behavior |
|-----------|----------|
| Nested `bitcode.tx()` inside `bitcode.tx()` | Uses savepoints (nested transaction). Rollback inner doesn't rollback outer. |
| `bitcode.tx()` with MongoDB | Works for MongoDB 4.1+ (multi-document transactions). Error for older versions: "transactions require MongoDB 4.1+" |
| Timeout inside transaction | Transaction rolled back after timeout. |
| `bitcode.call()` inside transaction | Called process shares the same transaction context. |
| Transaction exceeds max duration (configurable, default 30s) | Auto-rollback with error: "transaction timeout" |

**Maps to existing Go code:** `GenericRepository.Transaction()` which wraps GORM's `db.Transaction()`.

### 4.2 `bitcode.db` — Raw Database

```javascript
const results = await bitcode.db.query(
  "SELECT department, COUNT(*) as count FROM employees GROUP BY department"
);

const affected = await bitcode.db.execute(
  "UPDATE leads SET score = score + ? WHERE status = ?", [10, "qualified"]
);
// Returns: { rows_affected: number }
```

**Note**: Raw SQL bypasses permissions, record rules, and tenant filter. Parameterized queries only — `?` placeholder mandatory.

### 4.3 `bitcode.http` — HTTP Client (TLS-Client)

Powered by `bogdanfinn/tls-client` — supports TLS fingerprinting, proxy, cookie jar, header ordering.

```javascript
// Simple request
const resp = await bitcode.http.get("https://api.example.com/data", {
  headers: { "Authorization": "Bearer xxx" },
  timeout: 5000
});
// Returns: { status: 200, headers: {...}, body: {...} }

// Stealth request (anti-bot bypass)
const resp = await bitcode.http.get("https://target.com", {
  profile: "chrome_131",
  proxy: "socks5://user:pass@proxy.com:1080",
  headers: {
    "accept": "text/html",
    "accept-language": "en-US",
    "user-agent": "Mozilla/5.0 ..."
  },
  headerOrder: ["accept", "accept-language", "user-agent"],
  cookieJar: "session-1",
  timeout: 30000,
  followRedirects: true
});

// POST, PUT, PATCH, DELETE
const resp = await bitcode.http.post("https://api.example.com/webhook", {
  body: { event: "lead.won", data: { id: 123 } },
  headers: { "Content-Type": "application/json" }
});
```

**Why tls-client instead of net/http:**
- TLS fingerprint rotation (Chrome, Firefox, Safari profiles)
- Proxy support (HTTP, SOCKS5) built-in
- Header ordering (critical for anti-bot)
- Cookie jar with named sessions (persist across requests)
- General purpose: crawling, scraping, API calls all covered

### 4.4 `bitcode.cache` — Cache

```javascript
await bitcode.cache.set("lead:123:score", 85, { ttl: 3600 });
const score = await bitcode.cache.get("lead:123:score");
await bitcode.cache.del("lead:123:score");
```

### 4.5 `bitcode.env(key)` — Environment Variables

```javascript
const apiKey = bitcode.env("STRIPE_API_KEY");
```

Resolution order: OS env → `.env` → `bitcode.yaml`/`bitcode.toml` → default.
Security: controlled by `env_allow`/`env_deny` in `module.json` (see [§7](#7-security-rules)).
Synchronous — env vars don't change at runtime.

### 4.6 `bitcode.session` — Current Request Context

```javascript
bitcode.session.userId      // "john-123"
bitcode.session.username    // "john"
bitcode.session.tenantId    // "maju-jaya" (from middleware, null if single-tenant)
bitcode.session.groups      // ["base.user", "crm.manager"]
bitcode.session.locale      // "id"
bitcode.session.context     // { company_id: "comp-1", branch_id: "br-2", ... }
```

**`session.context`** — flexible key-value set by module (not engine). Engine doesn't hardcode `company_id` — module sets this at login or company-switch. Used in record rules as `{{user.company_id}}`.

| Caller | userId | tenantId | groups | context |
|--------|--------|----------|--------|---------|
| User request | user's ID | from middleware | user's groups | set by module at login |
| Cron job | `"system"` | null or per-tenant config | `["base.admin"]` | `{}` |
| Webhook (no auth) | `"anonymous"` | from middleware | `[]` | `{}` |
| Impersonated | impersonated user | same | impersonated user's groups | impersonated user's context |

### 4.7 `bitcode.config(key)` — Module Settings

```javascript
const limit = bitcode.config("view_revision_limit");
const threshold = bitcode.config("crm.lead_score_threshold");
```

Reads from `module.json` → `settings` field.

### 4.8 `bitcode.log(level, message, data?)` — Logging

```javascript
bitcode.log("info", "Lead qualified", { leadId: 123, score: 85 });
bitcode.log("warn", "API rate limit approaching");
bitcode.log("error", "Payment failed", { error: err.message });
```

Levels: `debug`, `info`, `warn`, `error`. Goes to engine's structured logger (not stdout).

### 4.9 `bitcode.emit(event, data)` — Event Bus

```javascript
await bitcode.emit("lead.qualified", { id: 123, score: 85 });
```

Triggers agents subscribed to this event.

### 4.10 `bitcode.call(process, input)` — Call Another Process

```javascript
const result = await bitcode.call("qualify_lead", { id: 123 });
```

Max nesting depth: 10 (existing engine limit).

### 4.11 `bitcode.fs` — Filesystem (Sandboxed)

```javascript
const data = await bitcode.fs.read("data/config.json");
await bitcode.fs.write("output/report.csv", csvContent);
const exists = await bitcode.fs.exists("data/config.json");
const files = await bitcode.fs.list("data/");
await bitcode.fs.mkdir("output/reports");
await bitcode.fs.remove("output/old_report.csv");
```

Relative paths resolve from module directory. Absolute paths require `fs_allow` in module.json.

### 4.12 `bitcode.exec(command, args, opts)` — External Commands

```javascript
const result = await bitcode.exec("pandoc", ["input.md", "-o", "output.pdf"], {
  cwd: "/tmp", timeout: 30000
});
// Returns: { stdout, stderr, exitCode }
```

Controlled by `exec_allow` in `module.json`.

---

## 5. Bridge API Specification — Extended (7)

### 5.1 `bitcode.email` — Send Email

```javascript
// Simple email
await bitcode.email.send({
  to: "manager@company.com",
  subject: "Deal Won: " + lead.name,
  body: "<h1>Congratulations!</h1><p>Revenue: $" + lead.revenue + "</p>"
});

// Template email
await bitcode.email.send({
  to: "hr@company.com",
  subject: "New Employee Onboarding",
  template: "onboarding_welcome",
  data: { name: employee.name, department: dept.name }
});
```

Maps to: existing `email.Sender` interface (`pkg/email/sender.go`).

### 5.2 `bitcode.notify` — Push Notification to UI

```javascript
// Notify specific user (via WebSocket)
await bitcode.notify.send({
  to: "user:john-123",
  title: "Leave Approved",
  message: "Your leave request has been approved",
  type: "success"   // success | warning | error | info
});

// Broadcast to channel (all subscribers)
await bitcode.notify.broadcast("lead.won", {
  leadName: lead.name,
  revenue: lead.expected_revenue
});
```

Maps to: existing `websocket.Hub.Broadcast()` and `BroadcastToTenant()`.

**Different from `bitcode.emit()`**: emit triggers internal agents/processes. Notify pushes to browser/UI via WebSocket.

### 5.3 `bitcode.storage` — Managed File Storage

```javascript
// Upload
const attachment = await bitcode.storage.upload({
  filename: "report.pdf",
  content: pdfBuffer,
  model: "lead",
  recordId: lead.id
});
// Returns: { id, url, filename, size, contentType }

// Get download URL
const url = await bitcode.storage.url(attachment.id);

// Download content
const content = await bitcode.storage.download(attachment.id);

// Delete
await bitcode.storage.delete(attachment.id);
```

Maps to: existing `StorageDriver` (Local + S3) + `AttachmentRepository`.

**Different from `bitcode.fs`**: fs is raw filesystem (read/write files on disk). Storage is managed file storage with metadata, thumbnails, S3 support, attachment tracking.

### 5.4 `bitcode.t(key)` — i18n Translation

```javascript
const msg = bitcode.t("lead_qualified_notification");
// Returns: "Lead telah dikualifikasi" (locale = "id")
// Returns: "Lead has been qualified" (locale = "en")
```

Locale auto-resolved from `bitcode.session.locale`. Maps to: existing `Translator.Translate()`.

### 5.5 `bitcode.security` — Permission & Group Checks

```javascript
// Check model permissions for current user
const perms = await bitcode.security.permissions("lead");
// Returns: { canRead, canWrite, canCreate, canDelete, canPrint, canEmail, canExport, canImport, canClone }

// Check if current user is in a specific group
const isManager = await bitcode.security.hasGroup("crm.manager");
// Returns: boolean

// Get current user's group list
const groups = await bitcode.security.groups();
// Returns: ["base.user", "crm.manager"]
```

Maps to: existing `PermissionService.GetModelPermissions()`, `ResolveUserGroupIDs()`.

**Why expose this when `bitcode.model()` already enforces permissions?**
- Conditional UI logic: show/hide buttons based on permission
- Custom authorization in scripts: "only managers can trigger this workflow"
- Audit: log who has access to what

### 5.6 `bitcode.audit` — Custom Audit Logging

```javascript
await bitcode.audit.log({
  action: "export",
  model: "lead",
  recordId: lead.id,
  detail: "Exported 500 leads to CSV"
});
```

Maps to: existing `AuditLogRepository.Write()`.

**Why expose this when engine already auto-audits CRUD?**
- Custom business events: "user exported data", "user ran report", "user approved workflow"
- Compliance: explicit audit trail for sensitive operations

### 5.7 `bitcode.crypto` — Encryption & Hashing

```javascript
// Encrypt (AES-256-GCM, using engine's encryption key)
const encrypted = await bitcode.crypto.encrypt("sensitive data");
const decrypted = await bitcode.crypto.decrypt(encrypted);

// Hash (bcrypt)
const hash = await bitcode.crypto.hash("password123");
const match = await bitcode.crypto.verify("password123", hash);
// Returns: boolean
```

Maps to: existing `FieldEncryptor.Encrypt/Decrypt()` and `HashPassword/CheckPassword()`.

---

## 6. Execution Log & `bitcode.execution`

### 6.1 Overview

Every process execution is logged to `process_execution` and `process_execution_step` models — standard JSON models in the base module, just like `user.json` or `audit_log.json`. This provides:

- **Debugging**: see exactly what happened, step by step, with input/output data
- **Flow visualization**: UI can render execution flow like n8n
- **Error tracking**: failed executions with full error context and stack trace
- **Performance monitoring**: duration per step, identify bottlenecks
- **Audit trail**: who ran what, when, with what data

### 6.2 Model Definitions

#### `process_execution.json`

```json
{
  "name": "process_execution",
  "module": "base",
  "label": "Process Execution",
  "tenant_scoped": true,
  "timestamps": true,
  "soft_deletes": false,
  "fields": {
    "process_name":  { "type": "string", "required": true, "max": 200, "index": true },
    "module":        { "type": "string", "max": 100 },
    "trigger":       { "type": "select", "options": ["api", "cron", "event", "script", "manual", "hook"], "default": "manual" },
    "status":        { "type": "select", "options": ["running", "success", "error", "timeout", "cancelled"], "default": "running", "index": true },
    "started_at":    { "type": "datetime", "required": true },
    "finished_at":   { "type": "datetime" },
    "duration_ms":   { "type": "integer", "default": 0 },
    "user_id":       { "type": "string", "max": 100, "index": true },
    "input":         { "type": "json" },
    "output":        { "type": "json" },
    "error":         { "type": "json" },
    "parent_id":     { "type": "many2one", "model": "process_execution" },
    "mode":          { "type": "select", "options": ["user", "sudo"], "default": "user" },
    "step_count":    { "type": "integer", "default": 0 },
    "runtime":       { "type": "string", "max": 20 }
  },
  "indexes": [
    ["process_name", "status"],
    ["started_at"],
    ["user_id", "started_at"]
  ],
  "api": {
    "enabled": true,
    "operations": ["read", "list"],
    "auth": true
  }
}
```

#### `process_execution_step.json`

```json
{
  "name": "process_execution_step",
  "module": "base",
  "label": "Process Execution Step",
  "tenant_scoped": true,
  "timestamps": false,
  "soft_deletes": false,
  "fields": {
    "execution_id":  { "type": "many2one", "model": "process_execution", "required": true, "index": true },
    "step_index":    { "type": "integer", "required": true },
    "step_name":     { "type": "string", "max": 200 },
    "step_type":     { "type": "string", "required": true, "max": 50 },
    "status":        { "type": "select", "options": ["success", "error", "skipped", "running"], "default": "running" },
    "started_at":    { "type": "datetime" },
    "duration_ms":   { "type": "integer", "default": 0 },
    "input":         { "type": "json" },
    "output":        { "type": "json" },
    "error":         { "type": "json" },
    "meta":          { "type": "json" }
  },
  "indexes": [
    ["execution_id", "step_index"]
  ],
  "api": {
    "enabled": true,
    "operations": ["read", "list"],
    "auth": true
  }
}
```

### 6.3 How Engine Populates Execution Log

The executor (`executor.go`) wraps every process execution:

```
Process triggered (API / cron / event / script / manual)
  │
  ├── INSERT process_execution (status: "running", started_at: now)
  │
  ├── For each step:
  │   ├── INSERT process_execution_step (status: "running", started_at: now)
  │   ├── Execute step handler
  │   ├── UPDATE step (status: "success"/"error", duration_ms, input, output)
  │   └── If error → stop execution
  │
  ├── UPDATE process_execution (status: "success"/"error", finished_at, duration_ms, output/error)
  │
  └── Done
```

**For DAG execution** (parallel steps): multiple steps can have the same `started_at` and run concurrently. The `step_index` reflects topological order, not execution order.

**For `bitcode.call()`** (nested processes): child execution gets `parent_id` pointing to the caller's execution. This creates a tree:

```
Execution: qualify_lead (parent_id: null)
  ├── Step 0: query leads
  ├── Step 1: script score_leads.js
  ├── Step 2: call "notify_manager"
  │   └── Execution: notify_manager (parent_id: qualify_lead.id)
  │       ├── Step 0: query manager
  │       └── Step 1: http webhook
  └── Step 3: update lead status
```

### 6.4 What Gets Captured Per Step

```javascript
// Example: step_type "query"
{
  step_index: 0,
  step_name: "fetch_new_leads",
  step_type: "query",
  status: "success",
  duration_ms: 45,
  input: { model: "lead", domain: [["status", "=", "new"]], limit: 100 },
  output: { count: 23, records: [{ id: "...", name: "..." }, ...] },
  meta: null
}

// Example: step_type "script"
{
  step_index: 1,
  step_name: "score_leads",
  step_type: "script",
  status: "success",
  duration_ms: 320,
  input: { leads_count: 23 },
  output: { scored: 23, avg_score: 72.5 },
  meta: { runtime: "node", script: "scripts/score_leads.ts" }
}

// Example: step_type "http"
{
  step_index: 2,
  step_name: "notify_crm",
  step_type: "http",
  status: "error",
  duration_ms: 5000,
  input: { url: "https://crm.api/webhook", method: "POST" },
  output: null,
  error: { code: "HTTP_TIMEOUT", message: "request timed out after 5000ms" },
  meta: { profile: "chrome_131", proxy: "socks5://..." }
}

// Example: step_type "condition"
{
  step_index: 3,
  step_name: "check_score",
  step_type: "condition",
  status: "success",
  duration_ms: 1,
  input: { expression: "score > 70", score: 85 },
  output: { result: true, branch: "then" },
  meta: null
}
```

### 6.5 Retention Config

In `bitcode.yaml`:

```yaml
execution_log:
  enabled: true                    # default: true
  
  # What to save
  save_input: true                 # save input data per execution and step
  save_output: true                # save output data per execution and step
  save_steps: true                 # save per-step detail (false = only execution summary)
  save_on_success: true            # save successful executions (false = only errors)
  save_on_error: true              # always true — errors are always saved
  
  # Retention / auto-cleanup
  max_age: "30d"                   # delete executions older than 30 days
  max_records: 100000              # max total records, oldest deleted first
  cleanup_interval: "1h"           # check and cleanup every hour
  
  # Data truncation (prevent huge JSON blobs)
  max_input_size: 10240            # max bytes for input JSON (10KB), truncated with "[truncated]"
  max_output_size: 10240           # max bytes for output JSON (10KB)
```

Per-process override in process JSON:

```json
{
  "name": "qualify_lead",
  "execution_log": {
    "save_input": true,
    "save_output": true,
    "save_steps": true,
    "max_age": "7d"
  },
  "steps": [...]
}
```

```json
{
  "name": "health_check",
  "execution_log": {
    "enabled": false
  },
  "steps": [...]
}
```

### 6.6 `bitcode.execution` — Bridge Namespace (#20)

Scripts can query execution history and access current execution context:

```javascript
// Search execution history
const failures = await bitcode.execution.search({
  process: "qualify_lead",
  status: "error",
  limit: 10,
  order: "started_at desc"
});
// Returns: [{ id, process_name, status, started_at, duration_ms, error, ... }]

// Get specific execution with steps
const exec = await bitcode.execution.get(executionId, {
  include: ["steps"]
});
// Returns: { id, process_name, ..., steps: [{ step_index, step_type, status, ... }] }

// Get current execution context (inside a running script)
const current = bitcode.execution.current();
// Returns: { id, processName, startedAt, parentId, stepIndex, trigger, mode }
// Returns null if not inside a process execution (e.g., direct script call)

// Retry a failed execution (re-run with same input)
await bitcode.execution.retry(executionId);

// Cancel a running execution
await bitcode.execution.cancel(executionId);
```

**`bitcode.execution.current()`** is useful for:
- Logging: "this script is running as part of execution X"
- Conditional logic: "if triggered by cron, do X; if triggered by API, do Y"
- Linking: "attach this result to the parent execution"

### 6.7 Flow Visualization Data

The execution + steps data is sufficient for UI to render flow visualization:

```
Process: qualify_lead
Status: ✅ success (2.15s)
Trigger: api | User: john-123 | 2026-07-14 10:30:00

┌──────────────┐    ┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│ query leads  │───►│ score leads  │───►│ score > 70?  │───►│ notify CRM   │
│ 45ms ✅      │    │ 320ms ✅     │    │ 1ms ✅       │    │ 350ms ✅     │
│ 23 records   │    │ avg: 72.5    │    │ branch: then │    │ status: 200  │
└──────────────┘    └──────────────┘    └──────────────┘    └──────────────┘

Input: { "source": "api", "filters": [...] }
Output: { "qualified": 15, "disqualified": 8 }
```

For DAG (parallel steps):

```
┌──────────────┐
│ fetch API 1  │──┐
│ 150ms ✅     │  │    ┌──────────────┐    ┌──────────────┐
└──────────────┘  ├───►│ merge results│───►│ save to DB   │
┌──────────────┐  │    │ 5ms ✅       │    │ 30ms ✅      │
│ fetch API 2  │──┘    └──────────────┘    └──────────────┘
│ 200ms ✅     │
└──────────────┘
```

Module `setting` can build this visualization using:
- `bitcode.execution.get(id, { include: ["steps"] })` for data
- Custom view with template for rendering

### 6.8 Edge Cases

| Edge Case | Behavior |
|-----------|----------|
| `execution_log.enabled = false` | No logging at all. Zero overhead. |
| `save_on_success = false` | Only errors saved. Reduces storage significantly. |
| `save_steps = false` | Only execution summary saved. No per-step detail. |
| Process with 100 steps | All 100 steps logged. Consider `max_input_size`/`max_output_size` to limit JSON size. |
| `bitcode.call()` 5 levels deep | 5 execution records, linked via `parent_id`. Tree structure. |
| Script outputs 50MB JSON | Truncated at `max_output_size` (default 10KB). Logged with `[truncated]` marker. |
| Execution cleanup deletes parent but child still referenced | Cascade delete: child executions deleted with parent. |
| Concurrent cleanup + new execution | Cleanup uses `DELETE WHERE started_at < ?` — no conflict with new inserts. |
| MongoDB execution log | Same model, same behavior. MongoDB handles JSON natively. |
| `execution_log.max_records = 100000` reached | Oldest executions deleted first until under limit. |

---

## 7. sudo() — System Mode

### Concept

Two execution modes for `bitcode.model()`:

| | User Mode (default) | System Mode (`sudo()`) |
|---|---|---|
| Permissions/ACL | ✅ Enforced | ❌ Bypassed |
| Record Rules | ✅ Filtered | ❌ Bypassed |
| Tenant Filter | ✅ Applied | ✅ Applied (unless `withTenant()`) |
| Soft Delete | ✅ Soft delete | ✅ Soft delete (unless `hardDelete()`) |
| Validation | ✅ Enforced | ✅ Enforced (unless `skipValidation()`) |
| Audit Log | ✅ Logged as user | ✅ Logged as "system" or "user (sudo)" |

### API

```javascript
// User mode (default)
await bitcode.model("lead").search({});          // filtered by permission + record rules + tenant
await bitcode.model("lead").delete(id);          // soft delete

// System mode
await bitcode.model("lead").sudo().search({});   // no permission filter, no record rules
await bitcode.model("lead").sudo().delete(id);   // still soft delete

// System mode — exclusive methods
await bitcode.model("lead").sudo().hardDelete(id);                    // permanent delete
await bitcode.model("lead").sudo().withTenant("berkah").search({});   // cross-tenant
await bitcode.model("lead").sudo().skipValidation().create(data);     // skip field validation
```

### Method Chain

```javascript
bitcode.model("lead")              // user mode, current tenant
  .sudo()                          // → system mode
  .withTenant("tenant-x")         // → cross-tenant (sudo only)
  .skipValidation()                // → skip field validation (sudo only)
  .search({...})                   // execute
```

Methods **only available in sudo mode**:
- `hardDelete(id)` — permanent delete from DB
- `withTenant(tenantId)` — cross-tenant access
- `skipValidation()` — skip field validation on create/write

### Who Can sudo()?

Controlled per module and per script:

```json
// module.json — module-level
{ "name": "setting", "sudo_allow": true }
{ "name": "crm", "sudo_allow": false }
```

```json
// process JSON — per script
{ "type": "script", "runtime": "node", "script": "scripts/cleanup.js", "sudo": true }
```

- Default: `sudo_allow: false`
- Cron jobs and agents: automatically `sudo_allow: true` (system context)
- If script calls `sudo()` without permission: error "sudo not allowed for module 'crm'"

### Edge Cases

| Scenario | Behavior |
|----------|----------|
| User script calls `sudo()` without permission | Error: "sudo not allowed for module 'crm'" |
| Cron script calls `sudo()` | Allowed — cron runs as system |
| `sudo().hardDelete()` on record with FK constraint | DB error: "foreign key constraint" — engine doesn't override DB constraints |
| `sudo().search()` in multi-tenant | Still filtered by current tenant. Must use `withTenant()` for cross-tenant. |
| `sudo().create(data)` without required field | Still validates by default. Use `skipValidation()` to bypass. |
| `sudo().withTenant("x")` on non-scoped model | `withTenant()` ignored — model has no tenant filter anyway |
| Audit log for sudo operations | Logged with `sudo: true` flag and `actor: "user-id (sudo)"` or `actor: "system"` |

---

## 8. Error Contract

### Universal Error Structure

All bridge methods across all 4 runtimes use the same error shape:

```javascript
{
  code: "RECORD_NOT_FOUND",           // machine-readable error code
  message: "Record not found in lead", // human-readable message
  details: {                           // structured context (optional)
    model: "lead",
    id: "nonexistent-id"
  },
  retryable: false                     // hint: safe to retry?
}
```

### Error Codes

| Code | When | Retryable |
|------|------|-----------|
| `RECORD_NOT_FOUND` | `get()`, `write()`, `delete()` with nonexistent ID | No |
| `MODEL_NOT_FOUND` | `bitcode.model("nonexistent")` | No |
| `VALIDATION_ERROR` | `create()`, `write()` with invalid data | No |
| `PERMISSION_DENIED` | User lacks permission for operation | No |
| `SUDO_NOT_ALLOWED` | `sudo()` called without `sudo_allow` | No |
| `TENANT_REQUIRED` | Create on tenant-scoped model without tenant context | No |
| `TENANT_NOT_FOUND` | `withTenant("nonexistent")` | No |
| `ENV_ACCESS_DENIED` | `env()` for denied key | No |
| `EXEC_DENIED` | `exec()` for non-whitelisted command | No |
| `EXEC_NOT_FOUND` | `exec()` command not in PATH | No |
| `EXEC_TIMEOUT` | `exec()` exceeded timeout | No |
| `FS_ACCESS_DENIED` | `fs` path outside sandbox | No |
| `FS_NOT_FOUND` | `fs.read()` file doesn't exist | No |
| `TX_TIMEOUT` | Transaction exceeded max duration | No |
| `TX_CONFLICT` | Transaction deadlock or serialization failure | Yes |
| `HTTP_TIMEOUT` | HTTP request exceeded timeout | Yes |
| `HTTP_ERROR` | HTTP connection failed | Yes |
| `EMAIL_NOT_CONFIGURED` | SMTP not configured | No |
| `STORAGE_ERROR` | Storage upload/download failed | Yes |
| `CRYPTO_ERROR` | Encryption/decryption failed | No |
| `INTERNAL_ERROR` | Unexpected engine error | No |

### Per-Runtime Behavior

| Runtime | How errors manifest | How scripts catch them |
|---------|--------------------|-----------------------|
| **Node.js** | Promise rejection with `BridgeError` object | `try/catch` or `.catch()` |
| **Python** | Raises `BridgeError` exception | `try/except BridgeError as e:` |
| **goja** (JS embedded) | Throws error object (goja supports try/catch) | `try/catch` |
| **yaegi** (Go embedded) | Returns `(result, *BridgeError)` tuple | `if err != nil { ... }` |

### Unhandled Errors

If a script throws/raises an error that is NOT caught:
1. Engine catches it
2. Logs with full context (module, script, user, error, stack trace)
3. Process step marked as failed
4. If inside `bitcode.tx()` → transaction rolled back
5. If inside `bitcode.call()` → error propagated to caller

### Go Type

```go
type BridgeError struct {
    Code      string         `json:"code"`
    Message   string         `json:"message"`
    Details   map[string]any `json:"details,omitempty"`
    Retryable bool           `json:"retryable"`
}

func (e *BridgeError) Error() string { return e.Message }
```

---

## 9. Security Rules

### 7.1 Environment Access

```json
{
  "env_allow": ["STRIPE_*", "SMTP_*", "CRM_*"],
  "env_deny": ["DB_PASSWORD", "JWT_SECRET", "ENCRYPTION_KEY"]
}
```

Rules:
1. `env_deny` always wins over `env_allow`
2. Engine secrets always denied: `JWT_SECRET`, `DB_PASSWORD`, `ENCRYPTION_KEY`, `SMTP_PASSWORD`, `STORAGE_S3_SECRET_KEY`
3. No config → module can only access own prefix (`CRM_*` for module `crm`)
4. Wildcard `*` supported
5. `env_allow: ["*"]` allows everything except engine secrets

### 7.2 Exec Whitelist

```json
{
  "exec_allow": ["pandoc", "wkhtmltopdf", "ffmpeg", "chromium"],
  "exec_deny": ["rm", "shutdown", "reboot"]
}
```

Rules:
1. No `exec_allow` → `bitcode.exec()` disabled for module
2. `exec_deny` always wins
3. Engine global deny: `rm`, `rmdir`, `del`, `format`, `shutdown`, `reboot`, `halt`, `poweroff`, `dd`, `mkfs`, `fdisk`
4. Matched by basename: `pandoc` matches `/usr/bin/pandoc`

### 7.3 Filesystem Sandbox

```json
{
  "fs_allow": ["/tmp", "/data/exports"],
  "fs_deny": ["/etc", "/root"]
}
```

Rules:
1. Relative paths → module directory (always allowed)
2. Absolute paths → require `fs_allow`
3. Engine dirs always denied: `internal/`, `plugins/`, `cmd/`
4. No config → module dir + `/tmp` only

### 7.4 Sudo Permission

```json
{
  "sudo_allow": true
}
```

Default: `false`. Only trusted modules (like `setting`) should enable this.

---

## 10. Go Interface Design

### 8.1 Core Context

```go
package bridge

type Context struct {
    txManager TxManager
    model     ModelFactory
    db        DB
    http     HTTPClient
    cache    Cache
    fs       FS
    session  Session
    config   ConfigReader
    env      EnvReader
    emitter  EventEmitter
    caller   ProcessCaller
    execer   CommandExecutor
    logger   Logger
    email    EmailSender
    notify   Notifier
    storage  Storage
    i18n     I18N
    security SecurityChecker
    audit     AuditLogger
    crypto    Crypto
    execution ExecutionLog
}

// Transaction
func (c *Context) Tx(fn func(tx *Context) error) error { return c.txManager.RunTx(c, fn) }

// Core
func (c *Context) Model(name string) ModelHandle  { return c.model.Model(name, c.session, false) }
func (c *Context) DB() DB                         { return c.db }
func (c *Context) HTTP() HTTPClient               { return c.http }
func (c *Context) Cache() Cache                   { return c.cache }
func (c *Context) FS() FS                         { return c.fs }
func (c *Context) Session() Session               { return c.session }
func (c *Context) Config(key string) any           { return c.config.Get(key) }
func (c *Context) Env(key string) string           { return c.env.Get(key) }
func (c *Context) Emit(event string, data map[string]any) error { return c.emitter.Emit(event, data) }
func (c *Context) Call(process string, input map[string]any) (any, error) { return c.caller.Call(process, input) }
func (c *Context) Exec(cmd string, args []string, opts *ExecOptions) (*ExecResult, error) { return c.execer.Exec(cmd, args, opts) }
func (c *Context) Log(level, msg string, data ...map[string]any) { c.logger.Log(level, msg, data...) }

// Extended (7)
func (c *Context) Email() EmailSender             { return c.email }
func (c *Context) Notify() Notifier               { return c.notify }
func (c *Context) Storage() Storage                { return c.storage }
func (c *Context) T(key string) string             { return c.i18n.Translate(c.session.Locale, key) }
func (c *Context) Security() SecurityChecker       { return c.security }
func (c *Context) Audit() AuditLogger              { return c.audit }
func (c *Context) Crypto() Crypto                  { return c.crypto }

// Execution Log (#20)
func (c *Context) Execution() ExecutionLog         { return c.execution }
```

### 8.2 Model (with sudo support)

```go
type ModelFactory interface {
    Model(name string, session Session, sudo bool) ModelHandle
}

type ModelHandle interface {
    // Single record CRUD
    Search(opts SearchOptions) ([]map[string]any, error)
    Get(id string, opts ...GetOptions) (map[string]any, error)
    Create(data map[string]any) (map[string]any, error)
    Write(id string, data map[string]any) error
    Delete(id string) error
    Count(opts SearchOptions) (int64, error)
    Sum(field string, opts SearchOptions) (float64, error)
    Upsert(data map[string]any, uniqueFields []string) (map[string]any, error)

    // Bulk operations
    CreateMany(records []map[string]any) ([]map[string]any, error)
    WriteMany(ids []string, data map[string]any) (*BulkResult, error)
    DeleteMany(ids []string) (*BulkResult, error)
    UpsertMany(records []map[string]any, uniqueFields []string) ([]map[string]any, error)

    // Relation operations (many2many)
    AddRelation(id string, field string, relatedIDs []string) error
    RemoveRelation(id string, field string, relatedIDs []string) error
    SetRelation(id string, field string, relatedIDs []string) error  // replace all
    LoadRelation(id string, field string) ([]map[string]any, error)

    // Mode switching
    Sudo() SudoModelHandle
}

type SudoModelHandle interface {
    ModelHandle  // inherits all methods (but without permission checks)

    // Sudo-only methods
    HardDelete(id string) error
    HardDeleteMany(ids []string) (*BulkResult, error)
    WithTenant(tenantId string) SudoModelHandle
    SkipValidation() SudoModelHandle
}

type SearchOptions struct {
    Domain  [][]any  `json:"domain,omitempty"`
    Fields  []string `json:"fields,omitempty"`
    Order   string   `json:"order,omitempty"`
    Limit   int      `json:"limit,omitempty"`   // default: 100, max: 10000
    Offset  int      `json:"offset,omitempty"`
    Include []string `json:"include,omitempty"` // eager load relations
}

type GetOptions struct {
    Include []string `json:"include,omitempty"` // eager load relations
}

type BulkResult struct {
    Affected int64 `json:"affected"`
}
```

### 8.3 HTTP (tls-client)

```go
type HTTPClient interface {
    Get(url string, opts *HTTPOptions) (*HTTPResponse, error)
    Post(url string, opts *HTTPOptions) (*HTTPResponse, error)
    Put(url string, opts *HTTPOptions) (*HTTPResponse, error)
    Patch(url string, opts *HTTPOptions) (*HTTPResponse, error)
    Delete(url string, opts *HTTPOptions) (*HTTPResponse, error)
}

type HTTPOptions struct {
    Headers        map[string]string `json:"headers,omitempty"`
    HeaderOrder    []string          `json:"headerOrder,omitempty"`
    Body           any               `json:"body,omitempty"`
    Timeout        int               `json:"timeout,omitempty"`        // ms
    Profile        string            `json:"profile,omitempty"`        // "chrome_131", "firefox_128", etc.
    Proxy          string            `json:"proxy,omitempty"`          // "socks5://user:pass@host:port"
    CookieJar      string            `json:"cookieJar,omitempty"`      // named jar for session persistence
    FollowRedirects *bool            `json:"followRedirects,omitempty"`
    InsecureSkipVerify bool          `json:"insecureSkipVerify,omitempty"`
}

type HTTPResponse struct {
    Status  int               `json:"status"`
    Headers map[string]string `json:"headers"`
    Body    any               `json:"body"`
}
```

### 8.4 Session (with context)

```go
type Session struct {
    UserID   string            `json:"userId"`
    Username string            `json:"username"`
    TenantID string            `json:"tenantId"`
    Groups   []string          `json:"groups"`
    Locale   string            `json:"locale"`
    Context  map[string]any    `json:"context"`  // module-set: company_id, branch_id, etc.
}
```

### 8.5 Email

```go
type EmailSender interface {
    Send(opts EmailOptions) error
}

type EmailOptions struct {
    To       string         `json:"to"`
    Subject  string         `json:"subject"`
    Body     string         `json:"body,omitempty"`      // raw HTML
    Template string         `json:"template,omitempty"`  // template name
    Data     map[string]any `json:"data,omitempty"`      // template data
}
```

### 8.6 Notify

```go
type Notifier interface {
    Send(opts NotifyOptions) error
    Broadcast(channel string, data map[string]any) error
}

type NotifyOptions struct {
    To      string `json:"to"`      // "user:john-123"
    Title   string `json:"title"`
    Message string `json:"message"`
    Type    string `json:"type"`    // success | warning | error | info
}
```

### 8.7 Storage

```go
type Storage interface {
    Upload(opts UploadOptions) (*Attachment, error)
    URL(id string) (string, error)
    Download(id string) ([]byte, error)
    Delete(id string) error
}

type UploadOptions struct {
    Filename string `json:"filename"`
    Content  []byte `json:"content"`
    Model    string `json:"model,omitempty"`
    RecordID string `json:"recordId,omitempty"`
}

type Attachment struct {
    ID          string `json:"id"`
    URL         string `json:"url"`
    Filename    string `json:"filename"`
    Size        int64  `json:"size"`
    ContentType string `json:"contentType"`
}
```

### 8.8 Security

```go
type SecurityChecker interface {
    Permissions(modelName string) (*ModelPermissions, error)
    HasGroup(groupName string) (bool, error)
    Groups() ([]string, error)
}

type ModelPermissions struct {
    CanRead   bool `json:"canRead"`
    CanWrite  bool `json:"canWrite"`
    CanCreate bool `json:"canCreate"`
    CanDelete bool `json:"canDelete"`
    CanPrint  bool `json:"canPrint"`
    CanEmail  bool `json:"canEmail"`
    CanExport bool `json:"canExport"`
    CanImport bool `json:"canImport"`
    CanClone  bool `json:"canClone"`
}
```

### 8.9 Audit, Crypto, DB, Cache, FS, Exec, Logger, Env, Config, Event, Process

```go
type AuditLogger interface {
    Log(opts AuditOptions) error
}
type AuditOptions struct {
    Action   string `json:"action"`
    Model    string `json:"model,omitempty"`
    RecordID string `json:"recordId,omitempty"`
    Detail   string `json:"detail,omitempty"`
}

type Crypto interface {
    Encrypt(plaintext string) (string, error)
    Decrypt(ciphertext string) (string, error)
    Hash(value string) (string, error)
    Verify(value, hash string) (bool, error)
}

type DB interface {
    Query(sql string, args ...any) ([]map[string]any, error)
    Execute(sql string, args ...any) (*ExecDBResult, error)
}
type ExecDBResult struct { RowsAffected int64 `json:"rows_affected"` }

type Cache interface {
    Get(key string) (any, error)
    Set(key string, value any, opts *CacheOptions) error
    Del(key string) error
}
type CacheOptions struct { TTL int `json:"ttl,omitempty"` }

type FS interface {
    Read(path string) (string, error)
    Write(path string, content string) error
    Exists(path string) (bool, error)
    List(path string) ([]string, error)
    Mkdir(path string) error
    Remove(path string) error
}

type CommandExecutor interface {
    Exec(cmd string, args []string, opts *ExecOptions) (*ExecResult, error)
}
type ExecOptions struct { Cwd string `json:"cwd,omitempty"`; Timeout int `json:"timeout,omitempty"` }
type ExecResult struct { Stdout string `json:"stdout"`; Stderr string `json:"stderr"`; ExitCode int `json:"exitCode"` }

type Logger interface { Log(level, msg string, data ...map[string]any) }
type EnvReader interface { Get(key string) string }
type ConfigReader interface { Get(key string) any }
type I18N interface { Translate(locale, key string) string }
type EventEmitter interface { Emit(event string, data map[string]any) error }
type ProcessCaller interface { Call(process string, input map[string]any) (any, error) }

type TxManager interface {
    RunTx(parent *Context, fn func(tx *Context) error) error
}

type ExecutionLog interface {
    Search(opts ExecutionSearchOptions) ([]map[string]any, error)
    Get(id string, opts ...GetOptions) (map[string]any, error)
    Current() *ExecutionInfo  // nil if not inside a process
    Retry(id string) (map[string]any, error)
    Cancel(id string) error
}

type ExecutionSearchOptions struct {
    Process string `json:"process,omitempty"`
    Status  string `json:"status,omitempty"`
    UserID  string `json:"userId,omitempty"`
    Limit   int    `json:"limit,omitempty"`
    Offset  int    `json:"offset,omitempty"`
    Order   string `json:"order,omitempty"`
}

type ExecutionInfo struct {
    ID          string `json:"id"`
    ProcessName string `json:"processName"`
    StartedAt   string `json:"startedAt"`
    ParentID    string `json:"parentId,omitempty"`
    StepIndex   int    `json:"stepIndex"`
    Trigger     string `json:"trigger"`
    Mode        string `json:"mode"`
}
```

### 8.10 Security Rules

```go
type SecurityRules struct {
    EnvAllow   []string `json:"env_allow,omitempty"`
    EnvDeny    []string `json:"env_deny,omitempty"`
    ExecAllow  []string `json:"exec_allow,omitempty"`
    ExecDeny   []string `json:"exec_deny,omitempty"`
    FSAllow    []string `json:"fs_allow,omitempty"`
    FSDeny     []string `json:"fs_deny,omitempty"`
    SudoAllow  bool     `json:"sudo_allow,omitempty"`
}

var EngineSecrets = []string{
    "JWT_SECRET", "DB_PASSWORD", "ENCRYPTION_KEY",
    "SMTP_PASSWORD", "STORAGE_S3_SECRET_KEY", "STORAGE_S3_ACCESS_KEY",
}

var DeniedCommands = []string{
    "rm", "rmdir", "del", "format", "shutdown", "reboot",
    "halt", "poweroff", "dd", "mkfs", "fdisk",
}
```

### 8.11 Factory

```go
type Factory struct {
    db            *gorm.DB
    modelRegistry ModelRegistry
    cache         CacheBackend
    processExec   ProcessExecutor
    eventBus      EventBus
    config        *viper.Viper
    emailSender   email.Sender
    wsHub         *websocket.Hub
    storageDriver storage.StorageDriver
    attachRepo    *storage.AttachmentRepository
    translator    *i18n.Translator
    permService   *persistence.PermissionService
    auditRepo     *persistence.AuditLogRepository
    encryptor     *security.FieldEncryptor
    tenantConfig  TenantConfig
    execLogConfig ExecutionLogConfig
}

func (f *Factory) NewContext(moduleName string, session Session, rules SecurityRules) *Context {
    return &Context{
        txManager: newTxManager(f.db),
        model:     newModelBridge(f.db, f.modelRegistry, session, f.tenantConfig),
        db:        newDBBridge(f.db),
        http:      newHTTPBridge(),  // tls-client instance
        cache:     newCacheBridge(f.cache),
        fs:        newFSBridge(moduleName, rules),
        session:   session,
        config:    newConfigBridge(f.config, moduleName),
        env:       newEnvBridge(f.config, rules),
        emitter:   newEventBridge(f.eventBus),
        caller:    newProcessBridge(f.processExec, session),
        execer:    newExecBridge(rules),
        logger:    newLogBridge(moduleName),
        email:     newEmailBridge(f.emailSender),
        notify:    newNotifyBridge(f.wsHub, session),
        storage:   newStorageBridge(f.storageDriver, f.attachRepo),
        i18n:      f.translator,
        security:  newSecurityBridge(f.permService, session),
        audit:     newAuditBridge(f.auditRepo, session, moduleName),
        crypto:    newCryptoBridge(f.encryptor),
        execution: newExecutionBridge(f.db, session),
    }
}
```

---

## 11. Edge Cases

### 9.1 Model Operations

| Edge Case | Behavior |
|-----------|----------|
| `bitcode.model("nonexistent").search({})` | Error: "model 'nonexistent' not found" |
| `bitcode.model("lead").get("nonexistent-id")` | Returns `null` (not error) |
| `bitcode.model("lead").write("nonexistent-id", {...})` | Error: "record not found" |
| `bitcode.model("lead").create({})` missing required fields | Error: "field 'name' is required" |
| `bitcode.model("lead").search({})` with record rules | Results filtered by user's record rules |
| `bitcode.model("lead").delete(id)` soft-delete model | Soft delete (set `deleted_at`) |
| `bitcode.model("lead").search({ limit: 10000 })` | Capped at engine max (1000) |
| Module `crm` accessing model from module `hrm` | Allowed — models are global. Record rules still apply. |
| Concurrent writes to same record | Last write wins (GORM default) |

### 9.2 sudo() Operations

| Edge Case | Behavior |
|-----------|----------|
| `sudo()` without `sudo_allow` | Error: "sudo not allowed for module 'crm'" |
| `sudo().hardDelete()` with FK constraint | DB error: "foreign key constraint" |
| `sudo().search()` in multi-tenant | Still filtered by current tenant |
| `sudo().withTenant("x").search()` | Filtered by tenant "x" |
| `sudo().create(data)` missing required field | Still validates. Use `skipValidation()` to bypass. |
| `sudo().withTenant("x")` on non-scoped model | `withTenant()` ignored |

### 9.3 HTTP (tls-client)

| Edge Case | Behavior |
|-----------|----------|
| URL unreachable | Error after timeout |
| Response not JSON | `body` contains raw string |
| Response > 10MB | Truncated with warning |
| Invalid proxy URL | Error: "invalid proxy URL" |
| Unknown profile name | Fallback to default (no TLS fingerprint) |
| Cookie jar "session-1" reused across requests | Cookies persist within same script execution |

### 9.4 Environment & Config

| Edge Case | Behavior |
|-----------|----------|
| `bitcode.env("NONEXISTENT")` | Returns `""` |
| `bitcode.env("DB_PASSWORD")` | Error: "access denied" |
| `bitcode.env("CRM_API_KEY")` from module `hrm` | Error: "access denied" |
| `bitcode.config("nonexistent")` | Returns `null` |

### 9.5 Email & Notify

| Edge Case | Behavior |
|-----------|----------|
| SMTP not configured | Error: "email not configured" |
| Template not found | Error: "template 'xxx' not found" |
| Notify to offline user | Message queued (if WebSocket reconnects) or dropped |
| Broadcast with no subscribers | No-op, no error |

### 9.6 Storage

| Edge Case | Behavior |
|-----------|----------|
| Upload > max size (config) | Error: "file exceeds max size" |
| Upload disallowed extension | Error: "extension '.exe' not allowed" |
| Download non-existent attachment | Error: "attachment not found" |
| S3 not configured, using local | Works — local storage is default |

### 9.7 Security & Audit

| Edge Case | Behavior |
|-----------|----------|
| `bitcode.security.permissions("nonexistent")` | Error: "model not found" |
| `bitcode.security.hasGroup("nonexistent.group")` | Returns `false` (not error) |
| `bitcode.audit.log({})` missing action | Error: "action required" |
| Cron script calling `bitcode.security.permissions()` | Returns full permissions (system user) |

### 9.8 Crypto

| Edge Case | Behavior |
|-----------|----------|
| `bitcode.crypto.encrypt()` without encryption key configured | Error: "encryption key not configured" |
| `bitcode.crypto.decrypt()` with wrong key | Error: "decryption failed" |
| `bitcode.crypto.verify()` with invalid hash | Returns `false` (not error) |

### 9.9 Cross-Runtime Consistency

| Edge Case | Behavior |
|-----------|----------|
| Same logic in JS vs Python vs Go | Same result. JSON serialization ensures consistency. |
| Async in goja (no event loop) | Bridge methods synchronous in goja. Engine handles async internally. |
| Goroutine in yaegi calling bridge | Thread-safe. Bridge methods use mutex where needed. |

---

## 12. Tech Debt — Future Additions

These are valuable features identified during design review but deferred from Phase 1. They are documented here so they are not forgotten.

### 10.1 `bitcode.queue` / `bitcode.job` — Background Jobs

```javascript
// Defer work to background
await bitcode.queue.push("process_invoice", { invoiceId: 123 }, {
  delay: "5m", retries: 3, priority: "high"
});

// Scheduled/recurring jobs
bitcode.job.schedule("daily_report", "0 8 * * *");
```

**Why deferred:** Requires job queue infrastructure (Redis-based or DB-based). Significant effort. Current workaround: use process engine's agent/cron system.

### 10.2 `bitcode.lock` — Distributed Locking

```javascript
await bitcode.lock("order:" + id, async () => {
  const order = await bitcode.model("order").get(id);
  await bitcode.model("order").write(id, { status: "shipped" });
}, { timeout: 5000 });
```

**Why deferred:** Requires distributed lock backend (Redis or DB advisory locks). Current workaround: use `bitcode.tx()` with database-level row locking.

### 10.3 `bitcode.parallel()` — Concurrent Execution

```javascript
const [api1, api2, api3] = await bitcode.parallel([
  () => bitcode.http.get("https://api1.com"),
  () => bitcode.http.get("https://api2.com"),
  () => bitcode.http.get("https://api3.com"),
]);
```

**Why deferred:** Trivial in Node.js (`Promise.all`), but needs goroutine-based implementation for goja/yaegi. Can be added per-runtime in Phase 2-5.

### 10.4 Rate Limiting / Resource Quotas

Per-module limits on expensive operations (max emails/hour, max HTTP calls/minute, max records created per execution).

**Why deferred:** Requires metrics collection infrastructure. Current workaround: engine-level rate limiting on API endpoints already exists.

### 10.5 Testing Framework / Mock Support

```javascript
import { createTestContext } from "bitcode/testing";
const ctx = createTestContext({ models: { lead: [{ id: 1, name: "Test" }] } });
const result = await myScript(ctx.bitcode, params);
```

**Why deferred:** Requires designing a mock layer for all 19 bridges. Important for ecosystem but not blocking for engine development.

### 10.6 Bridge API Versioning

Module manifest declaring required bridge version:
```json
{ "bridge_version": "1.x", "min_engine_version": "0.8.0" }
```

**Why deferred:** Bridge API is still being designed. Versioning makes sense after API stabilizes (post Phase 7).

### 10.7 Cursor-Based Pagination

```javascript
const cursor = bitcode.model("lead").cursor({ domain: [...], batchSize: 100 });
for await (const batch of cursor) {
  // process 100 records at a time, memory-safe
}
```

**Why deferred:** `search()` with `limit`/`offset` + default max limit (1000) is sufficient for most cases. Cursor pattern needed for data migration/export scripts processing millions of rows. Can be added to ModelHandle interface later without breaking changes.

### 10.8 Observability / Auto-Tracing

Automatic tracing of all bridge calls with execution timeline, duration, arguments, and call chain ID.

**Why deferred:** Requires tracing infrastructure (OpenTelemetry or custom). Current workaround: `bitcode.log()` + engine's structured logger. Can be added transparently in bridge layer without API changes.

---

## 13. Migration Path from Current API

### TypeScript

**Before:**
```typescript
import { definePlugin } from '@bitcode/sdk';
export default definePlugin({
  async execute(ctx, params) {
    const lead = params.input;
    console.log(`Deal won: ${lead.name}`);
    return { success: true };
  }
});
```

**After:**
```typescript
export default {
  async execute(bitcode, params) {
    const lead = params.input;
    await bitcode.model("activity").create({
      lead_id: lead.id, type: "task", summary: "Send welcome package"
    });
    await bitcode.email.send({
      to: "manager@company.com",
      subject: "Deal Won: " + lead.name,
      body: "<h1>Revenue: $" + lead.expected_revenue + "</h1>"
    });
    bitcode.log("info", "Deal won processed", { leadId: lead.id });
    return { success: true };
  }
};
```

### Python

**Before:**
```python
def execute(params):
    leads = params.get("leads", [])
    return {"total": len(leads)}
```

**After:**
```python
def execute(bitcode, params):
    leads = bitcode.model("lead").search({"domain": [["status", "=", "new"]]})
    bitcode.log("info", f"Found {len(leads)} new leads")
    return {"total": len(leads)}
```

### Backward Compatibility

During transition, support both signatures:
```python
if has_two_params(execute_func):
    result = execute(bitcode_ctx, params)   # new
else:
    result = execute(params)                 # legacy
```

---

## 14. Implementation Tasks

### Files to Create

```
engine/internal/runtime/bridge/
├── context.go          # Context struct + 20 accessors + Tx()
├── model.go            # ModelHandle + SudoModelHandle (CRUD + bulk + relations)
├── db.go               # DB interface + implementation
├── http.go             # HTTPClient using tls-client
├── cache.go            # Cache wrapper
├── fs.go               # FS with sandboxing
├── session.go          # Session struct with Context map
├── env.go              # EnvReader with whitelist/blacklist
├── config.go           # ConfigReader wrapper
├── event.go            # EventEmitter wrapper
├── process.go          # ProcessCaller wrapper
├── exec.go             # CommandExecutor with whitelist
├── logger.go           # Logger implementation
├── email.go            # EmailSender wrapper
├── notify.go           # Notifier via WebSocket hub
├── storage.go          # Storage wrapper (Local + S3)
├── i18n.go             # I18N wrapper
├── security.go         # SecurityChecker + SecurityRules + EngineSecrets
├── audit.go            # AuditLogger wrapper
├── crypto.go           # Crypto wrapper (encrypt/decrypt/hash/verify)
├── execution.go        # ExecutionLog (search, get, current, retry, cancel)
├── tx.go               # TxManager (transaction support)
├── errors.go           # BridgeError type + error codes
├── factory.go          # Factory to wire all 20 bridges
└── bridge_test.go      # Tests

engine/embedded/modules/base/models/
├── process_execution.json       # Execution log model
└── process_execution_step.json  # Execution step model
```

### Files to Modify

```
engine/go.mod
  → Add: github.com/bogdanfinn/tls-client

engine/internal/compiler/parser/module.go
  → Add SecurityRules fields: EnvAllow, EnvDeny, ExecAllow, ExecDeny, FSAllow, FSDeny, SudoAllow

engine/internal/runtime/executor/executor.go
  → Add bridge.Factory reference
  → Wrap Execute() with execution log recording (insert process_execution, per-step logging)

engine/internal/runtime/executor/steps/script.go
  → ScriptRunner.Run(): add bridge.Context parameter

engine/internal/infrastructure/persistence/repository.go
  → Add BulkUpdate(), BulkDelete(), BulkUpsert() methods

engine/internal/config.go
  → Add ExecutionLogConfig (retention settings)
```

### Task Breakdown

| # | Task | Effort | Priority |
|---|------|--------|----------|
| 1 | Create `bridge/` package with all 20 interfaces + BridgeError | Medium | Must |
| 2 | Implement `ModelHandle` + `SudoModelHandle` (single CRUD) | Large | Must |
| 3 | Implement bulk ops: `CreateMany`, `WriteMany`, `DeleteMany`, `UpsertMany` | Medium | Must |
| 4 | Implement relation ops: `AddRelation`, `RemoveRelation`, `SetRelation`, `LoadRelation` | Medium | Must |
| 5 | Implement eager loading: `include` in `Search()` and `Get()` | Medium | Must |
| 6 | Implement `TxManager` wrapping GORM Transaction | Medium | Must |
| 7 | Implement `DB` with parameterized query enforcement | Small | Must |
| 8 | Implement `HTTPClient` using `bogdanfinn/tls-client` | Medium | Must |
| 9 | Implement `Cache` wrapping existing cache | Small | Must |
| 10 | Implement `FS` with sandboxing | Medium | Must |
| 11 | Implement `EnvReader` with whitelist/blacklist | Small | Must |
| 12 | Implement `ConfigReader` wrapping viper | Small | Must |
| 13 | Implement `EventEmitter` wrapping EventBus | Small | Must |
| 14 | Implement `ProcessCaller` wrapping Executor | Small | Must |
| 15 | Implement `CommandExecutor` with whitelist + timeout | Medium | Must |
| 16 | Implement `Logger` with structured output | Small | Must |
| 17 | Implement `EmailSender` wrapping email.Sender | Small | Must |
| 18 | Implement `Notifier` wrapping WebSocket Hub | Small | Must |
| 19 | Implement `Storage` wrapping StorageDriver + AttachmentRepo | Medium | Must |
| 20 | Implement `I18N` wrapping Translator | Small | Must |
| 21 | Implement `SecurityChecker` wrapping PermissionService | Small | Must |
| 22 | Implement `AuditLogger` wrapping AuditLogRepository | Small | Must |
| 23 | Implement `Crypto` wrapping FieldEncryptor + password | Small | Must |
| 24 | Implement `ExecutionLog` bridge (search, get, current, retry, cancel) | Medium | Must |
| 25 | Create `process_execution.json` model in base module | Small | Must |
| 26 | Create `process_execution_step.json` model in base module | Small | Must |
| 27 | Wrap `executor.Execute()` with execution log recording | Large | Must |
| 28 | Add per-step input/output capture in executor | Medium | Must |
| 29 | Add `ExecutionLogConfig` to config.go (retention settings) | Small | Must |
| 30 | Implement execution log cleanup (cron-based, max_age + max_records) | Medium | Must |
| 31 | Implement `BridgeError` type with error codes | Small | Must |
| 32 | Add `BulkUpdate`, `BulkDelete`, `BulkUpsert` to GenericRepository | Medium | Must |
| 33 | Implement `Factory` to wire all 20 bridges | Medium | Must |
| 34 | Add `SecurityRules` + `SudoAllow` to parser.ModuleDefinition | Small | Must |
| 35 | Add `tls-client` to go.mod | Small | Must |
| 36 | Tests: security rules (env deny, exec deny, fs sandbox, sudo deny) | Medium | Must |
| 37 | Tests: model CRUD through bridge (user mode + sudo mode) | Medium | Must |
| 38 | Tests: bulk ops (createMany, writeMany, deleteMany) | Medium | Must |
| 39 | Tests: relation ops (addRelation, removeRelation, setRelation, loadRelation) | Medium | Must |
| 40 | Tests: transactions (commit, rollback, nested, timeout) | Medium | Must |
| 41 | Tests: sudo edge cases (hardDelete, withTenant, skipValidation) | Medium | Must |
| 42 | Tests: execution log (recording, retention, cleanup) | Medium | Must |
| 43 | Tests: error contract (BridgeError codes across operations) | Medium | Must |
| 44 | Tests: HTTP with tls-client (proxy, profile, cookie jar) | Medium | Should |
| 45 | Tests: email, notify, storage, crypto | Medium | Should |
