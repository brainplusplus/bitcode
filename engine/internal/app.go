package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/bitcode-engine/engine/embedded"
	"github.com/bitcode-engine/engine/internal/compiler/parser"
	"github.com/bitcode-engine/engine/internal/domain/event"
	domainModel "github.com/bitcode-engine/engine/internal/domain/model"
	"github.com/bitcode-engine/engine/internal/infrastructure/cache"
	"github.com/bitcode-engine/engine/internal/infrastructure/i18n"
	"github.com/bitcode-engine/engine/internal/infrastructure/module"
	"github.com/bitcode-engine/engine/internal/infrastructure/persistence"
	infrastorage "github.com/bitcode-engine/engine/internal/infrastructure/storage"
	"github.com/bitcode-engine/engine/internal/presentation/admin"
	"github.com/bitcode-engine/engine/internal/presentation/api"
	"github.com/bitcode-engine/engine/internal/presentation/middleware"
	tmpl "github.com/bitcode-engine/engine/internal/presentation/template"
	"github.com/bitcode-engine/engine/internal/presentation/view"
	ws "github.com/bitcode-engine/engine/internal/presentation/websocket"
	"github.com/bitcode-engine/engine/internal/runtime/executor"
	"github.com/bitcode-engine/engine/internal/runtime/executor/steps"
	"github.com/bitcode-engine/engine/internal/runtime/expression"
	"github.com/bitcode-engine/engine/internal/runtime/format"
	"github.com/bitcode-engine/engine/internal/runtime/pkgen"
	"github.com/bitcode-engine/engine/internal/runtime/plugin"
	wfEngine "github.com/bitcode-engine/engine/internal/runtime/workflow"
	"github.com/bitcode-engine/engine/pkg/security"
	"gorm.io/gorm"
)

type AppConfig struct {
	DB              persistence.DatabaseConfig
	Cache           cache.CacheConfig
	Tenant          middleware.TenantConfig
	Storage         infrastorage.StorageConfig
	JWTSecret       string
	Port            string
	ModuleDir       string
	GlobalModuleDir string
}

type viewEntry struct {
	Def    *parser.ViewDefinition
	Module string
	Path   string
}

type App struct {
	Config          AppConfig
	DB              *gorm.DB
	Fiber           *fiber.App
	ModelRegistry   *domainModel.Registry
	ModuleRegistry  *module.Registry
	ProcessRegistry *executor.ProcessRegistry
	EventBus        *event.Bus
	TemplateEngine  *tmpl.Engine
	ViewRenderer    *view.Renderer
	WorkflowEngine  *wfEngine.Engine
	Executor        *executor.Executor
	PluginManager   *plugin.Manager
	Cache           cache.Cache
	JWTConfig       security.JWTConfig
	WSHub           *ws.Hub
	Translator      *i18n.Translator
	SequenceEngine  *persistence.SequenceEngine
	FormatEngine    *format.Engine
	PKGenerator     *pkgen.Generator
	Hydrator        *expression.Hydrator
	viewDefs        map[string]*viewEntry
	moduleMenus     map[string][]parser.MenuItemDefinition
	moduleOrder     []string
}

func NewApp(cfg AppConfig) (*App, error) {
	db, err := persistence.NewDatabase(cfg.DB)
	if err != nil {
		return nil, fmt.Errorf("database connection failed: %w", err)
	}

	log.Printf("[DB] connected using %s driver", cfg.DB.Driver)

	if err := persistence.AutoMigrateViewRevisions(db); err != nil {
		log.Printf("[WARN] failed to migrate view_revisions: %v", err)
	}

	if err := persistence.AutoMigrateDataRevisions(db); err != nil {
		log.Printf("[WARN] failed to migrate data_revisions: %v", err)
	}

	if err := infrastorage.AutoMigrateAttachments(db); err != nil {
		log.Printf("[WARN] failed to migrate attachments: %v", err)
	}

	fiberApp := fiber.New(fiber.Config{
		AppName:      "LowCode Engine",
		ErrorHandler: defaultErrorHandler,
		BodyLimit:    50 * 1024 * 1024,
	})

	jwtCfg := security.JWTConfig{Secret: cfg.JWTSecret}
	templateEngine := tmpl.NewEngine()
	appCache := cache.NewCache(cfg.Cache)
	pluginMgr := plugin.NewManager()
	processReg := executor.NewProcessRegistry()
	wsHub := ws.NewHub()
	go wsHub.Run()
	translator := i18n.NewTranslator("en")

	seqEngine := persistence.NewSequenceEngine(db)
	if err := seqEngine.MigrateSequenceTable(); err != nil {
		log.Printf("[WARN] failed to migrate sequences table: %v", err)
	}
	fmtEngine := format.NewEngine(seqEngine)
	pkGen := pkgen.NewGenerator(fmtEngine, nil)
	modelReg := domainModel.NewRegistry()
	hydrator := expression.NewHydrator(db, modelReg)

	app := &App{
		Config:          cfg,
		DB:              db,
		Fiber:           fiberApp,
		ModelRegistry:   modelReg,
		ModuleRegistry:  module.NewRegistry(),
		ProcessRegistry: processReg,
		EventBus:        event.NewBus(),
		TemplateEngine:  templateEngine,
		ViewRenderer:    view.NewRenderer(db, templateEngine),
		WorkflowEngine:  wfEngine.NewEngine(),
		Executor:        executor.NewExecutor(),
		PluginManager:   pluginMgr,
		Cache:           appCache,
		JWTConfig:       jwtCfg,
		WSHub:           wsHub,
		Translator:      translator,
		SequenceEngine:  seqEngine,
		FormatEngine:    fmtEngine,
		PKGenerator:     pkGen,
		Hydrator:        hydrator,
		viewDefs:        make(map[string]*viewEntry),
		moduleMenus:     make(map[string][]parser.MenuItemDefinition),
	}

	app.ViewRenderer.SetModelRegistry(app.ModelRegistry)
	app.ViewRenderer.SetHydrator(app.Hydrator)
	app.registerStepHandlers()
	app.setupMiddleware()
	app.setupRoutes()
	app.startPluginRuntimes()

	return app, nil
}

