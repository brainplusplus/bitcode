package expression

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"
)

type tokenKind int

const (
	tokNumber tokenKind = iota
	tokIdent
	tokString
	tokPlus
	tokMinus
	tokStar
	tokSlash
	tokPercent
	tokLParen
	tokRParen
	tokComma
	tokDot
	tokEQ
	tokNE
	tokLT
	tokLE
	tokGT
	tokGE
	tokAnd
	tokOr
	tokNot
	tokEOF
)

type token struct {
	kind tokenKind
	sval string
	nval float64
}

type lexer struct {
	input []rune
	pos   int
}

func newLexer(input string) *lexer {
	return &lexer{input: []rune(input), pos: 0}
}

func (l *lexer) peek() rune {
	if l.pos >= len(l.input) {
		return 0
	}
	return l.input[l.pos]
}

func (l *lexer) advance() rune {
	ch := l.input[l.pos]
	l.pos++
	return ch
}

func (l *lexer) next() token {
	for l.pos < len(l.input) && unicode.IsSpace(l.input[l.pos]) {
		l.pos++
	}
	if l.pos >= len(l.input) {
		return token{kind: tokEOF}
	}

	ch := l.peek()

	if ch == '\'' || ch == '"' {
		return l.readString(ch)
	}

	if unicode.IsDigit(ch) || (ch == '.' && l.pos+1 < len(l.input) && unicode.IsDigit(l.input[l.pos+1])) {
		return l.readNumber()
	}

	if unicode.IsLetter(ch) || ch == '_' {
		return l.readIdent()
	}

	switch ch {
	case '+':
		l.advance()
		return token{kind: tokPlus}
	case '-':
		l.advance()
		return token{kind: tokMinus}
	case '*':
		l.advance()
		return token{kind: tokStar}
	case '/':
		l.advance()
		return token{kind: tokSlash}
	case '%':
		l.advance()
		return token{kind: tokPercent}
	case '(':
		l.advance()
		return token{kind: tokLParen}
	case ')':
		l.advance()
		return token{kind: tokRParen}
	case ',':
		l.advance()
		return token{kind: tokComma}
	case '.':
		l.advance()
		return token{kind: tokDot}
	case '=':
		l.advance()
		if l.pos < len(l.input) && l.input[l.pos] == '=' {
			l.advance()
		}
		return token{kind: tokEQ}
	case '!':
		l.advance()
		if l.pos < len(l.input) && l.input[l.pos] == '=' {
			l.advance()
			return token{kind: tokNE}
		}
		return token{kind: tokNot}
	case '<':
		l.advance()
		if l.pos < len(l.input) && l.input[l.pos] == '=' {
			l.advance()
			return token{kind: tokLE}
		}
		return token{kind: tokLT}
	case '>':
		l.advance()
		if l.pos < len(l.input) && l.input[l.pos] == '=' {
			l.advance()
			return token{kind: tokGE}
		}
		return token{kind: tokGT}
	case '&':
		l.advance()
		if l.pos < len(l.input) && l.input[l.pos] == '&' {
			l.advance()
		}
		return token{kind: tokAnd}
	case '|':
		l.advance()
		if l.pos < len(l.input) && l.input[l.pos] == '|' {
			l.advance()
		}
		return token{kind: tokOr}
	}

	l.advance()
	return token{kind: tokEOF}
}

func (l *lexer) readNumber() token {
	start := l.pos
	for l.pos < len(l.input) && (unicode.IsDigit(l.input[l.pos]) || l.input[l.pos] == '.') {
		l.pos++
	}
	s := string(l.input[start:l.pos])
	n, _ := strconv.ParseFloat(s, 64)
	return token{kind: tokNumber, nval: n, sval: s}
}

func (l *lexer) readIdent() token {
	start := l.pos
	for l.pos < len(l.input) && (unicode.IsLetter(l.input[l.pos]) || unicode.IsDigit(l.input[l.pos]) || l.input[l.pos] == '_') {
		l.pos++
	}
	s := string(l.input[start:l.pos])
	return token{kind: tokIdent, sval: s}
}

func (l *lexer) readString(quote rune) token {
	l.advance()
	start := l.pos
	for l.pos < len(l.input) && l.input[l.pos] != quote {
		l.pos++
	}
	s := string(l.input[start:l.pos])
	if l.pos < len(l.input) {
		l.advance()
	}
	return token{kind: tokString, sval: s}
}

