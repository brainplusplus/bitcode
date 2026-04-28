package embedded

import (
	"time"

	"github.com/bitcode-framework/bitcode/internal/runtime/bridge"
)

type EmbeddedRuntime interface {
	Name() string
	NewVM(opts VMOptions) (VM, error)
}

type VM interface {
	InjectBridge(bc *bridge.Context) error
	InjectParams(params map[string]any) error
	Execute(code string, filename string) (any, error)
	Interrupt(reason string)
	Close()
}

type VMOptions struct {
	Timeout       time.Duration
	MaxMemoryMB   int
	HardMaxMemMB  int
}
