# Codebase Map

## Directory Structure

```
engine/
├── cmd/                                    # Entry points
│   ├── engine/main.go                      # Server — connects DB, loads modules, starts Fiber
│   └── bitcode/
│       ├── main.go                         # CLI — init, dev, validate, module, user, db, publish, version
│       └── publish.go                      # `bitcode publish` — extract embedded modules to project
│
├── internal/                               # Private application code
│   ├── app.go                              # Central wiring — NewApp(), LoadModules(), Start()
│   │
│   ├── compiler/parser/                    # JSON → Go struct parsers
│   │   ├── model.go                        # ModelDefinition, FieldDefinition, field types, validation
│   │   ├── model_test.go                   # 10 tests (valid, inheritance, missing fields, relationships)
│   │   ├── api.go                          # APIDefinition, ExpandAutoCRUD(), GetBasePath()
│   │   ├── api_test.go                     # 8 tests (auto_crud, workflow, custom, RLS)
│   │   ├── process.go                      # ProcessDefinition, StepDefinition, 14 step types
│   │   ├── view.go                         # ViewDefinition, 6 view types (list/form/kanban/calendar/chart/custom)
│   │   ├── view_test.go                    # 6 tests
│   │   ├── agent.go                        # AgentDefinition, triggers, cron, retry
│   │   ├── module.go                       # ModuleDefinition, permissions, groups, menu, settings
│   │   ├── workflow.go                     # WorkflowDefinition, states, transitions, CanTransition()
│   │   └── workflow_test.go                # 3 tests (parse, transitions, multi-from)
│   │
│   ├── domain/                             # Business logic (no DB imports)
│   │   ├── model/
│   │   │   ├── registry.go                 # Register/Get/List/Has models, TableName()
│   │   │   └── registry_test.go            # 6 tests
│   │   ├── security/
│   │   │   ├── user.go                     # User aggregate — NewUser, CheckPassword, Activate/Deactivate, HasPermission
│   │   │   ├── role.go                     # Role aggregate — HasPermission (with inheritance), AllPermissions
│   │   │   ├── group.go                    # Group aggregate — AllGroupNames (with implied groups)
│   │   │   ├── permission.go               # Permission value object
│   │   │   ├── record_rule.go              # RecordRule — AppliesToGroup, AppliesToOperation, InterpolateDomain
│   │   │   └── security_test.go            # 9 tests (user, role inheritance, groups, record rules)
│   │   ├── event/
│   │   │   ├── bus.go                      # In-process event bus — Subscribe, SubscribeAll, Publish
│   │   │   └── bus_test.go                 # 4 tests
│   │   └── setting/
│   │       ├── setting.go                  # Key-value store — Get, Set, GetWithDefault, LoadDefaults
│   │       └── setting_test.go             # 5 tests
│   │
│   ├── runtime/                            # Execution engines
│   │   ├── executor/
│   │   │   ├── executor.go                 # ProcessExecutor — RegisterHandler, Execute
│   │   │   ├── executor_test.go            # 3 tests
│   │   │   └── steps/
│   │   │       ├── validate.go             # Validate step — eq, neq, required rules
│   │   │       ├── data.go                 # Query/Create/Update/Delete steps via GenericRepository
│   │   │       ├── control.go              # If/Switch/Loop steps — condition evaluation, variable resolution
│   │   │       ├── emit.go                 # Emit step — add event to context
│   │   │       ├── call.go                 # Call step — invoke sub-process
│   │   │       ├── script.go               # Script step — invoke plugin via ScriptRunner interface
│   │   │       ├── http.go                 # HTTP step — external API calls
│   │   │       ├── util.go                 # Assign + Log steps
│   │   │       └── steps_test.go           # 9 tests (validate, emit, assign, if, parse)
│   │   ├── agent/
│   │   │   ├── worker.go                   # Agent worker — subscribe to events, execute with retry
│   │   │   └── cron.go                     # Cron scheduler — periodic job execution
│   │   ├── workflow/
│   │   │   └── engine.go                   # Workflow engine — Register, ExecuteTransition, GetInitialState
│   │   └── plugin/
│   │       └── manager.go                  # Plugin manager — spawn process, JSON-RPC over stdin/stdout
│   │
│   ├── infrastructure/                     # External concerns
│   │   ├── persistence/
│   │   │   ├── database.go                 # NewDatabase() — SQLite/Postgres/MySQL via config
│   │   │   ├── dynamic_model.go            # MigrateModel() — CREATE TABLE from ModelDefinition, dialect-aware
│   │   │   ├── repository.go               # GenericRepository — Create/FindByID/FindAll/Update/Delete/HardDelete
│   │   │   ├── view_revision.go            # ViewRevision model + repository (CRUD, cleanup, auto-migrate)
│   │   │   └── view_revision_test.go       # 6 tests (create, list, get, cleanup)
│   │   ├── cache/
│   │   │   ├── cache.go                    # Cache interface + NewCache() factory (memory or redis)
│   │   │   ├── memory.go                   # MemoryCache — in-process with TTL
│   │   │   ├── memory_test.go              # 5 tests (set/get, TTL expiry, delete, clear)
│   │   │   └── redis.go                    # RedisCache — Redis-backed cache
│   │   ├── module/
│   │   │   ├── registry.go                 # Module registry — Register/Get/IsInstalled/List
│   │   │   ├── dependency.go               # ResolveDependencies() — topological sort, circular detection
│   │   │   ├── loader.go                   # LoadModule() + LoadModuleFromFS() — parse module, load models + APIs
│   │   │   ├── fs.go                       # ModuleFS interface, DiskFS, EmbedFS, LayeredFS, SubFS, ExtractModuleFS
│   │   │   ├── fs_test.go                  # 32 tests (DiskFS, EmbedFS, LayeredFS, SubFS, Extract)
│   │   │   ├── integration_test.go         # 4 integration tests (3-layer resolution, override, mixed)
│   │   │   └── module_test.go              # 7 tests (registry, dependencies, parse)
│   │   ├── i18n/
│   │   │   ├── loader.go                   # Translator — LoadFile/LoadJSON, Translate with fallback
│   │   │   └── i18n_test.go                # 4 tests
│   │   └── watcher/
│   │       └── watcher.go                  # FileWatcher — poll for .json/.html changes, trigger reload
│   │
│   └── presentation/                       # HTTP layer
│       ├── api/
│       │   ├── router.go                   # Dynamic route registration from API definitions
│       │   └── crud_handler.go             # Auto-CRUD handler — List/Read/Create/Update/Delete
│       ├── admin/
│       │   ├── admin.go                    # Admin panel — sidebar, dashboard, models (tabs), modules (tabs), views (list+detail+editor), health
│       │   └── admin_api.go                # Admin JSON API — view save, rollback, preview, publish
│       ├── middleware/
│       │   ├── auth.go                     # JWT validation, user context injection
│       │   ├── permission.go               # RBAC permission checking via PermissionChecker interface
│       │   ├── record_rule.go              # RLS filter injection via RecordRuleEngine interface
│       │   └── audit.go                    # Audit logging for write operations
│       ├── template/
│       │   ├── engine.go                   # Go html/template engine — LoadDirectory, LoadString, Render, RenderWithLayout, shared partials
│       │   └── engine_test.go              # 5 tests
│       └── view/
│           └── renderer.go                 # View renderer — list, form, kanban, calendar, chart, custom (SSR) with layout wrapping
│
├── pkg/                                    # Public packages
│   ├── ddd/
│   │   ├── entity.go                       # Entity interface + BaseEntity
│   │   ├── aggregate.go                    # Aggregate interface + BaseAggregate (with domain events)
│   │   ├── domain_event.go                 # DomainEvent interface + BaseDomainEvent + NewDomainEvent()
│   │   ├── repository.go                   # Repository[T] generic interface
│   │   ├── value_object.go                 # ValueObject interface
│   │   └── ddd_test.go                     # 3 tests
│   ├── security/
│   │   ├── password.go                     # HashPassword (bcrypt), CheckPassword
│   │   ├── jwt.go                          # GenerateToken, ValidateToken, Claims struct
│   │   └── security_test.go                # 5 tests
│   └── plugin/                             # Plugin SDK (for TS/Python plugins)
│
├── embedded/                               # Go-embedded assets compiled into binary
│   ├── embed.go                            # //go:embed directive for ModulesFS
│   ├── embed_test.go                       # Verify embedding works
│   └── modules/
│       └── base/                           # Core module (embedded, always available)
│           ├── module.json                 # 11 permissions, 2 groups, menu, menu_visibility: admin
│           ├── models/                     # user, role, group, permission, record_rule, audit_log, setting
│           ├── apis/                       # auth_api, user_api, group_api, etc.
│           ├── views/                      # 13 view definitions (list + form for each model)
│           ├── data/                       # default_users, default_roles, default_groups
│           └── templates/                  # Default UI templates
│               ├── layout.html             # Main layout (sidebar + navbar + content area)
│               ├── layout-app.html         # App layout variant
│               ├── partials/               # Reusable components (sidebar, navbar, pagination, badges, actions)
│               └── views/                  # View templates (list, form, kanban, calendar, chart, login, home)
│
├── Dockerfile                              # Multi-stage build
├── docker-compose.yml                      # Engine + Postgres + Redis
├── Makefile                                # build, cli, dev, test, lint, clean, tidy
├── go.mod
└── go.sum
```

