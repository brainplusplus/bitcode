package qjs_runtime

import (
	"fmt"

	"github.com/bitcode-framework/bitcode/internal/runtime/bridge"
	"github.com/bitcode-framework/bitcode/internal/runtime/embedded"
	"github.com/fastschema/qjs"
)

type QJSVM struct {
	rt   *qjs.Runtime
	opts embedded.VMOptions
}

func (v *QJSVM) InjectBridge(bc *bridge.Context) error {
	ctx := v.rt.Context()
	registerHostFunctions(ctx, bc)

	_, err := ctx.Eval("__bitcode_init__", qjs.Code(bitcodeInitJS))
	if err != nil {
		return bridge.NewErrorf(bridge.ErrInternalError, "failed to init bitcode wrapper: %s", err)
	}

	return nil
}

func (v *QJSVM) InjectParams(params map[string]any) error {
	ctx := v.rt.Context()
	paramsVal, err := qjs.ToJsValue(ctx, params)
	if err != nil {
		return bridge.NewErrorf(bridge.ErrInternalError, "failed to inject params: %s", err)
	}
	ctx.Global().SetPropertyStr("params", paramsVal)
	return nil
}

func (v *QJSVM) Execute(code string, filename string) (any, error) {
	result, err := v.rt.Eval(filename, qjs.Code(code))
	if err != nil {
		return nil, bridge.NewError("SCRIPT_ERROR", err.Error())
	}

	if result == nil || result.IsUndefined() || result.IsNull() {
		return nil, nil
	}

	if result.IsObject() {
		execProp := result.GetPropertyStr("execute")
		if execProp != nil && !execProp.IsUndefined() && execProp.IsFunction() {
			ctx := v.rt.Context()
			bitcodeVal := ctx.Global().GetPropertyStr("bitcode")
			paramsVal := ctx.Global().GetPropertyStr("params")
			callResult := result.Call("execute", bitcodeVal.Raw(), paramsVal.Raw())
			if callResult == nil {
				return nil, nil
			}
			return v.exportValue(callResult)
		}
	}

	return v.exportValue(result)
}

func (v *QJSVM) Interrupt(reason string) {
	v.rt.Close()
}

func (v *QJSVM) Close() {
	defer func() { recover() }()
	v.rt.Close()
}

func (v *QJSVM) exportValue(val *qjs.Value) (any, error) {
	if val == nil || val.IsUndefined() || val.IsNull() {
		return nil, nil
	}
	goVal, err := qjs.ToGoValue[any](val)
	if err != nil {
		return nil, bridge.NewErrorf(bridge.ErrInternalError, "failed to convert JS value: %s", err)
	}
	return goVal, nil
}

var _ = fmt.Sprintf
