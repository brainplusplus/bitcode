package view

import (
	"testing"

	"github.com/bitcode-framework/bitcode/internal/compiler/parser"
)

func TestGenerateListView(t *testing.T) {
	model := &parser.ModelDefinition{
		Name:   "contact",
		Module: "crm",
		Label:  "Contact",
		Fields: map[string]parser.FieldDefinition{
			"name":    {Type: parser.FieldString},
			"email":   {Type: parser.FieldEmail},
			"phone":   {Type: parser.FieldString},
			"company": {Type: parser.FieldString},
			"notes":   {Type: parser.FieldText},
			"tags":    {Type: parser.FieldMany2Many, Model: "tag"},
		},
		TitleField: "name",
	}

	view := GenerateListView(model, "crm")
	if view == nil {
		t.Fatal("expected view definition")
	}
	if view.Name != "contact_list" {
		t.Errorf("expected name 'contact_list', got %q", view.Name)
	}
	if view.Type != parser.ViewList {
		t.Errorf("expected type 'list', got %q", view.Type)
	}
	if view.Title != "Contact" {
		t.Errorf("expected title 'Contact', got %q", view.Title)
	}

	for _, f := range view.Fields {
		if f == "notes" {
			t.Error("text fields should be excluded from list")
		}
	}

	if view.Sort == nil || view.Sort.Field != "name" {
		t.Error("expected sort by title_field 'name'")
	}
}

func TestGenerateFormView(t *testing.T) {
	model := &parser.ModelDefinition{
		Name:   "order",
		Module: "sales",
		Label:  "Sales Order",
		Fields: map[string]parser.FieldDefinition{
			"customer_id": {Type: parser.FieldMany2One, Model: "contact"},
			"order_date":  {Type: parser.FieldDate},
			"status":      {Type: parser.FieldSelection, Options: []string{"draft", "confirmed"}},
			"notes":       {Type: parser.FieldText},
			"lines":       {Type: parser.FieldOne2Many, Model: "order_line", Inverse: "order_id", Label: "Lines"},
		},
	}

	view := GenerateFormView(model, "sales")
	if view == nil {
		t.Fatal("expected view definition")
	}
	if view.Name != "order_form" {
		t.Errorf("expected name 'order_form', got %q", view.Name)
	}
	if view.Type != parser.ViewForm {
		t.Errorf("expected type 'form', got %q", view.Type)
	}

	hasTabs := false
	for _, item := range view.Layout {
		if len(item.Tabs) > 0 {
			hasTabs = true
		}
	}
	if !hasTabs {
		t.Error("expected tabs for one2many field 'lines'")
	}
}

func TestShouldAutoGeneratePages(t *testing.T) {
	t.Run("api true", func(t *testing.T) {
		model := &parser.ModelDefinition{
			API: &parser.APIConfig{AutoCRUD: true},
		}
		if !ShouldAutoGeneratePages(model) {
			t.Error("expected true when api config exists")
		}
	})

	t.Run("api nil", func(t *testing.T) {
		model := &parser.ModelDefinition{}
		if ShouldAutoGeneratePages(model) {
			t.Error("expected false when no api config")
		}
	})

	t.Run("auto_pages false", func(t *testing.T) {
		falseJSON := []byte("false")
		model := &parser.ModelDefinition{
			API: &parser.APIConfig{AutoCRUD: true, AutoPages: falseJSON},
		}
		if ShouldAutoGeneratePages(model) {
			t.Error("expected false when auto_pages=false")
		}
	})
}
