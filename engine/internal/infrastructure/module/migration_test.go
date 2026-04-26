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

func (r *testResolver) TableName(modelName string) string { return modelName }

func setupEngine(t *testing.T, db *gorm.DB) *MigrationEngine {
	t.Helper()
	MigrateMigrationTable(db)
	store := persistence.NewGormMigrationStore(db)
	inserter := NewGormDataInserter(db)
	return NewMigrationEngine(store, inserter, &testResolver{})
}

func writeMigration(t *testing.T, dir string, filename string, def map[string]any) {
	t.Helper()
	data, _ := json.Marshal(def)
	os.WriteFile(filepath.Join(dir, filename), data, 0644)
}

func writeJSON(t *testing.T, path string, data any) {
	t.Helper()
	b, _ := json.Marshal(data)
	os.WriteFile(path, b, 0644)
}

func TestMigrationEngine_RunUp_JSON(t *testing.T) {
	db := setupTestDB(t)
	db.Exec("CREATE TABLE contact (id TEXT, name TEXT, email TEXT, created_at TEXT, updated_at TEXT)")
	engine := setupEngine(t, db)

	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "data"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "migrations"), 0755)

	writeJSON(t, filepath.Join(tmpDir, "data", "contacts.json"), []map[string]any{
		{"name": "Alice", "email": "alice@test.com"},
		{"name": "Bob", "email": "bob@test.com"},
	})

	writeMigration(t, filepath.Join(tmpDir, "migrations"), "20260101_000001_seed_contacts.json", map[string]any{
		"name": "seed_contacts", "model": "contact",
		"source":  map[string]any{"type": "json", "file": "data/contacts.json"},
		"options": map[string]any{"on_conflict": "skip"},
		"down":    map[string]any{"strategy": "delete_seeded"},
	})

	migrations, err := CollectModuleMigrations(tmpDir, []string{"migrations/*.json"})
	if err != nil {
		t.Fatalf("discover failed: %v", err)
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

func TestMigrationEngine_RunDown_DeleteSeeded(t *testing.T) {
	db := setupTestDB(t)
	db.Exec("CREATE TABLE tag (id TEXT, name TEXT, created_at TEXT, updated_at TEXT)")
	engine := setupEngine(t, db)

	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "data"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "migrations"), 0755)

	writeJSON(t, filepath.Join(tmpDir, "data", "tags.json"), []map[string]any{
		{"name": "VIP"}, {"name": "Partner"},
	})

	writeMigration(t, filepath.Join(tmpDir, "migrations"), "20260101_000001_seed_tags.json", map[string]any{
		"name": "seed_tags", "model": "tag",
		"source": map[string]any{"type": "json", "file": "data/tags.json"},
		"down":   map[string]any{"strategy": "delete_seeded"},
	})

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

func TestMigrationEngine_CompositeUniqueUpsert(t *testing.T) {
	db := setupTestDB(t)
	db.Exec("CREATE TABLE contact (id TEXT, name TEXT, email TEXT, company TEXT, phone TEXT, created_at TEXT, updated_at TEXT)")
	db.Exec("INSERT INTO contact (id, name, email, company, phone) VALUES ('1', 'Old', 'alice@test.com', 'Acme', '111')")
	engine := setupEngine(t, db)

	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "data"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "migrations"), 0755)

	writeJSON(t, filepath.Join(tmpDir, "data", "contacts.json"), []map[string]any{
		{"name": "Alice Updated", "email": "alice@test.com", "company": "Acme", "phone": "222"},
		{"name": "Bob", "email": "bob@test.com", "company": "Beta", "phone": "333"},
	})

	writeMigration(t, filepath.Join(tmpDir, "migrations"), "20260101_000001_upsert.json", map[string]any{
		"name": "upsert_contacts", "model": "contact",
		"source": map[string]any{"type": "json", "file": "data/contacts.json"},
		"options": map[string]any{
			"on_conflict":   "upsert",
			"unique_fields": []string{"email", "company"},
			"update_fields": []string{"name", "phone"},
		},
	})

	migrations, _ := CollectModuleMigrations(tmpDir, []string{"migrations/*.json"})
	ctx := context.Background()
	_, err := engine.RunUp(ctx, tmpDir, "test", migrations)
	if err != nil {
		t.Fatalf("RunUp failed: %v", err)
	}

	var result map[string]any
	db.Table("contact").Where("email = ? AND company = ?", "alice@test.com", "Acme").Take(&result)
	if result["name"] != "Alice Updated" {
		t.Errorf("expected 'Alice Updated', got %v", result["name"])
	}

	var total int64
	db.Table("contact").Count(&total)
	if total != 2 {
		t.Errorf("expected 2, got %d", total)
	}
}

