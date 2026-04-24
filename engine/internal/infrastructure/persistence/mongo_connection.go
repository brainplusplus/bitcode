package persistence

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type MongoConnection struct {
	Client   *mongo.Client
	Database *mongo.Database
}

func OpenMongoDB(cfg DatabaseConfig) (*MongoConnection, error) {
	host := cfg.Host
	if host == "" {
		host = "localhost"
	}
	port := cfg.Port
	if port == 0 {
		port = 27017
	}

	var uri string
	if cfg.User != "" && cfg.Password != "" {
		uri = fmt.Sprintf("mongodb://%s:%s@%s:%d", cfg.User, cfg.Password, host, port)
	} else {
		uri = fmt.Sprintf("mongodb://%s:%d", host, port)
	}

	if cfg.SSLMode != "" && cfg.SSLMode != "disable" {
		uri += "/?tls=true"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOpts := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(clientOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	dbName := cfg.DBName
	if dbName == "" {
		dbName = "bitcode"
	}

	return &MongoConnection{
		Client:   client,
		Database: client.Database(dbName),
	}, nil
}

func (mc *MongoConnection) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return mc.Client.Disconnect(ctx)
}

func (mc *MongoConnection) Collection(name string) *mongo.Collection {
	return mc.Database.Collection(name)
}
