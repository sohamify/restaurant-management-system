package entities

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type TableStatus string

const (
	TableAvailable TableStatus = "AVAILABLE"
	TableOccupied  TableStatus = "OCCUPIED"
	TableReserved  TableStatus = "RESERVED"
	TableCleaning  TableStatus = "CLEANING"
)

type Table struct {
	ID               primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	TableNumber      int                `bson:"table_number" json:"table_number"` // unique
	Capacity         int                `bson:"capacity" json:"capacity"`
	Status           TableStatus        `bson:"status" json:"status"`
	AssignedWaiterID primitive.ObjectID `bson:"assigned_waiter_id,omitempty" json:"assigned_waiter_id,omitempty"`
	CurrentOrderID   primitive.ObjectID `bson:"current_order_id,omitempty" json:"current_order_id,omitempty"`
	CreatedAt        time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt        time.Time          `bson:"updated_at" json:"updated_at"`
}

func (t *Table) ToDTO() *TableDTO {
	return &TableDTO{
		ID:               t.ID.Hex(),
		TableNumber:      t.TableNumber,
		Capacity:         t.Capacity,
		Status:           string(t.Status),
		AssignedWaiterID: t.AssignedWaiterID.Hex(),
		CurrentOrderID:   t.CurrentOrderID.Hex(),
		CreatedAt:        t.CreatedAt,
		UpdatedAt:        t.UpdatedAt,
	}
}

type TableDTO struct {
	ID               string    `json:"id"`
	TableNumber      int       `json:"table_number"`
	Capacity         int       `json:"capacity"`
	Status           string    `json:"status"`
	AssignedWaiterID string    `json:"assigned_waiter_id,omitempty"`
	CurrentOrderID   string    `json:"current_order_id,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type CreateTableRequest struct {
	TableNumber int `json:"table_number" binding:"required,min=1"`
	Capacity    int `json:"capacity" binding:"required,min=1"`
}

type UpdateTableRequest struct {
	Capacity         *int    `json:"capacity,omitempty"`
	Status           *string `json:"status,omitempty"`
	AssignedWaiterID *string `json:"assigned_waiter_id,omitempty"`
}
