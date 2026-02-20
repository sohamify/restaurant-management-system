package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.uber.org/zap"

	"github.com/sohamify/rms-backend/internal/application/services"
	deliveryHttp "github.com/sohamify/rms-backend/internal/delivery/http"
	"github.com/sohamify/rms-backend/internal/delivery/http/middleware"
	"github.com/sohamify/rms-backend/internal/infrastructure/persistence/mongo"
)

func main() {
	// 1. Load environment variables
	if err := godotenv.Load(); err != nil {
		fmt.Println("No .env file found – using system environment variables")
	}

	// 2. Initialize structured logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer func() { _ = logger.Sync() }()

	// 3. Connect to MongoDB
	if err := mongo.Init(); err != nil {
		logger.Fatal("MongoDB initialization failed", zap.Error(err))
	}
	defer func() {
		if err := mongo.Disconnect(); err != nil {
			logger.Error("Error disconnecting MongoDB", zap.Error(err))
		}
	}()

	db := mongo.GetDatabase()
	logger.Info("Connected to database", zap.String("name", db.Name()))

	// Create all required indexes
	if err := mongo.CreateIndexes(logger); err != nil {
		logger.Fatal("Failed to create indexes", zap.Error(err))
	}

	// 4. Repositories (all core ones)
	userRepo := mongo.NewUserRepository()
	roleRepo := mongo.NewRoleRepository()
	// Add more repos here later (e.g. tableRepo, menuRepo, orderRepo...)

	// 5. Services
	authService := services.NewAuthService(userRepo, roleRepo, logger)
	userService := services.NewUserService(userRepo, roleRepo, logger)
	roleService := services.NewRoleService(roleRepo, userRepo, logger) // note: userRepo injected for delete checks

	// 6. Handlers
	authHandler := deliveryHttp.NewAuthHandler(authService, logger)
	userHandler := deliveryHttp.NewUserHandler(userService, logger)
	roleHandler := deliveryHttp.NewRoleHandler(roleService, logger)

	// 7. Gin router setup
	gin.SetMode(gin.ReleaseMode) // change to gin.DebugMode during development

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.Logger(logger)) // request logging middleware

	// Public health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"db":      db.Name(),
			"version": "0.1.0",
		})
	})

	// API v1 group
	api := r.Group("/api/v1")

	// ── Public routes ────────────────────────────────────────────────────────
	api.POST("/auth/login", authHandler.Login)

	// ── Protected routes (JWT required) ──────────────────────────────────────
	protected := api.Group("/")
	protected.Use(middleware.JWTAuth(logger))

	// Users management (admin only)
	users := protected.Group("/users")
	users.Use(middleware.RequirePermission("USER_MANAGE", logger)) // or finer-grained perms
	users.POST("", userHandler.Create)
	users.GET("", userHandler.List)
	users.GET("/:id", userHandler.Get)
	users.PATCH("/:id", userHandler.Update)
	users.DELETE("/:id", userHandler.Delete)
	users.PATCH("/:id/password", userHandler.ChangePassword) // admin reset
	users.PATCH("/me/password", userHandler.ChangePassword)  // self change

	// Roles management (admin only)
	roles := protected.Group("/roles")
	roles.Use(middleware.RequirePermission("ROLE_MANAGE", logger))
	roles.POST("", roleHandler.Create)
	roles.GET("", roleHandler.List)
	roles.GET("/:id", roleHandler.Get)
	roles.PATCH("/:id", roleHandler.Update)
	roles.DELETE("/:id", roleHandler.Delete)

	// Placeholder for future modules (expand later)
	protected.GET("/orders", middleware.RequirePermission("ORDER_VIEW", logger), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Orders endpoint – implementation coming soon",
		})
	})

	// 8. Graceful shutdown
	srv := &http.Server{
		Addr:    ":" + getPort(),
		Handler: r,
	}

	// Start server in background
	go func() {
		logger.Info("HTTP server starting", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("ListenAndServe failed", zap.Error(err))
		}
	}()

	// Wait for interrupt signal (Ctrl+C or SIGTERM)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutdown signal received...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced shutdown", zap.Error(err))
	}

	logger.Info("Server stopped gracefully")
}

func getPort() string {
	port := os.Getenv("PORT")
	if port == "" {
		return "8080"
	}
	return port
}
