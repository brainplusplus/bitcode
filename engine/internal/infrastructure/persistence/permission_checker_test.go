package persistence

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupPermissionTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	sqlDB, _ := db.DB()
	sqlDB.Exec(`CREATE TABLE users (
		id TEXT PRIMARY KEY,
		username TEXT,
		email TEXT,
		password_hash TEXT,
		active INTEGER DEFAULT 1,
		is_superuser INTEGER DEFAULT 0
	)`)
	sqlDB.Exec(`CREATE TABLE groups (
		id TEXT PRIMARY KEY,
		name TEXT UNIQUE,
		display_name TEXT,
		category TEXT
	)`)
	sqlDB.Exec(`CREATE TABLE user_groups (
		user_id TEXT,
		group_id TEXT,
		PRIMARY KEY (user_id, group_id)
	)`)
	sqlDB.Exec(`CREATE TABLE group_implies (
		group_id TEXT,
		implied_group_id TEXT,
		PRIMARY KEY (group_id, implied_group_id)
	)`)
	sqlDB.Exec(`CREATE TABLE model_accesses (
		id TEXT PRIMARY KEY,
		name TEXT,
		model_name TEXT,
		group_id TEXT DEFAULT '',
		can_select INTEGER DEFAULT 0,
		can_read INTEGER DEFAULT 0,
		can_write INTEGER DEFAULT 0,
		can_create INTEGER DEFAULT 0,
		can_delete INTEGER DEFAULT 0,
		can_print INTEGER DEFAULT 0,
		can_email INTEGER DEFAULT 0,
		can_report INTEGER DEFAULT 0,
		can_export INTEGER DEFAULT 0,
		can_import INTEGER DEFAULT 0,
		can_mask INTEGER DEFAULT 0,
		can_clone INTEGER DEFAULT 0,
		module TEXT
	)`)

	return db
}

func TestPermissionService_SuperuserBypass(t *testing.T) {
	db := setupPermissionTestDB(t)
	svc := NewPermissionService(db)

	db.Exec("INSERT INTO users (id, username, email, is_superuser) VALUES (?, ?, ?, ?)", "u1", "admin", "admin@test.com", true)

	perms, err := svc.GetModelPermissions("u1", "contact")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ops := []string{"select", "read", "write", "create", "delete", "print", "email", "report", "export", "import", "mask", "clone"}
	for _, op := range ops {
		if !perms.Has(op) {
			t.Errorf("superuser should have %s permission", op)
		}
	}
}

func TestPermissionService_NoACL_DefaultDeny(t *testing.T) {
	db := setupPermissionTestDB(t)
	svc := NewPermissionService(db)

	db.Exec("INSERT INTO users (id, username, email, is_superuser) VALUES (?, ?, ?, ?)", "u1", "user1", "user1@test.com", false)

	perms, err := svc.GetModelPermissions("u1", "contact")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ops := []string{"select", "read", "write", "create", "delete", "print", "email", "report", "export", "import", "mask", "clone"}
	for _, op := range ops {
		if perms.Has(op) {
			t.Errorf("no ACL should deny %s permission", op)
		}
	}
}

func TestPermissionService_SingleGroupAccess(t *testing.T) {
	db := setupPermissionTestDB(t)
	svc := NewPermissionService(db)

	db.Exec("INSERT INTO users (id, username, email, is_superuser) VALUES (?, ?, ?, ?)", "u1", "user1", "user1@test.com", false)
	db.Exec("INSERT INTO groups (id, name) VALUES (?, ?)", "g1", "CRM/User")
	db.Exec("INSERT INTO user_groups (user_id, group_id) VALUES (?, ?)", "u1", "g1")
	db.Exec("INSERT INTO model_accesses (id, name, model_name, group_id, can_read, can_write) VALUES (?, ?, ?, ?, ?, ?)",
		"ma1", "contact_crm_user", "contact", "g1", true, true)

	perms, err := svc.GetModelPermissions("u1", "contact")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !perms.CanRead {
		t.Error("expected can_read = true")
	}
	if !perms.CanWrite {
		t.Error("expected can_write = true")
	}
	if perms.CanDelete {
		t.Error("expected can_delete = false")
	}
	if perms.CanCreate {
		t.Error("expected can_create = false")
	}
}

func TestPermissionService_AdditiveAcrossGroups(t *testing.T) {
	db := setupPermissionTestDB(t)
	svc := NewPermissionService(db)

	db.Exec("INSERT INTO users (id, username, email, is_superuser) VALUES (?, ?, ?, ?)", "u1", "user1", "user1@test.com", false)
	db.Exec("INSERT INTO groups (id, name) VALUES (?, ?)", "g1", "CRM/Reader")
	db.Exec("INSERT INTO groups (id, name) VALUES (?, ?)", "g2", "CRM/Writer")
	db.Exec("INSERT INTO user_groups (user_id, group_id) VALUES (?, ?)", "u1", "g1")
	db.Exec("INSERT INTO user_groups (user_id, group_id) VALUES (?, ?)", "u1", "g2")
	db.Exec("INSERT INTO model_accesses (id, name, model_name, group_id, can_read) VALUES (?, ?, ?, ?, ?)",
		"ma1", "contact_reader", "contact", "g1", true)
	db.Exec("INSERT INTO model_accesses (id, name, model_name, group_id, can_write) VALUES (?, ?, ?, ?, ?)",
		"ma2", "contact_writer", "contact", "g2", true)

	perms, err := svc.GetModelPermissions("u1", "contact")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !perms.CanRead {
		t.Error("expected can_read = true (from g1)")
	}
	if !perms.CanWrite {
		t.Error("expected can_write = true (from g2)")
	}
	if perms.CanDelete {
		t.Error("expected can_delete = false (neither group grants it)")
	}
}

