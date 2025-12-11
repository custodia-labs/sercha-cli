package driving

import "github.com/custodia-labs/sercha-cli/internal/core/domain"

// ProviderRegistry provides information about providers and their compatible connectors.
type ProviderRegistry interface {
	// GetProviders returns all available provider types.
	GetProviders() []domain.ProviderType

	// GetConnectorsForProvider returns connector types compatible with a provider.
	GetConnectorsForProvider(provider domain.ProviderType) []string

	// GetProviderForConnector returns the provider type for a connector.
	GetProviderForConnector(connectorType string) (domain.ProviderType, error)

	// IsCompatible checks if a connector can use a provider.
	IsCompatible(provider domain.ProviderType, connectorType string) bool

	// GetDefaultAuthMethod returns the typical auth method for a provider.
	GetDefaultAuthMethod(provider domain.ProviderType) domain.AuthMethod

	// GetAuthCapability returns the authentication capabilities for a provider.
	GetAuthCapability(provider domain.ProviderType) domain.AuthCapability

	// GetSupportedAuthMethods returns all auth methods supported by a provider.
	GetSupportedAuthMethods(provider domain.ProviderType) []domain.AuthMethod

	// SupportsMultipleAuthMethods returns true if the provider supports choosing between auth methods.
	SupportsMultipleAuthMethods(provider domain.ProviderType) bool

	// HasMultipleConnectors returns true if the provider supports multiple distinct connectors.
	HasMultipleConnectors(provider domain.ProviderType) bool

	// GetOAuthEndpoints returns the OAuth endpoints for a provider.
	GetOAuthEndpoints(provider domain.ProviderType) *OAuthEndpoints
}

// OAuthEndpoints contains OAuth configuration for a provider.
type OAuthEndpoints struct {
	AuthURL   string
	TokenURL  string
	DeviceURL string
	Scopes    []string
}
