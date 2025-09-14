package processor

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/generative-ai-go/genai"
	"github.com/xpanvictor/xarvis/pkg/Logger"
	"google.golang.org/api/option"
)

// GeminiProcessor implements Processor using Google Gemini API
type GeminiProcessor struct {
	client *genai.Client
	model  *genai.GenerativeModel
	logger *Logger.Logger
}

// GeminiConfig holds configuration for Gemini processor
type GeminiConfig struct {
	APIKey    string
	ModelName string // e.g., "gemini-1.5-flash"
}

// NewGeminiProcessor creates a new Gemini processor
func NewGeminiProcessor(config GeminiConfig, logger *Logger.Logger) (*GeminiProcessor, error) {
	ctx := context.Background()
	
	client, err := genai.NewClient(ctx, option.WithAPIKey(config.APIKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}
	
	modelName := config.ModelName
	if modelName == "" {
		modelName = "gemini-1.5-flash" // default model
	}
	
	model := client.GenerativeModel(modelName)
	
	// Configure model for JSON responses
	model.ResponseMIMEType = "application/json"
	model.Temperature = &[]float32{0.1}[0] // Low temperature for consistency
	
	return &GeminiProcessor{
		client: client,
		model:  model,
		logger: logger,
	}, nil
}

// Process implements Processor.Process
func (g *GeminiProcessor) Process(ctx context.Context, instruction string, input interface{}) (interface{}, error) {
	// Convert input to JSON string
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal input: %w", err)
	}
	
	// Create the prompt
	prompt := fmt.Sprintf("%s\n\nInput Data:\n%s\n\nPlease respond with valid JSON only.", 
		instruction, string(inputJSON))
	
	g.logger.Debug(fmt.Sprintf("Gemini processor prompt: %s", prompt))
	
	// Generate response
	resp, err := g.model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, fmt.Errorf("failed to generate content: %w", err)
	}
	
	if len(resp.Candidates) == 0 {
		return nil, fmt.Errorf("no response candidates received")
	}
	
	// Extract text response
	var responseText string
	for _, part := range resp.Candidates[0].Content.Parts {
		if textPart, ok := part.(genai.Text); ok {
			responseText += string(textPart)
		}
	}
	
	if responseText == "" {
		return nil, fmt.Errorf("empty response received")
	}
	
	g.logger.Debug(fmt.Sprintf("Gemini response: %s", responseText))
	
	// Parse JSON response
	var result interface{}
	if err := json.Unmarshal([]byte(responseText), &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}
	
	return result, nil
}

// ProcessWithType implements Processor.ProcessWithType
func (g *GeminiProcessor) ProcessWithType(ctx context.Context, instruction string, input interface{}, responseType interface{}) error {
	// Convert input to JSON string
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("failed to marshal input: %w", err)
	}
	
	// Get the expected response schema
	exampleJSON, err := json.Marshal(responseType)
	if err != nil {
		return fmt.Errorf("failed to marshal response type: %w", err)
	}
	
	// Create the prompt with schema example
	prompt := fmt.Sprintf(`%s

Input Data:
%s

Expected Response Format (please match this exact structure):
%s

Please respond with valid JSON only, matching the expected format exactly.`, 
		instruction, string(inputJSON), string(exampleJSON))
	
	g.logger.Debug(fmt.Sprintf("Gemini processor prompt: %s", prompt))
	
	// Generate response
	resp, err := g.model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return fmt.Errorf("failed to generate content: %w", err)
	}
	
	if len(resp.Candidates) == 0 {
		return fmt.Errorf("no response candidates received")
	}
	
	// Extract text response
	var responseText string
	for _, part := range resp.Candidates[0].Content.Parts {
		if textPart, ok := part.(genai.Text); ok {
			responseText += string(textPart)
		}
	}
	
	if responseText == "" {
		return fmt.Errorf("empty response received")
	}
	
	g.logger.Debug(fmt.Sprintf("Gemini response: %s", responseText))
	
	// Parse JSON response into the provided type
	if err := json.Unmarshal([]byte(responseText), responseType); err != nil {
		return fmt.Errorf("failed to parse JSON response into expected type: %w", err)
	}
	
	return nil
}

// Close cleans up the Gemini processor
func (g *GeminiProcessor) Close() error {
	if g.client != nil {
		return g.client.Close()
	}
	return nil
}
