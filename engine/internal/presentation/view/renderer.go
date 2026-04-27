package view

import (
	"context"
	"fmt"
	"html/template"
	"strings"

	"github.com/bitcode-framework/bitcode/internal/compiler/parser"
	"github.com/bitcode-framework/bitcode/internal/infrastructure/persistence"
	tmplEngine "github.com/bitcode-framework/bitcode/internal/presentation/template"
	"github.com/bitcode-framework/bitcode/internal/runtime/expression"
	"gorm.io/gorm"
)

type MenuChild struct {
	Label    string
	ViewPath string
	Active   bool
}

type MenuEntry struct {
	Module   string
	Label    string
	Icon     string
	Children []MenuChild
}

type RenderOptions struct {
	Module       string
	Username     string
	Menu         []MenuEntry
	SettingsMenu []MenuEntry
	ViewPath     string
	UseLayout    bool
	CurrentPath  string
	ActiveView   string
	RecordID     string
	Page         int
}

type Renderer struct {
	db                *gorm.DB
	template          *tmplEngine.Engine
	componentCompiler *ComponentCompiler
	modelRegistry     ModelLookup
	hydrator          *expression.Hydrator
	tableNameResolver interface{ TableName(string) string }
}

func (r *Renderer) SetTableNameResolver(resolver interface{ TableName(string) string }) {
	r.tableNameResolver = resolver
}

func (r *Renderer) resolveTable(modelName string) string {
	if r.tableNameResolver != nil {
		return r.tableNameResolver.TableName(modelName)
	}
	return modelName
}

type ModelLookup interface {
	Get(name string) (*parser.ModelDefinition, error)
}

func NewRenderer(db *gorm.DB, tmpl *tmplEngine.Engine) *Renderer {
	return &Renderer{db: db, template: tmpl, componentCompiler: NewComponentCompiler()}
}

func (r *Renderer) SetModelRegistry(registry ModelLookup) {
	r.modelRegistry = registry
	r.componentCompiler.modelLookup = registry
}

func (r *Renderer) SetHydrator(h *expression.Hydrator) {
	r.hydrator = h
}

func (r *Renderer) SetViewResolver(resolver func(name string) *parser.ViewDefinition) {
	r.componentCompiler.embeddedViewRenderer = func(viewName string) string {
		viewDef := resolver(viewName)
		if viewDef == nil {
			return fmt.Sprintf(`<p style="color:var(--text-muted);font-size:0.85rem;">View "%s" not found</p>`, viewName)
		}
		if viewDef.Type == parser.ViewList && viewDef.Model != "" {
			repo := persistence.NewGenericRepository(r.db, r.resolveTable(viewDef.Model))
			records, total, err := repo.FindAll(context.Background(), nil, 1, 10)
			if err != nil {
				return fmt.Sprintf(`<p style="color:var(--danger);">Error loading %s: %s</p>`, viewName, err.Error())
			}
			return r.renderEmbeddedList(viewDef, records, total)
		}
		return fmt.Sprintf(`<p style="color:var(--text-muted);font-size:0.85rem;">Embedded %s view: %s</p>`, viewDef.Type, viewName)
	}
}

func (r *Renderer) renderEmbeddedList(viewDef *parser.ViewDefinition, records []map[string]any, total int64) string {
	return r.defaultListHTML(viewDef, records, total)
}

func (r *Renderer) RenderView(ctx context.Context, viewDef *parser.ViewDefinition, opts ...RenderOptions) (string, error) {
	var opt RenderOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	var contentHTML string
	var err error

	switch viewDef.Type {
	case parser.ViewList:
		contentHTML, err = r.renderList(ctx, viewDef, opt)
	case parser.ViewForm:
		contentHTML, err = r.renderForm(ctx, viewDef, opt)
	case parser.ViewKanban:
		contentHTML, err = r.renderKanban(ctx, viewDef)
	case parser.ViewCalendar:
		contentHTML, err = r.renderCalendar(ctx, viewDef)
	case parser.ViewChart:
		contentHTML, err = r.renderChart(ctx, viewDef)
	case parser.ViewCustom:
		contentHTML, err = r.renderCustom(ctx, viewDef, opt.Module)
	default:
		return "", fmt.Errorf("view type %q not supported", viewDef.Type)
	}

	if err != nil {
		return "", err
	}

	if opt.UseLayout && r.template.Has("templates/layout.html") {
		layoutData := map[string]any{
			"Content":      template.HTML(contentHTML),
			"Title":        viewDef.Title,
			"Module":       opt.Module,
			"Username":     opt.Username,
			"Menu":         opt.Menu,
			"SettingsMenu": opt.SettingsMenu,
			"ViewPath":     opt.ViewPath,
			"CurrentPath":  opt.CurrentPath,
			"ActiveView":   opt.ActiveView,
		}
		return r.template.Render("templates/layout.html", layoutData)
	}

	return contentHTML, nil
}

