package api

import (
	"strings"

	"github.com/bitcode-framework/bitcode/internal/compiler/parser"
	"github.com/bitcode-framework/bitcode/internal/infrastructure/persistence"
)

func filterResponseFields(record map[string]any, modelDef *parser.ModelDefinition, userGroups []string, perms *persistence.ModelPermissions) map[string]any {
	if modelDef == nil {
		return record
	}

	result := make(map[string]any, len(record))
	for key, value := range record {
		fieldDef, exists := modelDef.Fields[key]
		if !exists {
			result[key] = value
			continue
		}

		if len(fieldDef.Groups) > 0 && !userInFieldGroups(userGroups, fieldDef.Groups) {
			continue
		}

		if fieldDef.Mask && perms != nil && !perms.CanMask {
			result[key] = maskValue(value, fieldDef.MaskLength)
			continue
		}

		result[key] = value
	}

	return result
}

func filterResponseList(records []map[string]any, modelDef *parser.ModelDefinition, userGroups []string, perms *persistence.ModelPermissions) []map[string]any {
	if modelDef == nil {
		return records
	}
	result := make([]map[string]any, len(records))
	for i, record := range records {
		result[i] = filterResponseFields(record, modelDef, userGroups, perms)
	}
	return result
}

func userInFieldGroups(userGroups []string, fieldGroups []string) bool {
	for _, fg := range fieldGroups {
		for _, ug := range userGroups {
			if fg == ug {
				return true
			}
		}
	}
	return false
}

func maskValue(value any, maskLength int) string {
	if value == nil {
		return ""
	}
	s, ok := value.(string)
	if !ok {
		return "****"
	}
	if len(s) == 0 {
		return ""
	}
	if maskLength <= 0 {
		maskLength = 4
	}
	if maskLength >= len(s) {
		return s
	}
	masked := strings.Repeat("*", len(s)-maskLength) + s[len(s)-maskLength:]
	return masked
}
