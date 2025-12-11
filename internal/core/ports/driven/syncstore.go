package driven

import (
	"context"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

// SyncStateStore persists sync progress.
type SyncStateStore interface {
	// Save stores or updates sync state.
	Save(ctx context.Context, state domain.SyncState) error

	// Get retrieves sync state for a source.
	Get(ctx context.Context, sourceID string) (*domain.SyncState, error)

	// Delete removes sync state for a source.
	Delete(ctx context.Context, sourceID string) error
}
