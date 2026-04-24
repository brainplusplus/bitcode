package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/bitcode-engine/engine/internal/compiler/parser"
	"github.com/bitcode-engine/engine/internal/infrastructure/persistence"
	"github.com/bitcode-engine/engine/internal/runtime/expression"
	"github.com/bitcode-engine/engine/internal/runtime/workflow"
	"github.com/bitcode-engine/engine/pkg/security"
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
		crud := NewCRUDHandler(repo, apiDef, r.workflowEngine)

		for _, ep := range endpoints {
			handler := r.resolveHandler(crud, ep)
			switch ep.Method {
			case "GET":
				group.Get(ep.Path, handler)
			case "POST":
				group.Post(ep.Path, handler)
			case "PUT":
				group.Put(ep.Path, handler)
			case "DELETE":
				group.Delete(ep.Path, handler)
			case "PATCH":
				group.Patch(ep.Path, handler)
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
