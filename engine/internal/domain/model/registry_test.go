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

func TestRegistry_TableName(t *testing.T) {
	reg := NewRegistry()
	if reg.TableName("order") != "orders" {
		t.Errorf("expected orders, got %s", reg.TableName("order"))
	}
}
