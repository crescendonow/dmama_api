package database

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// NewMongoClient connects to MongoDB and returns the client. It pings the server up front to fail
// fast on bad credentials/config. Use client.Database(name) to get individual databases (the same
// cluster hosts both the Vallaris feature store and the user directory).
func NewMongoClient(ctx context.Context, uri string) (*mongo.Client, error) {
	if uri == "" {
		return nil, fmt.Errorf("MONGO_URI is required")
	}

	connectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(connectCtx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("connecting to mongo: %w", err)
	}

	if err := client.Ping(connectCtx, nil); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, fmt.Errorf("pinging mongo: %w", err)
	}

	return client, nil
}

// NewMongoDatabase connects to MongoDB and returns a handle to the named database.
func NewMongoDatabase(ctx context.Context, uri, dbName string) (*mongo.Database, error) {
	if dbName == "" {
		return nil, fmt.Errorf("MONGO_DB is required")
	}
	client, err := NewMongoClient(ctx, uri)
	if err != nil {
		return nil, err
	}
	return client.Database(dbName), nil
}
