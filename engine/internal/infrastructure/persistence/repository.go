package persistence

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bitcode-framework/bitcode/internal/compiler/parser"
	"github.com/bitcode-framework/bitcode/internal/runtime/expression"
	"github.com/bitcode-framework/bitcode/internal/runtime/pkgen"
	"github.com/bitcode-framework/bitcode/pkg/security"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
	if query != nil {
		query.ApplyScopes()
	}

	baseQ := r.applyTenant(r.db.WithContext(ctx).Table(r.tableName))
	if query != nil && query.SoftDeleteScope != "" {
		baseQ = r.applySoftDeleteScope(baseQ, query)
	} else {
		baseQ = r.applyNotDeleted(baseQ)
	}

	countQ := baseQ.Session(&gorm.Session{})
	if query != nil {
		countQ = r.applyJoins(countQ, query)
		countQ = r.applyWhereClauses(countQ, query)
		countQ = r.applyWhereRaw(countQ, query)
		countQ = r.applyWhereSubQueries(countQ, query)
	}
	var total int64
	if err := countQ.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count records in %s: %w", r.tableName, err)
	}

	q := baseQ.Session(&gorm.Session{})
	q = r.applyQuery(q, query)

	if query != nil && query.Limit > 0 {
		q = q.Limit(query.Limit)
		if query.Offset > 0 {
			q = q.Offset(query.Offset)
		}
	} else {
		if page < 1 {
			page = 1
		}
		if pageSize < 1 {
			pageSize = 20
		}
		offset := (page - 1) * pageSize
		q = q.Offset(offset).Limit(pageSize)
	}

	var results []map[string]any
	if err := q.Find(&results).Error; err != nil {
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

	r.loadWithRelations(ctx, query, results)

	return results, total, nil
}

func (r *GenericRepository) FindAllActive(ctx context.Context, query *Query, page int, pageSize int) ([]map[string]any, int64, error) {
	if query != nil {
		query.ApplyScopes()
	}

	baseQ := r.applyActiveFilter(r.applyTenant(r.db.WithContext(ctx).Table(r.tableName)))

	countQ := baseQ.Session(&gorm.Session{})
	if query != nil {
		countQ = r.applyJoins(countQ, query)
		countQ = r.applyWhereClauses(countQ, query)
		countQ = r.applyWhereRaw(countQ, query)
		countQ = r.applyWhereSubQueries(countQ, query)
	}
	var total int64
	if err := countQ.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count records in %s: %w", r.tableName, err)
	}

	q := baseQ.Session(&gorm.Session{})
	q = r.applyQuery(q, query)

	if query != nil && query.Limit > 0 {
		q = q.Limit(query.Limit)
		if query.Offset > 0 {
			q = q.Offset(query.Offset)
		}
	} else {
		if page < 1 {
			page = 1
		}
		if pageSize < 1 {
			pageSize = 20
		}
		offset := (page - 1) * pageSize
		q = q.Offset(offset).Limit(pageSize)
	}

	var results []map[string]any
	if err := q.Find(&results).Error; err != nil {
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

	r.loadWithRelations(ctx, query, results)

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

	query.ApplyScopes()

	q = r.applyJoins(q, query)
	q = r.applyDistinct(q, query)
	q = r.applySelectAndAggregates(q, query)
	q = r.applyWhereClauses(q, query)
	q = r.applyWhereRaw(q, query)
	q = r.applyWhereSubQueries(q, query)
	q = r.applyGroupBy(q, query)
	q = r.applyHaving(q, query)
	q = r.applyOrderBy(q, query)
	q = r.applyLocking(q, query)
	q = r.applyUnions(q, query)

	return q
}

