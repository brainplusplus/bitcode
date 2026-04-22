package parser

import (
	"testing"
)

func TestParseAPI_AutoCRUD(t *testing.T) {
	data := []byte(`{
		"name": "customer_api",
		"model": "customer",
		"auto_crud": true,
		"auth": true
	}`)

	api, err := ParseAPI(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if api.Name != "customer_api" {
		t.Errorf("expected customer_api, got %s", api.Name)
	}
	if !api.Auth {
		t.Error("expected auth to be true")
	}
	if api.GetBasePath() != "/api/customers" {
		t.Errorf("expected /api/customers, got %s", api.GetBasePath())
	}

	endpoints := api.ExpandAutoCRUD()
	if len(endpoints) != 5 {
		t.Errorf("expected 5 CRUD endpoints, got %d", len(endpoints))
	}
}

func TestParseAPI_AutoCRUDWithWorkflow(t *testing.T) {
	data := []byte(`{
		"name": "order_api",
		"model": "order",
		"auto_crud": true,
		"auth": true,
		"workflow": "order_workflow",
		"actions": {
			"confirm":  { "transition": "confirm",  "permission": "order.confirm" },
			"cancel":   { "transition": "cancel",   "permission": "order.cancel" }
		}
	}`)

	api, err := ParseAPI(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	endpoints := api.ExpandAutoCRUD()
	if len(endpoints) != 7 {
		t.Errorf("expected 7 endpoints (5 CRUD + 2 actions), got %d", len(endpoints))
	}

	hasConfirm := false
	hasCancel := false
	for _, ep := range endpoints {
		if ep.Path == "/:id/confirm" {
			hasConfirm = true
			if ep.Permissions[0] != "order.confirm" {
				t.Errorf("expected order.confirm permission, got %s", ep.Permissions[0])
			}
		}
		if ep.Path == "/:id/cancel" {
			hasCancel = true
		}
	}
	if !hasConfirm {
		t.Error("expected confirm action endpoint")
	}
	if !hasCancel {
		t.Error("expected cancel action endpoint")
	}
}

func TestParseAPI_CustomEndpoints(t *testing.T) {
	data := []byte(`{
		"name": "report_api",
		"base_path": "/api/reports",
		"auth": true,
		"endpoints": [
			{ "method": "GET", "path": "/sales-summary", "permissions": ["report.read"] }
		]
	}`)

	api, err := ParseAPI(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if api.GetBasePath() != "/api/reports" {
		t.Errorf("expected /api/reports, got %s", api.GetBasePath())
	}
	if len(api.Endpoints) != 1 {
		t.Errorf("expected 1 endpoint, got %d", len(api.Endpoints))
	}
}

func TestParseAPI_MissingName(t *testing.T) {
	data := []byte(`{"model": "customer", "auto_crud": true}`)
	_, err := ParseAPI(data)
	if err == nil {
		t.Fatal("expected error for missing name")
	}
}

func TestParseAPI_AutoCRUDWithoutModel(t *testing.T) {
	data := []byte(`{"name": "bad", "auto_crud": true}`)
	_, err := ParseAPI(data)
	if err == nil {
		t.Fatal("expected error for auto_crud without model")
	}
}

func TestParseAPI_SoftDeleteDefault(t *testing.T) {
	data := []byte(`{"name": "test", "model": "test", "auto_crud": true}`)
	api, _ := ParseAPI(data)
	if !api.IsSoftDelete() {
		t.Error("expected soft_delete to default to true")
	}
}

func TestParseAPI_PageSizeDefault(t *testing.T) {
	data := []byte(`{"name": "test", "model": "test", "auto_crud": true}`)
	api, _ := ParseAPI(data)
	if api.GetPageSize() != 20 {
		t.Errorf("expected default page size 20, got %d", api.GetPageSize())
	}
}

func TestParseAPI_RLSImpliedByAuth(t *testing.T) {
	data := []byte(`{
		"name": "customer_api",
		"model": "customer",
		"auto_crud": true,
		"auth": true
	}`)

	api, err := ParseAPI(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !api.Auth {
		t.Error("auth: true should imply RLS is active")
	}
}
