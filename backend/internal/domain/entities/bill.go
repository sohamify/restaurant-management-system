package entities

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type BillStatus string

const (
	BillGenerated BillStatus = "GENERATED"
	BillPaid      BillStatus = "PAID"
	BillPartial   BillStatus = "PARTIAL_PAID"
	BillRefunded  BillStatus = "REFUNDED"
)

type BillItem struct {
	MenuItemID   primitive.ObjectID `bson:"menu_item_id" json:"menu_item_id"`
	Name         string             `bson:"name" json:"name"` // snapshot at creation
	Quantity     int                `bson:"quantity" json:"quantity"`
	PricePerUnit float64            `bson:"price_per_unit" json:"price_per_unit"`
	Total        float64            `bson:"total" json:"total"`
}

type Bill struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	OrderID        primitive.ObjectID `bson:"order_id" json:"order_id"`
	TableID        primitive.ObjectID `bson:"table_id" json:"table_id"`
	WaiterID       primitive.ObjectID `bson:"waiter_id" json:"waiter_id"`
	Items          []BillItem         `bson:"items" json:"items"`
	Subtotal       float64            `bson:"subtotal" json:"subtotal"`
	TaxRate        float64            `bson:"tax_rate" json:"tax_rate"` // percentage
	TaxAmount      float64            `bson:"tax_amount" json:"tax_amount"`
	ServiceCharge  float64            `bson:"service_charge" json:"service_charge,omitempty"`
	DiscountAmount float64            `bson:"discount_amount" json:"discount_amount,omitempty"`
	GrandTotal     float64            `bson:"grand_total" json:"grand_total"`
	Status         BillStatus         `bson:"status" json:"status"`
	GeneratedAt    time.Time          `bson:"generated_at" json:"generated_at"`
	PaidAt         *time.Time         `bson:"paid_at,omitempty" json:"paid_at,omitempty"`
	CreatedAt      time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt      time.Time          `bson:"updated_at" json:"updated_at"`
}

func (b *Bill) ToDTO() *BillDTO {
	items := make([]BillItemDTO, len(b.Items))
	for i, it := range b.Items {
		items[i] = BillItemDTO{
			MenuItemID:   it.MenuItemID.Hex(),
			Name:         it.Name,
			Quantity:     it.Quantity,
			PricePerUnit: it.PricePerUnit,
			Total:        it.Total,
		}
	}

	dto := &BillDTO{
		ID:             b.ID.Hex(),
		OrderID:        b.OrderID.Hex(),
		TableID:        b.TableID.Hex(),
		WaiterID:       b.WaiterID.Hex(),
		Items:          items,
		Subtotal:       b.Subtotal,
		TaxRate:        b.TaxRate,
		TaxAmount:      b.TaxAmount,
		ServiceCharge:  b.ServiceCharge,
		DiscountAmount: b.DiscountAmount,
		GrandTotal:     b.GrandTotal,
		Status:         string(b.Status),
		GeneratedAt:    b.GeneratedAt,
		CreatedAt:      b.CreatedAt,
		UpdatedAt:      b.UpdatedAt,
	}

	if b.PaidAt != nil {
		dto.PaidAt = *b.PaidAt
	}

	return dto
}

type BillDTO struct {
	ID             string        `json:"id"`
	OrderID        string        `json:"order_id"`
	TableID        string        `json:"table_id"`
	WaiterID       string        `json:"waiter_id"`
	Items          []BillItemDTO `json:"items"`
	Subtotal       float64       `json:"subtotal"`
	TaxRate        float64       `json:"tax_rate"`
	TaxAmount      float64       `json:"tax_amount"`
	ServiceCharge  float64       `json:"service_charge,omitempty"`
	DiscountAmount float64       `json:"discount_amount,omitempty"`
	GrandTotal     float64       `json:"grand_total"`
	Status         string        `json:"status"`
	GeneratedAt    time.Time     `json:"generated_at"`
	PaidAt         time.Time     `json:"paid_at,omitempty"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
}

type BillItemDTO struct {
	MenuItemID   string  `json:"menu_item_id"`
	Name         string  `json:"name"`
	Quantity     int     `json:"quantity"`
	PricePerUnit float64 `json:"price_per_unit"`
	Total        float64 `json:"total"`
}

type GenerateBillRequest struct {
	OrderID       string  `json:"order_id" binding:"required"`
	TaxRate       float64 `json:"tax_rate" binding:"min=0"`       // percentage
	ServiceCharge float64 `json:"service_charge" binding:"min=0"` // fixed amount
	Discount      float64 `json:"discount" binding:"min=0"`       // fixed amount
}
