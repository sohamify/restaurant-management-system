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

type InventoryHandler struct {
	service services.InventoryService
	logger  *zap.Logger
}

func NewInventoryHandler(service services.InventoryService, logger *zap.Logger) *InventoryHandler {
	return &InventoryHandler{service, logger}
}

func (h *InventoryHandler) Create(c *gin.Context) {
	var req entities.CreateInventoryItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid create inventory item request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	itemDTO, err := h.service.CreateItem(c.Request.Context(), &req)
	if err != nil {
		if errors.Is(err, services.ErrInventoryItemNameExists) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		h.logger.Error("Create inventory item failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create inventory item"})
		return
	}

	c.JSON(http.StatusCreated, itemDTO)
}

func (h *InventoryHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.Query("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.Query("limit"))
	if limit < 1 {
		limit = 20
	}
	search := c.Query("search")

	items, total, err := h.service.ListItems(c.Request.Context(), page, limit, search)
	if err != nil {
		h.logger.Error("List inventory items failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list inventory items"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  items,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

func (h *InventoryHandler) Get(c *gin.Context) {
	id := c.Param("id")
	itemDTO, err := h.service.GetItem(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, services.ErrInventoryItemNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "inventory item not found"})
			return
		}
		h.logger.Error("Get inventory item failed", zap.String("id", id), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusOK, itemDTO)
}

func (h *InventoryHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req entities.UpdateInventoryItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	itemDTO, err := h.service.UpdateItem(c.Request.Context(), id, &req)
	if err != nil {
		if errors.Is(err, services.ErrInventoryItemNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "inventory item not found"})
			return
		}
		h.logger.Error("Update inventory item failed", zap.String("id", id), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update inventory item"})
		return
	}

	c.JSON(http.StatusOK, itemDTO)
}

func (h *InventoryHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.DeleteItem(c.Request.Context(), id); err != nil {
		if errors.Is(err, services.ErrInventoryItemNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "inventory item not found"})
			return
		}
		h.logger.Error("Delete inventory item failed", zap.String("id", id), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete inventory item"})
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *InventoryHandler) GetReport(c *gin.Context) {
	report, err := h.service.GetInventoryReport(c.Request.Context())
	if err != nil {
		h.logger.Error("Failed to get inventory report", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get inventory report"})
		return
	}

	c.JSON(http.StatusOK, report)
}
