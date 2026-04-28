package parser

import (
	"encoding/json"
	"strings"
	"testing"
)

func mustUnmarshalModelJSON(t *testing.T, data string) ModelDefinition {
	t.Helper()

	var model ModelDefinition
	if err := json.Unmarshal([]byte(data), &model); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}

	return model
}

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

func TestParseModel_NoPrimaryKeyBackwardCompatible(t *testing.T) {
	data := []byte(`{
		"name": "customer",
		"fields": {
			"name": {"type": "string", "required": true}
		}
	}`)

	model, err := ParseModel(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if model.PrimaryKey != nil {
		t.Fatal("expected primary key to be nil for backward compatibility")
	}
}

func TestModelDefinition_UnmarshalPrimaryKeyStrategies(t *testing.T) {
	tests := []struct {
		name        string
		json        string
		assertions  func(t *testing.T, model ModelDefinition)
	}{
		{
			name: "auto_increment",
			json: `{
				"name": "invoice",
				"primary_key": {"strategy": "auto_increment"},
				"fields": {"id": {"type": "integer"}}
			}`,
			assertions: func(t *testing.T, model ModelDefinition) {
				if model.PrimaryKey == nil || model.PrimaryKey.Strategy != PKAutoIncrement {
					t.Fatalf("expected auto_increment strategy, got %+v", model.PrimaryKey)
				}
			},
		},
		{
			name: "composite",
			json: `{
				"name": "invoice_line",
				"primary_key": {"strategy": "composite", "fields": ["invoice_id", "line_no"], "surrogate": false},
				"fields": {
					"invoice_id": {"type": "many2one", "model": "invoice", "required": true},
					"line_no": {"type": "integer", "required": true}
				}
			}`,
			assertions: func(t *testing.T, model ModelDefinition) {
				if model.PrimaryKey == nil || model.PrimaryKey.Strategy != PKComposite {
					t.Fatalf("expected composite strategy, got %+v", model.PrimaryKey)
				}
				if len(model.PrimaryKey.Fields) != 2 {
					t.Fatalf("expected 2 composite fields, got %d", len(model.PrimaryKey.Fields))
				}
				if model.PrimaryKey.IsSurrogate() {
					t.Fatal("expected surrogate false")
				}
			},
		},
		{
			name: "uuid",
			json: `{
				"name": "contact",
				"primary_key": {"strategy": "uuid", "version": "v7", "field": "id"},
				"fields": {"id": {"type": "string"}}
			}`,
			assertions: func(t *testing.T, model ModelDefinition) {
				if model.PrimaryKey == nil || model.PrimaryKey.Strategy != PKUUID {
					t.Fatalf("expected uuid strategy, got %+v", model.PrimaryKey)
				}
				if model.PrimaryKey.Version != "v7" {
					t.Fatalf("expected uuid version v7, got %q", model.PrimaryKey.Version)
				}
			},
		},
		{
			name: "natural_key",
			json: `{
				"name": "country",
				"primary_key": {"strategy": "natural_key", "field": "code"},
				"fields": {"code": {"type": "string", "required": true}}
			}`,
			assertions: func(t *testing.T, model ModelDefinition) {
				if model.PrimaryKey == nil || model.PrimaryKey.Strategy != PKNaturalKey {
					t.Fatalf("expected natural_key strategy, got %+v", model.PrimaryKey)
				}
				if model.PrimaryKey.Field != "code" {
					t.Fatalf("expected field code, got %q", model.PrimaryKey.Field)
				}
			},
		},
		{
			name: "naming_series",
			json: `{
				"name": "sales_order",
				"primary_key": {
					"strategy": "naming_series",
					"field": "name",
					"format": "SO-{YYYY}-{#####}",
					"sequence": {"reset": "yearly", "step": 2}
				},
				"fields": {"name": {"type": "string", "required": true}}
			}`,
			assertions: func(t *testing.T, model ModelDefinition) {
				if model.PrimaryKey == nil || model.PrimaryKey.Strategy != PKNamingSeries {
					t.Fatalf("expected naming_series strategy, got %+v", model.PrimaryKey)
				}
				if model.PrimaryKey.Sequence == nil || model.PrimaryKey.Sequence.Reset != "yearly" || model.PrimaryKey.Sequence.Step != 2 {
					t.Fatalf("expected yearly sequence with step 2, got %+v", model.PrimaryKey.Sequence)
				}
			},
		},
		{
			name: "manual",
			json: `{
				"name": "legacy_import",
				"primary_key": {"strategy": "manual", "field": "legacy_id"},
				"fields": {"legacy_id": {"type": "string", "required": true}}
			}`,
			assertions: func(t *testing.T, model ModelDefinition) {
				if model.PrimaryKey == nil || model.PrimaryKey.Strategy != PKManual {
					t.Fatalf("expected manual strategy, got %+v", model.PrimaryKey)
				}
				if model.PrimaryKey.Field != "legacy_id" {
					t.Fatalf("expected field legacy_id, got %q", model.PrimaryKey.Field)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := mustUnmarshalModelJSON(t, tt.json)
			tt.assertions(t, model)
		})
	}
}

func TestModelDefinition_UnmarshalFieldAutoFormat(t *testing.T) {
	model := mustUnmarshalModelJSON(t, `{
		"name": "sales_order",
		"fields": {
			"name": {
				"type": "string",
				"name_format": "SO-{#####}",
				"auto_format": {
					"format": "SO-{YYYY}-{#####}",
					"sequence": {"reset": "monthly", "step": 5}
				}
			}
		}
	}`)

	field := model.Fields["name"]
	if field.AutoFormat == nil {
		t.Fatal("expected auto_format to be parsed")
	}
	if field.AutoFormat.Format != "SO-{YYYY}-{#####}" {
		t.Fatalf("expected auto_format format to be parsed, got %q", field.AutoFormat.Format)
	}
	if field.AutoFormat.Sequence == nil || field.AutoFormat.Sequence.Reset != "monthly" || field.AutoFormat.Sequence.Step != 5 {
		t.Fatalf("expected auto_format sequence to be parsed, got %+v", field.AutoFormat.Sequence)
	}
}

func TestParseModel_PrimaryKeyValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantErr string
	}{
		{
			name: "auto_increment invalid sequence reset",
			json: `{
				"name": "invoice",
				"primary_key": {"strategy": "auto_increment", "sequence": {"reset": "weekly"}},
				"fields": {"id": {"type": "integer"}}
			}`,
			wantErr: "primary key sequence reset must be one of",
		},
		{
			name: "composite requires at least two fields",
			json: `{
				"name": "invoice_line",
				"primary_key": {"strategy": "composite", "fields": ["invoice_id"]},
				"fields": {"invoice_id": {"type": "many2one", "model": "invoice", "required": true}}
			}`,
			wantErr: "composite primary key must specify at least two fields",
		},
		{
			name: "uuid format version requires format",
			json: `{
				"name": "contact",
				"primary_key": {"strategy": "uuid", "version": "format"},
				"fields": {"id": {"type": "string"}}
			}`,
			wantErr: "uuid primary key with format version must specify format",
		},
		{
			name: "natural_key field must be required",
			json: `{
				"name": "country",
				"primary_key": {"strategy": "natural_key", "field": "code"},
				"fields": {"code": {"type": "string"}}
			}`,
			wantErr: "natural key field \"code\" must be required",
		},
		{
			name: "naming_series requires format",
			json: `{
				"name": "sales_order",
				"primary_key": {"strategy": "naming_series", "field": "name"},
				"fields": {"name": {"type": "string", "required": true}}
			}`,
			wantErr: "naming_series primary key must specify format",
		},
		{
			name: "manual field must exist",
			json: `{
				"name": "legacy_import",
				"primary_key": {"strategy": "manual", "field": "legacy_id"},
				"fields": {"name": {"type": "string"}}
			}`,
			wantErr: "primary key field \"legacy_id\" does not exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseModel([]byte(tt.json))
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected error containing %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}

func TestModelDefinition_OptionsDefaults(t *testing.T) {
	data := []byte(`{
		"name": "customer",
		"fields": {
			"name": {"type": "string", "required": true}
		}
	}`)

	model, err := ParseModel(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if model.IsVersion() {
		t.Error("expected version default false")
	}
	if !model.IsTimestamps() {
		t.Error("expected timestamps default true")
	}
	if !model.IsTimestampsBy() {
		t.Error("expected timestamps_by default true")
	}
	if model.IsSoftDeletes() {
		t.Error("expected soft_deletes default false")
	}
	if model.IsSoftDeletesBy() {
		t.Error("expected soft_deletes_by default false")
	}
}

func TestModelDefinition_OptionsExplicit(t *testing.T) {
	data := []byte(`{
		"name": "order",
		"version": true,
		"timestamps": false,
		"timestamps_by": false,
		"soft_deletes": true,
		"soft_deletes_by": true,
		"fields": {
			"name": {"type": "string", "required": true}
		}
	}`)

	model, err := ParseModel(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !model.IsVersion() {
		t.Error("expected version true")
	}
	if model.IsTimestamps() {
		t.Error("expected timestamps false")
	}
	if model.IsTimestampsBy() {
		t.Error("expected timestamps_by false")
	}
	if !model.IsSoftDeletes() {
		t.Error("expected soft_deletes true")
	}
	if !model.IsSoftDeletesBy() {
		t.Error("expected soft_deletes_by true")
	}
}

func TestParseModel_WithAPITrue(t *testing.T) {
	data := []byte(`{"name":"tag","module":"crm","fields":{"name":{"type":"string"}},"api":true}`)
	model, err := ParseModel(data)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if model.API == nil {
		t.Fatal("expected API config from shorthand")
	}
	if !model.API.AutoCRUD {
		t.Error("expected auto_crud=true")
	}
	if !model.API.Auth {
		t.Error("expected auth=true")
	}
	if !model.API.Protocols.REST {
		t.Error("expected rest=true")
	}
}

func TestParseModel_WithAPIObject(t *testing.T) {
	data := []byte(`{"name":"contact","module":"crm","fields":{"name":{"type":"string"}},"api":{"auto_crud":true,"auth":true,"modal":true,"protocols":{"rest":true,"graphql":true,"websocket":false}}}`)
	model, err := ParseModel(data)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if model.API == nil {
		t.Fatal("expected API config")
	}
	if !model.API.Modal {
		t.Error("expected modal=true")
	}
	if !model.API.Protocols.GraphQL {
		t.Error("expected graphql=true")
	}
}

func TestParseModel_WithFieldMaskAndGroups(t *testing.T) {
	data := []byte(`{"name":"emp","module":"hrm","fields":{"phone":{"type":"string","mask":true,"mask_length":4},"salary":{"type":"decimal","groups":["hr.manager"]}}}`)
	model, err := ParseModel(data)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if !model.Fields["phone"].Mask {
		t.Error("expected mask=true")
	}
	if model.Fields["phone"].MaskLength != 4 {
		t.Error("expected mask_length=4")
	}
	if len(model.Fields["salary"].Groups) != 1 {
		t.Error("expected 1 group")
	}
}

func TestModelDefinition_OptionsPartial(t *testing.T) {
	data := []byte(`{
		"name": "product",
		"version": true,
		"soft_deletes": true,
		"fields": {
			"name": {"type": "string", "required": true}
		}
	}`)

	model, err := ParseModel(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !model.IsVersion() {
		t.Error("expected version true")
	}
	if !model.IsTimestamps() {
		t.Error("expected timestamps default true when not specified")
	}
	if !model.IsTimestampsBy() {
		t.Error("expected timestamps_by default true when not specified")
	}
	if !model.IsSoftDeletes() {
		t.Error("expected soft_deletes true")
	}
	if model.IsSoftDeletesBy() {
		t.Error("expected soft_deletes_by default false when not specified")
	}
}

func TestParseModel_NewTypes(t *testing.T) {
	data := []byte(`{
		"name": "test_new_types",
		"fields": {
			"token":     { "type": "uuid" },
			"client_ip": { "type": "ip" },
			"server_ip": { "type": "ipv6" },
			"birth_year": { "type": "year" },
			"embedding": { "type": "vector", "dimensions": 1536 },
			"raw_data":  { "type": "binary" },
			"metadata":  { "type": "json:object" },
			"tags_data": { "type": "json:array" }
		}
	}`)

	model, err := ParseModel(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	checks := map[string]FieldType{
		"token":      FieldUUID,
		"client_ip":  FieldIP,
		"server_ip":  FieldIPv6,
		"birth_year": FieldYear,
		"embedding":  FieldVector,
		"raw_data":   FieldBinary,
		"metadata":   FieldJSONObject,
		"tags_data":  FieldJSONArray,
	}

	for name, expected := range checks {
		if model.Fields[name].Type != expected {
			t.Errorf("field %q: expected type %q, got %q", name, expected, model.Fields[name].Type)
		}
	}

	if model.Fields["embedding"].Dimensions != 1536 {
		t.Errorf("expected dimensions 1536, got %d", model.Fields["embedding"].Dimensions)
	}
}

func TestParseModel_VectorWithoutDimensions(t *testing.T) {
	data := []byte(`{"name": "bad", "fields": {"emb": {"type": "vector"}}}`)
	_, err := ParseModel(data)
	if err == nil {
		t.Fatal("expected error for vector without dimensions")
	}
	if !strings.Contains(err.Error(), "dimensions") {
		t.Errorf("expected dimensions error, got: %v", err)
	}
}

func TestParseModel_InvalidTypeVariant(t *testing.T) {
	data := []byte(`{"name": "bad", "fields": {"x": {"type": "string:big"}}}`)
	_, err := ParseModel(data)
	if err == nil {
		t.Fatal("expected error for invalid type variant")
	}
	if !strings.Contains(err.Error(), "does not support variants") {
		t.Errorf("expected variant error, got: %v", err)
	}
}

func TestParseModel_IPVariants(t *testing.T) {
	data := []byte(`{
		"name": "test_ip",
		"fields": {
			"v4": { "type": "ip:v4" },
			"v6": { "type": "ip:v6" }
		}
	}`)

	model, err := ParseModel(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if model.Fields["v4"].Type != FieldIP {
		t.Errorf("ip:v4 should resolve to FieldIP, got %q", model.Fields["v4"].Type)
	}
	if model.Fields["v6"].Type != FieldIPv6 {
		t.Errorf("ip:v6 should resolve to FieldIPv6, got %q", model.Fields["v6"].Type)
	}
}

func TestParseModel_StorageHintValid(t *testing.T) {
	data := []byte(`{
		"name": "test_storage",
		"fields": {
			"big_id":   { "type": "integer", "storage": "bigint" },
			"amount":   { "type": "decimal", "storage": "numeric" },
			"content":  { "type": "text", "storage": "longtext" },
			"price":    { "type": "currency", "storage": "numeric" }
		}
	}`)

	_, err := ParseModel(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseModel_StorageHintInvalid(t *testing.T) {
	data := []byte(`{"name": "bad", "fields": {"x": {"type": "integer", "storage": "varchar"}}}`)
	_, err := ParseModel(data)
	if err == nil {
		t.Fatal("expected error for invalid storage hint")
	}
	if !strings.Contains(err.Error(), "invalid storage hint") {
		t.Errorf("expected storage hint error, got: %v", err)
	}
}

func TestParseModel_CurrencyFieldMutualExclusion(t *testing.T) {
	data := []byte(`{"name": "bad", "fields": {"amount": {"type": "currency", "currency": "IDR", "currency_field": "curr_id"}}}`)
	_, err := ParseModel(data)
	if err == nil {
		t.Fatal("expected error for currency + currency_field")
	}
	if !strings.Contains(err.Error(), "cannot have both") {
		t.Errorf("expected mutual exclusion error, got: %v", err)
	}
}

func TestParseModel_TitleFieldFormat(t *testing.T) {
	data := []byte(`{
		"name": "test_title",
		"title_field": "{data.code} - {data.name}",
		"fields": {
			"code": { "type": "string" },
			"name": { "type": "string" },
			"price": { "type": "decimal" }
		}
	}`)

	model, err := ParseModel(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(model.SearchField) != 2 {
		t.Fatalf("expected 2 search fields, got %d: %v", len(model.SearchField), model.SearchField)
	}

	hasCode := false
	hasName := false
	for _, f := range model.SearchField {
		if f == "code" {
			hasCode = true
		}
		if f == "name" {
			hasName = true
		}
	}
	if !hasCode || !hasName {
		t.Errorf("expected search fields [code, name], got %v", model.SearchField)
	}
}

func TestParseModel_TablePlural(t *testing.T) {
	data := []byte(`{
		"name": "contact",
		"table": { "prefix": "crm", "plural": true },
		"fields": { "name": { "type": "string" } }
	}`)

	model, err := ParseModel(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if model.TablePlural == nil || !*model.TablePlural {
		t.Error("expected TablePlural to be true")
	}
	if model.TablePrefix == nil || *model.TablePrefix != "crm" {
		t.Error("expected TablePrefix to be 'crm'")
	}
}

func TestParseModel_HiddenField(t *testing.T) {
	data := []byte(`{
		"name": "test_hidden",
		"fields": {
			"name":      { "type": "string" },
			"embedding": { "type": "vector", "dimensions": 768, "hidden": true }
		}
	}`)

	model, err := ParseModel(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !model.Fields["embedding"].Hidden {
		t.Error("expected embedding to be hidden")
	}
	if model.Fields["name"].Hidden {
		t.Error("expected name to not be hidden")
	}
}