func (r *GenericRepository) applyUnions(q *gorm.DB, query *Query) *gorm.DB {
	if len(query.Unions) == 0 {
		return q
	}
	for _, u := range query.Unions {
		if u.Query == nil {
			continue
		}
		subDB := r.db.Session(&gorm.Session{NewDB: true}).Table(r.tableName)
		subDB = r.applyQuery(subDB, u.Query)
		keyword := "UNION"
		if u.All {
			keyword = "UNION ALL"
		}
		q = r.db.Raw("? "+keyword+" ?", q, subDB)
	}
	return q
}

func (r *GenericRepository) applyJoins(q *gorm.DB, query *Query) *gorm.DB {
	for _, j := range query.Joins {
		tableRef := j.Table
		if j.Alias != "" {
			tableRef = j.Table + " " + j.Alias
		}
		if j.RawOn != "" {
			q = q.Joins(fmt.Sprintf("%s JOIN %s ON %s", j.Type, tableRef, j.RawOn))
			continue
		}
		if j.Type == JoinCross {
			q = q.Joins(fmt.Sprintf("CROSS JOIN %s", tableRef))
			continue
		}
		if !IsSafeFieldName(j.LocalKey) || !IsSafeFieldName(j.ForeignKey) {
			continue
		}
		q = q.Joins(fmt.Sprintf("%s JOIN %s ON %s = %s", j.Type, tableRef, j.LocalKey, j.ForeignKey))
	}
	return q
}

func (r *GenericRepository) applyDistinct(q *gorm.DB, query *Query) *gorm.DB {
	if query.Distinct {
		q = q.Distinct()
	}
	return q
}

func (r *GenericRepository) applySelectAndAggregates(q *gorm.DB, query *Query) *gorm.DB {
	var selectParts []string

	if len(query.Select) > 0 {
		for _, s := range query.Select {
			if IsSafeFieldName(s) {
				selectParts = append(selectParts, s)
			}
		}
	}

	for _, agg := range query.Aggregates {
		fn := strings.ToUpper(agg.Function)
		field := agg.Field
		alias := agg.Alias
		if alias == "" {
			alias = strings.ToLower(fn) + "_" + field
		}
		if agg.Distinct {
			selectParts = append(selectParts, fmt.Sprintf("%s(DISTINCT %s) AS %s", fn, field, alias))
		} else {
			selectParts = append(selectParts, fmt.Sprintf("%s(%s) AS %s", fn, field, alias))
		}
	}

	for _, raw := range query.SelectRaw {
		selectParts = append(selectParts, raw.SQL)
	}

	if len(selectParts) > 0 {
		q = q.Select(strings.Join(selectParts, ", "))
	}

	return q
}

func (r *GenericRepository) applyWhereClauses(q *gorm.DB, query *Query) *gorm.DB {
	clauses := query.GetEffectiveWhereClauses()
	for _, clause := range clauses {
		q = r.applyWhereClause(q, clause, false)
	}
	return q
}

func (r *GenericRepository) applyWhereClause(q *gorm.DB, clause WhereClause, isOr bool) *gorm.DB {
	if clause.Condition != nil {
		return r.applySingleCondition(q, *clause.Condition, isOr)
	}
	if clause.Group != nil {
		return r.applyConditionGroup(q, *clause.Group, isOr)
	}
	return q
}

