package lang

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
)

// ImportResolver resolves and loads imported go-json files.
type ImportResolver struct {
	cache   map[string]*Program
	cacheMu sync.RWMutex
}

// NewImportResolver creates a new import resolver with caching.
func NewImportResolver() *ImportResolver {
	return &ImportResolver{
		cache: make(map[string]*Program),
	}
}

// ResolveImports processes all imports for a program, loading and parsing imported files.
// basePath is the directory of the importing file.
// importStack tracks the chain for circular import detection.
func (ir *ImportResolver) ResolveImports(program *Program, basePath string, importStack []string) error {
	if len(program.Imports) == 0 {
		return nil
	}

	for _, imp := range program.Imports {
		if imp.PathType != "relative" {
			continue
		}

		resolvedPath := ir.resolvePath(imp.Path, basePath)

		if ir.isInStack(resolvedPath, importStack) {
			chain := append(importStack, resolvedPath)
			return CompileError("CIRCULAR_IMPORT",
				fmt.Sprintf("circular import detected: %s", strings.Join(chain, " → ")), -1)
		}

		imported, err := ir.loadFile(resolvedPath, append(importStack, resolvedPath))
		if err != nil {
			return CompileError("IMPORT_ERROR",
				fmt.Sprintf("error importing '%s' (alias '%s'): %s", imp.Path, imp.Alias, err.Error()), -1)
		}

		ir.mergeExports(program, imported, imp.Alias)
	}

	return nil
}

func (ir *ImportResolver) resolvePath(importPath, basePath string) string {
	if strings.HasSuffix(importPath, ".json") || strings.HasSuffix(importPath, ".jsonc") {
		return filepath.Join(basePath, importPath)
	}
	jsonPath := filepath.Join(basePath, importPath+".json")
	return jsonPath
}

func (ir *ImportResolver) isInStack(path string, stack []string) bool {
	for _, s := range stack {
		if s == path {
			return true
		}
	}
	return false
}

func (ir *ImportResolver) loadFile(path string, importStack []string) (*Program, error) {
	ir.cacheMu.RLock()
	if cached, ok := ir.cache[path]; ok {
		ir.cacheMu.RUnlock()
		return cached, nil
	}
	ir.cacheMu.RUnlock()

	program, err := ParseFile(path)
	if err != nil {
		return nil, err
	}

	dir := filepath.Dir(path)
	if err := ir.ResolveImports(program, dir, importStack); err != nil {
		return nil, err
	}

	ir.cacheMu.Lock()
	ir.cache[path] = program
	ir.cacheMu.Unlock()

	return program, nil
}

func (ir *ImportResolver) mergeExports(target, source *Program, alias string) {
	if source.Structs != nil {
		if target.Structs == nil {
			target.Structs = make(map[string]*StructDef)
		}
		for name, sd := range source.Structs {
			target.Structs[alias+"."+name] = sd
		}
	}

	if source.Functions != nil {
		if target.Functions == nil {
			target.Functions = make(map[string]*FuncDef)
		}
		for name, fd := range source.Functions {
			target.Functions[alias+"."+name] = fd
		}
	}
}
