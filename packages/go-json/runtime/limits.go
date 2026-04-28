package runtime

import (
	"time"

	"github.com/bitcode-framework/go-json/lang"
)

type Limits struct {
	MaxDepth          int
	MaxSteps          int
	MaxLoopIterations int
	MaxNodes          int
	MaxVariables      int
	MaxVariableSize   int
	MaxOutputSize     int
	Timeout           time.Duration
}

func DefaultLimits() Limits {
	d := lang.DefaultLimits()
	return Limits{
		MaxDepth:          d.MaxDepth,
		MaxSteps:          d.MaxSteps,
		MaxLoopIterations: d.MaxLoopIterations,
		MaxNodes:          d.MaxNodes,
		MaxVariables:      d.MaxVariables,
		MaxVariableSize:   d.MaxVariableSize,
		MaxOutputSize:     d.MaxOutputSize,
		Timeout:           d.Timeout,
	}
}

func HardLimits() Limits {
	h := lang.HardLimits()
	return Limits{
		MaxDepth:          h.MaxDepth,
		MaxSteps:          h.MaxSteps,
		MaxLoopIterations: h.MaxLoopIterations,
		MaxNodes:          h.MaxNodes,
		MaxVariables:      h.MaxVariables,
		MaxVariableSize:   h.MaxVariableSize,
		MaxOutputSize:     h.MaxOutputSize,
		Timeout:           h.Timeout,
	}
}

func (l Limits) ToResolved() lang.ResolvedLimits {
	return lang.ResolvedLimits{
		MaxDepth:          l.MaxDepth,
		MaxSteps:          l.MaxSteps,
		MaxLoopIterations: l.MaxLoopIterations,
		MaxNodes:          l.MaxNodes,
		MaxVariables:      l.MaxVariables,
		MaxVariableSize:   l.MaxVariableSize,
		MaxOutputSize:     l.MaxOutputSize,
		Timeout:           l.Timeout,
	}
}
