# BitCode Platform — Codebase Map

## Repository Structure

```
bitcode/
├── docs/                                       # Project-level documentation
│   ├── architecture.md                         # System design, data flow, concepts
│   ├── codebase.md                             # This file — full file map
│   └── plans/                                  # Design documents & implementation plans
│       ├── DESIGN.md
│       ├── 2026-04-18-component-system-design.md
│       ├── 2026-04-18-component-system-plan.md
│       ├── 2026-04-22-fix-process-engine-design.md
│       ├── 2026-04-22-i18n-implementation-plan.md
│       ├── 2026-04-22-i18n-stencil-components-design.md
        │       ├── 2026-04-25-model-options-design.md
        │       └── 2026-04-26-media-viewers-design.md
│
├── engine/                                     # Go runtime (the core)
├── packages/                                   # Shared libraries
│   ├── components/                             # Stencil Web Components
│   ├── go-json/                                # go-json — JSON/JSONC programming language engine (Go)
│   └── tauri/                                  # Tauri native shell (desktop + mobile)
├── samples/                                    # Example applications
│   └── erp/                                    # Full ERP sample
├── sprints/                                    # Sprint tracking
└── .gitignore
```

---

## Engine (`engine/`)

The Go backend that reads JSON definitions and runs the application.

```
engine/
├── cmd/                                        # Entry points
│   └── bitcode/
│       ├── main.go                             # CLI — serve, dev, init, validate, module, user, db, seed, version, publish, security
│       ├── publish.go                          # Module publish command (extract embedded modules)
│       ├── publish_crud.go                     # publish:crud command — generate API/page override files from auto-generated CRUD
│       ├── security.go                         # security CLI — load/export/diff/validate/history (JSON↔DB sync)
│       ├── backup.go                           # db backup/restore commands (SQLite/Postgres/MySQL)
│       └── seed.go                             # Data migration CLI — seed run/rollback/status/fresh/create
│
├── internal/                                   # Private application code
│   ├── app.go                                  # Central wiring — NewApp(), LoadModules(), Start(), Shutdown()
│   │                                           #   Registers step handlers, middleware, routes
│   │                                           #   Module install: parse → register → migrate → seed
│   ├── config.go                               # Viper-based config — env vars + TOML/YAML file
│   │
│   ├── compiler/parser/                        # JSON → Go struct parsers
│   │   ├── model.go                            # ModelDefinition, FieldDefinition, field types, validation rules, APIConfig, ProtocolConfig, mask/mask_length/groups on fields
│   │   ├── model_test.go                       # 13 tests (valid, inheritance, missing fields, relationships, API config, field mask/groups)
│   │   ├── api.go                              # APIDefinition, ExpandAutoCRUD(), GetBasePath()
│   │   ├── api_test.go                         # 8 tests (auto_crud, workflow, custom, RLS)
│   │   ├── security.go                         # SecurityDefinition, SecurityACL (with "all" shorthand), SecurityRuleDefinition, ParseSecurity/ParseSecurityFile
│   │   ├── security_test.go                    # 4 tests (basic group, all shorthand, validation errors, rule defaults)
│   │   ├── process.go                          # ProcessDefinition, StepDefinition, 14 step type constants
│   │   ├── view.go                             # ViewDefinition, 6 view types (list/form/kanban/calendar/chart/custom)
│   │   ├── view_test.go                        # 6 tests
│   │   ├── agent.go                            # AgentDefinition, triggers, cron expressions, retry config
│   │   ├── migration.go                        # MigrationDefinition, source types (JSON/CSV/XLSX/XML), processors, conflict modes
│   │   ├── module.go                           # ModuleDefinition, permissions, groups, securities, pages, menu (with groups), settings, migrations, i18n patterns
│   │   ├── workflow.go                         # WorkflowDefinition, states, transitions, CanTransition()
│   │   └── workflow_test.go                    # 3 tests (parse, transitions, multi-from)
│   │
│   │   ├── domain/                                 # Business logic (no DB imports)
│   │   ├── model/
│   │   │   ├── registry.go                     # Register/Get/List/Has models, TableName()
│   │   │   └── registry_test.go                # 6 tests
│   │   ├── security/
│   │   │   ├── user.go                         # User aggregate — NewUser, CheckPassword, IsSuperuser, AllGroupNames
│   │   │   ├── role.go                         # Role aggregate — HasPermission (with inheritance) [DEPRECATED — being replaced by Group]
│   │   │   ├── group.go                        # Group aggregate — AllGroupNames (with implied groups), share, comment, module, modified_source
│   │   │   ├── permission.go                   # Permission value object [DEPRECATED — replaced by ModelAccess]
│   │   │   ├── model_access.go                 # ModelAccess entity — 12 ERPNext-style permissions per model per group (select/read/write/create/delete/print/email/report/export/import/mask/clone)
│   │   │   ├── model_access_test.go            # 4 tests (HasPermission, AllPermissions, SetFromList, IsGlobal)
│   │   │   ├── security_history.go             # SecurityHistory entity — audit trail for group/ACL/rule changes with snapshot + rollback
│   │   │   ├── security_history_test.go        # 2 tests (create, update)
│   │   │   ├── record_rule.go                  # RecordRule — m2m Groups, AppliesToGroupNames, IsGlobal, module, modified_source
│   │   │   └── security_test.go                # 14 tests (user, role inheritance, groups, record rules, superuser, AllGroupNames, share, m2m)
│   │   ├── event/
│   │   │   ├── bus.go                          # In-process event bus — Subscribe, SubscribeAll, Publish
│   │   │   └── bus_test.go                     # 4 tests
│   │   ├── setting/
│   │   │   ├── setting.go                      # Key-value store — Get, Set, GetWithDefault, LoadDefaults
│   │   │   └── setting_test.go                 # 5 tests
│   │   └── storage/
│   │       ├── storage.go                      # StorageDriver interface, PutOptions, URLOptions, ScanHook interface
│   │       └── attachment.go                   # Attachment entity — GORM model for attachments table
│   │
│   ├── runtime/                                # Execution engines
│   │   ├── executor/
│   │   │   ├── executor.go                     # ProcessExecutor — RegisterHandler, Execute, step dispatch
│   │   │   ├── executor_test.go                # 3 tests
│   │   │   ├── process_loader.go               # ProcessRegistry — Register, Get, List processes by name
│   │   │   ├── dag.go                          # DAG executor for parallel step execution
│   │   │   ├── dag_test.go                     # DAG tests
│   │   │   └── steps/                          # Step handler implementations
│   │   │       ├── validate.go                 # Validate step — eq, neq, required rules
│   │   │       ├── data.go                     # Query/Create/Update/Delete steps via GenericRepository
│   │   │       ├── control.go                  # If/Switch/Loop steps — condition evaluation, variable resolution
│   │   │       ├── emit.go                     # Emit step — add domain event to context
│   │   │       ├── call.go                     # Call step — invoke sub-process by name
│   │   │       ├── script.go                   # Script step — invoke TS/Python plugin via ScriptRunner
│   │   │       ├── http.go                     # HTTP step — external API calls with method/url/headers/body
│   │   │       ├── util.go                     # Assign + Log step handlers
│   │   │       └── steps_test.go               # 9 tests (validate, emit, assign, if, parse)
│   │   ├── expression/
│   │   │   ├── evaluator.go                    # Expression evaluator — lexer, parser, AST, arithmetic/comparison/boolean/functions
│   │   │   ├── evaluator_test.go               # 17 tests (arithmetic, fields, aggregates, comparisons, functions)
│   │   │   └── hydrator.go                     # Computed field hydrator — loads one2many children, evaluates computed/formula fields
│   │   ├── validation/
│   │   │   ├── validator.go                    # Field validation engine — ValidateCreate, ValidateUpdate, auto-map, short-circuit
│   │   │   ├── rules.go                        # Built-in validation rules — email, phone, regex, date, etc.
│   │   │   ├── conditional.go                  # Conditional validators — required_if, when, expression evaluator
│   │   │   ├── sanitizer.go                    # Sanitization engine — trim, lowercase, slugify, strip_tags, etc.
│   │   │   ├── errors.go                       # ValidationErrors — per-field error accumulation
│   │   │   ├── adapter.go                      # ValidatorAdapter — implements persistence.FieldValidator interface
│   │   │   ├── validator_test.go               # 18 tests
│   │   │   └── sanitizer_test.go               # 10 tests
│   │   ├── hook/
│   │   │   ├── dispatcher.go                   # Event dispatcher — priority sort, condition eval, sync/async, retry, timeout
│   │   │   ├── context.go                      # EventContext — model, data, old_data, changes, session, bulk info
│   │   │   ├── model_hooks.go                  # ModelHookDispatcher — bridges repository interface with dispatcher
│   │   │   ├── expr.go                         # Expression evaluator for handler conditions
│   │   │   └── dispatcher_test.go              # 10 tests
│   │   ├── agent/
│   │   │   ├── worker.go                       # Agent worker — subscribe to events, execute with retry
│   │   │   └── cron.go                         # Cron scheduler — periodic job execution
│   │   ├── workflow/
│   │   │   └── engine.go                       # Workflow engine — Register, ExecuteTransition, GetInitialState
│   │   ├── model_process.go                    # ModelProcessRegistry — built-in model operations (CRUD, aggregates, dynamic finders)
│   │   ├── dynamic_finder.go                   # Dynamic finder parser — FindBy{Field}, CountBy, SumBy, etc. with And/Or/operator suffixes
│   │   ├── dynamic_finder_test.go              # 24 tests for dynamic finder parsing
│   │   └── plugin/
│   │       └── manager.go                      # Plugin manager — spawn TS/Python processes, JSON-RPC over stdin/stdout
│   │
│   ├── infrastructure/                         # External concerns
│   │   ├── persistence/
│   │   │   ├── database.go                     # NewDatabase() — SQLite/Postgres/MySQL connection via GORM
│   │   │   ├── repository_interface.go         # Repository interface, SystemRepository, SequenceEngine, MigrationEngine interfaces
│   │   │   ├── query.go                        # Comprehensive Query builder — OR/AND/NOT groups, JOINs, HAVING, DISTINCT,
│   │   │   │                                   #   aggregates (COUNT/SUM/AVG/MIN/MAX), subqueries, UNION, raw expressions,
│   │   │   │                                   #   scopes, eager loading (WITH/preload), locking, soft delete scopes,
│   │   │   │                                   #   field sanitization, JSON/Map/Domain parsers
│   │   │   ├── query_test.go                   # Tests for query builder (all features)
│   │   │   ├── oql.go                          # OQL (Object Query Language) parser — 3 syntax styles:
│   │   │   │                                   #   Style A: SQL-like (JPQL/HQL), Style B: Simplified DSL,
│   │   │   │                                   #   Style C: Dot-notation. Auto-detect via ParseOQL()
│   │   │   ├── oql_test.go                     # Tests for OQL parser (all 3 styles)
│   │   │   ├── mongo_connection.go             # OpenMongoDB() — MongoDB connection via official driver
│   │   │   ├── mongo_repository.go             # MongoRepository — MongoDB implementation of Repository interface
│   │   │   ├── mongo_migration.go              # MongoMigrationEngine — index creation, system collection setup
│   │   │   ├── mongo_sequence.go               # MongoSequenceEngine — counter collection pattern for sequences
│   │   │   ├── mongo_system.go                 # MongoSystemRepository + MongoAuditLogRepository
│   │   │   ├── dynamic_model.go                # MigrateModel() — CREATE TABLE from ModelDefinition, dialect-aware DDL
│   │   │   │                                   #   MergeInheritedFields() — model inheritance field merging
│   │   │   │                                   #   Auto-creates junction tables for many2many
│   │   │   ├── repository.go                   # GenericRepository (SQL) — implements Repository interface with GORM
│   │   │   │                                   #   Full query translation: JOINs, OR/AND/NOT, HAVING, DISTINCT,
│   │   │   │                                   #   subqueries, locking, soft delete scopes, eager loading
│   │   │   │                                   #   Avg/Min/Max/Pluck/Exists/Aggregate/Chunk/Increment/Decrement
│   │   │   │                                   #   Transaction support, RawQuery/RawExec, relation preloading
│   │   │   │                                   #   Computed field hydration via expression.Hydrator
│   │   │   │                                   #   Data revision snapshots on write operations
│   │   │   ├── data_revision.go                # DataRevision — full record snapshots for rollback/restore
│   │   │   │                                   #   Monotonic versioning per (model, record_id), change diff
│   │   │   ├── data_revision_test.go           # 7 tests (create, version increment, list, get, cleanup, changes, latest)
│   │   │   ├── view_revision.go                # ViewRevision — view JSON snapshots for editor versioning
│   │   │   ├── view_revision_test.go           # 6 tests
│   │   │   ├── audit_log.go                    # AuditLogRepository — persistent audit log writer with async support
│   │   │   │                                   #   FindByRecord, FindByUser, FindLoginHistory, FindRequests
│   │   │   │                                   #   ImpersonatedBy field, AutoMigrateAuditLog()
│   │   │   ├── audit_log_test.go               # 5 tests (write, find by record, requests, user, login history)
│   │   │   ├── permission_checker.go            # PermissionService — resolves 12 permissions per model per user via ModelAccess + Group chain (additive, default-deny, superuser bypass)
│   │   │   ├── permission_checker_test.go      # 14 tests (superuser, default-deny, single group, additive, global ACL, implied groups)
│   │   │   ├── record_rule_service.go          # RecordRuleService — Odoo-compatible rule composition: global INTERSECT, group UNION. Domain interpolation.
│   │   │   ├── record_rule_service_test.go     # 11 tests (superuser, no rules, global, group in/out, operation filter, implied, legacy, inactive, interpolation)
│   │   │   ├── migration_tracker.go            # MigrationTracker — ir_migration table, batch tracking, status
│   │   │   └── backup.go                       # Backup/Restore — driver-aware (SQLite copy, pg_dump, mysqldump)
│   │   ├── cache/
│   │   │   ├── cache.go                        # Cache interface + NewCache() factory
│   │   │   ├── memory.go                       # MemoryCache — in-process with TTL
│   │   │   ├── memory_test.go                  # 5 tests (set/get, TTL expiry, delete, clear)
│   │   │   └── redis.go                        # RedisCache — Redis-backed implementation
│   │   ├── module/
│   │   │   ├── registry.go                     # Module registry — Register/Get/IsInstalled/InstalledNames
│   │   │   ├── dependency.go                   # ResolveDependencies() — topological sort, circular detection
│   │   │   ├── loader.go                       # LoadModule() — parse module dir, collect models + APIs + securities
│   │   │   ├── auto_api.go                     # GenerateAPIFromModel() — auto-creates APIDefinition from model "api" config. MergeAPIs() — merge auto-generated + override APIs. pluralize()
│   │   │   ├── auto_api_test.go                # 6 tests (basic, no-api, override endpoints, custom API, workflow override, pluralize)
│   │   │   ├── security_loader.go              # SecurityLoader — loads securities/*.json, syncs groups/ACL/rules/menus/pages to DB. Respects modified_source="ui" (noupdate). Idempotent.
│   │   │   ├── security_loader_test.go         # 6 tests (basic sync, implies, record rules, UI-modified preservation, all 12 permissions, idempotent)
│   │   │   ├── fs.go                           # DiskFS, EmbedFS, LayeredFS — module filesystem abstraction
│   │   │   ├── fs_test.go                      # FS tests
│   │   │   ├── reader.go                       # Multi-format data readers (JSON, CSV, XLSX, XML)
│   │   │   ├── migration.go                    # MigrationEngine, DataInserter (GORM+Mongo), RunUp/RunDown, processors, depends_on topo sort
│   │   │   ├── migration_test.go               # 26 tests (readers, upsert, field mapping, defaults, tracker, topo sort, circular detection)
│   │   │   ├── module_test.go                  # 7 tests (registry, dependencies, parse)
│   │   │   └── integration_test.go             # Integration tests
│   │   ├── i18n/
│   │   │   ├── loader.go                       # Translator — LoadFile/LoadJSON, Translate with locale fallback
│   │   │   └── i18n_test.go                    # 4 tests
│   │   ├── storage/
│   │   │   ├── config.go                       # StorageConfig, LocalStorageConfig, S3StorageConfig, ThumbnailConfig
│   │   │   ├── local.go                        # LocalStorage — filesystem StorageDriver implementation
│   │   │   ├── s3.go                           # S3Storage — AWS S3 StorageDriver implementation
│   │   │   ├── formatter.go                    # FormatPath/FormatName — template variable resolution
│   │   │   │                                   #   NewStorageDriver() — factory for local/S3
│   │   │   ├── repository.go                   # AttachmentRepository — GORM CRUD for attachments table
│   │   │   │                                   #   AutoMigrateAttachments(), FindByHash, FindVersions, CleanupVersions
│   │   │   └── thumbnail.go                    # ThumbnailService — generate thumbnails, on-demand resize
│   │   └── watcher/
│   │       └── watcher.go                      # FileWatcher — poll for .json/.html changes, trigger reload
│   │
│   └── presentation/                           # HTTP layer
│       ├── api/
│       │   ├── router.go                       # Dynamic route registration — auto-CRUD from model + override merge, permission + record rule middleware wiring
│       │   ├── crud_handler.go                 # Auto-CRUD handler — List/Read/Create/Update/Delete with pagination, field masking, field groups filtering, permission injection in response
│       │   ├── field_filter.go                 # Server-side field filtering — field groups (hide), field masking (****1234), per-response
│       │   ├── field_filter_test.go            # 11 tests (masking, field groups, nil model, unknown fields, list filtering)
│       │   ├── swagger.go                      # SwaggerGenerator — auto-generates OpenAPI 3.0 spec from model + API definitions. Swagger UI at /api/v1/docs
│       │   ├── auth_handler.go                 # POST /auth/login, /register, /logout, /2fa/enable, /2fa/disable, /2fa/validate
│       │   ├── upload_handler.go               # Legacy upload handler (replaced by file_handler)
│       │   └── file_handler.go                 # FileHandler — upload, download, list, delete, versions, resize, thumbnail
│       ├── middleware/
│       │   ├── auth.go                         # JWT validation, user context injection, impersonated_by extraction
│       │   ├── permission.go                   # Permission checking via PermissionChecker interface (ModelAccess-based)
│       │   ├── record_rule.go                  # RLS filter injection with {{user.id}} interpolation
│       │   ├── audit.go                        # Audit logging for write operations (includes impersonated_by)
│       │   ├── ratelimit.go                    # Rate limiting middleware (Fiber limiter, tiered: global + auth)
│       │   ├── ipwhitelist.go                  # IP whitelist middleware (exact IP + CIDR, admin-only or global)
│       │   └── tenant.go                       # Multi-tenancy middleware (header/subdomain/path)
│       ├── graphql/
│       │   ├── schema.go                       # SchemaBuilder — auto-generates GraphQL schema from model definitions (types, queries, mutations)
│       │   ├── resolver.go                     # Resolver — CRUD resolvers with permission + record rule enforcement, context-based user ID
│       │   ├── handler.go                      # Fiber HTTP handler for GraphQL at POST /api/v1/graphql
│       │   └── schema_test.go                  # 5 tests (empty schema, model fields, skip non-graphql, mutations, field type mapping)
│       ├── template/
│       │   ├── engine.go                       # Go html/template engine — LoadDirectory, Render, RenderWithLayout
│       │   └── engine_test.go                  # 5 tests
│       ├── view/
│       │   ├── renderer.go                     # View renderer — list, form, kanban, calendar, chart, custom (SSR)
│       │   ├── auto_page_generator.go          # GenerateListView/GenerateFormView — auto-generates pages from model fields
│       │   ├── auto_page_generator_test.go     # 6 tests (list generation, form with tabs, auto_pages detection)
│       │   ├── component_compiler.go           # Compiles view JSON into Stencil Web Component HTML. CompileListDatatable() emits bc-datatable with permissions
│       │   └── component_compiler_test.go      # Component compiler tests
│       ├── admin/                              # Split into 7 files for maintainability
│       │   ├── admin.go                        # Core: types, constructor, RegisterRoutes, dashboard, sidebar (with Security section), CSS, helpers
│       │   ├── admin_models.go                 # Model pages: list, view (5 tabs: Form/Fields/Connections/Schema/API), data. Fields tab with MASK/GROUPS columns. API tab with config + generated endpoints preview
│       │   ├── admin_modules.go                # Module pages: list, view (3 tabs: Overview/Permissions/Menu)
│       │   ├── admin_views.go                  # View pages: list, detail (4 tabs: Info/Preview/Editor/Revisions)
│       │   ├── admin_audit.go                  # Health, login history, request log, impersonate/stop-impersonate
│       │   ├── admin_groups.go                 # Group pages: list + detail with 7 Odoo-style tabs (Users/Inherited/Menus/Pages/Access Rights/Record Rules/Notes). Group CRUD API
│       │   ├── admin_security.go               # Security sync page: Load/Export/Upload/Download buttons, history table with rollback. API handlers
│       │   └── admin_api.go                    # View API handlers, data revision handlers
│       ├── assets/
│       │   └── handler.go                      # Static asset serving
│       └── websocket/
│           ├── hub.go                          # WebSocket hub — connect to EventBus, broadcast domain events, route CRUD messages
│           ├── crud.go                         # CRUDHandler — CRUD over WebSocket with request/reply protocol, permission + record rule enforcement
│           └── crud_test.go                    # 7 tests (create, list, read, update, delete, model enable, permission denied)
│
├── pkg/                                        # Public packages (reusable outside engine)
│   ├── ddd/                                    # Domain-Driven Design building blocks
│   │   ├── entity.go                           # Entity interface + BaseEntity
│   │   ├── aggregate.go                        # Aggregate interface + BaseAggregate (with domain events)
│   │   ├── domain_event.go                     # DomainEvent interface + BaseDomainEvent
│   │   ├── repository.go                       # Repository[T] generic interface
│   │   ├── value_object.go                     # ValueObject interface
│   │   └── ddd_test.go                         # 3 tests
│   ├── security/                               # Auth & crypto utilities
│   │   ├── password.go                         # HashPassword (bcrypt), CheckPassword
│   │   ├── jwt.go                              # GenerateToken (with options), ValidateToken, Claims (ImpersonatedBy, Purpose)
│   │   ├── otp.go                              # GenerateOTP — crypto-secure 6-digit code
│   │   ├── encryption.go                       # FieldEncryptor — AES-256-GCM encrypt/decrypt with key versioning
│   │   └── security_test.go                    # 5 tests
│   ├── email/                                  # Email sending
│   │   ├── sender.go                           # SMTPSender — SMTP with TLS, NoopSender fallback
│   │   └── templates.go                        # HTML email templates (OTP code)
│   └── plugin/                                 # Plugin SDK
│       └── proto/
│           └── plugin.proto                    # gRPC service definition for plugins
│
├── embedded/                                   # Compiled-in modules
│   ├── embed.go                                # go:embed directive for modules/
│   ├── embed_test.go                           # Embed tests
│   └── modules/                                # Embedded module files (base, auth)
│       ├── base/                               # Core module (always available)
│       └── auth/                               # Auth module — login, register, forgot, reset, 2FA (i18n x11)
│
├── modules/                                    # Built-in modules (on disk)
│   ├── base/                                   # Core module — users, groups, model_access, security_history, record_rules, settings
│   │   ├── module.json                         # 11 permissions, 2 groups (user, manager), menu
│   │   ├── models/                             # user (is_superuser), role, group (share, comment, module), permission, model_access (12 ERPNext permissions), security_history (ir_security_histories), record_rule (module, modified_source), audit_log, setting
│   │   ├── apis/                               # auth_api, user_api, group_api, role_api, permission_api, etc.
│   │   ├── views/                              # CRUD views for all base models
│   │   ├── data/                               # default_roles, default_groups, default_users
│   │   └── templates/                          # Default UI templates
│   │       ├── layout.html                     # Base layout (full page)
│   │       ├── layout-app.html                 # App layout (sidebar + navbar + content)
│   │       ├── partials/                       # Reusable: sidebar, navbar, pagination, status_badge, actions
│   │       └── views/                          # View templates: list, form, kanban, calendar, chart, login, home
│   ├── crm/                                    # CRM module — contacts, leads
│   │   ├── module.json                         # securities glob, menu with groups
│   │   ├── models/                             # contact (api config, mask on phone), lead (api config)
│   │   ├── securities/                         # crm_user.json (ACL + rules), crm_manager.json (full access)
│   │   ├── apis/                               # contact_api, lead_api (override auto-generated)
│   │   └── views/                              # contact_list, lead_list
│   └── sales/                                  # Sales module — orders
│       ├── module.json
│       ├── models/                             # order, order_line
│       ├── apis/                               # order_api
│       ├── processes/                          # confirm_order
│       ├── views/                              # order_form, order_list
│       └── i18n/                               # Indonesian translations
│
├── plugins/                                    # Plugin runtimes
│   ├── typescript/
│   │   └── index.js                            # Node.js JSON-RPC server (stdin/stdout)
│   └── python/
│       └── runtime.py                          # Python JSON-RPC server (stdin/stdout)
│
├── Dockerfile                                  # Multi-stage build (Go build → minimal runtime)
├── docker-compose.yml                          # Engine + PostgreSQL + Redis
├── Makefile                                    # build, cli, dev, test, lint, clean, tidy
├── go.mod                                      # Go 1.23+, Fiber v2, GORM, Viper, Cobra, JWT, Redis
└── go.sum
```

