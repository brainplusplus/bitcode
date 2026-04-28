package persistence

import (
	"fmt"

	"github.com/bitcode-framework/bitcode/internal/compiler/parser"
	"gorm.io/gorm"
)

type DBDialect string

const (
	DialectSQLite   DBDialect = "sqlite"
	DialectPostgres DBDialect = "postgres"
	DialectMySQL    DBDialect = "mysql"
)

func DetectDialect(db *gorm.DB) DBDialect {
	name := db.Dialector.Name()
	switch name {
	case "postgres":
		return DialectPostgres
	case "mysql":
		return DialectMySQL
	default:
		return DialectSQLite
	}
}

type TableNameResolver interface {
	TableName(modelName string) string
}

func MigrateModel(db *gorm.DB, model *parser.ModelDefinition, resolver TableNameResolver, tenantEnabled ...bool) error {
	dialect := DetectDialect(db)
	isTenantEnabled := len(tenantEnabled) > 0 && tenantEnabled[0]
	columns := buildColumns(model, dialect, isTenantEnabled)
	tableName := model.Name
	if resolver != nil {
		tableName = resolver.TableName(model.Name)
	}

	if !db.Migrator().HasTable(tableName) {
		sql := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n%s\n)", tableName, columns)
		if err := db.Exec(sql).Error; err != nil {
			return fmt.Errorf("failed to create table %s: %w", tableName, err)
		}
	} else if isTenantEnabled && model.IsTenantScoped() {
		if !db.Migrator().HasColumn(&struct{}{}, "tenant_id") {
			hasCol := false
			switch dialect {
			case DialectPostgres:
				var count int64
				db.Raw("SELECT COUNT(*) FROM information_schema.columns WHERE table_name=? AND column_name='tenant_id'", tableName).Scan(&count)
				hasCol = count > 0
			case DialectMySQL:
				var count int64
				db.Raw("SELECT COUNT(*) FROM information_schema.columns WHERE table_name=? AND column_name='tenant_id'", tableName).Scan(&count)
				hasCol = count > 0
			default:
				var count int64
				db.Raw("SELECT COUNT(*) FROM pragma_table_info(?) WHERE name='tenant_id'", tableName).Scan(&count)
				hasCol = count > 0
			}
			if !hasCol {
				var alterSQL string
				switch dialect {
				case DialectPostgres:
					alterSQL = fmt.Sprintf("ALTER TABLE %s ADD COLUMN tenant_id VARCHAR(100) NOT NULL DEFAULT ''", tableName)
				case DialectMySQL:
					alterSQL = fmt.Sprintf("ALTER TABLE %s ADD COLUMN tenant_id VARCHAR(100) NOT NULL DEFAULT ''", tableName)
				default:
					alterSQL = fmt.Sprintf("ALTER TABLE %s ADD COLUMN tenant_id TEXT NOT NULL DEFAULT ''", tableName)
				}
				db.Exec(alterSQL)
			}
		}
	}

	for _, idx := range model.Indexes {
		idxName := fmt.Sprintf("idx_%s_%s", model.Name, joinStrings(idx, "_"))
		idxCols := joinStrings(idx, ", ")
		sql := fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s (%s)", idxName, tableName, idxCols)
		if err := db.Exec(sql).Error; err != nil {
			return fmt.Errorf("failed to create index %s: %w", idxName, err)
		}
	}

	if isTenantEnabled && model.IsTenantScoped() {
		idxName := fmt.Sprintf("idx_%s_tenant_id", model.Name)
		sql := fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s (tenant_id)", idxName, tableName)
		db.Exec(sql)
	}

	for fieldName, field := range model.Fields {
		if field.Type == parser.FieldMany2Many {
			if err := createJunctionTable(db, model.Name, fieldName, field.Model, dialect, resolver); err != nil {
				return err
			}
		}
	}

	return nil
}

