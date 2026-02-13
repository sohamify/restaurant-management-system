// internal/delivery/http/user_handler.go
package http

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/sohamify/rms-backend/internal/application/services"
	"github.com/sohamify/rms-backend/internal/domain/entities"
)

type UserHandler struct {
	service services.UserService
	logger  *zap.Logger
}

func NewUserHandler(service services.UserService, logger *zap.Logger) *UserHandler {
	return &UserHandler{service, logger}
}

// CreateUser godoc
// @Summary Create a new user/staff
// @Description Admin creates a new staff account
// @Tags Users
// @Accept json
// @Produce json
// @Param body body entities.CreateUserRequest true "User data"
// @Security BearerAuth
// @Success 201 {object} entities.UserDTO
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /users [post]
func (h *UserHandler) Create(c *gin.Context) {
	var req entities.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid create user request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	createdBy := c.GetString("user_id")
	userDTO, err := h.service.CreateUser(c.Request.Context(), &req, createdBy)
	if err != nil {
		if errors.Is(err, services.ErrEmailAlreadyExists) || errors.Is(err, services.ErrInvalidRole) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		h.logger.Error("Create user failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
		return
	}

	c.JSON(http.StatusCreated, userDTO)
}

// ListUsers godoc
// @Summary List users with pagination and filters
// @Description Get paginated list of active users (admin only)
// @Tags Users
// @Produce json
// @Param page query int false "Page number (default 1)"
// @Param limit query int false "Items per page (default 20)"
// @Param role query string false "Filter by role ID"
// @Param search query string false "Search by name or email"
// @Param is_active query bool false "Filter by active status"
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Failure 400,401,403,500 {object} map[string]string
// @Router /users [get]
func (h *UserHandler) List(c *gin.Context) {
	page := 1
	if p := c.Query("page"); p != "" {
		// simple parse - in production use strconv
		page = 1 // add proper parsing
	}
	limit := 20
	if l := c.Query("limit"); l != "" {
		limit = 20 // add parsing
	}

	role := c.Query("role")
	search := c.Query("search")
	var isActive *bool
	if act := c.Query("is_active"); act != "" {
		b := act == "true"
		isActive = &b
	}

	users, total, err := h.service.ListUsers(c.Request.Context(), page, limit, role, search, isActive)
	if err != nil {
		h.logger.Error("List users failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list users"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  users,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// GetUser godoc
// @Summary Get user by ID
// @Tags Users
// @Produce json
// @Param id path string true "User ID"
// @Security BearerAuth
// @Success 200 {object} entities.UserDTO
// @Failure 404,500 {object} map[string]string
// @Router /users/{id} [get]
func (h *UserHandler) Get(c *gin.Context) {
	id := c.Param("id")
	userDTO, err := h.service.GetUser(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.JSON(http.StatusOK, userDTO)
}

// UpdateUser godoc
// @Summary Update user details
// @Tags Users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Param body body entities.UpdateUserRequest true "Update fields"
// @Security BearerAuth
// @Success 200 {object} entities.UserDTO
// @Failure 400,403,404,500 {object} map[string]string
// @Router /users/{id} [patch]
func (h *UserHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req entities.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updatedBy := c.GetString("user_id")
	userDTO, err := h.service.UpdateUser(c.Request.Context(), id, &req, updatedBy)
	if err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		if errors.Is(err, services.ErrInvalidRole) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user"})
		return
	}

	c.JSON(http.StatusOK, userDTO)
}

// DeleteUser godoc
// @Summary Soft delete a user
// @Tags Users
// @Produce json
// @Param id path string true "User ID"
// @Security BearerAuth
// @Success 204
// @Failure 400,403,404,500 {object} map[string]string
// @Router /users/{id} [delete]
func (h *UserHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	deletedBy := c.GetString("user_id")

	if err := h.service.DeleteUser(c.Request.Context(), id, deletedBy); err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		if errors.Is(err, services.ErrCannotDeleteSelf) || errors.Is(err, services.ErrCannotDeleteLastAdmin) {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete user"})
		return
	}

	c.Status(http.StatusNoContent)
}

// ChangePassword godoc
// @Summary Change password (admin reset or self)
// @Description Admin can reset any user password without old one; self-change requires old password
// @Tags Users
// @Accept json
// @Produce json
// @Param id path string true "User ID" (use 'me' for self)
// @Param body body entities.ChangePasswordRequest true "New password"
// @Security BearerAuth
// @Success 200 {object} map[string]string
// @Failure 400,403,404,500 {object} map[string]string
// @Router /users/{id}/password [patch]
func (h *UserHandler) ChangePassword(c *gin.Context) {
	id := c.Param("id")
	requesterID := c.GetString("user_id")
	isAdmin := c.GetString("role") == "ADMIN" // simplistic check – improve with permissions

	if id == "me" {
		id = requesterID
		isAdmin = false // force self mode
	}

	var req entities.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.service.ChangePassword(c.Request.Context(), id, &req, requesterID, isAdmin)
	if err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		if errors.Is(err, services.ErrOldPasswordRequired) || errors.Is(err, services.ErrInvalidOldPassword) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "password updated successfully"})
}
