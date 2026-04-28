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
	conn           *MongoConnection
	collection     string
	modelDef       *parser.ModelDefinition
	modelName      string
	currentUser    string
	tenantID       string
	refLoader      func(modelName string, id string) (string, error)
	hookDispatcher HookDispatcher
	validator      FieldValidator
	sanitizer      FieldSanitizer
	eventBus       EventPublisher
	locale         string
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

func (r *MongoRepository) SetHookDispatcher(d HookDispatcher) {
	r.hookDispatcher = d
}

func (r *MongoRepository) SetValidator(v FieldValidator) {
	r.validator = v
}

func (r *MongoRepository) SetSanitizer(s FieldSanitizer) {
	r.sanitizer = s
}

func (r *MongoRepository) SetEventBus(bus EventPublisher) {
	r.eventBus = bus
}

func (r *MongoRepository) SetLocale(locale string) {
	r.locale = locale
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

	if r.modelDef != nil {
		if r.sanitizer != nil {
			r.sanitizer.SanitizeRecord(r.modelDef, data)
		}
		if r.validator != nil {
			if err := r.validator.ValidateCreate(r.modelDef, data, r.locale); err != nil {
				return nil, err
			}
		}
		if r.hookDispatcher != nil {
			session := r.mongoBuildSession()
			if err := r.hookDispatcher.DispatchCreate(ctx, r.modelDef, data, session); err != nil {
				return nil, err
			}
		}
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

	if r.modelDef != nil {
		if r.hookDispatcher != nil {
			session := r.mongoBuildSession()
			r.hookDispatcher.DispatchAfterCreate(ctx, r.modelDef, data, session)
		}
		if r.eventBus != nil {
			r.eventBus.Publish(ctx, "model."+r.modelName+".created", data)
		}
	}

	return data, nil
}

func (r *MongoRepository) Update(ctx context.Context, id string, data map[string]any) error {
	delete(data, "id")
	delete(data, "_id")

	if r.modelDef != nil {
		var before map[string]any
		if err := r.coll().FindOne(ctx, bson.M{"_id": id}).Decode(&before); err == nil {
			if bid, ok := before["_id"]; ok {
				before["id"] = bid
				delete(before, "_id")
			}

			changes := data
			merged := r.mongoMerge(before, data)

			if r.sanitizer != nil {
				r.sanitizer.SanitizeChangedFields(r.modelDef, merged, changes)
				for k := range changes {
					if v, ok := merged[k]; ok {
						data[k] = v
					}
				}
			}

			if r.validator != nil {
				merged["__old"] = before
				if err := r.validator.ValidateUpdate(r.modelDef, merged, changes, r.locale); err != nil {
					return err
				}
				delete(merged, "__old")
			}

			session := r.mongoBuildSession()
			if r.hookDispatcher != nil {
				if err := r.hookDispatcher.DispatchBeforeUpdate(ctx, r.modelDef, merged, before, changes, session); err != nil {
					return err
				}
				for k := range changes {
					if v, ok := merged[k]; ok {
						data[k] = v
					}
				}
			}
		}
	}

	r.populateExtendedRefs(ctx, data)

	result, err := r.coll().UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": data})
	if err != nil {
		return fmt.Errorf("failed to update in %s: %w", r.collection, err)
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("record not found in %s with id %s", r.collection, id)
	}

	if r.modelDef != nil {
		session := r.mongoBuildSession()
		if r.hookDispatcher != nil {
			r.hookDispatcher.DispatchAfterUpdate(ctx, r.modelDef, data, nil, data, session)
		}
		if r.eventBus != nil {
			r.eventBus.Publish(ctx, "model."+r.modelName+".updated", map[string]any{"record": data, "changes": data})
		}
	}

	return nil
}

func (r *MongoRepository) Delete(ctx context.Context, id string) error {
	if r.modelDef != nil && r.hookDispatcher != nil {
		var before map[string]any
		if err := r.coll().FindOne(ctx, bson.M{"_id": id}).Decode(&before); err == nil {
			session := r.mongoBuildSession()
			if err := r.hookDispatcher.DispatchBeforeDelete(ctx, r.modelDef, before, true, session); err != nil {
				return err
			}
		}
	}

	result, err := r.coll().UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{"active": false}})
	if err != nil {
		return fmt.Errorf("failed to soft-delete in %s: %w", r.collection, err)
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("record not found in %s with id %s", r.collection, id)
	}

	if r.modelDef != nil {
		session := r.mongoBuildSession()
		if r.hookDispatcher != nil {
			r.hookDispatcher.DispatchAfterDelete(ctx, r.modelDef, map[string]any{"id": id}, true, session)
		}
		if r.eventBus != nil {
			r.eventBus.Publish(ctx, "model."+r.modelName+".deleted", map[string]any{"id": id})
		}
	}

	return nil
}

