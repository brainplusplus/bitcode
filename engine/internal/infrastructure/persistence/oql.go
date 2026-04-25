package persistence

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// ---------------------------------------------------------------------------
// Style A: SQL-like OQL (JPQL/HQL style)
// Example: "SELECT name, email FROM contacts LEFT JOIN companies ON contacts.company_id = companies.id WHERE status = 'active' AND (city = 'Jakarta' OR city = 'Bandung') ORDER BY name ASC LIMIT 10 WITH company"
// ---------------------------------------------------------------------------

func ParseOQLSQL(input string) (*Query, string, error) {
	p := &sqlParser{input: strings.TrimSpace(input), pos: 0}
	return p.parse()
}

type sqlParser struct {
	input     string
	pos       int
	lastMatch string
}

func (p *sqlParser) parse() (*Query, string, error) {
	q := NewQuery()
	model := ""

	p.skipWhitespace()

	if p.matchKeyword("SELECT") {
		fields := p.parseCSVUntilKeyword("FROM")
		if len(fields) > 0 && !(len(fields) == 1 && fields[0] == "*") {
			q.SetSelect(fields...)
		}
	}

	if p.matchKeyword("FROM") {
		model = p.parseIdentifier()
	} else {
		model = p.parseIdentifier()
	}

	if model == "" {
		return nil, "", fmt.Errorf("OQL: model/table name required")
	}

	for p.matchKeyword("LEFT") || p.matchKeyword("RIGHT") || p.matchKeyword("INNER") || p.matchKeyword("CROSS") || p.matchKeyword("FULL") || p.matchKeyword("JOIN") {
		p.pos -= len(p.lastMatch)
		join, err := p.parseJoin()
		if err != nil {
			return nil, model, err
		}
		q.Joins = append(q.Joins, join)
	}

	if p.matchKeyword("WHERE") {
		clauses, err := p.parseWhereExpr()
		if err != nil {
			return nil, model, err
		}
		q.WhereClauses = append(q.WhereClauses, clauses...)
	}

	if p.matchKeyword("GROUP") {
		p.matchKeyword("BY")
		fields := p.parseCSVUntilKeyword("HAVING", "ORDER", "LIMIT", "OFFSET", "WITH", "UNION", "LOCK")
		q.SetGroupBy(fields...)
	}

	if p.matchKeyword("HAVING") {
		raw := p.parseUntilKeyword("ORDER", "LIMIT", "OFFSET", "WITH", "UNION", "LOCK")
		if raw != "" {
			q.HavingRaw = append(q.HavingRaw, RawExpression{SQL: raw})
		}
	}

	if p.matchKeyword("ORDER") {
		p.matchKeyword("BY")
		p.parseOrderBy(q)
	}

	if p.matchKeyword("LIMIT") {
		n := p.parseNumber()
		q.SetLimit(n)
	}

	if p.matchKeyword("OFFSET") {
		n := p.parseNumber()
		q.SetOffset(n)
	}

	if p.matchKeyword("WITH") {
		relations := p.parseCSVUntilKeyword("UNION", "LOCK", "FOR")
		for _, rel := range relations {
			q.WithRelation(strings.TrimSpace(rel))
		}
	}

	if p.matchKeyword("LOCK") {
		if p.matchKeyword("FOR") {
			if p.matchKeyword("UPDATE") {
				q.LockForUpdate()
			} else if p.matchKeyword("SHARE") {
				q.LockForShare()
			}
		}
	} else if p.matchKeyword("FOR") {
		if p.matchKeyword("UPDATE") {
			q.LockForUpdate()
		} else if p.matchKeyword("SHARE") {
			q.LockForShare()
		}
	}

	return q, model, nil
}

func (p *sqlParser) skipWhitespace() {
	for p.pos < len(p.input) && unicode.IsSpace(rune(p.input[p.pos])) {
		p.pos++
	}
}

