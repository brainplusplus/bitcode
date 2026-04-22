package parser

import "testing"

func TestParseWorkflow(t *testing.T) {
	data := []byte(`{
		"name": "order_workflow",
		"model": "order",
		"field": "status",
		"states": {
			"draft": { "label": "Draft" },
			"confirmed": { "label": "Confirmed" },
			"done": { "label": "Done" },
			"cancelled": { "label": "Cancelled" }
		},
		"transitions": [
			{ "from": "draft", "to": "confirmed", "action": "confirm", "permission": "order.confirm" },
			{ "from": "confirmed", "to": "done", "action": "complete" },
			{ "from": ["draft", "confirmed"], "to": "cancelled", "action": "cancel" }
		]
	}`)

	wf, err := ParseWorkflow(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wf.Name != "order_workflow" {
		t.Errorf("expected order_workflow, got %s", wf.Name)
	}
	if len(wf.States) != 4 {
		t.Errorf("expected 4 states, got %d", len(wf.States))
	}
	if len(wf.Transitions) != 3 {
		t.Errorf("expected 3 transitions, got %d", len(wf.Transitions))
	}
}

func TestWorkflow_CanTransition(t *testing.T) {
	data := []byte(`{
		"name": "test_wf",
		"model": "order",
		"field": "status",
		"states": { "draft": {}, "confirmed": {}, "cancelled": {} },
		"transitions": [
			{ "from": "draft", "to": "confirmed", "action": "confirm" },
			{ "from": ["draft", "confirmed"], "to": "cancelled", "action": "cancel" }
		]
	}`)

	wf, _ := ParseWorkflow(data)

	newState, err := wf.CanTransition("draft", "confirm")
	if err != nil {
		t.Fatalf("should allow draft->confirmed: %v", err)
	}
	if newState != "confirmed" {
		t.Errorf("expected confirmed, got %s", newState)
	}

	_, err = wf.CanTransition("confirmed", "confirm")
	if err == nil {
		t.Fatal("should not allow confirmed->confirm")
	}

	newState2, err := wf.CanTransition("confirmed", "cancel")
	if err != nil {
		t.Fatalf("should allow confirmed->cancelled: %v", err)
	}
	if newState2 != "cancelled" {
		t.Errorf("expected cancelled, got %s", newState2)
	}
}

func TestWorkflow_MultiFrom(t *testing.T) {
	data := []byte(`{
		"name": "test",
		"model": "order",
		"field": "status",
		"states": { "a": {}, "b": {}, "c": {} },
		"transitions": [
			{ "from": ["a", "b"], "to": "c", "action": "finish" }
		]
	}`)

	wf, _ := ParseWorkflow(data)

	if _, err := wf.CanTransition("a", "finish"); err != nil {
		t.Errorf("should allow a->c: %v", err)
	}
	if _, err := wf.CanTransition("b", "finish"); err != nil {
		t.Errorf("should allow b->c: %v", err)
	}
	if _, err := wf.CanTransition("c", "finish"); err == nil {
		t.Error("should not allow c->c")
	}
}
