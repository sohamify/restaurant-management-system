package services

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"

	"github.com/sohamify/rms-backend/internal/domain/entities"
	"github.com/sohamify/rms-backend/internal/domain/repositories"
)

var (
	ErrBillAlreadyExists = errors.New("bill already generated for this order")
	ErrBillNotFound      = errors.New("bill not found")
	ErrInvalidBillAmount = errors.New("invalid amount for bill")
	ErrOrderNotCompleted = errors.New("order must be completed to generate bill")
)

type BillService interface {
	GenerateBill(ctx context.Context, req *entities.GenerateBillRequest) (*entities.BillDTO, error)
	GetBillForOrder(ctx context.Context, orderID string) (*entities.BillDTO, error)
	UpdateBillPayment(ctx context.Context, billID string, paidAmount float64) error // called from payment service
}

type billService struct {
	billRepo  repositories.BillRepository
	orderRepo repositories.OrderRepository
	logger    *zap.Logger
}

func NewBillService(billRepo repositories.BillRepository, orderRepo repositories.OrderRepository, logger *zap.Logger) BillService {
	return &billService{billRepo, orderRepo, logger}
}

func (s *billService) GenerateBill(ctx context.Context, req *entities.GenerateBillRequest) (*entities.BillDTO, error) {
	orderID, err := primitive.ObjectIDFromHex(req.OrderID)
	if err != nil {
		return nil, ErrOrderNotFound
	}

	order, err := s.orderRepo.FindByID(ctx, orderID)
	if err != nil || order == nil {
		return nil, ErrOrderNotFound
	}

	if order.Status != entities.OrderCompleted {
		return nil, ErrOrderNotCompleted
	}

	// Check if bill already exists
	existing, err := s.billRepo.FindByOrderID(ctx, orderID)
	if err != nil {
		s.logger.Error("Failed to check existing bill", zap.Error(err))
		return nil, err
	}
	if existing != nil {
		return nil, ErrBillAlreadyExists
	}

	// Build bill items from order
	items := make([]entities.BillItem, len(order.Items))
	subtotal := 0.0
	for i, ordItem := range order.Items {
		items[i] = entities.BillItem{
			MenuItemID:   ordItem.MenuItemID,
			Name:         "Menu Item", // In real code, fetch name from menu item snapshot
			Quantity:     ordItem.Quantity,
			PricePerUnit: ordItem.PriceAtTime,
			Total:        ordItem.PriceAtTime * float64(ordItem.Quantity),
		}
		subtotal += items[i].Total
	}

	taxAmount := subtotal * (req.TaxRate / 100)
	grandTotal := subtotal + taxAmount + req.ServiceCharge - req.Discount

	bill := &entities.Bill{
		OrderID:        orderID,
		TableID:        order.TableID,
		WaiterID:       order.WaiterID,
		Items:          items,
		Subtotal:       subtotal,
		TaxRate:        req.TaxRate,
		TaxAmount:      taxAmount,
		ServiceCharge:  req.ServiceCharge,
		DiscountAmount: req.Discount,
		GrandTotal:     grandTotal,
		Status:         entities.BillGenerated,
		GeneratedAt:    time.Now(),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	createdBill, err := s.billRepo.Create(ctx, bill)
	if err != nil {
		s.logger.Error("Failed to generate bill", zap.Error(err))
		return nil, err
	}

	s.logger.Info("Bill generated", zap.String("bill_id", createdBill.ID.Hex()), zap.String("order_id", req.OrderID))
	return createdBill.ToDTO(), nil
}

func (s *billService) GetBillForOrder(ctx context.Context, orderID string) (*entities.BillDTO, error) {
	oid, err := primitive.ObjectIDFromHex(orderID)
	if err != nil {
		return nil, ErrOrderNotFound
	}

	bill, err := s.billRepo.FindByOrderID(ctx, oid)
	if err != nil {
		s.logger.Error("Failed to get bill", zap.Error(err))
		return nil, err
	}
	if bill == nil {
		return nil, ErrBillNotFound
	}

	return bill.ToDTO(), nil
}

func (s *billService) UpdateBillPayment(ctx context.Context, billID string, paidAmount float64) error {
	bid, err := primitive.ObjectIDFromHex(billID)
	if err != nil {
		return ErrBillNotFound
	}

	bill, err := s.billRepo.FindByID(ctx, bid)
	if err != nil || bill == nil {
		return ErrBillNotFound
	}

	// Simple logic: if paidAmount >= grand total → PAID
	newStatus := entities.BillPartial
	if paidAmount >= bill.GrandTotal {
		newStatus = entities.BillPaid
	}

	update := bson.M{
		"status":     newStatus,
		"paid_at":    time.Now(),
		"updated_at": time.Now(),
	}

	if err := s.billRepo.Update(ctx, bid, update); err != nil {
		s.logger.Error("Failed to update bill payment status", zap.Error(err))
		return err
	}

	return nil
}
