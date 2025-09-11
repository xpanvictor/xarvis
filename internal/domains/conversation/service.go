package conversation

// import (
// 	"context"
// 	"time"

// 	"github.com/google/uuid"
// 	"github.com/xpanvictor/xarvis/pkg/Logger"
// 	"github.com/xpanvictor/xarvis/pkg/assistant"
// )

// type ConversationManager interface {
// 	ProcessMessage(ctx context.Context, userId uuid.UUID, msg Message) (*Message, error)
// }

// type conversation struct {
// 	repository ConversationRepository
// 	logger     *Logger.Logger
// 	// TODO: Replace with contract-based adapter when assistant types are removed
// 	// adapter    adapters.ContractAdapter
// }

// // ProcessMessage implements Conversation.
// func (c conversation) ProcessMessage(
// 	ctx context.Context,
// 	userId uuid.UUID,
// 	msg Message,
// ) (*Message, error) {
// 	// store user msg
// 	c.repository.CreateMessage(userId.String(), msg)

// 	// TODO: Replace this with contract-based processing
// 	// This is a temporary placeholder during migration
// 	response := Message{
// 		Id:        uuid.New(),
// 		UserId:    userId,
// 		Text:      "Service processing temporarily disabled during migration to contract types",
// 		Timestamp: time.Now(),
// 		MsgRole:   assistant.ASSISTANT,
// 		Tags:      []string{"migration", "placeholder"},
// 	}

// 	/*
// 		// Legacy code - to be replaced with contract-based processing:
// 		ar, err := c.assistant.ProcessPrompt(
// 			ctx,
// 			assistant.NewAssistantInput(
// 				[]assistant.AssistantMessage{
// 					msg.ToAssistantMessage(),
// 				},
// 				nil,
// 			),
// 		)
// 		if err != nil {
// 			return nil, fmt.Errorf("error processing message %v", err)
// 		}
// 		// store assistant message
// 		response := Message{
// 			Id:        ar.Id,
// 			UserId:    userId,
// 			Text:      ar.Response.Content,
// 			Timestamp: ar.Response.CreatedAt,
// 			MsgRole:   ar.Response.MsgRole,
// 		}
// 	*/

// 	// store assistant message
// 	c.repository.CreateMessage(userId.String(), response)
// 	return &response, nil
// }

// func NewConversation(
// 	r ConversationRepository,
// 	l *Logger.Logger,
// ) ConversationManager {
// 	return conversation{
// 		logger:     l,
// 		repository: r,
// 	}
// }
