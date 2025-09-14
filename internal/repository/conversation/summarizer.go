package conversation

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/xpanvictor/xarvis/internal/models/processor"
	"github.com/xpanvictor/xarvis/internal/types"
	"github.com/xpanvictor/xarvis/pkg/Logger"
)

// MemorySummaryResult represents the output structure for memory summarization
type MemorySummaryResult struct {
	WorthMemory    bool   `json:"worth_memory"`    // Whether this content is worth storing as memory
	Saliency       uint8  `json:"saliency"`        // Importance score (1-10)
	MemoryContent  string `json:"memory_content"`  // Summarized content for memory storage
	MemoryType     string `json:"memory_type"`     // Type of memory: "episodic" or "semantic"
	Keywords       []string `json:"keywords"`      // Key terms for search/indexing
	Confidence     float64 `json:"confidence"`     // Confidence in the summarization (0.0-1.0)
	Reason         string `json:"reason"`          // Why this is/isn't worth storing
}

// MessageSummarizerInput represents input for message summarization
type MessageSummarizerInput struct {
	UserID       uuid.UUID       `json:"user_id"`
	Messages     []types.Message `json:"messages"`
	TimeWindow   string          `json:"time_window"` // e.g., "last 3 minutes"
	TotalCount   int             `json:"total_count"`
}

// MessageSummarizer handles converting messages to memories
type MessageSummarizer struct {
	processor  processor.Processor
	logger     *Logger.Logger
	repo       types.ConversationRepository
}

// NewMessageSummarizer creates a new message summarizer
func NewMessageSummarizer(
	processor processor.Processor,
	logger *Logger.Logger,
	repo types.ConversationRepository,
) *MessageSummarizer {
	return &MessageSummarizer{
		processor: processor,
		logger:    logger,
		repo:      repo,
	}
}

// SummarizeMessages processes recent messages and determines if they should become memories
func (ms *MessageSummarizer) SummarizeMessages(ctx context.Context, userID uuid.UUID) (*MemorySummaryResult, error) {
	// Fetch recent messages from Redis (last 3 minutes)
	endTime := time.Now().Unix()
	startTime := endTime - (3 * 60) // 3 minutes ago
	
	messages, err := ms.repo.FetchUserMessages(ctx, userID, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user messages: %w", err)
	}
	
	if len(messages) == 0 {
		ms.logger.Debug(fmt.Sprintf("No messages found for user %s in the last 3 minutes", userID))
		return &MemorySummaryResult{
			WorthMemory: false,
			Reason:      "No messages found in the time window",
		}, nil
	}
	
	// Prepare input for processor
	input := MessageSummarizerInput{
		UserID:     userID,
		Messages:   messages,
		TimeWindow: "last 3 minutes",
		TotalCount: len(messages),
	}
	
	// Create instruction for the processor
	instruction := `You are a memory formation system for a personal AI assistant. Analyze the provided conversation messages and determine if they contain information worth storing as long-term memory.

Consider these factors:
1. Important facts, plans, or commitments mentioned by the user
2. Significant events or experiences shared
3. Preferences, opinions, or personal information revealed
4. Task assignments or reminders
5. Meaningful conversations or emotional content

For each analysis, provide:
- worth_memory: true if this deserves to be stored as memory, false otherwise
- saliency: importance score 1-10 (1=trivial, 10=critical life information)
- memory_content: if worth storing, provide a clear, concise summary
- memory_type: "episodic" for events/experiences, "semantic" for facts/preferences
- keywords: 3-7 relevant terms for searching this memory later
- confidence: how confident you are in this assessment (0.0-1.0)
- reason: brief explanation of your decision

Only mark as worth_memory if there's genuinely valuable information. Casual greetings, small talk, or repeated information should not become memories.`

	// Create expected response structure for type safety
	expectedResponse := MemorySummaryResult{}
	
	// Process with the processor
	err = ms.processor.ProcessWithType(ctx, instruction, input, &expectedResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to process messages with summarizer: %w", err)
	}
	
	ms.logger.Info(fmt.Sprintf("Message summarization completed for user %s: worth_memory=%t, saliency=%d", 
		userID, expectedResponse.WorthMemory, expectedResponse.Saliency))
	
	return &expectedResponse, nil
}

