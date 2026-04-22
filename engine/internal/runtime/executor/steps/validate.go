package steps

import (
	"context"
	"fmt"
	"strings"

	"github.com/bitcode-engine/engine/internal/compiler/parser"
	"github.com/bitcode-engine/engine/internal/runtime/executor"
)

type ValidateHandler struct{}

func (h *ValidateHandler) Execute(ctx context.Context, execCtx *executor.Context, step parser.StepDefinition) error {
	data := execCtx.Input
	if execCtx.Result != nil {
		if m, ok := execCtx.Result.(map[string]any); ok {
			data = m
		}
	}

	for field, rules := range step.Rules {
		val, exists := data[field]
		for ruleName, ruleVal := range rules {
			switch ruleName {
			case "eq":
				if fmt.Sprintf("%v", val) != fmt.Sprintf("%v", ruleVal) {
					return fmt.Errorf("%s", translateError(step.Error, execCtx))
				}
			case "neq":
				if fmt.Sprintf("%v", val) == fmt.Sprintf("%v", ruleVal) {
					return fmt.Errorf("%s", translateError(step.Error, execCtx))
				}
			case "required":
				if ruleVal == true && (!exists || val == nil || val == "") {
					errMsg := step.Error
					if errMsg == "" {
						errMsg = fmt.Sprintf("field %s is required", field)
					}
					return fmt.Errorf("%s", translateError(errMsg, execCtx))
				}
			}
		}
	}
	return nil
}

func translateError(msg string, execCtx *executor.Context) string {
	msg = interpolate(msg, execCtx)
	if execCtx.Translator != nil && execCtx.Locale != "" && !strings.Contains(msg, "{{") {
		translated := execCtx.T(msg)
		if translated != msg {
			return translated
		}
	}
	return msg
}
