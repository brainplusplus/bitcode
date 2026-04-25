package runtime

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	"github.com/bitcode-framework/bitcode/internal/infrastructure/persistence"
)

type DynamicFinderResult struct {
	Type     string
	Fields   []DynamicFinderField
	OrderBy  []persistence.OrderClause
	AggField string
}

type DynamicFinderField struct {
	Name      string
	Operator  string
	Connector string
}

func ParseDynamicFinder(operation string) (*DynamicFinderResult, bool) {
	prefixes := []struct {
		prefix string
		typ    string
	}{
		{"FindAllBy", "find_all"},
		{"FindBy", "find_one"},
		{"CountBy", "count"},
		{"ExistsBy", "exists"},
		{"DeleteBy", "delete"},
		{"SumBy", "sum"},
		{"AvgBy", "avg"},
		{"MinBy", "min"},
		{"MaxBy", "max"},
		{"PluckBy", "pluck"},
	}

	var finderType string
	var remainder string

	for _, p := range prefixes {
		if strings.HasPrefix(operation, p.prefix) {
			finderType = p.typ
			remainder = operation[len(p.prefix):]
			break
		}
	}

	if finderType == "" || remainder == "" {
		return nil, false
	}

	var aggField string
	if finderType == "sum" || finderType == "avg" || finderType == "min" || finderType == "max" || finderType == "pluck" {
		parts := splitCamelCaseOnce(remainder)
		if len(parts) < 2 {
			return nil, false
		}
		aggField = camelToSnake(parts[0])
		remainder = parts[1]
	}

	var orderBy []persistence.OrderClause
	orderIdx := strings.Index(remainder, "OrderBy")
	if orderIdx >= 0 {
		orderPart := remainder[orderIdx+7:]
		remainder = remainder[:orderIdx]
		orderBy = parseOrderByPart(orderPart)
	}

	fields := parseFieldsPart(remainder)
	if len(fields) == 0 {
		return nil, false
	}

	return &DynamicFinderResult{
		Type:     finderType,
		Fields:   fields,
		OrderBy:  orderBy,
		AggField: aggField,
	}, true
}

func parseFieldsPart(s string) []DynamicFinderField {
	if s == "" {
		return nil
	}

	var fields []DynamicFinderField
	remaining := s

	for remaining != "" {
		andIdx := indexOfConnector(remaining, "And")
		orIdx := indexOfConnector(remaining, "Or")

		splitIdx := -1
		connector := "AND"
		connLen := 0

		if andIdx > 0 && (orIdx < 0 || andIdx <= orIdx) {
			splitIdx = andIdx
			connector = "AND"
			connLen = 3
		} else if orIdx > 0 {
			splitIdx = orIdx
			connector = "OR"
			connLen = 2
		}

		if splitIdx > 0 {
			field := parseFieldSegment(remaining[:splitIdx])
			if field.Name != "" {
				if len(fields) == 0 {
					field.Connector = "AND"
				}
				fields = append(fields, field)
			}
			remaining = remaining[splitIdx+connLen:]

			if remaining == "" {
				break
			}

			nextAndIdx := indexOfConnector(remaining, "And")
			nextOrIdx := indexOfConnector(remaining, "Or")
			if nextAndIdx < 0 && nextOrIdx < 0 {
				field := parseFieldSegment(remaining)
				if field.Name != "" {
					field.Connector = connector
					fields = append(fields, field)
				}
				break
			}
			continue
		}

		field := parseFieldSegment(remaining)
		if field.Name != "" {
			field.Connector = "AND"
			fields = append(fields, field)
		}
		break
	}

	return fields
}

func indexOfConnector(s, word string) int {
	idx := strings.Index(s, word)
	if idx <= 0 {
		return -1
	}
	afterIdx := idx + len(word)
	if afterIdx < len(s) && unicode.IsLower(rune(s[afterIdx])) {
		return -1
	}
	return idx
}

