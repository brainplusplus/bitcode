# Phase 6C Implementation Plan: Engine Enhancements

**Estimated effort**: 10-14 days
**Prerequisites**: Phase 6A (field modifiers, display_field), Phase 1 (bridge API for process data_source)
**Test command**: `go test ./...`

---

## Implementation Order

```
Stream 1: Array-Backed Models — Core (Day 1-3)
  ↓
Stream 2: Array-Backed Models — File Parsers (Day 3-4)
  ↓
Stream 3: Array-Backed Models — Process Source & Refresh (Day 4-5)
  ↓
Stream 4: Array-Backed Models — sync_source Write-Back (Day 5-6)
  ↓
Stream 5: View Modifiers (Day 6-7)
  ↓
Stream 6: Metadata API (Day 7-9)
  ↓
Stream 7: Embedded View filter_by + Process Data Source (Day 9-10)
  ↓
Stream 8: Eager Loading Fixes (Day 10-12)
  ↓
Stream 9: Project Config & Tests (Day 12-14)
```

---

## Stream 1: Array Model Core

### 1.1 Parser Changes

**File**: `internal/compiler/parser/model.go`

Add to ModelDefinition:
```go
Source    string            `json:"source,omitempty"`     // "db" | "array" | "process"
Writable *bool             `json:"writable,omitempty"`
Rows     []map[string]any  `json:"rows,omitempty"`
RowsFile string            `json:"rows_file,omitempty"`
SyncSource bool            `json:"sync_source,omitempty"`
Refresh    string          `json:"refresh,omitempty"`
Process    string          `json:"process,omitempty"`     // for source: "process"
Script     *ScriptRef      `json:"script,omitempty"`      // for source: "process"
```

Validation:
- `source: "array"` without `rows` or `rows_file` → error
- `rows` + `rows_file` both set → error
- `sync_source: true` + `writable: false` → error
- `sync_source: true` + no `rows_file` → error
- `sync_source: true` + `source: "process"` → error

### 1.2 Migration Integration

**File**: `internal/app.go` (or new `internal/infrastructure/persistence/array_sync.go`)

After `MigrateModel()`, add array sync logic:
```go
func syncArrayModel(db *gorm.DB, model *ModelDefinition, tableName string) error {
    rows := model.Rows
    if model.RowsFile != "" {
        rows = loadRowsFromFile(model.RowsFile, model.ModulePath)
    }
    
    isWritable := model.Writable != nil && *model.Writable
    
    if isWritable {
        // Seed only if empty
        var count int64
        db.Table(tableName).Count(&count)
        if count > 0 { return nil }
        return bulkInsert(db, tableName, rows)
    }
    
    // Read-only: DELETE all + re-INSERT
    db.Exec("DELETE FROM " + tableName)
    return bulkInsert(db, tableName, rows)
}
```

### 1.3 API Write Block

**File**: `internal/presentation/api/router.go`

For read-only array models, block POST/PUT/DELETE:
```go
if modelDef.Source == "array" && !modelDef.IsWritable() {
    return c.Status(405).JSON(fiber.Map{"error": "model is read-only (source: array)"})
}
```

---

## Stream 2: File Parsers

**File**: `internal/infrastructure/persistence/array_parser.go` (NEW)

```go
func loadRowsFromFile(filePath string, basePath string) ([]map[string]any, error) {
    fullPath := filepath.Join(basePath, filePath)
    ext := filepath.Ext(fullPath)
    
    switch ext {
    case ".json":
        return parseJSONRows(fullPath)
    case ".csv":
        return parseCSVRows(fullPath)
    case ".xlsx":
        return parseXLSXRows(fullPath)
    case ".xml":
        return parseXMLRows(fullPath)
    default:
        return nil, fmt.Errorf("unsupported file format: %s", ext)
    }
}
```

Dependencies:
- JSON: `encoding/json` (stdlib)
- CSV: `encoding/csv` (stdlib)
- XLSX: `github.com/xuri/excelize/v2`
- XML: `encoding/xml` (stdlib)

---

## Stream 3: Process Source & Refresh

### 3.1 Process Source

Execute process/script to get rows, then sync to table. Same as array sync but data comes from process execution.

### 3.2 Refresh Scheduler

**File**: `internal/runtime/refresh/scheduler.go` (NEW)

