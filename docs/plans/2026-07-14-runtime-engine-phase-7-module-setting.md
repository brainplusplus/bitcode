# Phase 7: Module "setting" — Admin Panel as JSON Module

**Date**: 14 July 2026
**Status**: Draft
**Depends on**: All previous phases (1-6C) — this is the ultimate stress test
**Unlocks**: Production readiness, admin.go deprecation
**Master doc**: `2026-07-14-runtime-engine-redesign-master.md`

---

## Table of Contents

1. [Goal](#1-goal)
2. [Why This Phase Exists](#2-why-this-phase-exists)
3. [Architecture: Module "setting" Structure](#3-architecture-module-setting-structure)
4. [Models](#4-models)
5. [Views](#5-views)
6. [Processes](#6-processes)
7. [Scripts — Multi-Runtime Stress Test](#7-scripts--multi-runtime-stress-test)
8. [Security](#8-security)
9. [Migration from admin.go](#9-migration-from-admingo)
10. [Implementation Tasks](#10-implementation-tasks)

---

## 1. Goal

Replace the hardcoded `admin.go` (~2645 lines, 9 files) with a **JSON-defined module** called `setting`. This module uses the engine's own building blocks — models, views, processes, scripts — to provide the same admin functionality.

This is the **ultimate stress test**: if the engine can build its own admin panel as a module, it can build anything.

### 1.1 Success Criteria

- Module "setting" provides all functionality currently in `admin.go`
- Uses at least 4 runtimes (goja, quickjs, Node.js/Bun, Python) in its scripts
- Uses array-backed models for fixture/config data
- Uses metadata API for model/view introspection
- Uses morph relations where appropriate (e.g., activity log)
- `admin.go` can be disabled via config (`admin.legacy = false`)
- Zero hardcoded HTML — all UI via views + templates

### 1.2 What This Phase Does NOT Do

- Does not delete `admin.go` — it becomes a fallback that can be disabled
- Does not change the engine core — only builds on top of it
- Does not add new engine features — all features should be in Phase 1-6C

### 1.3 Key Principle

> If module "setting" needs a feature that doesn't exist, that feature should be added to the appropriate earlier phase (1-6C), NOT hacked into module "setting".

This ensures the engine is genuinely capable, not just patched for one use case.

---

## 2. Why This Phase Exists

### 2.1 The Problem with admin.go

`admin.go` is **technical debt**:

1. **~2645 lines of hardcoded Go** — HTML strings, SQL queries, business logic all mixed
2. **Bypasses the engine** — doesn't use models, views, processes, or the bridge API
3. **Not extensible** — adding a new admin page requires Go code changes + recompile
4. **Not themeable** — hardcoded CSS inline
5. **Duplicates engine features** — has its own model listing, data tables, form rendering
6. **9 files** — admin.go, admin_api.go, admin_audit.go, admin_groups.go, admin_list_api.go, admin_models.go, admin_modules.go, admin_security.go, admin_views.go

### 2.2 The Solution

Build the same functionality as a **JSON module** that uses the engine's own capabilities:

```
admin.go (hardcoded):                    module "setting" (JSON-driven):
├── HTML string concatenation            ├── views/*.json (declarative)
├── Direct SQL queries                   ├── models/*.json (with array source)
├── Inline permission checks             ├── security/*.json (declarative)
├── Custom route handlers                ├── processes/*.json (declarative)
└── Hardcoded business logic             └── scripts/*.js/*.py/*.go (multi-runtime)
```

### 2.3 What admin.go Currently Does

| Feature | admin.go File | Module "setting" Equivalent |
|---------|--------------|---------------------------|
| Dashboard (stats) | admin.go | Custom view + process data source |
| Model list | admin_models.go | List view on metadata (array model or meta API) |
| Model detail (fields, indexes) | admin_models.go | Form view on metadata |
| Model data browser | admin_list_api.go | List view with dynamic model |
| Group management | admin_groups.go | CRUD views on `group` model |
| Group permissions | admin_groups.go | Form view with child tables |
| User management | admin.go | CRUD views on `user` model |
| Module list | admin_modules.go | List view on metadata |
| View inspector | admin_views.go | List/form views on metadata |
| Audit log | admin_audit.go | List view on `audit_log` model |
| Security history | admin_security.go | List view on `security_history` model |
| API explorer | admin_api.go | Custom view + meta API |
| Model JSON editor | admin_api.go | Form view with code editor widget |

---

## 3. Architecture: Module "setting" Structure

```
modules/setting/
├── module.json
├── models/
│   ├── _dashboard_stat.json          ← array model (process source)
│   ├── _nav_item.json                ← array model (admin navigation)
│   └── _system_info.json             ← array model (process source)
├── views/
│   ├── dashboard.json                ← custom view (dashboard)
│   ├── model_list.json               ← list view (all models)
│   ├── model_detail.json             ← form view (model inspector)
│   ├── model_data_list.json          ← list view (data browser)
│   ├── model_data_form.json          ← form view (record editor)
│   ├── group_list.json               ← list view
│   ├── group_form.json               ← form view
│   ├── user_list.json                ← list view
│   ├── user_form.json                ← form view
│   ├── module_list.json              ← list view
│   ├── audit_log_list.json           ← list view
│   ├── security_history_list.json    ← list view
│   └── api_explorer.json             ← custom view
├── processes/
│   ├── compute_dashboard_stats.json  ← process: count models, records, etc.
│   ├── compute_system_info.json      ← process: Go version, uptime, etc.
│   ├── impersonate_user.json         ← process: admin impersonation
│   └── export_model_data.json        ← process: export to CSV/JSON
├── scripts/
│   ├── dashboard_stats.js            ← JavaScript (goja/quickjs)
│   ├── system_info.py                ← Python
│   ├── model_validator.js            ← JavaScript (Node.js/Bun for complex validation)
│   └── data_export.go                ← Go (yaegi)
├── security/
│   └── groups.json                   ← admin group permissions
└── templates/
    ├── dashboard.html                ← custom dashboard template
    └── api_explorer.html             ← API explorer template
```

### 3.1 module.json

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

No table prefix — setting module uses base tables directly (user, group, etc.) and its own array models.

---

## 4. Models

### 4.1 Array Models for Admin Data

Module "setting" uses **array-backed models** (Phase 6C) for data that doesn't need a database table:

#### Navigation Items

```json
{
  "name": "_setting_nav",
  "source": "array",
  "primary_key": { "strategy": "natural_key", "field": "key" },
  "fields": {
    "key": { "type": "string" },
    "label": { "type": "string" },
    "icon": { "type": "string" },
    "view": { "type": "string" },
    "group": { "type": "string" },
    "order": { "type": "integer" }
  },
  "rows": [
    { "key": "dashboard", "label": "Dashboard", "icon": "home", "view": "setting.dashboard", "group": "main", "order": 1 },
    { "key": "models", "label": "Models", "icon": "database", "view": "setting.model_list", "group": "schema", "order": 2 },
    { "key": "users", "label": "Users", "icon": "users", "view": "setting.user_list", "group": "security", "order": 3 },
    { "key": "groups", "label": "Groups", "icon": "shield", "view": "setting.group_list", "group": "security", "order": 4 },
    { "key": "modules", "label": "Modules", "icon": "package", "view": "setting.module_list", "group": "system", "order": 5 },
    { "key": "audit", "label": "Audit Log", "icon": "file-text", "view": "setting.audit_log_list", "group": "system", "order": 6 }
  ]
}
```

#### Dashboard Stats (Process Source)

```json
{
  "name": "_dashboard_stat",
  "source": "process",
  "process": "setting.compute_dashboard_stats",
  "refresh": "5m",
  "primary_key": { "strategy": "natural_key", "field": "key" },
  "fields": {
    "key": { "type": "string" },
    "label": { "type": "string" },
    "value": { "type": "string" },
    "icon": { "type": "string" },
    "color": { "type": "string" }
  }
}
```

### 4.2 Existing Base Models Used

Module "setting" does NOT create new tables for users, groups, etc. — it uses existing base models:

| Model | Source | Module |
|-------|--------|--------|
| `user` | base module | Existing DB model |
| `group` | base module | Existing DB model |
| `model_access` | base module | Existing DB model |
| `record_rule` | base module | Existing DB model |
| `audit_log` | base module | Existing DB model |
| `security_history` | base module | Existing DB model |

Module "setting" only adds **views and processes** for these models — no schema changes.

---

## 5. Views

### 5.1 Dashboard (Custom View)

```json
{
  "name": "dashboard",
  "type": "custom",
  "title": "Dashboard",
  "template": "templates/dashboard.html",
  "data_sources": {
    "stats": { "model": "_dashboard_stat" },
    "nav": { "model": "_setting_nav" },
    "recent_audit": {
      "model": "audit_log",
      "domain": [["action", "!=", "read"]],
      "limit": 10
    }
  }
}
```

### 5.2 Model List (Metadata)

```json
{
  "name": "model_list",
  "type": "list",
  "title": "Models",
  "data_sources": {
    "models": { "process": "setting.list_models" }
  },
  "fields": ["name", "module", "label", "field_count", "table_name"],
  "sort": { "field": "name", "order": "asc" },
  "actions": [
    { "label": "Inspect", "process": "setting.inspect_model", "variant": "primary" }
  ]
}
```

### 5.3 User Management

```json
{
  "name": "user_list",
  "type": "list",
  "model": "user",
  "title": "Users",
  "fields": ["name", "email", "last_login", "active"],
  "filters": ["name", "email", "active"],
  "actions": [
    { "label": "Impersonate", "process": "setting.impersonate_user", "permission": "setting.admin", "confirm": "Impersonate this user?" }
  ]
}
```

```json
{
  "name": "user_form",
  "type": "form",
  "model": "user",
  "title": "User",
  "layout": [
    { "row": [
      { "field": "name", "width": 6 },
      { "field": "email", "width": 6 }
    ]},
    { "row": [
      { "field": "active", "width": 3 },
      { "field": "last_login", "width": 3, "readonly": true }
    ]},
    { "tabs": [
      { "label": "Groups", "view": "setting.user_groups", "filter_by": "user_id" },
      { "label": "Security History", "view": "setting.security_history_list", "filter_by": "user_id" },
      { "label": "Audit Log", "view": "setting.audit_log_list", "filter_by": "user_id" }
    ]}
  ]
}
```

### 5.4 Group Management with Permissions

```json
{
  "name": "group_form",
  "type": "form",
  "model": "group",
  "title": "Group",
  "layout": [
    { "row": [
      { "field": "name", "width": 6 },
      { "field": "display_name", "width": 6 }
    ]},
    { "tabs": [
      {
        "label": "Model Access",
        "view": "setting.model_access_list",
        "filter_by": "group_id"
      },
      {
        "label": "Record Rules",
        "view": "setting.record_rule_list",
        "filter_by": "group_id"
      },
      {
        "label": "Members",
        "view": "setting.group_members",
        "filter_by": "group_id"
      }
    ]}
  ]
}
```

---

## 6. Processes

### 6.1 Dashboard Stats

```json
{
  "name": "compute_dashboard_stats",
  "steps": [
    {
      "name": "compute",
      "type": "script",
      "script": { "lang": "javascript", "file": "scripts/dashboard_stats.js" }
    }
  ]
}
```

### 6.2 Model Introspection

```json
{
  "name": "list_models",
  "steps": [
    {
      "name": "get_models",
      "type": "script",
      "script": { "lang": "javascript", "file": "scripts/list_models.js" }
    }
  ]
}
```

### 6.3 Data Export

```json
{
  "name": "export_model_data",
  "steps": [
    {
      "name": "validate",
      "type": "validate",
      "rules": {
        "model": { "required": true },
        "format": { "required": true, "in": ["json", "csv", "xlsx"] }
      }
    },
    {
      "name": "export",
      "type": "script",
      "script": { "lang": "go", "file": "scripts/data_export.go" }
    }
  ]
}
```

---

## 7. Scripts — Multi-Runtime Stress Test

This is where module "setting" proves the engine's multi-runtime capability. Each script uses the runtime best suited for its task.

### 7.1 Dashboard Stats — JavaScript (goja/quickjs)

```javascript
// scripts/dashboard_stats.js
// Runtime: javascript (embedded — fast, no overhead)

const models = await bitcode.meta.models();
const users = await bitcode.db.count("user");
const groups = await bitcode.db.count("group");
const auditToday = await bitcode.db.count("audit_log", {
  domain: [["created_at", ">=", bitcode.time.today()]]
});

return [
  { key: "models", label: "Models", value: String(models.length), icon: "database", color: "#3b82f6" },
  { key: "users", label: "Users", value: String(users), icon: "users", color: "#10b981" },
  { key: "groups", label: "Groups", value: String(groups), icon: "shield", color: "#f59e0b" },
  { key: "audit_today", label: "Actions Today", value: String(auditToday), icon: "activity", color: "#8b5cf6" }
];
```

### 7.2 System Info — Python

```python
# scripts/system_info.py
# Runtime: python (good for system introspection)

import platform
import os

info = bitcode.env.get_all()
db_driver = info.get("DB_DRIVER", "sqlite")

return [
    {"key": "go_version", "label": "Go Version", "value": bitcode.system.go_version()},
    {"key": "os", "label": "OS", "value": platform.system() + " " + platform.release()},
    {"key": "db_driver", "label": "Database", "value": db_driver},
    {"key": "uptime", "label": "Uptime", "value": bitcode.system.uptime()},
    {"key": "modules", "label": "Loaded Modules", "value": str(len(bitcode.meta.modules()))},
]
```

### 7.3 Data Export — Go (yaegi)

```go
// scripts/data_export.go
// Runtime: go (best for file I/O, CSV/XLSX generation)

package main

import (
    "bitcode"
    "encoding/csv"
    "os"
)

func Run(ctx bitcode.Context) (any, error) {
    modelName := ctx.Input("model")
    format := ctx.Input("format")

    records, err := bitcode.DB.FindAll(modelName, nil)
    if err != nil {
        return nil, err
    }

    switch format {
    case "csv":
        return exportCSV(records, modelName)
    case "json":
        return exportJSON(records, modelName)
    case "xlsx":
        return exportXLSX(records, modelName)
    }

    return nil, nil
}
```

### 7.4 Model Validator — Node.js/Bun

```javascript
// scripts/model_validator.js
// Runtime: node (for complex JSON schema validation with npm packages)

const Ajv = require("ajv");
const ajv = new Ajv();

const modelJSON = bitcode.input("model_json");
const parsed = JSON.parse(modelJSON);

// Validate against BitCode model schema
const schema = await bitcode.fs.readJSON("schemas/model.schema.json");
const validate = ajv.compile(schema);
const valid = validate(parsed);

if (!valid) {
  return { valid: false, errors: validate.errors };
}

return { valid: true, model: parsed };
```

### 7.5 Runtime Distribution

| Script | Runtime | Why |
|--------|---------|-----|
| dashboard_stats.js | javascript (goja) | Fast, simple bridge calls, no npm needed |
| system_info.py | python | System introspection, platform module |
| data_export.go | go (yaegi) | File I/O, CSV/XLSX generation, goroutines |
| model_validator.js | node (Bun) | npm packages (ajv), complex validation |

This proves all 4+ runtimes work in a real module.

---

## 8. Security

### 8.1 Permission Group

```json
{
  "groups": {
    "setting.admin": {
      "label": "Settings Administrator",
      "permissions": [
        "setting.dashboard.read",
        "setting.model.read",
        "setting.model.write",
        "setting.user.read",
        "setting.user.write",
        "setting.group.read",
        "setting.group.write",
        "setting.audit.read",
        "setting.impersonate"
      ]
    },
    "setting.viewer": {
      "label": "Settings Viewer",
      "permissions": [
        "setting.dashboard.read",
        "setting.model.read",
        "setting.user.read",
        "setting.group.read",
        "setting.audit.read"
      ]
    }
  }
}
```

### 8.2 Access Control

All setting views and processes require `setting.*` permissions. Regular users cannot access admin functionality unless explicitly granted.

---

## 9. Migration from admin.go

### 9.1 Transition Strategy

```
Phase 1: Module "setting" built alongside admin.go
  → Both accessible: /admin (legacy) and /setting (new)
  → Feature parity validation

Phase 2: Module "setting" becomes default
  → /admin redirects to /setting
  → admin.go still available via config

Phase 3: admin.go deprecated
  → Config: admin.legacy = false (default)
  → admin.go code moved to archived/
  → Module "setting" is the only admin interface
```

### 9.2 Config

```toml
# bitcode.toml
[admin]
legacy = true       # true = admin.go active (default during transition)
                    # false = admin.go disabled, only module "setting"
```

### 9.3 Feature Parity Checklist

| Feature | admin.go | Module "setting" | Status |
|---------|:--------:|:----------------:|:------:|
| Dashboard with stats | ✅ | ⬜ | |
| Model list | ✅ | ⬜ | |
| Model field inspector | ✅ | ⬜ | |
| Model data browser | ✅ | ⬜ | |
| Model data CRUD | ✅ | ⬜ | |
| User list | ✅ | ⬜ | |
| User create/edit | ✅ | ⬜ | |
| Group list | ✅ | ⬜ | |
| Group permissions | ✅ | ⬜ | |
| Group members | ✅ | ⬜ | |
| Module list | ✅ | ⬜ | |
| View inspector | ✅ | ⬜ | |
| Audit log | ✅ | ⬜ | |
| Security history | ✅ | ⬜ | |
| User impersonation | ✅ | ⬜ | |
| API explorer | ✅ | ⬜ | |
| Model JSON editor | ✅ | ⬜ | |

---

## 10. Implementation Tasks

### 10.1 Module Structure

- [ ] Create `modules/setting/module.json`
- [ ] Create directory structure (models, views, processes, scripts, security, templates)
- [ ] Register module in embedded modules

### 10.2 Array Models

- [ ] Create `_setting_nav.json` (navigation items)
- [ ] Create `_dashboard_stat.json` (process source)
- [ ] Create `_system_info.json` (process source)

### 10.3 Views

- [ ] Dashboard (custom view + template)
- [ ] Model list
- [ ] Model detail/inspector
- [ ] Model data browser (dynamic model)
- [ ] Model data form (dynamic model)
- [ ] User list
- [ ] User form (with tabs: groups, security history, audit)
- [ ] Group list
- [ ] Group form (with tabs: model access, record rules, members)
- [ ] Module list
- [ ] Audit log list
- [ ] Security history list
- [ ] API explorer (custom view + template)

### 10.4 Processes

- [ ] compute_dashboard_stats
- [ ] compute_system_info
- [ ] list_models (via meta API)
- [ ] inspect_model (via meta API)
- [ ] impersonate_user
- [ ] export_model_data

### 10.5 Scripts (Multi-Runtime)

- [ ] dashboard_stats.js (goja/quickjs)
- [ ] system_info.py (Python)
- [ ] data_export.go (yaegi)
- [ ] model_validator.js (Node.js/Bun)

### 10.6 Security

- [ ] Create security groups (setting.admin, setting.viewer)
- [ ] Apply permissions to all views and processes

### 10.7 Templates

- [ ] Dashboard HTML template
- [ ] API explorer HTML template

### 10.8 Migration

- [ ] Add `admin.legacy` config
- [ ] Add redirect from /admin to /setting when legacy = false
- [ ] Feature parity testing against admin.go
- [ ] Documentation for migration

### 10.9 Testing

- [ ] Test all views render correctly
- [ ] Test all processes execute successfully
- [ ] Test all 4 runtimes work (goja, quickjs/Node.js, Python, Go)
- [ ] Test permissions (admin vs viewer vs unauthorized)
- [ ] Test array models load correctly
- [ ] Test process source models refresh correctly
- [ ] Test embedded view filter_by works in tabs
