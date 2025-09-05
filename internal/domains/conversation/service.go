package conversation

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/xpanvictor/xarvis/pkg/Logger"
	"github.com/xpanvictor/xarvis/pkg/assistant"
)

type Conversation interface {
	ProcessMessage(ctx context.Context, userId string, msg Message) (*Message, error)
}

type conversation struct {
	repository ConversationRepository
	logger     *Logger.Logger
	// TODO: Replace with contract-based adapter when assistant types are removed
	// adapter    adapters.ContractAdapter
}

// ProcessMessage implements Conversation.
func (c conversation) ProcessMessage(
	ctx context.Context,
	userId string,
	msg Message,
) (*Message, error) {
	// store user msg
	c.repository.CreateMessage(userId, msg)

	// TODO: Replace this with contract-based processing
	// This is a temporary placeholder during migration
	response := Message{
		Id:        uuid.New().String(),
		UserId:    userId,
		Text:      "Service processing temporarily disabled during migration to contract types",
		Timestamp: time.Now(),
		MsgRole:   assistant.ASSISTANT,
		Tags:      []string{"migration", "placeholder"},
	}

	/*
		// Legacy code - to be replaced with contract-based processing:
		ar, err := c.assistant.ProcessPrompt(
			ctx,
			assistant.NewAssistantInput(
				[]assistant.AssistantMessage{
					msg.ToAssistantMessage(),
				},
				nil,
			),
		)
		if err != nil {
			return nil, fmt.Errorf("error processing message %v", err)
		}
		// store assistant message
		response := Message{
			Id:        ar.Id,
			UserId:    userId,
			Text:      ar.Response.Content,
			Timestamp: ar.Response.CreatedAt,
			MsgRole:   ar.Response.MsgRole,
		}
	*/

	// store assistant message
	c.repository.CreateMessage(userId, response)
	return &response, nil
}

func NewConversation(
	r ConversationRepository,
	l *Logger.Logger,
) Conversation {
	return conversation{
		logger:     l,
		repository: r,
	}
}
