package api

import (
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	domainstorage "github.com/bitcode-engine/engine/internal/domain/storage"
	infrastorage "github.com/bitcode-engine/engine/internal/infrastructure/storage"
	"github.com/bitcode-engine/engine/pkg/security"
)

type FileHandler struct {
	repo       *infrastorage.AttachmentRepository
	storage    domainstorage.StorageDriver
	thumbnail  *infrastorage.ThumbnailService
	config     infrastorage.StorageConfig
	jwtCfg     security.JWTConfig
	scanHooks  []domainstorage.ScanHook
}

func NewFileHandler(
	repo *infrastorage.AttachmentRepository,
	storage domainstorage.StorageDriver,
	thumbnail *infrastorage.ThumbnailService,
	config infrastorage.StorageConfig,
	jwtCfg security.JWTConfig,
) *FileHandler {
	return &FileHandler{
		repo:      repo,
		storage:   storage,
		thumbnail: thumbnail,
		config:    config,
		jwtCfg:    jwtCfg,
	}
}

func (h *FileHandler) AddScanHook(hook domainstorage.ScanHook) {
	h.scanHooks = append(h.scanHooks, hook)
}

func (h *FileHandler) Register(app *fiber.App) {
	files := app.Group("/api")
	files.Post("/upload", h.Upload)
	files.Post("/uploads", h.UploadMultiple)
	files.Get("/files", h.List)
	files.Get("/files/:id", h.GetMetadata)
	files.Get("/files/:id/download", h.Download)
	files.Get("/files/:id/thumbnail", h.Thumbnail)
	files.Get("/files/:id/resize", h.Resize)
	files.Get("/files/:id/versions", h.Versions)
	files.Delete("/files/:id", h.Delete)
}

func (h *FileHandler) Upload(c *fiber.Ctx) error {
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "no file provided"})
	}

	att, err := h.processUpload(c, file)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(201).JSON(h.attachmentResponse(att))
}

func (h *FileHandler) UploadMultiple(c *fiber.Ctx) error {
	form, err := c.MultipartForm()
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid multipart form"})
	}

	files := form.File["files"]
	if len(files) == 0 {
		files = form.File["file"]
	}
	if len(files) == 0 {
		return c.Status(400).JSON(fiber.Map{"error": "no files provided"})
	}

	var results []fiber.Map
	var errors []string

	for _, file := range files {
		att, err := h.processUpload(c, file)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %s", file.Filename, err.Error()))
			continue
		}
		results = append(results, h.attachmentResponse(att))
	}

	return c.Status(201).JSON(fiber.Map{
		"files":  results,
		"total":  len(results),
		"errors": errors,
	})
}

func (h *FileHandler) GetMetadata(c *fiber.Ctx) error {
	id := c.Params("id")
	att, err := h.repo.FindByID(id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "file not found"})
	}

	if !att.IsPublic {
		if err := h.checkAuth(c); err != nil {
			return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
		}
	}

	return c.JSON(h.attachmentResponse(att))
}

func (h *FileHandler) Download(c *fiber.Ctx) error {
	id := c.Params("id")
	att, err := h.repo.FindByID(id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "file not found"})
	}

	if !att.IsPublic {
		if err := h.checkAuth(c); err != nil {
			return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
		}
	}

	reader, err := h.storage.Get(c.Context(), att.Path)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to read file"})
	}
	defer reader.Close()

	c.Set("Content-Type", att.MimeType)
	c.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, att.Name))
	c.Set("Content-Length", strconv.FormatInt(att.Size, 10))

	data, err := io.ReadAll(reader)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to read file"})
	}

	return c.Send(data)
}

func (h *FileHandler) Thumbnail(c *fiber.Ctx) error {
	id := c.Params("id")
	att, err := h.repo.FindByID(id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "file not found"})
	}

	if !att.IsPublic {
		if err := h.checkAuth(c); err != nil {
			return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
		}
	}

	if att.ThumbnailPath == "" {
		return c.Status(404).JSON(fiber.Map{"error": "no thumbnail available"})
	}

	reader, err := h.storage.Get(c.Context(), att.ThumbnailPath)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to read thumbnail"})
	}
	defer reader.Close()

	c.Set("Content-Type", att.MimeType)
	c.Set("Cache-Control", "public, max-age=86400")

	data, err := io.ReadAll(reader)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to read thumbnail"})
	}

	return c.Send(data)
}

