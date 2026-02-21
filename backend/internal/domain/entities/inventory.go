package entities

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type InventoryItem struct {
	ID                primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Name              string             `bson:"name" json:"name"`
	Unit              string             `bson:"unit" json:"unit"` // e.g., kg, liter, piece
	Quantity          float64            `bson:"quantity" json:"quantity"`
	LowStockThreshold float64            `bson:"low_stock_threshold" json:"low_stock_threshold"`
	SupplierInfo      string             `bson:"supplier_info" json:"supplier_info,omitempty"`
	CreatedAt         time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt         time.Time          `bson:"updated_at" json:"updated_at"`
}

func (i *InventoryItem) ToDTO() InventoryItemDTO { // return value, not pointer
	return InventoryItemDTO{
		ID:                i.ID.Hex(),
		Name:              i.Name,
		Unit:              i.Unit,
		Quantity:          i.Quantity,
		LowStockThreshold: i.LowStockThreshold,
		SupplierInfo:      i.SupplierInfo,
		CreatedAt:         i.CreatedAt,
		UpdatedAt:         i.UpdatedAt,
	}
}

type InventoryItemDTO struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	Unit              string    `json:"unit"`
	Quantity          float64   `json:"quantity"`
	LowStockThreshold float64   `json:"low_stock_threshold"`
	SupplierInfo      string    `json:"supplier_info,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type CreateInventoryItemRequest struct {
	Name              string  `json:"name" binding:"required,min=3,max=100"`
	Unit              string  `json:"unit" binding:"required"`
	Quantity          float64 `json:"quantity" binding:"required,min=0"`
	LowStockThreshold float64 `json:"low_stock_threshold" binding:"required,min=0"`
	SupplierInfo      string  `json:"supplier_info" binding:"omitempty"`
}

type UpdateInventoryItemRequest struct {
	Name              *string  `json:"name,omitempty"`
	Unit              *string  `json:"unit,omitempty"`
	Quantity          *float64 `json:"quantity,omitempty"`
	LowStockThreshold *float64 `json:"low_stock_threshold,omitempty"`
	SupplierInfo      *string  `json:"supplier_info,omitempty"`
}

type InventoryReport struct {
	LowStockItems []InventoryItemDTO `json:"low_stock_items"`
	TotalItems    int                `json:"total_items"`
}
