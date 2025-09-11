package database

import (
	"github.com/xpanvictor/xarvis/internal/repository"
	userRepo "github.com/xpanvictor/xarvis/internal/repository/user"
	"gorm.io/gorm"
)

func MigrateDB(db *gorm.DB) {
	db.AutoMigrate(
		&userRepo.UserEntity{},
		repository.GormConvoRepo{},
	)
}
