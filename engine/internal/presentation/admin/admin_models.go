package admin

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/bitcode-framework/bitcode/internal/compiler/parser"
	"github.com/bitcode-framework/bitcode/internal/infrastructure/persistence"
)

func (a *AdminPanel) listModels(c *fiber.Ctx) error {
	filterModule := c.Query("module")
	grouped := a.modelsByModule()

	var html strings.Builder
	html.WriteString(a.pageHeader("Models", "models"))

	allModels := a.modelRegistry.List()
	html.WriteString(`<div class="list-toolbar"><div class="list-filters">`)
	activeAll := ""
	if filterModule == "" {
		activeAll = " active"
	}
	html.WriteString(fmt.Sprintf(`<a href="/admin/models" class="filter-pill%s">All</a>`, activeAll))
	for _, mod := range grouped.order {
		active := ""
		if filterModule == mod {
			active = " active"
		}
		html.WriteString(fmt.Sprintf(`<a href="/admin/models?module=%s" class="filter-pill%s">%s</a>`, mod, active, mod))
	}
	count := 0
	for _, m := range allModels {
		if filterModule == "" || m.Module == filterModule {
			count++
		}
	}
	html.WriteString(fmt.Sprintf(`</div><div class="list-count text-muted">%d of %d</div></div>`, count, len(allModels)))

	html.WriteString(`<div class="card"><table><thead><tr><th>Name</th><th>Module</th><th>Label</th><th>Fields</th><th>Inherit</th></tr></thead><tbody>`)
	for _, mod := range grouped.order {
		if filterModule != "" && mod != filterModule {
			continue
		}
		for _, m := range grouped.models[mod] {
			label := m.Label
			if label == "" {
				label = `<span class="text-muted">&mdash;</span>`
			}
			inherit := `<span class="text-muted">&mdash;</span>`
			if m.Inherit != "" {
				inherit = fmt.Sprintf(`<a href="/admin/models/%s/%s">%s</a>`, m.Module, m.Inherit, m.Inherit)
			}
			moduleName := m.Module
			if moduleName == "" {
				moduleName = "base"
			}
			html.WriteString(fmt.Sprintf(`<tr><td><a href="/admin/models/%s/%s" class="fw-500">%s</a></td><td><span class="badge muted">%s</span></td><td>%s</td><td>%d</td><td>%s</td></tr>`,
				moduleName, m.Name, m.Name, moduleName, label, len(m.Fields), inherit))
		}
	}
	if len(allModels) == 0 {
		html.WriteString(`<tr><td colspan="5" class="empty-state">No models registered. Load modules first.</td></tr>`)
	}
	html.WriteString(`</tbody></table></div>`)
	html.WriteString(pageFooter())

	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.SendString(html.String())
}

func (a *AdminPanel) viewModelLegacy(c *fiber.Ctx) error {
	name := c.Params("name")
	model, err := a.modelRegistry.Get(name)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "model not found"})
	}
	moduleName := model.Module
	if moduleName == "" {
		moduleName = "base"
	}
	return c.Redirect(fmt.Sprintf("/admin/models/%s/%s?%s", moduleName, name, c.Request().URI().QueryString()), 301)
}

func (a *AdminPanel) viewModel(c *fiber.Ctx) error {
	name := c.Params("name")
	tab := c.Query("tab", "form")
	model, err := a.modelRegistry.Get(name)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "model not found"})
	}

	var html strings.Builder
	html.WriteString(a.modelPageHeader(model.Name, model.Module, tab))

	switch tab {
	case "fields":
		a.renderFieldsTab(&html, model)
	case "connections":
		a.renderConnectionsTab(&html, model)
	case "schema":
		a.renderSchemaTab(&html, model)
	case "api":
		a.renderAPITab(&html, model)
	default:
		a.renderFormTab(&html, model)
	}

	html.WriteString(pageFooter())

	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.SendString(html.String())
}

