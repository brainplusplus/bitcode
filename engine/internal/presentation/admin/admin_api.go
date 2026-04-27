package admin

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/bitcode-framework/bitcode/internal/compiler/parser"
	"github.com/bitcode-framework/bitcode/internal/infrastructure/persistence"
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

func (a *AdminPanel) apiDataRevisions(c *fiber.Ctx) error {
	modelName := c.Params("model")
	recordID := c.Params("id")

	revisions, err := a.dataRevisionRepo.ListByRecord(modelName, recordID, 50)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	revList := make([]fiber.Map, len(revisions))
	for i, r := range revisions {
		revList[i] = fiber.Map{
			"version":    r.Version,
			"action":     r.Action,
			"user_id":    r.UserID,
			"created_at": r.CreatedAt,
			"changes":    r.Changes,
		}
	}

	return c.JSON(fiber.Map{
		"model":     modelName,
		"record_id": recordID,
		"revisions": revList,
		"total":     len(revList),
	})
}

func (a *AdminPanel) apiDataRevisionDetail(c *fiber.Ctx) error {
	modelName := c.Params("model")
	recordID := c.Params("id")
	versionStr := c.Params("version")
	version, err := strconv.Atoi(versionStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid version number"})
	}

	rev, err := a.dataRevisionRepo.GetByVersion(modelName, recordID, version)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fmt.Sprintf("revision %d not found", version)})
	}

	snapshot, _ := a.dataRevisionRepo.GetSnapshotMap(rev)

	return c.JSON(fiber.Map{
		"version":    rev.Version,
		"action":     rev.Action,
		"snapshot":   snapshot,
		"changes":    rev.Changes,
		"user_id":    rev.UserID,
		"created_at": rev.CreatedAt,
	})
}

func (a *AdminPanel) apiDataRestore(c *fiber.Ctx) error {
	modelName := c.Params("model")
	recordID := c.Params("id")
	versionStr := c.Params("version")
	version, err := strconv.Atoi(versionStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid version number"})
	}

	rev, err := a.dataRevisionRepo.GetByVersion(modelName, recordID, version)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fmt.Sprintf("revision %d not found", version)})
	}

	snapshot, err := a.dataRevisionRepo.GetSnapshotMap(rev)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to parse snapshot"})
	}

	delete(snapshot, "created_at")
	delete(snapshot, "updated_at")

	tableName := a.modelRegistry.TableName(modelName)
	repo := persistence.NewGenericRepository(a.db, tableName)
	if err := repo.Update(c.Context(), recordID, snapshot); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fmt.Sprintf("failed to restore: %v", err)})
	}

	var after map[string]any
	a.db.Table(tableName).Where("id = ?", recordID).Take(&after)

	a.dataRevisionRepo.Create(modelName, recordID, "restore", after, nil, "admin")

	return c.JSON(fiber.Map{
		"ok":             true,
		"restored_from":  version,
		"message":        fmt.Sprintf("record %s restored from version %d", recordID, version),
	})
}

func (a *AdminPanel) apiModelJSON(c *fiber.Ctx) error {
	name := c.Params("name")
	model, err := a.modelRegistry.Get(name)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "model not found"})
	}

	data, err := json.MarshalIndent(model, "", "  ")
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to serialize model"})
	}

	c.Set("Content-Type", "application/json")
	return c.Send(data)
}

func (a *AdminPanel) apiModelSave(c *fiber.Ctx) error {
	name := c.Params("name")

	var body struct {
		Content string `json:"content"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	if _, err := parser.ParseModel([]byte(body.Content)); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fmt.Sprintf("invalid model JSON: %v", err)})
	}

	var prettyJSON []byte
	var raw json.RawMessage
	if err := json.Unmarshal([]byte(body.Content), &raw); err == nil {
		prettyJSON, _ = json.MarshalIndent(raw, "", "  ")
	} else {
		prettyJSON = []byte(body.Content)
	}

	modelDef, err := a.modelRegistry.Get(name)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "model not found"})
	}
	modelPath := a.findModelFile(modelDef)
	if modelPath == "" {
		return c.Status(404).JSON(fiber.Map{"error": "model file not found on disk"})
	}

	if err := os.WriteFile(modelPath, prettyJSON, 0644); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fmt.Sprintf("failed to write file: %v", err)})
	}

	return c.JSON(fiber.Map{"ok": true, "message": "model saved"})
}

func (a *AdminPanel) apiRecordTimeline(c *fiber.Ctx) error {
	modelName := c.Params("model")
	recordID := c.Params("id")

	revisions, _ := a.dataRevisionRepo.ListByRecord(modelName, recordID, 50)
	auditLogs, _ := a.auditLogRepo.FindByRecord(modelName, recordID, 50)

	var timeline []fiber.Map

	for _, r := range revisions {
		var changes any
		if r.Changes != "" && r.Changes != "null" {
			json.Unmarshal([]byte(r.Changes), &changes)
		}
		timeline = append(timeline, fiber.Map{
			"type":       "revision",
			"version":    r.Version,
			"action":     r.Action,
			"user_id":    r.UserID,
			"changes":    changes,
			"created_at": r.CreatedAt,
		})
	}

	for _, log := range auditLogs {
		timeline = append(timeline, fiber.Map{
			"type":           "audit",
			"action":         log["action"],
			"user_id":        log["user_id"],
			"request_method": log["request_method"],
			"request_path":   log["request_path"],
			"status_code":    log["status_code"],
			"ip_address":     log["ip_address"],
			"created_at":     log["created_at"],
		})
	}

	return c.JSON(fiber.Map{
		"model":     modelName,
		"record_id": recordID,
		"timeline":  timeline,
		"total":     len(timeline),
	})
}

func (a *AdminPanel) apiLoginHistory(c *fiber.Ctx) error {
	limitStr := c.Query("limit", "100")
	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 || limit > 500 {
		limit = 100
	}

	results, err := a.auditLogRepo.FindLoginHistory(limit)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"data":  results,
		"total": len(results),
	})
}

func (a *AdminPanel) apiRequestLog(c *fiber.Ctx) error {
	limitStr := c.Query("limit", "100")
	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	methodFilter := c.Query("method", "")

	results, err := a.auditLogRepo.FindRequests(limit, methodFilter)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"data":  results,
		"total": len(results),
	})
}

