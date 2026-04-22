# Embedded Module + Publish System — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Embed the base module into the engine binary via Go `embed.FS`, add 3-layer module resolution (project > global > embedded), and implement `bitcode publish` CLI command for extracting embedded files to project.

**Architecture:** New `ModuleFS` interface abstracts file access. `LayeredFS` composes DiskFS + EmbedFS with priority-based per-file resolution. The module loader is refactored to use `ModuleFS` instead of direct `os.ReadFile`. CLI gets a `publish` subcommand that extracts from embedded FS to project directory.

**Tech Stack:** Go 1.21+, `embed.FS`, `io/fs`, Cobra CLI, standard `testing`

**Design Doc:** `docs/plans/2026-04-22-embedded-module-publish-design.md`

---

## Task 1: Create `ModuleFS` Interface + `DiskFS` Implementation

**Files:**
- Create: `engine/internal/infrastructure/module/fs.go`
- Create: `engine/internal/infrastructure/module/fs_test.go`

**Step 1: Write failing tests for DiskFS**

```go
// fs_test.go
package module

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiskFS_ReadFile(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "models"), 0755)
	os.WriteFile(filepath.Join(dir, "models", "user.json"), []byte(`{"name":"user"}`), 0644)

	fs := NewDiskFS(dir)
	data, err := fs.ReadFile("models/user.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != `{"name":"user"}` {
		t.Errorf("expected user json, got %s", string(data))
	}
}

func TestDiskFS_ReadFile_NotFound(t *testing.T) {
	dir := t.TempDir()
	fs := NewDiskFS(dir)
	_, err := fs.ReadFile("nonexistent.json")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestDiskFS_Glob(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "models"), 0755)
	os.WriteFile(filepath.Join(dir, "models", "user.json"), []byte(`{}`), 0644)
	os.WriteFile(filepath.Join(dir, "models", "role.json"), []byte(`{}`), 0644)

	fs := NewDiskFS(dir)
	matches, err := fs.Glob("models/*.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(matches) != 2 {
		t.Errorf("expected 2 matches, got %d", len(matches))
	}
}

func TestDiskFS_Exists(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "module.json"), []byte(`{}`), 0644)

	fs := NewDiskFS(dir)
	if !fs.Exists("module.json") {
		t.Error("module.json should exist")
	}
	if fs.Exists("nonexistent.json") {
		t.Error("nonexistent.json should not exist")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/infrastructure/module/ -run TestDiskFS -v`
Expected: FAIL — `NewDiskFS` undefined

**Step 3: Implement `ModuleFS` interface and `DiskFS`**

```go
// fs.go
package module

import (
	"io/fs"
	"os"
	"path/filepath"
)

type ModuleFS interface {
	ReadFile(path string) ([]byte, error)
	Glob(pattern string) ([]string, error)
	Exists(path string) bool
	ListDir(path string) ([]string, error)
}

type DiskFS struct {
	root string
}

func NewDiskFS(root string) *DiskFS {
	return &DiskFS{root: root}
}

func (d *DiskFS) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(filepath.Join(d.root, path))
}

func (d *DiskFS) Glob(pattern string) ([]string, error) {
	fullPattern := filepath.Join(d.root, pattern)
	matches, err := filepath.Glob(fullPattern)
	if err != nil {
		return nil, err
	}
	rel := make([]string, len(matches))
	for i, m := range matches {
		r, _ := filepath.Rel(d.root, m)
		rel[i] = filepath.ToSlash(r)
	}
	return rel, nil
}

func (d *DiskFS) Exists(path string) bool {
	_, err := os.Stat(filepath.Join(d.root, path))
	return err == nil
}

func (d *DiskFS) ListDir(path string) ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(d.root, path))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fs.ErrNotExist
		}
		return nil, err
	}
	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.Name()
	}
	return names, nil
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/infrastructure/module/ -run TestDiskFS -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/infrastructure/module/fs.go internal/infrastructure/module/fs_test.go
git commit -m "feat(module): add ModuleFS interface and DiskFS implementation"
```

---

## Task 2: Implement `EmbedFS` Adapter

**Files:**
- Modify: `engine/internal/infrastructure/module/fs.go`
- Modify: `engine/internal/infrastructure/module/fs_test.go`

**Step 1: Write failing tests for EmbedFS**

```go
// Add to fs_test.go
import (
	"embed"
	"testing/fstest"
)

func TestEmbedFS_ReadFile(t *testing.T) {
	memFS := fstest.MapFS{
		"base/module.json":       {Data: []byte(`{"name":"base"}`)},
		"base/models/user.json":  {Data: []byte(`{"name":"user"}`)},
		"base/models/role.json":  {Data: []byte(`{"name":"role"}`)},
	}

	efs := NewEmbedFSFromFS(memFS, "base")
	data, err := efs.ReadFile("module.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != `{"name":"base"}` {
		t.Errorf("expected base json, got %s", string(data))
	}
}

func TestEmbedFS_Glob(t *testing.T) {
	memFS := fstest.MapFS{
		"base/models/user.json": {Data: []byte(`{}`)},
		"base/models/role.json": {Data: []byte(`{}`)},
		"base/apis/user_api.json": {Data: []byte(`{}`)},
	}

	efs := NewEmbedFSFromFS(memFS, "base")
	matches, err := efs.Glob("models/*.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(matches) != 2 {
		t.Errorf("expected 2 matches, got %d", len(matches))
	}
}

func TestEmbedFS_Exists(t *testing.T) {
	memFS := fstest.MapFS{
		"base/module.json": {Data: []byte(`{}`)},
	}

	efs := NewEmbedFSFromFS(memFS, "base")
	if !efs.Exists("module.json") {
		t.Error("module.json should exist")
	}
	if efs.Exists("nonexistent.json") {
		t.Error("nonexistent.json should not exist")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/infrastructure/module/ -run TestEmbedFS -v`
