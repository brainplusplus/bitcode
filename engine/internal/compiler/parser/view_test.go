package parser

import "testing"

func TestParseView_List(t *testing.T) {
	data := []byte(`{
		"name": "order_list",
		"type": "list",
		"model": "order",
		"title": "Sales Orders",
		"fields": ["order_date", "customer_id", "total", "status"],
		"filters": ["status", "order_date"],
		"sort": { "field": "order_date", "order": "desc" }
	}`)

	view, err := ParseView(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if view.Name != "order_list" {
		t.Errorf("expected order_list, got %s", view.Name)
	}
	if len(view.Fields) != 4 {
		t.Errorf("expected 4 fields, got %d", len(view.Fields))
	}
}

func TestParseView_Form(t *testing.T) {
	data := []byte(`{
		"name": "order_form",
		"type": "form",
		"model": "order",
		"title": "Sales Order",
		"layout": [
			{ "row": [{"field": "customer_id", "width": 6}] }
		],
		"actions": [
			{ "label": "Confirm", "process": "confirm_order", "variant": "primary" }
		]
	}`)

	view, err := ParseView(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if view.Type != ViewForm {
		t.Errorf("expected form, got %s", view.Type)
	}
	if len(view.Actions) != 1 {
		t.Errorf("expected 1 action, got %d", len(view.Actions))
	}
}

func TestParseView_Custom(t *testing.T) {
	data := []byte(`{
		"name": "dashboard",
		"type": "custom",
		"template": "templates/dashboard.html",
		"data_sources": {
			"orders": { "model": "order", "domain": [["status", "=", "confirmed"]] }
		}
	}`)

	view, err := ParseView(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if view.Template != "templates/dashboard.html" {
		t.Errorf("expected templates/dashboard.html, got %s", view.Template)
	}
}

func TestParseView_CustomRequiresTemplate(t *testing.T) {
	data := []byte(`{"name": "bad", "type": "custom"}`)
	_, err := ParseView(data)
	if err == nil {
		t.Fatal("expected error for custom view without template")
	}
}

func TestParseView_MissingName(t *testing.T) {
	data := []byte(`{"type": "list"}`)
	_, err := ParseView(data)
	if err == nil {
		t.Fatal("expected error for missing name")
	}
}

func TestParseView_MissingType(t *testing.T) {
	data := []byte(`{"name": "test"}`)
	_, err := ParseView(data)
	if err == nil {
		t.Fatal("expected error for missing type")
	}
}

func TestParseView_ExtendedLayout(t *testing.T) {
	data := []byte(`{
		"name": "order_form_ext",
		"type": "form",
		"model": "order",
		"layout": [
			{"header": {"status_field": "status", "widget": "statusbar", "buttons": [{"label": "Confirm", "process": "confirm"}]}},
			{"button_box": [{"label": "Invoices", "icon": "file-text", "count_model": "invoice"}]},
			{"section": {"title": "Info", "collapsible": true}, "rows": [
				{"row": [{"field": "name", "width": 6, "widget": "badge"}]}
			]},
			{"child_table": {"field": "lines", "columns": [{"field": "product", "width": 6}], "summary": {"total": "sum"}}},
			{"chatter": true},
			{"separator": {"label": "Details"}},
			{"tabs": [{"label": "Notes", "fields": ["notes"], "visible": "status != 'done'"}]}
		]
	}`)

	view, err := ParseView(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if view.Layout[0].Header == nil {
		t.Fatal("expected header to be parsed")
	}
	if view.Layout[0].Header.Widget != "statusbar" {
		t.Errorf("expected statusbar, got %s", view.Layout[0].Header.Widget)
	}
	if view.Layout[0].Header.StatusField != "status" {
		t.Errorf("expected status, got %s", view.Layout[0].Header.StatusField)
	}
	if len(view.Layout[0].Header.Buttons) != 1 {
		t.Errorf("expected 1 button, got %d", len(view.Layout[0].Header.Buttons))
	}
	if len(view.Layout[1].ButtonBox) != 1 {
		t.Fatal("expected 1 smart button")
	}
	if view.Layout[1].ButtonBox[0].CountModel != "invoice" {
		t.Errorf("expected count_model invoice, got %s", view.Layout[1].ButtonBox[0].CountModel)
	}
	if view.Layout[2].Section == nil {
		t.Fatal("expected section to be parsed")
	}
	if !view.Layout[2].Section.Collapsible {
		t.Error("expected collapsible true")
	}
	if len(view.Layout[2].Rows) != 1 {
		t.Errorf("expected 1 row in section, got %d", len(view.Layout[2].Rows))
	}
	if view.Layout[2].Rows[0].Row[0].Widget != "badge" {
		t.Errorf("expected widget badge, got %s", view.Layout[2].Rows[0].Row[0].Widget)
	}
	if view.Layout[3].ChildTable == nil {
		t.Fatal("expected child_table to be parsed")
	}
	if view.Layout[3].ChildTable.Field != "lines" {
		t.Errorf("expected field lines, got %s", view.Layout[3].ChildTable.Field)
	}
	if view.Layout[3].ChildTable.Summary["total"] != "sum" {
		t.Errorf("expected summary total=sum")
	}
	if !view.Layout[4].Chatter {
		t.Error("expected chatter true")
	}
	if view.Layout[5].Separator == nil {
		t.Fatal("expected separator to be parsed")
	}
	if view.Layout[5].Separator.Label != "Details" {
		t.Errorf("expected label Details, got %s", view.Layout[5].Separator.Label)
	}
	if view.Layout[6].Tabs[0].Visible != "status != 'done'" {
		t.Errorf("expected tab visible expression, got %s", view.Layout[6].Tabs[0].Visible)
	}
}

func TestParseView_NewViewTypes(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		viewType ViewType
	}{
		{"gantt", `{"name":"g","type":"gantt","model":"task","start_field":"start","end_field":"end"}`, ViewGantt},
		{"map", `{"name":"m","type":"map","model":"contact"}`, ViewMap},
		{"tree", `{"name":"t","type":"tree","model":"account","parent_field":"parent_id"}`, ViewTree},
		{"activity", `{"name":"a","type":"activity","model":"lead"}`, ViewActivity},
		{"report", `{"name":"r","type":"report","model":"order"}`, ViewReport},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view, err := ParseView([]byte(tt.json))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if view.Type != tt.viewType {
				t.Errorf("expected %s, got %s", tt.viewType, view.Type)
			}
		})
	}
}
