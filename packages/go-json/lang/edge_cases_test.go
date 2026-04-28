package lang

import (
	"strings"
	"testing"
)

func TestEdge_EmptyProgram(t *testing.T) {
	result := compileAndRun(t, `{"steps": []}`, nil)
	if result.Value != nil {
		t.Errorf("expected nil, got %v", result.Value)
	}
}

func TestEdge_NoSteps(t *testing.T) {
	result := compileAndRun(t, `{"name": "empty"}`, nil)
	if result.Value != nil {
		t.Errorf("expected nil, got %v", result.Value)
	}
}

func TestEdge_NilInput(t *testing.T) {
	result := compileAndRun(t, `{
		"steps": [
			{"return": "input"}
		]
	}`, nil)

	m, ok := result.Value.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", result.Value)
	}
	if len(m) != 0 {
		t.Errorf("expected empty input map, got %v", m)
	}
}

func TestEdge_ReturnBool(t *testing.T) {
	result := compileAndRun(t, `{
		"steps": [{"return": true}]
	}`, nil)

	if result.Value != true {
		t.Errorf("expected true, got %v", result.Value)
	}
}

func TestEdge_ReturnString(t *testing.T) {
	result := compileAndRun(t, `{
		"steps": [{"return": "'hello'"}]
	}`, nil)

	if result.Value != "hello" {
		t.Errorf("expected 'hello', got %v", result.Value)
	}
}

func TestEdge_NestedIfForWhile(t *testing.T) {
	result := compileAndRun(t, `{
		"steps": [
			{"let": "total", "value": 0},
			{
				"for": "i",
				"range": [0, 5],
				"steps": [
					{
						"if": "i % 2 == 0",
						"then": [
							{"let": "j", "value": 0},
							{
								"while": "j < 3",
								"steps": [
									{"set": "total", "expr": "total + 1"},
									{"set": "j", "expr": "j + 1"}
								]
							}
						]
					}
				]
			},
			{"return": "total"}
		]
	}`, nil)

	// i=0,2,4 are even → 3 iterations × 3 while loops = 9
	if !numEq(result.Value, 9) {
		t.Errorf("expected 9, got %v (%T)", result.Value, result.Value)
	}
}

func TestEdge_SwitchDefault(t *testing.T) {
	result := compileAndRun(t, `{
		"steps": [
			{"let": "x", "value": "unknown"},
			{"let": "result", "value": "none"},
			{
				"switch": "x",
				"cases": {
					"a": [{"set": "result", "value": "found_a"}],
					"default": [{"set": "result", "value": "default_hit"}]
				}
			},
			{"return": "result"}
		]
	}`, nil)

	if result.Value != "default_hit" {
		t.Errorf("expected 'default_hit', got %v", result.Value)
	}
}

func TestEdge_LoopIterationLimit(t *testing.T) {
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
	limits.MaxLoopIterations = 50
	compiled, _ := Compile(program, engine, limits)

	vm := NewVM(compiled, engine)
	_, err := vm.Execute(nil)

	if err == nil {
		t.Fatal("expected loop limit error")
	}
	if !strings.Contains(err.Error(), "loop iteration limit") {
		t.Errorf("expected loop limit error, got: %v", err)
	}
}

func TestEdge_SetUndefinedVariable(t *testing.T) {
	program, _ := Parse([]byte(`{
		"steps": [
			{"set": "undefined_var", "value": 42}
		]
	}`))

	engine := NewExprLangEngine()
	compiled, _ := Compile(program, engine, DefaultLimits())

	vm := NewVM(compiled, engine)
	_, err := vm.Execute(nil)

	if err == nil {
		t.Fatal("expected error for setting undefined variable")
	}
	if !strings.Contains(err.Error(), "not defined") {
		t.Errorf("expected 'not defined' error, got: %v", err)
	}
}

func TestEdge_DuplicateLetDeclaration(t *testing.T) {
	program, _ := Parse([]byte(`{
		"steps": [
			{"let": "x", "value": 1},
			{"let": "x", "value": 2}
		]
	}`))

	engine := NewExprLangEngine()
	compiled, _ := Compile(program, engine, DefaultLimits())

	vm := NewVM(compiled, engine)
	_, err := vm.Execute(nil)

	if err == nil {
		t.Fatal("expected error for duplicate let")
	}
	if !strings.Contains(err.Error(), "already declared") {
		t.Errorf("expected 'already declared' error, got: %v", err)
	}
}

func TestEdge_FunctionNotFound(t *testing.T) {
	program, _ := Parse([]byte(`{
		"steps": [
			{"call": "nonexistent", "with": {}}
		]
	}`))

	engine := NewExprLangEngine()
	compiled, _ := Compile(program, engine, DefaultLimits())

	vm := NewVM(compiled, engine)
	_, err := vm.Execute(nil)

	if err == nil {
		t.Fatal("expected error for undefined function")
	}
	if !strings.Contains(err.Error(), "not defined") {
		t.Errorf("expected 'not defined' error, got: %v", err)
	}
}

func TestEdge_MalformedJSON(t *testing.T) {
	_, err := Parse([]byte(`{invalid json`))
	if err == nil {
		t.Fatal("expected parse error for malformed JSON")
	}
}

