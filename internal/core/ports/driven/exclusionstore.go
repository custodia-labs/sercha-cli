package driven

import (
	"context"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

// ExclusionStore persists document exclusions.
// Excluded documents are skipped during re-sync operations.
type ExclusionStore interface {
	// Add creates a new exclusion.
	Add(ctx context.Context, exclusion *domain.Exclusion) error

	// Remove deletes an exclusion by ID.
	Remove(ctx context.Context, id string) error

	// GetBySourceID returns all exclusions for a source.
	GetBySourceID(ctx context.Context, sourceID string) ([]domain.Exclusion, error)

	// IsExcluded checks if a URI is excluded for a source.
	IsExcluded(ctx context.Context, sourceID, uri string) (bool, error)

	// List returns all exclusions.
	List(ctx context.Context) ([]domain.Exclusion, error)
}
