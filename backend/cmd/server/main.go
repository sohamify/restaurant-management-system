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

	// Create indexes
	if err := mongo.CreateIndexes(logger); err != nil {
		logger.Fatal("Failed to create indexes", zap.Error(err))
	}

	// 4. Repositories
	userRepo := mongo.NewUserRepository()
	roleRepo := mongo.NewRoleRepository()

	// 5. Services
	authService := services.NewAuthService(userRepo, roleRepo, logger)

	// 6. Handlers
	authHandler := deliveryHttp.NewAuthHandler(authService, logger)

	// 7. Gin router
	gin.SetMode(gin.ReleaseMode) // change to gin.DebugMode during dev if needed

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.Logger(logger)) // request logging

	// Public health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"db":      db.Name(),
			"version": "0.1.0",
		})
	})

	// API routes
	api := r.Group("/api/v1")

	// Public auth
	api.POST("/auth/login", authHandler.Login)

	// Protected routes (example)
	protected := api.Group("/")
	protected.Use(middleware.JWTAuth(logger))

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

	// Wait for shutdown signal
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
