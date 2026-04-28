package lang

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Sentinel types for control flow signals.
type returnValue struct{ value any }
type breakSignal struct{}
type continueSignal struct{}

type Logger interface {
	Log(level, message string, data map[string]any)
}

type VMOption func(*VM)

func WithContext(ctx context.Context) VMOption {
	return func(vm *VM) { vm.ctx = ctx }
}

func WithDebugger(d Debugger) VMOption {
	return func(vm *VM) { vm.debugger = d }
}

func WithLogger(l Logger) VMOption {
	return func(vm *VM) { vm.logger = l }
}

func WithTrace(enabled bool) VMOption {
	return func(vm *VM) {
		if enabled {
			vm.trace = NewExecutionTrace()
		}
	}
}

type VM struct {
	program   *CompiledProgram
	engine    ExprEngine
	scope     *Scope
	ctx       context.Context
	cancel    context.CancelFunc
	debugger  Debugger
	logger    Logger
	trace     *ExecutionTrace
	stepCount int
	depth     int
	callStack []StackFrame
	limits    ResolvedLimits
}

type ExecutionResult struct {
	Value    any
	Trace    []TraceEntry
	Steps    int
	Duration time.Duration
}

func NewVM(program *CompiledProgram, engine ExprEngine, opts ...VMOption) *VM {
	vm := &VM{
		program: program,
		engine:  engine,
		limits:  program.Limits,
	}

	for _, opt := range opts {
		opt(vm)
	}

	if vm.ctx == nil {
		vm.ctx = context.Background()
	}

	// Apply timeout.
	if vm.limits.Timeout > 0 {
		vm.ctx, vm.cancel = context.WithTimeout(vm.ctx, vm.limits.Timeout)
	}

	return vm
}

func (vm *VM) Execute(input map[string]any) (*ExecutionResult, error) {
	start := time.Now()

	if vm.cancel != nil {
		defer vm.cancel()
	}

	vm.scope = NewScope("main")

	// Inject input.
	if input != nil {
		vm.scope.Declare("input", input, "map")
	} else {
		vm.scope.Declare("input", map[string]any{}, "map")
	}

	// Register functions in scope for expression-level calls.
	for name, fn := range vm.program.Functions {
		wrapped := vm.wrapFunction(fn)
		vm.scope.Declare(name, wrapped, "func")
	}

	result, err := vm.executeSteps(vm.program.AST.Steps)
	if err != nil {
		return nil, err
	}

	// Unwrap return sentinel.
	var value any
	if rv, ok := result.(returnValue); ok {
		value = rv.value
	} else {
		value = result
	}

	er := &ExecutionResult{
		Value:    value,
		Steps:    vm.stepCount,
		Duration: time.Since(start),
	}
	if vm.trace != nil {
		er.Trace = vm.trace.Entries()
	}

	return er, nil
}

func (vm *VM) executeSteps(steps []Node) (any, error) {
	for _, step := range steps {
		// Context check (timeout/cancellation).
		select {
		case <-vm.ctx.Done():
			return nil, LimitError("TIMEOUT",
				fmt.Sprintf("execution timeout (%s) exceeded at step %d", vm.limits.Timeout, step.Meta().StepIndex),
				step.Meta().StepIndex)
		default:
		}

		// Step limit check.
		vm.stepCount++
		if vm.stepCount > vm.limits.MaxSteps {
			return nil, LimitError("STEP_LIMIT",
				fmt.Sprintf("step limit (%d) exceeded", vm.limits.MaxSteps),
				step.Meta().StepIndex)
		}

		// Debug hook.
		if vm.debugger != nil {
			action := vm.debugger.OnStep(StepInfo{
				Index: step.Meta().StepIndex,
				Type:  step.nodeType(),
				Node:  step,
			})
			if action == DebugPause {
				// In a real implementation, this would block.
				// For now, just continue.
			}
		}

		stepStart := time.Now()

		result, err := vm.executeStep(step)
		if err != nil {
			if vm.debugger != nil {
				vm.debugger.OnError(err, step.Meta().StepIndex)
			}
			return nil, err
		}

		// Trace capture.
		if vm.trace != nil {
			vm.trace.AddStep(TraceEntry{
				Step:       step.Meta().StepIndex,
				Type:       step.nodeType(),
				DurationUs: time.Since(stepStart).Microseconds(),
			})
		}

		// Handle control flow signals.
		if result != nil {
			switch result.(type) {
			case returnValue, breakSignal, continueSignal:
				return result, nil
			}
		}
	}
	return nil, nil
}

