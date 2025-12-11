package services

import (
	"context"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driving"
)

// Ensure CredentialsService implements the interface.
var _ driving.CredentialsService = (*CredentialsService)(nil)

// CredentialsService manages user-specific authentication credentials.
type CredentialsService struct {
	store driven.CredentialsStore
}

// NewCredentialsService creates a new credentials service.
func NewCredentialsService(store driven.CredentialsStore) *CredentialsService {
	return &CredentialsService{
		store: store,
	}
}

// Save creates or updates credentials.
func (s *CredentialsService) Save(ctx context.Context, creds domain.Credentials) error {
	if s.store == nil {
		return domain.ErrNotImplemented
	}
	if creds.ID == "" {
		return domain.ErrInvalidInput
	}
	return s.store.Save(ctx, creds)
}

// Get retrieves credentials by ID.
func (s *CredentialsService) Get(ctx context.Context, id string) (*domain.Credentials, error) {
	if s.store == nil {
		return nil, domain.ErrNotImplemented
	}
	return s.store.Get(ctx, id)
}

// GetBySourceID retrieves credentials for a specific source.
func (s *CredentialsService) GetBySourceID(ctx context.Context, sourceID string) (*domain.Credentials, error) {
	if s.store == nil {
		return nil, domain.ErrNotImplemented
	}
	return s.store.GetBySourceID(ctx, sourceID)
}

// Delete removes credentials by ID.
func (s *CredentialsService) Delete(ctx context.Context, id string) error {
	if s.store == nil {
		return domain.ErrNotImplemented
	}
	return s.store.Delete(ctx, id)
}
