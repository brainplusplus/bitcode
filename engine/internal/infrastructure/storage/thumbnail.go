package storage

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"

	domainstorage "github.com/bitcode-framework/bitcode/internal/domain/storage"
)

type ThumbnailService struct {
	storage domainstorage.StorageDriver
	config  ThumbnailConfig
}

func NewThumbnailService(storage domainstorage.StorageDriver, cfg ThumbnailConfig) *ThumbnailService {
	return &ThumbnailService{
		storage: storage,
		config:  cfg,
	}
}

func (s *ThumbnailService) IsImage(mimeType string) bool {
	return strings.HasPrefix(mimeType, "image/") &&
		!strings.Contains(mimeType, "svg") &&
		!strings.Contains(mimeType, "ico")
}

func (s *ThumbnailService) GenerateThumbnail(ctx context.Context, sourcePath string, mimeType string) (string, error) {
	if !s.config.Enabled {
		return "", nil
	}

	reader, err := s.storage.Get(ctx, sourcePath)
	if err != nil {
		return "", fmt.Errorf("failed to read source image: %w", err)
	}
	defer reader.Close()

	img, _, err := image.Decode(reader)
	if err != nil {
		return "", fmt.Errorf("failed to decode image: %w", err)
	}

	thumb := imaging.Fit(img, s.config.Width, s.config.Height, imaging.Lanczos)

	thumbPath := thumbnailPath(sourcePath)
	var buf bytes.Buffer
	if err := encodeImage(&buf, thumb, mimeType, s.config.Quality); err != nil {
		return "", fmt.Errorf("failed to encode thumbnail: %w", err)
	}

	if err := s.storage.Put(ctx, thumbPath, &buf, domainstorage.PutOptions{
		ContentType: mimeType,
	}); err != nil {
		return "", fmt.Errorf("failed to save thumbnail: %w", err)
	}

	return thumbPath, nil
}

func (s *ThumbnailService) Resize(ctx context.Context, sourcePath string, mimeType string, width, height int) (io.Reader, error) {
	if width <= 0 || width > 2000 {
		width = s.config.Width
	}
	if height <= 0 || height > 2000 {
		height = s.config.Height
	}

	cachePath := resizeCachePath(sourcePath, width, height)
	exists, _ := s.storage.Exists(ctx, cachePath)
	if exists {
		return s.storage.Get(ctx, cachePath)
	}

	reader, err := s.storage.Get(ctx, sourcePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read source image: %w", err)
	}
	defer reader.Close()

	img, _, err := image.Decode(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	resized := imaging.Fit(img, width, height, imaging.Lanczos)

	var buf bytes.Buffer
	if err := encodeImage(&buf, resized, mimeType, s.config.Quality); err != nil {
		return nil, fmt.Errorf("failed to encode resized image: %w", err)
	}

	resizedBytes := buf.Bytes()

	_ = s.storage.Put(ctx, cachePath, bytes.NewReader(resizedBytes), domainstorage.PutOptions{
		ContentType: mimeType,
	})

	return bytes.NewReader(resizedBytes), nil
}

func thumbnailPath(path string) string {
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)
	return base + "_thumb" + ext
}

func resizeCachePath(path string, w, h int) string {
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)
	return fmt.Sprintf("%s_w%d_h%d%s", base, w, h, ext)
}

func encodeImage(w io.Writer, img image.Image, mimeType string, quality int) error {
	switch {
	case strings.Contains(mimeType, "png"):
		return png.Encode(w, img)
	default:
		return jpeg.Encode(w, img, &jpeg.Options{Quality: quality})
	}
}
