package persistence

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// ConditionConnector defines how conditions are combined
type ConditionConnector string

const (
	ConnectorAnd ConditionConnector = "AND"
	ConnectorOr  ConditionConnector = "OR"
)

// JoinType defines SQL join types
type JoinType string

const (
	JoinInner JoinType = "INNER"
	JoinLeft  JoinType = "LEFT"
	JoinRight JoinType = "RIGHT"
	JoinCross JoinType = "CROSS"
	JoinFull  JoinType = "FULL"
)

// LockType defines row-level locking
type LockType string

const (
	LockNone      LockType = ""
	LockForUpdate LockType = "FOR UPDATE"
	LockForShare  LockType = "FOR SHARE"
)

// SoftDeleteScope controls soft-delete visibility
type SoftDeleteScope string

const (
	ScopeDefault     SoftDeleteScope = ""
	ScopeWithTrashed SoftDeleteScope = "with_trashed"
	ScopeOnlyTrashed SoftDeleteScope = "only_trashed"
)

// Condition represents a single WHERE condition
type Condition struct {
	Field    string `json:"field"`
	Operator string `json:"op"`
	Value    any    `json:"value"`
}

// ConditionGroup represents a group of conditions with AND/OR logic
type ConditionGroup struct {
	Connector  ConditionConnector `json:"connector"`
	Conditions []WhereClause      `json:"conditions"`
	Negate     bool               `json:"negate,omitempty"`
}

// WhereClause is either a single Condition or a nested ConditionGroup
type WhereClause struct {
	Condition *Condition      `json:"condition,omitempty"`
	Group     *ConditionGroup `json:"group,omitempty"`
}

// OrderClause defines ORDER BY
type OrderClause struct {
	Field     string `json:"field"`
	Direction string `json:"direction"`
}

// JoinClause defines a JOIN
type JoinClause struct {
	Type       JoinType    `json:"type"`
	Table      string      `json:"table"`
	Alias      string      `json:"alias,omitempty"`
	LocalKey   string      `json:"local_key"`
	ForeignKey string      `json:"foreign_key"`
	Conditions []Condition `json:"conditions,omitempty"`
	RawOn      string      `json:"raw_on,omitempty"`
}

// HavingClause defines HAVING conditions
type HavingClause struct {
	Raw       string `json:"raw,omitempty"`
	Field     string `json:"field,omitempty"`
	Operator  string `json:"op,omitempty"`
	Value     any    `json:"value,omitempty"`
	Aggregate string `json:"aggregate,omitempty"`
}

// WithClause defines eager loading / preload
type WithClause struct {
	Relation   string       `json:"relation"`
	Conditions []Condition  `json:"conditions,omitempty"`
	Select     []string     `json:"select,omitempty"`
	OrderBy    []OrderClause `json:"order,omitempty"`
	Limit      int          `json:"limit,omitempty"`
	Nested     []WithClause `json:"nested,omitempty"`
}

// SubQuery represents a subquery that can be used in WHERE, SELECT, or FROM
type SubQuery struct {
	Query *Query `json:"query"`
	Alias string `json:"alias,omitempty"`
}

// UnionClause defines UNION operations
type UnionClause struct {
	Query *Query `json:"query"`
	All   bool   `json:"all,omitempty"`
}

// AggregateField defines an aggregate expression in SELECT
type AggregateField struct {
	Function string `json:"function"` // COUNT, SUM, AVG, MIN, MAX
	Field    string `json:"field"`
	Alias    string `json:"alias"`
	Distinct bool   `json:"distinct,omitempty"`
}

// RawExpression represents a raw SQL/expression
type RawExpression struct {
	SQL    string `json:"sql"`
	Values []any  `json:"values,omitempty"`
}

// ScopeFunc is a reusable query modifier
type ScopeFunc func(q *Query) *Query