func (r *GenericRepository) applySingleCondition(q *gorm.DB, cond Condition, isOr bool) *gorm.DB {
	if !IsSafeFieldName(cond.Field) && !strings.HasPrefix(cond.Operator, "column:") {
		return q
	}

	whereFunc := q.Where
	if isOr {
		whereFunc = q.Or
	}

	switch cond.Operator {
	case "=", "!=", ">", "<", ">=", "<=":
		return whereFunc(fmt.Sprintf("%s %s ?", cond.Field, cond.Operator), cond.Value)
	case "like":
		return whereFunc(fmt.Sprintf("%s LIKE ?", cond.Field), cond.Value)
	case "not_like":
		return whereFunc(fmt.Sprintf("%s NOT LIKE ?", cond.Field), cond.Value)
	case "in":
		return whereFunc(fmt.Sprintf("%s IN ?", cond.Field), cond.Value)
	case "not_in":
		return whereFunc(fmt.Sprintf("%s NOT IN ?", cond.Field), cond.Value)
	case "is_null":
		return whereFunc(fmt.Sprintf("%s IS NULL", cond.Field))
	case "is_not_null":
		return whereFunc(fmt.Sprintf("%s IS NOT NULL", cond.Field))
	case "between":
		if vals, ok := cond.Value.([]any); ok && len(vals) == 2 {
			return whereFunc(fmt.Sprintf("%s BETWEEN ? AND ?", cond.Field), vals[0], vals[1])
		}
	case "not_between":
		if vals, ok := cond.Value.([]any); ok && len(vals) == 2 {
			return whereFunc(fmt.Sprintf("%s NOT BETWEEN ? AND ?", cond.Field), vals[0], vals[1])
		}
	default:
		if strings.HasPrefix(cond.Operator, "column:") {
			op := strings.TrimPrefix(cond.Operator, "column:")
			field2, ok := cond.Value.(string)
			if ok && IsSafeFieldName(cond.Field) && IsSafeFieldName(field2) {
				return whereFunc(fmt.Sprintf("%s %s %s", cond.Field, op, field2))
			}
		}
	}
	return q
}

func (r *GenericRepository) applyConditionGroup(q *gorm.DB, group ConditionGroup, isOr bool) *gorm.DB {
	subDB := r.db.Session(&gorm.Session{NewDB: true})

	for i, clause := range group.Conditions {
		useOr := group.Connector == ConnectorOr && i > 0
		subDB = r.applyWhereClause(subDB, clause, useOr)
	}

	stmt := subDB.Statement
	if stmt == nil {
		return q
	}

	if group.Negate {
		if isOr {
			q = q.Or("NOT (?)", subDB)
		} else {
			q = q.Where("NOT (?)", subDB)
		}
	} else {
		if isOr {
			q = q.Or(subDB)
		} else {
			q = q.Where(subDB)
		}
	}
	return q
}

func (r *GenericRepository) applyWhereRaw(q *gorm.DB, query *Query) *gorm.DB {
	for _, raw := range query.WhereRaw {
		if len(raw.Values) > 0 {
			q = q.Where(raw.SQL, raw.Values...)
		} else {
			q = q.Where(raw.SQL)
		}
	}
	return q
}

func (r *GenericRepository) applyWhereSubQueries(q *gorm.DB, query *Query) *gorm.DB {
	for _, ws := range query.WhereSubQueries {
		if ws.Sub.Query == nil {
			continue
		}
		subDB := r.buildSubQuery(ws.Sub.Query)
		switch ws.Operator {
		case "in":
			q = q.Where(fmt.Sprintf("%s IN (?)", ws.Field), subDB)
		case "not_in":
			q = q.Where(fmt.Sprintf("%s NOT IN (?)", ws.Field), subDB)
		}
	}

	for _, ws := range query.WhereExists {
		if ws.Query == nil {
			continue
		}
		subDB := r.buildSubQuery(ws.Query)
		q = q.Where("EXISTS (?)", subDB)
	}

	for _, ws := range query.WhereNotExists {
		if ws.Query == nil {
			continue
		}
		subDB := r.buildSubQuery(ws.Query)
		q = q.Where("NOT EXISTS (?)", subDB)
	}

	return q
}

func (r *GenericRepository) buildSubQuery(sub *Query) *gorm.DB {
	subDB := r.db.Session(&gorm.Session{NewDB: true}).Table(r.tableName)
	subDB = r.applyQuery(subDB, sub)
	return subDB
}

func (r *GenericRepository) applyGroupBy(q *gorm.DB, query *Query) *gorm.DB {
	for _, g := range query.GroupBy {
		if IsSafeFieldName(g) {
			q = q.Group(g)
		}
	}
	for _, raw := range query.GroupRaw {
		q = q.Group(raw.SQL)
	}
	return q
}