---

## Packages / Components (`packages/components/`)

Stencil.js Web Component library (`@bitcode/components`).

```
packages/components/
├── package.json                                # @bitcode/components v0.1.0
├── stencil.config.ts                           # Namespace: bc-components, output: dist + www
├── tsconfig.json
│
└── src/
    ├── components.d.ts                         # Auto-generated type declarations
    ├── declarations.d.ts                       # Module declarations
    │
    ├── core/                                   # Shared infrastructure
    │   ├── types.ts                            # FieldType (30+ types), WidgetType, event interfaces, FetchParams/FetchResult, ValidationResult, BcConfig
    │   ├── bc-setup.ts                         # BcSetup singleton — global config (auth, headers, theme, validators, reactivity rules)
    │   ├── bc-native.ts                        # BcNative — bridge abstraction for native capabilities (camera, GPS, SQLite, barcode, biometrics). Detects Tauri vs browser.
    │   ├── bc-native.spec.ts                   # BcNative tests (10 tests — environment detection, browser fallbacks)
    │   ├── offline-store.ts                    # OfflineStore — CRUD routing layer. SQLite for offline models, fetch() for online. Outbox recording.
    │   ├── offline-store.spec.ts               # OfflineStore tests (11 tests — routing, CRUD, outbox, table mapping)
    │   ├── data-fetcher.ts                     # 4-level data fetching (local, URL, event intercept, custom fetcher). Standalone — uses native fetch()
    │   ├── validation-engine.ts                # 3-level validation pipeline (built-in, custom JS, server-side). Uses validators.ts
    │   ├── field-utils.ts                      # Shared field utilities (dirty/touched tracking, ARIA attrs, CSS classes, FormProxy, debounce)
    │   ├── api-client.ts                       # HTTP client for engine REST APIs (BitCode-specific, optional fallback)
    │   ├── event-bus.ts                        # Cross-component event bus
    │   ├── form-engine.ts                      # Form state management, validation, submission (BitCode-specific)
    │   └── i18n.ts                             # Client-side i18n utilities
    │
    ├── global/
    │   ├── global.css                          # Global styles, CSS custom properties, size tokens, auto dark mode
    │   └── themes/
    │       └── dark.css                        # Dark theme overrides (applied via data-bc-theme="dark")
    │
    ├── i18n/                                   # Translation files
    │   ├── index.ts                            # i18n initialization (global script)
    │   ├── en.json                             # English
    │   ├── id.json                             # Indonesian
    │   ├── ar.json                             # Arabic
    │   ├── de.json                             # German
    │   ├── es.json                             # Spanish
    │   ├── fr.json                             # French
    │   ├── ja.json                             # Japanese
    │   ├── ko.json                             # Korean
    │   ├── pt-BR.json                          # Portuguese (Brazil)
    │   ├── ru.json                             # Russian
    │   └── zh-CN.json                          # Chinese (Simplified)
    │
    ├── utils/                                  # Shared utilities
    │   ├── expression-eval.ts                  # Expression evaluator for computed fields
    │   ├── format.ts                           # Number, date, currency formatting
    │   └── validators.ts                       # Field validation rules
    │
    └── components/                             # Web Components (each has .tsx + .css)
        ├── bc-placeholder/                     # Placeholder component
        │
        ├── fields/                             # 30+ field components
        │   ├── bc-field-string/                # Text input
        │   ├── bc-field-smalltext/             # Small textarea
        │   ├── bc-field-text/                  # Large textarea
        │   ├── bc-field-richtext/              # Tiptap rich text editor
        │   ├── bc-field-markdown/              # Markdown editor with preview
        │   ├── bc-field-html/                  # HTML editor
        │   ├── bc-field-code/                  # CodeMirror code editor
        │   ├── bc-field-password/              # Password input
        │   ├── bc-field-integer/               # Integer input
        │   ├── bc-field-float/                 # Float input
        │   ├── bc-field-decimal/               # Decimal input
        │   ├── bc-field-currency/              # Currency input with formatting
        │   ├── bc-field-percent/               # Percentage input
        │   ├── bc-field-checkbox/              # Checkbox
        │   ├── bc-field-toggle/                # Toggle switch
        │   ├── bc-field-select/                # Dropdown select
        │   ├── bc-field-radio/                 # Radio buttons
        │   ├── bc-field-multicheck/            # Multi-checkbox
        │   ├── bc-field-tags/                  # Tag input
        │   ├── bc-field-date/                  # Date picker
        │   ├── bc-field-time/                  # Time picker
        │   ├── bc-field-datetime/              # DateTime picker
        │   ├── bc-field-duration/              # Duration input
        │   ├── bc-field-file/                  # File upload
        │   ├── bc-field-image/                 # Image upload with preview
        │   ├── bc-field-signature/             # Signature pad (signature_pad)
        │   ├── bc-field-barcode/               # Barcode/QR generator (JsBarcode + QRCode)
        │   ├── bc-field-color/                 # Color picker
        │   ├── bc-field-geo/                   # Geolocation (Leaflet map)
        │   ├── bc-field-rating/                # Star rating
        │   ├── bc-field-json/                  # JSON editor
        │   ├── bc-field-link/                  # Many2one link field
        │   ├── bc-field-dynlink/               # Dynamic link field
        │   ├── bc-field-tableselect/           # Table multi-select
        │   └── field-base.css                  # Shared field styles
        │
        ├── layout/                             # Layout components
        │   ├── bc-row/                         # Flex row
        │   ├── bc-column/                      # Flex column
        │   ├── bc-section/                     # Collapsible section
        │   ├── bc-tabs/ + bc-tab/              # Tab container + tab panels
        │   ├── bc-sheet/                       # Card/sheet container
        │   ├── bc-header/                      # Section header
        │   ├── bc-separator/                   # Visual separator
        │   ├── bc-button-box/                  # Button group
        │   └── bc-html-block/                  # Raw HTML block
        │
        ├── views/                              # View components
        │   ├── bc-view-list/                   # Data table view
        │   ├── bc-view-form/                   # Record form view
        │   ├── bc-view-kanban/                 # Kanban board
        │   ├── bc-view-calendar/               # Calendar (FullCalendar)
        │   ├── bc-view-gantt/                  # Gantt chart (frappe-gantt)
        │   ├── bc-view-map/                    # Map view (Leaflet)
        │   ├── bc-view-tree/                   # Tree/hierarchy view
        │   ├── bc-view-report/                 # Report view
        │   └── bc-view-activity/               # Activity stream view
        │
        ├── charts/                             # Chart components (ECharts)
        │   ├── bc-chart-line/                  # Line chart
        │   ├── bc-chart-bar/                   # Bar chart
        │   ├── bc-chart-pie/                   # Pie/donut chart
        │   ├── bc-chart-area/                  # Area chart
        │   ├── bc-chart-gauge/                 # Gauge chart
        │   ├── bc-chart-funnel/                # Funnel chart
        │   ├── bc-chart-heatmap/               # Heatmap
        │   ├── bc-chart-pivot/                 # Pivot table
        │   ├── bc-chart-kpi/                   # KPI card
        │   ├── bc-chart-scorecard/             # Scorecard
        │   └── bc-chart-progress/              # Progress indicator
        │
        ├── datatable/                          # Data table components
        │   ├── bc-datatable/                   # Full-featured data table
        │   ├── bc-filter-builder/              # Advanced filter builder
        │   └── bc-lookup-modal/                # Record lookup modal
        │
        ├── dialogs/                            # Dialog components
        │   ├── bc-dialog-modal/                # Generic modal
        │   ├── bc-dialog-confirm/              # Confirmation dialog
        │   ├── bc-dialog-quickentry/           # Quick record entry
        │   ├── bc-dialog-wizard/               # Multi-step wizard
        │   └── bc-toast/                       # Toast notifications
        │
        ├── search/                             # Search & filter components
        │   ├── bc-search/                      # Search bar
        │   ├── bc-filter-bar/                  # Filter bar
        │   ├── bc-filter-panel/                # Filter panel
        │   └── bc-favorites/                   # Saved filters
        │
        ├── widgets/                            # Widget components
        │   ├── bc-widget-badge/                # Status badge
        │   ├── bc-widget-copy/                 # Copy to clipboard
        │   ├── bc-widget-phone/                # Phone link
        │   ├── bc-widget-email/                # Email link
        │   ├── bc-widget-url/                  # URL link
        │   ├── bc-widget-progress/             # Progress bar
        │   ├── bc-widget-statusbar/            # Status bar (workflow states)
        │   ├── bc-widget-priority/             # Priority indicator
        │   ├── bc-widget-handle/               # Drag handle
        │   ├── bc-widget-domain/               # Domain filter widget
        │   ├── bc-viewer-pdf/                  # PDF viewer (iframe + object fallback)
        │   ├── bc-viewer-image/                # Image viewer (zoom, lightbox)
        │   ├── bc-viewer-document/             # Office doc viewer (Microsoft/Google iframe)
        │   ├── bc-viewer-youtube/              # YouTube embed (full video + shorts)
        │   ├── bc-viewer-instagram/            # Instagram post embed (oEmbed)
        │   ├── bc-viewer-tiktok/               # TikTok video embed (oEmbed)
        │   ├── bc-viewer-video/                # HTML5 video player (mp4, webm, ogg)
        │   └── bc-viewer-audio/                # HTML5 audio player (mp3, m4a, aac, ogg, webm)
        │
        ├── table/                              # Table components
        │   └── bc-child-table/                 # Inline child table (one2many)
        │
        ├── print/                              # Print & export
        │   ├── bc-export/                      # Data export (XLSX)
        │   ├── bc-print/                       # Print view
        │   └── bc-report-link/                 # Report link
        │
        └── social/                             # Social/collaboration
            ├── bc-activity/                    # Activity log
            ├── bc-chatter/                     # Comment thread
            └── bc-timeline/                    # Timeline view
```

