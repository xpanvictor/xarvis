package brain

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/xpanvictor/xarvis/internal/config"
	"github.com/xpanvictor/xarvis/internal/domains/conversation"
	"github.com/xpanvictor/xarvis/pkg/Logger"
	"github.com/xpanvictor/xarvis/pkg/assistant"
	"github.com/xpanvictor/xarvis/pkg/assistant/adapters"
	"github.com/xpanvictor/xarvis/pkg/assistant/router"
	toolsystem "github.com/xpanvictor/xarvis/pkg/tool_system"
)

type Brain struct {
	cfg          config.BrainConfig
	registry     toolsystem.Registry
	executor     toolsystem.Executor
	logger       Logger.Logger
	mux          router.Mux // LLM router/multiplexer
	defaultModel string     // Default model name to use for requests
	// memory
	// msg history
}

// BrainSession represents an active conversation session
type BrainSession struct {
	UserID    uuid.UUID
	SessionID uuid.UUID
	Messages  []conversation.Message // conversation history
}

// ProcessMessage handles incoming messages and returns response channel for streaming
// The caller (BrainSystem) handles the pipeline broadcasting
func (b *Brain) ProcessMessage(ctx context.Context, session BrainSession, msg conversation.Message) (<-chan []adapters.ContractResponseDelta, error) {

	// generate sys message

	ct := time.Now()
	systemMessage := adapters.ContractMessage{
		Role:      adapters.SYSTEM,
		Content:   fmt.Sprintf("You are an AI agent called Xarvis and you're sort of my second brain. The current time is %v", ct),
		CreatedAt: ct,
	}

	// Debug logging for message content
	b.logger.Infof("Brain processing for UserID %s, SessionID %s: Message = '%s'",
		session.UserID.String(), session.SessionID.String(), msg.Text)

	// Convert conversation message to contract format
	contractMsgs := []adapters.ContractMessage{systemMessage, msg.ToContractMessage()}

	// Add conversation history
	for i, histMsg := range session.Messages {
		b.logger.Infof("Brain adding history message %d for UserID %s: '%s'",
			i, session.UserID.String(), histMsg.Text)
		contractMsgs = append(contractMsgs, histMsg.ToContractMessage())
	}

	// Get available tools from registry
	availableTools := b.registry.GetContractTools()

	// Create contract input
	contractInput := adapters.ContractInput{
		ID:       uuid.New(),
		ToolList: availableTools,
		Meta:     map[string]interface{}{"user_id": session.UserID.String()},
		Msgs:     contractMsgs,
		HandlerModel: adapters.ContractSelectedModel{
			Name:    b.defaultModel, // Use configurable model name
			Version: "8b",
		},
	}

	// Debug logging to see what model we're actually requesting
	b.logger.Infof("Processing message with model: %s", b.defaultModel)

	// Create response channel for streaming
	responseChannel := make(adapters.ContractResponseChannel, 10)

	// Start streaming through mux in goroutine
	go func() {
		defer func() {
			// Safely close channel using recovery to prevent panic
			defer func() {
				if r := recover(); r != nil {
					b.logger.Error(fmt.Sprintf("Recovered from channel close panic: %v", r))
				}
			}()
			// Try to close channel, but recover if it's already closed
			select {
			case <-responseChannel:
				// Channel is already closed
			default:
				close(responseChannel)
			}
		}()

		err := b.mux.Stream(ctx, contractInput, &responseChannel)
		if err != nil {
			b.logger.Error(fmt.Sprintf("Mux streaming error: %v", err))

			// If the error is about tools not being supported, try without tools
			if strings.Contains(err.Error(), "does not support tools") {
				b.logger.Info("Model doesn't support tools, retrying without tools")

				// Create a new input without tools
				contractInputNoTools := adapters.ContractInput{
					ID:           uuid.New(),
					ToolList:     []adapters.ContractTool{}, // Empty tool list
					Meta:         contractInput.Meta,
					Msgs:         contractInput.Msgs,
					HandlerModel: contractInput.HandlerModel,
				}

				// Try again without tools
				err = b.mux.Stream(ctx, contractInputNoTools, &responseChannel)
				if err != nil {
					b.logger.Error(fmt.Sprintf("Failed even without tools: %v", err))
					// Send error through channel
					select {
					case responseChannel <- []adapters.ContractResponseDelta{{Error: err}}:
					default:
						b.logger.Error("Could not send fallback error through response channel")
					}
				}
			} else {
				// Send original error through channel
				select {
				case responseChannel <- []adapters.ContractResponseDelta{{Error: err}}:
				default:
					b.logger.Error("Could not send error through response channel")
				}
			}
		}
	}()

	// Return the channel for the caller to handle streaming and tool execution
	return responseChannel, nil
}

