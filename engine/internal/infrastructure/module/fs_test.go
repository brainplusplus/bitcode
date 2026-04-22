package module

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func TestDiskFS_ReadFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := []byte("hello world")

	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	diskFS := NewDiskFS(tmpDir)
	got, err := diskFS.ReadFile("test.txt")
	if err != nil {
		t.Errorf("ReadFile() error = %v", err)
	}

	if string(got) != string(content) {
		t.Errorf("ReadFile() = %q, want %q", got, content)
	}
}

func TestDiskFS_ReadFile_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	diskFS := NewDiskFS(tmpDir)

	_, err := diskFS.ReadFile("nonexistent.txt")
	if err == nil {
		t.Error("ReadFile() expected error for nonexistent file, got nil")
	}
}

func TestDiskFS_Glob(t *testing.T) {
	tmpDir := t.TempDir()

	file1 := filepath.Join(tmpDir, "test1.txt")
	file2 := filepath.Join(tmpDir, "test2.txt")

	if err := os.WriteFile(file1, []byte("content1"), 0644); err != nil {
		t.Fatalf("failed to write test file 1: %v", err)
	}
	if err := os.WriteFile(file2, []byte("content2"), 0644); err != nil {
		t.Fatalf("failed to write test file 2: %v", err)
	}

	diskFS := NewDiskFS(tmpDir)
	matches, err := diskFS.Glob("*.txt")
	if err != nil {
		t.Fatalf("Glob() error = %v", err)
	}

	if len(matches) != 2 {
		t.Errorf("Glob() returned %d matches, want 2", len(matches))
	}

	matchMap := make(map[string]bool)
	for _, match := range matches {
		matchMap[match] = true
	}

	if !matchMap["test1.txt"] {
		t.Error("Glob() missing test1.txt")
	}
	if !matchMap["test2.txt"] {
		t.Error("Glob() missing test2.txt")
	}
}

func TestDiskFS_Exists(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "exists.txt")

	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	diskFS := NewDiskFS(tmpDir)

	if !diskFS.Exists("exists.txt") {
		t.Error("Exists() = false for existing file, want true")
	}

	if diskFS.Exists("nonexistent.txt") {
		t.Error("Exists() = true for nonexistent file, want false")
	}
}

func TestDiskFS_ListDir(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "subdir")

	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	file1 := filepath.Join(subDir, "file1.txt")
	file2 := filepath.Join(subDir, "file2.txt")

	if err := os.WriteFile(file1, []byte("content1"), 0644); err != nil {
		t.Fatalf("failed to write file1: %v", err)
	}
	if err := os.WriteFile(file2, []byte("content2"), 0644); err != nil {
		t.Fatalf("failed to write file2: %v", err)
	}

	diskFS := NewDiskFS(tmpDir)
	entries, err := diskFS.ListDir("subdir")
	if err != nil {
		t.Fatalf("ListDir() error = %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("ListDir() returned %d entries, want 2", len(entries))
	}

	entryMap := make(map[string]bool)
	for _, entry := range entries {
		entryMap[entry] = true
	}

	if !entryMap["file1.txt"] {
		t.Error("ListDir() missing file1.txt")
	}
	if !entryMap["file2.txt"] {
		t.Error("ListDir() missing file2.txt")
	}
}

func TestDiskFS_ListDir_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	diskFS := NewDiskFS(tmpDir)

	_, err := diskFS.ListDir("nonexistent")
	if err != fs.ErrNotExist {
		t.Errorf("ListDir() error = %v, want fs.ErrNotExist", err)
	}
}

func TestEmbedFS_ReadFile(t *testing.T) {
	mapFS := fstest.MapFS{
		"base/module.json": &fstest.MapFile{
			Data: []byte(`{"name":"test"}`),
		},
	}

	embedFS := NewEmbedFSFromFS(mapFS, "base")
	got, err := embedFS.ReadFile("module.json")
	if err != nil {
		t.Errorf("ReadFile() error = %v", err)
	}

	expected := `{"name":"test"}`
	if string(got) != expected {
		t.Errorf("ReadFile() = %q, want %q", got, expected)
	}
}

func TestEmbedFS_ReadFile_NotFound(t *testing.T) {
	mapFS := fstest.MapFS{
		"base/module.json": &fstest.MapFile{
			Data: []byte(`{"name":"test"}`),
		},
	}

	embedFS := NewEmbedFSFromFS(mapFS, "base")
	_, err := embedFS.ReadFile("nonexistent.json")
	if err == nil {
		t.Error("ReadFile() expected error for nonexistent file, got nil")
	}
}

