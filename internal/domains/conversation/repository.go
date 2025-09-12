package conversation

import (
	"context"

	"github.com/google/uuid"
	"github.com/xpanvictor/xarvis/internal/types"
)

// Single conversation per user
type ConversationRepository interface {
	// Conversation
	RetrieveUserConversation(ctx context.Context, userID uuid.UUID, csr *types.ConvFetchRequest) (*types.Conversation, error) // creates if doesn't exist
	// Messages
	CreateMessage(ctx context.Context, userId uuid.UUID, msg types.Message) (*types.Message, error)
	FetchUserMessages(ctx context.Context, userId uuid.UUID, start, end int64) ([]types.Message, error)
	FetchMessage(ctx context.Context, msgId uuid.UUID) (*types.Message, error)
	// Memories
	FindMemories(ctx context.Context, conversationID uuid.UUID, msr types.MemorySearchRequest) ([]types.Memory, error)
	CreateMemory(ctx context.Context, conversationID uuid.UUID, m types.Memory) (*types.Memory, error)
}
