package driving

import (
	"context"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

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

// ConnectorRegistry provides information about available connector types.
type ConnectorRegistry interface {
	// List returns all available connector types.
	List() []domain.ConnectorType

	// Get returns a specific connector type by ID.
	// Returns ErrNotFound if the connector type doesn't exist.
	Get(id string) (*domain.ConnectorType, error)

	// GetConnectorsForProvider returns all connector types for a given provider.
	// Returns empty slice if provider has no connectors.
	GetConnectorsForProvider(provider domain.ProviderType) []domain.ConnectorType

	// ValidateConfig validates configuration for a connector type.
	// Returns ErrNotFound if connector doesn't exist, ErrInvalidInput if validation fails.
	ValidateConfig(connectorID string, config map[string]string) error

	// GetOAuthDefaults returns default OAuth URLs and scopes for a connector type.
	// Returns nil if the connector type doesn't support OAuth.
	GetOAuthDefaults(connectorType string) *OAuthDefaults

	// SupportsOAuth returns true if the connector type supports OAuth authentication.
	SupportsOAuth(connectorType string) bool

	// BuildAuthURL constructs the OAuth authorization URL for a connector type.
	// Includes provider-specific parameters (e.g., access_type=offline for Google).
	BuildAuthURL(connectorType string, authProvider *domain.AuthProvider, redirectURI, state, codeChallenge string) (string, error)

	// GetUserInfo fetches the account identifier (email/username) for a connector type.
	// Used to identify which account was authenticated.
	GetUserInfo(ctx context.Context, connectorType string, accessToken string) (string, error)

	// GetSetupHint returns guidance text for setting up OAuth/PAT with a provider.
	// Returns empty string if no hint is available.
	GetSetupHint(connectorType string) string

	// ExchangeCode exchanges an authorization code for tokens using connector-specific logic.
	// This allows connectors with non-standard OAuth (e.g., Notion requiring JSON body + Basic Auth)
	// to implement their own token exchange while maintaining the factory abstraction.
	ExchangeCode(ctx context.Context, connectorType string, authProvider *domain.AuthProvider, code, redirectURI, codeVerifier string) (*domain.OAuthToken, error)
}
