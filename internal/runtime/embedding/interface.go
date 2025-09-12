package embedding

import (
	"context"

	"github.com/xpanvictor/xarvis/internal/database/dbtypes"
)

type Embedder interface {
	// Chunk splits text into optimal chunks for embedding
	Chunk(text string) []string
	// Embed creates embeddings for multiple text chunks
	Embed(ctx context.Context, chunks []string) ([]dbtypes.XVector, error)
	// EmbedSingle creates embedding for a single text (convenience method)
	EmbedSingle(ctx context.Context, text string) (dbtypes.XVector, error)
}
