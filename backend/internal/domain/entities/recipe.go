package entities

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type RecipeIngredient struct {
	InventoryItemID primitive.ObjectID `bson:"inventory_item_id" json:"inventory_item_id"`
	QuantityPerUnit float64            `bson:"quantity_per_unit" json:"quantity_per_unit"`
}

type Recipe struct {
	MenuItemID  primitive.ObjectID `bson:"menu_item_id" json:"menu_item_id"`
	Ingredients []RecipeIngredient `bson:"ingredients" json:"ingredients"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
}

func (r *Recipe) ToDTO() *RecipeDTO {
	ingDTO := make([]RecipeIngredientDTO, len(r.Ingredients))
	for i, ing := range r.Ingredients {
		ingDTO[i] = RecipeIngredientDTO{
			InventoryItemID: ing.InventoryItemID.Hex(),
			QuantityPerUnit: ing.QuantityPerUnit,
		}
	}

	return &RecipeDTO{
		MenuItemID:  r.MenuItemID.Hex(),
		Ingredients: ingDTO,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}

type RecipeDTO struct {
	MenuItemID  string                `json:"menu_item_id"`
	Ingredients []RecipeIngredientDTO `json:"ingredients"`
	CreatedAt   time.Time             `json:"created_at"`
	UpdatedAt   time.Time             `json:"updated_at"`
}

type RecipeIngredientDTO struct {
	InventoryItemID string  `json:"inventory_item_id"`
	QuantityPerUnit float64 `json:"quantity_per_unit"`
}

type CreateRecipeRequest struct {
	MenuItemID  string `json:"menu_item_id" binding:"required"`
	Ingredients []struct {
		InventoryItemID string  `json:"inventory_item_id" binding:"required"`
		QuantityPerUnit float64 `json:"quantity_per_unit" binding:"required,min=0.001"`
	} `json:"ingredients" binding:"required,min=1"`
}

type UpdateRecipeRequest struct {
	Ingredients *[]struct {
		InventoryItemID string  `json:"inventory_item_id" binding:"required"`
		QuantityPerUnit float64 `json:"quantity_per_unit" binding:"required,min=0.001"`
	} `json:"ingredients,omitempty"` // full replace
}
