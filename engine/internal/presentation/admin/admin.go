package admin

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/bitcode-framework/bitcode/internal/compiler/parser"
	domainModel "github.com/bitcode-framework/bitcode/internal/domain/model"
	"github.com/bitcode-framework/bitcode/internal/infrastructure/module"
	"github.com/bitcode-framework/bitcode/internal/infrastructure/persistence"
	"github.com/bitcode-framework/bitcode/pkg/security"
	"gorm.io/gorm"
)

type HealthInfo struct {
	Version     string
	DBDriver    string
	CacheDriver string
	Processes   []string
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
	admin.Get("/models/:module/:name", a.viewModel)
	admin.Get("/models/:module/:name/data", a.listModelData)
	admin.Get("/models/:name", a.viewModelLegacy)
	admin.Get("/modules", a.listModules)
	admin.Get("/modules/:name", a.viewModule)
	admin.Get("/views", a.listViews)
	admin.Get("/views/:module/:name", a.viewDetail)
	admin.Get("/health", a.healthPage)
	admin.Get("/audit/login-history", a.loginHistoryPage)
	admin.Get("/audit/request-log", a.requestLogPage)

	admin.Get("/groups", a.listGroups)
	admin.Get("/groups/:name", a.viewGroup)
	admin.Get("/securities", a.securitySyncPage)

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

	api.Get("/groups", a.apiListGroups)
	api.Get("/groups/:id", a.apiGetGroup)
	api.Post("/groups", a.apiCreateGroup)
	api.Put("/groups/:id", a.apiUpdateGroup)
	api.Delete("/groups/:id", a.apiDeleteGroup)

	api.Post("/securities/load", a.apiSecurityLoad)
	api.Post("/securities/export", a.apiSecurityExport)
	api.Get("/securities/download", a.apiSecurityDownload)
	api.Post("/securities/upload", a.apiSecurityUpload)
	api.Get("/securities/diff", a.apiSecurityDiff)
	api.Get("/securities/history", a.apiSecurityHistory)
	api.Post("/securities/rollback/:id", a.apiSecurityRollback)
}

func (a *AdminPanel) dashboard(c *fiber.Ctx) error {
	models := a.modelRegistry.List()
	modules := a.moduleRegistry.List()
	views := a.views()

	var html strings.Builder
	html.WriteString(a.pageHeader("Dashboard", "dashboard"))

	html.WriteString(`<div class="stats-grid">`)
	html.WriteString(statCard("Models", fmt.Sprintf("%d", len(models)), "var(--blue)"))
	html.WriteString(statCard("Modules", fmt.Sprintf("%d", len(modules)), "var(--green)"))
	html.WriteString(statCard("Views", fmt.Sprintf("%d", len(views)), "var(--amber)"))
	html.WriteString(statCard("Processes", fmt.Sprintf("%d", len(a.health.Processes)), "var(--primary)"))
	html.WriteString(`</div>`)

	html.WriteString(`<div class="conn-grid">`)

	html.WriteString(`<div class="card"><div class="card-title">Models</div><div class="conn-list">`)
	grouped := a.modelsByModule()
	for _, mod := range grouped.order {
		for _, m := range grouped.models[mod] {
			moduleName := m.Module
			if moduleName == "" {
				moduleName = "base"
			}
			html.WriteString(fmt.Sprintf(`<a href="/admin/models/%s/%s" class="conn-item"><div class="conn-model">%s</div><div class="conn-detail"><span class="badge muted">%s</span> %d fields</div></a>`,
				moduleName, m.Name, m.Name, moduleName, len(m.Fields)))
		}
	}
	html.WriteString(`</div></div>`)

	html.WriteString(`<div class="card"><div class="card-title">Modules</div><div class="conn-list">`)
	for _, m := range modules {
		html.WriteString(fmt.Sprintf(`<a href="/admin/modules/%s" class="conn-item"><div class="conn-model">%s</div><div class="conn-detail">v%s <span class="badge green">%s</span></div></a>`,
			m.Definition.Name, m.Definition.Name, m.Definition.Version, m.State))
	}
	html.WriteString(`</div></div>`)

	html.WriteString(`</div>`)
	html.WriteString(pageFooter())

	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.SendString(html.String())
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

	sb.WriteString(`<div class="sidebar-section">Security</div>`)
	sb.WriteString(fmt.Sprintf(`<a href="/admin/groups" class="sidebar-item%s"><svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>Groups</a>`, active("groups")))
	sb.WriteString(fmt.Sprintf(`<a href="/admin/securities" class="sidebar-item%s"><svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg>Security Sync</a>`, active("securities")))

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
:root{--primary:#5e64ff;--primary-light:#eef0ff;--blue:#2490ef;--green:#48bb78;--amber:#ed8936;--red:#fc4438;--yellow:#ecc94b;--text:#1f272e;--text-muted:#8d99a6;--bg:#f4f5f6;--card:#fff;--border:#e2e6e9;--sidebar-w:220px;--radius:6px}
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
.badge.green{background:#e6f9ee;color:#1b7a41}.badge.blue{background:#e8f0fe;color:#1a56db}.badge.muted{background:#f0f2f4;color:#5a6a7a}.badge.red{background:#fee;color:#c00}.badge.yellow{background:#fef9e7;color:#92400e}
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
.filter-pill{display:inline-block;padding:4px 12px;border-radius:14px;font-size:12px;font-weight:500;color:var(--text-muted);background:var(--card);border:1px solid var(--border);text-decoration:none;transition:all .15s;cursor:pointer}
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