func TestMigrationEngine_NoUpdate(t *testing.T) {
	db := setupTestDB(t)
	db.Exec("CREATE TABLE contact (id TEXT, name TEXT, email TEXT, created_at TEXT, updated_at TEXT)")
	db.Exec("INSERT INTO contact (id, name, email) VALUES ('1', 'Custom Name', 'alice@test.com')")
	engine := setupEngine(t, db)

	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "data"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "migrations"), 0755)

	writeJSON(t, filepath.Join(tmpDir, "data", "contacts.json"), []map[string]any{
		{"name": "Default Name", "email": "alice@test.com"},
	})

	writeMigration(t, filepath.Join(tmpDir, "migrations"), "20260101_000001_noupdate.json", map[string]any{
		"name": "noupdate_contacts", "model": "contact",
		"source": map[string]any{"type": "json", "file": "data/contacts.json"},
		"options": map[string]any{
			"on_conflict":   "upsert",
			"unique_fields": []string{"email"},
			"noupdate":      true,
		},
	})

	migrations, _ := CollectModuleMigrations(tmpDir, []string{"migrations/*.json"})
	ctx := context.Background()
	engine.RunUp(ctx, tmpDir, "test", migrations)

	var result map[string]any
	db.Table("contact").Where("email = ?", "alice@test.com").Take(&result)
	if result["name"] != "Custom Name" {
		t.Errorf("noupdate should preserve 'Custom Name', got %v", result["name"])
	}
}

func TestMigrationEngine_FieldMapping(t *testing.T) {
	db := setupTestDB(t)
	db.Exec("CREATE TABLE contact (id TEXT, name TEXT, email TEXT, created_at TEXT, updated_at TEXT)")
	engine := setupEngine(t, db)

	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "data"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "migrations"), 0755)

	writeJSON(t, filepath.Join(tmpDir, "data", "contacts.json"), []map[string]any{
		{"full_name": "Alice", "mail": "alice@test.com"},
	})

	writeMigration(t, filepath.Join(tmpDir, "migrations"), "20260101_000001_mapped.json", map[string]any{
		"name": "mapped", "model": "contact",
		"source":        map[string]any{"type": "json", "file": "data/contacts.json"},
		"field_mapping": map[string]string{"full_name": "name", "mail": "email"},
	})

	migrations, _ := CollectModuleMigrations(tmpDir, []string{"migrations/*.json"})
	ctx := context.Background()
	count, _ := engine.RunUp(ctx, tmpDir, "test", migrations)
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
	db.Exec("CREATE TABLE contact (id TEXT, name TEXT, email TEXT, type TEXT, created_at TEXT, updated_at TEXT)")
	engine := setupEngine(t, db)

	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "data"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "migrations"), 0755)

	writeJSON(t, filepath.Join(tmpDir, "data", "contacts.json"), []map[string]any{
		{"name": "Alice", "email": "alice@test.com"},
	})

	writeMigration(t, filepath.Join(tmpDir, "migrations"), "20260101_000001_defaults.json", map[string]any{
		"name": "defaults", "model": "contact",
		"source":   map[string]any{"type": "json", "file": "data/contacts.json"},
		"defaults": map[string]any{"type": "person"},
	})

	migrations, _ := CollectModuleMigrations(tmpDir, []string{"migrations/*.json"})
	ctx := context.Background()
	engine.RunUp(ctx, tmpDir, "test", migrations)

	var result map[string]any
	db.Table("contact").Take(&result)
	if result["type"] != "person" {
		t.Errorf("expected 'person', got %v", result["type"])
	}
}

func TestMigrationEngine_FieldTypes(t *testing.T) {
	db := setupTestDB(t)
	db.Exec("CREATE TABLE product (id TEXT, code TEXT, price REAL, created_at TEXT, updated_at TEXT)")
	engine := setupEngine(t, db)

	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "data"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "migrations"), 0755)

	os.WriteFile(filepath.Join(tmpDir, "data", "products.csv"), []byte("code,price\n00123,45.99\n"), 0644)

	writeMigration(t, filepath.Join(tmpDir, "migrations"), "20260101_000001_products.json", map[string]any{
		"name": "seed_products", "model": "product",
		"source": map[string]any{
			"type": "csv", "file": "data/products.csv",
			"options": map[string]any{
				"field_types": map[string]string{"code": "string"},
			},
		},
	})

	migrations, _ := CollectModuleMigrations(tmpDir, []string{"migrations/*.json"})
	ctx := context.Background()
	engine.RunUp(ctx, tmpDir, "test", migrations)

	var result map[string]any
	db.Table("product").Take(&result)
	if result["code"] != "00123" {
		t.Errorf("expected code '00123' (string), got %v (%T)", result["code"], result["code"])
	}
}

