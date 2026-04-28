package codegen

import (
	"strings"
	"testing"

	"github.com/bitcode-framework/go-json/lang"
	"github.com/bitcode-framework/go-json/stdlib"
)

var factorialJSON = []byte(`{
	"name": "factorial",
	"functions": {
		"factorial": {
			"params": {"n": "int"},
			"returns": "int",
			"steps": [
				{"if": "n <= 1", "then": [{"return": "1"}]},
				{"let": "sub", "call": "factorial", "with": {"n": "n - 1"}},
				{"return": "n * sub"}
			]
		}
	},
	"steps": [
		{"let": "result", "call": "factorial", "with": {"n": "10"}},
		{"return": "result"}
	]
}`)

func compileTestProgram(t *testing.T) *lang.CompiledProgram {
	t.Helper()
	program, err := lang.Parse(factorialJSON)
	if err != nil {
		t.Fatalf("parse error: %s", err.Error())
	}

	engine := lang.NewExprLangEngine()
	reg := stdlib.DefaultRegistry()
	engine.AddOptions(reg.All()...)

	compiled, err := lang.Compile(program, engine, lang.DefaultLimits())
	if err != nil {
		t.Fatalf("compile error: %s", err.Error())
	}
	return compiled
}

func TestGoGenerator_Factorial(t *testing.T) {
	compiled := compileTestProgram(t)
	gen := &GoGenerator{PackageName: "main"}

	code, err := gen.Generate(compiled)
	if err != nil {
		t.Fatalf("generate error: %s", err.Error())
	}

	if !strings.Contains(code, "package main") {
		t.Error("expected 'package main'")
	}
	if !strings.Contains(code, "func factorial") {
		t.Error("expected 'func factorial'")
	}
	if !strings.Contains(code, "func main()") {
		t.Error("expected 'func main()'")
	}
	if !strings.Contains(code, "return") {
		t.Error("expected return statement")
	}
}

func TestGoGenerator_Language(t *testing.T) {
	gen := &GoGenerator{}
	if gen.Language() != "go" {
		t.Errorf("expected 'go', got %q", gen.Language())
	}
}

func TestJSGenerator_Factorial(t *testing.T) {
	compiled := compileTestProgram(t)
	gen := &JSGenerator{}

	code, err := gen.Generate(compiled)
	if err != nil {
		t.Fatalf("generate error: %s", err.Error())
	}

	if !strings.Contains(code, "function factorial") {
		t.Error("expected 'function factorial'")
	}
	if !strings.Contains(code, "return") {
		t.Error("expected return statement")
	}
}

func TestJSGenerator_Language(t *testing.T) {
	gen := &JSGenerator{}
	if gen.Language() != "javascript" {
		t.Errorf("expected 'javascript', got %q", gen.Language())
	}
}

func TestPythonGenerator_Factorial(t *testing.T) {
	compiled := compileTestProgram(t)
	gen := &PythonGenerator{}

	code, err := gen.Generate(compiled)
	if err != nil {
		t.Fatalf("generate error: %s", err.Error())
	}

	if !strings.Contains(code, "def factorial") {
		t.Error("expected 'def factorial'")
	}
	if !strings.Contains(code, "if __name__") {
		t.Error("expected 'if __name__' guard")
	}
	if !strings.Contains(code, "return") {
		t.Error("expected return statement")
	}
}

func TestPythonGenerator_Language(t *testing.T) {
	gen := &PythonGenerator{}
	if gen.Language() != "python" {
		t.Errorf("expected 'python', got %q", gen.Language())
	}
}

func TestTransformExpr_Python(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"a && b", "a  and  b"},
		{"a || b", "a  or  b"},
		{"true", "True"},
		{"false", "False"},
		{"nil", "None"},
	}

	for _, tt := range tests {
		result := transformExpr(tt.input, "python")
		if result != tt.expected {
			t.Errorf("transformExpr(%q, python) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestGoTypeMap(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"string", "string"},
		{"int", "int64"},
		{"float", "float64"},
		{"bool", "bool"},
		{"any", "any"},
		{"[]string", "[]string"},
	}

	for _, tt := range tests {
		result := goTypeMap(tt.input)
		if result != tt.expected {
			t.Errorf("goTypeMap(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestPythonTypeMap(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"string", "str"},
		{"int", "int"},
		{"float", "float"},
		{"bool", "bool"},
		{"any", "Any"},
		{"", ""},
	}

	for _, tt := range tests {
		result := pythonTypeMap(tt.input)
		if result != tt.expected {
			t.Errorf("pythonTypeMap(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
