# APIs

APIs define HTTP endpoints. Use `auto_crud` for instant CRUD, or define custom endpoints.

## Auto-CRUD (Simplest)

```json
{
  "name": "customer_api",
  "model": "customer",
  "auto_crud": true,
  "auth": true
}
```

Generates:
- `GET /api/customers` — List (paginated)
- `GET /api/customers/:id` — Read one
- `POST /api/customers` — Create
- `PUT /api/customers/:id` — Update
- `DELETE /api/customers/:id` — Soft delete

`auth: true` enables JWT authentication + RBAC permissions + record rules (RLS). Permissions are auto-derived: `customer.read`, `customer.create`, `customer.write`, `customer.delete`.

## With Workflow Actions

```json
{
  "name": "order_api",
  "model": "order",
  "auto_crud": true,
  "auth": true,
  "workflow": "order_workflow",
  "actions": {
    "confirm":  { "transition": "confirm",  "permission": "order.confirm" },
    "complete": { "transition": "complete", "permission": "order.complete" }
  }
}
```

Generates CRUD endpoints **plus**:
- `POST /api/orders/:id/confirm`
- `POST /api/orders/:id/complete`

Each action validates the workflow transition, checks permission, and updates the status field.

## Custom Endpoints

```json
{
  "name": "report_api",
  "base_path": "/api/reports",
  "auth": true,
  "endpoints": [
    { "method": "GET", "path": "/sales-summary", "handler": "processes/sales_summary.json", "permissions": ["report.read"] }
  ]
}
```

## Public API (No Auth)

```json
{
  "name": "tag_api",
  "model": "tag",
  "auto_crud": true
}
```

No `auth` = no JWT, no permissions, no RLS. Good for public lookup tables.

## Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `auto_crud` | bool | false | Generate CRUD endpoints |
| `auth` | bool | false | JWT + permissions + RLS |
| `workflow` | string | - | Link to workflow definition |
| `actions` | object | - | Workflow action endpoints |
| `soft_delete` | bool | true | DELETE = soft delete |
| `pagination` | object | {page_size:20} | Pagination config |
| `search` | string[] | - | Fields for full-text search |
| `base_path` | string | /api/{model}s | Custom base path |

## Request Flow

```
POST /api/orders (create)
  → Auth (JWT validation)
  → Permission (order.create)
  → Validate input
  → Set initial workflow state
  → Create record
  → Return 201

GET /api/orders (list)
  → Auth (JWT)
  → Permission (order.read)
  → RLS (inject record_rules WHERE clause)
  → Query with pagination
  → Return 200

POST /api/orders/:id/confirm (workflow action)
  → Auth (JWT)
  → Permission (order.confirm)
  → RLS (can user see this record?)
  → Validate transition (draft → confirmed)
  → Run process if defined
  → Update status
  → Emit event
  → Return 200
```
