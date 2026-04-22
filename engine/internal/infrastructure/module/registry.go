package module

import (
	"fmt"
	"sync"

	"github.com/bitcode-engine/engine/internal/compiler/parser"
)

type ModuleState string

const (
	StateInstalled   ModuleState = "installed"
	StateUninstalled ModuleState = "uninstalled"
)

type InstalledModule struct {
	Definition *parser.ModuleDefinition
	State      ModuleState
	Path       string
}

type Registry struct {
	modules map[string]*InstalledModule
	mu      sync.RWMutex
}

func NewRegistry() *Registry {
	return &Registry{
		modules: make(map[string]*InstalledModule),
	}
}

func (r *Registry) Register(mod *parser.ModuleDefinition, path string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.modules[mod.Name] = &InstalledModule{
		Definition: mod,
		State:      StateInstalled,
		Path:       path,
	}
}

func (r *Registry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if m, ok := r.modules[name]; ok {
		m.State = StateUninstalled
	}
}

func (r *Registry) Get(name string) (*InstalledModule, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	m, ok := r.modules[name]
	if !ok {
		return nil, fmt.Errorf("module %q not found", name)
	}
	return m, nil
}

func (r *Registry) IsInstalled(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	m, ok := r.modules[name]
	return ok && m.State == StateInstalled
}

func (r *Registry) List() []*InstalledModule {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*InstalledModule, 0, len(r.modules))
	for _, m := range r.modules {
		result = append(result, m)
	}
	return result
}

func (r *Registry) InstalledNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var names []string
	for name, m := range r.modules {
		if m.State == StateInstalled {
			names = append(names, name)
		}
	}
	return names
}
