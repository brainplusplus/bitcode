# BitCode Platform — Architecture

## Overview

BitCode is a **JSON-driven low-code platform** for building business applications. Developers define models, APIs, processes, views, and workflows in JSON; the Go engine interprets those definitions at runtime to produce a fully functional application with REST APIs, server-rendered UI, background jobs, and security.

**JSON is the source code. Go is the runtime. Modules are the packaging. Web Components are the UI.**

```
bitcode/
├── engine/          Go runtime — reads JSON, runs the app
├── packages/        Shared libraries
│   ├── components/  Stencil Web Components (@bitcode/components)
│   ├── go-json/     go-json — JSON/JSONC programming language engine
│   └── tauri/       Tauri native shell (desktop + mobile)
├── samples/         Example applications
│   └── erp/         Full ERP sample (CRM + HRM)
├── docs/            Project-level documentation
└── sprints/         Sprint planning & tracking
```

---

## System Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                          Browser / Client                           │
│  ┌──────────────┐  ┌──────────────┐  ┌───────────────────────────┐ │
│  │  REST API    │  │  SSR Pages   │  │  WebSocket (real-time)    │ │
│  │  (JSON)      │  │  (HTML)      │  │  (domain events)          │ │
│  └──────┬───────┘  └──────┬───────┘  └────────────┬──────────────┘ │
└─────────┼─────────────────┼───────────────────────┼────────────────┘
          │                 │                       │
┌─────────▼─────────────────▼───────────────────────▼────────────────┐
│                      Fiber HTTP Server (Go)                        │
│                                                                     │
│  ┌─────────────────── Middleware Chain ──────────────────────────┐  │
│  │  Tenant → Auth (JWT) → Permission (RBAC) → RecordRule (RLS)  │  │
│  │  → Audit Logging                                              │  │
│  └──────────────────────────┬───────────────────────────────────┘  │
│                              │                                      │
│  ┌───────────────────────────▼──────────────────────────────────┐  │
│  │                     Route Handlers                            │  │
│  │  ┌────────────┐  ┌──────────────┐  ┌──────────────────────┐  │  │
│  │  │ Auto-CRUD  │  │   Process    │  │   View Renderer      │  │  │
│  │  │ (REST API) │  │   Executor   │  │   (SSR HTML)         │  │  │
│  │  └─────┬──────┘  └──────┬───────┘  └──────────┬───────────┘  │  │
│  └────────┼────────────────┼─────────────────────┼──────────────┘  │
│           │                │                     │                  │
│  ┌────────▼────────────────▼─────────────────────▼──────────────┐  │
│  │                   Internal Services                           │  │
│  │  ┌────────────┐ ┌────────────┐ ┌──────────┐ ┌────────────┐  │  │
│  │  │  Generic   │ │  Workflow   │ │  Event   │ │  Plugin    │  │  │
│  │  │ Repository │ │  Engine    │ │   Bus    │ │  Manager   │  │  │
│  │  └─────┬──────┘ └────────────┘ └────┬─────┘ └─────┬──────┘  │  │
│  │        │                            │              │          │  │
│  │  ┌─────▼──────┐  ┌─────────────────▼──┐  ┌───────▼───────┐  │  │
│  │  │  Database  │  │  Agent Worker      │  │  Bridge API   │  │  │
│  │  │  (GORM)   │  │  + Cron Scheduler  │  │  (20 ns)      │  │  │
│  │  └────────────┘  └────────────────────┘  └───────┬───────┘  │  │
│  │                                                   │          │  │
│  │  ┌────────────────────────────────────────────────▼───────┐  │  │
│  │  │              Script Runtimes                            │  │  │
│  │  │  ┌─────────┐ ┌──────────┐ ┌─────────┐ ┌────────────┐  │  │  │
│  │  │  │  goja   │ │ quickjs  │ │  yaegi  │ │ Node.js /  │  │  │  │
│  │  │  │  (JS)   │ │  (JS)   │ │  (Go)   │ │ Python     │  │  │  │
│  │  │  └─────────┘ └──────────┘ └─────────┘ └────────────┘  │  │  │
│  │  └────────────────────────────────────────────────────────┘  │  │
│  └──────────────────────────────────────────────────────────────┘  │
│                                                                     │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │                    Module System                              │  │
│  │  Layered FS: project modules → global modules → embedded     │  │
│  │  Dependency resolution (topological sort)                     │  │
│  │  Hot reload (file watcher in dev mode)                        │  │
│  └──────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────┐
│                    @bitcode/components (Stencil)                     │
│  Web Components: fields, layout, views, charts, dialogs, widgets    │
│  Served as static assets from /assets/components/                   │
└─────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────┐
│                    @bitcode/tauri (Native Shell)                     │
│  Tauri 2.0 — Stencil components run inside native WebView           │
│  Plugins: SQLite, filesystem, notifications, barcode, biometric     │
│  Bridge: bc-native.ts routes to Tauri IPC or Web API fallback       │
│  Platforms: Windows, macOS, Linux, iOS, Android                     │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Core Concepts