func TestEdge_UnknownStepType(t *testing.T) {
	_, err := Parse([]byte(`{
		"steps": [
			{"unknown_key": "value"}
		]
	}`))

	if err == nil {
		t.Fatal("expected error for unknown step type")
	}
	if !strings.Contains(err.Error(), "unknown step type") {
		t.Errorf("expected 'unknown step type' error, got: %v", err)
	}
}

func TestEdge_LetMissingValue(t *testing.T) {
	_, err := Parse([]byte(`{
		"steps": [
			{"let": "x"}
		]
	}`))

	if err == nil {
		t.Fatal("expected error for let without value/expr/with")
	}
}

func TestEdge_LetMultipleValues(t *testing.T) {
	_, err := Parse([]byte(`{
		"steps": [
			{"let": "x", "value": 1, "expr": "2"}
		]
	}`))

	if err == nil {
		t.Fatal("expected error for let with multiple value modes")
	}
}

func TestEdge_ArrayIndexSet(t *testing.T) {
	result := compileAndRun(t, `{
		"steps": [
			{"let": "items", "value": [10, 20, 30]},
			{"set": "items[1]", "value": 99},
			{"return": "items"}
		]
	}`, nil)

	arr, ok := result.Value.([]any)
	if !ok {
		t.Fatalf("expected array, got %T", result.Value)
	}
	if !numEq(arr[1], 99) {
		t.Errorf("expected items[1]=99, got %v", arr[1])
	}
}

func TestEdge_Levenshtein(t *testing.T) {
	tests := []struct {
		a, b     string
		expected int
	}{
		{"", "", 0},
		{"abc", "", 3},
		{"", "abc", 3},
		{"abc", "abc", 0},
		{"abc", "abd", 1},
		{"kitten", "sitting", 3},
	}

	for _, tt := range tests {
		got := levenshtein(tt.a, tt.b)
		if got != tt.expected {
			t.Errorf("levenshtein(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.expected)
		}
	}
}

func TestEdge_SuggestSimilar(t *testing.T) {
	candidates := []string{"username", "user_id", "email", "password"}

	suggestions := SuggestSimilar("user_name", candidates, 3, 3)
	if len(suggestions) == 0 {
		t.Fatal("expected suggestions, got none")
	}
	if suggestions[0] != "username" && suggestions[0] != "user_id" {
		t.Errorf("expected 'username' or 'user_id' as first suggestion, got %v", suggestions)
	}
}

func TestEdge_SuggestSimilar_Empty(t *testing.T) {
	suggestions := SuggestSimilar("", []string{"a", "b"}, 3, 3)
	if suggestions != nil {
		t.Errorf("expected nil for empty target, got %v", suggestions)
	}

	suggestions = SuggestSimilar("test", nil, 3, 3)
	if suggestions != nil {
		t.Errorf("expected nil for empty candidates, got %v", suggestions)
	}
}

func TestEdge_TypeInference(t *testing.T) {
	tests := []struct {
		value    any
		expected string
	}{
		{nil, "nil"},
		{true, "bool"},
		{42, "int"},
		{int64(42), "int"},
		{42.0, "int"},
		{3.14, "float"},
		{"hello", "string"},
		{[]any{1, 2}, "[]any"},
		{map[string]any{"k": "v"}, "map"},
	}

	for _, tt := range tests {
		got := InferType(tt.value)
		if got != tt.expected {
			t.Errorf("InferType(%v) = %q, want %q", tt.value, got, tt.expected)
		}
	}
}

func TestEdge_TypesCompatible(t *testing.T) {
	tests := []struct {
		existing, new string
		expected      bool
	}{
		{"any", "string", true},
		{"string", "string", true},
		{"int", "string", false},
		{"?string", "nil", true},
		{"string", "nil", false},
		{"", "string", true},
		{"int", "int", true},
	}

	for _, tt := range tests {
		got := TypesCompatible(tt.existing, tt.new)
		if got != tt.expected {
			t.Errorf("TypesCompatible(%q, %q) = %v, want %v", tt.existing, tt.new, got, tt.expected)
		}
	}
}

func TestEdge_GoJSONError_Format(t *testing.T) {
	err := RuntimeError("TEST", "test message", 5).
		InFunction("myFunc").
		InProgram("myProg").
		WithFix("try this").
		WithSuggestions("suggestion1", "suggestion2")

	errStr := err.Error()
	if !strings.Contains(errStr, "myProg") {
		t.Errorf("error should contain program name: %s", errStr)
	}
	if !strings.Contains(errStr, "myFunc") {
		t.Errorf("error should contain function name: %s", errStr)
	}
	if !strings.Contains(errStr, "test message") {
		t.Errorf("error should contain message: %s", errStr)
	}
	if !strings.Contains(errStr, "try this") {
		t.Errorf("error should contain fix: %s", errStr)
	}

	short := err.Short()
	if !strings.Contains(short, "TEST") {
		t.Errorf("short should contain code: %s", short)
	}

	jsonMap := err.JSON()
	if jsonMap["code"] != "TEST" {
		t.Errorf("JSON code should be TEST, got %v", jsonMap["code"])
	}
}
