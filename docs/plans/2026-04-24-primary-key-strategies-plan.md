# Primary Key Strategies & Sequence Engine — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add 6 configurable PK strategies (auto-increment, composite, UUID v4/v7/format, natural key, naming series, manual) with a format template engine and race-condition-safe sequence system to the BitCode engine.

**Architecture:** Model-level `primary_key` config in JSON parsed into `PrimaryKeyConfig` struct. Three new packages: format engine (template resolution), sequence engine (atomic DB counters), PK generator (strategy dispatcher). All create paths delegate to PK generator. Backward compatible — no config = UUID v4.

**Tech Stack:** Go 1.23+, GORM, google/uuid (v4/v5/v7), crypto/sha256 (hash function), encoding/base64 (composite key URLs), math/rand (random functions)

**Design Doc:** `docs/plans/2026-04-24-primary-key-strategies-design.md`

---

## Task 1: Parser — Add PK and AutoFormat Structs

**Files:**
- Modify: `engine/internal/compiler/parser/model.go`
- Test: `engine/internal/compiler/parser/model_test.go`

**Step 1: Add PK strategy types and config structs to model.go**

Add after the `FieldType` constants (after line 48):

```go
type PKStrategy string

const (
	PKAutoIncrement PKStrategy = "auto_increment"
	PKComposite     PKStrategy = "composite"
	PKUUID          PKStrategy = "uuid"
	PKNaturalKey    PKStrategy = "natural_key"
	PKNamingSeries  PKStrategy = "naming_series"
	PKManual        PKStrategy = "manual"
)

type SequenceConfig struct {
	Reset string `json:"reset,omitempty"`
	Step  int    `json:"step,omitempty"`
}

type PrimaryKeyConfig struct {
	Strategy  PKStrategy      `json:"strategy"`
	Field     string          `json:"field,omitempty"`
	Fields    []string        `json:"fields,omitempty"`
	Surrogate *bool           `json:"surrogate,omitempty"`
	Version   string          `json:"version,omitempty"`
	Format    string          `json:"format,omitempty"`
	Namespace string          `json:"namespace,omitempty"`
	Sequence  *SequenceConfig `json:"sequence,omitempty"`
}

func (pk *PrimaryKeyConfig) IsSurrogate() bool {
	if pk.Surrogate == nil {
		return true
	}
	return *pk.Surrogate
}

type AutoFormatConfig struct {
	Format   string          `json:"format"`
	Sequence *SequenceConfig `json:"sequence,omitempty"`
}
```

**Step 2: Add PrimaryKey to ModelDefinition and AutoFormat to FieldDefinition**

In `ModelDefinition` struct, add after `Inherit`:
```go
PrimaryKey *PrimaryKeyConfig `json:"primary_key,omitempty"`
```

In `FieldDefinition` struct, add after `NameFormat`:
```go
AutoFormat *AutoFormatConfig `json:"auto_format,omitempty"`
```

**Step 3: Add PK validation to ParseModel**

Add after the existing field validation loop (after line 146), before `return &model, nil`:

```go
if err := validatePrimaryKey(&model); err != nil {
	return nil, err
}
```

Add the validation function:

