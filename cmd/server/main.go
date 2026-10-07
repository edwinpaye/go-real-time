package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sales-system/go-real-time/config"
	httpAdapter "github.com/sales-system/go-real-time/internal/adapters/primary/http"
	wsAdapter "github.com/sales-system/go-real-time/internal/adapters/primary/websocket"
	"github.com/sales-system/go-real-time/internal/adapters/secondary/eventbus"
	"github.com/sales-system/go-real-time/internal/adapters/secondary/postgres"
	"github.com/sales-system/go-real-time/internal/adapters/secondary/security"
	"github.com/sales-system/go-real-time/internal/core/services"
	"github.com/sales-system/go-real-time/internal/infra/logger"
	"github.com/sales-system/go-real-time/internal/infra/workerpool"
)

func main() {
	// 1. Configuration
	cfg := config.LoadConfig()

	// 2. Logging Setup
	appLogger := logger.New(os.Stdout, cfg.AppEnv)
	logger.SetDefault(appLogger)

	logger.Info("Starting Enterprise Real-Time Sales Server...", logger.Fields{
		"port":       cfg.Port,
		"env":        cfg.AppEnv,
		"static_dir": cfg.StaticPath,
	})

	// 3. Background Worker Pool
	workerPool := workerpool.New(cfg.WorkerPoolSize, cfg.WorkerQueueCap)
	defer workerPool.Shutdown(5 * time.Second)

	// 4. Persistence & Database Initialization
	db, err := postgres.NewDB(cfg)
	if err != nil {
		logger.Error("Database connection initialization failed", logger.Fields{"error": err.Error()})
	}
	defer func() {
		if db != nil && db.DB != nil {
			_ = db.Close()
		}
	}()

	// Repositories
	userRepo := postgres.NewUserRepository(db)
	productRepo := postgres.NewProductRepository(db)
	orderRepo := postgres.NewOrderRepository(db)
	auditRepo := postgres.NewAuditRepository(db)

	// 5. Security Adapters
	tokenService := security.NewJWTService(cfg.JWTSecret)
	hasher := security.NewBcryptHasher(10)

	// 6. Real-time Event Bus
	realtimeBus := eventbus.NewInMemoryEventBus(4096)

	// 7. Core Application Services (Hexagonal Use Cases)
	auditService := services.NewAuditService(auditRepo, workerPool, realtimeBus)
	authService := services.NewAuthService(userRepo, tokenService, hasher, auditService, realtimeBus, cfg.JWTExpiryHours)
	productService := services.NewProductService(productRepo, auditService, realtimeBus)
	orderService := services.NewOrderService(db, orderRepo, productRepo, auditService, realtimeBus)

	// 8. WebSocket Hub & Handler
	wsHub := wsAdapter.NewHub(realtimeBus)
	wsHandler := wsAdapter.NewHandler(wsHub, tokenService, userRepo)

	// 9. HTTP Handlers
	authHandler := httpAdapter.NewAuthHandler(authService)
	productHandler := httpAdapter.NewProductHandler(productService)
	orderHandler := httpAdapter.NewOrderHandler(orderService)
	auditHandler := httpAdapter.NewAuditHandler(auditService)

	// 10. Router Configuration
	router := httpAdapter.NewRouter(httpAdapter.RouterConfig{
		AuthHandler:    authHandler,
		ProductHandler: productHandler,
		OrderHandler:   orderHandler,
		AuditHandler:   auditHandler,
		WSHandler:      wsHandler,
		TokenService:   tokenService,
		Config:         cfg,
	})

	// 11. HTTP Server
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 12. Graceful Shutdown Listener
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		logger.Info("HTTP and WebSocket server listening", logger.Fields{
			"address": server.Addr,
			"url":     "http://localhost:" + cfg.Port + cfg.StaticPrefix,
		})
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP Server fatal error", logger.Fields{"error": err.Error()})
			os.Exit(1)
		}
	}()

	// Block until signal received
	<-stopChan
	logger.Info("Shutting down server gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("HTTP server forced shutdown error", logger.Fields{"error": err.Error()})
	}

	logger.Info("Server stopped successfully")
}
