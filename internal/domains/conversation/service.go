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
	"github.com/xpanvictor/xarvis/pkg/assistant/router"
	"github.com/xpanvictor/xarvis/pkg/io/registry"
	"github.com/xpanvictor/xarvis/pkg/utils"
)

var (
	ErrProcMsg = errors.New("error processing msg")
)

type ConversationService interface {
	RetrieveConversation(ctx context.Context, userID uuid.UUID) (*types.Conversation, error)
	ProcessMsg(ctx context.Context, userID uuid.UUID, msg types.Message, sysMsgs []types.Message) (*types.Message, error)
	ProcessMsgAsStream(ctx context.Context, userID uuid.UUID, msg types.Message, sysMsgs []types.Message, outCh chan<- types.Message) error
	CreateMemory(ctx context.Context, userID uuid.UUID, memory types.Memory) (*types.Memory, error)
	SearchMemories(ctx context.Context, userID uuid.UUID, query string, memoryType *types.MemoryType, limit int) ([]types.Memory, error)
	DeleteMemory(ctx context.Context, userID uuid.UUID, memoryID uuid.UUID) error
}

type conversationService struct {
	brainSystemFactory *brain.BrainSystemFactory
	repository         ConversationRepository
	logger             *Logger.Logger
}

func (c *conversationService) historyContext(ctx context.Context, userID uuid.UUID, msg types.Message) []types.Message {
	hist := make([]types.Message, 0)
	mems, err := c.SearchMemories(ctx, userID, msg.Text, nil, 10)
	if err != nil {
		return []types.Message{}
	}
	for _, m := range mems {
		hist = append(hist, m.AsMsgForHistory())
	}
	histMsgs, err := c.repository.FetchUserMessages(ctx, userID, 0, -1)
	hist = append(hist, histMsgs...)

	return hist
}

// ProcessMsg implements ConversationService.
func (c *conversationService) ProcessMsg(ctx context.Context, userID uuid.UUID, msg types.Message, sysMsgs []types.Message) (*types.Message, error) {
	// store user msg
	nmsg, err := c.repository.CreateMessage(ctx, userID, msg)
	if err != nil {
		return nil, fmt.Errorf("couldn't save msg: %v", err)
	}
	// Create a brain system instance for this request
	brainSystem := c.brainSystemFactory.CreateBrainSystem()

	// process msg in brain
	msgs := make([]types.Message, 0)
	msgs = append(msgs, sysMsgs...)
	// history
	oldMsgs := c.historyContext(ctx, userID, msg)
	msgs = append(msgs, oldMsgs...)

	msgs = append(msgs, *nmsg)
	//todo: handle sessions
	sessionID := uuid.New()
	resp, err := brainSystem.ProcessMessage(ctx, userID, sessionID, msgs)
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
	// Create a brain system instance for this request (streaming)
	brainSystem := c.brainSystemFactory.CreateBrainSystem()

	// process msg in brain
	msgs := make([]types.Message, 0)

	// history for context
	oldMsgs := c.historyContext(ctx, userID, msg)
	msgs = append(msgs, oldMsgs...)

	msgs = append(msgs, sysMsgs...)
	msgs = append(msgs, *nmsg)
	// Generate a session ID for this processing request
	sessionID := uuid.New()
	resp, err := brainSystem.ProcessMessageWithStreaming(ctx, userID, sessionID, msgs, false)
	// store sys message
	go func() {
		// ctxn, cancel := context.WithTimeout(ctx, 2*time.Second)
		// defer cancel()
		c.repository.CreateMessage(ctx, userID, *resp)
	}()
	return err
}

// RetrieveConversation implements ConversationService.
func (c *conversationService) RetrieveConversation(ctx context.Context, userID uuid.UUID) (*types.Conversation, error) {
	csr := types.ConvFetchRequest{
		Msr: &types.MemorySearchRequest{},
	}
	return c.repository.RetrieveUserConversation(ctx, userID, &csr)
}