```go
func validatePrimaryKey(model *ModelDefinition) error {
	pk := model.PrimaryKey
	if pk == nil {
		return nil // default UUID v4
	}

	switch pk.Strategy {
	case PKAutoIncrement:
		// no extra config needed
	case PKComposite:
		if len(pk.Fields) < 2 {
			return fmt.Errorf("composite primary key requires at least 2 fields")
		}
		for _, f := range pk.Fields {
			if _, ok := model.Fields[f]; !ok {
				return fmt.Errorf("composite key field %q not found in model fields", f)
			}
		}
	case PKUUID:
		v := pk.Version
		if v == "" {
			v = "v4"
		}
		if v != "v4" && v != "v7" && v != "format" {
			return fmt.Errorf("uuid version must be v4, v7, or format, got %q", v)
		}
		if v == "format" && pk.Format == "" {
			return fmt.Errorf("uuid format version requires a format template")
		}
	case PKNaturalKey:
		if pk.Field == "" {
			return fmt.Errorf("natural_key strategy requires a field name")
		}
		f, ok := model.Fields[pk.Field]
		if !ok {
			return fmt.Errorf("natural_key field %q not found in model fields", pk.Field)
		}
		if !f.Required {
			return fmt.Errorf("natural_key field %q must be required", pk.Field)
		}
	case PKNamingSeries:
		if pk.Field == "" {
			return fmt.Errorf("naming_series strategy requires a field name")
		}
		if _, ok := model.Fields[pk.Field]; !ok {
			return fmt.Errorf("naming_series field %q not found in model fields", pk.Field)
		}
		if pk.Format == "" {
			return fmt.Errorf("naming_series strategy requires a format template")
		}
	case PKManual:
		if pk.Field == "" {
			return fmt.Errorf("manual strategy requires a field name")
		}
		if _, ok := model.Fields[pk.Field]; !ok {
			return fmt.Errorf("manual field %q not found in model fields", pk.Field)
		}
	default:
		return fmt.Errorf("unknown primary key strategy: %q", pk.Strategy)
	}

	if pk.Sequence != nil {
		validResets := map[string]bool{
			"never": true, "minutely": true, "hourly": true,
			"daily": true, "monthly": true, "yearly": true, "key": true,
		}
		if pk.Sequence.Reset != "" && !validResets[pk.Sequence.Reset] {
			return fmt.Errorf("invalid sequence reset mode: %q", pk.Sequence.Reset)
		}
	}

	return nil
}
```

**Step 4: Write tests for PK parsing and validation**

Create/extend `engine/internal/compiler/parser/model_test.go` with tests for:
- Default (no primary_key) parses successfully
- Each of the 6 strategies parses correctly
- Validation errors: composite with <2 fields, natural_key with missing field, naming_series without format, uuid format without format, unknown strategy
- AutoFormat on a field parses correctly

**Step 5: Run tests**

Run: `cd engine && go test ./internal/compiler/parser/ -v`
Expected: All tests pass

**Step 6: Commit**

```bash
git add engine/internal/compiler/parser/
git commit -m "feat(parser): add primary key strategy and auto-format config structs with validation"
```

---

## Task 2: Sequence Engine — Atomic Sequence Generation

**Files:**
- Create: `engine/internal/infrastructure/persistence/sequence.go`
- Test: `engine/internal/infrastructure/persistence/sequence_test.go`

**Step 1: Create the sequence engine**

Create `engine/internal/infrastructure/persistence/sequence.go`:

