# Table Prefix & Postgres Schema — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add per-module table prefix, per-model table override, remove naive pluralization, and add Postgres schema support.

**Architecture:** Centralized `ResolveTableName()` in model registry replaces 18+ hardcoded `model + "s"` sites. Module prefix stored per-model at registration time. Postgres schema via `search_path` at connection.

**Tech Stack:** Go 1.23+, GORM, Viper config, existing parser/module/persistence layers.

---

### Task 1: Add TableConfig to ModuleDefinition parser

**Files:**
- Modify: `engine/internal/compiler/parser/module.go`

**Step 1: Add TableConfig struct and Table field to ModuleDefinition**

Add after `IncludeMenuDefinition` struct:

```go
type TableConfig struct {
	Prefix string `json:"prefix"`
}
```

Add to `ModuleDefinition` struct:

```go
Table *TableConfig `json:"table,omitempty"`
```

**Step 2: Run tests**

Run: `cd engine && go test ./internal/compiler/parser/ -v`
Expected: PASS (no behavior change, just new optional field)

**Step 3: Commit**

```
feat(parser): add TableConfig to ModuleDefinition for table prefix support
```

---

### Task 2: Add Table field to ModelDefinition parser

**Files:**
- Modify: `engine/internal/compiler/parser/model.go`

**Step 1: Add ModelTableConfig and Table field**

The model `"table"` field can be either a string (direct table name) or an object (with prefix). Use `json.RawMessage` for flexible parsing.

Add to `ModelDefinition` struct:

```go
TableRaw json.RawMessage `json:"table,omitempty"`
TableName string          `json:"-"` // resolved: direct table name override
TablePrefix *string       `json:"-"` // resolved: prefix override (nil = use module, "" = no prefix)
```

Add parsing logic in `ParseModel()` after validation, before return:

```go
if len(model.TableRaw) > 0 {
    // Try as string first
    var tableName string
    if err := json.Unmarshal(model.TableRaw, &tableName); err == nil {
        model.TableName = tableName
    } else {
        // Try as object
        var tableConfig struct {
            Prefix string `json:"prefix"`
        }
        if err := json.Unmarshal(model.TableRaw, &tableConfig); err == nil {
            prefix := tableConfig.Prefix
            model.TablePrefix = &prefix
        }
    }
}
```

**Step 2: Run tests**

Run: `cd engine && go test ./internal/compiler/parser/ -v`
Expected: PASS

**Step 3: Commit**

```
feat(parser): add table name/prefix override to ModelDefinition
```

---

### Task 3: Refactor model Registry with centralized ResolveTableName

**Files:**
- Modify: `engine/internal/domain/model/registry.go`

**Step 1: Add prefix storage and ResolveTableName**

Add to Registry struct:

```go
type Registry struct {
    models       map[string]*parser.ModelDefinition
    tableNames   map[string]string // modelName → resolved table name
    mu           sync.RWMutex
}
```

Update `NewRegistry`:

```go
func NewRegistry() *Registry {
    return &Registry{
        models:     make(map[string]*parser.ModelDefinition),
        tableNames: make(map[string]string),
    }
}
```

Add new method `RegisterWithModule`:

```go
func (r *Registry) RegisterWithModule(model *parser.ModelDefinition, moduleDef *parser.ModuleDefinition) error {
    if err := r.Register(model); err != nil {
        return err
    }
    r.mu.Lock()
    defer r.mu.Unlock()
    r.tableNames[model.Name] = resolveTableName(model, moduleDef)
    return nil
}

func resolveTableName(model *parser.ModelDefinition, moduleDef *parser.ModuleDefinition) string {
    // 1. Model has direct table name override
    if model.TableName != "" {
        return model.TableName
    }
    // 2. Model has prefix override
    if model.TablePrefix != nil {
        prefix := *model.TablePrefix
        if prefix == "" {
            return model.Name
        }
        return prefix + "_" + model.Name
    }
    // 3. Module has table prefix
    if moduleDef != nil && moduleDef.Table != nil && moduleDef.Table.Prefix != "" {
        return moduleDef.Table.Prefix + "_" + model.Name
    }
    // 4. Default: model name as-is
    return model.Name
}
```

