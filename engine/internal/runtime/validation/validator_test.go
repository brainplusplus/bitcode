package validation

import (
	"testing"

	"github.com/bitcode-framework/bitcode/internal/compiler/parser"
)

func TestValidateCreate_Required(t *testing.T) {
	v := NewValidator()
	model := &parser.ModelDefinition{
		Name: "test",
		Fields: map[string]parser.FieldDefinition{
			"name": {Type: parser.FieldString, Label: "Name", Validation: &parser.FieldValidation{Required: true}},
		},
	}

	errs := v.ValidateCreate(model, map[string]any{}, "en")
	if !errs.HasErrors() {
		t.Fatal("expected validation error for missing required field")
	}
	if !errs.HasFieldErrors("name") {
		t.Fatal("expected error on 'name' field")
	}

	errs = v.ValidateCreate(model, map[string]any{"name": "John"}, "en")
	if errs.HasErrors() {
		t.Fatalf("unexpected errors: %v", errs.Errors)
	}
}

func TestValidateCreate_Email(t *testing.T) {
	v := NewValidator()
	model := &parser.ModelDefinition{
		Name: "test",
		Fields: map[string]parser.FieldDefinition{
			"email": {Type: parser.FieldEmail, Label: "Email", Validation: &parser.FieldValidation{Email: true}},
		},
	}

	errs := v.ValidateCreate(model, map[string]any{"email": "invalid"}, "en")
	if !errs.HasFieldErrors("email") {
		t.Fatal("expected email validation error")
	}

	errs = v.ValidateCreate(model, map[string]any{"email": "test@example.com"}, "en")
	if errs.HasErrors() {
		t.Fatalf("unexpected errors: %v", errs.Errors)
	}
}

func TestValidateCreate_MinMaxLength(t *testing.T) {
	minLen := 3
	maxLen := 10
	v := NewValidator()
	model := &parser.ModelDefinition{
		Name: "test",
		Fields: map[string]parser.FieldDefinition{
			"code": {Type: parser.FieldString, Label: "Code", Validation: &parser.FieldValidation{
				MinLength: &minLen,
				MaxLength: &maxLen,
			}},
		},
	}

	errs := v.ValidateCreate(model, map[string]any{"code": "ab"}, "en")
	if !errs.HasFieldErrors("code") {
		t.Fatal("expected min_length error")
	}

	errs = v.ValidateCreate(model, map[string]any{"code": "abcdefghijk"}, "en")
	if !errs.HasFieldErrors("code") {
		t.Fatal("expected max_length error")
	}

	errs = v.ValidateCreate(model, map[string]any{"code": "abcde"}, "en")
	if errs.HasErrors() {
		t.Fatalf("unexpected errors: %v", errs.Errors)
	}
}

func TestValidateCreate_Regex(t *testing.T) {
	v := NewValidator()
	model := &parser.ModelDefinition{
		Name: "test",
		Fields: map[string]parser.FieldDefinition{
			"code": {Type: parser.FieldString, Label: "Code", Validation: &parser.FieldValidation{
				Regex:        "^[A-Z]{2}-[0-9]{4}$",
				RegexMessage: "Must be XX-0000 format",
			}},
		},
	}

	errs := v.ValidateCreate(model, map[string]any{"code": "abc"}, "en")
	if !errs.HasFieldErrors("code") {
		t.Fatal("expected regex error")
	}
	if errs.Errors["code"][0].Message != "Must be XX-0000 format" {
		t.Fatalf("expected custom regex message, got: %s", errs.Errors["code"][0].Message)
	}

	errs = v.ValidateCreate(model, map[string]any{"code": "AB-1234"}, "en")
	if errs.HasErrors() {
		t.Fatalf("unexpected errors: %v", errs.Errors)
	}
}

