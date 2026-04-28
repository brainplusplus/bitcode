package stdlib

import (
	"fmt"
	"strings"

	"github.com/expr-lang/expr"
)

// RegisterMaps registers map/object helper functions.
func RegisterMaps(r *Registry) {
	r.Register(expr.Function("has", func(params ...any) (any, error) {
		m, ok := params[0].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("has: first argument must be a map")
		}
		key, ok := params[1].(string)
		if !ok {
			return nil, fmt.Errorf("has: second argument must be a string")
		}
		_, exists := m[key]
		return exists, nil
	}))

	r.Register(expr.Function("get", func(params ...any) (any, error) {
		m, ok := params[0].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("get: first argument must be a map")
		}
		path, ok := params[1].(string)
		if !ok {
			return nil, fmt.Errorf("get: second argument must be a string")
		}
		parts := strings.Split(path, ".")
		var current any = m
		for _, part := range parts {
			cm, ok := current.(map[string]any)
			if !ok {
				return nil, nil
			}
			current = cm[part]
			if current == nil {
				return nil, nil
			}
		}
		return current, nil
	}))

	r.Register(expr.Function("merge", func(params ...any) (any, error) {
		a, ok1 := params[0].(map[string]any)
		b, ok2 := params[1].(map[string]any)
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("merge: both arguments must be maps")
		}
		result := make(map[string]any, len(a)+len(b))
		for k, v := range a {
			result[k] = v
		}
		for k, v := range b {
			result[k] = v
		}
		return result, nil
	}))

	r.Register(expr.Function("pick", func(params ...any) (any, error) {
		m, ok := params[0].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("pick: first argument must be a map")
		}
		keys, ok := params[1].([]any)
		if !ok {
			return nil, fmt.Errorf("pick: second argument must be an array of strings")
		}
		result := make(map[string]any)
		for _, k := range keys {
			key, ok := k.(string)
			if !ok {
				continue
			}
			if v, exists := m[key]; exists {
				result[key] = v
			}
		}
		return result, nil
	}))

	r.Register(expr.Function("omit", func(params ...any) (any, error) {
		m, ok := params[0].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("omit: first argument must be a map")
		}
		keys, ok := params[1].([]any)
		if !ok {
			return nil, fmt.Errorf("omit: second argument must be an array of strings")
		}
		exclude := make(map[string]bool, len(keys))
		for _, k := range keys {
			if key, ok := k.(string); ok {
				exclude[key] = true
			}
		}
		result := make(map[string]any)
		for k, v := range m {
			if !exclude[k] {
				result[k] = v
			}
		}
		return result, nil
	}))
}