func TestEmbedFS_Glob(t *testing.T) {
	mapFS := fstest.MapFS{
		"base/models/user.json": &fstest.MapFile{
			Data: []byte(`{"name":"user"}`),
		},
		"base/models/role.json": &fstest.MapFile{
			Data: []byte(`{"name":"role"}`),
		},
	}

	embedFS := NewEmbedFSFromFS(mapFS, "base")
	matches, err := embedFS.Glob("models/*.json")
	if err != nil {
		t.Fatalf("Glob() error = %v", err)
	}

	if len(matches) != 2 {
		t.Errorf("Glob() returned %d matches, want 2", len(matches))
	}

	matchMap := make(map[string]bool)
	for _, match := range matches {
		matchMap[match] = true
	}

	if !matchMap["models/user.json"] {
		t.Error("Glob() missing models/user.json")
	}
	if !matchMap["models/role.json"] {
		t.Error("Glob() missing models/role.json")
	}
}

func TestEmbedFS_Exists(t *testing.T) {
	mapFS := fstest.MapFS{
		"base/module.json": &fstest.MapFile{
			Data: []byte(`{"name":"test"}`),
		},
	}

	embedFS := NewEmbedFSFromFS(mapFS, "base")

	if !embedFS.Exists("module.json") {
		t.Error("Exists() = false for existing file, want true")
	}

	if embedFS.Exists("nonexistent.json") {
		t.Error("Exists() = true for nonexistent file, want false")
	}
}