func (p *sqlParser) matchKeyword(kw string) bool {
	p.skipWhitespace()
	upper := strings.ToUpper(p.remaining())
	if strings.HasPrefix(upper, kw) {
		next := p.pos + len(kw)
		if next >= len(p.input) || !unicode.IsLetter(rune(p.input[next])) {
			p.lastMatch = kw
			p.pos = next
			return true
		}
	}
	return false
}

func (p *sqlParser) remaining() string {
	if p.pos >= len(p.input) {
		return ""
	}
	return p.input[p.pos:]
}

func (p *sqlParser) parseIdentifier() string {
	p.skipWhitespace()
	start := p.pos
	for p.pos < len(p.input) {
		ch := rune(p.input[p.pos])
		if unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '_' || ch == '.' {
			p.pos++
		} else {
			break
		}
	}
	return p.input[start:p.pos]
}

func (p *sqlParser) parseCSVUntilKeyword(keywords ...string) []string {
	p.skipWhitespace()
	start := p.pos
	for p.pos < len(p.input) {
		upper := strings.ToUpper(p.remaining())
		for _, kw := range keywords {
			if strings.HasPrefix(upper, kw) {
				next := p.pos + len(kw)
				if next >= len(p.input) || !unicode.IsLetter(rune(p.input[next])) {
					raw := strings.TrimSpace(p.input[start:p.pos])
					return splitCSV(raw)
				}
			}
		}
		p.pos++
	}
	raw := strings.TrimSpace(p.input[start:p.pos])
	if raw == "" {
		return nil
	}
	return splitCSV(raw)
}

func (p *sqlParser) parseUntilKeyword(keywords ...string) string {
	p.skipWhitespace()
	start := p.pos
	for p.pos < len(p.input) {
		upper := strings.ToUpper(p.remaining())
		for _, kw := range keywords {
			if strings.HasPrefix(upper, kw) {
				next := p.pos + len(kw)
				if next >= len(p.input) || !unicode.IsLetter(rune(p.input[next])) {
					return strings.TrimSpace(p.input[start:p.pos])
				}
			}
		}
		p.pos++
	}
	return strings.TrimSpace(p.input[start:p.pos])
}

func (p *sqlParser) parseNumber() int {
	p.skipWhitespace()
	start := p.pos
	for p.pos < len(p.input) && unicode.IsDigit(rune(p.input[p.pos])) {
		p.pos++
	}
	n, _ := strconv.Atoi(p.input[start:p.pos])
	return n
}

func (p *sqlParser) parseJoin() (JoinClause, error) {
	p.skipWhitespace()
	jc := JoinClause{}

	upper := strings.ToUpper(p.remaining())
	if strings.HasPrefix(upper, "LEFT") {
		jc.Type = JoinLeft
		p.pos += 4
	} else if strings.HasPrefix(upper, "RIGHT") {
		jc.Type = JoinRight
		p.pos += 5
	} else if strings.HasPrefix(upper, "INNER") {
		jc.Type = JoinInner
		p.pos += 5
	} else if strings.HasPrefix(upper, "CROSS") {
		jc.Type = JoinCross
		p.pos += 5
	} else if strings.HasPrefix(upper, "FULL") {
		jc.Type = JoinFull
		p.pos += 4
	}

	p.matchKeyword("OUTER")
	p.matchKeyword("JOIN")

	jc.Table = p.parseIdentifier()

	if p.matchKeyword("AS") {
		jc.Alias = p.parseIdentifier()
	} else {
		p.skipWhitespace()
		if p.pos < len(p.input) && !strings.HasPrefix(strings.ToUpper(p.remaining()), "ON") {
			peek := p.parseIdentifier()
			if strings.ToUpper(peek) != "ON" && peek != "" {
				jc.Alias = peek
			} else {
				p.pos -= len(peek)
			}
		}
	}

	if jc.Type == JoinCross {
		return jc, nil
	}

	if !p.matchKeyword("ON") {
		return jc, fmt.Errorf("OQL: expected ON after JOIN table")
	}

	jc.LocalKey = p.parseIdentifier()
	p.skipWhitespace()
	if p.pos < len(p.input) && p.input[p.pos] == '=' {
		p.pos++
	}
	jc.ForeignKey = p.parseIdentifier()

	return jc, nil
}

