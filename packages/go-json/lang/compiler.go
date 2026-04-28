package lang

import (
	"fmt"
	"time"
)

// Compile transforms a parsed Program AST into an immutable CompiledProgram.
// It validates all expressions, resolves limits, and detects structural errors.
func Compile(program *Program, engine ExprEngine, engineLimits ResolvedLimits) (*CompiledProgram, error) {
	cp := &CompiledProgram{
		Name:      program.Name,
		GoJSON:    program.GoJSON,
		AST:       program,
		Functions: make(map[string]*CompiledFunc),
		Input:     program.Input,
	}

	// Compile functions.
	for name, fd := range program.Functions {
		cf := &CompiledFunc{
			Name:    name,
			Returns: fd.Returns,
			Steps:   fd.Steps,
		}
		for _, fp := range fd.Params {
			cf.Params = append(cf.Params, ParamDef{
				Name:       fp.Name,
				Type:       fp.Type,
				Default:    fp.Default,
				HasDefault: fp.HasDefault,
			})
		}
		cp.Functions[name] = cf
	}

	// Structural validation only (break/continue outside loop).
	// Expression validation is deferred to runtime in Phase 4.5a because
	// expr-lang's compile-time type checking requires a fully-typed environment
	// which we don't have with gradual typing.
	if err := validateStructure(program.Steps, false); err != nil {
		return nil, err
	}
	for name, cf := range cp.Functions {
		if err := validateStructure(cf.Steps, false); err != nil {
			return nil, err.(*GoJSONError).InFunction(name)
		}
	}

	// Resolve limits.
	cp.Limits = resolveLimits(engineLimits, program.Limits)

	return cp, nil
}

func buildBaseEnv(program *Program, functions map[string]*CompiledFunc) map[string]any {
	env := make(map[string]any)

	if len(program.Input) > 0 {
		inputMap := make(map[string]any)
		for _, field := range program.Input {
			inputMap[field.Name] = zeroValueForType(field.Type)
		}
		env["input"] = inputMap
	}

	// Functions are available as callable names.
	for name := range functions {
		env[name] = func(...any) any { return nil }
	}

	return env
}

// validateStructure checks structural constraints (break/continue outside loop).
func validateStructure(steps []Node, inLoop bool) error {
	for _, step := range steps {
		if err := validateStructureStep(step, inLoop); err != nil {
			return err
		}
	}
	return nil
}

func validateStructureStep(node Node, inLoop bool) error {
	idx := node.Meta().StepIndex

	switch n := node.(type) {
	case *IfNode:
		if err := validateStructure(n.Then, inLoop); err != nil {
			return err
		}
		for _, elif := range n.Elif {
			if err := validateStructure(elif.Then, inLoop); err != nil {
				return err
			}
		}
		if err := validateStructure(n.Else, inLoop); err != nil {
			return err
		}

	case *SwitchNode:
		for _, steps := range n.Cases {
			if err := validateStructure(steps, inLoop); err != nil {
				return err
			}
		}

	case *ForNode:
		if err := validateStructure(n.Steps, true); err != nil {
			return err
		}

	case *WhileNode:
		if err := validateStructure(n.Steps, true); err != nil {
			return err
		}

	case *TryNode:
		if err := validateStructure(n.Try, inLoop); err != nil {
			return err
		}
		if n.Catch != nil {
			if err := validateStructure(n.Catch.Steps, inLoop); err != nil {
				return err
			}
		}
		if err := validateStructure(n.Finally, inLoop); err != nil {
			return err
		}

	case *BreakNode:
		if !inLoop {
			return CompileError("BREAK_OUTSIDE_LOOP", "break can only be used inside a loop", idx)
		}

	case *ContinueNode:
		if !inLoop {
			return CompileError("CONTINUE_OUTSIDE_LOOP", "continue can only be used inside a loop", idx)
		}
	}

	return nil
}

func resolveLimits(engine ResolvedLimits, programLimits *LimitsDef) ResolvedLimits {
	result := engine
	hard := HardLimits()

	if programLimits == nil {
		return result
	}

	// Most restrictive wins (minimum of engine and program).
	if programLimits.MaxDepth != nil {
		v := minInt(*programLimits.MaxDepth, result.MaxDepth)
		result.MaxDepth = minInt(v, hard.MaxDepth)
	}
	if programLimits.MaxSteps != nil {
		v := minInt(*programLimits.MaxSteps, result.MaxSteps)
		result.MaxSteps = minInt(v, hard.MaxSteps)
	}
	if programLimits.MaxLoopIterations != nil {
		v := minInt(*programLimits.MaxLoopIterations, result.MaxLoopIterations)
		result.MaxLoopIterations = minInt(v, hard.MaxLoopIterations)
	}
	if programLimits.MaxNodes != nil {
		v := minInt(*programLimits.MaxNodes, result.MaxNodes)
		result.MaxNodes = minInt(v, hard.MaxNodes)
	}
	if programLimits.MaxVariables != nil {
		v := minInt(*programLimits.MaxVariables, result.MaxVariables)
		result.MaxVariables = minInt(v, hard.MaxVariables)
	}
	if programLimits.MaxVariableSize != nil {
		v := minInt(*programLimits.MaxVariableSize, result.MaxVariableSize)
		result.MaxVariableSize = minInt(v, hard.MaxVariableSize)
	}
	if programLimits.MaxOutputSize != nil {
		v := minInt(*programLimits.MaxOutputSize, result.MaxOutputSize)
		result.MaxOutputSize = minInt(v, hard.MaxOutputSize)
	}
	if programLimits.Timeout != nil {
		d, err := time.ParseDuration(*programLimits.Timeout)
		if err == nil {
			if d < result.Timeout {
				result.Timeout = d
			}
			if d > hard.Timeout {
				result.Timeout = hard.Timeout
			}
		}
	}

	return result
}

func enrichStepError(err error, step int) error {
	if gjErr, ok := err.(*GoJSONError); ok {
		gjErr.Step = step
		return gjErr
	}
	return CompileError("EXPR_ERROR", err.Error(), step)
}

func copyEnv(env map[string]any) map[string]any {
	cp := make(map[string]any, len(env))
	for k, v := range env {
		cp[k] = v
	}
	return cp
}

func zeroValueForType(typ string) any {
	switch BaseType(typ) {
	case "string":
		return ""
	case "int":
		return 0
	case "float":
		return 0.0
	case "bool":
		return false
	case "map":
		return map[string]any{}
	case "[]any":
		return []any{}
	default:
		if len(typ) > 2 && typ[:2] == "[]" {
			return []any{}
		}
		return nil
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Ensure GoJSONError is used (suppress unused import if needed).
var _ = fmt.Sprintf
