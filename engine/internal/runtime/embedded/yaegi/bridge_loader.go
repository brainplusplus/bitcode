package yaegi_runtime

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type BridgeSource struct {
	Filename string
	Code     string
}

// LoadCustomBridges scans bridges/ folders at project and module level,
// returning source code to be evaluated in each VM's interpreter.
// Module-level bridges override project-level bridges with the same filename.
func LoadCustomBridges(projectDir string, moduleNames []string, modulesDir string) ([]BridgeSource, error) {
	bridgeMap := make(map[string]BridgeSource)

	projectBridgesDir := filepath.Join(projectDir, "bridges")
	if dirExists(projectBridgesDir) {
		if err := collectBridgeSources(projectBridgesDir, bridgeMap); err != nil {
			return nil, fmt.Errorf("loading project bridges: %w", err)
		}
	}

	for _, modName := range moduleNames {
		modBridgesDir := filepath.Join(modulesDir, modName, "bridges")
		if dirExists(modBridgesDir) {
			if err := collectBridgeSources(modBridgesDir, bridgeMap); err != nil {
				return nil, fmt.Errorf("loading bridges for module %s: %w", modName, err)
			}
		}
	}

	sources := make([]BridgeSource, 0, len(bridgeMap))
	for _, src := range bridgeMap {
		sources = append(sources, src)
	}
	return sources, nil
}

func collectBridgeSources(dir string, bridgeMap map[string]BridgeSource) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}

		fullPath := filepath.Join(dir, entry.Name())
		code, err := os.ReadFile(fullPath)
		if err != nil {
			return fmt.Errorf("reading bridge file %s: %w", entry.Name(), err)
		}

		bridgeMap[entry.Name()] = BridgeSource{
			Filename: entry.Name(),
			Code:     string(code),
		}
	}
	return nil
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
