# Phase 7 Implementation Plan: Module "setting"

**Estimated effort**: 10-14 days
**Prerequisites**: ALL previous phases (1-6C)
**Test command**: Manual testing + `go test ./...`

---

## Implementation Order

```
Stream 1: Module Structure & Array Models (Day 1-2)
  ↓
Stream 2: Views — Dashboard & Model Inspector (Day 2-4)
  ↓
Stream 3: Views — User & Group Management (Day 4-6)
  ↓
Stream 4: Views — Audit, Security, Modules (Day 6-7)
  ↓
Stream 5: Processes & Scripts (Multi-Runtime) (Day 7-10)
  ↓
Stream 6: Security & Permissions (Day 10-11)
  ↓
Stream 7: Migration from admin.go (Day 11-12)
  ↓
Stream 8: Testing & Polish (Day 12-14)
```

---

## Stream 1: Module Structure

### 1.1 Create Directory

```
modules/setting/
├── module.json
├── models/
│   ├── _setting_nav.json
│   ├── _dashboard_stat.json
│   └── _system_info.json
├── views/
├── processes/
├── scripts/
├── security/
└── templates/
```

### 1.2 module.json

```json
{
  "name": "setting",
  "label": "Settings & Administration",
  "version": "1.0.0",
  "depends": ["base"],
  "models": ["models/*.json"],
  "views": ["views/*.json"],
  "processes": ["processes/*.json"],
  "securities": ["security/*.json"],
  "table": { "prefix": "" }
}
```

### 1.3 Array Models

- `_setting_nav.json` — navigation items (source: "array", read-only)
- `_dashboard_stat.json` — dashboard stats (source: "process", refresh: "5m")
- `_system_info.json` — system info (source: "process", refresh: "5m")

---

## Stream 2: Dashboard & Model Inspector

### 2.1 Dashboard

- `views/dashboard.json` — custom view with data_sources (stats, nav, recent_audit)
- `templates/dashboard.html` — dashboard template with stat cards + recent activity

### 2.2 Model Inspector

- `views/model_list.json` — list view using metadata API
- `views/model_detail.json` — form view showing fields, indexes, config
- `views/model_data_list.json` — dynamic data browser (list view with dynamic model)
- `views/model_data_form.json` — record editor

---

## Stream 3: User & Group Management

- `views/user_list.json` — list view on `user` model
- `views/user_form.json` — form with tabs (groups, security history, audit)
- `views/group_list.json` — list view on `group` model
- `views/group_form.json` — form with tabs (model access, record rules, members)

These use existing base models — no new tables.

---

## Stream 4: Audit, Security, Modules

- `views/audit_log_list.json` — list view on `audit_log`
- `views/security_history_list.json` — list view on `security_history`
- `views/module_list.json` — list view using metadata API
- `views/api_explorer.json` — custom view + template

---

## Stream 5: Scripts (Multi-Runtime Stress Test)

### 5.1 JavaScript (goja/quickjs)

- `scripts/dashboard_stats.js` — compute stats via bridge
- `scripts/list_models.js` — list models via meta bridge

### 5.2 Python

- `scripts/system_info.py` — system introspection

### 5.3 Go (yaegi)

- `scripts/data_export.go` — export model data to CSV/JSON/XLSX

### 5.4 Node.js/Bun

- `scripts/model_validator.js` — validate model JSON with ajv (npm package)

**This stream proves all 4+ runtimes work in a real module.**

---

## Stream 6: Security

- `security/groups.json` — define `setting.admin` and `setting.viewer` groups
- Apply permissions to all views and processes

---

## Stream 7: Migration from admin.go

### 7.1 Config

Add to `internal/config.go`:
```go
AdminLegacy bool `mapstructure:"admin.legacy"` // default: true
```

### 7.2 Conditional Loading

In `app.go`:
```go
if config.AdminLegacy {
    // Load admin.go routes (existing)
} else {
    // Skip admin.go, module "setting" handles everything
}
```

### 7.3 Feature Parity Testing

Go through the 17-item checklist from design doc §9.3 and verify each feature works in module "setting".

---

## Stream 8: Testing

- All views render correctly
- All processes execute
- All 4 runtimes work
- Permissions enforced (admin vs viewer vs unauthorized)
- Array models load correctly
- Process source models refresh
- Embedded view filter_by works in tabs
- admin.go can be disabled without breaking anything

## Definition of Done

- [ ] Module "setting" provides ALL admin.go functionality
- [ ] Uses 4+ runtimes (goja, quickjs/Node.js, Python, Go)
- [ ] Uses array-backed models
- [ ] Uses metadata API
- [ ] Uses view modifiers (visible_if, etc.)
- [ ] Uses embedded view filter_by
- [ ] Security groups work
- [ ] admin.go can be disabled via config
- [ ] Feature parity checklist 100% complete
