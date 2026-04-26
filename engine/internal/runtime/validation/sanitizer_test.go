package validation

import (
	"testing"

	"github.com/bitcode-framework/bitcode/internal/compiler/parser"
)

func TestSanitize_Trim(t *testing.T) {
	s := NewSanitizer()
	model := &parser.ModelDefinition{
		Name: "test",
		Fields: map[string]parser.FieldDefinition{
			"name": {Type: parser.FieldString, Sanitize: []string{"trim"}},
		},
	}
	data := map[string]any{"name": "  hello  "}
	s.SanitizeRecord(model, data)
	if data["name"] != "hello" {
		t.Fatalf("expected 'hello', got '%v'", data["name"])
	}
}

func TestSanitize_Lowercase(t *testing.T) {
	s := NewSanitizer()
	model := &parser.ModelDefinition{
		Name: "test",
		Fields: map[string]parser.FieldDefinition{
			"email": {Type: parser.FieldEmail, Sanitize: []string{"trim", "lowercase"}},
		},
	}
	data := map[string]any{"email": "  John@Example.COM  "}
	s.SanitizeRecord(model, data)
	if data["email"] != "john@example.com" {
		t.Fatalf("expected 'john@example.com', got '%v'", data["email"])
	}
}

func TestSanitize_TitleCase(t *testing.T) {
	s := NewSanitizer()
	model := &parser.ModelDefinition{
		Name: "test",
		Fields: map[string]parser.FieldDefinition{
			"name": {Type: parser.FieldString, Sanitize: []string{"trim", "title_case"}},
		},
	}
	data := map[string]any{"name": "  john DOE  "}
	s.SanitizeRecord(model, data)
	if data["name"] != "John Doe" {
		t.Fatalf("expected 'John Doe', got '%v'", data["name"])
	}
}

func TestSanitize_Slugify(t *testing.T) {
	s := NewSanitizer()
	model := &parser.ModelDefinition{
		Name: "test",
		Fields: map[string]parser.FieldDefinition{
			"slug": {Type: parser.FieldString, Sanitize: []string{"slugify"}},
		},
	}
	data := map[string]any{"slug": "Hello World! This is a Test"}
	s.SanitizeRecord(model, data)
	if data["slug"] != "hello-world-this-is-a-test" {
		t.Fatalf("expected 'hello-world-this-is-a-test', got '%v'", data["slug"])
	}
}

func TestSanitize_StripTags(t *testing.T) {
	s := NewSanitizer()
	model := &parser.ModelDefinition{
		Name: "test",
		Fields: map[string]parser.FieldDefinition{
			"bio": {Type: parser.FieldText, Sanitize: []string{"strip_tags"}},
		},
	}
	data := map[string]any{"bio": "<p>Hello <b>World</b></p>"}
	s.SanitizeRecord(model, data)
	if data["bio"] != "Hello World" {
		t.Fatalf("expected 'Hello World', got '%v'", data["bio"])
	}
}

func TestSanitize_ModelLevel_AllStrings(t *testing.T) {
	s := NewSanitizer()
	model := &parser.ModelDefinition{
		Name: "test",
		Fields: map[string]parser.FieldDefinition{
			"name":     {Type: parser.FieldString},
			"email":    {Type: parser.FieldEmail},
			"age":      {Type: parser.FieldInteger},
			"password": {Type: parser.FieldPassword},
		},
		Sanitize: &parser.SanitizeConfig{AllStrings: []string{"trim"}},
	}
	data := map[string]any{
		"name":     "  John  ",
		"email":    "  test@test.com  ",
		"age":      25,
		"password": "  secret  ",
	}
	s.SanitizeRecord(model, data)
	if data["name"] != "John" {
		t.Fatalf("expected trimmed name, got '%v'", data["name"])
	}
	if data["email"] != "test@test.com" {
		t.Fatalf("expected trimmed email, got '%v'", data["email"])
	}
	if data["password"] != "  secret  " {
		t.Fatal("password should NOT be sanitized by _all_strings")
	}
}

func TestSanitize_Truncate(t *testing.T) {
	s := NewSanitizer()
	model := &parser.ModelDefinition{
		Name: "test",
		Fields: map[string]parser.FieldDefinition{
			"code": {Type: parser.FieldString, Sanitize: []string{"truncate:5"}},
		},
	}
	data := map[string]any{"code": "ABCDEFGH"}
	s.SanitizeRecord(model, data)
	if data["code"] != "ABCDE" {
		t.Fatalf("expected 'ABCDE', got '%v'", data["code"])
	}
}

func TestSanitize_NormalizePhone(t *testing.T) {
	s := NewSanitizer()
	model := &parser.ModelDefinition{
		Name: "test",
		Fields: map[string]parser.FieldDefinition{
			"phone": {Type: parser.FieldString, Sanitize: []string{"normalize_phone"}},
		},
	}
	data := map[string]any{"phone": "+62 812-345-6789"}
	s.SanitizeRecord(model, data)
	if data["phone"] != "+628123456789" {
		t.Fatalf("expected '+628123456789', got '%v'", data["phone"])
	}
}

func TestSanitize_SkipNonString(t *testing.T) {
	s := NewSanitizer()
	model := &parser.ModelDefinition{
		Name: "test",
		Fields: map[string]parser.FieldDefinition{
			"count": {Type: parser.FieldInteger, Sanitize: []string{"trim"}},
		},
	}
	data := map[string]any{"count": 42}
	s.SanitizeRecord(model, data)
	if data["count"] != 42 {
		t.Fatalf("expected 42, got '%v'", data["count"])
	}
}

func TestParseFileSize(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"5MB", 5 * 1024 * 1024},
		{"5mb", 5 * 1024 * 1024},
		{"10KB", 10 * 1024},
		{"1GB", 1024 * 1024 * 1024},
		{"100B", 100},
		{"2.5MB", int64(2.5 * 1024 * 1024)},
		{"1024", 1024},
		{"", 0},
	}
	for _, tt := range tests {
		result := parseFileSize(tt.input)
		if result != tt.expected {
			t.Errorf("parseFileSize(%q) = %d, want %d", tt.input, result, tt.expected)
		}
	}
}