func TestPermissionService_GlobalACL(t *testing.T) {
	db := setupPermissionTestDB(t)
	svc := NewPermissionService(db)

	db.Exec("INSERT INTO users (id, username, email, is_superuser) VALUES (?, ?, ?, ?)", "u1", "user1", "user1@test.com", false)
	// Global ACL: group_id = '' applies to all users
	db.Exec("INSERT INTO model_accesses (id, name, model_name, group_id, can_read, can_select) VALUES (?, ?, ?, ?, ?, ?)",
		"ma1", "contact_global", "contact", "", true, true)

	perms, err := svc.GetModelPermissions("u1", "contact")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !perms.CanRead {
		t.Error("expected can_read = true (global ACL)")
	}
	if !perms.CanSelect {
		t.Error("expected can_select = true (global ACL)")
	}
	if perms.CanWrite {
		t.Error("expected can_write = false (not granted)")
	}
}

func TestPermissionService_ImpliedGroups(t *testing.T) {
	db := setupPermissionTestDB(t)
	svc := NewPermissionService(db)

	db.Exec("INSERT INTO users (id, username, email, is_superuser) VALUES (?, ?, ?, ?)", "u1", "manager", "mgr@test.com", false)
	db.Exec("INSERT INTO groups (id, name) VALUES (?, ?)", "g_mgr", "CRM/Manager")
	db.Exec("INSERT INTO groups (id, name) VALUES (?, ?)", "g_user", "CRM/User")
	// CRM/Manager implies CRM/User
	db.Exec("INSERT INTO group_implies (group_id, implied_group_id) VALUES (?, ?)", "g_mgr", "g_user")
	// User is only directly in CRM/Manager
	db.Exec("INSERT INTO user_groups (user_id, group_id) VALUES (?, ?)", "u1", "g_mgr")
	// CRM/User has read on contact
	db.Exec("INSERT INTO model_accesses (id, name, model_name, group_id, can_read) VALUES (?, ?, ?, ?, ?)",
		"ma1", "contact_user", "contact", "g_user", true)
	// CRM/Manager has delete on contact
	db.Exec("INSERT INTO model_accesses (id, name, model_name, group_id, can_delete) VALUES (?, ?, ?, ?, ?)",
		"ma2", "contact_mgr", "contact", "g_mgr", true)

	perms, err := svc.GetModelPermissions("u1", "contact")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !perms.CanRead {
		t.Error("expected can_read = true (from implied CRM/User)")
	}
	if !perms.CanDelete {
		t.Error("expected can_delete = true (from direct CRM/Manager)")
	}
	if perms.CanWrite {
		t.Error("expected can_write = false (not granted by either group)")
	}
}

func TestPermissionService_UserHasPermission(t *testing.T) {
	db := setupPermissionTestDB(t)
	svc := NewPermissionService(db)

	db.Exec("INSERT INTO users (id, username, email, is_superuser) VALUES (?, ?, ?, ?)", "u1", "user1", "user1@test.com", false)
	db.Exec("INSERT INTO groups (id, name) VALUES (?, ?)", "g1", "Sales")
	db.Exec("INSERT INTO user_groups (user_id, group_id) VALUES (?, ?)", "u1", "g1")
	db.Exec("INSERT INTO model_accesses (id, name, model_name, group_id, can_read, can_create) VALUES (?, ?, ?, ?, ?, ?)",
		"ma1", "contact_sales", "contact", "g1", true, true)

	tests := []struct {
		permission string
		expected   bool
		wantErr    bool
	}{
		{"contact.read", true, false},
		{"contact.create", true, false},
		{"contact.delete", false, false},
		{"contact.write", false, false},
		{"order.read", false, false},
		{"invalid_format", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.permission, func(t *testing.T) {
			allowed, err := svc.UserHasPermission("u1", tt.permission)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error for invalid format")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if allowed != tt.expected {
				t.Errorf("UserHasPermission(%q) = %v, want %v", tt.permission, allowed, tt.expected)
			}
		})
	}
}

func TestPermissionService_UserNotFound(t *testing.T) {
	db := setupPermissionTestDB(t)
	svc := NewPermissionService(db)

	perms, err := svc.GetModelPermissions("nonexistent", "contact")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if perms.CanRead || perms.CanWrite || perms.CanCreate || perms.CanDelete {
		t.Error("nonexistent user should have no permissions")
	}
}
