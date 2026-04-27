package persistence

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupRecordRuleTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	sqlDB, _ := db.DB()
	sqlDB.Exec(`CREATE TABLE "user" (
		id TEXT PRIMARY KEY,
		username TEXT,
		is_superuser INTEGER DEFAULT 0
	)`)
	sqlDB.Exec(`CREATE TABLE "group" (
		id TEXT PRIMARY KEY,
		name TEXT UNIQUE,
		display_name TEXT,
		category TEXT
	)`)
	sqlDB.Exec(`CREATE TABLE user_group (
		user_id TEXT,
		group_id TEXT,
		PRIMARY KEY (user_id, group_id)
	)`)
	sqlDB.Exec(`CREATE TABLE group_implies (
		group_id TEXT,
		implied_group_id TEXT,
		PRIMARY KEY (group_id, implied_group_id)
	)`)
	sqlDB.Exec(`CREATE TABLE record_rule (
		id TEXT PRIMARY KEY,
		name TEXT,
		model_name TEXT,
		group_names TEXT DEFAULT '',
		domain_filter TEXT,
		can_read INTEGER DEFAULT 1,
		can_create INTEGER DEFAULT 1,
		can_write INTEGER DEFAULT 1,
		can_delete INTEGER DEFAULT 0,
		is_global INTEGER DEFAULT 0,
		active INTEGER DEFAULT 1,
		module TEXT,
		modified_source TEXT DEFAULT 'json'
	)`)
	sqlDB.Exec(`CREATE TABLE record_rule_groups (
		record_rule_id TEXT,
		group_id TEXT,
		PRIMARY KEY (record_rule_id, group_id)
	)`)

	return db
}

func TestRecordRuleService_SuperuserBypass(t *testing.T) {
	db := setupRecordRuleTestDB(t)
	sqlDB, _ := db.DB()
	sqlDB.Exec(`INSERT INTO "user" (id, username, is_superuser) VALUES ('u-1', 'admin', 1)`)
	sqlDB.Exec(`INSERT INTO record_rule (id, name, model_name, domain_filter, active) VALUES ('r-1', 'strict_rule', 'contact', '[["id","=","impossible"]]', 1)`)

	svc := NewRecordRuleService(db)
	filters, err := svc.GetFilters("u-1", "contact", "read")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if filters != nil {
		t.Errorf("expected nil filters for superuser, got %v", filters)
	}
}

func TestRecordRuleService_NoRules_DefaultAllow(t *testing.T) {
	db := setupRecordRuleTestDB(t)
	sqlDB, _ := db.DB()
	sqlDB.Exec(`INSERT INTO "user" (id, username, is_superuser) VALUES ('u-1', 'john', 0)`)

	svc := NewRecordRuleService(db)
	filters, err := svc.GetFilters("u-1", "contact", "read")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if filters != nil {
		t.Errorf("expected nil filters when no rules exist, got %v", filters)
	}
}

func TestRecordRuleService_GlobalRule(t *testing.T) {
	db := setupRecordRuleTestDB(t)
	sqlDB, _ := db.DB()
	sqlDB.Exec(`INSERT INTO "user" (id, username, is_superuser) VALUES ('u-1', 'john', 0)`)
	sqlDB.Exec(`INSERT INTO record_rule (id, name, model_name, domain_filter, can_read, active) VALUES ('r-1', 'active_only', 'contact', '[["active","=",true]]', 1, 1)`)

	svc := NewRecordRuleService(db)
	filters, err := svc.GetFilters("u-1", "contact", "read")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(filters) != 1 {
		t.Fatalf("expected 1 filter, got %d: %v", len(filters), filters)
	}
	if filters[0][0] != "active" {
		t.Errorf("expected filter field 'active', got %v", filters[0][0])
	}
}

