package embedding

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/xpanvictor/xarvis/internal/database/dbtypes"
	"github.com/xpanvictor/xarvis/pkg/Logger"
	"google.golang.org/genai"
)

type GeminiEmbedder struct {
	client    *genai.Client
	logger    *Logger.Logger
	maxTokens int // Maximum tokens per chunk
	modelName string
}

// NewGeminiEmbedder creates a new Gemini embedder
func NewGeminiEmbedder(apiKey string, logger *Logger.Logger) (*GeminiEmbedder, error) {
	ctx := context.Background()

	// Set the API key as environment variable (required by genai client)
	os.Setenv("GOOGLE_API_KEY", apiKey)

	client, err := genai.NewClient(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %v", err)
	}

	return &GeminiEmbedder{
		client:    client,
		logger:    logger,
		maxTokens: 2048,                 // Gemini embedding models typically support more tokens
		modelName: "text-embedding-004", // Latest Gemini embedding model
	}, nil
}

// Chunk implements Embedder interface
func (e *GeminiEmbedder) Chunk(text string) []string {
	if len(text) == 0 {
		return []string{}
	}

	// Rough estimation: 1 token ≈ 4 characters for English text
	// Use conservative estimate to avoid hitting token limits
	maxChars := e.maxTokens * 3

	if len(text) <= maxChars {
		return []string{text}
	}

	var chunks []string
	sentences := e.splitIntoSentences(text)

	var currentChunk strings.Builder
	for _, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if sentence == "" {
			continue
		}

		// If adding this sentence would exceed the limit, save current chunk
		if currentChunk.Len() > 0 && currentChunk.Len()+len(sentence)+1 > maxChars {
			chunks = append(chunks, strings.TrimSpace(currentChunk.String()))
			currentChunk.Reset()
		}

		// If a single sentence is too long, split it further
		if len(sentence) > maxChars {
			subChunks := e.splitLongSentence(sentence, maxChars)
			for i, subChunk := range subChunks {
				if i == 0 && currentChunk.Len() > 0 {
					// Add first subchunk to current chunk if there's space
					if currentChunk.Len()+len(subChunk)+1 <= maxChars {
						if currentChunk.Len() > 0 {
							currentChunk.WriteString(" ")
						}
						currentChunk.WriteString(subChunk)
					} else {
						chunks = append(chunks, strings.TrimSpace(currentChunk.String()))
						currentChunk.Reset()
						currentChunk.WriteString(subChunk)
					}
				} else {
					if currentChunk.Len() > 0 {
						chunks = append(chunks, strings.TrimSpace(currentChunk.String()))
						currentChunk.Reset()
					}
					currentChunk.WriteString(subChunk)
				}
			}
		} else {
			if currentChunk.Len() > 0 {
				currentChunk.WriteString(" ")
			}
			currentChunk.WriteString(sentence)
		}
	}

	// Add the last chunk if it's not empty
	if currentChunk.Len() > 0 {
		chunks = append(chunks, strings.TrimSpace(currentChunk.String()))
	}

	// Ensure we don't return empty chunks
	var filteredChunks []string
	for _, chunk := range chunks {
		if strings.TrimSpace(chunk) != "" {
			filteredChunks = append(filteredChunks, strings.TrimSpace(chunk))
		}
	}

	if len(filteredChunks) == 0 {
		return []string{text}
	}

	return filteredChunks
}

// splitIntoSentences splits text into sentences
func (e *GeminiEmbedder) splitIntoSentences(text string) []string {
	// Simple sentence splitting - can be improved with more sophisticated NLP
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.ReplaceAll(text, "\r", " ")

	var sentences []string
	var current strings.Builder

	for i, r := range text {
		current.WriteRune(r)

		// Check for sentence endings
		if r == '.' || r == '!' || r == '?' {
			// Look ahead to see if this is actually the end of a sentence
			nextIdx := i + 1
			if nextIdx < len([]rune(text)) {
				nextRune := []rune(text)[nextIdx]
				// If next character is whitespace or end of text, treat as sentence end
				if unicode.IsSpace(nextRune) {
					sentences = append(sentences, strings.TrimSpace(current.String()))
					current.Reset()
				}
			} else {
				// End of text
				sentences = append(sentences, strings.TrimSpace(current.String()))
				current.Reset()
			}
		}
	}

	// Add remaining text as last sentence
	if current.Len() > 0 {
		sentences = append(sentences, strings.TrimSpace(current.String()))
	}

	return sentences
}

// splitLongSentence splits a sentence that's too long into smaller parts
func (e *GeminiEmbedder) splitLongSentence(sentence string, maxChars int) []string {
	words := strings.Fields(sentence)
	var chunks []string
	var current strings.Builder

	for _, word := range words {
		// If adding this word would exceed limit, save current chunk
		if current.Len() > 0 && current.Len()+len(word)+1 > maxChars {
			chunks = append(chunks, strings.TrimSpace(current.String()))
			current.Reset()
		}

		// If a single word is too long, split it (rare edge case)
		if len(word) > maxChars {
			chunks = append(chunks, word[:maxChars])
			word = word[maxChars:]
			current.Reset()
		}

		if current.Len() > 0 {
			current.WriteString(" ")
		}
		current.WriteString(word)
	}

	if current.Len() > 0 {
		chunks = append(chunks, strings.TrimSpace(current.String()))
	}

	return chunks
}

// Embed implements Embedder interface
func (e *GeminiEmbedder) Embed(ctx context.Context, chunks []string) ([]dbtypes.XVector, error) {
	if len(chunks) == 0 {
		return []dbtypes.XVector{}, nil
	}

	// Filter out empty chunks
	var validChunks []string
	for _, chunk := range chunks {
		if strings.TrimSpace(chunk) != "" {
			validChunks = append(validChunks, strings.TrimSpace(chunk))
		}
	}

	if len(validChunks) == 0 {
		return []dbtypes.XVector{}, nil
	}

	// Create contents for Gemini API
	var contents []*genai.Content
	for _, chunk := range validChunks {
		contents = append(contents, genai.NewContentFromText(chunk, genai.RoleUser))
	}

	// Add timeout to context
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Call Gemini embedding API
	result, err := e.client.Models.EmbedContent(ctx, e.modelName, contents, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get embeddings from Gemini: %v", err)
	}

	if len(result.Embeddings) != len(validChunks) {
		return nil, fmt.Errorf("expected %d embeddings, got %d", len(validChunks), len(result.Embeddings))
	}

	// Convert Gemini embeddings to XVector format
	embeddings := make([]dbtypes.XVector, len(result.Embeddings))
	for i, embedding := range result.Embeddings {
		// Convert from Gemini embedding format to float32 slice
		if embedding.Values == nil {
			return nil, fmt.Errorf("embedding %d has no values", i)
		}

		// Values are already float32 in the API response
		embeddings[i] = dbtypes.XVector(embedding.Values)
	}

	e.logger.Debug("Successfully generated %d embeddings using Gemini", len(embeddings))
	return embeddings, nil
}

// EmbedSingle implements Embedder interface
func (e *GeminiEmbedder) EmbedSingle(ctx context.Context, text string) (dbtypes.XVector, error) {
	if strings.TrimSpace(text) == "" {
		return dbtypes.XVector{}, nil
	}

	embeddings, err := e.Embed(ctx, []string{text})
	if err != nil {
		return nil, err
	}

	if len(embeddings) == 0 {
		return dbtypes.XVector{}, nil
	}

	return embeddings[0], nil
}