func buildColumns(model *parser.ModelDefinition, dialect DBDialect, tenantEnabled ...bool) string {
	var cols string
	pkFieldName := ""
	isCompositeNoSurrogate := false

	if model.PrimaryKey != nil {
		switch model.PrimaryKey.Strategy {
		case parser.PKAutoIncrement:
			switch dialect {
			case DialectPostgres:
				cols = "  id BIGSERIAL PRIMARY KEY,\n"
			case DialectMySQL:
				cols = "  id BIGINT AUTO_INCREMENT PRIMARY KEY,\n"
			default:
				cols = "  id INTEGER PRIMARY KEY AUTOINCREMENT,\n"
			}
		case parser.PKNaturalKey:
			pkFieldName = model.PrimaryKey.Field
		case parser.PKNamingSeries:
			pkFieldName = model.PrimaryKey.Field
		case parser.PKManual:
			pkFieldName = model.PrimaryKey.Field
		case parser.PKComposite:
			if !model.PrimaryKey.IsSurrogate() {
				isCompositeNoSurrogate = true
			}
		}
	}

	if cols == "" && pkFieldName == "" && !isCompositeNoSurrogate {
		switch dialect {
		case DialectPostgres:
			cols = "  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),\n"
		case DialectMySQL:
			cols = "  id CHAR(36) PRIMARY KEY,\n"
		default:
			cols = "  id TEXT PRIMARY KEY,\n"
		}
	}

	if model.IsTimestamps() {
		switch dialect {
		case DialectPostgres:
			cols += "  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),\n"
			cols += "  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),\n"
		case DialectMySQL:
			cols += "  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,\n"
			cols += "  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,\n"
		default:
			cols += "  created_at DATETIME NOT NULL DEFAULT (datetime('now')),\n"
			cols += "  updated_at DATETIME NOT NULL DEFAULT (datetime('now')),\n"
		}
	}

	if model.IsTimestampsBy() {
		switch dialect {
		case DialectPostgres:
			cols += "  created_by UUID,\n"
			cols += "  updated_by UUID,\n"
		case DialectMySQL:
			cols += "  created_by CHAR(36),\n"
			cols += "  updated_by CHAR(36),\n"
		default:
			cols += "  created_by TEXT,\n"
			cols += "  updated_by TEXT,\n"
		}
	}

	switch dialect {
	case DialectPostgres:
		cols += "  active BOOLEAN NOT NULL DEFAULT TRUE"
	case DialectMySQL:
		cols += "  active BOOLEAN NOT NULL DEFAULT TRUE"
	default:
		cols += "  active INTEGER NOT NULL DEFAULT 1"
	}

	if model.IsVersion() {
		cols += ",\n  version INTEGER NOT NULL DEFAULT 1"
	}

	if model.IsSoftDeletes() {
		switch dialect {
		case DialectPostgres:
			cols += ",\n  deleted_at TIMESTAMPTZ"
		case DialectMySQL:
			cols += ",\n  deleted_at DATETIME"
		default:
			cols += ",\n  deleted_at DATETIME"
		}
	}

	if model.IsSoftDeletesBy() {
		switch dialect {
		case DialectPostgres:
			cols += ",\n  deleted_by UUID"
		case DialectMySQL:
			cols += ",\n  deleted_by CHAR(36)"
		default:
			cols += ",\n  deleted_by TEXT"
		}
	}

	isTenantEnabled := len(tenantEnabled) > 0 && tenantEnabled[0]
	if isTenantEnabled && model.IsTenantScoped() {
		switch dialect {
		case DialectPostgres:
			cols += ",\n  tenant_id VARCHAR(100) NOT NULL DEFAULT ''"
		case DialectMySQL:
			cols += ",\n  tenant_id VARCHAR(100) NOT NULL DEFAULT ''"
		default:
			cols += ",\n  tenant_id TEXT NOT NULL DEFAULT ''"
		}
	}

	for name, field := range model.Fields {
		sqlType := fieldTypeToSQL(field, dialect)
		if sqlType == "" {
			continue
		}
		col := fmt.Sprintf(",\n  %s %s", name, sqlType)

		if name == pkFieldName {
			col += " PRIMARY KEY"
		} else {
			if field.Required {
				col += " NOT NULL"
			}
			if field.Unique {
				col += " UNIQUE"
			}
		}
		if field.Default != nil {
			col += fmt.Sprintf(" DEFAULT %s", formatDefault(field.Default, dialect))
		}
		cols += col
	}

	if isCompositeNoSurrogate && len(model.PrimaryKey.Fields) > 0 {
		cols += fmt.Sprintf(",\n  PRIMARY KEY (%s)", joinStrings(model.PrimaryKey.Fields, ", "))
	}

	if model.PrimaryKey != nil && model.PrimaryKey.Strategy == parser.PKComposite && model.PrimaryKey.IsSurrogate() {
		cols += fmt.Sprintf(",\n  UNIQUE (%s)", joinStrings(model.PrimaryKey.Fields, ", "))
	}

	if model.OfflineModule {
		cols += OfflineColumns(dialect)
		cols += OfflineUUIDColumn(model.PrimaryKey, dialect)
	}

	return cols
}

