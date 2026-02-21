package repositories

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/sohamify/rms-backend/internal/domain/entities"
)

type OrderRepository interface {
	Create(ctx context.Context, order *entities.Order) (*entities.Order, error)
	FindByID(ctx context.Context, id primitive.ObjectID) (*entities.Order, error)
	FindByTableID(ctx context.Context, tableID primitive.ObjectID) (*entities.Order, error)
	FindAll(ctx context.Context, filter bson.M, opts *options.FindOptions) ([]*entities.Order, int64, error)
	FindActiveForKitchen(ctx context.Context) ([]*entities.Order, error)
	Update(ctx context.Context, id primitive.ObjectID, fields bson.M) error
	AddItem(ctx context.Context, id primitive.ObjectID, item entities.OrderItem) error
	RemoveItem(ctx context.Context, id primitive.ObjectID, menuItemID primitive.ObjectID) error
	UpdateStatus(ctx context.Context, id primitive.ObjectID, status entities.OrderStatus) error
	Cancel(ctx context.Context, id primitive.ObjectID, reason string) error
	MarkPaid(ctx context.Context, id primitive.ObjectID, paymentMethod string) error // ← added this
}