func (a *AdminPanel) renderFormTab(html *strings.Builder, model *parser.ModelDefinition) {
	html.WriteString(`<div class="card"><div class="card-title">Form Preview</div><div class="form-preview">`)
	sortedFields := sortedFieldNames(model.Fields)
	for _, fieldName := range sortedFields {
		field := model.Fields[fieldName]
		label := field.Label
		if label == "" {
			label = fieldName
		}
		reqStar := ""
		if field.Required {
			reqStar = `<span class="req-star">*</span>`
		}
		html.WriteString(fmt.Sprintf(`<div class="form-group"><label>%s%s</label>%s</div>`, label, reqStar, renderFormInput(field)))
	}
	html.WriteString(`</div></div>`)
}

func (a *AdminPanel) renderFieldsTab(html *strings.Builder, model *parser.ModelDefinition) {
	sortedFields := sortedFieldNames(model.Fields)

	html.WriteString(`<div class="card"><div class="card-title">Fields</div><table><thead><tr><th style="width:40px">No.</th><th>Name</th><th>Type</th><th>Label</th><th>Required</th><th>Unique</th><th>Default</th><th>Relation</th><th>Mask</th><th>Groups</th></tr></thead><tbody>`)
	for i, fieldName := range sortedFields {
		field := model.Fields[fieldName]
		req := `<span class="text-muted">&mdash;</span>`
		if field.Required {
			req = `<span class="dot green-dot"></span>`
		}
		uniq := `<span class="text-muted">&mdash;</span>`
		if field.Unique {
			uniq = `<span class="dot blue-dot"></span>`
		}
		defVal := fmt.Sprintf("%v", field.Default)
		if defVal == "" || defVal == "<nil>" {
			defVal = `<span class="text-muted">&mdash;</span>`
		}
		rel := `<span class="text-muted">&mdash;</span>`
		if field.Model != "" {
			moduleName := model.Module
			if moduleName == "" {
				moduleName = "base"
			}
			rel = fmt.Sprintf(`<a href="/admin/models/%s/%s">%s</a>`, moduleName, field.Model, field.Model)
		}
		label := field.Label
		if label == "" {
			label = `<span class="text-muted">&mdash;</span>`
		}
		maskStr := `<span class="text-muted">&mdash;</span>`
		if field.Mask {
			maskLen := field.MaskLength
			if maskLen <= 0 {
				maskLen = 4
			}
			maskStr = fmt.Sprintf(`<span class="badge green">✔ /%d</span>`, maskLen)
		}
		groupsStr := `<span class="text-muted">&mdash;</span>`
		if len(field.Groups) > 0 {
			groupsStr = fmt.Sprintf(`<code>%s</code>`, strings.Join(field.Groups, ", "))
		}
		html.WriteString(fmt.Sprintf(`<tr><td class="text-muted">%d</td><td class="fw-500">%s</td><td><code>%s</code></td><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td></tr>`,
			i+1, fieldName, field.Type, label, req, uniq, defVal, rel, maskStr, groupsStr))
	}
	html.WriteString(`</tbody></table></div>`)

	if len(model.RecordRules) > 0 {
		html.WriteString(`<div class="card"><div class="card-title">Record Rules</div><table><thead><tr><th>Groups</th><th>Domain</th></tr></thead><tbody>`)
		for _, rule := range model.RecordRules {
			groups := strings.Join(rule.Groups, ", ")
			html.WriteString(fmt.Sprintf(`<tr><td>%s</td><td><code>%v</code></td></tr>`, groups, rule.Domain))
		}
		html.WriteString(`</tbody></table></div>`)
	}

	if len(model.Indexes) > 0 {
		html.WriteString(`<div class="card"><div class="card-title">Indexes</div><table><thead><tr><th>Fields</th></tr></thead><tbody>`)
		for _, idx := range model.Indexes {
			html.WriteString(fmt.Sprintf(`<tr><td><code>%s</code></td></tr>`, strings.Join(idx, ", ")))
		}
		html.WriteString(`</tbody></table></div>`)
	}
}

