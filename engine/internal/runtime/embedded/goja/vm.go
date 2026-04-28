package goja_runtime

import (
	"fmt"
	"os"
	"sync"

	"github.com/bitcode-framework/bitcode/internal/runtime/bridge"
	"github.com/bitcode-framework/bitcode/internal/runtime/embedded"
	"github.com/dop251/goja"
)

var compiledScripts sync.Map

type GojaVM struct {
	rt   *goja.Runtime
	opts embedded.VMOptions
}

func (v *GojaVM) InjectBridge(bc *bridge.Context) error {
	v.rt.Set("bitcode", v.buildBitcodeObject(bc))
	v.rt.Set("console", map[string]any{
		"log":   func(args ...any) { bc.Log("info", fmt.Sprint(args...)) },
		"warn":  func(args ...any) { bc.Log("warn", fmt.Sprint(args...)) },
		"error": func(args ...any) { bc.Log("error", fmt.Sprint(args...)) },
		"debug": func(args ...any) { bc.Log("debug", fmt.Sprint(args...)) },
	})
	return nil
}

func (v *GojaVM) InjectParams(params map[string]any) error {
	v.rt.Set("params", params)
	return nil
}

func (v *GojaVM) Execute(code string, filename string) (any, error) {
	program, err := v.getCompiled(filename, code)
	if err != nil {
		return nil, bridge.NewError("SYNTAX_ERROR", err.Error())
	}

	val, err := v.rt.RunProgram(program)
	if err != nil {
		return v.handleError(err)
	}

	return v.resolveResult(val)
}

func (v *GojaVM) Interrupt(reason string) {
	v.rt.Interrupt(reason)
}

func (v *GojaVM) Close() {
	v.rt.ClearInterrupt()
}

func (v *GojaVM) getCompiled(filename, code string) (*goja.Program, error) {
	info, statErr := os.Stat(filename)
	if statErr != nil {
		return goja.Compile(filename, code, true)
	}

	cacheKey := filename + ":" + info.ModTime().String()
	if cached, ok := compiledScripts.Load(cacheKey); ok {
		return cached.(*goja.Program), nil
	}

	program, err := goja.Compile(filename, code, true)
	if err != nil {
		return nil, err
	}
	compiledScripts.Store(cacheKey, program)
	return program, nil
}

func (v *GojaVM) handleError(err error) (any, error) {
	if interrupted, ok := err.(*goja.InterruptedError); ok {
		return nil, bridge.NewError("STEP_TIMEOUT", fmt.Sprintf("%v", interrupted.Value()))
	}
	if exception, ok := err.(*goja.Exception); ok {
		return nil, bridge.NewError("SCRIPT_ERROR", exception.String())
	}
	return nil, bridge.NewError("SCRIPT_ERROR", err.Error())
}

func (v *GojaVM) resolveResult(val goja.Value) (any, error) {
	if val == nil || goja.IsUndefined(val) || goja.IsNull(val) {
		return nil, nil
	}

	exported := val.Export()

	if m, ok := exported.(map[string]any); ok {
		if execFn, exists := m["execute"]; exists {
			if callable, ok := goja.AssertFunction(v.rt.ToValue(execFn)); ok {
				return v.callExecuteFunc(callable)
			}
		}
	}

	if callable, ok := goja.AssertFunction(val); ok {
		return v.callExecuteFunc(callable)
	}

	return exported, nil
}

func (v *GojaVM) callExecuteFunc(fn goja.Callable) (any, error) {
	bitcodeVal := v.rt.Get("bitcode")
	paramsVal := v.rt.Get("params")

	result, err := fn(goja.Undefined(), bitcodeVal, paramsVal)
	if err != nil {
		return v.handleError(err)
	}

	if result == nil || goja.IsUndefined(result) || goja.IsNull(result) {
		return nil, nil
	}
	return result.Export(), nil
}
