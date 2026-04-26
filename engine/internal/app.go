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
	"github.com/google/uuid"
	"github.com/bitcode-framework/bitcode/embedded"
	"github.com/bitcode-framework/bitcode/internal/compiler/parser"
	"github.com/bitcode-framework/bitcode/internal/domain/event"
	domainModel "github.com/bitcode-framework/bitcode/internal/domain/model"
	"github.com/bitcode-framework/bitcode/internal/domain/setting"
	"github.com/bitcode-framework/bitcode/internal/infrastructure/cache"
	"github.com/bitcode-framework/bitcode/internal/infrastructure/i18n"
	"github.com/bitcode-framework/bitcode/internal/infrastructure/module"
	"github.com/bitcode-framework/bitcode/internal/infrastructure/persistence"
	infrastorage "github.com/bitcode-framework/bitcode/internal/infrastructure/storage"
	"github.com/bitcode-framework/bitcode/internal/presentation/admin"
	"github.com/bitcode-framework/bitcode/internal/presentation/api"
	"github.com/bitcode-framework/bitcode/internal/presentation/middleware"
	tmpl "github.com/bitcode-framework/bitcode/internal/presentation/template"
	"github.com/bitcode-framework/bitcode/internal/presentation/view"
	ws "github.com/bitcode-framework/bitcode/internal/presentation/websocket"
	"github.com/bitcode-framework/bitcode/pkg/email"
	"github.com/bitcode-framework/bitcode/internal/runtime/executor"
	"github.com/bitcode-framework/bitcode/internal/runtime/executor/steps"
	"github.com/bitcode-framework/bitcode/internal/runtime/expression"
	"github.com/bitcode-framework/bitcode/internal/runtime/format"
	"github.com/bitcode-framework/bitcode/internal/runtime/hook"
	"github.com/bitcode-framework/bitcode/internal/runtime/pkgen"
	"github.com/bitcode-framework/bitcode/internal/runtime/plugin"
	"github.com/bitcode-framework/bitcode/internal/runtime/validation"
	wfEngine "github.com/bitcode-framework/bitcode/internal/runtime/workflow"
	"github.com/bitcode-framework/bitcode/pkg/security"
	"gorm.io/gorm"
)

type AppConfig struct {
	DB              persistence.DatabaseConfig
	Cache           cache.CacheConfig
	Tenant          middleware.TenantConfig
	Storage         infrastorage.StorageConfig
	RateLimit       middleware.RateLimitConfig
	IPWhitelist     middleware.IPWhitelistConfig
	Security        SecurityConfig
	SMTP            SMTPConfig
	JWTSecret       string
	EncryptionKey   string
	Port            string
	ModuleDir       string
	GlobalModuleDir string
}

type SecurityConfig struct {
	SessionDuration time.Duration
	CookieSecure    bool
	CookieSameSite  string
}

type SMTPConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	From     string
	TLS      bool
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
	SequenceEngine  persistence.SequenceEngine
	FormatEngine    *format.Engine
	PKGenerator     *pkgen.Generator
	Hydrator        *expression.Hydrator
	AuditLogRepo    *persistence.AuditLogRepository
	MongoConn       *persistence.MongoConnection
	DBDriver        string
	FieldEncryptor  *security.FieldEncryptor
	SettingStore    *setting.Store
	HookDispatcher  *hook.ModelHookDispatcher
	Validator       *validation.ValidatorAdapter
	Sanitizer       *validation.Sanitizer
	MigrationEngine *module.MigrationEngine
	viewDefs        map[string]*viewEntry
	moduleMenus     map[string][]parser.MenuItemDefinition
	moduleOrder     []string
}

