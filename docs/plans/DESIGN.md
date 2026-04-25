# BitCode Engine — Design & Implementation

**Date:** 2026-04-17
**Principle:** Simple, Easy, Powerful
**Inspired by:** Yao Engine, Odoo, Frappe

---

## 1. Overview

A modular BitCode platform where you build applications by writing JSON definitions. The Go engine reads those definitions and produces a running app — complete with APIs, business logic, UI, background jobs, and security.

**Core idea:** JSON is the source code. Go is the runtime. Modules are the packaging.

```
┌──────────────────────────────────────────────┐
│              bitcode engine (Go)              │
│                                               │
│  JSON Definitions ──► Parser ──► Runtime      │
│  (model, api, process, view, agent)           │
│                                               │
│  ┌─────────┐  ┌──────────┐  ┌────────────┐  │
│  │ Module   │  │ Security │  │  Plugin    │  │
│  │ System   │  │ Layer    │  │  Runtime   │  │
│  └─────────┘  └──────────┘  └────────────┘  │
└──────────────────────────────────────────────┘
```

### Design Principles

1. **Convention over configuration** — Sensible defaults. Only configure what you need to change.
2. **Progressive complexity** — Simple things are simple. Complex things are possible.
3. **DDD internally, simple externally** — The engine uses DDD patterns internally. Users write flat JSON.
4. **Module = unit of deployment** — Everything lives in a module. Modules can depend on other modules.

### Tech Stack

| Layer | Technology | Why |
|-------|-----------|-----|
| Runtime | Go 1.21+ | Fast, single binary, great concurrency |
| HTTP | Fiber | Express-like, fast |
| ORM | GORM | Mature, auto-migration |
| Database | PostgreSQL | JSONB, reliable, free |
| Cache | Redis | Fast, pub/sub capable |
| Events | In-process bus (MVP) | Simple. Upgrade to NATS later if needed |
| Plugins | JSON-RPC over stdin/stdout | Simple IPC. Upgrade to gRPC later if needed |
| Templates | Go html/template | Built-in, zero dependency, fast |
| SPA (optional) | Mitosis | Framework-agnostic. Optional, not default |
| CLI | Cobra | Standard Go CLI framework |

### What Changed from Previous Design (Simplification)

| Before (over-engineered) | After (simplified) |
|---|---|
| Full DDD in user-facing JSON (aggregate_root, bounded_context, value_objects) | DDD is internal only. User JSON is flat and simple |
| CQRS + Event Sourcing + Saga | Simple repository + domain events. Add CQRS later if needed |
| gRPC for plugin communication | JSON-RPC over stdin/stdout. Simpler, good enough for MVP |
| NATS message queue | In-process event bus. Add NATS later for distributed |
| Handlebars + Mitosis + Hybrid SSR/SPA (3 approaches) | Go html/template (SSR default) + optional Mitosis for SPA |
| 100 tasks, 13 phases | ~60 tasks, 8 phases |
| No relationships in model | Full relationship support (many2one, one2many, many2many) |
| No conditional logic in process | if/else, switch, loop support |
| No workflow/state machine | Built-in workflow definition |
| No i18n, no cron, no computed fields | All included |

---

## 2. JSON Definitions

All definitions follow the principle: **minimal required fields, everything else has defaults.**

### 2.1 Model

```json
{
  "name": "order",
  "module": "sales",
  "label": "Sales Order",
  "fields": {
    "customer_id": { "type": "many2one", "model": "customer", "required": true, "label": "Customer" },
    "order_date":  { "type": "date", "default": "now", "label": "Order Date" },
    "status":      { "type": "selection", "options": ["draft", "confirmed", "done", "cancelled"], "default": "draft" },
    "total":       { "type": "decimal", "computed": "sum(lines.subtotal)", "label": "Total" },
    "notes":       { "type": "text" },
    "lines":       { "type": "one2many", "model": "order_line", "inverse": "order_id" }
  },
  "record_rules": [
    { "groups": ["sales.user"], "domain": [["created_by", "=", "{{user.id}}"]] },
    { "groups": ["sales.manager"], "domain": [] }
  ],
  "indexes": [
    ["customer_id", "order_date"]
  ]
}
```