func (a *App) startPluginRuntimes() {
	if err := a.PluginManager.StartTypescript(""); err != nil {
		log.Printf("[PLUGIN] TypeScript runtime not available: %v", err)
	} else {
		log.Println("[PLUGIN] TypeScript runtime started")
	}

	if err := a.PluginManager.StartPython(""); err != nil {
		log.Printf("[PLUGIN] Python runtime not available: %v", err)
	} else {
		log.Println("[PLUGIN] Python runtime started")
	}
}

func (a *App) registerStepHandlers() {
	a.Executor.RegisterHandler(parser.StepValidate, &steps.ValidateHandler{})
	a.Executor.RegisterHandler(parser.StepQuery, &steps.DataHandler{DB: a.DB})
	a.Executor.RegisterHandler(parser.StepCreate, &steps.DataHandler{DB: a.DB})
	a.Executor.RegisterHandler(parser.StepUpdate, &steps.DataHandler{DB: a.DB})
	a.Executor.RegisterHandler(parser.StepDelete, &steps.DataHandler{DB: a.DB})
	a.Executor.RegisterHandler(parser.StepIf, &steps.IfHandler{Executor: a.Executor})
	a.Executor.RegisterHandler(parser.StepSwitch, &steps.SwitchHandler{Executor: a.Executor})
	a.Executor.RegisterHandler(parser.StepLoop, &steps.LoopHandler{Executor: a.Executor})
	a.Executor.RegisterHandler(parser.StepEmit, &steps.EmitHandler{})
	a.Executor.RegisterHandler(parser.StepAssign, &steps.AssignHandler{})
	a.Executor.RegisterHandler(parser.StepLog, &steps.LogHandler{})
	a.Executor.RegisterHandler(parser.StepHTTP, &steps.HTTPHandler{})
	a.Executor.RegisterHandler(parser.StepScript, &steps.ScriptHandler{Runner: a.PluginManager})
	a.Executor.RegisterHandler(parser.StepCall, &steps.CallHandler{Executor: a.Executor, Loader: a.ProcessRegistry})
}

func (a *App) setupMiddleware() {
	if a.Config.Tenant.Enabled {
		a.Fiber.Use(middleware.TenantMiddleware(a.Config.Tenant))
	}
	a.Fiber.Use(middleware.AuditMiddleware())
}

func (a *App) setupRoutes() {
	a.Fiber.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":    "ok",
			"engine":    "bitcode",
			"version":   "0.1.0",
			"database":  a.Config.DB.Driver,
			"cache":     a.Config.Cache.Driver,
			"modules":   a.ModuleRegistry.InstalledNames(),
			"processes": a.ProcessRegistry.List(),
		})
	})

	authHandler := api.NewAuthHandler(a.DB, a.JWTConfig)
	authHandler.Register(a.Fiber)

	storageCfg := a.Config.Storage
	if storageCfg.Driver == "" {
		storageCfg = infrastorage.DefaultStorageConfig()
	}
	storageDriver, err := infrastorage.NewStorageDriver(storageCfg)
	if err != nil {
		log.Printf("[WARN] failed to initialize storage driver: %v, falling back to local", err)
		storageDriver = infrastorage.NewLocalStorage(storageCfg.Local)
	}
	attRepo := infrastorage.NewAttachmentRepository(a.DB)
	thumbSvc := infrastorage.NewThumbnailService(storageDriver, storageCfg.Thumbnail)
	fileHandler := api.NewFileHandler(attRepo, storageDriver, thumbSvc, storageCfg, a.JWTConfig)
	fileHandler.Register(a.Fiber)

	a.WSHub.ConnectToEventBus(a.EventBus)
	a.WSHub.RegisterRoutes(a.Fiber)

	adminPanel := admin.NewAdminPanel(a.DB, a.ModelRegistry, a.ModuleRegistry, a.viewsByName, admin.HealthInfo{
		Version:     "0.1.0",
		DBDriver:    a.Config.DB.Driver,
		CacheDriver: a.Config.Cache.Driver,
		Processes:   a.ProcessRegistry.List(),
	}, a.Config.ModuleDir)
	adminPanel.RegisterRoutes(a.Fiber)

	a.setupComponentAssets()

	a.Fiber.Get("/", func(c *fiber.Ctx) error {
		return c.Redirect("/app/home")
	})
	a.Fiber.Get("/app", a.handleAppRoot)
	a.Fiber.Get("/app/login", a.handleLoginPage)
	a.Fiber.Post("/app/login", a.handleLoginSubmit)
	a.Fiber.Get("/app/home", a.handleHomePage)
	a.Fiber.Get("/app/:module/*", a.handleViewGet)
	a.Fiber.Post("/app/:module/*", a.handleViewPost)
}