func (r *MongoRepository) HardDelete(ctx context.Context, id string) error {
	if r.modelDef != nil && r.hookDispatcher != nil {
		var before map[string]any
		if err := r.coll().FindOne(ctx, bson.M{"_id": id}).Decode(&before); err == nil {
			session := r.mongoBuildSession()
			if err := r.hookDispatcher.DispatchBeforeDelete(ctx, r.modelDef, before, false, session); err != nil {
				return err
			}
		}
	}

	result, err := r.coll().DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("failed to delete in %s: %w", r.collection, err)
	}
	if result.DeletedCount == 0 {
		return fmt.Errorf("record not found in %s with id %s", r.collection, id)
	}

	if r.modelDef != nil {
		session := r.mongoBuildSession()
		if r.hookDispatcher != nil {
			r.hookDispatcher.DispatchAfterDelete(ctx, r.modelDef, map[string]any{"id": id}, false, session)
		}
		if r.eventBus != nil {
			r.eventBus.Publish(ctx, "model."+r.modelName+".deleted", map[string]any{"id": id})
		}
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

func (r *MongoRepository) BulkUpdate(ctx context.Context, ids []string, data map[string]any) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	filter := bson.M{"_id": bson.M{"$in": ids}}
	if r.tenantID != "" {
		filter["tenant_id"] = r.tenantID
	}
	update := bson.M{"$set": data}
	result, err := r.coll().UpdateMany(ctx, filter, update)
	if err != nil {
		return 0, fmt.Errorf("failed to bulk update in %s: %w", r.collection, err)
	}
	return result.ModifiedCount, nil
}

func (r *MongoRepository) BulkDelete(ctx context.Context, ids []string) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	filter := bson.M{"_id": bson.M{"$in": ids}}
	if r.tenantID != "" {
		filter["tenant_id"] = r.tenantID
	}
	update := bson.M{"$set": bson.M{"active": false}}
	result, err := r.coll().UpdateMany(ctx, filter, update)
	if err != nil {
		return 0, fmt.Errorf("failed to bulk soft-delete in %s: %w", r.collection, err)
	}
	return result.ModifiedCount, nil
}

func (r *MongoRepository) BulkHardDelete(ctx context.Context, ids []string) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	filter := bson.M{"_id": bson.M{"$in": ids}}
	if r.tenantID != "" {
		filter["tenant_id"] = r.tenantID
	}
	result, err := r.coll().DeleteMany(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to bulk hard-delete in %s: %w", r.collection, err)
	}
	return result.DeletedCount, nil
}

func (r *MongoRepository) BulkUpsert(ctx context.Context, records []map[string]any, uniqueFields []string) ([]map[string]any, error) {
	if len(records) == 0 {
		return nil, nil
	}
	var models []mongo.WriteModel
	for _, rec := range records {
		if r.tenantID != "" {
			rec["tenant_id"] = r.tenantID
		}
		filter := bson.M{}
		for _, uf := range uniqueFields {
			if v, ok := rec[uf]; ok {
				filter[uf] = v
			}
		}
		if r.tenantID != "" {
			filter["tenant_id"] = r.tenantID
		}
		model := mongo.NewUpdateOneModel().
			SetFilter(filter).
			SetUpdate(bson.M{"$set": rec}).
			SetUpsert(true)
		models = append(models, model)
	}
	_, err := r.coll().BulkWrite(ctx, models)
	if err != nil {
		return nil, fmt.Errorf("failed to bulk upsert in %s: %w", r.collection, err)
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

func (r *MongoRepository) Avg(ctx context.Context, field string, query *Query) (float64, error) {
	return r.mongoAggregateSingle(ctx, "$avg", field, query, r.buildNotDeletedFilter())
}

func (r *MongoRepository) Min(ctx context.Context, field string, query *Query) (float64, error) {
	return r.mongoAggregateSingle(ctx, "$min", field, query, r.buildNotDeletedFilter())
}

func (r *MongoRepository) Max(ctx context.Context, field string, query *Query) (float64, error) {
	return r.mongoAggregateSingle(ctx, "$max", field, query, r.buildNotDeletedFilter())
}

func (r *MongoRepository) mongoAggregateSingle(ctx context.Context, op, field string, query *Query, baseFilter bson.M) (float64, error) {
	if r.tenantID != "" {
		baseFilter["tenant_id"] = r.tenantID
	}
	applyMongoConditions(baseFilter, query)

	pipeline := bson.A{
		bson.M{"$match": baseFilter},
		bson.M{"$group": bson.M{
			"_id":    nil,
			"result": bson.M{op: "$" + field},
		}},
	}

	cursor, err := r.coll().Aggregate(ctx, pipeline)
	if err != nil {
		return 0, fmt.Errorf("failed to %s %s in %s: %w", op, field, r.collection, err)
	}
	defer cursor.Close(ctx)

	if cursor.Next(ctx) {
		var result bson.M
		if err := cursor.Decode(&result); err == nil {
			if v, ok := result["result"]; ok {
				switch val := v.(type) {
				case float64:
					return val, nil
				case int32:
					return float64(val), nil
				case int64:
					return float64(val), nil
				}
			}
		}
	}
	return 0, nil
}

func (r *MongoRepository) Pluck(ctx context.Context, field string, query *Query) ([]any, error) {
	filter := r.buildNotDeletedFilter()
	if r.tenantID != "" {
		filter["tenant_id"] = r.tenantID
	}
	applyMongoConditions(filter, query)

	projection := bson.M{field: 1, "_id": 0}
	opts := options.Find().SetProjection(projection)

	cursor, err := r.coll().Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to pluck %s in %s: %w", field, r.collection, err)
	}
	defer cursor.Close(ctx)

	var results []any
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		if v, ok := doc[field]; ok {
			results = append(results, v)
		}
	}
	return results, nil
}

