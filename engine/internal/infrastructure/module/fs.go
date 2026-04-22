package module

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
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
	fullPath := filepath.Join(d.root, path)
	return os.ReadFile(fullPath)
}

func (d *DiskFS) Glob(pattern string) ([]string, error) {
	fullPattern := filepath.Join(d.root, pattern)
	matches, err := filepath.Glob(fullPattern)
	if err != nil {
		return nil, err
	}

	relMatches := make([]string, 0, len(matches))
	for _, match := range matches {
		rel, err := filepath.Rel(d.root, match)
		if err != nil {
			return nil, err
		}
		rel = filepath.ToSlash(rel)
		relMatches = append(relMatches, rel)
	}

	return relMatches, nil
}

func (d *DiskFS) Exists(path string) bool {
	fullPath := filepath.Join(d.root, path)
	_, err := os.Stat(fullPath)
	return err == nil
}

func (d *DiskFS) ListDir(path string) ([]string, error) {
	fullPath := filepath.Join(d.root, path)
	entries, err := os.ReadDir(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fs.ErrNotExist
		}
		return nil, err
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}

	return names, nil
}

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
	fullPath := path
	if e.prefix != "" {
		fullPath = e.prefix + "/" + path
	}
	return fs.ReadFile(e.fsys, fullPath)
}

func (e *EmbedFS) Glob(pattern string) ([]string, error) {
	fullPattern := pattern
	if e.prefix != "" {
		fullPattern = e.prefix + "/" + pattern
	}
	matches, err := fs.Glob(e.fsys, fullPattern)
	if err != nil {
		return nil, err
	}

	if e.prefix == "" {
		return matches, nil
	}

	relMatches := make([]string, 0, len(matches))
	prefixWithSlash := e.prefix + "/"
	for _, match := range matches {
		if strings.HasPrefix(match, prefixWithSlash) {
			relMatches = append(relMatches, strings.TrimPrefix(match, prefixWithSlash))
		}
	}

	return relMatches, nil
}

func (e *EmbedFS) Exists(path string) bool {
	fullPath := path
	if e.prefix != "" {
		fullPath = e.prefix + "/" + path
	}
	_, err := fs.Stat(e.fsys, fullPath)
	return err == nil
}

func (e *EmbedFS) ListDir(path string) ([]string, error) {
	fullPath := path
	if e.prefix != "" {
		if path == "" || path == "." {
			fullPath = e.prefix
		} else {
			fullPath = e.prefix + "/" + path
		}
	} else if path == "" {
		fullPath = "."
	}

	entries, err := fs.ReadDir(e.fsys, fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fs.ErrNotExist
		}
		return nil, err
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}

	return names, nil
}

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
			return nil, err
		}
		for _, match := range matches {
			normalized := filepath.ToSlash(match)
			if !seen[normalized] {
				seen[normalized] = true
				result = append(result, match)
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
		entries, err := layer.ListDir(path)
		if err != nil && err != fs.ErrNotExist {
			return nil, err
		}
		for _, entry := range entries {
			if !seen[entry] {
				seen[entry] = true
				result = append(result, entry)
			}
		}
	}

	return result, nil
}

func (l *LayeredFS) DiscoverModules() ([]string, error) {
	seen := make(map[string]bool)
	var modules []string

	for _, layer := range l.layers {
		entries, err := layer.ListDir(".")
		if err != nil && err != fs.ErrNotExist {
			return nil, err
		}
		for _, entry := range entries {
			if !seen[entry] && layer.Exists(entry+"/module.json") {
				seen[entry] = true
				modules = append(modules, entry)
			}
		}
	}

	return modules, nil
}

func (l *LayeredFS) SubFS(moduleName string) *LayeredFS {
	subLayers := make([]ModuleFS, len(l.layers))
	for i, layer := range l.layers {
		subLayers[i] = NewSubFS(layer, moduleName)
	}
	return NewLayeredFS(subLayers...)
}

type SubFS struct {
	parent ModuleFS
	prefix string
}

func NewSubFS(parent ModuleFS, prefix string) *SubFS {
	return &SubFS{parent: parent, prefix: prefix}
}

func (s *SubFS) ReadFile(path string) ([]byte, error) {
	fullPath := s.prefix + "/" + path
	return s.parent.ReadFile(fullPath)
}

func (s *SubFS) Glob(pattern string) ([]string, error) {
	fullPattern := s.prefix + "/" + pattern
	matches, err := s.parent.Glob(fullPattern)
	if err != nil {
		return nil, err
	}

	prefixWithSlash := s.prefix + "/"
	relMatches := make([]string, 0, len(matches))
	for _, match := range matches {
		if strings.HasPrefix(match, prefixWithSlash) {
			relMatches = append(relMatches, strings.TrimPrefix(match, prefixWithSlash))
		}
	}

	return relMatches, nil
}

func (s *SubFS) Exists(path string) bool {
	fullPath := s.prefix + "/" + path
	return s.parent.Exists(fullPath)
}

func (s *SubFS) ListDir(path string) ([]string, error) {
	fullPath := s.prefix
	if path != "." && path != "" {
		fullPath = s.prefix + "/" + path
	}
	return s.parent.ListDir(fullPath)
}

func ExtractModuleFS(mfs ModuleFS, modName string) (string, error) {
	tempDir, err := os.MkdirTemp("", "bitcode-module-"+modName+"-")
	if err != nil {
		return "", err
	}

	if err := extractRecursive(mfs, ".", tempDir); err != nil {
		os.RemoveAll(tempDir)
		return "", err
	}

	return tempDir, nil
}

func extractRecursive(mfs ModuleFS, dir string, targetDir string) error {
	entries, err := mfs.ListDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := entry
		if dir != "." && dir != "" {
			srcPath = dir + "/" + entry
		}

		targetPath := filepath.Join(targetDir, srcPath)

		data, err := mfs.ReadFile(srcPath)
		if err == nil {
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return err
			}
			if err := os.WriteFile(targetPath, data, 0644); err != nil {
				return err
			}
		} else {
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return err
			}
			if err := extractRecursive(mfs, srcPath, targetDir); err != nil {
				return err
			}
		}
	}

	return nil
}
