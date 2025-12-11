package driven

import (
	"context"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

// SearchEngine provides full-text search operations.
// Backed by Xapian for BM25 keyword search.
type SearchEngine interface {
	// Index adds or updates a chunk in the search index.
	Index(ctx context.Context, chunk domain.Chunk) error

	// Delete removes a chunk from the search index.
	Delete(ctx context.Context, chunkID string) error

	// Search performs a keyword search and returns matching chunk IDs with scores.
	Search(ctx context.Context, query string, limit int) ([]SearchHit, error)

	// Close releases resources.
	Close() error
}

// SearchHit represents a search result from the engine.
type SearchHit struct {
	// ChunkID is the matched chunk.
	ChunkID string

	// Score is the relevance score (e.g., BM25).
	Score float64
}
