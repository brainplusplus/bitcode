package steps

import (
	"context"

	"github.com/bitcode-engine/engine/internal/compiler/parser"
	"github.com/bitcode-engine/engine/internal/runtime/executor"
)

type EmitHandler struct{}

func (h *EmitHandler) Execute(ctx context.Context, execCtx *executor.Context, step parser.StepDefinition) error {
	event := executor.Event{
		Name: step.Event,
		Data: step.Data,
	}
	execCtx.Events = append(execCtx.Events, event)
	return nil
}
