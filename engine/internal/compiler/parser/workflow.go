package parser

import (
	"encoding/json"
	"fmt"
	"os"
)

type StateDefinition struct {
	Label string `json:"label"`
}

type TransitionDefinition struct {
	From       any    `json:"from"`
	To         string `json:"to"`
	Action     string `json:"action"`
	Permission string `json:"permission,omitempty"`
	Process    string `json:"process,omitempty"`
}

func (t *TransitionDefinition) FromStates() []string {
	switch v := t.From.(type) {
	case string:
		return []string{v}
	case []any:
		states := make([]string, len(v))
		for i, s := range v {
			states[i] = fmt.Sprintf("%v", s)
		}
		return states
	default:
		return nil
	}
}

type WorkflowDefinition struct {
	Name        string                     `json:"name"`
	Model       string                     `json:"model"`
	Field       string                     `json:"field"`
	States      map[string]StateDefinition `json:"states"`
	Transitions []TransitionDefinition     `json:"transitions"`
}

func (w *WorkflowDefinition) InitialState() string {
	for _, t := range w.Transitions {
		for name := range w.States {
			isTarget := false
			for _, t2 := range w.Transitions {
				if t2.To == name {
					isTarget = true
					break
				}
			}
			_ = t
			if !isTarget {
				return name
			}
		}
		break
	}
	for name := range w.States {
		return name
	}
	return ""
}

func (w *WorkflowDefinition) CanTransition(currentState string, action string) (string, error) {
	for _, t := range w.Transitions {
		if t.Action != action {
			continue
		}
		for _, from := range t.FromStates() {
			if from == currentState {
				return t.To, nil
			}
		}
	}
	return "", fmt.Errorf("transition %q not allowed from state %q", action, currentState)
}

func (w *WorkflowDefinition) GetTransition(action string) *TransitionDefinition {
	for i, t := range w.Transitions {
		if t.Action == action {
			return &w.Transitions[i]
		}
	}
	return nil
}

func ParseWorkflow(data []byte) (*WorkflowDefinition, error) {
	var wf WorkflowDefinition
	if err := json.Unmarshal(data, &wf); err != nil {
		return nil, fmt.Errorf("invalid workflow JSON: %w", err)
	}
	if wf.Name == "" {
		return nil, fmt.Errorf("workflow name is required")
	}
	if wf.Model == "" {
		return nil, fmt.Errorf("workflow model is required")
	}
	if wf.Field == "" {
		wf.Field = "status"
	}
	if len(wf.States) == 0 {
		return nil, fmt.Errorf("workflow must have at least one state")
	}
	if len(wf.Transitions) == 0 {
		return nil, fmt.Errorf("workflow must have at least one transition")
	}
	return &wf, nil
}

func ParseWorkflowFile(path string) (*WorkflowDefinition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read workflow file %s: %w", path, err)
	}
	return ParseWorkflow(data)
}
