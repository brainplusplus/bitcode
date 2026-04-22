package api

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type UploadConfig struct {
	UploadDir   string
	MaxSize     int64
	AllowedExts []string
}

func DefaultUploadConfig() UploadConfig {
	return UploadConfig{
		UploadDir:   "uploads",
		MaxSize:     10 * 1024 * 1024,
		AllowedExts: []string{".jpg", ".jpeg", ".png", ".gif", ".pdf", ".doc", ".docx", ".xls", ".xlsx", ".csv", ".txt", ".zip"},
	}
}

func RegisterUploadRoutes(app *fiber.App, cfg UploadConfig) {
	os.MkdirAll(cfg.UploadDir, 0755)

	app.Post("/api/upload", func(c *fiber.Ctx) error {
		file, err := c.FormFile("file")
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "no file provided"})
		}

		if file.Size > cfg.MaxSize {
			return c.Status(400).JSON(fiber.Map{"error": fmt.Sprintf("file too large (max %dMB)", cfg.MaxSize/1024/1024)})
		}

		ext := strings.ToLower(filepath.Ext(file.Filename))
		allowed := false
		for _, a := range cfg.AllowedExts {
			if ext == a {
				allowed = true
				break
			}
		}
		if !allowed {
			return c.Status(400).JSON(fiber.Map{"error": "file type not allowed"})
		}

		dateDir := time.Now().Format("2006/01")
		dir := filepath.Join(cfg.UploadDir, dateDir)
		os.MkdirAll(dir, 0755)

		filename := uuid.New().String() + ext
		savePath := filepath.Join(dir, filename)

		if err := c.SaveFile(file, savePath); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "failed to save file"})
		}

		url := "/uploads/" + dateDir + "/" + filename

		return c.Status(201).JSON(fiber.Map{
			"url":      url,
			"filename": file.Filename,
			"size":     file.Size,
			"type":     ext,
		})
	})

	app.Static("/uploads", cfg.UploadDir)
}
