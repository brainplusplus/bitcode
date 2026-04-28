package embedded

import (
	"testing"
)

func TestParseEngine(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"javascript", ""},
		{"javascript:goja", "goja"},
		{"javascript:quickjs", "quickjs"},
		{"go", "yaegi"},
		{"go:yaegi", "yaegi"},
		{"node", ""},
		{"python", ""},
		{"", ""},
	}
	for _, tt := range tests {
		result := ParseEngine(tt.input)
		if result != tt.expected {
			t.Errorf("ParseEngine(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestRegistryResolve(t *testing.T) {
	reg := NewRegistry()
	reg.Register("goja", &mockRuntime{name: "goja"})
	reg.Register("quickjs", &mockRuntime{name: "quickjs"})
	reg.Register("yaegi", &mockRuntime{name: "yaegi"})

	rt, err := reg.Resolve("javascript:quickjs", "goja")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rt.Name() != "quickjs" {
		t.Errorf("expected quickjs, got %s", rt.Name())
	}

	rt, err = reg.Resolve("javascript", "goja")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rt.Name() != "goja" {
		t.Errorf("expected goja (default), got %s", rt.Name())
	}

	rt, err = reg.Resolve("javascript", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rt.Name() != "goja" {
		t.Errorf("expected goja (hardcoded default), got %s", rt.Name())
	}

	rt, err = reg.Resolve("go", "goja")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rt.Name() != "yaegi" {
		t.Errorf("expected yaegi for 'go' runtime, got %s", rt.Name())
	}

	rt, err = reg.Resolve("go:yaegi", "goja")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rt.Name() != "yaegi" {
		t.Errorf("expected yaegi for 'go:yaegi' runtime, got %s", rt.Name())
	}

	_, err = reg.Resolve("javascript:nonexistent", "")
	if err == nil {
		t.Error("expected error for nonexistent engine")
	}
}

func TestRegistryNames(t *testing.T) {
	reg := NewRegistry()
	reg.Register("goja", &mockRuntime{name: "goja"})
	reg.Register("quickjs", &mockRuntime{name: "quickjs"})

	names := reg.Names()
	if len(names) != 2 {
		t.Errorf("expected 2 names, got %d", len(names))
	}
}

func TestParseSearchOpts(t *testing.T) {
	opts := ParseSearchOpts(map[string]any{
		"domain":  []any{[]any{"status", "=", "new"}},
		"fields":  []any{"name", "email"},
		"order":   "created_at desc",
		"limit":   float64(50),
		"offset":  float64(10),
		"include": []any{"contacts"},
	})

	if opts.Order != "created_at desc" {
		t.Errorf("expected order 'created_at desc', got '%s'", opts.Order)
	}
	if opts.Limit != 50 {
		t.Errorf("expected limit 50, got %d", opts.Limit)
	}
	if opts.Offset != 10 {
		t.Errorf("expected offset 10, got %d", opts.Offset)
	}
	if len(opts.Fields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(opts.Fields))
	}
	if len(opts.Include) != 1 {
		t.Errorf("expected 1 include, got %d", len(opts.Include))
	}
}

func TestParseSearchOptsNil(t *testing.T) {
	opts := ParseSearchOpts(nil)
	if opts.Limit != 0 {
		t.Errorf("expected limit 0 for nil, got %d", opts.Limit)
	}
}

func TestParseHTTPOpts(t *testing.T) {
	opts := ParseHTTPOpts(map[string]any{
		"headers": map[string]any{"Authorization": "Bearer token"},
		"body":    map[string]any{"key": "value"},
		"timeout": float64(5000),
		"proxy":   "http://proxy:8080",
		"profile": "chrome_133",
	})

	if opts == nil {
		t.Fatal("expected non-nil opts")
	}
	if opts.Headers["Authorization"] != "Bearer token" {
		t.Errorf("expected Authorization header")
	}
	if opts.Timeout != 5000 {
		t.Errorf("expected timeout 5000, got %d", opts.Timeout)
	}
	if opts.Proxy != "http://proxy:8080" {
		t.Errorf("expected proxy")
	}
	if opts.Profile != "chrome_133" {
		t.Errorf("expected profile chrome_133")
	}
}

func TestParseHTTPOptsNil(t *testing.T) {
	opts := ParseHTTPOpts(nil)
	if opts != nil {
		t.Error("expected nil for nil input")
	}
}

func TestToInt(t *testing.T) {
	tests := []struct {
		input    any
		expected int
	}{
		{42, 42},
		{int64(100), 100},
		{float64(3.14), 3},
		{float32(2.5), 2},
		{"not a number", 0},
		{nil, 0},
	}
	for _, tt := range tests {
		result := ToInt(tt.input)
		if result != tt.expected {
			t.Errorf("ToInt(%v) = %d, want %d", tt.input, result, tt.expected)
		}
	}
}

func TestToStringSlice(t *testing.T) {
	result := ToStringSlice([]any{"a", "b", "c"})
	if len(result) != 3 || result[0] != "a" {
		t.Errorf("unexpected result: %v", result)
	}

	result = ToStringSlice([]string{"x", "y"})
	if len(result) != 2 || result[0] != "x" {
		t.Errorf("unexpected result: %v", result)
	}

	result = ToStringSlice(42)
	if result != nil {
		t.Errorf("expected nil for non-slice, got %v", result)
	}
}

type mockRuntime struct {
	name string
}

func (m *mockRuntime) Name() string                    { return m.name }
func (m *mockRuntime) NewVM(opts VMOptions) (VM, error) { return nil, nil }