### 1. Modules

A module is a self-contained unit of functionality (like Odoo modules). Each module is a directory containing JSON definitions:

```
modules/crm/
├── module.json          # Metadata, dependencies, permissions, menu
├── models/*.json        # Data models (fields, relationships, rules)
├── apis/*.json          # REST API endpoints
├── processes/*.json     # Business logic (step-based)
├── views/*.json         # UI definitions (list, form, kanban, etc.)
├── templates/*.html     # Go html/template files
├── scripts/*.ts|*.py    # Plugin scripts
├── agents/*.json        # Event handlers + cron jobs
├── data/*.json          # Seed data
└── i18n/*.json          # Translations
```

**Module loading order** is resolved via topological sort of dependencies. The `base` module is always installed first and provides users, roles, groups, permissions, and default templates.

**Three-layer module resolution** (highest priority first):
1. **Project modules** — `./modules/` (local to the app)
2. **Global modules** — `~/.bitcode/modules/` (shared across apps)
3. **Embedded modules** — compiled into the engine binary via `go:embed`

### 2. Models

JSON definitions that map to database tables. The engine auto-creates tables, handles migrations, and generates CRUD operations.

```json
{
  "name": "contact",
  "fields": [
    { "name": "name", "type": "string", "required": true },
    { "name": "email", "type": "string" },
    { "name": "company_id", "type": "many2one", "model": "company" },
    { "name": "tags", "type": "many2many", "model": "tag" }
  ],
  "record_rules": [...]
}
```

**Auto-generated columns**: `id` (UUID), `created_at`, `updated_at`, `created_by`, `updated_by`, `active` (soft delete), `tenant_id` (when `tenant.enabled = true` and model is `tenant_scoped`).

**Relationship types**:
| Type | DB Implementation |
|------|-------------------|
| `many2one` | FK column (UUID) |
| `one2many` | Inverse FK query (no column) |
| `many2many` | Auto-created junction table |

**Model inheritance**: A model can set `"inherit": "parent_model"` to merge parent fields.

### 3. APIs

JSON definitions that register REST endpoints on the Fiber router.

```json
{
  "name": "contact_api",
  "model": "contact",
  "base_path": "/api/contacts",
  "auto_crud": true,
  "auth": true,
  "search": ["name", "email", "company"]
}
```

`auto_crud: true` generates: `GET /`, `GET /:id`, `POST /`, `PUT /:id`, `DELETE /:id` with pagination, search, and filtering.

`auth: true` enables the full security chain: JWT validation + RBAC permissions + record-level security (RLS).

### 4. Processes

Step-based business logic engine. 14 step types:

| Category | Steps |
|----------|-------|
| **Data** | `query`, `create`, `update`, `delete` |
| **Validation** | `validate` (eq, neq, required rules) |
| **Control** | `if`, `switch`, `loop` |
| **Integration** | `http` (external API), `script` (TS/Python plugin), `call` (sub-process) |
| **Side Effects** | `emit` (domain event), `assign` (set variable), `log` (audit) |

Processes execute within a **Context** that carries input, variables, result, and emitted events.

### 5. Workflows

State machines with permission-gated transitions:

```json
{
  "name": "lead_workflow",
  "model": "lead",
  "field": "status",
  "states": ["new", "contacted", "qualified", "proposal", "won", "lost"],
  "transitions": [
    { "name": "qualify", "from": ["new", "contacted"], "to": "qualified", "permission": "crm.qualify_lead" }
  ]
}
```

The workflow engine validates transitions and sets initial state on record creation.

### 6. Views

UI definitions rendered server-side (SSR) by the Go template engine:

| Type | Description |
|------|-------------|
| `list` | Data table with columns, pagination, search |
| `form` | Record editor with fields, tabs, sections |
| `kanban` | Card board grouped by a field |
| `calendar` | Date-based event view |
| `chart` | ECharts-powered visualizations |
| `custom` | Free-form Go html/template |

Views are wrapped in a layout template with sidebar navigation, navbar, and responsive CSS.

### 7. Web Components (`@bitcode/components`)

A Stencil.js component library providing 103 enterprise-grade UI widgets. **Standalone-capable** — works in plain HTML without BitCode framework.

| Category | Components |
|----------|------------|
| **Fields** | 34 field types (string, date, richtext, code, signature, barcode, geo, etc.) |
| **Layout** | row, column, section, tabs, sheet, header, separator |
| **Views** | list, form, kanban, calendar, gantt, map, tree, report, activity |
| **Charts** | line, bar, pie, area, gauge, funnel, heatmap, pivot, KPI, scorecard, progress |
| **Dialogs** | modal, confirm, quick-entry, wizard, toast |
| **Search** | search bar, filter bar, filter panel, filter builder, favorites |
| **Widgets** | badge, copy, phone, email, URL, progress, statusbar, priority, handle, domain |
| **Data** | datatable, lookup modal, child table |
| **Print** | export, print, report link |
| **Social** | activity feed, chatter, timeline |

**Core infrastructure** (`src/core/`):
- **BcSetup** — Global configuration (auth, headers, theme, validators). Standalone — no BitCode dependency.
- **Data Fetcher** — 4-level data strategy: local data, URL endpoint, event intercept, custom fetcher function.
- **Validation Engine** — 3-level validation: built-in rules, custom JS validators, server-side validation.
- **Field Utils** — Shared field logic (dirty/touched tracking, ARIA, render helpers).
- **Theming** — Light/dark/system-detect/custom themes via CSS custom properties. No Tailwind.
- **Event Bus** — Cross-component communication.
- **BcNative** — Bridge abstraction for native capabilities (camera, GPS, SQLite, barcode, biometrics). Detects Tauri vs browser, routes to Tauri IPC or Web API fallback.
- **Form Engine** — Form state management (optional, BitCode-specific).
- **API Client** — HTTP client for BitCode REST APIs (optional fallback).

### 8. Bridge API (`engine/internal/runtime/bridge/`)

The Bridge API is the unified interface between the Go engine and all script runtimes (goja, quickjs, yaegi, Node.js, Python). Scripts access engine services through a single `bitcode.*` namespace with 20 namespaces:

| Namespace | Purpose |
|-----------|---------|
| `bitcode.model("name")` | Permission-aware CRUD, bulk ops, relations |
| `bitcode.db` | Raw SQL queries |
| `bitcode.http` | TLS-fingerprinted HTTP client (tls-client) |
| `bitcode.cache` | Key-value cache (memory/Redis) |
| `bitcode.fs` | Sandboxed filesystem (path escape prevention) |
| `bitcode.env("KEY")` | Environment variables (engine secrets denied) |
| `bitcode.config("key")` | Module settings via Viper |
| `bitcode.emit(event, data)` | Event bus publishing |
| `bitcode.call(process, input)` | Cross-process invocation |
| `bitcode.exec(cmd, args)` | OS command execution (whitelist-only) |
| `bitcode.log(level, msg)` | Structured logging per module |
| `bitcode.email` | SMTP email sending |
| `bitcode.notify` | WebSocket notifications |
| `bitcode.storage` | File upload/download (local/S3) |
| `bitcode.t("key")` | i18n translation |
| `bitcode.security` | Permission checks, group membership |
| `bitcode.audit` | Audit log writing |
| `bitcode.crypto` | Encrypt/decrypt/hash/verify |
| `bitcode.execution` | Process execution log (search/get/cancel) |
| `bitcode.tx(fn)` | Database transactions |

Security is enforced per-module via `SecurityRules` in `module.json` (env_allow/deny, exec_allow/deny, fs_allow/deny, sudo_allow).

### 9. Multi-Tenancy

Engine-level tenant isolation using `shared_table` strategy. All tenant-scoped models automatically get a `tenant_id` column and filtered queries.

