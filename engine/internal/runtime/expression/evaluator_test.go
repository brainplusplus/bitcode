package expression

import (
	"math"
	"testing"
)

func TestBasicArithmetic(t *testing.T) {
	ctx := &EvalContext{Record: map[string]any{}}

	tests := []struct {
		expr     string
		expected float64
	}{
		{"2 + 3", 5},
		{"10 - 4", 6},
		{"3 * 7", 21},
		{"20 / 4", 5},
		{"10 % 3", 1},
		{"2 + 3 * 4", 14},
		{"(2 + 3) * 4", 20},
		{"-5 + 10", 5},
		{"10 / 0", 0},
	}

	for _, tt := range tests {
		val, err := EvaluateFloat(tt.expr, ctx)
		if err != nil {
			t.Errorf("EvaluateFloat(%q) error: %v", tt.expr, err)
			continue
		}
		if math.Abs(val-tt.expected) > 0.0001 {
			t.Errorf("EvaluateFloat(%q) = %v, want %v", tt.expr, val, tt.expected)
		}
	}
}

func TestFieldReferences(t *testing.T) {
	ctx := &EvalContext{
		Record: map[string]any{
			"quantity":   10.0,
			"unit_price": 25.5,
			"discount":   5.0,
		},
	}

	val, err := EvaluateFloat("quantity * unit_price", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if math.Abs(val-255.0) > 0.0001 {
		t.Errorf("got %v, want 255.0", val)
	}

	val, err = EvaluateFloat("quantity * unit_price - discount", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if math.Abs(val-250.0) > 0.0001 {
		t.Errorf("got %v, want 250.0", val)
	}
}

func TestComputedFieldFormula(t *testing.T) {
	ctx := &EvalContext{
		Record: map[string]any{
			"expected_revenue": 100000.0,
			"probability":      75.0,
		},
	}

	val, err := EvaluateFloat("expected_revenue * probability / 100", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if math.Abs(val-75000.0) > 0.0001 {
		t.Errorf("got %v, want 75000.0", val)
	}
}

func TestAggregateSum(t *testing.T) {
	ctx := &EvalContext{
		Record: map[string]any{},
		ChildCollections: map[string][]map[string]any{
			"lines": {
				{"subtotal": 100.0},
				{"subtotal": 200.0},
				{"subtotal": 50.0},
			},
		},
	}

	val, err := EvaluateFloat("sum(lines.subtotal)", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if math.Abs(val-350.0) > 0.0001 {
		t.Errorf("got %v, want 350.0", val)
	}
}

func TestAggregateCount(t *testing.T) {
	ctx := &EvalContext{
		Record: map[string]any{},
		ChildCollections: map[string][]map[string]any{
			"items": {
				{"qty": 1},
				{"qty": 2},
				{"qty": 3},
			},
		},
	}

	val, err := EvaluateFloat("count(items.qty)", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if math.Abs(val-3.0) > 0.0001 {
		t.Errorf("got %v, want 3.0", val)
	}
}

func TestAggregateAvg(t *testing.T) {
	ctx := &EvalContext{
		Record: map[string]any{},
		ChildCollections: map[string][]map[string]any{
			"scores": {
				{"value": 80.0},
				{"value": 90.0},
				{"value": 100.0},
			},
		},
	}

	val, err := EvaluateFloat("avg(scores.value)", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if math.Abs(val-90.0) > 0.0001 {
		t.Errorf("got %v, want 90.0", val)
	}
}

func TestAggregateMinMax(t *testing.T) {
	ctx := &EvalContext{
		Record: map[string]any{},
		ChildCollections: map[string][]map[string]any{
			"prices": {
				{"amount": 10.0},
				{"amount": 50.0},
				{"amount": 30.0},
			},
		},
	}

	val, err := EvaluateFloat("min(prices.amount)", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if math.Abs(val-10.0) > 0.0001 {
		t.Errorf("min got %v, want 10.0", val)
	}

	val, err = EvaluateFloat("max(prices.amount)", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if math.Abs(val-50.0) > 0.0001 {
		t.Errorf("max got %v, want 50.0", val)
	}
}

func TestEmptyCollection(t *testing.T) {
	ctx := &EvalContext{
		Record:           map[string]any{},
		ChildCollections: map[string][]map[string]any{},
	}

	val, err := EvaluateFloat("sum(lines.subtotal)", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != 0 {
		t.Errorf("got %v, want 0", val)
	}
}

func TestComparisons(t *testing.T) {
	ctx := &EvalContext{
		Record: map[string]any{
			"a": 10.0,
			"b": 20.0,
		},
	}

	tests := []struct {
		expr     string
		expected bool
	}{
		{"a < b", true},
		{"a > b", false},
		{"a == b", false},
		{"a != b", true},
		{"a <= 10", true},
		{"b >= 20", true},
	}

	for _, tt := range tests {
		val, err := Evaluate(tt.expr, ctx)
		if err != nil {
			t.Errorf("Evaluate(%q) error: %v", tt.expr, err)
			continue
		}
		if val != tt.expected {
			t.Errorf("Evaluate(%q) = %v, want %v", tt.expr, val, tt.expected)
		}
	}
}

func TestBooleanLogic(t *testing.T) {
	ctx := &EvalContext{
		Record: map[string]any{
			"active": true,
			"paid":   false,
		},
	}

	val, err := Evaluate("active && paid", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != false {
		t.Errorf("got %v, want false", val)
	}

	val, err = Evaluate("active || paid", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != true {
		t.Errorf("got %v, want true", val)
	}
}

func TestBuiltinFunctions(t *testing.T) {
	ctx := &EvalContext{Record: map[string]any{}}

	val, err := EvaluateFloat("abs(-42)", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != 42 {
		t.Errorf("abs got %v, want 42", val)
	}

	val, err = EvaluateFloat("round(3.456, 2)", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if math.Abs(val-3.46) > 0.0001 {
		t.Errorf("round got %v, want 3.46", val)
	}

	val, err = EvaluateFloat("ceil(3.2)", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != 4 {
		t.Errorf("ceil got %v, want 4", val)
	}

	val, err = EvaluateFloat("floor(3.8)", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != 3 {
		t.Errorf("floor got %v, want 3", val)
	}
}

func TestIfFunction(t *testing.T) {
	ctx := &EvalContext{
		Record: map[string]any{
			"status": "active",
			"amount": 100.0,
		},
	}

	val, err := EvaluateFloat("if(amount > 50, amount * 2, amount)", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != 200 {
		t.Errorf("got %v, want 200", val)
	}
}

func TestNilField(t *testing.T) {
	ctx := &EvalContext{
		Record: map[string]any{
			"quantity": 5.0,
		},
	}

	val, err := EvaluateFloat("quantity * unit_price", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != 0 {
		t.Errorf("got %v, want 0 (nil field should be 0)", val)
	}
}

func TestStringConcat(t *testing.T) {
	ctx := &EvalContext{
		Record: map[string]any{
			"first": "John",
			"last":  "Doe",
		},
	}

	val, err := Evaluate("first + ' ' + last", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "John Doe" {
		t.Errorf("got %v, want 'John Doe'", val)
	}
}

func TestEmptyExpression(t *testing.T) {
	ctx := &EvalContext{Record: map[string]any{}}
	val, err := Evaluate("", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != nil {
		t.Errorf("got %v, want nil", val)
	}
}

func TestIntegerFieldValues(t *testing.T) {
	ctx := &EvalContext{
		Record: map[string]any{
			"qty":   5,
			"price": 10,
		},
	}

	val, err := EvaluateFloat("qty * price", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != 50 {
		t.Errorf("got %v, want 50", val)
	}
}

func TestStringFieldValues(t *testing.T) {
	ctx := &EvalContext{
		Record: map[string]any{
			"qty":   "5",
			"price": "10.5",
		},
	}

	val, err := EvaluateFloat("qty * price", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if math.Abs(val-52.5) > 0.0001 {
		t.Errorf("got %v, want 52.5", val)
	}
}
