package store

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoStore struct {
	Client      *mongo.Client
	Database    *mongo.Database
	Transactions *mongo.Collection
}

func NewMongoStore(ctx context.Context, uri, dbName string, timeout time.Duration) (*MongoStore, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("connect mongo: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("ping mongo: %w", err)
	}

	db := client.Database(dbName)
	store := &MongoStore{
		Client:       client,
		Database:     db,
		Transactions: db.Collection("transactions"),
	}

	if err := store.EnsureIndexes(ctx); err != nil {
		return nil, err
	}

	return store, nil
}

func (s *MongoStore) EnsureIndexes(ctx context.Context) error {
	models := []mongo.IndexModel{
		{Keys: bson.D{{Key: "createdAt", Value: 1}, {Key: "type", Value: 1}}},
		{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "createdAt", Value: 1}, {Key: "type", Value: 1}}},
		{Keys: bson.D{{Key: "roundId", Value: 1}, {Key: "createdAt", Value: 1}}},
		{Keys: bson.D{{Key: "currency", Value: 1}, {Key: "createdAt", Value: 1}, {Key: "type", Value: 1}}},
	}

	_, err := s.Transactions.Indexes().CreateMany(ctx, models)
	if err != nil {
		return fmt.Errorf("create indexes: %w", err)
	}

	return nil
}

func (s *MongoStore) Close(ctx context.Context) error {
	return s.Client.Disconnect(ctx)
}