func (vm *VM) executeStep(node Node) (any, error) {
	switch n := node.(type) {
	case *LetNode:
		return nil, vm.executeLet(n)
	case *SetNode:
		return nil, vm.executeSet(n)
	case *IfNode:
		return vm.executeIf(n)
	case *SwitchNode:
		return vm.executeSwitch(n)
	case *ForNode:
		return vm.executeFor(n)
	case *WhileNode:
		return vm.executeWhile(n)
	case *ReturnNode:
		return vm.executeReturn(n)
	case *CallNode:
		return nil, vm.executeCall(n)
	case *TryNode:
		return vm.executeTry(n)
	case *ErrorNode:
		return nil, vm.executeError(n)
	case *LogNode:
		return nil, vm.executeLog(n)
	case *BreakNode:
		return breakSignal{}, nil
	case *ContinueNode:
		return continueSignal{}, nil
	case *CommentNode:
		return nil, nil
	default:
		return nil, RuntimeError("UNKNOWN_STEP", fmt.Sprintf("unknown step type: %T", node), node.Meta().StepIndex)
	}
}

func (vm *VM) executeLet(n *LetNode) error {
	idx := n.StepIndex

	// Call shorthand: {"let": "x", "call": "fn", "with": {...}}
	if n.Call != "" {
		result, err := vm.callFunction(n.Call, n.CallWith, idx)
		if err != nil {
			return err
		}
		typ := InferType(result)
		if n.Type != "" {
			typ = n.Type
		}
		return vm.scope.Declare(n.Name, result, typ)
	}

	var value any
	var err error

	if n.HasValue {
		value = n.Value
	} else if n.HasExpr {
		value, err = vm.evalExpr(n.Expr, idx)
		if err != nil {
			return err
		}
	} else if n.HasWith {
		value, err = vm.evalWith(n.With, idx)
		if err != nil {
			return err
		}
	}

	typ := InferType(value)
	if n.Type != "" {
		typ = n.Type
	}

	if err := vm.scope.Declare(n.Name, value, typ); err != nil {
		if gjErr, ok := err.(*GoJSONError); ok {
			gjErr.Step = idx
		}
		return err
	}

	if vm.debugger != nil {
		vm.debugger.OnVariable(n.Name, value, vm.scope.Name())
	}

	return nil
}

func (vm *VM) executeSet(n *SetNode) error {
	idx := n.StepIndex

	var value any
	var err error

	if n.HasValue {
		value = n.Value
	} else if n.HasExpr {
		value, err = vm.evalExpr(n.Expr, idx)
		if err != nil {
			return err
		}
	} else if n.HasWith {
		value, err = vm.evalWith(n.With, idx)
		if err != nil {
			return err
		}
	}

	// Handle dot-path mutation: "a.b.c" or "items[0].name"
	if strings.Contains(n.Target, ".") || strings.Contains(n.Target, "[") {
		return vm.setNestedProperty(n.Target, value, idx)
	}

	newType := InferType(value)
	if err := vm.scope.Set(n.Target, value, newType); err != nil {
		if gjErr, ok := err.(*GoJSONError); ok {
			gjErr.Step = idx
		}
		return err
	}

	if vm.debugger != nil {
		vm.debugger.OnVariable(n.Target, value, vm.scope.Name())
	}

	return nil
}

