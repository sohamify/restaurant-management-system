package mongo

import (
	"context"
	"time"

	"github.com/sohamify/rms-backend/internal/domain/entities"
	"github.com/sohamify/rms-backend/internal/domain/repositories"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type recipeRepo struct {
	coll *mongo.Collection
}

func NewRecipeRepository() repositories.RecipeRepository {
	return &recipeRepo{coll: GetDatabase().Collection("recipes")}
}

func (r *recipeRepo) Create(ctx context.Context, recipe *entities.Recipe) (*entities.Recipe, error) {
	recipe.CreatedAt = time.Now()
	recipe.UpdatedAt = time.Now()

	_, err := r.coll.InsertOne(ctx, recipe)
	if err != nil {
		return nil, err
	}

	return recipe, nil // Recipes don't have own ID - keyed by menu_item_id
}

func (r *recipeRepo) FindByMenuItemID(ctx context.Context, menuItemID primitive.ObjectID) (*entities.Recipe, error) {
	var recipe entities.Recipe
	err := r.coll.FindOne(ctx, bson.M{"menu_item_id": menuItemID}).Decode(&recipe)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &recipe, err
}

func (r *recipeRepo) Update(ctx context.Context, menuItemID primitive.ObjectID, fields bson.M) error {
	setFields := bson.M{
		"updated_at": time.Now(),
	}

	for k, v := range fields {
		setFields[k] = v
	}

	update := bson.M{"$set": setFields}

	_, err := r.coll.UpdateOne(ctx, bson.M{"menu_item_id": menuItemID}, update)
	return err
}

func (r *recipeRepo) Delete(ctx context.Context, menuItemID primitive.ObjectID) error {
	_, err := r.coll.DeleteOne(ctx, bson.M{"menu_item_id": menuItemID})
	return err
}
