package runtime

import (
	"testing"
)

func TestParseDynamicFinder_FindBy(t *testing.T) {
	result, ok := ParseDynamicFinder("FindByEmail")
	if !ok {
		t.Fatal("expected dynamic finder to parse")
	}
	if result.Type != "find_one" {
		t.Errorf("expected type 'find_one', got %q", result.Type)
	}
	if len(result.Fields) != 1 {
		t.Fatalf("expected 1 field, got %d", len(result.Fields))
	}
	if result.Fields[0].Name != "email" {
		t.Errorf("expected field 'email', got %q", result.Fields[0].Name)
	}
	if result.Fields[0].Operator != "=" {
		t.Errorf("expected operator '=', got %q", result.Fields[0].Operator)
	}
}

func TestParseDynamicFinder_FindAllBy(t *testing.T) {
	result, ok := ParseDynamicFinder("FindAllByStatus")
	if !ok {
		t.Fatal("expected dynamic finder to parse")
	}
	if result.Type != "find_all" {
		t.Errorf("expected type 'find_all', got %q", result.Type)
	}
	if result.Fields[0].Name != "status" {
		t.Errorf("expected field 'status', got %q", result.Fields[0].Name)
	}
}

func TestParseDynamicFinder_FindByAnd(t *testing.T) {
	result, ok := ParseDynamicFinder("FindByStatusAndCity")
	if !ok {
		t.Fatal("expected dynamic finder to parse")
	}
	if len(result.Fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(result.Fields))
	}
	if result.Fields[0].Name != "status" {
		t.Errorf("expected field 'status', got %q", result.Fields[0].Name)
	}
	if result.Fields[1].Name != "city" {
		t.Errorf("expected field 'city', got %q", result.Fields[1].Name)
	}
	if result.Fields[1].Connector != "AND" {
		t.Errorf("expected connector 'AND', got %q", result.Fields[1].Connector)
	}
}

func TestParseDynamicFinder_FindByOr(t *testing.T) {
	result, ok := ParseDynamicFinder("FindByEmailOrPhone")
	if !ok {
		t.Fatal("expected dynamic finder to parse")
	}
	if len(result.Fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(result.Fields))
	}
	if result.Fields[1].Connector != "OR" {
		t.Errorf("expected connector 'OR', got %q", result.Fields[1].Connector)
	}
}

func TestParseDynamicFinder_FindAllByIn(t *testing.T) {
	result, ok := ParseDynamicFinder("FindAllByCityIn")
	if !ok {
		t.Fatal("expected dynamic finder to parse")
	}
	if result.Fields[0].Operator != "in" {
		t.Errorf("expected operator 'in', got %q", result.Fields[0].Operator)
	}
}

func TestParseDynamicFinder_FindAllByLike(t *testing.T) {
	result, ok := ParseDynamicFinder("FindAllByNameLike")
	if !ok {
		t.Fatal("expected dynamic finder to parse")
	}
	if result.Fields[0].Operator != "like" {
		t.Errorf("expected operator 'like', got %q", result.Fields[0].Operator)
	}
}

func TestParseDynamicFinder_FindAllByBetween(t *testing.T) {
	result, ok := ParseDynamicFinder("FindAllByAgeBetween")
	if !ok {
		t.Fatal("expected dynamic finder to parse")
	}
	if result.Fields[0].Operator != "between" {
		t.Errorf("expected operator 'between', got %q", result.Fields[0].Operator)
	}
}

func TestParseDynamicFinder_FindAllByIsNull(t *testing.T) {
	result, ok := ParseDynamicFinder("FindAllByDeletedAtIsNull")
	if !ok {
		t.Fatal("expected dynamic finder to parse")
	}
	if result.Fields[0].Operator != "is_null" {
		t.Errorf("expected operator 'is_null', got %q", result.Fields[0].Operator)
	}
	if result.Fields[0].Name != "deleted_at" {
		t.Errorf("expected field 'deleted_at', got %q", result.Fields[0].Name)
	}
}

func TestParseDynamicFinder_FindAllByGt(t *testing.T) {
	result, ok := ParseDynamicFinder("FindAllByAgeGt")
	if !ok {
		t.Fatal("expected dynamic finder to parse")
	}
	if result.Fields[0].Operator != ">" {
		t.Errorf("expected operator '>', got %q", result.Fields[0].Operator)
	}
}

func TestParseDynamicFinder_FindAllByNot(t *testing.T) {
	result, ok := ParseDynamicFinder("FindAllByStatusNot")
	if !ok {
		t.Fatal("expected dynamic finder to parse")
	}
	if result.Fields[0].Operator != "!=" {
		t.Errorf("expected operator '!=', got %q", result.Fields[0].Operator)
	}
}

func TestParseDynamicFinder_CountBy(t *testing.T) {
	result, ok := ParseDynamicFinder("CountByStatus")
	if !ok {
		t.Fatal("expected dynamic finder to parse")
	}
	if result.Type != "count" {
		t.Errorf("expected type 'count', got %q", result.Type)
	}
}

func TestParseDynamicFinder_ExistsBy(t *testing.T) {
	result, ok := ParseDynamicFinder("ExistsByEmail")
	if !ok {
		t.Fatal("expected dynamic finder to parse")
	}
	if result.Type != "exists" {
		t.Errorf("expected type 'exists', got %q", result.Type)
	}
}

