package steps

import (
	"context"
	"fmt"

	"github.com/bitcode-engine/engine/internal/compiler/parser"
	"github.com/bitcode-engine/engine/internal/runtime/executor"
)

type AssignHandler struct{}

func (h *AssignHandler) Execute(ctx context.Context, execCtx *executor.Context, step parser.StepDefinition) error {
	if step.Variable == "" {
		return fmt.Errorf("assign step requires a variable name")
	}
	execCtx.Variables[step.Variable] = step.Value
	return nil
}

type LogHandler struct{}

func (h *LogHandler) Execute(ctx context.Context, execCtx *executor.Context, step parser.StepDefinition) error {
	msg := interpolate(step.Message, execCtx)
	if execCtx.Translator != nil && execCtx.Locale != "" {
		translated := execCtx.T(msg)
		if translated != msg {
			msg = translated
		}
	}
	fmt.Printf("[AUDIT] %s\n", msg)
	return nil
}
