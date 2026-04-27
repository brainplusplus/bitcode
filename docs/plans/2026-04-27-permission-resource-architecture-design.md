# Design: Group-Based Permission System + Convention-Driven Architecture

**Date**: 27 April 2026
**Status**: Approved (pending implementation)
**Scope**: Permission system overhaul, architecture restructuring, component upgrades

---

## Table of Contents

1. [Overview](#1-overview)
2. [Part A: Group-Based Permission System](#2-part-a-group-based-permission-system)
3. [Part B: Convention-Driven Architecture](#3-part-b-convention-driven-architecture)
4. [Part C: Component Upgrades](#4-part-c-component-upgrades)
5. [Part D: Admin UI Changes](#5-part-d-admin-ui-changes)
6. [Part E: CLI Commands](#6-part-e-cli-commands)
7. [Part F: Migration Plan](#7-part-f-migration-plan)
8. [Implementation Plan](#8-implementation-plan)

---

## 1. Overview

### Goals

1. **Unified permission model** — Replace dual Role+Group system with single Group concept (Odoo-style architecture, ERPNext-style permission naming)
2. **Convention-driven CRUD** — Model JSON is the source of truth; API and Pages auto-generated, override only when needed
3. **Multi-protocol API** — One definition → REST + GraphQL + WebSocket
4. **Permission-aware UI** — Datatable and forms respect permissions by default
5. **Bi-directional sync** — JSON ↔ DB with conflict detection, download/upload, versioning

### Design Principles

- **Odoo architecture** for groups, ACL, record rules, implied inheritance, share groups
- **ERPNext naming** for 12 permissions: select, read, write, create, delete, print, email, report, export, import, mask, clone
- **Convention over configuration** — zero boilerplate for standard CRUD
- **Override, don't replace** — customize only what differs from convention
- **Server-side security** — masking and field access enforced server-side, never client-only

---

## 2. Part A: Group-Based Permission System

### 2.1 Entities to DELETE

| Entity | Table(s) | Reason |
|--------|----------|--------|
| `Role` | `roles`, `role_permissions`, `role_inherits`, `user_roles` | Merged into Group |
| `Permission` (flat string) | `permissions` | Replaced by ModelAccess matrix |

### 2.2 Entities to CREATE

#### 2.2.1 ModelAccess (replaces Permission + role_permissions)

Equivalent to Odoo's `ir.model.access`. Per-model permission matrix per group.

```sql
CREATE TABLE model_access (
    id              TEXT PRIMARY KEY,  -- UUID
    name            TEXT NOT NULL,     -- "CRM User Contact Access"
    model_name      TEXT NOT NULL,     -- "contact"
    group_id        TEXT,              -- FK → groups (NULL = global, applies to ALL users)
    can_select      BOOLEAN DEFAULT FALSE,
    can_read        BOOLEAN DEFAULT FALSE,
    can_write       BOOLEAN DEFAULT FALSE,
    can_create      BOOLEAN DEFAULT FALSE,
    can_delete      BOOLEAN DEFAULT FALSE,
    can_print       BOOLEAN DEFAULT FALSE,
    can_email       BOOLEAN DEFAULT FALSE,
    can_report      BOOLEAN DEFAULT FALSE,
    can_export      BOOLEAN DEFAULT FALSE,
    can_import      BOOLEAN DEFAULT FALSE,
    can_mask        BOOLEAN DEFAULT FALSE,  -- can see unmasked values
    can_clone       BOOLEAN DEFAULT FALSE,
    module          TEXT NOT NULL,     -- "crm" (originating module)
    modified_source TEXT DEFAULT 'json', -- "json" | "ui"
    created_at      TIMESTAMP,
    updated_at      TIMESTAMP,
    UNIQUE(model_name, group_id),
    FOREIGN KEY (group_id) REFERENCES groups(id)
);
```

**Behavior rules (Odoo-compatible):**
- `group_id = NULL` → applies to **every user** (including portal/public)
- **Additive** — user in Group A (read) + Group B (write) = can read + write
- **Default-deny** — no matching ACL = access denied
- Superuser (`is_superuser=true`) bypasses all ACL checks

#### 2.2.2 SecurityHistory (versioning + rollback)

```sql
CREATE TABLE ir_security_histories (
    id              TEXT PRIMARY KEY,  -- UUID
    entity_type     TEXT NOT NULL,     -- "group" | "model_access" | "record_rule"
    entity_id       TEXT NOT NULL,     -- FK to changed entity
    entity_name     TEXT NOT NULL,     -- "crm.user" (human-readable)
    action          TEXT NOT NULL,     -- "create" | "update" | "delete"
    changes         TEXT,              -- JSON: { "can_delete": { "old": false, "new": true } }
    snapshot        TEXT,              -- JSON: full entity state before change
    user_id         TEXT,              -- FK → users (who made the change)
    source          TEXT NOT NULL,     -- "ui" | "json_load" | "json_export" | "cli" | "module_install"
    module          TEXT,              -- originating module
    created_at      TIMESTAMP
);
```

#### 2.2.3 GroupMenu (menu visibility per group)

```sql
CREATE TABLE group_menus (
    group_id        TEXT NOT NULL,     -- FK → groups
    menu_item_id    TEXT NOT NULL,     -- menu identifier "crm/contacts"
    module          TEXT NOT NULL,     -- "crm"
    PRIMARY KEY (group_id, menu_item_id),
    FOREIGN KEY (group_id) REFERENCES groups(id)
);
```

#### 2.2.4 GroupPage (page visibility per group)

```sql
CREATE TABLE group_pages (
    group_id        TEXT NOT NULL,     -- FK → groups
    page_name       TEXT NOT NULL,     -- "contact_list"
    module          TEXT NOT NULL,     -- "crm"
    PRIMARY KEY (group_id, page_name),
    FOREIGN KEY (group_id) REFERENCES groups(id)
);
```

### 2.3 Entities to UPGRADE

#### 2.3.1 Group (replace Role as primary security concept)

```sql
-- Existing table, add columns:
ALTER TABLE groups ADD COLUMN share BOOLEAN DEFAULT FALSE;
ALTER TABLE groups ADD COLUMN comment TEXT;
ALTER TABLE groups ADD COLUMN module TEXT;           -- originating module
ALTER TABLE groups ADD COLUMN modified_source TEXT DEFAULT 'json';

-- Existing m2m: group_implies (keep as-is)
-- Existing m2m: user_groups (keep as-is)
```

**Group fields:**

| Field | Type | Description |
|-------|------|-------------|
| `id` | UUID PK | |
| `name` | string unique | `"crm.user"` |
| `display_name` | string | `"CRM / User"` (existing, rename from `label`) |
| `category` | string | `"CRM"` — groups UI as exclusive radio per category |
| `share` | bool | `true` = portal/external user group |
| `comment` | text | Notes about the group |
| `module` | string | Originating module name |
| `modified_source` | string | `"json"` or `"ui"` |
| `implied_groups` | m2m → groups | Additive inheritance (existing `group_implies`) |

**Share group behavior:**
- `share=true` → portal/external user group
- Users in share group cannot see internal menus
- Share group cannot imply non-share group (safety guard)
- Typically has strict record rules (own data only)

#### 2.3.2 RecordRule (upgrade existing)

```sql
-- Change group storage from comma-separated string to proper m2m
CREATE TABLE record_rule_groups (
    record_rule_id  TEXT NOT NULL,
    group_id        TEXT NOT NULL,
    PRIMARY KEY (record_rule_id, group_id),
    FOREIGN KEY (record_rule_id) REFERENCES record_rules(id),
    FOREIGN KEY (group_id) REFERENCES groups(id)
);

-- Add columns
ALTER TABLE record_rules ADD COLUMN module TEXT;
ALTER TABLE record_rules ADD COLUMN modified_source TEXT DEFAULT 'json';

-- Remove old column (after migration)
-- ALTER TABLE record_rules DROP COLUMN group_names;
```

**Record rule composition (Odoo-compatible):**
- **Global rules INTERSECT** — all global rules must be satisfied (AND)
- **Group rules UNION** — any matching group rule is sufficient (OR)
- **Global ∩ Group** — global and group rulesets intersect
- `perm_read/write/create/delete` — which operations the rule applies to (default: all)
- Record rules stay at 4 operations (read/write/create/delete) — the other 8 permissions are UI/action gates, not row-level filters

#### 2.3.3 User (upgrade)

```go
type User struct {
    // ... existing fields ...
    IsSuperuser bool    `json:"is_superuser" gorm:"default:false"`
    // REMOVE: Roles []Role (user_roles table)
    // KEEP:   Groups []Group (user_groups table)
}
```

### 2.4 Permission Check Logic

```
CheckModelAccess(user, model, operation) → bool:
  1. if user.is_superuser → return true
  2. allGroups = resolveImpliedGroups(user.Groups)  // recursive
  3. for each ACL where model_name = model:
       if ACL.group_id == NULL OR ACL.group_id in allGroups:
         if ACL.hasPermission(operation):
           return true   // additive
  4. return false  // default-deny

GetRecordRuleFilters(user, model, operation) → []domain:
  1. if user.is_superuser → return []  // no filters
  2. allGroups = resolveImpliedGroups(user.Groups)
  3. globalDomains = []   // will be ANDed
  4. groupDomains = []    // will be ORed
  5. for each rule where model_name = model AND active = true:
       if !rule.appliesToOperation(operation) → skip
       if rule.global:
         globalDomains.append(rule.domain)
       else if rule.groups ∩ allGroups ≠ ∅:
         groupDomains.append(rule.domain)
  6. return AND(globalDomains) AND OR(groupDomains)
```

### 2.5 Fallback Behavior

```
Model has API (auto_crud) BUT no ModelAccess in DB:
  → base.admin group: auto-grant all 12 permissions
  → Log: "[WARN] No ACL for model 'invoice', only admin can access"
  → All non-admin users: denied (default-deny)
```

### 2.6 Security JSON File Format

**Location:** `modules/{module}/securities/*.json`

**One file per group:**

```json
// securities/crm_user.json
{
  "name": "crm.user",
  "label": "CRM / User",
  "category": "CRM",
  "implies": ["base.user"],
  "share": false,
  "access": {
    "contact": ["select", "read", "write", "create", "print", "email", "report", "export", "clone"],
    "lead":    ["select", "read", "write", "create", "print", "email", "report", "export", "clone"]
  },
  "rules": [
    {
      "name": "crm_user_own_contacts",
      "model": "contact",
      "domain": [["created_by", "=", "{{user.id}}"]],
      "perm_read": true,
      "perm_write": true,
      "perm_create": true,
      "perm_delete": false
    },
    {
      "name": "crm_user_own_leads",
      "model": "lead",
      "domain": [["assigned_to", "=", "{{user.id}}"]],
      "perm_read": true,
      "perm_write": true,
      "perm_create": true,
      "perm_delete": false
    }
  ],
  "menus": ["crm/contacts", "crm/leads"],
  "pages": ["contact_list", "contact_form", "lead_list", "lead_form"],
  "comment": "Basic CRM access for sales agents"
}
```

```json
// securities/crm_manager.json
{
  "name": "crm.manager",
  "label": "CRM / Manager",
  "category": "CRM",
  "implies": ["crm.user"],
  "access": {
    "contact": "all",
    "lead":    "all"
  },
  "rules": [
    {
      "name": "crm_manager_all_contacts",
      "model": "contact",
      "domain": []
    },
    {
      "name": "crm_manager_all_leads",
      "model": "lead",
      "domain": []
    }
  ],
  "menus": ["crm/contacts", "crm/leads", "crm/settings"],
  "pages": ["contact_list", "contact_form", "lead_list", "lead_form", "crm_dashboard", "crm_settings"],
  "comment": "Full CRM access including settings and all records"
}
```

**Access shorthand:**
- `"all"` = all 12 permissions enabled
- `["select", "read"]` = only listed permissions enabled
- Omitted model = no access to that model via this group

**module.json reference:**
```json
{
  "name": "crm",
  "securities": ["securities/*.json"]
}
```

### 2.7 Bi-directional Sync

```
┌─────────────────────────────────┐
│  securities/*.json (module files)  │
│  = Developer source code           │
│  = Git-tracked                     │
└──────────┬────────▲────────────┘
           │           │
     Load from JSON    Export to JSON
     (JSON → DB)       (DB → JSON)
           │           │
           ▼           │
┌──────────────────────────────┐
│  Database (runtime)              │
│  = Source of truth at runtime    │
│  = Admin edits go here           │
└──────────┬───────────────┘
           │
           ▼
┌──────────────────────────────┐
│  Admin UI                        │
│  = Edit groups, ACL, rules       │
│  = "Export to Module" button     │
│  = "Load from File" button       │
│  = Upload JSON/ZIP               │
│  = Download JSON/ZIP             │
│  = Conflict detection            │
│  = History + rollback            │
└──────────────────────────────┘
```

**Sync actions:**

| Action | Direction | Behavior |
|--------|-----------|----------|
| Module install/update | JSON → DB | UPSERT by group name. `modified_source="ui"` entries skipped unless `--force` |
| "Load from File" (UI) | JSON → DB | Same as above, with preview/diff |
| "Export to Module" (UI) | DB → JSON | Write securities/*.json files from DB state |
| Upload JSON/ZIP (UI) | File → DB | Parse uploaded files, UPSERT to DB |
| Download JSON/ZIP (UI) | DB → File | Generate JSON files, serve as download |

**Conflict detection:**
- Track `modified_source` ("json" or "ui") and `updated_at` per entity
- On JSON → DB sync: if `modified_source="ui"` and DB is newer → flag as conflict
- Admin resolves: keep DB version or overwrite with JSON

**History tracking:**
- Every change to groups/model_access/record_rules → insert into `ir_security_histories`
- Stores full snapshot before change
- Rollback = restore snapshot to entity

### 2.8 Field-Level Access

Two separate concepts in model JSON:

```json
{
  "fields": {
    "salary": {
      "type": "decimal",
      "groups": ["hr.manager"]
    },
    "phone": {
      "type": "string",
      "mask": true,
      "mask_length": 4
    },
    "ktp_number": {
      "type": "string",
      "groups": ["hr.manager", "hr.user"],
      "mask": true,
      "mask_length": 6
    }
  }
}
```

| Concept | Field Property | Behavior |
|---------|---------------|----------|
| **Field groups** | `"groups": ["hr.manager"]` | User NOT in group → field removed from API response, hidden in form/list. Security feature. |
| **Field mask** | `"mask": true, "mask_length": 4` | User WITHOUT `can_mask` permission → value masked server-side (`****1234`). User WITH `can_mask` → sees full value. |

**Server-side enforcement (critical):**
- Field groups: CRUD handler strips field from response before sending
- Field mask: CRUD handler replaces value with masked version before sending
- Both enforced in Go handler, NOT in client component

### 2.9 Record Rules Migration

**BEFORE** (in model JSON):
```json
// models/contact.json
{
  "name": "contact",
  "fields": { ... },
  "record_rules": [
    { "groups": ["crm.user"], "domain": [["created_by", "=", "{{user.id}}"]] },
    { "groups": ["crm.manager"], "domain": [] }
  ]
}
```

**AFTER** (in securities JSON):
```json
// securities/crm_user.json
{
  "name": "crm.user",
  "rules": [
    { "name": "own_contacts", "model": "contact", "domain": [["created_by", "=", "{{user.id}}"]] }
  ]
}
```

Model JSON becomes **pure schema** — no security concerns.

**Backward compatibility:** Engine should still parse `record_rules` from model JSON during a transition period, with deprecation warning.

---

## 3. Part B: Convention-Driven Architecture

### 3.1 Core Principle

**Model is the source of truth. API and Pages are auto-generated by convention. Override only when needed.**

### 3.2 Model JSON — `api` Field

```json
{
  "name": "contact",
  "module": "crm",
  "label": "Contact",
  "fields": { ... },
  "api": true
}
```

**`api` field variants:**

```json
// Shorthand: full auto (REST + auto pages)
"api": true

// Expanded: full control
"api": {
  "auto_crud": true,
  "auth": true,
  "auto_pages": true,
  "modal": false,
  "protocols": {
    "rest": true,
    "graphql": false,
    "websocket": false
  },
  "search": ["name", "email"],
  "soft_delete": true
}

// API only, no pages (child table, headless model)
"api": { "auto_crud": true, "auto_pages": false }

// API + pages, but pages only list (no form — e.g. log viewer)
"api": { "auto_crud": true, "auto_pages": { "list": true, "form": false } }

// Modal mode (CRUD in modal, no page navigation)
"api": { "auto_crud": true, "auto_pages": true, "modal": true }

// No API, no pages (internal model, accessed only via process/script)
"api": false
// or simply omit "api" field
```

**Defaults when `"api": true`:**
```json
{
  "auto_crud": true,
  "auth": true,
  "auto_pages": true,
  "modal": false,
  "protocols": { "rest": true, "graphql": false, "websocket": false },
  "search": [],          // auto-detect from search_field or title_field
  "soft_delete": true
}
```

**`auto_pages` variants:**
```json
auto_pages: true                    // list + form + create
auto_pages: false                   // no auto pages
auto_pages: { "list": true, "form": false }   // list only
auto_pages: { "list": true, "form": true, "create": true }  // explicit
```

**`modal` behavior:**
- `modal: true` → `bc-datatable` opens modal for create/edit/detail instead of navigating
- `modal: false` (default) → navigate to page URLs
- Best for simple models: tags, categories, types, settings

### 3.3 URL Convention

```
REST API:    /api/v1/{module}/{model_plural}[/:id][/:action]
GraphQL:     /api/v1/graphql
WebSocket:   /ws
Pages:       /{module}/{model_plural}[/:id][/edit|/new]
Swagger:     /api/v1/docs
Admin:       /admin/models/{module}/{model}
```

**Concrete example (module CRM):**

```
REST API:
  GET    /api/v1/crm/contacts              list
  GET    /api/v1/crm/contacts/:id          read
  POST   /api/v1/crm/contacts              create
  PUT    /api/v1/crm/contacts/:id          update
  DELETE /api/v1/crm/contacts/:id          delete
  POST   /api/v1/crm/contacts/:id/clone    clone
  POST   /api/v1/crm/contacts/onchange     onchange

Pages:
  GET    /crm/contacts                     list page
  GET    /crm/contacts/new                 create form
  GET    /crm/contacts/:id                 detail view
  GET    /crm/contacts/:id/edit            edit form

GraphQL:
  POST   /api/v1/graphql                   single endpoint

WebSocket:
  GET    /ws                               subscribe to events

Swagger:
  GET    /api/v1/docs                      Swagger UI
  GET    /api/v1/docs/openapi.json         OpenAPI 3.0 spec

Admin:
  GET    /admin/models/crm/contact         model admin page
```

### 3.4 Auto-Generated Pages

When `auto_pages: true`, engine generates pages from model fields:

**Auto list page:**
- Columns from model fields (exclude: text, richtext, json, one2many, computed — too wide)
- Search bar from `search_field` or `title_field`
- Filters from selection and many2one fields
- Sort by `title_field` or first string field
- Uses `bc-datatable` component (NOT `bc-view-list`)
- Permission-aware: buttons/actions gated by user's ModelAccess

**Auto form page:**
- Fields from model definition, one field per row
- Field type → component mapping via `fieldTypeToTag()`
- `readonly_if`, `mandatory_if`, `depends_on` → behavior props
- `one2many` fields → tabs with child table
- `many2many` fields → multi-select/tag input
- Permission-aware: save button gated by can_write/can_create

### 3.5 Override Mechanism

**Priority: override file > auto-generated from model**

#### API Override

File: `apis/{name}.json`

```json
// apis/contact_api.json — override auto-generated
{
  "name": "contact_api",
  "module": "crm",
  "model": "contact",
  "endpoints": [
    {
      "method": "PUT",
      "path": "/:id",
      "action": "update",
      "process": "validate_contact_update"
    },
    {
      "method": "POST",
      "path": "/:id/merge",
      "action": "merge",
      "process": "merge_contacts"
    }
  ]
}
```

**Merge behavior:**
- Endpoint matching same method + path → **override** handler
- New endpoint → **add** to auto-generated
- Auto-generated endpoints not overridden → **keep**

#### Page Override

File: `pages/{name}.json`

```json
// pages/contact_form.json — override auto-generated form
{
  "name": "contact_form",
  "type": "form",
  "module": "crm",
  "model": "contact",
  "title": "Contact",
  "layout": [
    { "row": [
      { "field": "name", "width": 6 },
      { "field": "email", "width": 6 }
    ]},
    { "row": [
      { "field": "phone", "width": 4 },
      { "field": "company", "width": 8 }
    ]},
    { "tabs": [
      { "label": "Notes", "fields": ["notes"] },
      { "label": "Tags", "fields": ["tags"] }
    ]}
  ],
  "actions": [
    { "label": "Send Email", "process": "send_contact_email", "variant": "secondary" }
  ]
}
```

**Match by:** `model` + `type` field in JSON content (not filename).

#### Custom Pages (non-CRUD)

```json
// pages/crm_dashboard.json — custom page, not tied to model CRUD
{
  "name": "crm_dashboard",
  "type": "custom",
  "title": "CRM Dashboard",
  "template": "crm_dashboard"
}
```

### 3.6 Cross-Module References

`"module"` field in api.json and pages.json enables borrowing models from other modules:

```json
// modules/crm/pages/sales_team_list.json
// CRM module displays HRM's employee model as "Sales Team"
{
  "name": "sales_team_list",
  "module": "hrm",
  "model": "employee",
  "type": "list",
  "title": "Sales Team",
  "fields": ["name", "email", "phone", "department"]
}
```

**Behavior:**
- `"module": "hrm"` → resolve model definition from `hrm.employee`
- Table name = `hrm_employee` (uses hrm prefix)
- Permission check = ACL for model "employee" (not "crm.employee")
- Record rules = apply rules for model "employee"
- URL mounting = in CRM context (`/crm/sales-team`)
- Menu = can be referenced from CRM menu

**Validation:**
- Target module must be in `depends` of current module
- Model must exist in target module
- Error if dependency missing: `"module crm references hrm.employee but hrm is not in depends"`

### 3.7 Folder Rename: views → pages

```
BEFORE:                    AFTER:
modules/crm/views/         modules/crm/pages/
  contact_list.json           contact_list.json
  lead_list.json              lead_list.json

module.json:               module.json:
  "views": [...]              "pages": [...]
```

### 3.8 Module Folder Structure (Final)

```
modules/crm/
├── module.json                ← manifest
├── models/
│   ├── contact.json               ← schema + api config + field groups/mask
│   └── lead.json
├── securities/
│   ├── crm_user.json              ← group + ACL + rules
│   ├── crm_manager.json
│   └── crm_portal.json            ← share group (optional)
├── apis/                          ← OPTIONAL — only for override/custom
│   └── contact_merge.json         ← custom endpoint
├── pages/                         ← OPTIONAL — only for override/custom
│   ├── contact_form.json          ← override auto-generated form
│   └── crm_dashboard.json         ← custom page
├── processes/*.json
├── templates/*.html
├── i18n/*.json
├── agents/*.json
└── migrations/*.json
```

**Minimal module (zero boilerplate):**
```
modules/crm/
├── module.json
├── models/
│   └── contact.json          ← THIS IS ENOUGH
└── securities/
    └── crm_user.json
```

### 3.9 module.json Changes

```json
{
  "name": "crm",
  "version": "1.0.0",
  "label": "CRM",
  "depends": ["base"],
  "category": "Sales",
  "table": { "prefix": "crm" },
  "models": ["models/*.json"],
  "securities": ["securities/*.json"],
  "apis": ["apis/*.json"],
  "pages": ["pages/*.json"],
  "processes": ["processes/*.json"],
  "agents": ["agents/*.json"],
  "migrations": ["migrations/*.json"],
  "templates": ["templates/*.html"],
  "i18n": ["i18n/*.json"],
  "menu": [
    { "label": "CRM", "icon": "users", "children": [
      { "label": "Contacts", "page": "contact_list", "groups": ["crm.user"] },
      { "label": "Leads", "page": "lead_list", "groups": ["crm.user"] },
      { "label": "Settings", "page": "crm_settings", "groups": ["crm.manager"] }
    ]}
  ]
}
```

**Changes from current:**
- `"views"` → `"pages"` (rename)
- `"securities"` added
- `"permissions"` removed (replaced by ModelAccess in securities)
- `"groups"` removed from module.json (moved to securities/*.json)
- Menu items gain `"groups"` for visibility
- Menu references `"page"` not `"view"`

### 3.10 Loading Order

```
1. module.json        → manifest parsing
2. models/*.json      → schema registration + DB migration
3. securities/*.json  → groups + ModelAccess + RecordRules → sync to DB
4. apis/*.json        → override/custom endpoints + wire permission middleware
5. pages/*.json       → override/custom pages
6. processes/*.json   → business logic
7. agents/*.json      → event handlers + cron
8. templates/*.html   → Go templates
9. i18n/*.json        → translations
10. migrations/*.json → seed data
```

### 3.11 Auto Swagger/OpenAPI

Engine auto-generates OpenAPI 3.0 spec from:
- Model definitions → schemas (fields → properties, types → OpenAPI types)
- API definitions → paths (endpoints → operations)
- Permission definitions → security requirements

```
GET /api/v1/docs              → Swagger UI (embedded static files)
GET /api/v1/docs/openapi.json → OpenAPI 3.0 spec (JSON)
GET /api/v1/docs/openapi.yaml → OpenAPI 3.0 spec (YAML)
```

Spec generated at startup, cached, refreshed on module reload (dev mode).

### 3.12 Multi-Protocol CRUD

Model `"api.protocols"` controls which protocols are active:

| Protocol | Default | Implementation |
|----------|---------|----------------|
| REST | `true` | Existing CRUD handler, upgraded URL pattern |
| GraphQL | `false` | Schema + resolvers auto-generated from model fields |
| WebSocket | `false` | CRUD over WS with request/reply pattern, extends existing hub |

All three protocols share the same permission check (ModelAccess + RecordRules).

---

## 4. Part C: Component Upgrades

### 4.1 `bc-datatable` — Upgrade to Permission-Aware CRUD Table

**Current state:** 442 lines, rich features (filter builder, column picker, drag columns, export, bulk actions, presets). NOT wired by compiler. NOT permission-aware.

**Action:** Upgrade as the primary list component for auto pages. Replace `bc-view-list` usage in `CompileList()`.

#### New Props

```typescript
// Permission props
@Prop() permissions: string = '{}';
// Format: {
//   "can_select": true, "can_read": true, "can_write": true,
//   "can_create": true, "can_delete": true, "can_print": true,
//   "can_email": true, "can_report": true, "can_export": true,
//   "can_import": true, "can_mask": true, "can_clone": true
// }

// Navigation props
@Prop() createUrl: string = '';      // URL for "New" button
@Prop() editUrl: string = '';        // URL pattern: "/crm/contacts/:id/edit"
@Prop() detailUrl: string = '';      // URL pattern: "/crm/contacts/:id"

// Modal mode
@Prop() modalMode: boolean = false;  // true = CRUD in modal, false = navigate
@Prop() formFields: string = '[]';   // field definitions for modal form

// Module context
@Prop() moduleName: string = '';     // "crm" — for API URL resolution
```

#### Permission-Aware Behavior

```
Toolbar:
  "New" button        → visible only if can_create
  "Export" button     → visible only if can_export
  "Import" button     → visible only if can_import
  Filter builder      → always visible (filtering is read operation)
  Column picker       → always visible

Bulk actions bar (when rows selected):
  "Delete" action     → visible only if can_delete
  "Export" action     → visible only if can_export
  "Clone" action      → visible only if can_clone
  Custom actions      → visible based on action.permission

Row actions (per row, right side):
  Edit icon/link      → visible only if can_write
  Delete icon         → visible only if can_delete
  Clone icon          → visible only if can_clone
  Print icon          → visible only if can_print
  Email icon          → visible only if can_email

Row click:
  modal=false → navigate to detailUrl (if can_read)
  modal=true  → open detail modal (if can_read)

Cell rendering:
  Masked fields       → show "****1234" (server already masks, but component should show mask indicator)
  Hidden fields       → column not rendered (server strips from response)
```

#### Modal Mode

When `modalMode: true`:

```
"New" button click    → open bc-dialog-modal with empty form
Row click             → open bc-dialog-modal with record data (read-only if !can_write)
Edit action           → open bc-dialog-modal with record data (editable)
Save in modal         → POST/PUT via API → refresh table → close modal
Delete in modal       → DELETE via API → refresh table → close modal
```

Modal uses `bc-view-form` internally for the form body.

### 4.2 `bc-view-form` — Upgrade to Permission-Aware

```typescript
// New props
@Prop() permissions: string = '{}';   // same format as datatable
@Prop() moduleName: string = '';      // for API URL resolution

// Permission-aware behavior:
// - Save button: visible only if can_write (edit) or can_create (new)
// - Delete button: visible only if can_delete
// - Clone button: visible only if can_clone
// - Print button: visible only if can_print
// - Email button: visible only if can_email
// - Fields with mask: show masked value + "reveal" button (if can_mask)
// - Fields with groups: not rendered (server strips from response)
```

### 4.3 `CompileList()` — Switch to `bc-datatable`

```go
// BEFORE
func (c *ComponentCompiler) CompileList(viewDef *parser.ViewDefinition) string {
    return fmt.Sprintf(`<bc-view-list model="%s" ...></bc-view-list>`, ...)
}

// AFTER
func (c *ComponentCompiler) CompileList(viewDef *parser.ViewDefinition, permissions map[string]bool, moduleName string, modal bool) string {
    columnsJSON := c.fieldsToColumnDefs(viewDef)
    permsJSON := toJSON(permissions)
    return fmt.Sprintf(`<bc-datatable model="%s" module-name="%s" columns='%s' permissions='%s' modal-mode="%t" create-url="%s" detail-url="%s" edit-url="%s"></bc-datatable>`,
        esc(viewDef.Model), esc(moduleName), columnsJSON, permsJSON, modal,
        createUrl, detailUrl, editUrl)
}
```

### 4.4 `bc-view-list` — Deprecate

Keep for backward compatibility but mark as deprecated. New auto pages use `bc-datatable`.

---

## 5. Part D: Admin UI Changes

### 5.1 Admin URL Change

```
BEFORE: /admin/models/contact
AFTER:  /admin/models/crm/contact
```

Module prefix prevents ambiguity when same model name exists in different modules.

### 5.2 Model Admin Page — Full Wireframe

Current tabs: Form, Fields, Connections, Schema.
New tabs: **API** (new), Fields (upgraded with MASK/GROUPS columns).

URL change: `/admin/models/contact` → `/admin/models/crm/contact`

```
+------------------------------------------------------------------------------------+
| Admin / Models / crm / contact                                                     |
+------------------------------------------------------------------------------------+
|                                                                                    |
|  contact                                                                           |
|  crm                                                                               |
|                                                                                    |
|  ┌────────┬──────────┬──────────────┬──────────┬───────┐                           |
|  │  Form  │  Fields  │ Connections  │  Schema  │  API  │                           |
|  └────────┴──────────┴──────────────┴──────────┴───────┘                           |
+------------------------------------------------------------------------------------+
```

#### Tab: Form (existing — no change)

Shows the rendered form preview of the model. No changes needed.

#### Tab: Fields (UPGRADED — add MASK, GROUPS, new field properties)

Current columns: NO, NAME, TYPE, LABEL, REQUIRED, UNIQUE, DEFAULT, RELATION.
New columns: **MASK**, **GROUPS** added.

```
+----------------------------------------------------------------------------------------------------------------------+
| Fields                                                                                                               |
+----------------------------------------------------------------------------------------------------------------------+
| NO | NAME       | TYPE       | LABEL      | REQUIRED | UNIQUE | DEFAULT | RELATION | MASK  | GROUPS                 |
|----|------------|------------|------------|----------|--------|---------|----------|-------|------------------------|
| 1  | name       | string     | Full Name  | ●        | —      | —       | —        | —     | —                      |
| 2  | email      | email      | Email      | —        | —      | —       | —        | —     | —                      |
| 3  | phone      | string     | Phone      | —        | —      | —       | —        | ✔ /4  | —                      |
| 4  | company    | string     | Company    | —        | —      | —       | —        | —     | —                      |
| 5  | notes      | richtext   | Notes      | —        | —      | —       | —        | —     | —                      |
| 6  | photo      | image      | Photo      | —        | —      | —       | —        | —     | —                      |
| 7  | salary     | decimal    | Salary     | —        | —      | —       | —        | —     | hr.manager             |
| 8  | ktp_number | string     | KTP No     | —        | —      | —       | —        | ✔ /6  | hr.manager, hr.user    |
| 9  | tags       | many2many  | Tags       | —        | —      | —       | tag      | —     | —                      |
| 10 | leads      | one2many   | Leads      | —        | —      | —       | lead     | —     | —                      |
| 11 | type       | selection  | Type       | —        | —      | person  | —        | —     | —                      |
+----------------------------------------------------------------------------------------------------------------------+

MASK column format: "✔ /N" where N = mask_length (number of visible chars from end)
  ✔ /4 on phone "08123456789" → displayed as "*******6789"
  ✔ /6 on ktp "3201234567890001" → displayed as "**********890001"
  — = no masking

GROUPS column: comma-separated group names
  If set, field is ONLY visible to users in those groups
  Users outside groups → field stripped from API response, hidden in form/list
  — = visible to all users with model-level read access
```

Behavior:
- Clicking MASK cell → toggle modal: enable/disable mask + set mask_length
- Clicking GROUPS cell → multi-select dropdown of available groups
- Changes saved to model JSON `fields.{name}.mask`, `fields.{name}.mask_length`, `fields.{name}.groups`
- Existing columns (REQUIRED, UNIQUE, DEFAULT, RELATION) unchanged

#### Tab: Connections (existing — no change)

Shows model relationships (many2one, one2many, many2many). No changes needed.

#### Tab: Schema (existing — no change)

Shows JSON editor for the raw model definition. No changes needed.
Note: the `api` field will be visible and editable here as raw JSON.

#### Tab: API (NEW)

Configure the `api` field in model JSON. This tab provides a UI for what would otherwise be manual JSON editing.

```
+------------------------------------------------------------------------------------+
| API Configuration                                                                  |
+------------------------------------------------------------------------------------+
|                                                                                    |
|  ┌─ CRUD ──────────────────────────────────────────────────────────────────────┐   |
|  │                                                                             │   |
|  │  Auto CRUD:     [✔]     Automatically generate REST CRUD endpoints          │   |
|  │  Auth Required: [✔]     Require JWT authentication for all endpoints        │   |
|  │  Soft Delete:   [✔]     DELETE sets deleted_at instead of hard delete       │   |
|  │                                                                             │   |
|  └─────────────────────────────────────────────────────────────────────────────┘   |
|                                                                                    |
|  ┌─ Pages ─────────────────────────────────────────────────────────────────────┐   |
|  │                                                                             │   |
|  │  Auto Pages:    [✔]     Auto-generate list and form pages                   │   |
|  │    List Page:   [✔]     Generate list page at /{module}/{model_plural}      │   |
|  │    Form Page:   [✔]     Generate form page at /{module}/{model_plural}/:id  │   |
|  │    Create Page: [✔]     Generate create page at /{module}/{model_plural}/new│   |
|  │  Modal Mode:    [ ]     Open CRUD in modal instead of navigating to page    │   |
|  │                                                                             │   |
|  └─────────────────────────────────────────────────────────────────────────────┘   |
|                                                                                    |
|  ┌─ Protocols ─────────────────────────────────────────────────────────────────┐   |
|  │                                                                             │   |
|  │  REST:          [✔]     /api/v1/crm/contacts                               │   |
|  │  GraphQL:       [ ]     Available via /api/v1/graphql                       │   |
|  │  WebSocket:     [ ]     CRUD over WebSocket at /ws                          │   |
|  │                                                                             │   |
|  └─────────────────────────────────────────────────────────────────────────────┘   |
|                                                                                    |
|  ┌─ Search & Display ──────────────────────────────────────────────────────────┐   |
|  │                                                                             │   |
|  │  Title Field:   [ name          ▼ ]   Field used as display name            │   |
|  │  Search Fields: [ name, email, company ]   Fields searchable via ?q=        │   |
|  │                 [+ Add field]                                               │   |
|  │                                                                             │   |
|  └─────────────────────────────────────────────────────────────────────────────┘   |
|                                                                                    |
|  ┌─ Model Options ─────────────────────────────────────────────────────────────┐   |
|  │                                                                             │   |
|  │  Timestamps:    [✔]     Auto-manage created_at, updated_at                  │   |
|  │  Timestamps By: [✔]     Auto-manage created_by, updated_by                  │   |
|  │  Soft Deletes:  [✔]     Add deleted_at column for soft delete               │   |
|  │  Soft Del. By:  [ ]     Track deleted_by on soft delete                     │   |
|  │  Versioning:    [ ]     Optimistic locking via version column               │   |
|  │                                                                             │   |
|  └─────────────────────────────────────────────────────────────────────────────┘   |
|                                                                                    |
|  ┌─ Generated Endpoints (read-only preview) ───────────────────────────────────┐   |
|  │                                                                             │   |
|  │  REST API:                                                                  │   |
|  │    GET    /api/v1/crm/contacts              list                            │   |
|  │    GET    /api/v1/crm/contacts/:id          read                            │   |
|  │    POST   /api/v1/crm/contacts              create                          │   |
|  │    PUT    /api/v1/crm/contacts/:id          update                          │   |
|  │    DELETE /api/v1/crm/contacts/:id          delete                          │   |
|  │    POST   /api/v1/crm/contacts/:id/clone    clone                           │   |
|  │    POST   /api/v1/crm/contacts/onchange     onchange                        │   |
|  │                                                                             │   |
|  │  Pages:                                                                     │   |
|  │    GET    /crm/contacts                     list page                       │   |
|  │    GET    /crm/contacts/new                 create form                     │   |
|  │    GET    /crm/contacts/:id                 detail view                     │   |
|  │    GET    /crm/contacts/:id/edit            edit form                       │   |
|  │                                                                             │   |
|  │  Overrides:                                                                 │   |
|  │    apis/contact_merge.json → POST /:id/merge (custom)                       │   |
|  │    pages/contact_form.json → form page (override)                           │   |
|  │                                                                             │   |
|  └─────────────────────────────────────────────────────────────────────────────┘   |
|                                                                                    |
|  ┌─ Primary Key ───────────────────────────────────────────────────────────────┐   |
|  │                                                                             │   |
|  │  Strategy:      [ UUID v4       ▼ ]                                         │   |
|  │                                                                             │   |
|  │  Options: auto-increment | UUID v4 | UUID v7 | naming_series |              │   |
|  │           natural_key | composite | format | manual                         │   |
|  │                                                                             │   |
|  │  (Additional fields shown based on strategy selection)                      │   |
|  │                                                                             │   |
|  └─────────────────────────────────────────────────────────────────────────────┘   |
|                                                                                    |
|  [Save]  [Discard]                                                                 |
|                                                                                    |
+------------------------------------------------------------------------------------+
```

Behavior:
- Toggling checkboxes updates the `api` field in model JSON
- "Generated Endpoints" section is **read-only preview** — shows what engine will auto-generate
- "Overrides" sub-section shows any `apis/*.json` or `pages/*.json` files that override auto-generated
- "Model Options" section maps to existing model JSON fields (`timestamps`, `soft_deletes`, `version`, etc.)
- "Primary Key" section maps to existing `primary_key` field in model JSON
- [Save] writes changes to model JSON file on disk
- Changes take effect on next server restart (or immediately in dev mode with hot reload)

#### Tab: API — Modal Mode Example (for simple models like Tag)

When Modal Mode is enabled, the preview changes:

```
+------------------------------------------------------------------------------------+
|  ┌─ Pages ─────────────────────────────────────────────────────────────────────┐   |
|  │                                                                             │   |
|  │  Auto Pages:    [✔]                                                         │   |
|  │    List Page:   [✔]                                                         │   |
|  │    Form Page:   [✔]  (rendered inside modal, not as separate page)          │   |
|  │    Create Page: [✔]  (rendered inside modal, not as separate page)          │   |
|  │  Modal Mode:    [✔]  ← ENABLED                                             │   |
|  │                                                                             │   |
|  │  ⓘ Modal mode: Create/Edit/Detail will open in a dialog overlay            │   |
|  │    on top of the list page. No page navigation for CRUD operations.         │   |
|  │    Best for simple models (tags, categories, types, settings).              │   |
|  │                                                                             │   |
|  └─────────────────────────────────────────────────────────────────────────────┘   |
|                                                                                    |
|  ┌─ Generated Endpoints (read-only preview) ───────────────────────────────────┐   |
|  │                                                                             │   |
|  │  REST API:                                                                  │   |
|  │    GET    /api/v1/crm/tags              list                                │   |
|  │    GET    /api/v1/crm/tags/:id          read                                │   |
|  │    POST   /api/v1/crm/tags              create                              │   |
|  │    PUT    /api/v1/crm/tags/:id          update                              │   |
|  │    DELETE /api/v1/crm/tags/:id          delete                              │   |
|  │                                                                             │   |
|  │  Pages:                                                                     │   |
|  │    GET    /crm/tags                     list page (with modal CRUD)         │   |
|  │    (no separate form/create/edit pages — handled via modal)                 │   |
|  │                                                                             │   |
|  └─────────────────────────────────────────────────────────────────────────────┘   |
+------------------------------------------------------------------------------------+
```

### 5.4 Group Admin Page (NEW)

```
/admin/groups

List of all groups with columns:
| Name        | Label          | Category | Share | Module | Users | Modified |
|-------------|----------------|----------|-------|--------|-------|----------|
| crm.user    | CRM / User     | CRM      | No    | crm    | 5     | JSON     |
| crm.manager | CRM / Manager  | CRM      | No    | crm    | 2     | UI       |
| base.admin  | Administrator  | Base     | No    | base   | 1     | JSON     |
```

### 5.5 Group Detail Page — Full Wireframe (Odoo-style)

Reference: Odoo `res.groups` form view with 7 tabs.

```
+------------------------------------------------------------------------------------+
| [New]  Groups > CRM / User                                                  1/12  |
+------------------------------------------------------------------------------------+
|                                                                                    |
|  Application : [ CRM          ▼ ]                                                 |
|  Name        : [ User                    ]                                         |
|  Full Name   : crm.user  (read-only, auto-generated from application.name)        |
|  Share Group : [ ]                                                                 |
|  Module      : crm  (read-only, set on creation)                                  |
|  Source      : JSON  (read-only, "JSON" or "UI" — last modified source)            |
|                                                                                    |
|  ┌────────┬───────────┬────────┬────────┬───────────────┬──────────────┬────────┐  |
|  │ Users  │ Inherited │ Menus  │ Pages  │ Access Rights │ Record Rules │ Notes  │  |
|  └────────┴───────────┴────────┴────────┴───────────────┴──────────────┴────────┘  |
+------------------------------------------------------------------------------------+
```

#### Tab: Users

Users assigned to this group. Adding a user here also adds them to all implied groups.

```
+------------------------------------------------------------------------------------+
| Users                                                                              |
+------------------------------------------------------------------------------------+
| Name                     | Username      | Email                    | Active       |
|--------------------------|---------------|--------------------------|--------------|
| John Doe                 | john          | john@example.com         | ✔            |
| Jane Smith               | jane          | jane@example.com         | ✔            |
| Ahmad Rizki              | ahmad         | ahmad@example.com        | ✔            |
+------------------------------------------------------------------------------------+
| + Add a line                                                                       |
+------------------------------------------------------------------------------------+
```

Behavior:
- Adding user here → auto-add to implied groups (e.g., adding to CRM/Manager auto-adds to CRM/User and Base/User)
- Removing user → only removes from THIS group, not from implied groups
- Shows all users including those inherited via implied groups (with indicator)

#### Tab: Inherited

Groups that are automatically included when a user is added to this group.
This is the `implied_ids` / `implies` relationship.

```
+------------------------------------------------------------------------------------+
| Users added to this group are automatically added to the following groups.          |
+------------------------------------------------------------------------------------+
| Group Name                                                                         |
|------------------------------------------------------------------------------------|
| Base / User                                                                        |
+------------------------------------------------------------------------------------+
| + Add a line                                                                       |
+------------------------------------------------------------------------------------+
```

Behavior:
- Adding "Sales / User" here means: anyone in CRM/User also gets Sales/User permissions
- This is additive inheritance — CRM/User gets its own permissions PLUS all permissions from Base/User
- Circular dependency detection: engine rejects if adding would create a cycle

#### Tab: Menus

Menu items visible to users in this group. This is **visibility, not security** — hiding a menu doesn't prevent API access.

```
+------------------------------------------------------------------------------------+
| Menus                                                                              |
+------------------------------------------------------------------------------------+
| Menu Name                     | Module          | Parent                           |
|-------------------------------|-----------------|----------------------------------|
| Contacts                      | crm             | CRM                              |
| Leads                         | crm             | CRM                              |
| Tags                          | crm             | CRM                              |
+------------------------------------------------------------------------------------+
| + Add a line                                                                       |
+------------------------------------------------------------------------------------+
```

Behavior:
- Menu items not listed here are hidden from sidebar for users in this group
- If user is in multiple groups, menus are UNION of all groups' menus
- Admin/superuser sees all menus regardless

#### Tab: Pages

Pages/views accessible to users in this group. Like menus, this is **visibility**.

```
+------------------------------------------------------------------------------------+
| Pages                                                                              |
+------------------------------------------------------------------------------------+
| Page Name              | Module          | Model           | Type                   |
|------------------------|-----------------|-----------------|------------------------|
| contact_list           | crm             | contact          | list                  |
| contact_form           | crm             | contact          | form                  |
| lead_list              | crm             | lead             | list                  |
| lead_form              | crm             | lead             | form                  |
| tag_list               | crm             | tag              | list                  |
+------------------------------------------------------------------------------------+
| + Add a line                                                                       |
+------------------------------------------------------------------------------------+
```

Behavior:
- Pages not listed → user gets 403 when navigating to that URL
- Auto-generated pages (from model auto_pages) are auto-added when security is synced
- Custom pages (dashboard, settings) must be explicitly added

#### Tab: Access Rights (MOST IMPORTANT)

Per-model permission matrix. This is the `model_access` table.
12 permissions grouped into 3 categories for readability.

```
+--------------------------------------------------------------------------------------------------------------+
| Access Rights                                                                                                |
+--------------------------------------------------------------------------------------------------------------+
|                    |              | Core                    | Action              | Data                      |
| Name               | Model        | Se  Re  Wr  Cr  De     | Pr  Em  Rp          | Ex  Im  Mk  Cl            |
|--------------------|--------------|-------------------------|---------------------|---------------------------|
| Contact Access     | contact      | ✔   ✔   ✔   ✔   ✖      | ✔   ✔   ✔           | ✔   ✖   ✖   ✔             |
| Lead Access        | lead         | ✔   ✔   ✔   ✔   ✖      | ✔   ✔   ✔           | ✔   ✖   ✖   ✔             |
| Tag Access         | tag          | ✔   ✔   ✖   ✖   ✖      | ✖   ✖   ✖           | ✖   ✖   ✖   ✖             |
+--------------------------------------------------------------------------------------------------------------+
| + Add a line                                                                                                 |
+--------------------------------------------------------------------------------------------------------------+

Column legend:
  Core:   Se=Select  Re=Read  Wr=Write  Cr=Create  De=Delete
  Action: Pr=Print   Em=Email Rp=Report
  Data:   Ex=Export  Im=Import Mk=Mask(unmask) Cl=Clone
```

Behavior:
- Each row = one `model_access` record
- Clicking a checkbox toggles the permission
- "Add a line" → dropdown to select model, then set permissions
- Permissions are **additive across groups** — if user is in CRM/User (read) and CRM/Manager (delete), user can read + delete
- Empty table = no access to any model (default-deny)
- `group_id = NULL` rows shown separately as "Global Access" section at top

#### Tab: Record Rules

Row-level security filters. This is the `record_rules` table.

```
+--------------------------------------------------------------------------------------------------------------+
| Record Rules                                                                                                 |
+--------------------------------------------------------------------------------------------------------------+
| Name                  | Model        | Domain                                    | Read | Write | Create | Del |
|-----------------------|--------------|-------------------------------------------|------|-------|--------|-----|
| Own Contacts Only     | contact      | [["created_by", "=", "{{user.id}}"]]     |  ✔   |  ✔    |  ✔     |  ✖  |
| Own Leads Only        | lead         | [["assigned_to", "=", "{{user.id}}"]]    |  ✔   |  ✔    |  ✔     |  ✖  |
+--------------------------------------------------------------------------------------------------------------+
| + Add a line                                                                                                 |
+--------------------------------------------------------------------------------------------------------------+
```

Behavior:
- Domain is a filter expression applied as WHERE clause
- `{{user.id}}` interpolated at runtime with current user's ID
- Empty domain `[]` = no filter (access all records)
- Perm checkboxes = which operations this rule applies to
- **Group rules UNION**: if user matches ANY group rule, access granted
- **Global rules INTERSECT**: ALL global rules must be satisfied
- "Add a line" → form to define: name, model, domain expression, perm flags

#### Tab: Notes

Free-text notes about the group's purpose and usage.

```
+------------------------------------------------------------------------------------+
| Notes                                                                              |
+------------------------------------------------------------------------------------+
|                                                                                    |
| Basic CRM access for sales agents.                                                |
| Users in this group can only see their own contacts and leads.                     |
| They cannot delete records or import data.                                         |
| Tag management is read-only (select + read only).                                  |
|                                                                                    |
| For full access including settings and all records, use CRM / Manager.             |
|                                                                                    |
+------------------------------------------------------------------------------------+
```

#### Group Form — Action Buttons

```
+------------------------------------------------------------------------------------+
| Action bar (top-right of form):                                                    |
|                                                                                    |
|  [Save]  [Discard]  [⚙ Actions ▼]                                                |
|                                                                                    |
|  Actions dropdown:                                                                 |
|    - Export to JSON          → download this group as single JSON file              |
|    - View History            → show ir_security_histories for this group            |
|    - Duplicate               → create copy of this group with new name             |
|    - Delete                  → delete group (with confirmation)                     |
|    - View Effective Perms    → show computed permissions including implied groups   |
+------------------------------------------------------------------------------------+
```

#### "View Effective Permissions" — Computed View

When clicked, shows the **resolved** permissions including all implied groups:

```
+------------------------------------------------------------------------------------+
| Effective Permissions for: CRM / Manager                                           |
| (includes: CRM / User → Base / User)                                              |
+------------------------------------------------------------------------------------+
|                    |              | Core                    | Data                  |
| Source Group       | Model        | Se  Re  Wr  Cr  De     | Ex  Im  Mk  Cl        |
|--------------------|--------------|-------------------------|-----------------------|
| CRM / Manager      | contact      | ✔   ✔   ✔   ✔   ✔      | ✔   ✔   ✔   ✔         |
| ↳ CRM / User       | contact      | ✔   ✔   ✔   ✔   ✖      | ✔   ✖   ✖   ✔         |
| ↳ Base / User       | user         | ✔   ✔   ✖   ✖   ✖      | ✖   ✖   ✖   ✖         |
|--------------------|--------------|-------------------------|-----------------------|
| EFFECTIVE          | contact      | ✔   ✔   ✔   ✔   ✔      | ✔   ✔   ✔   ✔         |
| EFFECTIVE          | user         | ✔   ✔   ✖   ✖   ✖      | ✖   ✖   ✖   ✖         |
+------------------------------------------------------------------------------------+
```

This is read-only, computed at display time by resolving the full implied chain.

### 5.6 Security Sync UI

```
/admin/securities

Actions:
  [Load from Files]     → JSON → DB (with preview/diff)
  [Export to Files]      → DB → JSON (write to filesystem)
  [Upload JSON/ZIP]     → upload → parse → preview → apply
  [Download All (ZIP)]  → download all securities as ZIP
  [Download Module ▼]   → dropdown: select module → download ZIP

History:
| Date                | Entity      | Action | Changes              | User  | Source |
|---------------------|-------------|--------|----------------------|-------|--------|
| 2026-04-27 10:30:00 | crm.user    | update | can_delete: F→T      | admin | ui     |
| 2026-04-27 09:00:00 | crm.manager | create | (new)                | system| json   |

[Rollback] button per history entry
```

---

## 6. Part E: CLI Commands

### 6.1 Security Commands

```bash
# Sync JSON → DB
bitcode security load crm              # load module securities
bitcode security load crm --force      # overwrite admin changes
bitcode security load --all            # load all modules

# Sync DB → JSON
bitcode security export crm            # export to securities/*.json
bitcode security export --all

# Diff
bitcode security diff crm             # show DB vs JSON differences

# Validate
bitcode security validate crm         # validate JSON files

# History
bitcode security history              # list recent changes
bitcode security history --entity=crm.user
bitcode security rollback <history_id>
```

### 6.2 CRUD Generate Commands

```bash
# Generate override files from auto-generated snapshot
bitcode publish:crud crm contact all          # api + all pages
bitcode publish:crud crm contact api          # api only
bitcode publish:crud crm contact pages        # all pages
bitcode publish:crud crm contact pages list   # list page only
bitcode publish:crud crm contact pages form   # form page only
```

Output:
```
Created apis/contact_api.json (from auto-generated)
Created pages/contact_list.json (from auto-generated)
Created pages/contact_form.json (from auto-generated)
```

---

## 7. Part F: Migration Plan

### 7.1 Database Migration

```sql
-- Step 1: Add new columns to groups
ALTER TABLE groups ADD COLUMN share BOOLEAN DEFAULT FALSE;
ALTER TABLE groups ADD COLUMN comment TEXT;
ALTER TABLE groups ADD COLUMN module TEXT;
ALTER TABLE groups ADD COLUMN modified_source TEXT DEFAULT 'json';

-- Step 2: Create model_access table
CREATE TABLE model_access (...);

-- Step 3: Create ir_security_histories table
CREATE TABLE ir_security_histories (...);

-- Step 4: Create group_menus table
CREATE TABLE group_menus (...);

-- Step 5: Create group_pages table
CREATE TABLE group_pages (...);

-- Step 6: Create record_rule_groups m2m table
CREATE TABLE record_rule_groups (...);

-- Step 7: Add is_superuser to users
ALTER TABLE users ADD COLUMN is_superuser BOOLEAN DEFAULT FALSE;

-- Step 8: Migrate Role permissions → ModelAccess
-- For each role with permissions:
--   Create corresponding group (if not exists)
--   Create ModelAccess entries from role permissions
--   Migrate user_roles → user_groups

-- Step 9: Migrate record_rules.group_names → record_rule_groups m2m

-- Step 10: Add modified_source to record_rules
ALTER TABLE record_rules ADD COLUMN module TEXT;
ALTER TABLE record_rules ADD COLUMN modified_source TEXT DEFAULT 'json';

-- Step 11: Drop old tables (after verification)
-- DROP TABLE role_permissions;
-- DROP TABLE role_inherits;
-- DROP TABLE user_roles;
-- DROP TABLE roles;
-- DROP TABLE permissions;
```

### 7.2 Code Migration

| File | Action |
|------|--------|
| `domain/security/role.go` | DELETE |
| `domain/security/permission.go` | DELETE |
| `domain/security/group.go` | UPGRADE — add share, comment, module, modified_source |
| `domain/security/record_rule.go` | UPGRADE — m2m groups, remove GroupNames string |
| `domain/security/user.go` | UPGRADE — remove Roles, add IsSuperuser |
| `domain/security/model_access.go` | CREATE — new entity |
| `domain/security/security_history.go` | CREATE — new entity |
| `compiler/parser/model.go` | UPGRADE — add `api` field, `mask`, `mask_length`, `groups` on fields |
| `compiler/parser/module.go` | UPGRADE — add `securities`, rename `views`→`pages`, remove `permissions`/`groups` |
| `compiler/parser/security.go` | CREATE — parse securities/*.json |
| `presentation/middleware/permission.go` | UPGRADE — implement PermissionChecker using ModelAccess |
| `presentation/middleware/record_rule.go` | UPGRADE — implement Global ∩ Group composition |
| `presentation/api/crud_handler.go` | UPGRADE — field masking, field groups filtering, permission injection |
| `presentation/api/router.go` | UPGRADE — wire permission middleware, auto-generate from model |
| `presentation/view/component_compiler.go` | UPGRADE — CompileList uses bc-datatable, pass permissions |
| `presentation/view/renderer.go` | UPGRADE — auto-generate pages from model |
| `presentation/admin/admin.go` | UPGRADE — /admin/models/{module}/{model}, group admin pages, security sync UI |
| `infrastructure/module/loader.go` | UPGRADE — load securities/*.json |
| `app.go` | UPGRADE — loading order, security sync, permission wiring |

### 7.3 Component Migration

| Component | Action |
|-----------|--------|
| `bc-datatable` | UPGRADE — permission props, modal mode, row actions, navigation |
| `bc-view-form` | UPGRADE — permission props |
| `bc-view-list` | DEPRECATE — keep for backward compat |
| `component_compiler.go` | UPGRADE — CompileList emits bc-datatable |

---

## 8. Implementation Plan

### Phase 1: Foundation (Week 1-2)

**Goal:** New security entities in DB, parser for securities/*.json, basic sync.

| # | Task | Files | Effort |
|---|------|-------|--------|
| 1.1 | Create `ModelAccess` domain entity | `domain/security/model_access.go` | S |
| 1.2 | Create `SecurityHistory` domain entity | `domain/security/security_history.go` | S |
| 1.3 | Upgrade `Group` entity — add share, comment, module, modified_source | `domain/security/group.go` | S |
| 1.4 | Upgrade `RecordRule` entity — m2m groups | `domain/security/record_rule.go` | M |
| 1.5 | Upgrade `User` entity — add is_superuser, remove Roles | `domain/security/user.go` | S |
| 1.6 | Create security JSON parser | `compiler/parser/security.go` | M |
| 1.7 | Upgrade module parser — add `securities`, rename views→pages | `compiler/parser/module.go` | S |
| 1.8 | Upgrade model parser — add `api` field, `mask`, `mask_length`, `groups` on fields | `compiler/parser/model.go` | M |
| 1.9 | DB migration — create new tables, alter existing | `infrastructure/persistence/` | M |
| 1.10 | Security loader — load securities/*.json, sync to DB | `infrastructure/module/security_loader.go` | L |
| 1.11 | Write tests for all new entities and parser | `*_test.go` | M |

### Phase 2: Permission Enforcement (Week 2-3)

**Goal:** Permissions actually enforced at runtime.

| # | Task | Files | Effort |
|---|------|-------|--------|
| 2.1 | Implement PermissionChecker — query ModelAccess via Group chain | `infrastructure/persistence/permission_checker.go` | L |
| 2.2 | Implement RecordRuleEngine — Global ∩ Group composition | `infrastructure/persistence/record_rule_engine.go` | L |
| 2.3 | Wire PermissionMiddleware in route registration | `presentation/api/router.go` | M |
| 2.4 | Wire RecordRuleMiddleware in route registration | `presentation/api/router.go` | M |
| 2.5 | CRUD handler — field masking (server-side) | `presentation/api/crud_handler.go` | M |
| 2.6 | CRUD handler — field groups filtering (server-side) | `presentation/api/crud_handler.go` | M |
| 2.7 | CRUD handler — inject permissions into page render context | `presentation/api/crud_handler.go` | S |
| 2.8 | Superuser bypass logic | `presentation/middleware/` | S |
| 2.9 | Fallback: auto-grant admin when no ACL exists | `presentation/api/router.go` | S |
| 2.10 | Write tests for permission check, record rules, masking | `*_test.go` | L |

### Phase 3: Convention-Driven CRUD (Week 3-4)

**Goal:** Model with `"api": true` auto-generates REST API + Pages.

| # | Task | Files | Effort |
|---|------|-------|--------|
| 3.1 | Auto-generate REST endpoints from model (no api.json needed) | `presentation/api/router.go` | L |
| 3.2 | URL convention: `/api/v1/{module}/{model_plural}` | `presentation/api/router.go` | M |
| 3.3 | API override merge logic (apis/*.json overrides auto-generated) | `presentation/api/router.go` | M |
| 3.4 | Auto-generate list page from model fields | `presentation/view/auto_page_generator.go` | L |
| 3.5 | Auto-generate form page from model fields | `presentation/view/auto_page_generator.go` | L |
| 3.6 | Page URL convention: `/{module}/{model_plural}` | `presentation/view/renderer.go` | M |
| 3.7 | Page override logic (pages/*.json overrides auto-generated) | `presentation/view/renderer.go` | M |
| 3.8 | Cross-module reference (`"module"` field in api/pages) | `infrastructure/module/loader.go` | M |
| 3.9 | Loading order: models → securities → apis → pages | `app.go` | M |
| 3.10 | Rename views → pages throughout codebase | Multiple files | M |
| 3.11 | Write tests | `*_test.go` | L |

### Phase 4: Component Upgrades (Week 4-5)

**Goal:** `bc-datatable` is permission-aware, supports modal CRUD.

| # | Task | Files | Effort |
|---|------|-------|--------|
| 4.1 | `bc-datatable` — add permission props | `bc-datatable.tsx` | M |
| 4.2 | `bc-datatable` — permission-aware toolbar (New, Export, Import buttons) | `bc-datatable.tsx` | M |
| 4.3 | `bc-datatable` — permission-aware bulk actions (Delete, Clone, Export) | `bc-datatable.tsx` | S |
| 4.4 | `bc-datatable` — row actions (edit, delete, clone, print, email icons) | `bc-datatable.tsx` | M |
| 4.5 | `bc-datatable` — navigation props (createUrl, editUrl, detailUrl) | `bc-datatable.tsx` | S |
| 4.6 | `bc-datatable` — modal mode (CRUD in modal) | `bc-datatable.tsx` | L |
| 4.7 | `bc-datatable` — field masking display | `bc-datatable.tsx` | S |
| 4.8 | `bc-view-form` — permission props | `bc-view-form.tsx` | M |
| 4.9 | `CompileList()` — switch to emit `bc-datatable` | `component_compiler.go` | M |
| 4.10 | Write component tests | `*.spec.tsx` | M |

### Phase 5: Admin UI (Week 5-7)

**Goal:** Full admin UI for groups, security sync, model API config.

| # | Task | Files | Effort |
|---|------|-------|--------|
| 5.1 | Admin URL change: `/admin/models/{module}/{model}` | `admin/admin.go` | M |
| 5.2 | Model admin — "API" tab (configure api field in model JSON) | `admin/admin.go` | M |
| 5.3 | Model admin — Fields table: add GROUPS and MASK columns | `admin/admin.go` | S |
| 5.4 | Group list page `/admin/groups` | `admin/admin.go` | M |
| 5.5 | Group detail page with all tabs (Users, Inherited, Menus, Pages, Access Rights, Record Rules, Notes) | `admin/admin.go` | XL |
| 5.6 | Security sync page — Load/Export/Upload/Download | `admin/admin.go` | L |
| 5.7 | Security history page — list + rollback | `admin/admin.go` | M |
| 5.8 | Upload JSON/ZIP endpoint | `presentation/api/security_handler.go` | M |
| 5.9 | Download JSON/ZIP endpoint | `presentation/api/security_handler.go` | M |
| 5.10 | Export to filesystem endpoint | `presentation/api/security_handler.go` | M |
| 5.11 | Conflict detection on load | `infrastructure/module/security_loader.go` | M |

### Phase 6: CLI + Swagger + Polish (Week 7-8)

**Goal:** CLI commands, auto Swagger, cleanup.

| # | Task | Files | Effort |
|---|------|-------|--------|
| 6.1 | `bitcode security load/export/diff/validate/history/rollback` CLI | `cmd/bitcode/` | L |
| 6.2 | `bitcode publish:crud` CLI | `cmd/bitcode/` | M |
| 6.3 | Auto Swagger/OpenAPI generation | `presentation/api/swagger.go` | L |
| 6.4 | Swagger UI serving at `/api/v1/docs` | `app.go` | S |
| 6.5 | Delete old Role/Permission code | `domain/security/` | S |
| 6.6 | Update all existing modules (base, crm, sales) to new format | `modules/`, `embedded/` | L |
| 6.7 | Update documentation | `docs/` | M |
| 6.8 | Full integration tests | `*_test.go` | L |

### Phase 7: Multi-Protocol (Week 8-10, optional)

**Goal:** GraphQL and WebSocket CRUD.

| # | Task | Files | Effort |
|---|------|-------|--------|
| 7.1 | GraphQL schema generator from model definitions | `presentation/graphql/` | XL |
| 7.2 | GraphQL resolvers (reuse CRUD handler logic) | `presentation/graphql/` | XL |
| 7.3 | WebSocket CRUD protocol (request/reply over WS) | `presentation/websocket/` | L |
| 7.4 | Permission enforcement for GraphQL + WS | `presentation/middleware/` | M |

---

## Appendix A: Complete Example — CRM Module (New Format)

### module.json
```json
{
  "name": "crm",
  "version": "1.0.0",
  "label": "CRM",
  "depends": ["base"],
  "category": "Sales",
  "table": { "prefix": "crm" },
  "models": ["models/*.json"],
  "securities": ["securities/*.json"],
  "apis": ["apis/*.json"],
  "pages": ["pages/*.json"],
  "processes": ["processes/*.json"],
  "migrations": ["migrations/*.json"],
  "i18n": ["i18n/*.json"],
  "menu": [
    { "label": "CRM", "icon": "users", "children": [
      { "label": "Contacts", "page": "contact_list", "groups": ["crm.user"] },
      { "label": "Leads", "page": "lead_list", "groups": ["crm.user"] },
      { "label": "Tags", "page": "tag_list", "groups": ["crm.user"] },
      { "label": "Settings", "page": "crm_settings", "groups": ["crm.manager"] }
    ]}
  ]
}
```

### models/contact.json
```json
{
  "name": "contact",
  "module": "crm",
  "label": "Contact",
  "api": {
    "auto_crud": true,
    "auto_pages": true,
    "protocols": { "rest": true, "graphql": true }
  },
  "fields": {
    "name":    { "type": "string", "required": true, "max": 200 },
    "email":   { "type": "email" },
    "phone":   { "type": "string", "max": 20, "mask": true, "mask_length": 4 },
    "company": { "type": "string", "max": 200 },
    "notes":   { "type": "text" },
    "tags":    { "type": "many2many", "model": "tag" },
    "salary":  { "type": "decimal", "groups": ["crm.manager"] }
  },
  "title_field": "name",
  "search_field": ["name", "email", "company"]
}
```

### models/tag.json
```json
{
  "name": "tag",
  "module": "crm",
  "label": "Tag",
  "api": {
    "auto_crud": true,
    "auto_pages": true,
    "modal": true
  },
  "fields": {
    "name":  { "type": "string", "required": true, "max": 50 },
    "color": { "type": "selection", "options": ["red", "blue", "green", "yellow", "purple"] }
  }
}
```

### securities/crm_user.json
```json
{
  "name": "crm.user",
  "label": "CRM / User",
  "category": "CRM",
  "implies": ["base.user"],
  "share": false,
  "access": {
    "contact": ["select", "read", "write", "create", "print", "email", "report", "export", "clone"],
    "lead":    ["select", "read", "write", "create", "print", "email", "report", "export", "clone"],
    "tag":     ["select", "read"]
  },
  "rules": [
    {
      "name": "crm_user_own_contacts",
      "model": "contact",
      "domain": [["created_by", "=", "{{user.id}}"]],
      "perm_read": true,
      "perm_write": true,
      "perm_create": true,
      "perm_delete": false
    },
    {
      "name": "crm_user_own_leads",
      "model": "lead",
      "domain": [["assigned_to", "=", "{{user.id}}"]],
      "perm_read": true,
      "perm_write": true,
      "perm_create": true,
      "perm_delete": false
    }
  ],
  "menus": ["crm/contacts", "crm/leads", "crm/tags"],
  "pages": ["contact_list", "contact_form", "lead_list", "lead_form", "tag_list"],
  "comment": "Basic CRM access for sales agents. Can only see own contacts and leads."
}
```

### securities/crm_manager.json
```json
{
  "name": "crm.manager",
  "label": "CRM / Manager",
  "category": "CRM",
  "implies": ["crm.user"],
  "access": {
    "contact": "all",
    "lead":    "all",
    "tag":     "all"
  },
  "rules": [
    {
      "name": "crm_manager_all_contacts",
      "model": "contact",
      "domain": []
    },
    {
      "name": "crm_manager_all_leads",
      "model": "lead",
      "domain": []
    }
  ],
  "menus": ["crm/contacts", "crm/leads", "crm/tags", "crm/settings"],
  "pages": ["contact_list", "contact_form", "lead_list", "lead_form", "tag_list", "crm_dashboard", "crm_settings"],
  "comment": "Full CRM access including settings, all records, and tag management."
}
```

### pages/contact_form.json (OPTIONAL — override auto-generated)
```json
{
  "name": "contact_form",
  "type": "form",
  "module": "crm",
  "model": "contact",
  "title": "Contact",
  "layout": [
    { "row": [
      { "field": "name", "width": 6 },
      { "field": "email", "width": 6 }
    ]},
    { "row": [
      { "field": "phone", "width": 4 },
      { "field": "company", "width": 8 }
    ]},
    { "tabs": [
      { "label": "Notes", "fields": ["notes"] },
      { "label": "Tags", "fields": ["tags"] }
    ]}
  ],
  "actions": [
    { "label": "Send Email", "process": "send_contact_email", "variant": "secondary", "visible": "email != ''" }
  ]
}
```

### apis/contact_merge.json (OPTIONAL — custom endpoint)
```json
{
  "name": "contact_merge_api",
  "module": "crm",
  "model": "contact",
  "endpoints": [
    {
      "method": "POST",
      "path": "/:id/merge",
      "action": "merge",
      "process": "merge_contacts",
      "permissions": ["crm.contact.write"]
    }
  ]
}
```

---

## Appendix B: API Endpoints Reference

### Security Management
```
GET    /admin/api/groups                    list groups
GET    /admin/api/groups/:id                get group detail
POST   /admin/api/groups                    create group
PUT    /admin/api/groups/:id                update group
DELETE /admin/api/groups/:id                delete group

POST   /admin/api/securities/load           JSON → DB
POST   /admin/api/securities/export         DB → JSON
POST   /admin/api/securities/upload         upload JSON/ZIP → DB
GET    /admin/api/securities/download       download ZIP
GET    /admin/api/securities/diff           DB vs JSON diff
POST   /admin/api/securities/validate       validate JSON files

GET    /admin/api/securities/history        list changes
POST   /admin/api/securities/rollback/:id   rollback to snapshot
```

### Auto-Generated CRUD (per model with api: true)
```
GET    /api/v1/{module}/{model_plural}              list
GET    /api/v1/{module}/{model_plural}/:id           read
POST   /api/v1/{module}/{model_plural}               create
PUT    /api/v1/{module}/{model_plural}/:id            update
DELETE /api/v1/{module}/{model_plural}/:id            delete
POST   /api/v1/{module}/{model_plural}/:id/clone      clone
POST   /api/v1/{module}/{model_plural}/onchange       onchange
```

### Auto-Generated Pages (per model with auto_pages: true)
```
GET    /{module}/{model_plural}                      list page
GET    /{module}/{model_plural}/new                  create form
GET    /{module}/{model_plural}/:id                  detail view
GET    /{module}/{model_plural}/:id/edit             edit form
```

### Swagger
```
GET    /api/v1/docs                                  Swagger UI
GET    /api/v1/docs/openapi.json                     OpenAPI spec
```
