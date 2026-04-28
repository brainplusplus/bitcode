# Phase 1.5 Implementation Plan: Multi-Tenancy

**Estimated effort**: 3-4 days
**Prerequisites**: Phase 1 (bridge API — `session.tenantId`, `sudo().withTenant()`)
**Test command**: `go test ./internal/infrastructure/persistence/... ./internal/domain/model/...`

---

## Implementation Order

```
Stream 1: Migration — Auto-add tenant_id Column (Day 1)
  ↓
Stream 2: Model-Level Control — tenant_scoped (Day 1-2)
  ↓
Stream 3: Repository Fix — Conditional Filtering (Day 2-3)
  ↓
Stream 4: Config & Bridge Integration (Day 3)
  ↓
Stream 5: Tests (Day 3-4)
```

---

## Stream 1: Migration — Auto-add tenant_id

**File**: `internal/infrastructure/persistence/dynamic_model.go`

In `MigrateModel()`, after building columns:
```go
if tenantEnabled && model.IsTenantScoped() {
    cols += ",\n  tenant_id VARCHAR(36) NOT NULL"
    // Add index on tenant_id
}
```

## Stream 2: Model-Level Control

**File**: `internal/compiler/parser/model.go`

Add to ModelDefinition:
```go
TenantScoped *bool `json:"tenant_scoped,omitempty"` // default: true when tenant enabled
```

Default behavior: all models are tenant_scoped when `tenant.enabled = true`, except models explicitly marked `tenant_scoped: false` (e.g., shared lookup tables).

## Stream 3: Repository Fix

**File**: `internal/infrastructure/persistence/repository.go`

Fix: only add `WHERE tenant_id = ?` when:
1. `tenant.enabled = true` AND
2. Model is `tenant_scoped: true` AND
3. `tenantID` is not empty in context

## Stream 4: Config & Bridge

- Config: `tenant.enabled`, `tenant.strategy`, `tenant.header` (already exist, verify)
- Bridge: `session.tenantId` populated from middleware
- Bridge: `sudo().withTenant(id)` allows cross-tenant access

## Definition of Done

- [ ] `MigrateModel()` auto-adds `tenant_id` column when tenant enabled
- [ ] `tenant_scoped: false` models skip tenant filtering
- [ ] Repository only filters tenant_scoped models
- [ ] `sudo().withTenant()` works for cross-tenant access
- [ ] Existing tests pass, new tenant tests pass
