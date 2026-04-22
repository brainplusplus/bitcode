package module

import (
	"fmt"

	"github.com/bitcode-engine/engine/internal/compiler/parser"
)

func ResolveDependencies(modules map[string]*parser.ModuleDefinition, targets ...string) ([]string, error) {
	visited := make(map[string]bool)
	inStack := make(map[string]bool)
	var order []string

	var visit func(name string) error
	visit = func(name string) error {
		if inStack[name] {
			return fmt.Errorf("circular dependency detected: %s", name)
		}
		if visited[name] {
			return nil
		}

		inStack[name] = true
		mod, ok := modules[name]
		if !ok {
			return fmt.Errorf("module %q not found", name)
		}

		for _, dep := range mod.Depends {
			if err := visit(dep); err != nil {
				return err
			}
		}

		inStack[name] = false
		visited[name] = true
		order = append(order, name)
		return nil
	}

	for _, target := range targets {
		if err := visit(target); err != nil {
			return nil, err
		}
	}
	return order, nil
}
