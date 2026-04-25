package steps

import (
	"context"
	"fmt"

	"github.com/bitcode-framework/bitcode/internal/compiler/parser"
	"github.com/bitcode-framework/bitcode/internal/runtime/executor"
)

type ScriptRunner interface {
	Run(ctx context.Context, script string, params map[string]any) (any, error)
}

type ScriptHandler struct {
	Runner ScriptRunner
}

func (h *ScriptHandler) Execute(ctx context.Context, execCtx *executor.Context, step parser.StepDefinition) error {
	if h.Runner == nil {
		return fmt.Errorf("no script runner configured")
	}
	if step.Script == "" {
		return fmt.Errorf("script step requires a script path")
	}

	params := map[string]any{
		"input":     execCtx.Input,
		"variables": execCtx.Variables,
		"result":    execCtx.Result,
		"user_id":   execCtx.UserID,
	}

	result, err := h.Runner.Run(ctx, step.Script, params)
	if err != nil {
		return fmt.Errorf("script %s failed: %w", step.Script, err)
	}

	varName := step.Into
	if varName == "" {
		varName = "script_result"
	}
	execCtx.Variables[varName] = result
	execCtx.Result = result
	return nil
}
