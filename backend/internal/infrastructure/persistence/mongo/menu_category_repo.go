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

type menuCategoryRepo struct {
	coll *mongo.Collection
}

func NewMenuCategoryRepository() repositories.MenuCategoryRepository {
	return &menuCategoryRepo{coll: GetDatabase().Collection("menu_categories")}
}

func (r *menuCategoryRepo) Create(ctx context.Context, category *entities.MenuCategory) (*entities.MenuCategory, error) {
	category.CreatedAt = time.Now()
	category.UpdatedAt = time.Now()

	res, err := r.coll.InsertOne(ctx, category)
	if err != nil {
		return nil, err
	}

	category.ID = res.InsertedID.(primitive.ObjectID)
	return category, nil
}

func (r *menuCategoryRepo) FindByID(ctx context.Context, id primitive.ObjectID) (*entities.MenuCategory, error) {
	var category entities.MenuCategory
	err := r.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&category)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &category, err
}

func (r *menuCategoryRepo) FindByName(ctx context.Context, name string) (*entities.MenuCategory, error) {
	var category entities.MenuCategory
	err := r.coll.FindOne(ctx, bson.M{"name": name}).Decode(&category)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &category, err
}

func (r *menuCategoryRepo) FindAll(ctx context.Context, filter bson.M, opts *options.FindOptions) ([]*entities.MenuCategory, int64, error) {
	cursor, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var categories []*entities.MenuCategory
	if err = cursor.All(ctx, &categories); err != nil {
		return nil, 0, err
	}

	count, err := r.coll.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	return categories, count, nil
}

func (r *menuCategoryRepo) Update(ctx context.Context, id primitive.ObjectID, fields bson.M) error {
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

func (r *menuCategoryRepo) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.coll.DeleteOne(ctx, bson.M{"_id": id})
	return err
}