| Config | Behavior |
|--------|----------|
| `tenant.enabled: false` | Single tenant, zero overhead |
| `tenant.enabled: true` | Auto-add `tenant_id` column, auto-filter queries, auto-set on create |
| `tenant_scoped: false` on model | Shared across tenants (e.g. plans, global settings) |
| `sudo().withTenant("x")` | Cross-tenant access in scripts |

Tenant detection: header (`X-Tenant-ID`), subdomain, or path — configured via `tenant.strategy`.

---

## Data Flow

### API Request Lifecycle

```
HTTP Request
  → Fiber Router (matched from API JSON definition)
  → Tenant Middleware (inject tenant context if multi-tenancy enabled)
  → Auth Middleware (validate JWT, extract user_id/roles/groups)
  → Permission Middleware (check RBAC: user has model.action permission)
  → Record Rule Middleware (inject row-level WHERE filters)
  → Audit Middleware (log write operations)
  → Handler
      → Auto-CRUD: GenericRepository.FindAll / Create / Update / Delete
      → Process: Executor.Execute(steps) → step handlers → Context
      → View: ViewRenderer.RenderView(template + query data)
  → Response (JSON for API, HTML for views)
  → Domain Events → Event Bus → Agent handlers / WebSocket broadcast
```

### Module Loading Sequence

```
1. Scan module directories (project → global → embedded)
2. Parse each module.json
3. Resolve dependencies (topological sort, circular detection)
4. For each module (in dependency order):
   a. Parse models → Register in ModelRegistry → Auto-migrate DB tables
   b. Parse APIs → Register Fiber routes (with middleware if auth: true)
   c. Parse views → Register in view map
   d. Load templates → Register in TemplateEngine
   e. Load i18n → Register translations
   f. Load processes → Register in ProcessRegistry
   g. Load workflows → Register in WorkflowEngine
   h. Seed data → Insert default records
   i. Register module in ModuleRegistry
5. Process cross-module view registrations
```

### Process Execution

```
Executor.Execute(process, input, userID)
  → For each step (sequential):
      → Dispatch to StepHandler by step.type
      → Handler reads/writes Context (input, variables, result, events)
      → Control steps (if/switch/loop) may recurse into sub-steps
      → Call step loads sub-process from ProcessRegistry
      → Script step invokes TS/Python plugin via JSON-RPC
      → On error: return error with step info
  → Return Context (result, variables, emitted events)
  → Publish events to EventBus → Agent handlers
```

---

## Database

### Supported Drivers

| Driver | Default | UUID Strategy | JSON Type | Notes |
|--------|---------|---------------|-----------|-------|
| SQLite | Yes | TEXT (app-generated) | TEXT | Zero config, single file |
| PostgreSQL | No | UUID (gen_random_uuid) | JSONB | Production recommended |
| MySQL | No | CHAR(36) (app-generated) | JSON | Full support |

All drivers use GORM as the ORM layer. Tables are auto-created from model definitions with dialect-aware SQL generation.

### Auto-generated Columns

Every table automatically includes:

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID | Primary key |
| `created_at` | Timestamp | Set on create |
| `updated_at` | Timestamp | Set on update |
| `created_by` | UUID (FK) | Creating user |
| `updated_by` | UUID (FK) | Last updating user |
| `active` | Boolean | Soft delete flag (default: true) |

---

## Security

### Architecture: Odoo-style Group-Based Permissions

**Group** is the sole security concept (replaces Role+Permission). Each group has:
- **ModelAccess** (ACL): 12 ERPNext-style permissions per model (select/read/write/create/delete/print/email/report/export/import/mask/clone)
- **RecordRules** (RLS): Row-level domain filters with Global∩Group composition
- **Implied Groups**: Additive inheritance chain (Manager implies User)
- **Menu/Page visibility**: Per-group UI access control
- **Share flag**: Portal/external user groups

### Permission Check Logic

- **Additive**: User in Group A (read) + Group B (write) = can read + write
- **Default-deny**: No matching ACL = access denied
- **Superuser bypass**: `is_superuser=true` bypasses all ACL + record rules
- **Field-level**: `groups` property hides field from non-members; `mask`/`mask_length` masks values server-side

### Middleware Chain

```
Tenant → Auth → Permission (ModelAccess) → RecordRule (RLS + interpolation) → Audit → Handler
```

