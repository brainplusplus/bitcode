package api

import (
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/bitcode-engine/engine/internal/compiler/parser"
	"github.com/bitcode-engine/engine/internal/infrastructure/persistence"
	"github.com/bitcode-engine/engine/internal/runtime/pkgen"
	"github.com/bitcode-engine/engine/internal/runtime/workflow"
)

type CRUDHandler struct {
	repo           *persistence.GenericRepository
	apiDef         *parser.APIDefinition
	workflowEngine *workflow.Engine
	modelDef       *parser.ModelDefinition
	pkGenerator    *pkgen.Generator
}

func NewCRUDHandler(repo *persistence.GenericRepository, apiDef *parser.APIDefinition, wfEngine *workflow.Engine) *CRUDHandler {
	return &CRUDHandler{repo: repo, apiDef: apiDef, workflowEngine: wfEngine}
}

func NewCRUDHandlerWithPK(repo *persistence.GenericRepository, apiDef *parser.APIDefinition, wfEngine *workflow.Engine, modelDef *parser.ModelDefinition, pkGen *pkgen.Generator) *CRUDHandler {
	return &CRUDHandler{repo: repo, apiDef: apiDef, workflowEngine: wfEngine, modelDef: modelDef, pkGenerator: pkGen}
}

func (h *CRUDHandler) List(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	pageSize, _ := strconv.Atoi(c.Query("page_size", strconv.Itoa(h.apiDef.GetPageSize())))

	var filters [][]any

	rlsFilters, ok := c.Locals("rls_filters").([][]any)
	if ok && len(rlsFilters) > 0 {
		filters = append(filters, rlsFilters...)
	}

	searchQuery := c.Query("q", "")
	if searchQuery != "" && len(h.apiDef.Search) > 0 {
		for _, field := range h.apiDef.Search {
			filters = append(filters, []any{field, "like", "%" + searchQuery + "%"})
		}
	}

	for _, field := range []string{"status", "type", "source", "priority", "department_id", "position_id", "assigned_to"} {
		if val := c.Query(field); val != "" {
			filters = append(filters, []any{field, "=", val})
		}
	}

	results, total, err := h.repo.FindAll(c.Context(), persistence.QueryFromDomain(filters), page, pageSize)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	totalPages := (total + int64(pageSize) - 1) / int64(pageSize)

	return c.JSON(fiber.Map{
		"data":        results,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": totalPages,
	})
}

func (h *CRUDHandler) Read(c *fiber.Ctx) error {
	id := c.Params("id")

	if h.modelDef != nil && pkgen.IsCompositeNoSurrogate(h.modelDef) {
		keys, err := pkgen.DecodeCompositePK(id)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid composite key"})
		}
		result, err := h.repo.FindByCompositePK(c.Context(), keys)
		if err != nil {
			return c.Status(404).JSON(fiber.Map{"error": "record not found"})
		}
		return c.JSON(fiber.Map{"data": result})
	}

	result, err := h.repo.FindByID(c.Context(), id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "record not found"})
	}
	return c.JSON(fiber.Map{"data": result})
}

func (h *CRUDHandler) Create(c *fiber.Ctx) error {
	var body map[string]any
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	if userID, ok := c.Locals("user_id").(string); ok {
		h.repo.SetCurrentUser(userID)
	}

	session := h.extractSession(c)

	if h.pkGenerator != nil && h.modelDef != nil {
		pkCol, pkVal, err := h.pkGenerator.GeneratePK(h.modelDef, body, session)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": fmt.Sprintf("failed to generate primary key: %v", err)})
		}
		if pkCol != "" && pkVal != nil {
			body[pkCol] = pkVal
		}

		for fieldName, fieldDef := range h.modelDef.Fields {
			if fieldDef.AutoFormat != nil {
				val, afErr := h.pkGenerator.GenerateAutoFormat(h.modelDef, fieldName, &fieldDef, body, session)
				if afErr == nil {
					body[fieldName] = val
				}
			}
		}
	} else {
		if _, hasID := body["id"]; !hasID {
			body["id"] = generateUUID()
		}
	}

	if userID, ok := c.Locals("user_id").(string); ok {
		body["created_by"] = userID
		body["updated_by"] = userID
	}

	if h.apiDef.Workflow != "" && h.workflowEngine != nil {
		initialState, err := h.workflowEngine.GetInitialState(h.apiDef.Workflow)
		if err == nil && initialState != "" {
			wf, _ := h.workflowEngine.Get(h.apiDef.Workflow)
			if wf != nil {
				body[wf.Field] = initialState
			}
		}
	}

	result, err := h.repo.Create(c.Context(), body)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(201).JSON(fiber.Map{"data": result})
}