```go
package persistence

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SequenceEngine struct {
	db *gorm.DB
}

func NewSequenceEngine(db *gorm.DB) *SequenceEngine {
	return &SequenceEngine{db: db}
}

func (s *SequenceEngine) MigrateSequenceTable() error {
	dialect := DetectDialect(s.db)
	var sql string

	switch dialect {
	case DialectPostgres:
		sql = `CREATE TABLE IF NOT EXISTS sequences (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			model_name VARCHAR(100) NOT NULL,
			field_name VARCHAR(100) NOT NULL,
			sequence_key VARCHAR(500) NOT NULL,
			next_value BIGINT NOT NULL DEFAULT 1,
			step INT NOT NULL DEFAULT 1,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE(model_name, field_name, sequence_key)
		)`
	case DialectMySQL:
		sql = `CREATE TABLE IF NOT EXISTS sequences (
			id CHAR(36) PRIMARY KEY,
			model_name VARCHAR(100) NOT NULL,
			field_name VARCHAR(100) NOT NULL,
			sequence_key VARCHAR(500) NOT NULL,
			next_value BIGINT NOT NULL DEFAULT 1,
			step INT NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			UNIQUE KEY uq_seq (model_name, field_name, sequence_key)
		)`
	default: // SQLite
		sql = `CREATE TABLE IF NOT EXISTS sequences (
			id TEXT PRIMARY KEY,
			model_name TEXT NOT NULL,
			field_name TEXT NOT NULL,
			sequence_key TEXT NOT NULL,
			next_value INTEGER NOT NULL DEFAULT 1,
			step INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL DEFAULT (datetime('now')),
			updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
			UNIQUE(model_name, field_name, sequence_key)
		)`
	}

	return s.db.Exec(sql).Error
}

func (s *SequenceEngine) NextValue(modelName, fieldName, sequenceKey string, step int) (int64, error) {
	if step <= 0 {
		step = 1
	}

	dialect := DetectDialect(s.db)

	switch dialect {
	case DialectPostgres:
		var val int64
		err := s.db.Raw(`
			INSERT INTO sequences (id, model_name, field_name, sequence_key, next_value, step)
			VALUES (gen_random_uuid(), ?, ?, ?, 2, ?)
			ON CONFLICT (model_name, field_name, sequence_key)
			DO UPDATE SET next_value = sequences.next_value + sequences.step,
			             updated_at = NOW()
			RETURNING next_value - step
		`, modelName, fieldName, sequenceKey, step).Scan(&val).Error
		return val, err

	case DialectMySQL:
		newID := uuid.New().String()
		if err := s.db.Exec(`
			INSERT INTO sequences (id, model_name, field_name, sequence_key, next_value, step)
			VALUES (?, ?, ?, ?, 2, ?)
			ON DUPLICATE KEY UPDATE next_value = next_value + step,
			                        updated_at = NOW()
		`, newID, modelName, fieldName, sequenceKey, step).Error; err != nil {
			return 0, err
		}
		var val int64
		err := s.db.Raw(`
			SELECT next_value - step FROM sequences
			WHERE model_name = ? AND field_name = ? AND sequence_key = ?
		`, modelName, fieldName, sequenceKey).Scan(&val).Error
		return val, err

	default: // SQLite
		newID := uuid.New().String()
		var val int64
		err := s.db.Raw(`
			INSERT INTO sequences (id, model_name, field_name, sequence_key, next_value, step)
			VALUES (?, ?, ?, ?, 2, ?)
			ON CONFLICT (model_name, field_name, sequence_key)
			DO UPDATE SET next_value = next_value + step,
			             updated_at = CURRENT_TIMESTAMP
			RETURNING next_value - step
		`, newID, modelName, fieldName, sequenceKey, step).Scan(&val).Error
		return val, err
	}
}

func BuildSequenceKey(modelName, fieldName, reset string) string {
	now := time.Now()
	switch reset {
	case "yearly":
		return fmt.Sprintf("%s:%s:%d", modelName, fieldName, now.Year())
	case "monthly":
		return fmt.Sprintf("%s:%s:%d-%02d", modelName, fieldName, now.Year(), now.Month())
	case "daily":
		return fmt.Sprintf("%s:%s:%d-%02d-%02d", modelName, fieldName, now.Year(), now.Month(), now.Day())
	case "hourly":
		return fmt.Sprintf("%s:%s:%d-%02d-%02dT%02d", modelName, fieldName, now.Year(), now.Month(), now.Day(), now.Hour())
	case "minutely":
		return fmt.Sprintf("%s:%s:%d-%02d-%02dT%02d:%02d", modelName, fieldName, now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute())
	default: // "never" or empty
		return fmt.Sprintf("%s:%s", modelName, fieldName)
	}
}
```

**Step 2: Write tests**

Test `NextValue` with SQLite (in-memory):
- First call returns 1
- Second call returns 2
- Different keys return independent sequences
- Step > 1 increments correctly
- Concurrent goroutines get unique values (race condition test)

Test `BuildSequenceKey` for each reset mode.

**Step 3: Run tests**

Run: `cd engine && go test ./internal/infrastructure/persistence/ -v -run TestSequence`
Expected: All pass

**Step 4: Commit**

```bash
git add engine/internal/infrastructure/persistence/sequence.go engine/internal/infrastructure/persistence/sequence_test.go
git commit -m "feat(sequence): add atomic sequence engine with race-condition-safe DB operations"
```

---

## Task 3: Format Template Engine

**Files:**
- Create: `engine/internal/runtime/format/engine.go`
- Test: `engine/internal/runtime/format/engine_test.go`

**Step 1: Create the format engine**

Create `engine/internal/runtime/format/engine.go` with:

```go
package format

// FormatContext holds all data available during template resolution
type FormatContext struct {
	Data      map[string]any // record field values
	Session   map[string]any // user_id, username, tenant_id, group_id, group_code
	Settings  map[string]string // key-value from settings table
	ModelName string
	Module    string
	Now       time.Time
}

// Engine resolves format templates like "INV/{time.year}/{sequence(6)}"
type Engine struct {
	sequenceEngine *persistence.SequenceEngine
}

func NewEngine(seqEngine *persistence.SequenceEngine) *Engine

// Resolve resolves a format template, replacing all tokens.
// sequenceKey params are needed for {sequence(N)} resolution.
func (e *Engine) Resolve(template string, ctx *FormatContext, modelName, fieldName, resetMode string) (string, error)
```

