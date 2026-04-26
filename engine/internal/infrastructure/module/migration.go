package module

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/bitcode-framework/bitcode/internal/compiler/parser"
	"github.com/bitcode-framework/bitcode/internal/infrastructure/persistence"
	"github.com/bitcode-framework/bitcode/pkg/security"
	"gorm.io/gorm"
)

type ProcessRunner interface {
	RunProcess(ctx context.Context, processName string, input map[string]any) (any, error)
}

type ScriptRunner interface {
	Run(ctx context.Context, scriptPath string, params map[string]any) (any, error)
}

type MigrationEngine struct {
	db            *gorm.DB
	tracker       *persistence.MigrationTracker
	resolver      TableNameResolver
	processRunner ProcessRunner
	scriptRunner  ScriptRunner
}

func NewMigrationEngine(db *gorm.DB, resolver TableNameResolver) *MigrationEngine {
	return &MigrationEngine{
		db:       db,
		tracker:  persistence.NewMigrationTracker(db),
		resolver: resolver,
	}
}

func (e *MigrationEngine) SetProcessRunner(runner ProcessRunner) {
	e.processRunner = runner
}

func (e *MigrationEngine) SetScriptRunner(runner ScriptRunner) {
	e.scriptRunner = runner
}

func (e *MigrationEngine) Tracker() *persistence.MigrationTracker {
	return e.tracker
}

func (e *MigrationEngine) RunUp(ctx context.Context, modulePath string, moduleName string, migrations []*parser.MigrationFile) (int, error) {
	batch := e.tracker.NextBatch()
	totalRecords := 0

	for _, mf := range migrations {
		if e.tracker.HasRun(moduleName, mf.Name) {
			log.Printf("[MIGRATION] skip (already run): %s/%s", moduleName, mf.Name)
			continue
		}

		start := time.Now()
		count, err := e.executeMigration(ctx, modulePath, moduleName, mf)
		duration := time.Since(start)

		if err != nil {
			e.tracker.Record(moduleName, mf.Name, mf.Def.Model, string(mf.Def.Source.Type), 0, batch, duration, err)
			return totalRecords, fmt.Errorf("migration %s failed: %w", mf.Name, err)
		}

		e.tracker.Record(moduleName, mf.Name, mf.Def.Model, string(mf.Def.Source.Type), count, batch, duration, nil)
		totalRecords += count
		log.Printf("[MIGRATION] %s/%s: %d records (%s)", moduleName, mf.Name, count, duration.Round(time.Millisecond))
	}

	return totalRecords, nil
}

func (e *MigrationEngine) RunDown(ctx context.Context, modulePath string, moduleName string, migrations []*parser.MigrationFile) error {
	for i := len(migrations) - 1; i >= 0; i-- {
		mf := migrations[i]
		if !e.tracker.HasRun(moduleName, mf.Name) {
			continue
		}

		if err := e.rollbackMigration(ctx, modulePath, moduleName, mf); err != nil {
			return fmt.Errorf("rollback %s failed: %w", mf.Name, err)
		}

		e.tracker.RemoveByName(moduleName, mf.Name)
		log.Printf("[MIGRATION] rolled back: %s/%s", moduleName, mf.Name)
	}
	return nil
}

func (e *MigrationEngine) RollbackBatch(ctx context.Context, modulePath string, moduleName string, migrations []*parser.MigrationFile) error {
	batch := e.tracker.CurrentBatch()
	if batch == 0 {
		return fmt.Errorf("no migrations to rollback")
	}

	records, err := e.tracker.GetByBatch(batch)
	if err != nil {
		return err
	}

	migMap := make(map[string]*parser.MigrationFile)
	for _, mf := range migrations {
		migMap[mf.Name] = mf
	}

	for _, rec := range records {
		if rec.Module != moduleName && moduleName != "" {
			continue
		}
		mf, ok := migMap[rec.Name]
		if !ok {
			log.Printf("[MIGRATION] warning: migration file not found for %s, skipping rollback logic", rec.Name)
			e.tracker.RemoveByName(rec.Module, rec.Name)
			continue
		}

		if err := e.rollbackMigration(ctx, modulePath, rec.Module, mf); err != nil {
			return fmt.Errorf("rollback %s failed: %w", rec.Name, err)
		}
		e.tracker.RemoveByName(rec.Module, rec.Name)
		log.Printf("[MIGRATION] rolled back: %s/%s", rec.Module, rec.Name)
	}

	return nil
}

