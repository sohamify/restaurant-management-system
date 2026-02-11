package http

import (
	"net/http"

	"github.com/sohamify/rms-backend/internal/application/services"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type AuthHandler struct {
	authService services.AuthService
	logger      *zap.Logger
}

func NewAuthHandler(authService services.AuthService, logger *zap.Logger) *AuthHandler {
	return &AuthHandler{authService, logger}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=8"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid login request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	}

	userDTO, token, err := h.authService.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user":  userDTO,
		"token": token,
	})
}