Refactor existing `TableName` method:

```go
func (r *Registry) TableName(modelName string) string {
    r.mu.RLock()
    defer r.mu.RUnlock()
    if tn, ok := r.tableNames[modelName]; ok {
        return tn
    }
    // Fallback for models registered without module context
    return modelName
}
```

**Step 2: Add unit tests**

Create `engine/internal/domain/model/registry_test.go`:

Test cases:
- Model with no prefix, no module prefix → model name
- Model with module prefix → prefix_modelname
- Model with direct table name override → that name
- Model with prefix override → override_prefix_modelname
- Model with empty prefix override (clear module prefix) → model name
- TableName fallback for unknown model → model name

**Step 3: Run tests**

Run: `cd engine && go test ./internal/domain/model/ -v`
Expected: PASS

**Step 4: Commit**

```
feat(registry): centralized ResolveTableName with module prefix support
```

---

### Task 4: Update app.go installModule to use RegisterWithModule

**Files:**
- Modify: `engine/internal/app.go`

**Step 1: Pass module definition to RegisterWithModule**

In `installModule()`, change model registration loop (around line 1286-1299):

```go
for _, m := range loaded.Models {
    if m.Inherit != "" {
        parent, err := a.ModelRegistry.Get(m.Inherit)
        if err == nil {
            m = persistence.MergeInheritedFields(parent, m)
        }
    }

    if err := a.ModelRegistry.RegisterWithModule(m, loaded.Definition); err != nil {
        return fmt.Errorf("failed to register model %s: %w", m.Name, err)
    }
    if err := persistence.MigrateModel(a.DB, m, a.ModelRegistry); err != nil {
        return fmt.Errorf("failed to migrate model %s: %w", m.Name, err)
    }
}
```

Also update form POST handler (line 973-978) to use registry:

```go
tableName := a.ModelRegistry.TableName(entry.Def.Model)
if modelDef != nil {
    repo = persistence.NewGenericRepositoryWithModel(a.DB, tableName, modelDef)
} else {
    repo = persistence.NewGenericRepository(a.DB, tableName)
}
```

And login handler (line 540):

```go
repo := persistence.NewGenericRepository(a.DB, a.ModelRegistry.TableName("user"))
```

And register handler (line 617), forgot handler (line 657), reset handler (line 721).

**Step 2: Run build**

Run: `cd engine && go build ./...`
Expected: compile success

**Step 3: Commit**

```
refactor(app): use RegisterWithModule and TableName resolver
```

---

### Task 5: Update MigrateModel to accept registry for table name

**Files:**
- Modify: `engine/internal/infrastructure/persistence/dynamic_model.go`

**Step 1: Add TableNameResolver interface and update MigrateModel**

```go
type TableNameResolver interface {
    TableName(modelName string) string
}

func MigrateModel(db *gorm.DB, model *parser.ModelDefinition, resolver TableNameResolver) error {
    dialect := DetectDialect(db)
    columns := buildColumns(model, dialect)
    tableName := resolver.TableName(model.Name)
    // ... rest unchanged but uses tableName variable
}
```

Update `createJunctionTable` to also accept resolver:

```go
func createJunctionTable(db *gorm.DB, model1 string, fieldName string, model2 string, dialect DBDialect, resolver TableNameResolver) error {
    tableName := resolver.TableName(model1) + "_" + fieldName
    // ... rest uses tableName
}
```

Update the many2many loop in MigrateModel to pass resolver.

**Step 2: Run tests**

Run: `cd engine && go test ./internal/infrastructure/persistence/ -v`
Expected: PASS (update test calls to pass resolver)

**Step 3: Commit**

```
refactor(migration): use TableNameResolver for table name derivation
```

---

### Task 6: Update API router to use registry TableName

**Files:**
- Modify: `engine/internal/presentation/api/router.go`

**Step 1: Add TableNameResolver to Router**

Add field and setter:

```go
type Router struct {
    // ... existing fields
    tableNameResolver interface{ TableName(string) string }
}

func (r *Router) SetTableNameResolver(resolver interface{ TableName(string) string }) {
    r.tableNameResolver = resolver
}
```

