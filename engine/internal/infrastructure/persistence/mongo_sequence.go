package persistence

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type MongoSequenceEngine struct {
	conn *MongoConnection
}

func NewMongoSequenceEngine(conn *MongoConnection) *MongoSequenceEngine {
	return &MongoSequenceEngine{conn: conn}
}

func (e *MongoSequenceEngine) MigrateSequenceTable() error {
	return nil
}

func (e *MongoSequenceEngine) NextValue(modelName, fieldName, sequenceKey string, step int) (int64, error) {
	if step <= 0 {
		step = 1
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	coll := e.conn.Collection("_sequences")

	filter := bson.M{
		"model_name":   modelName,
		"field_name":   fieldName,
		"sequence_key": sequenceKey,
	}

	update := bson.M{
		"$inc": bson.M{"next_value": int64(step)},
		"$setOnInsert": bson.M{
			"model_name":   modelName,
			"field_name":   fieldName,
			"sequence_key": sequenceKey,
			"step":         step,
			"created_at":   time.Now(),
		},
		"$set": bson.M{
			"updated_at": time.Now(),
		},
	}

	opts := options.FindOneAndUpdate().
		SetUpsert(true).
		SetReturnDocument(options.After)

	var result bson.M
	err := coll.FindOneAndUpdate(ctx, filter, update, opts).Decode(&result)
	if err != nil {
		return 0, fmt.Errorf("failed to get next sequence value: %w", err)
	}

	nextValue, ok := result["next_value"]
	if !ok {
		return 0, fmt.Errorf("next_value not found in sequence result")
	}

	var val int64
	switch v := nextValue.(type) {
	case int64:
		val = v
	case int32:
		val = int64(v)
	case float64:
		val = int64(v)
	default:
		return 0, fmt.Errorf("unexpected next_value type: %T", nextValue)
	}

	return val - int64(step), nil
}
