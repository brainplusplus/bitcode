package stdlib

import (
	"github.com/expr-lang/expr"
)

type Registry struct {
	functions []expr.Option
}

func NewRegistry() *Registry {
	return &Registry{}
}

func (r *Registry) Register(opt expr.Option) {
	r.functions = append(r.functions, opt)
}

// All returns all registered stdlib functions as expr.Option slice.
func (r *Registry) All() []expr.Option {
	return r.functions
}

// DefaultRegistry creates a registry with all Layer 2 stdlib functions.
func DefaultRegistry() *Registry {
	r := NewRegistry()
	RegisterMath(r)
	RegisterStrings(r)
	RegisterArrays(r)
	RegisterTypes(r)
	return r
}
