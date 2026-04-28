package persistence

import (
	"context"
	"testing"

	"github.com/bitcode-framework/bitcode/internal/compiler/parser"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	return db
}

func boolPtr(v bool) *bool { return &v }

func TestMigrateModelWithTenantEnabled(t *testing.T) {
	db := openTestDB(t)
	model := &parser.ModelDefinition{
		Name: "lead",
		Fields: map[string]parser.FieldDefinition{
			"name": {Type: parser.FieldString},
		},
	}

	err := MigrateModel(db, model, nil, true)
	if err != nil {
		t.Fatalf("MigrateModel failed: %v", err)
	}

	var count int64
	db.Raw("SELECT COUNT(*) FROM pragma_table_info('lead') WHERE name='tenant_id'").Scan(&count)
	if count != 1 {
		t.Error("expected tenant_id column to be created when tenant enabled")
	}

	var idxCount int64
	db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='idx_lead_tenant_id'").Scan(&idxCount)
	if idxCount != 1 {
		t.Error("expected tenant_id index to be created")
	}
}

func TestMigrateModelWithTenantDisabled(t *testing.T) {
	db := openTestDB(t)
	model := &parser.ModelDefinition{
		Name: "lead",
		Fields: map[string]parser.FieldDefinition{
			"name": {Type: parser.FieldString},
		},
	}

	err := MigrateModel(db, model, nil, false)
	if err != nil {
		t.Fatalf("MigrateModel failed: %v", err)
	}

	var count int64
	db.Raw("SELECT COUNT(*) FROM pragma_table_info('lead') WHERE name='tenant_id'").Scan(&count)
	if count != 0 {
		t.Error("expected NO tenant_id column when tenant disabled")
	}
}

func TestMigrateModelTenantScopedFalse(t *testing.T) {
	db := openTestDB(t)
	model := &parser.ModelDefinition{
		Name:         "plan",
		TenantScoped: boolPtr(false),
		Fields: map[string]parser.FieldDefinition{
			"name": {Type: parser.FieldString},
		},
	}

	err := MigrateModel(db, model, nil, true)
	if err != nil {
		t.Fatalf("MigrateModel failed: %v", err)
	}

	var count int64
	db.Raw("SELECT COUNT(*) FROM pragma_table_info('plan') WHERE name='tenant_id'").Scan(&count)
	if count != 0 {
		t.Error("expected NO tenant_id column for tenant_scoped=false model")
	}
}

func TestMigrateModelAlterTableAddTenantID(t *testing.T) {
	db := openTestDB(t)
	model := &parser.ModelDefinition{
		Name: "contact",
		Fields: map[string]parser.FieldDefinition{
			"email": {Type: parser.FieldEmail},
		},
	}

	err := MigrateModel(db, model, nil, false)
	if err != nil {
		t.Fatalf("first MigrateModel failed: %v", err)
	}

	var countBefore int64
	db.Raw("SELECT COUNT(*) FROM pragma_table_info('contact') WHERE name='tenant_id'").Scan(&countBefore)
	if countBefore != 0 {
		t.Fatal("expected no tenant_id before enabling tenant")
	}

	err = MigrateModel(db, model, nil, true)
	if err != nil {
		t.Fatalf("second MigrateModel failed: %v", err)
	}

	var countAfter int64
	db.Raw("SELECT COUNT(*) FROM pragma_table_info('contact') WHERE name='tenant_id'").Scan(&countAfter)
	if countAfter != 1 {
		t.Error("expected tenant_id column added via ALTER TABLE")
	}
}

func TestRepositoryConditionalTenantFilter(t *testing.T) {
	db := openTestDB(t)

	scopedModel := &parser.ModelDefinition{
		Name: "lead",
		Fields: map[string]parser.FieldDefinition{
			"name": {Type: parser.FieldString},
		},
	}
	MigrateModel(db, scopedModel, nil, true)

	repo := NewGenericRepositoryWithModelAndTenant(db, "lead", scopedModel, "tenant-a")

	repo.Create(context.Background(), map[string]any{"name": "Deal A"})
	repo.Create(context.Background(), map[string]any{"name": "Deal B"})

	repoB := NewGenericRepositoryWithModelAndTenant(db, "lead", scopedModel, "tenant-b")
	repoB.Create(context.Background(), map[string]any{"name": "Deal C"})

	results, total, err := repo.FindAll(context.Background(), NewQuery(), 1, 100)
	if err != nil {
		t.Fatalf("FindAll failed: %v", err)
	}
	if total != 2 {
		t.Errorf("expected 2 records for tenant-a, got %d", total)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results for tenant-a, got %d", len(results))
	}

	resultsB, totalB, _ := repoB.FindAll(context.Background(), NewQuery(), 1, 100)
	if totalB != 1 {
		t.Errorf("expected 1 record for tenant-b, got %d", totalB)
	}
	if len(resultsB) != 1 {
		t.Errorf("expected 1 result for tenant-b, got %d", len(resultsB))
	}
}

func TestRepositoryNonScopedModelSkipsTenantFilter(t *testing.T) {
	db := openTestDB(t)

	nonScopedModel := &parser.ModelDefinition{
		Name:         "plan",
		TenantScoped: boolPtr(false),
		Fields: map[string]parser.FieldDefinition{
			"name": {Type: parser.FieldString},
		},
	}
	MigrateModel(db, nonScopedModel, nil, false)

	repoA := NewGenericRepositoryWithModelAndTenant(db, "plan", nonScopedModel, "tenant-a")
	repoA.Create(context.Background(), map[string]any{"name": "Free Plan"})

	repoB := NewGenericRepositoryWithModelAndTenant(db, "plan", nonScopedModel, "tenant-b")
	repoB.Create(context.Background(), map[string]any{"name": "Pro Plan"})

	results, total, err := repoA.FindAll(context.Background(), NewQuery(), 1, 100)
	if err != nil {
		t.Fatalf("FindAll failed: %v", err)
	}
	if total != 2 {
		t.Errorf("expected 2 records (no tenant filter for non-scoped), got %d", total)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}

func TestRepositoryCreateDoesNotSetTenantIDForNonScoped(t *testing.T) {
	db := openTestDB(t)

	nonScopedModel := &parser.ModelDefinition{
		Name:         "setting",
		TenantScoped: boolPtr(false),
		Fields: map[string]parser.FieldDefinition{
			"key":   {Type: parser.FieldString},
			"value": {Type: parser.FieldString},
		},
	}
	MigrateModel(db, nonScopedModel, nil, false)

	repo := NewGenericRepositoryWithModelAndTenant(db, "setting", nonScopedModel, "tenant-x")
	record, err := repo.Create(context.Background(), map[string]any{"key": "theme", "value": "dark"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if _, hasTenant := record["tenant_id"]; hasTenant {
		t.Error("expected no tenant_id field for non-scoped model")
	}
}

func TestIsTenantScopedDefaults(t *testing.T) {
	m1 := &parser.ModelDefinition{Name: "lead"}
	if !m1.IsTenantScoped() {
		t.Error("expected default IsTenantScoped() = true")
	}

	m2 := &parser.ModelDefinition{Name: "plan", TenantScoped: boolPtr(false)}
	if m2.IsTenantScoped() {
		t.Error("expected IsTenantScoped() = false when explicitly set")
	}

	m3 := &parser.ModelDefinition{Name: "lead", TenantScoped: boolPtr(true)}
	if !m3.IsTenantScoped() {
		t.Error("expected IsTenantScoped() = true when explicitly set")
	}
}
