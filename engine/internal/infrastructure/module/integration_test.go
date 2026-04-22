package module

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func TestIntegration_DiscoverAndLoadEmbeddedModule(t *testing.T) {
	embedFS := fstest.MapFS{
		"base/module.json": {Data: []byte(`{
			"name": "base",
			"version": "1.0.0",
			"label": "Base",
			"depends": [],
			"models": ["models/*.json"],
			"apis": ["apis/*.json"]
		}`)},
		"base/models/user.json": {Data: []byte(`{
			"name": "user",
			"fields": {
				"username": {"type": "string", "required": true}
			}
		}`)},
		"base/apis/user_api.json": {Data: []byte(`{
			"name": "user_api",
			"model": "user",
			"base_path": "/api/users",
			"auto_crud": true
		}`)},
	}

	projectDir := t.TempDir()
	layered := NewLayeredFS(
		NewDiskFS(projectDir),
		NewEmbedFSFromFS(embedFS, ""),
	)

	modules, err := layered.DiscoverModules()
	if err != nil {
		t.Fatalf("DiscoverModules error: %v", err)
	}
	if len(modules) != 1 {
		t.Fatalf("expected 1 module, got %d: %v", len(modules), modules)
	}
	if modules[0] != "base" {
		t.Errorf("expected base, got %s", modules[0])
	}

	modFS := layered.SubFS("base")
	loaded, err := LoadModuleFromFS(modFS)
	if err != nil {
		t.Fatalf("LoadModuleFromFS error: %v", err)
	}
	if loaded.Definition.Name != "base" {
		t.Errorf("expected base, got %s", loaded.Definition.Name)
	}
	if len(loaded.Models) != 1 {
		t.Errorf("expected 1 model, got %d", len(loaded.Models))
	}
	if len(loaded.APIs) != 1 {
		t.Errorf("expected 1 API, got %d", len(loaded.APIs))
	}
}

func TestIntegration_ProjectOverridesEmbedded(t *testing.T) {
	embedFS := fstest.MapFS{
		"base/module.json": {Data: []byte(`{
			"name": "base",
			"version": "1.0.0",
			"depends": [],
			"models": ["models/*.json"],
			"apis": []
		}`)},
		"base/models/user.json": {Data: []byte(`{
			"name": "user",
			"fields": {
				"username": {"type": "string"}
			}
		}`)},
	}

	projectDir := t.TempDir()
	os.MkdirAll(filepath.Join(projectDir, "base", "models"), 0755)
	os.WriteFile(filepath.Join(projectDir, "base", "module.json"), []byte(`{
		"name": "base",
		"version": "1.0.0-custom",
		"depends": [],
		"models": ["models/*.json"],
		"apis": []
	}`), 0644)
	os.WriteFile(filepath.Join(projectDir, "base", "models", "user.json"), []byte(`{
		"name": "user",
		"fields": {
			"username": {"type": "string"},
			"email": {"type": "string"},
			"phone": {"type": "string"}
		}
	}`), 0644)

	layered := NewLayeredFS(
		NewDiskFS(projectDir),
		NewEmbedFSFromFS(embedFS, ""),
	)

	modFS := layered.SubFS("base")
	loaded, err := LoadModuleFromFS(modFS)
	if err != nil {
		t.Fatalf("LoadModuleFromFS error: %v", err)
	}

	if loaded.Definition.Version != "1.0.0-custom" {
		t.Errorf("expected 1.0.0-custom, got %s", loaded.Definition.Version)
	}

	if len(loaded.Models) != 1 {
		t.Fatalf("expected 1 model, got %d", len(loaded.Models))
	}
	fieldCount := len(loaded.Models[0].Fields)
	if fieldCount != 3 {
		t.Errorf("expected 3 fields (project override), got %d", fieldCount)
	}
}

