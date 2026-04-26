package admin

import (
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/bitcode-framework/bitcode/internal/compiler/parser"
	domainModel "github.com/bitcode-framework/bitcode/internal/domain/model"
	"github.com/bitcode-framework/bitcode/internal/infrastructure/module"
	"github.com/bitcode-framework/bitcode/internal/infrastructure/persistence"
	"github.com/bitcode-framework/bitcode/pkg/security"
	"gorm.io/gorm"
)

type HealthInfo struct {
	Version    string
	DBDriver   string
	CacheDriver string
	Processes  []string
}

type ViewInfo struct {
	Def      *parser.ViewDefinition
	Module   string
	FilePath string
	Editable bool
}

type ViewResolver func() map[string]*ViewInfo

type AdminPanel struct {
	db               *gorm.DB
	modelRegistry    *domainModel.Registry
	moduleRegistry   *module.Registry
	viewResolver     ViewResolver
	health           HealthInfo
	revisionRepo     *persistence.ViewRevisionRepository
	dataRevisionRepo *persistence.DataRevisionRepository
	auditLogRepo     *persistence.AuditLogRepository
	moduleDir        string
	jwtConfig        security.JWTConfig
}

func NewAdminPanel(db *gorm.DB, modelReg *domainModel.Registry, moduleReg *module.Registry, viewResolver ViewResolver, health HealthInfo, moduleDir string, jwtCfg security.JWTConfig) *AdminPanel {
	return &AdminPanel{
		db:               db,
		modelRegistry:    modelReg,
		moduleRegistry:   moduleReg,
		viewResolver:     viewResolver,
		health:           health,
		revisionRepo:     persistence.NewViewRevisionRepository(db),
		dataRevisionRepo: persistence.NewDataRevisionRepository(db),
		auditLogRepo:     persistence.NewAuditLogRepository(db),
		moduleDir:        moduleDir,
		jwtConfig:        jwtCfg,
	}
}

func (a *AdminPanel) views() map[string]*ViewInfo {
	return a.viewResolver()
}

func (a *AdminPanel) RegisterRoutes(app *fiber.App) {
	admin := app.Group("/admin")

	admin.Get("/", a.dashboard)
	admin.Get("/models", a.listModels)
	admin.Get("/models/:name", a.viewModel)
	admin.Get("/models/:name/data", a.listModelData)
	admin.Get("/modules", a.listModules)
	admin.Get("/modules/:name", a.viewModule)
	admin.Get("/views", a.listViews)
	admin.Get("/views/:module/:name", a.viewDetail)
	admin.Get("/health", a.healthPage)
	admin.Get("/audit/login-history", a.loginHistoryPage)
	admin.Get("/audit/request-log", a.requestLogPage)

	api := app.Group("/admin/api")
	api.Get("/views/:module/:name", a.apiViewDetail)
	api.Get("/views/:module/:name/json", a.apiViewJSON)
	api.Post("/views/:module/:name", a.apiViewSave)
	api.Post("/views/:module/:name/rollback/:version", a.apiViewRollback)
	api.Get("/views/:module/:name/preview", a.apiViewPreview)
	api.Post("/views/:module/:name/publish", a.apiViewPublish)

	api.Get("/models/:name/json", a.apiModelJSON)
	api.Post("/models/:name", a.apiModelSave)
	api.Get("/data/:model/:id/revisions", a.apiDataRevisions)
	api.Get("/data/:model/:id/revisions/:version", a.apiDataRevisionDetail)
	api.Post("/data/:model/:id/restore/:version", a.apiDataRestore)
	api.Get("/data/:model/:id/timeline", a.apiRecordTimeline)
	api.Get("/audit/login-history", a.apiLoginHistory)
	api.Get("/audit/request-log", a.apiRequestLog)

	api.Post("/impersonate/:user_id", a.apiImpersonate)
	api.Post("/stop-impersonate", a.apiStopImpersonate)
}

func (a *AdminPanel) dashboard(c *fiber.Ctx) error {
	models := a.modelRegistry.List()
	modules := a.moduleRegistry.List()

	var html strings.Builder
	html.WriteString(a.pageHeader("Dashboard", "dashboard"))

	html.WriteString(`<div class="stats-grid">`)
	html.WriteString(statCard("Models", fmt.Sprintf("%d", len(models)), "var(--blue)"))
	html.WriteString(statCard("Modules", fmt.Sprintf("%d", len(modules)), "var(--green)"))
	html.WriteString(statCard("Views", fmt.Sprintf("%d", len(a.views())), "var(--amber)"))
	html.WriteString(`</div>`)

	html.WriteString(`<div class="card"><div class="card-title">Installed Modules</div><table><thead><tr><th>Name</th><th>Version</th><th>Category</th><th>Dependencies</th><th>Status</th></tr></thead><tbody>`)
	for _, m := range modules {
		deps := strings.Join(m.Definition.Depends, ", ")
		if deps == "" {
			deps = `<span class="text-muted">&mdash;</span>`
		}
		html.WriteString(fmt.Sprintf(`<tr><td class="fw-500">%s</td><td>%s</td><td class="text-muted">%s</td><td>%s</td><td><span class="badge green">%s</span></td></tr>`,
			m.Definition.Name, m.Definition.Version, m.Definition.Category, deps, m.State))
	}
	html.WriteString(`</tbody></table></div>`)

	grouped := a.modelsByModule()
	html.WriteString(`<div class="card"><div class="card-title">Registered Models</div><table><thead><tr><th>Name</th><th>Module</th><th>Fields</th><th></th></tr></thead><tbody>`)
	for _, modName := range grouped.order {
		for _, m := range grouped.models[modName] {
			html.WriteString(fmt.Sprintf(`<tr><td><a href="/admin/models/%s">%s</a></td><td><span class="badge muted">%s</span></td><td>%d</td><td><a href="/admin/models/%s/data" class="btn-link">View Data</a></td></tr>`,
				m.Name, m.Name, m.Module, len(m.Fields), m.Name))
		}
	}
	html.WriteString(`</tbody></table></div>`)
	html.WriteString(pageFooter())

	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.SendString(html.String())
}

func (a *AdminPanel) listModels(c *fiber.Ctx) error {
	filterModule := c.Query("module")
	models := a.modelRegistry.List()
	grouped := a.modelsByModule()

	var html strings.Builder
	html.WriteString(a.pageHeader("Models", "models"))

	html.WriteString(`<div class="list-toolbar"><div class="list-filters">`)
	activeAll := ""
	if filterModule == "" {
		activeAll = " active"
	}
	html.WriteString(fmt.Sprintf(`<a href="/admin/models" class="filter-pill%s">All</a>`, activeAll))
	for _, modName := range grouped.order {
		active := ""
		if filterModule == modName {
			active = " active"
		}
		html.WriteString(fmt.Sprintf(`<a href="/admin/models?module=%s" class="filter-pill%s">%s</a>`, modName, active, modName))
	}
	html.WriteString(`</div><div class="list-count text-muted">`)
	count := 0
	for _, m := range models {
		if filterModule == "" || m.Module == filterModule {
			count++
		}
	}
	html.WriteString(fmt.Sprintf(`%d of %d`, count, len(models)))
	html.WriteString(`</div></div>`)

	html.WriteString(`<div class="card"><table class="list-table"><thead><tr><th>Name</th><th>Module</th><th>Label</th><th>Fields</th><th>Inherit</th></tr></thead><tbody>`)
	for _, modName := range grouped.order {
		if filterModule != "" && filterModule != modName {
			continue
		}
		for _, m := range grouped.models[modName] {
			inheritVal := `<span class="text-muted">&mdash;</span>`
			if m.Inherit != "" {
				inheritVal = fmt.Sprintf(`<a href="/admin/models/%s">%s</a>`, m.Inherit, m.Inherit)
			}
			label := m.Label
			if label == "" {
				label = m.Name
			}
			html.WriteString(fmt.Sprintf(`<tr><td><a href="/admin/models/%s" class="fw-500">%s</a></td><td><span class="badge muted">%s</span></td><td class="text-muted">%s</td><td>%d</td><td>%s</td></tr>`,
				m.Name, m.Name, m.Module, label, len(m.Fields), inheritVal))
		}
	}
	html.WriteString(`</tbody></table></div>`)
	html.WriteString(pageFooter())

	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.SendString(html.String())
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
		if field.Type == parser.FieldOne2Many || field.Type == parser.FieldMany2Many || field.Type == parser.FieldComputed {
			continue
		}

		label := field.Label
		if label == "" {
			label = fieldName
		}
		reqMark := ""
		if field.Required {
			reqMark = `<span class="req-star">*</span>`
		}

		inputHTML := renderFormInput(field)
		html.WriteString(fmt.Sprintf(`<div class="form-group"><label>%s%s</label>%s</div>`, label, reqMark, inputHTML))
	}

	for _, fieldName := range sortedFields {
		field := model.Fields[fieldName]
		if field.Type == parser.FieldOne2Many {
			label := field.Label
			if label == "" {
				label = fieldName
			}
			html.WriteString(fmt.Sprintf(`<div class="form-section"><div class="form-section-title">%s <span class="text-muted fw-400">(one2many &rarr; %s)</span></div>`, label, field.Model))
			html.WriteString(`<div class="form-child-table"><div class="text-muted" style="padding:16px;text-align:center;font-size:12px">Child table rows from <code>`)
			html.WriteString(field.Model)
			html.WriteString(`</code></div></div></div>`)
		}
		if field.Type == parser.FieldMany2Many {
			label := field.Label
			if label == "" {
				label = fieldName
			}
			html.WriteString(fmt.Sprintf(`<div class="form-section"><div class="form-section-title">%s <span class="text-muted fw-400">(many2many &harr; %s)</span></div>`, label, field.Model))
			html.WriteString(`<div class="form-child-table"><div class="text-muted" style="padding:16px;text-align:center;font-size:12px">Tags / linked records from <code>`)
			html.WriteString(field.Model)
			html.WriteString(`</code></div></div></div>`)
		}
	}

	html.WriteString(`</div></div>`)
	html.WriteString(fmt.Sprintf(`<div style="margin-top:12px"><a href="/admin/models/%s/data" class="btn-sm">View Data &rarr;</a></div>`, model.Name))
}

