package persistence

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type MongoSystemRepository struct {
	conn *MongoConnection
}

func NewMongoSystemRepository(conn *MongoConnection) *MongoSystemRepository {
	return &MongoSystemRepository{conn: conn}
}

func (r *MongoSystemRepository) Insert(ctx context.Context, collection string, data map[string]any) error {
	if id, ok := data["id"]; ok {
		data["_id"] = id
		delete(data, "id")
	}
	_, err := r.conn.Collection(collection).InsertOne(ctx, data)
	return err
}

func (r *MongoSystemRepository) Find(ctx context.Context, collection string, query *Query, limit int) ([]map[string]any, error) {
	filter := bson.M{}
	applyMongoConditions(filter, query)

	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	if limit > 0 {
		opts.SetLimit(int64(limit))
	}

	cursor, err := r.conn.Collection(collection).Find(ctx, filter, opts)
	if err != nil {
		return nil, err
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

func (r *MongoSystemRepository) FindOne(ctx context.Context, collection string, query *Query) (map[string]any, error) {
	filter := bson.M{}
	applyMongoConditions(filter, query)

	var doc bson.M
	err := r.conn.Collection(collection).FindOne(ctx, filter).Decode(&doc)
	if err != nil {
		return nil, err
	}
	return mongoDocToMap(doc), nil
}

func (r *MongoSystemRepository) Count(ctx context.Context, collection string, query *Query) (int64, error) {
	filter := bson.M{}
	applyMongoConditions(filter, query)
	return r.conn.Collection(collection).CountDocuments(ctx, filter)
}

func (r *MongoSystemRepository) Migrate(collection string, columns []SystemColumn) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = r.conn.Database.CreateCollection(ctx, collection)
	return nil
}

type GormSystemRepository struct {
	db interface {
		Table(string) interface {
			Create(any) interface{ Error() error }
		}
	}
}

type MongoAuditLogRepository struct {
	conn *MongoConnection
}

func NewMongoAuditLogRepository(conn *MongoConnection) *MongoAuditLogRepository {
	return &MongoAuditLogRepository{conn: conn}
}

func (r *MongoAuditLogRepository) Write(entry AuditLogEntry) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	record := bson.M{
		"_id":              fmt.Sprintf("%d", time.Now().UnixNano()),
		"user_id":          nilIfEmpty(entry.UserID),
		"action":           entry.Action,
		"model_name":       nilIfEmpty(entry.ModelName),
		"record_id":        nilIfEmpty(entry.RecordID),
		"changes":          nilIfEmpty(entry.Changes),
		"ip_address":       nilIfEmpty(entry.IPAddress),
		"user_agent":       nilIfEmpty(entry.UserAgent),
		"request_method":   nilIfEmpty(entry.RequestMethod),
		"request_path":     nilIfEmpty(entry.RequestPath),
		"status_code":      nilIfZero(entry.StatusCode),
		"duration_ms":      nilIfZero(entry.DurationMs),
		"impersonated_by":  nilIfEmpty(entry.ImpersonatedBy),
		"created_at":       time.Now(),
		"updated_at":       time.Now(),
		"active":           true,
	}

	_, err := r.conn.Collection("audit_logs").InsertOne(ctx, record)
	return err
}

func (r *MongoAuditLogRepository) WriteAsync(entry AuditLogEntry) {
	go func() {
		r.Write(entry)
	}()
}

func (r *MongoAuditLogRepository) FindByRecord(modelName, recordID string, limit int) ([]map[string]any, error) {
	ctx := context.Background()
	q := NewQuery().Where("model_name", "=", modelName).Where("record_id", "=", recordID)
	sysRepo := NewMongoSystemRepository(r.conn)
	return sysRepo.Find(ctx, "audit_logs", q, limit)
}

func (r *MongoAuditLogRepository) FindByAction(action string, limit int) ([]map[string]any, error) {
	ctx := context.Background()
	q := NewQuery().Where("action", "=", action)
	sysRepo := NewMongoSystemRepository(r.conn)
	return sysRepo.Find(ctx, "audit_logs", q, limit)
}

func (r *MongoAuditLogRepository) FindByUser(userID string, limit int) ([]map[string]any, error) {
	ctx := context.Background()
	q := NewQuery().Where("user_id", "=", userID)
	sysRepo := NewMongoSystemRepository(r.conn)
	return sysRepo.Find(ctx, "audit_logs", q, limit)
}

func (r *MongoAuditLogRepository) FindLoginHistory(limit int) ([]map[string]any, error) {
	ctx := context.Background()
	q := NewQuery().Where("action", "in", []string{"login", "logout", "register"})
	sysRepo := NewMongoSystemRepository(r.conn)
	return sysRepo.Find(ctx, "audit_logs", q, limit)
}

func (r *MongoAuditLogRepository) FindRequests(limit int, methodFilter string) ([]map[string]any, error) {
	ctx := context.Background()
	q := NewQuery().Where("action", "=", "request")
	if methodFilter != "" {
		q.Where("request_method", "=", methodFilter)
	}
	sysRepo := NewMongoSystemRepository(r.conn)
	return sysRepo.Find(ctx, "audit_logs", q, limit)
}