func (r *Renderer) renderList(ctx context.Context, viewDef *parser.ViewDefinition, opt RenderOptions) (string, error) {
	if viewDef.Model == "" {
		return "", fmt.Errorf("list view requires a model")
	}
	return r.defaultListHTML(viewDef, nil, 0), nil
}

func (r *Renderer) renderForm(ctx context.Context, viewDef *parser.ViewDefinition, opt RenderOptions) (string, error) {
	var record map[string]any
	recordID := opt.RecordID

	if recordID != "" {
		if viewDef.Model != "" {
			repo := persistence.NewGenericRepository(r.db, r.resolveTable(viewDef.Model))
			rec, err := repo.FindByID(ctx, recordID)
			if err == nil && rec != nil {
				record = rec
				if r.hydrator != nil && r.modelRegistry != nil {
					if modelDef, mErr := r.modelRegistry.Get(viewDef.Model); mErr == nil {
						r.hydrator.HydrateRecord(ctx, modelDef, record)
					}
				}
			}
		}
	}

	if record == nil {
		record = make(map[string]any)
	}

	listUrl := ""
	formAction := ""
	if opt.Module != "" {
		listUrl = fmt.Sprintf("/app/%s/%s/list", opt.Module, viewDef.Model)
		formAction = fmt.Sprintf("/app/%s/%s/form", opt.Module, viewDef.Model)
		if recordID != "" {
			formAction += "?id=" + recordID
		}
	}

	data := map[string]any{
		"title":      viewDef.Title,
		"model":      viewDef.Model,
		"layout":     viewDef.Layout,
		"actions":    viewDef.Actions,
		"record":     record,
		"recordId":   recordID,
		"listUrl":    listUrl,
		"formAction": formAction,
	}

	return r.componentCompiler.CompileFormFull(viewDef, record, data), nil
}

func (r *Renderer) renderKanban(ctx context.Context, viewDef *parser.ViewDefinition) (string, error) {
	if viewDef.Model == "" || viewDef.GroupBy == "" {
		return "", fmt.Errorf("kanban view requires model and group_by")
	}

	repo := persistence.NewGenericRepository(r.db, r.resolveTable(viewDef.Model))
	records, _, err := repo.FindAll(ctx, nil, 1, 200)
	if err != nil {
		return "", err
	}

	groups := make(map[string][]map[string]any)
	for _, rec := range records {
		groupVal := fmt.Sprintf("%v", rec[viewDef.GroupBy])
		groups[groupVal] = append(groups[groupVal], rec)
	}

	data := map[string]any{
		"title":  viewDef.Title,
		"groups": groups,
		"fields": viewDef.Fields,
	}

	if r.template.Has("templates/views/kanban.html") {
		return r.template.Render("templates/views/kanban.html", data)
	}

	if r.template.Has("views/kanban.html") {
		return r.template.Render("views/kanban.html", data)
	}

	var html strings.Builder
	html.WriteString(fmt.Sprintf(`<h2>%s</h2><div class="bc-kanban">`, viewDef.Title))
	for group, items := range groups {
		html.WriteString(fmt.Sprintf(`<div class="bc-kanban-col"><div class="bc-kanban-col-header"><span>%s</span><span class="bc-kanban-col-count">%d</span></div><div class="bc-kanban-col-body">`, group, len(items)))
		for _, item := range items {
			html.WriteString(`<div class="bc-kanban-card">`)
			for _, f := range viewDef.Fields {
				if v, ok := item[f]; ok {
					html.WriteString(fmt.Sprintf(`<div class="bc-kanban-card-field"><strong>%s:</strong> %v</div>`, f, v))
				}
			}
			html.WriteString(`</div>`)
		}
		html.WriteString(`</div></div>`)
	}
	html.WriteString(`</div>`)
	return html.String(), nil
}

