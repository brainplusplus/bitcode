package runtime

import (
	"testing"
)

func TestExtensionRegistry_Register(t *testing.T) {
	reg := newExtensionRegistry()

	ext := &Extension{
		Name:      "test",
		Functions: map[string]any{"hello": func() string { return "world" }},
	}

	err := reg.register("test", ext)
	if err != nil {
		t.Fatalf("unexpected error: %s", err.Error())
	}

	got := reg.get("test")
	if got == nil {
		t.Fatal("expected extension to be registered")
	}
	if got.Name != "test" {
		t.Errorf("expected name 'test', got %q", got.Name)
	}
}

func TestExtensionRegistry_DuplicateRegister(t *testing.T) {
	reg := newExtensionRegistry()

	ext := &Extension{Name: "test", Functions: map[string]any{}}
	reg.register("test", ext)

	err := reg.register("test", ext)
	if err == nil {
		t.Error("expected error for duplicate registration")
	}
}

func TestExtensionRegistry_GetNonExistent(t *testing.T) {
	reg := newExtensionRegistry()

	got := reg.get("nonexistent")
	if got != nil {
		t.Error("expected nil for non-existent extension")
	}
}

func TestExtensionRegistry_All(t *testing.T) {
	reg := newExtensionRegistry()

	reg.register("a", &Extension{Name: "a", Functions: map[string]any{}})
	reg.register("b", &Extension{Name: "b", Functions: map[string]any{}})

	all := reg.all()
	if len(all) != 2 {
		t.Errorf("expected 2 extensions, got %d", len(all))
	}
}

func TestWithExtension_NilStructsAndConstants(t *testing.T) {
	rt := NewRuntime(
		WithExtension("test", Extension{
			Name:      "test",
			Functions: map[string]any{"greet": func(name string) string { return "Hello " + name }},
			Structs:   nil,
			Constants: nil,
		}),
	)

	ext := rt.extensions.get("test")
	if ext == nil {
		t.Fatal("extension should be registered")
	}
	if ext.Functions == nil {
		t.Error("functions should not be nil")
	}
}

func TestWithExtension_MultipleExtensions(t *testing.T) {
	rt := NewRuntime(
		WithExtension("ext1", Extension{
			Name:      "ext1",
			Functions: map[string]any{"fn1": func() string { return "one" }},
		}),
		WithExtension("ext2", Extension{
			Name:      "ext2",
			Functions: map[string]any{"fn2": func() string { return "two" }},
		}),
	)

	ext1 := rt.extensions.get("ext1")
	ext2 := rt.extensions.get("ext2")

	if ext1 == nil || ext2 == nil {
		t.Fatal("both extensions should be registered")
	}
}

func TestExtensionFunctionsInEnv(t *testing.T) {
	rt := NewRuntime(
		WithExtension("myext", Extension{
			Name: "myext",
			Functions: map[string]any{
				"add": func(a, b int) int { return a + b },
			},
		}),
	)

	if _, ok := rt.stdlibEnv["myext"]; !ok {
		t.Error("extension functions should be injected into stdlibEnv")
	}
}
