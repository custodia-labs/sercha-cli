package driven

import (
	"context"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

// SourceStore persists source configurations.
type SourceStore interface {
	// Save stores or updates a source.
	Save(ctx context.Context, source domain.Source) error

	// Get retrieves a source by ID.
	Get(ctx context.Context, id string) (*domain.Source, error)

	// Delete removes a source.
	Delete(ctx context.Context, id string) error

	// List returns all configured sources.
	List(ctx context.Context) ([]domain.Source, error)
}
