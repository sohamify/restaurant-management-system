package services

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"

	"github.com/sohamify/rms-backend/internal/domain/entities"
	"github.com/sohamify/rms-backend/internal/domain/repositories"
)

var (
	ErrOrderNotFound      = errors.New("order not found")
	ErrInvalidOrderStatus = errors.New("invalid order status transition")
	ErrTableOccupied      = errors.New("table is already occupied")
	ErrNoActiveOrder      = errors.New("no active order on this table")
	ErrOrderAlreadyPaid   = errors.New("order is already paid")
	ErrOrderCancelled     = errors.New("order is already cancelled")
)

type OrderService interface {
	CreateOrder(ctx context.Context, req *entities.CreateOrderRequest, waiterID string) (*entities.OrderDTO, error)
	AddItemToOrder(ctx context.Context, orderID string, req *entities.AddItemToOrderRequest) (*entities.OrderDTO, error)
	RemoveItemFromOrder(ctx context.Context, orderID string, menuItemID string) (*entities.OrderDTO, error)
	UpdateOrderStatus(ctx context.Context, orderID string, status entities.OrderStatus) (*entities.OrderDTO, error)
	CancelOrder(ctx context.Context, orderID string, reason string) error
	MarkOrderPaid(ctx context.Context, orderID string, paymentMethod string) error
	GetOrder(ctx context.Context, id string) (*entities.OrderDTO, error)
	ListOrders(ctx context.Context, page, limit int, status string, tableID string) ([]*entities.OrderDTO, int64, error)
	ListActiveKitchenOrders(ctx context.Context) ([]*entities.OrderDTO, error)
}

type orderService struct {
	orderRepo    repositories.OrderRepository
	tableRepo    repositories.TableRepository
	menuItemRepo repositories.MenuItemRepository
	logger       *zap.Logger
}

func NewOrderService(
	orderRepo repositories.OrderRepository,
	tableRepo repositories.TableRepository,
	menuItemRepo repositories.MenuItemRepository,
	logger *zap.Logger,
) OrderService {
	return &orderService{orderRepo, tableRepo, menuItemRepo, logger}
}

