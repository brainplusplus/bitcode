package storage

import (
	"context"
	"io"
	"time"
)

type PutOptions struct {
	ContentType string
	Metadata    map[string]string
	IsPublic    bool
}

type URLOptions struct {
	Expiry   time.Duration
	IsPublic bool
}

type StorageDriver interface {
	Put(ctx context.Context, path string, reader io.Reader, opts PutOptions) error
	Get(ctx context.Context, path string) (io.ReadCloser, error)
	Delete(ctx context.Context, path string) error
	URL(ctx context.Context, path string, opts URLOptions) (string, error)
	Exists(ctx context.Context, path string) (bool, error)
}

type ScanHook interface {
	BeforePut(ctx context.Context, filename string, reader io.Reader) (io.Reader, error)
}
