package auth

import (
	"context"
	"fmt"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
)

// Ensure CredentialsFactory can implement TokenProviderFactory interface (duck typing).
var _ interface {
	CreateTokenProviderForSource(context.Context, domain.Source) (driven.TokenProvider, error)
} = (*CredentialsFactory)(nil)

// CredentialsFactory creates TokenProviders using the new Credentials and AuthProvider stores.
// This replaces the old Factory that used AuthorizationStore.
type CredentialsFactory struct {
	credentialsStore  driven.CredentialsStore
	authProviderStore driven.AuthProviderStore
}

// NewCredentialsFactory creates a token provider factory using the new stores.
func NewCredentialsFactory(
	credentialsStore driven.CredentialsStore,
	authProviderStore driven.AuthProviderStore,
) *CredentialsFactory {
	return &CredentialsFactory{
		credentialsStore:  credentialsStore,
		authProviderStore: authProviderStore,
	}
}

// CreateTokenProvider creates the appropriate TokenProvider for a source's credentials.
// It uses the source's CredentialsID and AuthProviderID to determine the token provider type.
// Returns NullTokenProvider for empty credentials IDs (e.g., filesystem sources).
func (f *CredentialsFactory) CreateTokenProvider(
	ctx context.Context,
	credentialsID string,
	authProviderID string,
) (driven.TokenProvider, error) {
	// Handle no-auth case
	if credentialsID == "" || credentialsID == "local" {
		return NewNullTokenProvider(), nil
	}

	// Get credentials from store
	creds, err := f.credentialsStore.Get(ctx, credentialsID)
	if err != nil {
		return nil, fmt.Errorf("get credentials %s: %w", credentialsID, err)
	}

	// Determine auth type based on which credentials are set
	switch {
	case creds.OAuth != nil:
		if authProviderID == "" {
			return nil, fmt.Errorf("OAuth credentials require an auth provider ID")
		}
		return NewCredentialsOAuthProvider(
			credentialsID,
			f.credentialsStore,
			authProviderID,
			f.authProviderStore,
		), nil

	case creds.PAT != nil:
		return NewCredentialsPATProvider(credentialsID, f.credentialsStore), nil

	default:
		// No credentials set - treat as no-auth
		return NewNullTokenProvider(), nil
	}
}

// CreateTokenProviderForSource creates a TokenProvider for a given source.
// This is a convenience method that extracts the IDs from the source.
func (f *CredentialsFactory) CreateTokenProviderForSource(
	ctx context.Context,
	source domain.Source,
) (driven.TokenProvider, error) {
	return f.CreateTokenProvider(ctx, source.CredentialsID, source.AuthProviderID)
}
