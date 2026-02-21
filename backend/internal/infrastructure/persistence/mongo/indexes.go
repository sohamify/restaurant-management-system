package mongo

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

func CreateIndexes(logger *zap.Logger) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := GetDatabase()

	// ── Users collection indexes ─────────────────────────────────────────────
	usersColl := db.Collection("users")
	_, err := usersColl.Indexes().CreateMany(ctx, []mongo.IndexModel{
		// Unique email (prevents duplicate users)
		{
			Keys:    bson.D{{Key: "email", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		// Index on role_id for fast role-based filtering
		{
			Keys: bson.D{{Key: "role_id", Value: 1}},
		},
		// Index on is_active for quick active/inactive queries
		{
			Keys: bson.D{{Key: "is_active", Value: 1}},
		},
	})
	if err != nil {
		logger.Error("Failed to create users indexes", zap.Error(err))
		return err
	}

	// ── Roles collection indexes ─────────────────────────────────────────────
	rolesColl := db.Collection("roles")
	_, err = rolesColl.Indexes().CreateOne(ctx, mongo.IndexModel{
		// Unique role name (prevents duplicate role names)
		Keys:    bson.D{{Key: "name", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		logger.Error("Failed to create roles name unique index", zap.Error(err))
		return err
	}

	// ── Tables collection indexes ────────────────────────────────────────────
	tablesColl := db.Collection("tables")
	_, err = tablesColl.Indexes().CreateMany(ctx, []mongo.IndexModel{
		// Unique table number (critical business rule)
		{
			Keys:    bson.D{{Key: "table_number", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		// Index on status for fast filtering (e.g. all AVAILABLE tables)
		{
			Keys: bson.D{{Key: "status", Value: 1}},
		},
		// Index on assigned_waiter_id for waiter-specific queries
		{
			Keys: bson.D{{Key: "assigned_waiter_id", Value: 1}},
		},
	})
	if err != nil {
		logger.Error("Failed to create tables indexes", zap.Error(err))
		return err
	}
	// Inventory
	inventoryColl := db.Collection("inventory")
	_, err = inventoryColl.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "name", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		logger.Error("Failed to create inventory name unique index", zap.Error(err))
	}

	_, err = inventoryColl.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "quantity", Value: 1}},
	})
	if err != nil {
		logger.Error("Failed to create inventory quantity index", zap.Error(err))
	}

	// Recipes (unique on menu_item_id)
	recipesColl := db.Collection("recipes")
	_, err = recipesColl.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "menu_item_id", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		logger.Error("Failed to create recipes menu_item_id unique index", zap.Error(err))
	}

	logger.Info("All MongoDB indexes created successfully",
		zap.Strings("collections_indexed", []string{"users", "roles", "tables"}))
	return nil

}
