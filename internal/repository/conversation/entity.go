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
	ConversationID uuid.UUID `gorm:"column:conversation_id;primaryKey;type:char(36);not null"`

	Type          string `gorm:"type:varchar(10)"`                        //should this be an enum
	SaliencyScore uint8  `gorm:"column:saliency_score;type:varchar(255)"` // Ever growing saliency score MRU
	Content       string `gorm:"type:varchar(255)"`
	// Embeddings
	EmbeddingRef dbtypes.XVector `gorm:"type:vector(768)"`

	CreatedAt time.Time      `gorm:"autoCreateTime(3)"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime(3)"`
	DeletedAt gorm.DeletedAt `gorm:"index"` // For soft delete
}

func (me *MemoryEntity) FromDomain(m types.Memory, embeddings []float32) {
	me.ID = m.ID
	me.Content = m.Content
	me.CreatedAt = m.CreatedAt
	me.UpdatedAt = m.UpdatedAt
	me.Type = string(m.Type)
	me.SaliencyScore = m.SaliencyScore
	me.ConversationID = m.ConversationID

	// embeddings
	xembeddings := dbtypes.XVector(embeddings)
	me.EmbeddingRef = xembeddings
}

func (me *MemoryEntity) ToDomain() *types.Memory {
	return &types.Memory{
		ID:             me.ID,
		ConversationID: me.ConversationID,
		Type:           types.MemoryType(me.Type),
		SaliencyScore:  me.SaliencyScore,
		Content:        me.Content,
		// no need for embeddings
		CreatedAt: me.CreatedAt,
		UpdatedAt: me.UpdatedAt,
	}
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
