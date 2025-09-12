package brain

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/xpanvictor/xarvis/internal/config"
	"github.com/xpanvictor/xarvis/internal/types"
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
	logger       *Logger.Logger
	mux          router.Mux // LLM router/multiplexer
	defaultModel string     // Default model name to use for requests
	// memory
	// msg history
}

// Brain Decision System using Messages (BDSM)
// when the brain gets a percept
// it enters a thinking <-> acting loop
// thinking -> digest(prev_information) from memory
//
//	-> draw inferences
//	-> consider adding to memory
//
// acting -> parallel actions
// will be using a Queue for this
func (b *Brain) Decide(ctx context.Context, msgs []types.Message, responseChannel ...*adapters.ContractResponseChannel) (types.Message, error) {
	// Convert messages to contract format
	contractMsgs := make([]adapters.ContractMessage, len(msgs))
	for i, msg := range msgs {
		contractMsgs[i] = msg.ToContractMessage()
	}

	availableTools := b.registry.GetContractTools()

	contractInput := adapters.ContractInput{
		ID:       uuid.New(),
		ToolList: availableTools,
		Meta:     map[string]interface{}{"user_id": msgs[0].UserId.String()}, // Use first message's user ID
		Msgs:     contractMsgs,
		HandlerModel: adapters.ContractSelectedModel{
			Name:    b.defaultModel,
			Version: "8b",
		},
	}

	// Create internal channel if none provided
	var internalChannel adapters.ContractResponseChannel
	var useExternalChannel bool
	if len(responseChannel) > 0 && responseChannel[0] != nil {
		useExternalChannel = true
	} else {
		internalChannel = make(adapters.ContractResponseChannel, 10)
	}

	channelToUse := &internalChannel
	if useExternalChannel {
		channelToUse = responseChannel[0]
	}

	// Start streaming
	go func() {
		defer func() {
			// Only close internal channel
			if !useExternalChannel {
				defer func() {
					if r := recover(); r != nil {
						b.logger.Error(fmt.Sprintf("Recovered from channel close panic in Decide: %v", r))
					}
				}()
				select {
				case <-internalChannel:
					// Channel is already closed
				default:
					close(internalChannel)
				}
			}
		}()

		err := b.mux.Stream(ctx, contractInput, channelToUse)
		if err != nil {
			b.logger.Error(fmt.Sprintf("Decide streaming error: %v", err))
			select {
			case *channelToUse <- []adapters.ContractResponseDelta{{Error: err}}:
			default:
				b.logger.Error("Could not send error through response channel in Decide")
			}
		}
	}()

	// Collect all responses and handle tool execution
	var finalContent string
	var toolCalls []adapters.ContractToolCall
	toolCallsCount := 0
	maxToolCalls := 5

	for deltas := range *channelToUse {
		for _, delta := range deltas {
			// Stream regular content
			if delta.Msg != nil && delta.Msg.Content != "" {
				finalContent += delta.Msg.Content
				// Stream non-tool content through external channel if provided
				if useExternalChannel {
					*responseChannel[0] <- []adapters.ContractResponseDelta{delta}
				}
			}

			// Handle tool calls
			if delta.ToolCalls != nil && len(*delta.ToolCalls) > 0 && toolCallsCount < maxToolCalls {
				newToolCalls := *delta.ToolCalls
				for _, toolCall := range newToolCalls {
					// Stream tool call notification
					toolCallNotification := []adapters.ContractResponseDelta{
						{
							Msg: &adapters.ContractMessage{
								Role:      adapters.ASSISTANT,
								Content:   fmt.Sprintf("🔧 Calling tool: %s", toolCall.ToolName),
								CreatedAt: time.Now(),
							},
							Done:      false,
							CreatedAt: time.Now(),
						},
					}
					if useExternalChannel {
						*responseChannel[0] <- toolCallNotification
					}
				}
				toolCalls = append(toolCalls, newToolCalls...)
			}

			if delta.Error != nil {
				return types.Message{}, fmt.Errorf("decide error: %w", delta.Error)
			}
		}
	}

	// Execute tool calls if any
	if len(toolCalls) > 0 && toolCallsCount < maxToolCalls {
		toolResponses, _ := b.ExecuteToolCallsParallel(ctx, toolCalls)

		// Stream tool results if using external channel
		for _, toolResp := range toolResponses {
			if useExternalChannel {
				toolDelta := []adapters.ContractResponseDelta{
					{
						Msg: &adapters.ContractMessage{
							Role:      adapters.TOOL,
							Content:   toolResp.Text,
							CreatedAt: toolResp.Timestamp,
						},
						Done:      true,
						CreatedAt: toolResp.Timestamp,
					},
				}
				*responseChannel[0] <- toolDelta
			}

			// Also append to final content for return value
			finalContent += fmt.Sprintf("\n[Tool Result: %s]", toolResp.Text)
		}
	}

	return types.Message{
		Id:        uuid.New(),
		UserId:    msgs[len(msgs)-1].UserId,
		Text:      finalContent,
		Timestamp: time.Now(),
		MsgRole:   assistant.ASSISTANT,
		Tags:      []string{"brain_response"},
	}, nil
}

func (b *Brain) ExecuteToolCallsParallel(
	ctx context.Context,
	toolCalls []adapters.ContractToolCall,
) ([]types.Message, int) {
	toolResponses := make([]types.Message, 0)
	// mutex & ...
	wg := sync.WaitGroup{}
	mu := sync.Mutex{}
	for _, toolCall := range toolCalls {
		wg.Add(1)
		go func(tc adapters.ContractToolCall) {
			defer wg.Done()
			// call fn using new executor interface
			result, err := b.executor.Execute(ctx, b.registry, tc)
			var toolResponse types.Message

			if err != nil {
				toolResponse = types.Message{
					Id: uuid.New(),
					// todo: fix this
					UserId:    uuid.New(), // Will be set by caller
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

				toolResponse = types.Message{
					Id: uuid.New(),
					// todo: fix this
					UserId:    uuid.New(), // Will be set by caller
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
	logger *Logger.Logger,
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