func (a *AdminPanel) renderFieldsTab(html *strings.Builder, model *parser.ModelDefinition) {
	sortedFields := sortedFieldNames(model.Fields)

	html.WriteString(`<div class="card"><div class="card-title">Fields</div><table><thead><tr><th style="width:40px">No.</th><th>Name</th><th>Type</th><th>Label</th><th>Required</th><th>Unique</th><th>Default</th><th>Relation</th></tr></thead><tbody>`)
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
			rel = fmt.Sprintf(`<a href="/admin/models/%s">%s</a>`, field.Model, field.Model)
		}
		label := field.Label
		if label == "" {
			label = `<span class="text-muted">&mdash;</span>`
		}
		html.WriteString(fmt.Sprintf(`<tr><td class="text-muted">%d</td><td class="fw-500">%s</td><td><code>%s</code></td><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td></tr>`,
			i+1, fieldName, field.Type, label, req, uniq, defVal, rel))
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
	allModels := a.modelRegistry.List()

	type connection struct {
		Model     string
		Field     string
		FieldType string
	}

	var outgoing []connection
	for fieldName, field := range model.Fields {
		if field.Model != "" && (field.Type == parser.FieldMany2One || field.Type == parser.FieldOne2Many || field.Type == parser.FieldMany2Many) {
			outgoing = append(outgoing, connection{Model: field.Model, Field: fieldName, FieldType: string(field.Type)})
		}
	}
	sort.Slice(outgoing, func(i, j int) bool { return outgoing[i].Model < outgoing[j].Model })

	var incoming []connection
	for _, m := range allModels {
		if m.Name == model.Name {
			continue
		}
		for fieldName, field := range m.Fields {
			if field.Model == model.Name {
				incoming = append(incoming, connection{Model: m.Name, Field: fieldName, FieldType: string(field.Type)})
			}
		}
	}
	sort.Slice(incoming, func(i, j int) bool { return incoming[i].Model < incoming[j].Model })

	var relatedViews []string
	for key, v := range a.views() {
		if v.Def.Model == model.Name {
			relatedViews = append(relatedViews, key)
		}
	}
	sort.Strings(relatedViews)

	html.WriteString(`<div class="conn-grid">`)

	html.WriteString(`<div class="card"><div class="card-title">Outgoing References</div>`)
	if len(outgoing) == 0 {
		html.WriteString(`<div class="empty-state">No outgoing references</div>`)
	} else {
		html.WriteString(`<div class="conn-list">`)
		for _, c := range outgoing {
			html.WriteString(fmt.Sprintf(`<a href="/admin/models/%s" class="conn-item"><div class="conn-model">%s</div><div class="conn-detail"><span class="badge muted">%s</span> via <code>%s</code></div></a>`,
				c.Model, c.Model, c.FieldType, c.Field))
		}
		html.WriteString(`</div>`)
	}
	html.WriteString(`</div>`)

	html.WriteString(`<div class="card"><div class="card-title">Incoming References</div>`)
	if len(incoming) == 0 {
		html.WriteString(`<div class="empty-state">No incoming references</div>`)
	} else {
		html.WriteString(`<div class="conn-list">`)
		for _, c := range incoming {
			html.WriteString(fmt.Sprintf(`<a href="/admin/models/%s" class="conn-item"><div class="conn-model">%s</div><div class="conn-detail"><span class="badge muted">%s</span> via <code>%s</code></div></a>`,
				c.Model, c.Model, c.FieldType, c.Field))
		}
		html.WriteString(`</div>`)
	}
	html.WriteString(`</div>`)

	html.WriteString(`<div class="card"><div class="card-title">Related Views</div>`)
	if len(relatedViews) == 0 {
		html.WriteString(`<div class="empty-state">No views for this model</div>`)
	} else {
		html.WriteString(`<div class="conn-list">`)
		for _, key := range relatedViews {
			v := a.views()[key]
			html.WriteString(fmt.Sprintf(`<a href="/app/%s" target="_blank" class="conn-item"><div class="conn-model">%s</div><div class="conn-detail"><span class="badge blue">%s</span> %s</div></a>`,
				key, key, v.Def.Type, v.Def.Title))
		}
		html.WriteString(`</div>`)
	}
	html.WriteString(`</div>`)

	html.WriteString(`</div>`)
}

