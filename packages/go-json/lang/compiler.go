package lang

import (
	"fmt"
	"strings"
	"time"
)

// Compile transforms a parsed Program AST into an immutable CompiledProgram.
// It validates all expressions, resolves limits, and detects structural errors.
func Compile(program *Program, engine ExprEngine, engineLimits ResolvedLimits) (*CompiledProgram, error) {
	cp := &CompiledProgram{
		Name:      program.Name,
		GoJSON:    program.GoJSON,
		AST:       program,
		Structs:   make(map[string]*CompiledStruct),
		Functions: make(map[string]*CompiledFunc),
		Input:     program.Input,
	}

	// Pass 1: register all struct names (enables forward references).
	if program.Structs != nil {
		for name := range program.Structs {
			cp.Structs[name] = nil
		}
	}

	// Pass 2: resolve aliases, then compile struct definitions.
	for name, sd := range program.Structs {
		if sd.Alias != "" {
			target, ok := program.Structs[sd.Alias]
			if !ok {
				return nil, CompileError("ALIAS_NOT_FOUND",
					fmt.Sprintf("struct '%s' aliases '%s' which is not defined", name, sd.Alias), -1)
			}
			program.Structs[name] = target
		}
	}

	for name, sd := range program.Structs {
		if sd.Alias != "" {
			continue
		}
		cs, err := compileStruct(sd, cp.Structs)
		if err != nil {
			return nil, err
		}
		cp.Structs[name] = cs
	}

	// Resolve aliased structs to their compiled targets.
	for name, sd := range program.Structs {
		if sd.Alias == "" {
			continue
		}
		target, ok := cp.Structs[sd.Alias]
		if ok && target != nil {
			cp.Structs[name] = target
		}
	}

	// Pass 3: detect non-nullable circular struct references.
	if err := detectCircularStructs(cp.Structs); err != nil {
		return nil, err
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

	// Structural validation (break/continue outside loop, frozen struct mutation).
	if err := validateStructure(program.Steps, false); err != nil {
		return nil, err
	}
	for name, cf := range cp.Functions {
		if err := validateStructure(cf.Steps, false); err != nil {
			return nil, err.(*GoJSONError).InFunction(name)
		}
	}

	// Validate frozen struct methods don't mutate self.
	for _, cs := range cp.Structs {
		if cs.Frozen && cs.Methods != nil {
			for methodName, cm := range cs.Methods {
				if err := validateNoSelfMutation(cm.Steps, cs.Name, methodName); err != nil {
					return nil, err
				}
			}
		}
		if cs.Methods != nil {
			for methodName, cm := range cs.Methods {
				if err := validateStructure(cm.Steps, false); err != nil {
					return nil, err.(*GoJSONError).InFunction(cs.Name + "." + methodName)
				}
			}
		}
	}

	cp.Limits = resolveLimits(engineLimits, program.Limits)

	return cp, nil
}

var builtinTypes = map[string]bool{
	"string": true, "int": true, "float": true, "bool": true,
	"any": true, "map": true, "nil": true, "[]any": true,
}

func isKnownType(typ string, structs map[string]*CompiledStruct) bool {
	base := BaseType(typ)
	if strings.HasPrefix(base, "[]") {
		base = base[2:]
	}
	if builtinTypes[base] {
		return true
	}
	_, exists := structs[base]
	return exists
}

func compileStruct(sd *StructDef, allStructs map[string]*CompiledStruct) (*CompiledStruct, error) {
	cs := &CompiledStruct{
		Name:   sd.Name,
		Frozen: sd.Frozen,
		Fields: sd.Fields,
	}

	for fieldName, fd := range sd.Fields {
		if !isKnownType(fd.Type, allStructs) {
			return nil, CompileError("UNKNOWN_TYPE",
				fmt.Sprintf("struct '%s' field '%s' has unknown type '%s'", sd.Name, fieldName, fd.Type), -1)
		}
	}

	if sd.Methods != nil {
		cs.Methods = make(map[string]*CompiledMethod)
		for name, md := range sd.Methods {
			cm := &CompiledMethod{
				Name:    name,
				Returns: md.Returns,
				Steps:   md.Steps,
			}
			for _, fp := range md.Params {
				cm.Params = append(cm.Params, ParamDef{
					Name:       fp.Name,
					Type:       fp.Type,
					Default:    fp.Default,
					HasDefault: fp.HasDefault,
				})
			}
			cs.Methods[name] = cm
		}
	}

	return cs, nil
}

func detectCircularStructs(structs map[string]*CompiledStruct) error {
	for name, cs := range structs {
		if cs == nil {
			continue
		}
		visited := map[string]bool{name: true}
		if err := checkStructCycle(name, cs, structs, visited); err != nil {
			return err
		}
	}
	return nil
}

func checkStructCycle(root string, cs *CompiledStruct, all map[string]*CompiledStruct, visited map[string]bool) error {
	for fieldName, fd := range cs.Fields {
		base := BaseType(fd.Type)
		if strings.HasPrefix(base, "[]") {
			base = base[2:]
		}
		if builtinTypes[base] {
			continue
		}
		if IsNullable(fd.Type) {
			continue
		}
		if visited[base] {
			return CompileError("CIRCULAR_STRUCT",
				fmt.Sprintf("struct '%s' has circular non-nullable reference through field '%s' (type '%s')",
					root, fieldName, fd.Type), -1)
		}
		ref, ok := all[base]
		if !ok || ref == nil {
			continue
		}
		visited[base] = true
		if err := checkStructCycle(root, ref, all, visited); err != nil {
			return err
		}
		delete(visited, base)
	}
	return nil
}

func validateNoSelfMutation(steps []Node, structName, methodName string) error {
	for _, step := range steps {
		if err := validateNoSelfMutationStep(step, structName, methodName); err != nil {
			return err
		}
	}
	return nil
}

func validateNoSelfMutationStep(node Node, structName, methodName string) error {
	switch n := node.(type) {
	case *SetNode:
		if strings.HasPrefix(n.Target, "self.") || n.Target == "self" {
			return CompileError("FROZEN_MUTATION",
				fmt.Sprintf("method '%s.%s' cannot mutate self — struct '%s' is frozen",
					structName, methodName, structName), n.StepIndex)
		}
	case *IfNode:
		if err := validateNoSelfMutation(n.Then, structName, methodName); err != nil {
			return err
		}
		for _, elif := range n.Elif {
			if err := validateNoSelfMutation(elif.Then, structName, methodName); err != nil {
				return err
			}
		}
		if err := validateNoSelfMutation(n.Else, structName, methodName); err != nil {
			return err
		}
	case *ForNode:
		if err := validateNoSelfMutation(n.Steps, structName, methodName); err != nil {
			return err
		}
	case *WhileNode:
		if err := validateNoSelfMutation(n.Steps, structName, methodName); err != nil {
			return err
		}
	case *TryNode:
		if err := validateNoSelfMutation(n.Try, structName, methodName); err != nil {
			return err
		}
		if n.Catch != nil {
			if err := validateNoSelfMutation(n.Catch.Steps, structName, methodName); err != nil {
				return err
			}
		}
		if err := validateNoSelfMutation(n.Finally, structName, methodName); err != nil {
			return err
		}
	case *SwitchNode:
		for _, steps := range n.Cases {
			if err := validateNoSelfMutation(steps, structName, methodName); err != nil {
				return err
			}
		}
	case *ParallelNode:
		for _, steps := range n.Branches {
			if err := validateNoSelfMutation(steps, structName, methodName); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateParallelBranchNoParentWrite(branchName string, steps []Node) error {
	declared := collectLetNames(steps)
	return checkSetTargets(branchName, steps, declared)
}

func collectLetNames(steps []Node) map[string]bool {
	names := make(map[string]bool)
	for _, step := range steps {
		if ln, ok := step.(*LetNode); ok {
			names[ln.Name] = true
		}
	}
	return names
}

func checkSetTargets(branchName string, steps []Node, declared map[string]bool) error {
	for _, step := range steps {
		switch n := step.(type) {
		case *SetNode:
			rootVar := n.Target
			if dotIdx := strings.Index(rootVar, "."); dotIdx > 0 {
				rootVar = rootVar[:dotIdx]
			}
			if bracketIdx := strings.Index(rootVar, "["); bracketIdx > 0 {
				rootVar = rootVar[:bracketIdx]
			}
			if !declared[rootVar] {
				return CompileError("PARALLEL_PARENT_WRITE",
					fmt.Sprintf("parallel branch '%s' cannot mutate parent variable '%s' — use 'into' to collect results",
						branchName, rootVar), n.StepIndex)
			}
		case *IfNode:
			if err := checkSetTargets(branchName, n.Then, declared); err != nil {
				return err
			}
			for _, elif := range n.Elif {
				if err := checkSetTargets(branchName, elif.Then, declared); err != nil {
					return err
				}
			}
			if err := checkSetTargets(branchName, n.Else, declared); err != nil {
				return err
			}
		case *ForNode:
			innerDeclared := make(map[string]bool)
			for k := range declared {
				innerDeclared[k] = true
			}
			innerDeclared[n.Variable] = true
			if n.Index != "" {
				innerDeclared[n.Index] = true
			}
			for _, s := range n.Steps {
				if ln, ok := s.(*LetNode); ok {
					innerDeclared[ln.Name] = true
				}
			}
			if err := checkSetTargets(branchName, n.Steps, innerDeclared); err != nil {
				return err
			}
		case *WhileNode:
			if err := checkSetTargets(branchName, n.Steps, declared); err != nil {
				return err
			}
		case *TryNode:
			if err := checkSetTargets(branchName, n.Try, declared); err != nil {
				return err
			}
			if n.Catch != nil {
				catchDeclared := make(map[string]bool)
				for k := range declared {
					catchDeclared[k] = true
				}
				catchDeclared[n.Catch.As] = true
				if err := checkSetTargets(branchName, n.Catch.Steps, catchDeclared); err != nil {
					return err
				}
			}
			if err := checkSetTargets(branchName, n.Finally, declared); err != nil {
				return err
			}
		}
	}
	return nil
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

	case *ParallelNode:
		for branchName, steps := range n.Branches {
			if err := validateStructure(steps, inLoop); err != nil {
				return err
			}
			if err := validateParallelBranchNoParentWrite(branchName, steps); err != nil {
				return err
			}
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
