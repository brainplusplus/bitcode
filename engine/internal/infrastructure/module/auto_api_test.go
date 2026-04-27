package module

import (
	"testing"

	"github.com/bitcode-framework/bitcode/internal/compiler/parser"
)

func TestGenerateAPIFromModel_Basic(t *testing.T) {
	model := &parser.ModelDefinition{
		Name:   "contact",
		Module: "crm",
		API: &parser.APIConfig{
			AutoCRUD: true,
			Auth:     true,
			Protocols: parser.ProtocolConfig{REST: true},
		},
		SearchField: []string{"name", "email"},
	}

	api := GenerateAPIFromModel(model, "crm")
	if api == nil {
		t.Fatal("expected API definition")
	}
	if api.Name != "contact_api" {
		t.Errorf("expected name 'contact_api', got %q", api.Name)
	}
	if api.Model != "contact" {
		t.Errorf("expected model 'contact', got %q", api.Model)
	}
	if !api.AutoCRUD {
		t.Error("expected auto_crud=true")
	}
	if api.BasePath != "/api/v1/crm/contacts" {
		t.Errorf("expected base_path '/api/v1/crm/contacts', got %q", api.BasePath)
	}
	if len(api.Search) != 2 {
		t.Errorf("expected 2 search fields, got %d", len(api.Search))
	}
}

func TestGenerateAPIFromModel_NoAPI(t *testing.T) {
	model := &parser.ModelDefinition{
		Name:   "internal_log",
		Module: "base",
	}
	api := GenerateAPIFromModel(model, "base")
	if api != nil {
		t.Error("expected nil API for model without api config")
	}
}

func TestMergeAPIs_OverrideEndpoints(t *testing.T) {
	auto := []*parser.APIDefinition{
		{Name: "contact_api", Model: "contact", AutoCRUD: true, BasePath: "/api/v1/crm/contacts"},
	}
	override := []*parser.APIDefinition{
		{
			Name:  "contact_api_override",
			Model: "contact",
			Endpoints: []parser.EndpointDefinition{
				{Method: "POST", Path: "/:id/merge", Action: "merge"},
			},
		},
	}

	merged := MergeAPIs(auto, override)
	if len(merged) != 1 {
		t.Fatalf("expected 1 merged API, got %d", len(merged))
	}
	if !merged[0].AutoCRUD {
		t.Error("expected auto_crud preserved from base")
	}
	if len(merged[0].Endpoints) != 1 {
		t.Errorf("expected 1 custom endpoint, got %d", len(merged[0].Endpoints))
	}
}

func TestMergeAPIs_CustomAPIWithoutModel(t *testing.T) {
	auto := []*parser.APIDefinition{
		{Name: "contact_api", Model: "contact", AutoCRUD: true},
	}
	override := []*parser.APIDefinition{
		{Name: "custom_report", Endpoints: []parser.EndpointDefinition{
			{Method: "GET", Path: "/report", Action: "report"},
		}},
	}

	merged := MergeAPIs(auto, override)
	if len(merged) != 2 {
		t.Fatalf("expected 2 APIs (1 auto + 1 custom), got %d", len(merged))
	}
}

func TestMergeAPIs_OverrideWorkflow(t *testing.T) {
	auto := []*parser.APIDefinition{
		{Name: "order_api", Model: "order", AutoCRUD: true},
	}
	override := []*parser.APIDefinition{
		{
			Name:     "order_api",
			Model:    "order",
			Workflow: "order_workflow",
			Actions: map[string]parser.WorkflowActionDefinition{
				"confirm": {Transition: "confirm", Permission: "sales.order.confirm"},
			},
		},
	}

	merged := MergeAPIs(auto, override)
	if len(merged) != 1 {
		t.Fatalf("expected 1 merged API, got %d", len(merged))
	}
	if merged[0].Workflow != "order_workflow" {
		t.Errorf("expected workflow 'order_workflow', got %q", merged[0].Workflow)
	}
	if len(merged[0].Actions) != 1 {
		t.Errorf("expected 1 action, got %d", len(merged[0].Actions))
	}
}

func TestPluralize(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"contact", "contacts"},
		{"order", "orders"},
		{"tag", "tags"},
		{"address", "addresses"},
		{"tax", "taxes"},
		{"company", "companies"},
		{"category", "categories"},
		{"key", "keys"},
		{"day", "days"},
		{"", ""},
	}
	for _, tt := range tests {
		result := pluralize(tt.input)
		if result != tt.expected {
			t.Errorf("pluralize(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
