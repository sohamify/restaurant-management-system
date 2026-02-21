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
	ErrPaymentNotAllowed   = errors.New("order must be completed before payment")
	ErrRefundNotAllowed    = errors.New("refund only possible on paid orders")
	ErrInvalidRefundAmount = errors.New("refund amount exceeds paid amount")
	ErrNoPaymentsFound     = errors.New("no payments found for this order")
)

type PaymentService interface {
	ProcessPayment(ctx context.Context, req *entities.ProcessPaymentRequest) (*entities.PaymentDTO, error)
	ProcessRefund(ctx context.Context, orderID string, req *entities.RefundRequest) error
	GetPaymentsForOrder(ctx context.Context, orderID string) ([]*entities.PaymentDTO, error)
}

type paymentService struct {
	paymentRepo  repositories.PaymentRepository
	orderRepo    repositories.OrderRepository
	tableRepo    repositories.TableRepository
	menuItemRepo repositories.MenuItemRepository
	logger       *zap.Logger
}

func NewPaymentService(
	paymentRepo repositories.PaymentRepository,
	orderRepo repositories.OrderRepository,
	tableRepo repositories.TableRepository,
	menuItemRepo repositories.MenuItemRepository,
	logger *zap.Logger,
) PaymentService {
	return &paymentService{paymentRepo, orderRepo, tableRepo, menuItemRepo, logger}
}

func (s *paymentService) ProcessPayment(ctx context.Context, req *entities.ProcessPaymentRequest) (*entities.PaymentDTO, error) {
	orderID, err := primitive.ObjectIDFromHex(req.OrderID)
	if err != nil {
		return nil, ErrOrderNotFound
	}

	order, err := s.orderRepo.FindByID(ctx, orderID)
	if err != nil || order == nil {
		return nil, ErrOrderNotFound
	}

	if order.Status != entities.OrderCompleted {
		return nil, ErrPaymentNotAllowed
	}

	// Simple validation (no real gateway)
	if req.Amount <= 0 || req.Amount > order.TotalAmount {
		return nil, errors.New("invalid payment amount")
	}

	now := time.Now()
	payment := &entities.Payment{
		OrderID:   orderID,
		Amount:    req.Amount,
		Method:    req.Method,
		Status:    entities.PaymentCompleted,
		PaidAt:    &now,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	createdPayment, err := s.paymentRepo.Create(ctx, payment)
	if err != nil {
		s.logger.Error("Failed to process payment", zap.Error(err))
		return nil, err
	}

	// Update order status to PAID
	if err := s.orderRepo.UpdateStatus(ctx, orderID, entities.OrderPaid); err != nil {
		s.logger.Error("Failed to update order status to PAID", zap.Error(err))
	}

	// Reset table status
	if err := s.tableRepo.Update(ctx, order.TableID, bson.M{
		"status":           entities.TableAvailable,
		"current_order_id": primitive.NilObjectID,
	}); err != nil {
		s.logger.Error("Failed to reset table on payment", zap.Error(err))
	}

	s.logger.Info("Payment processed", zap.String("payment_id", createdPayment.ID.Hex()), zap.String("order_id", req.OrderID))
	return createdPayment.ToDTO(), nil
}

func (s *paymentService) ProcessRefund(ctx context.Context, orderID string, req *entities.RefundRequest) error {
	oid, err := primitive.ObjectIDFromHex(orderID)
	if err != nil {
		return ErrOrderNotFound
	}

	order, err := s.orderRepo.FindByID(ctx, oid)
	if err != nil || order == nil {
		return ErrOrderNotFound
	}

	if order.Status != entities.OrderPaid {
		return ErrRefundNotAllowed
	}

	payments, err := s.paymentRepo.FindByOrderID(ctx, oid)
	if err != nil || len(payments) == 0 {
		return ErrNoPaymentsFound
	}

	paidTotal := 0.0
	for _, p := range payments {
		if p.Status == entities.PaymentCompleted {
			paidTotal += p.Amount
		}
	}

	if req.Amount > paidTotal {
		return ErrInvalidRefundAmount
	}

	// Simple refund logic (no real gateway reversal)
	refundTime := time.Now()
	for _, p := range payments {
		if p.Status == entities.PaymentCompleted && p.Amount >= req.Amount {
			update := bson.M{
				"status":       entities.PaymentRefunded,
				"refunded_at":  refundTime,
				"refund_notes": req.Reason,
				"updated_at":   time.Now(),
			}
			if err := s.paymentRepo.Update(ctx, p.ID, update); err != nil {
				s.logger.Error("Failed to update payment for refund", zap.Error(err))
				return err
			}
			break
		}
	}

	// Optional: restore inventory if refund full amount (business rule)
	if req.Amount == paidTotal {
		for _, item := range order.Items {
			if err := s.menuItemRepo.UpdateInventory(ctx, item.MenuItemID, item.Quantity); err != nil {
				s.logger.Error("Failed to restore inventory on full refund", zap.Error(err))
			}
		}
	}

	// Reset order status if full refund (optional - depends on business rule)
	if req.Amount == paidTotal {
		if err := s.orderRepo.UpdateStatus(ctx, oid, entities.OrderCompleted); err != nil {
			s.logger.Error("Failed to revert order status on full refund", zap.Error(err))
		}
	}

	s.logger.Info("Refund processed", zap.String("order_id", orderID), zap.Float64("amount", req.Amount))
	return nil
}

func (s *paymentService) GetPaymentsForOrder(ctx context.Context, orderID string) ([]*entities.PaymentDTO, error) {
	oid, err := primitive.ObjectIDFromHex(orderID)
	if err != nil {
		return nil, ErrOrderNotFound
	}

	payments, err := s.paymentRepo.FindByOrderID(ctx, oid)
	if err != nil {
		s.logger.Error("Failed to get payments for order", zap.Error(err))
		return nil, err
	}

	dtos := make([]*entities.PaymentDTO, len(payments))
	for i, p := range payments {
		dtos[i] = p.ToDTO()
	}

	return dtos, nil
}
