package admin

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/bitcode-engine/engine/internal/compiler/parser"
)

func (a *AdminPanel) apiViewDetail(c *fiber.Ctx) error {
	key := c.Params("module") + "/" + c.Params("name")
	views := a.views()
	info, ok := views[key]
	if !ok {
		return c.Status(404).JSON(fiber.Map{"error": "view not found"})
	}

	revisions, _ := a.revisionRepo.ListByViewKey(key, 50)

	revList := make([]fiber.Map, len(revisions))
	for i, r := range revisions {
		revList[i] = fiber.Map{
			"version":    r.Version,
			"created_by": r.CreatedBy,
			"created_at": r.CreatedAt,
		}
	}

	return c.JSON(fiber.Map{
		"key":       key,
		"name":      info.Def.Name,
		"type":      info.Def.Type,
		"model":     info.Def.Model,
		"title":     info.Def.Title,
		"module":    info.Module,
		"editable":  info.Editable,
		"file_path": info.FilePath,
		"revisions": revList,
	})
}

func (a *AdminPanel) apiViewJSON(c *fiber.Ctx) error {
	key := c.Params("module") + "/" + c.Params("name")
	views := a.views()
	info, ok := views[key]
	if !ok {
		return c.Status(404).JSON(fiber.Map{"error": "view not found"})
	}

	if info.FilePath == "" {
		return c.Status(404).JSON(fiber.Map{"error": "no file path for view"})
	}

	data, err := os.ReadFile(info.FilePath)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fmt.Sprintf("failed to read file: %v", err)})
	}

	c.Set("Content-Type", "application/json")
	return c.Send(data)
}

func (a *AdminPanel) apiViewSave(c *fiber.Ctx) error {
	key := c.Params("module") + "/" + c.Params("name")
	views := a.views()
	info, ok := views[key]
	if !ok {
		return c.Status(404).JSON(fiber.Map{"error": "view not found"})
	}

	if !info.Editable {
		return c.Status(403).JSON(fiber.Map{"error": "view is embedded (read-only). Use 'bitcode publish' or the publish button to make it editable."})
	}

	var body struct {
		Content string `json:"content"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	if _, err := parser.ParseView([]byte(body.Content)); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fmt.Sprintf("invalid view JSON: %v", err)})
	}

	var prettyJSON []byte
	var raw json.RawMessage
	if err := json.Unmarshal([]byte(body.Content), &raw); err == nil {
		prettyJSON, _ = json.MarshalIndent(raw, "", "  ")
	} else {
		prettyJSON = []byte(body.Content)
	}

	if err := os.WriteFile(info.FilePath, prettyJSON, 0644); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fmt.Sprintf("failed to write file: %v", err)})
	}

	rev, err := a.revisionRepo.Create(key, string(prettyJSON), "admin")
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fmt.Sprintf("failed to create revision: %v", err)})
	}

	a.revisionRepo.Cleanup(key, 50)

	return c.JSON(fiber.Map{"ok": true, "version": rev.Version})
}

func (a *AdminPanel) apiViewRollback(c *fiber.Ctx) error {
	key := c.Params("module") + "/" + c.Params("name")
	versionStr := c.Params("version")
	version, err := strconv.Atoi(versionStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid version number"})
	}

	views := a.views()
	info, ok := views[key]
	if !ok {
		return c.Status(404).JSON(fiber.Map{"error": "view not found"})
	}

	if !info.Editable {
		return c.Status(403).JSON(fiber.Map{"error": "view is embedded (read-only)"})
	}

	rev, err := a.revisionRepo.GetByVersion(key, version)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fmt.Sprintf("revision %d not found", version)})
	}

	if err := os.WriteFile(info.FilePath, []byte(rev.Content), 0644); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fmt.Sprintf("failed to write file: %v", err)})
	}

	newRev, err := a.revisionRepo.Create(key, rev.Content, "admin")
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fmt.Sprintf("failed to create revision: %v", err)})
	}

	return c.JSON(fiber.Map{"ok": true, "version": newRev.Version, "restored_from": version})
}

func (a *AdminPanel) apiViewPreview(c *fiber.Ctx) error {
	key := c.Params("module") + "/" + c.Params("name")
	views := a.views()
	info, ok := views[key]
	if !ok {
		return c.Status(404).SendString("View not found")
	}

	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.SendString(fmt.Sprintf(`<!DOCTYPE html><html><head><meta charset="utf-8"><style>body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',system-ui,sans-serif;padding:20px;color:#1f272e;font-size:13px}code{background:#f4f5f6;padding:1px 5px;border-radius:3px;font-size:12px}</style></head><body><h3>%s</h3><p>Type: <code>%s</code> | Model: <code>%s</code></p><p><a href="/app/%s" target="_blank">Open full view &rarr;</a></p></body></html>`,
		info.Def.Title, info.Def.Type, info.Def.Model, key))
}

func (a *AdminPanel) apiViewPublish(c *fiber.Ctx) error {
	key := c.Params("module") + "/" + c.Params("name")
	modName := c.Params("module")
	viewName := c.Params("name")

	views := a.views()
	info, ok := views[key]
	if !ok {
		return c.Status(404).JSON(fiber.Map{"error": "view not found"})
	}

	if info.Editable {
		return c.JSON(fiber.Map{"ok": true, "message": "view is already editable"})
	}

	if info.FilePath == "" {
		return c.Status(500).JSON(fiber.Map{"error": "no source file path"})
	}

	srcData, err := os.ReadFile(info.FilePath)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fmt.Sprintf("failed to read source: %v", err)})
	}

	targetDir := filepath.Join(a.moduleDir, modName, "views")
	os.MkdirAll(targetDir, 0755)
	targetPath := filepath.Join(targetDir, viewName+".json")

	if err := os.WriteFile(targetPath, srcData, 0644); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fmt.Sprintf("failed to write: %v", err)})
	}

	return c.JSON(fiber.Map{"ok": true, "message": "view published to " + targetPath})
}
