# BitCode

A JSON-driven low-code platform for building business applications. Define models, APIs, processes, views, and workflows in JSON — the Go engine interprets them at runtime to produce a fully functional application.

**JSON is the source code. Go is the runtime. Modules are the packaging.**

## Repository Structure

```
bitcode/
├── engine/              Go runtime — reads JSON, runs the app
├── packages/
│   └── components/      Stencil Web Components (@bitcode/components)
├── samples/
│   └── erp/             Sample ERP application (CRM + HRM)
├── docs/                Project-level documentation
└── sprints/             Sprint planning & tracking
```

## Quick Start

```bash
# Build the engine
cd engine
go mod tidy
go build -o bin/engine cmd/engine/main.go
go build -o bin/bitcode cmd/bitcode/main.go

# Run (SQLite, zero config)
./bin/engine

# Or use go run
go run cmd/engine/main.go
```

Server starts at `http://localhost:8080`. SQLite database created automatically as `bitcode.db`.

### Try the Sample ERP

```bash
cd samples/erp
MODULE_DIR=modules go run ../../engine/cmd/engine/main.go
```

```bash
curl http://localhost:8080/health
curl http://localhost:8080/api/contacts
curl -X POST http://localhost:8080/api/contacts \
  -H "Content-Type: application/json" \
  -d '{"name":"Budi","email":"budi@test.com","company":"Acme"}'
```

### Build Web Components

```bash
cd packages/components
npm install
npm run build
```

## CLI Commands

```bash
bitcode init my-app          # Scaffold new project
bitcode dev                  # Start dev server (hot reload)
bitcode validate             # Validate all JSON definitions
bitcode module list          # List available modules
bitcode module create mymod  # Scaffold new module
bitcode publish base         # Extract embedded module to project
bitcode publish base --models    # Extract only models
bitcode publish --list       # List publishable modules
bitcode user create admin admin@example.com
bitcode user list
bitcode db migrate           # Run database migrations
bitcode db backup            # Backup database (driver-aware)
bitcode db backup --gzip     # Compressed backup
bitcode db restore backup.db # Restore from backup
bitcode db restore --force   # Skip confirmation
bitcode version
```

## Configuration

All config via environment variables or `bitcode.toml`/`bitcode.yaml`. Defaults work out of the box (SQLite + memory cache).

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | Server port |
| `DB_DRIVER` | `sqlite` | Database: `sqlite`, `postgres`, `mysql` |
| `DB_SQLITE_PATH` | `bitcode.db` | SQLite file path |
| `DB_HOST` | `localhost` | DB host (postgres/mysql) |
| `DB_PORT` | `5432` | DB port |
| `DB_USER` | `bitcode` | DB user |
| `DB_PASSWORD` | `bitcode` | DB password |
| `DB_NAME` | `bitcode` | DB name |
| `CACHE_DRIVER` | `memory` | Cache: `memory`, `redis` |
| `REDIS_URL` | - | Redis URL (only if CACHE_DRIVER=redis) |
| `JWT_SECRET` | `change-me...` | JWT signing secret |
| `MODULE_DIR` | `modules` | Path to modules directory |
| `TENANT_ENABLED` | `false` | Enable multi-tenancy |
| `TENANT_STRATEGY` | `header` | Tenant detection: `header`, `subdomain`, `path` |
| `TENANT_HEADER` | `X-Tenant-ID` | Header name for tenant ID |
| `STORAGE_DRIVER` | `local` | Storage: `local`, `s3` |
| `STORAGE_LOCAL_PATH` | `uploads` | Local upload directory |
| `STORAGE_S3_BUCKET` | - | S3 bucket name |
| `STORAGE_S3_REGION` | - | S3 region |
| `RATE_LIMIT_ENABLED` | `true` | Enable rate limiting |
| `RATE_LIMIT_MAX` | `100` | Max requests per window |
| `RATE_LIMIT_WINDOW` | `1m` | Rate limit window |
| `RATE_LIMIT_AUTH_MAX` | `5` | Auth endpoint limit |
| `RATE_LIMIT_AUTH_WINDOW` | `1m` | Auth endpoint window |
| `SMTP_HOST` | - | SMTP server host |
| `SMTP_PORT` | `587` | SMTP server port |
| `SMTP_USER` | - | SMTP username |
| `SMTP_PASSWORD` | - | SMTP password |
| `SMTP_FROM` | - | From address |
| `SMTP_TLS` | `true` | Use TLS |
| `ENCRYPTION_KEY` | - | AES-256 key (base64, 32 bytes) |

### PostgreSQL