func (a *App) setupComponentAssets() {
	candidates := []string{
		filepath.Join("packages", "components", "dist", "bc-components"),
		filepath.Join("..", "packages", "components", "dist", "bc-components"),
		filepath.Join("..", "..", "packages", "components", "dist", "bc-components"),
		filepath.Join(a.Config.ModuleDir, "..", "packages", "components", "dist", "bc-components"),
		filepath.Join(a.Config.ModuleDir, "..", "..", "packages", "components", "dist", "bc-components"),
		filepath.Join("static", "components"),
	}
	for _, dir := range candidates {
		absDir, err := filepath.Abs(dir)
		if err != nil {
			continue
		}
		if _, err := os.Stat(absDir); err == nil {
			a.Fiber.Static("/assets/components", absDir, fiber.Static{
				Compress: true,
				MaxAge:   86400,
			})
			log.Printf("[ASSETS] serving components from %s", absDir)
			return
		}
	}
	log.Println("[ASSETS] component bundle not found — run 'npm run build' in packages/components/")
}

func (a *App) handleAppRoot(c *fiber.Ctx) error {
	token := c.Cookies("token")
	if token == "" {
		token = extractBearerToken(c)
	}
	if token != "" {
		if _, err := security.ValidateToken(a.JWTConfig, token); err == nil {
			return c.Redirect("/app/home")
		}
	}
	return c.Redirect("/app/login")
}

func (a *App) handleLoginPage(c *fiber.Ctx) error {
	errMsg := c.Query("error", "")

	if a.TemplateEngine.Has("templates/views/login.html") {
		html, err := a.TemplateEngine.Render("templates/views/login.html", map[string]any{
			"Error": errMsg,
		})
		if err == nil {
			c.Set("Content-Type", "text/html; charset=utf-8")
			return c.SendString(html)
		}
	}

	html := loginPageHTML(errMsg)
	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.SendString(html)
}

func (a *App) handleLoginSubmit(c *fiber.Ctx) error {
	username := c.FormValue("username")
	password := c.FormValue("password")

	if username == "" || password == "" {
		return c.Redirect("/app/login?error=Username+and+password+required")
	}

	repo := persistence.NewGenericRepository(a.DB, "users")
	users, _, err := repo.FindAll(c.Context(), [][]any{{"username", "=", username}}, 1, 1)
	if err != nil || len(users) == 0 {
		return c.Redirect("/app/login?error=Invalid+credentials")
	}

	hash, _ := users[0]["password_hash"].(string)
	if !security.CheckPassword(password, hash) {
		return c.Redirect("/app/login?error=Invalid+credentials")
	}

	userID, _ := users[0]["id"].(string)
	token, err := security.GenerateToken(a.JWTConfig, userID, username, nil, nil)
	if err != nil {
		return c.Redirect("/app/login?error=Server+error")
	}

	c.Cookie(&fiber.Cookie{
		Name:     "token",
		Value:    token,
		Path:     "/",
		HTTPOnly: true,
		MaxAge:   86400,
	})

	return c.Redirect("/app/home")
}

