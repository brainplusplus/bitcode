package model

import (
	"testing"

	"github.com/bitcode-framework/bitcode/internal/compiler/parser"
)

func TestRegistry_RegisterAndGet(t *testing.T) {
	reg := NewRegistry()

	model := &parser.ModelDefinition{
		Name:   "customer",
		Module: "crm",
		Fields: map[string]parser.FieldDefinition{
			"name": {Type: parser.FieldString},
		},
	}

	if err := reg.Register(model); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := reg.Get("customer")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "customer" {
		t.Errorf("expected customer, got %s", got.Name)
	}

	got2, err := reg.Get("crm.customer")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got2.Name != "customer" {
		t.Errorf("expected customer via module prefix, got %s", got2.Name)
	}
}

func TestRegistry_GetNotFound(t *testing.T) {
	reg := NewRegistry()
	_, err := reg.Get("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent model")
	}
}

func TestRegistry_RegisterEmptyName(t *testing.T) {
	reg := NewRegistry()
	err := reg.Register(&parser.ModelDefinition{})
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestRegistry_List(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&parser.ModelDefinition{Name: "order", Fields: map[string]parser.FieldDefinition{"x": {Type: "string"}}})
	reg.Register(&parser.ModelDefinition{Name: "customer", Fields: map[string]parser.FieldDefinition{"x": {Type: "string"}}})

	list := reg.List()
	if len(list) != 2 {
		t.Errorf("expected 2 models, got %d", len(list))
	}
}

func TestRegistry_Has(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&parser.ModelDefinition{Name: "order", Fields: map[string]parser.FieldDefinition{"x": {Type: "string"}}})

	if !reg.Has("order") {
		t.Error("expected Has(order) to be true")
	}
	if reg.Has("nonexistent") {
		t.Error("expected Has(nonexistent) to be false")
	}
}

func TestRegistry_TableName_Fallback(t *testing.T) {
	reg := NewRegistry()
	if reg.TableName("order") != "order" {
		t.Errorf("expected 'order', got %s", reg.TableName("order"))
	}
}

func strPtr(s string) *string { return &s }

func TestResolveTableName_NoPrefix(t *testing.T) {
	model := &parser.ModelDefinition{Name: "contact"}
	got := ResolveTableName(model, nil)
	if got != "contact" {
		t.Errorf("expected 'contact', got %q", got)
	}
}

func TestResolveTableName_ModulePrefix(t *testing.T) {
	model := &parser.ModelDefinition{Name: "contact"}
	mod := &parser.ModuleDefinition{Name: "crm", Table: &parser.TableConfig{Prefix: "crm"}}
	got := ResolveTableName(model, mod)
	if got != "crm_contact" {
		t.Errorf("expected 'crm_contact', got %q", got)
	}
}

func TestResolveTableName_DirectTableName(t *testing.T) {
	model := &parser.ModelDefinition{Name: "contact", TableName: "custom_tbl"}
	mod := &parser.ModuleDefinition{Name: "crm", Table: &parser.TableConfig{Prefix: "crm"}}
	got := ResolveTableName(model, mod)
	if got != "custom_tbl" {
		t.Errorf("expected 'custom_tbl', got %q", got)
	}
}

func TestResolveTableName_ModelPrefixOverride(t *testing.T) {
	model := &parser.ModelDefinition{Name: "log", TablePrefix: strPtr("sys")}
	mod := &parser.ModuleDefinition{Name: "crm", Table: &parser.TableConfig{Prefix: "crm"}}
	got := ResolveTableName(model, mod)
	if got != "sys_log" {
		t.Errorf("expected 'sys_log', got %q", got)
	}
}

func TestResolveTableName_ModelEmptyPrefixClearsModulePrefix(t *testing.T) {
	model := &parser.ModelDefinition{Name: "setting", TablePrefix: strPtr("")}
	mod := &parser.ModuleDefinition{Name: "base", Table: &parser.TableConfig{Prefix: "res"}}
	got := ResolveTableName(model, mod)
	if got != "setting" {
		t.Errorf("expected 'setting', got %q", got)
	}
}

