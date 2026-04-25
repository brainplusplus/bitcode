package persistence

import (
	"encoding/json"
	"testing"
)

func TestNewQuery(t *testing.T) {
	q := NewQuery()
	if q == nil {
		t.Fatal("NewQuery returned nil")
	}
	if len(q.WhereClauses) != 0 {
		t.Errorf("expected 0 where clauses, got %d", len(q.WhereClauses))
	}
}

func TestQueryWhere(t *testing.T) {
	q := NewQuery().Where("status", "=", "active").Where("age", ">", 18)
	clauses := q.GetEffectiveWhereClauses()
	if len(clauses) != 2 {
		t.Fatalf("expected 2 clauses, got %d", len(clauses))
	}
	if clauses[0].Condition.Field != "status" {
		t.Errorf("expected field 'status', got %q", clauses[0].Condition.Field)
	}
	if clauses[1].Condition.Operator != ">" {
		t.Errorf("expected operator '>', got %q", clauses[1].Condition.Operator)
	}
}

func TestQueryConvenienceMethods(t *testing.T) {
	q := NewQuery().
		WhereEq("name", "John").
		WhereNe("status", "deleted").
		WhereGt("age", 18).
		WhereGte("score", 90).
		WhereLt("price", 100).
		WhereLte("qty", 50).
		WhereLike("email", "%@gmail.com").
		WhereIn("city", []string{"Jakarta", "Bandung"}).
		WhereNotIn("role", []string{"admin"}).
		WhereBetween("created_at", "2024-01-01", "2024-12-31").
		WhereNull("deleted_at").
		WhereNotNull("email")

	clauses := q.GetEffectiveWhereClauses()
	if len(clauses) != 12 {
		t.Fatalf("expected 12 clauses, got %d", len(clauses))
	}
}

func TestQueryOrWhere(t *testing.T) {
	q := NewQuery().Where("status", "=", "active").OrWhere("status", "=", "pending")
	clauses := q.GetEffectiveWhereClauses()
	if len(clauses) != 1 {
		t.Fatalf("expected 1 top-level clause (OR group), got %d", len(clauses))
	}
	if clauses[0].Group == nil {
		t.Fatal("expected group clause")
	}
	if clauses[0].Group.Connector != ConnectorOr {
		t.Errorf("expected OR connector, got %s", clauses[0].Group.Connector)
	}
}

func TestQueryWhereGroup(t *testing.T) {
	q := NewQuery().
		Where("active", "=", true).
		WhereGroup(func(sub *Query) *Query {
			return sub.Where("city", "=", "Jakarta").Where("city", "=", "Bandung")
		})

	clauses := q.GetEffectiveWhereClauses()
	if len(clauses) != 2 {
		t.Fatalf("expected 2 clauses, got %d", len(clauses))
	}
	if clauses[1].Group == nil {
		t.Fatal("expected group clause")
	}
}

func TestQueryWhereNot(t *testing.T) {
	q := NewQuery().WhereNot(func(sub *Query) *Query {
		return sub.Where("status", "=", "deleted")
	})
	clauses := q.GetEffectiveWhereClauses()
	if len(clauses) != 1 {
		t.Fatalf("expected 1 clause, got %d", len(clauses))
	}
	if !clauses[0].Group.Negate {
		t.Error("expected negated group")
	}
}

func TestQueryJoins(t *testing.T) {
	q := NewQuery().
		InnerJoin("companies", "contacts.company_id", "companies.id").
		LeftJoin("addresses", "contacts.id", "addresses.contact_id")

	if len(q.Joins) != 2 {
		t.Fatalf("expected 2 joins, got %d", len(q.Joins))
	}
	if q.Joins[0].Type != JoinInner {
		t.Errorf("expected INNER join, got %s", q.Joins[0].Type)
	}
	if q.Joins[1].Type != JoinLeft {
		t.Errorf("expected LEFT join, got %s", q.Joins[1].Type)
	}
}

func TestQueryWith(t *testing.T) {
	q := NewQuery().WithRelation("company").WithRelation("tags")
	if len(q.With) != 2 {
		t.Fatalf("expected 2 with clauses, got %d", len(q.With))
	}
}

func TestQueryAggregates(t *testing.T) {
	q := NewQuery().
		SetGroupBy("department").
		SelectCount("*", "total").
		SelectSum("salary", "total_salary").
		SelectAvg("age", "avg_age").
		SelectMin("salary", "min_salary").
		SelectMax("salary", "max_salary")

	if len(q.Aggregates) != 5 {
		t.Fatalf("expected 5 aggregates, got %d", len(q.Aggregates))
	}
}

