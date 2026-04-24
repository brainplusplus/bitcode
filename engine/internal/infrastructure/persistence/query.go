package persistence

import (
	"encoding/json"
	"fmt"
)

type Condition struct {
	Field    string `json:"field"`
	Operator string `json:"op"`
	Value    any    `json:"value"`
}

type OrderClause struct {
	Field     string `json:"field"`
	Direction string `json:"direction"`
}

type Query struct {
	Conditions []Condition   `json:"wheres,omitempty"`
	OrderBy    []OrderClause `json:"order,omitempty"`
	Limit      int           `json:"limit,omitempty"`
	Offset     int           `json:"offset,omitempty"`
	Select     []string      `json:"select,omitempty"`
	GroupBy    []string      `json:"group_by,omitempty"`
}

func NewQuery() *Query {
	return &Query{}
}

func (q *Query) Where(field, operator string, value any) *Query {
	q.Conditions = append(q.Conditions, Condition{Field: field, Operator: operator, Value: value})
	return q
}

func (q *Query) Order(field, direction string) *Query {
	q.OrderBy = append(q.OrderBy, OrderClause{Field: field, Direction: direction})
	return q
}

func (q *Query) SetLimit(limit int) *Query {
	q.Limit = limit
	return q
}

func (q *Query) SetOffset(offset int) *Query {
	q.Offset = offset
	return q
}

func (q *Query) SetSelect(fields ...string) *Query {
	q.Select = fields
	return q
}

func (q *Query) SetGroupBy(fields ...string) *Query {
	q.GroupBy = fields
	return q
}

func ParseQueryFromJSON(data []byte) (*Query, error) {
	var q Query
	if err := json.Unmarshal(data, &q); err != nil {
		return nil, fmt.Errorf("invalid query JSON: %w", err)
	}
	return &q, nil
}

func ParseQueryFromMap(m map[string]any) *Query {
	q := NewQuery()

	if wheres, ok := m["wheres"].([]any); ok {
		for _, w := range wheres {
			if wm, ok := w.(map[string]any); ok {
				field, _ := wm["field"].(string)
				if field == "" {
					field, _ = wm["column"].(string)
				}
				op, _ := wm["op"].(string)
				if op == "" {
					op = "="
				}
				q.Where(field, op, wm["value"])
			}
		}
	}

	if orders, ok := m["order"].([]any); ok {
		for _, o := range orders {
			if om, ok := o.(map[string]any); ok {
				field, _ := om["field"].(string)
				if field == "" {
					field, _ = om["column"].(string)
				}
				dir, _ := om["direction"].(string)
				if dir == "" {
					dir = "asc"
				}
				q.Order(field, dir)
			}
		}
	}

	if limit, ok := m["limit"].(float64); ok {
		q.Limit = int(limit)
	}
	if offset, ok := m["offset"].(float64); ok {
		q.Offset = int(offset)
	}

	return q
}

func QueryFromDomain(filters [][]any) *Query {
	q := NewQuery()
	for _, filter := range filters {
		if len(filter) == 3 {
			field, ok1 := filter[0].(string)
			operator, ok2 := filter[1].(string)
			if ok1 && ok2 {
				q.Where(field, operator, filter[2])
			}
		}
	}
	return q
}