1. **Tenant**: Extract tenant ID from header/subdomain/path, scope all queries
2. **Auth**: Validate JWT token, load user context (user_id, groups)
3. **Permission**: Check ModelAccess via PermissionService — resolve group chain (implied, recursive BFS), query model_access table, additive union
4. **RecordRule**: Apply row-level filters via RecordRuleService — global rules INTERSECT, group rules UNION, `{{user.id}}` interpolation
5. **Audit**: Log all write operations (POST/PUT/DELETE)
6. **Handler**: CRUD handler applies field masking + field groups filtering before response, injects permissions in response metadata

### Security Definition Files

```
modules/{module}/securities/*.json  →  One file per group
                                       Synced to DB on module install
                                       Bi-directional: JSON↔DB with conflict detection
                                       Admin UI for editing (7-tab Odoo-style group form)
```

### Multi-Protocol Security

All three protocols (REST, GraphQL, WebSocket) share the same permission enforcement:
- REST: PermissionMiddleware + RecordRuleMiddleware per endpoint
- GraphQL: Resolver checks PermissionService + RecordRuleService per query/mutation
- WebSocket: CRUDHandler checks PermissionService + RecordRuleService per message

### Key Principle

`auth: true` on an API enables the **entire security chain** — JWT + Group-based ACL + RLS. Model `"api": true` auto-generates CRUD with permission enforcement. Security definitions in `securities/*.json` are synced to DB and editable from admin UI.

---

## Plugin System

External code execution via JSON-RPC over stdin/stdout:

| Runtime | Language | Entry Point |
|---------|----------|-------------|
| TypeScript | Node.js | `plugins/typescript/index.js` |
| Python | Python 3 | `plugins/python/runtime.py` |

Plugins are invoked by the `script` step in processes. The Plugin Manager spawns child processes and communicates via JSON-RPC.

A gRPC protocol is also defined (`pkg/plugin/proto/plugin.proto`) for high-throughput scenarios.

---

## Caching

| Driver | Use Case |
|--------|----------|
| Memory (default) | Development, single instance |
| Redis (optional) | Production, multi-instance |

Config: `CACHE_DRIVER=memory` or `CACHE_DRIVER=redis REDIS_URL=redis://...`

---

## Multi-tenancy

Three isolation strategies:

| Strategy | Detection | Example |
|----------|-----------|---------|
| `header` | `X-Tenant-ID` header | `curl -H "X-Tenant-ID: acme"` |
| `subdomain` | Hostname prefix | `acme.app.example.com` |
| `path` | URL path segment | `/tenant/acme/api/...` |

Tenant context is injected by middleware and scoped through the repository layer.

---

## Configuration

All configuration via environment variables or `bitcode.toml`/`bitcode.yaml`. Defaults work out of the box (SQLite + memory cache).

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | Server port |
| `DB_DRIVER` | `sqlite` | `sqlite`, `postgres`, `mysql` |
| `DB_SQLITE_PATH` | `bitcode.db` | SQLite file path |
| `CACHE_DRIVER` | `memory` | `memory`, `redis` |
| `JWT_SECRET` | (default) | JWT signing secret |
| `MODULE_DIR` | `modules` | Path to modules directory |
| `TENANT_ENABLED` | `false` | Enable multi-tenancy |
| `TENANT_STRATEGY` | `header` | `header`, `subdomain`, `path` |

See the root [`README.md`](../README.md) for the full configuration reference.

---

## Technology Stack

| Layer | Technology |
|-------|------------|
| **Runtime** | Go 1.23+, Fiber v2 (HTTP), GORM (ORM) |
| **go-json Engine** | Go 1.24+, expr-lang/expr v1.17+ (expression evaluation) |
| **Database** | SQLite / PostgreSQL / MySQL |
| **Cache** | In-memory / Redis |
| **Config** | Viper (env + TOML/YAML) |
| **CLI** | Cobra |
| **Templates** | Go html/template |
| **Web Components** | Stencil.js (TypeScript) |
| **Charts** | ECharts |
| **Rich Text** | Tiptap |
| **Code Editor** | CodeMirror |
| **Calendar** | FullCalendar |
| **Gantt** | frappe-gantt |
| **Maps** | Leaflet |
| **Native Shell** | Tauri 2.0 (Rust) — desktop + mobile |
| **Plugins** | Node.js (TS) + Python 3 via JSON-RPC |
| **Real-time** | WebSocket (Fiber contrib) |
| **Containerization** | Docker + docker-compose |