func (a *AdminPanel) renderSchemaTab(html *strings.Builder, model *parser.ModelDefinition) {
	modelJSON, _ := json.MarshalIndent(model, "", "  ")
	jsonContent := string(modelJSON)

	modelPath := a.findModelFile(model.Name)
	editable := modelPath != ""

	readonlyAttr := ""
	saveDisabled := ""
	if !editable {
		readonlyAttr = " readonly"
		saveDisabled = " disabled"
	}

	html.WriteString(`<div style="margin-bottom:8px;display:flex;gap:4px;align-items:center">`)
	html.WriteString(`<button onclick="setSchemaMode('visual')" class="filter-pill active" id="btn-visual">Visual</button>`)
	html.WriteString(`<button onclick="setSchemaMode('json')" class="filter-pill" id="btn-json">JSON</button>`)
	if !editable {
		html.WriteString(`<span class="text-muted" style="margin-left:auto;font-size:12px">Read-only (embedded)</span>`)
	}
	html.WriteString(`</div>`)

	html.WriteString(`<div id="panel-visual" class="card">`)
	html.WriteString(`<div class="card-title">Fields</div>`)
	html.WriteString(`<div style="padding:12px 16px">`)

	html.WriteString(`<table class="list-table"><thead><tr><th style="width:30px"></th><th>Name</th><th>Type</th><th>Label</th><th>Required</th><th>Options</th><th>Relation</th></tr></thead><tbody>`)

	sortedFields := sortedFieldNames(model.Fields)
	for i, fieldName := range sortedFields {
		field := model.Fields[fieldName]
		req := ""
		if field.Required {
			req = `<span class="dot green-dot"></span>`
		}
		opts := ""
		if len(field.Options) > 0 {
			opts = fmt.Sprintf(`<code>%s</code>`, strings.Join(field.Options, ", "))
		}
		rel := ""
		if field.Model != "" {
			rel = fmt.Sprintf(`<a href="/admin/models/%s">%s</a>`, field.Model, field.Model)
		}
		computed := ""
		if field.Computed != "" {
			computed = fmt.Sprintf(` <span class="badge muted">computed: %s</span>`, field.Computed)
		}
		if field.Formula != "" {
			computed = fmt.Sprintf(` <span class="badge muted">formula: %s</span>`, field.Formula)
		}
		label := field.Label
		if label == "" {
			label = `<span class="text-muted">&mdash;</span>`
		}
		html.WriteString(fmt.Sprintf(`<tr><td class="text-muted">%d</td><td class="fw-500">%s%s</td><td><code>%s</code></td><td>%s</td><td>%s</td><td>%s</td><td>%s</td></tr>`,
			i+1, fieldName, computed, field.Type, label, req, opts, rel))
	}
	html.WriteString(`</tbody></table>`)

	if model.PrimaryKey != nil {
		html.WriteString(fmt.Sprintf(`<div style="margin-top:12px;padding:8px 12px;background:var(--bg);border-radius:var(--radius);font-size:12px"><strong>Primary Key:</strong> strategy=<code>%s</code>`, model.PrimaryKey.Strategy))
		if model.PrimaryKey.Field != "" {
			html.WriteString(fmt.Sprintf(` field=<code>%s</code>`, model.PrimaryKey.Field))
		}
		if model.PrimaryKey.Format != "" {
			html.WriteString(fmt.Sprintf(` format=<code>%s</code>`, model.PrimaryKey.Format))
		}
		html.WriteString(`</div>`)
	}

	html.WriteString(`</div></div>`)

	bgColor := "#fff"
	if !editable {
		bgColor = "#f8f9fa"
	}
	html.WriteString(fmt.Sprintf(`<div id="panel-json" class="card" style="display:none"><div class="card-title">JSON Definition</div><div style="padding:12px 16px"><textarea id="schema-editor" style="width:100%%;min-height:500px;font-family:monospace;font-size:12px;padding:8px;border:1px solid var(--border);border-radius:var(--radius);resize:vertical;tab-size:2;background:%s;color:var(--text)"%s>%s</textarea></div></div>`,
		bgColor, readonlyAttr, template.HTMLEscapeString(jsonContent)))

	html.WriteString(fmt.Sprintf(`<div style="margin-top:8px;display:flex;gap:8px;align-items:center"><button onclick="saveModel()" class="btn-sm" style="cursor:pointer"%s>Save</button><span id="schema-save-status" class="text-muted"></span></div>`, saveDisabled))

	html.WriteString(fmt.Sprintf(`<script>
var schemaMode='visual';
function setSchemaMode(m){
schemaMode=m;
document.getElementById('panel-visual').style.display=(m==='visual')?'block':'none';
document.getElementById('panel-json').style.display=(m==='json')?'block':'none';
document.querySelectorAll('.filter-pill').forEach(function(b){b.classList.remove('active')});
document.getElementById('btn-'+m).classList.add('active');
}
function saveModel(){
var s=document.getElementById('schema-save-status');s.textContent='Saving...';
var content=document.getElementById('schema-editor').value;
fetch('/admin/api/models/%s',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({content:content})})
.then(function(r){return r.json()}).then(function(d){if(d.ok){s.textContent='Saved!';s.style.color='var(--green)'}else{s.textContent='Error: '+d.error;s.style.color='var(--red)'}})
.catch(function(e){s.textContent='Error: '+e;s.style.color='var(--red)'})
}
</script>`, model.Name))
}

func (a *AdminPanel) findModelFile(modelName string) string {
	modules := a.moduleRegistry.List()
	for _, m := range modules {
		modPath := filepath.Join(a.moduleDir, m.Definition.Name, "models", modelName+".json")
		if _, err := os.Stat(modPath); err == nil {
			return modPath
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

	repo := persistence.NewGenericRepository(a.db, a.modelRegistry.TableName(name))
	repo.SetTableNameResolver(a.modelRegistry)
	records, total, err := repo.FindAll(c.Context(), nil, 1, 50)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	fields := []string{"id"}
	for fieldName, field := range model.Fields {
		if field.Type != parser.FieldOne2Many && field.Type != parser.FieldMany2Many && field.Type != parser.FieldComputed {
			fields = append(fields, fieldName)
		}
	}

	var html strings.Builder
	html.WriteString(a.pageHeader(fmt.Sprintf("%s Data", model.Name), "models"))

	html.WriteString(fmt.Sprintf(`<div class="card"><div class="card-title">%s <span class="text-muted fw-400">(%d records)</span> <a href="/admin/models/%s" class="btn-link" style="float:right">Back to Model</a></div><table><thead><tr>`, model.Name, total, model.Name))
	for _, f := range fields {
		html.WriteString(fmt.Sprintf(`<th>%s</th>`, f))
	}
	html.WriteString(`<th>Actions</th>`)
	html.WriteString(`</tr></thead><tbody>`)

	if len(records) == 0 {
		html.WriteString(fmt.Sprintf(`<tr><td colspan="%d" class="empty-state">No records found</td></tr>`, len(fields)+1))
	}

	for _, rec := range records {
		html.WriteString(`<tr>`)
		for _, f := range fields {
			val := rec[f]
			if val == nil {
				val = ""
			}
			html.WriteString(fmt.Sprintf(`<td>%v</td>`, val))
		}
		recordID := ""
		if v, ok := rec["id"]; ok {
			recordID = fmt.Sprintf("%v", v)
		}
		html.WriteString(fmt.Sprintf(`<td><a href="#" onclick="showTimeline('%s','%s');return false" class="btn-link" style="font-size:11px">History</a></td>`, name, recordID))
		html.WriteString(`</tr>`)
	}
	html.WriteString(`</tbody></table></div>`)

	html.WriteString(fmt.Sprintf(`
<div id="timeline-modal" style="display:none;position:fixed;top:0;left:0;right:0;bottom:0;background:rgba(0,0,0,0.5);z-index:1000;overflow-y:auto">
<div style="max-width:700px;margin:40px auto;background:var(--card-bg);border-radius:var(--radius);box-shadow:0 8px 32px rgba(0,0,0,0.2);padding:24px">
<div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:16px">
<h3 style="margin:0" id="timeline-title">Record History</h3>
<button onclick="document.getElementById('timeline-modal').style.display='none'" style="background:none;border:none;font-size:20px;cursor:pointer;color:var(--text-muted)">&times;</button>
</div>
<div id="timeline-content">Loading...</div>
</div>
</div>
<script>
function showTimeline(model,id){
document.getElementById('timeline-modal').style.display='block';
document.getElementById('timeline-title').textContent='History: '+model+'/'+id;
document.getElementById('timeline-content').innerHTML='Loading...';
fetch('/admin/api/data/'+model+'/'+id+'/timeline')
.then(function(r){return r.json()}).then(function(d){
var h='';
if(!d.timeline||d.timeline.length===0){h='<div class="empty-state">No history</div>';}
else{
d.timeline.forEach(function(e){
var badge='blue';
if(e.action==='delete')badge='red';
else if(e.action==='create')badge='green';
else if(e.action==='restore')badge='yellow';
h+='<div style="border-left:2px solid var(--border);padding:8px 0 8px 16px;margin-left:8px;margin-bottom:4px">';
h+='<div style="display:flex;gap:8px;align-items:center"><span class="badge '+badge+'">'+e.action+'</span>';
h+='<span class="text-muted" style="font-size:11px">'+e.created_at+'</span>';
if(e.user_id)h+='<span style="font-size:11px">by '+e.user_id+'</span>';
if(e.version)h+='<span class="badge muted">v'+e.version+'</span>';
h+='</div>';
if(e.changes&&typeof e.changes==='object'){
h+='<div style="margin-top:6px;font-size:12px">';
Object.keys(e.changes).forEach(function(k){
var c=e.changes[k];
h+='<div style="padding:2px 0"><code style="color:var(--primary)">'+k+'</code>: ';
if(c&&c.old!==undefined){h+='<span style="text-decoration:line-through;color:var(--red)">'+JSON.stringify(c.old)+'</span> &rarr; <span style="color:var(--green)">'+JSON.stringify(c.new)+'</span>';}
else{h+=JSON.stringify(c);}
h+='</div>';
});
h+='</div>';
}
h+='</div>';
});
}
document.getElementById('timeline-content').innerHTML=h;
}).catch(function(e){document.getElementById('timeline-content').innerHTML='Error: '+e;});
}
</script>`))
	html.WriteString(pageFooter())

	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.SendString(html.String())
}

func (a *AdminPanel) listModules(c *fiber.Ctx) error {
	modules := a.moduleRegistry.List()

	var html strings.Builder
	html.WriteString(a.pageHeader("Modules", "modules"))

	html.WriteString(fmt.Sprintf(`<div class="list-toolbar"><div class="list-count text-muted">%d modules installed</div></div>`, len(modules)))

	html.WriteString(`<div class="card"><table><thead><tr><th>Name</th><th>Version</th><th>Label</th><th>Category</th><th>Dependencies</th><th>Status</th></tr></thead><tbody>`)
	for _, m := range modules {
		deps := strings.Join(m.Definition.Depends, ", ")
		if deps == "" {
			deps = `<span class="text-muted">&mdash;</span>`
		}
		label := m.Definition.Label
		if label == "" {
			label = m.Definition.Name
		}
		html.WriteString(fmt.Sprintf(`<tr><td><a href="/admin/modules/%s" class="fw-500">%s</a></td><td>%s</td><td class="text-muted">%s</td><td class="text-muted">%s</td><td>%s</td><td><span class="badge green">%s</span></td></tr>`,
			m.Definition.Name, m.Definition.Name, m.Definition.Version, label, m.Definition.Category, deps, m.State))
	}
	html.WriteString(`</tbody></table></div>`)
	html.WriteString(pageFooter())

	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.SendString(html.String())
}

func (a *AdminPanel) viewModule(c *fiber.Ctx) error {
	name := c.Params("name")
	tab := c.Query("tab", "overview")
	installed, err := a.moduleRegistry.Get(name)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "module not found"})
	}
	mod := installed.Definition

	breadcrumb := fmt.Sprintf(`<div class="breadcrumb"><a href="/admin">Admin</a> <span class="sep">/</span> <a href="/admin/modules">Modules</a> <span class="sep">/</span> <span class="fw-500">%s</span></div>`, mod.Name)

	tabs := fmt.Sprintf(`<div class="tabs"><a href="/admin/modules/%s?tab=overview" class="tab%s">Overview</a><a href="/admin/modules/%s?tab=permissions" class="tab%s">Permissions</a><a href="/admin/modules/%s?tab=menu" class="tab%s">Menu</a></div>`,
		mod.Name, activeClass(tab, "overview"),
		mod.Name, activeClass(tab, "permissions"),
		mod.Name, activeClass(tab, "menu"))

	meta := fmt.Sprintf(`<div class="model-header"><div class="model-name">%s</div><div class="model-meta"><span class="badge muted">v%s</span>`, mod.Name, mod.Version)
	if mod.Category != "" {
		meta += fmt.Sprintf(` <span class="text-muted">%s</span>`, mod.Category)
	}
	meta += fmt.Sprintf(` <span class="badge green">%s</span>`, installed.State)
	meta += `</div></div>`

	var html strings.Builder
	html.WriteString(fmt.Sprintf(`<!DOCTYPE html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>%s - BitCode Admin</title>
%s</head><body>
<div class="layout">%s<div class="main"><div class="topbar">%s</div><div class="content">%s%s`,
		mod.Name, cssBlock(), a.sidebarHTML("modules"), breadcrumb, meta, tabs))

	switch tab {
	case "permissions":
		a.renderModulePermissions(&html, mod)
	case "menu":
		a.renderModuleMenu(&html, mod)
	default:
		a.renderModuleOverview(&html, mod)
	}

	html.WriteString(pageFooter())

	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.SendString(html.String())
}