func TestValidateCreate_RequiredIf(t *testing.T) {
	v := NewValidator()
	model := &parser.ModelDefinition{
		Name: "test",
		Fields: map[string]parser.FieldDefinition{
			"status":           {Type: parser.FieldSelection, Label: "Status", Options: []string{"active", "rejected"}},
			"rejection_reason": {Type: parser.FieldText, Label: "Rejection Reason", Validation: &parser.FieldValidation{
				RequiredIf: map[string]any{"status": "rejected"},
			}},
		},
	}

	errs := v.ValidateCreate(model, map[string]any{"status": "rejected"}, "en")
	if !errs.HasFieldErrors("rejection_reason") {
		t.Fatal("expected required_if error when status=rejected")
	}

	errs = v.ValidateCreate(model, map[string]any{"status": "active"}, "en")
	if errs.HasFieldErrors("rejection_reason") {
		t.Fatal("should not require rejection_reason when status=active")
	}

	errs = v.ValidateCreate(model, map[string]any{"status": "rejected", "rejection_reason": "Bad fit"}, "en")
	if errs.HasErrors() {
		t.Fatalf("unexpected errors: %v", errs.Errors)
	}
}

func TestValidateCreate_RequiredOnCreate(t *testing.T) {
	v := NewValidator()
	model := &parser.ModelDefinition{
		Name: "test",
		Fields: map[string]parser.FieldDefinition{
			"password": {Type: parser.FieldPassword, Label: "Password", Validation: &parser.FieldValidation{
				Required: map[string]any{"on": "create"},
			}},
		},
	}

	errs := v.ValidateCreate(model, map[string]any{}, "en")
	if !errs.HasFieldErrors("password") {
		t.Fatal("expected required error on create")
	}

	errs = v.ValidateUpdate(model, map[string]any{"password": ""}, map[string]any{"password": ""}, "en")
	if errs.HasFieldErrors("password") {
		t.Fatal("should not require password on update")
	}
}

func TestValidateCreate_Immutable(t *testing.T) {
	v := NewValidator()
	model := &parser.ModelDefinition{
		Name: "test",
		Fields: map[string]parser.FieldDefinition{
			"username": {Type: parser.FieldString, Label: "Username", Validation: &parser.FieldValidation{
				Immutable: true,
			}},
		},
	}

	mergedData := map[string]any{
		"username": "new_name",
		"__old":    map[string]any{"username": "old_name"},
	}
	changes := map[string]any{"username": "new_name"}

	errs := v.ValidateUpdate(model, mergedData, changes, "en")
	if !errs.HasFieldErrors("username") {
		t.Fatal("expected immutable error")
	}
}

func TestValidateCreate_DateAfter(t *testing.T) {
	v := NewValidator()
	model := &parser.ModelDefinition{
		Name: "test",
		Fields: map[string]parser.FieldDefinition{
			"start_date": {Type: parser.FieldDate, Label: "Start Date"},
			"end_date":   {Type: parser.FieldDate, Label: "End Date", Validation: &parser.FieldValidation{
				DateAfter: "start_date",
			}},
		},
	}

	errs := v.ValidateCreate(model, map[string]any{
		"start_date": "2026-01-15",
		"end_date":   "2026-01-10",
	}, "en")
	if !errs.HasFieldErrors("end_date") {
		t.Fatal("expected date_after error")
	}

	errs = v.ValidateCreate(model, map[string]any{
		"start_date": "2026-01-10",
		"end_date":   "2026-01-15",
	}, "en")
	if errs.HasErrors() {
		t.Fatalf("unexpected errors: %v", errs.Errors)
	}
}

func TestValidateCreate_InNotIn(t *testing.T) {
	v := NewValidator()
	model := &parser.ModelDefinition{
		Name: "test",
		Fields: map[string]parser.FieldDefinition{
			"category": {Type: parser.FieldString, Label: "Category", Validation: &parser.FieldValidation{
				In: []any{"A", "B", "C"},
			}},
			"code": {Type: parser.FieldString, Label: "Code", Validation: &parser.FieldValidation{
				NotIn: []any{"ADMIN", "ROOT"},
			}},
		},
	}

	errs := v.ValidateCreate(model, map[string]any{"category": "D"}, "en")
	if !errs.HasFieldErrors("category") {
		t.Fatal("expected 'in' validation error")
	}

	errs = v.ValidateCreate(model, map[string]any{"code": "ADMIN"}, "en")
	if !errs.HasFieldErrors("code") {
		t.Fatal("expected 'not_in' validation error")
	}

	errs = v.ValidateCreate(model, map[string]any{"category": "A", "code": "USER"}, "en")
	if errs.HasErrors() {
		t.Fatalf("unexpected errors: %v", errs.Errors)
	}
}

