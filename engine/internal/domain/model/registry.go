package model

import (
	"fmt"
	"sync"

	"github.com/bitcode-framework/bitcode/internal/compiler/parser"
	"github.com/jinzhu/inflection"
)

type Registry struct {
	models       map[string]*parser.ModelDefinition
	tableNames   map[string]string
	moduleNames  map[string][]string
	mu           sync.RWMutex
	TableNaming  string // "singular" (default) or "plural"
}

func NewRegistry() *Registry {
	return &Registry{
		models:      make(map[string]*parser.ModelDefinition),
		tableNames:  make(map[string]string),
		moduleNames: make(map[string][]string),
	}
}

func (r *Registry) Register(model *parser.ModelDefinition) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if model.Name == "" {
		return fmt.Errorf("model name is required")
	}

	if model.Module != "" {
		qualifiedKey := model.Module + "." + model.Name
		if existing, ok := r.models[qualifiedKey]; ok {
			if existing.Module == model.Module && model.Inherit == "" {
				return fmt.Errorf(
					"duplicate model name %q in module %q (check models/ directory for duplicate \"name\" property)",
					model.Name, model.Module,
				)
			}
		}
		r.models[qualifiedKey] = model
		r.moduleNames[model.Name] = appendUnique(r.moduleNames[model.Name], model.Module)
	}

	r.models[model.Name] = model
	return nil
}

func (r *Registry) RegisterWithModule(model *parser.ModelDefinition, moduleDef *parser.ModuleDefinition) error {
	if err := r.Register(model); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	tableName := ResolveTableName(model, moduleDef, r.TableNaming)
	r.tableNames[model.Name] = tableName
	if model.Module != "" {
		r.tableNames[model.Module+"."+model.Name] = tableName
	}
	return nil
}

func ResolveTableName(model *parser.ModelDefinition, moduleDef *parser.ModuleDefinition, tableNaming ...string) string {
	if model.TableName != "" {
		return model.TableName
	}

	name := model.Name

	prefix := ""
	if model.TablePrefix != nil {
		prefix = *model.TablePrefix
	} else if moduleDef != nil && moduleDef.Table != nil && moduleDef.Table.Prefix != "" {
		prefix = moduleDef.Table.Prefix
	}

	shouldPlural := false
	if model.TablePlural != nil {
		shouldPlural = *model.TablePlural
	} else if len(tableNaming) > 0 && tableNaming[0] == "plural" {
		shouldPlural = true
	}

	if shouldPlural {
		name = inflection.Plural(name)
	}

	if prefix == "" {
		return name
	}
	return prefix + "_" + name
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

func (r *Registry) IsAmbiguous(modelName string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.moduleNames[modelName]) > 1
}

func (r *Registry) ModulesForModel(modelName string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.moduleNames[modelName]
}

func appendUnique(slice []string, val string) []string {
	for _, s := range slice {
		if s == val {
			return slice
		}
	}
	return append(slice, val)
}