Expected: FAIL — `NewEmbedFSFromFS` undefined

**Step 3: Implement EmbedFS**

```go
// Add to fs.go
import "io/fs"

type EmbedFS struct {
	fsys   fs.FS
	prefix string
}

func NewEmbedFSFromFS(fsys fs.FS, prefix string) *EmbedFS {
	return &EmbedFS{fsys: fsys, prefix: prefix}
}

func NewEmbedFSFromEmbed(efs embed.FS, prefix string) *EmbedFS {
	return &EmbedFS{fsys: efs, prefix: prefix}
}

func (e *EmbedFS) ReadFile(path string) ([]byte, error) {
	return fs.ReadFile(e.fsys, e.prefix+"/"+path)
}

func (e *EmbedFS) Glob(pattern string) ([]string, error) {
	matches, err := fs.Glob(e.fsys, e.prefix+"/"+pattern)
	if err != nil {
		return nil, err
	}
	rel := make([]string, len(matches))
	prefixLen := len(e.prefix) + 1
	for i, m := range matches {
		rel[i] = m[prefixLen:]
	}
	return rel, nil
}

func (e *EmbedFS) Exists(path string) bool {
	_, err := fs.Stat(e.fsys, e.prefix+"/"+path)
	return err == nil
}

func (e *EmbedFS) ListDir(path string) ([]string, error) {
	fullPath := e.prefix
	if path != "" && path != "." {
		fullPath = e.prefix + "/" + path
	}
	entries, err := fs.ReadDir(e.fsys, fullPath)
	if err != nil {
		return nil, err
	}
	names := make([]string, len(entries))
	for i, entry := range entries {
		names[i] = entry.Name()
	}
	return names, nil
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/infrastructure/module/ -run TestEmbedFS -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/infrastructure/module/fs.go internal/infrastructure/module/fs_test.go
git commit -m "feat(module): add EmbedFS adapter for embed.FS"
```

---

## Task 3: Implement `LayeredFS`

**Files:**
- Modify: `engine/internal/infrastructure/module/fs.go`
- Modify: `engine/internal/infrastructure/module/fs_test.go`

**Step 1: Write failing tests for LayeredFS**

```go
// Add to fs_test.go

func TestLayeredFS_ReadFile_ProjectOverridesEmbedded(t *testing.T) {
	projectDir := t.TempDir()
	os.MkdirAll(filepath.Join(projectDir, "models"), 0755)
	os.WriteFile(filepath.Join(projectDir, "models", "user.json"), []byte(`{"name":"custom_user"}`), 0644)

	embedFS := fstest.MapFS{
		"base/models/user.json": {Data: []byte(`{"name":"default_user"}`)},
		"base/models/role.json": {Data: []byte(`{"name":"default_role"}`)},
	}

	layered := NewLayeredFS(
		NewDiskFS(projectDir),
		NewEmbedFSFromFS(embedFS, "base"),
	)

	// Project override should win
	data, err := layered.ReadFile("models/user.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != `{"name":"custom_user"}` {
		t.Errorf("expected custom_user, got %s", string(data))
	}

	// Embedded fallback for non-overridden file
	data, err = layered.ReadFile("models/role.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != `{"name":"default_role"}` {
		t.Errorf("expected default_role, got %s", string(data))
	}
}

func TestLayeredFS_Glob_MergesAllLayers(t *testing.T) {
	projectDir := t.TempDir()
	os.MkdirAll(filepath.Join(projectDir, "models"), 0755)
	os.WriteFile(filepath.Join(projectDir, "models", "custom.json"), []byte(`{}`), 0644)

	embedFS := fstest.MapFS{
		"base/models/user.json": {Data: []byte(`{}`)},
		"base/models/role.json": {Data: []byte(`{}`)},
	}

	layered := NewLayeredFS(
		NewDiskFS(projectDir),
		NewEmbedFSFromFS(embedFS, "base"),
	)

	matches, err := layered.Glob("models/*.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should have 3: custom.json (project) + user.json + role.json (embedded)
	if len(matches) != 3 {
		t.Errorf("expected 3 matches, got %d: %v", len(matches), matches)
	}
}

func TestLayeredFS_Glob_DeduplicatesOverrides(t *testing.T) {
	projectDir := t.TempDir()
	os.MkdirAll(filepath.Join(projectDir, "models"), 0755)
	os.WriteFile(filepath.Join(projectDir, "models", "user.json"), []byte(`{}`), 0644)

	embedFS := fstest.MapFS{
		"base/models/user.json": {Data: []byte(`{}`)},
	}

	layered := NewLayeredFS(
		NewDiskFS(projectDir),
		NewEmbedFSFromFS(embedFS, "base"),
	)

	matches, err := layered.Glob("models/*.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// user.json exists in both layers, should appear only once
	if len(matches) != 1 {
		t.Errorf("expected 1 match (deduplicated), got %d: %v", len(matches), matches)
	}
}

func TestLayeredFS_ReadFile_AllLayersMissing(t *testing.T) {
	projectDir := t.TempDir()
	embedFS := fstest.MapFS{}

	layered := NewLayeredFS(
		NewDiskFS(projectDir),
		NewEmbedFSFromFS(embedFS, "base"),
	)

	_, err := layered.ReadFile("nonexistent.json")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/infrastructure/module/ -run TestLayeredFS -v`
