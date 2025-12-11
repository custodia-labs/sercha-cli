// Package driven provides interfaces for infrastructure adapters (secondary/outbound ports).
package driven

import "context"

// EmbeddingService generates vector embeddings from text.
// This is an optional service - when nil, vector/semantic search is disabled.
//
// Note: This is separate from VectorIndex which stores and searches vectors.
// EmbeddingService generates vectors; VectorIndex stores them.
//
// Implementations may include:
//   - OpenAI (text-embedding-3-small, text-embedding-3-large)
//   - Ollama (nomic-embed-text, all-minilm)
//   - Local models via inference servers
type EmbeddingService interface {
	// Embed generates a vector embedding for the given text.
	Embed(ctx context.Context, text string) ([]float32, error)

	// EmbedBatch generates embeddings for multiple texts efficiently.
	// This is more efficient than calling Embed in a loop for large batches.
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)

	// Dimensions returns the embedding vector size (e.g., 384, 1536, 3072).
	// This is determined by the model and must match VectorIndex configuration.
	Dimensions() int

	// ModelName returns the name of the embedding model being used.
	ModelName() string

	// Ping validates the service is reachable by making a lightweight test request.
	// This is used at startup to verify connectivity before committing to a search mode.
	Ping(ctx context.Context) error

	// Close releases resources.
	Close() error
}
