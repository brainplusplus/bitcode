# AGENTS.md — BitCode Platform

## Understanding the Project

Read these docs **before** making any changes:

| Doc | What It Covers | When to Read |
|-----|---------------|--------------|
| [`docs/architecture.md`](docs/architecture.md) | System design, data flow, core concepts, tech stack | First time touching the project, or when working across layers |
| [`docs/codebase.md`](docs/codebase.md) | Full file map with one-line descriptions per file | When you need to find where something lives |
| [`docs/features.md`](docs/features.md) | Feature inventory (67 features), status, gaps, roadmap | When planning work or understanding what exists vs what's missing |
| [`engine/docs/features/`](engine/docs/features/) | Per-feature deep docs (models, APIs, processes, views, etc.) | When working on a specific engine feature |

## Project Overview

Modular low-code ERP platform. Go engine reads JSON definitions and produces running applications. Inspired by Odoo, Frappe, NocoBase.

**JSON is the source code. Go is the runtime. Modules are the packaging. Stencil Web Components are the UI.**

```
bitcode/
├── engine/              Go runtime
│   ├── cmd/             Entry point (bitcode CLI — serve, dev, init, validate, etc.)
│   ├── internal/        Private app code (compiler, domain, runtime, infrastructure, presentation)
│   ├── pkg/             Public packages (ddd, security, plugin SDK)
│   ├── modules/         Built-in modules (base, crm, sales)
│   ├── embedded/        Go-embedded modules compiled into binary
│   └── plugins/         Plugin runtimes (TypeScript, Python)
├── packages/
│   └── components/      Stencil Web Components (@bitcode/components, 94 components)
├── samples/
│   └── erp/             Sample ERP application (CRM + HRM)
├── docs/                Project-level documentation
└── sprints/             Sprint planning & tracking
```

## Conventions

- **Go 1.23+**, standard project layout (`cmd/`, `internal/`, `pkg/`)
- **DDD internally**, flat JSON externally — users never see DDD terms
- **Convention over configuration** — sensible defaults everywhere
- **All PK and FK are UUID** (TEXT in SQLite, UUID in Postgres, CHAR(36) in MySQL)
- **Tests**: every package with logic has `_test.go`. Run `cd engine && go test ./...`
- **No comments** unless absolutely necessary (complex algorithms, public API docs)
- **No type suppression** — no `as any`, `@ts-ignore`, or equivalents

## File Structure Rules (Engine)

- `engine/internal/compiler/parser/` — JSON parsers. One file per definition type.
- `engine/internal/domain/` — Domain models. DDD patterns (entity, aggregate, events). No DB imports.
- `engine/internal/runtime/` — Execution engines (process executor, agents, workflow, plugins).
- `engine/internal/infrastructure/` — External concerns (DB, cache, module loader, i18n).
- `engine/internal/presentation/` — HTTP layer (routes, middleware, views, templates).
- `engine/pkg/` — Public packages reusable outside engine (ddd, security, plugin SDK).
- `engine/modules/` — Built-in modules (base is always installed).
- `engine/embedded/` — Go-embedded modules compiled into the binary.
- `engine/plugins/` — Plugin runtimes (TypeScript, Python).

## File Structure Rules (Components)

- `packages/components/src/components/` — Stencil Web Components (each has `.tsx` + `.css`).
- `packages/components/src/core/` — Shared infrastructure (types, API client, event bus, form engine, i18n).
- `packages/components/src/utils/` — Shared utilities (expression eval, format, validators).
- `packages/components/src/i18n/` — Translation files (11 languages).

## When Making Changes

### Adding a New Feature (Engine)

1. Add parser in `engine/internal/compiler/parser/` if it has a JSON definition
2. Add domain types in `engine/internal/domain/` if it has business logic
3. Add runtime in `engine/internal/runtime/` if it executes something
4. Add infrastructure in `engine/internal/infrastructure/` if it talks to external systems
5. Wire in `engine/internal/app.go`
6. Add feature doc in `engine/docs/features/`
7. Write tests

### Adding / Modifying Components

1. Add/edit component in `packages/components/src/components/`
2. Update `packages/components/src/components.d.ts` if new component
3. Run `npm run build` in `packages/components/` to verify

### Documentation Updates (MANDATORY)

**After ANY code change, update the relevant docs:**

| What Changed | Update These |
|-------------|-------------|
| New file or directory added | `docs/codebase.md` — add to file map |
| Architecture change (new layer, new service, new data flow) | `docs/architecture.md` — update diagrams and component tables |
| Feature added, completed, or status changed | `docs/features.md` — update status (✅/⚠️/❌), check off roadmap items |
| Engine feature added/changed | `engine/docs/features/*.md` — update or create per-feature doc |
| New module or module structure change | `docs/codebase.md` — update module section |
| New component added to packages/components | `docs/codebase.md` — update components section |
| Public API changed (new endpoint, new config) | `README.md` — update config table, CLI commands, or feature list |
| Major milestone or project-level change | `README.md` — update overview |