Implement token parsing with regex: `\{([^}]+)\}` to find all tokens.

For each token, resolve:
- `data.<field>` → `ctx.Data[field]`
- `time.year` → `fmt.Sprintf("%d", ctx.Now.Year())`
- `time.month` → `fmt.Sprintf("%02d", ctx.Now.Month())`
- `time.day` → `fmt.Sprintf("%02d", ctx.Now.Day())`
- `time.hour` → `fmt.Sprintf("%02d", ctx.Now.Hour())`
- `time.minute` → `fmt.Sprintf("%02d", ctx.Now.Minute())`
- `time.now` → `ctx.Now.Format(time.RFC3339)`
- `time.date` → `ctx.Now.Format("2006-01-02")`
- `time.unix` → `fmt.Sprintf("%d", ctx.Now.Unix())`
- `session.<key>` → `ctx.Session[key]`
- `setting.<key>` → `ctx.Settings[key]`
- `model.name` → `ctx.ModelName`
- `model.module` → `ctx.Module`
- `sequence(N)` → call `sequenceEngine.NextValue(...)`, zero-pad to N digits
- `substring(src,start,len)` → resolve src first, then substring
- `upper(src)` → resolve src first, then strings.ToUpper
- `lower(src)` → resolve src first, then strings.ToLower
- `hash(src)` → resolve src first, then sha256 first 8 hex chars
- `uuid4` → uuid.New().String()
- `uuid7` → uuid.Must(uuid.NewV7()).String()
- `random(N)` → N random alphanumeric chars
- `random_fixed(len,min,max)` → random int between min-max, zero-padded to len
- `hash(src)` → sha256 first 8 hex chars

For `key` reset mode: strip `{sequence(N)}` from template, resolve remaining tokens, use as sequence key.

**Step 2: Write comprehensive tests**

Test each token type individually, then combined templates like:
- `"INV/{time.year}/{sequence(6)}"` → `"INV/2026/000001"`
- `"{upper(data.dept)}-{sequence(4)}-{time.year}"` → `"SALES-0001-2026"`
- `"{substring(data.nik,0,3)}/{random_fixed(3,100,999)}"` → `"320/547"`

**Step 3: Run tests**

Run: `cd engine && go test ./internal/runtime/format/ -v`
Expected: All pass

**Step 4: Commit**

```bash
git add engine/internal/runtime/format/
git commit -m "feat(format): add template engine with 30+ built-in functions for PK and field generation"
```

---

## Task 4: PK Generator — Strategy Dispatcher

**Files:**
- Create: `engine/internal/runtime/pkgen/generator.go`
- Test: `engine/internal/runtime/pkgen/generator_test.go`

**Step 1: Create the PK generator**

```go
package pkgen

type Generator struct {
	formatEngine   *format.Engine
	sequenceEngine *persistence.SequenceEngine
	settingsLookup func(key string) string
}

func NewGenerator(fe *format.Engine, se *persistence.SequenceEngine, sl func(string) string) *Generator

// GeneratePK generates the primary key value(s) for a new record.
// Returns the PK column name and value to set on the record.
// For composite with surrogate, returns "id" + UUID.
// For auto_increment, returns "" (DB handles it).
func (g *Generator) GeneratePK(model *parser.ModelDefinition, record map[string]any, session map[string]any) (pkColumn string, pkValue any, err error)

// GenerateAutoFormat generates values for fields with auto_format config.
func (g *Generator) GenerateAutoFormat(model *parser.ModelDefinition, fieldName string, field *parser.FieldDefinition, record map[string]any, session map[string]any) (string, error)

// GetPKColumn returns the PK column name for a model.
func GetPKColumn(model *parser.ModelDefinition) string

// IsAutoGeneratedPK returns true if the PK is auto-generated (hidden in forms).
func IsAutoGeneratedPK(model *parser.ModelDefinition) bool

// IsCompositeNoSurrogate returns true if model uses composite PK without surrogate id.
func IsCompositeNoSurrogate(model *parser.ModelDefinition) bool
```