---

## Tauri Native Shell (`packages/tauri/`)

Tauri 2.0 project that wraps Stencil components in a native WebView for desktop and mobile.

```
packages/tauri/
├── package.json                                # @bitcode/tauri — npm scripts for dev/build (desktop, android, ios)
├── .gitignore                                  # Excludes target/, gen/, node_modules/
│
└── src-tauri/
    ├── Cargo.toml                              # Tauri 2.10 + plugins (sql, fs, notification, barcode, biometric)
    ├── build.rs                                # Tauri build script
    ├── tauri.conf.json                         # frontendDist → ../../components/www, withGlobalTauri, CSP, window config
    ├── capabilities/
    │   └── default.json                        # Permissions: core, sql, fs, notification
    ├── icons/                                  # App icons for all platforms (generated via cargo tauri icon)
    └── src/
        └── main.rs                             # Entry point — plugin registration, SQLite migrations for _off_* tables
```

---

## go-json Language Engine (`packages/go-json/`)

Standalone JSON/JSONC programming language engine. Embeddable in Go applications. Powered by expr-lang/expr for expression evaluation.

**Design doc:** `docs/plans/2026-07-14-runtime-engine-phase-4.5a-go-json-core-language.md`
**Implementation plan:** `docs/plans/2026-07-14-runtime-engine-phase-4.5a-go-json-core-language-plan.md`

