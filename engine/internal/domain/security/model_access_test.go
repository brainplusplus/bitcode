package security

import (
	"testing"
)

func TestModelAccess_HasPermission(t *testing.T) {
	ma := NewModelAccess("ma-1", "contact_access", "contact", "sales.user", "crm")

	operations := []string{"select", "read", "write", "create", "delete", "print", "email", "report", "export", "import", "mask", "clone"}

	for _, op := range operations {
		if ma.HasPermission(op) {
			t.Errorf("%s should be false by default", op)
		}
	}

	ma.CanSelect = true
	if !ma.HasPermission("select") {
		t.Error("select should be true")
	}

	ma.CanRead = true
	if !ma.HasPermission("read") {
		t.Error("read should be true")
	}

	ma.CanWrite = true
	if !ma.HasPermission("write") {
		t.Error("write should be true")
	}

	ma.CanCreate = true
	if !ma.HasPermission("create") {
		t.Error("create should be true")
	}

	ma.CanDelete = true
	if !ma.HasPermission("delete") {
		t.Error("delete should be true")
	}

	ma.CanPrint = true
	if !ma.HasPermission("print") {
		t.Error("print should be true")
	}

	ma.CanEmail = true
	if !ma.HasPermission("email") {
		t.Error("email should be true")
	}

	ma.CanReport = true
	if !ma.HasPermission("report") {
		t.Error("report should be true")
	}

	ma.CanExport = true
	if !ma.HasPermission("export") {
		t.Error("export should be true")
	}

	ma.CanImport = true
	if !ma.HasPermission("import") {
		t.Error("import should be true")
	}

	ma.CanMask = true
	if !ma.HasPermission("mask") {
		t.Error("mask should be true")
	}

	ma.CanClone = true
	if !ma.HasPermission("clone") {
		t.Error("clone should be true")
	}

	if ma.HasPermission("unknown") {
		t.Error("unknown operation should return false")
	}
}

func TestModelAccess_AllPermissions(t *testing.T) {
	ma := NewModelAccess("ma-1", "contact_access", "contact", "sales.user", "crm")
	ma.SetAll(true)

	perms := ma.AllPermissions()
	if len(perms) != 12 {
		t.Errorf("expected 12 permissions, got %d: %v", len(perms), perms)
	}

	expected := []string{"select", "read", "write", "create", "delete", "print", "email", "report", "export", "import", "mask", "clone"}
	for i, e := range expected {
		if perms[i] != e {
			t.Errorf("expected %s at index %d, got %s", e, i, perms[i])
		}
	}

	ma.SetAll(false)
	perms = ma.AllPermissions()
	if len(perms) != 0 {
		t.Errorf("expected 0 permissions after SetAll(false), got %d", len(perms))
	}
}

func TestModelAccess_SetFromList(t *testing.T) {
	ma := NewModelAccess("ma-1", "contact_access", "contact", "sales.user", "crm")
	ma.SetFromList([]string{"select", "read", "write"})

	if !ma.CanSelect {
		t.Error("select should be true")
	}
	if !ma.CanRead {
		t.Error("read should be true")
	}
	if !ma.CanWrite {
		t.Error("write should be true")
	}
	if ma.CanCreate {
		t.Error("create should be false")
	}
	if ma.CanDelete {
		t.Error("delete should be false")
	}
	if ma.CanPrint {
		t.Error("print should be false")
	}
	if ma.CanExport {
		t.Error("export should be false")
	}

	perms := ma.AllPermissions()
	if len(perms) != 3 {
		t.Errorf("expected 3 permissions, got %d: %v", len(perms), perms)
	}
}

func TestModelAccess_IsGlobal(t *testing.T) {
	global := NewModelAccess("ma-1", "global_access", "contact", "", "crm")
	if !global.IsGlobal() {
		t.Error("empty GroupID should be global")
	}

	scoped := NewModelAccess("ma-2", "scoped_access", "contact", "sales.user", "crm")
	if scoped.IsGlobal() {
		t.Error("non-empty GroupID should not be global")
	}
}
