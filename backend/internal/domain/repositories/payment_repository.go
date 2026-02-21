package repositories

import (
	"context"

	"github.com/sohamify/rms-backend/internal/domain/entities"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type PaymentRepository interface {
	Create(ctx context.Context, payment *entities.Payment) (*entities.Payment, error)
	FindByID(ctx context.Context, id primitive.ObjectID) (*entities.Payment, error)
	FindByOrderID(ctx context.Context, orderID primitive.ObjectID) ([]*entities.Payment, error)
	Update(ctx context.Context, id primitive.ObjectID, fields bson.M) error
	Delete(ctx context.Context, id primitive.ObjectID) error
}