func NewApp(cfg AppConfig) (*App, error) {
	driver := cfg.DB.Driver
	if driver == "" {
		driver = "sqlite"
	}

	fiberApp := fiber.New(fiber.Config{
		AppName:      "LowCode Engine",
		ErrorHandler: defaultErrorHandler,
		BodyLimit:    50 * 1024 * 1024,
	})

	jwtCfg := security.JWTConfig{Secret: cfg.JWTSecret, Expiration: cfg.Security.SessionDuration}
	templateEngine := tmpl.NewEngine()
	appCache := cache.NewCache(cfg.Cache)
	pluginMgr := plugin.NewManager()
	processReg := executor.NewProcessRegistry()
	wsHub := ws.NewHub()
	go wsHub.Run()
	translator := i18n.NewTranslator("en")
	settingStore := setting.NewStore()

	templateEngine.RegisterHelper("t", func(locale string, key string) string {
		return translator.Translate(locale, key)
	})

	modelReg := domainModel.NewRegistry()

	var db *gorm.DB
	var mongoConn *persistence.MongoConnection
	var seqEngine persistence.SequenceEngine
	var hydrator *expression.Hydrator
	var auditLogRepo *persistence.AuditLogRepository

	if driver == "mongodb" {
		conn, err := persistence.OpenMongoDB(cfg.DB)
		if err != nil {
			return nil, fmt.Errorf("mongodb connection failed: %w", err)
		}
		mongoConn = conn
		log.Printf("[DB] connected using mongodb driver")

		if err := persistence.MigrateMongoSystemTables(conn); err != nil {
			log.Printf("[WARN] failed to migrate mongo system tables: %v", err)
		}

		mongoSeq := persistence.NewMongoSequenceEngine(conn)
		seqEngine = mongoSeq
		hydrator = expression.NewHydrator(nil, modelReg)
	} else {
		var err error
		db, err = persistence.NewDatabase(cfg.DB)
		if err != nil {
			return nil, fmt.Errorf("database connection failed: %w", err)
		}
		log.Printf("[DB] connected using %s driver", driver)

		if err := persistence.AutoMigrateViewRevisions(db); err != nil {
			log.Printf("[WARN] failed to migrate view_revisions: %v", err)
		}
		if err := persistence.AutoMigrateDataRevisions(db); err != nil {
			log.Printf("[WARN] failed to migrate data_revisions: %v", err)
		}
		if err := infrastorage.AutoMigrateAttachments(db); err != nil {
			log.Printf("[WARN] failed to migrate attachments: %v", err)
		}
		if err := persistence.AutoMigrateAuditLog(db); err != nil {
			log.Printf("[WARN] failed to migrate audit_logs: %v", err)
		}
		if err := module.MigrateMigrationTable(db); err != nil {
			log.Printf("[WARN] failed to migrate ir_migration: %v", err)
		}

		gormSeq := persistence.NewGormSequenceEngine(db)
		if err := gormSeq.MigrateSequenceTable(); err != nil {
			log.Printf("[WARN] failed to migrate sequences table: %v", err)
		}
		seqEngine = gormSeq
		hydrator = expression.NewHydrator(db, modelReg)
		auditLogRepo = persistence.NewAuditLogRepository(db)
	}

	fmtEngine := format.NewEngine(seqEngine)
	pkGen := pkgen.NewGenerator(fmtEngine, nil)

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
		SettingStore:    settingStore,
		SequenceEngine:  seqEngine,
		FormatEngine:    fmtEngine,
		PKGenerator:     pkGen,
		Hydrator:        hydrator,
		AuditLogRepo:    auditLogRepo,
		MongoConn:       mongoConn,
		DBDriver:        driver,
		viewDefs:        make(map[string]*viewEntry),
		moduleMenus:     make(map[string][]parser.MenuItemDefinition),
	}

	if cfg.EncryptionKey != "" {
		enc, err := security.NewFieldEncryptor(cfg.EncryptionKey)
		if err != nil {
			log.Printf("[WARN] invalid encryption key: %v — field encryption disabled", err)
		} else {
			app.FieldEncryptor = enc
			log.Println("[SECURITY] field-level encryption enabled")
		}
	} else {
		log.Println("[SECURITY] no encryption key configured, field encryption disabled")
	}

	app.ViewRenderer.SetModelRegistry(app.ModelRegistry)
	app.ViewRenderer.SetHydrator(app.Hydrator)
	app.ViewRenderer.SetTableNameResolver(app.ModelRegistry)
	app.Hydrator.SetTableNameResolver(app.ModelRegistry)

	v := validation.NewValidator()
	v.SetTranslator(func(locale, key string) string {
		return translator.Translate(locale, key)
	})
	v.SetTableNameResolver(func(modelName string) string {
		return modelReg.TableName(modelName)
	})
	if db != nil {
		v.SetUniqueChecker(func(ctx context.Context, tableName string, fieldName string, value any, excludeID string, cfg *parser.UniqueConfig, isSoftDelete bool, data map[string]any) (bool, error) {
			q := db.WithContext(ctx).Table(tableName)
			if cfg != nil && cfg.CaseInsensitive {
				q = q.Where(fmt.Sprintf("LOWER(%s) = LOWER(?)", fieldName), value)
			} else {
				q = q.Where(fmt.Sprintf("%s = ?", fieldName), value)
			}
			if excludeID != "" {
				q = q.Where("id != ?", excludeID)
			}
			if isSoftDelete {
				if cfg == nil || !cfg.IncludeTrashed {
					q = q.Where("deleted_at IS NULL")
				}
			}
			if cfg != nil {
				for _, scope := range cfg.Scope {
					if scopeVal, ok := data[scope]; ok {
						q = q.Where(fmt.Sprintf("%s = ?", scope), scopeVal)
					}
				}
			}
			var count int64
			if err := q.Count(&count).Error; err != nil {
				return false, err
			}
			return count > 0, nil
		})
		v.SetExistsChecker(func(ctx context.Context, tableName string, id any, conditions map[string]any) (bool, error) {
			q := db.WithContext(ctx).Table(tableName).Where("id = ?", id)
			for k, val := range conditions {
				q = q.Where(fmt.Sprintf("%s = ?", k), val)
			}
			var count int64
			if err := q.Count(&count).Error; err != nil {
				return false, err
			}
			return count > 0, nil
		})
		v.SetRelationCounter(func(ctx context.Context, tableName string, foreignKey string, parentID any) (int64, error) {
			var count int64
			err := db.WithContext(ctx).Table(tableName).Where(fmt.Sprintf("%s = ?", foreignKey), parentID).Count(&count).Error
			return count, err
		})
	}
	v.SetCustomRunner(func(ctx context.Context, cv parser.CustomValidator, fieldName string, fieldValue any, data map[string]any, modulePath string) error {
		if cv.Process != "" {
			proc, err := processReg.LoadProcess(cv.Process)
			if err != nil {
				return fmt.Errorf("custom validator process %q not found: %w", cv.Process, err)
			}
			input := make(map[string]any)
			for k, val := range data {
				input[k] = val
			}
			input["_field"] = fieldName
			input["_value"] = fieldValue
			execCtx, err := app.Executor.Execute(ctx, proc, input, "")
			if err != nil {
				return err
			}
			if execCtx != nil && execCtx.Result != nil {
				if errStr, ok := execCtx.Result.(string); ok && errStr != "" {
					return fmt.Errorf("%s", errStr)
				}
			}
			return nil
		}
		if cv.Script != nil && pluginMgr != nil {
			scriptPath := cv.Script.File
			if !filepath.IsAbs(scriptPath) && modulePath != "" {
				resolved := filepath.Join(modulePath, scriptPath)
				if _, statErr := os.Stat(resolved); statErr == nil {
					scriptPath = resolved
				}
			}
			params := map[string]any{
				"field": fieldName,
				"value": fieldValue,
				"data":  data,
			}
			result, err := pluginMgr.Run(ctx, scriptPath, params)
			if err != nil {
				return err
			}
			if errStr, ok := result.(string); ok && errStr != "" {
				return fmt.Errorf("%s", errStr)
			}
		}
		return nil
	})
	app.Validator = validation.NewValidatorAdapter(v)
	app.Sanitizer = validation.NewSanitizer()

	hookDispatcher := hook.NewDispatcher(processReg, pluginMgr)
	hook.SetProcessExecutor(func(ctx context.Context, proc *parser.ProcessDefinition, input map[string]any, userID string) (any, error) {
		execCtx, err := app.Executor.Execute(ctx, proc, input, userID)
		if err != nil {
			return nil, err
		}
		if execCtx != nil {
			return execCtx.Result, nil
		}
		return nil, nil
	})
	app.HookDispatcher = hook.NewModelHookDispatcher(hookDispatcher)

	if db != nil {
		migEngine := module.NewMigrationEngine(db, modelReg)
		migEngine.SetScriptRunner(pluginMgr)
		app.MigrationEngine = migEngine
	}

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