**Rule: if you changed code, you changed docs. No exceptions.**

### i18n Check (MANDATORY)

**After ANY implementation that adds user-facing text (templates, error messages, labels, UI strings):**

1. Check if the text needs i18n support
2. If yes — use the `t` template function (`{{t .Locale "key"}}`) instead of hardcoded strings
3. Add translation keys to the module's `i18n/*.json` files
4. At minimum provide `en` (English) translations as the default
5. If the module already has other locale files (e.g., `id.json`), add translations there too

**Rule: no hardcoded user-facing strings in templates. Use i18n keys.**

**Required languages (11):** `en`, `id`, `ar`, `de`, `es`, `fr`, `ja`, `ko`, `pt-BR`, `ru`, `zh-CN`

All 11 locale files must be provided for every module that has user-facing text. English (`en`) is the default fallback.

## What To Work On Next

### Completed ✅

- [x] Data seeding — `module/seeder.go` loads `data/*.json` during module install
- [x] Many2many junction tables — Auto-created in `MigrateModel()` for many2many fields
- [x] Model inheritance — `MergeInheritedFields()` merges parent fields when `inherit` is set
- [x] Workflow integration in CRUD — Initial state on create, `WorkflowAction()` validates transitions
- [x] Auth endpoints — `POST /auth/login`, `POST /auth/register` wired via `AuthHandler`
- [x] Form submission — `POST /views/:name` handles form submissions
- [x] Kanban/Calendar/Chart renderers — All 6 view types implemented in `view/renderer.go`
- [x] Process `call` step loader — `ProcessRegistry` stores and loads processes by name
- [x] Plugin TypeScript runtime — `plugins/typescript/index.js` Node.js JSON-RPC process
- [x] Pagination in list views — `page`, `page_size`, `total_pages` in API responses
- [x] Search — `?q=term` searches across fields listed in API `search` config
- [x] File upload handler — Enhanced: local + S3 storage, attachments table, thumbnails, versioning, path formatting, duplicate detection
- [x] WebSocket — `websocket/hub.go` broadcasts domain events to connected clients
- [x] Multi-tenancy — Tenant middleware (header/subdomain/path) + repository isolation
- [x] Admin UI — `admin/admin.go` Frappe-inspired panel at `/admin` (sidebar, dashboard, models with tabs, modules with tabs, views, health)
- [x] Python plugin runtime — `plugins/python/runtime.py` JSON-RPC over stdin/stdout
- [x] gRPC plugin protocol — `pkg/plugin/proto/plugin.proto` service definition
- [x] Template layout system — Views wrapped in layout with sidebar, navbar, modern CSS
- [x] Shared partials — Partial templates available across all templates
- [x] Default templates — Base module ships with layout, list, form, kanban, calendar, chart, login, home templates
- [x] Cross-module views — `register_to` field + `module.view_name` URL syntax (graceful if module missing)
- [x] Modern UI — Polished CSS design system with cards, tables, badges, kanban boards, responsive layout
- [x] DAG executor — Parallel step execution for process engine
- [x] Component compiler — Compiles view JSON into Stencil Web Component HTML
- [x] Stencil Web Components — 102 components (fields, layout, views, charts, dialogs, widgets, media viewers/players, search, social, print)
- [x] Media viewers — 8 viewer/player components (PDF, image, document, YouTube, Instagram, TikTok, video, audio) + file upload preview integration
- [x] Embedded module system — Base module embedded in binary via `go:embed`, 3-layer resolution (project → global → embedded)
- [x] `bitcode publish` CLI — Extract embedded modules to project for customization (whole/per-type/per-file)
- [x] Menu visibility — `menu_visibility` field in module.json (`app` or `admin`)
- [x] Include menus — `include_menus` field to import menu items from other modules
- [x] View editor — Admin view detail with tabs (info, preview, editor, revisions), JSON editor, `bc-view-editor` Stencil component
- [x] View versioning — `view_revisions` DB table, auto-revision on save, rollback, configurable limit
- [x] Primary key strategies — 6 strategies (auto-increment, composite, UUID v4/v7/format, natural key, naming series, manual), format template engine (30+ functions), atomic sequence engine
- [x] Table prefix — Per-module `"table": {"prefix": "crm"}` config, per-model override, centralized `ResolveTableName()` resolver, no pluralization
- [x] Postgres schema — `DB_SCHEMA` config, `search_path` set at connection, `CREATE SCHEMA IF NOT EXISTS`
- [x] MongoDB support — Full parity MongoDB driver with extended reference pattern, unified query builder, MongoDB sequence engine, system collection migration
- [x] Query builder — Comprehensive `Query` type with JSON DSL + OQL (Object Query Language, 3 syntax styles). Supports: OR/AND/NOT condition groups, JOINs (INNER/LEFT/RIGHT/CROSS/FULL), HAVING, DISTINCT, aggregates (COUNT/SUM/AVG/MIN/MAX), subqueries (IN/EXISTS), UNION/UNION ALL, raw expressions, named scopes, eager loading (WITH/preload for many2one/one2many/many2many), locking (FOR UPDATE/FOR SHARE), soft delete scopes (withTrashed/onlyTrashed), field sanitization. Operators: =, !=, >, <, >=, <=, like, not_like, in, not_in, between, not_between, is_null, is_not_null, column comparison. Translates to GORM chain (SQL) or $and/$or bson (MongoDB) with full parity
- [x] Repository interface — `Repository` interface abstraction, `GenericRepository` (SQL) and `MongoRepository` (MongoDB) implementations, factory-based creation
- [x] Model process registry — `models.{name}.{op}` built-in functions: Get/Find, GetAll/FindAll, Paginate, Create, Update, Delete, Upsert, Count, Sum + Active variants (FindActive, FindAllActive, PaginateActive, CountActive, SumActive) + Avg, Min, Max, Pluck, Exists, Aggregate, WithTrashed, OnlyTrashed, Increment, Decrement. OQL support via `oql` arg. Dynamic finders: FindBy{Field}, FindAllBy{Field}, CountBy{Field}, ExistsBy{Field}, DeleteBy{Field}, SumBy{AggField}{Field}, AvgBy, MinBy, MaxBy, PluckBy — with And/Or connectors, operator suffixes (Gt/Gte/Lt/Lte/Like/In/NotIn/Between/IsNull/IsNotNull/Not), and OrderBy{Field}Asc/Desc. Module-qualified names: `models.crm.contact.FindAll` with ambiguity detection for cross-module models.
- [x] Model options — `version` (optimistic locking), `timestamps`, `timestamps_by`, `soft_deletes` (deleted_at), `soft_deletes_by` (deleted_by) in model JSON. Conditional column generation, UpdateWithVersion (409 conflict), SoftDeleteWithTimestamp
- [x] Extended model fields — `title_field` (display name resolution chain), `search_field` (searchable fields config)
- [x] Login with username or email — Auth handler accepts username or email in login field