```
packages/go-json/
├── go.mod                                      # module github.com/bitcode-framework/go-json (Go 1.24+, expr-lang/expr)
├── go.sum
│
├── lang/                                       # Core language engine
│   ├── ast.go                                  # AST node types — 15 step types (let, set, if, switch, for, while, break, continue, return, call, try, error, log, comment) + Program, FuncDef
│   ├── parser.go                               # JSONC/JSON → AST. Handles all step types, overloaded nodes (return/error/log), ordered function params
│   ├── compiler.go                             # AST → CompiledProgram. Structural validation (break/continue outside loop), limit resolution
│   ├── vm.go                                   # Tree-walk interpreter. All step types, scope isolation, resource limits, debug hooks, trace
│   ├── scope.go                                # Variable scoping — block scope (if/for/while), chain lookup, isolated child (functions), ToMap() for expr-lang
│   ├── types.go                                # Gradual type system — InferType (JSON float64→int detection), TypesCompatible, nullable (?T)
│   ├── errors.go                               # GoJSONError with enrichment, levenshtein "did you mean?", JSON/Short/Error output, fluent builder
│   ├── expr_engine.go                          # ExprEngine interface + ExprLangEngine (expr-lang/expr). Compiled expression cache, error enrichment
│   ├── debugger.go                             # Debugger interface (OnStep/OnVariable/OnError/OnFunctionCall/OnFunctionReturn) + ExecutionTrace
│   ├── program.go                              # Immutable CompiledProgram, CompiledFunc, ParamDef, ResolvedLimits, DefaultLimits(), HardLimits()
│   ├── preprocess.go                           # JSONC → JSON: strip // and /* */ comments, trailing commas. String-aware (won't strip inside strings)
│   ├── preprocess_test.go                      # 18 tests — comments, strings, escaped quotes, trailing commas, edge cases
│   ├── integration_test.go                     # 35 tests — full pipeline (parse → compile → execute) for all features
│   └── edge_cases_test.go                      # 22 tests — nil, types, overflow, unicode, malformed JSON, error formatting
│
├── stdlib/                                     # Layer 2 stdlib (Layer 1 = expr-lang built-ins, ~68 functions)
│   ├── registry.go                             # Function registry + DefaultRegistry() with all Layer 2 functions
│   ├── math.go                                 # 7 functions: clamp, sign, randomInt, randomFloat, pow, sqrt, mod
│   ├── strings.go                              # 5 functions: padLeft, padRight, substring, format, matches (regex)
│   ├── arrays.go                               # 5 functions: append, prepend, slice, chunk, zip
│   └── types.go                                # 2 functions: bool (truthy conversion), isNil
│
├── runtime/                                    # Runtime API
│   ├── runtime.go                              # NewRuntime(opts...), Compile(), Execute(), ExecuteJSON(). Program cache (SHA256 keyed). Concurrent-safe
│   ├── limits.go                               # Limits struct, DefaultLimits(), HardLimits(), ToResolved()
│   ├── context.go                              # Session (UserID, Locale, TenantID, Groups) + ExecutionMeta (ID, Program, StartedAt)
│   └── logger.go                               # Logger interface (Log(level, message, data)) + DefaultLogger (stdout)
│
├── cmd/go-json/
│   └── main.go                                 # CLI placeholder
│
├── codegen/                                    # Reserved for Phase 4.5b (struct codegen)
├── io/                                         # Reserved for Phase 4.5c (I/O modules)
│
└── testdata/                                   # Test fixture programs
    ├── hello.json                              # Minimal program
    ├── hello.jsonc                              # Same with JSONC comments
    ├── variables.json                           # let, set, value/expr/with modes
    ├── control_flow.json                        # if/elif/else, switch
    ├── loops.json                               # for, while, range, break, continue
    ├── functions.json                           # Function definition, call, recursion (factorial)
    └── error_handling.json                      # try/catch/finally, error throw
```

