package repositories

import (
	"context"

	"github.com/sohamify/rms-backend/internal/domain/entities"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type RecipeRepository interface {
	Create(ctx context.Context, recipe *entities.Recipe) (*entities.Recipe, error)
	FindByMenuItemID(ctx context.Context, menuItemID primitive.ObjectID) (*entities.Recipe, error)
	Update(ctx context.Context, menuItemID primitive.ObjectID, fields bson.M) error
	Delete(ctx context.Context, menuItemID primitive.ObjectID) error
}
