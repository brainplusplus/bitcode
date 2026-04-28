# Phase 1 Implementation Plan: Bridge API (`bitcode.*`)

**Estimated effort**: 10-14 days
**Prerequisites**: None (foundation phase)
**Test command**: `go test ./internal/runtime/bridge/...`

---

## Implementation Order

```
Stream 1: Package Structure & Interfaces (Day 1-2)
  ↓
Stream 2: Core Bridges — DB, Model, Session, Env, Config, Log (Day 2-4)
  ↓
Stream 3: Extended Bridges — HTTP, FS, Cache, Email, Notify, Event, Exec (Day 4-6)
  ↓
Stream 4: Advanced — Sudo, Transactions, Bulk Ops, Relations (Day 6-9)
  ↓
Stream 5: Execution Log & Error Contract (Day 9-11)
  ↓
Stream 6: Factory, Security Rules, Integration (Day 11-12)
  ↓
Stream 7: Tests (Day 12-14)
```

---

## Stream 1: Package Structure & Interfaces

### 1.1 Create Package

```
engine/internal/runtime/bridge/
├── interfaces.go       # All 20 namespace interfaces
├── context.go          # bridge.Context struct (holds all 20 bridges + session + params)
├── errors.go           # BridgeError type + error codes
└── types.go            # Shared types (SearchOptions, HTTPOptions, etc.)
```

### 1.2 Key Design Decisions

- `bridge.Context` is the **single struct** passed to all runtimes
- Each namespace is a Go interface — runtimes call methods on these interfaces
- Interfaces return `(any, error)` for JS/Python compatibility
- `BridgeError` wraps all errors with error code + message + details

### 1.3 Core Interface Pattern

```go
// interfaces.go
type ModelHandle interface {
    Get(id string, opts ...SearchOptions) (map[string]any, error)
    Search(opts SearchOptions) ([]map[string]any, error)
    Create(data map[string]any) (map[string]any, error)
    Write(id string, data map[string]any) (map[string]any, error)
    Delete(id string) error
    Count(opts ...SearchOptions) (int64, error)
    // Bulk
    CreateMany(data []map[string]any) ([]map[string]any, error)
    WriteMany(ids []string, data map[string]any) (int64, error)
    DeleteMany(ids []string) (int64, error)
    // Relations
    AddRelation(id string, field string, relatedIDs []string) error
    RemoveRelation(id string, field string, relatedIDs []string) error
    SetRelation(id string, field string, relatedIDs []string) error
    LoadRelation(id string, field string, opts ...SearchOptions) (any, error)
    // Sudo
    Sudo() SudoModelHandle
}
```

---

## Stream 2: Core Bridges

### 2.1 Implementation Order (dependency-based)

```
errors.go → types.go → context.go → interfaces.go
  ↓
db.go (raw SQL — wraps GORM)
  ↓
model.go (CRUD — wraps GenericRepository)
  ↓
session.go (read session data)
env.go (read env with whitelist/blacklist)
config.go (read viper config)
log.go (structured logger)
```

### 2.2 model.go — Largest Bridge

Wraps `GenericRepository`. Key methods:
- `Get()` → `repo.FindByID()`
- `Search()` → `repo.FindAll()` with query building
- `Create()` → `repo.Create()`
- `Write()` → `repo.Update()`
- `Delete()` → `repo.SoftDelete()` or `repo.Delete()`

**Critical**: Must respect permissions via `PermissionService` unless in sudo mode.

### 2.3 session.go

Read-only access to current user session:
```go
type SessionBridge struct {
    userID     string
    userName   string
    email      string
    groups     []string
    tenantID   string
    context    map[string]any  // custom context (company_id, currency_id, etc.)
}
```

---

## Stream 3: Extended Bridges

### 3.1 http.go — TLS Client

```go
import tls_client "github.com/bogdanfinn/tls-client"

type HTTPBridge struct {
    client    tls_client.HttpClient
    timeout   time.Duration
    proxyURL  string
}
```