func (e *MigrationEngine) executeMigration(ctx context.Context, modulePath string, moduleName string, mf *parser.MigrationFile) (int, error) {
	def := mf.Def

	if def.Module == "" {
		def.Module = moduleName
	}

	records, err := ReadSourceData(modulePath, def.Source)
	if err != nil {
		return 0, fmt.Errorf("failed to read source data: %w", err)
	}

	if len(records) == 0 {
		return 0, nil
	}

	if def.FieldMapping != nil && len(def.FieldMapping) > 0 {
		records = applyFieldMapping(records, def.FieldMapping)
	}

	if def.Defaults != nil && len(def.Defaults) > 0 {
		records = applyDefaults(records, def.Defaults)
	}

	if def.Processor != nil {
		records, err = e.runProcessor(ctx, modulePath, def, records)
		if err != nil {
			return 0, fmt.Errorf("processor failed: %w", err)
		}
	}

	records = e.prepareRecords(def, records, moduleName, mf.Name)

	if def.Options.DryRun {
		log.Printf("[MIGRATION] dry run: would insert %d records into %s", len(records), def.Model)
		return len(records), nil
	}

	table := resolveSeederTable(def.Model, e.resolver)
	count, err := e.insertRecords(ctx, table, def, records)
	if err != nil {
		return count, err
	}

	return count, nil
}

func (e *MigrationEngine) rollbackMigration(ctx context.Context, modulePath string, moduleName string, mf *parser.MigrationFile) error {
	def := mf.Def
	strategy := def.GetDownStrategy()
	table := resolveSeederTable(def.Model, e.resolver)

	switch strategy {
	case parser.DownNone:
		return nil

	case parser.DownTruncate:
		return e.db.Exec(fmt.Sprintf("DELETE FROM %s", table)).Error

	case parser.DownDeleteBySource:
		return e.db.Exec(
			fmt.Sprintf("DELETE FROM %s WHERE _migration_source = ?", table),
			fmt.Sprintf("migration:%s", mf.Name),
		).Error

	case parser.DownCustom:
		if def.Down.Process != "" && e.processRunner != nil {
			input := map[string]any{
				"migration": mf.Name,
				"module":    moduleName,
				"model":     def.Model,
				"table":     table,
			}
			_, err := e.processRunner.RunProcess(ctx, def.Down.Process, input)
			return err
		}
		if def.Down.Script != nil && e.scriptRunner != nil {
			scriptPath := ResolveSourcePath(modulePath, def.Down.Script.File)
			params := map[string]any{
				"migration": mf.Name,
				"module":    moduleName,
				"model":     def.Model,
				"table":     table,
			}
			_, err := e.scriptRunner.Run(ctx, scriptPath, params)
			return err
		}
		return nil

	default:
		return nil
	}
}

