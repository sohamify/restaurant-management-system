package repositories

import (
	"context"

	"github.com/sohamify/rms-backend/internal/domain/entities"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type InventoryRepository interface {
	Create(ctx context.Context, item *entities.InventoryItem) (*entities.InventoryItem, error)
	FindByID(ctx context.Context, id primitive.ObjectID) (*entities.InventoryItem, error)
	FindAll(ctx context.Context, filter bson.M, opts *options.FindOptions) ([]*entities.InventoryItem, int64, error)
	Update(ctx context.Context, id primitive.ObjectID, fields bson.M) error
	Delete(ctx context.Context, id primitive.ObjectID) error
	UpdateQuantity(ctx context.Context, id primitive.ObjectID, delta float64) error // for deduct/add
	GetLowStock(ctx context.Context) ([]*entities.InventoryItem, error)
}