func TestParseDynamicFinder_SumBy(t *testing.T) {
	result, ok := ParseDynamicFinder("SumByAmountStatus")
	if !ok {
		t.Fatal("expected dynamic finder to parse")
	}
	if result.Type != "sum" {
		t.Errorf("expected type 'sum', got %q", result.Type)
	}
	if result.AggField != "amount" {
		t.Errorf("expected agg field 'amount', got %q", result.AggField)
	}
	if result.Fields[0].Name != "status" {
		t.Errorf("expected field 'status', got %q", result.Fields[0].Name)
	}
}

func TestParseDynamicFinder_AvgBy(t *testing.T) {
	result, ok := ParseDynamicFinder("AvgByAgeStatus")
	if !ok {
		t.Fatal("expected dynamic finder to parse")
	}
	if result.Type != "avg" {
		t.Errorf("expected type 'avg', got %q", result.Type)
	}
	if result.AggField != "age" {
		t.Errorf("expected agg field 'age', got %q", result.AggField)
	}
}

func TestParseDynamicFinder_OrderBy(t *testing.T) {
	result, ok := ParseDynamicFinder("FindAllByStatusOrderByNameAsc")
	if !ok {
		t.Fatal("expected dynamic finder to parse")
	}
	if len(result.OrderBy) != 1 {
		t.Fatalf("expected 1 order, got %d", len(result.OrderBy))
	}
	if result.OrderBy[0].Field != "name" {
		t.Errorf("expected order field 'name', got %q", result.OrderBy[0].Field)
	}
	if result.OrderBy[0].Direction != "asc" {
		t.Errorf("expected direction 'asc', got %q", result.OrderBy[0].Direction)
	}
}

func TestParseDynamicFinder_OrderByDesc(t *testing.T) {
	result, ok := ParseDynamicFinder("FindAllByActiveOrderByCreatedAtDesc")
	if !ok {
		t.Fatal("expected dynamic finder to parse")
	}
	if result.OrderBy[0].Direction != "desc" {
		t.Errorf("expected direction 'desc', got %q", result.OrderBy[0].Direction)
	}
}

func TestParseDynamicFinder_DeleteBy(t *testing.T) {
	result, ok := ParseDynamicFinder("DeleteByStatus")
	if !ok {
		t.Fatal("expected dynamic finder to parse")
	}
	if result.Type != "delete" {
		t.Errorf("expected type 'delete', got %q", result.Type)
	}
}

func TestParseDynamicFinder_PluckBy(t *testing.T) {
	result, ok := ParseDynamicFinder("PluckByEmailStatus")
	if !ok {
		t.Fatal("expected dynamic finder to parse")
	}
	if result.Type != "pluck" {
		t.Errorf("expected type 'pluck', got %q", result.Type)
	}
	if result.AggField != "email" {
		t.Errorf("expected agg field 'email', got %q", result.AggField)
	}
}

func TestParseDynamicFinder_NotDynamic(t *testing.T) {
	_, ok := ParseDynamicFinder("GetAll")
	if ok {
		t.Error("expected GetAll to not be a dynamic finder")
	}
}

func TestParseDynamicFinder_CamelToSnake(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Status", "status"},
		{"CreatedAt", "created_at"},
		{"DepartmentId", "department_id"},
		{"", ""},
		{"name", "name"},
	}
	for _, tt := range tests {
		result := camelToSnake(tt.input)
		if result != tt.expected {
			t.Errorf("camelToSnake(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestParseDynamicFinder_BuildQuery(t *testing.T) {
	result, ok := ParseDynamicFinder("FindAllByStatusAndCityIn")
	if !ok {
		t.Fatal("expected dynamic finder to parse")
	}

	args := map[string]any{
		"status": "active",
		"city":   []string{"Jakarta", "Bandung"},
	}
	q := result.BuildQuery(args)
	clauses := q.GetEffectiveWhereClauses()
	if len(clauses) != 2 {
		t.Fatalf("expected 2 where clauses, got %d", len(clauses))
	}
}

func TestParseDynamicFinder_MultipleOrderBy(t *testing.T) {
	result, ok := ParseDynamicFinder("FindAllByStatusOrderByNameAscAndCreatedAtDesc")
	if !ok {
		t.Fatal("expected dynamic finder to parse")
	}
	if len(result.OrderBy) != 2 {
		t.Fatalf("expected 2 orders, got %d", len(result.OrderBy))
	}
	if result.OrderBy[0].Field != "name" || result.OrderBy[0].Direction != "asc" {
		t.Errorf("first order: got %s %s", result.OrderBy[0].Field, result.OrderBy[0].Direction)
	}
	if result.OrderBy[1].Field != "created_at" || result.OrderBy[1].Direction != "desc" {
		t.Errorf("second order: got %s %s", result.OrderBy[1].Field, result.OrderBy[1].Direction)
	}
}

func TestParseDynamicFinder_FindAllByNotIn(t *testing.T) {
	result, ok := ParseDynamicFinder("FindAllByStatusNotIn")
	if !ok {
		t.Fatal("expected dynamic finder to parse")
	}
	if result.Fields[0].Operator != "not_in" {
		t.Errorf("expected operator 'not_in', got %q", result.Fields[0].Operator)
	}
}

func TestParseDynamicFinder_FindAllByGteAndLte(t *testing.T) {
	result, ok := ParseDynamicFinder("FindAllByAgeGteAndAgeLte")
	if !ok {
		t.Fatal("expected dynamic finder to parse")
	}
	if len(result.Fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(result.Fields))
	}
	if result.Fields[0].Operator != ">=" {
		t.Errorf("expected operator '>=', got %q", result.Fields[0].Operator)
	}
	if result.Fields[1].Operator != "<=" {
		t.Errorf("expected operator '<=', got %q", result.Fields[1].Operator)
	}
}