func (e *MigrationEngine) runProcessor(ctx context.Context, modulePath string, def *parser.MigrationDefinition, records []map[string]any) ([]map[string]any, error) {
	proc := def.Processor

	switch proc.Type {
	case "process":
		if e.processRunner == nil {
			return nil, fmt.Errorf("process runner not configured")
		}
		input := map[string]any{
			"records": records,
			"model":   def.Model,
			"module":  def.Module,
		}
		result, err := e.processRunner.RunProcess(ctx, proc.Process, input)
		if err != nil {
			return nil, err
		}
		return extractProcessedRecords(result, records)

	case "script":
		if e.scriptRunner == nil {
			return nil, fmt.Errorf("script runner not configured")
		}
		scriptPath := ResolveSourcePath(modulePath, proc.Script.File)
		params := map[string]any{
			"records": records,
			"model":   def.Model,
			"module":  def.Module,
		}
		result, err := e.scriptRunner.Run(ctx, scriptPath, params)
		if err != nil {
			return nil, err
		}
		return extractProcessedRecords(result, records)

	default:
		return nil, fmt.Errorf("unsupported processor type: %s", proc.Type)
	}
}

func extractProcessedRecords(result any, fallback []map[string]any) ([]map[string]any, error) {
	if result == nil {
		return fallback, nil
	}

	switch v := result.(type) {
	case []map[string]any:
		return v, nil
	case []any:
		var records []map[string]any
		for _, item := range v {
			if m, ok := item.(map[string]any); ok {
				records = append(records, m)
			}
		}
		if len(records) > 0 {
			return records, nil
		}
	case map[string]any:
		if recs, ok := v["records"]; ok {
			if arr, ok := recs.([]any); ok {
				var records []map[string]any
				for _, item := range arr {
					if m, ok := item.(map[string]any); ok {
						records = append(records, m)
					}
				}
				return records, nil
			}
		}
	}

	data, err := json.Marshal(result)
	if err != nil {
		return fallback, nil
	}
	var records []map[string]any
	if err := json.Unmarshal(data, &records); err != nil {
		return fallback, nil
	}
	return records, nil
}

func (e *MigrationEngine) prepareRecords(def *parser.MigrationDefinition, records []map[string]any, moduleName string, migrationName string) []map[string]any {
	now := time.Now().Format("2006-01-02 15:04:05")

	for i := range records {
		if def.ShouldGenerateID() {
			if _, hasID := records[i]["id"]; !hasID {
				records[i]["id"] = uuid.New().String()
			}
		}

		if def.ShouldHashPasswords() {
			if pw, ok := records[i]["password"].(string); ok {
				hash, err := security.HashPassword(pw)
				if err == nil {
					records[i]["password_hash"] = hash
				}
				delete(records[i], "password")
			}
		}

		if def.ShouldSetTimestamps() {
			if _, ok := records[i]["created_at"]; !ok {
				records[i]["created_at"] = now
			}
			if _, ok := records[i]["updated_at"]; !ok {
				records[i]["updated_at"] = now
			}
		}

		if def.GetDownStrategy() == parser.DownDeleteBySource {
			records[i]["_migration_source"] = fmt.Sprintf("migration:%s", migrationName)
		}

		for k, v := range records[i] {
			switch val := v.(type) {
			case map[string]any, []any, []map[string]any:
				jsonBytes, err := json.Marshal(val)
				if err == nil {
					records[i][k] = string(jsonBytes)
				}
			}
		}
	}

	return records
}