func (vm *VM) executeIf(n *IfNode) (any, error) {
	condResult, err := vm.evalExpr(n.Condition, n.StepIndex)
	if err != nil {
		return nil, err
	}

	if toBool(condResult) {
		childScope := vm.scope.NewChild("if-then")
		return vm.withScope(childScope, func() (any, error) {
			return vm.executeSteps(n.Then)
		})
	}

	for _, elif := range n.Elif {
		condResult, err := vm.evalExpr(elif.Condition, n.StepIndex)
		if err != nil {
			return nil, err
		}
		if toBool(condResult) {
			childScope := vm.scope.NewChild("elif")
			return vm.withScope(childScope, func() (any, error) {
				return vm.executeSteps(elif.Then)
			})
		}
	}

	if len(n.Else) > 0 {
		childScope := vm.scope.NewChild("else")
		return vm.withScope(childScope, func() (any, error) {
			return vm.executeSteps(n.Else)
		})
	}

	return nil, nil
}

func (vm *VM) executeSwitch(n *SwitchNode) (any, error) {
	exprResult, err := vm.evalExpr(n.Expr, n.StepIndex)
	if err != nil {
		return nil, err
	}

	key := fmt.Sprintf("%v", exprResult)

	if steps, ok := n.Cases[key]; ok {
		childScope := vm.scope.NewChild("case-" + key)
		return vm.withScope(childScope, func() (any, error) {
			return vm.executeSteps(steps)
		})
	}

	if steps, ok := n.Cases["default"]; ok {
		childScope := vm.scope.NewChild("case-default")
		return vm.withScope(childScope, func() (any, error) {
			return vm.executeSteps(steps)
		})
	}

	return nil, nil
}

func (vm *VM) executeFor(n *ForNode) (any, error) {
	idx := n.StepIndex

	var items []any

	if n.In != "" {
		result, err := vm.evalExpr(n.In, idx)
		if err != nil {
			return nil, err
		}
		arr, ok := result.([]any)
		if !ok {
			return nil, RuntimeError("NOT_ITERABLE",
				fmt.Sprintf("for-in expression must evaluate to array, got %T", result), idx)
		}
		items = arr
	} else if n.Range != nil {
		generated, err := generateRange(n.Range)
		if err != nil {
			return nil, RuntimeError("INVALID_RANGE", err.Error(), idx)
		}
		items = generated
	}

	for i, item := range items {
		if i >= vm.limits.MaxLoopIterations {
			return nil, LimitError("LOOP_LIMIT",
				fmt.Sprintf("loop iteration limit (%d) exceeded", vm.limits.MaxLoopIterations), idx)
		}

		childScope := vm.scope.NewChild("for-iteration")
		childScope.Declare(n.Variable, item, InferType(item))
		if n.Index != "" {
			childScope.Declare(n.Index, i, "int")
		}

		result, err := vm.withScope(childScope, func() (any, error) {
			return vm.executeSteps(n.Steps)
		})
		if err != nil {
			return nil, err
		}

		if result != nil {
			switch result.(type) {
			case breakSignal:
				return nil, nil
			case continueSignal:
				continue
			case returnValue:
				return result, nil
			}
		}
	}

	return nil, nil
}

func (vm *VM) executeWhile(n *WhileNode) (any, error) {
	idx := n.StepIndex
	iterations := 0

	for {
		if iterations >= vm.limits.MaxLoopIterations {
			return nil, LimitError("LOOP_LIMIT",
				fmt.Sprintf("loop iteration limit (%d) exceeded", vm.limits.MaxLoopIterations), idx)
		}

		condResult, err := vm.evalExpr(n.Condition, idx)
		if err != nil {
			return nil, err
		}

		if !toBool(condResult) {
			break
		}

		childScope := vm.scope.NewChild("while-iteration")
		result, err := vm.withScope(childScope, func() (any, error) {
			return vm.executeSteps(n.Steps)
		})
		if err != nil {
			return nil, err
		}

		if result != nil {
			switch result.(type) {
			case breakSignal:
				return nil, nil
			case continueSignal:
				iterations++
				continue
			case returnValue:
				return result, nil
			}
		}

		iterations++
	}

	return nil, nil
}