**Field types:**

| Type | Description | Example |
|------|-------------|---------|
| string | Text (varchar) | "name": {"type": "string", "max": 100} |
| text | Long text | "notes": {"type": "text"} |
| integer | Integer | "quantity": {"type": "integer", "min": 0} |
| decimal | Decimal number | "price": {"type": "decimal", "precision": 2} |
| boolean | True/false | "active": {"type": "boolean", "default": true} |
| date | Date | "order_date": {"type": "date"} |
| datetime | Date + time | "created_at": {"type": "datetime", "auto": true} |
| selection | Enum | "status": {"type": "selection", "options": [...]} |
| email | Email (validated) | "email": {"type": "email", "unique": true} |
| many2one | FK to another model | "customer_id": {"type": "many2one", "model": "customer"} |
| one2many | Reverse FK | "lines": {"type": "one2many", "model": "order_line", "inverse": "order_id"} |
| many2many | Junction table | "tags": {"type": "many2many", "model": "tag"} |
| json | Arbitrary JSON | "metadata": {"type": "json"} |
| file | File attachment | "attachment": {"type": "file", "max_size": "10MB"} |
| computed | Virtual field | "total": {"type": "decimal", "computed": "sum(lines.subtotal)"} |

**Convention defaults (auto-added, you never write these):**
- id (uuid, primary key)
- created_at, updated_at (datetime)
- created_by, updated_by (many2one to user)
- active (boolean, default true) — for soft delete

