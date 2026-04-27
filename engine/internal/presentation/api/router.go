package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/bitcode-framework/bitcode/internal/compiler/parser"
	"github.com/bitcode-framework/bitcode/internal/infrastructure/persistence"
	"github.com/bitcode-framework/bitcode/internal/presentation/middleware"
	"github.com/bitcode-framework/bitcode/internal/runtime/expression"
	"github.com/bitcode-framework/bitcode/internal/runtime/workflow"
	"github.com/bitcode-framework/bitcode/pkg/security"
	"gorm.io/gorm"
)

type Router struct {
	app               *fiber.App
	db                *gorm.DB
	workflowEngine    *workflow.Engine
	hydrator          *expression.Hydrator
	revisionRepo      *persistence.DataRevisionRepository
	encryptor         *security.FieldEncryptor
	modelRegistry     interface{ Get(string) (*parser.ModelDefinition, error) }
	tableNameResolver interface{ TableName(string) string }
	hookDispatcher    persistence.HookDispatcher
	validator         persistence.FieldValidator
	sanitizer         persistence.FieldSanitizer
	eventBus          persistence.EventPublisher
	permissionService *persistence.PermissionService
	recordRuleService *persistence.RecordRuleService
}

func NewRouter(app *fiber.App, db *gorm.DB, wfEngine *workflow.Engine) *Router {
	return &Router{app: app, db: db, workflowEngine: wfEngine}
}

func NewRouterWithHydrator(app *fiber.App, db *gorm.DB, wfEngine *workflow.Engine, hydrator *expression.Hydrator) *Router {
	return &Router{app: app, db: db, workflowEngine: wfEngine, hydrator: hydrator}
}

func NewRouterFull(app *fiber.App, db *gorm.DB, wfEngine *workflow.Engine, hydrator *expression.Hydrator, revRepo *persistence.DataRevisionRepository) *Router {
	return &Router{app: app, db: db, workflowEngine: wfEngine, hydrator: hydrator, revisionRepo: revRepo}
}

func (r *Router) SetEncryptor(enc *security.FieldEncryptor) {
	r.encryptor = enc
}

func (r *Router) SetModelRegistry(reg interface{ Get(string) (*parser.ModelDefinition, error) }) {
	r.modelRegistry = reg
}

func (r *Router) SetTableNameResolver(resolver interface{ TableName(string) string }) {
	r.tableNameResolver = resolver
}

func (r *Router) SetHookDispatcher(d persistence.HookDispatcher) {
	r.hookDispatcher = d
}

func (r *Router) SetValidator(v persistence.FieldValidator) {
	r.validator = v
}

func (r *Router) SetSanitizer(s persistence.FieldSanitizer) {
	r.sanitizer = s
}

func (r *Router) SetEventBus(bus persistence.EventPublisher) {
	r.eventBus = bus
}

func (r *Router) SetPermissionService(svc *persistence.PermissionService) {
	r.permissionService = svc
}

func (r *Router) SetRecordRuleService(svc *persistence.RecordRuleService) {
	r.recordRuleService = svc
}

func (r *Router) RegisterAPI(apiDef *parser.APIDefinition) {
	basePath := apiDef.GetBasePath()
	endpoints := apiDef.ExpandAutoCRUD()

	group := r.app.Group(basePath)

	if apiDef.Model != "" {
		var modelDef *parser.ModelDefinition
		if r.modelRegistry != nil {
			modelDef, _ = r.modelRegistry.Get(apiDef.Model)
		}
		tableName := apiDef.Model
		if r.tableNameResolver != nil {
			tableName = r.tableNameResolver.TableName(apiDef.Model)
		}
		var repo *persistence.GenericRepository
		if modelDef != nil {
			repo = persistence.NewGenericRepositoryWithModel(r.db, tableName, modelDef)
		} else {
			repo = persistence.NewGenericRepository(r.db, tableName)
		}
		if r.hydrator != nil {
			repo.SetHydrator(r.hydrator)
		}
		if r.revisionRepo != nil {
			repo.SetRevisionRepo(r.revisionRepo)
			repo.SetModelName(apiDef.Model)
		}
		if r.encryptor != nil {
			repo.SetEncryptor(r.encryptor)
		}
		if r.tableNameResolver != nil {
			if tnr, ok := r.tableNameResolver.(persistence.TableNameResolver); ok {
				repo.SetTableNameResolver(tnr)
			}
		}
		if r.hookDispatcher != nil {
			repo.SetHookDispatcher(r.hookDispatcher)
		}
		if r.validator != nil {
			repo.SetValidator(r.validator)
		}
		if r.sanitizer != nil {
			repo.SetSanitizer(r.sanitizer)
		}
		if r.eventBus != nil {
			repo.SetEventBus(r.eventBus)
		}
		crud := NewCRUDHandler(repo, apiDef, r.workflowEngine)
		crud.modelDef = modelDef
		crud.hookDispatcher = r.hookDispatcher
		if r.permissionService != nil {
			crud.permissionService = r.permissionService
		}

		if apiDef.Model != "" && crud.modelDef != nil && crud.modelDef.Events != nil && len(crud.modelDef.Events.OnChange) > 0 {
			group.Post("/onchange", crud.OnChange)
		}

		for _, ep := range endpoints {
			handler := r.resolveHandler(crud, ep)
			var handlers []fiber.Handler

			if r.permissionService != nil && len(ep.Permissions) > 0 {
				handlers = append(handlers, middleware.PermissionMiddleware(r.permissionService, ep.Permissions))
			}

			if r.recordRuleService != nil && apiDef.Model != "" {
				op := actionToOperation(ep.Action)
				if op == "read" || op == "write" || op == "create" || op == "delete" {
					handlers = append(handlers, middleware.RecordRuleMiddleware(r.recordRuleService, apiDef.Model, op))
				}
			}

			handlers = append(handlers, handler)

			switch ep.Method {
			case "GET":
				group.Get(ep.Path, handlers...)
			case "POST":
				group.Post(ep.Path, handlers...)
			case "PUT":
				group.Put(ep.Path, handlers...)
			case "DELETE":
				group.Delete(ep.Path, handlers...)
			case "PATCH":
				group.Patch(ep.Path, handlers...)
			}
		}
		return
	}

	for _, ep := range endpoints {
		handler := func(c *fiber.Ctx) error {
			return c.Status(501).JSON(fiber.Map{"error": "custom handlers not yet implemented"})
		}
		switch ep.Method {
		case "GET":
			group.Get(ep.Path, handler)
		case "POST":
			group.Post(ep.Path, handler)
		}
	}
}

func actionToOperation(action string) string {
	switch action {
	case "list", "read":
		return "read"
	case "create":
		return "create"
	case "update":
		return "write"
	case "delete":
		return "delete"
	default:
		return "write"
	}
}

func (r *Router) resolveHandler(crud *CRUDHandler, ep parser.EndpointDefinition) fiber.Handler {
	switch ep.Action {
	case "list":
		return crud.List
	case "read":
		return crud.Read
	case "create":
		return crud.Create
	case "update":
		return crud.Update
	case "delete":
		return crud.Delete
	default:
		if crud.apiDef.Actions != nil {
			if _, isWorkflowAction := crud.apiDef.Actions[ep.Action]; isWorkflowAction {
				return crud.WorkflowAction(ep.Action)
			}
		}
		return func(c *fiber.Ctx) error {
			return c.Status(501).JSON(fiber.Map{"error": "action not implemented", "action": ep.Action})
		}
	}
}
