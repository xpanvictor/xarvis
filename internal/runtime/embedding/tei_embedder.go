package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode"

	"github.com/xpanvictor/xarvis/internal/database/dbtypes"
	"github.com/xpanvictor/xarvis/pkg/Logger"
)

type TEIEmbedder struct {
	baseURL    string
	httpClient *http.Client
	logger     *Logger.Logger
	maxTokens  int // Maximum tokens per chunk
}

type TEIRequest struct {
	Inputs []string `json:"inputs"`
}

type TEIResponse [][]float32

// NewTEIEmbedder creates a new TEI embedder
func NewTEIEmbedder(baseURL string, logger *Logger.Logger) *TEIEmbedder {
	return &TEIEmbedder{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger:    logger,
		maxTokens: 512, // Conservative token limit for most embedding models
	}
}

// Chunk implements Embedder interface
func (e *TEIEmbedder) Chunk(text string) []string {
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
func (e *TEIEmbedder) splitIntoSentences(text string) []string {
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
func (e *TEIEmbedder) splitLongSentence(sentence string, maxChars int) []string {
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
func (e *TEIEmbedder) Embed(ctx context.Context, chunks []string) ([]dbtypes.XVector, error) {
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

	reqBody := TEIRequest{
		Inputs: validChunks,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", e.baseURL+"/embed", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TEI API returned status %d", resp.StatusCode)
	}

	var teiResp TEIResponse
	if err := json.NewDecoder(resp.Body).Decode(&teiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	if len(teiResp) != len(validChunks) {
		return nil, fmt.Errorf("expected %d embeddings, got %d", len(validChunks), len(teiResp))
	}

	result := make([]dbtypes.XVector, len(teiResp))
	for i, embedding := range teiResp {
		result[i] = dbtypes.XVector(embedding)
	}

	return result, nil
}

// EmbedSingle implements Embedder interface
func (e *TEIEmbedder) EmbedSingle(ctx context.Context, text string) (dbtypes.XVector, error) {
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
