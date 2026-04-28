package io

import (
	"fmt"
	"sync"

	"github.com/expr-lang/expr"
)

// IOModule is the interface that all I/O modules must implement.
// Each module provides a set of functions that can be registered
// in the expr-lang environment for use in go-json programs.
type IOModule interface {
	// Name returns the module identifier (e.g. "http", "fs", "sql", "exec").
	Name() string
	// Functions returns the function map for expr-lang registration.
	Functions() map[string]any
	// SetConfig applies runtime configuration to the module.
	SetConfig(cfg map[string]any)
}

// IORegistry manages registered I/O modules.
type IORegistry struct {
	modules map[string]IOModule
	mu      sync.RWMutex
}

// NewIORegistry creates a new empty I/O registry.
func NewIORegistry() *IORegistry {
	return &IORegistry{
		modules: make(map[string]IOModule),
	}
}

// RegisterModule registers an I/O module by name.
// Returns an error if a module with the same name is already registered.
func (r *IORegistry) RegisterModule(name string, module IOModule) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.modules[name]; exists {
		return fmt.Errorf("I/O module '%s' already registered", name)
	}
	r.modules[name] = module
	return nil
}

// GetModule returns a registered module by name, or nil if not found.
func (r *IORegistry) GetModule(name string) IOModule {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.modules[name]
}

// AllModules returns all registered modules.
func (r *IORegistry) AllModules() map[string]IOModule {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]IOModule, len(r.modules))
	for k, v := range r.modules {
		result[k] = v
	}
	return result
}

// HasModule checks if a module is registered.
func (r *IORegistry) HasModule(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.modules[name]
	return exists
}

// ModuleNames returns the names of all registered modules.
func (r *IORegistry) ModuleNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.modules))
	for name := range r.modules {
		names = append(names, name)
	}
	return names
}

// ExprOptions returns expr.Option entries for all registered modules.
// Each function is registered as "moduleName.functionName".
func (r *IORegistry) ExprOptions() []expr.Option {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var opts []expr.Option
	for _, mod := range r.modules {
		for fnName, fn := range mod.Functions() {
			fullName := mod.Name() + "." + fnName
			opts = append(opts, expr.Function(fullName, fn.(func(...any) (any, error))))
		}
	}
	return opts
}

// EnvVars returns environment variable entries for all registered modules.
// This injects module function maps as namespace objects into the expression environment.
func (r *IORegistry) EnvVars() map[string]any {
	r.mu.RLock()
	defer r.mu.RUnlock()

	env := make(map[string]any)
	for _, mod := range r.modules {
		env[mod.Name()] = mod.Functions()
	}
	return env
}
