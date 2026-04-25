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
| `TENANT_STRATEGY` | `header` | How to detect tenant |
| `TENANT_HEADER` | `X-Tenant-ID` | Header name (for header strategy) |

## Strategies

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
2. **Repository** automatically adds `WHERE tenant_id = ?` to all queries
3. **Create** automatically sets `tenant_id` on new records
4. **WebSocket** can scope events to tenant via `tenant_id` query param

## Data Model

When multi-tenancy is enabled, every table gets an implicit `tenant_id` column. The engine handles this transparently — no changes needed in your JSON definitions.

## Tenant Isolation

- Tenant A cannot see Tenant B's data
- Record rules still apply within a tenant
- Admin users can see all tenants (if configured)