func (r *GenericRepository) applyHaving(q *gorm.DB, query *Query) *gorm.DB {
	for _, h := range query.Having {
		if h.Raw != "" {
			q = q.Having(h.Raw)
			continue
		}
		if h.Aggregate != "" && h.Field != "" && h.Operator != "" {
			expr := fmt.Sprintf("%s(%s) %s ?", strings.ToUpper(h.Aggregate), h.Field, h.Operator)
			q = q.Having(expr, h.Value)
		} else if h.Field != "" && h.Operator != "" {
			q = q.Having(fmt.Sprintf("%s %s ?", h.Field, h.Operator), h.Value)
		}
	}
	for _, raw := range query.HavingRaw {
		if len(raw.Values) > 0 {
			q = q.Having(raw.SQL, raw.Values...)
		} else {
			q = q.Having(raw.SQL)
		}
	}
	return q
}

func (r *GenericRepository) applyOrderBy(q *gorm.DB, query *Query) *gorm.DB {
	for _, order := range query.OrderBy {
		if !IsSafeFieldName(order.Field) {
			continue
		}
		dir := "ASC"
		if strings.ToLower(order.Direction) == "desc" {
			dir = "DESC"
		}
		q = q.Order(fmt.Sprintf("%s %s", order.Field, dir))
	}
	for _, raw := range query.OrderRaw {
		q = q.Order(gorm.Expr(raw.SQL))
	}
	return q
}

func (r *GenericRepository) applyLocking(q *gorm.DB, query *Query) *gorm.DB {
	switch query.Lock {
	case LockForUpdate:
		q = q.Clauses(clause.Locking{Strength: "UPDATE"})
	case LockForShare:
		q = q.Clauses(clause.Locking{Strength: "SHARE"})
	}
	return q
}

