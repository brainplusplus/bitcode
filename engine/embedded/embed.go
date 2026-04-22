package embedded

import "embed"

//go:embed all:modules
var ModulesFS embed.FS
