package repositories

import (
	"context"

	"github.com/sohamify/rms-backend/internal/domain/entities"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MenuCategoryRepository interface {
	Create(ctx context.Context, category *entities.MenuCategory) (*entities.MenuCategory, error)
	FindByID(ctx context.Context, id primitive.ObjectID) (*entities.MenuCategory, error)
	FindByName(ctx context.Context, name string) (*entities.MenuCategory, error)
	FindAll(ctx context.Context, filter bson.M, opts *options.FindOptions) ([]*entities.MenuCategory, int64, error)
	Update(ctx context.Context, id primitive.ObjectID, fields bson.M) error
	Delete(ctx context.Context, id primitive.ObjectID) error
}
