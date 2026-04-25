package persistence

import (
	"testing"
)

func TestParseOQLSQL_Basic(t *testing.T) {
	q, model, err := ParseOQLSQL("SELECT name, email FROM contacts WHERE status = 'active' ORDER BY name ASC LIMIT 10")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if model != "contacts" {
		t.Errorf("expected model 'contacts', got %q", model)
	}
	if len(q.Select) != 2 {
		t.Errorf("expected 2 select fields, got %d", len(q.Select))
	}
	if len(q.WhereClauses) != 1 {
		t.Errorf("expected 1 where clause, got %d", len(q.WhereClauses))
	}
	if q.Limit != 10 {
		t.Errorf("expected limit 10, got %d", q.Limit)
	}
}

func TestParseOQLSQL_Join(t *testing.T) {
	q, model, err := ParseOQLSQL("FROM contacts LEFT JOIN companies ON contacts.company_id = companies.id WHERE status = 'active'")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if model != "contacts" {
		t.Errorf("expected model 'contacts', got %q", model)
	}
	if len(q.Joins) != 1 {
		t.Fatalf("expected 1 join, got %d", len(q.Joins))
	}
	if q.Joins[0].Type != JoinLeft {
		t.Errorf("expected LEFT join, got %s", q.Joins[0].Type)
	}
}

func TestParseOQLSQL_OrCondition(t *testing.T) {
	q, _, err := ParseOQLSQL("FROM contacts WHERE city = 'Jakarta' OR city = 'Bandung'")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	clauses := q.GetEffectiveWhereClauses()
	if len(clauses) != 1 {
		t.Fatalf("expected 1 top-level clause (OR group), got %d", len(clauses))
	}
	if clauses[0].Group == nil {
		t.Fatal("expected group clause for OR")
	}
	if clauses[0].Group.Connector != ConnectorOr {
		t.Errorf("expected OR connector, got %s", clauses[0].Group.Connector)
	}
}

func TestParseOQLSQL_GroupedCondition(t *testing.T) {
	q, _, err := ParseOQLSQL("FROM contacts WHERE status = 'active' AND (city = 'Jakarta' OR city = 'Bandung')")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	clauses := q.GetEffectiveWhereClauses()
	if len(clauses) < 1 {
		t.Fatalf("expected at least 1 clause, got %d", len(clauses))
	}
}

func TestParseOQLSQL_WithRelation(t *testing.T) {
	q, _, err := ParseOQLSQL("FROM contacts WHERE status = 'active' WITH company, tags")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(q.With) != 2 {
		t.Errorf("expected 2 with relations, got %d", len(q.With))
	}
}

func TestParseOQLSQL_InOperator(t *testing.T) {
	q, _, err := ParseOQLSQL("FROM contacts WHERE city IN ('Jakarta', 'Bandung', 'Surabaya')")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	clauses := q.GetEffectiveWhereClauses()
	if len(clauses) != 1 {
		t.Fatalf("expected 1 clause, got %d", len(clauses))
	}
	if clauses[0].Condition.Operator != "in" {
		t.Errorf("expected 'in' operator, got %q", clauses[0].Condition.Operator)
	}
}

func TestParseOQLSQL_Between(t *testing.T) {
	q, _, err := ParseOQLSQL("FROM orders WHERE amount BETWEEN 100 AND 500")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	clauses := q.GetEffectiveWhereClauses()
	if len(clauses) != 1 {
		t.Fatalf("expected 1 clause, got %d", len(clauses))
	}
	if clauses[0].Condition.Operator != "between" {
		t.Errorf("expected 'between' operator, got %q", clauses[0].Condition.Operator)
	}
}

func TestParseOQLSQL_IsNull(t *testing.T) {
	q, _, err := ParseOQLSQL("FROM contacts WHERE deleted_at IS NULL")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	clauses := q.GetEffectiveWhereClauses()
	if len(clauses) != 1 {
		t.Fatalf("expected 1 clause, got %d", len(clauses))
	}
	if clauses[0].Condition.Operator != "is_null" {
		t.Errorf("expected 'is_null' operator, got %q", clauses[0].Condition.Operator)
	}
}

func TestParseOQLSimple_Basic(t *testing.T) {
	q, model, err := ParseOQLSimple("contacts[status='active', age > 18] ORDER BY name LIMIT 10")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if model != "contacts" {
		t.Errorf("expected model 'contacts', got %q", model)
	}
	if len(q.WhereClauses) != 2 {
		t.Errorf("expected 2 where clauses, got %d", len(q.WhereClauses))
	}
	if q.Limit != 10 {
		t.Errorf("expected limit 10, got %d", q.Limit)
	}
}

