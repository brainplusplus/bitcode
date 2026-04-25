package persistence

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bitcode-framework/bitcode/internal/compiler/parser"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type MongoRepository struct {
	conn       *MongoConnection
	collection string
	modelDef   *parser.ModelDefinition
	modelName  string
	currentUser string
	tenantID   string
	refLoader  func(modelName string, id string) (string, error)
}

func NewMongoRepository(conn *MongoConnection, collection string) *MongoRepository {
	return &MongoRepository{conn: conn, collection: collection}
}

func NewMongoRepositoryWithModel(conn *MongoConnection, collection string, model *parser.ModelDefinition) *MongoRepository {
	return &MongoRepository{conn: conn, collection: collection, modelDef: model}
}

func (r *MongoRepository) SetModelDef(def *parser.ModelDefinition) {
	r.modelDef = def
}

func (r *MongoRepository) SetModelName(name string) {
	r.modelName = name
}

func (r *MongoRepository) SetCurrentUser(userID string) {
	r.currentUser = userID
}

func (r *MongoRepository) SetRefLoader(fn func(modelName string, id string) (string, error)) {
	r.refLoader = fn
}

func (r *MongoRepository) GetTableName() string {
	return r.collection
}

func (r *MongoRepository) coll() *mongo.Collection {
	return r.conn.Collection(r.collection)
}

func (r *MongoRepository) FindByID(ctx context.Context, id string) (map[string]any, error) {
	filter := bson.M{"_id": id}
	if r.tenantID != "" {
		filter["tenant_id"] = r.tenantID
	}

	var result bson.M
	err := r.coll().FindOne(ctx, filter).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("record not found in %s", r.collection)
		}
		return nil, fmt.Errorf("failed to find record in %s: %w", r.collection, err)
	}

	return mongoDocToMap(result), nil
}

func (r *MongoRepository) FindAll(ctx context.Context, query *Query, page, pageSize int) ([]map[string]any, int64, error) {
	filter := r.buildNotDeletedFilter()
	if r.tenantID != "" {
		filter["tenant_id"] = r.tenantID
	}
	applyMongoConditions(filter, query)

	total, err := r.coll().CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count in %s: %w", r.collection, err)
	}

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	offset := int64((page - 1) * pageSize)

	opts := options.Find().SetSkip(offset).SetLimit(int64(pageSize))
	if query != nil {
		sort := bson.D{}
		for _, o := range query.OrderBy {
			dir := 1
			if o.Direction == "desc" {
				dir = -1
			}
			sort = append(sort, bson.E{Key: o.Field, Value: dir})
		}
		if len(sort) > 0 {
			opts.SetSort(sort)
		}
	}

	cursor, err := r.coll().Find(ctx, filter, opts)
	if err != nil {
		return []map[string]any{}, 0, fmt.Errorf("failed to query %s: %w", r.collection, err)
	}
	defer cursor.Close(ctx)

	var results []map[string]any
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		results = append(results, mongoDocToMap(doc))
	}
	if results == nil {
		results = []map[string]any{}
	}

	return results, total, nil
}

func (r *MongoRepository) FindActive(ctx context.Context, id string) (map[string]any, error) {
	filter := r.buildActiveFilter()
	filter["_id"] = id
	if r.tenantID != "" {
		filter["tenant_id"] = r.tenantID
	}

	var result bson.M
	err := r.coll().FindOne(ctx, filter).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("record not found in %s", r.collection)
		}
		return nil, fmt.Errorf("failed to find record in %s: %w", r.collection, err)
	}

	return mongoDocToMap(result), nil
}

func (r *MongoRepository) FindAllActive(ctx context.Context, query *Query, page, pageSize int) ([]map[string]any, int64, error) {
	filter := r.buildActiveFilter()
	if r.tenantID != "" {
		filter["tenant_id"] = r.tenantID
	}
	applyMongoConditions(filter, query)

	total, err := r.coll().CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count in %s: %w", r.collection, err)
	}

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	offset := int64((page - 1) * pageSize)

	opts := options.Find().SetSkip(offset).SetLimit(int64(pageSize))
	if query != nil {
		sort := bson.D{}
		for _, o := range query.OrderBy {
			dir := 1
			if o.Direction == "desc" {
				dir = -1
			}
			sort = append(sort, bson.E{Key: o.Field, Value: dir})
		}
		if len(sort) > 0 {
			opts.SetSort(sort)
		}
	}

	cursor, err := r.coll().Find(ctx, filter, opts)
	if err != nil {
		return []map[string]any{}, 0, fmt.Errorf("failed to query %s: %w", r.collection, err)
	}
	defer cursor.Close(ctx)

	var results []map[string]any
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		results = append(results, mongoDocToMap(doc))
	}
	if results == nil {
		results = []map[string]any{}
	}

	return results, total, nil
}

