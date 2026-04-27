package graphql

import (
	"encoding/json"
	"testing"

	gql "github.com/graphql-go/graphql"
	"github.com/bitcode-framework/bitcode/internal/compiler/parser"
)

func TestSchemaBuilder_BuildEmpty(t *testing.T) {
	resolver := &Resolver{}
	builder := NewSchemaBuilder(resolver)

	schema, err := builder.Build()
	if err != nil {
		t.Fatalf("failed to build empty schema: %v", err)
	}
	if schema == nil {
		t.Fatal("expected non-nil schema")
	}
}

func TestSchemaBuilder_BuildWithModel(t *testing.T) {
	resolver := &Resolver{}
	builder := NewSchemaBuilder(resolver)

	model := &parser.ModelDefinition{
		Name:   "contact",
		Module: "crm",
		Label:  "Contact",
		API: &parser.APIConfig{
			AutoCRUD:  true,
			Protocols: parser.ProtocolConfig{REST: true, GraphQL: true},
		},
		Fields: map[string]parser.FieldDefinition{
			"name":  {Type: parser.FieldString, Required: true},
			"email": {Type: parser.FieldEmail},
			"age":   {Type: parser.FieldInteger},
			"score": {Type: parser.FieldFloat},
			"active": {Type: parser.FieldBoolean},
		},
	}

	builder.AddModel(model)
	schema, err := builder.Build()
	if err != nil {
		t.Fatalf("failed to build schema: %v", err)
	}

	result := gql.Do(gql.Params{
		Schema:        *schema,
		RequestString: `{ __schema { queryType { fields { name } } } }`,
	})

	if len(result.Errors) > 0 {
		t.Fatalf("introspection errors: %v", result.Errors)
	}

	data, _ := json.Marshal(result.Data)
	dataStr := string(data)

	if !containsStr(dataStr, "contact_list") {
		t.Error("expected contact_list query field")
	}
	if !containsStr(dataStr, "contact") {
		t.Error("expected contact query field")
	}
}

func TestSchemaBuilder_SkipsNonGraphQLModel(t *testing.T) {
	resolver := &Resolver{}
	builder := NewSchemaBuilder(resolver)

	model := &parser.ModelDefinition{
		Name:   "internal_log",
		Module: "base",
		API: &parser.APIConfig{
			AutoCRUD:  true,
			Protocols: parser.ProtocolConfig{REST: true, GraphQL: false},
		},
		Fields: map[string]parser.FieldDefinition{
			"message": {Type: parser.FieldString},
		},
	}

	builder.AddModel(model)
	schema, err := builder.Build()
	if err != nil {
		t.Fatalf("failed to build schema: %v", err)
	}

	result := gql.Do(gql.Params{
		Schema:        *schema,
		RequestString: `{ __schema { queryType { fields { name } } } }`,
	})

	data, _ := json.Marshal(result.Data)
	dataStr := string(data)

	if containsStr(dataStr, "internal_log") {
		t.Error("internal_log should not be in schema (graphql=false)")
	}
}

func TestSchemaBuilder_MutationFields(t *testing.T) {
	resolver := &Resolver{}
	builder := NewSchemaBuilder(resolver)

	model := &parser.ModelDefinition{
		Name:   "tag",
		Module: "crm",
		API: &parser.APIConfig{
			AutoCRUD:  true,
			Protocols: parser.ProtocolConfig{REST: true, GraphQL: true},
		},
		Fields: map[string]parser.FieldDefinition{
			"name": {Type: parser.FieldString, Required: true},
		},
	}

	builder.AddModel(model)
	schema, err := builder.Build()
	if err != nil {
		t.Fatalf("failed to build schema: %v", err)
	}

	result := gql.Do(gql.Params{
		Schema:        *schema,
		RequestString: `{ __schema { mutationType { fields { name } } } }`,
	})

	if len(result.Errors) > 0 {
		t.Fatalf("introspection errors: %v", result.Errors)
	}

	data, _ := json.Marshal(result.Data)
	dataStr := string(data)

	if !containsStr(dataStr, "create_tag") {
		t.Error("expected create_tag mutation")
	}
	if !containsStr(dataStr, "update_tag") {
		t.Error("expected update_tag mutation")
	}
	if !containsStr(dataStr, "delete_tag") {
		t.Error("expected delete_tag mutation")
	}
}

func TestFieldTypeToGraphQL(t *testing.T) {
	tests := []struct {
		fieldType parser.FieldType
		expectNil bool
	}{
		{parser.FieldString, false},
		{parser.FieldInteger, false},
		{parser.FieldFloat, false},
		{parser.FieldBoolean, false},
		{parser.FieldDate, false},
		{parser.FieldSelection, false},
		{parser.FieldMany2One, false},
		{parser.FieldOne2Many, true},
		{parser.FieldMany2Many, true},
	}

	for _, tt := range tests {
		result := fieldTypeToGraphQL(tt.fieldType)
		if tt.expectNil && result != nil {
			t.Errorf("fieldTypeToGraphQL(%s) should be nil", tt.fieldType)
		}
		if !tt.expectNil && result == nil {
			t.Errorf("fieldTypeToGraphQL(%s) should not be nil", tt.fieldType)
		}
	}
}

func containsStr(haystack, needle string) bool {
	return len(haystack) > 0 && len(needle) > 0 && json.Valid([]byte(haystack)) && indexOf(haystack, needle) >= 0
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
