package executor

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/bitcode-framework/bitcode/internal/compiler/parser"
)

type trackingHandler struct {
	mu       sync.Mutex
	executed []string
}

func (h *trackingHandler) Execute(ctx context.Context, execCtx *Context, step parser.StepDefinition) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.executed = append(h.executed, step.Name)
	return nil
}

type failingHandler struct {
	failNode string
}

func (h *failingHandler) Execute(ctx context.Context, execCtx *Context, step parser.StepDefinition) error {
	if step.Name == h.failNode {
		return fmt.Errorf("node %s failed", step.Name)
	}
	return nil
}

func TestDAG_LinearChain(t *testing.T) {
	exec := NewExecutor()
	handler := &trackingHandler{}
	exec.RegisterHandler(parser.StepAssign, handler)

	proc := &parser.ProcessDefinition{
		Name: "linear",
		Nodes: map[string]parser.StepDefinition{
			"a": {Name: "a", Type: parser.StepAssign, Variable: "x", Value: 1},
			"b": {Name: "b", Type: parser.StepAssign, Variable: "y", Value: 2},
			"c": {Name: "c", Type: parser.StepAssign, Variable: "z", Value: 3},
		},
		Edges: []parser.EdgeDefinition{
			{From: "a", To: "b"},
			{From: "b", To: "c"},
		},
	}

	result, err := exec.Execute(context.Background(), proc, map[string]any{}, "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(handler.executed) != 3 {
		t.Fatalf("expected 3 nodes executed, got %d", len(handler.executed))
	}
	if handler.executed[0] != "a" {
		t.Errorf("expected first node 'a', got '%s'", handler.executed[0])
	}
	if handler.executed[2] != "c" {
		t.Errorf("expected last node 'c', got '%s'", handler.executed[2])
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestDAG_ParallelFanOut(t *testing.T) {
	exec := NewExecutor()
	handler := &trackingHandler{}
	exec.RegisterHandler(parser.StepAssign, handler)

	proc := &parser.ProcessDefinition{
		Name: "fanout",
		Nodes: map[string]parser.StepDefinition{
			"start":    {Name: "start", Type: parser.StepAssign, Variable: "s", Value: 1},
			"branch_a": {Name: "branch_a", Type: parser.StepAssign, Variable: "a", Value: 2},
			"branch_b": {Name: "branch_b", Type: parser.StepAssign, Variable: "b", Value: 3},
		},
		Edges: []parser.EdgeDefinition{
			{From: "start", To: "branch_a"},
			{From: "start", To: "branch_b"},
		},
	}

	_, err := exec.Execute(context.Background(), proc, map[string]any{}, "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(handler.executed) != 3 {
		t.Fatalf("expected 3 nodes executed, got %d", len(handler.executed))
	}
	if handler.executed[0] != "start" {
		t.Errorf("expected 'start' first, got '%s'", handler.executed[0])
	}
}

func TestDAG_FanInMerge(t *testing.T) {
	exec := NewExecutor()
	handler := &trackingHandler{}
	exec.RegisterHandler(parser.StepAssign, handler)

	proc := &parser.ProcessDefinition{
		Name: "fanin",
		Nodes: map[string]parser.StepDefinition{
			"a":     {Name: "a", Type: parser.StepAssign, Variable: "a", Value: 1},
			"b":     {Name: "b", Type: parser.StepAssign, Variable: "b", Value: 2},
			"merge": {Name: "merge", Type: parser.StepAssign, Variable: "m", Value: 3},
		},
		Edges: []parser.EdgeDefinition{
			{From: "a", To: "merge"},
			{From: "b", To: "merge"},
		},
	}

	_, err := exec.Execute(context.Background(), proc, map[string]any{}, "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(handler.executed) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(handler.executed))
	}
	if handler.executed[2] != "merge" {
		t.Errorf("expected 'merge' last, got '%s'", handler.executed[2])
	}
}

func TestDAG_ConditionalEdge(t *testing.T) {
	exec := NewExecutor()
	handler := &trackingHandler{}
	exec.RegisterHandler(parser.StepAssign, handler)

	proc := &parser.ProcessDefinition{
		Name: "conditional",
		Nodes: map[string]parser.StepDefinition{
			"start": {Name: "start", Type: parser.StepAssign, Variable: "s", Value: 1},
			"vip":   {Name: "vip", Type: parser.StepAssign, Variable: "v", Value: 2},
			"basic": {Name: "basic", Type: parser.StepAssign, Variable: "b", Value: 3},
		},
		Edges: []parser.EdgeDefinition{
			{From: "start", To: "vip", Condition: "{{input.is_vip}}"},
			{From: "start", To: "basic"},
		},
	}

	_, err := exec.Execute(context.Background(), proc, map[string]any{"is_vip": false}, "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	executed := make(map[string]bool)
	for _, name := range handler.executed {
		executed[name] = true
	}
	if !executed["start"] {
		t.Error("expected 'start' to execute")
	}
	if !executed["basic"] {
		t.Error("expected 'basic' to execute")
	}
}

func TestDAG_CycleDetection(t *testing.T) {
	exec := NewExecutor()
	handler := &trackingHandler{}
	exec.RegisterHandler(parser.StepAssign, handler)

	proc := &parser.ProcessDefinition{
		Name: "cycle",
		Nodes: map[string]parser.StepDefinition{
			"a": {Name: "a", Type: parser.StepAssign, Variable: "a", Value: 1},
			"b": {Name: "b", Type: parser.StepAssign, Variable: "b", Value: 2},
		},
		Edges: []parser.EdgeDefinition{
			{From: "a", To: "b"},
			{From: "b", To: "a"},
		},
	}

	_, err := exec.Execute(context.Background(), proc, map[string]any{}, "user-1")
	if err == nil {
		t.Fatal("expected cycle detection error")
	}
}

func TestDAG_NodeFailure(t *testing.T) {
	exec := NewExecutor()
	handler := &failingHandler{failNode: "bad"}
	exec.RegisterHandler(parser.StepAssign, handler)

	proc := &parser.ProcessDefinition{
		Name: "fail",
		Nodes: map[string]parser.StepDefinition{
			"good": {Name: "good", Type: parser.StepAssign, Variable: "g", Value: 1},
			"bad":  {Name: "bad", Type: parser.StepAssign, Variable: "b", Value: 2},
		},
		Edges: []parser.EdgeDefinition{
			{From: "good", To: "bad"},
		},
	}

	_, err := exec.Execute(context.Background(), proc, map[string]any{}, "user-1")
	if err == nil {
		t.Fatal("expected error from failing node")
	}
}

func TestDAG_SingleNode(t *testing.T) {
	exec := NewExecutor()
	handler := &trackingHandler{}
	exec.RegisterHandler(parser.StepAssign, handler)

	proc := &parser.ProcessDefinition{
		Name: "single",
		Nodes: map[string]parser.StepDefinition{
			"only": {Name: "only", Type: parser.StepAssign, Variable: "x", Value: 1},
		},
		Edges: []parser.EdgeDefinition{},
	}

	_, err := exec.Execute(context.Background(), proc, map[string]any{}, "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(handler.executed) != 1 {
		t.Fatalf("expected 1 node, got %d", len(handler.executed))
	}
}

func TestDAG_DiamondPattern(t *testing.T) {
	exec := NewExecutor()
	handler := &trackingHandler{}
	exec.RegisterHandler(parser.StepAssign, handler)

	proc := &parser.ProcessDefinition{
		Name: "diamond",
		Nodes: map[string]parser.StepDefinition{
			"start":  {Name: "start", Type: parser.StepAssign, Variable: "s", Value: 1},
			"left":   {Name: "left", Type: parser.StepAssign, Variable: "l", Value: 2},
			"right":  {Name: "right", Type: parser.StepAssign, Variable: "r", Value: 3},
			"finish": {Name: "finish", Type: parser.StepAssign, Variable: "f", Value: 4},
		},
		Edges: []parser.EdgeDefinition{
			{From: "start", To: "left"},
			{From: "start", To: "right"},
			{From: "left", To: "finish"},
			{From: "right", To: "finish"},
		},
	}

	_, err := exec.Execute(context.Background(), proc, map[string]any{}, "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(handler.executed) != 4 {
		t.Fatalf("expected 4 nodes, got %d", len(handler.executed))
	}
	if handler.executed[0] != "start" {
		t.Errorf("expected 'start' first, got '%s'", handler.executed[0])
	}
	if handler.executed[3] != "finish" {
		t.Errorf("expected 'finish' last, got '%s'", handler.executed[3])
	}
}

func TestParseProcess_DAGFormat(t *testing.T) {
	data := []byte(`{
		"name": "dag_test",
		"nodes": {
			"validate": { "type": "validate", "rules": { "status": { "eq": "draft" } }, "error": "Must be draft" },
			"update": { "type": "update", "model": "order", "set": { "status": "confirmed" } },
			"notify": { "type": "emit", "event": "order.confirmed" }
		},
		"edges": [
			{ "from": "validate", "to": "update" },
			{ "from": "update", "to": "notify" }
		]
	}`)

	proc, err := parser.ParseProcess(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !proc.IsDAG() {
		t.Error("expected DAG process")
	}
	if len(proc.Nodes) != 3 {
		t.Errorf("expected 3 nodes, got %d", len(proc.Nodes))
	}
	if len(proc.Edges) != 2 {
		t.Errorf("expected 2 edges, got %d", len(proc.Edges))
	}
}

func TestParseProcess_DAGInvalidEdge(t *testing.T) {
	data := []byte(`{
		"name": "bad_dag",
		"nodes": {
			"a": { "type": "assign", "variable": "x", "value": 1 }
		},
		"edges": [
			{ "from": "a", "to": "nonexistent" }
		]
	}`)

	_, err := parser.ParseProcess(data)
	if err == nil {
		t.Fatal("expected error for invalid edge reference")
	}
}

func TestParseProcess_SequentialStillWorks(t *testing.T) {
	data := []byte(`{
		"name": "sequential",
		"steps": [
			{ "type": "assign", "variable": "x", "value": 1 }
		]
	}`)

	proc, err := parser.ParseProcess(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if proc.IsDAG() {
		t.Error("expected sequential process, not DAG")
	}
	if len(proc.Steps) != 1 {
		t.Errorf("expected 1 step, got %d", len(proc.Steps))
	}
}
