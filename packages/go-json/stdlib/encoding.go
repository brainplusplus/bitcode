package stdlib

import (
	"fmt"
	"net/url"

	"github.com/expr-lang/expr"
)

// RegisterEncoding registers URL encoding/decoding functions.
func RegisterEncoding(r *Registry) {
	r.Register(expr.Function("urlEncode", func(params ...any) (any, error) {
		s, ok := params[0].(string)
		if !ok {
			return nil, fmt.Errorf("urlEncode: argument must be a string")
		}
		return url.QueryEscape(s), nil
	}))

	r.Register(expr.Function("urlDecode", func(params ...any) (any, error) {
		s, ok := params[0].(string)
		if !ok {
			return nil, fmt.Errorf("urlDecode: argument must be a string")
		}
		decoded, err := url.QueryUnescape(s)
		if err != nil {
			return nil, fmt.Errorf("urlDecode: %s", err)
		}
		return decoded, nil
	}))
}
