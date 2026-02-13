package repositories

import (
	"context"

	"github.com/sohamify/rms-backend/internal/domain/entities"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type UserRepository interface {
	FindByEmail(ctx context.Context, email string) (*entities.User, error)
	FindByID(ctx context.Context, id primitive.ObjectID) (*entities.User, error)
	FindAll(ctx context.Context, filter bson.M, opts *options.FindOptions) ([]*entities.User, int64, error)
	Create(ctx context.Context, user *entities.User) error
	Update(ctx context.Context, id primitive.ObjectID, update bson.M) error
	SoftDelete(ctx context.Context, id primitive.ObjectID) error
	// Optional: Count(ctx context.Context, filter bson.M) (int64, error)
}