func (a *AdminPanel) renderModuleOverview(html *strings.Builder, mod *parser.ModuleDefinition) {
	html.WriteString(`<div class="conn-grid">`)

	html.WriteString(`<div class="card"><div class="card-title">Module Info</div><table>`)
	html.WriteString(fmt.Sprintf(`<tr><td class="fw-500">Name</td><td>%s</td></tr>`, mod.Name))
	html.WriteString(fmt.Sprintf(`<tr><td class="fw-500">Version</td><td>%s</td></tr>`, mod.Version))
	label := mod.Label
	if label == "" {
		label = mod.Name
	}
	html.WriteString(fmt.Sprintf(`<tr><td class="fw-500">Label</td><td>%s</td></tr>`, label))
	if mod.Category != "" {
		html.WriteString(fmt.Sprintf(`<tr><td class="fw-500">Category</td><td>%s</td></tr>`, mod.Category))
	}
	deps := strings.Join(mod.Depends, ", ")
	if deps == "" {
		deps = `<span class="text-muted">none</span>`
	}
	html.WriteString(fmt.Sprintf(`<tr><td class="fw-500">Dependencies</td><td>%s</td></tr>`, deps))
	html.WriteString(`</table></div>`)

	html.WriteString(`<div class="card"><div class="card-title">Resources</div><table>`)
	html.WriteString(fmt.Sprintf(`<tr><td class="fw-500">Models</td><td>%d patterns</td></tr>`, len(mod.Models)))
	html.WriteString(fmt.Sprintf(`<tr><td class="fw-500">APIs</td><td>%d patterns</td></tr>`, len(mod.APIs)))
	html.WriteString(fmt.Sprintf(`<tr><td class="fw-500">Processes</td><td>%d patterns</td></tr>`, len(mod.Processes)))
	html.WriteString(fmt.Sprintf(`<tr><td class="fw-500">Views</td><td>%d patterns</td></tr>`, len(mod.Views)))
	html.WriteString(fmt.Sprintf(`<tr><td class="fw-500">Templates</td><td>%d patterns</td></tr>`, len(mod.Templates)))
	html.WriteString(fmt.Sprintf(`<tr><td class="fw-500">Scripts</td><td>%d patterns</td></tr>`, len(mod.Scripts)))
	html.WriteString(fmt.Sprintf(`<tr><td class="fw-500">Migrations</td><td>%d patterns</td></tr>`, len(mod.Migrations)))
	html.WriteString(fmt.Sprintf(`<tr><td class="fw-500">i18n</td><td>%d patterns</td></tr>`, len(mod.I18n)))
	html.WriteString(`</table></div>`)

	html.WriteString(`</div>`)

	models := a.modelRegistry.List()
	var moduleModels []*parser.ModelDefinition
	for _, m := range models {
		if m.Module == mod.Name {
			moduleModels = append(moduleModels, m)
		}
	}
	sort.Slice(moduleModels, func(i, j int) bool { return moduleModels[i].Name < moduleModels[j].Name })

	if len(moduleModels) > 0 {
		html.WriteString(`<div class="card"><div class="card-title">Models in this Module</div><table><thead><tr><th>Name</th><th>Label</th><th>Fields</th><th>Inherit</th></tr></thead><tbody>`)
		for _, m := range moduleModels {
			inheritVal := `<span class="text-muted">&mdash;</span>`
			if m.Inherit != "" {
				inheritVal = fmt.Sprintf(`<a href="/admin/models/%s">%s</a>`, m.Inherit, m.Inherit)
			}
			mlabel := m.Label
			if mlabel == "" {
				mlabel = m.Name
			}
			html.WriteString(fmt.Sprintf(`<tr><td><a href="/admin/models/%s" class="fw-500">%s</a></td><td class="text-muted">%s</td><td>%d</td><td>%s</td></tr>`,
				m.Name, m.Name, mlabel, len(m.Fields), inheritVal))
		}
		html.WriteString(`</tbody></table></div>`)
	}

	var moduleViews []string
	for key, v := range a.views() {
		parts := strings.SplitN(key, "/", 2)
		if len(parts) == 2 && parts[0] == mod.Name {
			moduleViews = append(moduleViews, key)
		} else {
			for _, mm := range moduleModels {
				if v.Def.Model == mm.Name {
					moduleViews = append(moduleViews, key)
					break
				}
			}
		}
	}
	sort.Strings(moduleViews)
	seen := make(map[string]bool)

	if len(moduleViews) > 0 {
		html.WriteString(`<div class="card"><div class="card-title">Views</div><table><thead><tr><th>Route</th><th>Type</th><th>Model</th><th>Title</th></tr></thead><tbody>`)
		for _, key := range moduleViews {
			if seen[key] {
				continue
			}
			seen[key] = true
			v := a.views()[key]
			html.WriteString(fmt.Sprintf(`<tr><td class="fw-500">%s</td><td><span class="badge blue">%s</span></td><td>%s</td><td>%s</td></tr>`,
				key, v.Def.Type, v.Def.Model, v.Def.Title))
		}
		html.WriteString(`</tbody></table></div>`)
	}
}

