package hook

import (
	"context"
	"fmt"
	"testing"

	"github.com/bitcode-framework/bitcode/internal/compiler/parser"
)

type mockProcessLoader struct {
	processes map[string]*parser.ProcessDefinition
}

func (m *mockProcessLoader) LoadProcess(name string) (*parser.ProcessDefinition, error) {
	proc, ok := m.processes[name]
	if !ok {
		return nil, fmt.Errorf("process %q not found", name)
	}
	return proc, nil
}

func TestDispatcher_EmptyHandlers(t *testing.T) {
	d := NewDispatcher(nil, nil)
	err := d.Dispatch(context.Background(), "before_create", nil, &EventContext{})
	if err != nil {
		t.Fatalf("expected no error for empty handlers, got: %v", err)
	}
}

func TestDispatcher_ConditionSkip(t *testing.T) {
	loader := &mockProcessLoader{processes: map[string]*parser.ProcessDefinition{
		"should_not_run": {Name: "should_not_run"},
	}}
	SetProcessExecutor(func(ctx context.Context, proc *parser.ProcessDefinition, input map[string]any, userID string) (any, error) {
		t.Fatal("handler should not have been called")
		return nil, nil
	})
	d := NewDispatcher(loader, nil)
	handlers := []parser.EventHandler{
		{Process: "should_not_run", Condition: "status == 'active'"},
	}
	eventCtx := &EventContext{
		Data: map[string]any{"status": "draft"},
	}
	err := d.Dispatch(context.Background(), "before_create", handlers, eventCtx)
	if err != nil {
		t.Fatalf("expected no error when condition not met, got: %v", err)
	}
}

func TestDispatcher_Priority(t *testing.T) {
	var order []int

	SetProcessExecutor(func(ctx context.Context, proc *parser.ProcessDefinition, input map[string]any, userID string) (any, error) {
		switch proc.Name {
		case "first":
			order = append(order, 1)
		case "second":
			order = append(order, 2)
		case "third":
			order = append(order, 3)
		}
		return nil, nil
	})

	loader := &mockProcessLoader{
		processes: map[string]*parser.ProcessDefinition{
			"first":  {Name: "first"},
			"second": {Name: "second"},
			"third":  {Name: "third"},
		},
	}
	d2 := NewDispatcher(loader, nil)

	handlers := []parser.EventHandler{
		{Process: "third", Priority: 30},
		{Process: "first", Priority: 10},
		{Process: "second", Priority: 20},
	}

	err := d2.Dispatch(context.Background(), "before_create", handlers, &EventContext{
		Data: map[string]any{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(order) != 3 || order[0] != 1 || order[1] != 2 || order[2] != 3 {
		t.Fatalf("expected order [1,2,3], got %v", order)
	}
}

func TestDispatcher_OnErrorLog(t *testing.T) {
	loader := &mockProcessLoader{
		processes: map[string]*parser.ProcessDefinition{
			"failing": {Name: "failing"},
		},
	}

	SetProcessExecutor(func(ctx context.Context, proc *parser.ProcessDefinition, input map[string]any, userID string) (any, error) {
		return nil, fmt.Errorf("intentional error")
	})

	d := NewDispatcher(loader, nil)
	onError := "log"
	handlers := []parser.EventHandler{
		{Process: "failing", OnError: onError},
	}

	err := d.Dispatch(context.Background(), "after_create", handlers, &EventContext{
		Data: map[string]any{},
	})
	if err != nil {
		t.Fatal("expected error to be logged, not propagated")
	}
}

func TestDispatcher_OnErrorFail(t *testing.T) {
	loader := &mockProcessLoader{
		processes: map[string]*parser.ProcessDefinition{
			"failing": {Name: "failing"},
		},
	}

	SetProcessExecutor(func(ctx context.Context, proc *parser.ProcessDefinition, input map[string]any, userID string) (any, error) {
		return nil, fmt.Errorf("intentional error")
	})

	d := NewDispatcher(loader, nil)
	handlers := []parser.EventHandler{
		{Process: "failing"},
	}

	err := d.Dispatch(context.Background(), "before_create", handlers, &EventContext{
		Data: map[string]any{},
	})
	if err == nil {
		t.Fatal("expected error to propagate for before_create")
	}
}

func TestDispatcher_DataModification(t *testing.T) {
	loader := &mockProcessLoader{
		processes: map[string]*parser.ProcessDefinition{
			"set_defaults": {Name: "set_defaults"},
		},
	}

	SetProcessExecutor(func(ctx context.Context, proc *parser.ProcessDefinition, input map[string]any, userID string) (any, error) {
		return map[string]any{"status": "processed", "score": 100}, nil
	})

	d := NewDispatcher(loader, nil)
	handlers := []parser.EventHandler{
		{Process: "set_defaults"},
	}

	eventCtx := &EventContext{
		Data: map[string]any{"name": "test", "status": "draft"},
	}

	err := d.Dispatch(context.Background(), "before_create", handlers, eventCtx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if eventCtx.Data["status"] != "processed" {
		t.Fatalf("expected status=processed, got %v", eventCtx.Data["status"])
	}
	if eventCtx.Data["score"] != 100 {
		t.Fatalf("expected score=100, got %v", eventCtx.Data["score"])
	}
	if eventCtx.Data["name"] != "test" {
		t.Fatal("existing data should be preserved")
	}
}

func TestEventContext_Clone(t *testing.T) {
	original := &EventContext{
		Model: "test",
		Data:  map[string]any{"name": "John"},
	}
	clone := original.Clone()

	clone.Data["name"] = "Jane"
	if original.Data["name"] != "John" {
		t.Fatal("clone should not affect original")
	}
}

func TestExprEvaluator_Basic(t *testing.T) {
	data := map[string]any{
		"status": "active",
		"count":  float64(10),
	}

	if !evaluateSimpleExpr("status == 'active'", data) {
		t.Fatal("expected true for status == 'active'")
	}
	if evaluateSimpleExpr("status == 'draft'", data) {
		t.Fatal("expected false for status == 'draft'")
	}
	if !evaluateSimpleExpr("count > 5", data) {
		t.Fatal("expected true for count > 5")
	}
	if !evaluateSimpleExpr("status != 'draft'", data) {
		t.Fatal("expected true for status != 'draft'")
	}
}

func TestExprEvaluator_OldData(t *testing.T) {
	data := map[string]any{
		"status": "active",
		"__old":  map[string]any{"status": "draft"},
	}

	if !evaluateSimpleExpr("old.status != status", data) {
		t.Fatal("expected true: old.status (draft) != status (active)")
	}
	if evaluateSimpleExpr("old.status == status", data) {
		t.Fatal("expected false: old.status (draft) == status (active)")
	}
}

func TestExprEvaluator_AndOr(t *testing.T) {
	data := map[string]any{
		"status": "active",
		"role":   "admin",
	}

	if !evaluateSimpleExpr("status == 'active' && role == 'admin'", data) {
		t.Fatal("expected true for AND")
	}
	if evaluateSimpleExpr("status == 'draft' && role == 'admin'", data) {
		t.Fatal("expected false for AND with one false")
	}
	if !evaluateSimpleExpr("status == 'draft' || role == 'admin'", data) {
		t.Fatal("expected true for OR with one true")
	}
}
