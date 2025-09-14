package brain

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/xpanvictor/xarvis/internal/config"
	"github.com/xpanvictor/xarvis/internal/domains/sys_manager/pipeline"
	"github.com/xpanvictor/xarvis/internal/types"
	"github.com/xpanvictor/xarvis/pkg/Logger"
	"github.com/xpanvictor/xarvis/pkg/assistant/adapters"
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
	logger   *Logger.Logger
}

// NewBrainSystem creates a BrainSystem with the provided dependencies
func NewBrainSystem(
	cfg config.BrainConfig,
	globalMux *router.Mux, // Use the shared global router
	deviceReg registry.DeviceRegistry,
	piperURL *url.URL,
	logger *Logger.Logger,
) *BrainSystem {
	// Create tool registry and executor
	toolRegistry := toolsystem.NewMemoryRegistry()
	executor := toolsystem.NewExecutor()

	// Register example tools (optional, can be disabled in production)
	if err := toolsystem.RegisterExampleTools(toolRegistry); err != nil {
		logger.Error("Failed to register example tools: %v", err)
	}

	// Create TTS and streaming pipeline
	piperClient := piper.New(piperURL.String())
	streamer := stream.New(&piperClient)
	publisher := io.New(deviceReg)
	pipeline := pipeline.New(&streamer, &publisher)

	// Create brain with shared global router (no longer per-user)
	defaultModel := "llama3.1:8b-instruct" // Use tool-compatible model
	brain := NewBrain(cfg, toolRegistry, executor, *globalMux, logger, defaultModel)

	return &BrainSystem{
		Brain:    brain,
		Registry: toolRegistry,
		Pipeline: &pipeline,
		logger:   logger,
	}
}

// RegisterAppTools registers application-specific tools from the tool factory
// This function is called from app setup to avoid import cycles
func (bs *BrainSystem) RegisterAppTools(toolFactory ToolFactory) error {
	if toolFactory == nil {
		bs.logger.Warn("Tool factory is nil, skipping app tools registration")
		return nil
	}

	// Get all built tools from the factory
	tools := toolFactory.GetTools()
	if len(tools) == 0 {
		bs.logger.Info("No tools available from factory")
		return nil
	}

	// Register each tool with the brain registry
	for name, tool := range tools {
		if err := bs.Registry.Register(tool); err != nil {
			return fmt.Errorf("failed to register tool '%s': %w", name, err)
		}
		bs.logger.Info("Registered tool: %s", name)
	}

	bs.logger.Info("Successfully registered %d application tools", len(tools))
	return nil
}

// ToolFactory interface to avoid import cycle
type ToolFactory interface {
	GetTools() map[string]toolsystem.Tool
}

func (bs *BrainSystem) ProcessMessage(
	ctx context.Context,
	userID uuid.UUID,
	sessionID uuid.UUID,
	msgs []types.Message,
) (*types.Message, error) {
	// Set user context on the executor before processing
	userCtx := &toolsystem.UserContext{
		UserID:          userID,
		Username:        "user",             // TODO: Get actual username from user service
		UserEmail:       "user@example.com", // TODO: Get actual email from user service
		CurrentDateTime: time.Now(),
	}
	bs.Brain.SetUserContext(userCtx)

	msg, err := bs.Brain.Decide(ctx, msgs, nil)
	return &msg, err
}

// ProcessMessageWithStreaming handles a message and streams the response through pipeline
func (bs *BrainSystem) ProcessMessageWithStreaming(
	ctx context.Context,
	userID uuid.UUID,
	sessionID uuid.UUID,
	msgs []types.Message,
	disableAudio bool,
) error {
	// Set user context on the executor before processing
	bs.logger.Infof("This users id is %v", userID)
	userCtx := &toolsystem.UserContext{
		UserID:          userID,
		Username:        "user",             // TODO: Get actual username from user service
		UserEmail:       "user@example.com", // TODO: Get actual email from user service
		CurrentDateTime: time.Now(),
	}
	bs.Brain.SetUserContext(userCtx)

	// Create response channel for streaming
	responseChannel := make(adapters.ContractResponseChannel, 10)

	// Start Brain.Decide with streaming in background
	go func() {
		defer func() {
			if r := recover(); r != nil {
				bs.logger.Error("Recovered from panic in brain goroutine: %v", r)
			}
		}()

		bs.logger.Info("Brain processing starting...")
		_, err := bs.Brain.Decide(ctx, msgs, &responseChannel)
		if err != nil {
			bs.logger.Error("Brain decide error: %v", err)
		}
		bs.logger.Info("Brain processing completed")
	}()

	// Pipeline reads from channel until it's closed (blocks until brain is done)
	bs.logger.Info("Pipeline starting...")
	err := bs.Pipeline.Broadcast(ctx, userID, sessionID, &responseChannel, disableAudio)
	if err != nil {
		bs.logger.Error("Pipeline broadcast error: %v", err)
	}
	bs.logger.Info("Pipeline completed")

	return nil
}
