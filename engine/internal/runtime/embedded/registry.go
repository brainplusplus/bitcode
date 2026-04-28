package embedded

import (
	"fmt"
	"strings"
	"sync"
)

type EngineRegistry struct {
	engines map[string]EmbeddedRuntime
	mu      sync.RWMutex
}

func NewRegistry() *EngineRegistry {
	return &EngineRegistry{engines: make(map[string]EmbeddedRuntime)}
}

func (r *EngineRegistry) Register(name string, runtime EmbeddedRuntime) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.engines[name] = runtime
}

func (r *EngineRegistry) Get(name string) (EmbeddedRuntime, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	engine, ok := r.engines[name]
	if !ok {
		return nil, fmt.Errorf("embedded JS engine '%s' not found, available: %v", name, r.Names())
	}
	return engine, nil
}

func (r *EngineRegistry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.engines))
	for name := range r.engines {
		names = append(names, name)
	}
	return names
}

func (r *EngineRegistry) Resolve(runtimeField string, defaultEngine string) (EmbeddedRuntime, error) {
	engine := ParseEngine(runtimeField)
	if engine == "" {
		engine = defaultEngine
	}
	if engine == "" {
		engine = "goja"
	}
	return r.Get(engine)
}

func ParseEngine(runtimeField string) string {
	if !strings.HasPrefix(runtimeField, "javascript") {
		return ""
	}
	parts := strings.SplitN(runtimeField, ":", 2)
	if len(parts) == 2 {
		return parts[1]
	}
	return ""
}
