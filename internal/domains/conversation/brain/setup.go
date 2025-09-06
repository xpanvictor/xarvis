package brain

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/xpanvictor/xarvis/internal/config"
	"github.com/xpanvictor/xarvis/internal/domains/conversation"
	"github.com/xpanvictor/xarvis/internal/domains/sys_manager/pipeline"
	"github.com/xpanvictor/xarvis/pkg/Logger"
	"github.com/xpanvictor/xarvis/pkg/assistant/adapters"
	"github.com/xpanvictor/xarvis/pkg/assistant/adapters/ollama"
	olp "github.com/xpanvictor/xarvis/pkg/assistant/providers/ollama"
	"github.com/xpanvictor/xarvis/pkg/assistant/router"
	"github.com/xpanvictor/xarvis/pkg/io"
	"github.com/xpanvictor/xarvis/pkg/io/registry"
	"github.com/xpanvictor/xarvis/pkg/io/tts/piper"
	"github.com/xpanvictor/xarvis/pkg/io/tts/piper/stream"
	toolsystem "github.com/xpanvictor/xarvis/pkg/tool_system"
)

// BrainSystem represents a complete brain system with all dependencies
// It handles the coordination between Brain, Pipeline, and Device Registry
type BrainSystem struct {
	Brain    *Brain
	Registry toolsystem.Registry
	Pipeline *pipeline.Pipeline
	Sessions map[uuid.UUID]*BrainSession // Track user sessions
	logger   Logger.Logger
}

// NewBrainSystem creates a BrainSystem with the provided dependencies
func NewBrainSystem(
	cfg config.BrainConfig,
	globalMux *router.Mux, // Keep for compatibility but create per-user
	deviceReg registry.DeviceRegistry,
	piperURL *url.URL,
	logger Logger.Logger,
) *BrainSystem {
	// Create tool registry and executor
	toolRegistry := toolsystem.NewMemoryRegistry()
	executor := toolsystem.NewExecutor()

	// Register example tools (optional)
	if err := toolsystem.RegisterExampleTools(toolRegistry); err != nil {
		logger.Error("Failed to register example tools: %v", err)
	}

	// Create a dedicated LLM router for this brain system to avoid shared state
	perUserMux, err := createPerUserLLMRouter(logger)
	if err != nil {
		logger.Errorf("Failed to create per-user LLM router, falling back to shared: %v", err)
		perUserMux = globalMux
	}

	// Create TTS and streaming pipeline
	piperClient := piper.New(piperURL.String())
	streamer := stream.New(&piperClient)
	publisher := io.New(deviceReg)
	pipeline := pipeline.New(&streamer, &publisher)

	// Create brain with dedicated dependencies
	defaultModel := "llama3.1:8b-instruct" // Use tool-compatible model
	brain := NewBrain(cfg, toolRegistry, executor, perUserMux, logger, defaultModel)

	return &BrainSystem{
		Brain:    brain,
		Registry: toolRegistry,
		Pipeline: &pipeline,
		Sessions: make(map[uuid.UUID]*BrainSession),
		logger:   logger,
	}
}

// Session management methods for BrainSystem

// GetOrCreateSession gets an existing session or creates a new one for a user
func (bs *BrainSystem) GetOrCreateSession(userID uuid.UUID) *BrainSession {
	if session, exists := bs.Sessions[userID]; exists {
		return session
	}

	session := &BrainSession{
		UserID:    userID,
		SessionID: uuid.New(),
		Messages:  make([]conversation.Message, 0),
	}
	bs.Sessions[userID] = session
	return session
}

// ProcessMessageWithStreaming handles a message and streams the response through pipeline
func (bs *BrainSystem) ProcessMessageWithStreaming(
	ctx context.Context,
	userID uuid.UUID,
	sessionID uuid.UUID,
	msg conversation.Message,
) error {
	session := bs.GetOrCreateSession(userID)
	session.SessionID = sessionID // Update session ID for this connection

	// Add message to session history
	session.Messages = append(session.Messages, msg)

	// Get response channel from brain
	responseChannel, err := bs.Brain.ProcessMessage(ctx, *session, msg)
	if err != nil {
		return fmt.Errorf("brain processing error: %w", err)
	}

	// Handle tool execution and streaming through pipeline
	return bs.handleResponseWithPipeline(ctx, userID, sessionID, responseChannel)
}

// handleResponseWithPipeline processes brain responses, executes tools, and streams through pipeline
func (bs *BrainSystem) handleResponseWithPipeline(
	ctx context.Context,
	userID uuid.UUID,
	sessionID uuid.UUID,
	responseChannel <-chan []adapters.ContractResponseDelta,
) error {
	toolCallsCount := 0
	maxToolCalls := 5 // TODO: Make configurable

	// Convert the readonly channel back to the type expected by pipeline
	pipelineChannel := make(adapters.ContractResponseChannel, 10)

	go func() {
		defer close(pipelineChannel)

		for deltas := range responseChannel {
			var toolCalls []adapters.ContractToolCall
			var hasToolCalls bool

			// Forward deltas to pipeline for streaming
			pipelineChannel <- deltas

			// Check for tool calls
			for _, delta := range deltas {
				if delta.ToolCalls != nil && len(*delta.ToolCalls) > 0 {
					toolCalls = append(toolCalls, *delta.ToolCalls...)
					hasToolCalls = true
				}
			}

			// Execute tool calls if present and under limit
			if hasToolCalls && toolCallsCount < maxToolCalls {
				toolResponses, executedCount := bs.Brain.ExecuteToolCallsParallel(ctx, toolCalls)
				toolCallsCount += executedCount

				// Convert tool responses to contract deltas and send through pipeline
				for _, toolResp := range toolResponses {
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
					pipelineChannel <- toolDelta
				}
			}
		}
	}()

	// Stream through pipeline in a separate goroutine to avoid blocking
	go func() {
		err := bs.Pipeline.Broadcast(ctx, userID, sessionID, &pipelineChannel)
		if err != nil {
			bs.logger.Error("Pipeline broadcast error: %v", err)
		}
	}()

	return nil // Return immediately, streaming happens in background
}

// createPerUserLLMRouter creates a dedicated LLM router for each brain system
// This ensures that each user has their own adapter instances with isolated state
func createPerUserLLMRouter(logger Logger.Logger) (*router.Mux, error) {
	// Parse Ollama URL - use the same configuration as the global router
	ollamaURL, err := url.Parse("http://ollama:11434")
	if err != nil {
		return nil, fmt.Errorf("invalid Ollama URL: %w", err)
	}

	// Create model configuration
	models := []config.LLMModelConfig{
		{
			Name: "llama3.1:8b-instruct",
			Url:  *ollamaURL,
		},
	}

	// Create fresh Ollama provider instance
	provider := olp.New(config.OllamaConfig{
		LLamaModels: models,
	})

	// Create fresh adapter instance with isolated state
	adapter := ollama.New(&provider, adapters.ContractLLMCfg{
		DeltaTimeDuration: 200 * time.Millisecond,
		DeltaBufferLimit:  32,
	}, nil)

	// Create router with the fresh adapter
	adapters := []adapters.ContractAdapter{adapter}
	mux := router.New(adapters)

	logger.Infof("Created dedicated LLM router for brain system with fresh adapter instances")
	return &mux, nil
}
