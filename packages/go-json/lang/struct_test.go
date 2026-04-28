package lang

import (
	"strings"
	"testing"
)

func TestStruct_BasicConstruction(t *testing.T) {
	result := compileAndRun(t, `{
		"structs": {
			"Person": {
				"fields": {
					"name": "string",
					"age": "int"
				}
			}
		},
		"steps": [
			{"let": "p", "new": "Person", "with": {"name": "'Alice'", "age": "30"}},
			{"return": "p"}
		]
	}`, nil)

	p, ok := result.Value.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", result.Value)
	}
	if p["name"] != "Alice" {
		t.Errorf("expected name=Alice, got %v", p["name"])
	}
	if !numEq(p["age"], 30) {
		t.Errorf("expected age=30, got %v", p["age"])
	}
	if p["_type"] != "Person" {
		t.Errorf("expected _type=Person, got %v", p["_type"])
	}
}

func TestStruct_FieldAccess(t *testing.T) {
	result := compileAndRun(t, `{
		"structs": {
			"Person": {
				"fields": {
					"name": "string",
					"age": "int"
				}
			}
		},
		"steps": [
			{"let": "p", "new": "Person", "with": {"name": "'Bob'", "age": "25"}},
			{"return": "p.name"}
		]
	}`, nil)

	if result.Value != "Bob" {
		t.Errorf("expected 'Bob', got %v", result.Value)
	}
}

func TestStruct_DefaultValues(t *testing.T) {
	result := compileAndRun(t, `{
		"structs": {
			"Config": {
				"fields": {
					"host": "string",
					"port": {"type": "int", "default": 8080}
				}
			}
		},
		"steps": [
			{"let": "c", "new": "Config", "with": {"host": "'localhost'"}},
			{"return": "c.port"}
		]
	}`, nil)

	if !numEq(result.Value, 8080) {
		t.Errorf("expected 8080, got %v", result.Value)
	}
}

func TestStruct_NullableFieldDefaultsToNil(t *testing.T) {
	result := compileAndRun(t, `{
		"structs": {
			"Person": {
				"fields": {
					"name": "string",
					"nickname": "?string"
				}
			}
		},
		"steps": [
			{"let": "p", "new": "Person", "with": {"name": "'Alice'"}},
			{"return": "p.nickname"}
		]
	}`, nil)

	if result.Value != nil {
		t.Errorf("expected nil, got %v", result.Value)
	}
}

func TestStruct_MissingRequiredField_Error(t *testing.T) {
	program, err := Parse([]byte(`{
		"structs": {
			"Person": {
				"fields": {
					"name": "string",
					"age": "int"
				}
			}
		},
		"steps": [
			{"let": "p", "new": "Person", "with": {"name": "'Alice'"}}
		]
	}`))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	engine := NewExprLangEngine()
	compiled, err := Compile(program, engine, DefaultLimits())
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	vm := NewVM(compiled, engine)
	_, err = vm.Execute(nil)
	if err == nil {
		t.Fatal("expected error for missing required field")
	}
	if !strings.Contains(err.Error(), "requires field") {
		t.Errorf("expected 'requires field' error, got: %v", err)
	}
}

func TestStruct_UnknownType_CompileError(t *testing.T) {
	program, err := Parse([]byte(`{
		"structs": {
			"Person": {
				"fields": {
					"address": "UnknownType"
				}
			}
		},
		"steps": []
	}`))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	engine := NewExprLangEngine()
	_, err = Compile(program, engine, DefaultLimits())
	if err == nil {
		t.Fatal("expected compile error for unknown type")
	}
	if !strings.Contains(err.Error(), "unknown type") {
		t.Errorf("expected 'unknown type' error, got: %v", err)
	}
}

func TestStruct_ForwardReference(t *testing.T) {
	result := compileAndRun(t, `{
		"structs": {
			"Person": {
				"fields": {
					"name": "string",
					"address": "?Address"
				}
			},
			"Address": {
				"fields": {
					"city": "string"
				}
			}
		},
		"steps": [
			{"let": "p", "new": "Person", "with": {"name": "'Alice'"}},
			{"return": "p.name"}
		]
	}`, nil)

	if result.Value != "Alice" {
		t.Errorf("expected 'Alice', got %v", result.Value)
	}
}

func TestStruct_CircularNonNullable_CompileError(t *testing.T) {
	program, err := Parse([]byte(`{
		"structs": {
			"A": {
				"fields": {
					"b": "B"
				}
			},
			"B": {
				"fields": {
					"a": "A"
				}
			}
		},
		"steps": []
	}`))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	engine := NewExprLangEngine()
	_, err = Compile(program, engine, DefaultLimits())
	if err == nil {
		t.Fatal("expected compile error for circular non-nullable struct")
	}
	if !strings.Contains(err.Error(), "circular") {
		t.Errorf("expected 'circular' error, got: %v", err)
	}
}

func TestStruct_CircularNullable_Allowed(t *testing.T) {
	program, err := Parse([]byte(`{
		"structs": {
			"A": {
				"fields": {
					"b": "?B"
				}
			},
			"B": {
				"fields": {
					"a": "?A"
				}
			}
		},
		"steps": [
			{"let": "a", "new": "A", "with": {}},
			{"return": "a"}
		]
	}`))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	engine := NewExprLangEngine()
	_, err = Compile(program, engine, DefaultLimits())
	if err != nil {
		t.Fatalf("expected no compile error for nullable circular, got: %v", err)
	}
}

func TestStruct_NestedPropertyMutation(t *testing.T) {
	result := compileAndRun(t, `{
		"structs": {
			"Person": {
				"fields": {
					"name": "string",
					"age": "int"
				}
			}
		},
		"steps": [
			{"let": "p", "new": "Person", "with": {"name": "'Alice'", "age": "30"}},
			{"set": "p.name", "value": "Bob"},
			{"return": "p.name"}
		]
	}`, nil)

	if result.Value != "Bob" {
		t.Errorf("expected 'Bob', got %v", result.Value)
	}
}

func TestStruct_WithNullValue(t *testing.T) {
	result := compileAndRun(t, `{
		"structs": {
			"Person": {
				"fields": {
					"name": "string",
					"nickname": "?string"
				}
			}
		},
		"steps": [
			{"let": "p", "new": "Person", "with": {"name": "'Alice'", "nickname": "nil"}},
			{"return": "p.nickname"}
		]
	}`, nil)

	if result.Value != nil {
		t.Errorf("expected nil, got %v", result.Value)
	}
}
