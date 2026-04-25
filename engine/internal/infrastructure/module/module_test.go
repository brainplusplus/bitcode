package module

import (
	"testing"

	"github.com/bitcode-framework/bitcode/internal/compiler/parser"
)

func TestRegistry_RegisterAndGet(t *testing.T) {
	reg := NewRegistry()
	mod := &parser.ModuleDefinition{Name: "sales", Version: "1.0.0"}
	reg.Register(mod, "/modules/sales")

	got, err := reg.Get("sales")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Definition.Name != "sales" {
		t.Errorf("expected sales, got %s", got.Definition.Name)
	}
	if got.State != StateInstalled {
		t.Errorf("expected installed, got %s", got.State)
	}
}

func TestRegistry_IsInstalled(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&parser.ModuleDefinition{Name: "base"}, "/modules/base")

	if !reg.IsInstalled("base") {
		t.Error("base should be installed")
	}
	if reg.IsInstalled("nonexistent") {
		t.Error("nonexistent should not be installed")
	}
}

func TestRegistry_Unregister(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&parser.ModuleDefinition{Name: "crm"}, "/modules/crm")
	reg.Unregister("crm")

	if reg.IsInstalled("crm") {
		t.Error("crm should not be installed after unregister")
	}
}

func TestResolveDependencies_Simple(t *testing.T) {
	modules := map[string]*parser.ModuleDefinition{
		"base":  {Name: "base"},
		"crm":   {Name: "crm", Depends: []string{"base"}},
		"sales": {Name: "sales", Depends: []string{"base", "crm"}},
	}

	order, err := ResolveDependencies(modules, "sales")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(order) != 3 {
		t.Fatalf("expected 3 modules, got %d: %v", len(order), order)
	}
	if order[0] != "base" {
		t.Errorf("base should be first, got %s", order[0])
	}
	if order[1] != "crm" {
		t.Errorf("crm should be second, got %s", order[1])
	}
	if order[2] != "sales" {
		t.Errorf("sales should be last, got %s", order[2])
	}
}

func TestResolveDependencies_Circular(t *testing.T) {
	modules := map[string]*parser.ModuleDefinition{
		"a": {Name: "a", Depends: []string{"b"}},
		"b": {Name: "b", Depends: []string{"a"}},
	}

	_, err := ResolveDependencies(modules, "a")
	if err == nil {
		t.Fatal("expected circular dependency error")
	}
}

func TestResolveDependencies_MissingDep(t *testing.T) {
	modules := map[string]*parser.ModuleDefinition{
		"sales": {Name: "sales", Depends: []string{"nonexistent"}},
	}

	_, err := ResolveDependencies(modules, "sales")
	if err == nil {
		t.Fatal("expected missing dependency error")
	}
}

func TestParseModule(t *testing.T) {
	data := []byte(`{
		"name": "sales",
		"version": "1.0.0",
		"label": "Sales Management",
		"depends": ["base", "crm"],
		"models": ["models/*.json"],
		"apis": ["apis/*.json"],
		"permissions": {
			"order.read": "Read orders",
			"order.create": "Create orders"
		},
		"groups": {
			"sales.user": { "label": "Sales / User", "implies": ["base.user"] },
			"sales.manager": { "label": "Sales / Manager", "implies": ["sales.user"] }
		}
	}`)

	mod, err := parser.ParseModule(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mod.Name != "sales" {
		t.Errorf("expected sales, got %s", mod.Name)
	}
	if len(mod.Depends) != 2 {
		t.Errorf("expected 2 deps, got %d", len(mod.Depends))
	}
	if len(mod.Permissions) != 2 {
		t.Errorf("expected 2 permissions, got %d", len(mod.Permissions))
	}
	if len(mod.Groups) != 2 {
		t.Errorf("expected 2 groups, got %d", len(mod.Groups))
	}
}
