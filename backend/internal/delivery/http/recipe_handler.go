package http

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/sohamify/rms-backend/internal/application/services"
	"github.com/sohamify/rms-backend/internal/domain/entities"
)

type RecipeHandler struct {
	service services.RecipeService
	logger  *zap.Logger
}

func NewRecipeHandler(service services.RecipeService, logger *zap.Logger) *RecipeHandler {
	return &RecipeHandler{service, logger}
}

func (h *RecipeHandler) Create(c *gin.Context) {
	var req entities.CreateRecipeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	recipeDTO, err := h.service.CreateRecipe(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("Create recipe failed", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, recipeDTO)
}

func (h *RecipeHandler) Get(c *gin.Context) {
	menuItemID := c.Param("menu_item_id")
	recipeDTO, err := h.service.GetRecipe(c.Request.Context(), menuItemID)
	if err != nil {
		if errors.Is(err, services.ErrRecipeNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "recipe not found"})
			return
		}
		h.logger.Error("Get recipe failed", zap.String("menu_item_id", menuItemID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusOK, recipeDTO)
}

func (h *RecipeHandler) Update(c *gin.Context) {
	menuItemID := c.Param("menu_item_id")
	var req entities.UpdateRecipeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	recipeDTO, err := h.service.UpdateRecipe(c.Request.Context(), menuItemID, &req)
	if err != nil {
		if errors.Is(err, services.ErrRecipeNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "recipe not found"})
			return
		}
		h.logger.Error("Update recipe failed", zap.String("menu_item_id", menuItemID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update recipe"})
		return
	}

	c.JSON(http.StatusOK, recipeDTO)
}

func (h *RecipeHandler) Delete(c *gin.Context) {
	menuItemID := c.Param("menu_item_id")
	if err := h.service.DeleteRecipe(c.Request.Context(), menuItemID); err != nil {
		if errors.Is(err, services.ErrRecipeNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "recipe not found"})
			return
		}
		h.logger.Error("Delete recipe failed", zap.String("menu_item_id", menuItemID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete recipe"})
		return
	}

	c.Status(http.StatusNoContent)
}
