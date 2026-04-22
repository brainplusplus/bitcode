# BitCode Engine

A modular BitCode platform where you build applications by writing JSON definitions. The Go engine reads those definitions and produces a running app — complete with APIs, business logic, UI, background jobs, and security.

**JSON is the source code. Go is the runtime. Modules are the packaging.**

## Quick Start

```bash
# Clone and build
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

## Try the Sample ERP

```bash
cd samples/erp
MODULE_DIR=modules go run ../../engine/cmd/engine/main.go
```

Then test:
```bash
curl http://localhost:8080/health
curl http://localhost:8080/api/contacts
curl -X POST http://localhost:8080/api/contacts \
  -H "Content-Type: application/json" \
  -d '{"name":"Budi","email":"budi@test.com","company":"Acme"}'
```

## CLI Commands

```bash
bitcode init my-app          # Scaffold new project
bitcode dev                  # Start dev server (hot reload)
bitcode validate             # Validate all JSON definitions
bitcode module list          # List available modules
bitcode module create mymod  # Scaffold new module
bitcode user create admin admin@example.com
bitcode user list
bitcode db migrate           # Run database migrations
bitcode version
```

## Configuration

All config via environment variables. Defaults work out of the box (SQLite + memory cache).

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

### PostgreSQL

```bash
DB_DRIVER=postgres DB_HOST=localhost DB_NAME=myapp go run cmd/engine/main.go
```

### MySQL

```bash
DB_DRIVER=mysql DB_HOST=localhost DB_USER=root DB_PASSWORD=root DB_NAME=myapp go run cmd/engine/main.go
```

### Docker

```bash
docker-compose up -d
```

## Features

- **JSON-driven development** — Models, APIs, processes, views, all defined in JSON
- **Module system** — Install/uninstall modules with dependency resolution (like Odoo)
- **Auto-CRUD** — One line JSON = full REST API with pagination
- **Security** — JWT auth, RBAC permissions, record rules (row-level security)
- **Workflow engine** — State machines with permission-gated transitions
- **Process engine** — 14 step types (validate, query, create, update, delete, if, switch, loop, emit, call, script, http, assign, log)
- **Plugin system** — Custom TypeScript/Python code via JSON-RPC
- **Template engine** — Go html/template with helpers and partials
- **View system** — List, form, kanban, calendar, chart, custom views
- **Event bus** — Domain events with agent handlers
- **Cron scheduler** — Scheduled background jobs
- **i18n** — Multi-language translations
- **Multi-database** — SQLite (default), PostgreSQL, MySQL
- **Cache** — Memory (default), Redis (optional)
- **Hot reload** — File watcher in dev mode
- **WebSocket** — Real-time domain event broadcasting
- **Multi-tenancy** — Tenant isolation via header/subdomain/path
- **Admin UI** — Built-in web admin panel at `/admin`
- **Python plugins** — Python scripts alongside TypeScript
- **gRPC protocol** — High-throughput plugin communication (proto defined)

## Documentation

- [Architecture](docs/architecture.md) — System design, data flow, components
- [Codebase](docs/codebase.md) — File map, package responsibilities
- [Features](docs/features/) — Per-feature documentation:
  - [Models](docs/features/models.md)
  - [APIs](docs/features/apis.md)
  - [Processes](docs/features/processes.md)
  - [Views & Templates](docs/features/views.md)
  - [Modules](docs/features/modules.md)
  - [Security](docs/features/security.md)
  - [Workflows](docs/features/workflows.md)
  - [Agents & Cron](docs/features/agents.md)
  - [Plugins](docs/features/plugins.md)
  - [i18n](docs/features/i18n.md)
  - [Configuration](docs/features/configuration.md)
  - [WebSocket](docs/features/websocket.md)
  - [Multi-tenancy](docs/features/multitenancy.md)
  - [Admin UI](docs/features/admin.md)

## License

MIT