func (a *AdminPanel) renderModulePermissions(html *strings.Builder, mod *parser.ModuleDefinition) {
	if len(mod.Permissions) > 0 {
		html.WriteString(`<div class="card"><div class="card-title">Permissions</div><table><thead><tr><th>Key</th><th>Description</th></tr></thead><tbody>`)
		keys := make([]string, 0, len(mod.Permissions))
		for k := range mod.Permissions {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			html.WriteString(fmt.Sprintf(`<tr><td><code>%s</code></td><td>%s</td></tr>`, k, mod.Permissions[k]))
		}
		html.WriteString(`</tbody></table></div>`)
	} else {
		html.WriteString(`<div class="card"><div class="empty-state">No permissions defined</div></div>`)
	}

	if len(mod.Groups) > 0 {
		html.WriteString(`<div class="card"><div class="card-title">Groups</div><table><thead><tr><th>Key</th><th>Label</th><th>Implies</th></tr></thead><tbody>`)
		gkeys := make([]string, 0, len(mod.Groups))
		for k := range mod.Groups {
			gkeys = append(gkeys, k)
		}
		sort.Strings(gkeys)
		for _, k := range gkeys {
			g := mod.Groups[k]
			implies := strings.Join(g.Implies, ", ")
			if implies == "" {
				implies = `<span class="text-muted">&mdash;</span>`
			}
			html.WriteString(fmt.Sprintf(`<tr><td><code>%s</code></td><td>%s</td><td>%s</td></tr>`, k, g.Label, implies))
		}
		html.WriteString(`</tbody></table></div>`)
	}
}

func (a *AdminPanel) renderModuleMenu(html *strings.Builder, mod *parser.ModuleDefinition) {
	if len(mod.Menu) == 0 {
		html.WriteString(`<div class="card"><div class="empty-state">No menu defined</div></div>`)
		return
	}

	html.WriteString(`<div class="card"><div class="card-title">Menu Structure</div><div style="padding:12px 16px">`)
	for _, item := range mod.Menu {
		icon := item.Icon
		if icon == "" {
			icon = "folder"
		}
		html.WriteString(fmt.Sprintf(`<div class="menu-group"><div class="menu-group-title">%s</div>`, item.Label))
		if len(item.Children) > 0 {
			for _, child := range item.Children {
				viewLink := child.Label
				if child.View != "" {
					viewLink = fmt.Sprintf(`<a href="/admin/views">%s</a> <span class="text-muted">(%s)</span>`, child.Label, child.View)
				}
				html.WriteString(fmt.Sprintf(`<div class="menu-child">%s</div>`, viewLink))
			}
		}
		html.WriteString(`</div>`)
	}
	html.WriteString(`</div></div>`)

	if len(mod.Settings) > 0 {
		html.WriteString(`<div class="card"><div class="card-title">Settings</div><table><thead><tr><th>Key</th><th>Type</th><th>Default</th></tr></thead><tbody>`)
		skeys := make([]string, 0, len(mod.Settings))
		for k := range mod.Settings {
			skeys = append(skeys, k)
		}
		sort.Strings(skeys)
		for _, k := range skeys {
			s := mod.Settings[k]
			defVal := fmt.Sprintf("%v", s.Default)
			if defVal == "" || defVal == "<nil>" {
				defVal = `<span class="text-muted">&mdash;</span>`
			}
			html.WriteString(fmt.Sprintf(`<tr><td><code>%s</code></td><td><code>%s</code></td><td>%s</td></tr>`, k, s.Type, defVal))
		}
		html.WriteString(`</tbody></table></div>`)
	}
}

