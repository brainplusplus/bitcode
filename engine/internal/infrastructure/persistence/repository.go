package persistence

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

type GenericRepository struct {
	db        *gorm.DB
	tableName string
	tenantID  string
}

func NewGenericRepository(db *gorm.DB, tableName string) *GenericRepository {
	return &GenericRepository{db: db, tableName: tableName}
}

func NewGenericRepositoryWithTenant(db *gorm.DB, tableName string, tenantID string) *GenericRepository {
	return &GenericRepository{db: db, tableName: tableName, tenantID: tenantID}
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
	return record, nil
}

func (r *GenericRepository) FindByID(ctx context.Context, id string) (map[string]any, error) {
	var result map[string]any
	query := r.applyTenant(r.db.WithContext(ctx).Table(r.tableName))
	err := query.Where("id = ?", id).Take(&result).Error
	if err != nil {
		return nil, fmt.Errorf("record not found in %s: %w", r.tableName, err)
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

	return results, total, nil
}

func (r *GenericRepository) Update(ctx context.Context, id string, data map[string]any) error {
	result := r.db.WithContext(ctx).Table(r.tableName).Where("id = ?", id).Updates(data)
	if result.Error != nil {
		return fmt.Errorf("failed to update record in %s: %w", r.tableName, result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("record not found in %s with id %s", r.tableName, id)
	}
	return nil
}

func (r *GenericRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Table(r.tableName).Where("id = ?", id).Update("active", false)
	if result.Error != nil {
		return fmt.Errorf("failed to soft-delete record in %s: %w", r.tableName, result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("record not found in %s with id %s", r.tableName, id)
	}
	return nil
}

func (r *GenericRepository) HardDelete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Table(r.tableName).Where("id = ?", id).Delete(nil)
	if result.Error != nil {
		return fmt.Errorf("failed to delete record in %s: %w", r.tableName, result.Error)
	}
	return nil
}