Update `RegisterAPI` (line 56-58):

```go
tableName := apiDef.Model
if r.tableNameResolver != nil {
    tableName = r.tableNameResolver.TableName(apiDef.Model)
} else {
    tableName = apiDef.Model
}
// use tableName instead of apiDef.Model+"s"
```

**Step 2: Update app.go to set resolver on router**

In `installModule()`:

```go
router.SetTableNameResolver(a.ModelRegistry)
```

**Step 3: Update GetBasePath in api.go**

Change line 46:

```go
return "/api/" + a.Model
```

(Remove `+ "s"`)

**Step 4: Run build**

Run: `cd engine && go build ./...`
Expected: compile success

**Step 5: Commit**

```
refactor(api): use TableNameResolver, remove pluralization from API paths
```

---

### Task 7: Update view renderer to use registry TableName

**Files:**
- Modify: `engine/internal/presentation/view/renderer.go`

**Step 1: Add TableNameResolver to Renderer**

Add field:

```go
type Renderer struct {
    // ... existing fields
    tableNameResolver interface{ TableName(string) string }
}
```

Add setter:

```go
func (r *Renderer) SetTableNameResolver(resolver interface{ TableName(string) string }) {
    r.tableNameResolver = resolver
}
```

Add helper:

```go
func (r *Renderer) resolveTable(modelName string) string {
    if r.tableNameResolver != nil {
        return r.tableNameResolver.TableName(modelName)
    }
    return modelName
}
```

**Step 2: Replace all 7 occurrences of `Model+"s"`**

Replace every `viewDef.Model+"s"` and `ds.Model+"s"` with `r.resolveTable(viewDef.Model)` / `r.resolveTable(ds.Model)`.

Lines: 74, 167, 218, 264, 314, 343, 369.

**Step 3: Wire in app.go**

After creating ViewRenderer:

```go
app.ViewRenderer.SetTableNameResolver(modelReg)
```

(This must be set after modelReg is created but before modules are loaded.)

**Step 4: Run build**

Run: `cd engine && go build ./...`
Expected: compile success

**Step 5: Commit**

```
refactor(view): use TableNameResolver in all view renderers
```

---

### Task 8: Update process data handler and hydrator

**Files:**
- Modify: `engine/internal/runtime/executor/steps/data.go`
- Modify: `engine/internal/runtime/expression/hydrator.go`

**Step 1: Update DataHandler**

Add resolver field:

```go
type DataHandler struct {
    DB       *gorm.DB
    Resolver interface{ TableName(string) string }
}
```

Update Execute (line 22):

```go
tableName := step.Model
if h.Resolver != nil {
    tableName = h.Resolver.TableName(step.Model)
}
repo := persistence.NewGenericRepository(h.DB, tableName)
```

**Step 2: Update Hydrator**

Add resolver field:

```go
type Hydrator struct {
    db                *gorm.DB
    modelRegistry     ModelLookup
    tableNameResolver interface{ TableName(string) string }
}
```

Add setter and update constructor:

```go
func (h *Hydrator) SetTableNameResolver(resolver interface{ TableName(string) string }) {
    h.tableNameResolver = resolver
}
```

Update `loadChildren` (line 97):

```go
tableName := childModel
if h.tableNameResolver != nil {
    tableName = h.tableNameResolver.TableName(childModel)
}
```

**Step 3: Wire in app.go**

Update `registerStepHandlers`:

```go
a.Executor.RegisterHandler(parser.StepQuery, &steps.DataHandler{DB: a.DB, Resolver: a.ModelRegistry})
a.Executor.RegisterHandler(parser.StepCreate, &steps.DataHandler{DB: a.DB, Resolver: a.ModelRegistry})
a.Executor.RegisterHandler(parser.StepUpdate, &steps.DataHandler{DB: a.DB, Resolver: a.ModelRegistry})
a.Executor.RegisterHandler(parser.StepDelete, &steps.DataHandler{DB: a.DB, Resolver: a.ModelRegistry})
```

And hydrator:

```go
hydrator.SetTableNameResolver(modelReg)
```

**Step 4: Run build**

Run: `cd engine && go build ./...`
Expected: compile success

