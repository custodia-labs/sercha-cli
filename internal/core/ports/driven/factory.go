package driven

import (
	"context"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

// ConnectorBuilder creates a Connector from a Source with auth support.
// TokenProvider may be nil for connectors that don't require authentication.
type ConnectorBuilder func(source domain.Source, tokenProvider TokenProvider) (Connector, error)

// OAuthDefaults provides default OAuth configuration for a connector type.
// Used when creating auth providers to suggest default URLs and scopes.
type OAuthDefaults struct {
	// AuthURL is the default authorization endpoint.
	AuthURL string
	// TokenURL is the default token exchange endpoint.
	TokenURL string
	// Scopes are the default OAuth scopes to request.
	Scopes []string
}

// ConnectorFactory creates connectors from source configuration.
// It maintains a registry of connector types and their builders.
// Also provides OAuth operations for connector types that support OAuth.
type ConnectorFactory interface {
	// Create returns a Connector for the given source.
	// Resolves TokenProvider from source.AuthorizationID internally.
	// Returns ErrUnsupportedType if the source type is unknown.
	Create(ctx context.Context, source domain.Source) (Connector, error)

	// Register adds a connector builder for the given type.
	Register(connectorType string, builder ConnectorBuilder)

	// SupportedTypes returns all registered connector types.
	SupportedTypes() []string

	// === OAuth Methods ===

	// BuildAuthURL constructs the OAuth authorization URL for a connector type.
	// Includes provider-specific parameters (e.g., access_type=offline for Google).
	// Returns error if the connector type doesn't support OAuth.
	BuildAuthURL(connectorType string, authProvider *domain.AuthProvider, redirectURI, state, codeVerifier string) (string, error)

	// ExchangeCode exchanges an authorization code for tokens.
	// Returns error if the connector type doesn't support OAuth.
	ExchangeCode(ctx context.Context, connectorType string, authProvider *domain.AuthProvider, code, redirectURI, codeVerifier string) (*domain.OAuthToken, error)

	// RefreshToken refreshes an expired access token using a refresh token.
	// Returns error if the connector type doesn't support OAuth.
	RefreshToken(ctx context.Context, connectorType string, authProvider *domain.AuthProvider, refreshToken string) (*domain.OAuthToken, error)

	// GetUserInfo fetches the account identifier (email/username) for a connector type.
	// Used to identify which account was authenticated.
	// Returns error if the connector type doesn't support OAuth.
	GetUserInfo(ctx context.Context, connectorType string, accessToken string) (string, error)

	// GetDefaultOAuthConfig returns default OAuth URLs and scopes for a connector type.
	// Returns nil if the connector type doesn't support OAuth.
	GetDefaultOAuthConfig(connectorType string) *OAuthDefaults

	// SupportsOAuth returns true if the connector type supports OAuth authentication.
	SupportsOAuth(connectorType string) bool

	// GetSetupHint returns guidance text for setting up OAuth/PAT with a provider.
	// Returns empty string if no hint is available.
	GetSetupHint(connectorType string) string
}
