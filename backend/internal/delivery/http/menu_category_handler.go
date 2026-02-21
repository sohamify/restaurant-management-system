// internal/delivery/http/menu_category_handler.go
package http

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/sohamify/rms-backend/internal/application/services"
	"github.com/sohamify/rms-backend/internal/domain/entities"
)

type MenuCategoryHandler struct {
	service services.MenuCategoryService
	logger  *zap.Logger
}

func NewMenuCategoryHandler(service services.MenuCategoryService, logger *zap.Logger) *MenuCategoryHandler {
	return &MenuCategoryHandler{service: service, logger: logger}
}

// CreateCategory godoc
// @Summary      Create a new menu category
// @Description  Creates a new menu category (admin/manager only)
// @Tags         Menu Categories
// @Accept       json
// @Produce      json
// @Param        body  body  entities.CreateMenuCategoryRequest  true  "Category data"
// @Security     BearerAuth
// @Success      201   {object}  entities.MenuCategoryDTO
// @Failure      400   {object}  map[string]string
// @Failure      401   {object}  map[string]string
// @Failure      403   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Router       /menu/categories [post]
func (h *MenuCategoryHandler) Create(c *gin.Context) {
	var req entities.CreateMenuCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid create menu category request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	categoryDTO, err := h.service.CreateCategory(c.Request.Context(), &req)
	if err != nil {
		if errors.Is(err, services.ErrCategoryNameExists) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		h.logger.Error("Create menu category failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create menu category"})
		return
	}

	c.JSON(http.StatusCreated, categoryDTO)
}

// ListCategories godoc
// @Summary      List menu categories
// @Description  Get paginated list of menu categories with optional search
// @Tags         Menu Categories
// @Produce      json
// @Param        page     query  int     false  "Page number (default 1)"
// @Param        limit    query  int     false  "Items per page (default 20)"
// @Param        search   query  string  false  "Search by name"
// @Security     BearerAuth
// @Success      200      {object}  map[string]interface{}
// @Failure      400,401,403,500  {object}  map[string]string
// @Router       /menu/categories [get]
func (h *MenuCategoryHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.Query("page"))
	if page < 1 {
		page = 1
	}

	limit, _ := strconv.Atoi(c.Query("limit"))
	if limit < 1 {
		limit = 20
	}

	search := c.Query("search")

	categories, total, err := h.service.ListCategories(c.Request.Context(), page, limit, search)
	if err != nil {
		h.logger.Error("List menu categories failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list menu categories"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  categories,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// GetCategory godoc
// @Summary      Get a menu category by ID
// @Description  Retrieve a single menu category
// @Tags         Menu Categories
// @Produce      json
// @Param        id   path  string  true  "Category ID"
// @Security     BearerAuth
// @Success      200  {object}  entities.MenuCategoryDTO
// @Failure      404,500  {object}  map[string]string
// @Router       /menu/categories/{id} [get]
func (h *MenuCategoryHandler) Get(c *gin.Context) {
	id := c.Param("id")
	categoryDTO, err := h.service.GetCategory(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, services.ErrCategoryNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "menu category not found"})
			return
		}
		h.logger.Error("Get menu category failed", zap.String("id", id), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusOK, categoryDTO)
}

// UpdateCategory godoc
// @Summary      Update a menu category
// @Description  Update name and/or description of a menu category
// @Tags         Menu Categories
// @Accept       json
// @Produce      json
// @Param        id    path   string                              true   "Category ID"
// @Param        body  body   entities.UpdateMenuCategoryRequest  true   "Update fields"
// @Security     BearerAuth
// @Success      200   {object}  entities.MenuCategoryDTO
// @Failure      400,404,500  {object}  map[string]string
// @Router       /menu/categories/{id} [patch]
func (h *MenuCategoryHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req entities.UpdateMenuCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	categoryDTO, err := h.service.UpdateCategory(c.Request.Context(), id, &req)
	if err != nil {
		if errors.Is(err, services.ErrCategoryNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "menu category not found"})
			return
		}
		if errors.Is(err, services.ErrCategoryNameExists) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		h.logger.Error("Update menu category failed", zap.String("id", id), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update menu category"})
		return
	}

	c.JSON(http.StatusOK, categoryDTO)
}

// DeleteCategory godoc
// @Summary      Delete a menu category
// @Description  Delete a menu category (only if no menu items are using it)
// @Tags         Menu Categories
// @Produce      json
// @Param        id   path  string  true  "Category ID"
// @Security     BearerAuth
// @Success      204
// @Failure      400,404,500  {object}  map[string]string
// @Router       /menu/categories/{id} [delete]
func (h *MenuCategoryHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.DeleteCategory(c.Request.Context(), id); err != nil {
		if errors.Is(err, services.ErrCategoryNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "menu category not found"})
			return
		}
		if errors.Is(err, services.ErrCategoryInUse) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		h.logger.Error("Delete menu category failed", zap.String("id", id), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete menu category"})
		return
	}

	c.Status(http.StatusNoContent)
}
