package embedded

import "testing"

func TestModulesFS_ContainsBase(t *testing.T) {
	data, err := ModulesFS.ReadFile("modules/base/module.json")
	if err != nil {
		t.Fatalf("failed to read embedded module.json: %v", err)
	}
	if len(data) == 0 {
		t.Error("embedded module.json is empty")
	}
}

func TestModulesFS_ContainsModels(t *testing.T) {
	entries, err := ModulesFS.ReadDir("modules/base/models")
	if err != nil {
		t.Fatalf("failed to read embedded models dir: %v", err)
	}
	if len(entries) == 0 {
		t.Error("embedded models dir is empty")
	}

	found := false
	for _, e := range entries {
		if e.Name() == "user.json" {
			found = true
			break
		}
	}
	if !found {
		t.Error("user.json not found in embedded models")
	}
}

func TestModulesFS_ContainsTemplates(t *testing.T) {
	_, err := ModulesFS.ReadFile("modules/base/templates/layout.html")
	if err != nil {
		t.Fatalf("failed to read embedded layout.html: %v", err)
	}
}

func TestModulesFS_ContainsViews(t *testing.T) {
	entries, err := ModulesFS.ReadDir("modules/base/views")
	if err != nil {
		t.Fatalf("failed to read embedded views dir: %v", err)
	}
	if len(entries) < 5 {
		t.Errorf("expected at least 5 view files, got %d", len(entries))
	}
}
