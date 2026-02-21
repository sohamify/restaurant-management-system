package entities

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MenuItem struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	CategoryID   primitive.ObjectID `bson:"category_id" json:"category_id"`
	Name         string             `bson:"name" json:"name"`
	Description  string             `bson:"description" json:"description,omitempty"`
	Price        float64            `bson:"price" json:"price"`
	IsAvailable  bool               `bson:"is_available" json:"is_available"`
	InventoryQty int                `bson:"inventory_qty" json:"inventory_qty"`
	CreatedAt    time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt    time.Time          `bson:"updated_at" json:"updated_at"`
}

func (m *MenuItem) ToDTO() *MenuItemDTO {
	return &MenuItemDTO{
		ID:           m.ID.Hex(),
		CategoryID:   m.CategoryID.Hex(),
		Name:         m.Name,
		Description:  m.Description,
		Price:        m.Price,
		IsAvailable:  m.IsAvailable,
		InventoryQty: m.InventoryQty,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
}

type MenuItemDTO struct {
	ID           string    `json:"id"`
	CategoryID   string    `json:"category_id"`
	Name         string    `json:"name"`
	Description  string    `json:"description,omitempty"`
	Price        float64   `json:"price"`
	IsAvailable  bool      `json:"is_available"`
	InventoryQty int       `json:"inventory_qty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type CreateMenuItemRequest struct {
	CategoryID   string  `json:"category_id" binding:"required"`
	Name         string  `json:"name" binding:"required,min=3,max=100"`
	Description  string  `json:"description" binding:"omitempty,max=500"`
	Price        float64 `json:"price" binding:"required,min=0"`
	InventoryQty int     `json:"inventory_qty" binding:"required,min=0"`
}

type UpdateMenuItemRequest struct {
	Name         *string  `json:"name,omitempty"`
	Description  *string  `json:"description,omitempty"`
	Price        *float64 `json:"price,omitempty"`
	IsAvailable  *bool    `json:"is_available,omitempty"`
	InventoryQty *int     `json:"inventory_qty,omitempty"`
}
