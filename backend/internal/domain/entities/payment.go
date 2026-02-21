package entities

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type PaymentStatus string

const (
	PaymentPending   PaymentStatus = "PENDING"
	PaymentCompleted PaymentStatus = "COMPLETED"
	PaymentRefunded  PaymentStatus = "REFUNDED"
	PaymentFailed    PaymentStatus = "FAILED"
)

type PaymentMethod string

const (
	MethodCash   PaymentMethod = "CASH"
	MethodUPI    PaymentMethod = "UPI"
	MethodCard   PaymentMethod = "CARD"
	MethodWallet PaymentMethod = "WALLET"
)

type Payment struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	OrderID     primitive.ObjectID `bson:"order_id" json:"order_id"`
	Amount      float64            `bson:"amount" json:"amount"` // paid amount
	Method      PaymentMethod      `bson:"method" json:"method"`
	Status      PaymentStatus      `bson:"status" json:"status"`
	PaidAt      *time.Time         `bson:"paid_at,omitempty" json:"paid_at,omitempty"`
	RefundedAt  *time.Time         `bson:"refunded_at,omitempty" json:"refunded_at,omitempty"`
	RefundNotes string             `bson:"refund_notes,omitempty" json:"refund_notes,omitempty"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
}

func (p *Payment) ToDTO() *PaymentDTO {
	dto := &PaymentDTO{
		ID:        p.ID.Hex(),
		OrderID:   p.OrderID.Hex(),
		Amount:    p.Amount,
		Method:    string(p.Method),
		Status:    string(p.Status),
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}

	if p.PaidAt != nil {
		dto.PaidAt = *p.PaidAt
	}
	if p.RefundedAt != nil {
		dto.RefundedAt = *p.RefundedAt
		dto.RefundNotes = p.RefundNotes
	}

	return dto
}

type PaymentDTO struct {
	ID          string    `json:"id"`
	OrderID     string    `json:"order_id"`
	Amount      float64   `json:"amount"`
	Method      string    `json:"method"`
	Status      string    `json:"status"`
	PaidAt      time.Time `json:"paid_at,omitempty"`
	RefundedAt  time.Time `json:"refunded_at,omitempty"`
	RefundNotes string    `json:"refund_notes,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ProcessPaymentRequest struct {
	OrderID string        `json:"order_id" binding:"required"`
	Amount  float64       `json:"amount" binding:"required,min=0.01"`
	Method  PaymentMethod `json:"method" binding:"required"`
	Notes   string        `json:"notes,omitempty"`
}

type RefundRequest struct {
	Amount float64 `json:"amount" binding:"required,min=0.01"`
	Reason string  `json:"reason" binding:"required"`
}