func TestValidateCreate_Phone(t *testing.T) {
	v := NewValidator()
	model := &parser.ModelDefinition{
		Name: "test",
		Fields: map[string]parser.FieldDefinition{
			"phone": {Type: parser.FieldString, Label: "Phone", Validation: &parser.FieldValidation{Phone: true}},
		},
	}

	errs := v.ValidateCreate(model, map[string]any{"phone": "abc"}, "en")
	if !errs.HasFieldErrors("phone") {
		t.Fatal("expected phone validation error")
	}

	errs = v.ValidateCreate(model, map[string]any{"phone": "+62812345678"}, "en")
	if errs.HasErrors() {
		t.Fatalf("unexpected errors: %v", errs.Errors)
	}
}

func TestValidateCreate_Between(t *testing.T) {
	v := NewValidator()
	model := &parser.ModelDefinition{
		Name: "test",
		Fields: map[string]parser.FieldDefinition{
			"score": {Type: parser.FieldDecimal, Label: "Score", Validation: &parser.FieldValidation{
				Between: []float64{0, 100},
			}},
		},
	}

	errs := v.ValidateCreate(model, map[string]any{"score": float64(150)}, "en")
	if !errs.HasFieldErrors("score") {
		t.Fatal("expected between error")
	}

	errs = v.ValidateCreate(model, map[string]any{"score": float64(50)}, "en")
	if errs.HasErrors() {
		t.Fatalf("unexpected errors: %v", errs.Errors)
	}
}

func TestValidateCreate_AutoMapRequired(t *testing.T) {
	v := NewValidator()
	model := &parser.ModelDefinition{
		Name: "test",
		Fields: map[string]parser.FieldDefinition{
			"name": {Type: parser.FieldString, Label: "Name", Required: true},
		},
	}

	errs := v.ValidateCreate(model, map[string]any{}, "en")
	if !errs.HasFieldErrors("name") {
		t.Fatal("expected auto-mapped required error")
	}
}

func TestValidateCreate_AutoMapMaxLength(t *testing.T) {
	v := NewValidator()
	model := &parser.ModelDefinition{
		Name: "test",
		Fields: map[string]parser.FieldDefinition{
			"name": {Type: parser.FieldString, Label: "Name", Max: 10},
		},
	}

	errs := v.ValidateCreate(model, map[string]any{"name": "this is a very long name"}, "en")
	if !errs.HasFieldErrors("name") {
		t.Fatal("expected auto-mapped max_length error")
	}
}

func TestValidateCreate_SkipComputedFields(t *testing.T) {
	v := NewValidator()
	model := &parser.ModelDefinition{
		Name: "test",
		Fields: map[string]parser.FieldDefinition{
			"total": {Type: parser.FieldCurrency, Label: "Total", Computed: "sum(lines.subtotal)", Validation: &parser.FieldValidation{Required: true}},
		},
	}

	errs := v.ValidateCreate(model, map[string]any{}, "en")
	if errs.HasErrors() {
		t.Fatal("should skip validation for computed fields")
	}
}

func TestValidateCreate_ModelLevelValidator(t *testing.T) {
	v := NewValidator()
	model := &parser.ModelDefinition{
		Name: "test",
		Fields: map[string]parser.FieldDefinition{
			"start_date": {Type: parser.FieldDate, Label: "Start Date"},
			"end_date":   {Type: parser.FieldDate, Label: "End Date"},
		},
		Validators: []parser.ModelValidator{
			{
				Name:       "date_range",
				Expression: "end_date >= start_date",
				Message:    "End date must be after start date",
			},
		},
	}

	errs := v.ValidateCreate(model, map[string]any{
		"start_date": "2026-01-15",
		"end_date":   "2026-01-10",
	}, "en")
	if !errs.HasFieldErrors("_model") {
		t.Fatal("expected model-level validation error")
	}
}