func (a *AdminPanel) renderConnectionsTab(html *strings.Builder, model *parser.ModelDefinition) {
	var m2o, o2m, m2m []struct{ name, model, inverse string }
	for name, field := range model.Fields {
		switch field.Type {
		case parser.FieldMany2One:
			m2o = append(m2o, struct{ name, model, inverse string }{name, field.Model, ""})
		case parser.FieldOne2Many:
			o2m = append(o2m, struct{ name, model, inverse string }{name, field.Model, field.Inverse})
		case parser.FieldMany2Many:
			m2m = append(m2m, struct{ name, model, inverse string }{name, field.Model, ""})
		}
	}

	html.WriteString(`<div class="conn-grid">`)
	if len(m2o) > 0 {
		html.WriteString(`<div class="card"><div class="card-title">Many-to-One</div><div class="conn-list">`)
		for _, r := range m2o {
			moduleName := model.Module
			if moduleName == "" {
				moduleName = "base"
			}
			html.WriteString(fmt.Sprintf(`<a href="/admin/models/%s/%s" class="conn-item"><div class="conn-model">%s</div><div class="conn-detail">field: <code>%s</code></div></a>`, moduleName, r.model, r.model, r.name))
		}
		html.WriteString(`</div></div>`)
	}
	if len(o2m) > 0 {
		html.WriteString(`<div class="card"><div class="card-title">One-to-Many</div><div class="conn-list">`)
		for _, r := range o2m {
			moduleName := model.Module
			if moduleName == "" {
				moduleName = "base"
			}
			html.WriteString(fmt.Sprintf(`<a href="/admin/models/%s/%s" class="conn-item"><div class="conn-model">%s</div><div class="conn-detail">field: <code>%s</code> inverse: <code>%s</code></div></a>`, moduleName, r.model, r.model, r.name, r.inverse))
		}
		html.WriteString(`</div></div>`)
	}
	if len(m2m) > 0 {
		html.WriteString(`<div class="card"><div class="card-title">Many-to-Many</div><div class="conn-list">`)
		for _, r := range m2m {
			moduleName := model.Module
			if moduleName == "" {
				moduleName = "base"
			}
			html.WriteString(fmt.Sprintf(`<a href="/admin/models/%s/%s" class="conn-item"><div class="conn-model">%s</div><div class="conn-detail">field: <code>%s</code></div></a>`, moduleName, r.model, r.model, r.name))
		}
		html.WriteString(`</div></div>`)
	}
	if len(m2o) == 0 && len(o2m) == 0 && len(m2m) == 0 {
		html.WriteString(`<div class="card"><div class="empty-state">No relationships defined.</div></div>`)
	}
	html.WriteString(`</div>`)
}

func (a *AdminPanel) renderSchemaTab(html *strings.Builder, model *parser.ModelDefinition) {
	filePath := a.findModelFile(model)
	if filePath == "" {
		html.WriteString(`<div class="card"><div class="empty-state">Model file not found on disk.</div></div>`)
		return
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		html.WriteString(fmt.Sprintf(`<div class="card"><div class="empty-state">Error reading file: %s</div></div>`, err.Error()))
		return
	}

	var pretty map[string]any
	json.Unmarshal(data, &pretty)
	prettyJSON, _ := json.MarshalIndent(pretty, "", "  ")

	html.WriteString(`<div class="card"><div class="card-title">JSON Schema</div><div style="padding:12px 16px">`)
	html.WriteString(fmt.Sprintf(`<textarea id="schema-editor" style="width:100%%;min-height:500px;font-family:monospace;font-size:12px;padding:8px;border:1px solid var(--border);border-radius:var(--radius);resize:vertical;tab-size:2;background:#fff;color:var(--text)">%s</textarea>`, string(prettyJSON)))
	html.WriteString(`</div></div>`)
	html.WriteString(fmt.Sprintf(`<div style="margin-top:8px;display:flex;gap:8px;align-items:center"><button onclick="saveModel()" class="btn-sm" style="cursor:pointer">Save</button><span id="save-status" class="text-muted"></span></div>`))
	html.WriteString(fmt.Sprintf(`<script>
function saveModel(){
var s=document.getElementById('save-status');s.textContent='Saving...';
var content=document.getElementById('schema-editor').value;
try{JSON.parse(content)}catch(e){s.textContent='Invalid JSON: '+e.message;s.style.color='var(--red)';return}
fetch('/admin/api/models/%s',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({content:content})})
.then(r=>r.json()).then(d=>{if(d.ok){s.textContent='Saved';s.style.color='var(--green)'}else{s.textContent='Error: '+d.error;s.style.color='var(--red)'}})
.catch(e=>{s.textContent='Error: '+e;s.style.color='var(--red)'})
}
</script>`, model.Name))
}

