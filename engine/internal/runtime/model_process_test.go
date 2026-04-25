package runtime

import (
	"testing"
)

func TestParseModelProcess_Simple(t *testing.T) {
	model, op, err := parseModelProcess("models.contact.FindAll")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if model != "contact" {
		t.Errorf("expected model 'contact', got %q", model)
	}
	if op != "FindAll" {
		t.Errorf("expected op 'FindAll', got %q", op)
	}
}

func TestParseModelProcess_ModuleQualified(t *testing.T) {
	model, op, err := parseModelProcess("models.crm.contact.FindAll")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if model != "crm.contact" {
		t.Errorf("expected model 'crm.contact', got %q", model)
	}
	if op != "FindAll" {
		t.Errorf("expected op 'FindAll', got %q", op)
	}
}

func TestParseModelProcess_DynamicFinder(t *testing.T) {
	model, op, err := parseModelProcess("models.contact.FindByEmail")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if model != "contact" {
		t.Errorf("expected model 'contact', got %q", model)
	}
	if op != "FindByEmail" {
		t.Errorf("expected op 'FindByEmail', got %q", op)
	}
}

func TestParseModelProcess_ModuleQualifiedDynamicFinder(t *testing.T) {
	model, op, err := parseModelProcess("models.crm.contact.FindAllByStatusAndCity")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if model != "crm.contact" {
		t.Errorf("expected model 'crm.contact', got %q", model)
	}
	if op != "FindAllByStatusAndCity" {
		t.Errorf("expected op 'FindAllByStatusAndCity', got %q", op)
	}
}

func TestParseModelProcess_Invalid(t *testing.T) {
	_, _, err := parseModelProcess("invalid")
	if err == nil {
		t.Error("expected error for invalid name")
	}

	_, _, err = parseModelProcess("models.contact")
	if err == nil {
		t.Error("expected error for missing operation")
	}
}

func TestModelProcessRegistry_Ambiguity(t *testing.T) {
	reg := NewModelProcessRegistry()
	reg.RegisterWithModule("crm", "contact", nil)
	reg.RegisterWithModule("hrm", "contact", nil)

	_, err := reg.Execute(nil, "models.contact.FindAll", map[string]any{})
	if err == nil {
		t.Fatal("expected ambiguity error")
	}
	if !contains(err.Error(), "ambiguous") {
		t.Errorf("expected ambiguity error message, got: %s", err.Error())
	}
}

func TestModelProcessRegistry_QualifiedNotAmbiguous(t *testing.T) {
	reg := NewModelProcessRegistry()
	reg.RegisterWithModule("crm", "contact", nil)
	reg.RegisterWithModule("hrm", "contact", nil)

	modelName, _, err := parseModelProcess("models.crm.contact.FindAll")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if modelName != "crm.contact" {
		t.Errorf("expected 'crm.contact', got %q", modelName)
	}

	reg.mu.RLock()
	_, ok := reg.repos["crm.contact"]
	reg.mu.RUnlock()
	if !ok {
		t.Error("expected crm.contact to be registered")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
