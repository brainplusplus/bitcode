package validation

import "fmt"

type FieldError struct {
	Rule    string         `json:"rule"`
	Message string         `json:"message"`
	Params  map[string]any `json:"params,omitempty"`
}

type ValidationErrors struct {
	Errors map[string][]FieldError `json:"errors"`
}

func NewValidationErrors() *ValidationErrors {
	return &ValidationErrors{Errors: make(map[string][]FieldError)}
}

func (ve *ValidationErrors) Add(field, rule, message string, params map[string]any) {
	ve.Errors[field] = append(ve.Errors[field], FieldError{
		Rule:    rule,
		Message: message,
		Params:  params,
	})
}

func (ve *ValidationErrors) AddModel(rule, message string) {
	ve.Add("_model", rule, message, nil)
}

func (ve *ValidationErrors) HasErrors() bool {
	return len(ve.Errors) > 0
}

func (ve *ValidationErrors) HasFieldErrors(field string) bool {
	return len(ve.Errors[field]) > 0
}

func (ve *ValidationErrors) Merge(other *ValidationErrors) {
	if other == nil {
		return
	}
	for field, errs := range other.Errors {
		ve.Errors[field] = append(ve.Errors[field], errs...)
	}
}

func (ve *ValidationErrors) Error() string {
	count := 0
	for _, errs := range ve.Errors {
		count += len(errs)
	}
	return fmt.Sprintf("validation failed: %d error(s)", count)
}