func (a *AdminPanel) listViews(c *fiber.Ctx) error {
	filterModule := c.Query("module")
	views := a.views()

	modules := make(map[string]bool)
	for _, v := range views {
		modules[v.Module] = true
	}
	modOrder := make([]string, 0, len(modules))
	for m := range modules {
		modOrder = append(modOrder, m)
	}
	sort.Strings(modOrder)

	var html strings.Builder
	html.WriteString(a.pageHeader("Views", "views"))

	html.WriteString(`<div class="list-toolbar"><div class="list-filters">`)
	activeAll := ""
	if filterModule == "" {
		activeAll = " active"
	}
	html.WriteString(fmt.Sprintf(`<a href="/admin/views" class="filter-pill%s">All</a>`, activeAll))
	for _, m := range modOrder {
		active := ""
		if filterModule == m {
			active = " active"
		}
		html.WriteString(fmt.Sprintf(`<a href="/admin/views?module=%s" class="filter-pill%s">%s</a>`, m, active, m))
	}
	count := 0
	for _, v := range views {
		if filterModule == "" || v.Module == filterModule {
			count++
		}
	}
	html.WriteString(fmt.Sprintf(`</div><div class="list-count text-muted">%d of %d</div></div>`, count, len(views)))

	keys := make([]string, 0, len(views))
	for k := range views {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	html.WriteString(`<div class="card"><table><thead><tr><th>Name</th><th>Type</th><th>Model</th><th>Title</th><th>Module</th><th>Status</th></tr></thead><tbody>`)
	for _, key := range keys {
		v := views[key]
		if filterModule != "" && v.Module != filterModule {
			continue
		}
		editBadge := `<span class="badge green">editable</span>`
		if !v.Editable {
			editBadge = `<span class="badge muted">embedded</span>`
		}
		html.WriteString(fmt.Sprintf(`<tr><td><a href="/admin/views/%s" class="fw-500">%s</a></td><td><span class="badge blue">%s</span></td><td>%s</td><td>%s</td><td><span class="badge muted">%s</span></td><td>%s</td></tr>`,
			key, v.Def.Name, v.Def.Type, v.Def.Model, v.Def.Title, v.Module, editBadge))
	}
	if len(views) == 0 {
		html.WriteString(`<tr><td colspan="6" class="empty-state">No views registered. Load modules first.</td></tr>`)
	}
	html.WriteString(`</tbody></table></div>`)
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

func (a *AdminPanel) healthPage(c *fiber.Ctx) error {
	models := a.modelRegistry.List()
	modules := a.moduleRegistry.List()

	var html strings.Builder
	html.WriteString(a.pageHeader("Health", "health"))

	statusColor := "var(--green)"
	statusText := "Healthy"

	html.WriteString(fmt.Sprintf(`<div class="card"><div class="card-title">System Status</div><div style="padding:20px 16px;display:flex;align-items:center;gap:12px"><div class="health-dot" style="background:%s"></div><span style="font-size:16px;font-weight:600">%s</span><span class="text-muted">v%s</span></div></div>`, statusColor, statusText, a.health.Version))

	html.WriteString(`<div class="stats-grid">`)
	html.WriteString(statCard("Models", fmt.Sprintf("%d", len(models)), "var(--blue)"))
	html.WriteString(statCard("Modules", fmt.Sprintf("%d", len(modules)), "var(--green)"))
	html.WriteString(statCard("Views", fmt.Sprintf("%d", len(a.views())), "var(--amber)"))
	html.WriteString(statCard("Processes", fmt.Sprintf("%d", len(a.health.Processes)), "var(--primary)"))
	html.WriteString(`</div>`)

	html.WriteString(`<div class="conn-grid">`)

	html.WriteString(`<div class="card"><div class="card-title">Infrastructure</div><table>`)
	html.WriteString(fmt.Sprintf(`<tr><td class="fw-500">Engine</td><td>bitcode</td></tr>`))
	html.WriteString(fmt.Sprintf(`<tr><td class="fw-500">Version</td><td>%s</td></tr>`, a.health.Version))
	html.WriteString(fmt.Sprintf(`<tr><td class="fw-500">Database</td><td><span class="badge muted">%s</span></td></tr>`, a.health.DBDriver))
	html.WriteString(fmt.Sprintf(`<tr><td class="fw-500">Cache</td><td><span class="badge muted">%s</span></td></tr>`, a.health.CacheDriver))
	html.WriteString(`</table></div>`)

	html.WriteString(`<div class="card"><div class="card-title">Installed Modules</div><div class="conn-list">`)
	for _, m := range modules {
		html.WriteString(fmt.Sprintf(`<div class="conn-item"><div class="conn-model">%s</div><div class="conn-detail">v%s <span class="badge green">%s</span></div></div>`,
			m.Definition.Name, m.Definition.Version, m.State))
	}
	html.WriteString(`</div></div>`)

	html.WriteString(`</div>`)

	if len(a.health.Processes) > 0 {
		html.WriteString(`<div class="card"><div class="card-title">Registered Processes</div><div style="padding:12px 16px;display:flex;flex-wrap:wrap;gap:6px">`)
		sort.Strings(a.health.Processes)
		for _, p := range a.health.Processes {
			html.WriteString(fmt.Sprintf(`<span class="badge muted">%s</span>`, p))
		}
		html.WriteString(`</div></div>`)
	}

	html.WriteString(fmt.Sprintf(`<div style="margin-top:8px"><a href="/health" target="_blank" class="btn-sm">View JSON API &rarr;</a></div>`))
	html.WriteString(pageFooter())

	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.SendString(html.String())
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

func (a *AdminPanel) loginHistoryPage(c *fiber.Ctx) error {
	var html strings.Builder
	html.WriteString(a.pageHeader("Login History", "login-history"))

	results, _ := a.auditLogRepo.FindLoginHistory(200)

	html.WriteString(`<div class="card"><div class="card-title">Login / Logout / Register Activity</div>`)
	html.WriteString(`<table class="list-table"><thead><tr><th>Time</th><th>Action</th><th>User</th><th>IP Address</th><th>User Agent</th></tr></thead><tbody>`)

	for _, r := range results {
		action, _ := r["action"].(string)
		badgeClass := "blue"
		if action == "logout" {
			badgeClass = "muted"
		} else if action == "register" {
			badgeClass = "green"
		}
		userID := fmt.Sprintf("%v", r["user_id"])
		ip := fmt.Sprintf("%v", r["ip_address"])
		ua := fmt.Sprintf("%v", r["user_agent"])
		if len(ua) > 80 {
			ua = ua[:80] + "..."
		}
		createdAt := fmt.Sprintf("%v", r["created_at"])

		html.WriteString(fmt.Sprintf(`<tr><td class="text-muted" style="white-space:nowrap">%s</td><td><span class="badge %s">%s</span></td><td>%s</td><td><code>%s</code></td><td style="font-size:11px;color:var(--text-muted);max-width:300px;overflow:hidden;text-overflow:ellipsis">%s</td></tr>`,
			createdAt, badgeClass, action, userID, ip, template.HTMLEscapeString(ua)))
	}

	html.WriteString(`</tbody></table></div>`)

	return c.Type("html").SendString(html.String())
}

func (a *AdminPanel) requestLogPage(c *fiber.Ctx) error {
	methodFilter := c.Query("method", "")

	var html strings.Builder
	html.WriteString(a.pageHeader("API Request Log", "request-log"))

	html.WriteString(`<div style="margin-bottom:12px;display:flex;gap:4px">`)
	methods := []string{"", "GET", "POST", "PUT", "DELETE", "PATCH"}
	labels := []string{"All", "GET", "POST", "PUT", "DELETE", "PATCH"}
	for i, m := range methods {
		active := ""
		if m == methodFilter {
			active = " active"
		}
		href := "/admin/audit/request-log"
		if m != "" {
			href += "?method=" + m
		}
		html.WriteString(fmt.Sprintf(`<a href="%s" class="filter-pill%s">%s</a>`, href, active, labels[i]))
	}
	html.WriteString(`</div>`)

	results, _ := a.auditLogRepo.FindRequests(200, methodFilter)

	html.WriteString(`<div class="card"><div class="card-title">Request Log</div>`)
	html.WriteString(`<table class="list-table"><thead><tr><th>Time</th><th>Method</th><th>Path</th><th>Status</th><th>Duration</th><th>User</th><th>IP</th></tr></thead><tbody>`)

	for _, r := range results {
		method := fmt.Sprintf("%v", r["request_method"])
		path := fmt.Sprintf("%v", r["request_path"])
		status := fmt.Sprintf("%v", r["status_code"])
		duration := fmt.Sprintf("%v", r["duration_ms"])
		userID := fmt.Sprintf("%v", r["user_id"])
		ip := fmt.Sprintf("%v", r["ip_address"])
		createdAt := fmt.Sprintf("%v", r["created_at"])

		methodBadge := "blue"
		if method == "POST" {
			methodBadge = "green"
		} else if method == "PUT" || method == "PATCH" {
			methodBadge = "yellow"
		} else if method == "DELETE" {
			methodBadge = "red"
		}

		statusColor := "var(--green)"
		if status >= "400" {
			statusColor = "var(--red)"
		} else if status >= "300" {
			statusColor = "var(--yellow)"
		}

		html.WriteString(fmt.Sprintf(`<tr><td class="text-muted" style="white-space:nowrap;font-size:11px">%s</td><td><span class="badge %s">%s</span></td><td style="font-family:monospace;font-size:12px">%s</td><td style="color:%s;font-weight:500">%s</td><td>%sms</td><td>%s</td><td><code style="font-size:11px">%s</code></td></tr>`,
			createdAt, methodBadge, method, template.HTMLEscapeString(path), statusColor, status, duration, userID, ip))
	}

	html.WriteString(`</tbody></table></div>`)

	return c.Type("html").SendString(html.String())
}

func (a *AdminPanel) sidebarHTML(activePage string) string {
	var sb strings.Builder
	sb.WriteString(`<div class="sidebar">`)
	sb.WriteString(`<div class="sidebar-brand"><a href="/admin"><svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="var(--primary)" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="16 18 22 12 16 6"/><polyline points="8 6 2 12 8 18"/></svg><span>BitCode</span></a></div>`)
	sb.WriteString(`<a href="/app/home" class="sidebar-item" style="font-size:0.75rem;color:var(--text-muted);padding:0.25rem 1rem 0.5rem;border-bottom:1px solid var(--border);margin-bottom:0.5rem"><svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M19 12H5M12 19l-7-7 7-7"/></svg>Back to App</a>`)

	active := func(page string) string {
		if page == activePage {
			return " active"
		}
		return ""
	}

	sb.WriteString(fmt.Sprintf(`<a href="/admin" class="sidebar-item%s"><svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/><polyline points="9 22 9 12 15 12 15 22"/></svg>Dashboard</a>`, active("dashboard")))

	sb.WriteString(`<div class="sidebar-section">Build</div>`)
	sb.WriteString(fmt.Sprintf(`<a href="/admin/modules" class="sidebar-item%s"><svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/></svg>Modules</a>`, active("modules")))
	sb.WriteString(fmt.Sprintf(`<a href="/admin/models" class="sidebar-item%s"><svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="7" height="7"/><rect x="14" y="3" width="7" height="7"/><rect x="14" y="14" width="7" height="7"/><rect x="3" y="14" width="7" height="7"/></svg>Models</a>`, active("models")))
	sb.WriteString(fmt.Sprintf(`<a href="/admin/views" class="sidebar-item%s"><svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="2" y="3" width="20" height="14" rx="2" ry="2"/><line x1="8" y1="21" x2="16" y2="21"/><line x1="12" y1="17" x2="12" y2="21"/></svg>Views</a>`, active("views")))
	sb.WriteString(fmt.Sprintf(`<a href="/admin/health" class="sidebar-item%s"><svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 12h-4l-3 9L9 3l-3 9H2"/></svg>Health</a>`, active("health")))

	sb.WriteString(`<div class="sidebar-section">Audit</div>`)
	sb.WriteString(fmt.Sprintf(`<a href="/admin/audit/login-history" class="sidebar-item%s"><svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M15 3h4a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2h-4"/><polyline points="10 17 15 12 10 7"/><line x1="15" y1="12" x2="3" y2="12"/></svg>Login History</a>`, active("login-history")))
	sb.WriteString(fmt.Sprintf(`<a href="/admin/audit/request-log" class="sidebar-item%s"><svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/></svg>Request Log</a>`, active("request-log")))

	sb.WriteString(`</div>`)
	return sb.String()
}

func (a *AdminPanel) pageHeader(title, activePage string) string {
	breadcrumb := `<div class="breadcrumb"><a href="/admin">Admin</a>`
	if activePage != "dashboard" {
		breadcrumb += fmt.Sprintf(` <span class="sep">/</span> <span class="fw-500">%s</span>`, title)
	}
	breadcrumb += `</div>`

	return fmt.Sprintf(`<!DOCTYPE html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>%s - BitCode Admin</title>
%s</head><body>
<div class="layout">%s<div class="main"><div class="topbar">%s</div><div class="content"><div class="content-title">%s</div>`,
		title, cssBlock(), a.sidebarHTML(activePage), breadcrumb, title)
}

func (a *AdminPanel) modelPageHeader(modelName, moduleName, activeTab string) string {
	breadcrumb := fmt.Sprintf(`<div class="breadcrumb"><a href="/admin">Admin</a> <span class="sep">/</span> <a href="/admin/models">Models</a> <span class="sep">/</span> <span class="fw-500">%s</span></div>`, modelName)

	tabs := fmt.Sprintf(`<div class="tabs"><a href="/admin/models/%s?tab=form" class="tab%s">Form</a><a href="/admin/models/%s?tab=fields" class="tab%s">Fields</a><a href="/admin/models/%s?tab=connections" class="tab%s">Connections</a><a href="/admin/models/%s?tab=schema" class="tab%s">Schema</a></div>`,
		modelName, activeClass(activeTab, "form"),
		modelName, activeClass(activeTab, "fields"),
		modelName, activeClass(activeTab, "connections"),
		modelName, activeClass(activeTab, "schema"))

	meta := fmt.Sprintf(`<div class="model-header"><div class="model-name">%s</div><div class="model-meta"><span class="badge muted">%s</span></div></div>`, modelName, moduleName)

	return fmt.Sprintf(`<!DOCTYPE html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>%s - BitCode Admin</title>
%s</head><body>
<div class="layout">%s<div class="main"><div class="topbar">%s</div><div class="content">%s%s`,
		modelName, cssBlock(), a.sidebarHTML("models"), breadcrumb, meta, tabs)
}

func activeClass(current, tab string) string {
	if current == tab {
		return " active"
	}
	return ""
}

func pageFooter() string {
	return `</div></div></div></body></html>`
}

func statCard(label, value, color string) string {
	return fmt.Sprintf(`<div class="stat"><div class="stat-value" style="color:%s">%s</div><div class="stat-label">%s</div></div>`, color, value, label)
}

func renderFormInput(field parser.FieldDefinition) string {
	switch field.Type {
	case parser.FieldText, parser.FieldRichText, parser.FieldMarkdown, parser.FieldHTML, parser.FieldCode:
		return `<div class="form-input textarea"></div>`
	case parser.FieldBoolean, parser.FieldToggle:
		return `<div class="form-input toggle"><div class="toggle-track"><div class="toggle-thumb"></div></div></div>`
	case parser.FieldSelection, parser.FieldRadio:
		opts := strings.Join(field.Options, ", ")
		return fmt.Sprintf(`<div class="form-input select">%s</div>`, opts)
	case parser.FieldMany2One:
		return fmt.Sprintf(`<div class="form-input link">Link &rarr; <a href="/admin/models/%s">%s</a></div>`, field.Model, field.Model)
	case parser.FieldFile, parser.FieldImage:
		return `<div class="form-input file">Attach file</div>`
	case parser.FieldDate:
		return `<div class="form-input">yyyy-mm-dd</div>`
	case parser.FieldDatetime:
		return `<div class="form-input">yyyy-mm-dd hh:mm</div>`
	case parser.FieldColor:
		return `<div class="form-input color"><div class="color-swatch"></div></div>`
	case parser.FieldRating:
		return `<div class="form-input rating">&#9733;&#9733;&#9733;&#9733;&#9734;</div>`
	case parser.FieldPassword:
		return `<div class="form-input">&#8226;&#8226;&#8226;&#8226;&#8226;&#8226;&#8226;&#8226;</div>`
	default:
		return `<div class="form-input"></div>`
	}
}

func sortedFieldNames(fields map[string]parser.FieldDefinition) []string {
	names := make([]string, 0, len(fields))
	for k := range fields {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

func cssBlock() string {
	return `<style>
:root{--primary:#5e64ff;--primary-light:#eef0ff;--blue:#2490ef;--green:#48bb78;--amber:#ed8936;--red:#fc4438;--text:#1f272e;--text-muted:#8d99a6;--bg:#f4f5f6;--card:#fff;--border:#e2e6e9;--sidebar-w:220px;--radius:6px}
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',system-ui,sans-serif;background:var(--bg);color:var(--text);font-size:13px;line-height:1.5}
.layout{display:flex;min-height:100vh}
.sidebar{width:var(--sidebar-w);background:var(--card);border-right:1px solid var(--border);position:fixed;top:0;left:0;bottom:0;overflow-y:auto;z-index:10}
.sidebar-brand{padding:14px 16px;border-bottom:1px solid var(--border)}
.sidebar-brand a{display:flex;align-items:center;gap:8px;text-decoration:none;color:var(--text);font-weight:700;font-size:15px}
.sidebar-section{padding:16px 16px 4px;font-size:10px;font-weight:700;text-transform:uppercase;letter-spacing:.08em;color:var(--text-muted)}
.sidebar-item{display:flex;align-items:center;gap:8px;padding:6px 16px;color:var(--text);text-decoration:none;font-size:13px;border-radius:var(--radius);margin:1px 6px;transition:background .15s}
.sidebar-item:hover{background:var(--bg);text-decoration:none}
.sidebar-item.active{background:var(--primary-light);color:var(--primary);font-weight:600}
.sidebar-item.sub{padding-left:24px;font-size:12px;color:#5a6a7a}
.sidebar-item.sub:hover{color:var(--primary)}
.sidebar-meta{font-size:10px;color:var(--text-muted);margin-left:auto}
.main{margin-left:var(--sidebar-w);flex:1;min-width:0}
.topbar{padding:10px 24px;border-bottom:1px solid var(--border);background:var(--card)}
.breadcrumb{font-size:12px;color:var(--text-muted)}
.breadcrumb a{color:var(--text-muted);text-decoration:none}.breadcrumb a:hover{color:var(--primary)}
.breadcrumb .sep{margin:0 5px;color:#ccc}
.content{padding:20px 24px}
.content-title{font-size:17px;font-weight:600;margin-bottom:16px;color:var(--text)}
.model-header{margin-bottom:4px}
.model-name{font-size:20px;font-weight:700;margin-bottom:4px}
.model-meta{display:flex;align-items:center;gap:8px;margin-bottom:8px}
.tabs{display:flex;gap:0;border-bottom:1px solid var(--border);margin-bottom:16px}
.tab{padding:10px 20px;font-size:13px;color:var(--text-muted);text-decoration:none;border-bottom:2px solid transparent;transition:all .15s}
.tab:hover{color:var(--text);text-decoration:none}
.tab.active{color:var(--text);font-weight:600;border-bottom-color:var(--text)}
.card{background:var(--card);border:1px solid var(--border);border-radius:var(--radius);margin-bottom:14px;overflow:hidden}
.card-title{padding:12px 16px;font-size:13px;font-weight:600;border-bottom:1px solid var(--border);color:var(--text)}
table{width:100%;border-collapse:collapse}
th{padding:7px 16px;text-align:left;font-weight:600;font-size:11px;text-transform:uppercase;letter-spacing:.04em;color:var(--text-muted);background:#fafbfc;border-bottom:1px solid var(--border)}
td{padding:8px 16px;border-bottom:1px solid #f0f2f4;font-size:13px}
tr:last-child td{border-bottom:none}
tr:hover td{background:#fafbfc}
a{color:var(--primary);text-decoration:none}a:hover{text-decoration:underline}
.stats-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(160px,1fr));gap:10px;margin-bottom:16px}
.stat{background:var(--card);border:1px solid var(--border);border-radius:var(--radius);padding:18px;text-align:center}
.stat-value{font-size:26px;font-weight:700;line-height:1}.stat-label{color:var(--text-muted);font-size:11px;margin-top:4px}
.badge{display:inline-block;padding:2px 8px;border-radius:10px;font-size:11px;font-weight:500;line-height:1.6}
.badge.green{background:#e6f9ee;color:#1b7a41}.badge.blue{background:#e8f0fe;color:#1a56db}.badge.muted{background:#f0f2f4;color:#5a6a7a}
.fw-400{font-weight:400}.fw-500{font-weight:500}
.text-muted{color:var(--text-muted)}
code{background:#f4f5f6;padding:1px 5px;border-radius:3px;font-size:12px;color:#d63384}
.btn-sm{display:inline-block;padding:4px 12px;border-radius:var(--radius);font-size:12px;font-weight:500;color:var(--primary);border:1px solid var(--border);text-decoration:none;transition:all .15s}
.btn-sm:hover{background:var(--primary-light);border-color:var(--primary);text-decoration:none}
.btn-link{font-size:12px;font-weight:500;color:var(--primary);text-decoration:none}
.btn-link:hover{text-decoration:underline}
.dot{display:inline-block;width:8px;height:8px;border-radius:50%}.green-dot{background:var(--green)}.blue-dot{background:var(--blue)}
.empty-state{padding:24px;text-align:center;color:var(--text-muted);font-size:13px}
.list-toolbar{display:flex;justify-content:space-between;align-items:center;margin-bottom:10px}
.list-filters{display:flex;gap:4px;flex-wrap:wrap}
.filter-pill{display:inline-block;padding:4px 12px;border-radius:14px;font-size:12px;font-weight:500;color:var(--text-muted);background:var(--card);border:1px solid var(--border);text-decoration:none;transition:all .15s}
.filter-pill:hover{border-color:var(--primary);color:var(--primary);text-decoration:none}
.filter-pill.active{background:var(--primary);color:#fff;border-color:var(--primary)}
.list-count{font-size:12px}
.form-preview{padding:16px 20px}
.form-group{margin-bottom:14px}
.form-group label{display:block;font-size:11px;font-weight:600;color:var(--text-muted);text-transform:uppercase;letter-spacing:.04em;margin-bottom:4px}
.req-star{color:var(--red);margin-left:2px}
.form-input{background:#f8f9fa;border:1px solid var(--border);border-radius:var(--radius);padding:7px 10px;font-size:13px;color:var(--text-muted);min-height:34px}
.form-input.textarea{min-height:72px}
.form-input.toggle{background:none;border:none;padding:4px 0}
.toggle-track{width:36px;height:20px;background:#ccc;border-radius:10px;position:relative}
.toggle-thumb{width:16px;height:16px;background:#fff;border-radius:50%;position:absolute;top:2px;left:2px;box-shadow:0 1px 2px rgba(0,0,0,.15)}
.form-input.select{color:var(--text)}
.form-input.link{color:var(--text)}
.form-input.link a{font-weight:500}
.form-input.file{color:var(--text-muted);font-style:italic}
.form-input.color{display:flex;align-items:center;gap:8px}
.color-swatch{width:20px;height:20px;border-radius:4px;background:linear-gradient(135deg,#ff6b6b,#ffd93d,#6bcb77,#4d96ff)}
.form-input.rating{color:#f5a623;font-size:16px;background:none;border:none;padding:4px 0}
.form-section{margin-top:16px}
.form-section-title{font-size:13px;font-weight:600;margin-bottom:6px}
.form-child-table{background:#fafbfc;border:1px dashed var(--border);border-radius:var(--radius)}
.conn-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(280px,1fr));gap:12px}
.conn-list{padding:4px}
.conn-item{display:block;padding:8px 12px;border-radius:var(--radius);text-decoration:none;color:var(--text);transition:background .15s}
.conn-item:hover{background:var(--bg);text-decoration:none}
.conn-model{font-weight:600;font-size:13px;margin-bottom:2px}
.conn-detail{font-size:11px;color:var(--text-muted)}
.conn-detail code{font-size:11px}
.health-dot{width:12px;height:12px;border-radius:50%;display:inline-block;animation:pulse 2s infinite}
@keyframes pulse{0%,100%{opacity:1}50%{opacity:.5}}
.menu-group{margin-bottom:12px}
.menu-group-title{font-weight:600;font-size:13px;margin-bottom:4px;padding:4px 0;border-bottom:1px solid var(--border)}
.menu-child{padding:4px 0 4px 16px;font-size:13px}
</style>`
}

func (a *AdminPanel) apiImpersonate(c *fiber.Ctx) error {
	adminToken := c.Get("Authorization")
	if adminToken == "" {
		adminToken = c.Cookies("token")
	} else if len(adminToken) > 7 && adminToken[:7] == "Bearer " {
		adminToken = adminToken[7:]
	}
	if adminToken == "" {
		return c.Status(401).JSON(fiber.Map{"error": "authentication required"})
	}

	adminClaims, err := security.ValidateToken(a.jwtConfig, adminToken)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"error": "invalid token"})
	}

	if !containsRole(adminClaims.Roles, "admin") {
		return c.Status(403).JSON(fiber.Map{"error": "only admin users can impersonate"})
	}

	if adminClaims.ImpersonatedBy != "" {
		return c.Status(400).JSON(fiber.Map{"error": "cannot impersonate while already impersonating"})
	}

	targetUserID := c.Params("user_id")
	if targetUserID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "user_id required"})
	}

	repo := persistence.NewGenericRepository(a.db, a.modelRegistry.TableName("user"))
	targetUser, err := repo.FindByID(c.Context(), targetUserID)
	if err != nil || targetUser == nil {
		return c.Status(404).JSON(fiber.Map{"error": "user not found"})
	}

	targetUsername, _ := targetUser["username"].(string)

	targetRoles := loadUserRoles(a.db, targetUserID, a.modelRegistry)
	if containsRole(targetRoles, "admin") {
		return c.Status(403).JSON(fiber.Map{"error": "cannot impersonate another admin user"})
	}

	targetGroups := loadUserGroups(a.db, targetUserID, a.modelRegistry)

	token, err := security.GenerateToken(
		a.jwtConfig,
		targetUserID,
		targetUsername,
		targetRoles,
		targetGroups,
		security.WithImpersonatedBy(adminClaims.UserID),
		security.WithExpiration(1*time.Hour),
	)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to generate impersonation token"})
	}

	a.auditLogRepo.WriteAsync(persistence.AuditLogEntry{
		UserID:         adminClaims.UserID,
		Action:         "impersonate_start",
		ModelName:      "user",
		RecordID:       targetUserID,
		ImpersonatedBy: adminClaims.UserID,
		IPAddress:      c.IP(),
		UserAgent:      c.Get("User-Agent"),
		RequestMethod:  c.Method(),
		RequestPath:    c.Path(),
	})

	return c.JSON(fiber.Map{
		"token": token,
		"impersonating": fiber.Map{
			"user_id":  targetUserID,
			"username": targetUsername,
		},
		"admin": fiber.Map{
			"user_id":  adminClaims.UserID,
			"username": adminClaims.Username,
		},
		"expires_in": 3600,
	})
}

