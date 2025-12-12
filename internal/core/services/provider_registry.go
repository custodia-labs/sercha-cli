package services

import (
	"fmt"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driving"
)

// ProviderRegistry provides information about providers and their compatible connectors.
// It derives all provider information from the registered connectors, ensuring no
// hardcoded provider knowledge exists in this service.
type ProviderRegistry struct {
	connectorRegistry driving.ConnectorRegistry
	connectorFactory  driven.ConnectorFactory
}

// Ensure ProviderRegistry implements the interface.
var _ driving.ProviderRegistry = (*ProviderRegistry)(nil)

// NewProviderRegistry creates a new ProviderRegistry.
// Both dependencies are required for full functionality but can be nil for limited use.
func NewProviderRegistry(
	connectorRegistry driving.ConnectorRegistry,
	connectorFactory driven.ConnectorFactory,
) *ProviderRegistry {
	return &ProviderRegistry{
		connectorRegistry: connectorRegistry,
		connectorFactory:  connectorFactory,
	}
}

// GetProviders returns all available provider types.
// Derives the list from registered connectors.
func (r *ProviderRegistry) GetProviders() []domain.ProviderType {
	if r.connectorRegistry == nil {
		return nil
	}

	seen := make(map[domain.ProviderType]bool)
	var providers []domain.ProviderType

	for _, connector := range r.connectorRegistry.List() {
		if !seen[connector.ProviderType] {
			seen[connector.ProviderType] = true
			providers = append(providers, connector.ProviderType)
		}
	}

	return providers
}

// GetConnectorsForProvider returns connector types compatible with a provider.
func (r *ProviderRegistry) GetConnectorsForProvider(provider domain.ProviderType) []string {
	if r.connectorRegistry == nil {
		return nil
	}

	connectors := r.connectorRegistry.GetConnectorsForProvider(provider)
	result := make([]string, len(connectors))
	for i, c := range connectors {
		result[i] = c.ID
	}
	return result
}

// GetProviderForConnector returns the provider type for a connector.
func (r *ProviderRegistry) GetProviderForConnector(connectorType string) (domain.ProviderType, error) {
	if r.connectorRegistry == nil {
		return "", fmt.Errorf("connector registry not available")
	}

	connector, err := r.connectorRegistry.Get(connectorType)
	if err != nil {
		return "", fmt.Errorf("unknown connector type: %s", connectorType)
	}

	return connector.ProviderType, nil
}

// IsCompatible checks if a connector can use a provider.
func (r *ProviderRegistry) IsCompatible(provider domain.ProviderType, connectorType string) bool {
	if r.connectorRegistry == nil {
		return false
	}

	connector, err := r.connectorRegistry.Get(connectorType)
	if err != nil {
		return false
	}

	return connector.ProviderType == provider
}

// GetDefaultAuthMethod returns the typical auth method for a provider.
// For providers supporting multiple methods, returns the recommended default.
func (r *ProviderRegistry) GetDefaultAuthMethod(provider domain.ProviderType) domain.AuthMethod {
	authCap := r.GetAuthCapability(provider)
	// PAT is simpler, so prefer it as default when available.
	if authCap.SupportsPAT() {
		return domain.AuthMethodPAT
	}
	if authCap.SupportsOAuth() {
		return domain.AuthMethodOAuth
	}
	return domain.AuthMethodNone
}

// GetAuthCapability returns the authentication capabilities for a provider.
// Derives this from the first connector registered for the provider.
func (r *ProviderRegistry) GetAuthCapability(provider domain.ProviderType) domain.AuthCapability {
	if r.connectorRegistry == nil {
		return domain.AuthCapNone
	}

	connectors := r.connectorRegistry.GetConnectorsForProvider(provider)
	if len(connectors) == 0 {
		return domain.AuthCapNone
	}

	// All connectors for a provider should have the same auth capability
	return connectors[0].AuthCapability
}

// GetSupportedAuthMethods returns all auth methods supported by a provider.
func (r *ProviderRegistry) GetSupportedAuthMethods(provider domain.ProviderType) []domain.AuthMethod {
	return r.GetAuthCapability(provider).SupportedMethods()
}

// SupportsMultipleAuthMethods returns true if the provider supports choosing between auth methods.
func (r *ProviderRegistry) SupportsMultipleAuthMethods(provider domain.ProviderType) bool {
	return r.GetAuthCapability(provider).SupportsMultipleMethods()
}

// HasMultipleConnectors returns true if the provider supports multiple distinct connectors
// that can share an OAuth app. For example, Google has Drive, Gmail, Calendar as separate
// connectors that can share the same OAuth app credentials.
func (r *ProviderRegistry) HasMultipleConnectors(provider domain.ProviderType) bool {
	if r.connectorRegistry == nil {
		return false
	}

	connectors := r.connectorRegistry.GetConnectorsForProvider(provider)
	return len(connectors) > 1
}

// GetOAuthEndpoints returns the OAuth endpoints for a provider.
// Delegates to the connector factory to get default OAuth config from the connector's OAuthHandler.
func (r *ProviderRegistry) GetOAuthEndpoints(provider domain.ProviderType) *driving.OAuthEndpoints {
	if r.connectorRegistry == nil || r.connectorFactory == nil {
		return nil
	}

	// Get any connector for this provider
	connectors := r.connectorRegistry.GetConnectorsForProvider(provider)
	if len(connectors) == 0 {
		return nil
	}

	// Get OAuth defaults from the connector's OAuthHandler via the factory
	defaults := r.connectorFactory.GetDefaultOAuthConfig(connectors[0].ID)
	if defaults == nil {
		return nil
	}

	return &driving.OAuthEndpoints{
		AuthURL:  defaults.AuthURL,
		TokenURL: defaults.TokenURL,
		Scopes:   defaults.Scopes,
	}
}