func (h *FileHandler) Resize(c *fiber.Ctx) error {
	id := c.Params("id")
	att, err := h.repo.FindByID(id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "file not found"})
	}

	if !att.IsPublic {
		if err := h.checkAuth(c); err != nil {
			return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
		}
	}

	if !h.thumbnail.IsImage(att.MimeType) {
		return c.Status(400).JSON(fiber.Map{"error": "file is not an image"})
	}

	width, _ := strconv.Atoi(c.Query("w", "300"))
	height, _ := strconv.Atoi(c.Query("h", "300"))

	reader, err := h.thumbnail.Resize(c.Context(), att.Path, att.MimeType, width, height)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to resize image"})
	}

	if rc, ok := reader.(io.ReadCloser); ok {
		defer rc.Close()
	}

	c.Set("Content-Type", att.MimeType)
	c.Set("Cache-Control", "public, max-age=86400")

	data, err := io.ReadAll(reader)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to read resized image"})
	}

	return c.Send(data)
}

func (h *FileHandler) List(c *fiber.Ctx) error {
	if err := h.checkAuth(c); err != nil {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}

	model := c.Query("model")
	recordID := c.Query("record_id")
	fieldName := c.Query("field_name")
	page, _ := strconv.Atoi(c.Query("page", "1"))
	pageSize, _ := strconv.Atoi(c.Query("page_size", "20"))

	attachments, total, err := h.repo.FindByModelRecord(model, recordID, fieldName, page, pageSize)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to list files"})
	}

	var items []fiber.Map
	for _, att := range attachments {
		items = append(items, h.attachmentResponse(&att))
	}
	if items == nil {
		items = []fiber.Map{}
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	return c.JSON(fiber.Map{
		"data":        items,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": totalPages,
	})
}

func (h *FileHandler) Versions(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := h.checkAuth(c); err != nil {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}

	att, err := h.repo.FindByID(id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "file not found"})
	}

	parentID := att.ParentID
	if parentID == "" {
		parentID = att.ID
	}

	versions, err := h.repo.FindVersions(parentID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to list versions"})
	}

	var items []fiber.Map
	for _, v := range versions {
		items = append(items, h.attachmentResponse(&v))
	}
	if items == nil {
		items = []fiber.Map{}
	}

	return c.JSON(fiber.Map{
		"versions": items,
		"total":    len(items),
	})
}

func (h *FileHandler) Delete(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := h.checkAuth(c); err != nil {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}

	if err := h.repo.SoftDelete(id); err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "file not found"})
	}

	return c.JSON(fiber.Map{"status": "deleted"})
}

