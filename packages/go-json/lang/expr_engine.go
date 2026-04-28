package lang

import (
	"fmt"
	"strings"
	"sync"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
)

// CompiledExpr wraps a compiled expr-lang program.
type CompiledExpr = *vm.Program

// ExprEngine abstracts expression compilation and evaluation.
// The VM never calls expr-lang directly — all expression work goes through this interface.
type ExprEngine interface {
	Compile(expression string, env map[string]any) (CompiledExpr, error)
	Run(compiled CompiledExpr, env map[string]any) (any, error)
	Eval(expression string, env map[string]any) (any, error)
	Validate(expression string, env map[string]any) error
	AddOptions(opts ...expr.Option)
}

// ExprLangEngine implements ExprEngine using expr-lang/expr.
type ExprLangEngine struct {
	cache   map[string]CompiledExpr
	mu      sync.RWMutex
	options []expr.Option
}

func NewExprLangEngine(opts ...expr.Option) *ExprLangEngine {
	return &ExprLangEngine{
		cache:   make(map[string]CompiledExpr),
		options: opts,
	}
}

func (e *ExprLangEngine) AddOptions(opts ...expr.Option) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.options = append(e.options, opts...)
}

func (e *ExprLangEngine) Compile(expression string, env map[string]any) (compiled CompiledExpr, err error) {
	// Check cache first.
	e.mu.RLock()
	if cached, ok := e.cache[expression]; ok {
		e.mu.RUnlock()
		return cached, nil
	}
	e.mu.RUnlock()

	// Protect against expr-lang panics.
	defer func() {
		if r := recover(); r != nil {
			err = RuntimeError("EXPR_PANIC",
				fmt.Sprintf("expression engine panic: %v", r), -1).
				WithContext(map[string]any{"expression": expression})
		}
	}()

	opts := make([]expr.Option, 0, len(e.options)+1)
	if env != nil {
		opts = append(opts, expr.Env(env))
	}
	e.mu.RLock()
	opts = append(opts, e.options...)
	e.mu.RUnlock()

	program, compileErr := expr.Compile(expression, opts...)
	if compileErr != nil {
		return nil, e.enrichError(compileErr, expression, env)
	}

	// Cache the compiled program.
	e.mu.Lock()
	e.cache[expression] = program
	e.mu.Unlock()

	return program, nil
}

func (e *ExprLangEngine) Run(compiled CompiledExpr, env map[string]any) (result any, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = RuntimeError("EXPR_PANIC",
				fmt.Sprintf("expression engine panic: %v", r), -1)
		}
	}()

	output, runErr := expr.Run(compiled, env)
	if runErr != nil {
		return nil, e.enrichRunError(runErr, env)
	}
	return output, nil
}

func (e *ExprLangEngine) Eval(expression string, env map[string]any) (any, error) {
	compiled, err := e.Compile(expression, env)
	if err != nil {
		return nil, err
	}
	return e.Run(compiled, env)
}

func (e *ExprLangEngine) Validate(expression string, env map[string]any) error {
	// Compile-only — we don't cache validation-only compilations
	// because the env may differ from runtime env.
	defer func() {
		if r := recover(); r != nil {
			// Swallow panic during validation.
		}
	}()

	opts := make([]expr.Option, 0, len(e.options)+1)
	if env != nil {
		opts = append(opts, expr.Env(env))
	}
	e.mu.RLock()
	opts = append(opts, e.options...)
	e.mu.RUnlock()

	_, err := expr.Compile(expression, opts...)
	if err != nil {
		return e.enrichError(err, expression, env)
	}
	return nil
}

// enrichError detects common expr-lang error patterns and adds suggestions.
func (e *ExprLangEngine) enrichError(err error, expression string, env map[string]any) *GoJSONError {
	msg := err.Error()

	// Detect "undefined variable" pattern.
	if strings.Contains(msg, "undeclared variable") || strings.Contains(msg, "undefined") {
		varName := extractVarName(msg)
		gjErr := CompileError("UNDEFINED_VAR",
			fmt.Sprintf("undefined variable '%s' in expression: %s", varName, expression), -1).
			WithContext(map[string]any{"expression": expression})

		if varName != "" && env != nil {
			candidates := mapKeys(env)
			if suggestions := SuggestSimilar(varName, candidates, 3, 3); len(suggestions) > 0 {
				gjErr.WithSuggestions(suggestions...)
				gjErr.WithFix("did you mean: " + strings.Join(suggestions, ", ") + "?")
			}
		}
		return gjErr
	}

	// Detect type mismatch.
	if strings.Contains(msg, "type") && (strings.Contains(msg, "mismatch") || strings.Contains(msg, "invalid operation")) {
		return CompileError("TYPE_MISMATCH", msg, -1).
			WithContext(map[string]any{"expression": expression}).
			WithFix("check operand types — you may need a conversion function like int(), float(), or string()")
	}

	// Detect division by zero.
	if strings.Contains(msg, "division by zero") {
		return RuntimeError("DIVISION_BY_ZERO", msg, -1).
			WithContext(map[string]any{"expression": expression}).
			WithFix("add a guard: if divisor != 0 { ... }")
	}

	// Fallback: wrap raw error.
	return CompileError("EXPR_ERROR", msg, -1).
		WithContext(map[string]any{"expression": expression})
}

// enrichRunError handles runtime errors from expr.Run.
func (e *ExprLangEngine) enrichRunError(err error, env map[string]any) *GoJSONError {
	msg := err.Error()

	if strings.Contains(msg, "nil pointer") || strings.Contains(msg, "interface conversion") {
		return RuntimeError("NIL_ACCESS", msg, -1).
			WithFix("use optional chaining (?.) or nil coalescing (??) to handle nil values")
	}

	if strings.Contains(msg, "division by zero") {
		return RuntimeError("DIVISION_BY_ZERO", msg, -1).
			WithFix("add a guard: if divisor != 0 { ... }")
	}

	return RuntimeError("EXPR_RUNTIME", msg, -1)
}

// extractVarName attempts to extract a variable name from an error message.
// Handles patterns like: "undeclared variable 'foo'" or "undefined: foo"
func extractVarName(msg string) string {
	// Pattern: "undeclared variable \"foo\""
	if idx := strings.Index(msg, "\""); idx >= 0 {
		end := strings.Index(msg[idx+1:], "\"")
		if end >= 0 {
			return msg[idx+1 : idx+1+end]
		}
	}
	// Pattern: "undefined: foo"
	if idx := strings.Index(msg, ": "); idx >= 0 {
		parts := strings.Fields(msg[idx+2:])
		if len(parts) > 0 {
			return strings.Trim(parts[0], "'\"`;,")
		}
	}
	return ""
}

func mapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
