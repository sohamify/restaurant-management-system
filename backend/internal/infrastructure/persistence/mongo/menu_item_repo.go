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

type menuItemRepo struct {
	coll *mongo.Collection
}

func NewMenuItemRepository() repositories.MenuItemRepository {
	return &menuItemRepo{coll: GetDatabase().Collection("menu_items")}
}

func (r *menuItemRepo) Create(ctx context.Context, item *entities.MenuItem) (*entities.MenuItem, error) {
	item.CreatedAt = time.Now()
	item.UpdatedAt = time.Now()
	item.IsAvailable = true // default

	res, err := r.coll.InsertOne(ctx, item)
	if err != nil {
		return nil, err
	}

	item.ID = res.InsertedID.(primitive.ObjectID)
	return item, nil
}

func (r *menuItemRepo) FindByID(ctx context.Context, id primitive.ObjectID) (*entities.MenuItem, error) {
	var item entities.MenuItem
	err := r.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&item)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &item, err
}

func (r *menuItemRepo) FindAll(ctx context.Context, filter bson.M, opts *options.FindOptions) ([]*entities.MenuItem, int64, error) {
	cursor, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var items []*entities.MenuItem
	if err = cursor.All(ctx, &items); err != nil {
		return nil, 0, err
	}

	count, err := r.coll.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	return items, count, nil
}

func (r *menuItemRepo) Update(ctx context.Context, id primitive.ObjectID, fields bson.M) error {
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

func (r *menuItemRepo) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.coll.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

func (r *menuItemRepo) UpdateInventory(ctx context.Context, id primitive.ObjectID, delta int) error {
	update := bson.M{
		"$inc": bson.M{"inventory_qty": delta},
		"$set": bson.M{"updated_at": time.Now()},
	}

	_, err := r.coll.UpdateOne(ctx, bson.M{"_id": id}, update)
	return err
}
