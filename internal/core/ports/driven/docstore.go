package driven

import (
	"context"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

// DocumentStore persists documents and chunks.
// Backed by SQLite for metadata storage.
type DocumentStore interface {
	// SaveDocument stores or updates a document.
	SaveDocument(ctx context.Context, doc *domain.Document) error

	// SaveChunks stores chunks for a document.
	SaveChunks(ctx context.Context, chunks []domain.Chunk) error

	// GetDocument retrieves a document by ID.
	GetDocument(ctx context.Context, id string) (*domain.Document, error)

	// GetChunks retrieves all chunks for a document.
	GetChunks(ctx context.Context, documentID string) ([]domain.Chunk, error)

	// GetChunk retrieves a specific chunk by ID.
	GetChunk(ctx context.Context, id string) (*domain.Chunk, error)

	// DeleteDocument removes a document and its chunks.
	DeleteDocument(ctx context.Context, id string) error

	// ListDocuments returns documents for a source.
	ListDocuments(ctx context.Context, sourceID string) ([]domain.Document, error)
}