func (vm *VM) executeReturn(n *ReturnNode) (any, error) {
	idx := n.StepIndex

	if n.HasValue {
		return returnValue{value: n.Value}, nil
	}

	if n.HasExpr {
		result, err := vm.evalExpr(n.Expr, idx)
		if err != nil {
			return nil, err
		}
		return returnValue{value: result}, nil
	}

	if n.HasWith {
		result, err := vm.evalWith(n.With, idx)
		if err != nil {
			return nil, err
		}
		return returnValue{value: result}, nil
	}

	return returnValue{value: nil}, nil
}

func (vm *VM) executeCall(n *CallNode) error {
	_, err := vm.callFunction(n.Function, n.With, n.StepIndex)
	return err
}

func (vm *VM) executeTry(n *TryNode) (any, error) {
	var tryResult any
	var tryErr error

	func() {
		defer func() {
			if r := recover(); r != nil {
				tryErr = RuntimeError("PANIC", fmt.Sprintf("%v", r), n.StepIndex)
			}
		}()

		childScope := vm.scope.NewChild("try")
		tryResult, tryErr = vm.withScope(childScope, func() (any, error) {
			return vm.executeSteps(n.Try)
		})
	}()

	if tryErr != nil && n.Catch != nil {
		errObj := vm.normalizeError(tryErr, n.StepIndex)

		catchScope := vm.scope.NewChild("catch")
		catchScope.Declare(n.Catch.As, errObj, "map")

		catchResult, catchErr := vm.withScope(catchScope, func() (any, error) {
			return vm.executeSteps(n.Catch.Steps)
		})

		if n.Finally != nil {
			finallyScope := vm.scope.NewChild("finally")
			_, finallyErr := vm.withScope(finallyScope, func() (any, error) {
				return vm.executeSteps(n.Finally)
			})
			if finallyErr != nil {
				return nil, finallyErr
			}
		}

		if catchErr != nil {
			return nil, catchErr
		}
		return catchResult, nil
	}

	if n.Finally != nil {
		finallyScope := vm.scope.NewChild("finally")
		_, finallyErr := vm.withScope(finallyScope, func() (any, error) {
			return vm.executeSteps(n.Finally)
		})
		if finallyErr != nil {
			return nil, finallyErr
		}
	}

	if tryErr != nil {
		return nil, tryErr
	}

	return tryResult, nil
}

func (vm *VM) executeError(n *ErrorNode) error {
	idx := n.StepIndex

	if n.IsStructured {
		code := "ERROR"
		message := ""
		var details any

		if n.Code != "" {
			result, err := vm.evalExpr(n.Code, idx)
			if err != nil {
				return err
			}
			code = fmt.Sprintf("%v", result)
		}
		if n.Message != "" {
			result, err := vm.evalExpr(n.Message, idx)
			if err != nil {
				return err
			}
			message = fmt.Sprintf("%v", result)
		}
		if n.Details != "" {
			result, err := vm.evalExpr(n.Details, idx)
			if err != nil {
				return err
			}
			details = result
		}

		return RuntimeError(code, message, idx).
			WithContext(map[string]any{"details": details}).
			WithStack(vm.callStack)
	}

	// Simple string error.
	result, err := vm.evalExpr(n.Message, idx)
	if err != nil {
		return err
	}
	return RuntimeError("ERROR", fmt.Sprintf("%v", result), idx).
		WithStack(vm.callStack)
}