func (a *AdminPanel) renderAPITab(html *strings.Builder, model *parser.ModelDefinition) {
	html.WriteString(`<div class="card"><div class="card-title">API Configuration</div><table>`)

	apiEnabled := model.API != nil
	autoCrud := apiEnabled && model.API.AutoCRUD
	auth := apiEnabled && model.API.Auth
	autoPages := apiEnabled && model.API.IsAutoPages()
	modal := apiEnabled && model.API.Modal
	restEnabled := !apiEnabled || model.API.Protocols.REST
	graphqlEnabled := apiEnabled && model.API.Protocols.GraphQL
	wsEnabled := apiEnabled && model.API.Protocols.WebSocket

	check := func(v bool) string {
		if v {
			return `<span class="badge green">✔</span>`
		}
		return `<span class="text-muted">✖</span>`
	}

	html.WriteString(fmt.Sprintf(`<tr><td class="fw-500">API Enabled</td><td>%s</td></tr>`, check(apiEnabled)))
	html.WriteString(fmt.Sprintf(`<tr><td class="fw-500">Auto CRUD</td><td>%s</td></tr>`, check(autoCrud)))
	html.WriteString(fmt.Sprintf(`<tr><td class="fw-500">Auth Required</td><td>%s</td></tr>`, check(auth)))
	html.WriteString(fmt.Sprintf(`<tr><td class="fw-500">Auto Pages</td><td>%s</td></tr>`, check(autoPages)))
	html.WriteString(fmt.Sprintf(`<tr><td class="fw-500">Modal Mode</td><td>%s</td></tr>`, check(modal)))
	html.WriteString(`</table></div>`)

	html.WriteString(`<div class="card"><div class="card-title">Protocols</div><table>`)
	html.WriteString(fmt.Sprintf(`<tr><td class="fw-500">REST</td><td>%s</td></tr>`, check(restEnabled)))
	html.WriteString(fmt.Sprintf(`<tr><td class="fw-500">GraphQL</td><td>%s</td></tr>`, check(graphqlEnabled)))
	html.WriteString(fmt.Sprintf(`<tr><td class="fw-500">WebSocket</td><td>%s</td></tr>`, check(wsEnabled)))
	html.WriteString(`</table></div>`)

	if apiEnabled && autoCrud {
		moduleName := model.Module
		if moduleName == "" {
			moduleName = "base"
		}
		plural := model.Name + "s"

		html.WriteString(`<div class="card"><div class="card-title">Generated Endpoints</div><table><thead><tr><th>Method</th><th>Path</th><th>Action</th></tr></thead><tbody>`)
		endpoints := []struct{ method, path, action string }{
			{"GET", fmt.Sprintf("/api/v1/%s/%s", moduleName, plural), "list"},
			{"GET", fmt.Sprintf("/api/v1/%s/%s/:id", moduleName, plural), "read"},
			{"POST", fmt.Sprintf("/api/v1/%s/%s", moduleName, plural), "create"},
			{"PUT", fmt.Sprintf("/api/v1/%s/%s/:id", moduleName, plural), "update"},
			{"DELETE", fmt.Sprintf("/api/v1/%s/%s/:id", moduleName, plural), "delete"},
			{"POST", fmt.Sprintf("/api/v1/%s/%s/:id/clone", moduleName, plural), "clone"},
			{"POST", fmt.Sprintf("/api/v1/%s/%s/onchange", moduleName, plural), "onchange"},
		}
		methodColors := map[string]string{"GET": "blue", "POST": "green", "PUT": "yellow", "DELETE": "red"}
		for _, ep := range endpoints {
			color := methodColors[ep.method]
			if color == "" {
				color = "muted"
			}
			html.WriteString(fmt.Sprintf(`<tr><td><span class="badge %s">%s</span></td><td><code>%s</code></td><td>%s</td></tr>`, color, ep.method, ep.path, ep.action))
		}
		html.WriteString(`</tbody></table></div>`)

		if autoPages {
			html.WriteString(`<div class="card"><div class="card-title">Generated Pages</div><table><thead><tr><th>URL</th><th>Type</th></tr></thead><tbody>`)
			pages := []struct{ url, typ string }{
				{fmt.Sprintf("/%s/%s", moduleName, plural), "list"},
				{fmt.Sprintf("/%s/%s/new", moduleName, plural), "create"},
				{fmt.Sprintf("/%s/%s/:id", moduleName, plural), "detail"},
				{fmt.Sprintf("/%s/%s/:id/edit", moduleName, plural), "edit"},
			}
			if modal {
				pages = []struct{ url, typ string }{
					{fmt.Sprintf("/%s/%s", moduleName, plural), "list (with modal CRUD)"},
				}
			}
			for _, p := range pages {
				html.WriteString(fmt.Sprintf(`<tr><td><code>%s</code></td><td>%s</td></tr>`, p.url, p.typ))
			}
			html.WriteString(`</tbody></table></div>`)
		}
	}

	html.WriteString(`<div class="card"><div class="card-title">Model Options</div><table>`)
	html.WriteString(fmt.Sprintf(`<tr><td class="fw-500">Timestamps</td><td>%s</td></tr>`, check(model.IsTimestamps())))
	html.WriteString(fmt.Sprintf(`<tr><td class="fw-500">Timestamps By</td><td>%s</td></tr>`, check(model.IsTimestampsBy())))
	html.WriteString(fmt.Sprintf(`<tr><td class="fw-500">Soft Deletes</td><td>%s</td></tr>`, check(model.IsSoftDeletes())))
	html.WriteString(fmt.Sprintf(`<tr><td class="fw-500">Versioning</td><td>%s</td></tr>`, check(model.IsVersion())))
	if model.TitleField != "" {
		html.WriteString(fmt.Sprintf(`<tr><td class="fw-500">Title Field</td><td><code>%s</code></td></tr>`, model.TitleField))
	}
	if len(model.SearchField) > 0 {
		html.WriteString(fmt.Sprintf(`<tr><td class="fw-500">Search Fields</td><td><code>%s</code></td></tr>`, strings.Join(model.SearchField, ", ")))
	}
	html.WriteString(`</table></div>`)
}