**75 tests** across 4 packages. `go build ./...` + `go vet ./...` clean.

---

## Samples (`samples/`)

### ERP Sample (`samples/erp/`)

Full ERP application demonstrating all engine features.

```
samples/erp/
├── bitcode.yaml                                # Project config (port 8989, SQLite)
├── bitcode.toml                                # Alternative TOML config
├── .env.example                                # Environment variable template
├── README.md                                   # Comprehensive feature documentation
├── run.bat / run.ps1 / run.sh                  # Cross-platform run scripts (go install + bitcode dev)
│
└── modules/
    ├── base/                                   # Core module (users, roles, groups, permissions)
    │   ├── module.json
    │   ├── models/                             # 7 models: user, role, group, permission, record_rule, audit_log, setting
    │   ├── apis/                               # 8 APIs: auth, user, group, role, permission, record_rule, setting, audit_log
    │   ├── views/                              # CRUD views for all models
    │   ├── data/                               # Default roles, groups, users
    │   └── templates/                          # Layout, partials, view templates
    │
    ├── crm/                                    # CRM Module (depends: base)
    │   ├── module.json                         # 9 permissions, 2 groups (user, manager), menu, settings
    │   ├── models/                             # contact, lead, activity, tag, vip_contact (inherits contact)
    │   ├── apis/                               # contact_api, lead_api, activity_api, tag_api
    │   ├── processes/                          # lead_workflow, qualify_lead, convert_lead, win_lead, lose_lead,
    │   │                                       #   log_activity, enrich_leads (DAG), enrich_lead_dag
    │   ├── agents/                             # lead_agent (3 event triggers + 2 cron jobs)
    │   ├── views/                              # list, form, kanban, dashboard, pipeline_chart
    │   ├── scripts/                            # TS: on_deal_won, on_deal_lost, notify_manager, weekly_report, stale_leads
    │   │                                       # Python: analyze_pipeline.py
    │   ├── templates/                          # dashboard.html, partials/lead_card.html
    │   ├── data/demo.json                      # Demo contacts, leads, tags
    │   └── i18n/id.json                        # Indonesian translations
    │
    └── hrm/                                    # HRM Module (depends: base)
        ├── module.json                         # 13 permissions, 3 groups (user, officer, manager), menu
        ├── models/                             # department, job_position, employee, leave_request
        ├── apis/                               # department_api, position_api, employee_api, leave_api
        ├── processes/                          # employee_workflow, leave_workflow, submit_leave, approve_leave,
        │                                       #   reject_leave, promote_employee, onboard_employee
        ├── agents/                             # hr_agent (3 event triggers + 1 cron)
        ├── views/                              # list, form, calendar, dashboard
        ├── scripts/                            # TS: on_leave_approved, on_leave_submitted, on_promotion, weekly_attendance
        │                                       # Python: calculate_leave_balance.py, generate_onboard_checklist.py
        ├── templates/                          # dashboard.html, partials/employee_card.html
        ├── data/demo.json                      # Demo departments, employees, positions
        └── i18n/id.json                        # Indonesian translations
```