func (s *orderService) CreateOrder(ctx context.Context, req *entities.CreateOrderRequest, waiterID string) (*entities.OrderDTO, error) {
	tableID, err := primitive.ObjectIDFromHex(req.TableID)
	if err != nil {
		return nil, ErrTableNotFound
	}

	table, err := s.tableRepo.FindByID(ctx, tableID)
	if err != nil || table == nil {
		return nil, ErrTableNotFound
	}

	if table.Status != entities.TableAvailable {
		return nil, ErrTableOccupied
	}

	total := 0.0
	items := make([]entities.OrderItem, len(req.Items))
	for i, it := range req.Items {
		mid, err := primitive.ObjectIDFromHex(it.MenuItemID)
		if err != nil {
			return nil, errors.New("invalid menu item ID")
		}

		menuItem, err := s.menuItemRepo.FindByID(ctx, mid)
		if err != nil || menuItem == nil {
			return nil, errors.New("menu item not found")
		}

		// Check inventory locally
		if menuItem.InventoryQty < it.Quantity {
			return nil, ErrInsufficientInventory
		}

		items[i] = entities.OrderItem{
			MenuItemID:  mid,
			Quantity:    it.Quantity,
			PriceAtTime: menuItem.Price,
			Notes:       it.Notes,
			Status:      "PENDING",
		}

		total += menuItem.Price * float64(it.Quantity)
	}

	waiterOID, _ := primitive.ObjectIDFromHex(waiterID)
	order := &entities.Order{
		TableID:     tableID,
		WaiterID:    waiterOID,
		Items:       items,
		TotalAmount: total,
		Status:      entities.OrderPlaced,
		Notes:       req.Notes,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	createdOrder, err := s.orderRepo.Create(ctx, order)
	if err != nil {
		s.logger.Error("Failed to create order", zap.Error(err))
		return nil, err
	}

	// Deduct inventory
	for _, item := range items {
		if err := s.menuItemRepo.UpdateInventory(ctx, item.MenuItemID, -item.Quantity); err != nil {
			s.logger.Error("Failed to deduct inventory during order creation", zap.Error(err))
			// In production: add rollback logic or compensation transaction
		}
	}

	// Update table
	tableUpdate := bson.M{
		"status":           entities.TableOccupied,
		"current_order_id": createdOrder.ID,
	}
	if err := s.tableRepo.Update(ctx, tableID, tableUpdate); err != nil {
		s.logger.Error("Failed to update table after order creation", zap.Error(err))
	}

	s.logger.Info("Order created", zap.String("order_id", createdOrder.ID.Hex()), zap.String("table_id", tableID.Hex()))
	return createdOrder.ToDTO(), nil
}

func (s *orderService) AddItemToOrder(ctx context.Context, orderID string, req *entities.AddItemToOrderRequest) (*entities.OrderDTO, error) {
	oid, err := primitive.ObjectIDFromHex(orderID)
	if err != nil {
		return nil, ErrOrderNotFound
	}

	order, err := s.orderRepo.FindByID(ctx, oid)
	if err != nil || order == nil {
		return nil, ErrOrderNotFound
	}

	if order.Status == entities.OrderPaid || order.Status == entities.OrderCancelled {
		return nil, errors.New("cannot add items to paid or cancelled order")
	}

	mid, err := primitive.ObjectIDFromHex(req.MenuItemID)
	if err != nil {
		return nil, errors.New("invalid menu item ID")
	}

	item, err := s.menuItemRepo.FindByID(ctx, mid)
	if err != nil || item == nil {
		return nil, errors.New("menu item not found")
	}

	if item.InventoryQty < req.Quantity {
		return nil, ErrInsufficientInventory
	}

	newItem := entities.OrderItem{
		MenuItemID:  mid,
		Quantity:    req.Quantity,
		PriceAtTime: item.Price,
		Notes:       req.Notes,
		Status:      "PENDING",
	}

	if err := s.orderRepo.AddItem(ctx, oid, newItem); err != nil {
		s.logger.Error("Failed to add item to order", zap.Error(err))
		return nil, err
	}

	// Deduct inventory
	if err := s.menuItemRepo.UpdateInventory(ctx, mid, -req.Quantity); err != nil {
		s.logger.Error("Failed to deduct inventory on add item", zap.Error(err))
	}

	updatedOrder, _ := s.orderRepo.FindByID(ctx, oid)
	return updatedOrder.ToDTO(), nil
}

func (s *orderService) RemoveItemFromOrder(ctx context.Context, orderID string, menuItemID string) (*entities.OrderDTO, error) {
	oid, err := primitive.ObjectIDFromHex(orderID)
	if err != nil {
		return nil, ErrOrderNotFound
	}

	mid, err := primitive.ObjectIDFromHex(menuItemID)
	if err != nil {
		return nil, errors.New("invalid menu item ID")
	}

	order, err := s.orderRepo.FindByID(ctx, oid)
	if err != nil || order == nil {
		return nil, ErrOrderNotFound
	}

	if order.Status == entities.OrderPaid || order.Status == entities.OrderCancelled {
		return nil, errors.New("cannot remove items from paid or cancelled order")
	}

	if err := s.orderRepo.RemoveItem(ctx, oid, mid); err != nil {
		return nil, err
	}

	// Restore inventory (assuming quantity 1 — adjust if needed)
	if err := s.menuItemRepo.UpdateInventory(ctx, mid, 1); err != nil {
		s.logger.Error("Failed to restore inventory on remove item", zap.Error(err))
	}

	updatedOrder, _ := s.orderRepo.FindByID(ctx, oid)
	return updatedOrder.ToDTO(), nil
}

func (s *orderService) UpdateOrderStatus(ctx context.Context, orderID string, status entities.OrderStatus) (*entities.OrderDTO, error) {
	oid, err := primitive.ObjectIDFromHex(orderID)
	if err != nil {
		return nil, ErrOrderNotFound
	}

	order, err := s.orderRepo.FindByID(ctx, oid)
	if err != nil || order == nil {
		return nil, ErrOrderNotFound
	}

	if !isValidStatusTransition(order.Status, status) {
		return nil, ErrInvalidOrderStatus
	}

	if err := s.orderRepo.UpdateStatus(ctx, oid, status); err != nil {
		return nil, err
	}

	updatedOrder, _ := s.orderRepo.FindByID(ctx, oid)
	return updatedOrder.ToDTO(), nil
}

func (s *orderService) CancelOrder(ctx context.Context, orderID string, reason string) error {
	oid, err := primitive.ObjectIDFromHex(orderID)
	if err != nil {
		return ErrOrderNotFound
	}

	order, err := s.orderRepo.FindByID(ctx, oid)
	if err != nil || order == nil {
		return ErrOrderNotFound
	}

	if order.Status == entities.OrderPaid {
		return ErrOrderAlreadyPaid
	}

	if err := s.orderRepo.Cancel(ctx, oid, reason); err != nil {
		return err
	}

	// Restore inventory for all items
	for _, item := range order.Items {
		if err := s.menuItemRepo.UpdateInventory(ctx, item.MenuItemID, item.Quantity); err != nil {
			s.logger.Error("Failed to restore inventory on cancel", zap.Error(err))
		}
	}

	// Reset table status
	if err := s.tableRepo.Update(ctx, order.TableID, bson.M{
		"status":           entities.TableAvailable,
		"current_order_id": primitive.NilObjectID,
	}); err != nil {
		s.logger.Error("Failed to reset table on cancel", zap.Error(err))
	}

	return nil
}

func (s *orderService) MarkOrderPaid(ctx context.Context, orderID string, paymentMethod string) error {
	oid, err := primitive.ObjectIDFromHex(orderID)
	if err != nil {
		return ErrOrderNotFound
	}

	order, err := s.orderRepo.FindByID(ctx, oid)
	if err != nil || order == nil {
		return ErrOrderNotFound
	}

	if order.Status != entities.OrderCompleted {
		return errors.New("order must be completed before marking as paid")
	}

	if err := s.orderRepo.MarkPaid(ctx, oid, paymentMethod); err != nil {
		return err
	}

	// Reset table status
	if err := s.tableRepo.Update(ctx, order.TableID, bson.M{
		"status":           entities.TableAvailable,
		"current_order_id": primitive.NilObjectID,
	}); err != nil {
		s.logger.Error("Failed to reset table on payment", zap.Error(err))
	}

	return nil
}

func (s *orderService) GetOrder(ctx context.Context, id string) (*entities.OrderDTO, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, ErrOrderNotFound
	}

	order, err := s.orderRepo.FindByID(ctx, oid)
	if err != nil || order == nil {
		return nil, ErrOrderNotFound
	}

	return order.ToDTO(), nil
}

