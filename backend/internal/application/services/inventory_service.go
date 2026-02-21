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
	ErrInventoryItemNameExists = errors.New("inventory item name already exists")
	ErrInventoryItemNotFound   = errors.New("inventory item not found")
	ErrLowStock                = errors.New("low stock alert")
)

type InventoryService interface {
	CreateItem(ctx context.Context, req *entities.CreateInventoryItemRequest) (*entities.InventoryItemDTO, error)
	ListItems(ctx context.Context, page, limit int, search string) ([]*entities.InventoryItemDTO, int64, error)
	GetItem(ctx context.Context, id string) (*entities.InventoryItemDTO, error)
	UpdateItem(ctx context.Context, id string, req *entities.UpdateInventoryItemRequest) (*entities.InventoryItemDTO, error)
	DeleteItem(ctx context.Context, id string) error
	DeductForOrder(ctx context.Context, order *entities.Order) error // auto-deduct based on recipe
	GetInventoryReport(ctx context.Context) (*entities.InventoryReport, error)
}

type inventoryService struct {
	inventoryRepo repositories.InventoryRepository
	recipeRepo    repositories.RecipeRepository
	logger        *zap.Logger
}

func NewInventoryService(inventoryRepo repositories.InventoryRepository, recipeRepo repositories.RecipeRepository, logger *zap.Logger) InventoryService {
	return &inventoryService{inventoryRepo, recipeRepo, logger}
}

func (s *inventoryService) CreateItem(ctx context.Context, req *entities.CreateInventoryItemRequest) (*entities.InventoryItemDTO, error) {
	// Check uniqueness
	filter := bson.M{"name": req.Name}
	items, _, _ := s.inventoryRepo.FindAll(ctx, filter, nil)
	if len(items) > 0 {
		return nil, ErrInventoryItemNameExists
	}

	item := &entities.InventoryItem{
		Name:              req.Name,
		Unit:              req.Unit,
		Quantity:          req.Quantity,
		LowStockThreshold: req.LowStockThreshold,
		SupplierInfo:      req.SupplierInfo,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	created, err := s.inventoryRepo.Create(ctx, item)
	if err != nil {
		s.logger.Error("Failed to create inventory item", zap.Error(err))
		return nil, err
	}

	s.logger.Info("Inventory item created", zap.String("name", created.Name), zap.String("id", created.ID.Hex()))
	dto := created.ToDTO()
	return &dto, nil
}

func (s *inventoryService) ListItems(ctx context.Context, page, limit int, search string) ([]*entities.InventoryItemDTO, int64, error) {
	filter := bson.M{}

	if search != "" {
		filter["name"] = bson.M{"$regex": search, "$options": "i"}
	}

	opts := options.Find().
		SetSkip(int64((page - 1) * limit)).
		SetLimit(int64(limit)).
		SetSort(bson.M{"name": 1})

	items, total, err := s.inventoryRepo.FindAll(ctx, filter, opts)
	if err != nil {
		s.logger.Error("Failed to list inventory items", zap.Error(err))
		return nil, 0, err
	}

	dtos := make([]*entities.InventoryItemDTO, len(items))
	for i, item := range items {
		dto := item.ToDTO()
		dtos[i] = &dto
	}

	return dtos, total, nil
}

func (s *inventoryService) GetItem(ctx context.Context, id string) (*entities.InventoryItemDTO, error) {
	iid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, ErrInventoryItemNotFound
	}

	item, err := s.inventoryRepo.FindByID(ctx, iid)
	if err != nil {
		s.logger.Error("Failed to get inventory item", zap.String("id", id), zap.Error(err))
		return nil, err
	}
	if item == nil {
		return nil, ErrInventoryItemNotFound
	}

	dto := item.ToDTO()
	return &dto, nil
}

func (s *inventoryService) UpdateItem(ctx context.Context, id string, req *entities.UpdateInventoryItemRequest) (*entities.InventoryItemDTO, error) {
	iid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, ErrInventoryItemNotFound
	}

	item, err := s.inventoryRepo.FindByID(ctx, iid)
	if err != nil || item == nil {
		return nil, ErrInventoryItemNotFound
	}

	updateFields := bson.M{}

	if req.Name != nil {
		updateFields["name"] = *req.Name
	}
	if req.Unit != nil {
		updateFields["unit"] = *req.Unit
	}
	if req.Quantity != nil {
		updateFields["quantity"] = *req.Quantity
	}
	if req.LowStockThreshold != nil {
		updateFields["low_stock_threshold"] = *req.LowStockThreshold
	}
	if req.SupplierInfo != nil {
		updateFields["supplier_info"] = *req.SupplierInfo
	}

	if len(updateFields) == 0 {
		dto := item.ToDTO()
		return &dto, nil
	}

	if err := s.inventoryRepo.Update(ctx, iid, updateFields); err != nil {
		s.logger.Error("Failed to update inventory item", zap.String("id", id), zap.Error(err))
		return nil, err
	}

	updated, _ := s.inventoryRepo.FindByID(ctx, iid)
	s.logger.Info("Inventory item updated", zap.String("id", id))
	dto := updated.ToDTO()
	return &dto, nil
}

func (s *inventoryService) DeleteItem(ctx context.Context, id string) error {
	iid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return ErrInventoryItemNotFound
	}

	item, err := s.inventoryRepo.FindByID(ctx, iid)
	if err != nil || item == nil {
		return ErrInventoryItemNotFound
	}

	if err := s.inventoryRepo.Delete(ctx, iid); err != nil {
		s.logger.Error("Failed to delete inventory item", zap.String("id", id), zap.Error(err))
		return err
	}

	s.logger.Info("Inventory item deleted", zap.String("id", id))
	return nil
}

func (s *inventoryService) DeductForOrder(ctx context.Context, order *entities.Order) error {
	for _, orderItem := range order.Items {
		recipe, err := s.recipeRepo.FindByMenuItemID(ctx, orderItem.MenuItemID)
		if err != nil || recipe == nil {
			s.logger.Warn("No recipe found for menu item", zap.String("menu_item_id", orderItem.MenuItemID.Hex()))
			continue
		}

		for _, ingredient := range recipe.Ingredients {
			quantityToDeduct := ingredient.QuantityPerUnit * float64(orderItem.Quantity)
			if err := s.inventoryRepo.UpdateQuantity(ctx, ingredient.InventoryItemID, -quantityToDeduct); err != nil {
				s.logger.Error("Failed to deduct inventory for ingredient", zap.Error(err))
				return err
			}
		}
	}

	return nil
}

func (s *inventoryService) GetInventoryReport(ctx context.Context) (*entities.InventoryReport, error) {
	lowStockItems, err := s.inventoryRepo.GetLowStock(ctx)
	if err != nil {
		s.logger.Error("Failed to get low stock items", zap.Error(err))
		return nil, err
	}

	// Convert []*InventoryItem → []InventoryItemDTO (value slice)
	lowStockDTOs := make([]entities.InventoryItemDTO, 0, len(lowStockItems))
	for _, item := range lowStockItems {
		if item != nil {
			lowStockDTOs = append(lowStockDTOs, item.ToDTO())
		}
	}

	// Count total items
	filter := bson.M{}
	_, total, err := s.inventoryRepo.FindAll(ctx, filter, nil)
	if err != nil {
		s.logger.Error("Failed to count total inventory items", zap.Error(err))
		return nil, err
	}

	return &entities.InventoryReport{
		LowStockItems: lowStockDTOs,
		TotalItems:    int(total),
	}, nil
}
