package admin

import (
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
)

func (a *AdminPanel) listViews(c *fiber.Ctx) error {
	var html strings.Builder
	html.WriteString(a.pageHeader("Views", "views"))

	columns := []map[string]any{
		{"field": "name", "label": "Name", "sortable": true, "filterable": true},
		{"field": "type", "label": "Type", "sortable": true, "filterable": true},
		{"field": "model", "label": "Model", "sortable": true, "filterable": true},
		{"field": "title", "label": "Title", "sortable": true},
		{"field": "module", "label": "Module", "sortable": true, "filterable": true},
		{"field": "status", "label": "Status", "filterable": true},
	}

	html.WriteString(adminDatatable(columns, "/admin/api/list/views", map[string]string{
		"detail-url": "/admin/views/:id",
	}))
	html.WriteString(pageFooter())

	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.SendString(html.String())
}

func (a *AdminPanel) viewDetail(c *fiber.Ctx) error {
	modName := c.Params("module")
	viewName := c.Params("name")
	key := modName + "/" + viewName
	tab := c.Query("tab", "info")

	views := a.views()
	info, ok := views[key]
	if !ok {
		return c.Status(404).SendString("View not found")
	}

	breadcrumb := fmt.Sprintf(`<div class="breadcrumb"><a href="/admin">Admin</a> <span class="sep">/</span> <a href="/admin/views">Views</a> <span class="sep">/</span> <span class="fw-500">%s</span></div>`, viewName)

	editBadge := `<span class="badge green">editable</span>`
	if !info.Editable {
		editBadge = `<span class="badge muted">embedded (read-only)</span>`
	}

	tabs := fmt.Sprintf(`<div class="tabs"><a href="/admin/views/%s?tab=info" class="tab%s">Info</a><a href="/admin/views/%s?tab=preview" class="tab%s">Preview</a><a href="/admin/views/%s?tab=editor" class="tab%s">Editor</a><a href="/admin/views/%s?tab=revisions" class="tab%s">Revisions</a></div>`,
		key, activeClass(tab, "info"),
		key, activeClass(tab, "preview"),
		key, activeClass(tab, "editor"),
		key, activeClass(tab, "revisions"))

	meta := fmt.Sprintf(`<div class="model-header"><div class="model-name">%s</div><div class="model-meta"><span class="badge blue">%s</span> <span class="badge muted">%s</span> %s</div></div>`,
		viewName, info.Def.Type, modName, editBadge)

	var html strings.Builder
	html.WriteString(fmt.Sprintf(`<!DOCTYPE html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>%s - BitCode Admin</title>
%s</head><body>
<div class="layout">%s<div class="main"><div class="topbar">%s</div><div class="content">%s%s`,
		viewName, cssBlock(), a.sidebarHTML("views"), breadcrumb, meta, tabs))

	switch tab {
	case "preview":
		html.WriteString(fmt.Sprintf(`<div class="card" style="padding:0;overflow:hidden"><iframe src="/admin/api/views/%s/preview" style="width:100%%;height:500px;border:none"></iframe></div>`, key))
		html.WriteString(fmt.Sprintf(`<div style="margin-top:8px"><a href="/app/%s" target="_blank" class="btn-sm">Open Full View &rarr;</a></div>`, key))

	case "editor":
		jsonContent := ""
		if info.FilePath != "" {
			data, err := os.ReadFile(info.FilePath)
			if err == nil {
				jsonContent = string(data)
			}
		}
		readonlyAttr := ""
		saveDisabled := ""
		readonlyJS := "false"
		if !info.Editable {
			readonlyAttr = " readonly"
			saveDisabled = " disabled"
			readonlyJS = "true"
		}

		modelFieldsJSON := "[]"
		if info.Def.Model != "" {
			if m, err := a.modelRegistry.Get(info.Def.Model); err == nil {
				var mf []map[string]any
				for name, f := range m.Fields {
					mf = append(mf, map[string]any{"name": name, "type": string(f.Type), "label": f.Label, "required": f.Required})
				}
				if b, err := json.Marshal(mf); err == nil {
					modelFieldsJSON = string(b)
				}
			}
		}

		html.WriteString(`<div style="margin-bottom:8px;display:flex;gap:4px">`)
		html.WriteString(`<button onclick="setMode('visual')" class="filter-pill active" id="btn-visual">Visual</button>`)
		html.WriteString(`<button onclick="setMode('json')" class="filter-pill" id="btn-json">JSON</button>`)
		html.WriteString(`<button onclick="setMode('split')" class="filter-pill" id="btn-split">Split</button>`)
		if !info.Editable {
			html.WriteString(`<span class="text-muted" style="margin-left:auto;font-size:12px;align-self:center">Read-only (embedded)</span>`)
		}
		html.WriteString(`</div>`)

		html.WriteString(`<div id="panel-visual" class="card" style="margin-bottom:12px">`)
		html.WriteString(fmt.Sprintf(`<bc-view-editor id="ve" view-json="%s" model-fields="%s" readonly="%s"></bc-view-editor>`,
			template.HTMLEscapeString(jsonContent),
			template.HTMLEscapeString(modelFieldsJSON),
			readonlyJS))
		html.WriteString(`</div>`)

		html.WriteString(`<div id="panel-json" class="card" style="display:none"><div class="card-title">JSON</div><div style="padding:12px 16px">`)
		bgColor := "#fff"
		if !info.Editable {
			bgColor = "#f8f9fa"
		}
		html.WriteString(fmt.Sprintf(`<textarea id="json-editor" style="width:100%%;min-height:400px;font-family:monospace;font-size:12px;padding:8px;border:1px solid var(--border);border-radius:var(--radius);resize:vertical;tab-size:2;background:%s;color:var(--text)"%s>%s</textarea>`,
			bgColor, readonlyAttr, template.HTMLEscapeString(jsonContent)))
		html.WriteString(`</div></div>`)

		html.WriteString(fmt.Sprintf(`<div style="margin-top:8px;display:flex;gap:8px;align-items:center"><button onclick="saveView()" class="btn-sm" style="cursor:pointer"%s>Save</button><span id="save-status" class="text-muted"></span>`, saveDisabled))
		if !info.Editable {
			html.WriteString(`<button onclick="publishView()" class="btn-sm" style="cursor:pointer;margin-left:auto">Publish to Edit</button>`)
		}
		html.WriteString(`</div>`)

		html.WriteString(fmt.Sprintf(`<script type="module" src="/assets/components/bc-components.esm.js"></script>
<script>
var mode='visual';
function setMode(m){
mode=m;
document.getElementById('panel-visual').style.display=(m==='visual'||m==='split')?'block':'none';
document.getElementById('panel-json').style.display=(m==='json'||m==='split')?'block':'none';
document.querySelectorAll('.filter-pill').forEach(function(b){b.classList.remove('active')});
document.getElementById('btn-'+m).classList.add('active');
}
document.addEventListener('viewChanged',function(e){
var ta=document.getElementById('json-editor');
if(ta&&e.detail&&e.detail.json)ta.value=e.detail.json;
});
document.getElementById('json-editor').addEventListener('input',function(){
var ve=document.getElementById('ve');
if(ve)ve.setAttribute('view-json',this.value);
});
function saveView(){
var s=document.getElementById('save-status');s.textContent='Saving...';
var content=document.getElementById('json-editor').value;
fetch('/admin/api/views/%s',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({content:content})})
.then(function(r){return r.json()}).then(function(d){if(d.ok){s.textContent='Saved (v'+d.version+')';s.style.color='var(--green)'}else{s.textContent='Error: '+d.error;s.style.color='var(--red)'}})
.catch(function(e){s.textContent='Error: '+e;s.style.color='var(--red)'})
}
function publishView(){
fetch('/admin/api/views/%s/publish',{method:'POST'})
.then(function(r){return r.json()}).then(function(d){if(d.ok){alert(d.message);location.reload()}else{alert('Error: '+d.error)}})
}
</script>`, key, key))

	case "revisions":
		revisions, _ := a.revisionRepo.ListByViewKey(key, 50)
		html.WriteString(`<div class="card"><div class="card-title">Revision History</div>`)
		if len(revisions) == 0 {
			html.WriteString(`<div class="empty-state">No revisions yet. Save from the editor to create the first revision.</div>`)
		} else {
			html.WriteString(`<table><thead><tr><th>Version</th><th>Created At</th><th>Created By</th><th></th></tr></thead><tbody>`)
			for _, r := range revisions {
				rollbackBtn := ""
				if info.Editable {
					rollbackBtn = fmt.Sprintf(`<button onclick="rollback(%d)" class="btn-sm" style="cursor:pointer">Rollback</button>`, r.Version)
				}
				html.WriteString(fmt.Sprintf(`<tr><td class="fw-500">v%d</td><td class="text-muted">%s</td><td>%s</td><td>%s</td></tr>`,
					r.Version, r.CreatedAt.Format("2006-01-02 15:04:05"), r.CreatedBy, rollbackBtn))
			}
			html.WriteString(`</tbody></table>`)
		}
		html.WriteString(`</div>`)
		html.WriteString(fmt.Sprintf(`<script>
function rollback(v){
if(!confirm('Rollback to version '+v+'?'))return;
fetch('/admin/api/views/%s/rollback/'+v,{method:'POST'})
.then(r=>r.json()).then(d=>{if(d.ok){alert('Restored from v'+d.restored_from+' as v'+d.version);location.reload()}else{alert('Error: '+d.error)}})
}
</script>`, key))

	default:
		html.WriteString(`<div class="conn-grid">`)
		html.WriteString(`<div class="card"><div class="card-title">View Info</div><table>`)
		html.WriteString(fmt.Sprintf(`<tr><td class="fw-500">Name</td><td>%s</td></tr>`, info.Def.Name))
		html.WriteString(fmt.Sprintf(`<tr><td class="fw-500">Type</td><td><span class="badge blue">%s</span></td></tr>`, info.Def.Type))
		html.WriteString(fmt.Sprintf(`<tr><td class="fw-500">Model</td><td>%s</td></tr>`, info.Def.Model))
		html.WriteString(fmt.Sprintf(`<tr><td class="fw-500">Title</td><td>%s</td></tr>`, info.Def.Title))
		html.WriteString(fmt.Sprintf(`<tr><td class="fw-500">Module</td><td>%s</td></tr>`, info.Module))
		html.WriteString(fmt.Sprintf(`<tr><td class="fw-500">Editable</td><td>%s</td></tr>`, editBadge))
		if info.FilePath != "" {
			html.WriteString(fmt.Sprintf(`<tr><td class="fw-500">File</td><td><code>%s</code></td></tr>`, info.FilePath))
		}
		html.WriteString(`</table></div>`)

		html.WriteString(`<div class="card"><div class="card-title">Quick Actions</div><div class="conn-list">`)
		html.WriteString(fmt.Sprintf(`<a href="/app/%s" target="_blank" class="conn-item"><div class="conn-model">Open View</div><div class="conn-detail">Open in app (full layout)</div></a>`, key))
		html.WriteString(fmt.Sprintf(`<a href="/admin/views/%s?tab=editor" class="conn-item"><div class="conn-model">Edit JSON</div><div class="conn-detail">Open JSON editor</div></a>`, key))
		if info.Def.Model != "" {
			html.WriteString(fmt.Sprintf(`<a href="/admin/models/%s" class="conn-item"><div class="conn-model">View Model</div><div class="conn-detail">%s model definition</div></a>`, info.Def.Model, info.Def.Model))
		}
		revCount := a.revisionRepo.Count(key)
		html.WriteString(fmt.Sprintf(`<a href="/admin/views/%s?tab=revisions" class="conn-item"><div class="conn-model">Revisions</div><div class="conn-detail">%d revision(s)</div></a>`, key, revCount))
		html.WriteString(`</div></div>`)
		html.WriteString(`</div>`)
	}

	html.WriteString(pageFooter())

	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.SendString(html.String())
}
