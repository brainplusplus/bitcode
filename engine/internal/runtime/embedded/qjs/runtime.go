package qjs_runtime

import (
	"github.com/bitcode-framework/bitcode/internal/runtime/embedded"
	"github.com/fastschema/qjs"
)

type QJSRuntime struct{}

func New() *QJSRuntime {
	return &QJSRuntime{}
}

func (r *QJSRuntime) Name() string { return "quickjs" }

func (r *QJSRuntime) NewVM(opts embedded.VMOptions) (embedded.VM, error) {
	rt, err := qjs.New()
	if err != nil {
		return nil, err
	}
	return &QJSVM{rt: rt, opts: opts}, nil
}
