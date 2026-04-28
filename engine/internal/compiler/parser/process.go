package parser

import (
	"encoding/json"
	"fmt"
	"os"
)

type StepType string

const (
	StepValidate StepType = "validate"
	StepQuery    StepType = "query"
	StepCreate   StepType = "create"
	StepUpdate   StepType = "update"
	StepDelete   StepType = "delete"
	StepIf       StepType = "if"
	StepSwitch   StepType = "switch"
	StepLoop     StepType = "loop"
	StepEmit     StepType = "emit"
	StepCall     StepType = "call"
	StepScript   StepType = "script"
	StepHTTP     StepType = "http"
	StepAssign   StepType = "assign"
	StepLog      StepType = "log"
	StepUpsert   StepType = "upsert"
	StepCount    StepType = "count"
	StepSum      StepType = "sum"
)

type StepDefinition struct {
	Name  string   `json:"name,omitempty"`
	Label string   `json:"label,omitempty"`
	Type  StepType `json:"type"`

	// validate
	Rules map[string]map[string]any `json:"rules,omitempty"`
	Error string                    `json:"error,omitempty"`

	// query
	Model  string  `json:"model,omitempty"`
	Domain [][]any `json:"domain,omitempty"`
	OQL    string  `json:"oql,omitempty"`
	Into   string  `json:"into,omitempty"`

	// create / update
	Set map[string]any `json:"set,omitempty"`

	// if
	Condition string           `json:"condition,omitempty"`
	Then      string           `json:"then,omitempty"`
	Else      string           `json:"else,omitempty"`
	ThenSteps []StepDefinition `json:"then_steps,omitempty"`
	ElseSteps []StepDefinition `json:"else_steps,omitempty"`

	// switch
	Field     string                       `json:"field,omitempty"`
	Cases     map[string]string            `json:"cases,omitempty"`
	CaseSteps map[string][]StepDefinition  `json:"case_steps,omitempty"`

	// loop
	Over  string           `json:"over,omitempty"`
	Steps []StepDefinition `json:"steps,omitempty"`

	// emit
	Event string         `json:"event,omitempty"`
	Data  map[string]any `json:"data,omitempty"`

	// call
	Process string `json:"process,omitempty"`

	// script
	Runtime string `json:"runtime,omitempty"`
	Script  string `json:"script,omitempty"`

	// http
	URL     string            `json:"url,omitempty"`
	Method  string            `json:"method,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    any               `json:"body,omitempty"`

	// assign
	Variable string `json:"variable,omitempty"`
	Value    any    `json:"value,omitempty"`

	// log
	Message string `json:"message,omitempty"`

	// upsert
	Unique []string `json:"unique,omitempty"`

	// sum
	SumField string `json:"sum_field,omitempty"`
}

type EdgeDefinition struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Condition string `json:"condition,omitempty"`
}

type ProcessDefinition struct {
	Name    string                    `json:"name"`
	Runtime string                    `json:"runtime,omitempty"`
	Steps   []StepDefinition          `json:"steps,omitempty"`
	Nodes   map[string]StepDefinition `json:"nodes,omitempty"`
	Edges   []EdgeDefinition          `json:"edges,omitempty"`
}

func (p *ProcessDefinition) IsDAG() bool {
	return len(p.Nodes) > 0
}

// IsGoJSON returns true if the process uses the go-json runtime.
func (p *ProcessDefinition) IsGoJSON() bool {
	return p.Runtime == "go-json"
}

func ParseProcess(data []byte) (*ProcessDefinition, error) {
	var proc ProcessDefinition
	if err := json.Unmarshal(data, &proc); err != nil {
		return nil, fmt.Errorf("invalid process JSON: %w", err)
	}
	if proc.Name == "" {
		return nil, fmt.Errorf("process name is required")
	}

	if proc.IsDAG() {
		return validateDAGProcess(&proc)
	}

	if len(proc.Steps) == 0 {
		return nil, fmt.Errorf("process must have at least one step")
	}
	for i, step := range proc.Steps {
		if step.Type == "" {
			return nil, fmt.Errorf("step %d must have a type", i)
		}
	}
	return &proc, nil
}

func validateDAGProcess(proc *ProcessDefinition) (*ProcessDefinition, error) {
	if len(proc.Nodes) == 0 {
		return nil, fmt.Errorf("DAG process must have at least one node")
	}
	for id, node := range proc.Nodes {
		if node.Type == "" {
			return nil, fmt.Errorf("node %q must have a type", id)
		}
	}
	for i, edge := range proc.Edges {
		if edge.From == "" || edge.To == "" {
			return nil, fmt.Errorf("edge %d must have 'from' and 'to'", i)
		}
		if _, ok := proc.Nodes[edge.From]; !ok {
			return nil, fmt.Errorf("edge %d references unknown node %q", i, edge.From)
		}
		if _, ok := proc.Nodes[edge.To]; !ok {
			return nil, fmt.Errorf("edge %d references unknown node %q", i, edge.To)
		}
	}
	return proc, nil
}

func ParseProcessFile(path string) (*ProcessDefinition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read process file %s: %w", path, err)
	}
	return ParseProcess(data)
}