func (a *AdminPanel) findModelFile(model *parser.ModelDefinition) string {
	if model.ModulePath != "" {
		path := filepath.Join(model.ModulePath, "models", model.Name+".json")
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

func (a *AdminPanel) listModelData(c *fiber.Ctx) error {
	name := c.Params("name")
	model, err := a.modelRegistry.Get(name)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "model not found"})
	}

	tableName := a.modelRegistry.TableName(name)
	repo := persistence.NewGenericRepositoryWithModel(a.db, tableName, model)

	page := 1
	pageSize := 20

	results, total, err := repo.FindAll(c.Context(), nil, page, pageSize)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	sortedFields := sortedFieldNames(model.Fields)
	displayFields := sortedFields
	if len(displayFields) > 8 {
		displayFields = displayFields[:8]
	}

	moduleName := model.Module
	if moduleName == "" {
		moduleName = "base"
	}

	var html strings.Builder
	html.WriteString(a.modelPageHeader(model.Name, moduleName, "data"))

	html.WriteString(fmt.Sprintf(`<div class="card"><div class="card-title">Data (%d records)</div>`, total))
	html.WriteString(`<table><thead><tr><th>ID</th>`)
	for _, f := range displayFields {
		html.WriteString(fmt.Sprintf(`<th>%s</th>`, f))
	}
	html.WriteString(`</tr></thead><tbody>`)

	for _, row := range results {
		id := fmt.Sprintf("%v", row["id"])
		html.WriteString(fmt.Sprintf(`<tr><td class="fw-500" style="font-size:11px"><code>%s</code></td>`, id))
		for _, f := range displayFields {
			val := fmt.Sprintf("%v", row[f])
			if len(val) > 50 {
				val = val[:50] + "..."
			}
			if val == "<nil>" {
				val = `<span class="text-muted">&mdash;</span>`
			}
			html.WriteString(fmt.Sprintf(`<td>%s</td>`, val))
		}
		html.WriteString(`</tr>`)
	}
	if len(results) == 0 {
		html.WriteString(fmt.Sprintf(`<tr><td colspan="%d" class="empty-state">No data</td></tr>`, len(displayFields)+1))
	}
	html.WriteString(`</tbody></table></div>`)
	html.WriteString(pageFooter())

	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.SendString(html.String())
}