func TestIntegration_MixedProjectAndEmbedded(t *testing.T) {
	embedFS := fstest.MapFS{
		"base/module.json": {Data: []byte(`{
			"name": "base",
			"version": "1.0.0",
			"depends": [],
			"models": ["models/*.json"],
			"apis": []
		}`)},
		"base/models/user.json": {Data: []byte(`{
			"name": "user",
			"fields": {
				"username": {"type": "string"}
			}
		}`)},
		"base/models/role.json": {Data: []byte(`{
			"name": "role",
			"fields": {
				"name": {"type": "string"}
			}
		}`)},
	}

	projectDir := t.TempDir()
	os.MkdirAll(filepath.Join(projectDir, "crm", "models"), 0755)
	os.WriteFile(filepath.Join(projectDir, "crm", "module.json"), []byte(`{
		"name": "crm",
		"version": "1.0.0",
		"depends": ["base"],
		"models": ["models/*.json"],
		"apis": []
	}`), 0644)
	os.WriteFile(filepath.Join(projectDir, "crm", "models", "contact.json"), []byte(`{
		"name": "contact",
		"fields": {
			"name": {"type": "string"}
		}
	}`), 0644)

	layered := NewLayeredFS(
		NewDiskFS(projectDir),
		NewEmbedFSFromFS(embedFS, ""),
	)

	modules, err := layered.DiscoverModules()
	if err != nil {
		t.Fatalf("DiscoverModules error: %v", err)
	}
	if len(modules) != 2 {
		t.Fatalf("expected 2 modules (base+crm), got %d: %v", len(modules), modules)
	}

	found := map[string]bool{}
	for _, m := range modules {
		found[m] = true
	}
	if !found["base"] {
		t.Error("base not discovered")
	}
	if !found["crm"] {
		t.Error("crm not discovered")
	}

	baseFS := layered.SubFS("base")
	baseLoaded, err := LoadModuleFromFS(baseFS)
	if err != nil {
		t.Fatalf("failed to load base: %v", err)
	}
	if len(baseLoaded.Models) != 2 {
		t.Errorf("expected 2 base models, got %d", len(baseLoaded.Models))
	}

	crmFS := layered.SubFS("crm")
	crmLoaded, err := LoadModuleFromFS(crmFS)
	if err != nil {
		t.Fatalf("failed to load crm: %v", err)
	}
	if len(crmLoaded.Models) != 1 {
		t.Errorf("expected 1 crm model, got %d", len(crmLoaded.Models))
	}
}

func TestIntegration_ThreeLayerResolution(t *testing.T) {
	embedFS := fstest.MapFS{
		"base/module.json": {Data: []byte(`{
			"name": "base",
			"version": "1.0.0-embedded",
			"depends": [],
			"models": [],
			"apis": []
		}`)},
	}

	globalDir := t.TempDir()
	os.MkdirAll(filepath.Join(globalDir, "base"), 0755)
	os.WriteFile(filepath.Join(globalDir, "base", "module.json"), []byte(`{
		"name": "base",
		"version": "1.0.0-global",
		"depends": [],
		"models": [],
		"apis": []
	}`), 0644)

	projectDir := t.TempDir()

	layered := NewLayeredFS(
		NewDiskFS(projectDir),
		NewDiskFS(globalDir),
		NewEmbedFSFromFS(embedFS, ""),
	)

	modFS := layered.SubFS("base")
	data, err := modFS.ReadFile("module.json")
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}

	if !contains(string(data), "1.0.0-global") {
		t.Errorf("expected global version to win over embedded, got: %s", string(data))
	}

	os.MkdirAll(filepath.Join(projectDir, "base"), 0755)
	os.WriteFile(filepath.Join(projectDir, "base", "module.json"), []byte(`{
		"name": "base",
		"version": "1.0.0-project",
		"depends": [],
		"models": [],
		"apis": []
	}`), 0644)

	data, err = modFS.ReadFile("module.json")
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}

	if !contains(string(data), "1.0.0-project") {
		t.Errorf("expected project version to win over global, got: %s", string(data))
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
