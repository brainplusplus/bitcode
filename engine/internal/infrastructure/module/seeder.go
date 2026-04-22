package module

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/bitcode-engine/engine/pkg/security"
	"gorm.io/gorm"
)

func SeedModule(db *gorm.DB, modulePath string, dataPatterns []string) error {
	for _, pattern := range dataPatterns {
		fullPattern := filepath.Join(modulePath, pattern)
		matches, err := filepath.Glob(fullPattern)
		if err != nil {
			continue
		}
		for _, match := range matches {
			if err := seedFile(db, match); err != nil {
				return fmt.Errorf("failed to seed %s: %w", match, err)
			}
		}
	}
	return nil
}

func seedFile(db *gorm.DB, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		var arr []map[string]any
		if err2 := json.Unmarshal(data, &arr); err2 == nil {
			base := filepath.Base(path)
			table := base[:len(base)-len(filepath.Ext(base))]
			table = strings.TrimPrefix(table, "default_")
			if table[len(table)-1] != 's' {
				table += "s"
			}
			return seedRecords(db, table, arr)
		}
		return fmt.Errorf("invalid seed JSON: %w", err)
	}

	for tableName, recordsRaw := range raw {
		var records []map[string]any
		if err := json.Unmarshal(recordsRaw, &records); err != nil {
			continue
		}

		table := tableName
		if table[len(table)-1] != 's' {
			table += "s"
		}

		if err := seedRecords(db, table, records); err != nil {
			return err
		}
	}
	return nil
}

func seedRecords(db *gorm.DB, table string, records []map[string]any) error {
	for _, record := range records {
		if _, hasID := record["id"]; !hasID {
			record["id"] = uuid.New().String()
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

func findUniqueField(record map[string]any) string {
	for _, field := range []string{"username", "name", "email", "employee_id", "code", "key", "title"} {
		if _, ok := record[field]; ok {
			return field
		}
	}
	return ""
}
