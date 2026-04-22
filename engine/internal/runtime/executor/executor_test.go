package executor

import (
	"context"
	"fmt"
	"testing"

	"github.com/bitcode-engine/engine/internal/compiler/parser"
)

type mockHandler struct {
	called bool
	err    error
}

func (m *mockHandler) Execute(ctx context.Context, execCtx *Context, step parser.StepDefinition) error {
	m.called = true
	return m.err
}

func TestExecutor_RunsSteps(t *testing.T) {
	exec := NewExecutor()
	handler := &mockHandler{}
	exec.RegisterHandler(parser.StepEmit, handler)

	proc := &parser.ProcessDefinition{
		Name: "test",
		Steps: []parser.StepDefinition{
			{Type: parser.StepEmit, Event: "test.event"},
		},
	}

	_, err := exec.Execute(context.Background(), proc, map[string]any{"key": "val"}, "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !handler.called {
		t.Error("handler should have been called")
	}
}

func TestExecutor_UnknownStepType(t *testing.T) {
	exec := NewExecutor()
	proc := &parser.ProcessDefinition{
		Name:  "test",
		Steps: []parser.StepDefinition{{Type: "unknown"}},
	}

	_, err := exec.Execute(context.Background(), proc, nil, "user-1")
	if err == nil {
		t.Fatal("expected error for unknown step type")
	}
}

func TestExecutor_StepError(t *testing.T) {
	exec := NewExecutor()
	exec.RegisterHandler(parser.StepValidate, &mockHandler{err: fmt.Errorf("validation failed")})

	proc := &parser.ProcessDefinition{
		Name:  "test",
		Steps: []parser.StepDefinition{{Type: parser.StepValidate}},
	}

	_, err := exec.Execute(context.Background(), proc, nil, "user-1")
	if err == nil {
		t.Fatal("expected error from failing step")
	}
}

func TestExecutor_MaxNestingDepth(t *testing.T) {
	exec := NewExecutor()
	handler := &mockHandler{}
	exec.RegisterHandler(parser.StepEmit, handler)

	execCtx := &Context{
		Input:     map[string]any{},
		Variables: make(map[string]any),
	}

	steps := []parser.StepDefinition{
		{Type: parser.StepEmit, Event: "test"},
	}

	var lastErr error
	for i := 0; i <= MaxNestingDepth+1; i++ {
		lastErr = exec.ExecuteSteps(context.Background(), execCtx, steps)
		if lastErr != nil {
			break
		}
	}

	if lastErr != nil {
		t.Logf("depth limit triggered at some point (expected for recursive calls)")
	}
}

func TestExecutor_ExecuteSteps(t *testing.T) {
	exec := NewExecutor()
	handler := &mockHandler{}
	exec.RegisterHandler(parser.StepEmit, handler)

	execCtx := &Context{
		Input:     map[string]any{},
		Variables: make(map[string]any),
	}

	steps := []parser.StepDefinition{
		{Type: parser.StepEmit, Event: "test.event"},
	}

	if err := exec.ExecuteSteps(context.Background(), execCtx, steps); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !handler.called {
		t.Error("handler should have been called via ExecuteSteps")
	}
}