func (r *Renderer) renderCalendar(ctx context.Context, viewDef *parser.ViewDefinition) (string, error) {
	if viewDef.Model == "" {
		return "", fmt.Errorf("calendar view requires a model")
	}

	repo := persistence.NewGenericRepository(r.db, r.resolveTable(viewDef.Model))
	records, _, err := repo.FindAll(ctx, nil, 1, 100)
	if err != nil {
		return "", err
	}

	data := map[string]any{
		"title":   viewDef.Title,
		"records": records,
		"fields":  viewDef.Fields,
	}

	if r.template.Has("templates/views/calendar.html") {
		return r.template.Render("templates/views/calendar.html", data)
	}

	if r.template.Has("views/calendar.html") {
		return r.template.Render("views/calendar.html", data)
	}

	return fmt.Sprintf(`<div class="bc-card"><div class="bc-card-header"><h2>%s (Calendar)</h2></div><div class="bc-card-body"><p>%d events</p></div></div>`, viewDef.Title, len(records)), nil
}

func (r *Renderer) renderChart(ctx context.Context, viewDef *parser.ViewDefinition) (string, error) {
	data := make(map[string]any)
	data["title"] = viewDef.Title

	for name, ds := range viewDef.DataSources {
		if ds.Model != "" {
			repo := persistence.NewGenericRepository(r.db, r.resolveTable(ds.Model))
			records, _, err := repo.FindAll(ctx, persistence.QueryFromDomain(ds.Domain), 1, 1000)
			if err != nil || records == nil {
				data[name] = []map[string]any{}
				continue
			}
			data[name] = records
		}
	}

	if r.template.Has("templates/views/chart.html") {
		return r.template.Render("templates/views/chart.html", data)
	}

	if r.template.Has("views/chart.html") {
		return r.template.Render("views/chart.html", data)
	}

	return fmt.Sprintf(`<div class="bc-card"><div class="bc-card-header"><h2>%s (Chart)</h2></div><div class="bc-card-body"><p>Chart data loaded</p></div></div>`, viewDef.Title), nil
}

func (r *Renderer) renderCustom(ctx context.Context, viewDef *parser.ViewDefinition, moduleName string) (string, error) {
	data := make(map[string]any)

	for name, ds := range viewDef.DataSources {
		if ds.Model != "" {
			repo := persistence.NewGenericRepository(r.db, r.resolveTable(ds.Model))
			records, _, err := repo.FindAll(ctx, persistence.QueryFromDomain(ds.Domain), 1, 100)
			if err != nil {
				data[name] = []map[string]any{}
				continue
			}
			if records == nil {
				records = []map[string]any{}
			}
			data[name] = records
		}
	}

	tmplName := viewDef.Template
	if moduleName != "" {
		modulePrefixed := strings.Replace(tmplName, "templates/", "templates/"+moduleName+"/", 1)
		if r.template.Has(modulePrefixed) {
			return r.template.Render(modulePrefixed, data)
		}
	}

	return r.template.Render(tmplName, data)
}

func (r *Renderer) defaultListHTML(viewDef *parser.ViewDefinition, records []map[string]any, total int64) string {
	moduleName := ""
	if r.modelRegistry != nil {
		if modelDef, err := r.modelRegistry.Get(viewDef.Model); err == nil && modelDef != nil {
			moduleName = modelDef.Module
		}
	}

	opts := &DatatableOptions{
		ModuleName: moduleName,
	}
	if moduleName != "" && viewDef.Model != "" {
		plural := viewDef.Model + "s"
		opts.CreateUrl = "/" + moduleName + "/" + plural + "/new"
		opts.DetailUrl = "/" + moduleName + "/" + plural + "/:id"
		opts.EditUrl = "/" + moduleName + "/" + plural + "/:id/edit"
	}

	return r.componentCompiler.CompileListDatatable(viewDef, opts)
}

func ComponentScriptTag() string {
	return `<script type="module" src="/assets/components/bc-components.esm.js"></script>
<script nomodule src="/assets/components/bc-components.js"></script>
<link rel="stylesheet" href="https://unpkg.com/leaflet@1.9.4/dist/leaflet.css" />`
}
