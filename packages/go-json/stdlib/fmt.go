package stdlib

import (
	"fmt"

	"github.com/expr-lang/expr"
)

// RegisterFormat registers the sprintf formatting function.
func RegisterFormat(r *Registry) {
	r.Register(expr.Function("sprintf", func(params ...any) (any, error) {
		if len(params) < 1 {
			return nil, fmt.Errorf("sprintf: requires at least a format string")
		}
		format, ok := params[0].(string)
		if !ok {
			return nil, fmt.Errorf("sprintf: first argument must be a format string")
		}
		return fmt.Sprintf(format, params[1:]...), nil
	}))
}
