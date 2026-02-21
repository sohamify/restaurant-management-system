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

	// 4. Repositories
	userRepo := mongo.NewUserRepository()
	roleRepo := mongo.NewRoleRepository()
	tableRepo := mongo.NewTableRepository()

	// 5. Services
	authService := services.NewAuthService(userRepo, roleRepo, logger)
	userService := services.NewUserService(userRepo, roleRepo, logger)
	roleService := services.NewRoleService(roleRepo, userRepo, logger)
	tableService := services.NewTableService(tableRepo, logger)

	// 6. Handlers
	authHandler := deliveryHttp.NewAuthHandler(authService, logger)
	userHandler := deliveryHttp.NewUserHandler(userService, logger)
	roleHandler := deliveryHttp.NewRoleHandler(roleService, logger)
	tableHandler := deliveryHttp.NewTableHandler(tableService, logger)

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
	users.Use(middleware.RequirePermission("USER_MANAGE", logger))
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

	// Tables management (admin/manager only)
	tables := protected.Group("/tables")
	tables.Use(middleware.RequirePermission("TABLE_MANAGE", logger))
	tables.POST("", tableHandler.Create)
	tables.GET("", tableHandler.List)
	tables.GET("/:id", tableHandler.Get)
	tables.PATCH("/:id", tableHandler.Update)
	tables.DELETE("/:id", tableHandler.Delete)

	// Waiter-specific: only see assigned tables
	waiterTables := protected.Group("/waiter/tables")
	waiterTables.Use(middleware.RequirePermission("TABLE_READ_ASSIGNED", logger))
	waiterTables.GET("", tableHandler.ListAssigned)

	// Menu Items
	menuItemRepo := mongo.NewMenuItemRepository()

	// Menu Categories
	menuCategoryRepo := mongo.NewMenuCategoryRepository()
	menuCategoryService := services.NewMenuCategoryService(menuCategoryRepo, menuItemRepo, logger) // menuItemRepo for delete check
	menuCategoryHandler := deliveryHttp.NewMenuCategoryHandler(menuCategoryService, logger)

	// Routes
	menuCategories := protected.Group("/menu/categories")
	menuCategories.Use(middleware.RequirePermission("MENU_CATEGORY_MANAGE", logger))
	menuCategories.POST("", menuCategoryHandler.Create)
	menuCategories.GET("", menuCategoryHandler.List)
	menuCategories.GET("/:id", menuCategoryHandler.Get)
	menuCategories.PATCH("/:id", menuCategoryHandler.Update)
	menuCategories.DELETE("/:id", menuCategoryHandler.Delete)

	// Menu Items
	menuItemService := services.NewMenuItemService(menuItemRepo, menuCategoryRepo, logger)
	menuItemHandler := deliveryHttp.NewMenuItemHandler(menuItemService, logger)

	// Routes
	menuItems := protected.Group("/menu/items")
	menuItems.Use(middleware.RequirePermission("MENU_ITEM_MANAGE", logger))
	menuItems.POST("", menuItemHandler.Create)
	menuItems.GET("", menuItemHandler.List)
	menuItems.GET("/:id", menuItemHandler.Get)
	menuItems.PATCH("/:id", menuItemHandler.Update)
	menuItems.DELETE("/:id", menuItemHandler.Delete)

	// Orders
	orderRepo := mongo.NewOrderRepository()

	// Bills
	billRepo := mongo.NewBillRepository()
	billService := services.NewBillService(billRepo, orderRepo, logger)
	billHandler := deliveryHttp.NewBillHandler(billService, logger)
	orderService := services.NewOrderService(orderRepo, tableRepo, menuItemRepo, billService, logger)
	orderHandler := deliveryHttp.NewOrderHandler(orderService, logger)

	// Routes
	orders := protected.Group("/orders")
	orders.Use(middleware.RequirePermission("ORDER_CREATE", logger)) // waiter
	orders.POST("", orderHandler.Create)

	orderByID := orders.Group("/:id")
	orderByID.Use(middleware.RequirePermission("ORDER_UPDATE_OWN", logger))
	orderByID.PATCH("/items", orderHandler.AddItem)

	// Inventory
	inventoryRepo := mongo.NewInventoryRepository()
	recipeRepo := mongo.NewRecipeRepository()
	inventoryService := services.NewInventoryService(inventoryRepo, recipeRepo, logger)
	inventoryHandler := deliveryHttp.NewInventoryHandler(inventoryService, logger)
	recipeService := services.NewRecipeService(recipeRepo, menuItemRepo, logger)
	recipeHandler := deliveryHttp.NewRecipeHandler(recipeService, logger)

	// Routes for inventory
	inventory := protected.Group("/inventory")
	inventory.Use(middleware.RequirePermission("INVENTORY_MANAGE", logger))
	inventory.POST("", inventoryHandler.Create)
	inventory.GET("", inventoryHandler.List)
	inventory.GET("/:id", inventoryHandler.Get)
	inventory.PATCH("/:id", inventoryHandler.Update)
	inventory.DELETE("/:id", inventoryHandler.Delete)
	inventory.GET("/report", inventoryHandler.GetReport)

	// Routes for recipes (per menu item)
	recipes := protected.Group("/recipes")
	recipes.Use(middleware.RequirePermission("MENU_ITEM_MANAGE", logger))
	recipes.POST("", recipeHandler.Create)
	recipes.GET("/:menu_item_id", recipeHandler.Get)
	recipes.PATCH("/:menu_item_id", recipeHandler.Update)
	recipes.DELETE("/:menu_item_id", recipeHandler.Delete)

	// Payments
	paymentRepo := mongo.NewPaymentRepository()
	paymentService := services.NewPaymentService(paymentRepo, orderRepo, tableRepo, menuItemRepo, logger)
	paymentHandler := deliveryHttp.NewPaymentHandler(paymentService, logger)

	// Routes - Cashier only
	payments := protected.Group("/payments")
	payments.Use(middleware.RequirePermission("PAYMENT_PROCESS_FULL", logger))
	payments.POST("", paymentHandler.ProcessPayment)
	payments.GET("/order/:order_id", paymentHandler.GetPayments)

	// Refund (cashier/manager)
	refunds := protected.Group("/refunds")
	refunds.Use(middleware.RequirePermission("REFUND_PROCESS", logger))
	refunds.POST("/order/:order_id", paymentHandler.ProcessRefund)

	// Routes - Cashier/Manager only
	bills := protected.Group("/bills")
	bills.Use(middleware.RequirePermission("BILL_GENERATE", logger))
	bills.POST("", billHandler.GenerateBill)
	bills.GET("/order/:order_id", billHandler.GetBill)

	// 8. Graceful shutdown
	srv := &http.Server{
		Addr:    ":" + getPort(),
		Handler: r,
	}

	go func() {
		logger.Info("HTTP server starting", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("ListenAndServe failed", zap.Error(err))
		}
	}()

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
