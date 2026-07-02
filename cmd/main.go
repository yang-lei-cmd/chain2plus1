package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/linqi/chain2plus1/internal/config"
	"github.com/linqi/chain2plus1/internal/event"
	"github.com/linqi/chain2plus1/internal/router"
	"github.com/linqi/chain2plus1/internal/service"
	"github.com/linqi/chain2plus1/pkg/database"
	"github.com/linqi/chain2plus1/pkg/jwt"
	"github.com/linqi/chain2plus1/pkg/logger"
	"github.com/linqi/chain2plus1/pkg/seed"

	// Swagger docs
	_ "github.com/linqi/chain2plus1/docs"
)

// @title           Chain2Plus1 API
// @version         2.0.0
// @description     链动2+1分销系统 API — 支持用户体系、链动分润、提现、第三方支付、灵活用工
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.email  support@chain2plus1.com

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @host      localhost:8080
// @BasePath  /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

// @externalDocs.description  OpenAPI Specification
// @externalDocs.url          https://swagger.io/resources/open-api/
func main() {
	// Load config (validates env vars at startup)
	cfg := config.LoadConfig()

	// Initialize logger
	logger.InitLogger(cfg.Log.Level)
	logger.Info("Starting Chain2Plus1 API server...")

	// Connect database
	database.Connect(&cfg.Database)
	database.Migrate()

	// Seed basic data (suppliers + products)
	seed.SeedBasicData(database.DB)

	// Initialize JWT
	jwt.InitJWT(cfg.JWT.Secret, cfg.JWT.ExpireDays)

	// Phase 6: Initialize WebSocket Hub
	wsHub := event.NewHub()
	go wsHub.Run()

	// Phase 8: Initialize PaymentService for expired payment cleanup
	paymentService := service.NewPaymentService(cfg)

	// Start expired payment cleanup goroutine (every 5 minutes)
	go func() {
		log.Println("[INFO] Starting expired payment cleanup scheduler (every 5min)")
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			if err := paymentService.CancelExpiredPayments(); err != nil {
				log.Printf("[ERROR] Failed to cancel expired payments: %v", err)
			}
		}
	}()

	// Initialize WS auth function using JWT package
	wsHandler := event.NewWSHandler(wsHub)
	wsHandler.SetAuthFunc(func(tokenString string) (uint, string, string, error) {
		claims, err := jwt.ParseToken(tokenString)
		if err != nil {
			return 0, "", "", err
		}
		userID, _ := strconv.ParseUint(claims.UserID, 10, 64)
		return uint(userID), claims.Username, claims.Role, nil
	})

	// Setup router (pass config for payment handler and WS handler)
	engine := router.Setup(cfg, wsHandler)

	// Create http.Server
	addr := fmt.Sprintf("%s:%s", "127.0.0.1", cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      engine,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
	}

	// Start server in a goroutine
	go func() {
		logger.Info("Server listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Failed to start server: %v", err)
			panic(err)
		}
	}()

	// ============================================================
	// Graceful Shutdown — wait for SIGINT/SIGTERM
	// ============================================================
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	logger.Info("Received signal %v, shutting down gracefully...", sig)

	// Give outstanding requests 10 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown: %v", err)
	}

	logger.Info("Server stopped")
}
