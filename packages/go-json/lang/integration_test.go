package lang

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"
)

func compileAndRun(t *testing.T, jsonProgram string, input map[string]any) *ExecutionResult {
	t.Helper()
	program, err := Parse([]byte(jsonProgram))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	engine := NewExprLangEngine()
	compiled, err := Compile(program, engine, DefaultLimits())
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	vm := NewVM(compiled, engine)
	result, err := vm.Execute(input)
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}
	return result
}

// numEq compares numeric values regardless of int/float64 type.
func numEq(a, b any) bool {
	af, aOk := toNum(a)
	bf, bOk := toNum(b)
	if aOk && bOk {
		return af == bf
	}
	return a == b
}

func toNum(v any) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case float64:
		return n, true
	}
	return 0, false
}

func TestIntegration_HelloWorld(t *testing.T) {
	result := compileAndRun(t, `{
		"name": "hello",
		"steps": [
			{"let": "greeting", "value": "Hello, World!"},
			{"return": "greeting"}
		]
	}`, nil)

	if result.Value != "Hello, World!" {
		t.Errorf("expected 'Hello, World!', got %v", result.Value)
	}
}

func TestIntegration_Variables_ValueMode(t *testing.T) {
	result := compileAndRun(t, `{
		"steps": [
			{"let": "x", "value": 42},
			{"let": "s", "value": "hello"},
			{"let": "b", "value": true},
			{"let": "arr", "value": [1, 2, 3]},
			{"return": "x"}
		]
	}`, nil)

	if !numEq(result.Value, 42) {
		t.Errorf("expected 42, got %v (%T)", result.Value, result.Value)
	}
}

func TestIntegration_Variables_ExprMode(t *testing.T) {
	result := compileAndRun(t, `{
		"steps": [
			{"let": "x", "value": 10},
			{"let": "y", "expr": "x + 5"},
			{"return": "y"}
		]
	}`, nil)

	if !numEq(result.Value, 15) {
		t.Errorf("expected 15, got %v (%T)", result.Value, result.Value)
	}
}

func TestIntegration_Variables_WithMode(t *testing.T) {
	result := compileAndRun(t, `{
		"steps": [
			{"let": "name", "value": "Alice"},
			{"let": "age", "value": 30},
			{"let": "profile", "with": {
				"name": "name",
				"age": "age",
				"adult": "age >= 18"
			}},
			{"return": "profile"}
		]
	}`, nil)

	profile, ok := result.Value.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", result.Value)
	}
	if profile["name"] != "Alice" {
		t.Errorf("expected name=Alice, got %v", profile["name"])
	}
	if profile["adult"] != true {
		t.Errorf("expected adult=true, got %v", profile["adult"])
	}
}

func TestIntegration_Set(t *testing.T) {
	result := compileAndRun(t, `{
		"steps": [
			{"let": "x", "value": 10},
			{"set": "x", "value": 20},
			{"return": "x"}
		]
	}`, nil)

	if !numEq(result.Value, 20) {
		t.Errorf("expected 20, got %v (%T)", result.Value, result.Value)
	}
}

func TestIntegration_IfElse(t *testing.T) {
	tests := []struct {
		score    int
		expected string
	}{
		{95, "A"},
		{85, "B"},
		{75, "C"},
		{50, "F"},
	}

	for _, tt := range tests {
		result := compileAndRun(t, `{
			"steps": [
				{"let": "grade", "value": "F"},
				{
					"if": "input.score >= 90",
					"then": [{"set": "grade", "value": "A"}],
					"elif": [
						{"condition": "input.score >= 80", "then": [{"set": "grade", "value": "B"}]},
						{"condition": "input.score >= 70", "then": [{"set": "grade", "value": "C"}]}
					],
					"else": [{"set": "grade", "value": "F"}]
				},
				{"return": "grade"}
			]
		}`, map[string]any{"score": tt.score})

		if result.Value != tt.expected {
			t.Errorf("score=%d: expected %s, got %v", tt.score, tt.expected, result.Value)
		}
	}
}

