# Phase 1.5: Multi-Tenancy Architecture

**Date**: 14 July 2026
**Status**: Draft
**Depends on**: Phase 1 (bridge API design — for `withTenant()` and `session.tenantId` contract)
**Unlocks**: Phase 2-7 (all runtime implementations need correct tenant behavior)
**Master doc**: `2026-07-14-runtime-engine-redesign-master.md`

---

## Table of Contents

1. [Problem Statement](#1-problem-statement)
2. [Concepts: Tenant vs Company](#2-concepts-tenant-vs-company)
3. [Scope: What We Build vs Tech Debt](#3-scope-what-we-build-vs-tech-debt)
4. [Strategy: shared_table](#4-strategy-shared_table)
5. [Configuration Design](#5-configuration-design)
6. [Model-Level Control](#6-model-level-control)
7. [Engine Implementation](#7-engine-implementation)
8. [Bridge API Integration](#8-bridge-api-integration)
9. [Edge Cases](#9-edge-cases)
10. [Tech Debt: Future Strategies](#10-tech-debt-future-strategies)
11. [Implementation Tasks](#11-implementation-tasks)

---

## 1. Problem Statement

### Current State

Engine has multi-tenancy support but it's **incomplete**:

```
✅ Middleware: extracts tenant_id from header/subdomain/path (tenant.go)
✅ Repository: auto-filters WHERE tenant_id = ? (repository.go)
✅ Repository: auto-sets tenant_id on create (repository.go)
✅ MongoDB: same filtering (mongo_repository.go)
✅ Config: tenant.enabled, tenant.strategy, tenant.header (bitcode.yaml)

❌ MigrateModel(): does NOT auto-add tenant_id column to tables
❌ No model-level control (tenant_scoped: true/false)
❌ Repository filters even non-tenant models when tenantID is set
❌ No PostgreSQL RLS support
❌ No schema-per-tenant or DB-per-tenant strategy
```

**Critical bug**: if `tenant.enabled = true`, repository tries `WHERE tenant_id = ?` but `MigrateModel()` never creates the column → SQL error.

### Target Use Cases (from real usage)

| Use Case | Tenant Config | How It Works |
|----------|--------------|--------------|
| **Single tenancy** (on-premise ERP) | `tenant.enabled: false` | No tenant overhead. Company isolation via record rules in module. |
| **SaaS medium** (multi-tenant platform) | `tenant.enabled: true`, `strategy: shared_table` | All tenants share one DB. `tenant_id` column auto-added. Filtered automatically. |
| **ERP** (single tenant + multi-company) | `tenant.enabled: false` | Company model in module. Record rules filter by `{{user.company_id}}`. |

---

## 2. Concepts: Tenant vs Company

Two different isolation levels. Engine handles tenant. Module handles company.

```
TENANT (infrastructure isolation — engine core)
  "Who OWNS this instance?"
  → Enforced by engine automatically
  → Transparent to scripts
  → Cross-tenant only via sudo().withTenant()

COMPANY (business isolation — module level)
  "Which company does this user work for?"
  → Enforced by record rules in model JSON
  → Module defines company model
  → Engine provides mechanism (record rules + session.context)
  → Engine does NOT hardcode "company" concept
```

### Why Company Is Not in Engine Core

Not all use cases need "company":
- **CMS**: no company concept
- **Project management**: uses "workspace"
- **E-commerce**: uses "store"
- **ERP**: uses "company"

Engine provides the **mechanism** (record rules + `session.context`), module provides the **policy**.

### How Company Works (via existing record rules)

```json
{
  "name": "invoice",
  "fields": {
    "company_id": { "type": "many2one", "model": "company", "required": true }
  },
  "record_rules": [
    { "groups": ["base.user"], "domain": [["company_id", "=", "{{user.company_id}}"]] },
    { "groups": ["base.admin"], "domain": [] }
  ]
}
```

No engine changes needed. `session.context.company_id` is set by module at login.

---

## 3. Scope: What We Build vs Tech Debt

### ✅ Build Now (Phase 1.5)

| Feature | Why |
|---------|-----|
| **`shared_table` strategy** | Most common. Already 80% implemented. Just need auto-column + model-level control. |
| **`tenant_scoped` in model JSON** | Essential — not all models need tenant isolation. |
| **Auto-add `tenant_id` column** in `MigrateModel()` | Fix the critical bug. |
| **Auto-add index** on `tenant_id` | Performance. |
| **Conditional filter** in repository (respect `tenant_scoped`) | Fix: currently filters ALL models when tenantID is set. |
| **ALTER TABLE** for existing tables | Handle enable/disable tenant toggle. |

### 🔮 Tech Debt (Future — Not Phase 1.5)

| Feature | Why Deferred | Stub/Note |
|---------|-------------|-----------|
| **`shared_schema` strategy** (PostgreSQL) | PostgreSQL-only. Complex schema management. Not needed for SQLite/MySQL/MongoDB. | Config accepts `strategy: "shared_schema"` but returns error: "shared_schema strategy not yet implemented, use shared_table" |
| **`separate_db` strategy** | Complex connection pooling. Rare use case. | Config accepts `strategy: "separate_db"` but returns error: "separate_db strategy not yet implemented, use shared_table" |
| **PostgreSQL RLS** (native `CREATE POLICY`) | Requires PostgreSQL-specific SQL generation, session variable management (`set_config`), and different behavior per DB driver. Significant effort for marginal benefit over app-layer filtering. | Document as future enhancement. App-layer filtering is sufficient and works across all DB drivers (SQLite, PostgreSQL, MySQL, MongoDB). |
| **Hybrid strategy** (small tenants shared, large tenants dedicated) | Requires tenant migration tooling, dual-mode routing. Very complex. | Document as future enhancement. |
| **Tenant lifecycle management** (create/suspend/delete tenant via API) | Useful but not blocking. Can be done manually or via module. | Document API shape but don't implement. Module `setting` can add this later. |

### Why This Scope Is Sufficient

1. **`shared_table` covers 90% of SaaS use cases** — it's what most multi-tenant apps use (Shopify, Salesforce multi-tenant mode, most B2B SaaS)
2. **App-layer filtering works across ALL databases** — SQLite, PostgreSQL, MySQL, MongoDB. No DB-specific code needed.
3. **The critical bug gets fixed** — `tenant_id` column auto-created, models can opt out
4. **Bridge API is correct from day one** — `withTenant()` works for shared_table now, and the interface is ready for other strategies later

---

## 4. Strategy: shared_table

### How It Works

```
Database: bitcode_db
Table: leads
┌────┬────────────┬────────┐
│ id │ tenant_id  │ name   │
├────┼────────────┼────────┤
│ 1  │ maju-jaya  │ Deal A │
│ 2  │ berkah     │ Deal B │
│ 3  │ maju-jaya  │ Deal C │
└────┴────────────┴────────┘

Query: SELECT * FROM leads WHERE tenant_id = 'maju-jaya'
Create: INSERT INTO leads (tenant_id, name) VALUES ('maju-jaya', 'Deal D')
```

### What Engine Does Automatically

1. **Migration**: auto-add `tenant_id` column + index to tenant-scoped tables
2. **Query**: auto-add `WHERE tenant_id = ?` to all queries on tenant-scoped models
3. **Create**: auto-set `tenant_id` on all new records in tenant-scoped models
4. **Detection**: extract tenant from request (header / subdomain / path)

### What Engine Does NOT Do (by design)

- Does not create separate schemas or databases
- Does not use PostgreSQL RLS policies
- Does not manage tenant lifecycle (create/delete tenant)
- Tenant registry is a regular model, not engine-managed

---

## 5. Configuration Design

All in `bitcode.yaml`:

```yaml
# Disabled (default) — single tenant, zero overhead
tenant:
  enabled: false
```

```yaml
# SaaS with shared table
tenant:
  enabled: true
  strategy: shared_table          # only supported strategy for now
  column: tenant_id               # column name, default "tenant_id"
  detection: subdomain            # header | subdomain | path
  header: X-Tenant-ID            # if detection = header
```

### Minimal SaaS Config

```yaml
tenant:
  enabled: true
  detection: subdomain
```

That's it. Strategy defaults to `shared_table`. Column defaults to `tenant_id`. Everything else is automatic.

### Stub for Future Strategies

```yaml
# These are accepted by config parser but return clear error at startup:
tenant:
  enabled: true
  strategy: shared_schema    # → error: "shared_schema not yet implemented, use shared_table"
  strategy: separate_db      # → error: "separate_db not yet implemented, use shared_table"
```

---

## 6. Model-Level Control

### `tenant_scoped` in Model JSON

Following the same pattern as `timestamps`, `soft_deletes`, `version`:

```json
{
  "name": "lead",
  "tenant_scoped": true,
  "fields": { ... }
}
```

```json
{
  "name": "plan",
  "tenant_scoped": false,
  "fields": { ... }
}
```

### Rules

| Condition | `tenant_scoped` default |
|-----------|------------------------|
| `tenant.enabled = false` | Ignored entirely. No column, no filter. |
| `tenant.enabled = true`, field omitted | `true` (all models scoped by default) |
| `tenant.enabled = true`, field = `false` | Not scoped. No column, no filter. Shared across tenants. |

### Go Implementation

```go
// Add to ModelDefinition (same pattern as existing options):
type ModelDefinition struct {
    // ... existing fields ...
    TenantScoped  *bool `json:"tenant_scoped,omitempty"`
}

func (m *ModelDefinition) IsTenantScoped() bool {
    if m.TenantScoped == nil {
        return true  // default: scoped when tenant enabled
    }
    return *m.TenantScoped
}
```

### Examples

| Model | `tenant_scoped` | Why |
|-------|----------------|-----|
| `lead`, `contact`, `invoice`, `employee` | `true` (default) | Business data — per tenant |
| `user`, `group` | `true` (default) | Users belong to a tenant |
| `audit_log` | `true` (default) | Audit per tenant |
| `tenant` (registry) | `false` | The tenant list itself is platform-level |
| `plan`, `subscription` | `false` | Billing/pricing is platform-level |
| `global_setting` | `false` | Platform configuration |

---

## 7. Engine Implementation

### 7.1 MigrateModel() — Auto-add tenant_id Column

In `dynamic_model.go`, inside `buildColumns()`, after soft_deletes columns:

```go
// After soft_deletes block, before field columns:
if tenantEnabled && model.IsTenantScoped() {
    switch dialect {
    case DialectPostgres:
        cols += ",\n  tenant_id VARCHAR(100) NOT NULL DEFAULT ''"
    case DialectMySQL:
        cols += ",\n  tenant_id VARCHAR(100) NOT NULL DEFAULT ''"
    default: // SQLite
        cols += ",\n  tenant_id TEXT NOT NULL DEFAULT ''"
    }
}
```

### 7.2 MigrateModel() — Auto-add Index

```go
if tenantEnabled && model.IsTenantScoped() {
    idxName := fmt.Sprintf("idx_%s_tenant_id", tableName)
    sql := fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s (tenant_id)", idxName, tableName)
    db.Exec(sql)
}
```

### 7.3 MigrateModel() — ALTER TABLE for Existing Tables

When tenant is enabled on an existing database:

```go
if tenantEnabled && model.IsTenantScoped() {
    if db.Migrator().HasTable(tableName) && !db.Migrator().HasColumn(tableName, "tenant_id") {
        // Table exists but no tenant_id column — add it
        var alterSQL string
        switch dialect {
        case DialectPostgres:
            alterSQL = fmt.Sprintf("ALTER TABLE %s ADD COLUMN tenant_id VARCHAR(100) NOT NULL DEFAULT ''", tableName)
        case DialectMySQL:
            alterSQL = fmt.Sprintf("ALTER TABLE %s ADD COLUMN tenant_id VARCHAR(100) NOT NULL DEFAULT ''", tableName)
        default:
            alterSQL = fmt.Sprintf("ALTER TABLE %s ADD COLUMN tenant_id TEXT NOT NULL DEFAULT ''", tableName)
        }
        db.Exec(alterSQL)
        // Also add index
        idxName := fmt.Sprintf("idx_%s_tenant_id", tableName)
        db.Exec(fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s (tenant_id)", idxName, tableName))
    }
}
```

### 7.4 Repository — Conditional Tenant Filter

Current code (repository.go line 189-190):
```go
// CURRENT — always filters when tenantID is set
if r.tenantID != "" {
    return query.Where("tenant_id = ?", r.tenantID)
}
```

Fixed code:
```go
// FIXED — only filter tenant-scoped models
if r.tenantID != "" && r.modelDef != nil && r.modelDef.IsTenantScoped() {
    return query.Where("tenant_id = ?", r.tenantID)
}
```

Same fix for create (line 211-212):
```go
// FIXED — only set tenant_id for tenant-scoped models
if r.tenantID != "" && r.modelDef != nil && r.modelDef.IsTenantScoped() {
    record["tenant_id"] = r.tenantID
}
```

### 7.5 MongoDB — Same Fixes

Same conditional logic in `mongo_repository.go`:
```go
// Only add tenant_id filter for tenant-scoped models
if r.tenantID != "" && r.modelDef != nil && r.modelDef.IsTenantScoped() {
    filter["tenant_id"] = r.tenantID
}
```

### 7.6 Config Validation at Startup

```go
func validateTenantConfig(cfg TenantConfig) error {
    if !cfg.Enabled {
        return nil
    }
    switch cfg.Strategy {
    case "shared_table", "":
        return nil // supported
    case "shared_schema":
        return fmt.Errorf("tenant strategy 'shared_schema' is not yet implemented. Use 'shared_table'. See docs/plans/2026-07-14-runtime-engine-phase-1.5-multi-tenancy.md §10 for roadmap.")
    case "separate_db":
        return fmt.Errorf("tenant strategy 'separate_db' is not yet implemented. Use 'shared_table'. See docs/plans/2026-07-14-runtime-engine-phase-1.5-multi-tenancy.md §10 for roadmap.")
    default:
        return fmt.Errorf("unknown tenant strategy: %s", cfg.Strategy)
    }
}
```

---

## 8. Bridge API Integration

### 8.1 `bitcode.session`

```javascript
bitcode.session.tenantId    // "maju-jaya" — from middleware
bitcode.session.context     // { company_id: "comp-1" } — set by module
```

### 8.2 `bitcode.model()` — Default Behavior

```javascript
// Tenant-scoped model: auto-filtered by current tenant
await bitcode.model("lead").search({});
// → WHERE tenant_id = 'maju-jaya' AND ...

// Non-scoped model: no tenant filter
await bitcode.model("plan").search({});
// → no WHERE tenant_id (plan.tenant_scoped = false)
```

### 8.3 `bitcode.model().sudo().withTenant()`

```javascript
// Sudo: bypass permission + record rules, still within current tenant
await bitcode.model("lead").sudo().search({});
// → WHERE tenant_id = 'maju-jaya' (tenant filter stays, only permissions bypassed)

// Cross-tenant: access another tenant's data
await bitcode.model("lead").sudo().withTenant("berkah").search({});
// → WHERE tenant_id = 'berkah'
```

**`withTenant()` implementation for shared_table**: simply changes the `tenant_id` value in the repository filter. When other strategies are implemented later, `withTenant()` will switch schema or DB connection instead — but the script API stays identical.

---

## 9. Edge Cases

### 9.1 Migration

| Edge Case | Behavior |
|-----------|----------|
| `tenant.enabled` changed from `false` to `true` | Auto-add `tenant_id` column via ALTER TABLE. Existing data gets `tenant_id = ''` (empty string). |
| `tenant.enabled` changed from `true` to `false` | Column stays (no destructive migration). Filter just stops being applied. |
| Model changes `tenant_scoped` from `true` to `false` | Column stays. Filter stops for this model. |
| Model changes `tenant_scoped` from `false` to `true` | Column added via ALTER TABLE if missing. Existing data gets `tenant_id = ''`. |
| New model added while tenant enabled | Column auto-created in CREATE TABLE. |

### 9.2 Query

| Edge Case | Behavior |
|-----------|----------|
| Query without tenant context (CLI, cron) | No tenant filter — returns all data. System context. |
| `tenant_scoped: false` model | No `WHERE tenant_id`, even if tenant context exists. |
| Existing data with `tenant_id = ''` after enabling tenant | Visible to all tenants (empty matches nothing specifically). Admin must backfill. |
| `bitcode.db.query()` (raw SQL) | No auto tenant filter. Raw SQL bypasses everything. Developer responsibility. |

### 9.3 Create

| Edge Case | Behavior |
|-----------|----------|
| Create without tenant context on scoped model | Error: "tenant_id required for model 'lead'" |
| Create with explicit `tenant_id` in data | Ignored — engine always overrides with session tenant. Prevents tenant spoofing. |
| `sudo().withTenant("x").create(data)` | Creates record with `tenant_id = 'x'`. |

### 9.4 MongoDB

| Edge Case | Behavior |
|-----------|----------|
| MongoDB with shared_table strategy | Same behavior — `tenant_id` field added to documents, filtered in queries. MongoDB doesn't have columns, so no ALTER TABLE needed. |
| MongoDB with shared_schema/separate_db | Tech debt — not implemented. Same error as SQL. |

---

## 10. Tech Debt: Future Strategies

### 10.1 `shared_schema` (PostgreSQL only)

**What it would do:**
- Each tenant gets a PostgreSQL schema: `tenant_maju_jaya`, `tenant_berkah`
- Per-request: `SET search_path TO 'tenant_xxx'`
- No `tenant_id` column needed — schema provides isolation

**Why deferred:**
- PostgreSQL-only (doesn't work with SQLite, MySQL, MongoDB)
- Requires schema creation/migration per tenant
- Requires per-request schema switching in GORM (doable but complex)

**Stub:** Config parser accepts `strategy: "shared_schema"` but engine returns clear error at startup with link to this doc.

**Estimated effort:** Medium (2-3 days)

### 10.2 `separate_db`

**What it would do:**
- Each tenant gets a separate database
- Connection pool per tenant
- Per-request: route to correct DB

**Why deferred:**
- Complex connection pool management
- Migration must run per DB
- Rare use case (enterprise only)

**Stub:** Config parser accepts `strategy: "separate_db"` but engine returns clear error at startup.

**Estimated effort:** Large (5-7 days)

### 10.3 PostgreSQL RLS (Native)

**What it would do:**
- `ALTER TABLE leads ENABLE ROW LEVEL SECURITY`
- `CREATE POLICY tenant_isolation ON leads USING (tenant_id = current_setting('app.tenant_id', true))`
- Per-request: `SELECT set_config('app.tenant_id', 'maju-jaya', true)`
- DB enforces isolation — even raw SQL can't leak data

**Why deferred:**
- PostgreSQL-only
- Requires generating SQL policies per model during migration
- Requires per-transaction session variable setup via GORM callbacks
- App-layer filtering already works and is cross-database
- Marginal security benefit for significant complexity

**When to implement:** When a customer requires DB-level isolation for compliance (SOC2, HIPAA, etc.)

**Estimated effort:** Medium-Large (3-5 days)

### 10.4 Hybrid (Shared → Dedicated Migration)

**What it would do:**
- Small tenants: shared_table
- Large tenants: separate_db
- Tooling to migrate tenant from shared to dedicated without downtime

**Why deferred:**
- Requires both strategies to be implemented first
- Complex routing logic (which tenant goes where)
- Migration tooling is significant effort

**When to implement:** When platform has 100+ tenants with varying sizes.

**Estimated effort:** Large (7-10 days)

---

## 11. Implementation Tasks

### Files to Modify

```
engine/internal/compiler/parser/model.go
  → Add TenantScoped *bool to ModelDefinition
  → Add IsTenantScoped() method

engine/internal/infrastructure/persistence/dynamic_model.go
  → buildColumns(): auto-add tenant_id when tenant enabled + model scoped
  → MigrateModel(): auto-add index, handle ALTER TABLE for existing tables
  → Pass tenant config to buildColumns (currently not available)

engine/internal/infrastructure/persistence/repository.go
  → Conditional tenant filter: only if model.IsTenantScoped()
  → Conditional tenant set on create: only if model.IsTenantScoped()

engine/internal/infrastructure/persistence/mongo_repository.go
  → Same conditional fixes as repository.go

engine/internal/config.go
  → Add Strategy, Column fields to TenantConfig
  → Add validateTenantConfig() with stub errors for unimplemented strategies

engine/internal/app.go
  → Pass tenant config to MigrateModel
  → Call validateTenantConfig at startup
```

### Task Breakdown

| # | Task | Effort | Priority |
|---|------|--------|----------|
| 1 | Add `TenantScoped` to `ModelDefinition` + `IsTenantScoped()` | Small | Must |
| 2 | Fix `buildColumns()`: auto-add `tenant_id` column | Small | Must |
| 3 | Fix `MigrateModel()`: auto-add tenant_id index | Small | Must |
| 4 | Fix `MigrateModel()`: ALTER TABLE for existing tables | Medium | Must |
| 5 | Fix `repository.go`: conditional tenant filter (respect IsTenantScoped) | Small | Must |
| 6 | Fix `repository.go`: conditional tenant set on create | Small | Must |
| 7 | Fix `mongo_repository.go`: same conditional fixes | Small | Must |
| 8 | Expand `TenantConfig`: strategy, column fields | Small | Must |
| 9 | Add `validateTenantConfig()` with stub errors for future strategies | Small | Must |
| 10 | Pass tenant config through to MigrateModel | Small | Must |
| 11 | Tests: auto-column creation (new table + ALTER TABLE) | Medium | Must |
| 12 | Tests: conditional filter (scoped vs non-scoped model) | Medium | Must |
| 13 | Tests: create with auto tenant_id (scoped vs non-scoped) | Small | Must |
| 14 | Tests: tenant enable/disable toggle on existing data | Medium | Must |
| 15 | Tests: MongoDB conditional filter | Small | Should |

**Total estimated effort: 3-4 days**

Most of the work is small fixes to existing code. The biggest task is ALTER TABLE handling (#4) and tests (#11-14).