// Query is the comprehensive query builder
type Query struct {
	// WHERE conditions (supports AND/OR/NOT nesting)
	WhereClauses []WhereClause `json:"wheres,omitempty"`

	// Legacy flat conditions (backward compat — merged as AND into WhereClauses)
	Conditions []Condition `json:"-"`

	// ORDER BY
	OrderBy []OrderClause `json:"order,omitempty"`

	// Pagination
	Limit  int `json:"limit,omitempty"`
	Offset int `json:"offset,omitempty"`

	// Projection
	Select     []string         `json:"select,omitempty"`
	SelectRaw  []RawExpression  `json:"select_raw,omitempty"`
	Aggregates []AggregateField `json:"aggregates,omitempty"`

	// GROUP BY + HAVING
	GroupBy []string       `json:"group_by,omitempty"`
	Having  []HavingClause `json:"having,omitempty"`

	// DISTINCT
	Distinct bool `json:"distinct,omitempty"`

	// JOINs
	Joins []JoinClause `json:"joins,omitempty"`

	// Eager loading / Preload
	With []WithClause `json:"with,omitempty"`

	// Subqueries
	WhereSubQueries []struct {
		Field    string   `json:"field"`
		Operator string   `json:"op"`
		Sub      SubQuery `json:"sub"`
	} `json:"where_sub,omitempty"`
	WhereExists    []SubQuery `json:"where_exists,omitempty"`
	WhereNotExists []SubQuery `json:"where_not_exists,omitempty"`

	// UNION
	Unions []UnionClause `json:"unions,omitempty"`

	// Raw expressions
	WhereRaw  []RawExpression `json:"where_raw,omitempty"`
	OrderRaw  []RawExpression `json:"order_raw,omitempty"`
	GroupRaw  []RawExpression `json:"group_raw,omitempty"`
	HavingRaw []RawExpression `json:"having_raw,omitempty"`

	// Locking
	Lock LockType `json:"lock,omitempty"`

	// Soft delete scope
	SoftDeleteScope SoftDeleteScope `json:"soft_delete_scope,omitempty"`

	// Scopes (applied at execution time)
	scopes []ScopeFunc
}

// ---------------------------------------------------------------------------
// Constructor
// ---------------------------------------------------------------------------

func NewQuery() *Query {
	return &Query{}
}

// ---------------------------------------------------------------------------
// WHERE — basic conditions (AND)
// ---------------------------------------------------------------------------

func (q *Query) Where(field, operator string, value any) *Query {
	q.WhereClauses = append(q.WhereClauses, WhereClause{
		Condition: &Condition{Field: field, Operator: operator, Value: value},
	})
	return q
}

func (q *Query) WhereEq(field string, value any) *Query {
	return q.Where(field, "=", value)
}

func (q *Query) WhereNe(field string, value any) *Query {
	return q.Where(field, "!=", value)
}

func (q *Query) WhereGt(field string, value any) *Query {
	return q.Where(field, ">", value)
}

func (q *Query) WhereGte(field string, value any) *Query {
	return q.Where(field, ">=", value)
}

func (q *Query) WhereLt(field string, value any) *Query {
	return q.Where(field, "<", value)
}

func (q *Query) WhereLte(field string, value any) *Query {
	return q.Where(field, "<=", value)
}

func (q *Query) WhereLike(field string, value any) *Query {
	return q.Where(field, "like", value)
}

func (q *Query) WhereIn(field string, values any) *Query {
	return q.Where(field, "in", values)
}

func (q *Query) WhereNotIn(field string, values any) *Query {
	return q.Where(field, "not_in", values)
}

func (q *Query) WhereBetween(field string, low, high any) *Query {
	return q.Where(field, "between", []any{low, high})
}

func (q *Query) WhereNull(field string) *Query {
	return q.Where(field, "is_null", nil)
}

func (q *Query) WhereNotNull(field string) *Query {
	return q.Where(field, "is_not_null", nil)
}

func (q *Query) WhereColumn(field1, operator, field2 string) *Query {
	return q.Where(field1, "column:"+operator, field2)
}

// ---------------------------------------------------------------------------
// WHERE — OR conditions
// ---------------------------------------------------------------------------

func (q *Query) OrWhere(field, operator string, value any) *Query {
	if len(q.WhereClauses) == 0 {
		return q.Where(field, operator, value)
	}
	existing := q.WhereClauses
	q.WhereClauses = []WhereClause{
		{
			Group: &ConditionGroup{
				Connector: ConnectorOr,
				Conditions: append(existing, WhereClause{
					Condition: &Condition{Field: field, Operator: operator, Value: value},
				}),
			},
		},
	}
	return q
}