// CreateMemory implements ConversationService.
func (c *conversationService) CreateMemory(ctx context.Context, userID uuid.UUID, memory types.Memory) (*types.Memory, error) {
	// Create a default fetch request to get the conversation
	// We don't need messages or memories for memory creation, just the conversation ID
	defaultFetchRequest := &types.ConvFetchRequest{
		MsgSearch: &utils.Range[int64]{
			Min: func() *int64 { v := int64(0); return &v }(),
			Max: func() *int64 { v := int64(0); return &v }(), // Fetch no messages since we don't need them
		},
		Msr: &types.MemorySearchRequest{}, // Empty memory search request
	}

	// First, get or create the user's conversation
	conversation, err := c.repository.RetrieveUserConversation(ctx, userID, defaultFetchRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve user conversation: %w", err)
	}

	// Set the conversation ID on the memory
	memory.ConversationID = conversation.ID

	// Create the memory using the repository
	createdMemory, err := c.repository.CreateMemory(ctx, conversation.ID, memory)
	if err != nil {
		return nil, fmt.Errorf("failed to create memory: %w", err)
	}

	c.logger.Info("Created memory for user %s: %s", userID, memory.Content)
	return createdMemory, nil
}

// SearchMemories implements ConversationService.
func (c *conversationService) SearchMemories(ctx context.Context, userID uuid.UUID, query string, memoryType *types.MemoryType, limit int) ([]types.Memory, error) {
	// Create a default fetch request to get the conversation
	defaultFetchRequest := &types.ConvFetchRequest{
		MsgSearch: &utils.Range[int64]{
			Min: func() *int64 { v := int64(0); return &v }(),
			Max: func() *int64 { v := int64(0); return &v }(), // Fetch no messages
		},
		Msr: &types.MemorySearchRequest{}, // Empty memory search request
	}

	// Get the user's conversation
	conversation, err := c.repository.RetrieveUserConversation(ctx, userID, defaultFetchRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve user conversation: %w", err)
	}

	// Create memory search request
	searchRequest := types.MemorySearchRequest{
		QueryStatement: &query,
	}

	// Apply memory type filter if specified
	// Note: The MemorySearchRequest doesn't have a memory type filter field yet
	// For now, we'll filter after getting results

	// Search memories using the repository
	memories, err := c.repository.FindMemories(ctx, conversation.ID, searchRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to search memories: %w", err)
	}

	// Filter by memory type if specified
	if memoryType != nil {
		var filteredMemories []types.Memory
		for _, memory := range memories {
			if memory.Type == *memoryType {
				filteredMemories = append(filteredMemories, memory)
			}
		}
		memories = filteredMemories
	}

	// Apply limit if specified
	if limit > 0 && len(memories) > limit {
		memories = memories[:limit]
	}

	c.logger.Info("Found %d memories for user %s with query: %s", len(memories), userID, query)
	return memories, nil
}

// DeleteMemory implements ConversationService.
func (c *conversationService) DeleteMemory(ctx context.Context, userID uuid.UUID, memoryID uuid.UUID) error {
	// Note: We could add additional validation here to ensure the memory belongs to the user
	// For now, we'll rely on the repository layer

	err := c.repository.DeleteMemory(ctx, memoryID)
	if err != nil {
		return fmt.Errorf("failed to delete memory: %w", err)
	}

	c.logger.Info("Deleted memory %s for user %s", memoryID, userID)
	return nil
}

func New(cfg config.Settings, gm *router.Mux,
	dr registry.DeviceRegistry,
	logger *Logger.Logger,
	repo ConversationRepository,
	brainSystemFactory *brain.BrainSystemFactory,
) ConversationService {
	return &conversationService{
		brainSystemFactory: brainSystemFactory,
		repository:         repo,
		logger:             logger,
	}
}

// SetBrainSystemFactory sets the brain system factory after creation
func (c *conversationService) SetBrainSystemFactory(factory *brain.BrainSystemFactory) {
	c.brainSystemFactory = factory
}
