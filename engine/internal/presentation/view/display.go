package view

import (
	"fmt"
	"strings"

	"github.com/bitcode-framework/bitcode/internal/compiler/parser"
	"github.com/bitcode-framework/bitcode/internal/runtime/format"
)

func ResolveDisplayLabel(
	targetModelDef *parser.ModelDefinition,
	fieldDef *parser.FieldDefinition,
	record map[string]any,
	formatEngine *format.Engine,
) string {
	tf := ""
	if fieldDef != nil && fieldDef.DisplayField != "" {
		tf = fieldDef.DisplayField
	} else if targetModelDef != nil && targetModelDef.TitleField != "" {
		tf = targetModelDef.TitleField
	}

	if tf == "" {
		if id, ok := record["id"]; ok {
			return fmt.Sprintf("%v", id)
		}
		return ""
	}

	if strings.Contains(tf, "{") && formatEngine != nil {
		ctx := &format.FormatContext{
			Data: record,
		}
		result, err := formatEngine.Resolve(tf, ctx, "", "", "", 0)
		if err == nil && result != "" {
			return result
		}
	}

	if val, ok := record[tf]; ok {
		return fmt.Sprintf("%v", val)
	}

	if id, ok := record["id"]; ok {
		return fmt.Sprintf("%v", id)
	}
	return ""
}
