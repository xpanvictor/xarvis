package brain

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/xpanvictor/xarvis/internal/config"
	"github.com/xpanvictor/xarvis/internal/domains/conversation"
	"github.com/xpanvictor/xarvis/pkg/Logger"
	"github.com/xpanvictor/xarvis/pkg/assistant"
	"github.com/xpanvictor/xarvis/pkg/assistant/adapters"
	toolsystem "github.com/xpanvictor/xarvis/pkg/tool_system"
)

type Brain struct {
	cfg      config.BrainConfig
	registry toolsystem.Registry
	executor toolsystem.Executor
	logger   Logger.Logger
	adapter  adapters.ContractAdapter // Contract-based LLM adapter
	// memory
	// msg history
}

// Brain Decision System using Messages (BDSM)
// when the brain gets a percept, it enters a thinking <-> acting loop
// using contract-based LLM processing with tool execution
func (b *Brain) Decide(ctx context.Context, msg conversation.Message) (conversation.Message, error) {
	// Convert conversation message to contract format
	contractMsgs := []adapters.ContractMessage{msg.ToContractMessage()}

	// Get available tools from registry
	availableTools := b.registry.GetContractTools()

	// Create contract input
	contractInput := adapters.ContractInput{
		ID:       uuid.New(),
		ToolList: availableTools,
		Meta:     map[string]interface{}{"user_id": msg.UserId},
		Msgs:     contractMsgs,
		HandlerModel: adapters.ContractSelectedModel{
			Name:    "llama3", // TODO: Make this configurable
			Version: "8b",
		},
	}

	toolCallsCount := 0
	maxToolCalls := 5 // Default limit, should use b.cfg.MaxToolCallLimit when available

	// Process with tool calling loop
	for toolCallsCount < maxToolCalls {
		// Create response channel
		responseChannel := make(adapters.ContractResponseChannel, 10)

		// Process through adapter
		contractResponse := b.adapter.Process(ctx, contractInput, &responseChannel)

		// Collect response deltas
		var finalMessage *adapters.ContractMessage
		var toolCalls []adapters.ContractToolCall

		// Read from response channel until closed
		for deltas := range responseChannel {
			for _, delta := range deltas {
				if delta.Msg != nil {
					finalMessage = delta.Msg
				}
				if delta.ToolCalls != nil && len(*delta.ToolCalls) > 0 {
					toolCalls = append(toolCalls, *delta.ToolCalls...)
				}
				if delta.Error != nil {
					return conversation.Message{}, fmt.Errorf("LLM processing error: %w", delta.Error)
				}
			}
		}

		// Check for errors in the contract response
		if contractResponse.Error != nil {
			return conversation.Message{}, fmt.Errorf("contract processing error: %w", contractResponse.Error)
		}

		// If no tool calls, we're done
		if len(toolCalls) == 0 {
			if finalMessage != nil {
				// Convert final message back to conversation format
				return conversation.ContractMsgToMessage(finalMessage, msg.UserId, uuid.New().String()), nil
			}
			// Fallback if no message received
			return conversation.Message{
				Id:        uuid.New().String(),
				UserId:    msg.UserId,
				Text:      "No response generated",
				Timestamp: time.Now(),
				MsgRole:   assistant.ASSISTANT,
				Tags:      []string{"empty_response"},
			}, nil
		}

		// Execute tool calls
		toolResponses, executedCount := b.ExecuteToolCallsParallel(ctx, toolCalls)
		toolCallsCount += executedCount

		// Convert tool responses to contract messages and add to conversation
		for _, toolResponse := range toolResponses {
			contractMsgs = append(contractMsgs, toolResponse.ToContractMessage())
		}

		// Update contract input for next iteration
		contractInput.Msgs = contractMsgs
	}

	// If we've exceeded tool call limit, return the last assistant message
	if len(contractMsgs) > 0 {
		lastMsg := contractMsgs[len(contractMsgs)-1]
		if lastMsg.Role == adapters.ASSISTANT {
			return conversation.ContractMsgToMessage(&lastMsg, msg.UserId, uuid.New().String()), nil
		}
	}

	// Fallback response
	return conversation.Message{
		Id:        uuid.New().String(),
		UserId:    msg.UserId,
		Text:      "Maximum tool calls exceeded. Unable to complete processing.",
		Timestamp: time.Now(),
		MsgRole:   assistant.ASSISTANT,
		Tags:      []string{"tool_limit_exceeded"},
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
	adapter adapters.ContractAdapter,
	logger Logger.Logger,
) *Brain {
	return &Brain{
		cfg:      cfg,
		registry: registry,
		executor: executor,
		adapter:  adapter,
		logger:   logger,
	}
}
