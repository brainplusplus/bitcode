package graphql

import (
	"context"
	"encoding/json"

	"github.com/gofiber/fiber/v2"
	gql "github.com/graphql-go/graphql"
)

type Handler struct {
	schema *gql.Schema
}

func NewHandler(schema *gql.Schema) *Handler {
	return &Handler{schema: schema}
}

func (h *Handler) RegisterRoutes(app *fiber.App) {
	app.Post("/api/v1/graphql", h.handleQuery)
	app.Get("/api/v1/graphql", h.handleQuery)
}

func (h *Handler) handleQuery(c *fiber.Ctx) error {
	var params struct {
		Query         string         `json:"query"`
		OperationName string         `json:"operationName"`
		Variables     map[string]any `json:"variables"`
	}

	if c.Method() == "GET" {
		params.Query = c.Query("query")
		params.OperationName = c.Query("operationName")
	} else {
		if err := c.BodyParser(&params); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
		}
	}

	if params.Query == "" {
		return c.Status(400).JSON(fiber.Map{"error": "query is required"})
	}

	ctx := c.Context()
	goCtx := context.Background()
	if userID, ok := c.Locals("user_id").(string); ok {
		goCtx = context.WithValue(goCtx, contextKeyUserID, userID)
	}
	_ = ctx

	result := gql.Do(gql.Params{
		Schema:         *h.schema,

		RequestString:  params.Query,
		OperationName:  params.OperationName,
		VariableValues: params.Variables,
		Context:        goCtx,
	})

	if len(result.Errors) > 0 {
		errMsgs := make([]map[string]any, len(result.Errors))
		for i, e := range result.Errors {
			errMsgs[i] = map[string]any{
				"message":   e.Message,
				"locations": e.Locations,
				"path":      e.Path,
			}
		}
		response := map[string]any{
			"data":   result.Data,
			"errors": errMsgs,
		}
		data, _ := json.Marshal(response)
		c.Set("Content-Type", "application/json")
		return c.Send(data)
	}

	response := map[string]any{"data": result.Data}
	data, _ := json.Marshal(response)
	c.Set("Content-Type", "application/json")
	return c.Send(data)
}
