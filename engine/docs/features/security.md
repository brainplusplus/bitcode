# Security

## Overview

Security is built into the base module. Three layers:

1. **Authentication** — JWT tokens
2. **Authorization** — RBAC permissions
3. **Row-Level Security** — Record rules

All three activate automatically when `auth: true` on an API.

## Authentication (JWT)

```
POST /auth/login { "username": "admin", "password": "secret" }
→ Returns { "token": "eyJ..." }

GET /api/orders -H "Authorization: Bearer eyJ..."
→ Token validated, user context loaded
```

Claims in token: `user_id`, `username`, `roles`, `groups`.

## Permissions (RBAC)

Permissions follow the pattern `module.model.action`:

```
sales.order.read
sales.order.create
sales.order.write
sales.order.delete
sales.order.confirm
```

Permissions are assigned to roles. Roles support inheritance:

```
sales_manager
  ├── sales.order.read
  ├── sales.order.confirm
  └── inherits: sales_user
        ├── sales.order.read
        └── sales.order.create
```

When `auto_crud: true`, permissions are auto-derived from the model name.

## Groups

Groups organize users and link to record rules:

```
base.user
  └── sales.user (implies base.user)
        └── sales.manager (implies sales.user)
```

Group hierarchy is resolved recursively — a user in `sales.manager` is also in `sales.user` and `base.user`.

## Record Rules (Row-Level Security)

Defined on the model:

```json
"record_rules": [
  { "groups": ["sales.user"],    "domain": [["created_by", "=", "{{user.id}}"]] },
  { "groups": ["sales.manager"], "domain": [] }
]
```

- `sales.user`: sees only their own records
- `sales.manager`: sees all records (empty domain = no filter)

Record rules are injected as WHERE clauses into every query. No opt-in needed — if the model has record_rules and the API has `auth: true`, they're enforced.

### Domain Filter Syntax

```json
[["field", "operator", "value"]]
```

Operators: `=`, `!=`, `>`, `<`, `>=`, `<=`, `in`, `not in`, `like`

Variables: `{{user.id}}`, `{{user.tenant_id}}`

## Middleware Chain

```
Request → Auth → Permission → RecordRule → Audit → Handler
```

Each middleware can reject the request:
- Auth: 401 Unauthorized
- Permission: 403 Forbidden
- RecordRule: silently filters data (no error, just fewer results)
- Audit: logs write operations (never rejects)
