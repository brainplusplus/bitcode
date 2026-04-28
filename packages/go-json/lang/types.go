package lang

import (
	"strings"
)

// InferType determines the go-json type string from a Go runtime value.
// JSON's float64-for-all-numbers is handled: whole numbers become "int".
func InferType(value any) string {
	if value == nil {
		return "nil"
	}

	switch v := value.(type) {
	case bool:
		return "bool"
	case int:
		return "int"
	case int64:
		return "int"
	case float64:
		// JSON unmarshals all numbers as float64.
		// Detect integers: if float64(int64(f)) == f, it's an int.
		if v == float64(int64(v)) && v >= -1<<53 && v <= 1<<53 {
			return "int"
		}
		return "float"
	case float32:
		return "float"
	case string:
		return "string"
	case []any:
		return "[]any"
	case map[string]any:
		return "map"
	default:
		return "any"
	}
}

// IsNullable returns true if the type string represents a nullable type (?T).
func IsNullable(typ string) bool {
	return strings.HasPrefix(typ, "?")
}

// BaseType strips the nullable prefix from a type string.
// "?string" → "string", "int" → "int"
func BaseType(typ string) string {
	return strings.TrimPrefix(typ, "?")
}

// TypesCompatible checks if a new value type can be assigned to a variable
// with the given existing type. Rules:
//   - "any" accepts all types
//   - "?T" accepts "nil" and base type T
//   - same base type required otherwise
//   - empty type ("") treated as "any"
func TypesCompatible(existingType, newType string) bool {
	if existingType == "" || existingType == "any" {
		return true
	}
	if newType == "" || newType == "any" {
		return true
	}

	// Nullable type accepts nil.
	if IsNullable(existingType) && newType == "nil" {
		return true
	}

	// Non-nullable rejects nil.
	if !IsNullable(existingType) && newType == "nil" {
		return false
	}

	// Compare base types.
	return BaseType(existingType) == BaseType(newType)
}

// TypeFromJSON maps JSON type declaration strings to internal type names.
func TypeFromJSON(jsonType string) string {
	switch jsonType {
	case "string":
		return "string"
	case "int", "integer":
		return "int"
	case "float", "number":
		return "float"
	case "bool", "boolean":
		return "bool"
	case "map", "object":
		return "map"
	case "any":
		return "any"
	default:
		// Array types like "[]string", nullable like "?int", or struct names.
		return jsonType
	}
}