func TestMigrationEngine_Transaction_Rollback(t *testing.T) {
	db := setupTestDB(t)
	db.Exec("CREATE TABLE contact (id TEXT PRIMARY KEY, name TEXT NOT NULL, email TEXT UNIQUE, created_at TEXT, updated_at TEXT)")
	engine := setupEngine(t, db)

	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "data"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "migrations"), 0755)

	writeJSON(t, filepath.Join(tmpDir, "data", "contacts.json"), []map[string]any{
		{"name": "Alice", "email": "alice@test.com"},
		{"name": "Bob", "email": "alice@test.com"},
	})

	writeMigration(t, filepath.Join(tmpDir, "migrations"), "20260101_000001_fail.json", map[string]any{
		"name": "fail_contacts", "model": "contact",
		"source":  map[string]any{"type": "json", "file": "data/contacts.json"},
		"options": map[string]any{"on_conflict": "error"},
	})

	migrations, _ := CollectModuleMigrations(tmpDir, []string{"migrations/*.json"})
	ctx := context.Background()
	_, err := engine.RunUp(ctx, tmpDir, "test", migrations)
	if err == nil {
		t.Error("expected error on duplicate email with conflict=error")
	}

	var count int64
	db.Table("contact").Count(&count)
	if count != 0 {
		t.Errorf("expected 0 records after transaction rollback, got %d", count)
	}
}

func TestMigrationEngine_LazyBatch(t *testing.T) {
	db := setupTestDB(t)
	db.Exec("CREATE TABLE tag (id TEXT, name TEXT, created_at TEXT, updated_at TEXT)")
	engine := setupEngine(t, db)

	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "data"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "migrations"), 0755)

	writeJSON(t, filepath.Join(tmpDir, "data", "tags.json"), []map[string]any{{"name": "VIP"}})

	writeMigration(t, filepath.Join(tmpDir, "migrations"), "20260101_000001_tags.json", map[string]any{
		"name": "seed_tags", "model": "tag",
		"source": map[string]any{"type": "json", "file": "data/tags.json"},
	})

	migrations, _ := CollectModuleMigrations(tmpDir, []string{"migrations/*.json"})
	ctx := context.Background()

	engine.RunUp(ctx, tmpDir, "test", migrations)
	batch1 := engine.Tracker().CurrentBatch()

	engine.RunUp(ctx, tmpDir, "test", migrations)
	batch2 := engine.Tracker().CurrentBatch()

	if batch2 != batch1 {
		t.Errorf("batch should not increment when all migrations skipped: batch1=%d batch2=%d", batch1, batch2)
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
	tracker.Record("crm", "seed_tags", "tag", "json", 5, 1, 0, nil, nil)
	tracker.Record("crm", "seed_contacts", "contact", "csv", 10, 1, 0, nil, nil)

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
	tracker.Record("crm", "seed_tags", "tag", "json", 5, 1, 0, nil, nil)

	pending := tracker.GetPending("crm", []string{"seed_tags", "seed_contacts", "seed_leads"})
	if len(pending) != 2 {
		t.Fatalf("expected 2 pending, got %d", len(pending))
	}
}

func TestMigrationTracker_RecordIDs(t *testing.T) {
	db := setupTestDB(t)
	MigrateMigrationTable(db)

	tracker := persistence.NewMigrationTracker(db)
	ids := []string{"id-1", "id-2", "id-3"}
	tracker.Record("crm", "seed_tags", "tag", "json", 3, 1, 0, ids, nil)

	rec, err := tracker.GetByName("crm", "seed_tags")
	if err != nil {
		t.Fatalf("GetByName failed: %v", err)
	}
	gotIDs := rec.GetRecordIDs()
	if len(gotIDs) != 3 {
		t.Fatalf("expected 3 IDs, got %d", len(gotIDs))
	}
	if gotIDs[0] != "id-1" {
		t.Errorf("expected 'id-1', got %s", gotIDs[0])
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

func TestDependsOn_TopologicalSort(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "migrations"), 0755)

	writeMigration(t, filepath.Join(tmpDir, "migrations"), "20260101_000003_seed_employees.json", map[string]any{
		"name": "seed_employees", "model": "employee",
		"depends_on": []string{"seed_departments", "seed_positions"},
		"source":     map[string]any{"type": "json", "file": "data/e.json"},
	})
	writeMigration(t, filepath.Join(tmpDir, "migrations"), "20260101_000001_seed_departments.json", map[string]any{
		"name": "seed_departments", "model": "department",
		"source": map[string]any{"type": "json", "file": "data/d.json"},
	})
	writeMigration(t, filepath.Join(tmpDir, "migrations"), "20260101_000002_seed_positions.json", map[string]any{
		"name": "seed_positions", "model": "job_position",
		"depends_on": []string{"seed_departments"},
		"source":     map[string]any{"type": "json", "file": "data/p.json"},
	})

	migrations, err := CollectModuleMigrations(tmpDir, []string{"migrations/*.json"})
	if err != nil {
		t.Fatalf("discover failed: %v", err)
	}

	if len(migrations) != 3 {
		t.Fatalf("expected 3, got %d", len(migrations))
	}

	if migrations[0].Name != "seed_departments" {
		t.Errorf("expected seed_departments first, got %s", migrations[0].Name)
	}
	if migrations[1].Name != "seed_positions" {
		t.Errorf("expected seed_positions second, got %s", migrations[1].Name)
	}
	if migrations[2].Name != "seed_employees" {
		t.Errorf("expected seed_employees third, got %s", migrations[2].Name)
	}
}