func (vm *VM) executeLog(n *LogNode) error {
	idx := n.StepIndex

	if vm.logger == nil {
		return nil
	}

	if n.IsStructured {
		message := ""
		level := "info"
		var data map[string]any

		if n.Message != "" {
			result, err := vm.evalExpr(n.Message, idx)
			if err != nil {
				return err
			}
			message = fmt.Sprintf("%v", result)
		}
		if n.Level != "" {
			result, err := vm.evalExpr(n.Level, idx)
			if err != nil {
				return err
			}
			level = fmt.Sprintf("%v", result)
		}
		if n.Data != nil {
			data = make(map[string]any)
			for k, expr := range n.Data {
				result, err := vm.evalExpr(expr, idx)
				if err != nil {
					return err
				}
				data[k] = result
			}
		}

		vm.logger.Log(level, message, data)
		return nil
	}

	// Simple string log.
	result, err := vm.evalExpr(n.Message, idx)
	if err != nil {
		return err
	}
	vm.logger.Log("info", fmt.Sprintf("%v", result), nil)
	return nil
}

// --- Function calling ---

func (vm *VM) callFunction(name string, withExprs map[string]string, stepIndex int) (any, error) {
	// Depth limit check.
	vm.depth++
	defer func() { vm.depth-- }()

	if vm.depth > vm.limits.MaxDepth {
		return nil, LimitError("DEPTH_LIMIT",
			fmt.Sprintf("call depth limit (%d) exceeded at function '%s'", vm.limits.MaxDepth, name),
			stepIndex).WithStack(vm.callStack)
	}

	fn, ok := vm.program.Functions[name]
	if !ok {
		// "Did you mean?" suggestion.
		funcNames := make([]string, 0, len(vm.program.Functions))
		for n := range vm.program.Functions {
			funcNames = append(funcNames, n)
		}
		gjErr := RuntimeError("FUNC_NOT_FOUND",
			fmt.Sprintf("function '%s' not defined", name), stepIndex)
		if suggestions := SuggestSimilar(name, funcNames, 3, 3); len(suggestions) > 0 {
			gjErr.WithSuggestions(suggestions...)
		}
		return nil, gjErr
	}

	// Push call stack.
	vm.callStack = append(vm.callStack, StackFrame{Function: name, Step: stepIndex})
	defer func() { vm.callStack = vm.callStack[:len(vm.callStack)-1] }()

	if vm.debugger != nil {
		args := make(map[string]any)
		if withExprs != nil {
			for k, v := range withExprs {
				args[k] = v
			}
		}
		vm.debugger.OnFunctionCall(name, args)
	}

	// Create isolated scope.
	funcScope := vm.scope.IsolatedChild("func:" + name)

	// Evaluate arg expressions in CALLER scope, bind in FUNCTION scope.
	for _, param := range fn.Params {
		if withExprs != nil {
			if expr, ok := withExprs[param.Name]; ok {
				val, err := vm.evalExpr(expr, stepIndex)
				if err != nil {
					return nil, err
				}
				funcScope.Declare(param.Name, val, param.Type)
				continue
			}
		}
		if param.HasDefault {
			funcScope.Declare(param.Name, param.Default, param.Type)
		} else {
			funcScope.Declare(param.Name, nil, param.Type)
		}
	}

	// Register functions for recursion.
	for fname, ffn := range vm.program.Functions {
		wrapped := vm.wrapFunction(ffn)
		funcScope.Declare(fname, wrapped, "func")
	}

	// Execute function steps.
	result, err := vm.withScope(funcScope, func() (any, error) {
		return vm.executeSteps(fn.Steps)
	})
	if err != nil {
		return nil, err
	}

	// Unwrap return sentinel.
	var retVal any
	if rv, ok := result.(returnValue); ok {
		retVal = rv.value
	}

	if vm.debugger != nil {
		vm.debugger.OnFunctionReturn(name, retVal)
	}

	return retVal, nil
}