// ---------------------------------------------------------------------------
// WHERE — grouped conditions (closures)
// ---------------------------------------------------------------------------

// WhereGroup adds a group of AND conditions: WHERE ... AND (sub-conditions)
func (q *Query) WhereGroup(fn func(sub *Query) *Query) *Query {
	sub := fn(NewQuery())
	if len(sub.WhereClauses) > 0 {
		q.WhereClauses = append(q.WhereClauses, WhereClause{
			Group: &ConditionGroup{
				Connector:  ConnectorAnd,
				Conditions: sub.WhereClauses,
			},
		})
	}
	return q
}

// OrWhereGroup adds a group of OR conditions: WHERE ... OR (sub-conditions)
func (q *Query) OrWhereGroup(fn func(sub *Query) *Query) *Query {
	sub := fn(NewQuery())
	if len(sub.WhereClauses) == 0 {
		return q
	}
	if len(q.WhereClauses) == 0 {
		q.WhereClauses = append(q.WhereClauses, WhereClause{
			Group: &ConditionGroup{
				Connector:  ConnectorAnd,
				Conditions: sub.WhereClauses,
			},
		})
		return q
	}
	existing := q.WhereClauses
	q.WhereClauses = []WhereClause{
		{
			Group: &ConditionGroup{
				Connector: ConnectorOr,
				Conditions: []WhereClause{
					{Group: &ConditionGroup{Connector: ConnectorAnd, Conditions: existing}},
					{Group: &ConditionGroup{Connector: ConnectorAnd, Conditions: sub.WhereClauses}},
				},
			},
		},
	}
	return q
}

// WhereNot negates a group of conditions: WHERE ... AND NOT (sub-conditions)
func (q *Query) WhereNot(fn func(sub *Query) *Query) *Query {
	sub := fn(NewQuery())
	if len(sub.WhereClauses) > 0 {
		q.WhereClauses = append(q.WhereClauses, WhereClause{
			Group: &ConditionGroup{
				Connector:  ConnectorAnd,
				Conditions: sub.WhereClauses,
				Negate:     true,
			},
		})
	}
	return q
}

// ---------------------------------------------------------------------------
// WHERE — raw expressions
// ---------------------------------------------------------------------------

func (q *Query) WhereRawExpr(sql string, values ...any) *Query {
	q.WhereRaw = append(q.WhereRaw, RawExpression{SQL: sql, Values: values})
	return q
}

// ---------------------------------------------------------------------------
// WHERE — subqueries
// ---------------------------------------------------------------------------

func (q *Query) WhereInSubQuery(field string, sub *Query) *Query {
	q.WhereSubQueries = append(q.WhereSubQueries, struct {
		Field    string   `json:"field"`
		Operator string   `json:"op"`
		Sub      SubQuery `json:"sub"`
	}{Field: field, Operator: "in", Sub: SubQuery{Query: sub}})
	return q
}

func (q *Query) WhereNotInSubQuery(field string, sub *Query) *Query {
	q.WhereSubQueries = append(q.WhereSubQueries, struct {
		Field    string   `json:"field"`
		Operator string   `json:"op"`
		Sub      SubQuery `json:"sub"`
	}{Field: field, Operator: "not_in", Sub: SubQuery{Query: sub}})
	return q
}

func (q *Query) WhereExistsQuery(sub *Query) *Query {
	q.WhereExists = append(q.WhereExists, SubQuery{Query: sub})
	return q
}

func (q *Query) WhereNotExistsQuery(sub *Query) *Query {
	q.WhereNotExists = append(q.WhereNotExists, SubQuery{Query: sub})
	return q
}

// ---------------------------------------------------------------------------
// SELECT
// ---------------------------------------------------------------------------

func (q *Query) SetSelect(fields ...string) *Query {
	q.Select = fields
	return q
}

func (q *Query) AddSelect(fields ...string) *Query {
	q.Select = append(q.Select, fields...)
	return q
}

func (q *Query) SelectRawExpr(sql string, values ...any) *Query {
	q.SelectRaw = append(q.SelectRaw, RawExpression{SQL: sql, Values: values})
	return q
}