func TestResolveTableName_ModuleWithoutTableConfig(t *testing.T) {
	model := &parser.ModelDefinition{Name: "token"}
	mod := &parser.ModuleDefinition{Name: "auth"}
	got := ResolveTableName(model, mod)
	if got != "token" {
		t.Errorf("expected 'token', got %q", got)
	}
}

func TestRegistry_TableName_WithModule(t *testing.T) {
	reg := NewRegistry()
	model := &parser.ModelDefinition{
		Name:   "contact",
		Fields: map[string]parser.FieldDefinition{"name": {Type: "string"}},
	}
	mod := &parser.ModuleDefinition{Name: "crm", Table: &parser.TableConfig{Prefix: "crm"}}
	if err := reg.RegisterWithModule(model, mod); err != nil {
		t.Fatal(err)
	}
	got := reg.TableName("contact")
	if got != "crm_contact" {
		t.Errorf("expected 'crm_contact', got %q", got)
	}
}

func TestRegistry_TableName_BasePrefix(t *testing.T) {
	reg := NewRegistry()
	model := &parser.ModelDefinition{
		Name:   "user",
		Fields: map[string]parser.FieldDefinition{"username": {Type: "string"}},
	}
	mod := &parser.ModuleDefinition{Name: "base", Table: &parser.TableConfig{Prefix: "res"}}
	if err := reg.RegisterWithModule(model, mod); err != nil {
		t.Fatal(err)
	}
	got := reg.TableName("user")
	if got != "res_user" {
		t.Errorf("expected 'res_user', got %q", got)
	}
}

func TestRegistry_AmbiguousModel(t *testing.T) {
	reg := NewRegistry()
	crmContact := &parser.ModelDefinition{Name: "contact", Module: "crm", Fields: map[string]parser.FieldDefinition{"name": {Type: "string"}}}
	hrmContact := &parser.ModelDefinition{Name: "contact", Module: "hrm", Fields: map[string]parser.FieldDefinition{"name": {Type: "string"}}}
	crmMod := &parser.ModuleDefinition{Name: "crm", Table: &parser.TableConfig{Prefix: "crm"}}
	hrmMod := &parser.ModuleDefinition{Name: "hrm", Table: &parser.TableConfig{Prefix: "hrm"}}

	if err := reg.RegisterWithModule(crmContact, crmMod); err != nil {
		t.Fatal(err)
	}
	if err := reg.RegisterWithModule(hrmContact, hrmMod); err != nil {
		t.Fatal(err)
	}

	if !reg.IsAmbiguous("contact") {
		t.Error("expected 'contact' to be ambiguous")
	}

	modules := reg.ModulesForModel("contact")
	if len(modules) != 2 {
		t.Fatalf("expected 2 modules, got %d", len(modules))
	}

	crmTable := reg.TableName("crm.contact")
	if crmTable != "crm_contact" {
		t.Errorf("expected 'crm_contact', got %q", crmTable)
	}

	hrmTable := reg.TableName("hrm.contact")
	if hrmTable != "hrm_contact" {
		t.Errorf("expected 'hrm_contact', got %q", hrmTable)
	}
}

func TestRegistry_QualifiedGet(t *testing.T) {
	reg := NewRegistry()
	crmContact := &parser.ModelDefinition{Name: "contact", Module: "crm", Fields: map[string]parser.FieldDefinition{"name": {Type: "string"}}}
	if err := reg.Register(crmContact); err != nil {
		t.Fatal(err)
	}

	model, err := reg.Get("crm.contact")
	if err != nil {
		t.Fatalf("expected to find crm.contact: %v", err)
	}
	if model.Module != "crm" {
		t.Errorf("expected module 'crm', got %q", model.Module)
	}

	model2, err := reg.Get("contact")
	if err != nil {
		t.Fatalf("expected to find contact: %v", err)
	}
	if model2.Name != "contact" {
		t.Errorf("expected name 'contact', got %q", model2.Name)
	}
}

