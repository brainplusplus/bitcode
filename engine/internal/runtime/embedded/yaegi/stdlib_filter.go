package yaegi_runtime

import (
	"reflect"
	"strings"

	"github.com/traefik/yaegi/stdlib"
)

// blockedPackages lists stdlib packages that scripts must NOT access.
// yaegi's default stdlib already excludes unsafe, syscall, os/exec, and plugin,
// but we list them here for defense-in-depth in case future yaegi versions add them.
var blockedPackages = []string{
	"os/exec",
	"unsafe",
	"syscall",
	"plugin",
}

// blockedOSSymbols lists os package symbols that are dangerous even in restricted mode.
// yaegi already replaces os.Exit with a panic in restricted mode, but we remove
// additional symbols that could affect the host process.
var blockedOSSymbols = []string{
	"Exit", // would kill engine process (yaegi replaces with panic, but we double-check)
}

// FilteredStdlib returns a copy of yaegi's stdlib.Symbols with blocked packages
// and dangerous symbols removed. This is the symbol set passed to interp.Use().
func FilteredStdlib() map[string]map[string]reflect.Value {
	filtered := make(map[string]map[string]reflect.Value, len(stdlib.Symbols))

	blockedSet := make(map[string]bool, len(blockedPackages))
	for _, pkg := range blockedPackages {
		// yaegi uses "pkg_path/pkg_name" as key, e.g. "os/exec/exec"
		parts := strings.Split(pkg, "/")
		pkgName := parts[len(parts)-1]
		blockedSet[pkg+"/"+pkgName] = true
	}

	for key, symbols := range stdlib.Symbols {
		if blockedSet[key] {
			continue
		}

		// For the "os/os" package, filter out dangerous symbols
		if key == "os/os" {
			safeCopy := make(map[string]reflect.Value, len(symbols))
			for symName, symVal := range symbols {
				if isBlockedOSSymbol(symName) {
					continue
				}
				safeCopy[symName] = symVal
			}
			filtered[key] = safeCopy
			continue
		}

		filtered[key] = symbols
	}

	return filtered
}

func isBlockedOSSymbol(name string) bool {
	for _, blocked := range blockedOSSymbols {
		if name == blocked {
			return true
		}
	}
	return false
}