// ProcessMessageWithToolHandling handles incoming messages with automatic tool execution
// Returns the final response after processing all tool calls
func (b *Brain) ProcessMessageWithToolHandling(ctx context.Context, session BrainSession, msg conversation.Message) (conversation.Message, error) {
	responseChannel, err := b.ProcessMessage(ctx, session, msg)
	if err != nil {
		return conversation.Message{}, err
	}

	// Handle tool execution loop
	toolCallsCount := 0
	maxToolCalls := 5 // TODO: Use b.cfg.MaxToolCallLimit when available
	var finalResponse conversation.Message

	for deltas := range responseChannel {
		var toolCalls []adapters.ContractToolCall
		var hasToolCalls bool
		var messageContent string

		// Process deltas
		for _, delta := range deltas {
			// Collect message content
			if delta.Msg != nil && delta.Msg.Content != "" {
				messageContent += delta.Msg.Content
			}

			// Collect tool calls
			if delta.ToolCalls != nil && len(*delta.ToolCalls) > 0 {
				toolCalls = append(toolCalls, *delta.ToolCalls...)
				hasToolCalls = true
			}

			// Handle errors
			if delta.Error != nil {
				return conversation.Message{}, fmt.Errorf("processing error: %w", delta.Error)
			}
		}

		// If we have tool calls and haven't exceeded the limit
		if hasToolCalls && toolCallsCount < maxToolCalls {
			toolResponses, executedCount := b.ExecuteToolCallsParallel(ctx, toolCalls)
			toolCallsCount += executedCount

			// For simplicity in this method, append tool results to message content
			for _, toolResp := range toolResponses {
				messageContent += fmt.Sprintf("\n[Tool Result: %s]", toolResp.Text)
			}
		}

		// Set final response
		if messageContent != "" {
			finalResponse = conversation.Message{
				Id:        uuid.New().String(),
				UserId:    session.UserID.String(),
				Text:      messageContent,
				Timestamp: time.Now(),
				MsgRole:   assistant.ASSISTANT,
				Tags:      []string{"brain_response"},
			}
		}
	}

	if finalResponse.Text == "" {
		return conversation.Message{
			Id:        uuid.New().String(),
			UserId:    session.UserID.String(),
			Text:      "No response generated",
			Timestamp: time.Now(),
			MsgRole:   assistant.ASSISTANT,
			Tags:      []string{"empty_response"},
		}, nil
	}

	return finalResponse, nil
}

// Decide provides a simple non-streaming interface (backwards compatibility)
func (b *Brain) Decide(ctx context.Context, msg conversation.Message) (conversation.Message, error) {
	// For simple decide, we'll collect the full response instead of streaming
	// This is less efficient but provides backwards compatibility
	contractMsgs := []adapters.ContractMessage{msg.ToContractMessage()}

	availableTools := b.registry.GetContractTools()

	contractInput := adapters.ContractInput{
		ID:       uuid.New(),
		ToolList: availableTools,
		Meta:     map[string]interface{}{"user_id": msg.UserId},
		Msgs:     contractMsgs,
	}

	responseChannel := make(adapters.ContractResponseChannel, 10)

	// Start streaming
	go func() {
		defer func() {
			// Safely close channel using recovery to prevent panic
			defer func() {
				if r := recover(); r != nil {
					b.logger.Error(fmt.Sprintf("Recovered from channel close panic in Decide: %v", r))
				}
			}()
			// Try to close channel, but recover if it's already closed
			select {
			case <-responseChannel:
				// Channel is already closed
			default:
				close(responseChannel)
			}
		}()

		err := b.mux.Stream(ctx, contractInput, &responseChannel)
		if err != nil {
			b.logger.Error(fmt.Sprintf("Simple decide streaming error: %v", err))
			// Send error through channel instead of just logging
			select {
			case responseChannel <- []adapters.ContractResponseDelta{{Error: err}}:
			default:
				// Channel is full or closed, just log
				b.logger.Error("Could not send error through response channel in Decide")
			}
		}
	}()

	// Collect all responses
	var finalContent string
	var toolCalls []adapters.ContractToolCall

	for deltas := range responseChannel {
		for _, delta := range deltas {
			if delta.Msg != nil && delta.Msg.Content != "" {
				finalContent += delta.Msg.Content
			}
			if delta.ToolCalls != nil && len(*delta.ToolCalls) > 0 {
				toolCalls = append(toolCalls, *delta.ToolCalls...)
			}
			if delta.Error != nil {
				return conversation.Message{}, fmt.Errorf("decide error: %w", delta.Error)
			}
		}
	}

	// Execute tool calls if any
	if len(toolCalls) > 0 {
		toolResponses, _ := b.ExecuteToolCallsParallel(ctx, toolCalls)
		// For simplicity, append tool results to the final content
		for _, toolResp := range toolResponses {
			finalContent += fmt.Sprintf("\n[Tool Result: %s]", toolResp.Text)
		}
	}

	return conversation.Message{
		Id:        uuid.New().String(),
		UserId:    msg.UserId,
		Text:      finalContent,
		Timestamp: time.Now(),
		MsgRole:   assistant.ASSISTANT,
		Tags:      []string{"brain_response"},
	}, nil
}

