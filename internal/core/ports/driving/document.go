package driving

import (
	"context"
	"time"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

// DocumentService manages documents within sources.
type DocumentService interface {
	// ListBySource returns all documents for a source.
	ListBySource(ctx context.Context, sourceID string) ([]domain.Document, error)

	// Get retrieves a document by ID.
	Get(ctx context.Context, documentID string) (*domain.Document, error)

	// GetContent returns the concatenated content of all chunks.
	GetContent(ctx context.Context, documentID string) (string, error)

	// GetDetails returns connector-agnostic metadata for display.
	GetDetails(ctx context.Context, documentID string) (*DocumentDetails, error)

	// Exclude removes a document and marks it to skip during re-sync.
	Exclude(ctx context.Context, documentID, reason string) error

	// Refresh re-syncs a single document from its source.
	Refresh(ctx context.Context, documentID string) error

	// Open opens the document in the default application.
	Open(ctx context.Context, documentID string) error
}

// DocumentDetails provides a standardised view of document metadata.
type DocumentDetails struct {
	// ID is the unique document identifier.
	ID string

	// SourceID links to the parent source.
	SourceID string

	// SourceName is the human-readable source name.
	SourceName string

	// SourceType is the connector type (e.g., "filesystem").
	SourceType string

	// Title is the document title.
	Title string

	// URI is the original location.
	URI string

	// ChunkCount is the number of chunks.
	ChunkCount int

	// CreatedAt is when the document was first indexed.
	CreatedAt time.Time

	// UpdatedAt is when the document was last updated.
	UpdatedAt time.Time

	// Metadata contains flattened key-value pairs for display.
	Metadata map[string]string
}