func resolveDecimalSQL(field parser.FieldDefinition, dialect DBDialect) string {
	if dialect == DialectSQLite {
		return "REAL"
	}
	if field.Storage == "numeric" {
		if dialect == DialectPostgres {
			return "NUMERIC"
		}
		return "DECIMAL(65,30)"
	}
	if field.Storage == "double" {
		if dialect == DialectPostgres {
			return "DOUBLE PRECISION"
		}
		return "DOUBLE"
	}
	p, s := 18, 2
	if field.Scale > 0 {
		s = field.Scale
		if field.Precision > 0 {
			p = field.Precision
		}
	} else if field.Precision > 0 {
		s = field.Precision
	}
	if dialect == DialectPostgres {
		return fmt.Sprintf("NUMERIC(%d,%d)", p, s)
	}
	return fmt.Sprintf("DECIMAL(%d,%d)", p, s)
}

func varcharOrText(dialect DBDialect, length int) string {
	if dialect == DialectSQLite {
		return "TEXT"
	}
	if length > 0 {
		return fmt.Sprintf("VARCHAR(%d)", length)
	}
	return "VARCHAR(255)"
}

func fieldTypeToSQL(field parser.FieldDefinition, dialect DBDialect) string {
	switch field.Type {

	// --- Core types ---

	case parser.FieldString:
		if field.Storage == "char" && dialect != DialectSQLite {
			if field.Max > 0 {
				return fmt.Sprintf("CHAR(%d)", field.Max)
			}
			return "CHAR(255)"
		}
		return varcharOrText(dialect, field.Max)

	case parser.FieldText:
		if dialect == DialectMySQL {
			switch field.Storage {
			case "mediumtext":
				return "MEDIUMTEXT"
			case "longtext":
				return "LONGTEXT"
			}
		}
		return "TEXT"

	case parser.FieldInteger:
		switch field.Storage {
		case "smallint":
			if dialect == DialectSQLite {
				return "INTEGER"
			}
			return "SMALLINT"
		case "bigint":
			if dialect == DialectSQLite {
				return "INTEGER"
			}
			return "BIGINT"
		}
		return "INTEGER"

	case parser.FieldDecimal:
		return resolveDecimalSQL(field, dialect)

	case parser.FieldFloat:
		if dialect == DialectSQLite {
			return "REAL"
		}
		if dialect == DialectPostgres {
			return "DOUBLE PRECISION"
		}
		return "DOUBLE"

	case parser.FieldBoolean, parser.FieldToggle:
		if dialect == DialectSQLite {
			return "INTEGER"
		}
		return "BOOLEAN"

	case parser.FieldDate:
		if dialect == DialectSQLite {
			return "TEXT"
		}
		return "DATE"

	case parser.FieldDatetime:
		if dialect == DialectSQLite {
			return "TEXT"
		}
		if field.Storage == "naive" {
			if dialect == DialectPostgres {
				return "TIMESTAMP"
			}
			return "DATETIME"
		}
		if dialect == DialectPostgres {
			return "TIMESTAMPTZ"
		}
		return "DATETIME"

	case parser.FieldTime:
		if dialect == DialectSQLite {
			return "TEXT"
		}
		return "TIME"

	case parser.FieldSelection, parser.FieldRadio:
		if dialect == DialectSQLite {
			return "TEXT"
		}
		return "VARCHAR(50)"

	// --- String-family semantic types ---

	case parser.FieldEmail, parser.FieldPassword, parser.FieldBarcode, parser.FieldDynamicLink:
		return varcharOrText(dialect, 255)

	case parser.FieldSmallText:
		return varcharOrText(dialect, 500)

	case parser.FieldColor:
		return varcharOrText(dialect, 7)

	case parser.FieldUUID:
		if dialect == DialectPostgres {
			return "UUID"
		}
		if dialect == DialectMySQL {
			return "CHAR(36)"
		}
		return "TEXT"

	case parser.FieldIP:
		return varcharOrText(dialect, 45)

	case parser.FieldIPv6:
		return varcharOrText(dialect, 45)

	// --- Text-family semantic types ---

	case parser.FieldRichText, parser.FieldMarkdown, parser.FieldHTML, parser.FieldCode, parser.FieldSignature:
		return "TEXT"

	// --- Number-family semantic types ---

	case parser.FieldCurrency:
		return resolveDecimalSQL(field, dialect)

	case parser.FieldPercent:
		if dialect == DialectSQLite {
			return "REAL"
		}
		if dialect == DialectPostgres {
			return "NUMERIC(5,2)"
		}
		return "DECIMAL(5,2)"

	case parser.FieldRating:
		if dialect == DialectSQLite {
			return "INTEGER"
		}
		return "SMALLINT"

	case parser.FieldYear:
		if dialect == DialectSQLite {
			return "INTEGER"
		}
		return "SMALLINT"

	case parser.FieldDuration:
		return "INTEGER"

	// --- Relation types ---

	case parser.FieldMany2One:
		if dialect == DialectSQLite {
			return "TEXT"
		}
		if dialect == DialectMySQL {
			return "CHAR(36)"
		}
		return "UUID"

	case parser.FieldOne2Many, parser.FieldMany2Many, parser.FieldComputed:
		return ""

	// --- JSON types ---

	case parser.FieldJSON, parser.FieldJSONObject, parser.FieldJSONArray:
		if dialect == DialectPostgres {
			return "JSONB"
		}
		if dialect == DialectMySQL {
			return "JSON"
		}
		return "TEXT"

	case parser.FieldGeolocation:
		if dialect == DialectPostgres {
			return "JSONB"
		}
		if dialect == DialectMySQL {
			return "JSON"
		}
		return "TEXT"

	// --- File types ---

	case parser.FieldFile, parser.FieldImage:
		return varcharOrText(dialect, 500)

	// --- Special types ---

	case parser.FieldVector:
		if dialect == DialectPostgres && field.Dimensions > 0 {
			return fmt.Sprintf("vector(%d)", field.Dimensions)
		}
		if dialect == DialectMySQL {
			return "JSON"
		}
		return "TEXT"

	case parser.FieldBinary:
		if dialect == DialectPostgres {
			return "BYTEA"
		}
		if dialect == DialectMySQL {
			switch field.Storage {
			case "mediumblob":
				return "MEDIUMBLOB"
			default:
				return "LONGBLOB"
			}
		}
		return "BLOB"
	}

	return "TEXT"
}