Strategy dispatch in `GeneratePK`:
- `nil` or `uuid` v4 → `"id", uuid.New().String()`
- `uuid` v7 → `"id", uuid.Must(uuid.NewV7()).String()`
- `uuid` format → resolve format template → `"id", uuid.NewSHA1(namespace, resolved).String()`
- `auto_increment` → `"", nil` (DB handles)
- `natural_key` → `pk.Field, record[pk.Field]` (user-provided, validate non-empty)
- `naming_series` → resolve format → `pk.Field, resolved`
- `manual` → `pk.Field, record[pk.Field]` (user-provided, validate non-empty)
- `composite` surrogate → `"id", uuid.New().String()` (plus validate composite fields)
- `composite` no-surrogate → `"", nil` (composite fields already in record)

**Step 2: Write tests for each strategy**

**Step 3: Run tests**

Run: `cd engine && go test ./internal/runtime/pkgen/ -v`

**Step 4: Commit**

```bash
git add engine/internal/runtime/pkgen/
git commit -m "feat(pkgen): add PK generator with 6-strategy dispatcher"
```

---

## Task 5: Migration — Strategy-Aware DDL Generation

**Files:**
- Modify: `engine/internal/infrastructure/persistence/dynamic_model.go`
- Test: `engine/internal/infrastructure/persistence/dynamic_model_test.go`

**Step 1: Refactor `buildColumns` to use PK strategy**

Replace the hardcoded `id UUID PRIMARY KEY...` block with strategy-aware PK column generation:

- `nil` / `uuid` (v4/v7/format) → current UUID PK behavior (unchanged)
- `auto_increment` → `id BIGSERIAL PRIMARY KEY` (Postgres), `id BIGINT AUTO_INCREMENT PRIMARY KEY` (MySQL), `id INTEGER PRIMARY KEY AUTOINCREMENT` (SQLite)
- `natural_key` → use the referenced field as PK (skip adding it again in field loop)
- `naming_series` → use the referenced field as PK with VARCHAR type
- `manual` → use the referenced field as PK
- `composite` surrogate → current UUID `id` + UNIQUE constraint on composite fields
- `composite` no-surrogate → no `id` column, `PRIMARY KEY (field1, field2, ...)` at end

Also update `fieldTypeToSQL` for `FieldMany2One` to check if the referenced model uses auto_increment (INTEGER FK) vs UUID (current behavior). For now, default to current UUID FK behavior — cross-model PK type resolution can be a follow-up.

**Step 2: Update `createJunctionTable` for auto_increment models**

If the model uses auto_increment, junction table FK columns should be INTEGER instead of UUID.

**Step 3: Write tests**

Test DDL output for each strategy + each dialect (SQLite focus for unit tests).

**Step 4: Run tests**

Run: `cd engine && go test ./internal/infrastructure/persistence/ -v -run TestMigrate`

**Step 5: Commit**

```bash
git add engine/internal/infrastructure/persistence/dynamic_model.go engine/internal/infrastructure/persistence/dynamic_model_test.go
git commit -m "feat(migration): strategy-aware DDL generation for all 6 PK types"
```

---

## Task 6: Repository — PK-Aware CRUD Operations

**Files:**
- Modify: `engine/internal/infrastructure/persistence/repository.go`

**Step 1: Add model awareness to GenericRepository**

Add `modelDef *parser.ModelDefinition` field to `GenericRepository`. Update constructors:

```go
func NewGenericRepository(db *gorm.DB, tableName string) *GenericRepository // backward compat
func NewGenericRepositoryWithModel(db *gorm.DB, tableName string, model *parser.ModelDefinition) *GenericRepository
```

**Step 2: Update Create to use PK generator**

Add optional `pkGenerator *pkgen.Generator` and `session map[string]any` to repository or accept them as params. In `Create`:
- If model has PK generator, call it to get PK column + value
- Set on record before insert
- For auto_increment, skip PK assignment (DB handles)
- Also process `auto_format` fields

**Step 3: Update FindByID, Update, Delete for custom PK column**

Use `pkgen.GetPKColumn(model)` instead of hardcoded `"id"`:
```go
pkCol := pkgen.GetPKColumn(r.modelDef) // returns "id", "code", "sku", etc.
query.Where(fmt.Sprintf("%s = ?", pkCol), id)
```

**Step 4: Add FindByCompositePK for no-surrogate composite**

```go
func (r *GenericRepository) FindByCompositePK(ctx context.Context, keys map[string]any) (map[string]any, error)
```

Builds `WHERE field1 = ? AND field2 = ? AND ...` from the keys map.

**Step 5: Add base64 decode helper**

```go
func DecodeCompositePK(encoded string) (map[string]any, error)
func EncodeCompositePK(keys map[string]any) string
```

