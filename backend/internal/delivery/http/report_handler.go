package http

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/sohamify/rms-backend/internal/application/services"
	"github.com/sohamify/rms-backend/internal/domain/entities"
)

type ReportHandler struct {
	service services.ReportService
	logger  *zap.Logger
}

func NewReportHandler(service services.ReportService, logger *zap.Logger) *ReportHandler {
	return &ReportHandler{service, logger}
}

// parseReportFilter extracts common filter params from query
func (h *ReportHandler) parseReportFilter(c *gin.Context) (entities.ReportFilter, bool) {
	startStr := c.Query("start_date")
	endStr := c.Query("end_date")
	periodStr := c.Query("period")

	if startStr == "" || endStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "start_date and end_date are required (format: YYYY-MM-DD)"})
		return entities.ReportFilter{}, false
	}

	start, err := time.Parse("2006-01-02", startStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start_date format (use YYYY-MM-DD)"})
		return entities.ReportFilter{}, false
	}

	end, err := time.Parse("2006-01-02", endStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end_date format (use YYYY-MM-DD)"})
		return entities.ReportFilter{}, false
	}

	if end.Before(start) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "end_date cannot be before start_date"})
		return entities.ReportFilter{}, false
	}

	period := entities.ReportDaily
	switch periodStr {
	case "weekly":
		period = entities.ReportWeekly
	case "monthly":
		period = entities.ReportMonthly
	case "", "daily":
		period = entities.ReportDaily
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid period (daily, weekly, monthly)"})
		return entities.ReportFilter{}, false
	}

	return entities.ReportFilter{
		StartDate: start,
		EndDate:   end,
		Period:    period,
	}, true
}

// GetSalesSummary
// GET /reports/sales?start_date=2025-01-01&end_date=2025-01-31&period=monthly
func (h *ReportHandler) GetSalesSummary(c *gin.Context) {
	filter, ok := h.parseReportFilter(c)
	if !ok {
		return
	}

	summary, err := h.service.GetSalesSummary(c.Request.Context(), filter)
	if err != nil {
		h.logger.Error("Failed to get sales summary", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate sales summary"})
		return
	}

	c.JSON(http.StatusOK, summary)
}

// GetTopSellingItems
// GET /reports/top-items?start_date=2025-02-01&end_date=2025-02-28&period=daily&limit=10
func (h *ReportHandler) GetTopItems(c *gin.Context) {
	filter, ok := h.parseReportFilter(c)
	if !ok {
		return
	}

	limitStr := c.Query("limit")
	limit := 10
	if limitStr != "" {
		l, err := strconv.Atoi(limitStr)
		if err == nil && l > 0 && l <= 50 {
			limit = l
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "limit must be between 1 and 50"})
			return
		}
	}

	items, err := h.service.GetTopSellingItems(c.Request.Context(), filter, limit)
	if err != nil {
		h.logger.Error("Failed to get top selling items", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get top items"})
		return
	}

	c.JSON(http.StatusOK, items)
}

// GetInventoryStatus
// GET /reports/inventory-status
func (h *ReportHandler) GetInventoryStatus(c *gin.Context) {
	items, err := h.service.GetInventoryStatus(c.Request.Context())
	if err != nil {
		h.logger.Error("Failed to get inventory status", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get inventory status"})
		return
	}

	c.JSON(http.StatusOK, items)
}

// GetPaymentBreakdown
// GET /reports/payment-breakdown?start_date=2025-01-01&end_date=2025-12-31&period=monthly
func (h *ReportHandler) GetPaymentBreakdown(c *gin.Context) {
	filter, ok := h.parseReportFilter(c)
	if !ok {
		return
	}

	breakdown, err := h.service.GetPaymentMethodBreakdown(c.Request.Context(), filter)
	if err != nil {
		h.logger.Error("Failed to get payment breakdown", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get payment breakdown"})
		return
	}

	c.JSON(http.StatusOK, breakdown)
}

// GetRefundSummary
// GET /reports/refund-summary?start_date=2025-01-01&end_date=2025-12-31&period=monthly
func (h *ReportHandler) GetRefundSummary(c *gin.Context) {
	filter, ok := h.parseReportFilter(c)
	if !ok {
		return
	}

	summary, err := h.service.GetRefundSummary(c.Request.Context(), filter)
	if err != nil {
		h.logger.Error("Failed to get refund summary", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get refund summary"})
		return
	}

	c.JSON(http.StatusOK, summary)
}
