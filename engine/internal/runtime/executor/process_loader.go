package executor

import (
	"fmt"
	"sync"

	"github.com/bitcode-framework/bitcode/internal/compiler/parser"
)

type ProcessRegistry struct {
	processes map[string]*parser.ProcessDefinition
	mu        sync.RWMutex
}

func NewProcessRegistry() *ProcessRegistry {
	return &ProcessRegistry{
		processes: make(map[string]*parser.ProcessDefinition),
	}
}

func (r *ProcessRegistry) Register(proc *parser.ProcessDefinition) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.processes[proc.Name] = proc
}

func (r *ProcessRegistry) LoadProcess(name string) (*parser.ProcessDefinition, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	proc, ok := r.processes[name]
	if !ok {
		return nil, fmt.Errorf("process %q not found", name)
	}
	return proc, nil
}

func (r *ProcessRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.processes))
	for name := range r.processes {
		names = append(names, name)
	}
	return names
}