func (r *GenericRepository) applySoftDeleteScope(q *gorm.DB, query *Query) *gorm.DB {
	if query == nil {
		return q
	}
	switch query.SoftDeleteScope {
	case ScopeWithTrashed:
		return q
	case ScopeOnlyTrashed:
		if r.modelDef != nil && r.modelDef.IsSoftDeletes() {
			return q.Where("deleted_at IS NOT NULL")
		}
		return q.Where("active = ?", false)
	default:
		return q
	}
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

func (r *GenericRepository) SoftDeleteWithTimestamp(ctx context.Context, id string, deletedAt time.Time, deletedBy string) error {
	var before map[string]any
	if r.revisionRepo != nil {
		r.db.WithContext(ctx).Table(r.tableName).Where(fmt.Sprintf("%s = ?", r.pkCol), id).Take(&before)
	}

	updates := map[string]any{"active": false, "deleted_at": deletedAt}
	if r.modelDef != nil && r.modelDef.IsSoftDeletesBy() && deletedBy != "" {
		updates["deleted_by"] = deletedBy
	}

	result := r.db.WithContext(ctx).Table(r.tableName).
		Where(fmt.Sprintf("%s = ?", r.pkCol), id).
		Updates(updates)
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
	if !IsSafeFieldName(field) {
		return 0, fmt.Errorf("invalid field name: %s", field)
	}
	q := r.applyNotDeleted(r.applyTenant(r.db.WithContext(ctx).Table(r.tableName)))
	q = r.applyQuery(q, query)
	var result float64
	if err := q.Select(fmt.Sprintf("COALESCE(SUM(%s), 0)", field)).Scan(&result).Error; err != nil {
		return 0, fmt.Errorf("failed to sum %s in %s: %w", field, r.tableName, err)
	}
	return result, nil
}

func (r *GenericRepository) SumActive(ctx context.Context, field string, query *Query) (float64, error) {
	if !IsSafeFieldName(field) {
		return 0, fmt.Errorf("invalid field name: %s", field)
	}
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

func (r *GenericRepository) Avg(ctx context.Context, field string, query *Query) (float64, error) {
	if !IsSafeFieldName(field) {
		return 0, fmt.Errorf("invalid field name: %s", field)
	}
	q := r.applyNotDeleted(r.applyTenant(r.db.WithContext(ctx).Table(r.tableName)))
	q = r.applyQuery(q, query)
	var result float64
	if err := q.Select(fmt.Sprintf("COALESCE(AVG(%s), 0)", field)).Scan(&result).Error; err != nil {
		return 0, fmt.Errorf("failed to avg %s in %s: %w", field, r.tableName, err)
	}
	return result, nil
}

func (r *GenericRepository) Min(ctx context.Context, field string, query *Query) (float64, error) {
	if !IsSafeFieldName(field) {
		return 0, fmt.Errorf("invalid field name: %s", field)
	}
	q := r.applyNotDeleted(r.applyTenant(r.db.WithContext(ctx).Table(r.tableName)))
	q = r.applyQuery(q, query)
	var result float64
	if err := q.Select(fmt.Sprintf("COALESCE(MIN(%s), 0)", field)).Scan(&result).Error; err != nil {
		return 0, fmt.Errorf("failed to min %s in %s: %w", field, r.tableName, err)
	}
	return result, nil
}

func (r *GenericRepository) Max(ctx context.Context, field string, query *Query) (float64, error) {
	if !IsSafeFieldName(field) {
		return 0, fmt.Errorf("invalid field name: %s", field)
	}
	q := r.applyNotDeleted(r.applyTenant(r.db.WithContext(ctx).Table(r.tableName)))
	q = r.applyQuery(q, query)
	var result float64
	if err := q.Select(fmt.Sprintf("COALESCE(MAX(%s), 0)", field)).Scan(&result).Error; err != nil {
		return 0, fmt.Errorf("failed to max %s in %s: %w", field, r.tableName, err)
	}
	return result, nil
}

func (r *GenericRepository) Pluck(ctx context.Context, field string, query *Query) ([]any, error) {
	q := r.applyNotDeleted(r.applyTenant(r.db.WithContext(ctx).Table(r.tableName)))
	q = r.applyQuery(q, query)
	var results []any
	if err := q.Pluck(field, &results).Error; err != nil {
		return nil, fmt.Errorf("failed to pluck %s in %s: %w", field, r.tableName, err)
	}
	return results, nil
}

func (r *GenericRepository) Exists(ctx context.Context, query *Query) (bool, error) {
	count, err := r.Count(ctx, query)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *GenericRepository) Aggregate(ctx context.Context, query *Query) ([]map[string]any, error) {
	q := r.applyNotDeleted(r.applyTenant(r.db.WithContext(ctx).Table(r.tableName)))

	if query != nil {
		query.ApplyScopes()
		q = r.applyJoins(q, query)
		q = r.applyWhereClauses(q, query)
		q = r.applyWhereRaw(q, query)

		var selectParts []string
		for _, g := range query.GroupBy {
			if IsSafeFieldName(g) {
				selectParts = append(selectParts, g)
			}
		}
		for _, agg := range query.Aggregates {
			fn := strings.ToUpper(agg.Function)
			if !IsSafeFieldName(agg.Field) && agg.Field != "*" {
				continue
			}
			alias := agg.Alias
			if alias == "" {
				alias = strings.ToLower(fn) + "_" + strings.ReplaceAll(agg.Field, "*", "all")
			}
			if agg.Distinct {
				selectParts = append(selectParts, fmt.Sprintf("%s(DISTINCT %s) AS %s", fn, agg.Field, alias))
			} else {
				selectParts = append(selectParts, fmt.Sprintf("%s(%s) AS %s", fn, agg.Field, alias))
			}
		}
		if len(selectParts) > 0 {
			q = q.Select(strings.Join(selectParts, ", "))
		}

		q = r.applyGroupBy(q, query)
		q = r.applyHaving(q, query)
		q = r.applyOrderBy(q, query)
	}

	var results []map[string]any
	if err := q.Scan(&results).Error; err != nil {
		return nil, fmt.Errorf("failed to aggregate in %s: %w", r.tableName, err)
	}
	if results == nil {
		results = []map[string]any{}
	}
	return results, nil
}

func (r *GenericRepository) Chunk(ctx context.Context, query *Query, size int, fn func(records []map[string]any) error) error {
	page := 1
	for {
		records, _, err := r.FindAll(ctx, query, page, size)
		if err != nil {
			return err
		}
		if len(records) == 0 {
			break
		}
		if err := fn(records); err != nil {
			return err
		}
		if len(records) < size {
			break
		}
		page++
	}
	return nil
}

func (r *GenericRepository) Increment(ctx context.Context, id string, field string, value int) error {
	if !IsSafeFieldName(field) {
		return fmt.Errorf("invalid field name: %s", field)
	}
	result := r.db.WithContext(ctx).Table(r.tableName).
		Where(fmt.Sprintf("%s = ?", r.pkCol), id).
		Update(field, gorm.Expr(fmt.Sprintf("%s + ?", field), value))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("record not found in %s with id %s", r.tableName, id)
	}
	return nil
}

func (r *GenericRepository) Decrement(ctx context.Context, id string, field string, value int) error {
	return r.Increment(ctx, id, field, -value)
}

func (r *GenericRepository) Transaction(ctx context.Context, fn func(txRepo Repository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txRepo := &GenericRepository{
			db:           tx,
			tableName:    r.tableName,
			tenantID:     r.tenantID,
			modelDef:     r.modelDef,
			pkCol:        r.pkCol,
			hydrator:     r.hydrator,
			revisionRepo: r.revisionRepo,
			modelName:    r.modelName,
			currentUser:  r.currentUser,
			encryptor:    r.encryptor,
		}
		return fn(txRepo)
	})
}

