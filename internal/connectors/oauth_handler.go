// Package connectors provides connector implementations for various data sources.
package connectors

import (
	"context"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
)

// OAuthHandler provides OAuth operations for a provider.
// Implementations are provider-specific (Google, GitHub, etc).
// Each handler encapsulates the provider's OAuth quirks (e.g., Google's access_type=offline).
type OAuthHandler interface {
	// BuildAuthURL constructs the OAuth authorization URL with provider-specific params.
	// Includes PKCE code challenge and any provider-specific parameters.
	BuildAuthURL(authProvider *domain.AuthProvider, redirectURI, state, codeChallenge string) string

	// ExchangeCode exchanges an authorization code for tokens.
	// Uses PKCE code verifier for security.
	ExchangeCode(ctx context.Context, authProvider *domain.AuthProvider, code, redirectURI, codeVerifier string) (*domain.OAuthToken, error)

	// RefreshToken refreshes an expired access token using a refresh token.
	RefreshToken(ctx context.Context, authProvider *domain.AuthProvider, refreshToken string) (*domain.OAuthToken, error)

	// GetUserInfo fetches the account identifier (email/username) from the provider.
	// Used to identify which account was authenticated.
	GetUserInfo(ctx context.Context, accessToken string) (string, error)

	// DefaultConfig returns default OAuth URLs and scopes for this provider.
	// Used when creating auth providers to suggest defaults.
	DefaultConfig() driven.OAuthDefaults

	// SetupHint returns guidance text for setting up OAuth app with this provider.
	// Shown to users during auth provider creation.
	SetupHint() string
}
