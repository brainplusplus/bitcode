package executor

import (
	"context"
	"fmt"

	"github.com/bitcode-framework/bitcode/internal/compiler/parser"
)

const MaxNestingDepth = 10

type Translator interface {
	Translate(locale string, key string) string
}

type Context struct {
	Input      map[string]any
	Variables  map[string]any
	Result     any
	UserID     string
	Locale     string
	Events     []Event
	Translator Translator
	depth      int
}

func (c *Context) T(key string) string {
	if c.Translator == nil || c.Locale == "" {
		return key
	}
	return c.Translator.Translate(c.Locale, key)
}

type Event struct {
	Name string
	Data map[string]any
}

type StepHandler interface {
	Execute(ctx context.Context, execCtx *Context, step parser.StepDefinition) error
}

type Executor struct {
	handlers map[parser.StepType]StepHandler
}

func NewExecutor() *Executor {
	return &Executor{
		handlers: make(map[parser.StepType]StepHandler),
	}
}

func (e *Executor) RegisterHandler(stepType parser.StepType, handler StepHandler) {
	e.handlers[stepType] = handler
}

func (e *Executor) Execute(ctx context.Context, process *parser.ProcessDefinition, input map[string]any, userID string) (*Context, error) {
	if process.IsDAG() {
		return e.ExecuteDAG(ctx, process, input, userID)
	}

	execCtx := &Context{
		Input:     input,
		Variables: make(map[string]any),
		UserID:    userID,
	}

	if err := e.ExecuteSteps(ctx, execCtx, process.Steps); err != nil {
		return nil, err
	}

	return execCtx, nil
}

func (e *Executor) ExecuteSteps(ctx context.Context, execCtx *Context, steps []parser.StepDefinition) error {
	if execCtx.depth >= MaxNestingDepth {
		return fmt.Errorf("maximum nesting depth (%d) exceeded", MaxNestingDepth)
	}

	execCtx.depth++
	defer func() { execCtx.depth-- }()

	for i, step := range steps {
		handler, ok := e.handlers[step.Type]
		if !ok {
			return fmt.Errorf("no handler for step type %q at step %d", step.Type, i)
		}

		if err := handler.Execute(ctx, execCtx, step); err != nil {
			return fmt.Errorf("step %d (%s) failed: %w", i, step.Type, err)
		}
	}

	return nil
}