// wrapFunction wraps a CompiledFunc as a Go func for expr-lang expression-level calls.
// Positional params: createUser('Alice', 30) → name='Alice', age=30
func (vm *VM) wrapFunction(fn *CompiledFunc) func(...any) (any, error) {
	return func(args ...any) (any, error) {
		funcScope := vm.scope.IsolatedChild("func:" + fn.Name)

		for i, param := range fn.Params {
			if i < len(args) {
				funcScope.Declare(param.Name, args[i], param.Type)
			} else if param.HasDefault {
				funcScope.Declare(param.Name, param.Default, param.Type)
			} else {
				funcScope.Declare(param.Name, nil, param.Type)
			}
		}

		// Register functions for recursion.
		for fname, ffn := range vm.program.Functions {
			wrapped := vm.wrapFunction(ffn)
			funcScope.Declare(fname, wrapped, "func")
		}

		vm.depth++
		defer func() { vm.depth-- }()

		if vm.depth > vm.limits.MaxDepth {
			return nil, LimitError("DEPTH_LIMIT",
				fmt.Sprintf("call depth limit (%d) exceeded at function '%s'", vm.limits.MaxDepth, fn.Name), -1)
		}

		vm.callStack = append(vm.callStack, StackFrame{Function: fn.Name, Step: -1})
		defer func() { vm.callStack = vm.callStack[:len(vm.callStack)-1] }()

		result, err := vm.withScope(funcScope, func() (any, error) {
			return vm.executeSteps(fn.Steps)
		})
		if err != nil {
			return nil, err
		}

		if rv, ok := result.(returnValue); ok {
			return rv.value, nil
		}
		return nil, nil
	}
}

// --- Expression evaluation ---

func (vm *VM) evalExpr(expression string, stepIndex int) (any, error) {
	env := vm.scope.ToMap()
	result, err := vm.engine.Eval(expression, env)
	if err != nil {
		if gjErr, ok := err.(*GoJSONError); ok {
			gjErr.Step = stepIndex
			if len(vm.callStack) > 0 {
				gjErr.Function = vm.callStack[len(vm.callStack)-1].Function
			}
			return nil, gjErr
		}
		return nil, RuntimeError("EXPR_ERROR", err.Error(), stepIndex)
	}
	return result, nil
}

func (vm *VM) evalWith(with map[string]string, stepIndex int) (map[string]any, error) {
	result := make(map[string]any, len(with))
	for key, expr := range with {
		val, err := vm.evalExpr(expr, stepIndex)
		if err != nil {
			return nil, err
		}
		result[key] = val
	}
	return result, nil
}

// --- Scope management ---

func (vm *VM) withScope(scope *Scope, fn func() (any, error)) (any, error) {
	prev := vm.scope
	vm.scope = scope
	result, err := fn()
	vm.scope = prev
	return result, err
}

// --- Nested property mutation ---

// setNestedProperty handles dot-path and bracket notation: "a.b.c", "items[0].name"
func (vm *VM) setNestedProperty(path string, value any, stepIndex int) error {
	parts := parseDotPath(path)
	if len(parts) == 0 {
		return RuntimeError("INVALID_PATH", "empty property path", stepIndex)
	}

	// Get the root variable.
	rootName := parts[0]
	rootVal, _, found := vm.scope.Get(rootName)
	if !found {
		return RuntimeError("VAR_NOT_FOUND",
			fmt.Sprintf("variable '%s' not defined", rootName), stepIndex)
	}

	if len(parts) == 1 {
		return vm.scope.Set(rootName, value, InferType(value))
	}

	// Traverse to the parent of the leaf.
	current := rootVal
	for i := 1; i < len(parts)-1; i++ {
		current = traverseProperty(current, parts[i])
		if current == nil {
			return RuntimeError("NIL_ACCESS",
				fmt.Sprintf("cannot access '%s' — intermediate value is nil at '%s'",
					path, strings.Join(parts[:i+1], ".")), stepIndex)
		}
	}

	// Set the leaf.
	leafKey := parts[len(parts)-1]

	if m, ok := current.(map[string]any); ok {
		m[leafKey] = value
		return nil
	}

	if arr, ok := current.([]any); ok {
		idx, err := strconv.Atoi(leafKey)
		if err != nil {
			return RuntimeError("INVALID_INDEX",
				fmt.Sprintf("array index '%s' is not a number", leafKey), stepIndex)
		}
		if idx < 0 || idx >= len(arr) {
			return RuntimeError("INDEX_OUT_OF_BOUNDS",
				fmt.Sprintf("array index %d out of bounds (length %d)", idx, len(arr)), stepIndex)
		}
		arr[idx] = value
		return nil
	}

	return RuntimeError("NOT_SETTABLE",
		fmt.Sprintf("cannot set property '%s' on %T", leafKey, current), stepIndex)
}

