package lang

import "time"

// CompiledProgram is the immutable output of the compiler.
// Safe for concurrent use — each execution gets its own VM and scope.
type CompiledProgram struct {
	Name      string
	GoJSON    string
	AST       *Program
	Functions map[string]*CompiledFunc
	Input     []InputField
	Limits    ResolvedLimits
}

// CompiledFunc is a compiled function ready for execution.
type CompiledFunc struct {
	Name    string
	Params  []ParamDef
	Returns string
	Steps   []Node
}

// ParamDef describes a function parameter with optional default value.
type ParamDef struct {
	Name       string
	Type       string
	Default    any
	HasDefault bool
}

// ResolvedLimits holds concrete limit values after resolution
// (most restrictive of engine, project, and program limits).
type ResolvedLimits struct {
	MaxDepth          int
	MaxSteps          int
	MaxLoopIterations int
	MaxNodes          int
	MaxVariables      int
	MaxVariableSize   int
	MaxOutputSize     int
	Timeout           time.Duration
}

// DefaultLimits returns the default resource limits.
func DefaultLimits() ResolvedLimits {
	return ResolvedLimits{
		MaxDepth:          1000,
		MaxSteps:          10000,
		MaxLoopIterations: 10000,
		MaxNodes:          1000,
		MaxVariables:      1000,
		MaxVariableSize:   10 * 1024 * 1024, // 10MB
		MaxOutputSize:     50 * 1024 * 1024,  // 50MB
		Timeout:           30 * time.Second,
	}
}

// HardLimits returns the absolute maximum limits that cannot be exceeded.
func HardLimits() ResolvedLimits {
	return ResolvedLimits{
		MaxDepth:          10000,
		MaxSteps:          100000,
		MaxLoopIterations: 100000,
		MaxNodes:          10000,
		MaxVariables:      10000,
		MaxVariableSize:   100 * 1024 * 1024, // 100MB
		MaxOutputSize:     500 * 1024 * 1024,  // 500MB
		Timeout:           5 * time.Minute,
	}
}
