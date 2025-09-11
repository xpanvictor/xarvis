package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xpanvictor/xarvis/internal/app"
	"github.com/xpanvictor/xarvis/internal/config"
	"github.com/xpanvictor/xarvis/internal/database"
	"github.com/xpanvictor/xarvis/internal/server"
	"github.com/xpanvictor/xarvis/pkg/Logger"

	_ "github.com/xpanvictor/xarvis/docs" // Import generated docs
)

// @title           Xarvis API
// @version         1.0
// @description     Xarvis AI Assistant API with user management and WebSocket communication
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.email  xpanvictor@gmail.com

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8088
// @BasePath  /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

// This is the main entry point for the API server.
// Loads in all system components
// Exposes functionalities
func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize logger
	logger := Logger.New(cfg.Debug)
	logger.Info("Logger initialized")

	// Initialize database
	db, err := database.InitDB(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	database.MigrateDB(db)

	// Create application with all dependencies
	application, err := app.NewApp(cfg, logger, db)
	if err != nil {
		log.Fatalf("Failed to initialize application: %v", err)
	}

	// Set up HTTP server
	router := gin.Default()
	server.InitializeRoutes(cfg, router, application.GetServerDependencies())

	logger.Info("Application initialized successfully")

	// Start server with graceful shutdown
	startServer(router, logger)
}

func startServer(router *gin.Engine, logger *Logger.Logger) {
	// Get port from environment or use default
	port := 8088
	if p := os.Getenv("PORT"); p != "" {
		if v, err := strconv.Atoi(p); err == nil {
			port = v
		}
	}

	addr := ":" + strconv.Itoa(port)
	srv := &http.Server{
		Addr:    addr,
		Handler: router.Handler(),
	}

	// Start server in goroutine
	go func() {
		logger.Infof("Server starting on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Give outstanding requests 5 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Errorf("Server forced to shutdown: %v", err)
	} else {
		logger.Info("Server shutdown complete")
	}
}
