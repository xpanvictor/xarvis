package brain

import (
	"context"

	"github.com/xpanvictor/xarvis/internal/config"
	"github.com/xpanvictor/xarvis/internal/domains/conversation"
	"github.com/xpanvictor/xarvis/pkg/Logger"
	"github.com/xpanvictor/xarvis/pkg/assistant/adapters"
	ollamaAdapter "github.com/xpanvictor/xarvis/pkg/assistant/adapters/ollama"
	"github.com/xpanvictor/xarvis/pkg/assistant/providers/ollama"
	toolsystem "github.com/xpanvictor/xarvis/pkg/tool_system"
)

// SetupBrainWithOllama creates a Brain instance configured with Ollama adapter and tools
func SetupBrainWithOllama(
	cfg config.BrainConfig,
	ollamaProvider *ollama.OllamaProvider,
	logger Logger.Logger,
) *Brain {
	// Create tool registry and executor
	registry := toolsystem.NewMemoryRegistry()
	executor := toolsystem.NewExecutor()

	// Register example tools (optional)
	if err := toolsystem.RegisterExampleTools(registry); err != nil {
		logger.Error("Failed to register example tools: %v", err)
	}

	// Create contract LLM configuration
	contractCfg := adapters.ContractLLMCfg{
		DeltaTimeDuration: 150, // milliseconds
		DeltaBufferLimit:  24,  // tokens
	}

	// Create Ollama adapter
	adapter := ollamaAdapter.New(ollamaProvider, contractCfg, nil)

	// Create and return brain
	return NewBrain(cfg, registry, executor, adapter, logger)
}

// BrainInterface defines the interface for brain functionality
type BrainInterface interface {
	Decide(ctx context.Context, msg conversation.Message) (conversation.Message, error)
	ExecuteToolCallsParallel(ctx context.Context, toolCalls []adapters.ContractToolCall) ([]conversation.Message, int)
	Think(ctx context.Context, userID string)
}

// Ensure Brain implements BrainInterface
var _ BrainInterface = (*Brain)(nil)
