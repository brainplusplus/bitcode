package api

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/bitcode-framework/bitcode/internal/compiler/parser"
)

type SwaggerGenerator struct {
	models []*parser.ModelDefinition
	apis   []*parser.APIDefinition
}

func NewSwaggerGenerator() *SwaggerGenerator {
	return &SwaggerGenerator{}
}

func (g *SwaggerGenerator) AddModel(model *parser.ModelDefinition) {
	g.models = append(g.models, model)
}

func (g *SwaggerGenerator) AddAPI(api *parser.APIDefinition) {
	g.apis = append(g.apis, api)
}

func (g *SwaggerGenerator) Generate() map[string]any {
	spec := map[string]any{
		"openapi": "3.0.3",
		"info": map[string]any{
			"title":       "BitCode API",
			"version":     "1.0.0",
			"description": "Auto-generated API documentation",
		},
		"paths":      g.generatePaths(),
		"components": g.generateComponents(),
		"security":   []map[string]any{{"bearerAuth": []string{}}},
	}
	return spec
}

func (g *SwaggerGenerator) generatePaths() map[string]any {
	paths := make(map[string]any)

	for _, apiDef := range g.apis {
		if apiDef.Model == "" {
			continue
		}
		basePath := apiDef.GetBasePath()
		endpoints := apiDef.ExpandAutoCRUD()

		for _, ep := range endpoints {
			fullPath := swaggerPath(basePath + ep.Path)
			if _, ok := paths[fullPath]; !ok {
				paths[fullPath] = make(map[string]any)
			}
			pathItem := paths[fullPath].(map[string]any)

			method := strings.ToLower(ep.Method)
			operation := g.buildOperation(apiDef.Model, ep)
			pathItem[method] = operation
		}
	}

	return paths
}

func (g *SwaggerGenerator) buildOperation(modelName string, ep parser.EndpointDefinition) map[string]any {
	op := map[string]any{
		"tags":    []string{modelName},
		"summary": fmt.Sprintf("%s %s", ep.Action, modelName),
		"responses": map[string]any{
			"200": map[string]any{"description": "Success"},
			"401": map[string]any{"description": "Unauthorized"},
			"403": map[string]any{"description": "Forbidden"},
			"404": map[string]any{"description": "Not Found"},
		},
	}

	if strings.Contains(ep.Path, ":id") {
		op["parameters"] = []map[string]any{
			{
				"name":     "id",
				"in":       "path",
				"required": true,
				"schema":   map[string]any{"type": "string"},
			},
		}
	}

	if ep.Method == "POST" || ep.Method == "PUT" || ep.Method == "PATCH" {
		op["requestBody"] = map[string]any{
			"required": true,
			"content": map[string]any{
				"application/json": map[string]any{
					"schema": map[string]any{"$ref": "#/components/schemas/" + modelName},
				},
			},
		}
	}

	if ep.Action == "list" {
		op["parameters"] = []map[string]any{
			{"name": "page", "in": "query", "schema": map[string]any{"type": "integer", "default": 1}},
			{"name": "page_size", "in": "query", "schema": map[string]any{"type": "integer", "default": 20}},
			{"name": "q", "in": "query", "schema": map[string]any{"type": "string"}, "description": "Search query"},
		}
	}

	return op
}

func (g *SwaggerGenerator) generateComponents() map[string]any {
	schemas := make(map[string]any)

	for _, model := range g.models {
		if model.API == nil {
			continue
		}
		properties := make(map[string]any)
		properties["id"] = map[string]any{"type": "string", "format": "uuid"}

		for name, field := range model.Fields {
			properties[name] = fieldToSwaggerSchema(field)
		}

		schemas[model.Name] = map[string]any{
			"type":       "object",
			"properties": properties,
		}
	}

	return map[string]any{
		"schemas": schemas,
		"securitySchemes": map[string]any{
			"bearerAuth": map[string]any{
				"type":         "http",
				"scheme":       "bearer",
				"bearerFormat": "JWT",
			},
		},
	}
}

func fieldToSwaggerSchema(field parser.FieldDefinition) map[string]any {
	switch field.Type {
	case parser.FieldString, parser.FieldEmail, parser.FieldSmallText, parser.FieldBarcode:
		s := map[string]any{"type": "string"}
		if field.Max > 0 {
			s["maxLength"] = field.Max
		}
		return s
	case parser.FieldText, parser.FieldRichText, parser.FieldMarkdown, parser.FieldHTML, parser.FieldCode:
		return map[string]any{"type": "string"}
	case parser.FieldInteger:
		return map[string]any{"type": "integer"}
	case parser.FieldFloat, parser.FieldDecimal, parser.FieldCurrency, parser.FieldPercent:
		return map[string]any{"type": "number"}
	case parser.FieldBoolean, parser.FieldToggle:
		return map[string]any{"type": "boolean"}
	case parser.FieldDate:
		return map[string]any{"type": "string", "format": "date"}
	case parser.FieldDatetime:
		return map[string]any{"type": "string", "format": "date-time"}
	case parser.FieldSelection, parser.FieldRadio:
		s := map[string]any{"type": "string"}
		if len(field.Options) > 0 {
			s["enum"] = field.Options
		}
		return s
	case parser.FieldJSON:
		return map[string]any{"type": "object"}
	case parser.FieldMany2One:
		return map[string]any{"type": "string", "description": "FK to " + field.Model}
	case parser.FieldOne2Many:
		return map[string]any{"type": "array", "items": map[string]any{"type": "object"}}
	case parser.FieldMany2Many:
		return map[string]any{"type": "array", "items": map[string]any{"type": "string"}}
	case parser.FieldFile, parser.FieldImage:
		return map[string]any{"type": "string", "format": "uri"}
	case parser.FieldPassword:
		return map[string]any{"type": "string", "format": "password"}
	case parser.FieldColor:
		return map[string]any{"type": "string", "pattern": "^#[0-9a-fA-F]{6}$"}
	case parser.FieldRating:
		return map[string]any{"type": "integer", "minimum": 0, "maximum": 5}
	default:
		return map[string]any{"type": "string"}
	}
}

func swaggerPath(path string) string {
	return strings.ReplaceAll(path, ":id", "{id}")
}

func RegisterSwaggerRoutes(app *fiber.App, generator *SwaggerGenerator) {
	spec := generator.Generate()
	specJSON, _ := json.MarshalIndent(spec, "", "  ")

	app.Get("/api/v1/docs/openapi.json", func(c *fiber.Ctx) error {
		c.Set("Content-Type", "application/json")
		return c.Send(specJSON)
	})

	app.Get("/api/v1/docs", func(c *fiber.Ctx) error {
		c.Set("Content-Type", "text/html; charset=utf-8")
		return c.SendString(swaggerUIHTML())
	})
}

func swaggerUIHTML() string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>BitCode API Docs</title>
<link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
</head>
<body>
<div id="swagger-ui"></div>
<script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
<script>
SwaggerUIBundle({
  url: '/api/v1/docs/openapi.json',
  dom_id: '#swagger-ui',
  presets: [SwaggerUIBundle.presets.apis, SwaggerUIBundle.SwaggerUIStandalonePreset],
  layout: 'BaseLayout'
});
</script>
</body>
</html>`
}