func (p *sqlParser) parseWhereExpr() ([]WhereClause, error) {
	var clauses []WhereClause

	for {
		p.skipWhitespace()
		if p.pos >= len(p.input) {
			break
		}

		upper := strings.ToUpper(p.remaining())
		if strings.HasPrefix(upper, "GROUP") || strings.HasPrefix(upper, "ORDER") ||
			strings.HasPrefix(upper, "LIMIT") || strings.HasPrefix(upper, "OFFSET") ||
			strings.HasPrefix(upper, "HAVING") || strings.HasPrefix(upper, "WITH") ||
			strings.HasPrefix(upper, "UNION") || strings.HasPrefix(upper, "LOCK") ||
			strings.HasPrefix(upper, "FOR") {
			break
		}

		if p.input[p.pos] == '(' {
			p.pos++
			subClauses, err := p.parseWhereExpr()
			if err != nil {
				return nil, err
			}
			p.skipWhitespace()
			if p.pos < len(p.input) && p.input[p.pos] == ')' {
				p.pos++
			}
			clauses = append(clauses, WhereClause{
				Group: &ConditionGroup{
					Connector:  ConnectorAnd,
					Conditions: subClauses,
				},
			})
		} else {
			cond, err := p.parseSingleCondition()
			if err != nil {
				return nil, err
			}
			clauses = append(clauses, WhereClause{Condition: cond})
		}

		p.skipWhitespace()
		if p.matchKeyword("AND") {
			continue
		}
		if p.matchKeyword("OR") {
			nextClauses, err := p.parseWhereExpr()
			if err != nil {
				return nil, err
			}
			return []WhereClause{
				{
					Group: &ConditionGroup{
						Connector:  ConnectorOr,
						Conditions: append(clauses, nextClauses...),
					},
				},
			}, nil
		}
		break
	}

	return clauses, nil
}

func (p *sqlParser) parseSingleCondition() (*Condition, error) {
	field := p.parseIdentifier()
	if field == "" {
		return nil, fmt.Errorf("OQL: expected field name at position %d", p.pos)
	}

	p.skipWhitespace()
	op := p.parseOperator()
	if op == "" {
		return nil, fmt.Errorf("OQL: expected operator after field %q", field)
	}

	p.skipWhitespace()

	opLower := strings.ToLower(op)
	switch opLower {
	case "is":
		p.skipWhitespace()
		if p.matchKeyword("NOT") {
			p.matchKeyword("NULL")
			return &Condition{Field: field, Operator: "is_not_null"}, nil
		}
		p.matchKeyword("NULL")
		return &Condition{Field: field, Operator: "is_null"}, nil
	case "in", "not in":
		vals := p.parseValueList()
		return &Condition{Field: field, Operator: strings.ReplaceAll(opLower, " ", "_"), Value: vals}, nil
	case "between", "not between":
		v1 := p.parseValue()
		p.matchKeyword("AND")
		v2 := p.parseValue()
		return &Condition{Field: field, Operator: strings.ReplaceAll(opLower, " ", "_"), Value: []any{v1, v2}}, nil
	case "like", "not like":
		val := p.parseValue()
		return &Condition{Field: field, Operator: strings.ReplaceAll(opLower, " ", "_"), Value: val}, nil
	default:
		val := p.parseValue()
		return &Condition{Field: field, Operator: opLower, Value: val}, nil
	}
}