func (r *MongoRepository) CountActive(ctx context.Context, query *Query) (int64, error) {
	filter := r.buildActiveFilter()
	if r.tenantID != "" {
		filter["tenant_id"] = r.tenantID
	}
	applyMongoConditions(filter, query)

	count, err := r.coll().CountDocuments(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to count in %s: %w", r.collection, err)
	}
	return count, nil
}

func (r *MongoRepository) SumActive(ctx context.Context, field string, query *Query) (float64, error) {
	matchFilter := r.buildActiveFilter()
	if r.tenantID != "" {
		matchFilter["tenant_id"] = r.tenantID
	}
	applyMongoConditions(matchFilter, query)

	pipeline := bson.A{
		bson.M{"$match": matchFilter},
		bson.M{"$group": bson.M{
			"_id":   nil,
			"total": bson.M{"$sum": "$" + field},
		}},
	}

	cursor, err := r.coll().Aggregate(ctx, pipeline)
	if err != nil {
		return 0, fmt.Errorf("failed to sum %s in %s: %w", field, r.collection, err)
	}
	defer cursor.Close(ctx)

	if cursor.Next(ctx) {
		var result bson.M
		if err := cursor.Decode(&result); err == nil {
			if total, ok := result["total"]; ok {
				switch v := total.(type) {
				case float64:
					return v, nil
				case int32:
					return float64(v), nil
				case int64:
					return float64(v), nil
				}
			}
		}
	}
	return 0, nil
}

func (r *MongoRepository) Create(ctx context.Context, data map[string]any) (map[string]any, error) {
	if r.tenantID != "" {
		data["tenant_id"] = r.tenantID
	}
	if _, hasID := data["id"]; hasID {
		data["_id"] = data["id"]
		delete(data, "id")
	}
	if _, hasActive := data["active"]; !hasActive {
		data["active"] = true
	}

	r.populateExtendedRefs(ctx, data)

	_, err := r.coll().InsertOne(ctx, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create in %s: %w", r.collection, err)
	}

	if id, ok := data["_id"]; ok {
		data["id"] = id
		delete(data, "_id")
	}
	return data, nil
}

func (r *MongoRepository) Update(ctx context.Context, id string, data map[string]any) error {
	delete(data, "id")
	delete(data, "_id")

	r.populateExtendedRefs(ctx, data)

	result, err := r.coll().UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": data})
	if err != nil {
		return fmt.Errorf("failed to update in %s: %w", r.collection, err)
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("record not found in %s with id %s", r.collection, id)
	}
	return nil
}

func (r *MongoRepository) Delete(ctx context.Context, id string) error {
	result, err := r.coll().UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{"active": false}})
	if err != nil {
		return fmt.Errorf("failed to soft-delete in %s: %w", r.collection, err)
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("record not found in %s with id %s", r.collection, id)
	}
	return nil
}

func (r *MongoRepository) HardDelete(ctx context.Context, id string) error {
	result, err := r.coll().DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("failed to delete in %s: %w", r.collection, err)
	}
	if result.DeletedCount == 0 {
		return fmt.Errorf("record not found in %s with id %s", r.collection, id)
	}
	return nil
}

func (r *MongoRepository) Upsert(ctx context.Context, data map[string]any, uniqueFields []string) (map[string]any, error) {
	if len(uniqueFields) == 0 {
		return r.Create(ctx, data)
	}

	filter := bson.M{}
	for _, f := range uniqueFields {
		if v, ok := data[f]; ok {
			filter[f] = v
		}
	}

	if _, hasActive := data["active"]; !hasActive {
		data["active"] = true
	}

	r.populateExtendedRefs(ctx, data)

	id := data["id"]
	if id != nil {
		data["_id"] = id
		delete(data, "id")
	}

	opts := options.UpdateOne().SetUpsert(true)
	_, err := r.coll().UpdateOne(ctx, filter, bson.M{"$set": data}, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert in %s: %w", r.collection, err)
	}

	var result bson.M
	if err := r.coll().FindOne(ctx, filter).Decode(&result); err != nil {
		return nil, err
	}
	return mongoDocToMap(result), nil
}

func (r *MongoRepository) Count(ctx context.Context, query *Query) (int64, error) {
	filter := r.buildNotDeletedFilter()
	if r.tenantID != "" {
		filter["tenant_id"] = r.tenantID
	}
	applyMongoConditions(filter, query)

	count, err := r.coll().CountDocuments(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to count in %s: %w", r.collection, err)
	}
	return count, nil
}

