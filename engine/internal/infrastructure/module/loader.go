package module

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bitcode-framework/bitcode/internal/compiler/parser"
)

type LoadedModule struct {
	Definition *parser.ModuleDefinition
	Models     []*parser.ModelDefinition
	APIs       []*parser.APIDefinition
	Path       string
}

func LoadModule(modulePath string) (*LoadedModule, error) {
	mfs := NewDiskFS(modulePath)
	loaded, err := LoadModuleFromFS(mfs)
	if err != nil {
		return nil, err
	}
	loaded.Path = modulePath
	return loaded, nil
}

func LoadModuleFromFS(mfs ModuleFS) (*LoadedModule, error) {
	data, err := mfs.ReadFile("module.json")
	if err != nil {
		return nil, fmt.Errorf("failed to read module.json: %w", err)
	}

	modDef, err := parser.ParseModule(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse module.json: %w", err)
	}

	loaded := &LoadedModule{
		Definition: modDef,
	}

	loaded.Models, err = loadGlobParsedFromFS(mfs, modDef.Models, func(data []byte) (*parser.ModelDefinition, error) {
		return parser.ParseModel(data)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to load models for %s: %w", modDef.Name, err)
	}

	loaded.APIs, err = loadGlobParsedFromFS(mfs, modDef.APIs, func(data []byte) (*parser.APIDefinition, error) {
		return parser.ParseAPI(data)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to load APIs for %s: %w", modDef.Name, err)
	}

	return loaded, nil
}

func loadGlobParsedFromFS[T any](mfs ModuleFS, patterns []string, parseFn func([]byte) (*T, error)) ([]*T, error) {
	var results []*T
	for _, pattern := range patterns {
		matches, err := mfs.Glob(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid glob pattern %s: %w", pattern, err)
		}
		for _, match := range matches {
			data, err := mfs.ReadFile(match)
			if err != nil {
				return nil, fmt.Errorf("cannot read %s: %w", match, err)
			}
			parsed, err := parseFn(data)
			if err != nil {
				return nil, fmt.Errorf("failed to parse %s: %w", match, err)
			}
			results = append(results, parsed)
		}
	}
	return results, nil
}

func loadGlobParsed[T any](basePath string, patterns []string, parseFn func([]byte) (*T, error)) ([]*T, error) {
	var results []*T
	for _, pattern := range patterns {
		fullPattern := filepath.Join(basePath, pattern)
		matches, err := filepath.Glob(fullPattern)
		if err != nil {
			return nil, fmt.Errorf("invalid glob pattern %s: %w", pattern, err)
		}
		for _, match := range matches {
			data, err := os.ReadFile(match)
			if err != nil {
				return nil, fmt.Errorf("cannot read %s: %w", match, err)
			}
			parsed, err := parseFn(data)
			if err != nil {
				return nil, fmt.Errorf("failed to parse %s: %w", match, err)
			}
			results = append(results, parsed)
		}
	}
	return results, nil
}
