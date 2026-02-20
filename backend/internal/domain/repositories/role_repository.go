package repositories

import (
	"context"

	"github.com/sohamify/rms-backend/internal/domain/entities"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type RoleRepository interface {
	FindByID(ctx context.Context, id primitive.ObjectID) (*entities.Role, error)
	FindByName(ctx context.Context, name string) (*entities.Role, error)
	Create(ctx context.Context, role *entities.Role) error
	Update(ctx context.Context, id primitive.ObjectID, update bson.M) error
	Delete(ctx context.Context, id primitive.ObjectID) error
	ListAll(ctx context.Context, filter bson.M, opts *options.FindOptions) ([]*entities.Role, int64, error)
}
