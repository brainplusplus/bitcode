# Multi-tenancy

Isolate data between tenants (companies, organizations) sharing the same engine instance.

## Enable

```bash
TENANT_ENABLED=true bitcode serve
```

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `TENANT_ENABLED` | `false` | Enable multi-tenancy |
| `TENANT_STRATEGY` | `header` | Detection method: `header`, `subdomain`, `path` |
| `TENANT_HEADER` | `X-Tenant-ID` | Header name (for header strategy) |
| `TENANT_ISOLATION` | `shared_table` | Isolation strategy (only `shared_table` supported currently) |
| `TENANT_COLUMN` | `tenant_id` | Column name for tenant identifier |

## Detection Strategies

### Header (default)

```bash
curl -H "X-Tenant-ID: company-a" http://localhost:8080/api/contacts
```

### Subdomain

```
company-a.app.example.com → tenant_id = "company-a"
company-b.app.example.com → tenant_id = "company-b"
```

### Path

```
/tenant/company-a/api/contacts → tenant_id = "company-a"
```

## How It Works

1. **Tenant middleware** extracts tenant_id from request (header/subdomain/path)
2. **MigrateModel** auto-adds `tenant_id` column + index to tenant-scoped tables
3. **Repository** automatically adds `WHERE tenant_id = ?` to all queries on tenant-scoped models
4. **Create** automatically sets `tenant_id` on new records for tenant-scoped models
5. **WebSocket** can scope events to tenant via `tenant_id` query param

## Model-Level Control (`tenant_scoped`)

Not all models need tenant isolation. Use `tenant_scoped` in model JSON:

```json
{
  "name": "lead",
  "tenant_scoped": true,
  "fields": { "name": { "type": "string" } }
}
```

| `tenant_scoped` value | Behavior |
|----------------------|----------|
| omitted (default) | `true` — model is tenant-scoped when tenant enabled |
| `true` | Explicitly scoped — `tenant_id` column, filtered queries |
| `false` | Shared across tenants — no column, no filter (e.g. plans, global settings) |

When `tenant.enabled = false`, `tenant_scoped` is ignored entirely.

## Bridge API

```javascript
bitcode.session.tenantId              // current tenant from middleware
bitcode.model("lead").search({})      // auto-filtered by tenant
bitcode.model("plan").search({})      // no filter (plan.tenant_scoped = false)

bitcode.model("lead").sudo().search({})                    // bypass permissions, still filtered by tenant
bitcode.model("lead").sudo().withTenant("other").search({}) // cross-tenant access
```

## ALTER TABLE Support

When tenant is enabled on an existing database, the engine automatically adds `tenant_id` column to existing tables via `ALTER TABLE`. Existing data gets `tenant_id = ''` (empty string).

When tenant is disabled, columns stay (no destructive migration) but filters stop being applied.

## Isolation Strategies

| Strategy | Status | Description |
|----------|--------|-------------|
| `shared_table` | ✅ Supported | All tenants share one DB, filtered by `tenant_id` column |
| `shared_schema` | 🔲 Planned | PostgreSQL-only, per-tenant schema |
| `separate_db` | 🔲 Planned | Per-tenant database with connection pooling |

Unsupported strategies return a clear error at startup with guidance to use `shared_table`.
