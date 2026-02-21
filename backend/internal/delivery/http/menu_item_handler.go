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

type MenuItemHandler struct {
	service services.MenuItemService
	logger  *zap.Logger
}

func NewMenuItemHandler(service services.MenuItemService, logger *zap.Logger) *MenuItemHandler {
	return &MenuItemHandler{service, logger}
}

func (h *MenuItemHandler) Create(c *gin.Context) {
	var req entities.CreateMenuItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid create menu item request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	itemDTO, err := h.service.CreateItem(c.Request.Context(), &req)
	if err != nil {
		if errors.Is(err, services.ErrInvalidCategory) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		h.logger.Error("Create menu item failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create menu item"})
		return
	}

	c.JSON(http.StatusCreated, itemDTO)
}

func (h *MenuItemHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.Query("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.Query("limit"))
	if limit < 1 {
		limit = 20
	}
	categoryID := c.Query("category_id")
	search := c.Query("search")
	var isAvailable *bool
	if availStr := c.Query("is_available"); availStr != "" {
		b := availStr == "true"
		isAvailable = &b
	}

	items, total, err := h.service.ListItems(c.Request.Context(), page, limit, categoryID, search, isAvailable)
	if err != nil {
		h.logger.Error("List menu items failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list menu items"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  items,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

func (h *MenuItemHandler) Get(c *gin.Context) {
	id := c.Param("id")
	itemDTO, err := h.service.GetItem(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, services.ErrMenuItemNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "menu item not found"})
			return
		}
		h.logger.Error("Get menu item failed", zap.String("id", id), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusOK, itemDTO)
}

func (h *MenuItemHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req entities.UpdateMenuItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	itemDTO, err := h.service.UpdateItem(c.Request.Context(), id, &req)
	if err != nil {
		if errors.Is(err, services.ErrMenuItemNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "menu item not found"})
			return
		}
		h.logger.Error("Update menu item failed", zap.String("id", id), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update menu item"})
		return
	}

	c.JSON(http.StatusOK, itemDTO)
}

func (h *MenuItemHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.DeleteItem(c.Request.Context(), id); err != nil {
		if errors.Is(err, services.ErrMenuItemNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "menu item not found"})
			return
		}
		h.logger.Error("Delete menu item failed", zap.String("id", id), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete menu item"})
		return
	}

	c.Status(http.StatusNoContent)
}
