package yaegi_runtime

import (
	"context"
	"fmt"
	"reflect"
	"sync/atomic"
	"time"

	"github.com/bitcode-framework/bitcode/internal/runtime/bridge"
	"github.com/bitcode-framework/bitcode/internal/runtime/embedded"
	"github.com/traefik/yaegi/interp"
)

type YaegiVM struct {
	interp    *interp.Interpreter
	opts      embedded.VMOptions
	cancelled atomic.Bool
	params    map[string]any
}

func (v *YaegiVM) InjectBridge(bc *bridge.Context) error {
	symbols := BuildBridgeSymbols(bc)
	return v.interp.Use(symbols)
}

func (v *YaegiVM) InjectParams(params map[string]any) error {
	v.params = params
	symbols := interp.Exports{
		"params/params": map[string]reflect.Value{
			"Get": reflect.ValueOf(func() map[string]any { return params }),
		},
	}
	return v.interp.Use(symbols)
}

func (v *YaegiVM) Execute(code string, filename string) (any, error) {
	_, err := v.interp.Eval(code)
	if err != nil {
		return nil, bridge.NewError("SYNTAX_ERROR", err.Error())
	}

	execFn, err := v.findExecuteFunc()
	if err != nil {
		return nil, bridge.NewError("SCRIPT_ERROR", err.Error())
	}

	return v.callWithTimeout(execFn)
}

func (v *YaegiVM) Interrupt(reason string) {
	v.cancelled.Store(true)
}

func (v *YaegiVM) Close() {}

// findExecuteFunc looks up main.Execute in the interpreted code.
// Supports two signatures:
//   - func Execute(ctx context.Context, params map[string]any) (any, error)
//   - func Execute(params map[string]any) (any, error)
func (v *YaegiVM) findExecuteFunc() (reflect.Value, error) {
	val, err := v.interp.Eval("main.Execute")
	if err != nil {
		return reflect.Value{}, fmt.Errorf("script must export func Execute: %w", err)
	}
	if !val.IsValid() || val.Kind() != reflect.Func {
		return reflect.Value{}, fmt.Errorf("main.Execute is not a function")
	}
	return val, nil
}

// callWithTimeout runs the Execute function in a goroutine with context-based timeout.
// If the script ignores ctx.Done(), the goroutine leaks — this is a documented limitation.
func (v *YaegiVM) callWithTimeout(fn reflect.Value) (any, error) {
	timeout := v.opts.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	type result struct {
		value any
		err   error
	}
	done := make(chan result, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- result{nil, bridge.NewErrorf(bridge.ErrInternalError, "Go script panic: %v", r)}
			}
		}()

		val, callErr := v.callExecuteFunc(ctx, fn)
		done <- result{val, callErr}
	}()

	select {
	case res := <-done:
		return res.value, res.err
	case <-ctx.Done():
		v.cancelled.Store(true)
		return nil, bridge.NewError("STEP_TIMEOUT",
			fmt.Sprintf("Go script timed out after %s. Scripts should check ctx.Done() for cooperative cancellation.", timeout))
	}
}

// callExecuteFunc detects the function signature and calls accordingly.
// Two-param: Execute(ctx, params) — recommended, supports cooperative timeout.
// One-param: Execute(params) — simple, no timeout cooperation.
func (v *YaegiVM) callExecuteFunc(ctx context.Context, fn reflect.Value) (any, error) {
	fnType := fn.Type()
	params := v.params
	if params == nil {
		params = make(map[string]any)
	}

	var results []reflect.Value

	switch fnType.NumIn() {
	case 2:
		results = fn.Call([]reflect.Value{
			reflect.ValueOf(ctx),
			reflect.ValueOf(params),
		})
	case 1:
		results = fn.Call([]reflect.Value{
			reflect.ValueOf(params),
		})
	case 0:
		results = fn.Call(nil)
	default:
		return nil, bridge.NewError("SCRIPT_ERROR",
			fmt.Sprintf("Execute function has %d parameters, expected 0-2", fnType.NumIn()))
	}

	return v.extractResults(results)
}

func (v *YaegiVM) extractResults(results []reflect.Value) (any, error) {
	switch len(results) {
	case 0:
		return nil, nil
	case 1:
		return interfaceOrNil(results[0]), nil
	case 2:
		val := interfaceOrNil(results[0])
		errVal := interfaceOrNil(results[1])
		if errVal != nil {
			if e, ok := errVal.(error); ok {
				return val, e
			}
		}
		return val, nil
	default:
		return interfaceOrNil(results[0]), nil
	}
}

func interfaceOrNil(v reflect.Value) any {
	if !v.IsValid() {
		return nil
	}
	if v.Kind() == reflect.Interface || v.Kind() == reflect.Ptr ||
		v.Kind() == reflect.Map || v.Kind() == reflect.Slice {
		if v.IsNil() {
			return nil
		}
	}
	return v.Interface()
}