Expected: FAIL — `NewLayeredFS` undefined

**Step 3: Implement LayeredFS**

```go
// Add to fs.go

type LayeredFS struct {
	layers []ModuleFS
}

func NewLayeredFS(layers ...ModuleFS) *LayeredFS {
	return &LayeredFS{layers: layers}
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

func (l *LayeredFS) Glob(pattern string) ([]string, error) {
	seen := make(map[string]bool)
	var result []string
	for _, layer := range l.layers {
		matches, err := layer.Glob(pattern)
		if err != nil {
			continue
		}
		for _, m := range matches {
			normalized := filepath.ToSlash(m)
			if !seen[normalized] {
				seen[normalized] = true
				result = append(result, normalized)
			}
		}
	}
	return result, nil
}

func (l *LayeredFS) Exists(path string) bool {
	for _, layer := range l.layers {
		if layer.Exists(path) {
			return true
		}
	}
	return false
}

func (l *LayeredFS) ListDir(path string) ([]string, error) {
	seen := make(map[string]bool)
	var result []string
	for _, layer := range l.layers {
		names, err := layer.ListDir(path)
		if err != nil {
			continue
		}
		for _, n := range names {
			if !seen[n] {
				seen[n] = true
				result = append(result, n)
			}
		}
	}
	if len(result) == 0 {
		return nil, fs.ErrNotExist
	}
	return result, nil
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/infrastructure/module/ -run TestLayeredFS -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/infrastructure/module/fs.go internal/infrastructure/module/fs_test.go
git commit -m "feat(module): add LayeredFS for multi-layer module resolution"
```

---

## Task 4: Move Base Module to `engine/embedded/` + Embed Directive

**Files:**
- Create: `engine/embedded/modules/base/` (move from `engine/modules/base/`)
- Create: `engine/embedded/embed.go`
- Delete: `engine/modules/base/` (moved)

**Step 1: Move base module files**

```bash
mkdir -p engine/embedded/modules
mv engine/modules/base engine/embedded/modules/base
```

**Step 2: Create embed.go**

```go
// engine/embedded/embed.go
package embedded

import "embed"

//go:embed modules/*
var ModulesFS embed.FS
```

Note: Go embed with `modules/*` may not include nested directories. If needed, use `all:modules` instead:

```go
//go:embed all:modules
var ModulesFS embed.FS
```

**Step 3: Write a quick test to verify embedding works**

```go
// engine/embedded/embed_test.go
package embedded

import "testing"

func TestModulesFS_ContainsBase(t *testing.T) {
	data, err := ModulesFS.ReadFile("modules/base/module.json")
	if err != nil {
		t.Fatalf("failed to read embedded module.json: %v", err)
	}
	if len(data) == 0 {
		t.Error("embedded module.json is empty")
	}
}

func TestModulesFS_ContainsModels(t *testing.T) {
	entries, err := ModulesFS.ReadDir("modules/base/models")
	if err != nil {
		t.Fatalf("failed to read embedded models dir: %v", err)
	}
	if len(entries) == 0 {
		t.Error("embedded models dir is empty")
	}

	found := false
	for _, e := range entries {
		if e.Name() == "user.json" {
			found = true
			break
		}
	}
	if !found {
		t.Error("user.json not found in embedded models")
	}
}

func TestModulesFS_ContainsTemplates(t *testing.T) {
	_, err := ModulesFS.ReadFile("modules/base/templates/layout.html")
	if err != nil {
		t.Fatalf("failed to read embedded layout.html: %v", err)
	}
}
```

**Step 4: Run tests**

Run: `go test ./embedded/ -v`
Expected: PASS

**Step 5: Verify build still works**

Run: `go build ./cmd/engine/`
Expected: Build succeeds

**Step 6: Commit**

```bash
git add embedded/ -A
git add modules/ -A
git commit -m "feat(embed): move base module to embedded/ with go:embed directive"
```

---

## Task 5: Refactor `LoadModule` to Use `ModuleFS`

**Files:**
- Modify: `engine/internal/infrastructure/module/loader.go`
- Modify: `engine/internal/infrastructure/module/fs_test.go` (add loader tests)

**Step 1: Write failing test for `LoadModuleFromFS`**

