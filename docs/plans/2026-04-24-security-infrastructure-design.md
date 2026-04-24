# Security & Infrastructure — Design Document

**Date**: 2026-04-24
**Status**: Approved
**Scope**: 6 security features + audit log enhancement + email infrastructure

---

## 1. Overview

Implement the remaining Security & Infrastructure features from the roadmap (features.md #57–#61), plus admin user impersonation with audit trail. CSRF is deferred to the public web forms/website phase.

### Features to Implement

| # | Feature | Effort | Approach |
|---|---------|--------|----------|
| 57 | Two-Factor Auth (2FA) | M | Email OTP (6-digit code via SMTP) |
| 58 | Data Encryption | M | AES-256-GCM field-level encryption |
| 59 | Backup & Restore | M | CLI commands, driver-aware |
| 60 | Rate Limiting | S | Fiber built-in limiter middleware |
| — | Admin Impersonation | M | Token-based with audit trail |
| — | Email Infrastructure | M | SMTP sender (foundation for 2FA, notifications, etc.) |

### Deferred

| # | Feature | Reason |
|---|---------|--------|
| 61 | CSRF Protection | Needed for future public web forms & website (job portal, etc.). Not needed now — API uses JWT (stateless), SSR forms use HTTPOnly cookie. Will implement when public web forms feature is built. |

---

## 2. Email Infrastructure (Foundation)

New email sending system — required by 2FA and reusable for future notifications (#26), password reset, scheduled reports (#43).

### 2.1 Configuration

Env vars:

```
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=noreply@example.com
SMTP_PASSWORD=app-password
SMTP_FROM=BitCode <noreply@example.com>
SMTP_TLS=true
```

### 2.2 Email Sender Interface

```go
// engine/pkg/email/sender.go
type Sender interface {
    Send(to, subject, htmlBody string) error
}

type SMTPSender struct {
    Host     string
    Port     int
    User     string
    Password string
    From     string
    UseTLS   bool
}
```

Uses Go stdlib `net/smtp` + `crypto/tls`. No external dependency needed.

### 2.3 Email Templates

Simple Go `html/template` based templates stored in memory. Initial templates:
- `otp_code` — 2FA verification code email

### 2.4 Files

| File | Purpose |
|------|---------|
| `engine/pkg/email/sender.go` | SMTP sender implementation |
| `engine/pkg/email/templates.go` | Email HTML templates |
| `engine/internal/config.go` | Add SMTP config fields |

---

## 3. Two-Factor Auth (2FA) — Email OTP

### 3.1 Flow

```
User enables 2FA:
  POST /auth/2fa/enable (requires auth)
  → Sets totp_enabled=true on user record
  → Returns { "ok": true }

User disables 2FA:
  POST /auth/2fa/disable (requires auth + OTP verification)
  → Sends OTP to email, user submits code
  → Sets totp_enabled=false

Login with 2FA:
  POST /auth/login { username, password }
  → Password valid + 2FA enabled
  → Generate 6-digit OTP, store in cache (TTL 5min)
  → Send OTP to user's email
  → Return { "requires_2fa": true, "temp_token": "..." }

  POST /auth/2fa/validate { temp_token, code }
  → Validate temp_token (short-lived JWT, 10min expiry, no permissions)
  → Check OTP code against cache
  → Return full JWT token
```

### 3.2 OTP Storage

- Stored in app cache (memory or Redis) with key `otp:{user_id}`
- TTL: 5 minutes
- Value: `{ code: "123456", attempts: 0 }`
- Max 3 attempts, then OTP invalidated (must re-login)

### 3.3 Temp Token

Short-lived JWT with:
- `user_id` — to identify who's verifying
- `purpose: "2fa"` — to prevent use as regular auth token
- Expiry: 10 minutes
- No roles/groups/permissions — cannot access any API

### 3.4 Schema Change

Add to `users` model in base module:

```json
{
  "totp_enabled": { "type": "boolean", "default": false },
  "totp_verified_at": { "type": "datetime" }
}
```

Note: field name is `totp_enabled` for consistency even though we use email OTP — the field represents "2FA is enabled" regardless of method. Future TOTP support can reuse this flag.

### 3.5 Rate Limiting on OTP

- Max 3 OTP requests per user per 15 minutes (prevent email spam)
- Max 3 validation attempts per OTP (prevent brute force)

### 3.6 Files

| File | Purpose |
|------|---------|
| `engine/internal/presentation/api/auth_handler.go` | Add 2FA endpoints, modify login flow |
| `engine/pkg/security/otp.go` | OTP generation (crypto/rand 6-digit) |
| `engine/internal/app.go` | Wire email sender, pass to auth handler |

---

## 4. Data Encryption (Field-Level)

### 4.1 Approach

AES-256-GCM authenticated encryption. Fields marked `"encrypted": true` in model JSON get transparent encrypt-on-write / decrypt-on-read.

### 4.2 Configuration

```
ENCRYPTION_KEY=base64-encoded-32-byte-key
```

If not set, encrypted fields are stored as plaintext with a startup warning.

### 4.3 Key Versioning

Encrypted values are prefixed with key version: `v1:<nonce>:<ciphertext>`. This enables future key rotation — new writes use current key version, reads try current then fallback to previous versions.

```go
// Encrypted format: "v{version}:{base64(nonce+ciphertext)}"
// Example: "v1:SGVsbG8gV29ybGQ..."
```

### 4.4 Model JSON

```json
{
  "fields": {
    "ssn": { "type": "string", "encrypted": true, "label": "SSN" },
    "bank_account": { "type": "string", "encrypted": true }
  }
}
```

### 4.5 Integration Points

| Point | Behavior |
|-------|----------|
| `GenericRepository.Create` | Encrypt marked fields before insert |
| `GenericRepository.Update` | Encrypt marked fields before update |
| `GenericRepository.FindByID` | Decrypt marked fields after read |
| `GenericRepository.FindAll` | Decrypt marked fields after read |
| `module/seeder.go` | Encrypt during seed import |
| `data_revision.go` | Store encrypted values (ciphertext in snapshots) |

### 4.6 Limitations

- Encrypted fields cannot be searched/filtered via SQL (ciphertext is opaque)
- Encrypted fields cannot be used in computed expressions
- Sorting by encrypted fields not supported

### 4.7 Files

| File | Purpose |
|------|---------|
| `engine/pkg/security/encryption.go` | AES-256-GCM encrypt/decrypt with key versioning |
| `engine/internal/compiler/parser/model.go` | Add `Encrypted bool` to FieldDefinition |
| `engine/internal/infrastructure/persistence/repository.go` | Hook encrypt/decrypt in CRUD |
| `engine/internal/infrastructure/module/seeder.go` | Encrypt during seed |
| `engine/internal/config.go` | Add encryption key config |

---

## 5. Backup & Restore

### 5.1 CLI Commands

```bash
bitcode db backup [output-path]     # Default: ./backup-{timestamp}.sql
bitcode db backup --gzip            # Compressed backup
bitcode db restore [backup-path]    # Restore from file
bitcode db restore --force          # Skip confirmation prompt
```

### 5.2 Driver-Specific Strategy

| Driver | Backup | Restore |
|--------|--------|---------|
| SQLite | File copy (`cp bitcode.db backup.db`) | File copy back |
| PostgreSQL | `pg_dump` via `os/exec` | `psql < backup.sql` via `os/exec` |
| MySQL | `mysqldump` via `os/exec` | `mysql < backup.sql` via `os/exec` |

### 5.3 Backup Metadata

Each backup includes a metadata header (JSON comment or separate `.meta.json`):

```json
{
  "engine_version": "0.1.0",
  "driver": "sqlite",
  "created_at": "2026-04-24T10:00:00Z",
  "modules": ["base", "crm", "hrm"],
  "compressed": false
}
```

### 5.4 Safety

- Restore requires `--force` flag or interactive confirmation
- Backup creates audit log entry: action `backup`
- Restore creates audit log entry: action `restore`

### 5.5 Files

| File | Purpose |
|------|---------|
| `engine/cmd/bitcode/backup.go` | Cobra commands for backup/restore |
| `engine/internal/infrastructure/persistence/backup.go` | Driver-specific backup/restore logic |

---

## 6. Rate Limiting

### 6.1 Approach

Fiber built-in `limiter` middleware. No new dependency — already bundled with Fiber v2.

### 6.2 Configuration

```
RATE_LIMIT_ENABLED=true
RATE_LIMIT_MAX=100              # requests per window
RATE_LIMIT_WINDOW=1m            # window duration
RATE_LIMIT_AUTH_MAX=5           # auth endpoint limit (stricter)
RATE_LIMIT_AUTH_WINDOW=1m       # auth endpoint window
```

### 6.3 Tiers

| Route Pattern | Max | Window | Reason |
|---------------|-----|--------|--------|
| `/auth/login` | 5 | 1 min | Brute force protection |
| `/auth/register` | 3 | 1 min | Spam prevention |
| `/auth/2fa/*` | 5 | 1 min | OTP brute force |
| `/api/*` | 100 | 1 min | General API |
| `/admin/*` | 200 | 1 min | Admin (higher limit) |

### 6.4 Key Strategy

Rate limit by IP address (`c.IP()`). For authenticated requests, optionally also by `user_id` to prevent distributed attacks from same account.

### 6.5 Response on Limit

```json
HTTP 429 Too Many Requests
{
  "error": "rate limit exceeded",
  "retry_after": 45
}
```

Header: `Retry-After: 45`

### 6.6 Files

| File | Purpose |
|------|---------|
| `engine/internal/presentation/middleware/ratelimit.go` | Rate limit config + middleware factory |
| `engine/internal/app.go` | Wire in setupMiddleware() |
| `engine/internal/config.go` | Add rate limit config fields |

---

## 7. Admin Impersonation

### 7.1 Flow

```
Start impersonation:
  POST /admin/api/impersonate/:user_id
  → Requires: authenticated admin user with role "admin"
  → Cannot impersonate another admin (safety guard)
  → Generates new JWT with target user's identity + claim impersonated_by=admin_id
  → Token TTL: 1 hour (shorter than normal 24h)
  → Audit log: action "impersonate_start", record_id=target_user_id, impersonated_by=admin_id
  → Returns { "token": "...", "impersonating": { "user_id": "...", "username": "..." } }

Stop impersonation:
  POST /admin/api/stop-impersonate
  → Requires: token with impersonated_by claim
  → Generates fresh admin token (original identity)
  → Audit log: action "impersonate_stop", impersonated_by=admin_id
  → Returns { "token": "...", "user_id": admin_id, "username": admin_username }

During impersonation:
  → All API calls use target user's permissions/record rules
  → All audit log entries include impersonated_by=admin_id
  → Admin UI shows impersonation banner (future UI work)
```

### 7.2 JWT Claims Change

```go
type Claims struct {
    UserID         string   `json:"user_id"`
    Username       string   `json:"username"`
    Roles          []string `json:"roles"`
    Groups         []string `json:"groups"`
    ImpersonatedBy string   `json:"impersonated_by,omitempty"` // NEW
    jwt.RegisteredClaims
}
```

`GenerateToken` signature updated to accept optional `impersonatedBy` parameter.

### 7.3 Audit Log Schema Change

Add column to `audit_logs` table:

```sql
impersonated_by TEXT  -- user_id of admin who initiated impersonation
```

### 7.4 Audit Log Entry Change

```go
type AuditLogEntry struct {
    // ... existing fields ...
    ImpersonatedBy string  // NEW
}
```

### 7.5 Audit Middleware Change

In `PersistentAuditMiddleware`, read `impersonated_by` from JWT claims (via `c.Locals("impersonated_by")`) and write to audit log entry.

In `AuthMiddleware`, extract `ImpersonatedBy` from claims and store in `c.Locals("impersonated_by")`.

### 7.6 Safety Guards

- Only users with role `admin` can impersonate
- Cannot impersonate users who have role `admin` (prevent privilege escalation loops)
- Impersonation token expires in 1 hour (not 24h)
- All impersonation actions are audit-logged with `impersonated_by`
- `impersonate_start` and `impersonate_stop` are always logged regardless of method filter

### 7.7 Files

| File | Purpose |
|------|---------|
| `engine/pkg/security/jwt.go` | Add ImpersonatedBy to Claims + GenerateToken |
| `engine/internal/presentation/admin/admin.go` | Impersonate/stop endpoints |
| `engine/internal/presentation/middleware/auth.go` | Extract ImpersonatedBy to Locals |
| `engine/internal/presentation/middleware/audit.go` | Write impersonated_by to audit |
| `engine/internal/infrastructure/persistence/audit_log.go` | Add ImpersonatedBy field + migration |

---

## 8. CSRF Protection (Deferred — Roadmap)

CSRF protection is deferred to the public web forms & website phase. Current state:
- API routes use JWT (stateless, no CSRF needed)
- SSR forms use HTTPOnly cookie with JWT — low CSRF risk since forms are authenticated

When to implement:
- When public web forms feature is built (job portal, contact forms, etc.)
- When website module is implemented
- These forms will be unauthenticated and use session-based CSRF tokens

Approach (when implemented):
- Fiber built-in CSRF middleware
- Apply to public form routes only
- Double-submit cookie pattern

Updated in features.md with this context.

---

## 9. Middleware Chain (After Implementation)

### Global (all requests):
```
Request → RateLimit → Tenant → Audit → Handler
```

### Protected API routes:
```
Request → RateLimit → Tenant → Auth → [ImpersonatedBy extraction] → Permission → RecordRule → Audit → Handler
```

### Auth endpoints:
```
Request → RateLimit(strict) → Handler
```

### SSR /app routes:
```
Request → RateLimit → Tenant → Audit → Handler (auth checked in handler)
```

---

## 10. New Dependencies

| Package | Purpose | Type |
|---------|---------|------|
| Go stdlib `net/smtp` | Email sending | stdlib (no new dep) |
| Go stdlib `crypto/aes`, `crypto/cipher` | AES-256-GCM encryption | stdlib (no new dep) |
| Go stdlib `crypto/rand` | OTP generation | stdlib (no new dep) |
| Fiber `middleware/limiter` | Rate limiting | already bundled with Fiber v2 |

**Zero new external dependencies.** All features use Go stdlib or existing Fiber packages.

---

## 11. Configuration Summary

New env vars added:

```
# Email (SMTP)
SMTP_HOST=
SMTP_PORT=587
SMTP_USER=
SMTP_PASSWORD=
SMTP_FROM=
SMTP_TLS=true

# Encryption
ENCRYPTION_KEY=          # base64-encoded 32-byte key

# Rate Limiting
RATE_LIMIT_ENABLED=true
RATE_LIMIT_MAX=100
RATE_LIMIT_WINDOW=1m
RATE_LIMIT_AUTH_MAX=5
RATE_LIMIT_AUTH_WINDOW=1m
```

---

## 12. Testing Strategy

| Feature | Test Approach |
|---------|--------------|
| Email sender | Unit test with mock SMTP server |
| 2FA OTP | Unit test OTP generation, integration test login flow |
| Encryption | Unit test encrypt/decrypt, round-trip with key versioning |
| Backup/Restore | Integration test per driver (SQLite file copy) |
| Rate limiting | Integration test hitting limit, verify 429 response |
| Impersonation | Integration test full flow: impersonate → act → audit check → stop |
| Audit impersonated_by | Verify audit entries contain impersonated_by during impersonation |
