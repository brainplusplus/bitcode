package persistence

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/bitcode-framework/bitcode/internal/compiler/parser"
	"github.com/bitcode-framework/bitcode/internal/runtime/expression"
	"github.com/bitcode-framework/bitcode/internal/runtime/pkgen"
	"github.com/bitcode-framework/bitcode/pkg/security"
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
	encryptor    *security.FieldEncryptor
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

func (r *GenericRepository) SetEncryptor(enc *security.FieldEncryptor) {
	r.encryptor = enc
}

func (r *GenericRepository) encryptFields(record map[string]any) {
	if r.encryptor == nil || r.modelDef == nil {
		return
	}
	for fieldName, fieldDef := range r.modelDef.Fields {
		if !fieldDef.Encrypted {
			continue
		}
		val, ok := record[fieldName]
		if !ok {
			continue
		}
		strVal, ok := val.(string)
		if !ok || strVal == "" {
			continue
		}
		if security.IsEncrypted(strVal) {
			continue
		}
		encrypted, err := r.encryptor.Encrypt(strVal)
		if err != nil {
			log.Printf("[ENCRYPT] failed to encrypt field %s: %v", fieldName, err)
			continue
		}
		record[fieldName] = encrypted
	}
}

func (r *GenericRepository) decryptFields(record map[string]any) {
	if r.encryptor == nil || r.modelDef == nil {
		return
	}
	for fieldName, fieldDef := range r.modelDef.Fields {
		if !fieldDef.Encrypted {
			continue
		}
		val, ok := record[fieldName]
		if !ok {
			continue
		}
		strVal, ok := val.(string)
		if !ok || strVal == "" {
			continue
		}
		if !security.IsEncrypted(strVal) {
			continue
		}
		decrypted, err := r.encryptor.Decrypt(strVal)
		if err != nil {
			log.Printf("[DECRYPT] failed to decrypt field %s: %v", fieldName, err)
			continue
		}
		record[fieldName] = decrypted
	}
}

func (r *GenericRepository) applyTenant(query *gorm.DB) *gorm.DB {
	if r.tenantID != "" {
		return query.Where("tenant_id = ?", r.tenantID)
	}
	return query
}

func (r *GenericRepository) applyNotDeleted(query *gorm.DB) *gorm.DB {
	if r.modelDef != nil && r.modelDef.IsSoftDeletes() {
		return query.Where("deleted_at IS NULL")
	}
	return query
}

func (r *GenericRepository) applyActiveFilter(query *gorm.DB) *gorm.DB {
	q := query.Where("active = ?", true)
	if r.modelDef != nil && r.modelDef.IsSoftDeletes() {
		q = q.Where("deleted_at IS NULL")
	}
	return q
}

func (r *GenericRepository) Create(ctx context.Context, record map[string]any) (map[string]any, error) {
	if r.tenantID != "" {
		record["tenant_id"] = r.tenantID
	}
	r.encryptFields(record)
	if err := r.db.WithContext(ctx).Table(r.tableName).Create(&record).Error; err != nil {
		return nil, fmt.Errorf("failed to create record in %s: %w", r.tableName, err)
	}
	r.saveRevision("create", r.resolveRecordID(record), nil, record)
	r.decryptFields(record)
	return record, nil
}

func (r *GenericRepository) FindByID(ctx context.Context, id string) (map[string]any, error) {
	var result map[string]any
	query := r.applyTenant(r.db.WithContext(ctx).Table(r.tableName))
	err := query.Where(fmt.Sprintf("%s = ?", r.pkCol), id).Take(&result).Error
	if err != nil {
		return nil, fmt.Errorf("record not found in %s: %w", r.tableName, err)
	}
	r.decryptFields(result)
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
	r.decryptFields(result)
	if r.hydrator != nil && r.modelDef != nil {
		r.hydrator.HydrateRecord(ctx, r.modelDef, result)
	}
	return result, nil
}