func TestQueryDistinct(t *testing.T) {
	q := NewQuery().SetDistinct(true)
	if !q.Distinct {
		t.Error("expected distinct to be true")
	}
}

func TestQueryHaving(t *testing.T) {
	q := NewQuery().
		SetGroupBy("department").
		HavingCondition("COUNT", "*", ">", 5)

	if len(q.Having) != 1 {
		t.Fatalf("expected 1 having clause, got %d", len(q.Having))
	}
}

func TestQueryUnion(t *testing.T) {
	q1 := NewQuery().Where("type", "=", "A")
	q2 := NewQuery().Where("type", "=", "B")
	q := q1.Union(q2)
	if len(q.Unions) != 1 {
		t.Fatalf("expected 1 union, got %d", len(q.Unions))
	}
}

func TestQueryLocking(t *testing.T) {
	q := NewQuery().LockForUpdate()
	if q.Lock != LockForUpdate {
		t.Errorf("expected LockForUpdate, got %s", q.Lock)
	}
}

func TestQuerySoftDeleteScopes(t *testing.T) {
	q := NewQuery().WithTrashed()
	if q.SoftDeleteScope != ScopeWithTrashed {
		t.Errorf("expected ScopeWithTrashed, got %s", q.SoftDeleteScope)
	}

	q2 := NewQuery().OnlyTrashed()
	if q2.SoftDeleteScope != ScopeOnlyTrashed {
		t.Errorf("expected ScopeOnlyTrashed, got %s", q2.SoftDeleteScope)
	}
}

func TestQueryScopes(t *testing.T) {
	activeScope := func(q *Query) *Query {
		return q.Where("active", "=", true)
	}
	q := NewQuery().Scope(activeScope).ApplyScopes()
	clauses := q.GetEffectiveWhereClauses()
	if len(clauses) != 1 {
		t.Fatalf("expected 1 clause after scope, got %d", len(clauses))
	}
}

func TestQueryOrderAndReorder(t *testing.T) {
	q := NewQuery().OrderAsc("name").OrderDesc("created_at")
	if len(q.OrderBy) != 2 {
		t.Fatalf("expected 2 orders, got %d", len(q.OrderBy))
	}
	q.Reorder()
	if len(q.OrderBy) != 0 {
		t.Fatalf("expected 0 orders after reorder, got %d", len(q.OrderBy))
	}
}

func TestSanitizeFieldName(t *testing.T) {
	tests := []struct {
		input string
		safe  bool
	}{
		{"name", true},
		{"user.name", true},
		{"table_name", true},
		{"1invalid", false},
		{"name; DROP TABLE", false},
		{"", false},
		{"valid_field_123", true},
	}
	for _, tt := range tests {
		if IsSafeFieldName(tt.input) != tt.safe {
			t.Errorf("IsSafeFieldName(%q) = %v, want %v", tt.input, !tt.safe, tt.safe)
		}
	}
}

func TestParseQueryFromMap(t *testing.T) {
	m := map[string]any{
		"wheres": []any{
			map[string]any{"field": "status", "op": "=", "value": "active"},
			map[string]any{"field": "age", "op": ">", "value": 18},
		},
		"order": []any{
			map[string]any{"field": "name", "direction": "asc"},
		},
		"limit":    float64(10),
		"offset":   float64(20),
		"select":   []any{"name", "email"},
		"group_by": []any{"department"},
		"distinct": true,
		"with":     []any{"company", "tags"},
	}

	q := ParseQueryFromMap(m)
	if len(q.WhereClauses) != 2 {
		t.Errorf("expected 2 where clauses, got %d", len(q.WhereClauses))
	}
	if len(q.OrderBy) != 1 {
		t.Errorf("expected 1 order, got %d", len(q.OrderBy))
	}
	if q.Limit != 10 {
		t.Errorf("expected limit 10, got %d", q.Limit)
	}
	if q.Offset != 20 {
		t.Errorf("expected offset 20, got %d", q.Offset)
	}
	if len(q.Select) != 2 {
		t.Errorf("expected 2 select fields, got %d", len(q.Select))
	}
	if len(q.GroupBy) != 1 {
		t.Errorf("expected 1 group_by, got %d", len(q.GroupBy))
	}
	if !q.Distinct {
		t.Error("expected distinct true")
	}
	if len(q.With) != 2 {
		t.Errorf("expected 2 with clauses, got %d", len(q.With))
	}
}

