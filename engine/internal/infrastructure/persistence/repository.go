package persistence

import (
	"context"
	"fmt"

	"github.com/bitcode-engine/engine/internal/compiler/parser"
	"github.com/bitcode-engine/engine/internal/runtime/expression"
	"github.com/bitcode-engine/engine/internal/runtime/pkgen"
	"gorm.io/gorm"
)

type GenericRepository struct {
	db           *gorm.DB
	tableName    string
	tenantID     string
	modelDef     *parser.ModelDefinition
	pkCol        string
	hydrator     *expression.Hydrator
	revisionRepo *DataRevisionRepository
	modelName    string
	currentUser  string
}

func NewGenericRepository(db *gorm.DB, tableName string) *GenericRepository {
	return &GenericRepository{db: db, tableName: tableName, pkCol: "id"}
}

func NewGenericRepositoryWithModel(db *gorm.DB, tableName string, model *parser.ModelDefinition) *GenericRepository {
	col := "id"
	if model != nil {
		col = pkgen.GetPKColumn(model)
	}
	return &GenericRepository{db: db, tableName: tableName, modelDef: model, pkCol: col}
}

func NewGenericRepositoryWithTenant(db *gorm.DB, tableName string, tenantID string) *GenericRepository {
	return &GenericRepository{db: db, tableName: tableName, tenantID: tenantID, pkCol: "id"}
}

func NewGenericRepositoryWithModelAndTenant(db *gorm.DB, tableName string, model *parser.ModelDefinition, tenantID string) *GenericRepository {
	col := "id"
	if model != nil {
		col = pkgen.GetPKColumn(model)
	}
	return &GenericRepository{db: db, tableName: tableName, modelDef: model, tenantID: tenantID, pkCol: col}
}

func (r *GenericRepository) SetHydrator(h *expression.Hydrator) {
	r.hydrator = h
}

func (r *GenericRepository) SetRevisionRepo(repo *DataRevisionRepository) {
	r.revisionRepo = repo
}

func (r *GenericRepository) SetModelName(name string) {
	r.modelName = name
}

func (r *GenericRepository) SetCurrentUser(userID string) {
	r.currentUser = userID
}

func (r *GenericRepository) applyTenant(query *gorm.DB) *gorm.DB {
	if r.tenantID != "" {
		return query.Where("tenant_id = ?", r.tenantID)
	}
	return query
}

func (r *GenericRepository) Create(ctx context.Context, record map[string]any) (map[string]any, error) {
	if r.tenantID != "" {
		record["tenant_id"] = r.tenantID
	}
	if err := r.db.WithContext(ctx).Table(r.tableName).Create(&record).Error; err != nil {
		return nil, fmt.Errorf("failed to create record in %s: %w", r.tableName, err)
	}
	r.saveRevision("create", r.resolveRecordID(record), nil, record)
	return record, nil
}

func (r *GenericRepository) FindByID(ctx context.Context, id string) (map[string]any, error) {
	var result map[string]any
	query := r.applyTenant(r.db.WithContext(ctx).Table(r.tableName))
	err := query.Where(fmt.Sprintf("%s = ?", r.pkCol), id).Take(&result).Error
	if err != nil {
		return nil, fmt.Errorf("record not found in %s: %w", r.tableName, err)
	}
	if r.hydrator != nil && r.modelDef != nil {
		r.hydrator.HydrateRecord(ctx, r.modelDef, result)
	}
	return result, nil
}

func (r *GenericRepository) FindByCompositePK(ctx context.Context, keys map[string]any) (map[string]any, error) {
	var result map[string]any
	query := r.applyTenant(r.db.WithContext(ctx).Table(r.tableName))
	for col, val := range keys {
		query = query.Where(fmt.Sprintf("%s = ?", col), val)
	}
	err := query.Take(&result).Error
	if err != nil {
		return nil, fmt.Errorf("record not found in %s: %w", r.tableName, err)
	}
	if r.hydrator != nil && r.modelDef != nil {
		r.hydrator.HydrateRecord(ctx, r.modelDef, result)
	}
	return result, nil
}

