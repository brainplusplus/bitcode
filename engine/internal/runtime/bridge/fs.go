package bridge

import (
	"os"
	"path/filepath"
	"strings"
)

type fsBridge struct {
	basePath string
	rules    SecurityRules
}

func newFSBridge(modulePath string, rules SecurityRules) *fsBridge {
	return &fsBridge{basePath: modulePath, rules: rules}
}

func (f *fsBridge) resolve(path string) (string, error) {
	if filepath.IsAbs(path) {
		if !f.isAllowedAbsPath(path) {
			return "", NewErrorf(ErrFSAccessDenied, "access denied for path '%s'", path)
		}
		return filepath.Clean(path), nil
	}

	resolved := filepath.Join(f.basePath, path)
	resolved = filepath.Clean(resolved)

	if !strings.HasPrefix(resolved, filepath.Clean(f.basePath)) {
		return "", NewErrorf(ErrFSAccessDenied, "path escapes module directory: '%s'", path)
	}
	return resolved, nil
}

func (f *fsBridge) isAllowedAbsPath(path string) bool {
	cleanPath := filepath.Clean(path)

	deniedDirs := []string{"internal", "plugins", "cmd"}
	for _, d := range deniedDirs {
		if strings.Contains(cleanPath, string(filepath.Separator)+d+string(filepath.Separator)) ||
			strings.HasSuffix(cleanPath, string(filepath.Separator)+d) {
			return false
		}
	}

	if matchesAny(cleanPath, f.rules.FSDeny) {
		return false
	}

	if len(f.rules.FSAllow) > 0 {
		for _, allowed := range f.rules.FSAllow {
			allowedClean := filepath.Clean(allowed)
			if strings.HasPrefix(cleanPath, allowedClean) {
				return true
			}
		}
		return false
	}

	tmpDir := os.TempDir()
	return strings.HasPrefix(cleanPath, tmpDir)
}

func (f *fsBridge) Read(path string) (string, error) {
	resolved, err := f.resolve(path)
	if err != nil {
		return "", err
	}
	data, readErr := os.ReadFile(resolved)
	if readErr != nil {
		if os.IsNotExist(readErr) {
			return "", NewErrorf(ErrFSNotFound, "file not found: '%s'", path)
		}
		return "", NewError(ErrInternalError, readErr.Error())
	}
	return string(data), nil
}

func (f *fsBridge) Write(path string, content string) error {
	resolved, err := f.resolve(path)
	if err != nil {
		return err
	}
	dir := filepath.Dir(resolved)
	if mkErr := os.MkdirAll(dir, 0755); mkErr != nil {
		return NewError(ErrInternalError, mkErr.Error())
	}
	return os.WriteFile(resolved, []byte(content), 0644)
}

func (f *fsBridge) Exists(path string) (bool, error) {
	resolved, err := f.resolve(path)
	if err != nil {
		return false, err
	}
	_, statErr := os.Stat(resolved)
	if statErr != nil {
		if os.IsNotExist(statErr) {
			return false, nil
		}
		return false, NewError(ErrInternalError, statErr.Error())
	}
	return true, nil
}

func (f *fsBridge) List(path string) ([]string, error) {
	resolved, err := f.resolve(path)
	if err != nil {
		return nil, err
	}
	entries, readErr := os.ReadDir(resolved)
	if readErr != nil {
		if os.IsNotExist(readErr) {
			return nil, NewErrorf(ErrFSNotFound, "directory not found: '%s'", path)
		}
		return nil, NewError(ErrInternalError, readErr.Error())
	}
	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.Name()
	}
	return names, nil
}

func (f *fsBridge) Mkdir(path string) error {
	resolved, err := f.resolve(path)
	if err != nil {
		return err
	}
	return os.MkdirAll(resolved, 0755)
}

func (f *fsBridge) Remove(path string) error {
	resolved, err := f.resolve(path)
	if err != nil {
		return err
	}
	return os.Remove(resolved)
}
