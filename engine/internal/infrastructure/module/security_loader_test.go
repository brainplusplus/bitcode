package module

import (
	"testing"

	"github.com/bitcode-framework/bitcode/internal/compiler/parser"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupSecurityLoaderDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	sqlDB, _ := db.DB()
	sqlDB.Exec(`CREATE TABLE groups (
		id TEXT PRIMARY KEY, name TEXT UNIQUE, display_name TEXT, category TEXT,
		share INTEGER DEFAULT 0, comment TEXT, module TEXT,
		modified_source TEXT DEFAULT 'json', created_at DATETIME, updated_at DATETIME
	)`)
	sqlDB.Exec(`CREATE TABLE group_implies (
		group_id TEXT, implied_group_id TEXT, PRIMARY KEY (group_id, implied_group_id)
	)`)
	sqlDB.Exec(`CREATE TABLE model_accesses (
		id TEXT PRIMARY KEY, name TEXT, model_name TEXT, group_id TEXT,
		can_select INTEGER DEFAULT 0, can_read INTEGER DEFAULT 0,
		can_write INTEGER DEFAULT 0, can_create INTEGER DEFAULT 0,
		can_delete INTEGER DEFAULT 0, can_print INTEGER DEFAULT 0,
		can_email INTEGER DEFAULT 0, can_report INTEGER DEFAULT 0,
		can_export INTEGER DEFAULT 0, can_import INTEGER DEFAULT 0,
		can_mask INTEGER DEFAULT 0, can_clone INTEGER DEFAULT 0,
		module TEXT, modified_source TEXT DEFAULT 'json',
		created_at DATETIME, updated_at DATETIME
	)`)
	sqlDB.Exec(`CREATE TABLE record_rules (
		id TEXT PRIMARY KEY, name TEXT UNIQUE, model_name TEXT, group_names TEXT DEFAULT '',
		domain_filter TEXT, can_read INTEGER DEFAULT 1, can_create INTEGER DEFAULT 1,
		can_write INTEGER DEFAULT 1, can_delete INTEGER DEFAULT 0,
		is_global INTEGER DEFAULT 0, active INTEGER DEFAULT 1,
		module TEXT, modified_source TEXT DEFAULT 'json',
		created_at DATETIME, updated_at DATETIME
	)`)
	sqlDB.Exec(`CREATE TABLE record_rule_groups (
		record_rule_id TEXT, group_id TEXT, PRIMARY KEY (record_rule_id, group_id)
	)`)
	sqlDB.Exec(`CREATE TABLE group_menus (
		group_id TEXT, menu_item_id TEXT, module TEXT, PRIMARY KEY (group_id, menu_item_id)
	)`)
	sqlDB.Exec(`CREATE TABLE group_pages (
		group_id TEXT, page_name TEXT, module TEXT, PRIMARY KEY (group_id, page_name)
	)`)

	return db
}

func TestSecurityLoader_SyncBasicGroup(t *testing.T) {
	db := setupSecurityLoaderDB(t)
	loader := NewSecurityLoader(db)

	secDef := &parser.SecurityDefinition{
		Name:     "crm.user",
		Label:    "CRM / User",
		Category: "CRM",
		Access: map[string]parser.SecurityACL{
			"contact": {"select", "read", "write", "create"},
		},
		Menus: []string{"crm/contacts"},
		Pages: []string{"contact_list"},
	}

	if err := loader.SyncToDB(secDef, "crm"); err != nil {
		t.Fatalf("sync error: %v", err)
	}

	var count int64
	db.Table("groups").Where("name = ?", "crm.user").Count(&count)
	if count != 1 {
		t.Errorf("expected 1 group, got %d", count)
	}

	var aclCount int64
	db.Table("model_accesses").Where("model_name = ?", "contact").Count(&aclCount)
	if aclCount != 1 {
		t.Errorf("expected 1 model_access, got %d", aclCount)
	}

	var canRead bool
	db.Table("model_accesses").Select("can_read").Where("model_name = ?", "contact").Pluck("can_read", &canRead)
	if !canRead {
		t.Error("expected can_read=true for contact")
	}

	var canDelete bool
	db.Table("model_accesses").Select("can_delete").Where("model_name = ?", "contact").Pluck("can_delete", &canDelete)
	if canDelete {
		t.Error("expected can_delete=false for contact")
	}

	var menuCount int64
	db.Table("group_menus").Count(&menuCount)
	if menuCount != 1 {
		t.Errorf("expected 1 menu, got %d", menuCount)
	}

	var pageCount int64
	db.Table("group_pages").Count(&pageCount)
	if pageCount != 1 {
		t.Errorf("expected 1 page, got %d", pageCount)
	}
}

func TestSecurityLoader_SyncWithImplies(t *testing.T) {
	db := setupSecurityLoaderDB(t)
	loader := NewSecurityLoader(db)

	baseDef := &parser.SecurityDefinition{
		Name:     "base.user",
		Label:    "Base / User",
		Category: "Base",
	}
	if err := loader.SyncToDB(baseDef, "base"); err != nil {
		t.Fatalf("sync base error: %v", err)
	}

	crmDef := &parser.SecurityDefinition{
		Name:     "crm.user",
		Label:    "CRM / User",
		Category: "CRM",
		Implies:  []string{"base.user"},
	}
	if err := loader.SyncToDB(crmDef, "crm"); err != nil {
		t.Fatalf("sync crm error: %v", err)
	}

	var impliesCount int64
	db.Table("group_implies").Count(&impliesCount)
	if impliesCount != 1 {
		t.Errorf("expected 1 implied group, got %d", impliesCount)
	}
}

