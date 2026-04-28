package lang

import (
	"sync"
	"time"
)

type DebugAction int

const (
	DebugContinue DebugAction = iota
	DebugStepOver
	DebugStepInto
	DebugPause
)

type StepInfo struct {
	Index int
	Type  string
	Node  Node
}

// Debugger receives callbacks during VM execution.
// All methods are optional — implement only what you need.
type Debugger interface {
	OnStep(info StepInfo) DebugAction
	OnVariable(name string, value any, scope string)
	OnError(err error, step int)
	OnFunctionCall(name string, args map[string]any)
	OnFunctionReturn(name string, result any)
}

// TraceEntry records a single step execution for post-mortem analysis.
type TraceEntry struct {
	Step       int    `json:"step"`
	Type       string `json:"type"`
	Var        string `json:"var,omitempty"`
	Value      any    `json:"value,omitempty"`
	Condition  string `json:"condition,omitempty"`
	Result     any    `json:"result,omitempty"`
	DurationUs int64  `json:"duration_us"`
}

// ExecutionTrace collects step-by-step execution data.
type ExecutionTrace struct {
	mu      sync.Mutex
	entries []TraceEntry
	start   time.Time
}

func NewExecutionTrace() *ExecutionTrace {
	return &ExecutionTrace{
		start: time.Now(),
	}
}

func (t *ExecutionTrace) AddStep(entry TraceEntry) {
	t.mu.Lock()
	t.entries = append(t.entries, entry)
	t.mu.Unlock()
}

func (t *ExecutionTrace) Entries() []TraceEntry {
	t.mu.Lock()
	defer t.mu.Unlock()
	cp := make([]TraceEntry, len(t.entries))
	copy(cp, t.entries)
	return cp
}

func (t *ExecutionTrace) TotalSteps() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.entries)
}

func (t *ExecutionTrace) TotalDurationUs() int64 {
	return time.Since(t.start).Microseconds()
}
