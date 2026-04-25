package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	domainstorage "github.com/bitcode-framework/bitcode/internal/domain/storage"
)

type LocalStorage struct {
	basePath string
	baseURL  string
}

func NewLocalStorage(cfg LocalStorageConfig) *LocalStorage {
	os.MkdirAll(cfg.Path, 0755)
	return &LocalStorage{
		basePath: cfg.Path,
		baseURL:  cfg.BaseURL,
	}
}

func (s *LocalStorage) Put(ctx context.Context, path string, reader io.Reader, opts domainstorage.PutOptions) error {
	fullPath := filepath.Join(s.basePath, path)
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	f, err := os.Create(fullPath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", fullPath, err)
	}
	defer f.Close()

	if _, err := io.Copy(f, reader); err != nil {
		return fmt.Errorf("failed to write file %s: %w", fullPath, err)
	}

	return nil
}

func (s *LocalStorage) Get(ctx context.Context, path string) (io.ReadCloser, error) {
	fullPath := filepath.Join(s.basePath, path)
	f, err := os.Open(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", fullPath, err)
	}
	return f, nil
}

func (s *LocalStorage) Delete(ctx context.Context, path string) error {
	fullPath := filepath.Join(s.basePath, path)
	if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete file %s: %w", fullPath, err)
	}
	return nil
}

func (s *LocalStorage) URL(ctx context.Context, path string, opts domainstorage.URLOptions) (string, error) {
	return s.baseURL + "/" + filepath.ToSlash(path), nil
}

func (s *LocalStorage) Exists(ctx context.Context, path string) (bool, error) {
	fullPath := filepath.Join(s.basePath, path)
	_, err := os.Stat(fullPath)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