## Test Coverage

| Package | Tests | What's Tested |
|---------|-------|---------------|
| `pkg/ddd` | 3 | Entity, Aggregate events, DomainEvent |
| `pkg/security` | 5 | Password hash/check, JWT generate/validate/wrong-secret |
| `compiler/parser` | 27 | Model (10), API (8), View (6), Workflow (3) |
| `domain/model` | 6 | Registry CRUD, module prefix, table name |
| `domain/security` | 9 | User lifecycle, role inheritance, groups, record rules |
| `domain/event` | 4 | Pub/sub, subscribe all, no subscribers |
| `domain/setting` | 5 | Get/set, defaults, all |
| `infrastructure/cache` | 5 | Memory cache set/get, TTL, delete, clear |
| `infrastructure/module` | 43 | Registry, dependencies, DiskFS, EmbedFS, LayeredFS, SubFS, Extract, LoadModuleFromFS, integration |
| `embedded` | 4 | Verify base module embedded correctly |
| `infrastructure/i18n` | 4 | Translate, fallback, locale detection |
| `presentation/template` | 5 | Load/render, helpers (truncate, dict, eq) |
| `runtime/executor` | 3 | Step dispatch, unknown type, step error |
| `runtime/executor/steps` | 9 | Validate (eq/fail/required), emit, assign, if, process parse |
| `infrastructure/persistence` | 6 | ViewRevision CRUD, list, cleanup |
| **Total** | **181** | |

## Key Interfaces

```go
// Cache — memory or redis
type Cache interface {
    Get(key string) (any, bool)
    Set(key string, value any, ttl time.Duration)
    Delete(key string)
    Clear()
}

// StepHandler — process step execution
type StepHandler interface {
    Execute(ctx context.Context, execCtx *Context, step StepDefinition) error
}

// ScriptRunner — plugin execution
type ScriptRunner interface {
    Run(ctx context.Context, script string, params map[string]any) (any, error)
}

// PermissionChecker — RBAC
type PermissionChecker interface {
    UserHasPermission(userID string, permission string) (bool, error)
}

// RecordRuleEngine — RLS
type RecordRuleEngine interface {
    GetFilters(userID string, modelName string, operation string) ([][]any, error)
}

// Repository[T] — generic data access
type Repository[T any] interface {
    Save(ctx context.Context, entity *T) error
    FindByID(ctx context.Context, id string) (*T, error)
    FindAll(ctx context.Context, filters map[string]interface{}, page int, pageSize int) ([]T, int64, error)
    Update(ctx context.Context, entity *T) error
    Delete(ctx context.Context, id string) error
}
```