func TestEmbedFS_ListDir(t *testing.T) {
	mapFS := fstest.MapFS{
		"base/models/user.json": &fstest.MapFile{
			Data: []byte(`{"name":"user"}`),
		},
		"base/models/role.json": &fstest.MapFile{
			Data: []byte(`{"name":"role"}`),
		},
	}

	embedFS := NewEmbedFSFromFS(mapFS, "base")
	entries, err := embedFS.ListDir("models")
	if err != nil {
		t.Fatalf("ListDir() error = %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("ListDir() returned %d entries, want 2", len(entries))
	}

	entryMap := make(map[string]bool)
	for _, entry := range entries {
		entryMap[entry] = true
	}

	if !entryMap["user.json"] {
		t.Error("ListDir() missing user.json")
	}
	if !entryMap["role.json"] {
		t.Error("ListDir() missing role.json")
	}
}

func TestEmbedFS_EmptyPrefix(t *testing.T) {
	mapFS := fstest.MapFS{
		"module.json": &fstest.MapFile{
			Data: []byte(`{"name":"test"}`),
		},
		"models/user.json": &fstest.MapFile{
			Data: []byte(`{"name":"user"}`),
		},
	}

	embedFS := NewEmbedFSFromFS(mapFS, "")

	got, err := embedFS.ReadFile("module.json")
	if err != nil {
		t.Errorf("ReadFile() error = %v", err)
	}
	if string(got) != `{"name":"test"}` {
		t.Errorf("ReadFile() = %q, want %q", got, `{"name":"test"}`)
	}

	if !embedFS.Exists("module.json") {
		t.Error("Exists() = false for existing file, want true")
	}

	matches, err := embedFS.Glob("models/*.json")
	if err != nil {
		t.Fatalf("Glob() error = %v", err)
	}
	if len(matches) != 1 {
		t.Errorf("Glob() returned %d matches, want 1", len(matches))
	}

	entries, err := embedFS.ListDir("")
	if err != nil {
		t.Fatalf("ListDir() error = %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("ListDir() returned %d entries, want 2", len(entries))
	}
}

func TestLayeredFS_ReadFile_ProjectOverridesEmbedded(t *testing.T) {
	tmpDir := t.TempDir()
	userFile := filepath.Join(tmpDir, "user.json")
	if err := os.WriteFile(userFile, []byte("custom"), 0644); err != nil {
		t.Fatalf("failed to write user.json: %v", err)
	}

	diskFS := NewDiskFS(tmpDir)

	mapFS := fstest.MapFS{
		"user.json": &fstest.MapFile{Data: []byte("default")},
		"role.json": &fstest.MapFile{Data: []byte("default")},
	}
	embedFS := NewEmbedFSFromFS(mapFS, "")

	layered := NewLayeredFS(diskFS, embedFS)

	got, err := layered.ReadFile("user.json")
	if err != nil {
		t.Fatalf("ReadFile(user.json) error = %v", err)
	}
	if string(got) != "custom" {
		t.Errorf("ReadFile(user.json) = %q, want %q", got, "custom")
	}

	got, err = layered.ReadFile("role.json")
	if err != nil {
		t.Fatalf("ReadFile(role.json) error = %v", err)
	}
	if string(got) != "default" {
		t.Errorf("ReadFile(role.json) = %q, want %q", got, "default")
	}
}

func TestLayeredFS_Glob_MergesAllLayers(t *testing.T) {
	tmpDir := t.TempDir()
	customFile := filepath.Join(tmpDir, "custom.json")
	if err := os.WriteFile(customFile, []byte("custom"), 0644); err != nil {
		t.Fatalf("failed to write custom.json: %v", err)
	}

	diskFS := NewDiskFS(tmpDir)

	mapFS := fstest.MapFS{
		"user.json": &fstest.MapFile{Data: []byte("user")},
		"role.json": &fstest.MapFile{Data: []byte("role")},
	}
	embedFS := NewEmbedFSFromFS(mapFS, "")

	layered := NewLayeredFS(diskFS, embedFS)

	matches, err := layered.Glob("*.json")
	if err != nil {
		t.Fatalf("Glob() error = %v", err)
	}

	if len(matches) != 3 {
		t.Errorf("Glob() returned %d matches, want 3", len(matches))
	}

	matchMap := make(map[string]bool)
	for _, match := range matches {
		matchMap[match] = true
	}

	if !matchMap["custom.json"] {
		t.Error("Glob() missing custom.json")
	}
	if !matchMap["user.json"] {
		t.Error("Glob() missing user.json")
	}
	if !matchMap["role.json"] {
		t.Error("Glob() missing role.json")
	}
}

func TestLayeredFS_Glob_DeduplicatesOverrides(t *testing.T) {
	tmpDir := t.TempDir()
	userFile := filepath.Join(tmpDir, "user.json")
	if err := os.WriteFile(userFile, []byte("custom"), 0644); err != nil {
		t.Fatalf("failed to write user.json: %v", err)
	}

	diskFS := NewDiskFS(tmpDir)

	mapFS := fstest.MapFS{
		"user.json": &fstest.MapFile{Data: []byte("default")},
	}
	embedFS := NewEmbedFSFromFS(mapFS, "")

	layered := NewLayeredFS(diskFS, embedFS)

	matches, err := layered.Glob("*.json")
	if err != nil {
		t.Fatalf("Glob() error = %v", err)
	}

	if len(matches) != 1 {
		t.Errorf("Glob() returned %d matches, want 1", len(matches))
	}

	if matches[0] != "user.json" {
		t.Errorf("Glob() = %q, want %q", matches[0], "user.json")
	}
}

func TestLayeredFS_ReadFile_AllLayersMissing(t *testing.T) {
	tmpDir := t.TempDir()
	diskFS := NewDiskFS(tmpDir)

	mapFS := fstest.MapFS{
		"other.json": &fstest.MapFile{Data: []byte("data")},
	}
	embedFS := NewEmbedFSFromFS(mapFS, "")

	layered := NewLayeredFS(diskFS, embedFS)

	_, err := layered.ReadFile("missing.json")
	if err != fs.ErrNotExist {
		t.Errorf("ReadFile() error = %v, want fs.ErrNotExist", err)
	}
}

func TestLayeredFS_Exists(t *testing.T) {
	tmpDir := t.TempDir()
	diskFile := filepath.Join(tmpDir, "disk.json")
	if err := os.WriteFile(diskFile, []byte("disk"), 0644); err != nil {
		t.Fatalf("failed to write disk.json: %v", err)
	}

	diskFS := NewDiskFS(tmpDir)

	mapFS := fstest.MapFS{
		"embed.json": &fstest.MapFile{Data: []byte("embed")},
	}
	embedFS := NewEmbedFSFromFS(mapFS, "")

	layered := NewLayeredFS(diskFS, embedFS)

	if !layered.Exists("disk.json") {
		t.Error("Exists(disk.json) = false, want true")
	}

	if !layered.Exists("embed.json") {
		t.Error("Exists(embed.json) = false, want true")
	}

	if layered.Exists("missing.json") {
		t.Error("Exists(missing.json) = true, want false")
	}
}

func TestLayeredFS_ListDir_MergesLayers(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "models")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}
	diskFile := filepath.Join(subDir, "custom.json")
	if err := os.WriteFile(diskFile, []byte("custom"), 0644); err != nil {
		t.Fatalf("failed to write custom.json: %v", err)
	}

	diskFS := NewDiskFS(tmpDir)

	mapFS := fstest.MapFS{
		"models/user.json": &fstest.MapFile{Data: []byte("user")},
		"models/role.json": &fstest.MapFile{Data: []byte("role")},
	}
	embedFS := NewEmbedFSFromFS(mapFS, "")

	layered := NewLayeredFS(diskFS, embedFS)

	entries, err := layered.ListDir("models")
	if err != nil {
		t.Fatalf("ListDir() error = %v", err)
	}

	if len(entries) != 3 {
		t.Errorf("ListDir() returned %d entries, want 3", len(entries))
	}

	entryMap := make(map[string]bool)
	for _, entry := range entries {
		entryMap[entry] = true
	}

	if !entryMap["custom.json"] {
		t.Error("ListDir() missing custom.json")
	}
	if !entryMap["user.json"] {
		t.Error("ListDir() missing user.json")
	}
	if !entryMap["role.json"] {
		t.Error("ListDir() missing role.json")
	}
}