func (p *sqlParser) parseOperator() string {
	p.skipWhitespace()
	start := p.pos

	twoChar := []string{"!=", ">=", "<=", "<>"}
	if p.pos+1 < len(p.input) {
		tc := p.input[p.pos : p.pos+2]
		for _, op := range twoChar {
			if tc == op {
				p.pos += 2
				if op == "<>" {
					return "!="
				}
				return op
			}
		}
	}

	oneChar := []string{"=", ">", "<"}
	if p.pos < len(p.input) {
		oc := string(p.input[p.pos])
		for _, op := range oneChar {
			if oc == op {
				p.pos++
				return op
			}
		}
	}

	upper := strings.ToUpper(p.remaining())
	kwOps := []string{"NOT BETWEEN", "NOT LIKE", "NOT IN", "BETWEEN", "LIKE", "IN", "IS"}
	for _, kw := range kwOps {
		if strings.HasPrefix(upper, kw) {
			next := p.pos + len(kw)
			if next >= len(p.input) || !unicode.IsLetter(rune(p.input[next])) {
				p.pos = next
				return strings.ToLower(kw)
			}
		}
	}

	p.pos = start
	return ""
}

func (p *sqlParser) parseValue() any {
	p.skipWhitespace()
	if p.pos >= len(p.input) {
		return nil
	}

	ch := p.input[p.pos]

	if ch == '\'' || ch == '"' {
		return p.parseString(ch)
	}

	upper := strings.ToUpper(p.remaining())
	if strings.HasPrefix(upper, "NULL") {
		p.pos += 4
		return nil
	}
	if strings.HasPrefix(upper, "TRUE") {
		p.pos += 4
		return true
	}
	if strings.HasPrefix(upper, "FALSE") {
		p.pos += 5
		return false
	}

	start := p.pos
	hasDecimal := false
	for p.pos < len(p.input) {
		c := p.input[p.pos]
		if c == '.' && !hasDecimal {
			hasDecimal = true
			p.pos++
		} else if c >= '0' && c <= '9' {
			p.pos++
		} else if c == '-' && p.pos == start {
			p.pos++
		} else {
			break
		}
	}
	if p.pos > start {
		numStr := p.input[start:p.pos]
		if hasDecimal {
			f, _ := strconv.ParseFloat(numStr, 64)
			return f
		}
		n, _ := strconv.ParseInt(numStr, 10, 64)
		return n
	}

	return p.parseIdentifier()
}

func (p *sqlParser) parseString(quote byte) string {
	p.pos++
	start := p.pos
	for p.pos < len(p.input) && p.input[p.pos] != quote {
		if p.input[p.pos] == '\\' {
			p.pos++
		}
		p.pos++
	}
	result := p.input[start:p.pos]
	if p.pos < len(p.input) {
		p.pos++
	}
	return result
}

func (p *sqlParser) parseValueList() []any {
	p.skipWhitespace()
	if p.pos < len(p.input) && p.input[p.pos] == '(' {
		p.pos++
	}
	var vals []any
	for {
		p.skipWhitespace()
		if p.pos >= len(p.input) || p.input[p.pos] == ')' {
			break
		}
		val := p.parseValue()
		vals = append(vals, val)
		p.skipWhitespace()
		if p.pos < len(p.input) && p.input[p.pos] == ',' {
			p.pos++
		}
	}
	if p.pos < len(p.input) && p.input[p.pos] == ')' {
		p.pos++
	}
	return vals
}

func (p *sqlParser) parseOrderBy(q *Query) {
	for {
		p.skipWhitespace()
		field := p.parseIdentifier()
		if field == "" {
			break
		}
		dir := "asc"
		if p.matchKeyword("DESC") {
			dir = "desc"
		} else {
			p.matchKeyword("ASC")
		}
		q.Order(field, dir)
		p.skipWhitespace()
		if p.pos < len(p.input) && p.input[p.pos] == ',' {
			p.pos++
		} else {
			break
		}
	}
}

// ---------------------------------------------------------------------------
// Style B: Simplified DSL
// Example: "contacts[status='active', city IN ('Jakarta','Bandung')] ORDER BY name WITH company LIMIT 10"
// ---------------------------------------------------------------------------

