package websocket

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/bitcode-framework/bitcode/internal/compiler/parser"
	"github.com/bitcode-framework/bitcode/internal/infrastructure/persistence"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type mockModelRegistry struct {
	models map[string]*parser.ModelDefinition
}

func (m *mockModelRegistry) Get(name string) (*parser.ModelDefinition, error) {
	if md, ok := m.models[name]; ok {
		return md, nil
	}
	return nil, nil
}

func (m *mockModelRegistry) TableName(name string) string {
	return name + "s"
}

func setupWSTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	sqlDB, _ := db.DB()
	sqlDB.Exec(`CREATE TABLE contacts (
		id TEXT PRIMARY KEY,
		name TEXT,
		email TEXT,
		created_by TEXT,
		updated_by TEXT,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	)`)
	return db
}

func TestCRUDHandler_Create(t *testing.T) {
	db := setupWSTestDB(t)
	reg := &mockModelRegistry{models: map[string]*parser.ModelDefinition{
		"contact": {Name: "contact", Module: "crm", Fields: map[string]parser.FieldDefinition{
			"name": {Type: parser.FieldString}, "email": {Type: parser.FieldEmail},
		}},
	}}

	handler := NewCRUDHandler(db, reg, nil, nil)
	handler.EnableModel("contact")

	resp := handler.handleCreate(&Client{UserID: "user-1"}, CRUDRequest{
		ID: "req-1", Model: "contact", Action: "create",
		Data: map[string]any{"name": "John", "email": "john@test.com"},
	})

	if !resp.Success {
		t.Fatalf("expected success, got error: %s", resp.Error)
	}

	var count int64
	db.Table("contacts").Count(&count)
	if count != 1 {
		t.Errorf("expected 1 record created, got %d", count)
	}
}

func TestCRUDHandler_List(t *testing.T) {
	db := setupWSTestDB(t)
	sqlDB, _ := db.DB()
	sqlDB.Exec(`INSERT INTO contacts (id, name, email) VALUES ('c-1', 'John', 'john@test.com')`)
	sqlDB.Exec(`INSERT INTO contacts (id, name, email) VALUES ('c-2', 'Jane', 'jane@test.com')`)

	reg := &mockModelRegistry{models: map[string]*parser.ModelDefinition{
		"contact": {Name: "contact", Module: "crm", Fields: map[string]parser.FieldDefinition{
			"name": {Type: parser.FieldString}, "email": {Type: parser.FieldEmail},
		}},
	}}

	handler := NewCRUDHandler(db, reg, nil, nil)
	handler.EnableModel("contact")

	resp := handler.handleList(&Client{UserID: "user-1"}, CRUDRequest{
		ID: "req-1", Model: "contact", Action: "list", Page: 1, PageSize: 10,
	})

	if !resp.Success {
		t.Fatalf("expected success, got error: %s", resp.Error)
	}

	data, ok := resp.Data.(map[string]any)
	if !ok {
		t.Fatal("expected map data")
	}
	total, _ := data["total"].(int64)
	if total != 2 {
		t.Errorf("expected total=2, got %d", total)
	}
}

func TestCRUDHandler_Read(t *testing.T) {
	db := setupWSTestDB(t)
	sqlDB, _ := db.DB()
	sqlDB.Exec(`INSERT INTO contacts (id, name, email) VALUES ('c-1', 'John', 'john@test.com')`)

	reg := &mockModelRegistry{models: map[string]*parser.ModelDefinition{
		"contact": {Name: "contact", Module: "crm"},
	}}

	handler := NewCRUDHandler(db, reg, nil, nil)
	handler.EnableModel("contact")

	resp := handler.handleRead(&Client{UserID: "user-1"}, CRUDRequest{
		ID: "req-1", Model: "contact", Action: "read", RecordID: "c-1",
	})

	if !resp.Success {
		t.Fatalf("expected success, got error: %s", resp.Error)
	}

	record, ok := resp.Data.(map[string]any)
	if !ok {
		t.Fatal("expected map data")
	}
	if record["name"] != "John" {
		t.Errorf("expected name=John, got %v", record["name"])
	}
}

