package driven

import "context"

// VectorIndex provides semantic similarity search operations.
// Backed by HNSWlib for approximate nearest neighbour search.
type VectorIndex interface {
	// Add inserts a vector for the given chunk ID.
	Add(ctx context.Context, chunkID string, embedding []float32) error

	// Delete removes a vector from the index.
	Delete(ctx context.Context, chunkID string) error

	// Search finds the k nearest neighbours to the query vector.
	Search(ctx context.Context, query []float32, k int) ([]VectorHit, error)

	// Close releases resources.
	Close() error
}

// VectorHit represents a similarity search result.
type VectorHit struct {
	// ChunkID is the matched chunk.
	ChunkID string

	// Similarity is the cosine similarity score (0-1).
	Similarity float64
}
