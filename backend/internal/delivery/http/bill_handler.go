package http

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/sohamify/rms-backend/internal/application/services"
	"github.com/sohamify/rms-backend/internal/domain/entities"
)

type BillHandler struct {
	service services.BillService
	logger  *zap.Logger
}

func NewBillHandler(service services.BillService, logger *zap.Logger) *BillHandler {
	return &BillHandler{service, logger}
}

func (h *BillHandler) GenerateBill(c *gin.Context) {
	var req entities.GenerateBillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	billDTO, err := h.service.GenerateBill(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("Bill generation failed", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, billDTO)
}

func (h *BillHandler) GetBill(c *gin.Context) {
	orderID := c.Param("order_id")
	billDTO, err := h.service.GetBillForOrder(c.Request.Context(), orderID)
	if err != nil {
		if errors.Is(err, services.ErrBillNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "bill not found"})
			return
		}
		h.logger.Error("Failed to get bill", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusOK, billDTO)
}
