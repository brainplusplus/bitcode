package admin

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/bitcode-engine/engine/internal/compiler/parser"
	domainModel "github.com/bitcode-engine/engine/internal/domain/model"
	"github.com/bitcode-engine/engine/internal/infrastructure/module"
	"github.com/bitcode-engine/engine/internal/infrastructure/persistence"
	"gorm.io/gorm"
)

type AdminPanel struct {
	db             *gorm.DB
	modelRegistry  *domainModel.Registry
	moduleRegistry *module.Registry
	viewDefs       map[string]*parser.ViewDefinition
}

func NewAdminPanel(db *gorm.DB, modelReg *domainModel.Registry, moduleReg *module.Registry, viewDefs map[string]*parser.ViewDefinition) *AdminPanel {
	return &AdminPanel{db: db, modelRegistry: modelReg, moduleRegistry: moduleReg, viewDefs: viewDefs}
}

func (a *AdminPanel) RegisterRoutes(app *fiber.App) {
	admin := app.Group("/admin")

	admin.Get("/", a.dashboard)
	admin.Get("/models", a.listModels)
	admin.Get("/models/:name", a.viewModel)
	admin.Get("/models/:name/data", a.listModelData)
	admin.Get("/modules", a.listModules)
	admin.Get("/views", a.listViews)
}

func (a *AdminPanel) dashboard(c *fiber.Ctx) error {
	models := a.modelRegistry.List()
	modules := a.moduleRegistry.List()

	var html strings.Builder
	html.WriteString(adminHeader("Dashboard"))
	html.WriteString(`<div class="stats">`)
	html.WriteString(statCard("Models", fmt.Sprintf("%d", len(models)), "#3B82F6"))
	html.WriteString(statCard("Modules", fmt.Sprintf("%d", len(modules)), "#10B981"))
	html.WriteString(statCard("Views", fmt.Sprintf("%d", len(a.viewDefs)), "#F59E0B"))
	html.WriteString(`</div>`)

	html.WriteString(`<h2>Installed Modules</h2><table><thead><tr><th>Name</th><th>Version</th><th>Status</th></tr></thead><tbody>`)
	for _, m := range modules {
		html.WriteString(fmt.Sprintf(`<tr><td>%s</td><td>%s</td><td><span class="badge green">%s</span></td></tr>`,
			m.Definition.Name, m.Definition.Version, m.State))
	}
	html.WriteString(`</tbody></table>`)

	html.WriteString(`<h2>Registered Models</h2><table><thead><tr><th>Name</th><th>Module</th><th>Fields</th><th>Actions</th></tr></thead><tbody>`)
	for _, m := range models {
		html.WriteString(fmt.Sprintf(`<tr><td><a href="/admin/models/%s">%s</a></td><td>%s</td><td>%d</td><td><a href="/admin/models/%s/data">View Data</a></td></tr>`,
			m.Name, m.Name, m.Module, len(m.Fields), m.Name))
	}
	html.WriteString(`</tbody></table>`)
	html.WriteString(adminFooter())

	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.SendString(html.String())
}

func (a *AdminPanel) listModels(c *fiber.Ctx) error {
	models := a.modelRegistry.List()

	var html strings.Builder
	html.WriteString(adminHeader("Models"))
	html.WriteString(`<table><thead><tr><th>Name</th><th>Module</th><th>Label</th><th>Fields</th><th>Inherit</th><th>Record Rules</th></tr></thead><tbody>`)
	for _, m := range models {
		html.WriteString(fmt.Sprintf(`<tr><td><a href="/admin/models/%s">%s</a></td><td>%s</td><td>%s</td><td>%d</td><td>%s</td><td>%d</td></tr>`,
			m.Name, m.Name, m.Module, m.Label, len(m.Fields), m.Inherit, len(m.RecordRules)))
	}
	html.WriteString(`</tbody></table>`)
	html.WriteString(adminFooter())

	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.SendString(html.String())
}

func (a *AdminPanel) viewModel(c *fiber.Ctx) error {
	name := c.Params("name")
	model, err := a.modelRegistry.Get(name)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "model not found"})
	}

	var html strings.Builder
	html.WriteString(adminHeader("Model: " + model.Name))
	html.WriteString(fmt.Sprintf(`<p><strong>Module:</strong> %s | <strong>Label:</strong> %s | <strong>Inherit:</strong> %s</p>`, model.Module, model.Label, model.Inherit))

	html.WriteString(`<h3>Fields</h3><table><thead><tr><th>Name</th><th>Type</th><th>Required</th><th>Unique</th><th>Default</th><th>Model (FK)</th></tr></thead><tbody>`)
	for fieldName, field := range model.Fields {
		html.WriteString(fmt.Sprintf(`<tr><td>%s</td><td>%s</td><td>%v</td><td>%v</td><td>%v</td><td>%s</td></tr>`,
			fieldName, field.Type, field.Required, field.Unique, field.Default, field.Model))
	}
	html.WriteString(`</tbody></table>`)

	if len(model.RecordRules) > 0 {
		html.WriteString(`<h3>Record Rules</h3><table><thead><tr><th>Groups</th><th>Domain</th></tr></thead><tbody>`)
		for _, rule := range model.RecordRules {
			html.WriteString(fmt.Sprintf(`<tr><td>%v</td><td>%v</td></tr>`, rule.Groups, rule.Domain))
		}
		html.WriteString(`</tbody></table>`)
	}

	html.WriteString(fmt.Sprintf(`<p><a href="/admin/models/%s/data">View Data &rarr;</a></p>`, model.Name))
	html.WriteString(adminFooter())

	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.SendString(html.String())
}

