# AGENTS.md — BitCode Engine

## Project Overview

Modular BitCode ERP platform. Go engine reads JSON definitions and produces running applications. Inspired by Yao Engine, Odoo, Frappe.

## Conventions

- **Go 1.21+**, standard project layout (`cmd/`, `internal/`, `pkg/`)
- **DDD internally**, flat JSON externally — users never see DDD terms
- **Convention over configuration** — sensible defaults everywhere
- **All PK and FK are UUID** (TEXT in SQLite, UUID in Postgres, CHAR(36) in MySQL)
- **Tests**: every package with logic has `_test.go`. Run `go test ./...`
- **No comments** unless absolutely necessary (complex algorithms, public API docs)
- **No `as any`, `@ts-ignore`** equivalents — no type suppression

## File Structure Rules

- `internal/compiler/parser/` — JSON parsers. One file per definition type.
- `internal/domain/` — Domain models. DDD patterns (entity, aggregate, events). No DB imports.
- `internal/runtime/` — Execution engines (process executor, agents, workflow, plugins).
- `internal/infrastructure/` — External concerns (DB, cache, module loader, i18n).
- `internal/presentation/` — HTTP layer (routes, middleware, views, templates).
- `pkg/` — Public packages reusable outside engine (ddd, security, plugin SDK).
- `modules/` — Built-in modules (base is always installed).
- `samples/` — Example applications.

## When Adding a New Feature

1. Add parser in `internal/compiler/parser/` if it has a JSON definition
2. Add domain types in `internal/domain/` if it has business logic
3. Add runtime in `internal/runtime/` if it executes something
4. Add infrastructure in `internal/infrastructure/` if it talks to external systems
5. Wire in `internal/app.go`
6. **Update `docs/architecture.md`** — add to component list and data flow
7. **Update `docs/codebase.md`** — add new files to the file map
8. Add feature doc in `docs/features/`
9. Write tests

## What To Work On Next

### Completed ✅
- [x] **Data seeding** — `module/seeder.go` loads `data/*.json` during module install
- [x] **Many2many junction tables** — Auto-created in `MigrateModel()` for many2many fields
- [x] **Model inheritance** — `MergeInheritedFields()` merges parent fields when `inherit` is set
- [x] **Workflow integration in CRUD** — Initial state on create, `WorkflowAction()` validates transitions
- [x] **Auth endpoints** — `POST /auth/login`, `POST /auth/register` wired via `AuthHandler`
- [x] **Form submission** — `POST /views/:name` handles form submissions
- [x] **Kanban/Calendar/Chart renderers** — All 6 view types implemented in `view/renderer.go`
- [x] **Process `call` step loader** — `ProcessRegistry` stores and loads processes by name
- [x] **Plugin TypeScript runtime** — `plugins/typescript/index.js` Node.js JSON-RPC process
- [x] **Pagination in list views** — `page`, `page_size`, `total_pages` in API responses
- [x] **Search** — `?q=term` searches across fields listed in API `search` config
- [x] **File upload handler** — `POST /api/upload` stores files, `GET /uploads/*` serves them

- [x] **WebSocket** — `websocket/hub.go` broadcasts domain events to connected clients
- [x] **Multi-tenancy** — Tenant middleware (header/subdomain/path) + repository isolation
- [x] **Admin UI** — `admin/admin.go` built-in panel at `/admin` (dashboard, models, modules, views)
- [x] **Python plugin runtime** — `plugins/python/runtime.py` JSON-RPC over stdin/stdout
- [x] **gRPC plugin protocol** — `pkg/plugin/proto/plugin.proto` service definition
- [x] **Template layout system** — Views wrapped in layout with sidebar, navbar, modern CSS
- [x] **Shared partials** — Partial templates available across all templates
- [x] **Default templates** — Base module ships with layout, list, form, kanban, calendar, chart, login, home templates
- [x] **Cross-module views** — `register_to` field + `module.view_name` URL syntax (graceful if module missing)
- [x] **Modern UI** — Polished CSS design system with cards, tables, badges, kanban boards, responsive layout

### Remaining
- [ ] **Computed field evaluation** — Expression evaluator for `sum(lines.subtotal)` at query time
- [ ] **Redis cache wiring** — Wire into permission checker and query result cache
- [ ] **GraphQL API** — Alternative to REST
- [ ] **Marketplace** — Community module sharing
- [ ] **NATS event bus** — Replace in-process bus for distributed deployments

## Testing

```bash
go test ./... -v          # All tests
go test ./pkg/ddd/        # Specific package
go test ./... -count=1    # No cache
```

Current: 93 tests, 0 failures. Build: OK.

## Build

```bash
make build    # Build engine binary
make cli      # Build CLI binary
make test     # Run tests
make dev      # Run dev server
make tidy     # go mod tidy
```
