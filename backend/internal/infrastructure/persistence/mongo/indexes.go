package mongo

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

func CreateIndexes(logger *zap.Logger) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	usersColl := GetDatabase().Collection("users")
	_, err := usersColl.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: map[string]int{"email": 1}, Options: options.Index().SetUnique(true)},
		{Keys: map[string]int{"role_id": 1}},
		{Keys: map[string]int{"is_active": 1}},
	})
	if err != nil {
		logger.Error("Failed to create users indexes", zap.Error(err))
		return err
	}

	rolesColl := GetDatabase().Collection("roles")
	_, err = rolesColl.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: map[string]int{"name": 1}, Options: options.Index().SetUnique(true),
	})
	if err != nil {
		logger.Error("Failed to create roles index", zap.Error(err))
		return err
	}

	logger.Info("MongoDB indexes created successfully")
	return nil
}
