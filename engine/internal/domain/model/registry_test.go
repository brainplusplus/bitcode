package model

import (
	"testing"

	"github.com/bitcode-engine/engine/internal/compiler/parser"
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