// parseDotPath splits "a.b[0].c" into ["a", "b", "0", "c"]
func parseDotPath(path string) []string {
	var parts []string
	current := ""

	for _, ch := range path {
		switch ch {
		case '.':
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		case '[':
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		case ']':
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		default:
			current += string(ch)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}

	return parts
}

func traverseProperty(obj any, key string) any {
	if m, ok := obj.(map[string]any); ok {
		return m[key]
	}
	if arr, ok := obj.([]any); ok {
		idx, err := strconv.Atoi(key)
		if err != nil || idx < 0 || idx >= len(arr) {
			return nil
		}
		return arr[idx]
	}
	return nil
}

// --- Range generation ---

func generateRange(rangeSpec []any) ([]any, error) {
	if len(rangeSpec) < 2 || len(rangeSpec) > 3 {
		return nil, fmt.Errorf("range requires [start, end] or [start, end, step]")
	}

	start, ok := toFloat(rangeSpec[0])
	if !ok {
		return nil, fmt.Errorf("range start must be a number")
	}
	end, ok := toFloat(rangeSpec[1])
	if !ok {
		return nil, fmt.Errorf("range end must be a number")
	}

	step := 1.0
	if len(rangeSpec) == 3 {
		s, ok := toFloat(rangeSpec[2])
		if !ok {
			return nil, fmt.Errorf("range step must be a number")
		}
		if s == 0 {
			return nil, fmt.Errorf("range step cannot be zero")
		}
		step = s
	}

	var items []any
	if step > 0 {
		for i := start; i < end; i += step {
			if float64(int64(i)) == i {
				items = append(items, int(i))
			} else {
				items = append(items, i)
			}
		}
	} else {
		for i := start; i > end; i += step {
			if float64(int64(i)) == i {
				items = append(items, int(i))
			} else {
				items = append(items, i)
			}
		}
	}

	return items, nil
}

func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	}
	return 0, false
}

// --- Error normalization ---

func (vm *VM) normalizeError(err error, stepIndex int) map[string]any {
	result := map[string]any{
		"message": err.Error(),
		"code":    "ERROR",
		"details": nil,
		"step":    stepIndex,
		"stack":   []string{},
	}

	if gjErr, ok := err.(*GoJSONError); ok {
		result["message"] = gjErr.Message
		result["code"] = gjErr.Code
		result["step"] = gjErr.Step
		if gjErr.Context != nil {
			if details, ok := gjErr.Context["details"]; ok {
				result["details"] = details
			}
		}
		stackStrs := make([]string, len(gjErr.Stack))
		for i, f := range gjErr.Stack {
			stackStrs[i] = fmt.Sprintf("%s() step %d", f.Function, f.Step)
		}
		result["stack"] = stackStrs
	}

	return result
}

// --- Helpers ---

func toBool(v any) bool {
	if v == nil {
		return false
	}
	switch b := v.(type) {
	case bool:
		return b
	case int:
		return b != 0
	case int64:
		return b != 0
	case float64:
		return b != 0
	case string:
		return b != ""
	case []any:
		return len(b) > 0
	case map[string]any:
		return len(b) > 0
	default:
		return true
	}
}