**Model inheritance (extend another module's model):**
```json
{
  "name": "customer",
  "inherit": "crm.contact",
  "fields": {
    "credit_limit": { "type": "decimal", "default": 0 },
    "payment_terms": { "type": "selection", "options": ["net30", "net60", "net90"] }
  }
}
```

### 2.2 API

**`auth: true` implies RLS.** When authentication is on, record rules from the model are always enforced — no separate flag needed. If a model has no record_rules defined, all authenticated users see all records (open access within auth).

**Public (no auth, no RLS):**
```json
{
  "name": "tag_api",
  "model": "tag",
  "auto_crud": true
}
```
Generates GET/POST/PUT/DELETE. No auth. Good for public lookup tables.

**Standard (auto CRUD + auth + RLS automatic):**
```json
{
  "name": "customer_api",
  "model": "customer",
  "auto_crud": true,
  "auth": true
}
```
Every request goes through:
1. **Auth** — JWT validation, reject 401 if no token
2. **Permission** — auto-derived from model name: `customer.read`, `customer.create`, `customer.write`, `customer.delete`
3. **RLS** — record rules from model's `record_rules` applied automatically (no opt-in needed)

**With approval workflow:**
```json
{
  "name": "order_api",
  "model": "order",
  "auto_crud": true,
  "auth": true,
  "workflow": "order_workflow",
  "actions": {
    "confirm":  { "transition": "confirm",  "permission": "order.confirm" },
    "complete": { "transition": "complete", "permission": "order.complete" },
    "cancel":   { "transition": "cancel",   "permission": "order.cancel" }
  }
}
```
This generates all standard CRUD endpoints **plus** workflow action endpoints:
- `POST /api/orders/:id/confirm` — triggers "confirm" transition
- `POST /api/orders/:id/complete` — triggers "complete" transition
- `POST /api/orders/:id/cancel` — triggers "cancel" transition

Each action endpoint:
1. Validates the transition is allowed from current state (e.g., only `draft` → `confirmed`)
2. Checks the user has the required permission
3. RLS — can this user access this record?
4. Runs the associated process if defined in the workflow
5. Updates the status field
6. Emits a domain event (e.g., `order.confirmed`)

**How auth, RLS, workflow flow together:**

```
POST /api/orders (create)
  ├─► Auth middleware (JWT)
  ├─► Permission middleware (order.create)
  ├─► Validate input against model schema
  ├─► Set status = workflow's initial state ("draft")
  ├─► Create record
  └─► Return 201

GET /api/orders (list)
  ├─► Auth middleware (JWT)
  ├─► Permission middleware (order.read)
  ├─► RLS (inject model's record_rules as WHERE clause)
  ├─► Query with filters, sorting, pagination
  └─► Return 200 (only records user is allowed to see)

POST /api/orders/:id/confirm (workflow action)
  ├─► Auth middleware (JWT)
  ├─► Permission middleware (order.confirm)
  ├─► RLS (can user access this record?)
  ├─► Workflow engine: validate transition draft → confirmed
  ├─► Run process "confirm_order" if defined
  ├─► Update status = "confirmed"
  ├─► Emit event "order.confirmed"
  └─► Return 200
```

**The rule is simple:** `auth: true` = permissions + RLS always on. Record rules are defined on the **model**, not the API. The API just enforces whatever the model declares.

**Full control (manual endpoints):**
```json
{
  "name": "report_api",
  "base_path": "/api/reports",
  "auth": true,
  "endpoints": [
    { "method": "GET", "path": "/sales-summary", "handler": "processes/sales_summary.json", "permissions": ["report.read"] },
    { "method": "GET", "path": "/top-customers", "handler": "processes/top_customers.json", "permissions": ["report.read"] }
  ]
}
```
For non-CRUD APIs (reports, aggregations, custom logic). Full manual control.

**API options summary:**

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| auto_crud | bool | false | Generate standard CRUD endpoints |
| auth | bool | false | Require JWT authentication |
| rls | bool | false | Apply record rules (row-level security) |
| workflow | string | null | Link to workflow definition for state transitions |
| actions | object | null | Workflow action endpoints (transition + permission) |
| soft_delete | bool | true | DELETE sets active=false instead of hard delete |
| pagination | object | {page_size: 20, max: 100} | Default pagination settings |
| search | string[] | null | Fields to include in full-text search |

### 2.3 Process (Business Logic)

```json
{
  "name": "confirm_order",
  "steps": [
    { "type": "validate", "rules": { "status": { "eq": "draft" } }, "error": "Only draft orders can be confirmed" },
    { "type": "update",   "set": { "status": "confirmed" } },
    { "type": "if",       "condition": "{{input.total > 10000}}", "then": "notify_manager", "else": "send_confirmation" },
    { "type": "emit",     "event": "order.confirmed" }
  ]
}
```

**Step types:**

| Type | Description |
|------|-------------|
| validate | Check conditions, return error if failed |
| query | Read data from database |
| create | Create a new record |
| update | Update current record |
| delete | Delete a record |
| if | Conditional branching |
| switch | Multi-way branching |
| loop | Iterate over a list |
| emit | Emit a domain event |
| call | Call another process |
| script | Run custom TypeScript/Python code |
| http | Call external HTTP API |
| assign | Set a variable in context |
| log | Write to audit log |

### 2.4 Agent (Background Jobs & Cron)

```json
{
  "name": "order_notifications",
  "triggers": [
    {
      "event": "order.confirmed",
      "action": "send_confirmation_email",
      "script": "scripts/send_email.ts"
    }
  ],
  "cron": [
    {
      "schedule": "0 9 * * *",
      "action": "daily_order_report",
      "script": "scripts/daily_report.ts"
    }
  ],
  "retry": { "max": 3, "backoff": "exponential" }
}
```

### 2.5 View (UI)

**List view:**
```json
{
  "name": "order_list",
  "type": "list",
  "model": "order",
  "title": "Sales Orders",
  "fields": ["order_date", "customer_id", "total", "status"],
  "filters": ["status", "order_date", "customer_id"],
  "sort": { "field": "order_date", "order": "desc" },
  "actions": [
    { "label": "Confirm", "process": "confirm_order", "permission": "order.confirm", "visible": "status == 'draft'" }
  ]
}
```

**Form view:**
```json
{
  "name": "order_form",
  "type": "form",
  "model": "order",
  "title": "Sales Order",
  "layout": [
    { "row": [
      { "field": "customer_id", "width": 6 },
      { "field": "order_date", "width": 3 },
      { "field": "status", "width": 3, "readonly": true }
    ]},
    { "tabs": [
      { "label": "Lines", "view": "order_line_list" },
      { "label": "Notes", "fields": ["notes"] }
    ]},
    { "row": [
      { "field": "total", "readonly": true }
    ]}
  ],
  "actions": [
    { "label": "Confirm", "process": "confirm_order", "variant": "primary", "visible": "status == 'draft'" },
    { "label": "Cancel", "process": "cancel_order", "variant": "danger", "confirm": "Are you sure?" }
  ]
}
```

**View types:**

| Type | Description |
|------|-------------|
| list | Table view with sorting, filtering, pagination |
| form | Detail/edit form with layout |
| kanban | Kanban board (grouped by field) |
| calendar | Calendar view |
| chart | Dashboard chart |
| custom | Custom template (escape hatch) |

**Custom template (escape hatch for full control):**
```json
{
  "name": "order_dashboard",
  "type": "custom",
  "template": "templates/order_dashboard.html",
  "data_sources": {
    "orders": { "model": "order", "domain": [["status", "=", "confirmed"]] },
    "stats": { "process": "get_order_stats" }
  }
}
```

Template file uses Go html/template:
```html
{{/* templates/order_dashboard.html */}}
<div class="dashboard">
  <div class="stats-row">
    {{template "partials/stat_card.html" dict "title" "Total Orders" "value" .stats.total}}
    {{template "partials/stat_card.html" dict "title" "Revenue" "value" (formatCurrency .stats.revenue)}}
  </div>

  <h2>Recent Orders</h2>
  <table>
    {{range .orders}}
    <tr>
      <td>{{.customer_id.name}}</td>
      <td>{{formatDate .order_date}}</td>
      <td>{{formatCurrency .total}}</td>
      <td>{{template "partials/status_badge.html" .}}</td>
    </tr>
    {{end}}
  </table>
</div>
```

Partial view:
```html
{{/* partials/status_badge.html */}}
<span class="badge badge-{{.status}}">{{.status}}</span>
```

### 2.6 Module

```json
{
  "name": "sales",
  "version": "1.0.0",
  "label": "Sales Management",
  "depends": ["base", "crm"],
  "category": "Sales",
  "models": ["models/*.json"],
  "apis": ["apis/*.json"],
  "processes": ["processes/*.json"],
  "agents": ["agents/*.json"],
  "views": ["views/*.json"],
  "templates": ["templates/*.html"],
  "scripts": ["scripts/*"],
  "data": ["data/*.json"],
  "i18n": ["i18n/*.json"],
  "permissions": {
    "order.read":    "Read orders",
    "order.create":  "Create orders",
    "order.write":   "Edit orders",
    "order.delete":  "Delete orders",
    "order.confirm": "Confirm orders"
  },
  "groups": {
    "sales.user":    { "label": "Sales / User",    "implies": ["base.user"] },
    "sales.manager": { "label": "Sales / Manager", "implies": ["sales.user"] }
  },
  "menu": [
    { "label": "Sales", "icon": "shopping-cart", "children": [
      { "label": "Orders",    "view": "order_list" },
      { "label": "Customers", "view": "customer_list" }
    ]}
  ],
  "settings": {
    "default_currency": { "type": "string", "default": "USD" },
    "auto_confirm_threshold": { "type": "decimal", "default": 0 }
  }
}
```

### 2.7 Workflow (State Machine)

```json
{
  "name": "order_workflow",
  "model": "order",
  "field": "status",
  "states": {
    "draft":     { "label": "Draft" },
    "confirmed": { "label": "Confirmed" },
    "done":      { "label": "Done" },
    "cancelled": { "label": "Cancelled" }
  },
  "transitions": [
    { "from": "draft",     "to": "confirmed", "action": "confirm", "permission": "order.confirm" },
    { "from": "confirmed", "to": "done",      "action": "complete", "permission": "order.write" },
    { "from": ["draft", "confirmed"], "to": "cancelled", "action": "cancel", "permission": "order.write" }
  ]
}
```

### 2.8 i18n

```json
{
  "locale": "id",
  "translations": {
    "Sales Order": "Pesanan Penjualan",
    "Customer": "Pelanggan",
    "Confirm": "Konfirmasi",
    "Draft": "Draf",
    "Total": "Total"
  }
}
```

---

## 3. Architecture

### 3.1 Project Structure (Engine)

```
bitcode-framework/bitcode/
├── cmd/
│   ├── engine/main.go              # Server entry point
│   └── bitcode/main.go             # CLI entry point
├── internal/
│   ├── compiler/                   # JSON → internal representation
│   │   └── parser/
│   │       ├── model.go
│   │       ├── api.go
│   │       ├── process.go
│   │       ├── view.go
│   │       ├── agent.go
│   │       ├── module.go
│   │       └── workflow.go
│   ├── domain/                     # Core domain (DDD internal only)
│   │   ├── model/                  # Dynamic model registry
│   │   ├── security/               # User, Role, Group, Permission, RecordRule
│   │   └── event/                  # Domain events
│   ├── runtime/                    # Execution engine
│   │   ├── executor/               # Process step executor
│   │   ├── agent/                  # Background job runner
│   │   ├── workflow/               # State machine engine
│   │   └── plugin/                 # Plugin manager (JSON-RPC)
│   ├── infrastructure/
│   │   ├── persistence/            # GORM repositories
│   │   ├── cache/                  # Redis cache
│   │   └── module/                 # Module loader, registry, dependency resolver
│   └── presentation/
│       ├── api/                    # Dynamic route registration
│       ├── view/                   # View renderer (SSR)
│       ├── template/               # Template engine + partials + helpers
│       └── middleware/             # Auth, permission, record_rule, audit
├── pkg/
│   ├── security/                   # JWT, password hashing
│   └── plugin/                     # Plugin SDK (for TS/Python plugins)
├── modules/
│   └── base/                       # Built-in base module
│       ├── module.json
│       ├── models/                 # user, role, group, permission, record_rule, audit_log, setting
│       ├── apis/                   # auth, user, role, group
│       ├── processes/              # login, register, check_permission
│       ├── views/                  # user_list, user_form, login_form
│       ├── templates/              # layouts, partials
│       └── data/                   # default roles, groups, admin user
└── plugins/
    └── typescript/                 # TypeScript plugin runtime
```

### 3.2 Module Structure (User Project)

```
my-erp/
├── modules/
│   ├── crm/
│   │   ├── module.json
│   │   ├── models/
│   │   │   ├── contact.json
│   │   │   └── lead.json
│   │   ├── apis/
│   │   │   └── contact_api.json
│   │   ├── processes/
│   │   │   └── convert_lead.json
│   │   ├── views/
│   │   │   ├── contact_list.json
│   │   │   └── contact_form.json
│   │   ├── templates/
│   │   │   └── partials/
│   │   ├── scripts/
│   │   │   └── enrich_contact.ts
│   │   ├── i18n/
│   │   │   └── id.json
│   │   └── data/
│   │       └── demo.json
│   └── sales/
│       ├── module.json
│       └── ...
└── bitcode.yaml                    # Project config
```

### 3.3 Security Architecture

**Built into base module. Always available. Zero config needed.**

```
HTTP Request
  │
  ├─► Auth Middleware ──► Validate JWT, load user context
  ├─► Permission Middleware ──► Check RBAC permission
  ├─► Record Rule Middleware ──► Apply row-level filters
  └─► Handler ──► Execute business logic
```

**User → Roles → Permissions:**
```
User "john"
  └─► Role "sales_manager"
        ├─► Permission "order.read"
        ├─► Permission "order.create"
        ├─► Permission "order.confirm"
        └─► Inherits Role "sales_user"
              ├─► Permission "order.read"
              └─► Permission "order.create"
```

**User → Groups → Record Rules:**
```
User "john" in Group "sales.user"
  └─► Record Rule: order WHERE created_by = john.id
      (john can only see his own orders)

User "jane" in Group "sales.manager"
  └─► Record Rule: order WHERE domain = []
      (jane can see all orders)
```

### 3.4 How Model JSON Becomes Database Tables

```
model.json ──► Parser ──► ModelDefinition (Go struct)
                              │
                              ├─► GORM AutoMigrate ──► CREATE TABLE / ALTER TABLE
                              ├─► Generate Repository (generic CRUD)
                              ├─► Register in ModelRegistry
                              └─► Register computed fields
```

**Relationship mapping:**
- many2one → foreign key column (e.g., customer_id UUID REFERENCES customers(id))
- one2many → no column (resolved via inverse FK query)
- many2many → junction table (e.g., order_tags with order_id + tag_id)

### 3.5 Request Flow

```
Client Request
  │
  ├─► Fiber Router (matched from API definition)
  ├─► Middleware Chain (auth → permission → record_rule → audit)
  ├─► Handler
  │     ├─► Standard CRUD: Generic repository operation
  │     └─► Custom: Load process JSON → Execute steps
  ├─► Response (JSON for API, HTML for views)
  └─► Domain Events → Event Bus → Agent handlers
```

### 3.6 Plugin System

```
Engine (Go) ◄──JSON-RPC──► Plugin Process (Node.js / Python)
```

**How it works:**
1. Engine spawns plugin process (e.g., node plugins/typescript/index.js)
2. Communication via stdin/stdout using JSON-RPC protocol
3. Engine sends: { method: "execute", params: { script: "scripts/send_email.ts", context: {...} } }
4. Plugin responds: { result: {...} } or { error: {...} }

**Plugin SDK (TypeScript):**
```typescript
import { definePlugin } from '@bitcode/sdk';

export default definePlugin({
  async sendEmail(ctx, params) {
    await ctx.http.post('https://api.sendgrid.com/v3/mail/send', {
      to: params.to,
      subject: params.subject,
      body: params.body
    });
    return { sent: true };
  }
});
```

---

## 4. Base Module (Built-in)

The base module is always installed. It provides:

| Feature | Models | APIs |
|---------|--------|------|
| Authentication | user | POST /auth/login, POST /auth/register, POST /auth/logout |
| User Management | user | CRUD /api/users |
| Roles | role | CRUD /api/roles |
| Groups | group | CRUD /api/groups |
| Permissions | permission | GET /api/permissions |
| Record Rules | record_rule | CRUD /api/record-rules |
| Audit Log | audit_log | GET /api/audit-logs |
| Settings | setting | GET/PUT /api/settings |

**Default data on first install:**
- Admin user (admin/admin — must change on first login)
- Admin role (all permissions)
- Groups: base.user, base.admin
- Default settings

---

## 5. CLI Commands

```bash
# Project management
bitcode init my-erp              # Create new project
bitcode dev                      # Start dev server (hot reload)
bitcode build                    # Build for production
bitcode validate                 # Validate all JSON definitions

### Module Management
bitcode module install sales      # Install module
bitcode module uninstall sales    # Uninstall module
bitcode module list               # List installed modules
bitcode module create my-module   # Scaffold new module

### User Management
bitcode user create admin admin@example.com
bitcode user set-password admin
bitcode user add-role admin sales_manager

### Database
bitcode db migrate               # Run migrations
bitcode db seed                  # Load demo data
bitcode db reset                 # Reset database

### View & Template
bitcode view list                # List all views
bitcode template validate        # Validate templates
```

---

## 6. Implementation Plan

### Phase 1: Foundation (Tasks 1-10)

| # | Task | Files |
|---|------|-------|
| 1 | Project setup (go mod, directories, Makefile) | cmd/, internal/, pkg/, go.mod |
| 2 | DDD building blocks (Entity, Repository, Event interfaces) | pkg/ddd/ |
| 3 | Model parser (JSON → ModelDefinition struct) | internal/compiler/parser/model.go |
| 4 | Database layer (GORM setup, auto-migration from ModelDefinition) | internal/infrastructure/persistence/ |
| 5 | Generic repository (CRUD for any model, relationship loading) | internal/infrastructure/persistence/repository.go |
| 6 | Model registry (register/get models, computed fields) | internal/domain/model/registry.go |
| 7 | HTTP server (Fiber setup, health check, graceful shutdown) | cmd/engine/main.go |
| 8 | API parser + dynamic route registration | internal/compiler/parser/api.go, internal/presentation/api/ |
| 9 | Auto-CRUD handler (generic list/read/create/update/delete) | internal/presentation/api/crud_handler.go |
| 10 | CLI framework (cobra, init command, dev command) | cmd/bitcode/ |

### Phase 2: Security (Tasks 11-20)

| # | Task | Files |
|---|------|-------|
| 11 | User model + repository | internal/domain/security/ |
| 12 | Password hashing (bcrypt) | pkg/security/password.go |
| 13 | JWT auth (generate, validate, refresh) | pkg/security/jwt.go |
| 14 | Auth middleware (JWT validation, user context) | internal/presentation/middleware/auth.go |
| 15 | Role + Permission models + repository | internal/domain/security/ |
| 16 | Permission checker (RBAC with inheritance, cached) | internal/domain/security/permission.go |
| 17 | Permission middleware | internal/presentation/middleware/permission.go |
| 18 | Group model + repository | internal/domain/security/ |
| 19 | Record rule model + engine (domain filter, interpolation) | internal/domain/security/record_rule.go |
| 20 | Record rule middleware (inject filters into queries) | internal/presentation/middleware/record_rule.go |

### Phase 3: Module System (Tasks 21-28)

| # | Task | Files |
|---|------|-------|
| 21 | Module parser (module.json → ModuleDefinition) | internal/compiler/parser/module.go |
| 22 | Module registry (installed modules, state tracking) | internal/infrastructure/module/registry.go |
| 23 | Dependency resolver (topological sort, circular detection) | internal/infrastructure/module/dependency.go |
| 24 | Module loader (parse all definitions from module dir) | internal/infrastructure/module/loader.go |
| 25 | Module installer (validate → resolve → migrate → seed → register) | internal/infrastructure/module/installer.go |
| 26 | Base module JSON definitions (user, role, group, permission, record_rule, audit_log, setting) | modules/base/ |
| 27 | Base module default data (admin user, default roles/groups) | modules/base/data/ |
| 28 | Module CLI commands (install, uninstall, list, create) | cmd/bitcode/module.go |

### Phase 4: Process Engine (Tasks 29-38)

| # | Task | Files |
|---|------|-------|
| 29 | Process parser (JSON → ProcessDefinition) | internal/compiler/parser/process.go |
| 30 | Process executor core (step dispatcher, context passing) | internal/runtime/executor/executor.go |
| 31 | Step: validate | internal/runtime/executor/steps/validate.go |
| 32 | Step: query, create, update, delete | internal/runtime/executor/steps/data.go |
| 33 | Step: if, switch, loop | internal/runtime/executor/steps/control.go |
| 34 | Step: emit (domain events) | internal/runtime/executor/steps/emit.go |
| 35 | Step: call (invoke another process) | internal/runtime/executor/steps/call.go |
| 36 | Step: assign, log | internal/runtime/executor/steps/util.go |
| 37 | Step: http (external API calls) | internal/runtime/executor/steps/http.go |
| 38 | Error handling + transaction support | internal/runtime/executor/error.go |

### Phase 5: Events, Agents & Plugins (Tasks 39-46)

| # | Task | Files |
|---|------|-------|
| 39 | In-process event bus (publish/subscribe) | internal/domain/event/bus.go |
| 40 | Agent parser (JSON → AgentDefinition) | internal/compiler/parser/agent.go |
| 41 | Agent worker (event subscription, handler execution) | internal/runtime/agent/worker.go |
| 42 | Cron scheduler (parse cron expressions, run jobs) | internal/runtime/agent/cron.go |
| 43 | Plugin manager (spawn process, JSON-RPC communication) | internal/runtime/plugin/manager.go |
| 44 | TypeScript plugin runtime (Node.js process, SDK) | plugins/typescript/ |
| 45 | Step: script (invoke plugin from process) | internal/runtime/executor/steps/script.go |
| 46 | Retry logic (exponential backoff for agents) | internal/runtime/agent/retry.go |

### Phase 6: Views & Templates (Tasks 47-55)

| # | Task | Files |
|---|------|-------|
| 47 | View parser (JSON → ViewDefinition) | internal/compiler/parser/view.go |
| 48 | Template engine (Go html/template, helpers, partials) | internal/presentation/template/ |
| 49 | List view renderer (table, sorting, filtering, pagination) | internal/presentation/view/list.go |
| 50 | Form view renderer (layout, fields, validation, actions) | internal/presentation/view/form.go |
| 51 | Custom view renderer (load template, fetch data sources) | internal/presentation/view/custom.go |
| 52 | Built-in template helpers (formatDate, formatCurrency, etc) | internal/presentation/template/helpers.go |
| 53 | Base module views (login, user_list, user_form) | modules/base/views/ |
| 54 | Base module templates (main layout, partials) | modules/base/templates/ |
| 55 | Asset serving (static files, CSS/JS) | internal/presentation/assets/ |

### Phase 7: Workflow, i18n & Polish (Tasks 56-62)

| # | Task | Files |
|---|------|-------|
| 56 | Workflow parser (JSON → WorkflowDefinition) | internal/compiler/parser/workflow.go |
| 57 | Workflow engine (state machine, transition validation) | internal/runtime/workflow/engine.go |
| 58 | i18n loader (load translations from module i18n/) | internal/infrastructure/i18n/loader.go |
| 59 | i18n middleware (detect locale, translate labels) | internal/presentation/middleware/i18n.go |
| 60 | Audit middleware (log write operations) | internal/presentation/middleware/audit.go |
| 61 | Settings system (per-module settings, CRUD) | internal/domain/setting/ |
| 62 | Redis cache (permission cache, query cache) | internal/infrastructure/cache/ |

### Phase 8: Example, Testing & Deployment (Tasks 63-70)

| # | Task | Files |
|---|------|-------|
| 63 | CRM module (contact, lead, opportunity models + APIs + views) | modules/crm/ |
| 64 | Sales module (order, order_line, invoice models + APIs + views + workflow) | modules/sales/ |
| 65 | Integration tests (install modules, CRUD, permissions, record rules) | tests/ |
| 66 | Performance benchmarks (process execution, permission checking) | tests/benchmarks/ |
| 67 | Dockerfile (multi-stage build) | Dockerfile |
| 68 | Docker Compose (engine + postgres + redis) | docker-compose.yml |
| 69 | Documentation (architecture, module dev guide, security guide) | docs/ |
| 70 | CI/CD pipeline (GitHub Actions: test, lint, build, release) | .github/workflows/ |

---

## 7. Success Criteria

- [ ] bitcode init creates a working project
- [ ] bitcode dev starts server with hot reload
- [ ] Base module auto-installs with user/role/group/permission
- [ ] User can register, login, get JWT token
- [ ] auto_crud generates working CRUD endpoints
- [ ] Permissions block unauthorized access (403)
- [ ] Record rules filter data per user/group
- [ ] Process definitions execute with if/else/loop
- [ ] Custom scripts run via TypeScript plugin
- [ ] Views render list/form/custom pages
- [ ] Modules install/uninstall with dependency resolution
- [ ] CRM + Sales example modules work end-to-end

---

## 8. Future (Post-MVP)

| Feature | Priority | Notes |
|---------|----------|-------|
| Python plugin runtime | High | Add go-python or subprocess |
| Kanban/Calendar/Chart views | High | Extend view renderer |
| File upload/attachment | High | S3-compatible storage |
| Reporting engine | Medium | Aggregate queries + chart views |
| Multi-tenancy | Medium | Tenant column + middleware |
| Mitosis SPA components | Medium | Optional, for rich interactive UIs |
| NATS for distributed events | Low | When scaling beyond single instance |
| gRPC for plugins | Low | When JSON-RPC becomes bottleneck |
| Marketplace | Low | Community module sharing |
| Cloud platform | Low | Managed hosting |
| GraphQL API | Low | Alternative to REST |
| WebSocket real-time | Low | Live updates |
