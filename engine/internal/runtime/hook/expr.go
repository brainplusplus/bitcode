package hook

import (
	"fmt"
	"strconv"
	"strings"
)

func evaluateSimpleExpr(expr string, data map[string]any) bool {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return true
	}

	if strings.Contains(expr, " && ") {
		parts := strings.Split(expr, " && ")
		for _, part := range parts {
			if !evaluateSimpleExpr(strings.TrimSpace(part), data) {
				return false
			}
		}
		return true
	}

	if strings.Contains(expr, " || ") {
		parts := strings.Split(expr, " || ")
		for _, part := range parts {
			if evaluateSimpleExpr(strings.TrimSpace(part), data) {
				return true
			}
		}
		return false
	}

	if strings.HasPrefix(expr, "!") {
		return !evaluateSimpleExpr(strings.TrimSpace(expr[1:]), data)
	}

	if strings.Contains(expr, " != ") {
		parts := strings.SplitN(expr, " != ", 2)
		return exprToString(resolveToken(strings.TrimSpace(parts[0]), data)) != exprToString(resolveToken(strings.TrimSpace(parts[1]), data))
	}

	if strings.Contains(expr, " == ") {
		parts := strings.SplitN(expr, " == ", 2)
		return exprToString(resolveToken(strings.TrimSpace(parts[0]), data)) == exprToString(resolveToken(strings.TrimSpace(parts[1]), data))
	}

	if strings.Contains(expr, " >= ") {
		parts := strings.SplitN(expr, " >= ", 2)
		l, lok := exprToFloat(resolveToken(strings.TrimSpace(parts[0]), data))
		r, rok := exprToFloat(resolveToken(strings.TrimSpace(parts[1]), data))
		return lok && rok && l >= r
	}

	if strings.Contains(expr, " <= ") {
		parts := strings.SplitN(expr, " <= ", 2)
		l, lok := exprToFloat(resolveToken(strings.TrimSpace(parts[0]), data))
		r, rok := exprToFloat(resolveToken(strings.TrimSpace(parts[1]), data))
		return lok && rok && l <= r
	}

	if strings.Contains(expr, " > ") {
		parts := strings.SplitN(expr, " > ", 2)
		l, lok := exprToFloat(resolveToken(strings.TrimSpace(parts[0]), data))
		r, rok := exprToFloat(resolveToken(strings.TrimSpace(parts[1]), data))
		return lok && rok && l > r
	}

	if strings.Contains(expr, " < ") {
		parts := strings.SplitN(expr, " < ", 2)
		l, lok := exprToFloat(resolveToken(strings.TrimSpace(parts[0]), data))
		r, rok := exprToFloat(resolveToken(strings.TrimSpace(parts[1]), data))
		return lok && rok && l < r
	}

	val := data[expr]
	return val != nil && val != "" && val != false && val != 0 && val != float64(0)
}

func resolveToken(token string, data map[string]any) any {
	if (strings.HasPrefix(token, "'") && strings.HasSuffix(token, "'")) ||
		(strings.HasPrefix(token, "\"") && strings.HasSuffix(token, "\"")) {
		return token[1 : len(token)-1]
	}
	if f, err := strconv.ParseFloat(token, 64); err == nil {
		return f
	}
	if token == "true" {
		return true
	}
	if token == "false" {
		return false
	}
	if token == "nil" || token == "null" {
		return nil
	}
	if strings.HasPrefix(token, "old.") {
		if old, ok := data["__old"].(map[string]any); ok {
			return old[token[4:]]
		}
		return nil
	}
	if strings.HasPrefix(token, "session.") {
		if sess, ok := data["__session"].(map[string]any); ok {
			return sess[token[8:]]
		}
		return nil
	}
	return data[token]
}

func exprToString(val any) string {
	if val == nil {
		return ""
	}
	return fmt.Sprintf("%v", val)
}

func exprToFloat(val any) (float64, bool) {
	switch v := val.(type) {
	case float64:
		return v, true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case string:
		f, err := strconv.ParseFloat(v, 64)
		return f, err == nil
	}
	return 0, false
}
