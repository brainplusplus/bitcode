# Embedded Module + Publish System Design

**Date**: 2026-04-22
**Status**: Approved
**Author**: Brainstorming session

## Problem

BitCode Engine ships a `base` module (auth, users, roles, groups, permissions, settings, audit logs) that every project needs. Currently, users must copy this module into their project's `modules/` directory. We want:

1. **Zero-config**: Engine runs out-of-the-box with base module embedded in the binary
2. **Override**: Users can publish (extract) specific files to their project for customization
3. **Granularity**: Publish whole module, per-resource-type, or per-file (like Laravel's `artisan vendor:publish`)

## Decision

**Approach 1: Go `embed.FS` + 3-Layer Module Resolution**

Base module embedded in the Go binary. Engine resolves module files from 3 layers (project > global > embedded). CLI `bitcode publish` extracts files from embedded to project.

## Architecture

### Embedding

Base module files live at `engine/embedded/modules/base/` and are embedded via Go's `//go:embed` directive:

```go
// engine/embedded/embed.go
package embedded

import "embed"

//go:embed modules/*
var ModulesFS embed.FS
```

Only the `base` module is embedded. Other modules (CRM, HRM, etc.) come from a registry or are user-created.

### 3-Layer Resolution

```
Layer 1: Project modules    -> ./modules/base/...          (highest priority)
Layer 2: Global modules     -> ~/.bitcode/modules/base/... (shared across projects)
Layer 3: Embedded modules   -> [binary]/base/...           (default fallback)
```

Resolution is **per-file**. If `./modules/base/models/user.json` exists in the project, it overrides the embedded version. All other base files fall back to embedded.

### ModuleFS Abstraction

```go
type ModuleFS interface {
    ReadFile(path string) ([]byte, error)
    Glob(pattern string) ([]string, error)
    ReadDir(path string) ([]fs.DirEntry, error)
}
```

Three implementations:
- **DiskFS**: Reads from filesystem (project or global directory)
- **EmbedFS**: Reads from `embed.FS` (binary)
- **LayeredFS**: Composes multiple FS implementations, resolves per-file with priority

```go
type LayeredFS struct {
    layers []ModuleFS  // index 0 = highest priority
}

func (l *LayeredFS) ReadFile(path string) ([]byte, error) {
    for _, layer := range l.layers {
        data, err := layer.ReadFile(path)
        if err == nil {
            return data, nil
        }
    }
    return nil, fs.ErrNotExist
}
```

### Module Loading (Updated)

```go
func (a *App) LoadModules() error {
    projectFS := NewDiskFS(a.Config.ModuleDir)
    globalFS  := NewDiskFS(filepath.Join(homeDir, ".bitcode", "modules"))
    embedFS   := NewEmbedFS(embedded.ModulesFS)
    layered   := NewLayeredFS(projectFS, globalFS, embedFS)

    moduleNames := layered.DiscoverModules()
    for _, name := range moduleNames {
        modFS := layered.SubFS(name)
        loaded, err := module.LoadModuleFromFS(modFS)
        // ... install as before
    }
}
```

### CLI: `bitcode publish`

```bash
# Publish entire module
bitcode publish base

# Publish per-resource type
bitcode publish base --models
bitcode publish base --views
bitcode publish base --templates
bitcode publish base --apis
bitcode publish base --processes
bitcode publish base --data

# Publish specific file
bitcode publish base models/user.json

# Options
bitcode publish base --force      # Overwrite existing files
bitcode publish base --dry-run    # Preview without writing
bitcode publish --list            # List publishable modules
```

**Default behavior**: Skip existing files (no overwrite). Print warning for skipped files. Use `--force` to overwrite.

## Files Changed

| File | Change |
|------|--------|
| `engine/embedded/modules/base/` | NEW - base module moved here |
| `engine/embedded/embed.go` | NEW - `//go:embed` directive |
| `engine/internal/infrastructure/module/fs.go` | NEW - ModuleFS, DiskFS, EmbedFS, LayeredFS |
| `engine/internal/infrastructure/module/loader.go` | MODIFY - LoadModuleFromFS() using ModuleFS |
| `engine/internal/app.go` | MODIFY - LoadModules() using LayeredFS |
| `engine/internal/config.go` | MODIFY - add GlobalModuleDir config |
| `engine/cmd/bitcode/main.go` | MODIFY - add publish command |
| `engine/cmd/bitcode/publish.go` | NEW - publish command implementation |
| `engine/modules/base/` | DELETE - moved to engine/embedded/ |

## What Does NOT Change

- `samples/erp/` - unaffected
- Module JSON format - zero change
- `bitcode.yaml` - unchanged
- Existing CLI commands - unchanged
- Runtime behavior - identical, only resolution path changes

## Migration

**Breaking changes**: NONE. Purely additive.

- Projects with `modules/base/` (copied from sample): Still works, project layer wins
- Projects WITHOUT `modules/base/`: Engine auto-loads from embedded, zero config

## Testing Strategy

- Unit test `LayeredFS` - resolve priority, file not found fallback
- Unit test `LoadModuleFromFS` - load from embed vs disk
- Integration test - engine boots with embedded base only (no project modules)
- Integration test - engine boots with project override on top of embedded
- CLI test - publish extracts correct files, respects --force/--dry-run

## User Experience

1. `bitcode init my-app` -> empty project, base auto-loaded from binary
2. `bitcode dev` -> app runs with auth, users, roles out-of-the-box
3. Want custom user model -> `bitcode publish base models/user.json` -> edit
4. Want full control -> `bitcode publish base` -> all files in project

## Diagram

```
+---------------------------+
|      bitcode binary       |
|  +---------------------+  |
|  | embedded/modules/   |  |
|  |   base/             |  |
|  |     module.json     |  |
|  |     models/         |  |
|  |     apis/           |  |
|  |     views/          |  |
|  |     templates/      |  |
|  |     processes/      |  |
|  |     data/           |  |
|  +---------------------+  |
+---------------------------+
            |
       LoadModules()
            |
   +--------v---------+
   |    LayeredFS      |
   |                   |
   | 1. Project FS     | <- ./modules/base/ (user overrides)
   | 2. Global FS      | <- ~/.bitcode/modules/base/
   | 3. Embedded FS    | <- binary (always available)
   |                   |
   | Per-file merge:   |
   | First match wins  |
   +-------------------+
            |
   +--------v---------+
   | bitcode publish   |
   |                   |
   | Extract from      |
   | embedded -> project|
   |                   |
   | --models, --views |
   | --force, --dry-run|
   +-------------------+
```
