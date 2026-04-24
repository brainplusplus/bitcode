package persistence

import (
	"context"
	"fmt"
	"time"

	"github.com/bitcode-engine/engine/internal/compiler/parser"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type MongoMigrationEngine struct {
	conn *MongoConnection
}

func NewMongoMigrationEngine(conn *MongoConnection) *MongoMigrationEngine {
	return &MongoMigrationEngine{conn: conn}
}

func (e *MongoMigrationEngine) MigrateModel(model *parser.ModelDefinition, resolver TableNameResolver) error {
	tableName := model.Name
	if resolver != nil {
		tableName = resolver.TableName(model.Name)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	coll := e.conn.Collection(tableName)

	for fieldName, field := range model.Fields {
		if field.Unique {
			indexModel := mongo.IndexModel{
				Keys:    bson.D{{Key: fieldName, Value: 1}},
				Options: options.Index().SetUnique(true).SetSparse(true),
			}
			coll.Indexes().CreateOne(ctx, indexModel)
		}

		if field.Type == parser.FieldMany2One {
			indexModel := mongo.IndexModel{
				Keys: bson.D{{Key: fieldName, Value: 1}},
			}
			coll.Indexes().CreateOne(ctx, indexModel)
		}
	}

	for _, indexFields := range model.Indexes {
		keys := bson.D{}
		for _, f := range indexFields {
			keys = append(keys, bson.E{Key: f, Value: 1})
		}
		indexModel := mongo.IndexModel{Keys: keys}
		coll.Indexes().CreateOne(ctx, indexModel)
	}

	if model.PrimaryKey != nil && model.PrimaryKey.Strategy == parser.PKComposite {
		keys := bson.D{}
		for _, f := range model.PrimaryKey.Fields {
			keys = append(keys, bson.E{Key: f, Value: 1})
		}
		indexModel := mongo.IndexModel{
			Keys:    keys,
			Options: options.Index().SetUnique(true),
		}
		coll.Indexes().CreateOne(ctx, indexModel)
	}

	return nil
}

func (e *MongoMigrationEngine) MigrateSystemCollection(name string, columns []SystemColumn) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_ = e.conn.Database.CreateCollection(ctx, name)
	return nil
}

func MigrateMongoSystemTables(conn *MongoConnection) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	systemCollections := []struct {
		name    string
		indexes []mongo.IndexModel
	}{
		{
			name: "audit_logs",
			indexes: []mongo.IndexModel{
				{Keys: bson.D{{Key: "user_id", Value: 1}}},
				{Keys: bson.D{{Key: "action", Value: 1}}},
				{Keys: bson.D{{Key: "model_name", Value: 1}, {Key: "record_id", Value: 1}}},
				{Keys: bson.D{{Key: "created_at", Value: -1}}},
			},
		},
		{
			name: "view_revisions",
			indexes: []mongo.IndexModel{
				{Keys: bson.D{{Key: "view_key", Value: 1}, {Key: "version", Value: -1}}},
			},
		},
		{
			name: "data_revisions",
			indexes: []mongo.IndexModel{
				{Keys: bson.D{{Key: "model_name", Value: 1}, {Key: "record_id", Value: 1}, {Key: "version", Value: -1}}},
			},
		},
		{
			name: "sequences",
			indexes: []mongo.IndexModel{
				{
					Keys:    bson.D{{Key: "model_name", Value: 1}, {Key: "field_name", Value: 1}, {Key: "sequence_key", Value: 1}},
					Options: options.Index().SetUnique(true),
				},
			},
		},
		{
			name: "attachments",
			indexes: []mongo.IndexModel{
				{Keys: bson.D{{Key: "model_name", Value: 1}, {Key: "record_id", Value: 1}}},
				{Keys: bson.D{{Key: "hash", Value: 1}}},
			},
		},
	}

	for _, sc := range systemCollections {
		_ = conn.Database.CreateCollection(ctx, sc.name)
		coll := conn.Collection(sc.name)
		for _, idx := range sc.indexes {
			if _, err := coll.Indexes().CreateOne(ctx, idx); err != nil {
				fmt.Printf("[MONGO] warning: failed to create index on %s: %v\n", sc.name, err)
			}
		}
	}

	return nil
}