type exprParser struct {
	lex     *lexer
	current token
}

func newExprParser(input string) *exprParser {
	p := &exprParser{lex: newLexer(input)}
	p.current = p.lex.next()
	return p
}

func (p *exprParser) eat(kind tokenKind) token {
	t := p.current
	if t.kind != kind {
		return t
	}
	p.current = p.lex.next()
	return t
}

func (p *exprParser) parseExpr() (node, error) {
	return p.parseOr()
}

func (p *exprParser) parseOr() (node, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	for p.current.kind == tokOr {
		p.eat(tokOr)
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = &binaryNode{op: "||", left: left, right: right}
	}
	return left, nil
}

func (p *exprParser) parseAnd() (node, error) {
	left, err := p.parseComparison()
	if err != nil {
		return nil, err
	}
	for p.current.kind == tokAnd {
		p.eat(tokAnd)
		right, err := p.parseComparison()
		if err != nil {
			return nil, err
		}
		left = &binaryNode{op: "&&", left: left, right: right}
	}
	return left, nil
}

func (p *exprParser) parseComparison() (node, error) {
	left, err := p.parseAddSub()
	if err != nil {
		return nil, err
	}
	for {
		var op string
		switch p.current.kind {
		case tokEQ:
			op = "=="
		case tokNE:
			op = "!="
		case tokLT:
			op = "<"
		case tokLE:
			op = "<="
		case tokGT:
			op = ">"
		case tokGE:
			op = ">="
		default:
			return left, nil
		}
		p.eat(p.current.kind)
		right, err := p.parseAddSub()
		if err != nil {
			return nil, err
		}
		left = &binaryNode{op: op, left: left, right: right}
	}
}

func (p *exprParser) parseAddSub() (node, error) {
	left, err := p.parseMulDiv()
	if err != nil {
		return nil, err
	}
	for p.current.kind == tokPlus || p.current.kind == tokMinus {
		op := "+"
		if p.current.kind == tokMinus {
			op = "-"
		}
		p.eat(p.current.kind)
		right, err := p.parseMulDiv()
		if err != nil {
			return nil, err
		}
		left = &binaryNode{op: op, left: left, right: right}
	}
	return left, nil
}

func (p *exprParser) parseMulDiv() (node, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	for p.current.kind == tokStar || p.current.kind == tokSlash || p.current.kind == tokPercent {
		op := "*"
		if p.current.kind == tokSlash {
			op = "/"
		} else if p.current.kind == tokPercent {
			op = "%"
		}
		p.eat(p.current.kind)
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		left = &binaryNode{op: op, left: left, right: right}
	}
	return left, nil
}

func (p *exprParser) parseUnary() (node, error) {
	if p.current.kind == tokMinus {
		p.eat(tokMinus)
		operand, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &unaryNode{op: "-", operand: operand}, nil
	}
	if p.current.kind == tokNot {
		p.eat(tokNot)
		operand, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &unaryNode{op: "!", operand: operand}, nil
	}
	return p.parsePrimary()
}

func (p *exprParser) parsePrimary() (node, error) {
	switch p.current.kind {
	case tokNumber:
		t := p.eat(tokNumber)
		return &numberNode{val: t.nval}, nil

	case tokString:
		t := p.eat(tokString)
		return &stringNode{val: t.sval}, nil

	case tokIdent:
		t := p.eat(tokIdent)
		name := t.sval

		if strings.ToLower(name) == "true" {
			return &boolNode{val: true}, nil
		}
		if strings.ToLower(name) == "false" {
			return &boolNode{val: false}, nil
		}

		if p.current.kind == tokLParen {
			return p.parseFunctionCall(name)
		}

		for p.current.kind == tokDot {
			p.eat(tokDot)
			if p.current.kind != tokIdent {
				return nil, fmt.Errorf("expected identifier after '.'")
			}
			next := p.eat(tokIdent)
			name = name + "." + next.sval
		}

		return &identNode{name: name}, nil

	case tokLParen:
		p.eat(tokLParen)
		expr, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		p.eat(tokRParen)
		return expr, nil
	}

	return nil, fmt.Errorf("unexpected token: %v", p.current)
}

