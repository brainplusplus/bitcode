# BitCode

A JSON-driven low-code platform for building business applications. Define models, APIs, processes, views, and workflows in JSON — the Go engine interprets them at runtime to produce a fully functional application.

**JSON is the source code. Go is the runtime. Modules are the packaging.**

## Repository Structure

```
bitcode/
├── engine/              Go runtime — reads JSON, runs the app
├── packages/
│   ├── components/      Stencil Web Components (@bitcode/components)
│   └── tauri/           Tauri native shell (desktop + mobile)
├── samples/
│   └── erp/             Sample ERP application (CRM + HRM)
├── docs/                Project-level documentation
└── sprints/             Sprint planning & tracking
```

## Quick Start

```bash
# Install the CLI
cd engine
go install ./cmd/bitcode/

# Run (SQLite, zero config)
bitcode serve

# Or use go run
go run ./cmd/bitcode/ serve
```

Server starts at `http://localhost:8080`. SQLite database created automatically as `bitcode.db`.

### Try the Sample ERP

```bash
cd samples/erp
MODULE_DIR=modules go run ../../engine/cmd/bitcode/ serve
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

### Run Desktop App (Tauri)

Prerequisites: [Rust](https://rustup.rs/), [Tauri CLI](https://v2.tauri.app/start/prerequisites/)

```bash
# Quick start (Windows)
.\run_desktop.bat

# Quick start (macOS/Linux)
./run_desktop.sh

# Quick start (PowerShell — any OS)
.\run_desktop.ps1

# Or manually:
cd packages/components && npm install
cd ../tauri && npm run dev:desktop
```

The desktop app builds the Stencil components first, then launches a native window with SQLite, offline sync, and all native capabilities.

**Production build:**

```bash
cd packages/tauri
npm run build:desktop          # Release build (optimized, installer generated)
npm run build:desktop:debug    # Debug build (faster, no optimization)
```

Output: `packages/tauri/src-tauri/target/release/bundle/` — contains `.msi` (Windows), `.dmg` (macOS), `.deb`/`.AppImage` (Linux).

### Run Mobile App (Tauri)

Prerequisites: [Android Studio](https://developer.android.com/studio) or [Xcode](https://developer.apple.com/xcode/) + Tauri mobile prerequisites.

```bash
# Android — first time setup
cd packages/tauri
npm run build:android-init     # Generate Android project
npm run dev:android            # Dev mode on connected device/emulator

# iOS — first time setup (macOS only)
npm run build:ios-init         # Generate Xcode project
npm run dev:ios                # Dev mode on simulator/device

# Production builds
npm run build:android          # Release APK/AAB
npm run build:ios              # Release IPA
```

### Sync Status Widget

Drop `<bc-sync-status>` into any page to show offline sync status:

```html
<!-- Full view: online/offline indicator, pending count, sync button -->
<bc-sync-status></bc-sync-status>

<!-- Compact mode for toolbars -->
<bc-sync-status compact></bc-sync-status>

