// delivery/http/middleware/rbac.go

package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

func RequirePermission(requiredPerm string, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get claims from JWT (already validated by JWT middleware)
		claims, exists := c.Get("claims")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "claims not found"})
			c.Abort()
			return
		}

		permList, ok := claims.(jwt.MapClaims)["permissions"].([]interface{})
		if !ok {
			logger.Warn("No permissions found in JWT claims")
			c.JSON(http.StatusForbidden, gin.H{"error": "access denied - no permissions"})
			c.Abort()
			return
		}

		hasPerm := false
		for _, p := range permList {
			if str, ok := p.(string); ok && str == requiredPerm {
				hasPerm = true
				break
			}
		}

		if !hasPerm {
			logger.Warn("Permission denied",
				zap.String("required", requiredPerm),
				zap.Any("user_claims", claims))
			c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
			c.Abort()
			return
		}

		c.Next()
	}
}
