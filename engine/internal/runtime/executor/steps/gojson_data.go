package steps

import (
	"context"
	"fmt"

	"github.com/bitcode-framework/bitcode/internal/compiler/parser"
	"github.com/bitcode-framework/bitcode/internal/runtime/bridge"
	"github.com/bitcode-framework/bitcode/internal/runtime/executor"
)

// GoJSONDataHandler adapts old data step definitions to go-json bridge calls
// for processes running with runtime: "go-json".
type GoJSONDataHandler struct {
	Runner *GoJSONRunner
}

func (h *GoJSONDataHandler) Execute(ctx context.Context, execCtx *executor.Context, step parser.StepDefinition) error {
	modelName := step.Model
	if modelName == "" {
		return fmt.Errorf("go-json data step requires a model name")
	}

	if h.Runner == nil || h.Runner.BridgeCtx == nil {
		return fmt.Errorf("go-json data handler: bridge context not configured")
	}

	bc := h.Runner.BridgeCtx
	handle := bc.Model(modelName)

	var result any
	var err error

	switch step.Type {
	case "query":
		opts := bridge.SearchOptions{}
		if step.Domain != nil {
			opts.Domain = step.Domain
		}
		result, err = handle.Search(opts)

	case "create":
		data := step.Set
		if data == nil {
			data = make(map[string]any)
		}
		result, err = handle.Create(data)

	case "update":
		data := step.Set
		if data == nil {
			data = make(map[string]any)
		}
		id := ""
		if v, ok := execCtx.Variables["id"]; ok {
			id = fmt.Sprintf("%v", v)
		}
		err = handle.Write(id, data)

	case "delete":
		id := ""
		if v, ok := execCtx.Variables["id"]; ok {
			id = fmt.Sprintf("%v", v)
		}
		err = handle.Delete(id)

	case "upsert":
		data := step.Set
		if data == nil {
			data = make(map[string]any)
		}
		result, err = handle.Upsert(data, step.Unique)

	case "count":
		opts := bridge.SearchOptions{}
		if step.Domain != nil {
			opts.Domain = step.Domain
		}
		result, err = handle.Count(opts)

	case "sum":
		opts := bridge.SearchOptions{}
		if step.Domain != nil {
			opts.Domain = step.Domain
		}
		result, err = handle.Sum(step.SumField, opts)

	default:
		return fmt.Errorf("go-json data handler: unsupported step type %q", step.Type)
	}

	if err != nil {
		return err
	}

	varName := step.Into
	if varName == "" {
		varName = "result"
	}
	execCtx.Variables[varName] = result
	execCtx.Result = result
	return nil
}
