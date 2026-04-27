package websocket

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/bitcode-framework/bitcode/internal/compiler/parser"
	"github.com/bitcode-framework/bitcode/internal/infrastructure/persistence"
	"gorm.io/gorm"
)

type CRUDRequest struct {
	ID        string         `json:"id"`
	Type      string         `json:"type"`
	Model     string         `json:"model"`
	Action    string         `json:"action"`
	RecordID  string         `json:"record_id,omitempty"`
	Data      map[string]any `json:"data,omitempty"`
	Page      int            `json:"page,omitempty"`
	PageSize  int            `json:"page_size,omitempty"`
	Query     string         `json:"q,omitempty"`
}

type CRUDResponse struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Model   string `json:"model"`
	Action  string `json:"action"`
	Success bool   `json:"success"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

type ModelRegistry interface {
	Get(name string) (*parser.ModelDefinition, error)
	TableName(name string) string
}

type CRUDHandler struct {
	db                *gorm.DB
	modelRegistry     ModelRegistry
	permissionService *persistence.PermissionService
	recordRuleService *persistence.RecordRuleService
	enabledModels     map[string]bool
}

func NewCRUDHandler(db *gorm.DB, modelReg ModelRegistry, permSvc *persistence.PermissionService, rrSvc *persistence.RecordRuleService) *CRUDHandler {
	return &CRUDHandler{
		db:                db,
		modelRegistry:     modelReg,
		permissionService: permSvc,
		recordRuleService: rrSvc,
		enabledModels:     make(map[string]bool),
	}
}

func (h *CRUDHandler) EnableModel(modelName string) {
	h.enabledModels[modelName] = true
}

func (h *CRUDHandler) HandleMessage(client *Client, msgBytes []byte) {
	var req CRUDRequest
	if err := json.Unmarshal(msgBytes, &req); err != nil {
		return
	}

	if req.Type != "crud" {
		return
	}

	if !h.enabledModels[req.Model] {
		client.Send(Message{
			Type: "crud_response",
			Data: CRUDResponse{ID: req.ID, Type: "crud_response", Model: req.Model, Action: req.Action, Success: false, Error: "model not enabled for websocket"},
		})
		return
	}

	var resp CRUDResponse
	switch req.Action {
	case "list":
		resp = h.handleList(client, req)
	case "read":
		resp = h.handleRead(client, req)
	case "create":
		resp = h.handleCreate(client, req)
	case "update":
		resp = h.handleUpdate(client, req)
	case "delete":
		resp = h.handleDelete(client, req)
	default:
		resp = CRUDResponse{ID: req.ID, Type: "crud_response", Model: req.Model, Action: req.Action, Success: false, Error: fmt.Sprintf("unknown action: %s", req.Action)}
	}

	client.Send(Message{Type: "crud_response", Data: resp})
}

func (h *CRUDHandler) getRepo(modelName string) *persistence.GenericRepository {
	tableName := h.modelRegistry.TableName(modelName)
	modelDef, _ := h.modelRegistry.Get(modelName)
	if modelDef != nil {
		return persistence.NewGenericRepositoryWithModel(h.db, tableName, modelDef)
	}
	return persistence.NewGenericRepository(h.db, tableName)
}

func (h *CRUDHandler) checkPermission(userID, modelName, operation string) error {
	if h.permissionService == nil || userID == "" {
		return nil
	}
	allowed, err := h.permissionService.UserHasPermission(userID, modelName+"."+operation)
	if err != nil {
		return err
	}
	if !allowed {
		return fmt.Errorf("permission denied: %s.%s", modelName, operation)
	}
	return nil
}

func (h *CRUDHandler) handleList(client *Client, req CRUDRequest) CRUDResponse {
	if err := h.checkPermission(client.UserID, req.Model, "read"); err != nil {
		return CRUDResponse{ID: req.ID, Type: "crud_response", Model: req.Model, Action: "list", Success: false, Error: err.Error()}
	}

	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}

	var filters [][]any
	if h.recordRuleService != nil && client.UserID != "" {
		rlsFilters, _ := h.recordRuleService.GetFilters(client.UserID, req.Model, "read")
		if len(rlsFilters) > 0 {
			vars := map[string]string{"user.id": client.UserID}
			rlsFilters = persistence.InterpolateDomainFilters(rlsFilters, vars)
			filters = append(filters, rlsFilters...)
		}
	}

	var query *persistence.Query
	if len(filters) > 0 {
		query = persistence.QueryFromDomain(filters)
	}

	repo := h.getRepo(req.Model)
	results, total, err := repo.FindAll(context.Background(), query, page, pageSize)
	if err != nil {
		return CRUDResponse{ID: req.ID, Type: "crud_response", Model: req.Model, Action: "list", Success: false, Error: err.Error()}
	}

	totalPages := (total + int64(pageSize) - 1) / int64(pageSize)

	return CRUDResponse{
		ID: req.ID, Type: "crud_response", Model: req.Model, Action: "list", Success: true,
		Data: map[string]any{"data": results, "total": total, "page": page, "page_size": pageSize, "total_pages": totalPages},
	}
}

func (h *CRUDHandler) handleRead(client *Client, req CRUDRequest) CRUDResponse {
	if err := h.checkPermission(client.UserID, req.Model, "read"); err != nil {
		return CRUDResponse{ID: req.ID, Type: "crud_response", Model: req.Model, Action: "read", Success: false, Error: err.Error()}
	}

	if req.RecordID == "" {
		return CRUDResponse{ID: req.ID, Type: "crud_response", Model: req.Model, Action: "read", Success: false, Error: "record_id required"}
	}

	repo := h.getRepo(req.Model)
	result, err := repo.FindByID(context.Background(), req.RecordID)
	if err != nil {
		return CRUDResponse{ID: req.ID, Type: "crud_response", Model: req.Model, Action: "read", Success: false, Error: "record not found"}
	}

	return CRUDResponse{ID: req.ID, Type: "crud_response", Model: req.Model, Action: "read", Success: true, Data: result}
}

func (h *CRUDHandler) handleCreate(client *Client, req CRUDRequest) CRUDResponse {
	if err := h.checkPermission(client.UserID, req.Model, "create"); err != nil {
		return CRUDResponse{ID: req.ID, Type: "crud_response", Model: req.Model, Action: "create", Success: false, Error: err.Error()}
	}

	if req.Data == nil {
		return CRUDResponse{ID: req.ID, Type: "crud_response", Model: req.Model, Action: "create", Success: false, Error: "data required"}
	}

	if _, hasID := req.Data["id"]; !hasID {
		req.Data["id"] = uuid.New().String()
	}
	if client.UserID != "" {
		req.Data["created_by"] = client.UserID
		req.Data["updated_by"] = client.UserID
	}

	repo := h.getRepo(req.Model)
	result, err := repo.Create(context.Background(), req.Data)
	if err != nil {
		return CRUDResponse{ID: req.ID, Type: "crud_response", Model: req.Model, Action: "create", Success: false, Error: err.Error()}
	}

	return CRUDResponse{ID: req.ID, Type: "crud_response", Model: req.Model, Action: "create", Success: true, Data: result}
}

func (h *CRUDHandler) handleUpdate(client *Client, req CRUDRequest) CRUDResponse {
	if err := h.checkPermission(client.UserID, req.Model, "write"); err != nil {
		return CRUDResponse{ID: req.ID, Type: "crud_response", Model: req.Model, Action: "update", Success: false, Error: err.Error()}
	}

	if req.RecordID == "" || req.Data == nil {
		return CRUDResponse{ID: req.ID, Type: "crud_response", Model: req.Model, Action: "update", Success: false, Error: "record_id and data required"}
	}

	delete(req.Data, "id")
	delete(req.Data, "created_at")
	delete(req.Data, "created_by")
	if client.UserID != "" {
		req.Data["updated_by"] = client.UserID
	}

	repo := h.getRepo(req.Model)
	if err := repo.Update(context.Background(), req.RecordID, req.Data); err != nil {
		return CRUDResponse{ID: req.ID, Type: "crud_response", Model: req.Model, Action: "update", Success: false, Error: err.Error()}
	}

	return CRUDResponse{ID: req.ID, Type: "crud_response", Model: req.Model, Action: "update", Success: true, Data: map[string]any{"message": "updated"}}
}

func (h *CRUDHandler) handleDelete(client *Client, req CRUDRequest) CRUDResponse {
	if err := h.checkPermission(client.UserID, req.Model, "delete"); err != nil {
		return CRUDResponse{ID: req.ID, Type: "crud_response", Model: req.Model, Action: "delete", Success: false, Error: err.Error()}
	}

	if req.RecordID == "" {
		return CRUDResponse{ID: req.ID, Type: "crud_response", Model: req.Model, Action: "delete", Success: false, Error: "record_id required"}
	}

	repo := h.getRepo(req.Model)
	modelDef, _ := h.modelRegistry.Get(req.Model)
	var deleteErr error
	if modelDef != nil && modelDef.IsSoftDeletes() {
		deleteErr = repo.Delete(context.Background(), req.RecordID)
	} else {
		deleteErr = repo.HardDelete(context.Background(), req.RecordID)
	}
	if deleteErr != nil {
		return CRUDResponse{ID: req.ID, Type: "crud_response", Model: req.Model, Action: "delete", Success: false, Error: deleteErr.Error()}
	}

	return CRUDResponse{ID: req.ID, Type: "crud_response", Model: req.Model, Action: "delete", Success: true, Data: map[string]any{"message": "deleted"}}
}


