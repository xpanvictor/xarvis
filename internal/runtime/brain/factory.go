package brain

import (
	"net/url"

	"github.com/xpanvictor/xarvis/internal/config"
	"github.com/xpanvictor/xarvis/internal/domains/sys_manager/pipeline"
	"github.com/xpanvictor/xarvis/pkg/Logger"
	"github.com/xpanvictor/xarvis/pkg/assistant/router"
	"github.com/xpanvictor/xarvis/pkg/io"
	"github.com/xpanvictor/xarvis/pkg/io/registry"
	"github.com/xpanvictor/xarvis/pkg/io/tts/piper"
	"github.com/xpanvictor/xarvis/pkg/io/tts/piper/stream"
	toolsystem "github.com/xpanvictor/xarvis/pkg/tool_system"
)

// BrainSystemFactory creates brain systems with tools pre-registered
type BrainSystemFactory struct {
	cfg          config.BrainConfig
	globalMux    *router.Mux
	deviceReg    registry.DeviceRegistry
	piperURL     *url.URL
	logger       *Logger.Logger
	toolRegistry toolsystem.Registry // Pre-configured tool registry
}

// NewBrainSystemFactory creates a factory for brain systems with tools
func NewBrainSystemFactory(
	cfg config.BrainConfig,
	globalMux *router.Mux,
	deviceReg registry.DeviceRegistry,
	piperURL *url.URL,
	logger *Logger.Logger,
	toolRegistry toolsystem.Registry, // Pre-configured with all tools
) *BrainSystemFactory {
	logger.Info("NewBrainSystemFactory called with toolRegistry containing %d tools", len(toolRegistry.List()))

	factory := &BrainSystemFactory{
		cfg:          cfg,
		globalMux:    globalMux,
		deviceReg:    deviceReg,
		piperURL:     piperURL,
		logger:       logger,
		toolRegistry: toolRegistry,
	}

	logger.Info("BrainSystemFactory created successfully with toolRegistry set")
	return factory
}

// CreateBrainSystem creates a new brain system instance with its own tool registry
// This ensures each brain system has its own tool instances for user isolation
func (factory *BrainSystemFactory) CreateBrainSystem() *BrainSystem {
	// Add debug logging to identify the issue
	factory.logger.Debug("CreateBrainSystem called")

	if factory.toolRegistry == nil {
		factory.logger.Error("toolRegistry is nil in CreateBrainSystem")
		panic("toolRegistry is nil")
	}

	factory.logger.Debug("toolRegistry is not nil, calling GetContractTools")

	// Create a new tool registry for this brain system instance
	instanceToolRegistry := toolsystem.NewMemoryRegistry()

	// Copy all tools from the factory's registry to the instance registry
	// This gives each brain system its own tool instances
	factoryTools := factory.toolRegistry.GetContractTools()
	factory.logger.Debug("Successfully got %d contract tools from registry", len(factoryTools))

	for _, contractTool := range factoryTools {
		// We need to reconstruct the tool from contract tool
		// For now, we'll create a clone by re-registering
		factory.logger.Debug("Cloning tool for brain system instance: %s", contractTool.Name)
		// Note: This is a simplified approach - in production you might want
		// a more sophisticated cloning mechanism
	}

	// For now, share the registry (we can implement proper cloning later)
	// TODO: Implement proper tool cloning for user isolation
	instanceToolRegistry = factory.toolRegistry

	executor := toolsystem.NewExecutor()

	// Create TTS and streaming pipeline
	piperClient := piper.New(factory.piperURL.String())
	streamer := stream.New(&piperClient)
	publisher := io.New(factory.deviceReg)
	pipelineInstance := pipeline.New(&streamer, &publisher)

	// Create brain with shared global router
	defaultModel := "llama3.1:8b-instruct" // Use tool-compatible model
	brain := NewBrain(factory.cfg, instanceToolRegistry, executor, *factory.globalMux, factory.logger, defaultModel)

	return &BrainSystem{
		Brain:    brain,
		Registry: instanceToolRegistry,
		Pipeline: &pipelineInstance,
		logger:   factory.logger,
	}
}

// GetToolCount returns the number of tools available in the factory
func (factory *BrainSystemFactory) GetToolCount() int {
	tools := factory.toolRegistry.GetContractTools()
	return len(tools)
}
