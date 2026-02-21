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

type inventoryRepo struct {
	coll *mongo.Collection
}

func NewInventoryRepository() repositories.InventoryRepository {
	return &inventoryRepo{coll: GetDatabase().Collection("inventory")}
}

func (r *inventoryRepo) Create(ctx context.Context, item *entities.InventoryItem) (*entities.InventoryItem, error) {
	item.CreatedAt = time.Now()
	item.UpdatedAt = time.Now()

	res, err := r.coll.InsertOne(ctx, item)
	if err != nil {
		return nil, err
	}

	item.ID = res.InsertedID.(primitive.ObjectID)
	return item, nil
}

func (r *inventoryRepo) FindByID(ctx context.Context, id primitive.ObjectID) (*entities.InventoryItem, error) {
	var item entities.InventoryItem
	err := r.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&item)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &item, err
}

func (r *inventoryRepo) FindAll(ctx context.Context, filter bson.M, opts *options.FindOptions) ([]*entities.InventoryItem, int64, error) {
	cursor, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var items []*entities.InventoryItem
	if err = cursor.All(ctx, &items); err != nil {
		return nil, 0, err
	}

	count, err := r.coll.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	return items, count, nil
}

func (r *inventoryRepo) Update(ctx context.Context, id primitive.ObjectID, fields bson.M) error {
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

func (r *inventoryRepo) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.coll.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

func (r *inventoryRepo) UpdateQuantity(ctx context.Context, id primitive.ObjectID, delta float64) error {
	update := bson.M{
		"$inc": bson.M{"quantity": delta},
		"$set": bson.M{"updated_at": time.Now()},
	}

	_, err := r.coll.UpdateOne(ctx, bson.M{"_id": id}, update)
	return err
}

func (r *inventoryRepo) GetLowStock(ctx context.Context) ([]*entities.InventoryItem, error) {
	filter := bson.M{"quantity": bson.M{"$lt": bson.M{"$low_stock_threshold": 1}}}
	cursor, err := r.coll.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var items []*entities.InventoryItem
	if err = cursor.All(ctx, &items); err != nil {
		return nil, err
	}

	return items, nil
}
