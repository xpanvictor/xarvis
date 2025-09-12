package conversation

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/xpanvictor/xarvis/internal/database/dbtypes"
	"github.com/xpanvictor/xarvis/internal/types"
	"github.com/xpanvictor/xarvis/pkg/assistant"
	"gorm.io/gorm"
)

type MemoryEntity struct {
	ID             uuid.UUID `gorm:"primaryKey;type:char(36);not null"`
	ConversationID uuid.UUID `gorm:"column:conversation_id;type:char(36);not null"`

	Type          string `gorm:"type:varchar(10)"`      // should this be an enum
	SaliencyScore uint8  `gorm:"column:saliency_score"` // Ever growing saliency score MRU
	Content       string `gorm:"type:text"`             // Full content, increased from varchar(255)

	CreatedAt time.Time      `gorm:"autoCreateTime(3)"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime(3)"`
	DeletedAt gorm.DeletedAt `gorm:"index"` // For soft delete

	// Relationship to chunks
	Chunks []MemoryChunkEntity `gorm:"foreignKey:MemoryID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

// MemoryChunkEntity represents a chunk of memory content with its embedding
type MemoryChunkEntity struct {
	ID       uuid.UUID `gorm:"primaryKey;type:char(36);not null"`
	MemoryID uuid.UUID `gorm:"column:memory_id;type:char(36);not null"`

	ChunkIndex   int             `gorm:"column:chunk_index"` // Order of chunk within the memory
	ChunkContent string          `gorm:"type:text"`          // The chunked content
	EmbeddingRef dbtypes.XVector `gorm:"type:vector(768)"`   // Embedding for this chunk

	CreatedAt time.Time      `gorm:"autoCreateTime(3)"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime(3)"`
	DeletedAt gorm.DeletedAt `gorm:"index"` // For soft delete

	// Back reference to memory
	Memory MemoryEntity `gorm:"foreignKey:MemoryID;references:ID"`
}

func (me *MemoryEntity) FromDomain(m types.Memory) {
	me.ID = m.ID
	me.Content = m.Content
	me.CreatedAt = m.CreatedAt
	me.UpdatedAt = m.UpdatedAt
	me.Type = string(m.Type)
	me.SaliencyScore = m.SaliencyScore
	me.ConversationID = m.ConversationID
	// Chunks will be handled separately
}

func (me *MemoryEntity) ToDomain() *types.Memory {
	return &types.Memory{
		ID:             me.ID,
		ConversationID: me.ConversationID,
		Type:           types.MemoryType(me.Type),
		SaliencyScore:  me.SaliencyScore,
		Content:        me.Content,
		CreatedAt:      me.CreatedAt,
		UpdatedAt:      me.UpdatedAt,
	}
}

func (mce *MemoryChunkEntity) FromChunk(memoryID uuid.UUID, chunkIndex int, content string, embedding dbtypes.XVector) {
	mce.ID = uuid.New()
	mce.MemoryID = memoryID
	mce.ChunkIndex = chunkIndex
	mce.ChunkContent = content
	mce.EmbeddingRef = embedding
	mce.CreatedAt = time.Now()
	mce.UpdatedAt = time.Now()
}

type ConversationEntity struct {
	ID        uuid.UUID      `gorm:"primaryKey;type:char(36);not null"`
	OwnerID   uuid.UUID      `gorm:"column:owner_id;type:char(36);not null"` //foreign key?
	CreatedAt time.Time      `gorm:"autoCreateTime(3)"`
	UpdatedAt time.Time      `gorm:"autoCreateTime(3)"`
	DeletedAt gorm.DeletedAt `gorm:"index"` // For soft delete
	Summary   string         `gorm:"type:varchar(255)"`
	// Relationships
	Messages []MessageEntity `json:"messages" gorm:"-"` // - won't be persisted to db, ignored by GORM
	Memories []MemoryEntity  `gorm:"-"`                 // handled separately, not as association
}

func (c *ConversationEntity) ToDomain() types.Conversation {
	var msgs []types.Message
	for _, m := range c.Messages {
		msgs = append(msgs, *m.ToDomain())
	}
	var mems []types.Memory
	for _, m := range c.Memories {
		mems = append(mems, *m.ToDomain())
	}
	return types.Conversation{
		ID:        c.ID,
		OwnerID:   c.OwnerID,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
		Summary:   c.Summary,
		Messages:  msgs,
		Memories:  mems,
	}
}

type MessageEntity struct {
	ID             uuid.UUID `json:"id"`
	UserID         uuid.UUID `json:"user_id"`
	ConversationID uuid.UUID `json:"conversation_id"`
	Text           string    `json:"text"`
	Tags           []string  `json:"tags" gorm:"-"` // Not persisted to DB, stored in Redis
	Timestamp      time.Time `json:"timestamp"`
	// use diff enum
	MsgRole assistant.Role `json:"msg_role"`
}

func (me *MessageEntity) Key() string {
	return fmt.Sprintf("msg:%s", me.ID.String())
}

func (ce *ConversationEntity) ConvoKey(k uuid.UUID) string {
	return fmt.Sprintf("conv:%s", ce.ID.String())
}

func (me *MessageEntity) ToDomain() *types.Message {
	return &types.Message{
		Id:             me.ID,
		UserId:         me.UserID,
		ConversationID: me.ConversationID,
		Text:           me.Text,
		Tags:           me.Tags,
		Timestamp:      me.Timestamp,
		// should be mapped
		MsgRole: me.MsgRole,
	}
}

func (me *MessageEntity) FromDomain(msg *types.Message) {
	me.ID = msg.Id
	me.ConversationID = msg.ConversationID
	me.Tags = msg.Tags
	me.Text = msg.Text
	me.UserID = msg.UserId
	me.MsgRole = msg.MsgRole
}