func (p *exprParser) parseFunctionCall(name string) (node, error) {
	p.eat(tokLParen)
	var args []node
	if p.current.kind != tokRParen {
		arg, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		args = append(args, arg)
		for p.current.kind == tokComma {
			p.eat(tokComma)
			arg, err = p.parseExpr()
			if err != nil {
				return nil, err
			}
			args = append(args, arg)
		}
	}
	p.eat(tokRParen)
	return &funcNode{name: strings.ToLower(name), args: args}, nil
}

type node interface {
	eval(ctx *EvalContext) (any, error)
}

type numberNode struct{ val float64 }
type stringNode struct{ val string }
type boolNode struct{ val bool }
type identNode struct{ name string }
type binaryNode struct {
	op    string
	left  node
	right node
}
type unaryNode struct {
	op      string
	operand node
}
type funcNode struct {
	name string
	args []node
}

func (n *numberNode) eval(_ *EvalContext) (any, error) { return n.val, nil }
func (n *stringNode) eval(_ *EvalContext) (any, error) { return n.val, nil }
func (n *boolNode) eval(_ *EvalContext) (any, error)   { return n.val, nil }

func (n *identNode) eval(ctx *EvalContext) (any, error) {
	return ctx.Resolve(n.name)
}

func (n *unaryNode) eval(ctx *EvalContext) (any, error) {
	val, err := n.operand.eval(ctx)
	if err != nil {
		return nil, err
	}
	switch n.op {
	case "-":
		return -toFloat(val), nil
	case "!":
		return !toBool(val), nil
	}
	return nil, fmt.Errorf("unknown unary operator: %s", n.op)
}

func (n *binaryNode) eval(ctx *EvalContext) (any, error) {
	leftVal, err := n.left.eval(ctx)
	if err != nil {
		return nil, err
	}
	rightVal, err := n.right.eval(ctx)
	if err != nil {
		return nil, err
	}

	switch n.op {
	case "+":
		ls, lok := leftVal.(string)
		rs, rok := rightVal.(string)
		if lok && rok {
			return ls + rs, nil
		}
		return toFloat(leftVal) + toFloat(rightVal), nil
	case "-":
		return toFloat(leftVal) - toFloat(rightVal), nil
	case "*":
		return toFloat(leftVal) * toFloat(rightVal), nil
	case "/":
		d := toFloat(rightVal)
		if d == 0 {
			return float64(0), nil
		}
		return toFloat(leftVal) / d, nil
	case "%":
		d := toFloat(rightVal)
		if d == 0 {
			return float64(0), nil
		}
		return math.Mod(toFloat(leftVal), d), nil
	case "==":
		return fmt.Sprintf("%v", leftVal) == fmt.Sprintf("%v", rightVal), nil
	case "!=":
		return fmt.Sprintf("%v", leftVal) != fmt.Sprintf("%v", rightVal), nil
	case "<":
		return toFloat(leftVal) < toFloat(rightVal), nil
	case "<=":
		return toFloat(leftVal) <= toFloat(rightVal), nil
	case ">":
		return toFloat(leftVal) > toFloat(rightVal), nil
	case ">=":
		return toFloat(leftVal) >= toFloat(rightVal), nil
	case "&&":
		return toBool(leftVal) && toBool(rightVal), nil
	case "||":
		return toBool(leftVal) || toBool(rightVal), nil
	}

	return nil, fmt.Errorf("unknown operator: %s", n.op)
}

func (n *funcNode) eval(ctx *EvalContext) (any, error) {
	if len(n.args) == 1 {
		if ident, ok := n.args[0].(*identNode); ok {
			if strings.Contains(ident.name, ".") {
				return ctx.ResolveAggregate(n.name, ident.name)
			}
		}
	}

	var vals []float64
	for _, arg := range n.args {
		v, err := arg.eval(ctx)
		if err != nil {
			return nil, err
		}
		vals = append(vals, toFloat(v))
	}

	switch n.name {
	case "sum":
		s := 0.0
		for _, v := range vals {
			s += v
		}
		return s, nil
	case "count":
		return float64(len(vals)), nil
	case "avg":
		if len(vals) == 0 {
			return 0.0, nil
		}
		s := 0.0
		for _, v := range vals {
			s += v
		}
		return s / float64(len(vals)), nil
	case "min":
		if len(vals) == 0 {
			return 0.0, nil
		}
		m := vals[0]
		for _, v := range vals[1:] {
			if v < m {
				m = v
			}
		}
		return m, nil
	case "max":
		if len(vals) == 0 {
			return 0.0, nil
		}
		m := vals[0]
		for _, v := range vals[1:] {
			if v > m {
				m = v
			}
		}
		return m, nil
	case "abs":
		if len(vals) > 0 {
			return math.Abs(vals[0]), nil
		}
		return 0.0, nil
	case "round":
		if len(vals) > 0 {
			precision := 0.0
			if len(vals) > 1 {
				precision = vals[1]
			}
			pow := math.Pow(10, precision)
			return math.Round(vals[0]*pow) / pow, nil
		}
		return 0.0, nil
	case "ceil":
		if len(vals) > 0 {
			return math.Ceil(vals[0]), nil
		}
		return 0.0, nil
	case "floor":
		if len(vals) > 0 {
			return math.Floor(vals[0]), nil
		}
		return 0.0, nil
	case "if":
		if len(n.args) >= 3 {
			condVal, err := n.args[0].eval(ctx)
			if err != nil {
				return nil, err
			}
			if toBool(condVal) {
				return n.args[1].eval(ctx)
			}
			return n.args[2].eval(ctx)
		}
		return 0.0, nil
	}

	return nil, fmt.Errorf("unknown function: %s", n.name)
}

