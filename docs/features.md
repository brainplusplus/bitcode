# BitCode Platform — Features & Roadmap

**Last Updated**: 24 April 2026
**Benchmark**: Frappe/ERPNext, Odoo, NocoBase
**Engine Version**: 0.1.0

---

## Status Overview

| Metric | Count |
|--------|-------|
| Total Features Tracked | 70 |
| ✅ Implemented | 43 |
| ⚠️ Partial | 2 |
| ❌ Not Yet | 25 |
| **Completion** | **61.4%** (effective ~63% counting partials as 0.5) |

### Per-Category Summary

| # | Category | ✅ | ⚠️ | ❌ | Total | Score |
|---|----------|-----|------|------|-------|-------|
| 1 | Core Framework & Data Modeling | 5 | 2 | 0 | 7 | 86% |
| 2 | Permission & Access Control | 6 | 1 | 1 | 8 | 81% |
| 3 | Audit Log & Monitoring | 5 | 0 | 1 | 6 | 83% |
| 4 | Workflow & Automation | 4 | 1 | 3 | 8 | 56% |
| 5 | Form & UI Builder | 4 | 1 | 3 | 8 | 56% |
| 6 | Reporting & Analytics | 1 | 2 | 3 | 6 | 33% |
| 7 | Integration & API | 1 | 1 | 4 | 6 | 25% |
| 8 | Configuration & Customization | 4 | 1 | 2 | 7 | 64% |
| 9 | Security & Infrastructure | 7 | 1 | 0 | 8 | 94% |
| 10 | Collaboration & Communication | 0 | 1 | 4 | 5 | 10% |

> Score = (✅ + ⚠️×0.5) / Total × 100%

---

## Legend

| Symbol | Meaning |
|--------|---------|
| ✅ | Implemented and functional |
| ⚠️ | Partially implemented — foundation exists but incomplete |
| ❌ | Not yet implemented |

**Effort Estimates**:
- **S** = Small (1–2 days, 1 file/module)
- **M** = Medium (3–5 days, multiple files)
- **L** = Large (1–2 weeks, new feature)
- **XL** = Extra Large (2–4 weeks, new subsystem)

---

## Platform Strengths

Before the gap list — what's already **production-solid**:

1. **JSON-driven development** — Models, APIs, processes, views, workflows — all defined in JSON. Go engine interprets at runtime.
2. **Module system (Odoo-style)** — Dependency resolution (topological sort), data seeding, cross-module views, 3-layer FS (project → global → embedded).
3. **Security stack** — JWT auth + RBAC permissions + Record Rules (row-level security) + audit logging. Full middleware chain.
4. **Process engine** — 14 step types: validate, query, create, update, delete, if, switch, loop, emit, call, script, http, assign, log. DAG executor for parallel steps.
5. **Workflow engine** — State machines with permission-gated transitions, initial state on create, process linking.
6. **Plugin system** — Dual runtime (TypeScript + Python), JSON-RPC over stdin/stdout, health monitoring, auto-restart.
7. **View system** — 6 view types (list, form, kanban, calendar, chart, custom) with SSR rendering + layout system.
8. **Web Components** — 94 Stencil.js components: 30+ field types, layout, views, charts, dialogs, widgets, search, social, print.
9. **Multi-database** — SQLite (zero-config default), PostgreSQL, MySQL, MongoDB. Auto-migration from JSON definitions. Per-module table prefix, Postgres schema support. Unified query builder with JSON DSL.
10. **Real-time** — WebSocket hub broadcasting domain events to connected clients.
11. **File Storage** — Local + S3 storage with attachments table, path/name formatting, thumbnails, versioning, duplicate detection.

---

## 1. Core Framework & Data Modeling

