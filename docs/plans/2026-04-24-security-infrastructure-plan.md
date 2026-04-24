# Security & Infrastructure Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement 6 security features (email infra, 2FA, encryption, backup/restore, rate limiting, impersonation) with full audit trail integration.

**Architecture:** Layer new security features into existing Fiber middleware chain and JWT auth system. Email OTP for 2FA, AES-256-GCM for field encryption, driver-aware CLI for backup, Fiber limiter for rate limiting, JWT claim extension for impersonation. Zero new external dependencies.

**Tech Stack:** Go 1.24, Fiber v2, GORM, Go stdlib (net/smtp, crypto/aes, crypto/cipher, crypto/rand)

---

## Implementation Order

Features are ordered by dependency — each builds on the previous:

1. **Rate Limiting** (no dependencies, standalone middleware)
2. **Audit Log Enhancement** (add impersonated_by column — needed by impersonation)
3. **JWT Claims Extension** (add ImpersonatedBy — needed by impersonation)
4. **Admin Impersonation** (depends on #2, #3)
5. **Email Infrastructure** (standalone, needed by 2FA)
6. **2FA Email OTP** (depends on #5)
7. **Field-Level Encryption** (standalone)
8. **Backup & Restore** (standalone CLI)
9. **Update docs & features.md**
10. **Tests, build verification, commit & push**

---

### Task 1: Rate Limiting Middleware

**Files:**
- Create: `engine/internal/presentation/middleware/ratelimit.go`
- Modify: `engine/internal/app.go`
- Modify: `engine/internal/config.go`

**Step 1: Add rate limit config to config.go**

Add to `LoadConfig()`:

```go
// Rate Limiting
RateLimitEnabled    bool
RateLimitMax        int
RateLimitWindow     time.Duration
RateLimitAuthMax    int
RateLimitAuthWindow time.Duration
```

Env vars: `RATE_LIMIT_ENABLED`, `RATE_LIMIT_MAX`, `RATE_LIMIT_WINDOW`, `RATE_LIMIT_AUTH_MAX`, `RATE_LIMIT_AUTH_WINDOW`.

**Step 2: Create ratelimit.go middleware**

```go
package middleware

import (
    "time"
    "github.com/gofiber/fiber/v2"
    "github.com/gofiber/fiber/v2/middleware/limiter"
)

type RateLimitConfig struct {
    Enabled    bool
    Max        int
    Window     time.Duration
    AuthMax    int
    AuthWindow time.Duration
}

func DefaultRateLimitConfig() RateLimitConfig {
    return RateLimitConfig{
        Enabled:    true,
        Max:        100,
        Window:     1 * time.Minute,
        AuthMax:    5,
        AuthWindow: 1 * time.Minute,
    }
}

func RateLimitMiddleware(cfg RateLimitConfig) fiber.Handler {
    return limiter.New(limiter.Config{
        Max:        cfg.Max,
        Expiration: cfg.Window,
        KeyGenerator: func(c *fiber.Ctx) string {
            return c.IP()
        },
        LimitReached: func(c *fiber.Ctx) error {
            return c.Status(429).JSON(fiber.Map{
                "error":       "rate limit exceeded",
                "retry_after": int(cfg.Window.Seconds()),
            })
        },
    })
}

func AuthRateLimitMiddleware(cfg RateLimitConfig) fiber.Handler {
    return limiter.New(limiter.Config{
        Max:        cfg.AuthMax,
        Expiration: cfg.AuthWindow,
        KeyGenerator: func(c *fiber.Ctx) string {
            return c.IP()
        },
        LimitReached: func(c *fiber.Ctx) error {
            return c.Status(429).JSON(fiber.Map{
                "error":       "too many attempts, please try again later",
                "retry_after": int(cfg.AuthWindow.Seconds()),
            })
        },
    })
}
```

**Step 3: Wire rate limiting in app.go setupMiddleware()**

Add rate limit middleware globally and stricter on auth routes.

**Step 4: Run tests**

```bash
cd engine && go build ./...
cd engine && go test ./...
```

---

### Task 2: Audit Log Enhancement (impersonated_by)

**Files:**
- Modify: `engine/internal/infrastructure/persistence/audit_log.go`
- Modify: `engine/internal/presentation/middleware/audit.go`

**Step 1: Add ImpersonatedBy to AuditLogEntry**

```go
type AuditLogEntry struct {
    // ... existing fields ...
    ImpersonatedBy string
}
```

**Step 2: Update Write() to include impersonated_by**

Add `"impersonated_by": nilIfEmpty(entry.ImpersonatedBy)` to the record map.

**Step 3: Add migration for impersonated_by column**

Add `AutoMigrateAuditLog(db)` function that ensures `impersonated_by` column exists.

**Step 4: Update PersistentAuditMiddleware to read impersonated_by from Locals**

```go
impersonatedBy, _ := c.Locals("impersonated_by").(string)
entry.ImpersonatedBy = impersonatedBy
```

**Step 5: Run tests**

```bash
cd engine && go test ./internal/infrastructure/persistence/ -v
```

---

### Task 3: JWT Claims Extension

**Files:**
- Modify: `engine/pkg/security/jwt.go`
- Modify: `engine/internal/presentation/middleware/auth.go`

**Step 1: Add ImpersonatedBy to Claims struct**

```go
type Claims struct {
    UserID         string   `json:"user_id"`
    Username       string   `json:"username"`
    Roles          []string `json:"roles"`
    Groups         []string `json:"groups"`
    ImpersonatedBy string   `json:"impersonated_by,omitempty"`
    Purpose        string   `json:"purpose,omitempty"`
    jwt.RegisteredClaims
}
```

**Step 2: Update GenerateToken to accept ImpersonatedBy**

Add optional parameter or use options pattern.

**Step 3: Update AuthMiddleware to extract ImpersonatedBy**

```go
if claims.ImpersonatedBy != "" {
    c.Locals("impersonated_by", claims.ImpersonatedBy)
}
```

**Step 4: Run tests**

```bash
cd engine && go test ./pkg/security/ -v
```

---

### Task 4: Admin Impersonation Endpoints

**Files:**
- Modify: `engine/internal/presentation/admin/admin.go`
- Modify: `engine/internal/app.go` (wire audit repo to admin)

**Step 1: Add POST /admin/api/impersonate/:user_id**

- Validate caller is admin
- Validate target is not admin
- Load target user's roles/groups
- Generate JWT with target identity + ImpersonatedBy=admin_id, TTL 1h
- Audit log: action "impersonate_start"

**Step 2: Add POST /admin/api/stop-impersonate**

- Validate token has ImpersonatedBy claim
- Load original admin user
- Generate fresh admin JWT
- Audit log: action "impersonate_stop"

**Step 3: Run tests**

```bash
cd engine && go build ./...
```

---

### Task 5: Email Infrastructure

**Files:**
- Create: `engine/pkg/email/sender.go`
- Create: `engine/pkg/email/templates.go`
- Modify: `engine/internal/config.go`

**Step 1: Create email sender**

SMTP sender using Go stdlib `net/smtp` + `crypto/tls`.

**Step 2: Create email templates**

OTP code template with HTML styling.

**Step 3: Add SMTP config to config.go**

Env vars: `SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASSWORD`, `SMTP_FROM`, `SMTP_TLS`.

**Step 4: Run tests**

```bash
cd engine && go build ./...
```

---

### Task 6: 2FA Email OTP

**Files:**
- Create: `engine/pkg/security/otp.go`
- Modify: `engine/internal/presentation/api/auth_handler.go`
- Modify: `engine/internal/app.go`

**Step 1: Create OTP generator**

Crypto-secure 6-digit code using `crypto/rand`.

**Step 2: Modify login flow**

If user has `totp_enabled=true`:
- Generate OTP, store in cache
- Send email
- Return `{ requires_2fa: true, temp_token: "..." }`

**Step 3: Add 2FA endpoints**

- `POST /auth/2fa/enable`
- `POST /auth/2fa/disable`
- `POST /auth/2fa/validate`

**Step 4: Wire email sender and cache into auth handler**

**Step 5: Run tests**

```bash
cd engine && go build ./...
cd engine && go test ./... -v
```

---

### Task 7: Field-Level Encryption

**Files:**
- Create: `engine/pkg/security/encryption.go`
- Modify: `engine/internal/compiler/parser/model.go`
- Modify: `engine/internal/infrastructure/persistence/repository.go`
- Modify: `engine/internal/infrastructure/module/seeder.go`
- Modify: `engine/internal/config.go`

**Step 1: Create encryption.go**

AES-256-GCM with key versioning prefix `v1:`.

**Step 2: Add Encrypted field to parser**

`Encrypted bool` in FieldDefinition.

**Step 3: Hook encrypt/decrypt in GenericRepository**

Encrypt before Create/Update, decrypt after FindByID/FindAll.

**Step 4: Hook encrypt in seeder**

**Step 5: Run tests**

```bash
cd engine && go test ./pkg/security/ -v
cd engine && go test ./... -v
```

---

### Task 8: Backup & Restore CLI

**Files:**
- Create: `engine/cmd/bitcode/backup.go`
- Create: `engine/internal/infrastructure/persistence/backup.go`

**Step 1: Create backup.go driver logic**

SQLite file copy, Postgres pg_dump, MySQL mysqldump.

**Step 2: Create CLI commands**

`bitcode db backup [path]` and `bitcode db restore [path]` with flags.

**Step 3: Wire into main.go dbCmd()**

**Step 4: Run tests**

```bash
cd engine && go build ./cmd/bitcode/
```

---

### Task 9: Update Documentation

**Files:**
- Modify: `docs/features.md` — update status for #57-#61, add impersonation, update CSRF note
- Modify: `docs/codebase.md` — add new files
- Modify: `docs/architecture.md` — update middleware chain
- Modify: `engine/docs/features/security.md` — add 2FA, encryption, impersonation, rate limiting sections
- Modify: `README.md` — add new config vars, CLI commands
- Modify: `AGENTS.md` — move completed items, update remaining

---

### Task 10: Final Verification, Commit & Push

**Step 1:** `cd engine && go test ./... -v` — all tests pass
**Step 2:** `cd engine && go build ./...` — clean build
**Step 3:** `git add -A && git commit` with descriptive message
**Step 4:** `git push`