func TestValidateCreate_RequiredWithout(t *testing.T) {
	v := NewValidator()
	model := &parser.ModelDefinition{
		Name: "test",
		Fields: map[string]parser.FieldDefinition{
			"email": {Type: parser.FieldEmail, Label: "Email"},
			"phone": {Type: parser.FieldString, Label: "Phone", Validation: &parser.FieldValidation{
				RequiredWithout: []string{"email"},
			}},
		},
	}

	errs := v.ValidateCreate(model, map[string]any{}, "en")
	if !errs.HasFieldErrors("phone") {
		t.Fatal("expected required_without error when email is absent")
	}

	errs = v.ValidateCreate(model, map[string]any{"email": "test@test.com"}, "en")
	if errs.HasFieldErrors("phone") {
		t.Fatal("should not require phone when email is present")
	}
}

func TestValidateCreate_Confirmed(t *testing.T) {
	v := NewValidator()
	model := &parser.ModelDefinition{
		Name: "test",
		Fields: map[string]parser.FieldDefinition{
			"password":         {Type: parser.FieldPassword, Label: "Password"},
			"password_confirm": {Type: parser.FieldPassword, Label: "Confirm Password", Validation: &parser.FieldValidation{
				Confirmed: "password",
			}},
		},
	}

	errs := v.ValidateCreate(model, map[string]any{
		"password":         "secret123",
		"password_confirm": "different",
	}, "en")
	if !errs.HasFieldErrors("password_confirm") {
		t.Fatal("expected confirmed error")
	}

	errs = v.ValidateCreate(model, map[string]any{
		"password":         "secret123",
		"password_confirm": "secret123",
	}, "en")
	if errs.HasErrors() {
		t.Fatalf("unexpected errors: %v", errs.Errors)
	}
}

func TestValidateUpdate_PartialMerge(t *testing.T) {
	v := NewValidator()
	model := &parser.ModelDefinition{
		Name: "test",
		Fields: map[string]parser.FieldDefinition{
			"name":  {Type: parser.FieldString, Label: "Name", Validation: &parser.FieldValidation{Required: true}},
			"email": {Type: parser.FieldEmail, Label: "Email", Validation: &parser.FieldValidation{Required: true, Email: true}},
		},
	}

	mergedData := map[string]any{
		"name":  "John",
		"email": "john@test.com",
		"phone": "08123",
		"__old": map[string]any{"name": "John", "email": "john@test.com"},
	}
	changes := map[string]any{"phone": "08123"}

	errs := v.ValidateUpdate(model, mergedData, changes, "en")
	if errs.HasErrors() {
		t.Fatalf("should not validate unchanged fields: %v", errs.Errors)
	}
}

func TestValidateCreate_WhenCondition(t *testing.T) {
	v := NewValidator()
	model := &parser.ModelDefinition{
		Name: "test",
		Fields: map[string]parser.FieldDefinition{
			"country": {Type: parser.FieldString, Label: "Country"},
			"tax_id": {Type: parser.FieldString, Label: "Tax ID", Validation: &parser.FieldValidation{
				Regex: "^\\d{5}$",
				When:  map[string]any{"country": "ID"},
			}},
		},
	}

	errs := v.ValidateCreate(model, map[string]any{"country": "US", "tax_id": "abc"}, "en")
	if errs.HasFieldErrors("tax_id") {
		t.Fatal("should skip regex validation when country != ID")
	}

	errs = v.ValidateCreate(model, map[string]any{"country": "ID", "tax_id": "abc"}, "en")
	if !errs.HasFieldErrors("tax_id") {
		t.Fatal("expected regex error when country == ID")
	}

	errs = v.ValidateCreate(model, map[string]any{"country": "ID", "tax_id": "12345"}, "en")
	if errs.HasErrors() {
		t.Fatalf("unexpected errors: %v", errs.Errors)
	}
}
