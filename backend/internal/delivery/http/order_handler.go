package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/sohamify/rms-backend/internal/application/services"
	"github.com/sohamify/rms-backend/internal/domain/entities"
)

type OrderHandler struct {
	service services.OrderService
	logger  *zap.Logger
}

func NewOrderHandler(service services.OrderService, logger *zap.Logger) *OrderHandler {
	return &OrderHandler{service, logger}
}

func (h *OrderHandler) Create(c *gin.Context) {
	var req entities.CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	waiterID := c.GetString("user_id")
	orderDTO, err := h.service.CreateOrder(c.Request.Context(), &req, waiterID)
	if err != nil {
		h.logger.Error("Create order failed", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, orderDTO)
}

func (h *OrderHandler) AddItem(c *gin.Context) {
	orderID := c.Param("id")
	var req entities.AddItemToOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	orderDTO, err := h.service.AddItemToOrder(c.Request.Context(), orderID, &req)
	if err != nil {
		h.logger.Error("Add item to order failed", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, orderDTO)
}

// ... similar handlers for RemoveItem, UpdateStatus, Cancel, MarkPaid, List, Get, etc.