func (s *orderService) ListOrders(ctx context.Context, page, limit int, status string, tableID string) ([]*entities.OrderDTO, int64, error) {
	filter := bson.M{}

	if status != "" {
		filter["status"] = status
	}
	if tableID != "" {
		tid, err := primitive.ObjectIDFromHex(tableID)
		if err == nil {
			filter["table_id"] = tid
		}
	}

	opts := options.Find().
		SetSkip(int64((page - 1) * limit)).
		SetLimit(int64(limit)).
		SetSort(bson.M{"created_at": -1})

	orders, total, err := s.orderRepo.FindAll(ctx, filter, opts)
	if err != nil {
		s.logger.Error("Failed to list orders", zap.Error(err))
		return nil, 0, err
	}

	dtos := make([]*entities.OrderDTO, len(orders))
	for i, o := range orders {
		dtos[i] = o.ToDTO()
	}

	return dtos, total, nil
}

func (s *orderService) ListActiveKitchenOrders(ctx context.Context) ([]*entities.OrderDTO, error) {
	orders, err := s.orderRepo.FindActiveForKitchen(ctx)
	if err != nil {
		s.logger.Error("Failed to list active kitchen orders", zap.Error(err))
		return nil, err
	}

	dtos := make([]*entities.OrderDTO, len(orders))
	for i, o := range orders {
		dtos[i] = o.ToDTO()
	}

	return dtos, nil
}

// Helper: Validate status transitions
func isValidStatusTransition(current, next entities.OrderStatus) bool {
	validTransitions := map[entities.OrderStatus][]entities.OrderStatus{
		entities.OrderPlaced:    {entities.OrderPreparing, entities.OrderCancelled},
		entities.OrderPreparing: {entities.OrderReady, entities.OrderCancelled},
		entities.OrderReady:     {entities.OrderCompleted, entities.OrderCancelled},
		entities.OrderCompleted: {entities.OrderPaid},
		entities.OrderPaid:      {}, // terminal
		entities.OrderCancelled: {}, // terminal
	}

	allowed, ok := validTransitions[current]
	if !ok {
		return false
	}

	for _, ns := range allowed {
		if ns == next {
			return true
		}
	}
	return false
}
