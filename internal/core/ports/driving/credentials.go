package driving

import (
	"context"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

// CredentialsService manages user-specific authentication credentials.
// Credentials store OAuth tokens or PAT along with the account identifier.
type CredentialsService interface {
	// Save creates or updates credentials.
	Save(ctx context.Context, creds domain.Credentials) error

	// Get retrieves credentials by ID.
	Get(ctx context.Context, id string) (*domain.Credentials, error)

	// GetBySourceID retrieves credentials for a specific source.
	// Returns nil if no credentials exist for the source.
	GetBySourceID(ctx context.Context, sourceID string) (*domain.Credentials, error)

	// Delete removes credentials by ID.
	Delete(ctx context.Context, id string) error
}
