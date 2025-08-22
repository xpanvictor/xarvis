package conversation

import (
	"time"

	"github.com/xpanvictor/xarvis/pkg/assistant"
)

type Message struct {
	Id        string         `json:"id"`
	UserId    string         `json:"user_id"`
	Text      string         `json:"text"`
	Tags      []string       `json:"tags"`
	Timestamp time.Time      `json:"timestamp"`
	MsgRole   assistant.Role `json:"msg_role"`
}

// Single conversation per user
type ConversationRepository interface {
	CreateMessage(userId string, msg Message) (Message, error)
	FetchUserMessages(userId string) ([]Message, error)
	FetchMessage(msgId string) (Message, error)
}

func (m *Message) ToAssistantMessage() assistant.AssistantMessage {
	return assistant.AssistantMessage{
		Content:   m.Text,
		CreatedAt: m.Timestamp,
		MsgRole:   m.MsgRole,
	}
}