func (r *GenericRepository) RawQuery(ctx context.Context, sql string, values ...any) ([]map[string]any, error) {
	var results []map[string]any
	if err := r.db.WithContext(ctx).Raw(sql, values...).Scan(&results).Error; err != nil {
		return nil, fmt.Errorf("raw query failed: %w", err)
	}
	if results == nil {
		results = []map[string]any{}
	}
	return results, nil
}

func (r *GenericRepository) RawExec(ctx context.Context, sql string, values ...any) (int64, error) {
	result := r.db.WithContext(ctx).Exec(sql, values...)
	if result.Error != nil {
		return 0, fmt.Errorf("raw exec failed: %w", result.Error)
	}
	return result.RowsAffected, nil
}

func (r *GenericRepository) FindAllWithTrashed(ctx context.Context, query *Query, page, pageSize int) ([]map[string]any, int64, error) {
	if query == nil {
		query = NewQuery()
	}
	query.SoftDeleteScope = ScopeWithTrashed
	return r.FindAll(ctx, query, page, pageSize)
}

func (r *GenericRepository) FindAllOnlyTrashed(ctx context.Context, query *Query, page, pageSize int) ([]map[string]any, int64, error) {
	if query == nil {
		query = NewQuery()
	}
	query.SoftDeleteScope = ScopeOnlyTrashed
	return r.FindAll(ctx, query, page, pageSize)
}

func (r *GenericRepository) loadWithRelations(ctx context.Context, query *Query, results []map[string]any) {
	if query == nil || len(query.With) == 0 || len(results) == 0 {
		return
	}
	if r.modelDef == nil {
		return
	}

	for _, w := range query.With {
		fieldDef, ok := r.modelDef.Fields[w.Relation]
		if !ok {
			continue
		}

		switch fieldDef.Type {
		case parser.FieldMany2One:
			r.loadMany2OneRelation(ctx, w, fieldDef, results)
		case parser.FieldOne2Many:
			r.loadOne2ManyRelation(ctx, w, fieldDef, results)
		case parser.FieldMany2Many:
			r.loadMany2ManyRelation(ctx, w, results)
		}
	}
}