func (r *MongoRepository) Exists(ctx context.Context, query *Query) (bool, error) {
	count, err := r.Count(ctx, query)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *MongoRepository) Aggregate(ctx context.Context, query *Query) ([]map[string]any, error) {
	filter := r.buildNotDeletedFilter()
	if r.tenantID != "" {
		filter["tenant_id"] = r.tenantID
	}
	applyMongoConditions(filter, query)

	pipeline := bson.A{bson.M{"$match": filter}}

	if query != nil && len(query.GroupBy) > 0 {
		groupID := bson.M{}
		for _, g := range query.GroupBy {
			groupID[g] = "$" + g
		}
		groupStage := bson.M{"_id": groupID}
		for _, agg := range query.Aggregates {
			alias := agg.Alias
			if alias == "" {
				alias = strings.ToLower(agg.Function) + "_" + agg.Field
			}
			switch strings.ToUpper(agg.Function) {
			case "COUNT":
				groupStage[alias] = bson.M{"$sum": 1}
			case "SUM":
				groupStage[alias] = bson.M{"$sum": "$" + agg.Field}
			case "AVG":
				groupStage[alias] = bson.M{"$avg": "$" + agg.Field}
			case "MIN":
				groupStage[alias] = bson.M{"$min": "$" + agg.Field}
			case "MAX":
				groupStage[alias] = bson.M{"$max": "$" + agg.Field}
			}
		}
		pipeline = append(pipeline, bson.M{"$group": groupStage})
	}

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
			pipeline = append(pipeline, bson.M{"$sort": sort})
		}
	}

	cursor, err := r.coll().Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate in %s: %w", r.collection, err)
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
	return results, nil
}