**Step 6: Run tests**

Run: `cd engine && go test ./internal/infrastructure/persistence/ -v`

**Step 7: Commit**

```bash
git add engine/internal/infrastructure/persistence/repository.go
git commit -m "feat(repository): PK-aware CRUD with composite key support and base64 encoding"
```

---

## Task 7: CRUD Handler — Remove Hardcoded UUID

**Files:**
- Modify: `engine/internal/presentation/api/crud_handler.go`

**Step 1: Add model and PK generator to CRUDHandler**

Add `modelDef *parser.ModelDefinition` and `pkGenerator *pkgen.Generator` fields.

**Step 2: Update Create handler**

Remove `body["id"] = uuid.New().String()`. Instead:
- Call `pkGenerator.GeneratePK(model, body, session)` to get PK
- Process `auto_format` fields
- Set PK on body if returned

**Step 3: Update Read/Update/Delete for custom PK**

- `Read`: detect composite no-surrogate → decode base64 → `FindByCompositePK`
- `Update`: use correct PK column name; strip PK fields from update body
- `Delete`: use correct PK column

**Step 4: Run tests**

Run: `cd engine && go test ./internal/presentation/api/ -v`

**Step 5: Commit**

```bash
git add engine/internal/presentation/api/crud_handler.go
git commit -m "feat(crud): delegate PK generation to pkgen, support all 6 strategies"
```

---

## Task 8: App Wiring — Initialize and Connect Everything

**Files:**
- Modify: `engine/internal/app.go`

**Step 1: Initialize sequence engine and format engine in App**

In `App` struct, add:
```go
SequenceEngine *persistence.SequenceEngine
FormatEngine   *format.Engine
PKGenerator    *pkgen.Generator
```

In app initialization:
```go
a.SequenceEngine = persistence.NewSequenceEngine(a.DB)
a.SequenceEngine.MigrateSequenceTable()
a.FormatEngine = format.NewEngine(a.SequenceEngine)
a.PKGenerator = pkgen.NewGenerator(a.FormatEngine, a.SequenceEngine, a.settingsLookup)
```

**Step 2: Pass PK generator to repository and CRUD handler creation**

Where `NewGenericRepository` and `NewCRUDHandler` are called, pass the model definition and PK generator.

**Step 3: Update SSR form POST handler**

In `handleViewPost`, before `repo.Create(...)`:
- Call PK generator for the model
- Process auto_format fields

**Step 4: Run full test suite**

Run: `cd engine && go test ./... -v`

**Step 5: Commit**

```bash
git add engine/internal/app.go
git commit -m "feat(app): wire sequence engine, format engine, and PK generator into app lifecycle"
```

---

## Task 9: Seeder & Process Steps — Use PK Generator

**Files:**
- Modify: `engine/internal/infrastructure/module/seeder.go`
- Modify: `engine/internal/runtime/executor/steps/data.go`

**Step 1: Update seeder**

In `seedRecords`, replace hardcoded UUID fallback:
```go
// Old: record["id"] = uuid.New().String()
// New: use PK generator if model is known, else fallback to UUID
```

The seeder needs access to model definitions. Add a `ModelLookup` parameter or keep UUID fallback for seed data (seed data typically has explicit IDs).

**Step 2: Update process data create step**

In `DataHandler`, add PK generator. In `executeCreate`, call PK generator before `repo.Create`.

**Step 3: Run tests**

Run: `cd engine && go test ./... -v`

**Step 4: Commit**

```bash
git add engine/internal/infrastructure/module/seeder.go engine/internal/runtime/executor/steps/data.go
git commit -m "feat(seeder,process): use PK generator for record creation"
```

---

## Task 10: Form/View Compiler — Auto-Hide PK Fields

**Files:**
- Modify: `engine/internal/presentation/view/component_compiler.go`

**Step 1: Add PK visibility logic**

In `compileFormRow` and `compileFormFull`, when rendering a field:
- Check if the field is the PK field for an auto-generated strategy
- If so, skip rendering (hidden on create) or render as read-only (on update)
- Check if the field has `auto_format` → hidden on create, read-only on update

Add helper method:
```go
func (c *ComponentCompiler) shouldHideField(model *parser.ModelDefinition, fieldName string, isEdit bool) bool
func (c *ComponentCompiler) shouldReadonlyField(model *parser.ModelDefinition, fieldName string, isEdit bool) bool
```

