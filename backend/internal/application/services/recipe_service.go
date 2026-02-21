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
	ErrRecipeAlreadyExists = errors.New("recipe already exists for this menu item")
	ErrRecipeNotFound      = errors.New("recipe not found")
	ErrInvalidMenuItem     = errors.New("invalid menu item ID")
	ErrInvalidIngredient   = errors.New("invalid ingredient or quantity")
)

type RecipeService interface {
	CreateRecipe(ctx context.Context, req *entities.CreateRecipeRequest) (*entities.RecipeDTO, error)
	GetRecipe(ctx context.Context, menuItemID string) (*entities.RecipeDTO, error)
	UpdateRecipe(ctx context.Context, menuItemID string, req *entities.UpdateRecipeRequest) (*entities.RecipeDTO, error)
	DeleteRecipe(ctx context.Context, menuItemID string) error
}

type recipeService struct {
	recipeRepo   repositories.RecipeRepository
	menuItemRepo repositories.MenuItemRepository // to validate menu item existence
	logger       *zap.Logger
}

func NewRecipeService(
	recipeRepo repositories.RecipeRepository,
	menuItemRepo repositories.MenuItemRepository,
	logger *zap.Logger,
) RecipeService {
	return &recipeService{
		recipeRepo:   recipeRepo,
		menuItemRepo: menuItemRepo,
		logger:       logger,
	}
}

func (s *recipeService) CreateRecipe(ctx context.Context, req *entities.CreateRecipeRequest) (*entities.RecipeDTO, error) {
	// Validate menu item exists
	menuItemID, err := primitive.ObjectIDFromHex(req.MenuItemID)
	if err != nil {
		return nil, ErrInvalidMenuItem
	}

	_, err = s.menuItemRepo.FindByID(ctx, menuItemID)
	if err != nil {
		s.logger.Error("Failed to validate menu item", zap.String("menu_item_id", req.MenuItemID), zap.Error(err))
		return nil, ErrInvalidMenuItem
	}

	// Check if recipe already exists for this menu item
	existing, err := s.recipeRepo.FindByMenuItemID(ctx, menuItemID)
	if err != nil {
		s.logger.Error("Failed to check existing recipe", zap.Error(err))
		return nil, errors.New("internal error")
	}
	if existing != nil {
		return nil, ErrRecipeAlreadyExists
	}

	// Validate ingredients
	ingredients := make([]entities.RecipeIngredient, len(req.Ingredients))
	for i, ing := range req.Ingredients {
		iid, err := primitive.ObjectIDFromHex(ing.InventoryItemID)
		if err != nil {
			return nil, ErrInvalidIngredient
		}

		if ing.QuantityPerUnit <= 0 {
			return nil, ErrInvalidIngredient
		}

		ingredients[i] = entities.RecipeIngredient{
			InventoryItemID: iid,
			QuantityPerUnit: ing.QuantityPerUnit,
		}
	}

	recipe := &entities.Recipe{
		MenuItemID:  menuItemID,
		Ingredients: ingredients,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	createdRecipe, err := s.recipeRepo.Create(ctx, recipe)
	if err != nil {
		s.logger.Error("Failed to create recipe", zap.Error(err))
		return nil, err
	}

	s.logger.Info("Recipe created",
		zap.String("menu_item_id", createdRecipe.MenuItemID.Hex()),
		zap.Int("ingredient_count", len(createdRecipe.Ingredients)),
	)

	return createdRecipe.ToDTO(), nil
}

func (s *recipeService) GetRecipe(ctx context.Context, menuItemID string) (*entities.RecipeDTO, error) {
	mid, err := primitive.ObjectIDFromHex(menuItemID)
	if err != nil {
		return nil, ErrRecipeNotFound
	}

	recipe, err := s.recipeRepo.FindByMenuItemID(ctx, mid)
	if err != nil {
		s.logger.Error("Failed to get recipe", zap.String("menu_item_id", menuItemID), zap.Error(err))
		return nil, err
	}
	if recipe == nil {
		return nil, ErrRecipeNotFound
	}

	return recipe.ToDTO(), nil
}

func (s *recipeService) UpdateRecipe(ctx context.Context, menuItemID string, req *entities.UpdateRecipeRequest) (*entities.RecipeDTO, error) {
	mid, err := primitive.ObjectIDFromHex(menuItemID)
	if err != nil {
		return nil, ErrRecipeNotFound
	}

	// Check recipe exists
	existing, err := s.recipeRepo.FindByMenuItemID(ctx, mid)
	if err != nil || existing == nil {
		return nil, ErrRecipeNotFound
	}

	if req.Ingredients == nil {
		return existing.ToDTO(), nil // no changes
	}

	// Validate new ingredients
	ingredients := make([]entities.RecipeIngredient, len(*req.Ingredients))
	for i, ing := range *req.Ingredients {
		iid, err := primitive.ObjectIDFromHex(ing.InventoryItemID)
		if err != nil {
			return nil, ErrInvalidIngredient
		}

		if ing.QuantityPerUnit <= 0 {
			return nil, ErrInvalidIngredient
		}

		ingredients[i] = entities.RecipeIngredient{
			InventoryItemID: iid,
			QuantityPerUnit: ing.QuantityPerUnit,
		}
	}

	updateFields := bson.M{
		"ingredients": ingredients,
		"updated_at":  time.Now(),
	}

	if err := s.recipeRepo.Update(ctx, mid, updateFields); err != nil {
		s.logger.Error("Failed to update recipe", zap.String("menu_item_id", menuItemID), zap.Error(err))
		return nil, err
	}

	updated, err := s.recipeRepo.FindByMenuItemID(ctx, mid)
	if err != nil {
		s.logger.Error("Failed to fetch updated recipe", zap.Error(err))
		return nil, err
	}

	s.logger.Info("Recipe updated", zap.String("menu_item_id", menuItemID))
	return updated.ToDTO(), nil
}

func (s *recipeService) DeleteRecipe(ctx context.Context, menuItemID string) error {
	mid, err := primitive.ObjectIDFromHex(menuItemID)
	if err != nil {
		return ErrRecipeNotFound
	}

	recipe, err := s.recipeRepo.FindByMenuItemID(ctx, mid)
	if err != nil || recipe == nil {
		return ErrRecipeNotFound
	}

	if err := s.recipeRepo.Delete(ctx, mid); err != nil {
		s.logger.Error("Failed to delete recipe", zap.String("menu_item_id", menuItemID), zap.Error(err))
		return err
	}

	s.logger.Info("Recipe deleted", zap.String("menu_item_id", menuItemID))
	return nil
}