```go
// Add to fs_test.go

func TestLoadModuleFromFS(t *testing.T) {
	memFS := fstest.MapFS{
		"module.json": {Data: []byte(`{
			"name": "test",
			"version": "1.0.0",
			"label": "Test",
			"depends": [],
			"models": ["models/*.json"],
			"apis": ["apis/*.json"]
		}`)},
		"models/item.json": {Data: []byte(`{
			"name": "item",
			"fields": [
				{"name": "id", "type": "uuid", "primary": true},
				{"name": "title", "type": "string", "required": true}
			]
		}`)},
		"apis/item_api.json": {Data: []byte(`{
			"name": "item_api",
			"model": "item",
			"base_path": "/api/items",
			"auto_crud": true
		}`)},
	}

	efs := NewEmbedFSFromFS(memFS, "")
	loaded, err := LoadModuleFromFS(efs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loaded.Definition.Name != "test" {
		t.Errorf("expected test, got %s", loaded.Definition.Name)
	}
	if len(loaded.Models) != 1 {
		t.Errorf("expected 1 model, got %d", len(loaded.Models))
	}
	if len(loaded.APIs) != 1 {
		t.Errorf("expected 1 API, got %d", len(loaded.APIs))
	}
}
```

Note: The `memFS` prefix for `NewEmbedFSFromFS` should be `""` (empty) since files are at root level in the test MapFS. Adjust the `EmbedFS` implementation to handle empty prefix.

**Step 2: Run test to verify it fails**

Run: `go test ./internal/infrastructure/module/ -run TestLoadModuleFromFS -v`
Expected: FAIL — `LoadModuleFromFS` undefined

**Step 3: Implement `LoadModuleFromFS`**

Add to `loader.go`:

```go
func LoadModuleFromFS(mfs ModuleFS) (*LoadedModule, error) {
	moduleData, err := mfs.ReadFile("module.json")
	if err != nil {
		return nil, fmt.Errorf("failed to read module.json: %w", err)
	}

	modDef, err := parser.ParseModule(moduleData)
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
```

Keep the original `LoadModule(modulePath string)` as a backward-compatible wrapper:

```go
func LoadModule(modulePath string) (*LoadedModule, error) {
	mfs := NewDiskFS(modulePath)
	loaded, err := LoadModuleFromFS(mfs)
	if err != nil {
		return nil, err
	}
	loaded.Path = modulePath
	return loaded, nil
}
```

**Step 4: Run all module tests**

Run: `go test ./internal/infrastructure/module/ -v`
Expected: ALL PASS (new + existing tests)

**Step 5: Commit**

```bash
git add internal/infrastructure/module/loader.go internal/infrastructure/module/fs_test.go
git commit -m "feat(module): add LoadModuleFromFS using ModuleFS abstraction"
```

---

## Task 6: Add `DiscoverModules` to `LayeredFS`

**Files:**
- Modify: `engine/internal/infrastructure/module/fs.go`
- Modify: `engine/internal/infrastructure/module/fs_test.go`

**Step 1: Write failing test**

```go
func TestLayeredFS_DiscoverModules(t *testing.T) {
	projectDir := t.TempDir()
	os.MkdirAll(filepath.Join(projectDir, "crm"), 0755)
	os.WriteFile(filepath.Join(projectDir, "crm", "module.json"), []byte(`{"name":"crm"}`), 0644)

	embedFS := fstest.MapFS{
		"base/module.json": {Data: []byte(`{"name":"base"}`)},
	}

	layered := NewLayeredFS(
		NewDiskFS(projectDir),
		NewEmbedFSFromFS(embedFS, ""),
	)

	modules := layered.DiscoverModules()
	if len(modules) != 2 {
		t.Fatalf("expected 2 modules, got %d: %v", len(modules), modules)
	}

	found := map[string]bool{}
	for _, m := range modules {
		found[m] = true
	}
	if !found["base"] {
		t.Error("base module not discovered")
	}
	if !found["crm"] {
		t.Error("crm module not discovered")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/infrastructure/module/ -run TestLayeredFS_DiscoverModules -v`
Expected: FAIL

**Step 3: Implement DiscoverModules**

```go
func (l *LayeredFS) DiscoverModules() []string {
	seen := make(map[string]bool)
	var result []string
	for _, layer := range l.layers {
		names, err := layer.ListDir(".")
		if err != nil {
			continue
		}
		for _, name := range names {
			if seen[name] {
				continue
			}
			if layer.Exists(name + "/module.json") {
				seen[name] = true
				result = append(result, name)
			}
		}
	}
	return result
}

func (l *LayeredFS) SubFS(moduleName string) *LayeredFS {
	var subLayers []ModuleFS
	for _, layer := range l.layers {
		subLayers = append(subLayers, NewSubFS(layer, moduleName))
	}
	return NewLayeredFS(subLayers...)
}
```

Also add `SubFS` wrapper:

