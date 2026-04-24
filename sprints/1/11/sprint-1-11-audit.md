# Sprint 1.11b — Audit Log & Monitoring (Features #16-#20)

**Date**: 24 April 2026
**Status**: Complete

## Objectives

Complete the Audit Log & Monitoring section (features #16-#20) plus fix underlying gaps.

## Deliverables

### A: Fix #16 — Persistent Audit Logging ✅

- Extended `audit_log` model with: `user_agent`, `request_method`, `request_path`, `status_code`, `duration_ms`
- Created `AuditLogRepository` (`persistence/audit_log.go`) with async write support
- Created `PersistentAuditMiddleware` that writes all write requests to DB
- Skips static assets, health checks, favicon
- Extracts model/record from API paths

### B: Fix Missing Revision Hooks ✅

- Form POST handler in `app.go` now has `revisionRepo` attached
- Process engine `data.go` steps now have `revisionRepo` attached
- `UpdateByCompositePK` now saves revision with before/after snapshot

### C: Logout Endpoint ✅

- Added `POST /auth/logout` endpoint
- Writes audit_log row on logout

### D: Feature #17 — Record Activity Timeline ✅

- API: `GET /admin/api/data/:model/:id/timeline` — combines `data_revisions` + `audit_log`
- Admin UI: "History" button per record in model data table
- Modal timeline with field-level diff rendering

### E: Feature #18 — Login History ✅

- Login/logout/register persisted from both `auth_handler.go` (API) and `app.go` (HTML form)
- Admin page: `/admin/audit/login-history` with User-Agent, IP, action badges
- Sidebar entry under "Audit" section

### F: Feature #19 — API Request Log ✅

- `PersistentAuditMiddleware` persists all write requests with method, path, status, duration, user, IP
- Admin page: `/admin/audit/request-log` with method filtering (All/GET/POST/PUT/DELETE/PATCH)
- Color-coded method badges and status codes

### G: Feature #20 — Data Change Diff ✅

- `data_revisions` already stores `{field: {old, new}}` changes
- Timeline modal renders field-level diff: old value (red strikethrough) → new value (green)
- Integrated into record history modal

## Test Results

- **Before**: 247 tests
- **After**: 252 tests (+5 audit log tests)
- **Build**: Clean

## Files Changed

| File | Change |
|------|--------|
| `embedded/modules/base/models/audit_log.json` | MODIFIED — Added 5 new fields |
| `embedded/modules/base/views/audit_log_list.json` | MODIFIED — Added new fields to list |
| `internal/infrastructure/persistence/audit_log.go` | NEW — AuditLogRepository |
| `internal/infrastructure/persistence/audit_log_test.go` | NEW — 5 tests |
| `internal/presentation/middleware/audit.go` | MODIFIED — Added PersistentAuditMiddleware |
| `internal/presentation/api/auth_handler.go` | MODIFIED — Added logout, audit logging |
| `internal/presentation/admin/admin.go` | MODIFIED — Login history, request log pages, timeline modal, sidebar |
| `internal/presentation/admin/admin_api.go` | MODIFIED — Timeline, login history, request log API endpoints |
| `internal/infrastructure/persistence/repository.go` | MODIFIED — UpdateByCompositePK revision hook |
| `internal/runtime/executor/steps/data.go` | MODIFIED — Attached revisionRepo |
| `internal/app.go` | MODIFIED — Wired audit repo, form POST revision hooks, login audit |
