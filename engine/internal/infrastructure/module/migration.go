package module

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
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

type DataInserter interface {
	Insert(ctx context.Context, table string, record map[string]any) error
	Exists(ctx context.Context, table string, fields map[string]any) (bool, error)
	Update(ctx context.Context, table string, fields map[string]any, updates map[string]any) error
	Delete(ctx context.Context, table string, ids []string) error
	DeleteAll(ctx context.Context, table string) error
	WithTransaction(ctx context.Context, fn func(tx DataInserter) error) error
}

type GormDataInserter struct {
	db *gorm.DB
}

func NewGormDataInserter(db *gorm.DB) *GormDataInserter {
	return &GormDataInserter{db: db}
}

func (g *GormDataInserter) Insert(ctx context.Context, table string, record map[string]any) error {
	return g.db.WithContext(ctx).Table(table).Create(&record).Error
}

func (g *GormDataInserter) Exists(ctx context.Context, table string, fields map[string]any) (bool, error) {
	q := g.db.WithContext(ctx).Table(table)
	for k, v := range fields {
		q = q.Where(fmt.Sprintf("%s = ?", k), v)
	}
	var count int64
	err := q.Count(&count).Error
	return count > 0, err
}

func (g *GormDataInserter) Update(ctx context.Context, table string, fields map[string]any, updates map[string]any) error {
	q := g.db.WithContext(ctx).Table(table)
	for k, v := range fields {
		q = q.Where(fmt.Sprintf("%s = ?", k), v)
	}
	return q.Updates(updates).Error
}

func (g *GormDataInserter) Delete(ctx context.Context, table string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	return g.db.WithContext(ctx).Table(table).Where("id IN ?", ids).Delete(nil).Error
}

func (g *GormDataInserter) DeleteAll(ctx context.Context, table string) error {
	return g.db.WithContext(ctx).Exec(fmt.Sprintf("DELETE FROM %s", table)).Error
}

func (g *GormDataInserter) WithTransaction(ctx context.Context, fn func(tx DataInserter) error) error {
	return g.db.WithContext(ctx).Transaction(func(gormTx *gorm.DB) error {
		return fn(&GormDataInserter{db: gormTx})
	})
}

type MongoDataInserter struct {
	conn *persistence.MongoConnection
}

func NewMongoDataInserter(conn *persistence.MongoConnection) *MongoDataInserter {
	return &MongoDataInserter{conn: conn}
}

func (m *MongoDataInserter) Insert(ctx context.Context, table string, record map[string]any) error {
	if id, ok := record["id"]; ok {
		record["_id"] = id
		delete(record, "id")
	}
	_, err := m.conn.Collection(table).InsertOne(ctx, record)
	return err
}

func (m *MongoDataInserter) Exists(ctx context.Context, table string, fields map[string]any) (bool, error) {
	import_bson := make(map[string]any)
	for k, v := range fields {
		if k == "id" {
			import_bson["_id"] = v
		} else {
			import_bson[k] = v
		}
	}
	count, err := m.conn.Collection(table).CountDocuments(ctx, import_bson)
	return count > 0, err
}

func (m *MongoDataInserter) Update(ctx context.Context, table string, fields map[string]any, updates map[string]any) error {
	import_bson := make(map[string]any)
	for k, v := range fields {
		if k == "id" {
			import_bson["_id"] = v
		} else {
			import_bson[k] = v
		}
	}
	import_set := make(map[string]any)
	import_set["$set"] = updates
	_, err := m.conn.Collection(table).UpdateOne(ctx, import_bson, import_set)
	return err
}

func (m *MongoDataInserter) Delete(ctx context.Context, table string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	import_in := make(map[string]any)
	import_in["$in"] = ids
	filter := map[string]any{"_id": import_in}
	_, err := m.conn.Collection(table).DeleteMany(ctx, filter)
	return err
}

func (m *MongoDataInserter) DeleteAll(ctx context.Context, table string) error {
	_, err := m.conn.Collection(table).DeleteMany(ctx, map[string]any{})
	return err
}

func (m *MongoDataInserter) WithTransaction(ctx context.Context, fn func(tx DataInserter) error) error {
	return fn(m)
}

type MigrationEngine struct {
	tracker       *persistence.MigrationTracker
	inserter      DataInserter
	resolver      TableNameResolver
	processRunner ProcessRunner
	scriptRunner  ScriptRunner
}

func NewMigrationEngine(store persistence.MigrationStore, inserter DataInserter, resolver TableNameResolver) *MigrationEngine {
	return &MigrationEngine{
		tracker:  persistence.NewMigrationTrackerFromStore(store),
		inserter: inserter,
		resolver: resolver,
	}
}

