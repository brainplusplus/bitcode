package admin

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/bitcode-framework/bitcode/internal/infrastructure/module"
)

func (a *AdminPanel) securitySyncPage(c *fiber.Ctx) error {
	var html strings.Builder
	html.WriteString(a.pageHeader("Security", "securities"))

	html.WriteString(`<div class="card"><div class="card-title">Security Sync</div><div style="padding:16px">`)
	html.WriteString(`<div style="display:flex;gap:8px;flex-wrap:wrap;margin-bottom:16px">`)
	html.WriteString(`<button onclick="loadFromFiles()" class="btn-sm" style="cursor:pointer">Load from Files</button>`)
	html.WriteString(`<button onclick="exportToFiles()" class="btn-sm" style="cursor:pointer">Export to Files</button>`)
	html.WriteString(`<a href="/admin/api/securities/download" class="btn-sm">Download All (ZIP)</a>`)
	html.WriteString(`<label class="btn-sm" style="cursor:pointer">Upload JSON/ZIP<input type="file" accept=".json,.zip" onchange="uploadFile(this)" style="display:none"></label>`)
	html.WriteString(`</div>`)
	html.WriteString(`<div id="sync-status" class="text-muted"></div>`)
	html.WriteString(`</div></div>`)

	var history []map[string]any
	a.db.Table("ir_security_histories").Order("created_at DESC").Limit(50).Find(&history)

	html.WriteString(`<div class="card"><div class="card-title">Security History</div>`)
	html.WriteString(`<table><thead><tr><th>Date</th><th>Entity</th><th>Action</th><th>Changes</th><th>Source</th><th></th></tr></thead><tbody>`)
	for _, h := range history {
		createdAt := fmt.Sprintf("%v", h["created_at"])
		entityName := fmt.Sprintf("%v", h["entity_name"])
		action := fmt.Sprintf("%v", h["action"])
		changes := fmt.Sprintf("%v", h["changes"])
		source := fmt.Sprintf("%v", h["source"])
		hID := fmt.Sprintf("%v", h["id"])

		if len(changes) > 60 {
			changes = changes[:60] + "..."
		}
		if changes == "<nil>" || changes == "" {
			changes = `<span class="text-muted">(new)</span>`
		}

		actionBadge := "muted"
		if action == "create" {
			actionBadge = "green"
		} else if action == "update" {
			actionBadge = "blue"
		} else if action == "delete" {
			actionBadge = "red"
		}

		html.WriteString(fmt.Sprintf(`<tr><td class="text-muted" style="white-space:nowrap;font-size:11px">%s</td><td class="fw-500">%s</td><td><span class="badge %s">%s</span></td><td style="font-size:11px"><code>%s</code></td><td><span class="badge muted">%s</span></td><td><button onclick="rollback('%s')" class="btn-sm" style="cursor:pointer;font-size:11px">Rollback</button></td></tr>`,
			createdAt, entityName, actionBadge, action, changes, source, hID))
	}
	if len(history) == 0 {
		html.WriteString(`<tr><td colspan="6" class="empty-state">No security changes recorded</td></tr>`)
	}
	html.WriteString(`</tbody></table></div>`)

	html.WriteString(`<script>
function loadFromFiles(){
var s=document.getElementById('sync-status');s.textContent='Loading...';
fetch('/admin/api/securities/load',{method:'POST'})
.then(r=>r.json()).then(d=>{s.textContent=d.message||'Done';location.reload()})
.catch(e=>{s.textContent='Error: '+e})
}
function exportToFiles(){
var s=document.getElementById('sync-status');s.textContent='Exporting...';
fetch('/admin/api/securities/export',{method:'POST'})
.then(r=>r.json()).then(d=>{s.textContent=d.message||'Done'})
.catch(e=>{s.textContent='Error: '+e})
}
function uploadFile(input){
if(!input.files[0])return;
var s=document.getElementById('sync-status');s.textContent='Uploading...';
var fd=new FormData();fd.append('file',input.files[0]);
fetch('/admin/api/securities/upload',{method:'POST',body:fd})
.then(r=>r.json()).then(d=>{s.textContent=d.message||'Done';location.reload()})
.catch(e=>{s.textContent='Error: '+e})
}
function rollback(id){
if(!confirm('Rollback this change?'))return;
fetch('/admin/api/securities/rollback/'+id,{method:'POST'})
.then(r=>r.json()).then(d=>{alert(d.message||'Done');location.reload()})
.catch(e=>{alert('Error: '+e)})
}
</script>`)

	html.WriteString(pageFooter())

	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.SendString(html.String())
}

func (a *AdminPanel) apiSecurityLoad(c *fiber.Ctx) error {
	moduleName := c.Query("module", "")
	loader := module.NewSecurityLoader(a.db)

	modules := a.moduleRegistry.List()
	loaded := 0
	for _, m := range modules {
		if moduleName != "" && m.Definition.Name != moduleName {
			continue
		}
		secDir := m.Path + "/securities"
		if err := loader.LoadFromDirectory(secDir, m.Definition.Name); err == nil {
			loaded++
		}
	}

	return c.JSON(fiber.Map{"ok": true, "message": fmt.Sprintf("Loaded securities from %d modules", loaded)})
}

func (a *AdminPanel) apiSecurityExport(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"ok": true, "message": "Export to files: not yet implemented"})
}

func (a *AdminPanel) apiSecurityDownload(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"ok": false, "message": "Download ZIP: not yet implemented"})
}

func (a *AdminPanel) apiSecurityUpload(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"ok": false, "message": "Upload: not yet implemented"})
}

func (a *AdminPanel) apiSecurityDiff(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"ok": false, "message": "Diff: not yet implemented"})
}

func (a *AdminPanel) apiSecurityHistory(c *fiber.Ctx) error {
	var history []map[string]any
	a.db.Table("ir_security_histories").Order("created_at DESC").Limit(100).Find(&history)
	return c.JSON(fiber.Map{"data": history})
}

func (a *AdminPanel) apiSecurityRollback(c *fiber.Ctx) error {
	historyID := c.Params("id")

	var entry map[string]any
	if err := a.db.Table("ir_security_histories").Where("id = ?", historyID).First(&entry).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "history entry not found"})
	}

	return c.JSON(fiber.Map{"ok": true, "message": fmt.Sprintf("Rollback for %v: not yet fully implemented", entry["entity_name"])})
}