```go
type SubFS struct {
	parent ModuleFS
	prefix string
}

func NewSubFS(parent ModuleFS, prefix string) *SubFS {
	return &SubFS{parent: parent, prefix: prefix}
}

func (s *SubFS) ReadFile(path string) ([]byte, error) {
	return s.parent.ReadFile(s.prefix + "/" + path)
}

func (s *SubFS) Glob(pattern string) ([]string, error) {
	return s.parent.Glob(s.prefix + "/" + pattern)
}

func (s *SubFS) Exists(path string) bool {
	return s.parent.Exists(s.prefix + "/" + path)
}

func (s *SubFS) ListDir(path string) ([]string, error) {
	fullPath := s.prefix
	if path != "" && path != "." {
		fullPath = s.prefix + "/" + path
	}
	return s.parent.ListDir(fullPath)
}
```

**Step 4: Run tests**

Run: `go test ./internal/infrastructure/module/ -run TestLayeredFS -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/infrastructure/module/fs.go internal/infrastructure/module/fs_test.go
git commit -m "feat(module): add DiscoverModules and SubFS to LayeredFS"
```

---

## Task 7: Refactor `app.go LoadModules()` to Use LayeredFS

**Files:**
- Modify: `engine/internal/app.go`
- Modify: `engine/internal/config.go`

**Step 1: Add `GlobalModuleDir` to config**

In `config.go`, add default and env binding:

```go
v.SetDefault("global_module_dir", "")
v.BindEnv("global_module_dir", "GLOBAL_MODULE_DIR")
```

In `AppConfig` struct:

```go
type AppConfig struct {
	// ... existing fields
	GlobalModuleDir string
}
```

In config loading:

```go
cfg := AppConfig{
	// ... existing
	GlobalModuleDir: v.GetString("global_module_dir"),
}
```

Default `GlobalModuleDir`: if empty, resolve to `~/.bitcode/modules` at runtime.

**Step 2: Refactor `LoadModules()` in `app.go`**

Replace the current `LoadModules()` implementation. The current code:
- Scans `MODULE_DIR/*/module.json` via `filepath.Glob`
- Loads each module via `module.LoadModule(modPath)`

New code:

```go
func (a *App) LoadModules() error {
	projectFS := module.NewDiskFS(a.Config.ModuleDir)

	var globalFS module.ModuleFS
	globalDir := a.Config.GlobalModuleDir
	if globalDir == "" {
		home, _ := os.UserHomeDir()
		globalDir = filepath.Join(home, ".bitcode", "modules")
	}
	globalFS = module.NewDiskFS(globalDir)

	embedFS := module.NewEmbedFSFromEmbed(embedded.ModulesFS, "modules")

	layered := module.NewLayeredFS(projectFS, globalFS, embedFS)

	moduleNames := layered.DiscoverModules()

	allModules := make(map[string]*parser.ModuleDefinition)
	for _, name := range moduleNames {
		modFS := layered.SubFS(name)
		data, err := modFS.ReadFile("module.json")
		if err != nil {
			log.Printf("[WARN] skipping module %s: %v", name, err)
			continue
		}
		modDef, err := parser.ParseModule(data)
		if err != nil {
			log.Printf("[WARN] skipping invalid module %s: %v", name, err)
			continue
		}
		allModules[modDef.Name] = modDef
	}

	installOrder, err := module.ResolveDependencies(allModules, findRootModules(allModules)...)
	if err != nil {
		return fmt.Errorf("dependency resolution failed: %w", err)
	}

	for _, modName := range installOrder {
		modFS := layered.SubFS(modName)
		if err := a.installModuleFromFS(modName, modFS); err != nil {
			return fmt.Errorf("failed to install module %s: %w", modName, err)
		}
		log.Printf("[MODULE] installed: %s", modName)
	}

	a.processViewRegistrations()

	a.ViewRenderer.SetViewResolver(func(name string) *parser.ViewDefinition {
		for _, entry := range a.viewDefs {
			if entry.Def.Name == name {
				return entry.Def
			}
		}
		return nil
	})

	return nil
}
```

**Step 3: Create `installModuleFromFS`**

This mirrors the existing `installModule(modPath string)` but uses `ModuleFS`. The tricky parts are:
- Template loading (currently uses `filepath.Join(modPath, "templates")`)
- View loading (currently uses `filepath.Walk`)
- i18n, processes, workflows (currently use modPath)

For this task, implement `installModuleFromFS` that delegates to `LoadModuleFromFS` for models/APIs, but still needs a disk path for templates/views/processes. Strategy: if the module has project-level files, use that path. Otherwise, extract embedded files to a temp dir.

**Alternative (simpler)**: Keep `installModule(modPath)` for project modules. For embedded-only modules, extract to a temp dir first, then call `installModule`. This minimizes changes to the template/view/process loading code.

```go
func (a *App) installModuleFromFS(modName string, mfs *module.LayeredFS) error {
	// Try to get a disk path (project or global)
	diskPath := a.resolveModuleDiskPath(modName)
	if diskPath != "" {
		return a.installModule(diskPath)
	}

	// Embedded-only: extract to temp dir
	tmpDir, err := module.ExtractModuleFS(mfs, modName)
	if err != nil {
		return fmt.Errorf("failed to extract embedded module %s: %w", modName, err)
	}
	return a.installModule(tmpDir)
}

func (a *App) resolveModuleDiskPath(modName string) string {
	projectPath := filepath.Join(a.Config.ModuleDir, modName)
	if _, err := os.Stat(filepath.Join(projectPath, "module.json")); err == nil {
		return projectPath
	}

	globalDir := a.Config.GlobalModuleDir
	if globalDir == "" {
		home, _ := os.UserHomeDir()
		globalDir = filepath.Join(home, ".bitcode", "modules")
	}
	globalPath := filepath.Join(globalDir, modName)
	if _, err := os.Stat(filepath.Join(globalPath, "module.json")); err == nil {
		return globalPath
	}

	return ""
}
```

