package conversation

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/xpanvictor/xarvis/internal/config"
	"github.com/xpanvictor/xarvis/internal/runtime/brain"
	"github.com/xpanvictor/xarvis/internal/types"
	"github.com/xpanvictor/xarvis/pkg/Logger"
)

var (
	ErrProcMsg = errors.New("error processing msg")
)

type ConversationService interface {
	RetrieveConversation(ctx context.Context, userID uuid.UUID) (*types.Conversation, error)
	ProcessMsg(ctx context.Context, userID uuid.UUID, msg types.Message, sysMsgs []types.Message) (*types.Message, error)
	ProcessMsgAsStream(ctx context.Context, userID uuid.UUID, msg types.Message, sysMsgs []types.Message, outCh chan<- types.Message) error
}

type conversationService struct {
	bs         brain.BrainSystem
	repository ConversationRepository
	logger     *Logger.Logger
}

// ProcessMsg implements ConversationService.
func (c *conversationService) ProcessMsg(ctx context.Context, userID uuid.UUID, msg types.Message, sysMsgs []types.Message) (*types.Message, error) {
	// store user msg
	nmsg, err := c.repository.CreateMessage(ctx, userID, msg)
	if err != nil {
		return nil, fmt.Errorf("couldn't save msg: %v", err)
	}
	// process msg in brain
	msgs := make([]types.Message, 0)
	msgs = append(msgs, sysMsgs...)
	msgs = append(msgs, *nmsg)
	//todo: handle sessions
	sessionID := uuid.New()
	resp, err := c.bs.ProcessMessage(ctx, userID, sessionID, msgs)
	if err != nil {
		c.logger.Errorf("proc msg: %v", err)
		return nil, ErrProcMsg
	}
	// store sys message
	go func() {
		ctxn, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		c.repository.CreateMessage(ctxn, userID, *resp)
	}()
	return resp, nil
}

// ProcessMsgAsStream implements ConversationService.
func (c *conversationService) ProcessMsgAsStream(ctx context.Context, userID uuid.UUID, msg types.Message, sysMsgs []types.Message, outCh chan<- types.Message) error {
	// store user msg
	nmsg, err := c.repository.CreateMessage(ctx, userID, msg)
	if err != nil {
		return fmt.Errorf("couldn't save msg: %v", err)
	}
	// process msg in brain
	msgs := make([]types.Message, 0)
	msgs = append(msgs, sysMsgs...)
	msgs = append(msgs, *nmsg)
	//todo: handle sessions
	sessionID := uuid.New()
	err = c.bs.ProcessMessageWithStreaming(ctx, userID, sessionID, msgs, false)
	return err
}

// RetrieveConversation implements ConversationService.
func (c *conversationService) RetrieveConversation(ctx context.Context, userID uuid.UUID) (*types.Conversation, error) {
	csr := types.ConvFetchRequest{
		Msr: &types.MemorySearchRequest{},
	}
	return c.repository.RetrieveUserConversation(ctx, userID, &csr)
}

func New(cfg config.Settings) ConversationService {
	return &conversationService{}
}
