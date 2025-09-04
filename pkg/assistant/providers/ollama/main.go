package ollama

import (
	"context"
	"fmt"
	"log"

	"github.com/ollama/ollama/api"
	"github.com/presbrey/ollamafarm"
	"github.com/xpanvictor/xarvis/internal/config"
)

// Supports different models
// hence created as necessary

type OllamaProvider struct {
	ollamafarm *ollamafarm.Farm
}

func New(cfg config.OllamaConfig) OllamaProvider {
	farm := ollamafarm.New()

	// register servers
	for _, modelSrv := range cfg.LLamaModels {
		// todo: group name and priority
		err := farm.RegisterURL(modelSrv.Url.String(), nil)
		if err != nil {
			log.Printf("model-sk er %v", err)
		}
	}

	return OllamaProvider{
		ollamafarm: farm,
	}
}

func (o *OllamaProvider) Chat(
	ctx context.Context,
	req api.ChatRequest,
	fn api.ChatResponseFunc,
) error {
	// pick first available client
	ollama := o.ollamafarm.First(&ollamafarm.Where{Offline: false})
	if ollama != nil {
		return ollama.Client().Chat(ctx, &req, fn)
	}
	// error here
	return fmt.Errorf("model not found, %v", req.Model)
}

func (o *OllamaProvider) GetAvailableModels() []string {
	panic("unimp")
}