func (h *FileHandler) processUpload(c *fiber.Ctx, file *multipart.FileHeader) (*domainstorage.Attachment, error) {
	if file.Size > h.config.MaxSize {
		return nil, fmt.Errorf("file too large (max %dMB)", h.config.MaxSize/1024/1024)
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !h.isAllowedExt(ext) {
		return nil, fmt.Errorf("file type %s not allowed", ext)
	}

	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open file")
	}
	defer src.Close()

	data, err := io.ReadAll(src)
	if err != nil {
		return nil, fmt.Errorf("failed to read file")
	}

	hash := fmt.Sprintf("%x", sha256.Sum256(data))

	mimeType := http.DetectContentType(data)
	if ct := file.Header.Get("Content-Type"); ct != "" && ct != "application/octet-stream" {
		mimeType = ct
	}

	model := c.FormValue("model")
	recordID := c.FormValue("record_id")
	fieldName := c.FormValue("field_name")
	isPublicStr := c.FormValue("is_public")
	isPublic := isPublicStr == "true" || isPublicStr == "1"

	existing, _ := h.repo.FindByHash(hash, model, recordID, fieldName)
	if existing != nil {
		return existing, nil
	}

	pathFormat := c.FormValue("path_format")
	if pathFormat == "" {
		pathFormat = h.config.PathFormat
	}
	nameFormat := c.FormValue("name_format")
	if nameFormat == "" {
		nameFormat = h.config.NameFormat
	}

	var tenantID, userID string
	token := c.Cookies("token")
	if token == "" {
		authHeader := c.Get("Authorization")
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			token = authHeader[7:]
		}
	}
	if token != "" {
		claims, _ := security.ValidateToken(h.jwtCfg, token)
		if claims != nil {
			userID = claims.UserID
		}
	}
	tenantID = c.Get("X-Tenant-ID")

	original := infrastorage.OriginalWithoutExt(infrastorage.SanitizeFilename(file.Filename))

	fmtCtx := infrastorage.FormatContext{
		TenantID: tenantID,
		UserID:   userID,
		Model:    model,
		Original: original,
		Ext:      ext,
	}

	dir := infrastorage.FormatPath(pathFormat, fmtCtx)
	name := infrastorage.FormatName(nameFormat, fmtCtx)
	storagePath := dir + "/" + name

	for _, hook := range h.scanHooks {
		reader, err := hook.BeforePut(c.Context(), file.Filename, strings.NewReader(string(data)))
		if err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		scanned, _ := io.ReadAll(reader)
		if len(scanned) > 0 {
			data = scanned
		}
	}

	if err := h.storage.Put(c.Context(), storagePath, strings.NewReader(string(data)), domainstorage.PutOptions{
		ContentType: mimeType,
		IsPublic:    isPublic,
	}); err != nil {
		return nil, fmt.Errorf("failed to store file: %w", err)
	}

	url, _ := h.storage.URL(c.Context(), storagePath, domainstorage.URLOptions{IsPublic: isPublic})

	parentID := c.FormValue("parent_id")
	version := 1
	if parentID != "" {
		latestVersion, _ := h.repo.GetLatestVersion(model, recordID, fieldName, parentID)
		version = latestVersion + 1
	}

	att := &domainstorage.Attachment{
		ID:        uuid.New().String(),
		TenantID:  tenantID,
		UserID:    userID,
		Model:     model,
		RecordID:  recordID,
		FieldName: fieldName,
		Name:      infrastorage.SanitizeFilename(file.Filename),
		Path:      storagePath,
		URL:       url,
		Storage:   h.config.Driver,
		Size:      file.Size,
		MimeType:  mimeType,
		Ext:       ext,
		Hash:      hash,
		IsPublic:  isPublic,
		Version:   version,
		ParentID:  parentID,
		Active:    true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if h.thumbnail.IsImage(mimeType) {
		thumbPath, err := h.thumbnail.GenerateThumbnail(c.Context(), storagePath, mimeType)
		if err != nil {
			log.Printf("[STORAGE] thumbnail generation failed for %s: %v", storagePath, err)
		} else {
			att.ThumbnailPath = thumbPath
		}
	}

	if err := h.repo.Create(att); err != nil {
		return nil, fmt.Errorf("failed to save attachment record: %w", err)
	}

	if parentID != "" {
		h.repo.CleanupVersions(parentID, 5)
	}

	return att, nil
}

func (h *FileHandler) isAllowedExt(ext string) bool {
	if len(h.config.AllowedExtensions) == 0 {
		return true
	}
	for _, allowed := range h.config.AllowedExtensions {
		if ext == allowed {
			return true
		}
	}
	return false
}

func (h *FileHandler) checkAuth(c *fiber.Ctx) error {
	token := c.Cookies("token")
	if token == "" {
		authHeader := c.Get("Authorization")
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			token = authHeader[7:]
		}
	}
	if token == "" {
		return fmt.Errorf("no token")
	}
	_, err := security.ValidateToken(h.jwtCfg, token)
	return err
}

func (h *FileHandler) attachmentResponse(att *domainstorage.Attachment) fiber.Map {
	resp := fiber.Map{
		"id":        att.ID,
		"name":      att.Name,
		"url":       fmt.Sprintf("/api/files/%s/download", att.ID),
		"size":      att.Size,
		"mime_type": att.MimeType,
		"ext":       att.Ext,
		"hash":      att.Hash,
		"is_public": att.IsPublic,
		"version":   att.Version,
		"storage":   att.Storage,
		"model":     att.Model,
		"record_id": att.RecordID,
		"field_name": att.FieldName,
		"created_at": att.CreatedAt.Format(time.RFC3339),
		"updated_at": att.UpdatedAt.Format(time.RFC3339),
	}

	if att.ThumbnailPath != "" {
		resp["thumbnail_url"] = fmt.Sprintf("/api/files/%s/thumbnail", att.ID)
	} else {
		resp["thumbnail_url"] = nil
	}

	if att.Metadata != "" {
		resp["metadata"] = att.Metadata
	}

	if att.ParentID != "" {
		resp["parent_id"] = att.ParentID
	}

	return resp
}