func TestRecordRuleService_GroupRule_UserInGroup(t *testing.T) {
	db := setupRecordRuleTestDB(t)
	sqlDB, _ := db.DB()
	sqlDB.Exec(`INSERT INTO "user" (id, username, is_superuser) VALUES ('u-1', 'john', 0)`)
	sqlDB.Exec(`INSERT INTO "group" (id, name) VALUES ('g-1', 'crm.user')`)
	sqlDB.Exec(`INSERT INTO user_group (user_id, group_id) VALUES ('u-1', 'g-1')`)
	sqlDB.Exec(`INSERT INTO record_rule (id, name, model_name, domain_filter, can_read, active) VALUES ('r-1', 'own_contacts', 'contact', '[["created_by","=","{{user.id}}"]]', 1, 1)`)
	sqlDB.Exec(`INSERT INTO record_rule_groups (record_rule_id, group_id) VALUES ('r-1', 'g-1')`)

	svc := NewRecordRuleService(db)
	filters, err := svc.GetFilters("u-1", "contact", "read")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(filters) != 1 {
		t.Fatalf("expected 1 filter, got %d", len(filters))
	}
	if filters[0][0] != "created_by" {
		t.Errorf("expected filter field 'created_by', got %v", filters[0][0])
	}
}

func TestRecordRuleService_GroupRule_UserNotInGroup(t *testing.T) {
	db := setupRecordRuleTestDB(t)
	sqlDB, _ := db.DB()
	sqlDB.Exec(`INSERT INTO "user" (id, username, is_superuser) VALUES ('u-1', 'john', 0)`)
	sqlDB.Exec(`INSERT INTO "group" (id, name) VALUES ('g-1', 'crm.user'), ('g-2', 'sales.user')`)
	sqlDB.Exec(`INSERT INTO user_group (user_id, group_id) VALUES ('u-1', 'g-2')`)
	sqlDB.Exec(`INSERT INTO record_rule (id, name, model_name, domain_filter, can_read, active) VALUES ('r-1', 'crm_only', 'contact', '[["created_by","=","{{user.id}}"]]', 1, 1)`)
	sqlDB.Exec(`INSERT INTO record_rule_groups (record_rule_id, group_id) VALUES ('r-1', 'g-1')`)

	svc := NewRecordRuleService(db)
	filters, err := svc.GetFilters("u-1", "contact", "read")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if filters != nil {
		t.Errorf("expected nil filters (user not in rule's group), got %v", filters)
	}
}

func TestRecordRuleService_OperationFiltering(t *testing.T) {
	db := setupRecordRuleTestDB(t)
	sqlDB, _ := db.DB()
	sqlDB.Exec(`INSERT INTO "user" (id, username, is_superuser) VALUES ('u-1', 'john', 0)`)
	sqlDB.Exec(`INSERT INTO record_rule (id, name, model_name, domain_filter, can_read, can_write, can_create, can_delete, active) VALUES ('r-1', 'read_only_rule', 'contact', '[["active","=",true]]', 1, 0, 0, 0, 1)`)

	svc := NewRecordRuleService(db)

	readFilters, _ := svc.GetFilters("u-1", "contact", "read")
	if len(readFilters) != 1 {
		t.Errorf("expected 1 filter for read, got %d", len(readFilters))
	}

	writeFilters, _ := svc.GetFilters("u-1", "contact", "write")
	if writeFilters != nil {
		t.Errorf("expected nil filters for write (rule doesn't apply), got %v", writeFilters)
	}

	deleteFilters, _ := svc.GetFilters("u-1", "contact", "delete")
	if deleteFilters != nil {
		t.Errorf("expected nil filters for delete, got %v", deleteFilters)
	}
}

