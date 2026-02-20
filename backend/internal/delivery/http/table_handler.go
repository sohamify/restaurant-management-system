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

type TableHandler struct {
	service services.TableService
	logger  *zap.Logger
}

func NewTableHandler(service services.TableService, logger *zap.Logger) *TableHandler {
	return &TableHandler{service, logger}
}

func (h *TableHandler) Create(c *gin.Context) {
	var req entities.CreateTableRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid create table request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tableDTO, err := h.service.CreateTable(c.Request.Context(), &req)
	if err != nil {
		if errors.Is(err, services.ErrTableNumberExists) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		h.logger.Error("Create table failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create table"})
		return
	}

	c.JSON(http.StatusCreated, tableDTO)
}

func (h *TableHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.Query("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.Query("limit"))
	if limit < 1 {
		limit = 20
	}
	status := c.Query("status")

	tables, total, err := h.service.ListTables(c.Request.Context(), page, limit, status)
	if err != nil {
		h.logger.Error("List tables failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list tables"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  tables,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

func (h *TableHandler) Get(c *gin.Context) {
	id := c.Param("id")
	tableDTO, err := h.service.GetTable(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, services.ErrTableNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "table not found"})
			return
		}
		h.logger.Error("Get table failed", zap.String("id", id), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusOK, tableDTO)
}

func (h *TableHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req entities.UpdateTableRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tableDTO, err := h.service.UpdateTable(c.Request.Context(), id, &req)
	if err != nil {
		if errors.Is(err, services.ErrTableNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "table not found"})
			return
		}
		if errors.Is(err, services.ErrInvalidTableStatus) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		h.logger.Error("Update table failed", zap.String("id", id), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update table"})
		return
	}

	c.JSON(http.StatusOK, tableDTO)
}

func (h *TableHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.DeleteTable(c.Request.Context(), id); err != nil {
		if errors.Is(err, services.ErrTableNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "table not found"})
			return
		}
		if errors.Is(err, services.ErrCannotDeleteActive) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		h.logger.Error("Delete table failed", zap.String("id", id), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete table"})
		return
	}

	c.Status(http.StatusNoContent)
}

// Waiter-specific endpoint (only sees assigned tables)
func (h *TableHandler) ListAssigned(c *gin.Context) {
	waiterID := c.GetString("user_id") // from JWT claims

	tables, err := h.service.ListAssignedTables(c.Request.Context(), waiterID)
	if err != nil {
		h.logger.Error("Failed to list assigned tables", zap.String("waiter_id", waiterID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list assigned tables"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": tables,
	})
}
