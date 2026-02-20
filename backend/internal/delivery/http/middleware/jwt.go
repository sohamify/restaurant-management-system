package middleware

import (
	"errors"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

func JWTAuth(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			logger.Warn("Missing Authorization header")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization header required"})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			logger.Warn("Invalid Authorization header format", zap.String("header", authHeader))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header format – use Bearer <token>"})
			c.Abort()
			return
		}

		tokenStr := parts[1]

		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			secret := os.Getenv("JWT_SECRET")
			if secret == "" {
				logger.Error("JWT_SECRET environment variable is empty")
				return nil, errors.New("server configuration error")
			}
			return []byte(secret), nil
		})

		if err != nil {
			logger.Warn("JWT parse error", zap.Error(err), zap.String("token_prefix", tokenStr[:20]+"..."))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			c.Abort()
			return
		}

		if !token.Valid {
			logger.Warn("JWT is not valid", zap.Any("claims", token.Claims))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "token is not valid"})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			logger.Error("JWT claims are not MapClaims")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid claims format"})
			c.Abort()
			return
		}

		// Store everything useful
		c.Set("claims", claims)
		c.Set("user_id", claims["user_id"])
		c.Set("role", claims["role"])

		logger.Debug("JWT authenticated successfully",
			zap.String("user_id", claims["user_id"].(string)),
			zap.String("role", claims["role"].(string)),
		)

		c.Next()
	}
}
