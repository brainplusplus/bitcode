package lang

import (
	"strings"
	"testing"
)

func TestMethod_StepLevelCall(t *testing.T) {
	result := compileAndRun(t, `{
		"structs": {
			"Counter": {
				"fields": {
					"count": "int"
				},
				"methods": {
					"increment": {
						"steps": [
							{"set": "self.count", "expr": "self.count + 1"}
						]
					}
				}
			}
		},
		"steps": [
			{"let": "c", "new": "Counter", "with": {"count": "0"}},
			{"call": "c.increment"},
			{"call": "c.increment"},
			{"return": "c.count"}
		]
	}`, nil)

	if !numEq(result.Value, 2) {
		t.Errorf("expected 2, got %v", result.Value)
	}
}

func TestMethod_LetCallShorthand(t *testing.T) {
	result := compileAndRun(t, `{
		"structs": {
			"Person": {
				"fields": {
					"first": "string",
					"last": "string"
				},
				"methods": {
					"fullName": {
						"returns": "string",
						"steps": [
							{"return": "self.first + ' ' + self.last"}
						]
					}
				}
			}
		},
		"steps": [
			{"let": "p", "new": "Person", "with": {"first": "'Alice'", "last": "'Smith'"}},
			{"let": "name", "call": "p.fullName"},
			{"return": "name"}
		]
	}`, nil)

	if result.Value != "Alice Smith" {
		t.Errorf("expected 'Alice Smith', got %v", result.Value)
	}
}

func TestMethod_ExpressionLevelCall(t *testing.T) {
	result := compileAndRun(t, `{
		"structs": {
			"Person": {
				"fields": {
					"name": "string"
				},
				"methods": {
					"greet": {
						"params": {"greeting": "string"},
						"returns": "string",
						"steps": [
							{"return": "greeting + ', ' + self.name + '!'"}
						]
					}
				}
			}
		},
		"steps": [
			{"let": "p", "new": "Person", "with": {"name": "'Alice'"}},
			{"return": "p.greet('Hello')"}
		]
	}`, nil)

	if result.Value != "Hello, Alice!" {
		t.Errorf("expected 'Hello, Alice!', got %v", result.Value)
	}
}

func TestMethod_SelfMutation(t *testing.T) {
	result := compileAndRun(t, `{
		"structs": {
			"Person": {
				"fields": {
					"name": "string",
					"age": "int"
				},
				"methods": {
					"birthday": {
						"steps": [
							{"set": "self.age", "expr": "self.age + 1"}
						]
					}
				}
			}
		},
		"steps": [
			{"let": "p", "new": "Person", "with": {"name": "'Alice'", "age": "30"}},
			{"call": "p.birthday"},
			{"return": "p.age"}
		]
	}`, nil)

	if !numEq(result.Value, 31) {
		t.Errorf("expected 31, got %v", result.Value)
	}
}

func TestMethod_FrozenStruct_MutationBlocked(t *testing.T) {
	program, err := Parse([]byte(`{
		"structs": {
			"Point": {
				"frozen": true,
				"fields": {
					"x": "int",
					"y": "int"
				},
				"methods": {
					"moveX": {
						"steps": [
							{"set": "self.x", "expr": "self.x + 1"}
						]
					}
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
		t.Fatal("expected compile error for frozen struct mutation")
	}
	if !strings.Contains(err.Error(), "frozen") {
		t.Errorf("expected 'frozen' error, got: %v", err)
	}
}

func TestMethod_FrozenStruct_ReadOnlyAllowed(t *testing.T) {
	result := compileAndRun(t, `{
		"structs": {
			"Point": {
				"frozen": true,
				"fields": {
					"x": "int",
					"y": "int"
				},
				"methods": {
					"sum": {
						"returns": "int",
						"steps": [
							{"return": "self.x + self.y"}
						]
					}
				}
			}
		},
		"steps": [
			{"let": "p", "new": "Point", "with": {"x": "3", "y": "4"}},
			{"return": "p.sum()"}
		]
	}`, nil)

	if !numEq(result.Value, 7) {
		t.Errorf("expected 7, got %v", result.Value)
	}
}

func TestMethod_SelfMethodCall(t *testing.T) {
	result := compileAndRun(t, `{
		"structs": {
			"Calc": {
				"fields": {
					"value": "int"
				},
				"methods": {
					"double": {
						"returns": "int",
						"steps": [
							{"return": "self.value * 2"}
						]
					},
					"quadruple": {
						"returns": "int",
						"steps": [
							{"let": "d", "expr": "self.double()"},
							{"return": "d * 2"}
						]
					}
				}
			}
		},
		"steps": [
			{"let": "c", "new": "Calc", "with": {"value": "5"}},
			{"return": "c.quadruple()"}
		]
	}`, nil)

	if !numEq(result.Value, 20) {
		t.Errorf("expected 20, got %v", result.Value)
	}
}

func TestMethod_UndefinedMethod_Error(t *testing.T) {
	program, err := Parse([]byte(`{
		"structs": {
			"Person": {
				"fields": {"name": "string"}
			}
		},
		"steps": [
			{"let": "p", "new": "Person", "with": {"name": "'Alice'"}},
			{"call": "p.nonexistent"}
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
		t.Fatal("expected error for undefined method")
	}
}