func TestRegistry_NotAmbiguous(t *testing.T) {
	reg := NewRegistry()
	model := &parser.ModelDefinition{Name: "order", Module: "sales", Fields: map[string]parser.FieldDefinition{"total": {Type: "decimal"}}}
	if err := reg.Register(model); err != nil {
		t.Fatal(err)
	}
	if reg.IsAmbiguous("order") {
		t.Error("expected 'order' to not be ambiguous")
	}
}

func TestRegistry_DuplicateModelSameModule(t *testing.T) {
	reg := NewRegistry()
	m1 := &parser.ModelDefinition{Name: "contact", Module: "crm", Fields: map[string]parser.FieldDefinition{"name": {Type: "string"}}}
	m2 := &parser.ModelDefinition{Name: "contact", Module: "crm", Fields: map[string]parser.FieldDefinition{"email": {Type: "email"}}}

	if err := reg.Register(m1); err != nil {
		t.Fatal(err)
	}
	err := reg.Register(m2)
	if err == nil {
		t.Fatal("expected error for duplicate model in same module")
	}
}

func TestRegistry_DuplicateModelDifferentModule(t *testing.T) {
	reg := NewRegistry()
	m1 := &parser.ModelDefinition{Name: "contact", Module: "crm", Fields: map[string]parser.FieldDefinition{"name": {Type: "string"}}}
	m2 := &parser.ModelDefinition{Name: "contact", Module: "hrm", Fields: map[string]parser.FieldDefinition{"name": {Type: "string"}}}

	if err := reg.Register(m1); err != nil {
		t.Fatal(err)
	}
	if err := reg.Register(m2); err != nil {
		t.Fatal("cross-module same name should be allowed")
	}
}

func TestRegistry_InheritNotDuplicate(t *testing.T) {
	reg := NewRegistry()
	m1 := &parser.ModelDefinition{Name: "user", Module: "base", Fields: map[string]parser.FieldDefinition{"name": {Type: "string"}}}
	m2 := &parser.ModelDefinition{Name: "user", Module: "base", Inherit: "base.user", Fields: map[string]parser.FieldDefinition{"phone": {Type: "string"}}}

	if err := reg.Register(m1); err != nil {
		t.Fatal(err)
	}
	if err := reg.Register(m2); err != nil {
		t.Fatal("inheritance should not be flagged as duplicate")
	}
}

func TestResolveTableName_Plural(t *testing.T) {
	model := &parser.ModelDefinition{Name: "contact"}
	mod := &parser.ModuleDefinition{Name: "crm", Table: &parser.TableConfig{Prefix: "crm"}}

	got := ResolveTableName(model, mod, "plural")
	if got != "crm_contacts" {
		t.Errorf("expected 'crm_contacts', got %q", got)
	}
}

func TestResolveTableName_PluralPerModel(t *testing.T) {
	plural := true
	model := &parser.ModelDefinition{Name: "person", TablePlural: &plural}
	mod := &parser.ModuleDefinition{Name: "hr", Table: &parser.TableConfig{Prefix: "hr"}}

	got := ResolveTableName(model, mod)
	if got != "hr_people" {
		t.Errorf("expected 'hr_people', got %q", got)
	}
}

func TestResolveTableName_SingularDefault(t *testing.T) {
	model := &parser.ModelDefinition{Name: "contact"}
	mod := &parser.ModuleDefinition{Name: "crm", Table: &parser.TableConfig{Prefix: "crm"}}

	got := ResolveTableName(model, mod)
	if got != "crm_contact" {
		t.Errorf("expected 'crm_contact', got %q", got)
	}
}

func TestResolveTableName_ExplicitTableNameIgnoresPlural(t *testing.T) {
	model := &parser.ModelDefinition{Name: "contact", TableName: "my_custom_table"}

	got := ResolveTableName(model, nil, "plural")
	if got != "my_custom_table" {
		t.Errorf("expected 'my_custom_table', got %q", got)
	}
}
