package conversation

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/xpanvictor/xarvis/pkg/assistant"
	"github.com/xpanvictor/xarvis/pkg/assistant/adapters"
	"github.com/xpanvictor/xarvis/pkg/utils"
)

// short term info
type Message struct {
	Id             uuid.UUID      `json:"id"`
	UserId         uuid.UUID      `json:"user_id"`
	ConversationID uuid.UUID      `json:"conversation_id"`
	Text           string         `json:"text"`
	Tags           []string       `json:"tags"`
	Timestamp      time.Time      `json:"timestamp"`
	MsgRole        assistant.Role `json:"msg_role"`
}

type MemoryType string

const (
	EPISODIC MemoryType = "episodic"
	SEMANTIC MemoryType = "semantic"
)

type Memory struct {
	ID             uuid.UUID  `json:"id" example:""`
	ConversationID uuid.UUID  `json:"conversation_id"`
	Type           MemoryType `json:"memory_type"`
	SaliencyScore  uint8      `json:"saliency_score"` // Ever growing saliency score MRU
	Content        string     `json:"string"`
	// Embeddings
	EmbeddingRef any       `json:"embedding_ref"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Conversation struct {
	ID        uuid.UUID `json:"id"`
	OwnerID   uuid.UUID `json:"owner_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Summary   string    `json:"summary"`
	// Relationships
	Messages []Message `json:"messages"`
	Memories []Memory  `json:"memories"`
}

type MemorySearchRequest struct {
	WithinPeriod  *utils.Range[time.Time]
	SaliencyRange *utils.Range[uint8]
}

// Single conversation per user
type ConversationRepository interface {
	// Conversation
	RetrieveUserConversation(userID uuid.UUID) (*Conversation, error) // creates if doesn't exist
	// Messages
	CreateMessage(ctx context.Context, userId uuid.UUID, msg Message) (*Message, error)
	FetchUserMessages(ctx context.Context, userId uuid.UUID, start, end int64) ([]Message, error)
	FetchMessage(ctx context.Context, msgId uuid.UUID) (*Message, error)
	// Memories
	FindMemories(conversationID uuid.UUID, msr MemorySearchRequest) ([]Memory, error)
	CreateMemory(ctx context.Context, conversationID uuid.UUID, m Memory) (*Memory, error)
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
func AssistantMsgToMessage(am *assistant.AssistantMessage, userId uuid.UUID) Message {
	return Message{
		Id:        uuid.New(),
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
func ContractMsgToMessage(cm *adapters.ContractMessage, userId uuid.UUID, messageId uuid.UUID) Message {
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
