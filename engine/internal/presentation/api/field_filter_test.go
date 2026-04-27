package api

import (
	"testing"

	"github.com/bitcode-framework/bitcode/internal/compiler/parser"
	"github.com/bitcode-framework/bitcode/internal/infrastructure/persistence"
)

func TestMaskValue(t *testing.T) {
	tests := []struct {
		input      any
		maskLength int
		expected   string
	}{
		{"08123456789", 4, "*******6789"},
		{"08123456789", 0, "*******6789"},
		{"3201234567890001", 6, "**********890001"},
		{"abc", 4, "abc"},
		{"ab", 4, "ab"},
		{"", 4, ""},
		{nil, 4, ""},
		{12345, 4, "****"},
	}

	for _, tt := range tests {
		result := maskValue(tt.input, tt.maskLength)
		if result != tt.expected {
			t.Errorf("maskValue(%v, %d) = %q, want %q", tt.input, tt.maskLength, result, tt.expected)
		}
	}
}

func TestFilterResponseFields_FieldGroups(t *testing.T) {
	modelDef := &parser.ModelDefinition{
		Fields: map[string]parser.FieldDefinition{
			"name":   {Type: "string"},
			"email":  {Type: "email"},
			"salary": {Type: "decimal", Groups: []string{"hr.manager"}},
			"ktp":    {Type: "string", Groups: []string{"hr.manager", "hr.user"}},
		},
	}

	record := map[string]any{
		"id":     "1",
		"name":   "John",
		"email":  "john@test.com",
		"salary": 5000000,
		"ktp":    "3201234567890001",
	}

	t.Run("user in hr.manager sees all fields", func(t *testing.T) {
		result := filterResponseFields(record, modelDef, []string{"hr.manager"}, &persistence.ModelPermissions{CanMask: true})
		if _, ok := result["salary"]; !ok {
			t.Error("hr.manager should see salary")
		}
		if _, ok := result["ktp"]; !ok {
			t.Error("hr.manager should see ktp")
		}
	})

	t.Run("user in hr.user sees ktp but not salary", func(t *testing.T) {
		result := filterResponseFields(record, modelDef, []string{"hr.user"}, &persistence.ModelPermissions{})
		if _, ok := result["salary"]; ok {
			t.Error("hr.user should NOT see salary")
		}
		if _, ok := result["ktp"]; !ok {
			t.Error("hr.user should see ktp")
		}
	})

	t.Run("user with no groups sees neither", func(t *testing.T) {
		result := filterResponseFields(record, modelDef, []string{"base.user"}, &persistence.ModelPermissions{})
		if _, ok := result["salary"]; ok {
			t.Error("base.user should NOT see salary")
		}
		if _, ok := result["ktp"]; ok {
			t.Error("base.user should NOT see ktp")
		}
		if _, ok := result["name"]; !ok {
			t.Error("base.user should see name (no groups restriction)")
		}
	})
}

func TestFilterResponseFields_Masking(t *testing.T) {
	modelDef := &parser.ModelDefinition{
		Fields: map[string]parser.FieldDefinition{
			"name":  {Type: "string"},
			"phone": {Type: "string", Mask: true, MaskLength: 4},
			"ktp":   {Type: "string", Mask: true, MaskLength: 6},
		},
	}

	record := map[string]any{
		"name":  "John",
		"phone": "08123456789",
		"ktp":   "3201234567890001",
	}

	t.Run("user without can_mask sees masked values", func(t *testing.T) {
		result := filterResponseFields(record, modelDef, []string{}, &persistence.ModelPermissions{CanMask: false})
		if result["phone"] != "*******6789" {
			t.Errorf("expected masked phone, got %v", result["phone"])
		}
		if result["ktp"] != "**********890001" {
			t.Errorf("expected masked ktp, got %v", result["ktp"])
		}
		if result["name"] != "John" {
			t.Error("name should not be masked")
		}
	})

	t.Run("user with can_mask sees full values", func(t *testing.T) {
		result := filterResponseFields(record, modelDef, []string{}, &persistence.ModelPermissions{CanMask: true})
		if result["phone"] != "08123456789" {
			t.Errorf("expected full phone, got %v", result["phone"])
		}
		if result["ktp"] != "3201234567890001" {
			t.Errorf("expected full ktp, got %v", result["ktp"])
		}
	})
}

func TestFilterResponseFields_NilModelDef(t *testing.T) {
	record := map[string]any{"name": "John", "email": "john@test.com"}
	result := filterResponseFields(record, nil, nil, nil)
	if len(result) != 2 {
		t.Errorf("expected 2 fields with nil modelDef, got %d", len(result))
	}
}

func TestFilterResponseFields_UnknownFieldsPassThrough(t *testing.T) {
	modelDef := &parser.ModelDefinition{
		Fields: map[string]parser.FieldDefinition{
			"name": {Type: "string"},
		},
	}
	record := map[string]any{"name": "John", "id": "1", "created_at": "2026-01-01"}
	result := filterResponseFields(record, modelDef, nil, nil)
	if _, ok := result["id"]; !ok {
		t.Error("unknown fields (id, created_at) should pass through")
	}
	if _, ok := result["created_at"]; !ok {
		t.Error("unknown fields should pass through")
	}
}

func TestFilterResponseList(t *testing.T) {
	modelDef := &parser.ModelDefinition{
		Fields: map[string]parser.FieldDefinition{
			"name":   {Type: "string"},
			"salary": {Type: "decimal", Groups: []string{"hr.manager"}},
		},
	}
	records := []map[string]any{
		{"name": "John", "salary": 5000},
		{"name": "Jane", "salary": 6000},
	}

	result := filterResponseList(records, modelDef, []string{"base.user"}, &persistence.ModelPermissions{})
	for i, r := range result {
		if _, ok := r["salary"]; ok {
			t.Errorf("record %d: base.user should not see salary", i)
		}
		if _, ok := r["name"]; !ok {
			t.Errorf("record %d: should see name", i)
		}
	}
}
