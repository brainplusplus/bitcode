package model

import (
	"fmt"
	"sync"

	"github.com/bitcode-engine/engine/internal/compiler/parser"
)

type Registry struct {
	models     map[string]*parser.ModelDefinition
	tableNames map[string]string
	mu         sync.RWMutex
}

func NewRegistry() *Registry {
	return &Registry{
		models:     make(map[string]*parser.ModelDefinition),
		tableNames: make(map[string]string),
	}
}

func (r *Registry) Register(model *parser.ModelDefinition) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if model.Name == "" {
		return fmt.Errorf("model name is required")
	}

	key := model.Name
	if model.Module != "" {
		key = model.Module + "." + model.Name
	}

	r.models[key] = model
	r.models[model.Name] = model
	return nil
}

func (r *Registry) RegisterWithModule(model *parser.ModelDefinition, moduleDef *parser.ModuleDefinition) error {
	if err := r.Register(model); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tableNames[model.Name] = ResolveTableName(model, moduleDef)
	return nil
}

func ResolveTableName(model *parser.ModelDefinition, moduleDef *parser.ModuleDefinition) string {
	if model.TableName != "" {
		return model.TableName
	}
	if model.TablePrefix != nil {
		prefix := *model.TablePrefix
		if prefix == "" {
			return model.Name
		}
		return prefix + "_" + model.Name
	}
	if moduleDef != nil && moduleDef.Table != nil && moduleDef.Table.Prefix != "" {
		return moduleDef.Table.Prefix + "_" + model.Name
	}
	return model.Name
}

func (r *Registry) Get(name string) (*parser.ModelDefinition, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	model, ok := r.models[name]
	if !ok {
		return nil, fmt.Errorf("model %q not found", name)
	}
	return model, nil
}

func (r *Registry) List() []*parser.ModelDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	seen := make(map[string]bool)
	var result []*parser.ModelDefinition
	for _, m := range r.models {
		if !seen[m.Name] {
			seen[m.Name] = true
			result = append(result, m)
		}
	}
	return result
}

func (r *Registry) Has(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.models[name]
	return ok
}

func (r *Registry) TableName(modelName string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if tn, ok := r.tableNames[modelName]; ok {
		return tn
	}
	return modelName
}