func (a *AdminPanel) modelPageHeader(modelName, moduleName, activeTab string) string {
	if moduleName == "" {
		moduleName = "base"
	}
	breadcrumb := fmt.Sprintf(`<div class="breadcrumb"><a href="/admin">Admin</a> <span class="sep">/</span> <a href="/admin/models">Models</a> <span class="sep">/</span> <span class="badge muted">%s</span> <span class="sep">/</span> <span class="fw-500">%s</span></div>`, moduleName, modelName)

	urlBase := fmt.Sprintf("/admin/models/%s/%s", moduleName, modelName)
	tabs := fmt.Sprintf(`<div class="tabs"><a href="%s?tab=form" class="tab%s">Form</a><a href="%s?tab=fields" class="tab%s">Fields</a><a href="%s?tab=connections" class="tab%s">Connections</a><a href="%s?tab=schema" class="tab%s">Schema</a><a href="%s?tab=api" class="tab%s">API</a></div>`,
		urlBase, activeClass(activeTab, "form"),
		urlBase, activeClass(activeTab, "fields"),
		urlBase, activeClass(activeTab, "connections"),
		urlBase, activeClass(activeTab, "schema"),
		urlBase, activeClass(activeTab, "api"))

	meta := fmt.Sprintf(`<div class="model-header"><div class="model-name">%s</div><div class="model-meta"><span class="badge muted">%s</span></div></div>`, modelName, moduleName)

	return fmt.Sprintf(`<!DOCTYPE html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>%s - BitCode Admin</title>
%s</head><body>
<div class="layout">%s<div class="main"><div class="topbar">%s</div><div class="content">%s%s`,
		modelName, cssBlock(), a.sidebarHTML("models"), breadcrumb, meta, tabs)
}

type groupedModels struct {
	order  []string
	models map[string][]*parser.ModelDefinition
}

func (a *AdminPanel) modelsByModule() groupedModels {
	models := a.modelRegistry.List()
	grouped := make(map[string][]*parser.ModelDefinition)
	for _, m := range models {
		grouped[m.Module] = append(grouped[m.Module], m)
	}
	order := make([]string, 0, len(grouped))
	for k := range grouped {
		order = append(order, k)
	}
	sort.Strings(order)
	for _, k := range order {
		sort.Slice(grouped[k], func(i, j int) bool { return grouped[k][i].Name < grouped[k][j].Name })
	}
	return groupedModels{order: order, models: grouped}
}
