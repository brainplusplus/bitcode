package validation

import (
	"context"

	"github.com/bitcode-framework/bitcode/internal/compiler/parser"
)

type ValidatorAdapter struct {
	validator *Validator
}

func NewValidatorAdapter(v *Validator) *ValidatorAdapter {
	return &ValidatorAdapter{validator: v}
}

func (a *ValidatorAdapter) ValidateCreate(modelDef *parser.ModelDefinition, data map[string]any, locale string) error {
	a.validator.SetCurrentRecordID("")
	errs := a.validator.ValidateCreate(modelDef, data, locale)
	if errs.HasErrors() {
		return errs
	}
	return nil
}

func (a *ValidatorAdapter) ValidateUpdate(modelDef *parser.ModelDefinition, mergedData map[string]any, changes map[string]any, locale string) error {
	recordID := ""
	if old, ok := mergedData["__old"].(map[string]any); ok {
		if id, ok := old["id"].(string); ok {
			recordID = id
		}
	}
	a.validator.SetCurrentRecordID(recordID)
	errs := a.validator.ValidateUpdate(modelDef, mergedData, changes, locale)
	if errs.HasErrors() {
		return errs
	}
	return nil
}

func (a *ValidatorAdapter) SetContext(ctx context.Context) {
	a.validator.SetContext(ctx)
}