func (a *App) handleHomePage(c *fiber.Ctx) error {
	token := c.Cookies("token")
	if token == "" {
		token = extractBearerToken(c)
	}

	var username string
	if token != "" {
		claims, err := security.ValidateToken(a.JWTConfig, token)
		if err != nil {
			return c.Redirect("/app/login")
		}
		username = claims.Username
	} else {
		return c.Redirect("/app/login")
	}

	modules := a.ModuleRegistry.InstalledNames()
	menu := a.buildMenu("")

	type moduleCard struct {
		Name      string
		Label     string
		Color     string
		IconSVG   template.HTML
		FirstView string
		ViewCount int
	}

	moduleColors := map[string]string{
		"crm":       "#8B5CF6",
		"sales":     "#6366F1",
		"hrm":       "#10B981",
		"inventory": "#F59E0B",
		"purchase":  "#EF4444",
		"account":   "#3B82F6",
		"project":   "#EC4899",
		"website":   "#14B8A6",
	}
	defaultColor := "#64748B"

	moduleIcons := map[string]template.HTML{
		"crm":       template.HTML(`<svg width="24" height="24" viewBox="0 0 20 20" fill="#fff"><path d="M9 6a3 3 0 11-6 0 3 3 0 016 0zm8 0a3 3 0 11-6 0 3 3 0 016 0zm-4.07 11c.046-.327.07-.66.07-1a6.97 6.97 0 00-1.5-4.33A5 5 0 0119 16v1h-6.07zM6 11a5 5 0 015 5v1H1v-1a5 5 0 015-5z"/></svg>`),
		"sales":     template.HTML(`<svg width="24" height="24" viewBox="0 0 20 20" fill="#fff"><path fill-rule="evenodd" d="M10 2a4 4 0 00-4 4v1H5a1 1 0 00-.994.89l-1 9A1 1 0 004 18h12a1 1 0 00.994-1.11l-1-9A1 1 0 0015 7h-1V6a4 4 0 00-4-4zm2 5V6a2 2 0 10-4 0v1h4zm-6 3a1 1 0 112 0 1 1 0 01-2 0zm7-1a1 1 0 100 2 1 1 0 000-2z" clip-rule="evenodd"/></svg>`),
		"hrm":       template.HTML(`<svg width="24" height="24" viewBox="0 0 20 20" fill="#fff"><path fill-rule="evenodd" d="M6 6V5a3 3 0 013-3h2a3 3 0 013 3v1h2a2 2 0 012 2v3.57A22.952 22.952 0 0110 13a22.95 22.95 0 01-8-1.43V8a2 2 0 012-2h2zm2-1a1 1 0 011-1h2a1 1 0 011 1v1H8V5zm1 5a1 1 0 011-1h.01a1 1 0 110 2H10a1 1 0 01-1-1z" clip-rule="evenodd"/><path d="M2 13.692V16a2 2 0 002 2h12a2 2 0 002-2v-2.308A24.974 24.974 0 0110 15c-2.796 0-5.487-.46-8-1.308z"/></svg>`),
		"inventory": template.HTML(`<svg width="24" height="24" viewBox="0 0 20 20" fill="#fff"><path d="M4 3a2 2 0 100 4h12a2 2 0 100-4H4z"/><path fill-rule="evenodd" d="M3 8h14v7a2 2 0 01-2 2H5a2 2 0 01-2-2V8zm5 3a1 1 0 011-1h2a1 1 0 110 2H9a1 1 0 01-1-1z" clip-rule="evenodd"/></svg>`),
		"purchase":  template.HTML(`<svg width="24" height="24" viewBox="0 0 20 20" fill="#fff"><path d="M3 1a1 1 0 000 2h1.22l.305 1.222a.997.997 0 00.01.042l1.358 5.43-.893.892C3.74 11.846 4.632 14 6.414 14H15a1 1 0 000-2H6.414l1-1H14a1 1 0 00.894-.553l3-6A1 1 0 0017 3H6.28l-.31-1.243A1 1 0 005 1H3zm13 15.5a1.5 1.5 0 11-3 0 1.5 1.5 0 013 0zM6.5 18a1.5 1.5 0 100-3 1.5 1.5 0 000 3z"/></svg>`),
		"account":   template.HTML(`<svg width="24" height="24" viewBox="0 0 20 20" fill="#fff"><path fill-rule="evenodd" d="M4 4a2 2 0 00-2 2v4a2 2 0 002 2V6h10a2 2 0 00-2-2H4zm2 6a2 2 0 012-2h8a2 2 0 012 2v4a2 2 0 01-2 2H8a2 2 0 01-2-2v-4zm6 4a2 2 0 100-4 2 2 0 000 4z" clip-rule="evenodd"/></svg>`),
		"project":   template.HTML(`<svg width="24" height="24" viewBox="0 0 20 20" fill="#fff"><path d="M2 6a2 2 0 012-2h5l2 2h5a2 2 0 012 2v6a2 2 0 01-2 2H4a2 2 0 01-2-2V6z"/></svg>`),
	}
	defaultIcon := template.HTML(`<svg width="24" height="24" viewBox="0 0 20 20" fill="#fff"><path fill-rule="evenodd" d="M3 5a2 2 0 012-2h10a2 2 0 012 2v8a2 2 0 01-2 2h-2.22l.123.489.804.804A1 1 0 0113 18H7a1 1 0 01-.707-1.707l.804-.804L7.22 15H5a2 2 0 01-2-2V5zm5.771 7H5V5h10v7H8.771z" clip-rule="evenodd"/></svg>`)

	var cards []moduleCard
	for _, mod := range a.moduleOrder {
		if mod == "base" {
			continue
		}

		viewCount := 0
		firstView := mod + "/dashboard"
		for key, entry := range a.viewDefs {
			if entry.Module == mod {
				viewCount++
				if firstView == mod+"/dashboard" {
					firstView = key
				}
			}
		}
		if viewCount == 0 {
			continue
		}

		menuItems := a.moduleMenus[mod]
		if len(menuItems) > 0 && len(menuItems[0].Children) > 0 {
			firstView = mod + "/" + menuItems[0].Children[0].View
		}

		label := strings.Title(mod)
		if installed, err := a.ModuleRegistry.Get(mod); err == nil && installed.Definition.Label != "" {
			label = installed.Definition.Label
		}

		color := moduleColors[mod]
		if color == "" {
			color = defaultColor
		}
		icon := moduleIcons[mod]
		if icon == "" {
			icon = defaultIcon
		}

		cards = append(cards, moduleCard{
			Name:      mod,
			Label:     label,
			Color:     color,
			IconSVG:   icon,
			FirstView: firstView,
			ViewCount: viewCount,
		})
	}

	data := map[string]any{
		"Title":        "Home",
		"Username":     username,
		"Module":       "",
		"Menu":         menu,
		"SettingsMenu": []view.MenuEntry{},
		"CurrentPath":  "/app/home",
		"ActiveView":   "",
		"Modules":      modules,
		"Views":        a.viewDefs,
		"ModuleCards":  cards,
	}

	if a.TemplateEngine.Has("templates/views/home.html") {
		contentHTML, err := a.TemplateEngine.Render("templates/views/home.html", data)
		if err != nil {
			log.Printf("[WARN] home template render failed: %v", err)
		} else {
			data["Content"] = template.HTML(contentHTML)
			layoutName := "templates/layout-app.html"
			if !a.TemplateEngine.Has(layoutName) {
				layoutName = "templates/layout.html"
			}
			if a.TemplateEngine.Has(layoutName) {
				html, err := a.TemplateEngine.Render(layoutName, data)
				if err != nil {
					log.Printf("[WARN] home layout render failed: %v", err)
				} else {
					c.Set("Content-Type", "text/html; charset=utf-8")
					return c.SendString(html)
				}
			}
		}
	}

	html := homePageHTML(username, modules, a.viewDefs)
	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.SendString(html)
}

func extractBearerToken(c *fiber.Ctx) string {
	header := c.Get("Authorization")
	if len(header) > 7 && header[:7] == "Bearer " {
		return header[7:]
	}
	return ""
}