- [x] Two-Factor Auth (2FA) — Email OTP with temp token flow, enable/disable/validate endpoints
- [x] Field-Level Encryption — AES-256-GCM with `"encrypted": true` in model JSON, key versioning
- [x] Backup & Restore — `bitcode db backup/restore`, driver-aware (SQLite/Postgres/MySQL), gzip support
- [x] Rate Limiting — Fiber limiter middleware, tiered (global 100/min, auth 5/min), configurable
- [x] Admin Impersonation — Token-based, JWT `impersonated_by` claim, audit trail, safety guards
- [x] Email Infrastructure — SMTP sender (`pkg/email`), HTML templates, configurable via `smtp.*`
- [x] Audit Log Impersonation — `impersonated_by` column in audit_logs, auto-populated from JWT claims
- [x] IP Whitelist / Session Policy — IP whitelist middleware (exact IP + CIDR), configurable session duration, cookie Secure/SameSite flags
- [x] Auth Module — Embedded `auth` module (login, register, forgot, reset, 2FA verify), `module.json` `auth` field, `menu_visibility: "none"`, i18n (11 languages), `?next=` sanitization, settings-driven OTP config

### Remaining (Engine)

- [x] Computed field evaluation — Expression evaluator for `sum(lines.subtotal)` at query time
- [x] Model lifecycle events — `events` in model JSON: 16 event types (before/after validate, create, update, delete, save, soft_delete, hard_delete, restore + on_change). Process/script handlers with condition, sync/async, retry, priority, bulk_mode. Repository-layer injection. Auto event bus publish.
- [x] Field validation — `validation` in model JSON: 40+ built-in validators, conditional (required_if, when, on), custom (process/script), model-level cross-field validators. 422 error response. Auto-maps existing required/max/min.
- [x] Field sanitization — `sanitize` in model JSON: 14 built-in sanitizers (trim, lowercase, slugify, etc.). Model-level `_all_strings` shorthand.
- [ ] Redis cache wiring — Wire into permission checker and query result cache
- [ ] GraphQL API — Alternative to REST
- [ ] Marketplace — Community module sharing
- [ ] NATS event bus — Replace in-process bus for distributed deployments

> **Full roadmap with 69 features**: see [`docs/features.md`](docs/features.md)

## Testing

```bash
cd engine
go test ./... -v          # All tests
go test ./pkg/ddd/        # Specific package
go test ./... -count=1    # No cache
```

Current: 381 tests, 0 failures. Build: OK.

## Build

```bash
cd engine
make build    # Build bitcode binary
make install  # Install bitcode to $GOPATH/bin
make serve    # Start production server
make dev      # Start dev server (auto-detects mode)
make test     # Run tests
make tidy     # go mod tidy
```
