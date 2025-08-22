package conversation

import (
	"context"
	"fmt"

	"github.com/xpanvictor/xarvis/pkg/Logger"
	"github.com/xpanvictor/xarvis/pkg/assistant"
)

type Conversation interface {
	ProcessMessage(ctx context.Context, userId string, msg Message) (*Message, error)
}

type conversation struct {
	repository ConversationRepository
	logger     *Logger.Logger
	assistant  assistant.Assistant
}

// ProcessMessage implements Conversation.
func (c conversation) ProcessMessage(
	ctx context.Context,
	userId string,
	msg Message,
) (*Message, error) {
	// store user msg
	c.repository.CreateMessage(userId, msg)
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
	c.repository.CreateMessage(userId, Message{
		Id:        ar.Id,
		UserId:    userId,
		Text:      ar.Response.Content,
		Timestamp: ar.Response.CreatedAt,
		MsgRole:   ar.Response.MsgRole,
	})
	return &msg, nil
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
