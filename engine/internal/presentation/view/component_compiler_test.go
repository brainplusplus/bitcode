package view

import (
	"strings"
	"testing"

	"github.com/bitcode-framework/bitcode/internal/compiler/parser"
)

func TestCompileForm_BasicLayout(t *testing.T) {
	cc := NewComponentCompiler()
	viewDef := &parser.ViewDefinition{
		Name:  "order_form",
		Type:  parser.ViewForm,
		Model: "order",
		Title: "Sales Order",
		Layout: []parser.LayoutItem{
			{Row: []parser.LayoutRow{
				{Field: "customer_id", Width: 6},
				{Field: "status", Width: 3, Readonly: true},
			}},
		},
	}

	html := cc.CompileForm(viewDef, map[string]any{"customer_id": "cust-1", "status": "draft"})

	if !strings.Contains(html, `<bc-view-form model="order"`) {
		t.Error("expected bc-view-form with model")
	}
	if !strings.Contains(html, `<bc-row>`) {
		t.Error("expected bc-row")
	}
	if !strings.Contains(html, `<bc-column width="6">`) {
		t.Error("expected bc-column with width 6")
	}
	if !strings.Contains(html, `name="customer_id"`) {
		t.Error("expected field name customer_id")
	}
	if !strings.Contains(html, `value="cust-1"`) {
		t.Error("expected value cust-1")
	}
	if !strings.Contains(html, `readonly`) {
		t.Error("expected readonly on status field")
	}
}

func TestCompileForm_WithHeader(t *testing.T) {
	cc := NewComponentCompiler()
	viewDef := &parser.ViewDefinition{
		Name:  "test_form",
		Type:  parser.ViewForm,
		Model: "order",
		Title: "Test",
		Layout: []parser.LayoutItem{
			{Header: &parser.HeaderDefinition{
				StatusField: "status",
				Widget:      "statusbar",
				Buttons: []parser.ActionDefinition{
					{Label: "Confirm", Process: "confirm_order", Variant: "primary"},
				},
			}},
		},
	}

	html := cc.CompileForm(viewDef, map[string]any{"status": "draft"})

	if !strings.Contains(html, `<bc-header`) {
		t.Error("expected bc-header")
	}
	if !strings.Contains(html, `status-value="draft"`) {
		t.Error("expected status-value=draft")
	}
	if !strings.Contains(html, `Confirm`) {
		t.Error("expected Confirm button in header")
	}
}

func TestCompileForm_WithSection(t *testing.T) {
	cc := NewComponentCompiler()
	viewDef := &parser.ViewDefinition{
		Name:  "test_form",
		Type:  parser.ViewForm,
		Model: "order",
		Title: "Test",
		Layout: []parser.LayoutItem{
			{
				Section: &parser.SectionDefinition{Title: "Info", Collapsible: true},
				Rows: []parser.LayoutItem{
					{Row: []parser.LayoutRow{{Field: "name", Width: 12}}},
				},
			},
		},
	}

	html := cc.CompileForm(viewDef, nil)

	if !strings.Contains(html, `<bc-section section-title="Info" collapsible>`) {
		t.Error("expected bc-section with title and collapsible")
	}
}

func TestCompileForm_WithChildTable(t *testing.T) {
	cc := NewComponentCompiler()
	viewDef := &parser.ViewDefinition{
		Name:  "test_form",
		Type:  parser.ViewForm,
		Model: "order",
		Title: "Test",
		Layout: []parser.LayoutItem{
			{ChildTable: &parser.ChildTableDefinition{
				Field: "lines",
				Columns: []parser.ChildTableColumn{
					{Field: "product", Width: 6},
					{Field: "qty", Width: 3},
				},
			}},
		},
	}

	html := cc.CompileForm(viewDef, nil)

	if !strings.Contains(html, `<bc-child-table field="lines"`) {
		t.Error("expected bc-child-table")
	}
	if !strings.Contains(html, `"product"`) {
		t.Error("expected product column")
	}
}

func TestCompileForm_WithChatter(t *testing.T) {
	cc := NewComponentCompiler()
	viewDef := &parser.ViewDefinition{
		Name:  "test_form",
		Type:  parser.ViewForm,
		Model: "order",
		Title: "Test",
		Layout: []parser.LayoutItem{
			{Chatter: true},
		},
	}

	html := cc.CompileForm(viewDef, nil)

	if !strings.Contains(html, `<bc-chatter></bc-chatter>`) {
		t.Error("expected bc-chatter")
	}
}

func TestCompileList(t *testing.T) {
	cc := NewComponentCompiler()
	viewDef := &parser.ViewDefinition{
		Name:   "order_list",
		Type:   parser.ViewList,
		Model:  "order",
		Title:  "Orders",
		Fields: []string{"name", "status", "total"},
	}

	html := cc.CompileList(viewDef)

	if !strings.Contains(html, `<bc-datatable`) {
		t.Error("expected bc-datatable")
	}
	if !strings.Contains(html, `model="order"`) {
		t.Error("expected model=order")
	}
	if !strings.Contains(html, `"name"`) {
		t.Error("expected name in columns")
	}
}

func TestCompileKanban(t *testing.T) {
	cc := NewComponentCompiler()
	viewDef := &parser.ViewDefinition{
		Name:    "lead_kanban",
		Type:    parser.ViewKanban,
		Model:   "lead",
		Title:   "Pipeline",
		Fields:  []string{"name", "company"},
		GroupBy: "status",
	}

	html := cc.CompileKanban(viewDef)

	if !strings.Contains(html, `<bc-view-kanban model="lead"`) {
		t.Error("expected bc-view-kanban")
	}
	if !strings.Contains(html, `"group_by":"status"`) {
		t.Error("expected group_by in config")
	}
}