| # | Feature | Status | Effort | What Exists | What's Missing |
|---|---------|--------|--------|-------------|----------------|
| 1 | Schema Builder | ⚠️ | L | JSON-based model definitions (`models/*.json`), parsed by `compiler/parser/model.go`. Admin UI at `/admin/models/:name` shows fields & rules. Schema tab with visual field table + JSON editor + save/validate API. | Full drag-and-drop visual builder with field palette. Current schema tab is visual table + JSON editor, not drag-and-drop yet. |
| 2 | Field Types | ✅ | — | 16+ types: string, text, integer, decimal, boolean, date, datetime, selection, email, many2one, one2many, many2many, json, file, computed. Stencil components cover 30+ field types. | — |
| 3 | Model Relationships | ✅ | — | `many2one` (FK column), `one2many` (reverse FK), `many2many` (auto junction table). `MigrateModel()` handles DDL. | — |
| 4 | Child Table / Inline Table | ✅ | — | `one2many` field + form view tabs embedding child views. `bc-child-table` Stencil component. | — |
| 5 | Computed / Formula Fields | ✅ | — | `computed` type defined in parser. Runtime expression evaluator (`engine/internal/runtime/expression/`) evaluates scalar formulas (`quantity * unit_price`) and aggregate expressions (`sum(lines.subtotal)`) at query time. Hydrated in repository FindByID/FindAll and view renderer. Supports: arithmetic, comparisons, boolean logic, functions (sum/count/avg/min/max/abs/round/ceil/floor/if), dot-path field access, one2many child collection aggregates. 17 tests. | — |
| 6 | Data Versioning | ✅ | — | `data_revisions` table stores full before/after snapshots on every create/update/delete via GenericRepository hooks. Monotonic per-record versioning. Admin API: `GET /admin/api/data/:model/:id/revisions`, `GET .../revisions/:version`, `POST .../restore/:version`. Restore creates new head revision. Change diff computed automatically. Cleanup support. 7 tests. | — |
| 7 | Multi-Source Data | ⚠️ | L | 3 database drivers (SQLite/Postgres/MySQL) via `DB_DRIVER`. | Only 1 database per instance. No simultaneous connections to multiple external databases. |

---

## 2. Permission & Access Control

| # | Feature | Status | Effort | What Exists | What's Missing |
|---|---------|--------|--------|-------------|----------------|
| 8 | Role Management | ✅ | — | `role` model, domain logic in `security/role.go`, admin UI. CRUD via API. | — |
| 9 | Permission Matrix | ✅ | — | Pattern `module.model.action` (read/create/write/delete). Defined in `module.json`. Middleware enforcement. Auto-derived for `auto_crud`. | — |
| 10 | Record Rules (RLS) | ✅ | — | `record_rules` in model JSON. Domain filter with operators + `{{user.id}}` interpolation. Middleware injects WHERE clauses. | — |
| 11 | Field-Level Permission | ❌ | M | — | Can't hide/readonly specific fields per role. Need field-level permission config in model JSON + filter in API response and form renderer. |
| 12 | Menu Access Control | ✅ | — | Menu defined per module in `module.json`. Only visible if module installed. | — |
| 13 | UI Visibility Rules | ✅ | — | View actions have `"visible": "status == 'draft'"`. Form fields have `"readonly": true`. Component compiler evaluates conditions. | — |
| 14 | IP Whitelist / Session Policy | ✅ | — | IP whitelist middleware (exact IP + CIDR support, admin-only or global). Session policy: configurable JWT duration (`security.session_duration`), cookie `Secure`/`SameSite` flags. All via `security.*` config. | — |
| 15 | Plugin Permission | ⚠️ | S | Permission per module exists. Plugin scripts run in module context. | No granular per-plugin/script permission. Extend pattern to `module.plugin.script_name`. |

---

## 3. Audit Log & Monitoring

