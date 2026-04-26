package module

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/bitcode-framework/bitcode/internal/compiler/parser"
	"github.com/bitcode-framework/bitcode/internal/infrastructure/persistence"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	return db
}

type testResolver struct{}

func (r *testResolver) TableName(modelName string) string {
	return modelName
}

func TestMigrationEngine_RunUp_JSON(t *testing.T) {
	db := setupTestDB(t)
	MigrateMigrationTable(db)
	db.Exec("CREATE TABLE contact (id TEXT, name TEXT, email TEXT, created_at TEXT, updated_at TEXT, _migration_source TEXT)")

	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "data"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "migrations"), 0755)

	contacts := []map[string]any{
		{"name": "Alice", "email": "alice@test.com"},
		{"name": "Bob", "email": "bob@test.com"},
	}
	data, _ := json.Marshal(contacts)
	os.WriteFile(filepath.Join(tmpDir, "data", "contacts.json"), data, 0644)

	migDef := map[string]any{
		"name": "seed_contacts", "model": "contact",
		"source":  map[string]any{"type": "json", "file": "data/contacts.json"},
		"options": map[string]any{"on_conflict": "skip"},
		"down":    map[string]any{"strategy": "delete_by_source"},
	}
	migData, _ := json.Marshal(migDef)
	os.WriteFile(filepath.Join(tmpDir, "migrations", "20260101_000001_seed_contacts.json"), migData, 0644)

	engine := NewMigrationEngine(db, &testResolver{})
	migrations, err := CollectModuleMigrations(tmpDir, []string{"migrations/*.json"})
	if err != nil {
		t.Fatalf("discover failed: %v", err)
	}
	if len(migrations) != 1 {
		t.Fatalf("expected 1 migration, got %d", len(migrations))
	}

	ctx := context.Background()
	count, err := engine.RunUp(ctx, tmpDir, "test", migrations)
	if err != nil {
		t.Fatalf("RunUp failed: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 records, got %d", count)
	}
	if !engine.Tracker().HasRun("test", "seed_contacts") {
		t.Error("migration should be tracked")
	}

	count2, _ := engine.RunUp(ctx, tmpDir, "test", migrations)
	if count2 != 0 {
		t.Errorf("expected 0 on re-run, got %d", count2)
	}
}

func TestMigrationEngine_RunDown(t *testing.T) {
	db := setupTestDB(t)
	MigrateMigrationTable(db)
	db.Exec("CREATE TABLE tag (id TEXT, name TEXT, created_at TEXT, updated_at TEXT, _migration_source TEXT)")

	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "data"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "migrations"), 0755)

	tags := []map[string]any{{"name": "VIP"}, {"name": "Partner"}}
	data, _ := json.Marshal(tags)
	os.WriteFile(filepath.Join(tmpDir, "data", "tags.json"), data, 0644)

	migDef := map[string]any{
		"name": "seed_tags", "model": "tag",
		"source": map[string]any{"type": "json", "file": "data/tags.json"},
		"down":   map[string]any{"strategy": "delete_by_source"},
	}
	migData, _ := json.Marshal(migDef)
	os.WriteFile(filepath.Join(tmpDir, "migrations", "20260101_000001_seed_tags.json"), migData, 0644)

	engine := NewMigrationEngine(db, &testResolver{})
	migrations, _ := CollectModuleMigrations(tmpDir, []string{"migrations/*.json"})

	ctx := context.Background()
	engine.RunUp(ctx, tmpDir, "test", migrations)

	var count int64
	db.Table("tag").Count(&count)
	if count != 2 {
		t.Fatalf("expected 2 tags, got %d", count)
	}

	err := engine.RunDown(ctx, tmpDir, "test", migrations)
	if err != nil {
		t.Fatalf("RunDown failed: %v", err)
	}

	db.Table("tag").Count(&count)
	if count != 0 {
		t.Errorf("expected 0 after rollback, got %d", count)
	}
	if engine.Tracker().HasRun("test", "seed_tags") {
		t.Error("should not be tracked after rollback")
	}
}