func TestDependsOn_CircularDetection(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "migrations"), 0755)

	writeMigration(t, filepath.Join(tmpDir, "migrations"), "20260101_000001_a.json", map[string]any{
		"name": "a", "model": "x", "depends_on": []string{"b"},
		"source": map[string]any{"type": "json", "file": "data/a.json"},
	})
	writeMigration(t, filepath.Join(tmpDir, "migrations"), "20260101_000002_b.json", map[string]any{
		"name": "b", "model": "x", "depends_on": []string{"a"},
		"source": map[string]any{"type": "json", "file": "data/b.json"},
	})

	_, err := CollectModuleMigrations(tmpDir, []string{"migrations/*.json"})
	if err == nil {
		t.Error("expected circular dependency error")
	}
}

func TestCoerceType(t *testing.T) {
	if coerceType(int64(123), "string") != "123" {
		t.Error("coerce int to string failed")
	}
	if coerceType("00123", "string") != "00123" {
		t.Error("coerce string to string should preserve")
	}
	if coerceType("42", "int") != int64(42) {
		t.Error("coerce string to int failed")
	}
	if coerceType("true", "bool") != true {
		t.Error("coerce string to bool failed")
	}
	if coerceType("abc", "int") != "abc" {
		t.Error("coerce invalid int should return original string")
	}
}

func TestSafeFieldName(t *testing.T) {
	valid := []string{"name", "email", "user_id", "field123", "_private"}
	for _, f := range valid {
		if !safeFieldName.MatchString(f) {
			t.Errorf("expected %q to be valid", f)
		}
	}

	invalid := []string{"name; DROP TABLE", "field-name", "123start", "a.b", "x=1"}
	for _, f := range invalid {
		if safeFieldName.MatchString(f) {
			t.Errorf("expected %q to be invalid", f)
		}
	}
}

func TestGormDataInserter_RejectsUnsafeFieldNames(t *testing.T) {
	db := setupTestDB(t)
	db.Exec("CREATE TABLE contact (id TEXT, name TEXT)")
	inserter := NewGormDataInserter(db)

	ctx := context.Background()
	_, err := inserter.Exists(ctx, "contact", map[string]any{"name; DROP TABLE contact--": "x"})
	if err == nil {
		t.Error("expected error for unsafe field name in Exists")
	}

	err = inserter.Update(ctx, "contact", map[string]any{"name; DROP TABLE contact--": "x"}, map[string]any{"name": "y"})
	if err == nil {
		t.Error("expected error for unsafe field name in Update")
	}
}

func TestExtraFilesLoading(t *testing.T) {
	db := setupTestDB(t)
	db.Exec("CREATE TABLE contact (id TEXT, name TEXT, email TEXT, region TEXT, created_at TEXT, updated_at TEXT)")
	_ = setupEngine(t, db)

	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "data"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "migrations"), 0755)

	writeJSON(t, filepath.Join(tmpDir, "data", "contacts.json"), []map[string]any{
		{"name": "Alice", "email": "alice@test.com"},
	})
	writeJSON(t, filepath.Join(tmpDir, "data", "regions.json"), map[string]any{
		"default": "Asia",
	})

	writeMigration(t, filepath.Join(tmpDir, "migrations"), "20260101_000001_contacts.json", map[string]any{
		"name": "seed_contacts", "model": "contact",
		"source": map[string]any{"type": "json", "file": "data/contacts.json"},
		"processor": map[string]any{
			"type": "script",
			"script": map[string]any{"lang": "typescript", "file": "scripts/noop.ts"},
			"extra_files": map[string]string{
				"regions": "data/regions.json",
			},
		},
	})

	migrations, err := CollectModuleMigrations(tmpDir, []string{"migrations/*.json"})
	if err != nil {
		t.Fatalf("discover failed: %v", err)
	}

	if migrations[0].Def.Processor.ExtraFiles["regions"] != "data/regions.json" {
		t.Error("extra_files not parsed correctly")
	}
}
