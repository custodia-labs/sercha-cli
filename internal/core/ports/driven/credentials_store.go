package driven

import (
	"context"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

// CredentialsStore persists user-specific authentication credentials.
// Credentials are tied to a specific source (1:1 relationship) and store
// OAuth tokens or PAT along with the account identifier.
type CredentialsStore interface {
	// Save stores credentials. Creates if new, updates if exists.
	Save(ctx context.Context, creds domain.Credentials) error

	// Get retrieves credentials by ID.
	Get(ctx context.Context, id string) (*domain.Credentials, error)

	// GetBySourceID retrieves credentials for a specific source.
	// Returns nil if no credentials exist for the source.
	GetBySourceID(ctx context.Context, sourceID string) (*domain.Credentials, error)

	// Delete removes credentials by ID.
	Delete(ctx context.Context, id string) error
}
