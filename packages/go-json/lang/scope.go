package lang

import (
	"sync"
)

type VarInfo struct {
	Value    any
	Type     string
	Declared bool
}

type Scope struct {
	vars   map[string]*VarInfo
	parent *Scope
	name   string
	mu     sync.RWMutex
}

func NewScope(name string) *Scope {
	return &Scope{
		vars: make(map[string]*VarInfo),
		name: name,
	}
}

// Declare creates a new variable in the current scope.
// Returns error if the variable already exists in this scope (not parent).
func (s *Scope) Declare(name string, value any, typ string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.vars[name]; exists {
		return CompileError("VAR_EXISTS", "variable '"+name+"' already declared in this scope", -1)
	}

	s.vars[name] = &VarInfo{
		Value:    value,
		Type:     typ,
		Declared: true,
	}
	return nil
}

// Get searches up the scope chain for a variable.
// Returns value, type, and whether the variable was found.
func (s *Scope) Get(name string) (any, string, bool) {
	s.mu.RLock()
	if v, ok := s.vars[name]; ok {
		s.mu.RUnlock()
		return v.Value, v.Type, true
	}
	s.mu.RUnlock()

	if s.parent != nil {
		return s.parent.Get(name)
	}
	return nil, "", false
}

// Set updates an existing variable, searching up the scope chain.
// Returns error if variable not found or type is incompatible.
func (s *Scope) Set(name string, value any, newType string) error {
	s.mu.Lock()
	if v, ok := s.vars[name]; ok {
		if v.Type != "any" && v.Type != "" && newType != "" && !TypesCompatible(v.Type, newType) {
			s.mu.Unlock()
			return RuntimeError("TYPE_MISMATCH",
				"cannot assign "+newType+" to variable '"+name+"' (type "+v.Type+")", -1)
		}
		v.Value = value
		if v.Type == "" || v.Type == "any" {
			// Keep existing type unless it was unset.
		} else if newType != "" {
			// Type already set and compatible — keep original type.
		}
		s.mu.Unlock()
		return nil
	}
	s.mu.Unlock()

	if s.parent != nil {
		return s.parent.Set(name, value, newType)
	}

	return RuntimeError("VAR_NOT_FOUND", "variable '"+name+"' not defined", -1)
}

// Has checks if a variable exists in the current scope only (not parent).
func (s *Scope) Has(name string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.vars[name]
	return ok
}

// AllNames returns all accessible variable names (current + ancestors).
// Used for "did you mean?" suggestions.
func (s *Scope) AllNames() []string {
	seen := make(map[string]bool)
	var names []string

	current := s
	for current != nil {
		current.mu.RLock()
		for name := range current.vars {
			if !seen[name] {
				seen[name] = true
				names = append(names, name)
			}
		}
		current.mu.RUnlock()
		current = current.parent
	}

	return names
}

// NewChild creates a child scope with parent link (for if/for/while blocks).
// The child can read and mutate parent variables.
func (s *Scope) NewChild(name string) *Scope {
	return &Scope{
		vars:   make(map[string]*VarInfo),
		parent: s,
		name:   name,
	}
}

// IsolatedChild creates a child scope WITHOUT parent link (for function calls).
// The child cannot access parent variables — full isolation.
func (s *Scope) IsolatedChild(name string) *Scope {
	return &Scope{
		vars: make(map[string]*VarInfo),
		name: name,
		// parent intentionally nil — isolation
	}
}

// ToMap exports all accessible variables as map[string]any for expr-lang env.
// Child variables shadow parent variables with the same name.
func (s *Scope) ToMap() map[string]any {
	result := make(map[string]any)

	// Collect from root to leaf so child values override parent values.
	var chain []*Scope
	current := s
	for current != nil {
		chain = append(chain, current)
		current = current.parent
	}

	// Iterate from root (last) to leaf (first).
	for i := len(chain) - 1; i >= 0; i-- {
		sc := chain[i]
		sc.mu.RLock()
		for name, v := range sc.vars {
			result[name] = v.Value
		}
		sc.mu.RUnlock()
	}

	return result
}

// Name returns the scope's name (for debugging).
func (s *Scope) Name() string {
	return s.name
}

// VarCount returns the total number of variables accessible (current + ancestors).
func (s *Scope) VarCount() int {
	count := 0
	current := s
	for current != nil {
		current.mu.RLock()
		count += len(current.vars)
		current.mu.RUnlock()
		current = current.parent
	}
	return count
}

// GetVarInfo returns the full VarInfo for a variable, searching up the chain.
func (s *Scope) GetVarInfo(name string) (*VarInfo, bool) {
	s.mu.RLock()
	if v, ok := s.vars[name]; ok {
		s.mu.RUnlock()
		return v, true
	}
	s.mu.RUnlock()

	if s.parent != nil {
		return s.parent.GetVarInfo(name)
	}
	return nil, false
}
