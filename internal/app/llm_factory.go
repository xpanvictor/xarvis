package app

import (
	"fmt"
	"time"

	"github.com/xpanvictor/xarvis/internal/config"
	"github.com/xpanvictor/xarvis/pkg/Logger"
	"github.com/xpanvictor/xarvis/pkg/assistant/adapters"
	geminiadapter "github.com/xpanvictor/xarvis/pkg/assistant/adapters/gemini"
	"github.com/xpanvictor/xarvis/pkg/assistant/adapters/ollama"
	geminiprovider "github.com/xpanvictor/xarvis/pkg/assistant/providers/gemini"
	olpprovider "github.com/xpanvictor/xarvis/pkg/assistant/providers/ollama"
	"github.com/xpanvictor/xarvis/pkg/assistant/router"
)

const (
	defaultDeltaTime = 200 * time.Millisecond
	defaultBuffer    = 32
)

// LLMRouterFactory creates LLM routers with different provider configurations.
type LLMRouterFactory struct {
	config *config.Settings
	logger *Logger.Logger
}

// NewLLMRouterFactory creates a new LLM router factory.
func NewLLMRouterFactory(config *config.Settings, logger *Logger.Logger) *LLMRouterFactory {
	return &LLMRouterFactory{
		config: config,
		logger: logger,
	}
}

// CreateRouter creates an LLM router with configured providers.
func (f *LLMRouterFactory) CreateRouter() (*router.Mux, error) {
	var createdAdapters []adapters.ContractAdapter

	// Prioritize Gemini as the default if configured.
	if f.config.AssistantKeys.Gemini.APIKey != "" {
		geminiAdapter, err := f.createGeminiAdapter()
		if err != nil {
			f.logger.Warnf("failed to create Gemini adapter, skipping: %v", err)
		} else {
			createdAdapters = append(createdAdapters, geminiAdapter)
		}
	}

	// Add Ollama provider if configured.
	if len(f.config.AssistantKeys.OllamaCredentials.LLamaModels) > 0 {
		ollamaAdapter, err := f.createOllamaAdapter()
		if err != nil {
			f.logger.Warnf("failed to create Ollama adapter, skipping: %v", err)
		} else {
			createdAdapters = append(createdAdapters, ollamaAdapter)
		}
	}

	if len(createdAdapters) == 0 {
		return nil, fmt.Errorf("no LLM adapters could be configured, please check API keys and settings")
	}

	mux := router.New(createdAdapters)
	f.logger.Infof("LLM router created with %d adapter(s)", len(createdAdapters))

	return &mux, nil
}

// createGeminiAdapter creates a Gemini adapter.
func (f *LLMRouterFactory) createGeminiAdapter() (adapters.ContractAdapter, error) {
	provider, err := geminiprovider.New(f.config.AssistantKeys.Gemini)
	if err != nil {
		return nil, fmt.Errorf("failed to create gemini provider: %w", err)
	}

	adapter := geminiadapter.New(provider, adapters.ContractLLMCfg{
		DeltaTimeDuration: defaultDeltaTime,
		DeltaBufferLimit:  uint(defaultBuffer),
	}, nil)

	f.logger.Info("Gemini adapter created successfully.")
	return adapter, nil
}

// createOllamaAdapter creates an Ollama adapter.
func (f *LLMRouterFactory) createOllamaAdapter() (adapters.ContractAdapter, error) {
	provider := olpprovider.New(f.config.AssistantKeys.OllamaCredentials)

	adapter := ollama.New(&provider, adapters.ContractLLMCfg{
		DeltaTimeDuration: defaultDeltaTime,
		DeltaBufferLimit:  uint(defaultBuffer),
	}, nil)

	f.logger.Infof("Ollama adapter created for %d models.", len(f.config.AssistantKeys.OllamaCredentials.LLamaModels))
	return adapter, nil
}