func TestLoadModuleFromFS(t *testing.T) {
	mapFS := fstest.MapFS{
		"module.json": &fstest.MapFile{
			Data: []byte(`{
				"name": "test_module",
				"version": "1.0.0",
				"label": "Test Module",
				"models": ["models/*.json"],
				"apis": ["apis/*.json"]
			}`),
		},
		"models/test_model.json": &fstest.MapFile{
			Data: []byte(`{
				"name": "test_model",
				"fields": {
					"id": {"type": "uuid", "primary": true},
					"name": {"type": "string", "max": 100}
				}
			}`),
		},
		"apis/test_api.json": &fstest.MapFile{
			Data: []byte(`{
				"name": "test_api",
				"model": "test_model",
				"auto_crud": true
			}`),
		},
	}

	mfs := NewEmbedFSFromFS(mapFS, "")
	loaded, err := LoadModuleFromFS(mfs)
	if err != nil {
		t.Fatalf("LoadModuleFromFS() error = %v", err)
	}

	if loaded.Definition.Name != "test_module" {
		t.Errorf("Definition.Name = %q, want %q", loaded.Definition.Name, "test_module")
	}

	if len(loaded.Models) != 1 {
		t.Errorf("len(Models) = %d, want 1", len(loaded.Models))
	}

	if len(loaded.Models) > 0 && loaded.Models[0].Name != "test_model" {
		t.Errorf("Models[0].Name = %q, want %q", loaded.Models[0].Name, "test_model")
	}

	if len(loaded.APIs) != 1 {
		t.Errorf("len(APIs) = %d, want 1", len(loaded.APIs))
	}

	if len(loaded.APIs) > 0 && loaded.APIs[0].Name != "test_api" {
		t.Errorf("APIs[0].Name = %q, want %q", loaded.APIs[0].Name, "test_api")
	}

	if loaded.Path != "" {
		t.Errorf("Path = %q, want empty string", loaded.Path)
	}
}

func TestLayeredFS_DiscoverModules(t *testing.T) {
	tmpDir := t.TempDir()
	crmDir := filepath.Join(tmpDir, "crm")
	if err := os.Mkdir(crmDir, 0755); err != nil {
		t.Fatalf("failed to create crm dir: %v", err)
	}
	crmModule := filepath.Join(crmDir, "module.json")
	if err := os.WriteFile(crmModule, []byte(`{"name":"crm"}`), 0644); err != nil {
		t.Fatalf("failed to write crm module.json: %v", err)
	}

	diskFS := NewDiskFS(tmpDir)

	mapFS := fstest.MapFS{
		"base/module.json": &fstest.MapFile{Data: []byte(`{"name":"base"}`)},
	}
	embedFS := NewEmbedFSFromFS(mapFS, "")

	layered := NewLayeredFS(diskFS, embedFS)

	modules, err := layered.DiscoverModules()
	if err != nil {
		t.Fatalf("DiscoverModules() error = %v", err)
	}

	if len(modules) != 2 {
		t.Errorf("DiscoverModules() returned %d modules, want 2", len(modules))
	}

	moduleMap := make(map[string]bool)
	for _, mod := range modules {
		moduleMap[mod] = true
	}

	if !moduleMap["crm"] {
		t.Error("DiscoverModules() missing crm")
	}
	if !moduleMap["base"] {
		t.Error("DiscoverModules() missing base")
	}
}

