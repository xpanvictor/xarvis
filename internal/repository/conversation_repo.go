package repository

import (
	"github.com/xpanvictor/xarvis/internal/domains/conversation"
	"gorm.io/gorm"
)

type GormConvoRepo struct {
	// db *gorm.DB
}

// CreateMessage implements conversation.ConversationRepository.
func (g GormConvoRepo) CreateMessage(userId string, msg conversation.Message) (conversation.Message, error) {
	panic("unimplemented")
}

// FetchMessage implements conversation.ConversationRepository.
func (g GormConvoRepo) FetchMessage(msgId string) (conversation.Message, error) {
	panic("unimplemented")
}

// FetchUserMessages implements conversation.ConversationRepository.
func (g GormConvoRepo) FetchUserMessages(userId string) ([]conversation.Message, error) {
	panic("unimplemented")
}

func NewGormConversationRepo(db *gorm.DB) conversation.ConversationRepository {
	// return GormConvoRepo{db: db}
	return nil
}