**Step 5: Commit**

```
refactor(runtime): use TableNameResolver in data handler and hydrator
```

---

### Task 9: Update admin panel

**Files:**
- Modify: `engine/internal/presentation/admin/admin.go`
- Modify: `engine/internal/presentation/admin/admin_api.go`

**Step 1: Add resolver to AdminPanel**

```go
type AdminPanel struct {
    // ... existing fields
    tableNameResolver interface{ TableName(string) string }
}
```

Update constructor to accept it. Update `NewAdminPanel` signature.

**Step 2: Patch admin.go**

Line 532: `name+"s"` → `a.tableNameResolver.TableName(name)` (or fallback to `name` if nil)
Line 1534, 1609: `"users"` → `a.tableNameResolver.TableName("user")`

**Step 3: Patch admin_api.go**

Line 272: `modelName + "s"` → resolver
Line 279: same table name variable

**Step 4: Wire in app.go**

Pass `a.ModelRegistry` to `NewAdminPanel`.

**Step 5: Run build**

Run: `cd engine && go build ./...`
Expected: compile success

**Step 6: Commit**

```
refactor(admin): use TableNameResolver for table name resolution
```

---

### Task 10: Update CLI (cmd/bitcode/main.go)

**Files:**
- Modify: `engine/cmd/bitcode/main.go`

**Step 1: Patch hardcoded "users" table reference**

Line 335: This is in CLI context where registry may not be available. Use a simple approach — accept that CLI commands use direct table names. For now, keep `"users"` but add a TODO comment. Or better: load base module config to get prefix.

Actually, the CLI `user list` and `user create` commands need to know the base module prefix. The simplest approach: add a helper function that reads base module.json to get prefix, or accept a `--table-prefix` flag.

Simplest: hardcode `res_user` since base module always uses `res` prefix. But this couples CLI to module config.

Better: Load module config in CLI too. The CLI already loads DB config. Add module config loading for table name resolution.

**Step 2: Run build**

Run: `cd engine && go build ./cmd/bitcode/`
Expected: compile success

**Step 3: Commit**

```
refactor(cli): update table references for prefix support
```

---

### Task 11: Update seeder to use resolver

**Files:**
- Modify: `engine/internal/infrastructure/module/seeder.go`

**Step 1: Update SeedModule signature**

```go
func SeedModule(db *gorm.DB, modulePath string, dataPatterns []string, resolver interface{ TableName(string) string }) error
```

**Step 2: Update seedFile to accept resolver**

The seeder currently derives table names from JSON keys or filenames. Update to use resolver:

- For keyed format (`{"users": [...]}`) → key is model name, resolve via `resolver.TableName(key)`
- For array format (filename-based) → derive model name from filename, resolve via resolver

Remove the `+ "s"` logic and the `if table[len(table)-1] != 's'` check.

**Step 3: Update app.go call site**

```go
if err := module.SeedModule(a.DB, modPath, loaded.Definition.Data, a.ModelRegistry); err != nil {
```

**Step 4: Run build**

Run: `cd engine && go build ./...`
Expected: compile success

**Step 5: Commit**

```
refactor(seeder): use TableNameResolver for table name derivation
```

---

### Task 12: Update all module.json files with table prefix

**Files:**
- Modify: `engine/embedded/modules/base/module.json` — add `"table": {"prefix": "res"}`
- Modify: `engine/modules/crm/module.json` — add `"table": {"prefix": "crm"}`
- Modify: `engine/modules/sales/module.json` — add `"table": {"prefix": "sale"}`
- Modify: `samples/erp/modules/crm/module.json` — add `"table": {"prefix": "crm"}`
- Modify: `samples/erp/modules/hrm/module.json` — add `"table": {"prefix": "hr"}`

Note: `engine/embedded/modules/auth/module.json` has no models, no change needed.

**Step 1: Add table config to each module.json**

**Step 2: Commit**

```
feat(modules): add table prefix config to all module.json files
```

---

### Task 13: Update seed data JSON files