Methods: `Get`, `Post`, `Put`, `Patch`, `Delete`, `Request` (generic)

### 3.2 fs.go — Sandboxed File System

```go
type FSBridge struct {
    basePath  string   // module path — cannot escape
    allowRead []string // glob patterns
    allowWrite []string
}
```

### 3.3 Others (Small)

- `cache.go` — wraps existing cache (Get/Set/Delete/Has/Flush)
- `email.go` — wraps `email.Sender`
- `notify.go` — wraps WebSocket Hub broadcast
- `event.go` — wraps EventBus (Emit/On)
- `exec.go` — wraps `os/exec` with whitelist + timeout
- `i18n.go` — wraps Translator
- `storage.go` — wraps StorageDriver
- `crypto.go` — wraps FieldEncryptor + password
- `audit.go` — wraps AuditLogRepository
- `security.go` — wraps PermissionService

---

## Stream 4: Advanced Features

### 4.1 sudo.go

```go
type SudoModelHandle struct {
    inner     *ModelBridge
    options   SudoOptions
}

type SudoOptions struct {
    SkipPermission  bool
    SkipValidation  bool
    SkipRecordRules bool
    HardDelete      bool
    TenantID        string // cross-tenant
}
```

### 4.2 tx.go — Transactions

```go
type TxManager struct {
    db *gorm.DB
}

func (t *TxManager) Run(fn func(tx *TxContext) error) error {
    return t.db.Transaction(func(gormTx *gorm.DB) error {
        txCtx := &TxContext{db: gormTx}
        return fn(txCtx)
    })
}
```

### 4.3 Bulk Operations

Add to `GenericRepository`:
- `BulkCreate(records []map[string]any) ([]map[string]any, error)`
- `BulkUpdate(ids []string, data map[string]any) (int64, error)`
- `BulkDelete(ids []string) (int64, error)`
- `BulkUpsert(records []map[string]any, conflictFields []string) ([]map[string]any, error)`

---

## Stream 5: Execution Log

### 5.1 Models

Create `embedded/modules/base/models/`:
- `process_execution.json` — execution log record
- `process_execution_step.json` — per-step log

### 5.2 Executor Wrapper

Wrap `executor.Execute()` to record:
- Start time, end time, duration
- Input params, output result
- Per-step: name, type, status, input, output, error, duration
- Final status: success/error/timeout/cancelled

---

## Stream 6: Factory & Integration

### 6.1 factory.go

```go
func NewBridgeContext(deps BridgeDeps, session SessionData, params map[string]any) *Context {
    return &Context{
        Model:    NewModelBridge(deps.DB, deps.ModelRegistry, deps.PermissionService, session),
        DB:       NewDBBridge(deps.DB),
        Session:  NewSessionBridge(session),
        // ... all 20 bridges
    }
}
```

### 6.2 Security Rules Integration

Parse `SecurityRules` from `module.json`:
```go
type SecurityRules struct {
    EnvAllow  []string `json:"env_allow,omitempty"`
    EnvDeny   []string `json:"env_deny,omitempty"`
    ExecAllow []string `json:"exec_allow,omitempty"`
    FSAllow   []string `json:"fs_allow,omitempty"`
    FSDeny    []string `json:"fs_deny,omitempty"`
    SudoAllow bool     `json:"sudo_allow,omitempty"`
}
```

---

## Definition of Done

- [ ] `bridge/` package with all 20 interfaces implemented
- [ ] `bridge.Context` struct wires all bridges
- [ ] `BridgeError` with error codes
- [ ] Sudo mode works (skip permission, hard delete, cross-tenant)
- [ ] Transactions work (commit, rollback)
- [ ] Bulk ops work (createMany, writeMany, deleteMany, upsertMany)
- [ ] Relation ops work (add, remove, set, load)
- [ ] Execution log records process runs with per-step detail
- [ ] Security rules enforced (env, exec, fs, sudo)
- [ ] HTTP client uses tls-client
- [ ] All tests pass
