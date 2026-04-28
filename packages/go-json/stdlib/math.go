package stdlib

import (
	"fmt"
	"math"
	"math/rand"

	"github.com/expr-lang/expr"
)

func RegisterMath(r *Registry) {
	r.Register(expr.Function("clamp", func(params ...any) (any, error) {
		x, ok1 := toFloat64(params[0])
		min, ok2 := toFloat64(params[1])
		max, ok3 := toFloat64(params[2])
		if !ok1 || !ok2 || !ok3 {
			return nil, fmt.Errorf("clamp: all arguments must be numbers")
		}
		if min > max {
			return nil, fmt.Errorf("clamp: min (%v) must be <= max (%v)", min, max)
		}
		if x < min {
			return min, nil
		}
		if x > max {
			return max, nil
		}
		return x, nil
	}))

	r.Register(expr.Function("sign", func(params ...any) (any, error) {
		x, ok := toFloat64(params[0])
		if !ok {
			return nil, fmt.Errorf("sign: argument must be a number")
		}
		if x > 0 {
			return 1, nil
		}
		if x < 0 {
			return -1, nil
		}
		return 0, nil
	}))

	r.Register(expr.Function("randomInt", func(params ...any) (any, error) {
		min, ok1 := toFloat64(params[0])
		max, ok2 := toFloat64(params[1])
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("randomInt: arguments must be numbers")
		}
		minI := int(min)
		maxI := int(max)
		if minI > maxI {
			return nil, fmt.Errorf("randomInt: min (%d) must be <= max (%d)", minI, maxI)
		}
		if minI == maxI {
			return minI, nil
		}
		return minI + rand.Intn(maxI-minI+1), nil
	}))

	r.Register(expr.Function("randomFloat", func(params ...any) (any, error) {
		min, ok1 := toFloat64(params[0])
		max, ok2 := toFloat64(params[1])
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("randomFloat: arguments must be numbers")
		}
		if min > max {
			return nil, fmt.Errorf("randomFloat: min must be <= max")
		}
		return min + rand.Float64()*(max-min), nil
	}))

	r.Register(expr.Function("pow", func(params ...any) (any, error) {
		base, ok1 := toFloat64(params[0])
		exp, ok2 := toFloat64(params[1])
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("pow: arguments must be numbers")
		}
		return math.Pow(base, exp), nil
	}))

	r.Register(expr.Function("sqrt", func(params ...any) (any, error) {
		x, ok := toFloat64(params[0])
		if !ok {
			return nil, fmt.Errorf("sqrt: argument must be a number")
		}
		return math.Sqrt(x), nil
	}))

	r.Register(expr.Function("mod", func(params ...any) (any, error) {
		a, ok1 := toFloat64(params[0])
		b, ok2 := toFloat64(params[1])
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("mod: arguments must be numbers")
		}
		if b == 0 {
			return nil, fmt.Errorf("mod: division by zero")
		}
		return math.Mod(a, b), nil
	}))
}

func toFloat64(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case float32:
		return float64(n), true
	}
	return 0, false
}