func TestParseOQLSimple_WithRelation(t *testing.T) {
	q, _, err := ParseOQLSimple("contacts[status='active'] WITH company, tags")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(q.With) != 2 {
		t.Errorf("expected 2 with relations, got %d", len(q.With))
	}
}

func TestParseOQLDot_Basic(t *testing.T) {
	q, model, err := ParseOQLDot("contacts.where(status.eq('active')).orderBy('name').limit(10).with('company')")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if model != "contacts" {
		t.Errorf("expected model 'contacts', got %q", model)
	}
	if len(q.WhereClauses) != 1 {
		t.Errorf("expected 1 where clause, got %d", len(q.WhereClauses))
	}
	if q.Limit != 10 {
		t.Errorf("expected limit 10, got %d", q.Limit)
	}
	if len(q.With) != 1 {
		t.Errorf("expected 1 with relation, got %d", len(q.With))
	}
}

func TestParseOQLDot_MultipleConditions(t *testing.T) {
	q, _, err := ParseOQLDot("contacts.where(status.eq('active')).where(age.gt(18)).select('name','email')")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(q.WhereClauses) != 2 {
		t.Errorf("expected 2 where clauses, got %d", len(q.WhereClauses))
	}
	if len(q.Select) != 2 {
		t.Errorf("expected 2 select fields, got %d", len(q.Select))
	}
}

func TestParseOQL_AutoDetect_SQL(t *testing.T) {
	q, model, err := ParseOQL("SELECT * FROM contacts WHERE status = 'active'")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if model != "contacts" {
		t.Errorf("expected model 'contacts', got %q", model)
	}
	_ = q
}

func TestParseOQL_AutoDetect_Simple(t *testing.T) {
	q, model, err := ParseOQL("contacts[status='active']")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if model != "contacts" {
		t.Errorf("expected model 'contacts', got %q", model)
	}
	_ = q
}

func TestParseOQL_AutoDetect_Dot(t *testing.T) {
	q, model, err := ParseOQL("contacts.where(status.eq('active')).limit(10)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if model != "contacts" {
		t.Errorf("expected model 'contacts', got %q", model)
	}
	if q.Limit != 10 {
		t.Errorf("expected limit 10, got %d", q.Limit)
	}
}

func TestParseOQL_Empty(t *testing.T) {
	_, _, err := ParseOQL("")
	if err == nil {
		t.Error("expected error for empty input")
	}
}

func TestParseOQLSQL_ComplexQuery(t *testing.T) {
	input := "SELECT c.name, c.email FROM contacts LEFT JOIN companies ON contacts.company_id = companies.id WHERE status = 'active' AND (city = 'Jakarta' OR city = 'Bandung') ORDER BY name ASC, created_at DESC LIMIT 20 OFFSET 40 WITH company, tags"
	q, model, err := ParseOQLSQL(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if model != "contacts" {
		t.Errorf("expected model 'contacts', got %q", model)
	}
	if len(q.Joins) != 1 {
		t.Errorf("expected 1 join, got %d", len(q.Joins))
	}
	if q.Limit != 20 {
		t.Errorf("expected limit 20, got %d", q.Limit)
	}
	if q.Offset != 40 {
		t.Errorf("expected offset 40, got %d", q.Offset)
	}
	if len(q.With) != 2 {
		t.Errorf("expected 2 with relations, got %d", len(q.With))
	}
	if len(q.OrderBy) != 2 {
		t.Errorf("expected 2 order clauses, got %d", len(q.OrderBy))
	}
}

func TestParseOQLSQL_LikeOperator(t *testing.T) {
	q, _, err := ParseOQLSQL("FROM contacts WHERE name LIKE '%john%'")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	clauses := q.GetEffectiveWhereClauses()
	if len(clauses) != 1 {
		t.Fatalf("expected 1 clause, got %d", len(clauses))
	}
	if clauses[0].Condition.Operator != "like" {
		t.Errorf("expected 'like' operator, got %q", clauses[0].Condition.Operator)
	}
}

func TestParseOQLSQL_NumericValues(t *testing.T) {
	q, _, err := ParseOQLSQL("FROM orders WHERE amount > 100.50 AND qty >= 5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	clauses := q.GetEffectiveWhereClauses()
	if len(clauses) != 2 {
		t.Fatalf("expected 2 clauses, got %d", len(clauses))
	}
}

func TestParseOQLSQL_ForUpdate(t *testing.T) {
	q, _, err := ParseOQLSQL("FROM contacts WHERE id = '123' FOR UPDATE")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q.Lock != LockForUpdate {
		t.Errorf("expected LockForUpdate, got %s", q.Lock)
	}
}
