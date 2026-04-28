package goja_runtime

import (
	"github.com/bitcode-framework/bitcode/internal/runtime/embedded"
	"github.com/dop251/goja"
)

type GojaRuntime struct{}

func New() *GojaRuntime {
	return &GojaRuntime{}
}

func (r *GojaRuntime) Name() string { return "goja" }

func (r *GojaRuntime) NewVM(opts embedded.VMOptions) (embedded.VM, error) {
	rt := goja.New()
	rt.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))
	return &GojaVM{rt: rt, opts: opts}, nil
}