func toFloat(v any) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case int32:
		return float64(val)
	case string:
		f, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return 0
		}
		return f
	case bool:
		if val {
			return 1
		}
		return 0
	case nil:
		return 0
	default:
		s := fmt.Sprintf("%v", val)
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return 0
		}
		return f
	}
}

func toBool(v any) bool {
	switch val := v.(type) {
	case bool:
		return val
	case float64:
		return val != 0
	case int:
		return val != 0
	case string:
		return val != "" && val != "0" && val != "false"
	case nil:
		return false
	default:
		return true
	}
}

type EvalContext struct {
	Record           map[string]any
	ChildCollections map[string][]map[string]any
}

func (ctx *EvalContext) Resolve(path string) (any, error) {
	parts := strings.Split(path, ".")
	if len(parts) == 1 {
		val, ok := ctx.Record[path]
		if !ok {
			return nil, nil
		}
		return val, nil
	}

	current := ctx.Record
	for i, part := range parts {
		if i == len(parts)-1 {
			val, ok := current[part]
			if !ok {
				return nil, nil
			}
			return val, nil
		}
		next, ok := current[part]
		if !ok {
			return nil, nil
		}
		if m, ok := next.(map[string]any); ok {
			current = m
		} else {
			return nil, nil
		}
	}
	return nil, nil
}

func (ctx *EvalContext) ResolveAggregate(funcName string, path string) (any, error) {
	parts := strings.SplitN(path, ".", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("aggregate path must be collection.field, got: %s", path)
	}

	collectionName := parts[0]
	fieldName := parts[1]

	records, ok := ctx.ChildCollections[collectionName]
	if !ok {
		return 0.0, nil
	}

	var vals []float64
	for _, rec := range records {
		if v, ok := rec[fieldName]; ok {
			vals = append(vals, toFloat(v))
		}
	}

	switch funcName {
	case "sum":
		s := 0.0
		for _, v := range vals {
			s += v
		}
		return s, nil
	case "count":
		return float64(len(vals)), nil
	case "avg":
		if len(vals) == 0 {
			return 0.0, nil
		}
		s := 0.0
		for _, v := range vals {
			s += v
		}
		return s / float64(len(vals)), nil
	case "min":
		if len(vals) == 0 {
			return 0.0, nil
		}
		m := vals[0]
		for _, v := range vals[1:] {
			if v < m {
				m = v
			}
		}
		return m, nil
	case "max":
		if len(vals) == 0 {
			return 0.0, nil
		}
		m := vals[0]
		for _, v := range vals[1:] {
			if v > m {
				m = v
			}
		}
		return m, nil
	}

	return nil, fmt.Errorf("unknown aggregate function: %s", funcName)
}

func Evaluate(expr string, ctx *EvalContext) (any, error) {
	if expr == "" {
		return nil, nil
	}
	p := newExprParser(expr)
	ast, err := p.parseExpr()
	if err != nil {
		return nil, fmt.Errorf("parse error in expression %q: %w", expr, err)
	}
	return ast.eval(ctx)
}

func EvaluateFloat(expr string, ctx *EvalContext) (float64, error) {
	val, err := Evaluate(expr, ctx)
	if err != nil {
		return 0, err
	}
	return toFloat(val), nil
}