func (a *App) handleViewGet(c *fiber.Ctx) error {
	moduleName := c.Params("module")
	viewPath := c.Params("*")

	if viewPath == "" || viewPath == "/" {
		firstView := a.findFirstView(moduleName)
		if firstView != "" {
			return c.Redirect("/app/" + moduleName + "/" + firstView)
		}
		return c.Redirect("/app/home")
	}

	entry := a.resolveView(moduleName, viewPath)
	if entry == nil {
		return c.Status(404).JSON(fiber.Map{
			"error":  "view not found",
			"module": moduleName,
			"path":   viewPath,
			"hint":   "available at /app/{module}/{view_name}",
		})
	}

	token := c.Cookies("token")
	if token == "" {
		token = extractBearerToken(c)
	}
	var username string
	if token != "" {
		claims, _ := security.ValidateToken(a.JWTConfig, token)
		if claims != nil {
			username = claims.Username
		}
	}

	menu := a.buildMenu(moduleName)
	activeView := moduleName + "/" + viewPath

	opts := view.RenderOptions{
		Module:       moduleName,
		Username:     username,
		Menu:         menu,
		SettingsMenu: []view.MenuEntry{},
		ViewPath:     moduleName + "/" + viewPath,
		UseLayout:    true,
		CurrentPath:  c.Path(),
		ActiveView:   activeView,
		RecordID:     c.Query("id"),
		Page:         c.QueryInt("page", 1),
	}

	html, err := a.ViewRenderer.RenderView(c.Context(), entry.Def, opts)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.SendString(html)
}

func (a *App) handleViewPost(c *fiber.Ctx) error {
	moduleName := c.Params("module")
	viewPath := c.Params("*")

	entry := a.resolveView(moduleName, viewPath)
	if entry == nil {
		return c.Status(404).JSON(fiber.Map{"error": "view not found"})
	}

	if entry.Def.Type != parser.ViewForm || entry.Def.Model == "" {
		return c.Status(400).JSON(fiber.Map{"error": "only form views support POST"})
	}

	body := make(map[string]any)
	c.Request().PostArgs().VisitAll(func(key, value []byte) {
		k := string(key)
		if k == "_id" || k == "_method" {
			return
		}
		body[k] = string(value)
	})

	modelDef, _ := a.ModelRegistry.Get(entry.Def.Model)
	var repo *persistence.GenericRepository
	if modelDef != nil {
		repo = persistence.NewGenericRepositoryWithModel(a.DB, entry.Def.Model+"s", modelDef)
	} else {
		repo = persistence.NewGenericRepository(a.DB, entry.Def.Model+"s")
	}

	recordID := c.Query("id")
	if recordID == "" {
		recordID = c.FormValue("_id")
	}
	method := c.FormValue("_method")

	if recordID != "" && method == "PUT" {
		if err := repo.Update(c.Context(), recordID, body); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Redirect(fmt.Sprintf("/app/%s/%s?id=%s", moduleName, viewPath, recordID))
	}

	if a.PKGenerator != nil && modelDef != nil {
		session := map[string]any{}
		token := c.Cookies("token")
		if token != "" {
			if claims, err := security.ValidateToken(a.JWTConfig, token); err == nil {
				session["user_id"] = claims.UserID
				session["username"] = claims.Username
			}
		}
		pkCol, pkVal, pkErr := a.PKGenerator.GeneratePK(modelDef, body, session)
		if pkErr == nil && pkCol != "" && pkVal != nil {
			body[pkCol] = pkVal
		}
		for fieldName, fieldDef := range modelDef.Fields {
			if fieldDef.AutoFormat != nil {
				if val, afErr := a.PKGenerator.GenerateAutoFormat(modelDef, fieldName, &fieldDef, body, session); afErr == nil {
					body[fieldName] = val
				}
			}
		}
	}

	created, err := repo.Create(c.Context(), body)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	pkCol := pkgen.GetPKColumn(modelDef)
	if id, ok := created[pkCol]; ok {
		return c.Redirect(fmt.Sprintf("/app/%s/%s?id=%v", moduleName, viewPath, id))
	}
	if id, ok := created["id"].(string); ok {
		return c.Redirect(fmt.Sprintf("/app/%s/%s?id=%s", moduleName, viewPath, id))
	}

	listPath := strings.Replace(viewPath, "/form", "/list", 1)
	return c.Redirect(fmt.Sprintf("/app/%s/%s", moduleName, listPath))
}

func (a *App) resolveView(moduleName string, viewPath string) *viewEntry {
	viewPath = strings.TrimSuffix(viewPath, "/")

	key := moduleName + "/" + viewPath
	if entry, ok := a.viewDefs[key]; ok {
		return entry
	}

	if strings.Contains(viewPath, ".") {
		parts := strings.SplitN(viewPath, ".", 2)
		crossModule := parts[0]
		crossView := parts[1]
		crossKey := crossModule + "/" + crossView
		if entry, ok := a.viewDefs[crossKey]; ok {
			return entry
		}
	}

	for k, entry := range a.viewDefs {
		if entry.Module == moduleName && entry.Def.Name == viewPath {
			return a.viewDefs[k]
		}
	}

	return nil
}

func (a *App) findFirstView(moduleName string) string {
	menuItems, ok := a.moduleMenus[moduleName]
	if ok && len(menuItems) > 0 && len(menuItems[0].Children) > 0 {
		return menuItems[0].Children[0].View
	}
	for key, entry := range a.viewDefs {
		if entry.Module == moduleName {
			rel := strings.TrimPrefix(key, moduleName+"/")
			return rel
		}
	}
	return ""
}