func (r *GenericRepository) FindAll(ctx context.Context, filters [][]any, page int, pageSize int) ([]map[string]any, int64, error) {
	query := r.applyTenant(r.db.WithContext(ctx).Table(r.tableName)).Where("active = ?", true)

	for _, filter := range filters {
		if len(filter) == 3 {
			field, ok1 := filter[0].(string)
			operator, ok2 := filter[1].(string)
			if ok1 && ok2 {
				query = query.Where(fmt.Sprintf("%s %s ?", field, operator), filter[2])
			}
		}
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count records in %s: %w", r.tableName, err)
	}

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var results []map[string]any
	if err := query.Offset(offset).Limit(pageSize).Find(&results).Error; err != nil {
		return []map[string]any{}, 0, fmt.Errorf("failed to query records in %s: %w", r.tableName, err)
	}

	if results == nil {
		results = []map[string]any{}
	}

	if r.hydrator != nil && r.modelDef != nil {
		r.hydrator.HydrateRecords(ctx, r.modelDef, results)
	}

	return results, total, nil
}

func (r *GenericRepository) Update(ctx context.Context, id string, data map[string]any) error {
	var before map[string]any
	if r.revisionRepo != nil {
		r.db.WithContext(ctx).Table(r.tableName).Where(fmt.Sprintf("%s = ?", r.pkCol), id).Take(&before)
	}

	result := r.db.WithContext(ctx).Table(r.tableName).Where(fmt.Sprintf("%s = ?", r.pkCol), id).Updates(data)
	if result.Error != nil {
		return fmt.Errorf("failed to update record in %s: %w", r.tableName, result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("record not found in %s with id %s", r.tableName, id)
	}

	if before != nil {
		var after map[string]any
		r.db.WithContext(ctx).Table(r.tableName).Where(fmt.Sprintf("%s = ?", r.pkCol), id).Take(&after)
		r.saveRevision("update", id, before, after)
	}
	return nil
}

func (r *GenericRepository) UpdateByCompositePK(ctx context.Context, keys map[string]any, data map[string]any) error {
	var before map[string]any
	if r.revisionRepo != nil {
		q := r.db.WithContext(ctx).Table(r.tableName)
		for col, val := range keys {
			q = q.Where(fmt.Sprintf("%s = ?", col), val)
		}
		q.Take(&before)
	}

	query := r.db.WithContext(ctx).Table(r.tableName)
	for col, val := range keys {
		query = query.Where(fmt.Sprintf("%s = ?", col), val)
	}
	result := query.Updates(data)
	if result.Error != nil {
		return fmt.Errorf("failed to update record in %s: %w", r.tableName, result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("record not found in %s", r.tableName)
	}

	if before != nil {
		var after map[string]any
		q := r.db.WithContext(ctx).Table(r.tableName)
		for col, val := range keys {
			q = q.Where(fmt.Sprintf("%s = ?", col), val)
		}
		q.Take(&after)
		recordID := fmt.Sprintf("%v", keys)
		r.saveRevision("update", recordID, before, after)
	}
	return nil
}

func (r *GenericRepository) Delete(ctx context.Context, id string) error {
	var before map[string]any
	if r.revisionRepo != nil {
		r.db.WithContext(ctx).Table(r.tableName).Where(fmt.Sprintf("%s = ?", r.pkCol), id).Take(&before)
	}

	result := r.db.WithContext(ctx).Table(r.tableName).Where(fmt.Sprintf("%s = ?", r.pkCol), id).Update("active", false)
	if result.Error != nil {
		return fmt.Errorf("failed to soft-delete record in %s: %w", r.tableName, result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("record not found in %s with id %s", r.tableName, id)
	}

	r.saveRevision("delete", id, before, nil)
	return nil
}

func (r *GenericRepository) HardDelete(ctx context.Context, id string) error {
	var before map[string]any
	if r.revisionRepo != nil {
		r.db.WithContext(ctx).Table(r.tableName).Where(fmt.Sprintf("%s = ?", r.pkCol), id).Take(&before)
	}

	result := r.db.WithContext(ctx).Table(r.tableName).Where(fmt.Sprintf("%s = ?", r.pkCol), id).Delete(nil)
	if result.Error != nil {
		return fmt.Errorf("failed to delete record in %s: %w", r.tableName, result.Error)
	}

	r.saveRevision("delete", id, before, nil)
	return nil
}

func (r *GenericRepository) saveRevision(action string, recordID string, before, after map[string]any) {
	if r.revisionRepo == nil || r.modelName == "" {
		return
	}

	snapshot := after
	if snapshot == nil {
		snapshot = before
	}

	var changes map[string]any
	if before != nil && after != nil {
		changes = ComputeChanges(before, after)
	}

	r.revisionRepo.Create(r.modelName, recordID, action, snapshot, changes, r.currentUser)
}

func (r *GenericRepository) resolveRecordID(record map[string]any) string {
	if v, ok := record[r.pkCol]; ok {
		return fmt.Sprintf("%v", v)
	}
	if v, ok := record["id"]; ok {
		return fmt.Sprintf("%v", v)
	}
	return ""
}