func TestIntegration_Switch(t *testing.T) {
	result := compileAndRun(t, `{
		"steps": [
			{"let": "status", "value": "active"},
			{"let": "label", "value": "unknown"},
			{
				"switch": "status",
				"cases": {
					"active": [{"set": "label", "value": "Active User"}],
					"pending": [{"set": "label", "value": "Pending User"}],
					"default": [{"set": "label", "value": "Other"}]
				}
			},
			{"return": "label"}
		]
	}`, nil)

	if result.Value != "Active User" {
		t.Errorf("expected 'Active User', got %v", result.Value)
	}
}

func TestIntegration_ForEach(t *testing.T) {
	result := compileAndRun(t, `{
		"steps": [
			{"let": "items", "value": [1, 2, 3, 4, 5]},
			{"let": "total", "value": 0},
			{
				"for": "item",
				"in": "items",
				"steps": [
					{"set": "total", "expr": "total + item"}
				]
			},
			{"return": "total"}
		]
	}`, nil)

	if !numEq(result.Value, 15) {
		t.Errorf("expected 15, got %v (%T)", result.Value, result.Value)
	}
}

func TestIntegration_ForRange(t *testing.T) {
	result := compileAndRun(t, `{
		"steps": [
			{"let": "total", "value": 0},
			{
				"for": "i",
				"range": [1, 6],
				"steps": [
					{"set": "total", "expr": "total + i"}
				]
			},
			{"return": "total"}
		]
	}`, nil)

	if !numEq(result.Value, 15) {
		t.Errorf("expected 15, got %v (%T)", result.Value, result.Value)
	}
}

func TestIntegration_ForWithIndex(t *testing.T) {
	result := compileAndRun(t, `{
		"steps": [
			{"let": "items", "value": ["a", "b", "c"]},
			{"let": "last_index", "value": -1},
			{
				"for": "item",
				"in": "items",
				"index": "i",
				"steps": [
					{"set": "last_index", "expr": "i"}
				]
			},
			{"return": "last_index"}
		]
	}`, nil)

	if !numEq(result.Value, 2) {
		t.Errorf("expected 2, got %v (%T)", result.Value, result.Value)
	}
}

func TestIntegration_While(t *testing.T) {
	result := compileAndRun(t, `{
		"steps": [
			{"let": "count", "value": 1},
			{
				"while": "count < 100",
				"steps": [
					{"set": "count", "expr": "count * 2"}
				]
			},
			{"return": "count"}
		]
	}`, nil)

	if !numEq(result.Value, 128) {
		t.Errorf("expected 128, got %v (%T)", result.Value, result.Value)
	}
}

func TestIntegration_Break(t *testing.T) {
	result := compileAndRun(t, `{
		"steps": [
			{"let": "items", "value": [1, 2, 3, 4, 5]},
			{"let": "total", "value": 0},
			{
				"for": "item",
				"in": "items",
				"steps": [
					{"if": "item > 3", "then": [{"break": true}]},
					{"set": "total", "expr": "total + item"}
				]
			},
			{"return": "total"}
		]
	}`, nil)

	if !numEq(result.Value, 6) {
		t.Errorf("expected 6 (1+2+3), got %v (%T)", result.Value, result.Value)
	}
}

func TestIntegration_Continue(t *testing.T) {
	result := compileAndRun(t, `{
		"steps": [
			{"let": "items", "value": [1, 2, 3, 4, 5]},
			{"let": "total", "value": 0},
			{
				"for": "item",
				"in": "items",
				"steps": [
					{"if": "item == 3", "then": [{"continue": true}]},
					{"set": "total", "expr": "total + item"}
				]
			},
			{"return": "total"}
		]
	}`, nil)

	// 1+2+4+5 = 12 (skipped 3)
	if !numEq(result.Value, 12) {
		t.Errorf("expected 12, got %v (%T)", result.Value, result.Value)
	}
}

func TestIntegration_FunctionCall(t *testing.T) {
	result := compileAndRun(t, `{
		"functions": {
			"add": {
				"params": {"a": "int", "b": "int"},
				"returns": "int",
				"steps": [{"return": "a + b"}]
			}
		},
		"steps": [
			{"let": "result", "call": "add", "with": {"a": "3", "b": "4"}},
			{"return": "result"}
		]
	}`, nil)

	if !numEq(result.Value, 7) {
		t.Errorf("expected 7, got %v (%T)", result.Value, result.Value)
	}
}

