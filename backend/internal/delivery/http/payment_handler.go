package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/sohamify/rms-backend/internal/application/services"
	"github.com/sohamify/rms-backend/internal/domain/entities"
)

type PaymentHandler struct {
	service services.PaymentService
	logger  *zap.Logger
}

func NewPaymentHandler(service services.PaymentService, logger *zap.Logger) *PaymentHandler {
	return &PaymentHandler{service, logger}
}

func (h *PaymentHandler) ProcessPayment(c *gin.Context) {
	var req entities.ProcessPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	paymentDTO, err := h.service.ProcessPayment(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("Payment processing failed", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, paymentDTO)
}

func (h *PaymentHandler) ProcessRefund(c *gin.Context) {
	orderID := c.Param("order_id")
	var req entities.RefundRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.ProcessRefund(c.Request.Context(), orderID, &req); err != nil {
		h.logger.Error("Refund failed", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *PaymentHandler) GetPayments(c *gin.Context) {
	orderID := c.Param("order_id")
	payments, err := h.service.GetPaymentsForOrder(c.Request.Context(), orderID)
	if err != nil {
		h.logger.Error("Failed to get payments", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, payments)
}
