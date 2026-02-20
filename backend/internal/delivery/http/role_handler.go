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

type RoleHandler struct {
	service services.RoleService
	logger  *zap.Logger
}

func NewRoleHandler(service services.RoleService, logger *zap.Logger) *RoleHandler {
	return &RoleHandler{service, logger}
}

func (h *RoleHandler) Create(c *gin.Context) {
	var req entities.CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid create role request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	roleDTO, err := h.service.CreateRole(c.Request.Context(), &req)
	if err != nil {
		if errors.Is(err, services.ErrRoleAlreadyExists) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		h.logger.Error("Create role failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create role"})
		return
	}

	c.JSON(http.StatusCreated, roleDTO)
}

func (h *RoleHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.Query("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.Query("limit"))
	if limit < 1 {
		limit = 20
	}
	search := c.Query("search")

	roles, total, err := h.service.ListRoles(c.Request.Context(), page, limit, search)
	if err != nil {
		h.logger.Error("List roles failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list roles"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  roles,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

func (h *RoleHandler) Get(c *gin.Context) {
	id := c.Param("id")
	roleDTO, err := h.service.GetRole(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, services.ErrRoleNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "role not found"})
			return
		}
		h.logger.Error("Get role failed", zap.String("id", id), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusOK, roleDTO)
}

func (h *RoleHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req entities.UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	roleDTO, err := h.service.UpdateRole(c.Request.Context(), id, &req)
	if err != nil {
		if errors.Is(err, services.ErrRoleNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "role not found"})
			return
		}
		if errors.Is(err, services.ErrRoleAlreadyExists) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		h.logger.Error("Update role failed", zap.String("id", id), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update role"})
		return
	}

	c.JSON(http.StatusOK, roleDTO)
}

func (h *RoleHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.DeleteRole(c.Request.Context(), id); err != nil {
		if errors.Is(err, services.ErrRoleNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "role not found"})
			return
		}
		if errors.Is(err, services.ErrCannotDeleteAdmin) {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		h.logger.Error("Delete role failed", zap.String("id", id), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete role"})
		return
	}

	c.Status(http.StatusNoContent)
}