func (a *App) createEmailSender() email.Sender {
	cfg := a.Config.SMTP
	if cfg.Host == "" {
		log.Println("[EMAIL] SMTP not configured, email features disabled")
		return email.NewNoopSender()
	}
	sender := email.NewSMTPSender(email.Config{
		Host:     cfg.Host,
		Port:     cfg.Port,
		User:     cfg.User,
		Password: cfg.Password,
		From:     cfg.From,
		TLS:      cfg.TLS,
	})
	log.Printf("[EMAIL] SMTP configured: %s:%d", cfg.Host, cfg.Port)
	return sender
}

func (a *App) registerStepHandlers() {
	a.Executor.RegisterHandler(parser.StepValidate, &steps.ValidateHandler{})
	dataHandler := &steps.DataHandler{DB: a.DB, Resolver: a.ModelRegistry}
	a.Executor.RegisterHandler(parser.StepQuery, dataHandler)
	a.Executor.RegisterHandler(parser.StepCreate, dataHandler)
	a.Executor.RegisterHandler(parser.StepUpdate, dataHandler)
	a.Executor.RegisterHandler(parser.StepDelete, dataHandler)
	a.Executor.RegisterHandler(parser.StepUpsert, dataHandler)
	a.Executor.RegisterHandler(parser.StepCount, dataHandler)
	a.Executor.RegisterHandler(parser.StepSum, dataHandler)
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
	if a.Config.IPWhitelist.Enabled {
		a.Fiber.Use(middleware.IPWhitelistMiddleware(a.Config.IPWhitelist))
	}
	if a.Config.RateLimit.Enabled {
		a.Fiber.Use(middleware.RateLimitMiddleware(a.Config.RateLimit))
	}
	if a.Config.Tenant.Enabled {
		a.Fiber.Use(middleware.TenantMiddleware(a.Config.Tenant))
	}
	auditRepo := persistence.NewAuditLogRepository(a.DB)
	a.AuditLogRepo = auditRepo
	a.Fiber.Use(middleware.PersistentAuditMiddleware(auditRepo, false))
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

	if a.Config.RateLimit.Enabled {
		a.Fiber.Use("/auth", middleware.AuthRateLimitMiddleware(a.Config.RateLimit))
	}
	emailSender := a.createEmailSender()
	authHandler := api.NewAuthHandlerFull(a.DB, a.JWTConfig, a.AuditLogRepo, a.Cache, emailSender)
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
	}, a.Config.ModuleDir, a.JWTConfig)
	adminPanel.RegisterRoutes(a.Fiber)

	a.setupComponentAssets()

	a.Fiber.Get("/", func(c *fiber.Ctx) error {
		return c.Redirect("/app/home")
	})
	a.Fiber.Get("/app", a.handleAppRoot)
	a.Fiber.Get("/app/login", func(c *fiber.Ctx) error {
		return c.Redirect("/app/auth/login")
	})
	a.Fiber.Post("/app/login", func(c *fiber.Ctx) error {
		return c.Redirect("/app/auth/login", 307)
	})
	a.Fiber.Get("/app/home", a.handleHomePage)
	a.Fiber.Post("/app/auth/login", a.handleAuthLoginPost)
	a.Fiber.Post("/app/auth/register", a.handleAuthRegisterPost)
	a.Fiber.Post("/app/auth/forgot", a.handleAuthForgotPost)
	a.Fiber.Post("/app/auth/reset", a.handleAuthResetPost)
	a.Fiber.Post("/app/auth/verify-2fa", a.handleAuthVerify2FAPost)
	a.Fiber.Get("/app/auth/logout", a.handleAuthLogout)
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
	return c.Redirect("/app/auth/login")
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
			return c.Redirect("/app/auth/login")
		}
		username = claims.Username
	} else {
		return c.Redirect("/app/auth/login")
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

func (a *App) handleAuthLoginPost(c *fiber.Ctx) error {
	username := c.FormValue("username")
	password := c.FormValue("password")
	next := sanitizeNextURL(c.Query("next", ""))

	if username == "" || password == "" {
		return c.Redirect("/app/auth/login?error=Username+and+password+required" + nextParam(next))
	}

	repo := persistence.NewGenericRepository(a.DB, a.ModelRegistry.TableName("user"))
	loginQuery := persistence.NewQuery().Where("username", "=", username)
	users, _, err := repo.FindAll(c.Context(), loginQuery, 1, 1)
	if (err != nil || len(users) == 0) && strings.Contains(username, "@") {
		emailQuery := persistence.NewQuery().Where("email", "=", username)
		users, _, err = repo.FindAll(c.Context(), emailQuery, 1, 1)
	}
	if err != nil || len(users) == 0 {
		return c.Redirect("/app/auth/login?error=Invalid+credentials" + nextParam(next))
	}

	user := users[0]
	hash, _ := user["password_hash"].(string)
	if !security.CheckPassword(password, hash) {
		return c.Redirect("/app/auth/login?error=Invalid+credentials" + nextParam(next))
	}

	userID, _ := user["id"].(string)
	uname, _ := user["username"].(string)

	token, err := security.GenerateToken(a.JWTConfig, userID, uname, nil, nil)
	if err != nil {
		return c.Redirect("/app/auth/login?error=Server+error" + nextParam(next))
	}

	sameSite := "Lax"
	if a.Config.Security.CookieSameSite != "" {
		sameSite = a.Config.Security.CookieSameSite
	}
	c.Cookie(&fiber.Cookie{
		Name:     "token",
		Value:    token,
		Path:     "/",
		HTTPOnly: true,
		Secure:   a.Config.Security.CookieSecure,
		SameSite: sameSite,
		MaxAge:   int(a.Config.Security.SessionDuration.Seconds()),
	})

	repo.Update(c.Context(), userID, map[string]any{"last_login": time.Now()})

	if a.AuditLogRepo != nil {
		a.AuditLogRepo.WriteAsync(persistence.AuditLogEntry{
			UserID:        userID,
			Action:        "login",
			ModelName:     "user",
			RecordID:      userID,
			IPAddress:     c.IP(),
			UserAgent:     c.Get("User-Agent"),
			RequestMethod: c.Method(),
			RequestPath:   c.Path(),
		})
	}

	if next != "" {
		return c.Redirect(next)
	}
	return c.Redirect("/app/home")
}

func (a *App) handleAuthRegisterPost(c *fiber.Ctx) error {
	if a.SettingStore.GetWithDefault("auth.register_enabled", "false") != "true" {
		return c.Redirect("/app/auth/register?error=Registration+is+disabled")
	}

	username := c.FormValue("username")
	email := c.FormValue("email")
	password := c.FormValue("password")
	confirmPassword := c.FormValue("confirm_password")

	if username == "" || email == "" || password == "" {
		return c.Redirect("/app/auth/register?error=All+fields+are+required")
	}
	if password != confirmPassword {
		return c.Redirect("/app/auth/register?error=Passwords+do+not+match")
	}

	hash, err := security.HashPassword(password)
	if err != nil {
		return c.Redirect("/app/auth/register?error=Invalid+password")
	}

	repo := persistence.NewGenericRepository(a.DB, a.ModelRegistry.TableName("user"))

	existing, _, _ := repo.FindAll(c.Context(), persistence.QueryFromDomain([][]any{{"username", "=", username}}), 1, 1)
	if len(existing) > 0 {
		return c.Redirect("/app/auth/register?error=Username+already+exists")
	}

	existingEmail, _, _ := repo.FindAll(c.Context(), persistence.QueryFromDomain([][]any{{"email", "=", email}}), 1, 1)
	if len(existingEmail) > 0 {
		return c.Redirect("/app/auth/register?error=Email+already+exists")
	}

	record := map[string]any{
		"id":            fmt.Sprintf("%s", newUUID()),
		"username":      username,
		"email":         email,
		"password_hash": hash,
		"active":        true,
	}

	if _, err := repo.Create(c.Context(), record); err != nil {
		return c.Redirect("/app/auth/register?error=Registration+failed")
	}

	if a.AuditLogRepo != nil {
		a.AuditLogRepo.WriteAsync(persistence.AuditLogEntry{
			UserID: record["id"].(string), Action: "register", ModelName: "user", RecordID: record["id"].(string),
			IPAddress: c.IP(), UserAgent: c.Get("User-Agent"), RequestMethod: c.Method(), RequestPath: c.Path(),
		})
	}

	return c.Redirect("/app/auth/login?success=Account+created.+Please+sign+in.")
}

func (a *App) handleAuthForgotPost(c *fiber.Ctx) error {
	emailAddr := c.FormValue("email")
	if emailAddr == "" {
		return c.Redirect("/app/auth/forgot?error=Email+is+required")
	}

	repo := persistence.NewGenericRepository(a.DB, a.ModelRegistry.TableName("user"))
	users, _, _ := repo.FindAll(c.Context(), persistence.QueryFromDomain([][]any{{"email", "=", emailAddr}}), 1, 1)
	// Always show success to prevent email enumeration
	if len(users) == 0 {
		return c.Redirect("/app/auth/forgot?success=If+an+account+exists+with+that+email,+a+code+has+been+sent.")
	}

	user := users[0]
	userID, _ := user["id"].(string)

	otpCode, err := security.GenerateOTP(6)
	if err != nil {
		return c.Redirect("/app/auth/forgot?error=Server+error")
	}

	a.Cache.Set(fmt.Sprintf("reset:%s", emailAddr), otpCode, 10*time.Minute)

	emailSender := a.createEmailSender()
	if emailSender.IsConfigured() {
		htmlBody, _ := email.RenderOTPEmail(otpCode, 10)
		go emailSender.Send(emailAddr, "Password Reset Code", htmlBody)
	}

	if a.AuditLogRepo != nil {
		a.AuditLogRepo.WriteAsync(persistence.AuditLogEntry{
			UserID: userID, Action: "forgot_password", ModelName: "user", RecordID: userID,
			IPAddress: c.IP(), UserAgent: c.Get("User-Agent"), RequestMethod: c.Method(), RequestPath: c.Path(),
		})
	}

	return c.Redirect("/app/auth/reset?email=" + emailAddr)
}

func (a *App) handleAuthResetPost(c *fiber.Ctx) error {
	emailAddr := c.FormValue("email")
	code := c.FormValue("code")
	password := c.FormValue("password")
	confirmPassword := c.FormValue("confirm_password")

	if code == "" || password == "" {
		return c.Redirect("/app/auth/reset?email=" + emailAddr + "&error=All+fields+are+required")
	}
	if password != confirmPassword {
		return c.Redirect("/app/auth/reset?email=" + emailAddr + "&error=Passwords+do+not+match")
	}

	cacheKey := fmt.Sprintf("reset:%s", emailAddr)
	cached, ok := a.Cache.Get(cacheKey)
	if !ok {
		return c.Redirect("/app/auth/forgot?error=Code+expired.+Please+try+again.")
	}

	storedCode, _ := cached.(string)
	if storedCode != code {
		return c.Redirect("/app/auth/reset?email=" + emailAddr + "&error=Invalid+code")
	}

	a.Cache.Delete(cacheKey)

	hash, err := security.HashPassword(password)
	if err != nil {
		return c.Redirect("/app/auth/reset?email=" + emailAddr + "&error=Invalid+password")
	}

	repo := persistence.NewGenericRepository(a.DB, a.ModelRegistry.TableName("user"))
	users, _, _ := repo.FindAll(c.Context(), persistence.QueryFromDomain([][]any{{"email", "=", emailAddr}}), 1, 1)
	if len(users) == 0 {
		return c.Redirect("/app/auth/login?error=User+not+found")
	}

	userID, _ := users[0]["id"].(string)
	repo.Update(c.Context(), userID, map[string]any{"password_hash": hash})

	if a.AuditLogRepo != nil {
		a.AuditLogRepo.WriteAsync(persistence.AuditLogEntry{
			UserID: userID, Action: "password_reset", ModelName: "user", RecordID: userID,
			IPAddress: c.IP(), UserAgent: c.Get("User-Agent"), RequestMethod: c.Method(), RequestPath: c.Path(),
		})
	}

	return c.Redirect("/app/auth/login?success=Password+reset+successful.+Please+sign+in.")
}

func (a *App) handleAuthVerify2FAPost(c *fiber.Ctx) error {
	tempToken := c.FormValue("temp_token")
	code := c.FormValue("code")

	if tempToken == "" || code == "" {
		return c.Redirect("/app/auth/login?error=Verification+failed")
	}

	claims, err := security.ValidateToken(a.JWTConfig, tempToken)
	if err != nil || claims.Purpose != "2fa" {
		return c.Redirect("/app/auth/login?error=Session+expired.+Please+login+again.")
	}

	cacheKey := fmt.Sprintf("otp:%s", claims.UserID)
	cached, ok := a.Cache.Get(cacheKey)
	if !ok {
		return c.Redirect("/app/auth/login?error=Code+expired.+Please+login+again.")
	}

	type otpEntry struct {
		Code     string
		Attempts int
	}
	entry, ok := cached.(*otpEntry)
	if !ok {
		a.Cache.Delete(cacheKey)
		return c.Redirect("/app/auth/login?error=Verification+failed")
	}

	if entry.Code != code {
		return c.Redirect("/app/auth/verify-2fa?error=Invalid+code&temp_token=" + tempToken)
	}

	a.Cache.Delete(cacheKey)

	token, err := security.GenerateToken(a.JWTConfig, claims.UserID, claims.Username, nil, nil)
	if err != nil {
		return c.Redirect("/app/auth/login?error=Server+error")
	}

	sameSite := "Lax"
	if a.Config.Security.CookieSameSite != "" {
		sameSite = a.Config.Security.CookieSameSite
	}
	c.Cookie(&fiber.Cookie{
		Name:     "token",
		Value:    token,
		Path:     "/",
		HTTPOnly: true,
		Secure:   a.Config.Security.CookieSecure,
		SameSite: sameSite,
		MaxAge:   int(a.Config.Security.SessionDuration.Seconds()),
	})

	return c.Redirect("/app/home")
}

func (a *App) handleAuthLogout(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	if a.AuditLogRepo != nil && userID != "" {
		a.AuditLogRepo.WriteAsync(persistence.AuditLogEntry{
			UserID: userID, Action: "logout", ModelName: "user", RecordID: userID,
			IPAddress: c.IP(), UserAgent: c.Get("User-Agent"), RequestMethod: c.Method(), RequestPath: c.Path(),
		})
	}
	c.Cookie(&fiber.Cookie{
		Name:   "token",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
	return c.Redirect("/app/auth/login")
}

func nextParam(next string) string {
	if next == "" {
		return ""
	}
	return "&next=" + next
}

func newUUID() string {
	return fmt.Sprintf("%s", uuid.New().String())
}

func (a *App) moduleRequiresAuth(moduleName string) bool {
	installed, err := a.ModuleRegistry.Get(moduleName)
	if err != nil {
		return true
	}
	return installed.Definition.RequiresAuth()
}

func sanitizeNextURL(next string) string {
	if next == "" {
		return ""
	}
	if !strings.HasPrefix(next, "/app/") {
		return ""
	}
	if strings.Contains(next, "://") || strings.Contains(next, "\\") || strings.HasPrefix(next, "//") {
		return ""
	}
	return next
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
	var validToken bool
	if token != "" {
		claims, _ := security.ValidateToken(a.JWTConfig, token)
		if claims != nil {
			username = claims.Username
			validToken = true
		}
	}

	if a.moduleRequiresAuth(moduleName) && !validToken {
		next := sanitizeNextURL(c.OriginalURL())
		redirectURL := "/app/auth/login"
		if next != "" {
			redirectURL += "?next=" + next
		}
		return c.Redirect(redirectURL)
	}

	locale := c.Query("lang", "en")
	if moduleName == "auth" {
		extraData := map[string]any{
			"Locale":            locale,
			"Error":             c.Query("error", ""),
			"Success":           c.Query("success", ""),
			"Next":              sanitizeNextURL(c.Query("next", "")),
			"RegisterEnabled":   a.SettingStore.GetWithDefault("auth.register_enabled", "false") == "true",
			"ChannelConfigured": a.Config.SMTP.Host != "",
		}
		if a.TemplateEngine.Has("templates/views/" + viewPath + ".html") {
			html, err := a.TemplateEngine.Render("templates/views/"+viewPath+".html", extraData)
			if err == nil {
				c.Set("Content-Type", "text/html; charset=utf-8")
				return c.SendString(html)
			}
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

	if a.moduleRequiresAuth(moduleName) {
		token := c.Cookies("token")
		if token == "" {
			token = extractBearerToken(c)
		}
		valid := false
		if token != "" {
			claims, _ := security.ValidateToken(a.JWTConfig, token)
			valid = claims != nil
		}
		if !valid {
			return c.Redirect("/app/auth/login")
		}
	}

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
	tableName := a.ModelRegistry.TableName(entry.Def.Model)
	var repo *persistence.GenericRepository
	if modelDef != nil {
		repo = persistence.NewGenericRepositoryWithModel(a.DB, tableName, modelDef)
	} else {
		repo = persistence.NewGenericRepository(a.DB, tableName)
	}

	revisionRepo := persistence.NewDataRevisionRepository(a.DB)
	repo.SetRevisionRepo(revisionRepo)
	repo.SetModelName(entry.Def.Model)
	repo.SetTableNameResolver(a.ModelRegistry)
	token := c.Cookies("token")
	if token != "" {
		if claims, clErr := security.ValidateToken(a.JWTConfig, token); clErr == nil {
			repo.SetCurrentUser(claims.UserID)
		}
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
		if vis == "admin" || vis == "none" {
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

		if err := a.ModelRegistry.RegisterWithModule(m, loaded.Definition); err != nil {
			return fmt.Errorf("failed to register model %s: %w", m.Name, err)
		}
		if a.DBDriver == "mongodb" {
			mongoMigration := persistence.NewMongoMigrationEngine(a.MongoConn)
			if err := mongoMigration.MigrateModel(m, a.ModelRegistry); err != nil {
				return fmt.Errorf("failed to migrate model %s: %w", m.Name, err)
			}
		} else {
			if err := persistence.MigrateModel(a.DB, m, a.ModelRegistry); err != nil {
				return fmt.Errorf("failed to migrate model %s: %w", m.Name, err)
			}
		}
	}

	revisionRepo := persistence.NewDataRevisionRepository(a.DB)
	router := api.NewRouterFull(a.Fiber, a.DB, a.WorkflowEngine, a.Hydrator, revisionRepo)
	if a.FieldEncryptor != nil {
		router.SetEncryptor(a.FieldEncryptor)
	}
	router.SetModelRegistry(a.ModelRegistry)
	router.SetTableNameResolver(a.ModelRegistry)
	if a.HookDispatcher != nil {
		router.SetHookDispatcher(a.HookDispatcher)
	}
	if a.Validator != nil {
		router.SetValidator(a.Validator)
	}
	if a.Sanitizer != nil {
		router.SetSanitizer(a.Sanitizer)
	}
	router.SetEventBus(a.EventBus)
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

	if err := module.SeedModule(a.DB, modPath, loaded.Definition.Data, a.ModelRegistry); err != nil {
		log.Printf("[WARN] seeding failed for %s: %v", modName, err)
	}

	if a.MigrationEngine != nil {
		a.MigrationEngine.SetProcessRunner(&appProcessRunner{
			executor: a.Executor,
			registry: a.ProcessRegistry,
		})
		migrations, err := module.CollectModuleMigrations(modPath, loaded.Definition.Migrations)
		if err != nil {
			log.Printf("[WARN] failed to discover migrations for %s: %v", modName, err)
		} else if len(migrations) > 0 {
			ctx := context.Background()
			count, err := a.MigrationEngine.RunUp(ctx, modPath, modName, migrations)
			if err != nil {
				log.Printf("[WARN] migration failed for %s: %v", modName, err)
			} else if count > 0 {
				log.Printf("[MIGRATION] %s: %d records seeded", modName, count)
			}
		}
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
<div><a href="/admin">Admin</a><a href="/health">Health</a><a href="/app/auth/logout">Logout</a></div>
</nav>
<div class="container">
<div class="welcome"><h2>Welcome, %s</h2><p>Select a module to get started.</p></div>
<div class="grid">%s</div>
</div></body></html>`, username, moduleCards.String())
}

type appProcessRunner struct {
	executor *executor.Executor
	registry *executor.ProcessRegistry
}

func (r *appProcessRunner) RunProcess(ctx context.Context, processName string, input map[string]any) (any, error) {
	proc, err := r.registry.LoadProcess(processName)
	if err != nil {
		return nil, fmt.Errorf("process %q not found: %w", processName, err)
	}
	execCtx, err := r.executor.Execute(ctx, proc, input, "")
	if err != nil {
		return nil, err
	}
	if execCtx != nil {
		return execCtx.Result, nil
	}
	return nil, nil
}