func TestLayeredFS_DiscoverModules_Deduplicates(t *testing.T) {
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, "base")
	if err := os.Mkdir(baseDir, 0755); err != nil {
		t.Fatalf("failed to create base dir: %v", err)
	}
	baseModule := filepath.Join(baseDir, "module.json")
	if err := os.WriteFile(baseModule, []byte(`{"name":"base"}`), 0644); err != nil {
		t.Fatalf("failed to write base module.json: %v", err)
	}

	diskFS := NewDiskFS(tmpDir)

	mapFS := fstest.MapFS{
		"base/module.json": &fstest.MapFile{Data: []byte(`{"name":"base"}`)},
	}
	embedFS := NewEmbedFSFromFS(mapFS, "")

	layered := NewLayeredFS(diskFS, embedFS)

	modules, err := layered.DiscoverModules()
	if err != nil {
		t.Fatalf("DiscoverModules() error = %v", err)
	}

	if len(modules) != 1 {
		t.Errorf("DiscoverModules() returned %d modules, want 1", len(modules))
	}

	if modules[0] != "base" {
		t.Errorf("DiscoverModules() = %q, want %q", modules[0], "base")
	}
}

func TestLayeredFS_SubFS(t *testing.T) {
	mapFS := fstest.MapFS{
		"base/module.json": &fstest.MapFile{Data: []byte(`{"name":"base"}`)},
		"base/models/user.json": &fstest.MapFile{Data: []byte(`{"name":"user"}`)},
	}
	embedFS := NewEmbedFSFromFS(mapFS, "")

	layered := NewLayeredFS(embedFS)
	subFS := layered.SubFS("base")

	data, err := subFS.ReadFile("module.json")
	if err != nil {
		t.Fatalf("SubFS.ReadFile() error = %v", err)
	}

	if string(data) != `{"name":"base"}` {
		t.Errorf("SubFS.ReadFile() = %q, want %q", data, `{"name":"base"}`)
	}
}

func TestSubFS_ReadFile(t *testing.T) {
	mapFS := fstest.MapFS{
		"base/module.json": &fstest.MapFile{Data: []byte(`{"name":"base"}`)},
	}
	embedFS := NewEmbedFSFromFS(mapFS, "")

	subFS := NewSubFS(embedFS, "base")

	data, err := subFS.ReadFile("module.json")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if string(data) != `{"name":"base"}` {
		t.Errorf("ReadFile() = %q, want %q", data, `{"name":"base"}`)
	}
}

func TestSubFS_Glob(t *testing.T) {
	mapFS := fstest.MapFS{
		"base/models/user.json": &fstest.MapFile{Data: []byte(`{"name":"user"}`)},
		"base/models/role.json": &fstest.MapFile{Data: []byte(`{"name":"role"}`)},
	}
	embedFS := NewEmbedFSFromFS(mapFS, "")

	subFS := NewSubFS(embedFS, "base")

	matches, err := subFS.Glob("models/*.json")
	if err != nil {
		t.Fatalf("Glob() error = %v", err)
	}

	if len(matches) != 2 {
		t.Errorf("Glob() returned %d matches, want 2", len(matches))
	}

	matchMap := make(map[string]bool)
	for _, match := range matches {
		matchMap[match] = true
	}

	if !matchMap["models/user.json"] {
		t.Error("Glob() missing models/user.json")
	}
	if !matchMap["models/role.json"] {
		t.Error("Glob() missing models/role.json")
	}
}

func TestExtractModuleFS(t *testing.T) {
	mapFS := fstest.MapFS{
		"module.json": &fstest.MapFile{Data: []byte(`{"name":"test"}`)},
		"models/user.json": &fstest.MapFile{Data: []byte(`{"name":"user"}`)},
		"views/list.html": &fstest.MapFile{Data: []byte(`<html></html>`)},
	}
	mfs := NewEmbedFSFromFS(mapFS, "")

	tempDir, err := ExtractModuleFS(mfs, "test")
	if err != nil {
		t.Fatalf("ExtractModuleFS() error = %v", err)
	}
	defer os.RemoveAll(tempDir)

	moduleJSON := filepath.Join(tempDir, "module.json")
	if _, err := os.Stat(moduleJSON); err != nil {
		t.Errorf("module.json not found: %v", err)
	}

	userJSON := filepath.Join(tempDir, "models", "user.json")
	if _, err := os.Stat(userJSON); err != nil {
		t.Errorf("models/user.json not found: %v", err)
	}

	listHTML := filepath.Join(tempDir, "views", "list.html")
	if _, err := os.Stat(listHTML); err != nil {
		t.Errorf("views/list.html not found: %v", err)
	}

	data, err := os.ReadFile(moduleJSON)
	if err != nil {
		t.Fatalf("failed to read module.json: %v", err)
	}
	if string(data) != `{"name":"test"}` {
		t.Errorf("module.json content = %q, want %q", data, `{"name":"test"}`)
	}
}
