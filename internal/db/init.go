package db

import (
	"fmt"
	"time"

	"github.com/xpanvictor/xarvis/internal/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func InitDB(dsn string, cfg config.Settings) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		// Configure GORM with the provided settings
	})
	if err != nil {
		// Handle error
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	// configure db
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(cfg.DB.PoolSize)
	sqlDB.SetConnMaxLifetime(time.Hour)

	return db, nil
}