func (h *CRUDHandler) Update(c *fiber.Ctx) error {
	id := c.Params("id")

	var body map[string]any
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	if userID, ok := c.Locals("user_id").(string); ok {
		h.repo.SetCurrentUser(userID)
	}

	delete(body, "id")
	delete(body, "created_at")
	delete(body, "created_by")

	if h.modelDef != nil && h.modelDef.PrimaryKey != nil {
		if h.modelDef.PrimaryKey.Field != "" {
			delete(body, h.modelDef.PrimaryKey.Field)
		}
		for _, f := range h.modelDef.PrimaryKey.Fields {
			delete(body, f)
		}
	}

	if userID, ok := c.Locals("user_id").(string); ok {
		body["updated_by"] = userID
	}

	if h.modelDef != nil && pkgen.IsCompositeNoSurrogate(h.modelDef) {
		keys, err := pkgen.DecodeCompositePK(id)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid composite key"})
		}
		if err := h.repo.UpdateByCompositePK(c.Context(), keys, body); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"message": "updated"})
	}

	if err := h.repo.Update(c.Context(), id, body); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "updated"})
}

func (h *CRUDHandler) Delete(c *fiber.Ctx) error {
	id := c.Params("id")

	if userID, ok := c.Locals("user_id").(string); ok {
		h.repo.SetCurrentUser(userID)
	}

	var err error
	if h.apiDef.IsSoftDelete() {
		err = h.repo.Delete(c.Context(), id)
	} else {
		err = h.repo.HardDelete(c.Context(), id)
	}

	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "deleted"})
}

func (h *CRUDHandler) WorkflowAction(actionName string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Params("id")

		record, err := h.repo.FindByID(c.Context(), id)
		if err != nil {
			return c.Status(404).JSON(fiber.Map{"error": "record not found"})
		}

		if h.workflowEngine == nil || h.apiDef.Workflow == "" {
			return c.Status(400).JSON(fiber.Map{"error": "no workflow configured"})
		}

		wf, err := h.workflowEngine.Get(h.apiDef.Workflow)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		currentState := fmt.Sprintf("%v", record[wf.Field])

		newState, err := h.workflowEngine.ExecuteTransition(c.Context(), h.apiDef.Workflow, currentState, actionName)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}

		updateData := map[string]any{wf.Field: newState}

		var updateBody map[string]any
		c.BodyParser(&updateBody)
		for k, v := range updateBody {
			updateData[k] = v
		}

		if userID, ok := c.Locals("user_id").(string); ok {
			updateData["updated_by"] = userID
		}

		if err := h.repo.Update(c.Context(), id, updateData); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{
			"message":    fmt.Sprintf("%s completed", actionName),
			"new_status": newState,
			"id":         id,
		})
	}
}

func (h *CRUDHandler) extractSession(c *fiber.Ctx) map[string]any {
	session := make(map[string]any)
	if userID, ok := c.Locals("user_id").(string); ok {
		session["user_id"] = userID
	}
	if username, ok := c.Locals("username").(string); ok {
		session["username"] = username
	}
	if tenantID, ok := c.Locals("tenant_id").(string); ok {
		session["tenant_id"] = tenantID
	}
	if groupID, ok := c.Locals("group_id").(string); ok {
		session["group_id"] = groupID
	}
	if groupCode, ok := c.Locals("group_code").(string); ok {
		session["group_code"] = groupCode
	}
	return session
}

func generateUUID() string {
	return uuid.New().String()
}
