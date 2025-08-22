package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xpanvictor/xarvis/internal/config"
	"github.com/xpanvictor/xarvis/internal/database"
	"github.com/xpanvictor/xarvis/internal/repository"
	"github.com/xpanvictor/xarvis/internal/server"
	"github.com/xpanvictor/xarvis/pkg/Logger"
)

// This is the main entry point for the API server.
// Loads in all system components
// Exposes functionalities
func main() {
	// fetch cfg
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	// load global logger
	logger := Logger.New(cfg.Debug)
	logger.Info("Logger initialized")
	// fetch database connection
	db, err := database.InitDB(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	// handle migrations
	database.MigrateDB(db)
	// compose router
	router := gin.Default()
	dep := server.NewServerDependencies(
		repository.NewGormConversationRepo(db),
	)
	server.InitializeRoutes(router, dep)

	// listen with graceful exist
	srv := &http.Server{
		Addr:    ":0",
		Handler: router.Handler(),
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Server existing %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// 5 secs then cancel
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err != srv.Shutdown(ctx) {
		logger.Errorf("Shutdown err &v", err)
	}
	logger.Info("Shutdown system")
}