**Step 2: Run tests**

Run: `cd engine && go test ./internal/presentation/view/ -v`

**Step 3: Commit**

```bash
git add engine/internal/presentation/view/component_compiler.go
git commit -m "feat(view): auto-hide/readonly PK and auto_format fields in forms"
```

---

## Task 11: Update Sample ERP Models

**Files:**
- Modify: `samples/erp/modules/crm/models/lead.json` — add naming_series example
- Modify: `samples/erp/modules/hrm/models/employee.json` — add natural_key example
- Create: `samples/erp/modules/crm/models/country.json` — natural key example
- Modify: `engine/modules/sales/models/order.json` — add naming_series for order code

**Step 1: Add PK config to sample models**

Example for lead.json:
```json
{
  "name": "lead",
  "primary_key": {
    "strategy": "naming_series",
    "field": "lead_code",
    "format": "LEAD/{time.year}/{sequence(5)}",
    "sequence": { "reset": "yearly", "step": 1 }
  },
  ...add "lead_code": { "type": "string", "max": 30 } to fields...
}
```

**Step 2: Verify samples load correctly**

Run the sample ERP and verify models migrate and CRUD works.

**Step 3: Commit**

```bash
git add samples/ engine/modules/
git commit -m "feat(samples): add PK strategy examples to sample ERP models"
```

---

## Task 12: Update Documentation

**Files:**
- Modify: `docs/codebase.md` — add new files (sequence.go, format/engine.go, pkgen/generator.go)
- Modify: `docs/features.md` — mark PK strategies as complete
- Modify: `docs/architecture.md` — add PK generator to data flow
- Create: `engine/docs/features/primary-keys.md` — per-feature deep doc
- Modify: `engine/docs/features/models.md` — add PK strategy section
- Modify: `README.md` — mention PK strategies in features list
- Modify: `AGENTS.md` — move PK strategies from remaining to completed

**Step 1: Write all doc updates**

**Step 2: Commit**

```bash
git add docs/ engine/docs/ README.md AGENTS.md
git commit -m "docs: add primary key strategies documentation"
```

---

## Task 13: Run Full Test Suite & Final Verification

**Step 1: Run all tests**

```bash
cd engine && go test ./... -v -count=1
```

Expected: All tests pass, 0 failures.

**Step 2: Build**

```bash
cd engine && go build -o bin/engine cmd/engine/main.go && go build -o bin/bitcode cmd/bitcode/main.go
```

Expected: Clean build.

**Step 3: Smoke test with sample ERP**

```bash
cd samples/erp && MODULE_DIR=modules go run ../../engine/cmd/engine/main.go
```

Verify:
- Server starts
- Models with PK strategies migrate correctly
- CRUD operations work for each strategy type
- Sequence numbers increment correctly

**Step 4: Final commit and push**

```bash
git add -A
git commit -m "feat: primary key strategies with format engine and sequence system

Add 6 configurable PK strategies: auto-increment, composite (surrogate/no-surrogate),
UUID (v4/v7/format), natural key, naming series, and manual input.

New subsystems:
- Format template engine with 30+ built-in functions
- Atomic sequence engine with race-condition-safe DB operations
- PK generator strategy dispatcher

Backward compatible: no primary_key config = UUID v4 (existing behavior)."

git push
```

---

## Dependency Graph

```
Task 1 (Parser) ──────────┐
                           ├──→ Task 4 (PK Generator) ──→ Task 7 (CRUD Handler) ──→ Task 8 (App Wiring)
Task 2 (Sequence Engine) ──┤                                                              │
                           ├──→ Task 3 (Format Engine) ──────────────────────────────────────┤
                           │                                                              │
                           └──────────────────────────────────────────────────────────────→ Task 5 (Migration)
                                                                                          │
Task 6 (Repository) ←── depends on Task 4                                                │
Task 9 (Seeder/Process) ←── depends on Task 4, Task 8                                    │
Task 10 (View Compiler) ←── depends on Task 1                                            │
Task 11 (Samples) ←── depends on Task 8                                                  │
Task 12 (Docs) ←── depends on all above                                                  │
Task 13 (Final) ←── depends on all above                                                 │
```

**Parallelizable:** Tasks 1, 2, 3 can run in parallel. Tasks 5, 6, 10 can run in parallel after Task 1+4.