func formatDefault(val any, dialect DBDialect) string {
	switch v := val.(type) {
	case string:
		if v == "now" {
			if dialect == DialectSQLite {
				return "(datetime('now'))"
			}
			if dialect == DialectMySQL {
				return "CURRENT_TIMESTAMP"
			}
			return "NOW()"
		}
		return fmt.Sprintf("'%s'", v)
	case bool:
		if dialect == DialectSQLite {
			if v {
				return "1"
			}
			return "0"
		}
		if v {
			return "TRUE"
		}
		return "FALSE"
	case float64:
		return fmt.Sprintf("%v", v)
	default:
		return fmt.Sprintf("'%v'", v)
	}
}

func joinStrings(strs []string, sep string) string {
	result := ""
	for i, s := range strs {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}

func createJunctionTable(db *gorm.DB, model1 string, fieldName string, model2 string, dialect DBDialect, resolver TableNameResolver) error {
	baseTable := model1
	if resolver != nil {
		baseTable = resolver.TableName(model1)
	}
	tableName := baseTable + "_" + fieldName
	if db.Migrator().HasTable(tableName) {
		return nil
	}

	var idType, fkType string
	switch dialect {
	case DialectPostgres:
		idType = "UUID PRIMARY KEY DEFAULT gen_random_uuid()"
		fkType = "UUID NOT NULL"
	case DialectMySQL:
		idType = "CHAR(36) PRIMARY KEY"
		fkType = "CHAR(36) NOT NULL"
	default:
		idType = "TEXT PRIMARY KEY"
		fkType = "TEXT NOT NULL"
	}

	sql := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
  id %s,
  %s_id %s,
  %s_id %s
)`, tableName, idType, model1, fkType, model2, fkType)

	if err := db.Exec(sql).Error; err != nil {
		return fmt.Errorf("failed to create junction table %s: %w", tableName, err)
	}

	idx1 := fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_%s_id ON %s (%s_id)", tableName, model1, tableName, model1)
	idx2 := fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_%s_id ON %s (%s_id)", tableName, model2, tableName, model2)
	db.Exec(idx1)
	db.Exec(idx2)

	return nil
}

func MergeInheritedFields(parent *parser.ModelDefinition, child *parser.ModelDefinition) *parser.ModelDefinition {
	merged := &parser.ModelDefinition{
		Name:         child.Name,
		Module:       child.Module,
		Label:        child.Label,
		Inherit:      child.Inherit,
		Fields:       make(map[string]parser.FieldDefinition),
		RecordRules:  parent.RecordRules,
		Indexes:      parent.Indexes,
		Version:       parent.Version,
		Timestamps:    parent.Timestamps,
		TimestampsBy:  parent.TimestampsBy,
		SoftDeletes:   parent.SoftDeletes,
		SoftDeletesBy: parent.SoftDeletesBy,
	}

	for name, field := range parent.Fields {
		merged.Fields[name] = field
	}
	for name, field := range child.Fields {
		merged.Fields[name] = field
	}

	if len(child.RecordRules) > 0 {
		merged.RecordRules = child.RecordRules
	}
	if len(child.Indexes) > 0 {
		merged.Indexes = append(merged.Indexes, child.Indexes...)
	}

	if merged.Label == "" {
		merged.Label = parent.Label
	}

	if child.Version != nil {
		merged.Version = child.Version
	}
	if child.Timestamps != nil {
		merged.Timestamps = child.Timestamps
	}
	if child.TimestampsBy != nil {
		merged.TimestampsBy = child.TimestampsBy
	}
	if child.SoftDeletes != nil {
		merged.SoftDeletes = child.SoftDeletes
	}
	if child.SoftDeletesBy != nil {
		merged.SoftDeletesBy = child.SoftDeletesBy
	}

	merged.Events = mergeEvents(parent.Events, child.Events)
	merged.Validators = mergeModelValidators(parent.Validators, child.Validators)

	if child.Sanitize != nil {
		merged.Sanitize = child.Sanitize
	} else {
		merged.Sanitize = parent.Sanitize
	}

	if child.ModulePath != "" {
		merged.ModulePath = child.ModulePath
	} else {
		merged.ModulePath = parent.ModulePath
	}

	return merged
}

func mergeEvents(parent *parser.EventsDefinition, child *parser.EventsDefinition) *parser.EventsDefinition {
	if parent == nil && child == nil {
		return nil
	}
	if parent == nil {
		return child
	}
	if child == nil {
		return parent
	}

	merged := &parser.EventsDefinition{
		BeforeValidate:   append(parent.BeforeValidate, child.BeforeValidate...),
		AfterValidate:    append(parent.AfterValidate, child.AfterValidate...),
		BeforeCreate:     append(parent.BeforeCreate, child.BeforeCreate...),
		AfterCreate:      append(parent.AfterCreate, child.AfterCreate...),
		BeforeUpdate:     append(parent.BeforeUpdate, child.BeforeUpdate...),
		AfterUpdate:      append(parent.AfterUpdate, child.AfterUpdate...),
		BeforeDelete:     append(parent.BeforeDelete, child.BeforeDelete...),
		AfterDelete:      append(parent.AfterDelete, child.AfterDelete...),
		BeforeSave:       append(parent.BeforeSave, child.BeforeSave...),
		AfterSave:        append(parent.AfterSave, child.AfterSave...),
		BeforeSoftDelete: append(parent.BeforeSoftDelete, child.BeforeSoftDelete...),
		AfterSoftDelete:  append(parent.AfterSoftDelete, child.AfterSoftDelete...),
		BeforeHardDelete: append(parent.BeforeHardDelete, child.BeforeHardDelete...),
		AfterHardDelete:  append(parent.AfterHardDelete, child.AfterHardDelete...),
		BeforeRestore:    append(parent.BeforeRestore, child.BeforeRestore...),
		AfterRestore:     append(parent.AfterRestore, child.AfterRestore...),
	}

	if len(parent.OnChange) > 0 || len(child.OnChange) > 0 {
		merged.OnChange = make(map[string][]parser.EventHandler)
		for field, handlers := range parent.OnChange {
			merged.OnChange[field] = append(merged.OnChange[field], handlers...)
		}
		for field, handlers := range child.OnChange {
			merged.OnChange[field] = append(merged.OnChange[field], handlers...)
		}
	}

	return merged
}

func mergeModelValidators(parent []parser.ModelValidator, child []parser.ModelValidator) []parser.ModelValidator {
	if len(parent) == 0 {
		return child
	}
	if len(child) == 0 {
		return parent
	}
	return append(parent, child...)
}