func TestCRUDHandler_Update(t *testing.T) {
	db := setupWSTestDB(t)
	sqlDB, _ := db.DB()
	sqlDB.Exec(`INSERT INTO contacts (id, name, email) VALUES ('c-1', 'John', 'john@test.com')`)

	reg := &mockModelRegistry{models: map[string]*parser.ModelDefinition{
		"contact": {Name: "contact", Module: "crm"},
	}}

	handler := NewCRUDHandler(db, reg, nil, nil)
	handler.EnableModel("contact")

	resp := handler.handleUpdate(&Client{UserID: "user-1"}, CRUDRequest{
		ID: "req-1", Model: "contact", Action: "update", RecordID: "c-1",
		Data: map[string]any{"name": "John Updated"},
	})

	if !resp.Success {
		t.Fatalf("expected success, got error: %s", resp.Error)
	}

	var name string
	db.Table("contacts").Select("name").Where("id = ?", "c-1").Pluck("name", &name)
	if name != "John Updated" {
		t.Errorf("expected updated name, got %q", name)
	}
}

func TestCRUDHandler_Delete(t *testing.T) {
	db := setupWSTestDB(t)
	sqlDB, _ := db.DB()
	sqlDB.Exec(`INSERT INTO contacts (id, name, email) VALUES ('c-1', 'John', 'john@test.com')`)

	reg := &mockModelRegistry{models: map[string]*parser.ModelDefinition{
		"contact": {Name: "contact", Module: "crm"},
	}}

	handler := NewCRUDHandler(db, reg, nil, nil)
	handler.EnableModel("contact")

	resp := handler.handleDelete(&Client{UserID: "user-1"}, CRUDRequest{
		ID: "req-1", Model: "contact", Action: "delete", RecordID: "c-1",
	})

	if !resp.Success {
		t.Fatalf("expected success, got error: %s", resp.Error)
	}

	var count int64
	db.Table("contacts").Count(&count)
	if count != 0 {
		t.Errorf("expected 0 records after delete, got %d", count)
	}
}

func TestCRUDHandler_ModelNotEnabled(t *testing.T) {
	db := setupWSTestDB(t)
	reg := &mockModelRegistry{models: map[string]*parser.ModelDefinition{
		"contact": {Name: "contact", Module: "crm"},
	}}

	handler := NewCRUDHandler(db, reg, nil, nil)

	if handler.enabledModels["contact"] {
		t.Error("contact should not be enabled by default")
	}

	handler.EnableModel("contact")
	if !handler.enabledModels["contact"] {
		t.Error("contact should be enabled after EnableModel")
	}
}

func TestCRUDHandler_PermissionDenied(t *testing.T) {
	db := setupWSTestDB(t)
	sqlDB, _ := db.DB()
	sqlDB.Exec(`CREATE TABLE "user" (id TEXT PRIMARY KEY, username TEXT, is_superuser INTEGER DEFAULT 0)`)
	sqlDB.Exec(`INSERT INTO "user" (id, username) VALUES ('user-1', 'john')`)
	sqlDB.Exec(`CREATE TABLE user_group (user_id TEXT, group_id TEXT)`)
	sqlDB.Exec(`CREATE TABLE group_implies (group_id TEXT, implied_group_id TEXT)`)
	sqlDB.Exec(`CREATE TABLE model_access (id TEXT PRIMARY KEY, name TEXT, model_name TEXT, group_id TEXT, can_read INTEGER DEFAULT 0, can_select INTEGER DEFAULT 0, can_write INTEGER DEFAULT 0, can_create INTEGER DEFAULT 0, can_delete INTEGER DEFAULT 0, can_print INTEGER DEFAULT 0, can_email INTEGER DEFAULT 0, can_report INTEGER DEFAULT 0, can_export INTEGER DEFAULT 0, can_import INTEGER DEFAULT 0, can_mask INTEGER DEFAULT 0, can_clone INTEGER DEFAULT 0, module TEXT, modified_source TEXT)`)

	reg := &mockModelRegistry{models: map[string]*parser.ModelDefinition{
		"contact": {Name: "contact", Module: "crm"},
	}}

	permSvc := persistence.NewPermissionService(db)
	handler := NewCRUDHandler(db, reg, permSvc, nil)
	handler.EnableModel("contact")

	resp := handler.handleList(&Client{UserID: "user-1"}, CRUDRequest{
		ID: "req-1", Model: "contact", Action: "list",
	})

	if resp.Success {
		t.Error("expected permission denied (no ACL for contact)")
	}
	if resp.Error == "" {
		t.Error("expected error message")
	}
}