```go
type RefreshScheduler struct {
    jobs map[string]*RefreshJob
}

type RefreshJob struct {
    ModelName string
    Interval  time.Duration
    Ticker    *time.Ticker
    SyncFn    func() error
}
```

Parse `refresh` config: `"5m"` → `time.ParseDuration("5m")`

### 3.3 Manual Refresh Endpoint

**File**: `internal/presentation/api/meta_handler.go`

```
POST /api/v1/_meta/models/:name/refresh
```

---

## Stream 4: sync_source Write-Back

**File**: `internal/infrastructure/persistence/array_writer.go` (NEW)

On every write operation (create/update/delete) for writable array models with `sync_source: true`:
1. Query all records from DB
2. Write to file in original format (JSON/CSV/XLSX/XML)

```go
func writeRowsToFile(filePath string, rows []map[string]any, fields map[string]FieldDefinition) error {
    ext := filepath.Ext(filePath)
    switch ext {
    case ".json":
        return writeJSONRows(filePath, rows)
    case ".csv":
        return writeCSVRows(filePath, rows, fields)
    case ".xlsx":
        return writeXLSXRows(filePath, rows, fields)
    case ".xml":
        return writeXMLRows(filePath, rows, fields)
    }
    return nil
}
```

---

## Stream 5: View Modifiers

**File**: `internal/compiler/parser/view.go`

Add to LayoutRow:
```go
VisibleIf  string `json:"visible_if,omitempty"`
DisabledIf string `json:"disabled_if,omitempty"`
ReadonlyIf string `json:"readonly_if,omitempty"`
CSSClass   string `json:"css_class,omitempty"`
HelpText   string `json:"help_text,omitempty"`
```

**File**: `internal/presentation/view/component_compiler.go`

Render `data-visible-if`, `data-disabled-if` attributes on field containers.

---

## Stream 6: Metadata API

**File**: `internal/presentation/api/meta_handler.go` (NEW)

10 endpoints:
- `GET /api/v1/_meta/models` — list models
- `GET /api/v1/_meta/models/:name` — model detail
- `GET /api/v1/_meta/models/:name/fields` — fields only
- `GET /api/v1/_meta/views` — list views
- `GET /api/v1/_meta/views/:name` — view detail
- `GET /api/v1/_meta/modules` — list modules
- `GET /api/v1/_meta/modules/:name` — module detail
- `GET /api/v1/_meta/processes` — list processes
- `GET /api/v1/_meta/processes/:name` — process detail
- `GET /api/v1/_meta/field-types` — field type catalog

Plus bridge: `bitcode.meta.models()`, `bitcode.meta.model("name")`, etc.

---

## Stream 7: Embedded View + Process Data Source

### 7.1 filter_by

**File**: `internal/compiler/parser/view.go` — add `FilterBy` to TabDefinition
**File**: `internal/presentation/view/renderer.go` — apply filter when rendering embedded views

### 7.2 Process Data Source

**File**: `internal/compiler/parser/view.go` — add `Process` to DataSourceDefinition
**File**: `internal/presentation/view/renderer.go` — execute process in `resolveDataSource()`

---

## Stream 8: Eager Loading Fixes

**File**: `internal/infrastructure/persistence/repository.go`

- Apply `WithClause.Conditions` in ALL relation loaders
- Apply `WithClause.Select`, `OrderBy`, `Limit` consistently
- Implement `WithClause.Nested` — recursive loading (max depth from config)

---

## Stream 9: Project Config

**File**: `internal/config.go`

Add all new config keys with Viper bindings:
- `database.table_naming`
- `locale.currency`, `locale.timezone`, `locale.number_format`, `locale.thousand_separator`, `locale.decimal_separator`
- `meta_api.enabled`, `meta_api.auth`, `meta_api.admin_only`
- `eager_loading.max_depth`

## Definition of Done

- [ ] Array models (source: "array") load data from JSON/CSV/XLSX/XML into main DB
- [ ] Writable mode: seed only if empty, full CRUD after
- [ ] Read-only mode: sync from source on startup, block writes
- [ ] Process source: execute process, populate table
- [ ] Refresh scheduler works (interval + manual)
- [ ] sync_source: write-back to file on DB change
- [ ] View modifiers (visible_if, disabled_if) render correctly
- [ ] Metadata API: all 10 endpoints work
- [ ] Embedded view filter_by works
- [ ] Process data source works in views
- [ ] Eager loading: conditions, nested loading work
- [ ] All config keys with Viper bindings
- [ ] All tests pass
