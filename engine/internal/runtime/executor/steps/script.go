package steps

import (
	"context"
	"fmt"
	"strings"

	"github.com/bitcode-framework/bitcode/internal/compiler/parser"
	"github.com/bitcode-framework/bitcode/internal/runtime/executor"
)

type ScriptRunner interface {
	Run(ctx context.Context, script string, params map[string]any) (any, error)
}

type EmbeddedRunner interface {
	CanHandle(runtime string) bool
	Run(ctx context.Context, script string, params map[string]any) (any, error)
}

type ScriptHandler struct {
	Runner         ScriptRunner
	EmbeddedRunner EmbeddedRunner
}

func (h *ScriptHandler) Execute(ctx context.Context, execCtx *executor.Context, step parser.StepDefinition) error {
	if step.Script == "" {
		return fmt.Errorf("script step requires a script path")
	}

	params := map[string]any{
		"input":     execCtx.Input,
		"variables": execCtx.Variables,
		"result":    execCtx.Result,
		"user_id":   execCtx.UserID,
	}

	runner := h.selectRunner(step)
	if runner == nil {
		return fmt.Errorf("no script runner configured for runtime %q", step.Runtime)
	}

	if step.Runtime != "" {
		params["__runtime"] = step.Runtime
	}

	result, err := runner.Run(ctx, step.Script, params)
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

func (h *ScriptHandler) selectRunner(step parser.StepDefinition) ScriptRunner {
	rt := step.Runtime
	if rt == "" {
		rt = detectRuntimeFromExtension(step.Script)
	}

	if h.EmbeddedRunner != nil && h.EmbeddedRunner.CanHandle(rt) {
		return h.EmbeddedRunner
	}

	return h.Runner
}

func detectRuntimeFromExtension(script string) string {
	switch {
	case strings.HasSuffix(script, ".js"):
		return "javascript"
	case strings.HasSuffix(script, ".go"):
		return "go"
	case strings.HasSuffix(script, ".ts"):
		return "node"
	case strings.HasSuffix(script, ".py"):
		return "python"
	case strings.HasSuffix(script, ".json"):
		return "go-json"
	default:
		return ""
	}
}