func TestIntegration_Recursion_Factorial(t *testing.T) {
	result := compileAndRun(t, `{
		"functions": {
			"factorial": {
				"params": {"n": "int"},
				"returns": "int",
				"steps": [
					{"if": "n <= 1", "then": [{"return": 1}]},
					{"let": "sub", "call": "factorial", "with": {"n": "n - 1"}},
					{"return": "n * sub"}
				]
			}
		},
		"steps": [
			{"let": "result", "call": "factorial", "with": {"n": "5"}},
			{"return": "result"}
		]
	}`, nil)

	if !numEq(result.Value, 120) {
		t.Errorf("expected 120, got %v (%T)", result.Value, result.Value)
	}
}

func TestIntegration_TryCatch(t *testing.T) {
	result := compileAndRun(t, `{
		"steps": [
			{"let": "caught", "value": "none"},
			{
				"try": [
					{"error": "'test error'"}
				],
				"catch": {
					"as": "err",
					"steps": [
						{"set": "caught", "expr": "err.message"}
					]
				}
			},
			{"return": "caught"}
		]
	}`, nil)

	if result.Value != "test error" {
		t.Errorf("expected 'test error', got %v", result.Value)
	}
}

func TestIntegration_TryCatchFinally(t *testing.T) {
	result := compileAndRun(t, `{
		"steps": [
			{"let": "result", "value": "initial"},
			{
				"try": [
					{"set": "result", "value": "try"}
				],
				"finally": [
					{"set": "result", "expr": "result + '_finally'"}
				]
			},
			{"return": "result"}
		]
	}`, nil)

	if result.Value != "try_finally" {
		t.Errorf("expected 'try_finally', got %v", result.Value)
	}
}

func TestIntegration_StructuredError(t *testing.T) {
	result := compileAndRun(t, `{
		"steps": [
			{"let": "caught_code", "value": "none"},
			{
				"try": [
					{"error": {"code": "'VALIDATION'", "message": "'bad input'"}}
				],
				"catch": {
					"as": "err",
					"steps": [
						{"set": "caught_code", "expr": "err.code"}
					]
				}
			},
			{"return": "caught_code"}
		]
	}`, nil)

	if result.Value != "VALIDATION" {
		t.Errorf("expected 'VALIDATION', got %v", result.Value)
	}
}

func TestIntegration_ReturnLiteral(t *testing.T) {
	result := compileAndRun(t, `{
		"steps": [
			{"return": 42}
		]
	}`, nil)

	// JSON numbers are float64.
	if !numEq(result.Value, 42) {
		t.Errorf("expected 42, got %v (%T)", result.Value, result.Value)
	}
}

func TestIntegration_ReturnNull(t *testing.T) {
	result := compileAndRun(t, `{
		"steps": [
			{"return": null}
		]
	}`, nil)

	if result.Value != nil {
		t.Errorf("expected nil, got %v", result.Value)
	}
}

func TestIntegration_ReturnWithMode(t *testing.T) {
	result := compileAndRun(t, `{
		"steps": [
			{"let": "x", "value": 42},
			{"return": {"with": {"answer": "x", "doubled": "x * 2"}}}
		]
	}`, nil)

	m, ok := result.Value.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", result.Value)
	}
	if !numEq(m["answer"], 42) {
		t.Errorf("expected answer=42, got %v (%T)", m["answer"], m["answer"])
	}
	if !numEq(m["doubled"], 84) {
		t.Errorf("expected doubled=84, got %v (%T)", m["doubled"], m["doubled"])
	}
}

func TestIntegration_EmptySteps(t *testing.T) {
	result := compileAndRun(t, `{
		"steps": []
	}`, nil)

	if result.Value != nil {
		t.Errorf("expected nil for empty steps, got %v", result.Value)
	}
}

