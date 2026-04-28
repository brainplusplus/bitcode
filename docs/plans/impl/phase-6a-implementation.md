# Phase 6A Implementation Plan: Schema Compatibility

**Estimated effort**: 5-7 days
**Prerequisites**: None (independent phase)
**Test command**: `go test ./internal/compiler/parser/... ./internal/infrastructure/persistence/... ./internal/domain/model/...`

---

## Implementation Order

Phase 6A has 6 work streams. Order matters — later streams depend on earlier ones.

```
Stream 1: Parser — New Types & Fields (Day 1-2)
  ↓
Stream 2: Migration — fieldTypeToSQL Rewrite (Day 2-3)
  ↓
Stream 3: Validation — Built-in Validators (Day 3)
  ↓
Stream 4: Table Naming & Duplicate Detection (Day 3-4)
  ↓
Stream 5: Title Field Format & Display Field (Day 4-5)
  ↓
Stream 6: Repository & View Updates (Day 5-6)
  ↓
Stream 7: Tests & Documentation (Day 6-7)
```

---

## Stream 1: Parser — New Types & Fields

### 1.1 Add FieldType Constants

**File**: `internal/compiler/parser/model.go`
**Location**: After line 47 (after `FieldRating`)

```go
// New types
FieldUUID       FieldType = "uuid"
FieldIP         FieldType = "ip"
FieldIPv6       FieldType = "ipv6"
FieldYear       FieldType = "year"
FieldVector     FieldType = "vector"
FieldBinary     FieldType = "binary"

// JSON variants
FieldJSONObject FieldType = "json:object"
FieldJSONArray  FieldType = "json:array"
```

### 1.2 Add FieldMaskConfig Struct

**File**: `internal/compiler/parser/model.go`
**Location**: After `FieldValidation` struct

```go
type FieldMaskConfig struct {
    ThousandSeparator string `json:"thousand_separator,omitempty"`
    DecimalSeparator  string `json:"decimal_separator,omitempty"`
    Precision         int    `json:"precision,omitempty"`
    Prefix            string `json:"prefix,omitempty"`
    Suffix            string `json:"suffix,omitempty"`
}
```

### 1.3 Add New Fields to FieldDefinition

**File**: `internal/compiler/parser/model.go`
**Location**: FieldDefinition struct (line 340-389)

Add after `Groups` field:

```go
// Phase 6A additions
Hidden        bool             `json:"hidden,omitempty"`
Storage       string           `json:"storage,omitempty"`
Scale         int              `json:"scale,omitempty"`
DisplayField  string           `json:"display_field,omitempty"`
CurrencyField string           `json:"currency_field,omitempty"`
Dimensions    int              `json:"dimensions,omitempty"`
FieldIndex    any              `json:"index,omitempty"`
FieldMask     *FieldMaskConfig `json:"mask,omitempty"`
```

**Note**: `Mask` field name conflicts with existing `Mask bool` on line 386. Rename existing `Mask` → `MaskPassword` (or similar) and update all references. Check with `grep -r "\.Mask\b" internal/` first.

### 1.4 Add Plural Field to ModelTableConfig

**File**: `internal/compiler/parser/model.go`
**Location**: ModelTableConfig struct (line 401-403)

```go
type ModelTableConfig struct {
    Prefix string `json:"prefix"`
    Plural *bool  `json:"plural,omitempty"`  // NEW
    Name   string `json:"name,omitempty"`    // NEW: explicit table name alternative
}
```

### 1.5 Update ParseModel — Type Variant Parsing

**File**: `internal/compiler/parser/model.go`
**Location**: ParseModel function (line 505-559)

Before the field validation loop (line 516), add type variant resolution:

```go
for name, field := range model.Fields {
    // Resolve type variants
    rawType := string(field.Type)
    switch rawType {
    case "json:object":
        field.Type = FieldJSONObject
    case "json:array":
        field.Type = FieldJSONArray
    case "ip:v4":
        field.Type = FieldIP
    case "ip:v6", "ipv6":
        field.Type = FieldIPv6
    default:
        if strings.Contains(rawType, ":") {
            base := strings.SplitN(rawType, ":", 2)[0]
            if base != "json" && base != "ip" {
                return nil, fmt.Errorf("field %q: type %q does not support variants (only json and ip do)", name, rawType)
            }
        }
    }
    model.Fields[name] = field
    // ... existing validation ...
}
```

### 1.6 Update ParseModel — New Validation Rules

**File**: `internal/compiler/parser/model.go`
**Location**: Inside the field validation loop

Add after existing validations (line 537):

```go
// Vector must have dimensions
if field.Type == FieldVector && field.Dimensions == 0 {
    return nil, fmt.Errorf("vector field %q must specify dimensions", name)
}

// currency and currency_field mutually exclusive
if field.CurrencyCode != "" && field.CurrencyField != "" {
    return nil, fmt.Errorf("field %q cannot have both currency and currency_field", name)
}

// display_field only valid on many2one
if field.DisplayField != "" && field.Type != FieldMany2One {
    // Warning, not error — log and continue
    log.Printf("WARN: display_field on non-many2one field %q will be ignored", name)
}

// storage hint validation
if field.Storage != "" {
    if !isValidStorageHint(field.Type, field.Storage) {
        return nil, fmt.Errorf("field %q: invalid storage hint %q for type %q", name, field.Storage, field.Type)
    }
}
```

### 1.7 Add isValidStorageHint Function

**File**: `internal/compiler/parser/model.go`
**Location**: New function

```go
func isValidStorageHint(fieldType FieldType, storage string) bool {
    valid := map[FieldType][]string{
        FieldInteger:  {"smallint", "bigint"},
        FieldDecimal:  {"numeric", "double"},
        FieldFloat:    {"double", "real"},
        FieldText:     {"mediumtext", "longtext"},
        FieldString:   {"char"},
        FieldBinary:   {"mediumblob", "longblob"},
        FieldDatetime: {"naive"},
        FieldCurrency: {"numeric"},
    }
    hints, ok := valid[fieldType]
    if !ok {
        return false // type doesn't support storage hints
    }
    for _, h := range hints {
        if h == storage {
            return true
        }
    }
    return false
}
```

### 1.8 Update search_field Auto-Extraction

**File**: `internal/compiler/parser/model.go`
**Location**: After `resolveTitleField` call (line 542-547)

Replace:
```go
if len(model.SearchField) == 0 {
    model.SearchField = []string{model.TitleField}
}
```

With:
```go
if len(model.SearchField) == 0 {
    if strings.Contains(model.TitleField, "{") {
        model.SearchField = extractSearchableFields(&model, model.TitleField)
    } else {
        model.SearchField = []string{model.TitleField}
    }
}
```

### 1.9 Add extractSearchableFields Function

**File**: `internal/compiler/parser/model.go`