Add `ExtractModuleFS` helper in `module/fs.go`:

```go
func ExtractModuleFS(mfs ModuleFS, modName string) (string, error) {
	tmpDir, err := os.MkdirTemp("", "bitcode-module-"+modName+"-")
	if err != nil {
		return "", err
	}
	// Walk all files and extract
	return tmpDir, extractRecursive(mfs, ".", tmpDir)
}

func extractRecursive(mfs ModuleFS, dir string, targetDir string) error {
	entries, err := mfs.ListDir(dir)
	if err != nil {
		return nil // empty dir is fine
	}
	for _, entry := range entries {
		srcPath := entry
		if dir != "." {
			srcPath = dir + "/" + entry
		}
		data, err := mfs.ReadFile(srcPath)
		if err != nil {
			// Might be a directory, try recursing
			subTarget := filepath.Join(targetDir, entry)
			os.MkdirAll(subTarget, 0755)
			extractRecursive(mfs, srcPath, subTarget)
			continue
		}
		targetPath := filepath.Join(targetDir, filepath.FromSlash(srcPath))
		os.MkdirAll(filepath.Dir(targetPath), 0755)
		os.WriteFile(targetPath, data, 0644)
	}
	return nil
}
```

**Step 4: Add import for embedded package**

In `app.go`:
```go
import "github.com/bitcode-engine/engine/embedded"
```

**Step 5: Verify build**

Run: `go build ./cmd/engine/`
Expected: Build succeeds

**Step 6: Run existing tests**

Run: `go test ./... -count=1`
Expected: ALL PASS (93 tests)

**Step 7: Commit**

```bash
git add internal/app.go internal/config.go internal/infrastructure/module/fs.go
git commit -m "feat(module): refactor LoadModules to use 3-layer resolution with embedded base"
```

---

## Task 8: Implement `bitcode publish` CLI Command

**Files:**
- Create: `engine/cmd/bitcode/publish.go`
- Modify: `engine/cmd/bitcode/main.go`

**Step 1: Create publish.go**

```go
// engine/cmd/bitcode/publish.go
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bitcode-engine/engine/embedded"
	"github.com/bitcode-engine/engine/internal/infrastructure/module"
	"github.com/spf13/cobra"
)

func publishCmd() *cobra.Command {
	var force bool
	var dryRun bool
	var models, apis, views, templates, processes, data bool

	cmd := &cobra.Command{
		Use:   "publish [module] [file]",
		Short: "Publish embedded module files to project for customization",
		Long: `Extract embedded module files to your project's modules directory.
Similar to Laravel's artisan vendor:publish.

Examples:
  bitcode publish base                    # Publish entire base module
  bitcode publish base --models           # Publish only models
  bitcode publish base models/user.json   # Publish single file
  bitcode publish --list                  # List publishable modules`,
		Args: cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			listFlag, _ := cmd.Flags().GetBool("list")
			if listFlag {
				return listPublishable()
			}

			if len(args) == 0 {
				return fmt.Errorf("module name required. Use --list to see available modules")
			}

			moduleName := args[0]
			moduleDir := envOrDefault("MODULE_DIR", "modules")
			targetDir := filepath.Join(moduleDir, moduleName)

			embedFS := module.NewEmbedFSFromEmbed(embedded.ModulesFS, "modules/"+moduleName)

			if !embedFS.Exists("module.json") {
				return fmt.Errorf("module %q not found in embedded modules", moduleName)
			}

			// Determine which files to publish
			var specificFile string
			if len(args) == 2 {
				specificFile = args[1]
			}

			resourceFilter := buildResourceFilter(models, apis, views, templates, processes, data)

			published := 0
			skipped := 0

			err := publishFiles(embedFS, targetDir, specificFile, resourceFilter, force, dryRun, &published, &skipped)
			if err != nil {
				return err
			}

			if dryRun {
				fmt.Printf("[DRY RUN] Would publish %d files to %s (%d skipped)\n", published, targetDir, skipped)
			} else {
				fmt.Printf("Published %d files to %s (%d skipped)\n", published, targetDir, skipped)
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing files")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview without writing")
	cmd.Flags().Bool("list", false, "List publishable modules")
	cmd.Flags().BoolVar(&models, "models", false, "Publish only models")
	cmd.Flags().BoolVar(&apis, "apis", false, "Publish only APIs")
	cmd.Flags().BoolVar(&views, "views", false, "Publish only views")
	cmd.Flags().BoolVar(&templates, "templates", false, "Publish only templates")
	cmd.Flags().BoolVar(&processes, "processes", false, "Publish only processes")
	cmd.Flags().BoolVar(&data, "data", false, "Publish only data")

	return cmd
}