func (r *MongoRepository) Sum(ctx context.Context, field string, query *Query) (float64, error) {
	matchFilter := r.buildNotDeletedFilter()
	if r.tenantID != "" {
		matchFilter["tenant_id"] = r.tenantID
	}
	applyMongoConditions(matchFilter, query)

	pipeline := bson.A{
		bson.M{"$match": matchFilter},
		bson.M{"$group": bson.M{
			"_id":   nil,
			"total": bson.M{"$sum": "$" + field},
		}},
	}

	cursor, err := r.coll().Aggregate(ctx, pipeline)
	if err != nil {
		return 0, fmt.Errorf("failed to sum %s in %s: %w", field, r.collection, err)
	}
	defer cursor.Close(ctx)

	if cursor.Next(ctx) {
		var result bson.M
		if err := cursor.Decode(&result); err == nil {
			if total, ok := result["total"]; ok {
				switch v := total.(type) {
				case float64:
					return v, nil
				case int32:
					return float64(v), nil
				case int64:
					return float64(v), nil
				}
			}
		}
	}
	return 0, nil
}

func (r *MongoRepository) BulkCreate(ctx context.Context, records []map[string]any) ([]map[string]any, error) {
	docs := make([]any, len(records))
	for i, rec := range records {
		if r.tenantID != "" {
			rec["tenant_id"] = r.tenantID
		}
		if _, hasID := rec["id"]; hasID {
			rec["_id"] = rec["id"]
			delete(rec, "id")
		}
		if _, hasActive := rec["active"]; !hasActive {
			rec["active"] = true
		}
		r.populateExtendedRefs(ctx, rec)
		docs[i] = rec
	}

	_, err := r.coll().InsertMany(ctx, docs)
	if err != nil {
		return nil, fmt.Errorf("failed to bulk create in %s: %w", r.collection, err)
	}

	for i := range records {
		if id, ok := records[i]["_id"]; ok {
			records[i]["id"] = id
			delete(records[i], "_id")
		}
	}
	return records, nil
}

func (r *MongoRepository) AddMany2Many(ctx context.Context, id string, field string, relatedIDs []string) error {
	refs, err := r.buildM2MRefs(ctx, field, relatedIDs)
	if err != nil {
		return err
	}

	arrayField := field + "_ids"
	update := bson.M{
		"$addToSet": bson.M{
			arrayField: bson.M{"$each": relatedIDs},
		},
	}
	if refs != nil {
		update["$set"] = bson.M{
			"_refs." + arrayField: refs,
		}
	}

	_, err = r.coll().UpdateOne(ctx, bson.M{"_id": id}, update)
	return err
}

func (r *MongoRepository) RemoveMany2Many(ctx context.Context, id string, field string, relatedIDs []string) error {
	arrayField := field + "_ids"
	update := bson.M{
		"$pull": bson.M{
			arrayField: bson.M{"$in": relatedIDs},
		},
	}

	_, err := r.coll().UpdateOne(ctx, bson.M{"_id": id}, update)
	if err != nil {
		return err
	}

	var doc bson.M
	if err := r.coll().FindOne(ctx, bson.M{"_id": id}).Decode(&doc); err == nil {
		if ids, ok := doc[arrayField]; ok {
			if idArr, ok := ids.(bson.A); ok {
				var remaining []string
				for _, v := range idArr {
					remaining = append(remaining, fmt.Sprintf("%v", v))
				}
				refs, _ := r.buildM2MRefs(ctx, field, remaining)
				if refs != nil {
					r.coll().UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{"_refs." + arrayField: refs}})
				}
			}
		}
	}

	return nil
}

func (r *MongoRepository) LoadMany2Many(ctx context.Context, id string, field string) ([]map[string]any, error) {
	var doc bson.M
	if err := r.coll().FindOne(ctx, bson.M{"_id": id}).Decode(&doc); err != nil {
		return nil, err
	}

	arrayField := field + "_ids"
	ids, ok := doc[arrayField]
	if !ok {
		return []map[string]any{}, nil
	}

	idArr, ok := ids.(bson.A)
	if !ok {
		return []map[string]any{}, nil
	}

	var relatedIDs []any
	for _, v := range idArr {
		relatedIDs = append(relatedIDs, v)
	}

	if len(relatedIDs) == 0 {
		return []map[string]any{}, nil
	}

	relatedModel := field
	if r.modelDef != nil {
		if fd, ok := r.modelDef.Fields[field]; ok && fd.Model != "" {
			relatedModel = fd.Model
		}
	}

	relatedColl := r.conn.Collection(relatedModel)
	cursor, err := relatedColl.Find(ctx, bson.M{"_id": bson.M{"$in": relatedIDs}})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []map[string]any
	for cursor.Next(ctx) {
		var d bson.M
		if err := cursor.Decode(&d); err != nil {
			continue
		}
		results = append(results, mongoDocToMap(d))
	}
	if results == nil {
		results = []map[string]any{}
	}
	return results, nil
}