// ProcessAndStoreMemory processes messages and stores memory if worthy
func (ms *MessageSummarizer) ProcessAndStoreMemory(ctx context.Context, userID uuid.UUID) error {
	// Summarize messages
	summary, err := ms.SummarizeMessages(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to summarize messages: %w", err)
	}
	
	// If not worth storing, just log and return
	if !summary.WorthMemory {
		ms.logger.Debug(fmt.Sprintf("Messages not worth storing as memory for user %s: %s", userID, summary.Reason))
		return nil
	}
	
	// Get conversation for this user
	conversation, err := ms.repo.RetrieveUserConversation(ctx, userID, &types.ConvFetchRequest{})
	if err != nil {
		return fmt.Errorf("failed to retrieve conversation: %w", err)
	}
	
	// Create memory from summary
	memoryType := types.EPISODIC
	if summary.MemoryType == "semantic" {
		memoryType = types.SEMANTIC
	}
	
	memory := types.Memory{
		ID:             uuid.New(),
		ConversationID: conversation.ID,
		Type:           memoryType,
		SaliencyScore:  summary.Saliency,
		Content:        summary.MemoryContent,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	
	// Store the memory
	_, err = ms.repo.CreateMemory(ctx, conversation.ID, memory)
	if err != nil {
		return fmt.Errorf("failed to create memory: %w", err)
	}
	
	ms.logger.Info(fmt.Sprintf("Created memory for user %s: type=%s, saliency=%d, content=%s", 
		userID, memory.Type, memory.SaliencyScore, truncateString(memory.Content, 100)))
	
	// Clean up processed messages from Redis
	err = ms.cleanupProcessedMessages(ctx, userID)
	if err != nil {
		ms.logger.Error(fmt.Sprintf("Failed to cleanup messages for user %s: %v", userID, err))
		// Don't return error as memory was successfully created
	}
	
	return nil
}

// cleanupProcessedMessages removes processed messages from Redis
func (ms *MessageSummarizer) cleanupProcessedMessages(ctx context.Context, userID uuid.UUID) error {
	// Get messages older than 3 minutes to remove them
	endTime := time.Now().Unix() - (3 * 60) // 3 minutes ago
	
	// This is a placeholder for Redis cleanup
	// In the actual implementation, this would:
	// 1. Use Redis commands to remove messages from the sorted set
	// 2. Clean up based on timestamp scores
	// 3. Maintain recent messages while removing processed ones
	
	ms.logger.Debug(fmt.Sprintf("Would cleanup messages for user %s older than %d", userID, endTime))
	
	// TODO: Implement actual Redis cleanup
	// Example Redis commands that would be used:
	// ZREMRANGEBYSCORE user:{userID}:msgs -inf {endTime}
	
	return nil
}

// GetActiveUsersWithRecentMessages retrieves users who have messages in the last time window
func (ms *MessageSummarizer) GetActiveUsersWithRecentMessages(ctx context.Context, timeWindow time.Duration) ([]uuid.UUID, error) {
	// This is a placeholder that should be implemented based on your Redis structure
	// In a real implementation, this would:
	// 1. Scan Redis for all user message keys (user:*:msgs)
	// 2. Check which ones have messages in the specified time window
	// 3. Return the user IDs
	
	ms.logger.Debug(fmt.Sprintf("Would scan for active users with messages in last %s", timeWindow))
	
	// TODO: Implement actual Redis scanning
	// This would involve Redis SCAN commands to find user keys
	// and then checking message timestamps
	
	return []uuid.UUID{}, nil
}

// GetMessagesContent creates a readable string from messages for logging
func (ms *MessageSummarizer) GetMessagesContent(messages []types.Message) string {
	var contentParts []string
	for _, msg := range messages {
		contentParts = append(contentParts, fmt.Sprintf("[%s] %s: %s", 
			msg.Timestamp.Format("15:04:05"), msg.MsgRole, msg.Text))
	}
	return strings.Join(contentParts, "\n")
}

// Helper function to truncate strings for logging
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
