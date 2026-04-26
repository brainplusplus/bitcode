package validation

import (
	"strings"
)

func evaluateConditionMap(cond map[string]any, data map[string]any) bool {
	for field, expected := range cond {
		val := data[field]
		switch exp := expected.(type) {
		case []any:
			if !anyInList(val, exp) {
				return false
			}
		case []string:
			list := make([]any, len(exp))
			for i, s := range exp {
				list[i] = s
			}
			if !anyInList(val, list) {
				return false
			}
		default:
			if !anyEquals(val, exp) {
				return false
			}
		}
	}
	return true
}

func evaluateWhen(when any, data map[string]any) bool {
	if when == nil {
		return true
	}
	switch w := when.(type) {
	case map[string]any:
		return evaluateConditionMap(w, data)
	case string:
		return evaluateSimpleExpression(w, data)
	case bool:
		return w
	}
	return true
}

func evaluateSimpleExpression(expr string, data map[string]any) bool {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return true
	}

	if strings.Contains(expr, " && ") {
		parts := strings.Split(expr, " && ")
		for _, part := range parts {
			if !evaluateSimpleExpression(strings.TrimSpace(part), data) {
				return false
			}
		}
		return true
	}

	if strings.Contains(expr, " || ") {
		parts := strings.Split(expr, " || ")
		for _, part := range parts {
			if evaluateSimpleExpression(strings.TrimSpace(part), data) {
				return true
			}
		}
		return false
	}

	if strings.HasPrefix(expr, "!") {
		inner := strings.TrimSpace(expr[1:])
		return !evaluateSimpleExpression(inner, data)
	}

	if strings.Contains(expr, " != ") {
		parts := strings.SplitN(expr, " != ", 2)
		left := resolveExprValue(strings.TrimSpace(parts[0]), data)
		right := resolveExprValue(strings.TrimSpace(parts[1]), data)
		return toString(left) != toString(right)
	}

	if strings.Contains(expr, " == ") {
		parts := strings.SplitN(expr, " == ", 2)
		left := resolveExprValue(strings.TrimSpace(parts[0]), data)
		right := resolveExprValue(strings.TrimSpace(parts[1]), data)
		return toString(left) == toString(right)
	}

	if strings.Contains(expr, " >= ") {
		parts := strings.SplitN(expr, " >= ", 2)
		left, lok := toFloat(resolveExprValue(strings.TrimSpace(parts[0]), data))
		right, rok := toFloat(resolveExprValue(strings.TrimSpace(parts[1]), data))
		if lok && rok {
			return left >= right
		}
		return false
	}

	if strings.Contains(expr, " <= ") {
		parts := strings.SplitN(expr, " <= ", 2)
		left, lok := toFloat(resolveExprValue(strings.TrimSpace(parts[0]), data))
		right, rok := toFloat(resolveExprValue(strings.TrimSpace(parts[1]), data))
		if lok && rok {
			return left <= right
		}
		return false
	}

	if strings.Contains(expr, " > ") {
		parts := strings.SplitN(expr, " > ", 2)
		left, lok := toFloat(resolveExprValue(strings.TrimSpace(parts[0]), data))
		right, rok := toFloat(resolveExprValue(strings.TrimSpace(parts[1]), data))
		if lok && rok {
			return left > right
		}
		return false
	}

	if strings.Contains(expr, " < ") {
		parts := strings.SplitN(expr, " < ", 2)
		left, lok := toFloat(resolveExprValue(strings.TrimSpace(parts[0]), data))
		right, rok := toFloat(resolveExprValue(strings.TrimSpace(parts[1]), data))
		if lok && rok {
			return left < right
		}
		return false
	}

	val := data[expr]
	return !isEmpty(val)
}

func resolveExprValue(token string, data map[string]any) any {
	if (strings.HasPrefix(token, "'") && strings.HasSuffix(token, "'")) ||
		(strings.HasPrefix(token, "\"") && strings.HasSuffix(token, "\"")) {
		return token[1 : len(token)-1]
	}

	if f, ok := toFloat(token); ok {
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
		fieldName := token[4:]
		if oldData, ok := data["__old"].(map[string]any); ok {
			return oldData[fieldName]
		}
		return nil
	}

	if strings.HasPrefix(token, "session.") {
		fieldName := token[8:]
		if session, ok := data["__session"].(map[string]any); ok {
			return session[fieldName]
		}
		return nil
	}

	return data[token]
}

func isFieldPresent(data map[string]any, field string) bool {
	val, exists := data[field]
	return exists && !isEmpty(val)
}

func checkRequiredIf(cond map[string]any, data map[string]any) bool {
	return evaluateConditionMap(cond, data)
}

func checkRequiredUnless(cond map[string]any, data map[string]any) bool {
	return !evaluateConditionMap(cond, data)
}

func checkRequiredWith(fields []string, data map[string]any) bool {
	for _, f := range fields {
		if isFieldPresent(data, f) {
			return true
		}
	}
	return false
}

func checkRequiredWithAll(fields []string, data map[string]any) bool {
	for _, f := range fields {
		if !isFieldPresent(data, f) {
			return false
		}
	}
	return true
}

func checkRequiredWithout(fields []string, data map[string]any) bool {
	for _, f := range fields {
		if !isFieldPresent(data, f) {
			return true
		}
	}
	return false
}

func checkRequiredWithoutAll(fields []string, data map[string]any) bool {
	for _, f := range fields {
		if isFieldPresent(data, f) {
			return false
		}
	}
	return true
}

func checkExcludeIf(cond map[string]any, data map[string]any) bool {
	return evaluateConditionMap(cond, data)
}

func checkExcludeUnless(cond map[string]any, data map[string]any) bool {
	return !evaluateConditionMap(cond, data)
}