func (e *MigrationEngine) SetProcessRunner(runner ProcessRunner) { e.processRunner = runner }
func (e *MigrationEngine) SetScriptRunner(runner ScriptRunner)   { e.scriptRunner = runner }
func (e *MigrationEngine) Tracker() *persistence.MigrationTracker { return e.tracker }

func (e *MigrationEngine) RunUp(ctx context.Context, modulePath string, moduleName string, migrations []*parser.MigrationFile) (int, error) {
	batch := 0
	totalRecords := 0

	for _, mf := range migrations {
		if e.tracker.HasRun(moduleName, mf.Name) {
			continue
		}

		if batch == 0 {
			batch = e.tracker.NextBatch()
		}

		start := time.Now()
		count, insertedIDs, err := e.executeMigration(ctx, modulePath, moduleName, mf)
		duration := time.Since(start)

		if err != nil {
			e.tracker.Record(moduleName, mf.Name, mf.Def.Model, string(mf.Def.Source.Type), 0, batch, duration, nil, err)
			return totalRecords, fmt.Errorf("migration %s failed: %w", mf.Name, err)
		}

		e.tracker.Record(moduleName, mf.Name, mf.Def.Model, string(mf.Def.Source.Type), count, batch, duration, insertedIDs, nil)
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

func (e *MigrationEngine) executeMigration(ctx context.Context, modulePath string, moduleName string, mf *parser.MigrationFile) (int, []string, error) {
	def := mf.Def

	if def.Module == "" {
		def.Module = moduleName
	}

	records, err := ReadSourceData(modulePath, def.Source)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to read source data: %w", err)
	}

	if len(records) == 0 {
		return 0, nil, nil
	}

	if def.FieldMapping != nil && len(def.FieldMapping) > 0 {
		records = applyFieldMapping(records, def.FieldMapping)
	}

	if def.Defaults != nil && len(def.Defaults) > 0 {
		records = applyDefaults(records, def.Defaults)
	}

	if def.Source.Options.FieldTypes != nil && len(def.Source.Options.FieldTypes) > 0 {
		records = applyFieldTypes(records, def.Source.Options.FieldTypes)
	}

	if def.Processor != nil {
		records, err = e.runProcessor(ctx, modulePath, def, records)
		if err != nil {
			return 0, nil, fmt.Errorf("processor failed: %w", err)
		}
	}

	records = e.prepareRecords(def, records)

	if def.Options.DryRun {
		log.Printf("[MIGRATION] dry run: would insert %d records into %s", len(records), def.Model)
		return len(records), nil, nil
	}

	table := resolveSeederTable(def.Model, e.resolver)

	var count int
	var insertedIDs []string

	txErr := e.inserter.WithTransaction(ctx, func(tx DataInserter) error {
		var txErr error
		count, insertedIDs, txErr = e.insertRecords(ctx, tx, table, def, records)
		return txErr
	})

	if txErr != nil {
		return 0, nil, txErr
	}

	return count, insertedIDs, nil
}

func (e *MigrationEngine) rollbackMigration(ctx context.Context, modulePath string, moduleName string, mf *parser.MigrationFile) error {
	def := mf.Def
	strategy := def.GetDownStrategy()
	table := resolveSeederTable(def.Model, e.resolver)

	switch strategy {
	case parser.DownNone:
		return nil

	case parser.DownDeleteSeeded:
		rec, err := e.tracker.GetByName(moduleName, mf.Name)
		if err != nil {
			return nil
		}
		ids := rec.GetRecordIDs()
		if len(ids) == 0 {
			return nil
		}
		return e.inserter.Delete(ctx, table, ids)

	case parser.DownTruncate:
		return e.inserter.DeleteAll(ctx, table)

	case parser.DownCustom:
		if def.Down.Process != "" && e.processRunner != nil {
			input := map[string]any{
				"migration": mf.Name, "module": moduleName,
				"model": def.Model, "table": table,
			}
			_, err := e.processRunner.RunProcess(ctx, def.Down.Process, input)
			return err
		}
		if def.Down.Script != nil && e.scriptRunner != nil {
			scriptPath := ResolveSourcePath(modulePath, def.Down.Script.File)
			params := map[string]any{
				"migration": mf.Name, "module": moduleName,
				"model": def.Model, "table": table,
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
		input := map[string]any{"records": records, "model": def.Model, "module": def.Module}
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
		params := map[string]any{"records": records, "model": def.Model, "module": def.Module}
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

func (e *MigrationEngine) prepareRecords(def *parser.MigrationDefinition, records []map[string]any) []map[string]any {
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

func (e *MigrationEngine) insertRecords(ctx context.Context, inserter DataInserter, table string, def *parser.MigrationDefinition, records []map[string]any) (int, []string, error) {
	conflictMode := def.GetConflictMode()
	noupdate := def.Options.NoUpdate
	totalInserted := 0
	var insertedIDs []string

	for _, record := range records {
		switch conflictMode {
		case parser.ConflictSkip:
			uniqueWhere := buildUniqueWhere(record, def.Options.UniqueFields)
			if len(uniqueWhere) > 0 {
				exists, _ := inserter.Exists(ctx, table, uniqueWhere)
				if exists {
					continue
				}
			}
			if err := inserter.Insert(ctx, table, record); err != nil {
				log.Printf("[MIGRATION] skip insert error for %s: %v", table, err)
				continue
			}
			totalInserted++
			if id, ok := record["id"]; ok {
				insertedIDs = append(insertedIDs, fmt.Sprintf("%v", id))
			}

		case parser.ConflictUpsert:
			uniqueWhere := buildUniqueWhere(record, def.Options.UniqueFields)
			if len(uniqueWhere) == 0 {
				return totalInserted, insertedIDs, fmt.Errorf("upsert requires unique_fields")
			}

			exists, _ := inserter.Exists(ctx, table, uniqueWhere)
			if exists {
				if noupdate {
					continue
				}
				updates := buildUpdateFields(record, def)
				if len(updates) > 0 {
					inserter.Update(ctx, table, uniqueWhere, updates)
				}
			} else {
				if err := inserter.Insert(ctx, table, record); err != nil {
					log.Printf("[MIGRATION] upsert insert error for %s: %v", table, err)
					continue
				}
				if id, ok := record["id"]; ok {
					insertedIDs = append(insertedIDs, fmt.Sprintf("%v", id))
				}
			}
			totalInserted++

		case parser.ConflictError:
			if err := inserter.Insert(ctx, table, record); err != nil {
				return totalInserted, insertedIDs, fmt.Errorf("insert failed for %s: %w", table, err)
			}
			totalInserted++
			if id, ok := record["id"]; ok {
				insertedIDs = append(insertedIDs, fmt.Sprintf("%v", id))
			}
		}
	}

	return totalInserted, insertedIDs, nil
}

func buildUniqueWhere(record map[string]any, configuredFields []string) map[string]any {
	where := make(map[string]any)

	if len(configuredFields) > 0 {
		for _, f := range configuredFields {
			if v, ok := record[f]; ok {
				where[f] = v
			}
		}
		if len(where) > 0 {
			return where
		}
	}

	field := findUniqueField(record)
	if field != "" {
		where[field] = record[field]
	}
	return where
}

func buildUpdateFields(record map[string]any, def *parser.MigrationDefinition) map[string]any {
	updates := make(map[string]any)

	if len(def.Options.UpdateFields) > 0 {
		for _, field := range def.Options.UpdateFields {
			if v, ok := record[field]; ok {
				updates[field] = v
			}
		}
		return updates
	}

	uniqueSet := make(map[string]bool)
	for _, uf := range def.Options.UniqueFields {
		uniqueSet[uf] = true
	}

	for k, v := range record {
		if k == "id" || k == "created_at" || uniqueSet[k] {
			continue
		}
		updates[k] = v
	}
	return updates
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

func applyFieldTypes(records []map[string]any, fieldTypes map[string]string) []map[string]any {
	for i := range records {
		for field, typ := range fieldTypes {
			val, ok := records[i][field]
			if !ok {
				continue
			}
			records[i][field] = coerceType(val, typ)
		}
	}
	return records
}

func coerceType(val any, typ string) any {
	s := fmt.Sprintf("%v", val)
	switch strings.ToLower(typ) {
	case "string", "text":
		return s
	case "int", "integer":
		var i int64
		fmt.Sscanf(s, "%d", &i)
		return i
	case "float", "decimal", "number":
		var f float64
		fmt.Sscanf(s, "%f", &f)
		return f
	case "bool", "boolean":
		return s == "true" || s == "1" || s == "yes"
	default:
		return val
	}
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

type OrderedModuleMigrations struct {
	Module     string
	Path       string
	Migrations []*parser.MigrationFile
}

func CollectAllModuleMigrationsOrdered(moduleDir string, moduleOrder []string) ([]OrderedModuleMigrations, error) {
	var result []OrderedModuleMigrations

	for _, modName := range moduleOrder {
		modPath := filepath.Join(moduleDir, modName)
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
			result = append(result, OrderedModuleMigrations{
				Module:     modDef.Name,
				Path:       modPath,
				Migrations: migrations,
			})
		}
	}

	return result, nil
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
