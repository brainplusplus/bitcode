package stdlib

import (
	"github.com/expr-lang/expr"
)

func RegisterTypes(r *Registry) {
	r.Register(expr.Function("bool", func(params ...any) (any, error) {
		v := params[0]
		if v == nil {
			return false, nil
		}
		switch val := v.(type) {
		case bool:
			return val, nil
		case int:
			return val != 0, nil
		case int64:
			return val != 0, nil
		case float64:
			return val != 0, nil
		case string:
			return val != "", nil
		case []any:
			return len(val) > 0, nil
		case map[string]any:
			return len(val) > 0, nil
		default:
			return true, nil
		}
	}))

	r.Register(expr.Function("isNil", func(params ...any) (any, error) {
		return params[0] == nil, nil
	}))
}
