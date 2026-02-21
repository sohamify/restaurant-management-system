package entities

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type OrderStatus string

const (
	OrderPlaced    OrderStatus = "PLACED"
	OrderPreparing OrderStatus = "PREPARING"
	OrderReady     OrderStatus = "READY"
	OrderCompleted OrderStatus = "COMPLETED"
	OrderPaid      OrderStatus = "PAID"
	OrderCancelled OrderStatus = "CANCELLED"
)

type OrderItem struct {
	MenuItemID  primitive.ObjectID `bson:"menu_item_id" json:"menu_item_id"`
	Quantity    int                `bson:"quantity" json:"quantity"`
	PriceAtTime float64            `bson:"price_at_time" json:"price_at_time"`
	Notes       string             `bson:"notes" json:"notes,omitempty"`
	Status      string             `bson:"status" json:"status"` // PENDING, PREPARING, READY, SERVED
}

type Order struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	TableID     primitive.ObjectID `bson:"table_id" json:"table_id"`
	WaiterID    primitive.ObjectID `bson:"waiter_id" json:"waiter_id"`
	Items       []OrderItem        `bson:"items" json:"items"`
	TotalAmount float64            `bson:"total_amount" json:"total_amount"`
	Status      OrderStatus        `bson:"status" json:"status"`
	Notes       string             `bson:"notes" json:"notes,omitempty"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
	CompletedAt *time.Time         `bson:"completed_at" json:"completed_at,omitempty"`
}

func (o *Order) ToDTO() *OrderDTO {
	itemsDTO := make([]OrderItemDTO, len(o.Items))
	for i, item := range o.Items {
		itemsDTO[i] = OrderItemDTO{
			MenuItemID:  item.MenuItemID.Hex(),
			Quantity:    item.Quantity,
			PriceAtTime: item.PriceAtTime,
			Notes:       item.Notes,
			Status:      item.Status,
		}
	}

	dto := &OrderDTO{
		ID:          o.ID.Hex(),
		TableID:     o.TableID.Hex(),
		WaiterID:    o.WaiterID.Hex(),
		Items:       itemsDTO,
		TotalAmount: o.TotalAmount,
		Status:      string(o.Status),
		Notes:       o.Notes,
		CreatedAt:   o.CreatedAt,
		UpdatedAt:   o.UpdatedAt,
	}

	if o.CompletedAt != nil {
		dto.CompletedAt = *o.CompletedAt
	}

	return dto
}

type OrderDTO struct {
	ID          string         `json:"id"`
	TableID     string         `json:"table_id"`
	WaiterID    string         `json:"waiter_id"`
	Items       []OrderItemDTO `json:"items"`
	TotalAmount float64        `json:"total_amount"`
	Status      string         `json:"status"`
	Notes       string         `json:"notes,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	CompletedAt time.Time      `json:"completed_at,omitempty"`
}

type OrderItemDTO struct {
	MenuItemID  string  `json:"menu_item_id"`
	Quantity    int     `json:"quantity"`
	PriceAtTime float64 `json:"price_at_time"`
	Notes       string  `json:"notes,omitempty"`
	Status      string  `json:"status"`
}

type CreateOrderRequest struct {
	TableID string `json:"table_id" binding:"required"`
	Items   []struct {
		MenuItemID string `json:"menu_item_id" binding:"required"`
		Quantity   int    `json:"quantity" binding:"required,min=1"`
		Notes      string `json:"notes,omitempty"`
	} `json:"items" binding:"required,min=1"`
	Notes string `json:"notes,omitempty"`
}

type AddItemToOrderRequest struct {
	MenuItemID string `json:"menu_item_id" binding:"required"`
	Quantity   int    `json:"quantity" binding:"required,min=1"`
	Notes      string `json:"notes,omitempty"`
}

type UpdateOrderStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

type CancelOrderRequest struct {
	Reason string `json:"reason" binding:"required,min=5"`
}