| # | Feature | Status | Effort | What Exists | What's Missing |
|---|---------|--------|--------|-------------|----------------|
| 16 | Audit Log | ✅ | — | `audit_log` model with user_id, action, model_name, record_id, changes, ip_address, user_agent, request_method, request_path, status_code, duration_ms. `PersistentAuditMiddleware` writes ALL write requests to DB. `AuditLogRepository` with async writes. | — |
| 17 | Record Activity Timeline | ✅ | — | `GET /admin/api/data/:model/:id/timeline` combines `data_revisions` + `audit_log` entries. Admin model data page has "History" button per record with modal timeline showing changes with field-level diffs. | — |
| 18 | Login History | ✅ | — | Login/logout/register persisted to `audit_log` from both API auth handler and app HTML login. `POST /auth/logout` endpoint added. Admin page at `/admin/audit/login-history` with User-Agent, IP. | — |
| 19 | API Request Log | ✅ | — | `PersistentAuditMiddleware` persists all write requests to `audit_logs` table with method, path, status, duration, user, IP. Admin page at `/admin/audit/request-log` with method filtering. | — |
| 20 | Data Change Diff | ✅ | — | `data_revisions` stores structured `{field: {old, new}}` changes. Timeline modal renders field-level before/after diff with color-coded old (red strikethrough) → new (green). | — |
| 21 | Export/Import Log | ❌ | S* | — | No export/import feature exists yet (see #41, #47), so no log either. *Trivial once export/import is built. |

---

## 4. Workflow & Automation

| # | Feature | Status | Effort | What Exists | What's Missing |
|---|---------|--------|--------|-------------|----------------|
| 22 | Workflow Builder (Visual) | ⚠️ | L | Full workflow engine via JSON: states, transitions, permissions, process linking. Runtime in `runtime/workflow/`. | Visual drag-and-drop builder in browser. Need frontend state machine editor. |
| 23 | Approval Chain | ✅ | — | Workflow transitions with `permission` field. Multi-level via chained transitions. | — |
| 24 | Trigger & Action Rules | ✅ | — | Agent system with event triggers + process `emit` step. Documented in `docs/features/agents.md`. | — |
| 25 | Scheduled Tasks / Cron | ✅ | — | Agent cron with standard cron format. Runtime in `runtime/agent/`. Retry + backoff. | — |
| 26 | Email / Notification | ❌ | L | — | No SMTP/email sending. Agent can trigger scripts, but no built-in email integration. Need SMTP config, email queue, template engine, notification preferences. |
| 27 | Assignment Rules | ❌ | M | — | No auto-assignment based on rules. Need assignment rule JSON definition + evaluator on record create/update. |
| 28 | Webhook | ❌ | M | Process engine has `http` step for outbound calls, but not a configurable webhook system. | Need webhook definition (URL, events, headers) + dispatcher listening to event bus. |
| 29 | Server Script / Business Logic | ✅ | — | Plugin system (TS + Python) via JSON-RPC. Process `script` step. Documented in `docs/features/plugins.md`. | — |

---

## 5. Form & UI Builder

| # | Feature | Status | Effort | What Exists | What's Missing |
|---|---------|--------|--------|-------------|----------------|
| 30 | Form Builder (Visual) | ⚠️ | L | JSON-based form layout (rows, fields, widths, tabs). SSR rendering. 94 Stencil.js components. | Visual drag-and-drop form designer in browser. |
| 31 | Conditional Field Logic | ✅ | — | `"visible": "expression"`, `"readonly": true`. Component compiler evaluates conditions. Stencil components support `behavior.dependsOn`, `readonlyIf`, `mandatoryIf`. | — |
| 32 | Custom Validation Rules | ✅ | — | Process `validate` step with eq, neq, required operators. | — |
| 33 | Multi-Step Form / Wizard | ❌ | M | — | No wizard component. Need wizard JSON definition (steps + fields per step) + `bc-dialog-wizard` exists in Stencil but not wired to engine. |
| 34 | Print Format / PDF | ❌ | L | Template engine exists (Go html/template) but HTML only. No PDF generation. | Need print template JSON definition + PDF renderer (wkhtmltopdf/chromedp/gotenberg). |
| 35 | Web Form (Public) | ❌ | M | — | All forms require auth. Need public form route (bypass auth) + CAPTCHA + rate limiting. |
| 36 | View Types (List/Kanban/Calendar) | ✅ | — | 6 view types: list, form, kanban, calendar, chart, custom. All implemented in `view/renderer.go` + templates. Stencil has 9 view components (+ gantt, map, tree, report, activity). | — |
| 37 | Dashboard Builder | ✅ | — | Custom view type with `data_sources`. Admin dashboard at `/admin`. | — |

---

## 6. Reporting & Analytics

| # | Feature | Status | Effort | What Exists | What's Missing |
|---|---------|--------|--------|-------------|----------------|
| 38 | Report Builder | ⚠️ | L | List view with filters + sort. Custom API endpoints can serve report data. | Dedicated report builder with group-by, aggregation, calculated columns. |
| 39 | Query / Script Report | ⚠️ | M | Process `query` step + plugin scripts can execute custom logic. | Dedicated SQL/script report system runnable from UI. |
| 40 | Chart Builder | ✅ | — | Chart view type. 11 Stencil chart components (line, bar, pie, area, gauge, funnel, heatmap, pivot, KPI, scorecard, progress). | — |
| 41 | Export Data (CSV/Excel/PDF) | ❌ | M | `bc-export` Stencil component exists (uses xlsx library). | No server-side export handler. Need export endpoint per model (CSV via encoding/csv, Excel via excelize). |
| 42 | Pivot Table | ❌ | L | `bc-chart-pivot` Stencil component exists. | No server-side pivot engine (dimensions, measures, aggregations). Frontend component needs data feed. |
| 43 | Scheduled Report | ❌ | M* | Cron system exists. | No report + email delivery integration. *Depends on email (#26) + report builder (#38). |

---

## 7. Integration & API

| # | Feature | Status | Effort | What Exists | What's Missing |
|---|---------|--------|--------|-------------|----------------|
| 44 | REST API (Auto-Generated) | ✅ | — | `auto_crud: true` → GET (list+detail), POST, PUT, DELETE. Pagination, search, soft delete, auth, RLS. | — |
| 45 | OAuth2 / SSO | ❌ | L | — | Only JWT login (username/password). No OAuth2, SSO, or LDAP. |
| 46 | API Key Management | ❌ | S | — | No API key system for M2M integration. Need api_key model + auth middleware. |
| 47 | Data Import / Export | ❌ | L | Data seeding via `data/*.json` (module install only, not user-facing). | CSV/Excel import wizard with field mapping + validation. |
| 48 | Third-Party Connectors | ⚠️ | XL | Process `http` step + plugin scripts for integration. | No pre-built connectors (Slack, WhatsApp, payment gateways). Need connector framework. |
| 49 | GraphQL API | ❌ | L | — | Listed in AGENTS.md as "Remaining". Need schema generator from model definitions + resolver layer. |

---

## 8. Configuration & Customization

| # | Feature | Status | Effort | What Exists | What's Missing |
|---|---------|--------|--------|-------------|----------------|
| 50 | Custom App / Module | ✅ | — | Full module system. CLI: `bitcode module create`. Dependency resolution, topological sort. 3-layer FS. | — |
| 51 | Workspace / Menu Builder | ✅ | — | Menu in `module.json` with label, icon, children, view links. Sidebar navigation. | — |
| 52 | Branding / White Label | ❌ | S | — | No UI for logo, colors, app name. Templates hardcoded. Need branding settings + template injection. |
| 53 | Multi-Language / i18n | ✅ | — | Translation files per module (`i18n/*.json`). Translator with fallback chain. 11 languages in Stencil components. | — |
| 54 | Multi-Currency | ❌ | L | `formatCurrency` template helper exists but hardcoded format. `bc-field-currency` Stencil component. | No currency model, exchange rates, or automatic conversion. |
| 55 | Multi-Company | ⚠️ | L | Multi-tenancy (header/subdomain/path) can serve as multi-company. | No dedicated multi-company with inter-company transactions, consolidated reporting. |
| 56 | Plugin / Extension System | ✅ | — | TypeScript + Python runtime, JSON-RPC, gRPC proto, health monitoring, auto-restart. | — |

---

## 9. Security & Infrastructure

| # | Feature | Status | Effort | What Exists | What's Missing |
|---|---------|--------|--------|-------------|----------------|
| 65 | Auth Module | ✅ | — | Embedded `auth` module with login, register, forgot password, reset, 2FA verify pages. All templates i18n-ready (11 languages). `module.json` `auth` field controls per-module auth requirement (default: true). Settings-driven: `register_enabled`, `otp_enabled`, `otp_channel`, `otp_type`. `menu_visibility: "none"` support. `?next=` redirect with sanitization. | — |
| 57 | Two-Factor Auth (2FA) | ✅ | — | Email OTP 2FA: `POST /auth/2fa/enable`, `/auth/2fa/disable`, `/auth/2fa/validate`. Login returns `requires_2fa` + temp token when 2FA enabled. OTP via SMTP email, cached with TTL, max 3 attempts. | — |
| 58 | Data Encryption | ✅ | — | Password hashing (bcrypt) + JWT signing + AES-256-GCM field-level encryption. Fields marked `"encrypted": true` in model JSON get transparent encrypt-on-write / decrypt-on-read. Key versioning (`v1:` prefix) for future rotation. | — |
| 59 | Backup & Restore | ✅ | — | `bitcode db backup [path]` and `bitcode db restore [path]`. Driver-aware: SQLite file copy, PostgreSQL pg_dump/psql, MySQL mysqldump/mysql. Supports `--gzip` compression and `--force` restore. Metadata JSON per backup. | Scheduled backups (depends on cron + storage). |
| 60 | Rate Limiting | ✅ | — | Fiber limiter middleware. Global rate limit (default 100/min) + stricter auth endpoint limit (5/min). Configurable via `rate_limit.*` config. Returns 429 with `Retry-After` header. | — |
| 61 | CSRF & XSS Protection | ⚠️ | S | XSS: Go html/template auto-escapes. API uses JWT (stateless, no CSRF needed). Current SSR forms use HTTPOnly JWT cookie (low risk). | CSRF token needed for future public web forms (job portal, contact forms, website module). Will implement when public web forms feature is built. |
| 62 | Soft Delete / Recycle Bin | ✅ | — | Model-level `soft_deletes: true` adds `deleted_at` column. DELETE sets `deleted_at = NOW()` + `active = false`. `FindAll`/`Count`/`Sum` exclude soft-deleted records (`deleted_at IS NULL`). Active variants (`FindAllActive`, `FindActive`, `CountActive`, `SumActive`, `PaginateActive`) filter by `active = true`. `active` is a separate business field (inactive ≠ deleted). Optimistic locking via `version: true`. Conditional timestamps/timestamps_by. | Recycle bin UI to view and restore deleted records (effort S). |
| 63 | Admin Impersonation | ✅ | — | Token-based impersonation: `POST /admin/api/impersonate/:user_id` (admin-only, cannot impersonate other admins, 1h TTL). `POST /admin/api/stop-impersonate` returns admin token. JWT `impersonated_by` claim. All audit logs include `impersonated_by` field. | — |
| 64 | Email Infrastructure | ✅ | — | SMTP email sender (`pkg/email`). Configurable via `smtp.*` config (host, port, user, password, from, TLS). HTML email templates. Used by 2FA, reusable for notifications, password reset, scheduled reports. | — |

---

## 10. Collaboration & Communication

| # | Feature | Status | Effort | What Exists | What's Missing |
|---|---------|--------|--------|-------------|----------------|
| 63 | Comment & Mention | ❌ | M | — | No comment system per record. Need comment model + API + `bc-chatter` component exists in Stencil but not wired. |
| 64 | Activity Feed | ❌ | M | — | No activity feed. `bc-activity` and `bc-timeline` Stencil components exist but not wired. Need feed API + WebSocket integration. |
| 65 | Email Inbox Integration | ❌ | XL | — | No email integration at all. Need IMAP client + email parsing + thread linking + SMTP send. |
| 66 | Task / To-Do per Record | ❌ | M | — | No task/todo system. Need todo model + API + UI widget. |
| 67 | In-App Notification | ⚠️ | M | WebSocket event broadcasting exists. Subscribe per channel. | No persistent notification model (read/unread), no notification preferences, no bell UI. |

---

## Web Components Status

The `@bitcode/components` Stencil library (94 components) provides rich UI widgets. Many are **built but not yet wired** to the engine's SSR rendering:

| Category | Components | Count | Engine Integration |
|----------|------------|-------|--------------------|
| Fields | string, text, richtext, code, date, select, link, file, image, signature, barcode, geo, rating, json, etc. | 34 | ⚠️ Component compiler maps some; SSR uses Go templates |
| Layout | row, column, section, tabs, sheet, header, separator, button-box, html-block | 9 | ⚠️ Partial via component compiler |
| Views | list, form, kanban, calendar, gantt, map, tree, report, activity | 9 | ⚠️ SSR renders HTML; components for SPA mode |
| Charts | line, bar, pie, area, gauge, funnel, heatmap, pivot, KPI, scorecard, progress | 11 | ✅ Chart view type uses ECharts |
| Datatable | datatable, filter-builder, lookup-modal | 3 | ❌ Not wired |
| Dialogs | modal, confirm, quick-entry, wizard, toast | 5 | ❌ Not wired |
| Search | search, filter-bar, filter-panel, favorites | 4 | ❌ Not wired |
| Widgets | badge, copy, phone, email, url, progress, statusbar, priority, handle, domain | 10 | ⚠️ Some via component compiler |
| Table | child-table | 1 | ❌ Not wired |
| Print | export, print, report-link | 3 | ❌ Not wired |
| Social | activity, chatter, timeline | 3 | ❌ Not wired |
| Other | placeholder | 1 | — |
| Editor | bc-view-editor (drag-drop form layout builder) | 1 | ✅ Wired to admin panel |
| **Total** | | **95** | |

**Key gap**: Components are built but the engine primarily uses SSR (Go templates). Full SPA mode with client-side component rendering is not yet implemented.

---

## Roadmap

### Phase 1 — Quick Wins (1–2 weeks, effort S)

Low-hanging fruit that immediately improves the platform:

- [ ] **#14** IP Whitelist / Session Policy — middleware for IP check + JWT expiry config
- [ ] **#15** Plugin Permission (granular) — extend permission pattern to `module.plugin.script_name`
- [ ] **#18** Login History view — extend audit_log fields + dedicated view
- [ ] **#20** Data Change Diff UI — store old_value/new_value per field + diff component
- [ ] **#46** API Key Management — api_key model + auth middleware accepting API key header
- [ ] **#52** Branding / White Label — branding settings (logo, app name, color) + template injection
- [ ] **#60** Rate Limiting — gofiber/limiter middleware
- [ ] **#61** CSRF Protection — CSRF middleware for SSR form routes
- [ ] **#62** Recycle Bin UI — filter `active = false` view + restore endpoint

### Phase 2 — Core Gaps (3–4 weeks, effort M)

Features most requested by users and blocking other features:

- [ ] **#5** Computed Field Evaluator — expression parser + evaluator at query time
- [ ] **#11** Field-Level Permission — per-field permission config in model JSON + API/form filter
- [ ] **#17** Record Activity Timeline — API endpoint + timeline UI component
- [ ] **#27** Assignment Rules — assignment rule JSON definition + evaluator on create/update
- [ ] **#28** Webhook System — webhook definition (URL, events, headers) + event bus dispatcher
- [ ] **#33** Multi-Step Form / Wizard — wire `bc-dialog-wizard` component + wizard JSON definition
- [ ] **#35** Web Form (Public) — public form route (bypass auth) + CAPTCHA + rate limiting
- [ ] **#41** Export Data (CSV/Excel) — server-side export handler per model
- [ ] **#57** 2FA (TOTP) — TOTP library + setup flow + verification middleware
- [ ] **#59** Backup & Restore — db dump/restore commands + optional scheduling
- [ ] **#63** Comment & Mention — comment model + API + wire `bc-chatter` component
- [ ] **#67** In-App Notification — notification model + preferences + bell UI + WebSocket delivery

### Phase 3 — Visual Builders (6–8 weeks, effort L)

The gap between "JSON-code" and true "low-code":

- [ ] **#1** Visual Schema Builder — frontend CRUD for model JSON + field type picker + relation builder
- [ ] **#22** Visual Workflow Builder — frontend state machine editor (nodes + edges + properties)
- [ ] **#30** Visual Form Builder — drag-and-drop form designer outputting JSON layout
- [ ] **#34** Print Format / PDF — print template JSON + PDF renderer (wkhtmltopdf/chromedp/gotenberg)
- [ ] **#38** Report Builder — report JSON definition (columns, filters, group_by, aggregations) + renderer
- [ ] **#47** Data Import / Export Wizard — upload → map columns → validate → insert

### Phase 4 — Enterprise Features (8–12 weeks, effort L–XL)

Enterprise-grade capabilities:

- [ ] **#7** Multi-Source Data — connection pool manager for multiple datasources
- [ ] **#26** Email / Notification Automation — SMTP config, email queue, template engine
- [ ] **#42** Pivot Table — server-side pivot engine + wire `bc-chart-pivot` component
- [ ] **#45** OAuth2 / SSO — OAuth2 client (Google, Microsoft, GitHub) + LDAP connector
- [ ] **#49** GraphQL API — schema generator from model definitions + resolver layer
- [ ] **#54** Multi-Currency — currency model + exchange rate table + conversion logic
- [ ] **#55** Multi-Company — company model + company-level settings + inter-company transactions
- [ ] **#64** Activity Feed — feed API (aggregate audit_log + comments) + wire `bc-activity` component
- [ ] **#66** Task / To-Do per Record — todo model + API + UI widget

### Phase 5 — Ecosystem (ongoing)

- [ ] **#43** Scheduled Reports — depends on email (#26) + report builder (#38)
- [ ] **#48** Third-Party Connectors — connector framework + individual implementations
- [ ] **#65** Email Inbox Integration — IMAP client + email parsing + thread linking
- [ ] **AGENTS.md** Redis cache wiring — wire into permission checker and query result cache
- [ ] **AGENTS.md** NATS event bus — replace in-process bus for distributed deployments
- [ ] **AGENTS.md** Marketplace — community module sharing
- [ ] **SPA Mode** — full client-side rendering using Stencil components (currently SSR only)

---

## Engine-Specific Feature Docs

Detailed per-feature documentation lives in `engine/docs/features/`:

| Feature | Doc | Status |
|---------|-----|--------|
| Models | [models.md](../engine/docs/features/models.md) | ✅ |
| APIs | [apis.md](../engine/docs/features/apis.md) | ✅ |
| Processes | [processes.md](../engine/docs/features/processes.md) | ✅ |
| Views & Templates | [views.md](../engine/docs/features/views.md) | ✅ |
| Modules | [modules.md](../engine/docs/features/modules.md) | ✅ |
| Security | [security.md](../engine/docs/features/security.md) | ✅ |
| Workflows | [workflows.md](../engine/docs/features/workflows.md) | ✅ |
| Agents & Cron | [agents.md](../engine/docs/features/agents.md) | ✅ |
| Plugins | [plugins.md](../engine/docs/features/plugins.md) | ✅ |
| i18n | [i18n.md](../engine/docs/features/i18n.md) | ✅ |
| Configuration | [configuration.md](../engine/docs/features/configuration.md) | ✅ |
| WebSocket | [websocket.md](../engine/docs/features/websocket.md) | ✅ |
| Multi-tenancy | [multitenancy.md](../engine/docs/features/multitenancy.md) | ✅ |
| Admin UI | [admin.md](../engine/docs/features/admin.md) | ✅ |
| File Storage | [storage.md](../engine/docs/features/storage.md) | ✅ |

---

## Test Coverage

181 tests across 28 packages. All passing. See [engine/docs/codebase.md](../engine/docs/codebase.md) for the full breakdown.

```bash
cd engine && go test ./... -v
```
