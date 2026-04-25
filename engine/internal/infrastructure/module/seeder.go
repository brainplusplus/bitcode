package module

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/bitcode-framework/bitcode/pkg/security"
	"gorm.io/gorm"
)

type TableNameResolver interface {
	TableName(modelName string) string
}

func SeedModule(db *gorm.DB, modulePath string, dataPatterns []string, resolver TableNameResolver) error {
	for _, pattern := range dataPatterns {
		fullPattern := filepath.Join(modulePath, pattern)
		matches, err := filepath.Glob(fullPattern)
		if err != nil {
			continue
		}
		for _, match := range matches {
			if err := seedFile(db, match, resolver); err != nil {
				return fmt.Errorf("failed to seed %s: %w", match, err)
			}
		}
	}
	return nil
}

func resolveSeederTable(modelName string, resolver TableNameResolver) string {
	if resolver != nil {
		return resolver.TableName(modelName)
	}
	return modelName
}

func seedFile(db *gorm.DB, path string, resolver TableNameResolver) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		var arr []map[string]any
		if err2 := json.Unmarshal(data, &arr); err2 == nil {
			base := filepath.Base(path)
			modelName := base[:len(base)-len(filepath.Ext(base))]
			modelName = strings.TrimPrefix(modelName, "default_")
			table := resolveSeederTable(modelName, resolver)
			return seedRecords(db, table, arr)
		}
		return fmt.Errorf("invalid seed JSON: %w", err)
	}

	for modelName, recordsRaw := range raw {
		var records []map[string]any
		if err := json.Unmarshal(recordsRaw, &records); err != nil {
			continue
		}

		table := resolveSeederTable(modelName, resolver)

		if err := seedRecords(db, table, records); err != nil {
			return err
		}
	}
	return nil
}

func seedRecords(db *gorm.DB, table string, records []map[string]any) error {
	for _, record := range records {
		if _, hasID := record["id"]; !hasID {
			if !isAutoIncrementTable(table) {
				record["id"] = uuid.New().String()
			}
		}

		if pw, ok := record["password"].(string); ok {
			hash, err := security.HashPassword(pw)
			if err == nil {
				record["password_hash"] = hash
			}
			delete(record, "password")
		}

		var count int64
		checkField := findUniqueField(record)
		if checkField != "" {
			db.Table(table).Where(fmt.Sprintf("%s = ?", checkField), record[checkField]).Count(&count)
			if count > 0 {
				continue
			}
		}

		if err := db.Table(table).Create(&record).Error; err != nil {
			continue
		}
		log.Printf("[SEED] %s: %v", table, record[checkField])
	}
	return nil
}

func isAutoIncrementTable(table string) bool {
	return false
}

func findUniqueField(record map[string]any) string {
	for _, field := range []string{"username", "name", "email", "employee_id", "code", "key", "title"} {
		if _, ok := record[field]; ok {
			return field
		}
	}
	return ""
}