func ParseOQLSimple(input string) (*Query, string, error) {
	input = strings.TrimSpace(input)
	q := NewQuery()

	bracketStart := strings.Index(input, "[")
	var model string
	var rest string

	if bracketStart >= 0 {
		model = strings.TrimSpace(input[:bracketStart])
		bracketEnd := findMatchingBracket(input, bracketStart)
		if bracketEnd < 0 {
			return nil, "", fmt.Errorf("OQL: unmatched '[' in expression")
		}
		filterStr := input[bracketStart+1 : bracketEnd]
		rest = strings.TrimSpace(input[bracketEnd+1:])

		conditions, err := parseSimpleConditions(filterStr)
		if err != nil {
			return nil, model, err
		}
		q.WhereClauses = conditions
	} else {
		parts := strings.Fields(input)
		if len(parts) == 0 {
			return nil, "", fmt.Errorf("OQL: model name required")
		}
		model = parts[0]
		rest = strings.TrimSpace(strings.TrimPrefix(input, model))
	}

	sp := &sqlParser{input: rest, pos: 0}

	if sp.matchKeyword("ORDER") {
		sp.matchKeyword("BY")
		sp.parseOrderBy(q)
	}

	if sp.matchKeyword("LIMIT") {
		q.SetLimit(sp.parseNumber())
	}

	if sp.matchKeyword("OFFSET") {
		q.SetOffset(sp.parseNumber())
	}

	if sp.matchKeyword("WITH") {
		relations := sp.parseCSVUntilKeyword("LIMIT", "OFFSET", "LOCK", "FOR")
		for _, rel := range relations {
			q.WithRelation(strings.TrimSpace(rel))
		}
	}

	return q, model, nil
}

func findMatchingBracket(s string, start int) int {
	depth := 0
	inStr := false
	strChar := byte(0)
	for i := start; i < len(s); i++ {
		if inStr {
			if s[i] == strChar && (i == 0 || s[i-1] != '\\') {
				inStr = false
			}
			continue
		}
		if s[i] == '\'' || s[i] == '"' {
			inStr = true
			strChar = s[i]
			continue
		}
		if s[i] == '[' || s[i] == '(' {
			depth++
		} else if s[i] == ']' || s[i] == ')' {
			depth--
			if depth == 0 && s[i] == ']' {
				return i
			}
		}
	}
	return -1
}

func parseSimpleConditions(input string) ([]WhereClause, error) {
	parts := splitRespectingParens(input, ',')
	var clauses []WhereClause

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if strings.Contains(part, " OR ") || strings.Contains(part, " or ") {
			orParts := splitRespectingParens(part, '|')
			if len(orParts) <= 1 {
				idx := strings.Index(strings.ToUpper(part), " OR ")
				if idx >= 0 {
					orParts = []string{part[:idx], part[idx+4:]}
				}
			}
			if len(orParts) > 1 {
				var orClauses []WhereClause
				for _, op := range orParts {
					subClauses, err := parseSimpleConditions(strings.TrimSpace(op))
					if err != nil {
						return nil, err
					}
					orClauses = append(orClauses, subClauses...)
				}
				clauses = append(clauses, WhereClause{
					Group: &ConditionGroup{
						Connector:  ConnectorOr,
						Conditions: orClauses,
					},
				})
				continue
			}
		}

		sp := &sqlParser{input: part, pos: 0}
		cond, err := sp.parseSingleCondition()
		if err != nil {
			return nil, err
		}
		clauses = append(clauses, WhereClause{Condition: cond})
	}

	return clauses, nil
}

