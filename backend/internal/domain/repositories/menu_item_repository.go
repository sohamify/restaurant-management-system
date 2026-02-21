package repositories

import (
	"context"

	"github.com/sohamify/rms-backend/internal/domain/entities"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MenuItemRepository interface {
	Create(ctx context.Context, item *entities.MenuItem) (*entities.MenuItem, error)
	FindByID(ctx context.Context, id primitive.ObjectID) (*entities.MenuItem, error)
	FindAll(ctx context.Context, filter bson.M, opts *options.FindOptions) ([]*entities.MenuItem, int64, error)
	Update(ctx context.Context, id primitive.ObjectID, fields bson.M) error
	Delete(ctx context.Context, id primitive.ObjectID) error
	UpdateInventory(ctx context.Context, id primitive.ObjectID, delta int) error // for deduct/add inventory
}
