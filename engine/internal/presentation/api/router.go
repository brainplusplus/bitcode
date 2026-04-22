package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/bitcode-engine/engine/internal/compiler/parser"
	"github.com/bitcode-engine/engine/internal/infrastructure/persistence"
	"github.com/bitcode-engine/engine/internal/runtime/workflow"
	"gorm.io/gorm"
)

type Router struct {
	app            *fiber.App
	db             *gorm.DB
	workflowEngine *workflow.Engine
}

func NewRouter(app *fiber.App, db *gorm.DB, wfEngine *workflow.Engine) *Router {
	return &Router{app: app, db: db, workflowEngine: wfEngine}
}

func (r *Router) RegisterAPI(apiDef *parser.APIDefinition) {
	basePath := apiDef.GetBasePath()
	endpoints := apiDef.ExpandAutoCRUD()

	group := r.app.Group(basePath)

	if apiDef.Model != "" {
		repo := persistence.NewGenericRepository(r.db, apiDef.Model+"s")
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
