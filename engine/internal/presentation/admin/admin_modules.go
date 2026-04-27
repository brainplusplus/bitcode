package admin

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/bitcode-framework/bitcode/internal/compiler/parser"
)

func (a *AdminPanel) listModules(c *fiber.Ctx) error {
	var html strings.Builder
	html.WriteString(a.pageHeader("Modules", "modules"))

	columns := []map[string]any{
		{"field": "name", "label": "Name", "sortable": true},
		{"field": "version", "label": "Version"},
		{"field": "label", "label": "Label", "sortable": true},
		{"field": "category", "label": "Category", "sortable": true, "filterable": true},
		{"field": "dependencies", "label": "Dependencies"},
		{"field": "status", "label": "Status"},
	}

	html.WriteString(adminDatatable(columns, "/admin/api/list/modules", map[string]string{
		"detail-url": "/admin/modules/:id",
	}))
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
	html.WriteString(fmt.Sprintf(`<tr><td class="fw-500">Securities</td><td>%d patterns</td></tr>`, len(mod.Securities)))
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
				inheritVal = fmt.Sprintf(`<a href="/admin/models/%s/%s">%s</a>`, m.Module, m.Inherit, m.Inherit)
			}
			mlabel := m.Label
			if mlabel == "" {
				mlabel = m.Name
			}
			html.WriteString(fmt.Sprintf(`<tr><td><a href="/admin/models/%s/%s" class="fw-500">%s</a></td><td class="text-muted">%s</td><td>%d</td><td>%s</td></tr>`,
				m.Module, m.Name, m.Name, mlabel, len(m.Fields), inheritVal))
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