func (e *MigrationEngine) insertRecords(ctx context.Context, table string, def *parser.MigrationDefinition, records []map[string]any) (int, error) {
	batchSize := def.GetBatchSize()
	conflictMode := def.GetConflictMode()
	totalInserted := 0

	for i := 0; i < len(records); i += batchSize {
		end := i + batchSize
		if end > len(records) {
			end = len(records)
		}
		batch := records[i:end]

		switch conflictMode {
		case parser.ConflictSkip:
			for _, record := range batch {
				checkField := findBestUniqueField(record, def.Options.UniqueFields)
				if checkField != "" {
					var count int64
					e.db.Table(table).Where(fmt.Sprintf("%s = ?", checkField), record[checkField]).Count(&count)
					if count > 0 {
						continue
					}
				}
				if err := e.db.Table(table).Create(&record).Error; err != nil {
					log.Printf("[MIGRATION] skip insert error for %s: %v", table, err)
					continue
				}
				totalInserted++
			}

		case parser.ConflictUpsert:
			if len(def.Options.UniqueFields) == 0 {
				return totalInserted, fmt.Errorf("upsert requires unique_fields")
			}

			updateFields := def.Options.UpdateFields
			if len(updateFields) == 0 {
				for k := range batch[0] {
					isUnique := false
					for _, uf := range def.Options.UniqueFields {
						if k == uf {
							isUnique = true
							break
						}
					}
					if !isUnique && k != "id" && k != "created_at" {
						updateFields = append(updateFields, k)
					}
				}
			}

			updateMap := make(map[string]interface{})
			for _, field := range updateFields {
				updateMap[field] = e.db.Raw(fmt.Sprintf("excluded.%s", field))
			}

			for _, record := range batch {
				checkField := def.Options.UniqueFields[0]
				checkVal := record[checkField]
				var count int64
				e.db.Table(table).Where(fmt.Sprintf("%s = ?", checkField), checkVal).Count(&count)
				if count > 0 {
					updates := make(map[string]any)
					for _, field := range updateFields {
						if v, ok := record[field]; ok {
							updates[field] = v
						}
					}
					if len(updates) > 0 {
						e.db.Table(table).Where(fmt.Sprintf("%s = ?", checkField), checkVal).Updates(updates)
					}
				} else {
					if err := e.db.Table(table).Create(&record).Error; err != nil {
						log.Printf("[MIGRATION] upsert insert error for %s: %v", table, err)
						continue
					}
				}
				totalInserted++
			}

		case parser.ConflictError:
			for _, record := range batch {
				if err := e.db.Table(table).Create(&record).Error; err != nil {
					return totalInserted, fmt.Errorf("insert failed for %s: %w", table, err)
				}
				totalInserted++
			}
		}
	}

	return totalInserted, nil
}

func applyFieldMapping(records []map[string]any, mapping map[string]string) []map[string]any {
	for i := range records {
		mapped := make(map[string]any, len(records[i]))
		for k, v := range records[i] {
			if newKey, ok := mapping[k]; ok {
				mapped[newKey] = v
			} else {
				mapped[k] = v
			}
		}
		records[i] = mapped
	}
	return records
}

func applyDefaults(records []map[string]any, defaults map[string]any) []map[string]any {
	for i := range records {
		for k, v := range defaults {
			if _, exists := records[i][k]; !exists {
				records[i][k] = v
			}
		}
	}
	return records
}

func findBestUniqueField(record map[string]any, configuredFields []string) string {
	if len(configuredFields) > 0 {
		for _, f := range configuredFields {
			if _, ok := record[f]; ok {
				return f
			}
		}
	}
	return findUniqueField(record)
}

func MigrateMigrationTable(db *gorm.DB) error {
	return persistence.AutoMigrateMigrationTracker(db)
}

func CollectModuleMigrations(modulePath string, patterns []string) ([]*parser.MigrationFile, error) {
	if len(patterns) == 0 {
		patterns = []string{"migrations/*.json"}
	}
	return parser.DiscoverMigrations(modulePath, patterns)
}

func CollectAllModuleMigrations(moduleDir string) (map[string][]*parser.MigrationFile, error) {
	result := make(map[string][]*parser.MigrationFile)

	entries, err := os.ReadDir(moduleDir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		modPath := filepath.Join(moduleDir, entry.Name())
		modFile := filepath.Join(modPath, "module.json")
		if _, err := os.Stat(modFile); err != nil {
			continue
		}

		modDef, err := parser.ParseModuleFile(modFile)
		if err != nil {
			continue
		}

		patterns := modDef.Migrations
		if len(patterns) == 0 {
			patterns = []string{"migrations/*.json"}
		}

		migrations, err := parser.DiscoverMigrations(modPath, patterns)
		if err != nil {
			continue
		}

		if len(migrations) > 0 {
			result[modDef.Name] = migrations
		}
	}

	return result, nil
}
