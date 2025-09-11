package conversation

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/xpanvictor/xarvis/internal/database/dbtypes"
	"github.com/xpanvictor/xarvis/internal/domains/conversation"
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
	EmbeddingRef dbtypes.XVector `gorm:"type:varchar(255)"`

	CreatedAt time.Time      `gorm:"autoCreateTime(3)"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime(3)"`
	DeletedAt gorm.DeletedAt `gorm:"index"` // For soft delete
}

type ConversationEntity struct {
	ID        uuid.UUID      `gorm:"primaryKey;type:char(36);not null"`
	OwnerID   uuid.UUID      `gorm:"column:owner_id;type:char(36);not null"` //foreign key?
	CreatedAt time.Time      `gorm:"autoCreateTime(3)"`
	UpdatedAt time.Time      `gorm:"autoCreateTime(3)"`
	DeletedAt gorm.DeletedAt `gorm:"index"` // For soft delete
	Summary   string         `gorm:"type:varchar(255)"`
	// Relationships
	Messages []MessageEntity `json:"messages"` // - won't be persisted to db
	Memories []MemoryEntity  `gorm:"type:json"`
}

type MessageEntity struct {
	ID             uuid.UUID `json:"id"`
	UserID         uuid.UUID `json:"user_id"`
	ConversationID uuid.UUID `json:"conversation_id"`
	Text           string    `json:"text"`
	Tags           []string  `json:"tags"`
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

func (me *MessageEntity) ToDomain() *conversation.Message {
	return &conversation.Message{
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

func (me *MessageEntity) FromDomain(msg *conversation.Message) {
	me.ID = msg.Id
	me.ConversationID = msg.ConversationID
	me.Tags = msg.Tags
	me.Text = msg.Text
	me.UserID = msg.UserId
	me.MsgRole = msg.MsgRole
}
