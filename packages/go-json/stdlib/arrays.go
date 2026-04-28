package stdlib

import (
	"fmt"

	"github.com/expr-lang/expr"
)

func RegisterArrays(r *Registry) {
	r.Register(expr.Function("append", func(params ...any) (any, error) {
		arr, ok := params[0].([]any)
		if !ok {
			return nil, fmt.Errorf("append: first argument must be an array")
		}
		result := make([]any, len(arr)+1)
		copy(result, arr)
		result[len(arr)] = params[1]
		return result, nil
	}))

	r.Register(expr.Function("prepend", func(params ...any) (any, error) {
		arr, ok := params[0].([]any)
		if !ok {
			return nil, fmt.Errorf("prepend: first argument must be an array")
		}
		result := make([]any, len(arr)+1)
		result[0] = params[1]
		copy(result[1:], arr)
		return result, nil
	}))

	r.Register(expr.Function("slice", func(params ...any) (any, error) {
		arr, ok := params[0].([]any)
		if !ok {
			return nil, fmt.Errorf("slice: first argument must be an array")
		}
		startF, ok := toFloat64(params[1])
		if !ok {
			return nil, fmt.Errorf("slice: second argument must be a number")
		}
		start := int(startF)
		if start < 0 {
			start = 0
		}
		if start >= len(arr) {
			return []any{}, nil
		}
		end := len(arr)
		if len(params) > 2 {
			if endF, ok := toFloat64(params[2]); ok {
				end = int(endF)
			}
		}
		if end > len(arr) {
			end = len(arr)
		}
		if end < start {
			return []any{}, nil
		}
		result := make([]any, end-start)
		copy(result, arr[start:end])
		return result, nil
	}))

	r.Register(expr.Function("chunk", func(params ...any) (any, error) {
		arr, ok := params[0].([]any)
		if !ok {
			return nil, fmt.Errorf("chunk: first argument must be an array")
		}
		sizeF, ok := toFloat64(params[1])
		if !ok {
			return nil, fmt.Errorf("chunk: second argument must be a number")
		}
		size := int(sizeF)
		if size <= 0 {
			return nil, fmt.Errorf("chunk: size must be > 0")
		}
		if len(arr) == 0 {
			return []any{}, nil
		}
		var chunks []any
		for i := 0; i < len(arr); i += size {
			end := i + size
			if end > len(arr) {
				end = len(arr)
			}
			chunk := make([]any, end-i)
			copy(chunk, arr[i:end])
			chunks = append(chunks, chunk)
		}
		return chunks, nil
	}))

	r.Register(expr.Function("zip", func(params ...any) (any, error) {
		arr1, ok1 := params[0].([]any)
		arr2, ok2 := params[1].([]any)
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("zip: both arguments must be arrays")
		}
		length := len(arr1)
		if len(arr2) < length {
			length = len(arr2)
		}
		result := make([]any, length)
		for i := 0; i < length; i++ {
			result[i] = []any{arr1[i], arr2[i]}
		}
		return result, nil
	}))
}
