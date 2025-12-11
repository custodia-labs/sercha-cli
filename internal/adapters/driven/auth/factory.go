package auth

import (
	"context"
	"fmt"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
)

// Factory creates TokenProviders for sources with credentials.
type Factory struct {
	credentialsStore  driven.CredentialsStore
	authProviderStore driven.AuthProviderStore
}

// NewFactory creates a token provider factory.
func NewFactory(
	credentialsStore driven.CredentialsStore,
	authProviderStore driven.AuthProviderStore,
) *Factory {
	return &Factory{
		credentialsStore:  credentialsStore,
		authProviderStore: authProviderStore,
	}
}

// CreateTokenProvider creates the appropriate TokenProvider for a source.
// Uses the new Credentials system (CredentialsID + AuthProviderID).
// Returns NullTokenProvider for sources without credentials.
func (f *Factory) CreateTokenProvider(ctx context.Context, source *domain.Source) (driven.TokenProvider, error) {
	// Handle no-auth case
	if source.CredentialsID == "" {
		return NewNullTokenProvider(), nil
	}

	// Get credentials from store
	creds, err := f.credentialsStore.Get(ctx, source.CredentialsID)
	if err != nil {
		return nil, fmt.Errorf("get credentials %s: %w", source.CredentialsID, err)
	}

	// Create appropriate provider based on credential type
	if creds.PAT != nil {
		return NewCredentialsPATProvider(source.CredentialsID, f.credentialsStore), nil
	}

	if creds.OAuth != nil {
		if source.AuthProviderID == "" {
			return nil, fmt.Errorf("OAuth credentials require AuthProviderID")
		}
		return NewCredentialsOAuthProvider(
			source.CredentialsID,
			f.credentialsStore,
			source.AuthProviderID,
			f.authProviderStore,
		), nil
	}

	return NewNullTokenProvider(), nil
}

// CreateTokenProviderForCredentials creates a TokenProvider from credentials ID and auth provider ID.
// This is useful when you have the IDs directly rather than a Source object.
func (f *Factory) CreateTokenProviderForCredentials(
	ctx context.Context,
	credentialsID string,
	authProviderID string,
) (driven.TokenProvider, error) {
	if credentialsID == "" {
		return NewNullTokenProvider(), nil
	}

	creds, err := f.credentialsStore.Get(ctx, credentialsID)
	if err != nil {
		return nil, fmt.Errorf("get credentials %s: %w", credentialsID, err)
	}

	if creds.PAT != nil {
		return NewCredentialsPATProvider(credentialsID, f.credentialsStore), nil
	}

	if creds.OAuth != nil {
		if authProviderID == "" {
			return nil, fmt.Errorf("OAuth credentials require AuthProviderID")
		}
		return NewCredentialsOAuthProvider(
			credentialsID,
			f.credentialsStore,
			authProviderID,
			f.authProviderStore,
		), nil
	}

	return NewNullTokenProvider(), nil
}