func TestIntegration_JSONC(t *testing.T) {
	result := compileAndRun(t, `{
		// This is a comment
		"name": "jsonc_test",
		"steps": [
			{"let": "x", "value": 42}, // inline comment
			{"return": "x"},
		]
	}`, nil)

	if !numEq(result.Value, 42) {
		t.Errorf("expected 42, got %v (%T)", result.Value, result.Value)
	}
}

func TestIntegration_CommentNode(t *testing.T) {
	result := compileAndRun(t, `{
		"steps": [
			{"_c": "This is a standalone comment"},
			{"let": "x", "value": 42},
			{"_c": ["Multi-line", "comment"]},
			{"return": "x"}
		]
	}`, nil)

	if !numEq(result.Value, 42) {
		t.Errorf("expected 42, got %v (%T)", result.Value, result.Value)
	}
}

func TestIntegration_NestedPropertySet(t *testing.T) {
	result := compileAndRun(t, `{
		"steps": [
			{"let": "person", "value": {"name": "Alice", "address": {"city": "Jakarta"}}},
			{"set": "person.address.city", "value": "Bandung"},
			{"return": "person"}
		]
	}`, nil)

	person, ok := result.Value.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", result.Value)
	}
	addr, ok := person["address"].(map[string]any)
	if !ok {
		t.Fatalf("expected address map, got %T", person["address"])
	}
	if addr["city"] != "Bandung" {
		t.Errorf("expected city=Bandung, got %v", addr["city"])
	}
}

func TestIntegration_FunctionScopeIsolation(t *testing.T) {
	result := compileAndRun(t, `{
		"functions": {
			"getX": {
				"params": {},
				"returns": "any",
				"steps": [
					{"return": "nil"}
				]
			}
		},
		"steps": [
			{"let": "x", "value": 42},
			{"let": "result", "call": "getX", "with": {}},
			{"return": "x"}
		]
	}`, nil)

	if !numEq(result.Value, 42) {
		t.Errorf("expected 42 (x unchanged), got %v (%T)", result.Value, result.Value)
	}
}

func TestIntegration_BlockScope(t *testing.T) {
	// Variable declared in if-then should not be visible outside.
	prog := `{
		"steps": [
			{"let": "x", "value": 10},
			{
				"if": "true",
				"then": [
					{"let": "y", "value": 20}
				]
			},
			{"return": "x"}
		]
	}`

	result := compileAndRun(t, prog, nil)
	if !numEq(result.Value, 10) {
		t.Errorf("expected 10, got %v (%T)", result.Value, result.Value)
	}
}

func TestIntegration_StepLimit(t *testing.T) {
	program, _ := Parse([]byte(`{
		"steps": [
			{"let": "i", "value": 0},
			{
				"while": "true",
				"steps": [
					{"set": "i", "expr": "i + 1"}
				]
			}
		]
	}`))

	engine := NewExprLangEngine()
	limits := DefaultLimits()
	limits.MaxSteps = 100
	compiled, _ := Compile(program, engine, limits)

	vm := NewVM(compiled, engine)
	_, err := vm.Execute(nil)

	if err == nil {
		t.Fatal("expected step limit error")
	}
	if !strings.Contains(err.Error(), "step limit") {
		t.Errorf("expected step limit error, got: %v", err)
	}
}

func TestIntegration_DepthLimit(t *testing.T) {
	program, _ := Parse([]byte(`{
		"functions": {
			"infinite": {
				"params": {},
				"steps": [
					{"call": "infinite", "with": {}}
				]
			}
		},
		"steps": [
			{"call": "infinite", "with": {}}
		]
	}`))

	engine := NewExprLangEngine()
	limits := DefaultLimits()
	limits.MaxDepth = 10
	compiled, _ := Compile(program, engine, limits)

	vm := NewVM(compiled, engine)
	_, err := vm.Execute(nil)

	if err == nil {
		t.Fatal("expected depth limit error")
	}
	if !strings.Contains(err.Error(), "depth limit") {
		t.Errorf("expected depth limit error, got: %v", err)
	}
}