<!-- Custom poll interval (ms) -->
<bc-sync-status poll-interval="10000"></bc-sync-status>
```

Events: `bcSyncTriggered`, `bcSyncCompleted` — listen for sync lifecycle.

### Offline Mode with Encryption

```bash
# Enable SQLite encryption (optional)
# Requires SQLCipher: brew install sqlcipher (macOS) / apt install libsqlcipher-dev (Ubuntu)
cd packages/tauri
BITCODE_DB_KEY=your-secret-key cargo tauri build --features encryption
```

## CLI Commands

```bash
bitcode serve                # Start production server
bitcode dev                  # Start dev server (auto-detects mode, hot reload)
bitcode init my-app          # Scaffold new project
bitcode validate             # Validate all JSON definitions
bitcode module list          # List available modules
bitcode module create mymod  # Scaffold new module
bitcode publish base         # Extract embedded module to project
bitcode publish base --models    # Extract only models
bitcode publish air.toml     # Generate .air.toml config (auto-detects mode)
bitcode publish --list       # List publishable modules
bitcode user create admin admin@example.com
bitcode user list
bitcode db migrate           # Run database migrations
bitcode db backup            # Backup database (driver-aware)
bitcode db backup --gzip     # Compressed backup
bitcode db restore backup.db # Restore from backup
bitcode db restore --force   # Skip confirmation
bitcode seed run             # Run pending data migrations
bitcode seed run -m crm      # Run for specific module
bitcode seed rollback        # Rollback last batch
bitcode seed status          # Show migration status
bitcode seed fresh           # Reset and re-run all
bitcode seed create name -m mod --model model  # Create migration file
bitcode version
```

## Configuration

All config via environment variables or `bitcode.toml`/`bitcode.yaml`. Defaults work out of the box (SQLite + memory cache).

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | Server port |
| `DB_DRIVER` | `sqlite` | Database: `sqlite`, `postgres`, `mysql` |
| `DB_SQLITE_PATH` | `bitcode.db` | SQLite file path |
| `DB_SCHEMA` | `public` | Postgres schema (ignored for SQLite/MySQL) |
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
| `SECURITY_IP_WHITELIST_ENABLED` | `false` | Enable IP whitelist |
| `SECURITY_IP_WHITELIST` | - | Allowed IPs (comma-separated, supports CIDR) |
| `SECURITY_IP_WHITELIST_ADMIN_ONLY` | `true` | Restrict only admin routes |
| `SECURITY_SESSION_DURATION` | `24h` | JWT token / cookie lifetime |
| `SECURITY_COOKIE_SECURE` | `false` | HTTPS-only cookies |
| `SECURITY_COOKIE_SAMESITE` | `Lax` | Cookie SameSite policy |
| `AUTH_REGISTER_ENABLED` | `false` | Enable user registration page |

### PostgreSQL

```bash
DB_DRIVER=postgres DB_HOST=localhost DB_NAME=myapp bitcode serve
```

### MySQL

```bash
DB_DRIVER=mysql DB_HOST=localhost DB_USER=root DB_PASSWORD=root DB_NAME=myapp bitcode serve
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
- **Web Components** — 103 Stencil.js enterprise components (fields, layout, views, charts, dialogs, widgets, media viewers/players). **Standalone-capable** — works without BitCode framework
- **Component Theming** — Light/dark/system-detect/custom themes via CSS custom properties
- **4-Level Data Fetching** — Local data, URL endpoint, event intercept, custom fetcher function
- **3-Level Validation** — Built-in rules, custom JS validators, server-side validation
- **Event bus** — Domain events with agent handlers
- **Cron scheduler** — Scheduled background jobs
- **Multi-database** — SQLite (default), PostgreSQL, MySQL, MongoDB
- **Table prefix** — Per-module table name prefix (`"table": {"prefix": "crm"}` → `crm_contact`)
- **Postgres schema** — Configurable schema via `DB_SCHEMA` (default: `public`)
- **Query builder** — Comprehensive query builder for SQL and MongoDB with JSON DSL + OQL (Object Query Language — 3 syntax styles: SQL-like, simplified DSL, dot-notation). Supports JOINs, OR/AND/NOT groups, HAVING, DISTINCT, aggregates, subqueries, UNION, raw expressions, scopes, eager loading, locking, soft delete scopes
- **Model process registry** — Built-in `models.{name}.{op}` functions (Get, FindAll, Create, Update, Delete, Upsert, Count, Sum, Avg, Min, Max, Pluck, Exists, Aggregate, WithTrashed, OnlyTrashed, Increment, Decrement)
- **Cache** — Memory (default), Redis (optional)
- **Real-time** — WebSocket domain event broadcasting
- **Multi-tenancy** — Tenant isolation via header/subdomain/path
- **i18n** — Multi-language (11 languages in components)
- **Admin UI** — Built-in panel at `/admin`
- **Hot reload** — File watcher in dev mode
- **Native shell (Tauri)** — Tauri 2.0 wraps Stencil components for desktop (Win/Mac/Linux) and mobile (iOS/Android). `bc-native.ts` bridge abstracts native capabilities with Web API fallback
- **Offline mode** — One toggle (`mode:"offline"`) enables offline-first with auto-generated sync infrastructure, local SQLite, outbox pattern, delta sync, field-level conflict resolution, offline auth (72h), encrypted storage, and inventory tracking

## Documentation

| Doc | Description |
|-----|-------------|
| [Architecture](docs/architecture.md) | System design, data flow, core concepts, tech stack |
| [Codebase](docs/codebase.md) | Full file map for engine, components, and samples |
| [Features & Roadmap](docs/features.md) | 73-feature inventory, completion status, phased roadmap |
| [Component Docs](packages/components/docs/README.md) | Per-component reference (props, events, methods, examples) |
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
| Native Shell | Tauri 2.0 (Rust) |
| Plugins | Node.js + Python via JSON-RPC |

## License

MIT
