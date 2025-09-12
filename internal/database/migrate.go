package database

import (
	"github.com/xpanvictor/xarvis/internal/repository/conversation"
	userRepo "github.com/xpanvictor/xarvis/internal/repository/user"
	"gorm.io/gorm"
)

func MigrateDB(db *gorm.DB) {
	db.AutoMigrate(
		&userRepo.UserEntity{},
		&conversation.ConversationEntity{},
		&conversation.MemoryEntity{},
	)

	db.Exec(`
	CREATE VECTOR INDEX idx_mem_embedding
	ON memory_entities ((VEC_COSINE_DISTANCE(embedding_ref)))
	USING HNSW;
	`)
}
