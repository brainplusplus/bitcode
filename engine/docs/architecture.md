# Architecture

## System Overview

```
┌──────────────────────────────────────────────────────────────┐
│                        HTTP Client                           │
└──────────────────────────┬───────────────────────────────────┘
                           │
┌──────────────────────────▼───────────────────────────────────┐
│                     Fiber HTTP Server                        │
│  ┌─────────┐ ┌────────────┐ ┌─────────────┐ ┌───────────┐  │
│  │  Auth   │→│ Permission │→│ Record Rule │→│  Audit    │  │
│  │Middleware│ │ Middleware │ │ Middleware  │ │Middleware │  │
│  └─────────┘ └────────────┘ └─────────────┘ └───────────┘  │
│                           │                                  │
│  ┌────────────────────────▼─────────────────────────────┐   │
│  │              Route Handler                            │   │
│  │  ┌──────────┐  ┌──────────┐  ┌───────────────────┐  │   │
│  │  │Auto-CRUD │  │ Process  │  │  View Renderer    │  │   │
│  │  │ Handler  │  │ Executor │  │  (SSR HTML)       │  │   │
│  │  └────┬─────┘  └────┬─────┘  └────────┬──────────┘  │   │
│  └───────┼──────────────┼─────────────────┼─────────────┘   │
└──────────┼──────────────┼─────────────────┼─────────────────┘
           │              │                 │
┌──────────▼──────────────▼─────────────────▼─────────────────┐
│                    Internal Services                         │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────────┐   │
│  │ Generic  │ │ Workflow │ │  Event   │ │   Plugin     │   │
│  │Repository│ │  Engine  │ │   Bus    │ │  Manager     │   │
│  └────┬─────┘ └──────────┘ └────┬─────┘ └──────┬───────┘   │
│       │                         │               │            │
│  ┌────▼─────┐            ┌──────▼──────┐ ┌─────▼────────┐  │
│  │ Database │            │Agent Worker │ │TS/Python     │  │
│  │SQLite/PG │            │Cron Scheduler│ │Process       │  │
│  │  /MySQL  │            └─────────────┘ └──────────────┘  │
│  └──────────┘                                               │
└─────────────────────────────────────────────────────────────┘
```

## Components

### Compiler Layer (`internal/compiler/parser/`)

Parses JSON definitions into Go structs. One parser per definition type.

| Parser | Input | Output |
|--------|-------|--------|
| `model.go` | Model JSON | `ModelDefinition` (fields, relationships, record_rules, indexes) |
| `api.go` | API JSON | `APIDefinition` (endpoints, auto_crud, workflow actions) |
| `process.go` | Process JSON | `ProcessDefinition` (steps with types) |
| `view.go` | View JSON | `ViewDefinition` (list, form, kanban, custom) |
| `agent.go` | Agent JSON | `AgentDefinition` (triggers, cron, retry) |
| `module.go` | Module JSON | `ModuleDefinition` (deps, permissions, groups, menu, settings) |
| `workflow.go` | Workflow JSON | `WorkflowDefinition` (states, transitions) |

### Domain Layer (`internal/domain/`)

Business logic. No database imports. Pure Go.

| Package | Responsibility |
|---------|---------------|
| `model/` | Model registry — register/get/list model definitions |
| `security/` | User, Role, Group, Permission, RecordRule aggregates |
| `event/` | In-process event bus (publish/subscribe) |
| `setting/` | Key-value settings store per module |

### Runtime Layer (`internal/runtime/`)

Execution engines.

| Package | Responsibility |
|---------|---------------|
| `executor/` | Process executor — dispatches steps, manages context |
| `executor/steps/` | Step handlers: validate, data (CRUD), control (if/switch/loop), emit, call, script, http, assign, log |
| `agent/` | Agent worker (event subscription) + cron scheduler |
| `workflow/` | State machine engine — validate transitions, get initial state |
| `plugin/` | Plugin manager — spawn processes, JSON-RPC communication |

### Infrastructure Layer (`internal/infrastructure/`)

External concerns.

| Package | Responsibility |
|---------|---------------|
| `persistence/` | Database connection (SQLite/Postgres/MySQL), dynamic table migration, generic repository |
| `cache/` | Cache interface + MemoryCache (default) + RedisCache (optional) |
| `module/` | Module registry, dependency resolver, module loader, ModuleFS (DiskFS/EmbedFS/LayeredFS), 3-layer resolution |
| `i18n/` | Translation loader and translator |
| `watcher/` | File watcher for hot reload in dev mode |

