package conversation

import (
	"time"

	"github.com/xpanvictor/xarvis/pkg/assistant"
	"github.com/xpanvictor/xarvis/pkg/assistant/adapters"
)

type Message struct {
	Id        string         `json:"id"`
	UserId    string         `json:"user_id"`
	Text      string         `json:"text"`
	Tags      []string       `json:"tags"`
	Timestamp time.Time      `json:"timestamp"`
	MsgRole   assistant.Role `json:"msg_role"`
}

// type Conversation struct {
// 	Id string `json:"id"`
// }

// Single conversation per user
type ConversationRepository interface {
	CreateMessage(userId string, msg Message) (Message, error)
	FetchUserMessages(userId string) ([]Message, error)
	FetchMessage(msgId string) (Message, error)
}

// Legacy conversion method - will be removed when assistant types are removed
func (m *Message) ToAssistantMessage() assistant.AssistantMessage {
	return assistant.AssistantMessage{
		Content:   m.Text,
		CreatedAt: m.Timestamp,
		MsgRole:   m.MsgRole,
	}
}

// Legacy conversion method - will be removed when assistant types are removed
func AssistantMsgToMessage(am *assistant.AssistantMessage, userId string) Message {
	return Message{
		Id:        "todo",
		UserId:    userId,
		Text:      am.Content,
		MsgRole:   am.MsgRole,
		Timestamp: am.CreatedAt,
	}
}

// New contract-based conversion methods
func (m *Message) ToContractMessage() adapters.ContractMessage {
	// Convert assistant.Role to adapters.MsgRole
	var contractRole adapters.MsgRole
	switch m.MsgRole {
	case assistant.USER:
		contractRole = adapters.USER
	case assistant.ASSISTANT:
		contractRole = adapters.ASSISTANT
	case assistant.SYSTEM:
		contractRole = adapters.SYSTEM
	case assistant.TOOL:
		contractRole = adapters.TOOL
	default:
		contractRole = adapters.USER // default fallback
	}

	return adapters.ContractMessage{
		Role:      contractRole,
		Content:   m.Text,
		CreatedAt: m.Timestamp,
	}
}

// Convert contract message to conversation message
func ContractMsgToMessage(cm *adapters.ContractMessage, userId string, messageId string) Message {
	// Convert adapters.MsgRole to assistant.Role
	var assistantRole assistant.Role
	switch cm.Role {
	case adapters.USER:
		assistantRole = assistant.USER
	case adapters.ASSISTANT:
		assistantRole = assistant.ASSISTANT
	case adapters.SYSTEM:
		assistantRole = assistant.SYSTEM
	case adapters.TOOL:
		assistantRole = assistant.TOOL
	default:
		assistantRole = assistant.USER // default fallback
	}

	return Message{
		Id:        messageId,
		UserId:    userId,
		Text:      cm.Content,
		MsgRole:   assistantRole,
		Timestamp: cm.CreatedAt,
		Tags:      []string{}, // empty tags by default
	}
}

// Convert slice of messages to contract messages
func MessagesToContractMessages(messages []Message) []adapters.ContractMessage {
	contractMsgs := make([]adapters.ContractMessage, len(messages))
	for i, msg := range messages {
		contractMsgs[i] = msg.ToContractMessage()
	}
	return contractMsgs
}
