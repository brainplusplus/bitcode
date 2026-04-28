package stdlib

import (
	"fmt"
	"time"

	"github.com/expr-lang/expr"
)

// RegisterDateTime registers date/time helper functions.
func RegisterDateTime(r *Registry) {
	r.Register(expr.Function("formatDate", func(params ...any) (any, error) {
		dt, err := toTime(params[0])
		if err != nil {
			return nil, fmt.Errorf("formatDate: first argument must be a time value: %s", err)
		}
		format, ok := params[1].(string)
		if !ok {
			return nil, fmt.Errorf("formatDate: second argument must be a format string")
		}
		return dt.Format(format), nil
	}))

	r.Register(expr.Function("addDuration", func(params ...any) (any, error) {
		dt, err := toTime(params[0])
		if err != nil {
			return nil, fmt.Errorf("addDuration: first argument must be a time value: %s", err)
		}
		durStr, ok := params[1].(string)
		if !ok {
			return nil, fmt.Errorf("addDuration: second argument must be a duration string (e.g. '1h30m')")
		}
		dur, err := time.ParseDuration(durStr)
		if err != nil {
			return nil, fmt.Errorf("addDuration: invalid duration '%s': %s", durStr, err)
		}
		return dt.Add(dur), nil
	}))

	r.Register(expr.Function("diffDates", func(params ...any) (any, error) {
		a, err := toTime(params[0])
		if err != nil {
			return nil, fmt.Errorf("diffDates: first argument must be a time value: %s", err)
		}
		b, err := toTime(params[1])
		if err != nil {
			return nil, fmt.Errorf("diffDates: second argument must be a time value: %s", err)
		}
		return a.Sub(b).String(), nil
	}))
}

func toTime(v any) (time.Time, error) {
	switch t := v.(type) {
	case time.Time:
		return t, nil
	case string:
		formats := []string{
			time.RFC3339,
			"2006-01-02T15:04:05",
			"2006-01-02",
		}
		for _, f := range formats {
			if parsed, err := time.Parse(f, t); err == nil {
				return parsed, nil
			}
		}
		return time.Time{}, fmt.Errorf("cannot parse '%s' as date", t)
	default:
		return time.Time{}, fmt.Errorf("expected time or string, got %T", v)
	}
}
