package goja_runtime

import (
	"testing"

	"github.com/bitcode-framework/bitcode/internal/runtime/embedded"
)

func TestGojaRuntimeName(t *testing.T) {
	rt := New()
	if rt.Name() != "goja" {
		t.Errorf("expected 'goja', got '%s'", rt.Name())
	}
}

func TestGojaVMCreateAndClose(t *testing.T) {
	rt := New()
	vm, err := rt.NewVM(embedded.VMOptions{})
	if err != nil {
		t.Fatalf("failed to create VM: %v", err)
	}
	vm.Close()
}

func TestGojaVMExecuteSimple(t *testing.T) {
	rt := New()
	vm, _ := rt.NewVM(embedded.VMOptions{})
	defer vm.Close()

	vm.InjectParams(map[string]any{"name": "test"})
	result, err := vm.Execute("1 + 2", "test.js")
	if err != nil {
		t.Fatalf("execution failed: %v", err)
	}
	if result != int64(3) {
		t.Errorf("expected 3, got %v (type %T)", result, result)
	}
}

func TestGojaVMExecuteWithParams(t *testing.T) {
	rt := New()
	vm, _ := rt.NewVM(embedded.VMOptions{})
	defer vm.Close()

	vm.InjectParams(map[string]any{"x": 10, "y": 20})
	result, err := vm.Execute("params.x + params.y", "test.js")
	if err != nil {
		t.Fatalf("execution failed: %v", err)
	}
	if result != int64(30) {
		t.Errorf("expected 30, got %v", result)
	}
}

func TestGojaVMExecuteModuleExports(t *testing.T) {
	rt := New()
	vm, _ := rt.NewVM(embedded.VMOptions{})
	defer vm.Close()

	vm.InjectParams(map[string]any{"value": 42})

	code := `
	({
		execute: function(bitcode, params) {
			return { result: params.value * 2 };
		}
	})
	`
	result, err := vm.Execute(code, "test.js")
	if err != nil {
		t.Fatalf("execution failed: %v", err)
	}
	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", result)
	}
	if m["result"] != int64(84) {
		t.Errorf("expected 84, got %v", m["result"])
	}
}

func TestGojaVMSyntaxError(t *testing.T) {
	rt := New()
	vm, _ := rt.NewVM(embedded.VMOptions{})
	defer vm.Close()

	_, err := vm.Execute("function {{{", "bad.js")
	if err == nil {
		t.Error("expected syntax error")
	}
}

func TestGojaVMInterrupt(t *testing.T) {
	rt := New()
	vm, _ := rt.NewVM(embedded.VMOptions{})
	defer vm.Close()

	go func() {
		vm.Interrupt("test timeout")
	}()

	_, err := vm.Execute("while(true) {}", "infinite.js")
	if err == nil {
		t.Error("expected interrupt error")
	}
}

func TestGojaVMUndefinedResult(t *testing.T) {
	rt := New()
	vm, _ := rt.NewVM(embedded.VMOptions{})
	defer vm.Close()

	result, err := vm.Execute("undefined", "test.js")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil for undefined, got %v", result)
	}
}

func TestGojaVMNullResult(t *testing.T) {
	rt := New()
	vm, _ := rt.NewVM(embedded.VMOptions{})
	defer vm.Close()

	result, err := vm.Execute("null", "test.js")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil for null, got %v", result)
	}
}

func TestGojaVMObjectResult(t *testing.T) {
	rt := New()
	vm, _ := rt.NewVM(embedded.VMOptions{})
	defer vm.Close()

	result, err := vm.Execute(`({ name: "test", count: 42 })`, "test.js")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", result)
	}
	if m["name"] != "test" {
		t.Errorf("expected name=test, got %v", m["name"])
	}
}

func TestGojaVMArrayResult(t *testing.T) {
	rt := New()
	vm, _ := rt.NewVM(embedded.VMOptions{})
	defer vm.Close()

	result, err := vm.Execute(`[1, 2, 3]`, "test.js")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	arr, ok := result.([]any)
	if !ok {
		t.Fatalf("expected []any, got %T", result)
	}
	if len(arr) != 3 {
		t.Errorf("expected 3 elements, got %d", len(arr))
	}
}

func TestGojaVMDirectFunction(t *testing.T) {
	rt := New()
	vm, _ := rt.NewVM(embedded.VMOptions{})
	defer vm.Close()

	vm.InjectParams(map[string]any{"n": 5})

	code := `(function(bitcode, params) { return params.n * 3; })`
	result, err := vm.Execute(code, "test.js")
	if err != nil {
		t.Fatalf("execution failed: %v", err)
	}
	if result != int64(15) {
		t.Errorf("expected 15, got %v", result)
	}
}