func TestSecurityLoader_SyncRecordRules(t *testing.T) {
	db := setupSecurityLoaderDB(t)
	loader := NewSecurityLoader(db)

	f := false
	secDef := &parser.SecurityDefinition{
		Name:     "crm.user",
		Label:    "CRM / User",
		Category: "CRM",
		Rules: []parser.SecurityRuleDefinition{
			{
				Name:       "crm_user_own_contacts",
				Model:      "contact",
				Domain:     [][]any{{"created_by", "=", "{{user.id}}"}},
				PermDelete: &f,
			},
		},
	}

	if err := loader.SyncToDB(secDef, "crm"); err != nil {
		t.Fatalf("sync error: %v", err)
	}

	var ruleCount int64
	db.Table("record_rules").Count(&ruleCount)
	if ruleCount != 1 {
		t.Errorf("expected 1 record rule, got %d", ruleCount)
	}

	var canDelete bool
	db.Table("record_rules").Select("can_delete").Where("name = ?", "crm_user_own_contacts").Pluck("can_delete", &canDelete)
	if canDelete {
		t.Error("expected can_delete=false")
	}

	var canRead bool
	db.Table("record_rules").Select("can_read").Where("name = ?", "crm_user_own_contacts").Pluck("can_read", &canRead)
	if !canRead {
		t.Error("expected can_read=true (default)")
	}

	var rrGroupCount int64
	db.Table("record_rule_groups").Count(&rrGroupCount)
	if rrGroupCount != 1 {
		t.Errorf("expected 1 record_rule_group, got %d", rrGroupCount)
	}
}

func TestSecurityLoader_UIModifiedNotOverwritten(t *testing.T) {
	db := setupSecurityLoaderDB(t)
	loader := NewSecurityLoader(db)

	secDef := &parser.SecurityDefinition{
		Name:     "crm.user",
		Label:    "CRM / User",
		Category: "CRM",
	}
	if err := loader.SyncToDB(secDef, "crm"); err != nil {
		t.Fatalf("first sync error: %v", err)
	}

	db.Table("groups").Where("name = ?", "crm.user").Update("modified_source", "ui")
	db.Table("groups").Where("name = ?", "crm.user").Update("display_name", "CRM / User (Custom)")

	secDef.Label = "CRM / User (Overwritten)"
	if err := loader.SyncToDB(secDef, "crm"); err != nil {
		t.Fatalf("second sync error: %v", err)
	}

	var displayName string
	db.Table("groups").Select("display_name").Where("name = ?", "crm.user").Pluck("display_name", &displayName)
	if displayName != "CRM / User (Custom)" {
		t.Errorf("expected UI-modified name preserved, got %q", displayName)
	}
}

func TestSecurityLoader_AllPermissions(t *testing.T) {
	db := setupSecurityLoaderDB(t)
	loader := NewSecurityLoader(db)

	secDef := &parser.SecurityDefinition{
		Name:     "crm.manager",
		Label:    "CRM / Manager",
		Category: "CRM",
		Access: map[string]parser.SecurityACL{
			"contact": {"select", "read", "write", "create", "delete", "print", "email", "report", "export", "import", "mask", "clone"},
		},
	}

	if err := loader.SyncToDB(secDef, "crm"); err != nil {
		t.Fatalf("sync error: %v", err)
	}

	var row struct {
		CanSelect, CanRead, CanWrite, CanCreate, CanDelete bool
		CanPrint, CanEmail, CanReport                      bool
		CanExport, CanImport, CanMask, CanClone            bool
	}
	db.Table("model_accesses").
		Select("can_select, can_read, can_write, can_create, can_delete, can_print, can_email, can_report, can_export, can_import, can_mask, can_clone").
		Where("model_name = ?", "contact").
		First(&row)

	if !row.CanSelect || !row.CanRead || !row.CanWrite || !row.CanCreate || !row.CanDelete ||
		!row.CanPrint || !row.CanEmail || !row.CanReport ||
		!row.CanExport || !row.CanImport || !row.CanMask || !row.CanClone {
		t.Error("expected all 12 permissions to be true")
	}
}

func TestSecurityLoader_IdempotentSync(t *testing.T) {
	db := setupSecurityLoaderDB(t)
	loader := NewSecurityLoader(db)

	secDef := &parser.SecurityDefinition{
		Name:     "crm.user",
		Label:    "CRM / User",
		Category: "CRM",
		Access: map[string]parser.SecurityACL{
			"contact": {"select", "read"},
		},
	}

	if err := loader.SyncToDB(secDef, "crm"); err != nil {
		t.Fatalf("first sync: %v", err)
	}
	if err := loader.SyncToDB(secDef, "crm"); err != nil {
		t.Fatalf("second sync: %v", err)
	}

	var groupCount int64
	db.Table("groups").Where("name = ?", "crm.user").Count(&groupCount)
	if groupCount != 1 {
		t.Errorf("expected 1 group after idempotent sync, got %d", groupCount)
	}
}
