package repositories

import (
	"context"

	"github.com/sohamify/rms-backend/internal/domain/entities"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type BillRepository interface {
	Create(ctx context.Context, bill *entities.Bill) (*entities.Bill, error)
	FindByID(ctx context.Context, id primitive.ObjectID) (*entities.Bill, error)
	FindByOrderID(ctx context.Context, orderID primitive.ObjectID) (*entities.Bill, error)
	Update(ctx context.Context, id primitive.ObjectID, fields bson.M) error
}
