package services

import (
	"context"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driving"
)

// Ensure AuthProviderService implements the interface.
var _ driving.AuthProviderService = (*AuthProviderService)(nil)

// AuthProviderService manages authentication provider configurations.
type AuthProviderService struct {
	store       driven.AuthProviderStore
	sourceStore driven.SourceStore
}

// NewAuthProviderService creates a new auth provider service.
func NewAuthProviderService(store driven.AuthProviderStore, sourceStore driven.SourceStore) *AuthProviderService {
	return &AuthProviderService{
		store:       store,
		sourceStore: sourceStore,
	}
}

// Save creates or updates an auth provider.
func (s *AuthProviderService) Save(ctx context.Context, provider domain.AuthProvider) error {
	if s.store == nil {
		return domain.ErrNotImplemented
	}
	if provider.ID == "" {
		return domain.ErrInvalidInput
	}
	return s.store.Save(ctx, provider)
}

// Get retrieves an auth provider by ID.
func (s *AuthProviderService) Get(ctx context.Context, id string) (*domain.AuthProvider, error) {
	if s.store == nil {
		return nil, domain.ErrNotImplemented
	}
	return s.store.Get(ctx, id)
}

// List returns all auth providers.
func (s *AuthProviderService) List(ctx context.Context) ([]domain.AuthProvider, error) {
	if s.store == nil {
		return nil, domain.ErrNotImplemented
	}
	return s.store.List(ctx)
}

// ListByProvider returns auth providers for a specific provider type.
func (s *AuthProviderService) ListByProvider(
	ctx context.Context,
	providerType domain.ProviderType,
) ([]domain.AuthProvider, error) {
	if s.store == nil {
		return nil, domain.ErrNotImplemented
	}
	return s.store.ListByProvider(ctx, providerType)
}

// Delete removes an auth provider.
// Returns an error if the provider is still in use by any source.
func (s *AuthProviderService) Delete(ctx context.Context, id string) error {
	if s.store == nil {
		return domain.ErrNotImplemented
	}

	// Check if any sources are using this auth provider
	if s.sourceStore != nil {
		sources, err := s.sourceStore.List(ctx)
		if err != nil {
			return err
		}
		for _, source := range sources {
			if source.AuthProviderID == id {
				return domain.ErrAuthProviderInUse
			}
		}
	}

	return s.store.Delete(ctx, id)
}