func (a *App) buildMenu(currentModule string) []view.MenuEntry {
	var menu []view.MenuEntry

	includedModules := map[string][]string{}
	if currentModule != "" {
		if installed, err := a.ModuleRegistry.Get(currentModule); err == nil {
			for _, inc := range installed.Definition.IncludeMenus {
				includedModules[inc.Module] = inc.Views
			}
		}
	}

	for _, modName := range a.moduleOrder {
		installed, err := a.ModuleRegistry.Get(modName)
		if err != nil {
			continue
		}
		modDef := installed.Definition

		vis := modDef.MenuVisibility
		if vis == "admin" {
			continue
		}

		if currentModule != "" && modName != currentModule {
			allowedViews, isIncluded := includedModules[modName]
			if !isIncluded {
				continue
			}
			menuItems, ok := a.moduleMenus[modName]
			if !ok || len(menuItems) == 0 {
				continue
			}
			for _, item := range menuItems {
				entry := view.MenuEntry{
					Module: modName,
					Label:  item.Label,
					Icon:   item.Icon,
				}
				for _, child := range item.Children {
					if len(allowedViews) > 0 && !containsStr(allowedViews, child.View) {
						continue
					}
					entry.Children = append(entry.Children, view.MenuChild{
						Label:    child.Label,
						ViewPath: modName + "/" + child.View,
					})
				}
				if len(entry.Children) > 0 {
					menu = append(menu, entry)
				}
			}
			continue
		}

		menuItems, ok := a.moduleMenus[modName]
		if !ok || len(menuItems) == 0 {
			continue
		}

		for _, item := range menuItems {
			entry := view.MenuEntry{
				Module: modName,
				Label:  item.Label,
				Icon:   item.Icon,
			}
			for _, child := range item.Children {
				entry.Children = append(entry.Children, view.MenuChild{
					Label:    child.Label,
					ViewPath: modName + "/" + child.View,
				})
			}
			menu = append(menu, entry)
		}
	}

	return menu
}

func containsStr(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

func (a *App) viewsByName() map[string]*admin.ViewInfo {
	result := make(map[string]*admin.ViewInfo)
	for _, entry := range a.viewDefs {
		key := entry.Module + "/" + entry.Def.Name
		editable := entry.Path != "" && !strings.HasPrefix(entry.Path, os.TempDir())
		result[key] = &admin.ViewInfo{
			Def:      entry.Def,
			Module:   entry.Module,
			FilePath: entry.Path,
			Editable: editable,
		}
	}
	return result
}

func (a *App) LoadModules() error {
	projectFS := module.NewDiskFS(a.Config.ModuleDir)

	globalDir := a.Config.GlobalModuleDir
	if globalDir == "" {
		home, _ := os.UserHomeDir()
		globalDir = filepath.Join(home, ".bitcode", "modules")
	}
	globalFS := module.NewDiskFS(globalDir)

	embedFS := module.NewEmbedFSFromEmbed(embedded.ModulesFS, "modules")

	layered := module.NewLayeredFS(projectFS, globalFS, embedFS)

	moduleNames, err := layered.DiscoverModules()
	if err != nil {
		return fmt.Errorf("failed to discover modules: %w", err)
	}

	allModules := make(map[string]*parser.ModuleDefinition)
	for _, name := range moduleNames {
		modFS := layered.SubFS(name)
		data, err := modFS.ReadFile("module.json")
		if err != nil {
			log.Printf("[WARN] skipping module %s: %v", name, err)
			continue
		}
		modDef, err := parser.ParseModule(data)
		if err != nil {
			log.Printf("[WARN] skipping invalid module %s: %v", name, err)
			continue
		}
		allModules[modDef.Name] = modDef
	}

	installOrder, err := module.ResolveDependencies(allModules, findRootModules(allModules)...)
	if err != nil {
		return fmt.Errorf("dependency resolution failed: %w", err)
	}

	for _, modName := range installOrder {
		diskPath := a.resolveModuleDiskPath(modName)
		if diskPath != "" {
			if err := a.installModule(diskPath); err != nil {
				return fmt.Errorf("failed to install module %s: %w", modName, err)
			}
		} else {
			modFS := layered.SubFS(modName)
			tmpDir, err := module.ExtractModuleFS(modFS, modName)
			if err != nil {
				return fmt.Errorf("failed to extract embedded module %s: %w", modName, err)
			}
			if err := a.installModule(tmpDir); err != nil {
				return fmt.Errorf("failed to install module %s: %w", modName, err)
			}
		}
		log.Printf("[MODULE] installed: %s", modName)
	}

	a.processViewRegistrations()

	a.ViewRenderer.SetViewResolver(func(name string) *parser.ViewDefinition {
		for _, entry := range a.viewDefs {
			if entry.Def.Name == name {
				return entry.Def
			}
		}
		return nil
	})

	return nil
}

func (a *App) resolveModuleDiskPath(modName string) string {
	projectPath := filepath.Join(a.Config.ModuleDir, modName)
	if _, err := os.Stat(filepath.Join(projectPath, "module.json")); err == nil {
		return projectPath
	}

	globalDir := a.Config.GlobalModuleDir
	if globalDir == "" {
		home, _ := os.UserHomeDir()
		globalDir = filepath.Join(home, ".bitcode", "modules")
	}
	globalPath := filepath.Join(globalDir, modName)
	if _, err := os.Stat(filepath.Join(globalPath, "module.json")); err == nil {
		return globalPath
	}

	return ""
}

func (a *App) installModule(modPath string) error {
	loaded, err := module.LoadModule(modPath)
	if err != nil {
		return err
	}

	modName := loaded.Definition.Name

	for _, m := range loaded.Models {
		if m.Inherit != "" {
			parent, err := a.ModelRegistry.Get(m.Inherit)
			if err == nil {
				m = persistence.MergeInheritedFields(parent, m)
			}
		}

		if err := a.ModelRegistry.Register(m); err != nil {
			return fmt.Errorf("failed to register model %s: %w", m.Name, err)
		}
		if err := persistence.MigrateModel(a.DB, m); err != nil {
			return fmt.Errorf("failed to migrate model %s: %w", m.Name, err)
		}
	}

	revisionRepo := persistence.NewDataRevisionRepository(a.DB)
	router := api.NewRouterFull(a.Fiber, a.DB, a.WorkflowEngine, a.Hydrator, revisionRepo)
	for _, apiDef := range loaded.APIs {
		if apiDef.Auth {
			basePath := apiDef.GetBasePath()
			a.Fiber.Use(basePath, middleware.AuthMiddleware(a.JWTConfig))
		}
		router.RegisterAPI(apiDef)
	}

	a.loadI18n(modPath, loaded.Definition.I18n)
	a.loadProcesses(modPath, loaded.Definition.Processes)
	a.loadWorkflows(modPath, loaded.Definition.Processes)

	a.loadViews(modPath, modName)

	if loaded.Definition.Menu != nil {
		a.moduleMenus[modName] = loaded.Definition.Menu
	}
	a.moduleOrder = append(a.moduleOrder, modName)

	templateDir := filepath.Join(modPath, "templates")
	if modName == "base" {
		a.TemplateEngine.LoadDirectoryWithPrefix(templateDir, "templates")
	} else {
		a.TemplateEngine.LoadDirectoryWithPrefix(templateDir, "templates/"+modName)
		a.TemplateEngine.LoadDirectoryWithPrefix(templateDir, "templates")
	}

	if err := module.SeedModule(a.DB, modPath, loaded.Definition.Data); err != nil {
		log.Printf("[WARN] seeding failed for %s: %v", modName, err)
	}

	a.ModuleRegistry.Register(loaded.Definition, modPath)
	return nil
}

func (a *App) loadViews(modPath string, modName string) {
	viewDir := filepath.Join(modPath, "views")
	filepath.Walk(viewDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".json") {
			return nil
		}

		viewDef, err := parser.ParseViewFile(path)
		if err != nil {
			log.Printf("[WARN] skipping invalid view %s: %v", path, err)
			return nil
		}

		relPath, _ := filepath.Rel(viewDir, path)
		relPath = filepath.ToSlash(relPath)
		relPath = strings.TrimSuffix(relPath, ".json")

		key := modName + "/" + relPath

		a.viewDefs[key] = &viewEntry{
			Def:    viewDef,
			Module: modName,
			Path:   relPath,
		}

		nameKey := modName + "/" + viewDef.Name
		if nameKey != key {
			a.viewDefs[nameKey] = a.viewDefs[key]
		}

		log.Printf("[VIEW] %s → /app/%s", key, key)
		return nil
	})
}

