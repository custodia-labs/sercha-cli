package driven

import (
	"context"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

// AuthProviderStore persists authentication provider configurations.
// An AuthProvider stores OAuth app credentials or PAT provider info
// that can be reused across multiple sources.
type AuthProviderStore interface {
	// Save stores an auth provider. Creates if new, updates if exists.
	Save(ctx context.Context, provider domain.AuthProvider) error

	// Get retrieves an auth provider by ID.
	Get(ctx context.Context, id string) (*domain.AuthProvider, error)

	// List returns all auth providers.
	List(ctx context.Context) ([]domain.AuthProvider, error)

	// ListByProvider returns all auth providers for a specific provider type.
	ListByProvider(ctx context.Context, providerType domain.ProviderType) ([]domain.AuthProvider, error)

	// Delete removes an auth provider by ID.
	// Returns an error if the provider is still in use by any source.
	Delete(ctx context.Context, id string) error
}
