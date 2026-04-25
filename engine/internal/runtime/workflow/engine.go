package workflow

import (
	"context"
	"fmt"
	"sync"

	"github.com/bitcode-framework/bitcode/internal/compiler/parser"
)

type Engine struct {
	workflows map[string]*parser.WorkflowDefinition
	mu        sync.RWMutex
}

func NewEngine() *Engine {
	return &Engine{
		workflows: make(map[string]*parser.WorkflowDefinition),
	}
}

func (e *Engine) Register(wf *parser.WorkflowDefinition) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.workflows[wf.Name] = wf
	e.workflows[wf.Model] = wf
}

func (e *Engine) Get(name string) (*parser.WorkflowDefinition, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	wf, ok := e.workflows[name]
	if !ok {
		return nil, fmt.Errorf("workflow %q not found", name)
	}
	return wf, nil
}

func (e *Engine) ExecuteTransition(ctx context.Context, workflowName string, currentState string, action string) (string, error) {
	wf, err := e.Get(workflowName)
	if err != nil {
		return "", err
	}

	newState, err := wf.CanTransition(currentState, action)
	if err != nil {
		return "", err
	}

	return newState, nil
}

func (e *Engine) GetInitialState(workflowName string) (string, error) {
	wf, err := e.Get(workflowName)
	if err != nil {
		return "", err
	}
	return wf.InitialState(), nil
}