func (a *App) processViewRegistrations() {
	for _, entry := range a.viewDefs {
		if entry.Def.RegisterTo == nil {
			continue
		}
		for _, targetModule := range entry.Def.RegisterTo {
			if !a.ModuleRegistry.IsInstalled(targetModule) {
				log.Printf("[VIEW] cross-module registration skipped: module %q not installed (view %s)", targetModule, entry.Def.Name)
				continue
			}
			crossKey := targetModule + "/" + entry.Module + "." + entry.Def.Name
			if _, exists := a.viewDefs[crossKey]; !exists {
				a.viewDefs[crossKey] = entry
				log.Printf("[VIEW] cross-module: %s registered in %s as %s", entry.Def.Name, targetModule, crossKey)
			}
		}
	}
}

func (a *App) loadI18n(modPath string, patterns []string) {
	for _, pattern := range patterns {
		fullPattern := filepath.Join(modPath, pattern)
		matches, _ := filepath.Glob(fullPattern)
		for _, match := range matches {
			if err := a.Translator.LoadFile(match); err != nil {
				log.Printf("[WARN] failed to load i18n %s: %v", match, err)
			} else {
				log.Printf("[I18N] loaded %s", match)
			}
		}
	}
}

func (a *App) loadProcesses(modPath string, patterns []string) {
	for _, pattern := range patterns {
		fullPattern := filepath.Join(modPath, pattern)
		matches, _ := filepath.Glob(fullPattern)
		for _, match := range matches {
			data, err := os.ReadFile(match)
			if err != nil {
				continue
			}

			var raw map[string]json.RawMessage
			if err := json.Unmarshal(data, &raw); err != nil {
				continue
			}

			if _, hasSteps := raw["steps"]; hasSteps {
				proc, err := parser.ParseProcess(data)
				if err != nil {
					log.Printf("[WARN] invalid process %s: %v", match, err)
					continue
				}
				a.ProcessRegistry.Register(proc)
			}
		}
	}
}

func (a *App) loadWorkflows(modPath string, patterns []string) {
	for _, pattern := range patterns {
		fullPattern := filepath.Join(modPath, pattern)
		matches, _ := filepath.Glob(fullPattern)
		for _, match := range matches {
			data, err := os.ReadFile(match)
			if err != nil {
				continue
			}

			var raw map[string]json.RawMessage
			if err := json.Unmarshal(data, &raw); err != nil {
				continue
			}

			if _, hasStates := raw["states"]; hasStates {
				wf, err := parser.ParseWorkflow(data)
				if err != nil {
					log.Printf("[WARN] invalid workflow %s: %v", match, err)
					continue
				}
				a.WorkflowEngine.Register(wf)
				log.Printf("[WORKFLOW] registered: %s", wf.Name)
			}
		}
	}
}

func (a *App) Start() error {
	port := a.Config.Port
	if port == "" {
		port = "8080"
	}
	log.Printf("LowCode Engine starting on :%s", port)
	return a.Fiber.Listen(":" + port)
}