---

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
| `infrastructure/module` | 7+ | Registry, dependency resolution, circular detection, FS |
| `infrastructure/i18n` | 4 | Translate, fallback, locale detection |
| `presentation/template` | 5 | Load/render, helpers (truncate, dict, eq) |
| `presentation/view` | — | Component compiler tests |
| `runtime/executor` | 3+ | Step dispatch, unknown type, step error, DAG |
| `runtime/executor/steps` | 9 | Validate (eq/fail/required), emit, assign, if, process parse |
| `runtime/validation` | 28 | Field validators (required, email, regex, conditional, etc.), sanitizers, model-level validators, update merge |
| `runtime/hook` | 10 | Event dispatcher (priority, condition, sync/async, retry, on_change cascade), context, expression eval |
| `embedded` | — | Embed FS tests |
| **Total** | **131+** | |

Run all tests: `cd engine && go test ./... -v`

---

## Key Interfaces

```go
// Cache — pluggable caching (memory or redis)
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

// ScriptRunner — plugin execution (TS/Python)
type ScriptRunner interface {
    Run(ctx context.Context, script string, params map[string]any) (any, error)
}

// PermissionChecker — RBAC authorization
type PermissionChecker interface {
    UserHasPermission(userID string, permission string) (bool, error)
}

// RecordRuleEngine — row-level security
type RecordRuleEngine interface {
    GetFilters(userID string, modelName string, operation string) ([][]any, error)
}

// Repository[T] — generic data access (DDD)
type Repository[T any] interface {
    Save(ctx context.Context, entity *T) error
    FindByID(ctx context.Context, id string) (*T, error)
    FindAll(ctx context.Context, filters map[string]interface{}, page int, pageSize int) ([]T, int64, error)
    Update(ctx context.Context, entity *T) error
    Delete(ctx context.Context, id string) error
}

// ModuleFS — filesystem abstraction for module loading
type ModuleFS interface {
    ReadFile(name string) ([]byte, error)
    ReadDir(name string) ([]os.DirEntry, error)
    Glob(pattern string) ([]string, error)
}
```

