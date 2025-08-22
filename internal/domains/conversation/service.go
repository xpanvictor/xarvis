package conversation

import "github.com/xpanvictor/xarvis/pkg/Logger"

type Conversation interface {
	ProcessMessage(userId string, msg Message) Message
}

type conversation struct {
	repository ConversationRepository
	logger     *Logger.Logger
}

// ProcessMessage implements Conversation.
func (c conversation) ProcessMessage(userId string, msg Message) Message {
	c.logger.Warn("received msg and pending processing")
	c.repository.CreateMessage(userId, msg)
	return msg
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
