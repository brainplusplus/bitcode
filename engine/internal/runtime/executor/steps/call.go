package steps

import (
	"context"
	"fmt"

	"github.com/bitcode-engine/engine/internal/compiler/parser"
	"github.com/bitcode-engine/engine/internal/runtime/executor"
)

type ProcessLoader interface {
	LoadProcess(name string) (*parser.ProcessDefinition, error)
}

type CallHandler struct {
	Executor *executor.Executor
	Loader   ProcessLoader
}

func (h *CallHandler) Execute(ctx context.Context, execCtx *executor.Context, step parser.StepDefinition) error {
	if step.Process == "" {
		return fmt.Errorf("call step requires a process name")
	}
	if h.Loader == nil {
		return fmt.Errorf("no process loader configured")
	}

	proc, err := h.Loader.LoadProcess(step.Process)
	if err != nil {
		return fmt.Errorf("failed to load process %s: %w", step.Process, err)
	}

	subCtx, err := h.Executor.Execute(ctx, proc, execCtx.Input, execCtx.UserID)
	if err != nil {
		return fmt.Errorf("sub-process %s failed: %w", step.Process, err)
	}

	execCtx.Result = subCtx.Result
	for k, v := range subCtx.Variables {
		execCtx.Variables[k] = v
	}
	execCtx.Events = append(execCtx.Events, subCtx.Events...)
	return nil
}
