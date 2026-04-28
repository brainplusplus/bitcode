package bridge

import (
	"bytes"
	"context"
	"io"

	domainstorage "github.com/bitcode-framework/bitcode/internal/domain/storage"
)

type storageBridge struct {
	driver domainstorage.StorageDriver
}

func newStorageBridge(driver domainstorage.StorageDriver) *storageBridge {
	return &storageBridge{driver: driver}
}

func (s *storageBridge) Upload(opts UploadOptions) (*Attachment, error) {
	ctx := context.Background()
	path := opts.Filename
	if opts.Model != "" {
		path = opts.Model + "/" + opts.Filename
	}

	reader := bytes.NewReader(opts.Content)
	putOpts := domainstorage.PutOptions{
		ContentType: detectContentType(opts.Filename),
	}

	if err := s.driver.Put(ctx, path, reader, putOpts); err != nil {
		return nil, NewRetryableError(ErrStorageError, err.Error())
	}

	url, err := s.driver.URL(ctx, path, domainstorage.URLOptions{})
	if err != nil {
		url = path
	}

	return &Attachment{
		ID:          path,
		URL:         url,
		Filename:    opts.Filename,
		Size:        int64(len(opts.Content)),
		ContentType: putOpts.ContentType,
	}, nil
}

func (s *storageBridge) URL(id string) (string, error) {
	url, err := s.driver.URL(context.Background(), id, domainstorage.URLOptions{})
	if err != nil {
		return "", NewError(ErrStorageError, err.Error())
	}
	return url, nil
}

func (s *storageBridge) Download(id string) ([]byte, error) {
	reader, err := s.driver.Get(context.Background(), id)
	if err != nil {
		return nil, NewError(ErrStorageError, err.Error())
	}
	defer reader.Close()
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, NewRetryableError(ErrStorageError, err.Error())
	}
	return data, nil
}

func (s *storageBridge) Delete(id string) error {
	if err := s.driver.Delete(context.Background(), id); err != nil {
		return NewError(ErrStorageError, err.Error())
	}
	return nil
}

func detectContentType(filename string) string {
	ext := ""
	for i := len(filename) - 1; i >= 0; i-- {
		if filename[i] == '.' {
			ext = filename[i:]
			break
		}
	}
	switch ext {
	case ".pdf":
		return "application/pdf"
	case ".json":
		return "application/json"
	case ".csv":
		return "text/csv"
	case ".txt":
		return "text/plain"
	case ".html", ".htm":
		return "text/html"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".svg":
		return "image/svg+xml"
	case ".zip":
		return "application/zip"
	case ".xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	default:
		return "application/octet-stream"
	}
}