func (r *GenericRepository) FindAll(ctx context.Context, query *Query, page int, pageSize int) ([]map[string]any, int64, error) {
	q := r.applyNotDeleted(r.applyTenant(r.db.WithContext(ctx).Table(r.tableName)))
	q = r.applyQuery(q, query)

	var total int64
	if err := q.Count(&total).Error; err != nil {
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
	if err := q.Offset(offset).Limit(pageSize).Find(&results).Error; err != nil {
		return []map[string]any{}, 0, fmt.Errorf("failed to query records in %s: %w", r.tableName, err)
	}

	if results == nil {
		results = []map[string]any{}
	}

	for _, record := range results {
		r.decryptFields(record)
	}

	if r.hydrator != nil && r.modelDef != nil {
		r.hydrator.HydrateRecords(ctx, r.modelDef, results)
	}

	return results, total, nil
}

func (r *GenericRepository) FindAllActive(ctx context.Context, query *Query, page int, pageSize int) ([]map[string]any, int64, error) {
	q := r.applyActiveFilter(r.applyTenant(r.db.WithContext(ctx).Table(r.tableName)))
	q = r.applyQuery(q, query)

	var total int64
	if err := q.Count(&total).Error; err != nil {
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
	if err := q.Offset(offset).Limit(pageSize).Find(&results).Error; err != nil {
		return []map[string]any{}, 0, fmt.Errorf("failed to query records in %s: %w", r.tableName, err)
	}

	if results == nil {
		results = []map[string]any{}
	}

	for _, record := range results {
		r.decryptFields(record)
	}

	if r.hydrator != nil && r.modelDef != nil {
		r.hydrator.HydrateRecords(ctx, r.modelDef, results)
	}

	return results, total, nil
}

func (r *GenericRepository) FindActive(ctx context.Context, id string) (map[string]any, error) {
	var result map[string]any
	query := r.applyActiveFilter(r.applyTenant(r.db.WithContext(ctx).Table(r.tableName)))
	err := query.Where(fmt.Sprintf("%s = ?", r.pkCol), id).Take(&result).Error
	if err != nil {
		return nil, fmt.Errorf("record not found in %s: %w", r.tableName, err)
	}
	r.decryptFields(result)
	if r.hydrator != nil && r.modelDef != nil {
		r.hydrator.HydrateRecord(ctx, r.modelDef, result)
	}
	return result, nil
}

func (r *GenericRepository) applyQuery(q *gorm.DB, query *Query) *gorm.DB {
	if query == nil {
		return q
	}
	for _, cond := range query.Conditions {
		switch cond.Operator {
		case "=", "!=", ">", "<", ">=", "<=":
			q = q.Where(fmt.Sprintf("%s %s ?", cond.Field, cond.Operator), cond.Value)
		case "like":
			q = q.Where(fmt.Sprintf("%s LIKE ?", cond.Field), cond.Value)
		case "in":
			q = q.Where(fmt.Sprintf("%s IN ?", cond.Field), cond.Value)
		case "not_in":
			q = q.Where(fmt.Sprintf("%s NOT IN ?", cond.Field), cond.Value)
		case "is_null":
			q = q.Where(fmt.Sprintf("%s IS NULL", cond.Field))
		case "is_not_null":
			q = q.Where(fmt.Sprintf("%s IS NOT NULL", cond.Field))
		case "between":
			if vals, ok := cond.Value.([]any); ok && len(vals) == 2 {
				q = q.Where(fmt.Sprintf("%s BETWEEN ? AND ?", cond.Field), vals[0], vals[1])
			}
		}
	}
	for _, order := range query.OrderBy {
		dir := "ASC"
		if order.Direction == "desc" {
			dir = "DESC"
		}
		q = q.Order(fmt.Sprintf("%s %s", order.Field, dir))
	}
	if len(query.Select) > 0 {
		q = q.Select(query.Select)
	}
	if len(query.GroupBy) > 0 {
		for _, g := range query.GroupBy {
			q = q.Group(g)
		}
	}
	return q
}

func (r *GenericRepository) Update(ctx context.Context, id string, data map[string]any) error {
	var before map[string]any
	if r.revisionRepo != nil {
		r.db.WithContext(ctx).Table(r.tableName).Where(fmt.Sprintf("%s = ?", r.pkCol), id).Take(&before)
	}

	r.encryptFields(data)
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

func (r *GenericRepository) UpdateWithVersion(ctx context.Context, id string, data map[string]any, expectedVersion int) error {
	var before map[string]any
	if r.revisionRepo != nil {
		r.db.WithContext(ctx).Table(r.tableName).Where(fmt.Sprintf("%s = ?", r.pkCol), id).Take(&before)
	}

	r.encryptFields(data)
	data["version"] = gorm.Expr("version + 1")
	result := r.db.WithContext(ctx).Table(r.tableName).
		Where(fmt.Sprintf("%s = ? AND version = ?", r.pkCol), id, expectedVersion).
		Updates(data)
	if result.Error != nil {
		return fmt.Errorf("failed to update record in %s: %w", r.tableName, result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("version conflict")
	}

	if before != nil {
		var after map[string]any
		r.db.WithContext(ctx).Table(r.tableName).Where(fmt.Sprintf("%s = ?", r.pkCol), id).Take(&after)
		r.saveRevision("update", id, before, after)
	}
	return nil
}

func (r *GenericRepository) SoftDeleteWithTimestamp(ctx context.Context, id string, deletedAt time.Time) error {
	var before map[string]any
	if r.revisionRepo != nil {
		r.db.WithContext(ctx).Table(r.tableName).Where(fmt.Sprintf("%s = ?", r.pkCol), id).Take(&before)
	}

	result := r.db.WithContext(ctx).Table(r.tableName).
		Where(fmt.Sprintf("%s = ?", r.pkCol), id).
		Updates(map[string]any{"active": false, "deleted_at": deletedAt})
	if result.Error != nil {
		return fmt.Errorf("failed to soft-delete record in %s: %w", r.tableName, result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("record not found in %s with id %s", r.tableName, id)
	}

	r.saveRevision("delete", id, before, nil)
	return nil
}

func (r *GenericRepository) SetModelDef(def *parser.ModelDefinition) {
	r.modelDef = def
}

func (r *GenericRepository) GetTableName() string {
	return r.tableName
}

func (r *GenericRepository) Upsert(ctx context.Context, data map[string]any, uniqueFields []string) (map[string]any, error) {
	if len(uniqueFields) == 0 {
		return r.Create(ctx, data)
	}
	q := r.db.WithContext(ctx).Table(r.tableName)
	for _, f := range uniqueFields {
		if v, ok := data[f]; ok {
			q = q.Where(fmt.Sprintf("%s = ?", f), v)
		}
	}
	var existing map[string]any
	if err := q.Take(&existing).Error; err == nil {
		id := fmt.Sprintf("%v", existing[r.pkCol])
		if err := r.Update(ctx, id, data); err != nil {
			return nil, err
		}
		var updated map[string]any
		r.db.WithContext(ctx).Table(r.tableName).Where(fmt.Sprintf("%s = ?", r.pkCol), id).Take(&updated)
		return updated, nil
	}
	return r.Create(ctx, data)
}

func (r *GenericRepository) Count(ctx context.Context, query *Query) (int64, error) {
	q := r.applyNotDeleted(r.applyTenant(r.db.WithContext(ctx).Table(r.tableName)))
	q = r.applyQuery(q, query)
	var count int64
	if err := q.Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count in %s: %w", r.tableName, err)
	}
	return count, nil
}

func (r *GenericRepository) CountActive(ctx context.Context, query *Query) (int64, error) {
	q := r.applyActiveFilter(r.applyTenant(r.db.WithContext(ctx).Table(r.tableName)))
	q = r.applyQuery(q, query)
	var count int64
	if err := q.Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count in %s: %w", r.tableName, err)
	}
	return count, nil
}

func (r *GenericRepository) Sum(ctx context.Context, field string, query *Query) (float64, error) {
	q := r.applyNotDeleted(r.applyTenant(r.db.WithContext(ctx).Table(r.tableName)))
	q = r.applyQuery(q, query)
	var result float64
	if err := q.Select(fmt.Sprintf("COALESCE(SUM(%s), 0)", field)).Scan(&result).Error; err != nil {
		return 0, fmt.Errorf("failed to sum %s in %s: %w", field, r.tableName, err)
	}
	return result, nil
}

func (r *GenericRepository) SumActive(ctx context.Context, field string, query *Query) (float64, error) {
	q := r.applyActiveFilter(r.applyTenant(r.db.WithContext(ctx).Table(r.tableName)))
	q = r.applyQuery(q, query)
	var result float64
	if err := q.Select(fmt.Sprintf("COALESCE(SUM(%s), 0)", field)).Scan(&result).Error; err != nil {
		return 0, fmt.Errorf("failed to sum %s in %s: %w", field, r.tableName, err)
	}
	return result, nil
}

func (r *GenericRepository) BulkCreate(ctx context.Context, records []map[string]any) ([]map[string]any, error) {
	for i := range records {
		if r.tenantID != "" {
			records[i]["tenant_id"] = r.tenantID
		}
		r.encryptFields(records[i])
	}
	if err := r.db.WithContext(ctx).Table(r.tableName).Create(&records).Error; err != nil {
		return nil, fmt.Errorf("failed to bulk create in %s: %w", r.tableName, err)
	}
	return records, nil
}

func (r *GenericRepository) AddMany2Many(ctx context.Context, id string, field string, relatedIDs []string) error {
	junctionTable := r.tableName + "_" + field
	for _, relID := range relatedIDs {
		record := map[string]any{
			r.modelName + "_id": id,
			field + "_id":       relID,
		}
		var count int64
		r.db.WithContext(ctx).Table(junctionTable).
			Where(fmt.Sprintf("%s_id = ? AND %s_id = ?", r.modelName, field), id, relID).
			Count(&count)
		if count == 0 {
			r.db.WithContext(ctx).Table(junctionTable).Create(&record)
		}
	}
	return nil
}

func (r *GenericRepository) RemoveMany2Many(ctx context.Context, id string, field string, relatedIDs []string) error {
	junctionTable := r.tableName + "_" + field
	for _, relID := range relatedIDs {
		r.db.WithContext(ctx).Table(junctionTable).
			Where(fmt.Sprintf("%s_id = ? AND %s_id = ?", r.modelName, field), id, relID).
			Delete(nil)
	}
	return nil
}

func (r *GenericRepository) LoadMany2Many(ctx context.Context, id string, field string) ([]map[string]any, error) {
	junctionTable := r.tableName + "_" + field
	var junctionRecords []map[string]any
	if err := r.db.WithContext(ctx).Table(junctionTable).
		Where(fmt.Sprintf("%s_id = ?", r.modelName), id).
		Find(&junctionRecords).Error; err != nil {
		return nil, err
	}
	var relatedIDs []string
	for _, jr := range junctionRecords {
		if rid, ok := jr[field+"_id"]; ok {
			relatedIDs = append(relatedIDs, fmt.Sprintf("%v", rid))
		}
	}
	if len(relatedIDs) == 0 {
		return []map[string]any{}, nil
	}
	var results []map[string]any
	if err := r.db.WithContext(ctx).Table(field+"s").
		Where("id IN ?", relatedIDs).
		Find(&results).Error; err != nil {
		return nil, err
	}
	if results == nil {
		results = []map[string]any{}
	}
	return results, nil
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
