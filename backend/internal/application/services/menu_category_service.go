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
	ErrCategoryNameExists = errors.New("category name already exists")
	ErrCategoryNotFound   = errors.New("menu category not found")
	ErrCategoryInUse      = errors.New("cannot delete category in use by menu items")
)

type MenuCategoryService interface {
	CreateCategory(ctx context.Context, req *entities.CreateMenuCategoryRequest) (*entities.MenuCategoryDTO, error)
	ListCategories(ctx context.Context, page, limit int, search string) ([]*entities.MenuCategoryDTO, int64, error)
	GetCategory(ctx context.Context, id string) (*entities.MenuCategoryDTO, error)
	UpdateCategory(ctx context.Context, id string, req *entities.UpdateMenuCategoryRequest) (*entities.MenuCategoryDTO, error)
	DeleteCategory(ctx context.Context, id string) error
}

type menuCategoryService struct {
	categoryRepo repositories.MenuCategoryRepository
	menuItemRepo repositories.MenuItemRepository // injected to check usage on delete
	logger       *zap.Logger
}

func NewMenuCategoryService(categoryRepo repositories.MenuCategoryRepository, menuItemRepo repositories.MenuItemRepository, logger *zap.Logger) MenuCategoryService {
	return &menuCategoryService{categoryRepo, menuItemRepo, logger}
}

func (s *menuCategoryService) CreateCategory(ctx context.Context, req *entities.CreateMenuCategoryRequest) (*entities.MenuCategoryDTO, error) {
	// Check uniqueness
	existing, err := s.categoryRepo.FindByName(ctx, req.Name)
	if err != nil {
		s.logger.Error("Failed to check category name uniqueness", zap.Error(err))
		return nil, errors.New("internal error")
	}
	if existing != nil {
		return nil, ErrCategoryNameExists
	}

	category := &entities.MenuCategory{
		Name:        strings.ToUpper(req.Name),
		Description: req.Description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	createdCategory, err := s.categoryRepo.Create(ctx, category)
	if err != nil {
		s.logger.Error("Failed to create category", zap.Error(err))
		return nil, err
	}

	s.logger.Info("Menu category created", zap.String("name", createdCategory.Name), zap.String("id", createdCategory.ID.Hex()))
	return createdCategory.ToDTO(), nil
}

func (s *menuCategoryService) ListCategories(ctx context.Context, page, limit int, search string) ([]*entities.MenuCategoryDTO, int64, error) {
	filter := bson.M{}

	if search != "" {
		filter["name"] = bson.M{"$regex": search, "$options": "i"}
	}

	opts := options.Find().
		SetSkip(int64((page - 1) * limit)).
		SetLimit(int64(limit)).
		SetSort(bson.M{"name": 1})

	categories, total, err := s.categoryRepo.FindAll(ctx, filter, opts)
	if err != nil {
		s.logger.Error("Failed to list menu categories", zap.Error(err))
		return nil, 0, err
	}

	dtos := make([]*entities.MenuCategoryDTO, len(categories))
	for i, c := range categories {
		dtos[i] = c.ToDTO()
	}

	return dtos, total, nil
}

func (s *menuCategoryService) GetCategory(ctx context.Context, id string) (*entities.MenuCategoryDTO, error) {
	cid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, ErrCategoryNotFound
	}

	category, err := s.categoryRepo.FindByID(ctx, cid)
	if err != nil {
		s.logger.Error("Failed to get menu category", zap.String("id", id), zap.Error(err))
		return nil, err
	}
	if category == nil {
		return nil, ErrCategoryNotFound
	}

	return category.ToDTO(), nil
}

func (s *menuCategoryService) UpdateCategory(ctx context.Context, id string, req *entities.UpdateMenuCategoryRequest) (*entities.MenuCategoryDTO, error) {
	cid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, ErrCategoryNotFound
	}

	category, err := s.categoryRepo.FindByID(ctx, cid)
	if err != nil || category == nil {
		return nil, ErrCategoryNotFound
	}

	updateFields := bson.M{}

	if req.Name != nil {
		// Check uniqueness if changing name
		existing, _ := s.categoryRepo.FindByName(ctx, *req.Name)
		if existing != nil && existing.ID != cid {
			return nil, ErrCategoryNameExists
		}
		updateFields["name"] = strings.ToUpper(*req.Name)
	}

	if req.Description != nil {
		updateFields["description"] = *req.Description
	}

	if len(updateFields) == 0 {
		return category.ToDTO(), nil // no changes
	}

	if err := s.categoryRepo.Update(ctx, cid, updateFields); err != nil {
		s.logger.Error("Failed to update menu category", zap.String("id", id), zap.Error(err))
		return nil, err
	}

	updated, _ := s.categoryRepo.FindByID(ctx, cid)
	s.logger.Info("Menu category updated", zap.String("id", id))
	return updated.ToDTO(), nil
}

func (s *menuCategoryService) DeleteCategory(ctx context.Context, id string) error {
	cid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return ErrCategoryNotFound
	}

	category, err := s.categoryRepo.FindByID(ctx, cid)
	if err != nil || category == nil {
		return ErrCategoryNotFound
	}

	// Check if any menu items use this category
	itemsFilter := bson.M{"category_id": cid}
	_, count, err := s.menuItemRepo.FindAll(ctx, itemsFilter, nil)
	if err != nil {
		s.logger.Error("Failed to count menu items for category", zap.String("category_id", id), zap.Error(err))
		return err
	}
	if count > 0 {
		return ErrCategoryInUse
	}

	if err := s.categoryRepo.Delete(ctx, cid); err != nil {
		s.logger.Error("Failed to delete menu category", zap.String("id", id), zap.Error(err))
		return err
	}

	s.logger.Info("Menu category deleted", zap.String("id", id))
	return nil
}