func splitRespectingParens(s string, sep byte) []string {
	var parts []string
	depth := 0
	inStr := false
	strChar := byte(0)
	start := 0

	for i := 0; i < len(s); i++ {
		if inStr {
			if s[i] == strChar && (i == 0 || s[i-1] != '\\') {
				inStr = false
			}
			continue
		}
		if s[i] == '\'' || s[i] == '"' {
			inStr = true
			strChar = s[i]
			continue
		}
		if s[i] == '(' || s[i] == '[' {
			depth++
		} else if s[i] == ')' || s[i] == ']' {
			depth--
		} else if s[i] == sep && depth == 0 {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	parts = append(parts, s[start:])
	return parts
}

// ---------------------------------------------------------------------------
// Style C: Dot-notation / method chain style
// Example: "contacts.where(status.eq('active')).where(city.in('Jakarta','Bandung')).orderBy('name').with('company').limit(10)"
// ---------------------------------------------------------------------------

func ParseOQLDot(input string) (*Query, string, error) {
	input = strings.TrimSpace(input)
	q := NewQuery()

	parts := strings.SplitN(input, ".", 2)
	model := parts[0]
	if len(parts) < 2 {
		return q, model, nil
	}

	chain := parts[1]
	calls := parseDotCalls(chain)

	for _, call := range calls {
		method := strings.ToLower(call.name)
		switch method {
		case "where":
			for _, arg := range call.args {
				cond := parseDotCondition(arg)
				if cond != nil {
					q.WhereClauses = append(q.WhereClauses, WhereClause{Condition: cond})
				}
			}
		case "orwhere":
			if len(call.args) > 0 {
				cond := parseDotCondition(call.args[0])
				if cond != nil {
					q.OrWhere(cond.Field, cond.Operator, cond.Value)
				}
			}
		case "orderby", "order":
			for _, arg := range call.args {
				arg = strings.Trim(arg, "'\"")
				parts := strings.Fields(arg)
				field := parts[0]
				dir := "asc"
				if len(parts) > 1 && strings.ToLower(parts[1]) == "desc" {
					dir = "desc"
				}
				q.Order(field, dir)
			}
		case "limit":
			if len(call.args) > 0 {
				n, _ := strconv.Atoi(strings.TrimSpace(call.args[0]))
				q.SetLimit(n)
			}
		case "offset":
			if len(call.args) > 0 {
				n, _ := strconv.Atoi(strings.TrimSpace(call.args[0]))
				q.SetOffset(n)
			}
		case "with":
			for _, arg := range call.args {
				q.WithRelation(strings.Trim(arg, "'\""))
			}
		case "select":
			for _, arg := range call.args {
				q.AddSelect(strings.Trim(arg, "'\""))
			}
		case "groupby", "group":
			for _, arg := range call.args {
				q.AddGroupBy(strings.Trim(arg, "'\""))
			}
		case "distinct":
			q.SetDistinct(true)
		case "lock":
			if len(call.args) > 0 {
				lockType := strings.Trim(call.args[0], "'\"")
				if strings.ToLower(lockType) == "update" {
					q.LockForUpdate()
				} else {
					q.LockForShare()
				}
			}
		}
	}

	return q, model, nil
}

type dotCall struct {
	name string
	args []string
}

func parseDotCalls(chain string) []dotCall {
	var calls []dotCall
	i := 0
	for i < len(chain) {
		start := i
		for i < len(chain) && chain[i] != '(' && chain[i] != '.' {
			i++
		}
		name := chain[start:i]
		if name == "" {
			i++
			continue
		}

		var args []string
		if i < len(chain) && chain[i] == '(' {
			i++
			depth := 1
			argStart := i
			for i < len(chain) && depth > 0 {
				if chain[i] == '(' {
					depth++
				} else if chain[i] == ')' {
					depth--
				}
				if depth > 0 {
					i++
				}
			}
			argStr := chain[argStart:i]
			if argStr != "" {
				args = splitRespectingParens(argStr, ',')
				for j := range args {
					args[j] = strings.TrimSpace(args[j])
				}
			}
			if i < len(chain) {
				i++
			}
		}

		if i < len(chain) && chain[i] == '.' {
			i++
		}

		calls = append(calls, dotCall{name: name, args: args})
	}
	return calls
}

func parseDotCondition(expr string) *Condition {
	expr = strings.TrimSpace(expr)

	dotIdx := strings.Index(expr, ".")
	if dotIdx < 0 {
		return nil
	}

	field := expr[:dotIdx]
	rest := expr[dotIdx+1:]

	parenIdx := strings.Index(rest, "(")
	if parenIdx < 0 {
		return nil
	}

	method := strings.ToLower(rest[:parenIdx])
	valueStr := rest[parenIdx+1:]
	if strings.HasSuffix(valueStr, ")") {
		valueStr = valueStr[:len(valueStr)-1]
	}

	switch method {
	case "eq":
		return &Condition{Field: field, Operator: "=", Value: parseDotValue(valueStr)}
	case "ne", "neq":
		return &Condition{Field: field, Operator: "!=", Value: parseDotValue(valueStr)}
	case "gt":
		return &Condition{Field: field, Operator: ">", Value: parseDotValue(valueStr)}
	case "gte":
		return &Condition{Field: field, Operator: ">=", Value: parseDotValue(valueStr)}
	case "lt":
		return &Condition{Field: field, Operator: "<", Value: parseDotValue(valueStr)}
	case "lte":
		return &Condition{Field: field, Operator: "<=", Value: parseDotValue(valueStr)}
	case "like":
		return &Condition{Field: field, Operator: "like", Value: parseDotValue(valueStr)}
	case "in":
		return &Condition{Field: field, Operator: "in", Value: parseDotValues(valueStr)}
	case "notin":
		return &Condition{Field: field, Operator: "not_in", Value: parseDotValues(valueStr)}
	case "isnull", "null":
		return &Condition{Field: field, Operator: "is_null"}
	case "isnotnull", "notnull":
		return &Condition{Field: field, Operator: "is_not_null"}
	case "between":
		vals := parseDotValues(valueStr)
		return &Condition{Field: field, Operator: "between", Value: vals}
	}
	return nil
}

func parseDotValue(s string) any {
	s = strings.TrimSpace(s)
	if (strings.HasPrefix(s, "'") && strings.HasSuffix(s, "'")) ||
		(strings.HasPrefix(s, "\"") && strings.HasSuffix(s, "\"")) {
		return s[1 : len(s)-1]
	}
	if s == "null" || s == "nil" {
		return nil
	}
	if s == "true" {
		return true
	}
	if s == "false" {
		return false
	}
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		return n
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}
	return s
}

func parseDotValues(s string) []any {
	parts := splitRespectingParens(s, ',')
	var vals []any
	for _, p := range parts {
		vals = append(vals, parseDotValue(strings.TrimSpace(p)))
	}
	return vals
}

// ---------------------------------------------------------------------------
// Auto-detect OQL style and parse
// ---------------------------------------------------------------------------

func ParseOQL(input string) (*Query, string, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, "", fmt.Errorf("OQL: empty input")
	}

	upper := strings.ToUpper(input)
	if strings.HasPrefix(upper, "SELECT ") || strings.HasPrefix(upper, "FROM ") {
		return ParseOQLSQL(input)
	}

	if strings.Contains(input, ".where(") || strings.Contains(input, ".orderBy(") ||
		strings.Contains(input, ".limit(") || strings.Contains(input, ".with(") ||
		strings.Contains(input, ".select(") || strings.Contains(input, ".groupBy(") {
		return ParseOQLDot(input)
	}

	if strings.Contains(input, "[") {
		return ParseOQLSimple(input)
	}

	if strings.Contains(upper, " WHERE ") || strings.Contains(upper, " JOIN ") ||
		strings.Contains(upper, " ORDER ") || strings.Contains(upper, " LIMIT ") {
		return ParseOQLSQL(input)
	}

	return ParseOQLSimple(input)
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