func TestIntegration_Timeout(t *testing.T) {
	program, _ := Parse([]byte(`{
		"steps": [
			{"let": "i", "value": 0},
			{
				"while": "true",
				"steps": [
					{"set": "i", "expr": "i + 1"}
				]
			}
		]
	}`))

	engine := NewExprLangEngine()
	limits := DefaultLimits()
	limits.MaxSteps = 10000000
	limits.MaxLoopIterations = 10000000
	limits.Timeout = 100 * time.Millisecond
	compiled, _ := Compile(program, engine, limits)

	vm := NewVM(compiled, engine, WithContext(context.Background()))
	_, err := vm.Execute(nil)

	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "timeout") && !strings.Contains(err.Error(), "TIMEOUT") {
		t.Errorf("expected timeout error, got: %v", err)
	}
}

func TestIntegration_ConcurrentExecution(t *testing.T) {
	program, _ := Parse([]byte(`{
		"steps": [
			{"let": "x", "expr": "input.value * 2"},
			{"return": "x"}
		]
	}`))

	engine := NewExprLangEngine()
	compiled, _ := Compile(program, engine, DefaultLimits())

	var wg sync.WaitGroup
	results := make([]any, 10)
	errors := make([]error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			vm := NewVM(compiled, engine)
			result, err := vm.Execute(map[string]any{
				"value": idx,
			})
			if err != nil {
				errors[idx] = err
			} else {
				results[idx] = result.Value
			}
		}(i)
	}

	wg.Wait()

	for i := 0; i < 10; i++ {
		if errors[i] != nil {
			t.Errorf("goroutine %d error: %v", i, errors[i])
			continue
		}
		expected := i * 2
		if !numEq(results[i], expected) {
			t.Errorf("goroutine %d: expected %d, got %v (%T)", i, expected, results[i], results[i])
		}
	}
}

func TestIntegration_BreakOutsideLoop_CompileError(t *testing.T) {
	program, _ := Parse([]byte(`{
		"steps": [
			{"break": true}
		]
	}`))

	engine := NewExprLangEngine()
	_, err := Compile(program, engine, DefaultLimits())

	if err == nil {
		t.Fatal("expected compile error for break outside loop")
	}
	if !strings.Contains(err.Error(), "break") {
		t.Errorf("expected break error, got: %v", err)
	}
}

func TestIntegration_ContinueOutsideLoop_CompileError(t *testing.T) {
	program, _ := Parse([]byte(`{
		"steps": [
			{"continue": true}
		]
	}`))

	engine := NewExprLangEngine()
	_, err := Compile(program, engine, DefaultLimits())

	if err == nil {
		t.Fatal("expected compile error for continue outside loop")
	}
	if !strings.Contains(err.Error(), "continue") {
		t.Errorf("expected continue error, got: %v", err)
	}
}

func TestIntegration_ReturnInsideLoop(t *testing.T) {
	result := compileAndRun(t, `{
		"functions": {
			"findFirst": {
				"params": {"items": "[]any", "target": "any"},
				"returns": "int",
				"steps": [
					{
						"for": "item",
						"in": "items",
						"index": "i",
						"steps": [
							{"if": "item == target", "then": [{"return": "i"}]}
						]
					},
					{"return": -1}
				]
			}
		},
		"steps": [
			{"let": "idx", "call": "findFirst", "with": {
				"items": "[10, 20, 30, 40]",
				"target": "30"
			}},
			{"return": "idx"}
		]
	}`, nil)

	if !numEq(result.Value, 2) {
		t.Errorf("expected 2, got %v (%T)", result.Value, result.Value)
	}
}

func TestIntegration_WithTrace(t *testing.T) {
	program, _ := Parse([]byte(`{
		"steps": [
			{"let": "x", "value": 42},
			{"return": "x"}
		]
	}`))

	engine := NewExprLangEngine()
	compiled, _ := Compile(program, engine, DefaultLimits())

	vm := NewVM(compiled, engine, WithTrace(true))
	result, err := vm.Execute(nil)
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}

	if len(result.Trace) == 0 {
		t.Error("expected trace entries, got none")
	}
	if result.Steps < 2 {
		t.Errorf("expected at least 2 steps, got %d", result.Steps)
	}
}