func (r *MongoRepository) populateExtendedRefs(ctx context.Context, data map[string]any) {
	if r.modelDef == nil || r.refLoader == nil {
		return
	}

	refs, _ := data["_refs"].(map[string]any)
	if refs == nil {
		refs = map[string]any{}
	}
	changed := false

	for fieldName, field := range r.modelDef.Fields {
		if field.Type != parser.FieldMany2One {
			continue
		}
		fkValue, ok := data[fieldName]
		if !ok {
			continue
		}
		fkStr := fmt.Sprintf("%v", fkValue)
		if fkStr == "" || fkStr == "<nil>" {
			continue
		}

		title, err := r.refLoader(field.Model, fkStr)
		if err != nil {
			continue
		}

		refs[fieldName] = map[string]any{
			"_id":    fkStr,
			"_title": title,
		}
		changed = true
	}

	if changed {
		data["_refs"] = refs
	}
}

func (r *MongoRepository) buildM2MRefs(ctx context.Context, field string, ids []string) ([]map[string]any, error) {
	if r.refLoader == nil || r.modelDef == nil {
		return nil, nil
	}

	fd, ok := r.modelDef.Fields[field]
	if !ok {
		return nil, nil
	}

	var refs []map[string]any
	for _, id := range ids {
		title, err := r.refLoader(fd.Model, id)
		if err != nil {
			title = ""
		}
		refs = append(refs, map[string]any{
			"_id":    id,
			"_title": title,
		})
	}
	return refs, nil
}

func (r *MongoRepository) buildNotDeletedFilter() bson.M {
	if r.modelDef != nil && r.modelDef.IsSoftDeletes() {
		return bson.M{"deleted_at": bson.M{"$eq": nil}}
	}
	return bson.M{}
}

func (r *MongoRepository) buildActiveFilter() bson.M {
	filter := bson.M{"active": true}
	if r.modelDef != nil && r.modelDef.IsSoftDeletes() {
		filter["deleted_at"] = bson.M{"$eq": nil}
	}
	return filter
}

func (r *MongoRepository) UpdateWithVersion(ctx context.Context, id string, data map[string]any, expectedVersion int) error {
	delete(data, "id")
	delete(data, "_id")

	r.populateExtendedRefs(ctx, data)

	data["version"] = expectedVersion + 1
	filter := bson.M{"_id": id, "version": expectedVersion}
	result, err := r.coll().UpdateOne(ctx, filter, bson.M{"$set": data})
	if err != nil {
		return fmt.Errorf("failed to update in %s: %w", r.collection, err)
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("version conflict")
	}
	return nil
}

func (r *MongoRepository) SoftDeleteWithTimestamp(ctx context.Context, id string, deletedAt time.Time, deletedBy string) error {
	updates := bson.M{"active": false, "deleted_at": deletedAt}
	if r.modelDef != nil && r.modelDef.IsSoftDeletesBy() && deletedBy != "" {
		updates["deleted_by"] = deletedBy
	}

	result, err := r.coll().UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": updates})
	if err != nil {
		return fmt.Errorf("failed to soft-delete in %s: %w", r.collection, err)
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("record not found in %s with id %s", r.collection, id)
	}
	return nil
}

func applyMongoConditions(filter bson.M, query *Query) {
	if query == nil {
		return
	}
	for _, cond := range query.Conditions {
		field := cond.Field
		if field == "id" {
			field = "_id"
		}
		switch cond.Operator {
		case "=":
			filter[field] = cond.Value
		case "!=":
			filter[field] = bson.M{"$ne": cond.Value}
		case ">":
			filter[field] = bson.M{"$gt": cond.Value}
		case "<":
			filter[field] = bson.M{"$lt": cond.Value}
		case ">=":
			filter[field] = bson.M{"$gte": cond.Value}
		case "<=":
			filter[field] = bson.M{"$lte": cond.Value}
		case "like":
			pattern := fmt.Sprintf("%v", cond.Value)
			pattern = strings.TrimPrefix(pattern, "%")
			pattern = strings.TrimSuffix(pattern, "%")
			filter[field] = bson.M{"$regex": pattern, "$options": "i"}
		case "in":
			filter[field] = bson.M{"$in": cond.Value}
		case "not_in":
			filter[field] = bson.M{"$nin": cond.Value}
		case "is_null":
			filter[field] = bson.M{"$eq": nil}
		case "is_not_null":
			filter[field] = bson.M{"$ne": nil}
		case "between":
			if vals, ok := cond.Value.([]any); ok && len(vals) == 2 {
				filter[field] = bson.M{"$gte": vals[0], "$lte": vals[1]}
			}
		}
	}
}

func mongoDocToMap(doc bson.M) map[string]any {
	result := make(map[string]any, len(doc))
	for k, v := range doc {
		if k == "_id" {
			result["id"] = v
			continue
		}
		result[k] = v
	}
	return result
}