---

## Build & Run

```bash
# Engine
cd engine
make build          # Build bitcode binary → bin/bitcode
make install        # Install bitcode to $GOPATH/bin
make serve          # Start production server
make dev            # Start dev server with hot reload
make test           # Run all tests

# Components
cd packages/components
npm install
npm run build       # Build Stencil components → dist/
npm run start       # Dev server with watch

# Sample ERP
cd samples/erp
./run.sh            # Linux/macOS
.\run.bat           # Windows CMD
.\run.ps1           # Windows PowerShell
```

---

## Dependencies

### Engine (Go)

| Dependency | Purpose |
|------------|---------|
| `gofiber/fiber/v2` | HTTP framework |
| `gorm.io/gorm` | ORM (SQLite, Postgres, MySQL) |
| `glebarez/sqlite` | Pure-Go SQLite driver |
| `gorm.io/driver/postgres` | PostgreSQL driver |
| `gorm.io/driver/mysql` | MySQL driver |
| `spf13/viper` | Configuration (env + file) |
| `spf13/cobra` | CLI framework |
| `golang-jwt/jwt/v5` | JWT tokens |
| `google/uuid` | UUID generation |
| `redis/go-redis/v9` | Redis client |
| `golang.org/x/crypto` | bcrypt password hashing |
| `gofiber/contrib/websocket` | WebSocket support |
| `fsnotify/fsnotify` | File watching (dev mode) |

### Components (TypeScript)

| Dependency | Purpose |
|------------|---------|
| `@stencil/core` | Web Component compiler |
| `echarts` | Charts |
| `@tiptap/*` | Rich text editor |
| `@codemirror/*` | Code editor |
| `@fullcalendar/*` | Calendar view |
| `frappe-gantt` | Gantt chart |
| `leaflet` | Maps |
| `sortablejs` | Drag & drop |
| `signature_pad` | Signature capture |
| `jsbarcode` + `qrcode` | Barcode/QR generation |
| `markdown-it` | Markdown rendering |
| `xlsx` | Excel export |
