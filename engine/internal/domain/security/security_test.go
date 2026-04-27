package security

import (
	"testing"
)

func TestNewUser(t *testing.T) {
	u, err := NewUser("u-1", "john", "john@example.com", "secret123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.Username != "john" {
		t.Errorf("expected john, got %s", u.Username)
	}
	if !u.Active {
		t.Error("new user should be active")
	}
	if !u.CheckPassword("secret123") {
		t.Error("password should match")
	}
	if u.CheckPassword("wrong") {
		t.Error("wrong password should not match")
	}
	if len(u.GetDomainEvents()) != 1 {
		t.Errorf("expected 1 event, got %d", len(u.GetDomainEvents()))
	}
	if u.GetDomainEvents()[0].EventName() != "user.created" {
		t.Errorf("expected user.created, got %s", u.GetDomainEvents()[0].EventName())
	}
}

func TestUser_ActivateDeactivate(t *testing.T) {
	u, _ := NewUser("u-1", "john", "john@example.com", "secret123")
	u.ClearDomainEvents()

	u.Deactivate()
	if u.Active {
		t.Error("user should be inactive")
	}
	if len(u.GetDomainEvents()) != 1 || u.GetDomainEvents()[0].EventName() != "user.deactivated" {
		t.Error("expected user.deactivated event")
	}

	u.ClearDomainEvents()
	u.Activate()
	if !u.Active {
		t.Error("user should be active")
	}
}

func TestUser_HasPermission(t *testing.T) {
	u, _ := NewUser("u-1", "john", "john@example.com", "secret123")
	role := NewRole("r-1", "sales_user", "Sales User")
	role.Permissions = []Permission{
		{Name: "order.read"},
		{Name: "order.create"},
	}
	u.Roles = []Role{*role}

	if !u.HasPermission("order.read") {
		t.Error("should have order.read")
	}
	if u.HasPermission("order.delete") {
		t.Error("should not have order.delete")
	}
}

func TestRole_InheritedPermissions(t *testing.T) {
	parent := NewRole("r-1", "sales_user", "Sales User")
	parent.Permissions = []Permission{{Name: "order.read"}}

	child := NewRole("r-2", "sales_manager", "Sales Manager")
	child.Permissions = []Permission{{Name: "order.approve"}}
	child.Inherits = []Role{*parent}

	if !child.HasPermission("order.read") {
		t.Error("should inherit order.read from parent")
	}
	if !child.HasPermission("order.approve") {
		t.Error("should have own order.approve")
	}

	all := child.AllPermissions()
	if len(all) != 2 {
		t.Errorf("expected 2 permissions, got %d", len(all))
	}
}

func TestGroup_AllGroupNames(t *testing.T) {
	base := NewGroup("g-1", "base.user", "Base User", "Base")
	sales := NewGroup("g-2", "sales.user", "Sales User", "Sales")
	sales.ImpliedGroups = []Group{*base}
	manager := NewGroup("g-3", "sales.manager", "Sales Manager", "Sales")
	manager.ImpliedGroups = []Group{*sales}

	names := manager.AllGroupNames()
	if len(names) != 3 {
		t.Errorf("expected 3 groups (manager + sales + base), got %d: %v", len(names), names)
	}
}

func TestRecordRule_AppliesToGroup(t *testing.T) {
	rule := NewRecordRule("rr-1", "order_user_rule", "order", []string{"sales.user"})

	if !rule.AppliesToGroup([]string{"sales.user", "base.user"}) {
		t.Error("should apply to sales.user")
	}
	if rule.AppliesToGroup([]string{"hr.user"}) {
		t.Error("should not apply to hr.user")
	}
}

func TestRecordRule_Global(t *testing.T) {
	rule := NewRecordRule("rr-1", "global_rule", "order", nil)
	rule.Global = true

	if !rule.AppliesToGroup([]string{"any.group"}) {
		t.Error("global rule should apply to any group")
	}
}

func TestRecordRule_AppliesToOperation(t *testing.T) {
	rule := NewRecordRule("rr-1", "test", "order", []string{"sales.user"})
	rule.CanDelete = false

	if !rule.AppliesToOperation("read") {
		t.Error("should apply to read")
	}
	if rule.AppliesToOperation("delete") {
		t.Error("should not apply to delete")
	}
}

func TestInterpolateDomain(t *testing.T) {
	domain := `[["created_by", "=", "{{user.id}}"]]`
	result := InterpolateDomain(domain, map[string]string{"user.id": "user-42"})
	expected := `[["created_by", "=", "user-42"]]`
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestGroup_ShareField(t *testing.T) {
	g := NewGroup("g-1", "portal.user", "Portal User", "Portal")
	g.Share = true
	if !g.Share {
		t.Error("expected share=true")
	}
}

func TestUser_IsSuperuser(t *testing.T) {
	u, _ := NewUser("u-1", "admin", "admin@test.com", "password")
	u.IsSuperuser = true
	if !u.IsSuperuser {
		t.Error("expected superuser")
	}
}

func TestUser_AllGroupNames(t *testing.T) {
	base := NewGroup("g-1", "base.user", "Base User", "Base")
	crm := NewGroup("g-2", "crm.user", "CRM User", "CRM")
	crm.ImpliedGroups = []Group{*base}

	u, _ := NewUser("u-1", "john", "john@test.com", "password")
	u.Groups = []Group{*crm}

	names := u.AllGroupNames()
	if len(names) != 2 {
		t.Errorf("expected 2 groups (crm.user + base.user), got %d: %v", len(names), names)
	}
}

func TestRecordRule_AppliesToGroupNamesM2M(t *testing.T) {
	g1 := NewGroup("g-1", "crm.user", "CRM User", "CRM")
	rule := &RecordRule{
		Name:      "own_contacts",
		ModelName: "contact",
		Groups:    []Group{*g1},
		CanRead:   true,
		Active:    true,
	}

	if !rule.AppliesToGroupNames([]string{"crm.user"}) {
		t.Error("expected rule to apply to crm.user")
	}
	if rule.AppliesToGroupNames([]string{"sales.user"}) {
		t.Error("expected rule NOT to apply to sales.user")
	}
}

func TestRecordRule_IsGlobalNew(t *testing.T) {
	rule := &RecordRule{
		Name:      "global_rule",
		ModelName: "contact",
		CanRead:   true,
		Active:    true,
	}
	if !rule.IsGlobal() {
		t.Error("expected global when no groups")
	}
}