### Presentation Layer (`internal/presentation/`)

HTTP-facing code.

| Package | Responsibility |
|---------|---------------|
| `api/` | Dynamic route registration, auto-CRUD handler |
| `middleware/` | Auth (JWT), Permission (RBAC), RecordRule (RLS), Audit logging |
| `template/` | Go html/template engine with helpers (formatDate, formatCurrency, truncate, dict, eq) |
| `view/` | View renderer — list, form, custom (SSR HTML) |

### Public Packages (`pkg/`)

Reusable outside the engine.

| Package | Responsibility |
|---------|---------------|
| `ddd/` | Entity, Aggregate, DomainEvent, Repository, ValueObject interfaces |
| `security/` | Password hashing (bcrypt), JWT generation/validation |
| `plugin/` | Plugin SDK (for TypeScript/Python plugins) |

## Data Flow

### API Request

```
HTTP Request
  → Fiber Router (matched from API definition)
  → Auth Middleware (validate JWT, load user context)
  → Permission Middleware (check RBAC permission)
  → Record Rule Middleware (inject row-level filters)
  → Audit Middleware (log write operations)
  → Handler
      → Auto-CRUD: GenericRepository.FindAll/Create/Update/Delete
      → Custom: ProcessExecutor.Execute(steps)
      → View: ViewRenderer.RenderView(template + data)
  → Response (JSON for API, HTML for views)
  → Domain Events → Event Bus → Agent handlers
```

### Module Loading (3-Layer Resolution)

```
Build LayeredFS:
  Layer 1: Project FS    → ./modules/         (highest priority, user overrides)
  Layer 2: Global FS     → ~/.bitcode/modules/ (shared across projects)
  Layer 3: Embedded FS   → binary (go:embed)   (default fallback, base module)

Discover modules across all layers
  → Per-file merge: project file overrides embedded file
  → Parse each module.json
  → Resolve dependencies (topological sort)
  → For each module (in dependency order):
      → Parse models → Register in ModelRegistry → Migrate DB tables
      → Parse APIs → Register Fiber routes
      → Parse views → Register in view map
      → Load templates
      → Build menu (respect menu_visibility + include_menus)
      → Register in ModuleRegistry
```

CLI: `bitcode publish base` extracts embedded module files to project for customization.

### Process Execution

```
ProcessExecutor.Execute(process, input, userID)
  → For each step:
      → Dispatch to StepHandler by type
      → Handler reads/writes Context (input, variables, result, events)
      → On error: return error with step info
  → Return Context (result, variables, events)
  → Publish events to EventBus
```

## Database

### Supported Drivers

| Driver | Default | UUID Strategy | JSON Type |
|--------|---------|--------------|-----------|
| SQLite | ✅ Yes | TEXT (app-generated) | TEXT |
| PostgreSQL | No | UUID (gen_random_uuid) | JSONB |
| MySQL | No | CHAR(36) (app-generated) | JSON |

### Auto-generated Columns

Every table gets these columns automatically:

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID | Primary key |
| `created_at` | Timestamp | Auto-set on create |
| `updated_at` | Timestamp | Auto-set on update |
| `created_by` | UUID (FK) | User who created |
| `updated_by` | UUID (FK) | User who last updated |
| `active` | Boolean | Soft delete flag (default true) |

### Relationship Mapping

| JSON Type | DB Implementation |
|-----------|------------------|
| `many2one` | FK column (UUID) |
| `one2many` | No column — resolved via inverse FK query |
| `many2many` | Junction table (model1_model2 with both FKs) |

## Security

### Middleware Chain

```
Auth → Permission → RecordRule → Handler
```

1. **Auth**: Validate JWT token, extract user_id/roles/groups into context
2. **Permission**: Check user has required permission (auto-derived from model: `model.action`)
3. **RecordRule**: Apply row-level filters from model's `record_rules` based on user's groups
4. **Audit**: Log all write operations (POST/PUT/DELETE)

### Rule: `auth: true` = Permission + RLS

When `auth: true` is set on an API, permissions and record rules are automatically enforced. No separate flag needed. Record rules are defined on the model, not the API.

## Caching

| Driver | When to Use |
|--------|-------------|
| Memory (default) | Development, single instance, small datasets |
| Redis (optional) | Production, multiple instances, large datasets |

Config: `CACHE_DRIVER=memory` (default) or `CACHE_DRIVER=redis REDIS_URL=redis://...`