func parseFieldSegment(s string) DynamicFinderField {
	suffixes := []struct {
		suffix   string
		operator string
	}{
		{"IsNotNull", "is_not_null"},
		{"IsNull", "is_null"},
		{"NotIn", "not_in"},
		{"NotLike", "not_like"},
		{"NotBetween", "not_between"},
		{"Between", "between"},
		{"Like", "like"},
		{"In", "in"},
		{"Gte", ">="},
		{"Gt", ">"},
		{"Lte", "<="},
		{"Lt", "<"},
		{"Not", "!="},
	}

	for _, sf := range suffixes {
		if strings.HasSuffix(s, sf.suffix) {
			fieldName := s[:len(s)-len(sf.suffix)]
			if fieldName == "" {
				continue
			}
			return DynamicFinderField{
				Name:     camelToSnake(fieldName),
				Operator: sf.operator,
			}
		}
	}

	return DynamicFinderField{
		Name:     camelToSnake(s),
		Operator: "=",
	}
}

func parseOrderByPart(s string) []persistence.OrderClause {
	var orders []persistence.OrderClause
	remaining := s
	for remaining != "" {
		andIdx := indexOfConnector(remaining, "And")
		var part string
		if andIdx > 0 {
			part = remaining[:andIdx]
			remaining = remaining[andIdx+3:]
		} else {
			part = remaining
			remaining = ""
		}

		dir := "asc"
		if strings.HasSuffix(part, "Desc") {
			dir = "desc"
			part = part[:len(part)-4]
		} else if strings.HasSuffix(part, "Asc") {
			part = part[:len(part)-3]
		}
		field := camelToSnake(part)
		if field != "" {
			orders = append(orders, persistence.OrderClause{Field: field, Direction: dir})
		}
	}
	return orders
}

func camelToSnake(s string) string {
	if s == "" {
		return ""
	}
	var result []rune
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				result = append(result, '_')
			}
			result = append(result, unicode.ToLower(r))
		} else {
			result = append(result, r)
		}
	}
	return string(result)
}

func splitCamelCaseOnce(s string) []string {
	for i := 1; i < len(s); i++ {
		if unicode.IsUpper(rune(s[i])) {
			return []string{s[:i], s[i:]}
		}
	}
	return []string{s}
}

func (r *DynamicFinderResult) BuildQuery(args map[string]any) *persistence.Query {
	q := persistence.NewQuery()

	for _, f := range r.Fields {
		val := args[f.Name]
		if val == nil {
			val = args["value"]
		}

		switch f.Operator {
		case "is_null":
			q.WhereNull(f.Name)
		case "is_not_null":
			q.WhereNotNull(f.Name)
		case "between", "not_between":
			if vals, ok := val.([]any); ok && len(vals) == 2 {
				q.Where(f.Name, f.Operator, vals)
			}
		default:
			q.Where(f.Name, f.Operator, val)
		}
	}

	for _, o := range r.OrderBy {
		q.Order(o.Field, o.Direction)
	}

	return q
}

func executeDynamicFinder(ctx context.Context, finder *DynamicFinderResult, repo persistence.Repository, args map[string]any) (any, error) {
	query := finder.BuildQuery(args)
	page := intFromMap(args, "page", 1)
	pageSize := intFromMap(args, "page_size", 20)

	switch finder.Type {
	case "find_all":
		records, total, err := repo.FindAll(ctx, query, page, pageSize)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"data":      records,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		}, nil

	case "find_one":
		records, _, err := repo.FindAll(ctx, query, 1, 1)
		if err != nil {
			return nil, err
		}
		if len(records) == 0 {
			return nil, fmt.Errorf("record not found")
		}
		return records[0], nil

	case "count":
		return repo.Count(ctx, query)

	case "exists":
		return repo.Exists(ctx, query)

	case "delete":
		records, _, err := repo.FindAll(ctx, query, 1, 1000)
		if err != nil {
			return nil, err
		}
		deleted := 0
		for _, rec := range records {
			if id, ok := rec["id"]; ok {
				if err := repo.Delete(ctx, fmt.Sprintf("%v", id)); err == nil {
					deleted++
				}
			}
		}
		return deleted, nil

	case "sum":
		return repo.Sum(ctx, finder.AggField, query)

	case "avg":
		return repo.Avg(ctx, finder.AggField, query)

	case "min":
		return repo.Min(ctx, finder.AggField, query)

	case "max":
		return repo.Max(ctx, finder.AggField, query)

	case "pluck":
		return repo.Pluck(ctx, finder.AggField, query)

	default:
		return nil, fmt.Errorf("unknown dynamic finder type: %s", finder.Type)
	}
}
