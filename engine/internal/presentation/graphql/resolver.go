package graphql

import (
	"context"
	"fmt"

	gql "github.com/graphql-go/graphql"
	"github.com/google/uuid"
	"github.com/bitcode-framework/bitcode/internal/compiler/parser"
	"github.com/bitcode-framework/bitcode/internal/infrastructure/persistence"
	"gorm.io/gorm"
)

type Resolver struct {
	db                *gorm.DB
	modelRegistry     ModelRegistry
	tableNameResolver TableNameResolver
	permissionService *persistence.PermissionService
	recordRuleService *persistence.RecordRuleService
}

type ModelRegistry interface {
	Get(name string) (*parser.ModelDefinition, error)
	TableName(name string) string
}

type TableNameResolver interface {
	TableName(name string) string
}

func NewResolver(db *gorm.DB, modelReg ModelRegistry, permSvc *persistence.PermissionService, rrSvc *persistence.RecordRuleService) *Resolver {
	return &Resolver{
		db:                db,
		modelRegistry:     modelReg,
		permissionService: permSvc,
		recordRuleService: rrSvc,
	}
}

func (r *Resolver) getRepo(modelName string) *persistence.GenericRepository {
	tableName := r.modelRegistry.TableName(modelName)
	modelDef, _ := r.modelRegistry.Get(modelName)
	if modelDef != nil {
		return persistence.NewGenericRepositoryWithModel(r.db, tableName, modelDef)
	}
	return persistence.NewGenericRepository(r.db, tableName)
}

func (r *Resolver) checkPermission(ctx context.Context, modelName, operation string) error {
	if r.permissionService == nil {
		return nil
	}
	userID, ok := ctx.Value(contextKeyUserID).(string)
	if !ok || userID == "" {
		return nil
	}
	allowed, err := r.permissionService.UserHasPermission(userID, modelName+"."+operation)
	if err != nil {
		return fmt.Errorf("permission check failed: %w", err)
	}
	if !allowed {
		return fmt.Errorf("permission denied: %s.%s", modelName, operation)
	}
	return nil
}

func (r *Resolver) getRecordRuleFilters(ctx context.Context, modelName, operation string) [][]any {
	if r.recordRuleService == nil {
		return nil
	}
	userID, ok := ctx.Value(contextKeyUserID).(string)
	if !ok || userID == "" {
		return nil
	}
	filters, _ := r.recordRuleService.GetFilters(userID, modelName, operation)
	if len(filters) > 0 {
		vars := map[string]string{"user.id": userID}
		filters = persistence.InterpolateDomainFilters(filters, vars)
	}
	return filters
}

func (r *Resolver) List(modelName string) gql.FieldResolveFn {
	return func(p gql.ResolveParams) (any, error) {
		if err := r.checkPermission(p.Context, modelName, "read"); err != nil {
			return nil, err
		}

		page := 1
		pageSize := 20
		if v, ok := p.Args["page"].(int); ok && v > 0 {
			page = v
		}
		if v, ok := p.Args["page_size"].(int); ok && v > 0 {
			pageSize = v
		}

		var filters [][]any
		if rlsFilters := r.getRecordRuleFilters(p.Context, modelName, "read"); len(rlsFilters) > 0 {
			filters = append(filters, rlsFilters...)
		}
		if q, ok := p.Args["q"].(string); ok && q != "" {
			modelDef, _ := r.modelRegistry.Get(modelName)
			if modelDef != nil && len(modelDef.SearchField) > 0 {
				for _, sf := range modelDef.SearchField {
					filters = append(filters, []any{sf, "like", "%" + q + "%"})
				}
			}
		}

		var query *persistence.Query
		if len(filters) > 0 {
			query = persistence.QueryFromDomain(filters)
		}

		repo := r.getRepo(modelName)
		results, total, err := repo.FindAll(p.Context, query, page, pageSize)
		if err != nil {
			return nil, err
		}

		totalPages := (total + int64(pageSize) - 1) / int64(pageSize)

		return map[string]any{
			"data":        results,
			"total":       total,
			"page":        page,
			"page_size":   pageSize,
			"total_pages": totalPages,
		}, nil
	}
}

func (r *Resolver) Read(modelName string) gql.FieldResolveFn {
	return func(p gql.ResolveParams) (any, error) {
		if err := r.checkPermission(p.Context, modelName, "read"); err != nil {
			return nil, err
		}

		id, ok := p.Args["id"].(string)
		if !ok || id == "" {
			return nil, fmt.Errorf("id is required")
		}

		repo := r.getRepo(modelName)
		result, err := repo.FindByID(p.Context, id)
		if err != nil {
			return nil, fmt.Errorf("record not found")
		}
		return result, nil
	}
}

func (r *Resolver) Create(modelName string) gql.FieldResolveFn {
	return func(p gql.ResolveParams) (any, error) {
		if err := r.checkPermission(p.Context, modelName, "create"); err != nil {
			return nil, err
		}

		input, ok := p.Args["input"].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("input is required")
		}

		if _, hasID := input["id"]; !hasID {
			input["id"] = uuid.New().String()
		}

		if userID, ok := p.Context.Value(contextKeyUserID).(string); ok {
			input["created_by"] = userID
			input["updated_by"] = userID
		}

		repo := r.getRepo(modelName)
		result, err := repo.Create(p.Context, input)
		if err != nil {
			return nil, err
		}
		return result, nil
	}
}

func (r *Resolver) Update(modelName string) gql.FieldResolveFn {
	return func(p gql.ResolveParams) (any, error) {
		if err := r.checkPermission(p.Context, modelName, "write"); err != nil {
			return nil, err
		}

		id, ok := p.Args["id"].(string)
		if !ok || id == "" {
			return nil, fmt.Errorf("id is required")
		}

		input, ok := p.Args["input"].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("input is required")
		}

		delete(input, "id")
		delete(input, "created_at")
		delete(input, "created_by")

		if userID, ok := p.Context.Value(contextKeyUserID).(string); ok {
			input["updated_by"] = userID
		}

		repo := r.getRepo(modelName)
		if err := repo.Update(p.Context, id, input); err != nil {
			return nil, err
		}

		result, _ := repo.FindByID(p.Context, id)
		return result, nil
	}
}

func (r *Resolver) Delete(modelName string) gql.FieldResolveFn {
	return func(p gql.ResolveParams) (any, error) {
		if err := r.checkPermission(p.Context, modelName, "delete"); err != nil {
			return nil, err
		}

		id, ok := p.Args["id"].(string)
		if !ok || id == "" {
			return nil, fmt.Errorf("id is required")
		}

		repo := r.getRepo(modelName)
		if err := repo.Delete(p.Context, id); err != nil {
			return nil, err
		}

		return map[string]any{"message": "deleted"}, nil
	}
}

type contextKey string

const contextKeyUserID contextKey = "user_id"