**Files:**
- Modify: `engine/embedded/modules/base/data/default_users.json` — key `"users"` → `"user"` (model name, resolver adds prefix)
- Modify: `engine/embedded/modules/base/data/default_roles.json` — similar
- Modify: `engine/embedded/modules/base/data/default_groups.json` — similar
- Modify: `samples/erp/modules/crm/data/demo.json` — keys `"contacts"` → `"contact"`, `"leads"` → `"lead"`, `"tags"` → `"tag"`
- Modify: `samples/erp/modules/hrm/data/demo.json` — update keys similarly

**Step 1: Update JSON keys to model names (singular)**

**Step 2: Commit**

```
feat(data): update seed data keys to match model names (no pluralization)
```

---

### Task 14: Add Postgres schema support

**Files:**
- Modify: `engine/internal/infrastructure/persistence/database.go` — add Schema to DatabaseConfig, set search_path
- Modify: `engine/internal/config.go` — add database.schema binding

**Step 1: Add Schema field to DatabaseConfig**

```go
type DatabaseConfig struct {
    Driver     string
    Host       string
    Port       int
    User       string
    Password   string
    DBName     string
    SSLMode    string
    SQLitePath string
    Schema     string // Postgres only, default "public"
}
```

**Step 2: Update openPostgres to set search_path**

```go
func openPostgres(cfg DatabaseConfig, gormCfg *gorm.Config) (*gorm.DB, error) {
    // ... existing DSN building ...

    schema := cfg.Schema
    if schema == "" {
        schema = "public"
    }

    if schema != "" && schema != "public" {
        dsn += fmt.Sprintf(" search_path=%s", schema)
    }

    db, err := gorm.Open(postgres.Open(dsn), gormCfg)
    if err != nil {
        return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
    }

    // Create schema if not exists and set search_path
    if schema != "public" {
        db.Exec(fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", schema))
        db.Exec(fmt.Sprintf("SET search_path TO %s", schema))
    }

    return db, nil
}
```

**Step 3: Update config.go**

Add default and binding:

```go
v.SetDefault("database.schema", "public")
v.BindEnv("database.schema", "DB_SCHEMA")
```

Add to config struct population:

```go
DB: persistence.DatabaseConfig{
    // ... existing fields ...
    Schema: v.GetString("database.schema"),
},
```

**Step 4: Run build**

Run: `cd engine && go build ./...`
Expected: compile success

**Step 5: Commit**

```
feat(postgres): add schema support via DB_SCHEMA config
```

---

### Task 15: Run full test suite and fix failures

**Step 1: Run all tests**

Run: `cd engine && go test ./... -v -count=1`

**Step 2: Fix any test failures**

Most likely failures:
- Tests that create repos with `model + "s"` table names
- Tests that reference old table names
- MigrateModel tests that need resolver parameter

Fix each failure.

**Step 3: Commit**

```
fix(tests): update tests for table prefix and no-pluralization changes
```

---

### Task 16: Update documentation

**Files:**
- Modify: `docs/architecture.md` — mention table prefix system
- Modify: `docs/codebase.md` — update if new files added
- Modify: `docs/features.md` — mark table prefix and schema as completed
- Modify: `engine/docs/features/modules.md` — document table config in module.json
- Modify: `engine/docs/features/configuration.md` — document DB_SCHEMA
- Modify: `README.md` — add DB_SCHEMA to config table
- Modify: `samples/erp/README.md` — update API paths (no 's')
- Modify: `AGENTS.md` — update completed items

**Step 1: Update all docs**

**Step 2: Commit**

```
docs: update documentation for table prefix and schema support
```

---

### Task 17: Final verification and commit

**Step 1: Run full build**

Run: `cd engine && go build ./cmd/engine/ && go build ./cmd/bitcode/`

**Step 2: Run full tests**

Run: `cd engine && go test ./... -count=1`

**Step 3: Verify with sample app**

Run: `cd samples/erp && MODULE_DIR=modules go run ../../engine/cmd/engine/main.go`

Check:
- Tables created with correct prefix names
- API endpoints work at new paths (`/api/contact` not `/api/contacts`)
- Seed data loaded correctly
- Admin panel works

**Step 4: Final commit and push**

```
git add -A
git commit -m "feat: table prefix support + postgres schema support"
git push
```
