package app

import (
	"fmt"
	"net/url"
	"time"

	"github.com/xpanvictor/xarvis/internal/config"
	"github.com/xpanvictor/xarvis/pkg/Logger"
	"github.com/xpanvictor/xarvis/pkg/assistant/adapters"
	"github.com/xpanvictor/xarvis/pkg/assistant/adapters/ollama"
	olp "github.com/xpanvictor/xarvis/pkg/assistant/providers/ollama"
	"github.com/xpanvictor/xarvis/pkg/assistant/router"
)

// LLMConfig represents LLM configuration options
type LLMConfig struct {
	DefaultDeltaTime time.Duration
	DefaultBuffer    int
	OllamaURL        string
	Models           []string
}

// DefaultLLMConfig returns sensible defaults for LLM configuration,
// with overrides from application config if available
func DefaultLLMConfig() LLMConfig {
	return LLMConfig{
		DefaultDeltaTime: 200 * time.Millisecond,
		DefaultBuffer:    32,
		OllamaURL:        "http://ollama:11434", // Use config from yaml
		Models:           []string{"llama3.1:8b-instruct"},
	}
}

// LLMConfigFromSettings creates LLM configuration from application settings
func LLMConfigFromSettings(cfg *config.Settings) LLMConfig {
	// Start with defaults
	llmConfig := DefaultLLMConfig()

	// TODO: Override with values from cfg.ModelRouter when the struct is available
	// For now, we use the defaults which match the config file values

	return llmConfig
}

// LLMRouterFactory creates LLM routers with different provider configurations
type LLMRouterFactory struct {
	config LLMConfig
	logger *Logger.Logger
}

// NewLLMRouterFactory creates a new LLM router factory
func NewLLMRouterFactory(config LLMConfig, logger *Logger.Logger) *LLMRouterFactory {
	return &LLMRouterFactory{
		config: config,
		logger: logger,
	}
}

// CreateRouter creates an LLM router with configured providers
func (f *LLMRouterFactory) CreateRouter() (*router.Mux, error) {
	var adapters []adapters.ContractAdapter

	// Create Ollama provider if configured
	if f.config.OllamaURL != "" {
		ollamaAdapter, err := f.createOllamaAdapter()
		if err != nil {
			return nil, fmt.Errorf("failed to create Ollama adapter: %w", err)
		}
		adapters = append(adapters, ollamaAdapter)
	}

	if len(adapters) == 0 {
		return nil, fmt.Errorf("no LLM adapters configured")
	}

	mux := router.New(adapters)
	f.logger.Infof("LLM router created with %d adapter(s)", len(adapters))

	return &mux, nil
}

// createOllamaAdapter creates an Ollama adapter with the configured settings
func (f *LLMRouterFactory) createOllamaAdapter() (adapters.ContractAdapter, error) {
	// Parse Ollama URL
	ollamaURL, err := url.Parse(f.config.OllamaURL)
	if err != nil {
		return nil, fmt.Errorf("invalid Ollama URL: %w", err)
	}

	// Create model configurations
	var models []config.LLMModelConfig
	for _, modelName := range f.config.Models {
		models = append(models, config.LLMModelConfig{
			Name: modelName,
			Url:  *ollamaURL,
		})
	}

	// Create Ollama provider
	provider := olp.New(config.OllamaConfig{
		LLamaModels: models,
	})

	// Create adapter with configured batching
	adapter := ollama.New(&provider, adapters.ContractLLMCfg{
		DeltaTimeDuration: f.config.DefaultDeltaTime,
		DeltaBufferLimit:  uint(f.config.DefaultBuffer),
	}, nil)

	f.logger.Infof("Ollama adapter created for URL: %s, Models: %v", f.config.OllamaURL, f.config.Models)
	return adapter, nil
}
