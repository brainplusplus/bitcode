package lang

import (
	"testing"
)

func TestIntegration45b_StructWithMethods_FullProgram(t *testing.T) {
	result := compileAndRun(t, `{
		"structs": {
			"BankAccount": {
				"fields": {
					"owner": "string",
					"balance": "float"
				},
				"methods": {
					"deposit": {
						"params": {"amount": "float"},
						"steps": [
							{"set": "self.balance", "expr": "self.balance + amount"}
						]
					},
					"withdraw": {
						"params": {"amount": "float"},
						"returns": "bool",
						"steps": [
							{"if": "amount > self.balance", "then": [
								{"return": false}
							]},
							{"set": "self.balance", "expr": "self.balance - amount"},
							{"return": true}
						]
					},
					"getBalance": {
						"returns": "float",
						"steps": [
							{"return": "self.balance"}
						]
					}
				}
			}
		},
		"steps": [
			{"let": "acct", "new": "BankAccount", "with": {"owner": "'Alice'", "balance": "100.0"}},
			{"call": "acct.deposit", "with": {"amount": "50.0"}},
			{"let": "ok", "call": "acct.withdraw", "with": {"amount": "30.0"}},
			{"return": {"with": {
				"balance": "acct.getBalance()",
				"withdraw_ok": "ok"
			}}}
		]
	}`, nil)

	m, ok := result.Value.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", result.Value)
	}
	if !numEq(m["balance"], 120.0) {
		t.Errorf("expected balance=120, got %v", m["balance"])
	}
	if m["withdraw_ok"] != true {
		t.Errorf("expected withdraw_ok=true, got %v", m["withdraw_ok"])
	}
}

func TestIntegration45b_StructWithFunctions(t *testing.T) {
	result := compileAndRun(t, `{
		"structs": {
			"Point": {
				"fields": {
					"x": "int",
					"y": "int"
				}
			}
		},
		"functions": {
			"sumCoords": {
				"params": {"p": "Point"},
				"returns": "int",
				"steps": [
					{"return": "p.x + p.y"}
				]
			}
		},
		"steps": [
			{"let": "p", "new": "Point", "with": {"x": "3", "y": "4"}},
			{"let": "s", "call": "sumCoords", "with": {"p": "p"}},
			{"return": "s"}
		]
	}`, nil)

	if !numEq(result.Value, 7) {
		t.Errorf("expected 7, got %v (%T)", result.Value, result.Value)
	}
}

func TestIntegration45b_ParallelWithStructs(t *testing.T) {
	result := compileAndRun(t, `{
		"structs": {
			"Config": {
				"fields": {
					"name": "string",
					"value": "int"
				}
			}
		},
		"steps": [
			{"let": "base", "value": 10},
			{
				"parallel": {
					"doubled": [{"return": "base * 2"}],
					"tripled": [{"return": "base * 3"}]
				},
				"into": "results"
			},
			{"return": {"with": {
				"doubled": "results.doubled",
				"tripled": "results.tripled",
				"sum": "results.doubled + results.tripled"
			}}}
		]
	}`, nil)

	m, ok := result.Value.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", result.Value)
	}
	if !numEq(m["doubled"], 20) {
		t.Errorf("expected doubled=20, got %v", m["doubled"])
	}
	if !numEq(m["tripled"], 30) {
		t.Errorf("expected tripled=30, got %v", m["tripled"])
	}
	if !numEq(m["sum"], 50) {
		t.Errorf("expected sum=50, got %v", m["sum"])
	}
}

func TestIntegration45b_StructNotFound_Error(t *testing.T) {
	program, err := Parse([]byte(`{
		"steps": [
			{"let": "p", "new": "NonExistent", "with": {}}
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
		t.Fatal("expected error for non-existent struct")
	}
}

func TestIntegration45b_ImportParser(t *testing.T) {
	program, err := Parse([]byte(`{
		"imports": {
			"models": "./models.json",
			"validators": "stdlib:validators",
			"db": "io:database"
		},
		"steps": []
	}`))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if len(program.Imports) != 3 {
		t.Fatalf("expected 3 imports, got %d", len(program.Imports))
	}

	found := map[string]bool{}
	for _, imp := range program.Imports {
		found[imp.Alias] = true
		switch imp.Alias {
		case "models":
			if imp.PathType != "relative" {
				t.Errorf("expected models pathType=relative, got %s", imp.PathType)
			}
		case "validators":
			if imp.PathType != "stdlib" {
				t.Errorf("expected validators pathType=stdlib, got %s", imp.PathType)
			}
		case "db":
			if imp.PathType != "io" {
				t.Errorf("expected db pathType=io, got %s", imp.PathType)
			}
		}
	}

	if !found["models"] || !found["validators"] || !found["db"] {
		t.Errorf("missing expected imports: %v", found)
	}
}

func TestIntegration45b_ParallelParser(t *testing.T) {
	program, err := Parse([]byte(`{
		"steps": [
			{
				"parallel": {
					"a": [{"return": 1}],
					"b": [{"return": 2}]
				},
				"join": "all",
				"on_error": "continue",
				"into": "results"
			}
		]
	}`))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if len(program.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(program.Steps))
	}

	pn, ok := program.Steps[0].(*ParallelNode)
	if !ok {
		t.Fatalf("expected ParallelNode, got %T", program.Steps[0])
	}
	if len(pn.Branches) != 2 {
		t.Errorf("expected 2 branches, got %d", len(pn.Branches))
	}
	if pn.Join != "all" {
		t.Errorf("expected join=all, got %s", pn.Join)
	}
	if pn.OnError != "continue" {
		t.Errorf("expected on_error=continue, got %s", pn.OnError)
	}
	if pn.Into != "results" {
		t.Errorf("expected into=results, got %s", pn.Into)
	}
}