func TestParseQueryFromMapWithGroups(t *testing.T) {
	m := map[string]any{
		"where_groups": []any{
			map[string]any{
				"connector": "OR",
				"conditions": []any{
					map[string]any{"field": "city", "op": "=", "value": "Jakarta"},
					map[string]any{"field": "city", "op": "=", "value": "Bandung"},
				},
			},
		},
	}

	q := ParseQueryFromMap(m)
	if len(q.WhereClauses) != 1 {
		t.Fatalf("expected 1 where clause, got %d", len(q.WhereClauses))
	}
	if q.WhereClauses[0].Group == nil {
		t.Fatal("expected group clause")
	}
	if q.WhereClauses[0].Group.Connector != ConnectorOr {
		t.Errorf("expected OR connector, got %s", q.WhereClauses[0].Group.Connector)
	}
}

func TestQueryFromDomain(t *testing.T) {
	filters := [][]any{
		{"status", "=", "active"},
		{"age", ">", 18},
	}
	q := QueryFromDomain(filters)
	clauses := q.GetEffectiveWhereClauses()
	if len(clauses) != 2 {
		t.Fatalf("expected 2 clauses, got %d", len(clauses))
	}
}

func TestParseQueryFromJSON(t *testing.T) {
	jsonStr := `{
		"select": ["name", "email"],
		"order": [{"field": "name", "direction": "asc"}],
		"limit": 10,
		"distinct": true
	}`
	q, err := ParseQueryFromJSON([]byte(jsonStr))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(q.Select) != 2 {
		t.Errorf("expected 2 select fields, got %d", len(q.Select))
	}
	if q.Limit != 10 {
		t.Errorf("expected limit 10, got %d", q.Limit)
	}
	if !q.Distinct {
		t.Error("expected distinct true")
	}
}

func TestQueryMergeLegacyConditions(t *testing.T) {
	q := &Query{
		Conditions: []Condition{
			{Field: "a", Operator: "=", Value: 1},
		},
		WhereClauses: []WhereClause{
			{Condition: &Condition{Field: "b", Operator: "=", Value: 2}},
		},
	}
	q.MergeLegacyConditions()
	if len(q.WhereClauses) != 2 {
		t.Errorf("expected 2 where clauses after merge, got %d", len(q.WhereClauses))
	}
	if len(q.Conditions) != 0 {
		t.Errorf("expected 0 legacy conditions after merge, got %d", len(q.Conditions))
	}
}

func TestQueryWhereColumn(t *testing.T) {
	q := NewQuery().WhereColumn("updated_at", ">", "created_at")
	clauses := q.GetEffectiveWhereClauses()
	if len(clauses) != 1 {
		t.Fatalf("expected 1 clause, got %d", len(clauses))
	}
	if clauses[0].Condition.Operator != "column:>" {
		t.Errorf("expected operator 'column:>', got %q", clauses[0].Condition.Operator)
	}
}

func TestQuerySubQuery(t *testing.T) {
	sub := NewQuery().SetSelect("id").Where("active", "=", true)
	q := NewQuery().WhereInSubQuery("company_id", sub)
	if len(q.WhereSubQueries) != 1 {
		t.Fatalf("expected 1 subquery, got %d", len(q.WhereSubQueries))
	}
}

func TestQueryRawExpressions(t *testing.T) {
	q := NewQuery().
		WhereRawExpr("age > ? AND age < ?", 18, 65).
		SelectRawExpr("COUNT(*) as total").
		OrderRawExpr("FIELD(status, 'active', 'pending', 'closed')").
		HavingRawExpr("COUNT(*) > ?", 5)

	if len(q.WhereRaw) != 1 {
		t.Errorf("expected 1 where raw, got %d", len(q.WhereRaw))
	}
	if len(q.SelectRaw) != 1 {
		t.Errorf("expected 1 select raw, got %d", len(q.SelectRaw))
	}
	if len(q.OrderRaw) != 1 {
		t.Errorf("expected 1 order raw, got %d", len(q.OrderRaw))
	}
	if len(q.HavingRaw) != 1 {
		t.Errorf("expected 1 having raw, got %d", len(q.HavingRaw))
	}
}

func TestQueryJSONSerialization(t *testing.T) {
	q := NewQuery().
		Where("status", "=", "active").
		InnerJoin("companies", "contacts.company_id", "companies.id").
		WithRelation("tags").
		SetDistinct(true).
		SetLimit(10)

	data, err := json.Marshal(q)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var q2 Query
	if err := json.Unmarshal(data, &q2); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(q2.Joins) != 1 {
		t.Errorf("expected 1 join after roundtrip, got %d", len(q2.Joins))
	}
	if len(q2.With) != 1 {
		t.Errorf("expected 1 with after roundtrip, got %d", len(q2.With))
	}
	if !q2.Distinct {
		t.Error("expected distinct after roundtrip")
	}
}
