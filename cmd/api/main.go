package main

import (
	"log"

	"github.com/xpanvictor/xarvis/internal/config"
	"github.com/xpanvictor/xarvis/internal/database"
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

}