```bash
DB_DRIVER=postgres DB_HOST=localhost DB_NAME=myapp go run engine/cmd/engine/main.go
```

### MySQL

```bash
DB_DRIVER=mysql DB_HOST=localhost DB_USER=root DB_PASSWORD=root DB_NAME=myapp go run engine/cmd/engine/main.go
```

### Docker

```bash
cd engine
docker-compose up -d
```

## Features

- **JSON-driven development** — Models, APIs, processes, views, workflows — all defined in JSON
- **Module system** — Dependency resolution, data seeding, cross-module views (Odoo-style)
- **Auto-CRUD** — One JSON file = full REST API with pagination, search, filtering
- **Security** — JWT auth, RBAC permissions, record rules (row-level security), audit logging, 2FA (email OTP), field-level encryption (AES-256-GCM), rate limiting, admin impersonation
- **Workflow engine** — State machines with permission-gated transitions
- **Process engine** — 14 step types (validate, query, create, update, delete, if, switch, loop, emit, call, script, http, assign, log)
- **Plugin system** — TypeScript + Python via JSON-RPC, gRPC proto defined
- **Template engine** — Go html/template with helpers and partials
- **File storage** — Local + S3 with attachments table, thumbnails, versioning, path formatting
- **View system** — List, form, kanban, calendar, chart, custom views (SSR)
- **Web Components** — 94 Stencil.js components (fields, layout, views, charts, dialogs, widgets)
- **Event bus** — Domain events with agent handlers
- **Cron scheduler** — Scheduled background jobs
- **Multi-database** — SQLite (default), PostgreSQL, MySQL
- **Cache** — Memory (default), Redis (optional)
- **Real-time** — WebSocket domain event broadcasting
- **Multi-tenancy** — Tenant isolation via header/subdomain/path
- **i18n** — Multi-language (11 languages in components)
- **Admin UI** — Built-in panel at `/admin`
- **Hot reload** — File watcher in dev mode

## Documentation

| Doc | Description |
|-----|-------------|
| [Architecture](docs/architecture.md) | System design, data flow, core concepts, tech stack |
| [Codebase](docs/codebase.md) | Full file map for engine, components, and samples |
| [Features & Roadmap](docs/features.md) | 67-feature inventory, completion status, phased roadmap |
| [Engine Architecture](engine/docs/architecture.md) | Engine internals, layer diagram, data flow |
| [Engine Codebase](engine/docs/codebase.md) | Engine file map, test coverage, key interfaces |
| [Engine Features](engine/docs/features/) | Per-feature deep docs: |

- [Models](engine/docs/features/models.md) — Field types, relationships, record rules, inheritance
- [APIs](engine/docs/features/apis.md) — Auto-CRUD, custom endpoints, auth, search
- [Processes](engine/docs/features/processes.md) — 14 step types, execution context, DAG
- [Views & Templates](engine/docs/features/views.md) — List, form, kanban, calendar, chart, custom
- [Modules](engine/docs/features/modules.md) — Module system, dependencies, 3-layer FS
- [Security](engine/docs/features/security.md) — JWT, RBAC, record rules, audit
- [Workflows](engine/docs/features/workflows.md) — State machines, transitions, permissions
- [Agents & Cron](engine/docs/features/agents.md) — Event triggers, cron scheduler, retry
- [Plugins](engine/docs/features/plugins.md) — TypeScript + Python runtime, JSON-RPC
- [i18n](engine/docs/features/i18n.md) — Translation files, locale fallback
- [Configuration](engine/docs/features/configuration.md) — Env vars, TOML/YAML, Viper
- [WebSocket](engine/docs/features/websocket.md) — Real-time event broadcasting
- [Multi-tenancy](engine/docs/features/multitenancy.md) — Header/subdomain/path strategies
- [Admin UI](engine/docs/features/admin.md) — Built-in admin panel
- [File Storage](engine/docs/features/storage.md) — Local + S3, attachments, thumbnails, versioning

| Other | |
|-------|---|
| [Sample ERP](samples/erp/README.md) | CRM + HRM sample with full feature coverage |

## Tech Stack

| Layer | Technology |
|-------|------------|
| Runtime | Go 1.23+, Fiber v2, GORM |
| Database | SQLite / PostgreSQL / MySQL |
| Cache | In-memory / Redis |
| Config | Viper (env + TOML/YAML) |
| CLI | Cobra |
| Web Components | Stencil.js (TypeScript) |
| Charts | ECharts |
| Rich Text | Tiptap |
| Code Editor | CodeMirror |
| Plugins | Node.js + Python via JSON-RPC |

## License

MIT
