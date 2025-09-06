package gemini

import (
	"context"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"github.com/xpanvictor/xarvis/internal/config"
	"google.golang.org/api/option"
)

// GeminiProvider supports different models, similar to your Ollama example.
type GeminiProvider struct {
	client *genai.Client
}

// New creates a new GeminiProvider instance.
func New(cfg config.GeminiConfig) (*GeminiProvider, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("gemini API key is not configured")
	}

	client, err := genai.NewClient(context.Background(), option.WithAPIKey(cfg.APIKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini API client: %w", err)
	}

	return &GeminiProvider{
		client: client,
	}, nil
}

// Chat streams responses from the Gemini API.
func (gp *GeminiProvider) Chat(
	ctx context.Context,
	modelName string,
	iter *genai.GenerateContentResponseIterator,
	fn func(resp *genai.GenerateContentResponse) error,
) error {
	if gp.client == nil {
		return fmt.Errorf("gemini client is not initialized")
	}

	for {
		resp, err := iter.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			// Check if this is the normal "no more items" condition
			if strings.Contains(err.Error(), "no more items") || strings.Contains(err.Error(), "iterator stopped") {
				break
			}
			return fmt.Errorf("failed to receive from Gemini stream: %w", err)
		}

		if err := fn(resp); err != nil {
			log.Printf("Error processing Gemini stream chunk: %v", err)
			return err
		}
	}

	return nil
}

func (gp *GeminiProvider) GetModel(modelName string) *genai.GenerativeModel {
	return gp.client.GenerativeModel(modelName)
}

// GetAvailableModels returns a list of available Gemini models.
func (gp *GeminiProvider) GetAvailableModels() []string {
	// In a real application, you'd query the Gemini API for supported models.
	return []string{"gemini-1.5-flash-latest", "gemini-pro"}
}