func (a *AdminPanel) apiStopImpersonate(c *fiber.Ctx) error {
	tokenStr := c.Get("Authorization")
	if tokenStr == "" {
		tokenStr = c.Cookies("token")
	} else if len(tokenStr) > 7 && tokenStr[:7] == "Bearer " {
		tokenStr = tokenStr[7:]
	}
	if tokenStr == "" {
		return c.Status(401).JSON(fiber.Map{"error": "authentication required"})
	}

	claims, err := security.ValidateToken(a.jwtConfig, tokenStr)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"error": "invalid token"})
	}

	if claims.ImpersonatedBy == "" {
		return c.Status(400).JSON(fiber.Map{"error": "not currently impersonating"})
	}

	adminID := claims.ImpersonatedBy
	repo := persistence.NewGenericRepository(a.db, a.modelRegistry.TableName("user"))
	adminUser, err := repo.FindByID(c.Context(), adminID)
	if err != nil || adminUser == nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to load admin user"})
	}

	adminUsername, _ := adminUser["username"].(string)
	adminRoles := loadUserRoles(a.db, adminID, a.modelRegistry)
	adminGroups := loadUserGroups(a.db, adminID, a.modelRegistry)

	newToken, err := security.GenerateToken(
		a.jwtConfig,
		adminID,
		adminUsername,
		adminRoles,
		adminGroups,
	)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to generate admin token"})
	}

	a.auditLogRepo.WriteAsync(persistence.AuditLogEntry{
		UserID:         adminID,
		Action:         "impersonate_stop",
		ModelName:      "user",
		RecordID:       claims.UserID,
		ImpersonatedBy: adminID,
		IPAddress:      c.IP(),
		UserAgent:      c.Get("User-Agent"),
		RequestMethod:  c.Method(),
		RequestPath:    c.Path(),
	})

	return c.JSON(fiber.Map{
		"token":    newToken,
		"user_id":  adminID,
		"username": adminUsername,
	})
}

func containsRole(roles []string, target string) bool {
	for _, r := range roles {
		if r == target {
			return true
		}
	}
	return false
}

func loadUserRoles(db *gorm.DB, userID string, reg *domainModel.Registry) []string {
	roleTable := reg.TableName("role")
	userRoleTable := reg.TableName("user") + "_roles"
	var roles []string
	db.Raw(fmt.Sprintf(`SELECT r.name FROM %s r
		INNER JOIN %s ur ON ur.role_id = r.id
		WHERE ur.user_id = ?`, roleTable, userRoleTable), userID).Scan(&roles)
	return roles
}

func loadUserGroups(db *gorm.DB, userID string, reg *domainModel.Registry) []string {
	groupTable := reg.TableName("group")
	userGroupTable := reg.TableName("user") + "_groups"
	var groups []string
	db.Raw(fmt.Sprintf(`SELECT g.name FROM %s g
		INNER JOIN %s ug ON ug.group_id = g.id
		WHERE ug.user_id = ?`, groupTable, userGroupTable), userID).Scan(&groups)
	return groups
}