func (a *AdminPanel) listModelData(c *fiber.Ctx) error {
	name := c.Params("name")
	model, err := a.modelRegistry.Get(name)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "model not found"})
	}

	repo := persistence.NewGenericRepository(a.db, name+"s")
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
	html.WriteString(adminHeader(fmt.Sprintf("Data: %s (%d records)", model.Name, total)))

	html.WriteString(`<table><thead><tr>`)
	for _, f := range fields {
		html.WriteString(fmt.Sprintf(`<th>%s</th>`, f))
	}
	html.WriteString(`</tr></thead><tbody>`)

	for _, rec := range records {
		html.WriteString(`<tr>`)
		for _, f := range fields {
			val := rec[f]
			if val == nil {
				val = ""
			}
			html.WriteString(fmt.Sprintf(`<td>%v</td>`, val))
		}
		html.WriteString(`</tr>`)
	}
	html.WriteString(`</tbody></table>`)
	html.WriteString(adminFooter())

	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.SendString(html.String())
}

func (a *AdminPanel) listModules(c *fiber.Ctx) error {
	modules := a.moduleRegistry.List()

	var html strings.Builder
	html.WriteString(adminHeader("Modules"))
	html.WriteString(`<table><thead><tr><th>Name</th><th>Version</th><th>Label</th><th>Category</th><th>Dependencies</th><th>Status</th></tr></thead><tbody>`)
	for _, m := range modules {
		deps := strings.Join(m.Definition.Depends, ", ")
		html.WriteString(fmt.Sprintf(`<tr><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td><span class="badge green">%s</span></td></tr>`,
			m.Definition.Name, m.Definition.Version, m.Definition.Label, m.Definition.Category, deps, m.State))
	}
	html.WriteString(`</tbody></table>`)
	html.WriteString(adminFooter())

	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.SendString(html.String())
}

func (a *AdminPanel) listViews(c *fiber.Ctx) error {
	var html strings.Builder
	html.WriteString(adminHeader("Views"))
	html.WriteString(`<table><thead><tr><th>Route</th><th>Type</th><th>Model</th><th>Title</th><th>Preview</th></tr></thead><tbody>`)
	for key, v := range a.viewDefs {
		html.WriteString(fmt.Sprintf(`<tr><td>%s</td><td><span class="badge blue">%s</span></td><td>%s</td><td>%s</td><td><a href="/app/%s" target="_blank">Open</a></td></tr>`,
			key, v.Type, v.Model, v.Title, key))
	}
	html.WriteString(`</tbody></table>`)
	html.WriteString(adminFooter())

	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.SendString(html.String())
}

func adminHeader(title string) string {
	return fmt.Sprintf(`<!DOCTYPE html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>%s - Admin</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;background:#f5f5f5;color:#333}
.nav{background:#1a1a2e;color:#fff;padding:1rem 2rem;display:flex;gap:2rem;align-items:center}
.nav a{color:#ccc;text-decoration:none;font-size:0.9rem}.nav a:hover{color:#fff}
.nav h1{font-size:1.2rem;margin-right:2rem}
.container{max-width:1200px;margin:2rem auto;padding:0 2rem}
h2{margin:1.5rem 0 1rem;color:#1a1a2e}
h3{margin:1rem 0 0.5rem}
table{width:100%%;border-collapse:collapse;background:#fff;border-radius:8px;overflow:hidden;box-shadow:0 1px 3px rgba(0,0,0,0.1);margin-bottom:1.5rem}
th{background:#f8f9fa;padding:0.75rem 1rem;text-align:left;font-weight:600;font-size:0.85rem;color:#666;border-bottom:2px solid #eee}
td{padding:0.75rem 1rem;border-bottom:1px solid #f0f0f0;font-size:0.9rem}
tr:hover{background:#f8f9fa}
a{color:#3B82F6;text-decoration:none}a:hover{text-decoration:underline}
.stats{display:flex;gap:1rem;margin-bottom:2rem}
.stat{flex:1;background:#fff;padding:1.5rem;border-radius:8px;box-shadow:0 1px 3px rgba(0,0,0,0.1);text-align:center}
.stat .value{font-size:2.5rem;font-weight:700}.stat .label{color:#666;font-size:0.85rem;margin-top:0.25rem}
.badge{padding:2px 8px;border-radius:4px;font-size:0.8rem;font-weight:500}
.badge.green{background:#D1FAE5;color:#065F46}.badge.blue{background:#DBEAFE;color:#1E40AF}
p{margin:0.5rem 0}
</style></head><body>
<nav class="nav"><h1>LowCode Admin</h1><a href="/admin">Dashboard</a><a href="/admin/models">Models</a><a href="/admin/modules">Modules</a><a href="/admin/views">Views</a><a href="/health">Health</a></nav>
<div class="container"><h1>%s</h1>`, title, title)
}

func adminFooter() string {
	return `</div></body></html>`
}

func statCard(label string, value string, color string) string {
	return fmt.Sprintf(`<div class="stat"><div class="value" style="color:%s">%s</div><div class="label">%s</div></div>`, color, value, label)
}
