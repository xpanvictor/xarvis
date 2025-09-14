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
}

// sessionBuffer keeps the active processing context of one Decide call
type sessionBuffer struct {
	allMsgs []adapters.ContractMessage
	mu      sync.Mutex
}

func (s *sessionBuffer) Append(msgs ...adapters.ContractMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.allMsgs = append(s.allMsgs, msgs...)
}

func (s *sessionBuffer) Snapshot() []adapters.ContractMessage {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]adapters.ContractMessage, len(s.allMsgs))
	copy(out, s.allMsgs)
	return out
}

// Brain Decision System using Messages (BDSM)
func (b *Brain) Decide(ctx context.Context, msgs []types.Message, outCh *adapters.ContractResponseChannel) (types.Message, error) {
	session := &sessionBuffer{allMsgs: make([]adapters.ContractMessage, 0)}

	// seed initial user messages into session
	for _, msg := range msgs {
		session.Append(msg.ToContractMessage())
	}

	// processing queue (each step may append)
	cntrctMsgQueue := []adapters.ContractInput{
		{
			Msgs:     session.Snapshot(),
			ID:       uuid.New(),
			ToolList: b.registry.GetContractTools(),
			HandlerModel: adapters.ContractSelectedModel{
				Name:    b.defaultModel,
				Version: "8b",
			},
		},
	}

	toolCallsCount := 0
	var finalMessage types.Message
	round := 0

	// loop until no more queued inputs
	for len(cntrctMsgQueue) > 0 {

		// per-round cancelable context
		roundCtx, cancel := context.WithCancel(ctx)

		// buffered channel to reduce blocking
		inputCh := make(adapters.ContractResponseChannel, 32)

		currCntrIn := cntrctMsgQueue[0]
		cntrctMsgQueue = cntrctMsgQueue[1:]

		// start streaming
		go b.processMsg(roundCtx, currCntrIn, inputCh)

		toolCalls := make([]adapters.ContractToolCall, 0)
		messageBuffer := ""
		var lastTimestamp time.Time
		sawToolCalls := false

		// READ LOOP
		for elems := range inputCh {
			for _, elem := range elems {

				// STRICT TOOL MODE
				if len(elem.ToolCalls) > 0 {
					toolCalls = append(toolCalls, elem.ToolCalls...)
					sawToolCalls = true
					messageBuffer = "" // wipe assistant buffer
					cancel()           // stop producer
					continue           // still drain until channel closes
				}

				if sawToolCalls {
					// ignore assistant text after tool call
					continue
				}

				if elem.Msg != nil {
					messageBuffer += elem.Msg.Content
					lastTimestamp = elem.Msg.CreatedAt
					if outCh != nil {
						*outCh <- []adapters.ContractResponseDelta{elem}
					}
				}
			}
		}

		// ensure we don’t leak round context
		cancel()

		// commit assistant message if we got one and no tool calls
		if messageBuffer != "" && !sawToolCalls {
			finalMessage = types.Message{
				Id:        uuid.New(),
				UserId:    uuid.New(), // TODO: set by caller
				Text:      messageBuffer,
				Timestamp: lastTimestamp,
				MsgRole:   assistant.ASSISTANT,
				Tags:      []string{"llm_response"},
			}
			session.Append(finalMessage.ToContractMessage())
		}

		// if tools requested, execute them and enqueue next round
		if sawToolCalls {
			toolMsgs, callLength := b.ExecuteToolCallsParallel(ctx, toolCalls)
			if len(toolMsgs) > 0 {
				session.Append(toolMsgs...)
			}

			if toolCallsCount >= b.cfg.MaxToolCallLimit {
				break
			}
			toolCallsCount += callLength

			if callLength > 0 {
				cntrctMsgQueue = append(cntrctMsgQueue,
					adapters.ContractInput{
						Msgs:     session.Snapshot(),
						ID:       uuid.New(),
						ToolList: b.registry.GetContractTools(),
						HandlerModel: adapters.ContractSelectedModel{
							Name:    b.defaultModel,
							Version: "8b",
						},
					},
				)
			}
		}

		round++
	}

	return finalMessage, nil
}

func (b *Brain) processMsg(ctx context.Context, contractInput adapters.ContractInput, responseChannel adapters.ContractResponseChannel) {
	err := b.mux.Stream(ctx, contractInput, &responseChannel)
	if err != nil {
		b.logger.Error(fmt.Sprintf("Mux streaming error: %v", err))
		select {
		case responseChannel <- []adapters.ContractResponseDelta{{Error: err}}:
		default:
			b.logger.Error("Could not send error through response channel")
		}
	}
	// mux.Stream should close responseChannel or exit when ctx is canceled
}

func (b *Brain) ExecuteToolCallsParallel(
	ctx context.Context,
	toolCalls []adapters.ContractToolCall,
) ([]adapters.ContractMessage, int) {
	toolResponses := make([]types.Message, 0)
	b.logger.Infof("Received calls %+v", toolCalls)

	wg := sync.WaitGroup{}
	mu := sync.Mutex{}
	for _, toolCall := range toolCalls {
		wg.Add(1)
		go func(tc adapters.ContractToolCall) {
			defer wg.Done()
			result, err := b.executor.Execute(ctx, b.registry, tc)
			var toolResponse types.Message
			if err != nil {
				toolResponse = types.Message{
					Id:        uuid.New(),
					UserId:    uuid.New(),
					Text:      fmt.Sprintf("Tool execution failed: %v", err),
					Timestamp: time.Now(),
					MsgRole:   assistant.TOOL,
					Tags:      []string{"error", "tool_call"},
				}
			} else {
				resultStr := "Tool execution for user request completed. Now explain result to user properly."
				if result.Result != nil {
					if content, ok := result.Result["content"].(string); ok {
						resultStr = content
					} else if jsonBytes, err := json.Marshal(result.Result); err == nil {
						resultStr = string(jsonBytes)
					}
				}
				toolResponse = types.Message{
					Id:        uuid.New(),
					UserId:    uuid.New(),
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

	toolMsgs := make([]adapters.ContractMessage, 0, len(toolResponses))
	for _, msg := range toolResponses {
		toolMsgs = append(toolMsgs, msg.ToContractMessage())
	}
	return toolMsgs, len(toolCalls)
}

// stubs for Think path
func (b *Brain) Think(ctx context.Context, userID string) {
	ok := b.acquireMindLock(ctx, userID)
	if !ok {
		return
	}
	userCtxInfo, _ := b.gatherUserContextInfo(ctx, userID)
	reflection, _ := b.reflect(ctx, userID, *userCtxInfo)
	plan, err := b.plan(ctx, *reflection)
	if err != nil {
		return
	}
	b.logger.Debug(plan)
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
	return true
}

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