func (q *Query) SelectAggregate(function, field, alias string) *Query {
	q.Aggregates = append(q.Aggregates, AggregateField{
		Function: strings.ToUpper(function),
		Field:    field,
		Alias:    alias,
	})
	return q
}

func (q *Query) SelectCount(field, alias string) *Query {
	return q.SelectAggregate("COUNT", field, alias)
}

func (q *Query) SelectSum(field, alias string) *Query {
	return q.SelectAggregate("SUM", field, alias)
}

func (q *Query) SelectAvg(field, alias string) *Query {
	return q.SelectAggregate("AVG", field, alias)
}

func (q *Query) SelectMin(field, alias string) *Query {
	return q.SelectAggregate("MIN", field, alias)
}

func (q *Query) SelectMax(field, alias string) *Query {
	return q.SelectAggregate("MAX", field, alias)
}

func (q *Query) SelectCountDistinct(field, alias string) *Query {
	q.Aggregates = append(q.Aggregates, AggregateField{
		Function: "COUNT",
		Field:    field,
		Alias:    alias,
		Distinct: true,
	})
	return q
}

// ---------------------------------------------------------------------------
// DISTINCT
// ---------------------------------------------------------------------------

func (q *Query) SetDistinct(distinct bool) *Query {
	q.Distinct = distinct
	return q
}

// ---------------------------------------------------------------------------
// ORDER BY
// ---------------------------------------------------------------------------

func (q *Query) Order(field, direction string) *Query {
	q.OrderBy = append(q.OrderBy, OrderClause{Field: field, Direction: direction})
	return q
}

func (q *Query) OrderAsc(field string) *Query {
	return q.Order(field, "asc")
}

func (q *Query) OrderDesc(field string) *Query {
	return q.Order(field, "desc")
}

func (q *Query) OrderRawExpr(sql string, values ...any) *Query {
	q.OrderRaw = append(q.OrderRaw, RawExpression{SQL: sql, Values: values})
	return q
}

func (q *Query) Reorder() *Query {
	q.OrderBy = nil
	q.OrderRaw = nil
	return q
}

// ---------------------------------------------------------------------------
// LIMIT / OFFSET
// ---------------------------------------------------------------------------

func (q *Query) SetLimit(limit int) *Query {
	q.Limit = limit
	return q
}

func (q *Query) SetOffset(offset int) *Query {
	q.Offset = offset
	return q
}

// ---------------------------------------------------------------------------
// GROUP BY
// ---------------------------------------------------------------------------

func (q *Query) SetGroupBy(fields ...string) *Query {
	q.GroupBy = fields
	return q
}

func (q *Query) AddGroupBy(fields ...string) *Query {
	q.GroupBy = append(q.GroupBy, fields...)
	return q
}

func (q *Query) GroupRawExpr(sql string, values ...any) *Query {
	q.GroupRaw = append(q.GroupRaw, RawExpression{SQL: sql, Values: values})
	return q
}

// ---------------------------------------------------------------------------
// HAVING
// ---------------------------------------------------------------------------

func (q *Query) HavingCondition(aggregate, field, operator string, value any) *Query {
	q.Having = append(q.Having, HavingClause{
		Aggregate: aggregate,
		Field:     field,
		Operator:  operator,
		Value:     value,
	})
	return q
}

func (q *Query) HavingRawExpr(sql string, values ...any) *Query {
	q.HavingRaw = append(q.HavingRaw, RawExpression{SQL: sql, Values: values})
	return q
}

// ---------------------------------------------------------------------------
// JOINs
// ---------------------------------------------------------------------------

func (q *Query) Join(joinType JoinType, table, localKey, foreignKey string) *Query {
	q.Joins = append(q.Joins, JoinClause{
		Type:       joinType,
		Table:      table,
		LocalKey:   localKey,
		ForeignKey: foreignKey,
	})
	return q
}

func (q *Query) JoinWithAlias(joinType JoinType, table, alias, localKey, foreignKey string) *Query {
	q.Joins = append(q.Joins, JoinClause{
		Type:       joinType,
		Table:      table,
		Alias:      alias,
		LocalKey:   localKey,
		ForeignKey: foreignKey,
	})
	return q
}