func (b *Brain) ExecuteToolCallsParallel(
	ctx context.Context,
	toolCalls []adapters.ContractToolCall,
) ([]conversation.Message, int) {
	toolResponses := make([]conversation.Message, 0)
	// mutex & ...
	wg := sync.WaitGroup{}
	mu := sync.Mutex{}
	for _, toolCall := range toolCalls {
		wg.Add(1)
		go func(tc adapters.ContractToolCall) {
			defer wg.Done()
			// call fn using new executor interface
			result, err := b.executor.Execute(ctx, b.registry, tc)
			var toolResponse conversation.Message

			if err != nil {
				toolResponse = conversation.Message{
					Id:        uuid.New().String(),
					UserId:    "", // Will be set by caller
					Text:      fmt.Sprintf("Tool execution failed: %v", err),
					Timestamp: time.Now(),
					MsgRole:   assistant.TOOL,
					Tags:      []string{"error", "tool_call"},
				}
			} else {
				// Convert tool result to message content
				resultStr := "Tool execution completed"
				if result.Result != nil {
					if content, ok := result.Result["content"].(string); ok {
						resultStr = content
					} else {
						// Convert result to JSON string for display
						if jsonBytes, err := json.Marshal(result.Result); err == nil {
							resultStr = string(jsonBytes)
						}
					}
				}

				toolResponse = conversation.Message{
					Id:        uuid.New().String(),
					UserId:    "", // Will be set by caller
					Text:      resultStr,
					Timestamp: time.Now(),
					MsgRole:   assistant.TOOL,
					Tags:      []string{"success", "tool_call", tc.ToolName},
				}
			}
			mu.Lock()
			toolResponses = append(toolResponses, toolResponse)
			mu.Unlock()
		}(toolCall)
	}
	wg.Wait()
	return toolResponses, len(toolCalls)
}

func (b *Brain) Think(ctx context.Context, userID string) {
	// acquire shared distributed lock
	ok := b.acquireMindLock(ctx, userID)
	if !ok {
		return // handle proper assessment & retry
	}
	// gather context about user using:
	// memory, projects, approvals, persona
	userCtxInfo, _ := b.gatherUserContextInfo(ctx, userID)
	reflection, _ := b.reflect(ctx, userID, *userCtxInfo)
	plan, err := b.plan(ctx, *reflection)
	if err != nil {
		return // only plan's error matters now
	}
	b.logger.Debug(plan)
	// enter decision loop based on plan
	// observe and update memory
	// update project, persona, approvals
	// actions can involve requesting attention or reaching out
	panic("unimpl think")
}

func (b *Brain) gatherUserContextInfo(ctx context.Context, userID string) (*UserCtxInfo, error) {
	panic("unimpl info")
}

func (b *Brain) reflect(ctx context.Context, userId string, userContext UserCtxInfo) (*Reflection, error) {
	panic("unimpl reflect")
}

func (b *Brain) plan(ctx context.Context, reflection Reflection) (*Plan, error) {
	panic("unimpl plan")
}

func (b *Brain) acquireMindLock(ctx context.Context, userId string) bool {
	// TODO: handle mind lock
	return true
}

// NewBrain creates a new Brain instance with contract-based processing
func NewBrain(
	cfg config.BrainConfig,
	registry toolsystem.Registry,
	executor toolsystem.Executor,
	mux router.Mux,
	logger Logger.Logger,
	defaultModel string,
) *Brain {
	return &Brain{
		cfg:          cfg,
		registry:     registry,
		executor:     executor,
		mux:          mux,
		logger:       logger,
		defaultModel: defaultModel,
	}
}

// Session management methods

// CreateSession creates a new brain session for a user
func (b *Brain) CreateSession(userID uuid.UUID) BrainSession {
	return BrainSession{
		UserID:    userID,
		SessionID: uuid.New(),
		Messages:  make([]conversation.Message, 0),
	}
}

// AddMessageToSession adds a message to the session history
func (b *Brain) AddMessageToSession(session *BrainSession, msg conversation.Message) {
	session.Messages = append(session.Messages, msg)
}

// GetSessionHistory returns the conversation history for a session
func (b *Brain) GetSessionHistory(session BrainSession) []adapters.ContractMessage {
	contractMsgs := make([]adapters.ContractMessage, len(session.Messages))
	for i, msg := range session.Messages {
		contractMsgs[i] = msg.ToContractMessage()
	}
	return contractMsgs
}