func (a *App) Shutdown() error {
	a.PluginManager.StopAll()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return a.Fiber.ShutdownWithContext(ctx)
}

func findRootModules(modules map[string]*parser.ModuleDefinition) []string {
	var roots []string
	for name := range modules {
		isDepOf := false
		for _, other := range modules {
			for _, dep := range other.Depends {
				if dep == name {
					isDepOf = true
					break
				}
			}
			if isDepOf {
				break
			}
		}
		if !isDepOf {
			roots = append(roots, name)
		}
	}
	if len(roots) == 0 {
		for name := range modules {
			roots = append(roots, name)
		}
	}
	return roots
}

func defaultErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
	}
	return c.Status(code).JSON(fiber.Map{"error": err.Error()})
}

func loginPageHTML(errMsg string) string {
	errorBlock := ""
	if errMsg != "" {
		errorBlock = fmt.Sprintf(`<div style="background:#FEE2E2;color:#991B1B;padding:0.75rem;border-radius:6px;margin-bottom:1rem;">%s</div>`, errMsg)
	}

	return fmt.Sprintf(`<!DOCTYPE html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Login - LowCode Engine</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;background:linear-gradient(135deg,#1a1a2e 0%%,#16213e 50%%,#0f3460 100%%);display:flex;justify-content:center;align-items:center;min-height:100vh}
.card{background:#fff;padding:2.5rem;border-radius:16px;box-shadow:0 20px 60px rgba(0,0,0,0.3);width:100%%;max-width:400px}
h1{font-size:1.5rem;margin-bottom:0.5rem;color:#1a1a2e}
.subtitle{color:#666;margin-bottom:1.5rem;font-size:0.9rem}
label{display:block;font-weight:600;margin-bottom:0.25rem;font-size:0.85rem;color:#333}
input{width:100%%;padding:0.7rem 0.9rem;border:1.5px solid #e2e8f0;border-radius:8px;font-size:0.95rem;margin-bottom:1rem;outline:none;transition:border 0.2s,box-shadow 0.2s}
input:focus{border-color:#3B82F6;box-shadow:0 0 0 3px rgba(59,130,246,0.15)}
button{width:100%%;padding:0.75rem;background:#3B82F6;color:#fff;border:none;border-radius:8px;font-size:1rem;font-weight:600;cursor:pointer;transition:background 0.2s}
button:hover{background:#2563EB}
.footer{text-align:center;margin-top:1.5rem;font-size:0.8rem;color:#a0aec0}
</style></head><body>
<div class="card">
<h1>LowCode Engine</h1>
<p class="subtitle">Sign in to your workspace</p>
%s
<form method="POST" action="/app/login">
<label>Username</label>
<input type="text" name="username" placeholder="admin" autofocus required>
<label>Password</label>
<input type="password" name="password" placeholder="password" required>
<button type="submit">Sign In</button>
</form>
<div class="footer">Powered by LowCode Engine</div>
</div></body></html>`, errorBlock)
}

func homePageHTML(username string, modules []string, views map[string]*viewEntry) string {
	var moduleCards strings.Builder
	for _, mod := range modules {
		if mod == "base" {
			continue
		}
		var viewLinks strings.Builder
		for key, entry := range views {
			if entry.Module == mod {
				viewLinks.WriteString(fmt.Sprintf(`<a href="/app/%s" style="display:flex;align-items:center;justify-content:space-between;padding:0.6rem 1.25rem;border-bottom:1px solid #f0f0f0;color:#1a202c;text-decoration:none;font-size:0.875rem;transition:background 0.15s;"><span>%s</span><span style="background:#DBEAFE;color:#1E40AF;padding:2px 8px;border-radius:4px;font-size:0.7rem;">%s</span></a>`, key, entry.Def.Title, entry.Def.Type))
			}
		}
		moduleCards.WriteString(fmt.Sprintf(`<div style="background:#fff;border-radius:8px;box-shadow:0 1px 3px rgba(0,0,0,0.08);border:1px solid #e2e8f0;overflow:hidden;"><div style="padding:1rem 1.25rem;border-bottom:1px solid #e2e8f0;display:flex;align-items:center;justify-content:space-between;"><h3 style="font-size:0.95rem;font-weight:600;text-transform:capitalize;">%s</h3><span style="background:#D1FAE5;color:#065F46;padding:2px 8px;border-radius:4px;font-size:0.75rem;">Active</span></div>%s</div>`, mod, viewLinks.String()))
	}

	return fmt.Sprintf(`<!DOCTYPE html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Home - LowCode Engine</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;background:#f4f6f9}
.nav{background:#1a1a2e;color:#fff;padding:1rem 2rem;display:flex;justify-content:space-between;align-items:center}
.nav h1{font-size:1.1rem}.nav a{color:#ccc;text-decoration:none;font-size:0.9rem;margin-left:1.5rem}.nav a:hover{color:#fff}
.container{max-width:900px;margin:2rem auto;padding:0 1.5rem}
.welcome{margin-bottom:2rem}
.welcome h2{font-size:1.4rem;color:#1a202c}.welcome p{color:#718096;margin-top:0.25rem}
.grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(300px,1fr));gap:1rem}
</style></head><body>
<nav class="nav">
<h1>LowCode Engine</h1>
<div><a href="/admin">Admin</a><a href="/health">Health</a><a href="/app/login" onclick="document.cookie='token=;path=/;max-age=0'">Logout</a></div>
</nav>
<div class="container">
<div class="welcome"><h2>Welcome, %s</h2><p>Select a module to get started.</p></div>
<div class="grid">%s</div>
</div></body></html>`, username, moduleCards.String())
}
