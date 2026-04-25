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

func MigrateModel(db *gorm.DB, model *parser.ModelDefinition, resolver TableNameResolver) error {
	dialect := DetectDialect(db)
	columns := buildColumns(model, dialect)
	tableName := model.Name
	if resolver != nil {
		tableName = resolver.TableName(model.Name)
	}

	if !db.Migrator().HasTable(tableName) {
		sql := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n%s\n)", tableName, columns)
		if err := db.Exec(sql).Error; err != nil {
			return fmt.Errorf("failed to create table %s: %w", tableName, err)
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

	for fieldName, field := range model.Fields {
		if field.Type == parser.FieldMany2Many {
			if err := createJunctionTable(db, model.Name, fieldName, field.Model, dialect, resolver); err != nil {
				return err
			}
		}
	}

	return nil
}

func buildColumns(model *parser.ModelDefinition, dialect DBDialect) string {
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

	switch dialect {
	case DialectPostgres:
		cols += "  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),\n"
		cols += "  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),\n"
		cols += "  created_by UUID,\n"
		cols += "  updated_by UUID,\n"
		cols += "  active BOOLEAN NOT NULL DEFAULT TRUE"
	case DialectMySQL:
		cols += "  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,\n"
		cols += "  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,\n"
		cols += "  created_by CHAR(36),\n"
		cols += "  updated_by CHAR(36),\n"
		cols += "  active BOOLEAN NOT NULL DEFAULT TRUE"
	default:
		cols += "  created_at DATETIME NOT NULL DEFAULT (datetime('now')),\n"
		cols += "  updated_at DATETIME NOT NULL DEFAULT (datetime('now')),\n"
		cols += "  created_by TEXT,\n"
		cols += "  updated_by TEXT,\n"
		cols += "  active INTEGER NOT NULL DEFAULT 1"
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

	return cols
}

func fieldTypeToSQL(field parser.FieldDefinition, dialect DBDialect) string {
	switch field.Type {
	case parser.FieldString:
		if dialect == DialectSQLite {
			return "TEXT"
		}
		if field.Max > 0 {
			return fmt.Sprintf("VARCHAR(%d)", field.Max)
		}
		return "VARCHAR(255)"
	case parser.FieldText:
		return "TEXT"
	case parser.FieldInteger:
		return "INTEGER"
	case parser.FieldDecimal:
		if dialect == DialectSQLite {
			return "REAL"
		}
		if field.Precision > 0 {
			return fmt.Sprintf("DECIMAL(18,%d)", field.Precision)
		}
		return "DECIMAL(18,2)"
	case parser.FieldBoolean:
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
		if dialect == DialectPostgres {
			return "TIMESTAMPTZ"
		}
		return "DATETIME"
	case parser.FieldSelection:
		if dialect == DialectSQLite {
			return "TEXT"
		}
		return "VARCHAR(50)"
	case parser.FieldEmail:
		if dialect == DialectSQLite {
			return "TEXT"
		}
		return "VARCHAR(255)"
	case parser.FieldMany2One:
		if dialect == DialectSQLite {
			return "TEXT"
		}
		if dialect == DialectMySQL {
			return "CHAR(36)"
		}
		return "UUID"
	case parser.FieldJSON:
		if dialect == DialectPostgres {
			return "JSONB"
		}
		if dialect == DialectMySQL {
			return "JSON"
		}
		return "TEXT"
	case parser.FieldFile:
		if dialect == DialectSQLite {
			return "TEXT"
		}
		return "VARCHAR(500)"
	case parser.FieldOne2Many, parser.FieldMany2Many, parser.FieldComputed:
		return ""
	default:
		return "TEXT"
	}
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
		Name:        child.Name,
		Module:      child.Module,
		Label:       child.Label,
		Inherit:     child.Inherit,
		Fields:      make(map[string]parser.FieldDefinition),
		RecordRules: parent.RecordRules,
		Indexes:     parent.Indexes,
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

	return merged
}