func TestMigrationEngine_Upsert(t *testing.T) {
	db := setupTestDB(t)
	MigrateMigrationTable(db)
	db.Exec("CREATE TABLE contact (id TEXT, name TEXT, email TEXT, phone TEXT, created_at TEXT, updated_at TEXT)")
	db.Exec("INSERT INTO contact (id, name, email, phone) VALUES ('1', 'Old', 'alice@test.com', '111')")

	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "data"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "migrations"), 0755)

	contacts := []map[string]any{
		{"name": "Alice Updated", "email": "alice@test.com", "phone": "222"},
		{"name": "Bob", "email": "bob@test.com", "phone": "333"},
	}
	data, _ := json.Marshal(contacts)
	os.WriteFile(filepath.Join(tmpDir, "data", "contacts.json"), data, 0644)

	migDef := map[string]any{
		"name": "upsert_contacts", "model": "contact",
		"source": map[string]any{"type": "json", "file": "data/contacts.json"},
		"options": map[string]any{
			"on_conflict":   "upsert",
			"unique_fields": []string{"email"},
			"update_fields": []string{"name", "phone"},
		},
	}
	migData, _ := json.Marshal(migDef)
	os.WriteFile(filepath.Join(tmpDir, "migrations", "20260101_000001_upsert.json"), migData, 0644)

	engine := NewMigrationEngine(db, &testResolver{})
	migrations, _ := CollectModuleMigrations(tmpDir, []string{"migrations/*.json"})

	ctx := context.Background()
	_, err := engine.RunUp(ctx, tmpDir, "test", migrations)
	if err != nil {
		t.Fatalf("RunUp failed: %v", err)
	}

	var result map[string]any
	db.Table("contact").Where("email = ?", "alice@test.com").Take(&result)
	if result["name"] != "Alice Updated" {
		t.Errorf("expected 'Alice Updated', got %v", result["name"])
	}

	var total int64
	db.Table("contact").Count(&total)
	if total != 2 {
		t.Errorf("expected 2, got %d", total)
	}
}

func TestMigrationEngine_FieldMapping(t *testing.T) {
	db := setupTestDB(t)
	MigrateMigrationTable(db)
	db.Exec("CREATE TABLE contact (id TEXT, name TEXT, email TEXT, created_at TEXT, updated_at TEXT)")

	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "data"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "migrations"), 0755)

	contacts := []map[string]any{{"full_name": "Alice", "mail": "alice@test.com"}}
	data, _ := json.Marshal(contacts)
	os.WriteFile(filepath.Join(tmpDir, "data", "contacts.json"), data, 0644)

	migDef := map[string]any{
		"name": "mapped", "model": "contact",
		"source":        map[string]any{"type": "json", "file": "data/contacts.json"},
		"field_mapping": map[string]string{"full_name": "name", "mail": "email"},
	}
	migData, _ := json.Marshal(migDef)
	os.WriteFile(filepath.Join(tmpDir, "migrations", "20260101_000001_mapped.json"), migData, 0644)

	engine := NewMigrationEngine(db, &testResolver{})
	migrations, _ := CollectModuleMigrations(tmpDir, []string{"migrations/*.json"})

	ctx := context.Background()
	count, err := engine.RunUp(ctx, tmpDir, "test", migrations)
	if err != nil {
		t.Fatalf("RunUp failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1, got %d", count)
	}

	var result map[string]any
	db.Table("contact").Take(&result)
	if result["name"] != "Alice" {
		t.Errorf("expected 'Alice', got %v", result["name"])
	}
}

func TestMigrationEngine_Defaults(t *testing.T) {
	db := setupTestDB(t)
	MigrateMigrationTable(db)
	db.Exec("CREATE TABLE contact (id TEXT, name TEXT, email TEXT, type TEXT, created_at TEXT, updated_at TEXT)")

	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "data"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "migrations"), 0755)

	contacts := []map[string]any{{"name": "Alice", "email": "alice@test.com"}}
	data, _ := json.Marshal(contacts)
	os.WriteFile(filepath.Join(tmpDir, "data", "contacts.json"), data, 0644)

	migDef := map[string]any{
		"name": "defaults", "model": "contact",
		"source":   map[string]any{"type": "json", "file": "data/contacts.json"},
		"defaults": map[string]any{"type": "person"},
	}
	migData, _ := json.Marshal(migDef)
	os.WriteFile(filepath.Join(tmpDir, "migrations", "20260101_000001_defaults.json"), migData, 0644)

	engine := NewMigrationEngine(db, &testResolver{})
	migrations, _ := CollectModuleMigrations(tmpDir, []string{"migrations/*.json"})

	ctx := context.Background()
	engine.RunUp(ctx, tmpDir, "test", migrations)

	var result map[string]any
	db.Table("contact").Take(&result)
	if result["type"] != "person" {
		t.Errorf("expected 'person', got %v", result["type"])
	}
}

func TestCSVReader(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "test.csv"), []byte("name,email,age\nAlice,alice@test.com,30\nBob,bob@test.com,25\n"), 0644)

	reader := &CSVReader{}
	records, err := reader.Read(filepath.Join(tmpDir, "test.csv"), parser.MigrationSourceOptions{})
	if err != nil {
		t.Fatalf("CSV read failed: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2, got %d", len(records))
	}
	if records[0]["name"] != "Alice" {
		t.Errorf("expected 'Alice', got %v", records[0]["name"])
	}
	if records[0]["age"] != int64(30) {
		t.Errorf("expected 30, got %v (%T)", records[0]["age"], records[0]["age"])
	}
}

