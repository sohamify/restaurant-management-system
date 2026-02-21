package mongo

import (
	"context"
	"time"

	"github.com/sohamify/rms-backend/internal/domain/entities"
	"github.com/sohamify/rms-backend/internal/domain/repositories"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type orderRepo struct {
	coll *mongo.Collection
}

func NewOrderRepository() repositories.OrderRepository {
	return &orderRepo{coll: GetDatabase().Collection("orders")}
}

func (r *orderRepo) Create(ctx context.Context, order *entities.Order) (*entities.Order, error) {
	order.CreatedAt = time.Now()
	order.UpdatedAt = time.Now()
	order.Status = entities.OrderPlaced

	res, err := r.coll.InsertOne(ctx, order)
	if err != nil {
		return nil, err
	}

	order.ID = res.InsertedID.(primitive.ObjectID)
	return order, nil
}

func (r *orderRepo) FindByID(ctx context.Context, id primitive.ObjectID) (*entities.Order, error) {
	var order entities.Order
	err := r.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&order)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &order, err
}

func (r *orderRepo) FindByTableID(ctx context.Context, tableID primitive.ObjectID) (*entities.Order, error) {
	var order entities.Order
	filter := bson.M{
		"table_id": tableID,
		"status":   bson.M{"$nin": []string{string(entities.OrderPaid), string(entities.OrderCancelled)}},
	}
	err := r.coll.FindOne(ctx, filter).Decode(&order)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &order, err
}

func (r *orderRepo) FindAll(ctx context.Context, filter bson.M, opts *options.FindOptions) ([]*entities.Order, int64, error) {
	cursor, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var orders []*entities.Order
	if err = cursor.All(ctx, &orders); err != nil {
		return nil, 0, err
	}

	count, err := r.coll.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	return orders, count, nil
}

func (r *orderRepo) FindActiveForKitchen(ctx context.Context) ([]*entities.Order, error) {
	filter := bson.M{
		"status": bson.M{
			"$in": []string{
				string(entities.OrderPlaced),
				string(entities.OrderPreparing),
			},
		},
	}
	cursor, err := r.coll.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var orders []*entities.Order
	if err = cursor.All(ctx, &orders); err != nil {
		return nil, err
	}

	return orders, nil
}

func (r *orderRepo) Update(ctx context.Context, id primitive.ObjectID, fields bson.M) error {
	setFields := bson.M{
		"updated_at": time.Now(),
	}

	for k, v := range fields {
		setFields[k] = v
	}

	update := bson.M{"$set": setFields}

	_, err := r.coll.UpdateOne(ctx, bson.M{"_id": id}, update)
	return err
}

func (r *orderRepo) AddItem(ctx context.Context, id primitive.ObjectID, item entities.OrderItem) error {
	update := bson.M{
		"$push": bson.M{"items": item},
		"$set":  bson.M{"updated_at": time.Now()},
	}

	_, err := r.coll.UpdateOne(ctx, bson.M{"_id": id}, update)
	return err
}

func (r *orderRepo) RemoveItem(ctx context.Context, id primitive.ObjectID, menuItemID primitive.ObjectID) error {
	update := bson.M{
		"$pull": bson.M{"items": bson.M{"menu_item_id": menuItemID}},
		"$set":  bson.M{"updated_at": time.Now()},
	}

	_, err := r.coll.UpdateOne(ctx, bson.M{"_id": id}, update)
	return err
}

func (r *orderRepo) UpdateStatus(ctx context.Context, id primitive.ObjectID, status entities.OrderStatus) error {
	update := bson.M{
		"$set": bson.M{
			"status":     status,
			"updated_at": time.Now(),
		},
	}

	if status == entities.OrderCompleted {
		update["$set"].(bson.M)["completed_at"] = time.Now()
	}

	_, err := r.coll.UpdateOne(ctx, bson.M{"_id": id}, update)
	return err
}

func (r *orderRepo) Cancel(ctx context.Context, id primitive.ObjectID, reason string) error {
	update := bson.M{
		"$set": bson.M{
			"status":     entities.OrderCancelled,
			"notes":      reason,
			"updated_at": time.Now(),
		},
	}

	_, err := r.coll.UpdateOne(ctx, bson.M{"_id": id}, update)
	return err
}

func (r *orderRepo) MarkPaid(ctx context.Context, id primitive.ObjectID, paymentMethod string) error {
	update := bson.M{
		"$set": bson.M{
			"status":         entities.OrderPaid,
			"payment_method": paymentMethod,
			"updated_at":     time.Now(),
		},
	}

	_, err := r.coll.UpdateOne(ctx, bson.M{"_id": id}, update)
	return err
}
