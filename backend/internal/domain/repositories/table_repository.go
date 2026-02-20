package repositories

import (
	"context"

	"github.com/sohamify/rms-backend/internal/domain/entities"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type TableRepository interface {
	Create(ctx context.Context, table *entities.Table) (*entities.Table, error)
	FindByID(ctx context.Context, id primitive.ObjectID) (*entities.Table, error)
	FindByTableNumber(ctx context.Context, number int) (*entities.Table, error)
	FindAll(ctx context.Context, filter bson.M, opts *options.FindOptions) ([]*entities.Table, int64, error)
	FindAssignedToWaiter(ctx context.Context, waiterID primitive.ObjectID) ([]*entities.Table, error)
	Update(ctx context.Context, id primitive.ObjectID, fields bson.M) error
	Delete(ctx context.Context, id primitive.ObjectID) error
}