func (q *Query) JoinRaw(joinType JoinType, table, rawOn string) *Query {
	q.Joins = append(q.Joins, JoinClause{
		Type:  joinType,
		Table: table,
		RawOn: rawOn,
	})
	return q
}

func (q *Query) InnerJoin(table, localKey, foreignKey string) *Query {
	return q.Join(JoinInner, table, localKey, foreignKey)
}

func (q *Query) LeftJoin(table, localKey, foreignKey string) *Query {
	return q.Join(JoinLeft, table, localKey, foreignKey)
}

func (q *Query) RightJoin(table, localKey, foreignKey string) *Query {
	return q.Join(JoinRight, table, localKey, foreignKey)
}

func (q *Query) CrossJoin(table string) *Query {
	q.Joins = append(q.Joins, JoinClause{
		Type:  JoinCross,
		Table: table,
	})
	return q
}

func (q *Query) FullJoin(table, localKey, foreignKey string) *Query {
	return q.Join(JoinFull, table, localKey, foreignKey)
}

// ---------------------------------------------------------------------------
// WITH / Preload (eager loading)
// ---------------------------------------------------------------------------

func (q *Query) WithRelation(relation string) *Query {
	q.With = append(q.With, WithClause{Relation: relation})
	return q
}

func (q *Query) WithRelationConditions(relation string, conditions []Condition) *Query {
	q.With = append(q.With, WithClause{Relation: relation, Conditions: conditions})
	return q
}

func (q *Query) WithRelationFull(w WithClause) *Query {
	q.With = append(q.With, w)
	return q
}

// ---------------------------------------------------------------------------
// UNION
// ---------------------------------------------------------------------------

func (q *Query) Union(other *Query) *Query {
	q.Unions = append(q.Unions, UnionClause{Query: other, All: false})
	return q
}

func (q *Query) UnionAll(other *Query) *Query {
	q.Unions = append(q.Unions, UnionClause{Query: other, All: true})
	return q
}

// ---------------------------------------------------------------------------
// Locking
// ---------------------------------------------------------------------------

func (q *Query) LockForUpdate() *Query {
	q.Lock = LockForUpdate
	return q
}

func (q *Query) LockForShare() *Query {
	q.Lock = LockForShare
	return q
}

// ---------------------------------------------------------------------------
// Soft delete scopes
// ---------------------------------------------------------------------------

func (q *Query) WithTrashed() *Query {
	q.SoftDeleteScope = ScopeWithTrashed
	return q
}

func (q *Query) OnlyTrashed() *Query {
	q.SoftDeleteScope = ScopeOnlyTrashed
	return q
}

// ---------------------------------------------------------------------------
// Scopes (reusable query modifiers)
// ---------------------------------------------------------------------------

func (q *Query) Scope(fn ScopeFunc) *Query {
	q.scopes = append(q.scopes, fn)
	return q
}

func (q *Query) ApplyScopes() *Query {
	for _, fn := range q.scopes {
		q = fn(q)
	}
	q.scopes = nil
	return q
}

// ---------------------------------------------------------------------------
// Merge legacy Conditions into WhereClauses
// ---------------------------------------------------------------------------

func (q *Query) MergeLegacyConditions() {
	for _, c := range q.Conditions {
		cond := c
		q.WhereClauses = append(q.WhereClauses, WhereClause{Condition: &cond})
	}
	q.Conditions = nil
}

// ---------------------------------------------------------------------------
// GetEffectiveWhereClauses returns all where clauses including legacy
// ---------------------------------------------------------------------------

func (q *Query) GetEffectiveWhereClauses() []WhereClause {
	result := make([]WhereClause, 0, len(q.WhereClauses)+len(q.Conditions))
	for _, c := range q.Conditions {
		cond := c
		result = append(result, WhereClause{Condition: &cond})
	}
	result = append(result, q.WhereClauses...)
	return result
}

// ---------------------------------------------------------------------------
// Field name sanitization
// ---------------------------------------------------------------------------

var safeFieldRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_.]*$`)

func SanitizeFieldName(field string) string {
	if safeFieldRegex.MatchString(field) {
		return field
	}
	return ""
}

func IsSafeFieldName(field string) bool {
	return safeFieldRegex.MatchString(field)
}

// ---------------------------------------------------------------------------
// JSON parsing
// ---------------------------------------------------------------------------

func ParseQueryFromJSON(data []byte) (*Query, error) {
	var q Query
	if err := json.Unmarshal(data, &q); err != nil {
		return nil, fmt.Errorf("invalid query JSON: %w", err)
	}
	return &q, nil
}

// ---------------------------------------------------------------------------
// Map parsing (comprehensive — supports all fields)
// ---------------------------------------------------------------------------

func ParseQueryFromMap(m map[string]any) *Query {
	q := NewQuery()

	// Parse wheres (legacy flat format)
	if wheres, ok := m["wheres"].([]any); ok {
		for _, w := range wheres {
			if wm, ok := w.(map[string]any); ok {
				field, _ := wm["field"].(string)
				if field == "" {
					field, _ = wm["column"].(string)
				}
				if !IsSafeFieldName(field) {
					continue
				}
				op, _ := wm["op"].(string)
				if op == "" {
					op = "="
				}
				q.Where(field, op, wm["value"])
			}
		}
	}

	// Parse where_groups (nested AND/OR/NOT)
	if groups, ok := m["where_groups"].([]any); ok {
		for _, g := range groups {
			if gm, ok := g.(map[string]any); ok {
				clause := parseWhereClauseFromMap(gm)
				if clause != nil {
					q.WhereClauses = append(q.WhereClauses, *clause)
				}
			}
		}
	}

	// Parse order
	if orders, ok := m["order"].([]any); ok {
		for _, o := range orders {
			if om, ok := o.(map[string]any); ok {
				field, _ := om["field"].(string)
				if field == "" {
					field, _ = om["column"].(string)
				}
				if !IsSafeFieldName(field) {
					continue
				}
				dir, _ := om["direction"].(string)
				if dir == "" {
					dir = "asc"
				}
				q.Order(field, dir)
			}
		}
	}

	// Parse limit/offset
	if limit, ok := m["limit"].(float64); ok {
		q.Limit = int(limit)
	}
	if offset, ok := m["offset"].(float64); ok {
		q.Offset = int(offset)
	}

	// Parse select
	if sel, ok := m["select"].([]any); ok {
		for _, s := range sel {
			if str, ok := s.(string); ok && IsSafeFieldName(str) {
				q.Select = append(q.Select, str)
			}
		}
	}

	// Parse group_by
	if gb, ok := m["group_by"].([]any); ok {
		for _, g := range gb {
			if str, ok := g.(string); ok && IsSafeFieldName(str) {
				q.GroupBy = append(q.GroupBy, str)
			}
		}
	}

	// Parse having
	if havings, ok := m["having"].([]any); ok {
		for _, h := range havings {
			if hm, ok := h.(map[string]any); ok {
				hc := HavingClause{}
				hc.Raw, _ = hm["raw"].(string)
				hc.Field, _ = hm["field"].(string)
				hc.Operator, _ = hm["op"].(string)
				hc.Value = hm["value"]
				hc.Aggregate, _ = hm["aggregate"].(string)
				q.Having = append(q.Having, hc)
			}
		}
	}

	// Parse distinct
	if distinct, ok := m["distinct"].(bool); ok {
		q.Distinct = distinct
	}

	// Parse joins
	if joins, ok := m["joins"].([]any); ok {
		for _, j := range joins {
			if jm, ok := j.(map[string]any); ok {
				jc := JoinClause{}
				jt, _ := jm["type"].(string)
				jc.Type = JoinType(strings.ToUpper(jt))
				if jc.Type == "" {
					jc.Type = JoinInner
				}
				jc.Table, _ = jm["table"].(string)
				jc.Alias, _ = jm["alias"].(string)
				jc.LocalKey, _ = jm["local_key"].(string)
				jc.ForeignKey, _ = jm["foreign_key"].(string)
				jc.RawOn, _ = jm["raw_on"].(string)
				if jc.Table != "" {
					q.Joins = append(q.Joins, jc)
				}
			}
		}
	}

	// Parse with (eager loading)
	if withs, ok := m["with"].([]any); ok {
		for _, w := range withs {
			switch wv := w.(type) {
			case string:
				q.With = append(q.With, WithClause{Relation: wv})
			case map[string]any:
				wc := parseWithClauseFromMap(wv)
				if wc != nil {
					q.With = append(q.With, *wc)
				}
			}
		}
	}

	// Parse lock
	if lock, ok := m["lock"].(string); ok {
		switch strings.ToLower(lock) {
		case "for_update", "update":
			q.Lock = LockForUpdate
		case "for_share", "share":
			q.Lock = LockForShare
		}
	}

	// Parse soft_delete_scope
	if scope, ok := m["soft_delete_scope"].(string); ok {
		q.SoftDeleteScope = SoftDeleteScope(scope)
	}

	// Parse aggregates
	if aggs, ok := m["aggregates"].([]any); ok {
		for _, a := range aggs {
			if am, ok := a.(map[string]any); ok {
				af := AggregateField{}
				af.Function, _ = am["function"].(string)
				af.Field, _ = am["field"].(string)
				af.Alias, _ = am["alias"].(string)
				if d, ok := am["distinct"].(bool); ok {
					af.Distinct = d
				}
				if af.Function != "" && af.Field != "" {
					q.Aggregates = append(q.Aggregates, af)
				}
			}
		}
	}

	return q
}

func parseWhereClauseFromMap(m map[string]any) *WhereClause {
	// Single condition
	if field, ok := m["field"].(string); ok && field != "" {
		op, _ := m["op"].(string)
		if op == "" {
			op = "="
		}
		return &WhereClause{
			Condition: &Condition{Field: field, Operator: op, Value: m["value"]},
		}
	}

	// Group
	if conditions, ok := m["conditions"].([]any); ok {
		connector := ConnectorAnd
		if c, ok := m["connector"].(string); ok && strings.ToUpper(c) == "OR" {
			connector = ConnectorOr
		}
		negate, _ := m["negate"].(bool)

		group := &ConditionGroup{
			Connector: connector,
			Negate:    negate,
		}
		for _, c := range conditions {
			if cm, ok := c.(map[string]any); ok {
				clause := parseWhereClauseFromMap(cm)
				if clause != nil {
					group.Conditions = append(group.Conditions, *clause)
				}
			}
		}
		if len(group.Conditions) > 0 {
			return &WhereClause{Group: group}
		}
	}

	return nil
}

func parseWithClauseFromMap(m map[string]any) *WithClause {
	relation, _ := m["relation"].(string)
	if relation == "" {
		return nil
	}
	wc := &WithClause{Relation: relation}

	if conditions, ok := m["conditions"].([]any); ok {
		for _, c := range conditions {
			if cm, ok := c.(map[string]any); ok {
				field, _ := cm["field"].(string)
				op, _ := cm["op"].(string)
				if op == "" {
					op = "="
				}
				wc.Conditions = append(wc.Conditions, Condition{Field: field, Operator: op, Value: cm["value"]})
			}
		}
	}

	if sel, ok := m["select"].([]any); ok {
		for _, s := range sel {
			if str, ok := s.(string); ok {
				wc.Select = append(wc.Select, str)
			}
		}
	}

	if limit, ok := m["limit"].(float64); ok {
		wc.Limit = int(limit)
	}

	if nested, ok := m["nested"].([]any); ok {
		for _, n := range nested {
			if nm, ok := n.(map[string]any); ok {
				nc := parseWithClauseFromMap(nm)
				if nc != nil {
					wc.Nested = append(wc.Nested, *nc)
				}
			}
		}
	}

	return wc
}

// ---------------------------------------------------------------------------
// Domain filter parsing (backward compatible)
// ---------------------------------------------------------------------------

func QueryFromDomain(filters [][]any) *Query {
	q := NewQuery()
	for _, filter := range filters {
		if len(filter) == 3 {
			field, ok1 := filter[0].(string)
			operator, ok2 := filter[1].(string)
			if ok1 && ok2 && IsSafeFieldName(field) {
				q.Where(field, operator, filter[2])
			}
		}
	}
	return q
}