func (r *MongoRepository) Chunk(ctx context.Context, query *Query, size int, fn func(records []map[string]any) error) error {
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

func (r *MongoRepository) Increment(ctx context.Context, id string, field string, value int) error {
	result, err := r.coll().UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$inc": bson.M{field: value}})
	if err != nil {
		return fmt.Errorf("failed to increment %s in %s: %w", field, r.collection, err)
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("record not found in %s with id %s", r.collection, id)
	}
	return nil
}

func (r *MongoRepository) Decrement(ctx context.Context, id string, field string, value int) error {
	return r.Increment(ctx, id, field, -value)
}

func (r *MongoRepository) Transaction(ctx context.Context, fn func(txRepo Repository) error) error {
	session, err := r.conn.Client.StartSession()
	if err != nil {
		return fmt.Errorf("failed to start session: %w", err)
	}
	defer session.EndSession(ctx)

	_, err = session.WithTransaction(ctx, func(sc context.Context) (any, error) {
		txRepo := &MongoRepository{
			conn:        r.conn,
			collection:  r.collection,
			modelDef:    r.modelDef,
			modelName:   r.modelName,
			currentUser: r.currentUser,
			tenantID:    r.tenantID,
			refLoader:   r.refLoader,
		}
		return nil, fn(txRepo)
	})
	return err
}

func (r *MongoRepository) RawQuery(ctx context.Context, sql string, values ...any) ([]map[string]any, error) {
	return nil, fmt.Errorf("RawQuery not supported for MongoDB — use Aggregate instead")
}

func (r *MongoRepository) RawExec(ctx context.Context, sql string, values ...any) (int64, error) {
	return 0, fmt.Errorf("RawExec not supported for MongoDB")
}

func (r *MongoRepository) FindAllWithTrashed(ctx context.Context, query *Query, page, pageSize int) ([]map[string]any, int64, error) {
	filter := bson.M{}
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

func (r *MongoRepository) FindAllOnlyTrashed(ctx context.Context, query *Query, page, pageSize int) ([]map[string]any, int64, error) {
	filter := bson.M{}
	if r.modelDef != nil && r.modelDef.IsSoftDeletes() {
		filter["deleted_at"] = bson.M{"$ne": nil}
	} else {
		filter["active"] = false
	}
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

	query.ApplyScopes()

	clauses := query.GetEffectiveWhereClauses()
	if len(clauses) > 0 {
		andConditions := buildMongoWhereClauses(clauses)
		if len(andConditions) > 0 {
			if existing, ok := filter["$and"]; ok {
				if arr, ok := existing.(bson.A); ok {
					filter["$and"] = append(arr, andConditions...)
				}
			} else {
				filter["$and"] = andConditions
			}
		}
	}

	for _, raw := range query.WhereRaw {
		if raw.SQL != "" {
			if existing, ok := filter["$and"]; ok {
				if arr, ok := existing.(bson.A); ok {
					filter["$and"] = append(arr, bson.M{"$where": raw.SQL})
				}
			} else {
				filter["$and"] = bson.A{bson.M{"$where": raw.SQL}}
			}
		}
	}
}

func buildMongoWhereClauses(clauses []WhereClause) bson.A {
	var result bson.A
	for _, clause := range clauses {
		if clause.Condition != nil {
			cond := buildMongoSingleCondition(*clause.Condition)
			if cond != nil {
				result = append(result, cond)
			}
		}
		if clause.Group != nil {
			groupFilter := buildMongoConditionGroup(*clause.Group)
			if groupFilter != nil {
				result = append(result, groupFilter)
			}
		}
	}
	return result
}

func buildMongoConditionGroup(group ConditionGroup) bson.M {
	subConditions := buildMongoWhereClauses(group.Conditions)
	if len(subConditions) == 0 {
		return nil
	}

	var result bson.M
	if group.Connector == ConnectorOr {
		result = bson.M{"$or": subConditions}
	} else {
		result = bson.M{"$and": subConditions}
	}

	if group.Negate {
		return bson.M{"$nor": bson.A{result}}
	}
	return result
}

func buildMongoSingleCondition(cond Condition) bson.M {
	field := cond.Field
	if field == "id" {
		field = "_id"
	}

	switch cond.Operator {
	case "=":
		return bson.M{field: cond.Value}
	case "!=":
		return bson.M{field: bson.M{"$ne": cond.Value}}
	case ">":
		return bson.M{field: bson.M{"$gt": cond.Value}}
	case "<":
		return bson.M{field: bson.M{"$lt": cond.Value}}
	case ">=":
		return bson.M{field: bson.M{"$gte": cond.Value}}
	case "<=":
		return bson.M{field: bson.M{"$lte": cond.Value}}
	case "like":
		pattern := fmt.Sprintf("%v", cond.Value)
		pattern = strings.TrimPrefix(pattern, "%")
		pattern = strings.TrimSuffix(pattern, "%")
		return bson.M{field: bson.M{"$regex": pattern, "$options": "i"}}
	case "not_like":
		pattern := fmt.Sprintf("%v", cond.Value)
		pattern = strings.TrimPrefix(pattern, "%")
		pattern = strings.TrimSuffix(pattern, "%")
		return bson.M{field: bson.M{"$not": bson.M{"$regex": pattern, "$options": "i"}}}
	case "in":
		return bson.M{field: bson.M{"$in": cond.Value}}
	case "not_in":
		return bson.M{field: bson.M{"$nin": cond.Value}}
	case "is_null":
		return bson.M{field: bson.M{"$eq": nil}}
	case "is_not_null":
		return bson.M{field: bson.M{"$ne": nil}}
	case "between":
		if vals, ok := cond.Value.([]any); ok && len(vals) == 2 {
			return bson.M{field: bson.M{"$gte": vals[0], "$lte": vals[1]}}
		}
	case "not_between":
		if vals, ok := cond.Value.([]any); ok && len(vals) == 2 {
			return bson.M{"$or": bson.A{
				bson.M{field: bson.M{"$lt": vals[0]}},
				bson.M{field: bson.M{"$gt": vals[1]}},
			}}
		}
	}
	return nil
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

func (r *MongoRepository) mongoBuildSession() map[string]any {
	session := make(map[string]any)
	if r.currentUser != "" {
		session["user_id"] = r.currentUser
	}
	if r.tenantID != "" {
		session["tenant_id"] = r.tenantID
	}
	return session
}

func (r *MongoRepository) mongoMerge(old map[string]any, incoming map[string]any) map[string]any {
	merged := make(map[string]any, len(old))
	for k, v := range old {
		merged[k] = v
	}
	for k, v := range incoming {
		merged[k] = v
	}
	return merged
}