func (r *GenericRepository) loadMany2OneRelation(ctx context.Context, w WithClause, fieldDef parser.FieldDefinition, results []map[string]any) {
	if fieldDef.Model == "" {
		return
	}

	fkIDs := make(map[string]bool)
	for _, rec := range results {
		if fk, ok := rec[w.Relation]; ok && fk != nil {
			fkIDs[fmt.Sprintf("%v", fk)] = true
		}
	}
	if len(fkIDs) == 0 {
		return
	}

	ids := make([]string, 0, len(fkIDs))
	for id := range fkIDs {
		ids = append(ids, id)
	}

	relatedTable := fieldDef.Model
	relQ := r.db.WithContext(ctx).Table(relatedTable).Where("id IN ?", ids)
	if len(w.Select) > 0 {
		relQ = relQ.Select(w.Select)
	}

	var related []map[string]any
	if err := relQ.Find(&related).Error; err != nil {
		return
	}

	relatedMap := make(map[string]map[string]any)
	for _, rel := range related {
		if id, ok := rel["id"]; ok {
			relatedMap[fmt.Sprintf("%v", id)] = rel
		}
	}

	for _, rec := range results {
		if fk, ok := rec[w.Relation]; ok && fk != nil {
			if rel, ok := relatedMap[fmt.Sprintf("%v", fk)]; ok {
				rec["_"+w.Relation] = rel
			}
		}
	}
}

func (r *GenericRepository) loadOne2ManyRelation(ctx context.Context, w WithClause, fieldDef parser.FieldDefinition, results []map[string]any) {
	if fieldDef.Model == "" {
		return
	}

	parentIDs := make([]string, 0, len(results))
	for _, rec := range results {
		if id, ok := rec[r.pkCol]; ok {
			parentIDs = append(parentIDs, fmt.Sprintf("%v", id))
		}
	}
	if len(parentIDs) == 0 {
		return
	}

	inverseField := fieldDef.Inverse
	if inverseField == "" {
		inverseField = r.modelName + "_id"
	}

	relatedTable := fieldDef.Model
	relQ := r.db.WithContext(ctx).Table(relatedTable).Where(fmt.Sprintf("%s IN ?", inverseField), parentIDs)
	if len(w.Select) > 0 {
		relQ = relQ.Select(w.Select)
	}
	if len(w.OrderBy) > 0 {
		for _, o := range w.OrderBy {
			dir := "ASC"
			if strings.ToLower(o.Direction) == "desc" {
				dir = "DESC"
			}
			relQ = relQ.Order(fmt.Sprintf("%s %s", o.Field, dir))
		}
	}
	if w.Limit > 0 {
		relQ = relQ.Limit(w.Limit)
	}

	var related []map[string]any
	if err := relQ.Find(&related).Error; err != nil {
		return
	}

	childMap := make(map[string][]map[string]any)
	for _, rel := range related {
		if fk, ok := rel[inverseField]; ok {
			key := fmt.Sprintf("%v", fk)
			childMap[key] = append(childMap[key], rel)
		}
	}

	for _, rec := range results {
		if id, ok := rec[r.pkCol]; ok {
			key := fmt.Sprintf("%v", id)
			if children, ok := childMap[key]; ok {
				rec["_"+w.Relation] = children
			} else {
				rec["_"+w.Relation] = []map[string]any{}
			}
		}
	}
}

func (r *GenericRepository) loadMany2ManyRelation(ctx context.Context, w WithClause, results []map[string]any) {
	for i, rec := range results {
		if id, ok := rec[r.pkCol]; ok {
			related, err := r.LoadMany2Many(ctx, fmt.Sprintf("%v", id), w.Relation)
			if err != nil {
				results[i]["_"+w.Relation] = []map[string]any{}
				continue
			}
			results[i]["_"+w.Relation] = related
		}
	}
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
