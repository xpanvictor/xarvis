package database

import (
	"context"

	"github.com/xpanvictor/xarvis/internal/repository/conversation"
	noteRepo "github.com/xpanvictor/xarvis/internal/repository/note"
	projectRepo "github.com/xpanvictor/xarvis/internal/repository/project"
	userRepo "github.com/xpanvictor/xarvis/internal/repository/user"
	"gorm.io/gorm"
)

func MigrateDB(db *gorm.DB) {
	db.AutoMigrate(
		&userRepo.UserEntity{},
		&conversation.ConversationEntity{},
		&conversation.MemoryEntity{},
		&conversation.MemoryChunkEntity{},
		&projectRepo.ProjectEntity{},
		&noteRepo.NoteEntity{},
	)

	// Create vector index with TiDB-specific syntax for columnar replica
	if err := db.Exec(`
	CREATE VECTOR INDEX idx_mem_chunk_embedding
	ON memory_chunk_entities ((VEC_COSINE_DISTANCE(embedding_ref)))
	USING HNSW
	ADD_COLUMNAR_REPLICA_ON_DEMAND;
	`).Error; err != nil {
		// Ignore "already exists" error since that's expected on subsequent runs
		if err.Error() != "Error 1061 (42000): vector index 'idx_mem_chunk_embedding' with VEC_COSINE_DISTANCE already exist on column embedding_ref" {
			db.Logger.Error(context.TODO(), "Failed to create vector index: %v", err)
		}
	}
}
