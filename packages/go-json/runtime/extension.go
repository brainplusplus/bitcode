package runtime

import (
	"fmt"
	"log"
	"sync"
)

// Extension defines an external capability set that can be injected into the go-json runtime.
// Programs access extensions via "ext:name" imports.
type Extension struct {
	Name      string
	Functions map[string]any
	Structs   map[string]any // reserved for future struct injection
	Constants map[string]any // reserved for future constant injection
}

type extensionRegistry struct {
	extensions map[string]*Extension
	mu         sync.RWMutex
}

func newExtensionRegistry() *extensionRegistry {
	return &extensionRegistry{
		extensions: make(map[string]*Extension),
	}
}

func (r *extensionRegistry) register(name string, ext *Extension) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.extensions[name]; exists {
		return fmt.Errorf("extension '%s' already registered", name)
	}
	r.extensions[name] = ext
	return nil
}

func (r *extensionRegistry) get(name string) *Extension {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.extensions[name]
}

func (r *extensionRegistry) all() map[string]*Extension {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]*Extension, len(r.extensions))
	for k, v := range r.extensions {
		result[k] = v
	}
	return result
}

// WithExtension registers an extension that programs can import via "ext:name".
func WithExtension(name string, ext Extension) Option {
	return func(r *Runtime) {
		if ext.Structs != nil && len(ext.Structs) > 0 {
			log.Printf("go-json: extension '%s' has Structs field populated — this is reserved for future use and will be ignored", name)
		}
		if ext.Constants != nil && len(ext.Constants) > 0 {
			log.Printf("go-json: extension '%s' has Constants field populated — this is reserved for future use and will be ignored", name)
		}

		extCopy := &Extension{
			Name:      name,
			Functions: ext.Functions,
			Structs:   ext.Structs,
			Constants: ext.Constants,
		}
		r.extensions.register(name, extCopy)
	}
}
