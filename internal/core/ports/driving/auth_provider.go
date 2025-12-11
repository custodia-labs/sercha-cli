package driving

import (
	"context"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

// OAuthFlowState holds the state for an OAuth flow in progress.
// Used by driving adapters (TUI/CLI) to track the OAuth authorization flow.
type OAuthFlowState struct {
	// AuthURL is the URL to open in the browser for user authorization.
	AuthURL string

	// CodeVerifier is the PKCE code verifier for token exchange.
	CodeVerifier string

	// State is the OAuth state parameter for CSRF protection.
	State string

	// RedirectURI is the local callback URL for the OAuth flow.
	RedirectURI string

	// RedirectPort is the port the callback server is listening on.
	RedirectPort int
}

// AuthProviderService manages authentication provider configurations.
// Auth providers store reusable OAuth app credentials or PAT provider info.
type AuthProviderService interface {
	// Save creates or updates an auth provider.
	Save(ctx context.Context, provider domain.AuthProvider) error

	// Get retrieves an auth provider by ID.
	Get(ctx context.Context, id string) (*domain.AuthProvider, error)

	// List returns all auth providers.
	List(ctx context.Context) ([]domain.AuthProvider, error)

	// ListByProvider returns auth providers for a specific provider type.
	ListByProvider(ctx context.Context, providerType domain.ProviderType) ([]domain.AuthProvider, error)

	// Delete removes an auth provider.
	// Returns an error if the provider is still in use by any source.
	Delete(ctx context.Context, id string) error
}