func TestCSVReader_CustomDelimiter(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "test.csv"), []byte("name;email\nAlice;alice@test.com\n"), 0644)

	reader := &CSVReader{}
	records, err := reader.Read(filepath.Join(tmpDir, "test.csv"), parser.MigrationSourceOptions{Delimiter: ";"})
	if err != nil {
		t.Fatalf("CSV read failed: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1, got %d", len(records))
	}
}

func TestJSONReader_Array(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "test.json"), []byte(`[{"name":"Alice"},{"name":"Bob"}]`), 0644)

	reader := &JSONReader{}
	records, err := reader.Read(filepath.Join(tmpDir, "test.json"), parser.MigrationSourceOptions{})
	if err != nil {
		t.Fatalf("JSON read failed: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2, got %d", len(records))
	}
}

func TestJSONReader_ObjectWithPath(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "test.json"), []byte(`{"data":{"contacts":[{"name":"Alice"}]}}`), 0644)

	reader := &JSONReader{}
	records, err := reader.Read(filepath.Join(tmpDir, "test.json"), parser.MigrationSourceOptions{RootElement: "data.contacts"})
	if err != nil {
		t.Fatalf("JSON read failed: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1, got %d", len(records))
	}
}

func TestXMLReader(t *testing.T) {
	tmpDir := t.TempDir()
	xml := `<?xml version="1.0"?><contacts><contact><name>Alice</name><email>alice@test.com</email></contact><contact><name>Bob</name><email>bob@test.com</email></contact></contacts>`
	os.WriteFile(filepath.Join(tmpDir, "test.xml"), []byte(xml), 0644)

	reader := &XMLReader{}
	records, err := reader.Read(filepath.Join(tmpDir, "test.xml"), parser.MigrationSourceOptions{})
	if err != nil {
		t.Fatalf("XML read failed: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2, got %d", len(records))
	}
	if records[0]["name"] != "Alice" {
		t.Errorf("expected 'Alice', got %v", records[0]["name"])
	}
}

func TestParseMigration_Valid(t *testing.T) {
	def, err := parser.ParseMigration([]byte(`{"name":"seed","model":"contact","source":{"type":"json","file":"data.json"}}`))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if def.Name != "seed" {
		t.Errorf("expected 'seed', got %s", def.Name)
	}
}

func TestParseMigration_MissingName(t *testing.T) {
	_, err := parser.ParseMigration([]byte(`{"model":"contact","source":{"type":"json","file":"data.json"}}`))
	if err == nil {
		t.Error("expected error")
	}
}

func TestParseMigration_UpsertWithoutUnique(t *testing.T) {
	_, err := parser.ParseMigration([]byte(`{"name":"t","model":"c","source":{"type":"json","file":"d.json"},"options":{"on_conflict":"upsert"}}`))
	if err == nil {
		t.Error("expected error")
	}
}

func TestParseMigration_InvalidSource(t *testing.T) {
	_, err := parser.ParseMigration([]byte(`{"name":"t","model":"c","source":{"type":"yaml","file":"d.yaml"}}`))
	if err == nil {
		t.Error("expected error")
	}
}

func TestMigrationTracker_Status(t *testing.T) {
	db := setupTestDB(t)
	MigrateMigrationTable(db)

	tracker := persistence.NewMigrationTracker(db)
	tracker.Record("crm", "seed_tags", "tag", "json", 5, 1, 0, nil)
	tracker.Record("crm", "seed_contacts", "contact", "csv", 10, 1, 0, nil)

	entries, err := tracker.Status()
	if err != nil {
		t.Fatalf("status failed: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2, got %d", len(entries))
	}
}

func TestMigrationTracker_GetPending(t *testing.T) {
	db := setupTestDB(t)
	MigrateMigrationTable(db)

	tracker := persistence.NewMigrationTracker(db)
	tracker.Record("crm", "seed_tags", "tag", "json", 5, 1, 0, nil)

	pending := tracker.GetPending("crm", []string{"seed_tags", "seed_contacts", "seed_leads"})
	if len(pending) != 2 {
		t.Fatalf("expected 2 pending, got %d", len(pending))
	}
}

func TestInferType(t *testing.T) {
	tests := []struct {
		input    string
		expected any
	}{
		{"42", int64(42)},
		{"3.14", 3.14},
		{"true", true},
		{"false", false},
		{"hello", "hello"},
		{"", ""},
		{"007", "007"},
		{"0", int64(0)},
	}
	for _, tt := range tests {
		result := inferType(tt.input)
		if result != tt.expected {
			t.Errorf("inferType(%q) = %v (%T), want %v (%T)", tt.input, result, result, tt.expected, tt.expected)
		}
	}
}
