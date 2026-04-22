package parser

import (
	"testing"
)

func TestParseModel_ValidOrder(t *testing.T) {
	data := []byte(`{
		"name": "order",
		"module": "sales",
		"label": "Sales Order",
		"fields": {
			"customer_id": { "type": "many2one", "model": "customer", "required": true },
			"order_date":  { "type": "date", "default": "now" },
			"status":      { "type": "selection", "options": ["draft", "confirmed", "done"] },
			"total":       { "type": "decimal", "computed": "sum(lines.subtotal)" },
			"notes":       { "type": "text" },
			"lines":       { "type": "one2many", "model": "order_line", "inverse": "order_id" }
		},
		"record_rules": [
			{ "groups": ["sales.user"], "domain": [["created_by", "=", "{{user.id}}"]] }
		],
		"indexes": [["customer_id", "order_date"]]
	}`)

	model, err := ParseModel(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if model.Name != "order" {
		t.Errorf("expected name order, got %s", model.Name)
	}
	if model.Module != "sales" {
		t.Errorf("expected module sales, got %s", model.Module)
	}
	if len(model.Fields) != 6 {
		t.Errorf("expected 6 fields, got %d", len(model.Fields))
	}
	if model.Fields["customer_id"].Type != FieldMany2One {
		t.Errorf("expected many2one, got %s", model.Fields["customer_id"].Type)
	}
	if model.Fields["customer_id"].Model != "customer" {
		t.Errorf("expected model customer, got %s", model.Fields["customer_id"].Model)
	}
	if len(model.RecordRules) != 1 {
		t.Errorf("expected 1 record rule, got %d", len(model.RecordRules))
	}
	if len(model.Indexes) != 1 {
		t.Errorf("expected 1 index, got %d", len(model.Indexes))
	}
}

func TestParseModel_Inheritance(t *testing.T) {
	data := []byte(`{
		"name": "vip_customer",
		"inherit": "customer",
		"fields": {
			"vip_level": { "type": "selection", "options": ["gold", "platinum"] }
		}
	}`)

	model, err := ParseModel(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if model.Inherit != "customer" {
		t.Errorf("expected inherit customer, got %s", model.Inherit)
	}
}

func TestParseModel_MissingName(t *testing.T) {
	data := []byte(`{"fields": {"name": {"type": "string"}}}`)
	_, err := ParseModel(data)
	if err == nil {
		t.Fatal("expected error for missing name")
	}
}

func TestParseModel_NoFields(t *testing.T) {
	data := []byte(`{"name": "empty"}`)
	_, err := ParseModel(data)
	if err == nil {
		t.Fatal("expected error for no fields")
	}
}

func TestParseModel_Many2OneWithoutModel(t *testing.T) {
	data := []byte(`{"name": "bad", "fields": {"ref": {"type": "many2one"}}}`)
	_, err := ParseModel(data)
	if err == nil {
		t.Fatal("expected error for many2one without model")
	}
}

func TestParseModel_One2ManyWithoutInverse(t *testing.T) {
	data := []byte(`{"name": "bad", "fields": {"lines": {"type": "one2many", "model": "line"}}}`)
	_, err := ParseModel(data)
	if err == nil {
		t.Fatal("expected error for one2many without inverse")
	}
}

func TestParseModel_SelectionWithoutOptions(t *testing.T) {
	data := []byte(`{"name": "bad", "fields": {"status": {"type": "selection"}}}`)
	_, err := ParseModel(data)
	if err == nil {
		t.Fatal("expected error for selection without options")
	}
}

func TestParseModel_InvalidJSON(t *testing.T) {
	data := []byte(`{not valid json}`)
	_, err := ParseModel(data)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestParseModel_FieldWithoutType(t *testing.T) {
	data := []byte(`{"name": "bad", "fields": {"name": {}}}`)
	_, err := ParseModel(data)
	if err == nil {
		t.Fatal("expected error for field without type")
	}
}

func TestParseModel_AutoCrud(t *testing.T) {
	data := []byte(`{
		"name": "customer",
		"fields": {
			"name":  { "type": "string", "required": true, "max": 100 },
			"email": { "type": "email", "unique": true },
			"active": { "type": "boolean", "default": true }
		}
	}`)

	model, err := ParseModel(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !model.Fields["name"].Required {
		t.Error("expected name to be required")
	}
	if !model.Fields["email"].Unique {
		t.Error("expected email to be unique")
	}
}

func TestParseModel_NewFieldTypes(t *testing.T) {
	data := []byte(`{
		"name": "test_all_types",
		"fields": {
			"amount":     {"type": "currency", "currency": "IDR", "precision": 0},
			"progress":   {"type": "percent", "min": 0, "max": 100},
			"snippet":    {"type": "code", "language": "python"},
			"stars":      {"type": "rating", "max_stars": 5, "half_stars": true},
			"hex":        {"type": "color"},
			"location":   {"type": "geolocation", "draw_mode": "point"},
			"priority":   {"type": "radio", "options": ["low","medium","high"]},
			"active":     {"type": "toggle"},
			"start_time": {"type": "time"},
			"span":       {"type": "duration"},
			"photo":      {"type": "image", "max_size": "5MB", "accept": "image/*"},
			"sign":       {"type": "signature"},
			"code_val":   {"type": "barcode", "format": "qr"},
			"bio":        {"type": "richtext", "toolbar": "minimal"},
			"readme":     {"type": "markdown"},
			"short_note": {"type": "smalltext", "rows": 3},
			"secret":     {"type": "password"},
			"price":      {"type": "float", "precision": 4},
			"content":    {"type": "html"},
			"ref":        {"type": "dynamic_link", "model": "contact"}
		}
	}`)

	model, err := ParseModel(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if model.Fields["amount"].Type != FieldCurrency {
		t.Errorf("expected currency, got %s", model.Fields["amount"].Type)
	}
	if model.Fields["amount"].CurrencyCode != "IDR" {
		t.Errorf("expected IDR, got %s", model.Fields["amount"].CurrencyCode)
	}
	if model.Fields["stars"].MaxStars != 5 {
		t.Errorf("expected max_stars 5, got %d", model.Fields["stars"].MaxStars)
	}
	if !model.Fields["stars"].HalfStars {
		t.Error("expected half_stars true")
	}
	if model.Fields["snippet"].Language != "python" {
		t.Errorf("expected language python, got %s", model.Fields["snippet"].Language)
	}
	if model.Fields["location"].DrawMode != "point" {
		t.Errorf("expected draw_mode point, got %s", model.Fields["location"].DrawMode)
	}
	if model.Fields["bio"].Toolbar != "minimal" {
		t.Errorf("expected toolbar minimal, got %s", model.Fields["bio"].Toolbar)
	}
	if model.Fields["short_note"].Rows != 3 {
		t.Errorf("expected rows 3, got %d", model.Fields["short_note"].Rows)
	}
	if model.Fields["photo"].Accept != "image/*" {
		t.Errorf("expected accept image/*, got %s", model.Fields["photo"].Accept)
	}
	if model.Fields["code_val"].Format != "qr" {
		t.Errorf("expected format qr, got %s", model.Fields["code_val"].Format)
	}
	if model.Fields["price"].Type != FieldFloat {
		t.Errorf("expected float, got %s", model.Fields["price"].Type)
	}
}

func TestParseModel_WidgetAndBehavior(t *testing.T) {
	data := []byte(`{
		"name": "test_behavior",
		"fields": {
			"status": {"type": "selection", "options": ["draft","done"], "widget": "statusbar"},
			"name":   {"type": "string", "depends_on": "status == 'draft'", "readonly_if": "status != 'draft'"},
			"email":  {"type": "email", "mandatory_if": "type == 'company'"},
			"total":  {"type": "currency", "currency": "USD", "formula": "qty * price"},
			"source": {"type": "string", "fetch_from": "contact_id.source"}
		}
	}`)

	model, err := ParseModel(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if model.Fields["status"].Widget != "statusbar" {
		t.Errorf("expected widget statusbar, got %s", model.Fields["status"].Widget)
	}
	if model.Fields["name"].DependsOn != "status == 'draft'" {
		t.Errorf("expected depends_on, got %s", model.Fields["name"].DependsOn)
	}
	if model.Fields["name"].ReadOnlyIf != "status != 'draft'" {
		t.Errorf("expected readonly_if, got %s", model.Fields["name"].ReadOnlyIf)
	}
	if model.Fields["email"].MandatoryIf != "type == 'company'" {
		t.Errorf("expected mandatory_if, got %s", model.Fields["email"].MandatoryIf)
	}
	if model.Fields["total"].Formula != "qty * price" {
		t.Errorf("expected formula, got %s", model.Fields["total"].Formula)
	}
	if model.Fields["source"].FetchFrom != "contact_id.source" {
		t.Errorf("expected fetch_from, got %s", model.Fields["source"].FetchFrom)
	}
}

func TestParseModel_RadioWithoutOptions(t *testing.T) {
	data := []byte(`{"name": "bad", "fields": {"prio": {"type": "radio"}}}`)
	_, err := ParseModel(data)
	if err == nil {
		t.Fatal("expected error for radio without options")
	}
}

func TestParseModel_DynamicLinkWithoutModel(t *testing.T) {
	data := []byte(`{"name": "bad", "fields": {"ref": {"type": "dynamic_link"}}}`)
	_, err := ParseModel(data)
	if err == nil {
		t.Fatal("expected error for dynamic_link without model")
	}
}