```go
func extractSearchableFields(model *ModelDefinition, format string) []string {
    // Extract {data.xxx} tokens, including inside functions like {upper(data.xxx)}
    re := regexp.MustCompile(`\{(?:[a-z]+\()?data\.([a-z_]+)`)
    matches := re.FindAllStringSubmatch(format, -1)

    var fields []string
    seen := make(map[string]bool)
    for _, m := range matches {
        fieldName := m[1]
        if seen[fieldName] {
            continue
        }
        seen[fieldName] = true
        // Only include string/text family fields
        if fd, ok := model.Fields[fieldName]; ok {
            if isTextSearchable(fd.Type) {
                fields = append(fields, fieldName)
            }
        }
    }

    if len(fields) == 0 {
        // Fallback to auto-detected title field
        return []string{resolveTitleField(model)}
    }
    return fields
}

func isTextSearchable(ft FieldType) bool {
    switch ft {
    case FieldString, FieldText, FieldEmail, FieldPassword, FieldBarcode,
         FieldColor, FieldCode, FieldSmallText, FieldRichText, FieldMarkdown,
         FieldHTML, FieldIP, FieldIPv6, FieldUUID:
        return true
    }
    return false
}
```

### 1.10 Update TableRaw Parsing for Plural

**File**: `internal/compiler/parser/model.go`
**Location**: TableRaw parsing block (line 548-559)

Update to also parse `plural` from table config:

```go
if len(model.TableRaw) > 0 {
    var tableName string
    if err := json.Unmarshal(model.TableRaw, &tableName); err == nil {
        model.TableName = tableName
    } else {
        var tableConfig ModelTableConfig
        if err := json.Unmarshal(model.TableRaw, &tableConfig); err == nil {
            prefix := tableConfig.Prefix
            model.TablePrefix = &prefix
            model.TablePlural = tableConfig.Plural  // NEW
            if tableConfig.Name != "" {
                model.TableName = tableConfig.Name
            }
        }
    }
}
```

Add `TablePlural *bool` to ModelDefinition:

```go
TablePlural  *bool   `json:"-"`
```

### Tests for Stream 1

**File**: `internal/compiler/parser/model_test.go`

- Test new type constants parse correctly
- Test json:object, json:array variant parsing
- Test ip:v4, ip:v6, ipv6 variant parsing
- Test invalid variant (e.g., "string:big") returns error
- Test vector without dimensions returns error
- Test currency + currency_field mutual exclusion
- Test display_field on non-many2one logs warning
- Test storage hint validation (valid and invalid)
- Test title_field format extraction
- Test search_field auto-extraction from format
- Test table config plural parsing

---

## Stream 2: Migration — fieldTypeToSQL Rewrite

### 2.1 Rewrite fieldTypeToSQL

**File**: `internal/infrastructure/persistence/dynamic_model.go`
**Location**: Replace entire function (line 206-282)

Rewrite to handle ALL types explicitly. No `default: return "TEXT"`.

Key additions:
- `FieldFloat` → DOUBLE PRECISION / DOUBLE / REAL
- `FieldCurrency` → NUMERIC(18,2) / DECIMAL(18,2) / REAL (respects precision/scale)
- `FieldPercent` → NUMERIC(5,2) / DECIMAL(5,2) / REAL
- `FieldRating` → SMALLINT / SMALLINT / INTEGER
- `FieldTime` → TIME / TIME / TEXT
- `FieldDuration` → INTEGER / INT / INTEGER
- `FieldToggle` → BOOLEAN / BOOLEAN / INTEGER
- `FieldRadio` → VARCHAR(50) / VARCHAR(50) / TEXT
- `FieldPassword` → VARCHAR(255) / VARCHAR(255) / TEXT
- `FieldSmallText` → VARCHAR(500) / VARCHAR(500) / TEXT
- `FieldRichText`, `FieldMarkdown`, `FieldHTML`, `FieldCode` → TEXT
- `FieldImage` → VARCHAR(500) / VARCHAR(500) / TEXT
- `FieldSignature` → TEXT
- `FieldBarcode` → VARCHAR(255) / VARCHAR(255) / TEXT
- `FieldColor` → VARCHAR(7) / VARCHAR(7) / TEXT
- `FieldGeolocation` → JSONB / JSON / TEXT
- `FieldDynamicLink` → VARCHAR(255) / VARCHAR(255) / TEXT
- `FieldUUID` → UUID / CHAR(36) / TEXT
- `FieldIP` → VARCHAR(45) / VARCHAR(45) / TEXT
- `FieldIPv6` → VARCHAR(45) / VARCHAR(45) / TEXT
- `FieldYear` → SMALLINT / SMALLINT / INTEGER
- `FieldVector` → vector(N) / JSON / TEXT
- `FieldBinary` → BYTEA / LONGBLOB / BLOB
- `FieldJSONObject`, `FieldJSONArray` → same as FieldJSON

### 2.2 Add Storage Hint Resolution

**File**: `internal/infrastructure/persistence/dynamic_model.go`

In the rewritten `fieldTypeToSQL`, add storage hint handling:

```go
case parser.FieldInteger:
    switch field.Storage {
    case "smallint":
        if dialect == DialectSQLite { return "INTEGER" }
        return "SMALLINT"
    case "bigint":
        if dialect == DialectSQLite { return "INTEGER" }
        return "BIGINT"
    default:
        return "INTEGER"
    }
```

Similar for decimal, text, datetime, binary.

### 2.3 Add Precision/Scale Resolution

**File**: `internal/infrastructure/persistence/dynamic_model.go`

```go
func resolveDecimalPrecision(field parser.FieldDefinition) (int, int) {
    if field.Scale > 0 {
        p := field.Precision
        if p == 0 { p = 18 }
        return p, field.Scale
    }
    if field.Precision > 0 {
        return 18, field.Precision // legacy: precision = scale
    }
    return 18, 2
}
```

### 2.4 Add Vector Column Support

**File**: `internal/infrastructure/persistence/dynamic_model.go`

In `MigrateModel`, before creating columns, check for pgvector extension:

```go
// Check for vector fields
for _, field := range model.Fields {
    if field.Type == parser.FieldVector && dialect == DialectPostgres {
        db.Exec("CREATE EXTENSION IF NOT EXISTS vector")
        break
    }
}
```

### 2.5 Add Vector Index Support

**File**: `internal/infrastructure/persistence/dynamic_model.go`

After table creation, create vector indexes:

```go
for fieldName, field := range model.Fields {
    if field.Type == parser.FieldVector && field.FieldIndex != nil {
        indexType := "hnsw" // default
        if s, ok := field.FieldIndex.(string); ok {
            indexType = s
        }
        if dialect == DialectPostgres {
            sql := fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_%s_vector ON %s USING %s (%s vector_cosine_ops)",
                tableName, fieldName, tableName, indexType, fieldName)
            db.Exec(sql)
        }
    }
}
```

### 2.6 Update mongo_migration.go

**File**: `internal/infrastructure/persistence/mongo_migration.go`

Add composite indexes for new types that need them (vector, etc.). MongoDB type mapping is application-level, not schema-level.

### Tests for Stream 2

**File**: `internal/infrastructure/persistence/dynamic_model_test.go` (NEW)

- Test fieldTypeToSQL for ALL types × 3 dialects
- Test storage hint resolution (integer+bigint, decimal+numeric, etc.)
- Test precision/scale resolution (legacy, new, default)
- Test vector column SQL generation
- Test vector index SQL generation
- Test no `default: TEXT` — every type has explicit mapping

---

## Stream 3: Validation — Built-in Validators

### 3.1 Add Type Validators

**File**: `internal/infrastructure/persistence/validator.go` (NEW or extend existing)

```go
func ValidateFieldValue(fieldName string, value any, field parser.FieldDefinition) error {
    switch field.Type {
    case parser.FieldEmail:
        return validateEmail(value)
    case parser.FieldUUID:
        return validateUUID(value)
    case parser.FieldIP:
        return validateIP(value) // both v4 and v6
    case parser.FieldIPv6:
        return validateIPv6(value) // strict v6
    case parser.FieldYear:
        return validateYear(value) // 1900-2300
    case parser.FieldColor:
        return validateColor(value) // #RRGGBB
    case parser.FieldVector:
        return validateVector(value, field.Dimensions)
    case parser.FieldJSONObject:
        return validateJSONObject(value)
    case parser.FieldJSONArray:
        return validateJSONArray(value)
    }
    return nil
}
```

### Tests for Stream 3

**File**: `internal/infrastructure/persistence/validator_test.go` (NEW)

- Test each validator with valid and invalid input
- Test edge cases (empty string, nil, wrong type)

---

## Stream 4: Table Naming & Duplicate Detection

### 4.1 Add jinzhu/inflection Dependency

```bash
go get github.com/jinzhu/inflection
```

### 4.2 Update ResolveTableName

**File**: `internal/domain/model/registry.go`
**Location**: ResolveTableName function (line 57-72)

Add `projectConfig` parameter and plural logic.

### 4.3 Replace Hand-Rolled pluralize()

**Files**:
- `internal/infrastructure/module/auto_api.go` — replace `pluralize()` with `inflection.Plural()`
- `internal/presentation/graphql/schema.go` — replace `pluralizeModel()` with `inflection.Plural()`

### 4.4 Add Duplicate Detection in Registry.Register

**File**: `internal/domain/model/registry.go`
**Location**: Register function (line 25-41)

Add duplicate check before assignment.

### 4.5 Update API Path Generation

**File**: `internal/infrastructure/module/auto_api.go`
**Location**: GenerateAPIFromModel (line 31)

Apply plural to API path based on project config.

### Tests for Stream 4

**File**: `internal/domain/model/registry_test.go` (extend existing)

- Test plural table name resolution
- Test per-model plural override
- Test duplicate model detection (same module = error)
- Test cross-module same name (OK)
- Test inheritance not flagged as duplicate

---

## Stream 5: Title Field Format & Display Field

### 5.1 Add Format Engine Integration for Title Field

**File**: `internal/infrastructure/persistence/repository.go`
**Location**: `loadMany2OneRelation` (line 1500-1545)

Update to resolve display label using title_field format or display_field.

### 5.2 Add resolveDisplayLabel Helper

**File**: `internal/infrastructure/persistence/repository.go`

```go
func resolveDisplayLabel(modelDef *parser.ModelDefinition, fieldDef *parser.FieldDefinition, record map[string]any) string {
    // 1. display_field override (per many2one field)
    tf := ""
    if fieldDef != nil && fieldDef.DisplayField != "" {
        tf = fieldDef.DisplayField
    } else if modelDef != nil {
        tf = modelDef.TitleField
    }
    if tf == "" {
        return fmt.Sprintf("%v", record["id"])
    }

    // 2. Format template
    if strings.Contains(tf, "{") {
        result, _ := formatEngine.Resolve(tf, &format.FormatContext{Data: record})
        return result
    }

    // 3. Simple field name
    if val, ok := record[tf]; ok {
        return fmt.Sprintf("%v", val)
    }
    return fmt.Sprintf("%v", record["id"])
}
```

### Tests for Stream 5

- Test title_field simple mode
- Test title_field format mode
- Test display_field override on many2one
- Test fallback to "id" with warning

---

## Stream 6: Repository & View Updates

### 6.1 Update Component Compiler — Widget Mapping

**File**: `internal/presentation/view/component_compiler.go`

Add widget mapping for all new types.

### 6.2 Update Component Compiler — Hidden Field

**File**: `internal/presentation/view/component_compiler.go`

Skip rendering fields with `hidden: true` in auto-generated views.

### 6.3 Update Auto Page Generator

**File**: `internal/presentation/view/auto_page_generator.go`

Exclude hidden fields from auto-generated list/form views.

### 6.4 Update Currency Rendering

**File**: `internal/presentation/view/component_compiler.go`

Support `currency_field` in currency field rendering.

---

## Stream 7: Tests & Documentation

### 7.1 Integration Tests

- End-to-end: define model with new types → migrate → insert → query → verify
- Test with PostgreSQL, MySQL, SQLite (if CI supports)

### 7.2 Documentation

- Update engine docs with new types
- Add examples for each new type
- Document storage hints
- Document title_field format

---

## Definition of Done

- [ ] All 40+ field types have explicit SQL mapping (no `default: TEXT`)
- [ ] New types (uuid, ip, ipv6, year, vector, binary) parse, migrate, and validate
- [ ] JSON variants (json:object, json:array) parse with correct defaults
- [ ] Storage hints work for integer, decimal, float, text, string, binary, datetime, currency
- [ ] Precision/scale backward compatible (legacy precision = scale)
- [ ] title_field supports format engine templates
- [ ] display_field works on many2one fields
- [ ] currency_field works for dynamic currency
- [ ] Table naming plural works (project + per-model)
- [ ] Duplicate model detection works (same module = error)
- [ ] jinzhu/inflection replaces hand-rolled pluralize
- [ ] All existing tests pass
- [ ] New tests cover all additions
- [ ] `go test ./...` passes