func buildResourceFilter(models, apis, views, templates, processes, data bool) []string {
	if !models && !apis && !views && !templates && !processes && !data {
		return nil // no filter = publish all
	}
	var filters []string
	if models {
		filters = append(filters, "models/")
	}
	if apis {
		filters = append(filters, "apis/")
	}
	if views {
		filters = append(filters, "views/")
	}
	if templates {
		filters = append(filters, "templates/")
	}
	if processes {
		filters = append(filters, "processes/")
	}
	if data {
		filters = append(filters, "data/")
	}
	return filters
}

func publishFiles(src module.ModuleFS, targetDir, specificFile string, resourceFilter []string, force, dryRun bool, published, skipped *int) error {
	if specificFile != "" {
		return publishSingleFile(src, targetDir, specificFile, force, dryRun, published, skipped)
	}
	return publishRecursive(src, ".", targetDir, resourceFilter, force, dryRun, published, skipped)
}

func publishSingleFile(src module.ModuleFS, targetDir, file string, force, dryRun bool, published, skipped *int) error {
	data, err := src.ReadFile(file)
	if err != nil {
		return fmt.Errorf("file %q not found in embedded module", file)
	}

	targetPath := filepath.Join(targetDir, filepath.FromSlash(file))
	if !force {
		if _, err := os.Stat(targetPath); err == nil {
			fmt.Printf("  SKIP %s (exists, use --force to overwrite)\n", file)
			*skipped++
			return nil
		}
	}

	if dryRun {
		fmt.Printf("  PUBLISH %s\n", file)
		*published++
		return nil
	}

	os.MkdirAll(filepath.Dir(targetPath), 0755)
	if err := os.WriteFile(targetPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", targetPath, err)
	}
	fmt.Printf("  PUBLISH %s\n", file)
	*published++
	return nil
}

func publishRecursive(src module.ModuleFS, dir, targetDir string, resourceFilter []string, force, dryRun bool, published, skipped *int) error {
	entries, err := src.ListDir(dir)
	if err != nil {
		return nil
	}

	for _, entry := range entries {
		path := entry
		if dir != "." {
			path = dir + "/" + entry
		}

		if len(resourceFilter) > 0 {
			matched := false
			for _, f := range resourceFilter {
				if strings.HasPrefix(path, f) || strings.HasPrefix(path+"/", f) {
					matched = true
					break
				}
			}
			if !matched && path != "module.json" {
				continue
			}
		}

		data, err := src.ReadFile(path)
		if err != nil {
			// Directory — recurse
			publishRecursive(src, path, targetDir, resourceFilter, force, dryRun, published, skipped)
			continue
		}

		targetPath := filepath.Join(targetDir, filepath.FromSlash(path))
		if !force {
			if _, err := os.Stat(targetPath); err == nil {
				fmt.Printf("  SKIP %s (exists)\n", path)
				*skipped++
				continue
			}
		}

		if dryRun {
			fmt.Printf("  PUBLISH %s\n", path)
			*published++
			continue
		}

		os.MkdirAll(filepath.Dir(targetPath), 0755)
		if err := os.WriteFile(targetPath, data, 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", targetPath, err)
		}
		fmt.Printf("  PUBLISH %s\n", path)
		*published++
	}
	return nil
}

func listPublishable() error {
	embedFS := module.NewEmbedFSFromEmbed(embedded.ModulesFS, "modules")
	entries, err := embedFS.ListDir(".")
	if err != nil {
		fmt.Println("No embedded modules found.")
		return nil
	}

	fmt.Println("Publishable modules:")
	for _, entry := range entries {
		if embedFS.Exists(entry + "/module.json") {
			data, _ := embedFS.ReadFile(entry + "/module.json")
			fmt.Printf("  %s (%d bytes)\n", entry, len(data))
		}
	}
	return nil
}
```

**Step 2: Wire publish command in main.go**

In `main.go`, add to root command:

```go
root.AddCommand(publishCmd())
```

**Step 3: Build and test manually**

```bash
go build -o bin/bitcode cmd/bitcode/main.go
./bin/bitcode publish --list
./bin/bitcode publish base --dry-run
./bin/bitcode publish base models/user.json --dry-run
```

**Step 4: Commit**

```bash
git add cmd/bitcode/publish.go cmd/bitcode/main.go
git commit -m "feat(cli): add bitcode publish command for extracting embedded modules"
```

---

## Task 9: Integration Test — Engine Boots with Embedded Base Only

**Files:**
- Create: `engine/internal/infrastructure/module/integration_test.go`

**Step 1: Write integration test**

```go
// integration_test.go
package module

import (
	"testing"
	"testing/fstest"
)