func TestRecordRuleService_ImpliedGroups(t *testing.T) {
	db := setupRecordRuleTestDB(t)
	sqlDB, _ := db.DB()
	sqlDB.Exec(`INSERT INTO "user" (id, username, is_superuser) VALUES ('u-1', 'john', 0)`)
	sqlDB.Exec(`INSERT INTO "group" (id, name) VALUES ('g-1', 'base.user'), ('g-2', 'crm.user')`)
	sqlDB.Exec(`INSERT INTO group_implies (group_id, implied_group_id) VALUES ('g-2', 'g-1')`)
	sqlDB.Exec(`INSERT INTO user_group (user_id, group_id) VALUES ('u-1', 'g-2')`)
	sqlDB.Exec(`INSERT INTO record_rule (id, name, model_name, domain_filter, can_read, active) VALUES ('r-1', 'base_rule', 'user', '[["active","=",true]]', 1, 1)`)
	sqlDB.Exec(`INSERT INTO record_rule_groups (record_rule_id, group_id) VALUES ('r-1', 'g-1')`)

	svc := NewRecordRuleService(db)
	filters, err := svc.GetFilters("u-1", "user", "read")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(filters) != 1 {
		t.Fatalf("expected 1 filter via implied group, got %d", len(filters))
	}
}

func TestRecordRuleService_LegacyGroupNames(t *testing.T) {
	db := setupRecordRuleTestDB(t)
	sqlDB, _ := db.DB()
	sqlDB.Exec(`INSERT INTO "user" (id, username, is_superuser) VALUES ('u-1', 'john', 0)`)
	sqlDB.Exec(`INSERT INTO "group" (id, name) VALUES ('g-1', 'crm.user')`)
	sqlDB.Exec(`INSERT INTO user_group (user_id, group_id) VALUES ('u-1', 'g-1')`)
	sqlDB.Exec(`INSERT INTO record_rule (id, name, model_name, group_names, domain_filter, can_read, active) VALUES ('r-1', 'legacy_rule', 'contact', 'crm.user', '[["created_by","=","{{user.id}}"]]', 1, 1)`)

	svc := NewRecordRuleService(db)
	filters, err := svc.GetFilters("u-1", "contact", "read")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(filters) != 1 {
		t.Fatalf("expected 1 filter via legacy group_names, got %d", len(filters))
	}
}

func TestRecordRuleService_InactiveRuleIgnored(t *testing.T) {
	db := setupRecordRuleTestDB(t)
	sqlDB, _ := db.DB()
	sqlDB.Exec(`INSERT INTO "user" (id, username, is_superuser) VALUES ('u-1', 'john', 0)`)
	sqlDB.Exec(`INSERT INTO record_rule (id, name, model_name, domain_filter, can_read, active) VALUES ('r-1', 'inactive', 'contact', '[["id","=","1"]]', 1, 0)`)

	svc := NewRecordRuleService(db)
	filters, _ := svc.GetFilters("u-1", "contact", "read")
	if filters != nil {
		t.Errorf("expected nil filters for inactive rule, got %v", filters)
	}
}

func TestInterpolateDomainFilters(t *testing.T) {
	filters := [][]any{
		{"created_by", "=", "{{user.id}}"},
		{"company_id", "=", "{{user.company_id}}"},
		{"status", "=", "active"},
	}
	vars := map[string]string{
		"user.id":         "user-123",
		"user.company_id": "comp-456",
	}

	result := InterpolateDomainFilters(filters, vars)

	if result[0][2] != "user-123" {
		t.Errorf("expected user.id interpolated, got %v", result[0][2])
	}
	if result[1][2] != "comp-456" {
		t.Errorf("expected company_id interpolated, got %v", result[1][2])
	}
	if result[2][2] != "active" {
		t.Errorf("expected 'active' unchanged, got %v", result[2][2])
	}

	if filters[0][2] != "{{user.id}}" {
		t.Error("original filters should not be mutated")
	}
}

func TestInterpolateDomainFilters_NilInput(t *testing.T) {
	result := InterpolateDomainFilters(nil, map[string]string{"user.id": "123"})
	if result != nil {
		t.Errorf("expected nil for nil input, got %v", result)
	}
}
