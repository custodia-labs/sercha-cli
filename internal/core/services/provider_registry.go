package services

import (
	"fmt"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driving"
)

// providerConnectors maps provider types to their compatible connector types.
var providerConnectors = map[domain.ProviderType][]string{
	domain.ProviderLocal:  {"filesystem"},
	domain.ProviderGoogle: {"google-drive", "gmail", "google-calendar"},
	domain.ProviderGitHub: {"github"},
}

// providerCapabilities maps provider types to their supported auth methods.
var providerCapabilities = map[domain.ProviderType]domain.AuthCapability{
	domain.ProviderLocal:  domain.AuthCapNone,
	domain.ProviderGitHub: domain.AuthCapPAT | domain.AuthCapOAuth, // GitHub supports both!
	domain.ProviderGoogle: domain.AuthCapOAuth,
}

// connectorProviders is the inverse mapping (connector -> provider).
var connectorProviders map[string]domain.ProviderType

//nolint:gochecknoinits // Package-level static mapping initialization
func init() {
	connectorProviders = make(map[string]domain.ProviderType)
	for provider, connectors := range providerConnectors {
		for _, connector := range connectors {
			connectorProviders[connector] = provider
		}
	}
}

// ProviderRegistry provides information about providers and their compatible connectors.
type ProviderRegistry struct{}

// Ensure ProviderRegistry implements the interface.
var _ driving.ProviderRegistry = (*ProviderRegistry)(nil)

// NewProviderRegistry creates a new ProviderRegistry.
func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{}
}

// GetProviders returns all available provider types.
func (r *ProviderRegistry) GetProviders() []domain.ProviderType {
	providers := make([]domain.ProviderType, 0, len(providerConnectors))
	for provider := range providerConnectors {
		providers = append(providers, provider)
	}
	return providers
}

// GetConnectorsForProvider returns connector types compatible with a provider.
func (r *ProviderRegistry) GetConnectorsForProvider(provider domain.ProviderType) []string {
	if connectors, ok := providerConnectors[provider]; ok {
		// Return a copy to prevent modification
		result := make([]string, len(connectors))
		copy(result, connectors)
		return result
	}
	return nil
}

// GetProviderForConnector returns the provider type for a connector.
func (r *ProviderRegistry) GetProviderForConnector(connectorType string) (domain.ProviderType, error) {
	if provider, ok := connectorProviders[connectorType]; ok {
		return provider, nil
	}
	return "", fmt.Errorf("unknown connector type: %s", connectorType)
}

// IsCompatible checks if a connector can use a provider.
func (r *ProviderRegistry) IsCompatible(provider domain.ProviderType, connectorType string) bool {
	connectors, ok := providerConnectors[provider]
	if !ok {
		return false
	}
	for _, c := range connectors {
		if c == connectorType {
			return true
		}
	}
	return false
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
func (r *ProviderRegistry) GetAuthCapability(provider domain.ProviderType) domain.AuthCapability {
	if authCap, ok := providerCapabilities[provider]; ok {
		return authCap
	}
	return domain.AuthCapNone
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
// Single-connector providers like GitHub, Notion, and Slack return false.
func (r *ProviderRegistry) HasMultipleConnectors(provider domain.ProviderType) bool {
	switch provider {
	case domain.ProviderGoogle:
		// Google has Drive, Gmail, Calendar, Docs - users may want to share OAuth app
		return true
	case domain.ProviderGitHub, domain.ProviderLocal, domain.ProviderSlack, domain.ProviderNotion:
		// Single-connector providers or not yet implemented - go straight to credentials
		return false
	default:
		return false
	}
}

// GetOAuthEndpoints returns the OAuth endpoints for a provider.
// These are the standard endpoints that users should use when creating an OAuth app.
func (r *ProviderRegistry) GetOAuthEndpoints(provider domain.ProviderType) *driving.OAuthEndpoints {
	switch provider { //nolint:exhaustive // Local provider doesn't have OAuth endpoints
	case domain.ProviderGitHub:
		return &driving.OAuthEndpoints{
			AuthURL:   "https://github.com/login/oauth/authorize",
			TokenURL:  "https://github.com/login/oauth/access_token",
			DeviceURL: "https://github.com/login/device/code",
			Scopes:    []string{"repo", "read:user"},
		}
	case domain.ProviderGoogle:
		return &driving.OAuthEndpoints{
			AuthURL:   "https://accounts.google.com/o/oauth2/v2/auth",
			TokenURL:  "https://oauth2.googleapis.com/token",
			DeviceURL: "",
			Scopes: []string{
				// Non-sensitive scopes (for user identification)
				"https://www.googleapis.com/auth/userinfo.email",
				"https://www.googleapis.com/auth/userinfo.profile",
				// Sensitive scopes
				"https://www.googleapis.com/auth/calendar.readonly",
				// Restricted scopes (OK for user-created internal apps)
				"https://www.googleapis.com/auth/drive.readonly",
				"https://www.googleapis.com/auth/gmail.readonly",
			},
		}
	default:
		return nil
	}
}
