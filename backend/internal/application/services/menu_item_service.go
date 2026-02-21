package services

import (
	"context"
	"errors"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"

	"github.com/sohamify/rms-backend/internal/domain/entities"
	"github.com/sohamify/rms-backend/internal/domain/repositories"
)

var (
	ErrMenuItemNameExists    = errors.New("menu item name already exists")
	ErrMenuItemNotFound      = errors.New("menu item not found")
	ErrInvalidCategory       = errors.New("invalid category ID")
	ErrInsufficientInventory = errors.New("insufficient inventory")
)

type MenuItemService interface {
	CreateItem(ctx context.Context, req *entities.CreateMenuItemRequest) (*entities.MenuItemDTO, error)
	ListItems(ctx context.Context, page, limit int, categoryID, search string, isAvailable *bool) ([]*entities.MenuItemDTO, int64, error)
	GetItem(ctx context.Context, id string) (*entities.MenuItemDTO, error)
	UpdateItem(ctx context.Context, id string, req *entities.UpdateMenuItemRequest) (*entities.MenuItemDTO, error)
	DeleteItem(ctx context.Context, id string) error
	DeductInventory(ctx context.Context, id string, quantity int) error
}

type menuItemService struct {
	menuItemRepo repositories.MenuItemRepository
	categoryRepo repositories.MenuCategoryRepository // for validation
	logger       *zap.Logger
}

func NewMenuItemService(menuItemRepo repositories.MenuItemRepository, categoryRepo repositories.MenuCategoryRepository, logger *zap.Logger) MenuItemService {
	return &menuItemService{menuItemRepo, categoryRepo, logger}
}

func (s *menuItemService) CreateItem(ctx context.Context, req *entities.CreateMenuItemRequest) (*entities.MenuItemDTO, error) {
	cid, err := primitive.ObjectIDFromHex(req.CategoryID)
	if err != nil {
		return nil, ErrInvalidCategory
	}

	_, err = s.categoryRepo.FindByID(ctx, cid)
	if err != nil {
		return nil, ErrInvalidCategory
	}

	// Check unique name within category (optional - adjust if needed)
	filter := bson.M{"name": req.Name, "category_id": cid}
	existing, _, _ := s.menuItemRepo.FindAll(ctx, filter, nil)
	if len(existing) > 0 {
		return nil, ErrMenuItemNameExists
	}

	item := &entities.MenuItem{
		CategoryID:   cid,
		Name:         strings.ToTitle(req.Name),
		Description:  req.Description,
		Price:        req.Price,
		IsAvailable:  true,
		InventoryQty: req.InventoryQty,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	created, err := s.menuItemRepo.Create(ctx, item)
	if err != nil {
		s.logger.Error("Failed to create menu item", zap.Error(err))
		return nil, err
	}

	s.logger.Info("Menu item created", zap.String("name", created.Name), zap.String("id", created.ID.Hex()))
	return created.ToDTO(), nil
}

func (s *menuItemService) ListItems(ctx context.Context, page, limit int, categoryID, search string, isAvailable *bool) ([]*entities.MenuItemDTO, int64, error) {
	filter := bson.M{}

	if categoryID != "" {
		cid, err := primitive.ObjectIDFromHex(categoryID)
		if err == nil {
			filter["category_id"] = cid
		}
	}

	if isAvailable != nil {
		filter["is_available"] = *isAvailable
	}

	if search != "" {
		filter["$or"] = []bson.M{
			{"name": bson.M{"$regex": search, "$options": "i"}},
			{"description": bson.M{"$regex": search, "$options": "i"}},
		}
	}

	opts := options.Find().
		SetSkip(int64((page - 1) * limit)).
		SetLimit(int64(limit)).
		SetSort(bson.M{"name": 1})

	items, total, err := s.menuItemRepo.FindAll(ctx, filter, opts)
	if err != nil {
		s.logger.Error("Failed to list menu items", zap.Error(err))
		return nil, 0, err
	}

	dtos := make([]*entities.MenuItemDTO, len(items))
	for i, item := range items {
		dtos[i] = item.ToDTO()
	}

	return dtos, total, nil
}

func (s *menuItemService) GetItem(ctx context.Context, id string) (*entities.MenuItemDTO, error) {
	mid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, ErrMenuItemNotFound
	}

	item, err := s.menuItemRepo.FindByID(ctx, mid)
	if err != nil {
		s.logger.Error("Failed to get menu item", zap.String("id", id), zap.Error(err))
		return nil, err
	}
	if item == nil {
		return nil, ErrMenuItemNotFound
	}

	return item.ToDTO(), nil
}

func (s *menuItemService) UpdateItem(ctx context.Context, id string, req *entities.UpdateMenuItemRequest) (*entities.MenuItemDTO, error) {
	mid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, ErrMenuItemNotFound
	}

	item, err := s.menuItemRepo.FindByID(ctx, mid)
	if err != nil || item == nil {
		return nil, ErrMenuItemNotFound
	}

	updateFields := bson.M{}

	if req.Name != nil {
		updateFields["name"] = strings.ToTitle(*req.Name)
	}
	if req.Description != nil {
		updateFields["description"] = *req.Description
	}
	if req.Price != nil {
		if *req.Price < 0 {
			return nil, errors.New("price cannot be negative")
		}
		updateFields["price"] = *req.Price
	}
	if req.IsAvailable != nil {
		updateFields["is_available"] = *req.IsAvailable
	}
	if req.InventoryQty != nil {
		if *req.InventoryQty < 0 {
			return nil, errors.New("inventory quantity cannot be negative")
		}
		updateFields["inventory_qty"] = *req.InventoryQty
	}

	if len(updateFields) == 0 {
		return item.ToDTO(), nil
	}

	if err := s.menuItemRepo.Update(ctx, mid, updateFields); err != nil {
		s.logger.Error("Failed to update menu item", zap.String("id", id), zap.Error(err))
		return nil, err
	}

	updated, _ := s.menuItemRepo.FindByID(ctx, mid)
	s.logger.Info("Menu item updated", zap.String("id", id))
	return updated.ToDTO(), nil
}

func (s *menuItemService) DeleteItem(ctx context.Context, id string) error {
	mid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return ErrMenuItemNotFound
	}

	item, err := s.menuItemRepo.FindByID(ctx, mid)
	if err != nil || item == nil {
		return ErrMenuItemNotFound
	}

	if err := s.menuItemRepo.Delete(ctx, mid); err != nil {
		s.logger.Error("Failed to delete menu item", zap.String("id", id), zap.Error(err))
		return err
	}

	s.logger.Info("Menu item deleted", zap.String("id", id))
	return nil
}

func (s *menuItemService) DeductInventory(ctx context.Context, id string, quantity int) error {
	mid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return ErrMenuItemNotFound
	}

	item, err := s.menuItemRepo.FindByID(ctx, mid)
	if err != nil || item == nil {
		return ErrMenuItemNotFound
	}

	if item.InventoryQty < quantity {
		return ErrInsufficientInventory
	}

	if err := s.menuItemRepo.UpdateInventory(ctx, mid, -quantity); err != nil {
		s.logger.Error("Failed to deduct inventory", zap.String("id", id), zap.Int("quantity", quantity), zap.Error(err))
		return err
	}

	s.logger.Info("Inventory deducted", zap.String("id", id), zap.Int("quantity", quantity))
	return nil
}
