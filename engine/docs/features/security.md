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

## Auth Module

Embedded `auth` module provides login, register, forgot password, reset, and 2FA verification pages at `/app/auth/*`.

### Per-Module Auth Control

`module.json` supports an `auth` field:

```json
{ "auth": true }   // default — views require login
{ "auth": false }  // public — views accessible without login
```

If `auth` is not set, it defaults to `true` (require authentication).

### Auth Settings (Admin UI)

| Setting | Default | Description |
|---------|---------|-------------|
| `auth.register_enabled` | `false` | Show registration page |
| `auth.otp_enabled` | `false` | Enable 2FA on login |
| `auth.otp_channel` | `email` | OTP channel (email, whatsapp, telegram) |
| `auth.otp_type` | `code` | OTP type (code, magic_link) |

### Auth Routes

| Route | Method | Description |
|-------|--------|-------------|
| `/app/auth/login` | GET/POST | Login page |
| `/app/auth/register` | GET/POST | Registration (if enabled) |
| `/app/auth/forgot` | GET/POST | Forgot password (always available if SMTP configured) |
| `/app/auth/reset` | GET/POST | Reset password with OTP code |
| `/app/auth/verify-2fa` | GET/POST | 2FA verification |
| `/app/auth/logout` | GET | Logout (clear cookie) |

### i18n

All auth templates use `{{t .Locale "key"}}` for translations. 11 languages supported: en, id, ar, de, es, fr, ja, ko, pt-BR, ru, zh-CN.

## Middleware Chain

```
Request → IPWhitelist → RateLimit → Tenant → Auth → Permission → RecordRule → Audit → Handler
```

Each middleware can reject the request:
- IPWhitelist: 403 Access Denied (IP not in whitelist)
- RateLimit: 429 Too Many Requests (with `Retry-After` header)
- Auth: 401 Unauthorized
- Permission: 403 Forbidden
- RecordRule: silently filters data (no error, just fewer results)
- Audit: logs write operations (never rejects), includes `impersonated_by` if applicable

## Two-Factor Authentication (2FA)

Email OTP-based 2FA. When enabled for a user, login requires a 6-digit verification code sent to their email.

### Endpoints

```
POST /auth/2fa/enable     — Enable 2FA (requires auth)
POST /auth/2fa/disable    — Disable 2FA (requires auth)
POST /auth/2fa/validate   — Validate OTP code with temp token
```

### Login Flow with 2FA

```
POST /auth/login { "username": "admin", "password": "secret" }
→ If 2FA enabled: { "requires_2fa": true, "temp_token": "..." }
→ OTP sent to user's email

POST /auth/2fa/validate { "temp_token": "...", "code": "123456" }
→ Returns { "token": "eyJ..." }
```

### Configuration

Requires SMTP configuration (`smtp.*` in config). OTP stored in cache with 5-minute TTL, max 3 attempts.

## Rate Limiting

Fiber limiter middleware with tiered limits:

| Route | Max | Window |
|-------|-----|--------|
| `/auth/*` | 5 | 1 min |
| All other | 100 | 1 min |

Configurable via `rate_limit.*` config keys. Returns HTTP 429 with `Retry-After` header.

## Field-Level Encryption

AES-256-GCM encryption for sensitive fields. Mark fields with `"encrypted": true` in model JSON:

```json
{
  "fields": {
    "ssn": { "type": "string", "encrypted": true },
    "bank_account": { "type": "string", "encrypted": true }
  }
}
```

- Transparent encrypt-on-write / decrypt-on-read in GenericRepository
- Key versioning (`v1:` prefix) for future key rotation
- Requires `ENCRYPTION_KEY` env var (base64-encoded 32-byte key)
- Encrypted fields cannot be searched/filtered/sorted via SQL

## Admin Impersonation

Admins can impersonate other users for debugging/support:

```
POST /admin/api/impersonate/:user_id
→ Returns impersonation token (1h TTL)
→ Token has impersonated_by claim

POST /admin/api/stop-impersonate
→ Returns original admin token
```

Safety guards:
- Only users with `admin` role can impersonate
- Cannot impersonate other admin users
- Impersonation token expires in 1 hour
- All audit logs include `impersonated_by` field during impersonation

## Backup & Restore

CLI commands for database backup and restore:

```bash
bitcode db backup [output-path]     # Create backup
bitcode db backup --gzip            # Compressed backup
bitcode db restore [backup-path]    # Restore from backup
bitcode db restore --force          # Skip confirmation
```

Driver-aware: SQLite (file copy), PostgreSQL (pg_dump/psql), MySQL (mysqldump/mysql).

## Email Infrastructure

SMTP email sender for transactional emails (2FA codes, future notifications):

```
smtp.host=smtp.gmail.com
smtp.port=587
smtp.user=noreply@example.com
smtp.password=app-password
smtp.from=BitCode <noreply@example.com>
smtp.tls=true
```

HTML email templates with Go `html/template`. NoopSender fallback when SMTP not configured.

## IP Whitelist

Restrict access by IP address. Supports exact IPs and CIDR ranges. Can be applied globally or admin-only.

```
security.ip_whitelist_enabled=true
security.ip_whitelist=["192.168.1.0/24", "10.0.0.1", "203.0.113.50"]
security.ip_whitelist_admin_only=true
```

- `ip_whitelist_admin_only=true` (default): only `/admin/*` routes are restricted
- `ip_whitelist_admin_only=false`: all routes are restricted
- Supports both exact IP matching and CIDR notation
- Returns 403 with `"access denied: IP not allowed"` for blocked IPs

## Session Policy

Configurable JWT token duration and cookie security settings:

```
security.session_duration=24h       # JWT token lifetime (default: 24h)
security.cookie_secure=false        # Set true in production (HTTPS only)
security.cookie_samesite=Lax        # Lax, Strict, or None
```

- `session_duration` controls both JWT `exp` claim and cookie `MaxAge`
- `cookie_secure=true` ensures cookies are only sent over HTTPS
- `cookie_samesite` controls cross-site cookie behavior
- Impersonation tokens always use 1h duration regardless of this setting