func TestIntegration_DiscoverAndLoadEmbeddedModule(t *testing.T) {
	embedFS := fstest.MapFS{
		"base/module.json": {Data: []byte(`{
			"name": "base",
			"version": "1.0.0",
			"label": "Base",
			"depends": [],
			"models": ["models/*.json"],
			"apis": ["apis/*.json"]
		}`)},
		"base/models/user.json": {Data: []byte(`{
			"name": "user",
			"fields": [
				{"name": "id", "type": "uuid", "primary": true},
				{"name": "username", "type": "string", "required": true}
			]
		}`)},
		"base/apis/user_api.json": {Data: []byte(`{
			"name": "user_api",
			"model": "user",
			"base_path": "/api/users",
			"auto_crud": true
		}`)},
	}

	// No project modules, no global modules
	projectDir := t.TempDir()
	layered := NewLayeredFS(
		NewDiskFS(projectDir),
		NewEmbedFSFromFS(embedFS, ""),
	)

	modules := layered.DiscoverModules()
	if len(modules) != 1 {
		t.Fatalf("expected 1 module, got %d: %v", len(modules), modules)
	}
	if modules[0] != "base" {
		t.Errorf("expected base, got %s", modules[0])
	}

	modFS := layered.SubFS("base")
	loaded, err := LoadModuleFromFS(modFS)
	if err != nil {
		t.Fatalf("failed to load module: %v", err)
	}
	if loaded.Definition.Name != "base" {
		t.Errorf("expected base, got %s", loaded.Definition.Name)
	}
	if len(loaded.Models) != 1 {
		t.Errorf("expected 1 model, got %d", len(loaded.Models))
	}
}

func TestIntegration_ProjectOverridesEmbedded(t *testing.T) {
	embedFS := fstest.MapFS{
		"base/module.json": {Data: []byte(`{
			"name": "base",
			"version": "1.0.0",
			"depends": [],
			"models": ["models/*.json"],
			"apis": []
		}`)},
		"base/models/user.json": {Data: []byte(`{
			"name": "user",
			"fields": [
				{"name": "id", "type": "uuid", "primary": true},
				{"name": "username", "type": "string"}
			]
		}`)},
	}

	projectDir := t.TempDir()
	// Create project override with extra field
	modDir := projectDir + "/base/models"
	os.MkdirAll(modDir, 0755)
	os.WriteFile(projectDir+"/base/module.json", []byte(`{
		"name": "base",
		"version": "1.0.0-custom",
		"depends": [],
		"models": ["models/*.json"],
		"apis": []
	}`), 0644)
	os.WriteFile(modDir+"/user.json", []byte(`{
		"name": "user",
		"fields": [
			{"name": "id", "type": "uuid", "primary": true},
			{"name": "username", "type": "string"},
			{"name": "phone", "type": "string"}
		]
	}`), 0644)

	layered := NewLayeredFS(
		NewDiskFS(projectDir),
		NewEmbedFSFromFS(embedFS, ""),
	)

	modFS := layered.SubFS("base")
	loaded, err := LoadModuleFromFS(modFS)
	if err != nil {
		t.Fatalf("failed to load module: %v", err)
	}

	// Should use project version
	if loaded.Definition.Version != "1.0.0-custom" {
		t.Errorf("expected 1.0.0-custom, got %s", loaded.Definition.Version)
	}

	// Should have 3 fields (project override)
	if len(loaded.Models) != 1 {
		t.Fatalf("expected 1 model, got %d", len(loaded.Models))
	}
	if len(loaded.Models[0].Fields) != 3 {
		t.Errorf("expected 3 fields (project override), got %d", len(loaded.Models[0].Fields))
	}
}
```

**Step 2: Run integration tests**

Run: `go test ./internal/infrastructure/module/ -run TestIntegration -v`
Expected: PASS

**Step 3: Run full test suite**

Run: `go test ./... -count=1`
Expected: ALL PASS

**Step 4: Commit**

```bash
git add internal/infrastructure/module/integration_test.go
git commit -m "test(module): add integration tests for layered module resolution"
```

---

## Task 10: Final Verification + Build

**Step 1: Run full test suite**

Run: `go test ./... -v -count=1`
Expected: ALL PASS

**Step 2: Build both binaries**

```bash
go build -o bin/engine cmd/engine/main.go
go build -o bin/bitcode cmd/bitcode/main.go
```

Expected: Both build successfully

**Step 3: Manual smoke test — engine boots with embedded base**

```bash
# Create empty project
mkdir /tmp/test-embedded && cd /tmp/test-embedded
echo 'name: test' > bitcode.yaml

# Run engine (no modules/ dir — should use embedded base)
/path/to/bin/engine
```

Expected: Engine starts, base module installed from embedded

**Step 4: Manual smoke test — publish command**

```bash
cd /tmp/test-embedded
/path/to/bin/bitcode publish --list
/path/to/bin/bitcode publish base --dry-run
/path/to/bin/bitcode publish base models/user.json
ls modules/base/models/user.json  # should exist
```

**Step 5: Manual smoke test — override works**

Edit `modules/base/models/user.json`, add a field. Restart engine. Verify the custom field appears.

**Step 6: Commit final state**

```bash
git add -A
git commit -m "feat: embedded module system with 3-layer resolution and bitcode publish CLI

- Base module embedded in binary via go:embed
- 3-layer resolution: project > global > embedded (per-file merge)
- bitcode publish command: whole module, per-resource, per-file
- ModuleFS abstraction (DiskFS, EmbedFS, LayeredFS)
- Backward compatible: existing projects unaffected"
```
