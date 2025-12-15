package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
)

// mockConnectorFactoryForProvider is a mock for testing OAuth endpoint delegation.
type mockConnectorFactoryForProvider struct {
	oauthDefaults map[string]*driven.OAuthDefaults
}

func (m *mockConnectorFactoryForProvider) Create(_ context.Context, _ domain.Source) (driven.Connector, error) {
	return nil, nil
}

func (m *mockConnectorFactoryForProvider) Register(_ string, _ driven.ConnectorBuilder) {}

func (m *mockConnectorFactoryForProvider) SupportedTypes() []string {
	return nil
}

func (m *mockConnectorFactoryForProvider) BuildAuthURL(_ string, _ *domain.AuthProvider, _, _, _ string) (string, error) {
	return "", nil
}

func (m *mockConnectorFactoryForProvider) ExchangeCode(_ context.Context, _ string, _ *domain.AuthProvider, _, _, _ string) (*domain.OAuthToken, error) {
	return nil, nil
}

func (m *mockConnectorFactoryForProvider) RefreshToken(_ context.Context, _ string, _ *domain.AuthProvider, _ string) (*domain.OAuthToken, error) {
	return nil, nil
}

func (m *mockConnectorFactoryForProvider) GetUserInfo(_ context.Context, _ string, _ string) (string, error) {
	return "", nil
}

func (m *mockConnectorFactoryForProvider) GetDefaultOAuthConfig(connectorType string) *driven.OAuthDefaults {
	if m.oauthDefaults == nil {
		return nil
	}
	return m.oauthDefaults[connectorType]
}

func (m *mockConnectorFactoryForProvider) SupportsOAuth(_ string) bool {
	return false
}

func (m *mockConnectorFactoryForProvider) GetSetupHint(_ string) string {
	return ""
}

// Helper to create a registry with real connector registry for tests.
func newTestProviderRegistry(factory driven.ConnectorFactory) *ProviderRegistry {
	connectorRegistry := NewConnectorRegistry(factory)
	return NewProviderRegistry(connectorRegistry, factory)
}

func TestNewProviderRegistry(t *testing.T) {
	registry := newTestProviderRegistry(nil)
	require.NotNil(t, registry)
}

func TestProviderRegistry_GetProviders(t *testing.T) {
	registry := newTestProviderRegistry(nil)

	providers := registry.GetProviders()

	// Should have local, google, github, microsoft, dropbox, notion (6 providers)
	assert.Len(t, providers, 6)

	// Verify all expected providers are present
	providerSet := make(map[domain.ProviderType]bool)
	for _, p := range providers {
		providerSet[p] = true
	}
	assert.True(t, providerSet[domain.ProviderLocal])
	assert.True(t, providerSet[domain.ProviderGoogle])
	assert.True(t, providerSet[domain.ProviderGitHub])
	assert.True(t, providerSet[domain.ProviderMicrosoft])
	assert.True(t, providerSet[domain.ProviderDropbox])
	assert.True(t, providerSet[domain.ProviderNotion])
}

func TestProviderRegistry_GetConnectorsForProvider_Local(t *testing.T) {
	registry := newTestProviderRegistry(nil)

	connectors := registry.GetConnectorsForProvider(domain.ProviderLocal)

	require.NotEmpty(t, connectors)
	assert.Contains(t, connectors, "filesystem")
}

func TestProviderRegistry_GetConnectorsForProvider_Google(t *testing.T) {
	registry := newTestProviderRegistry(nil)

	connectors := registry.GetConnectorsForProvider(domain.ProviderGoogle)

	// Google has multiple connectors
	require.NotEmpty(t, connectors)
	assert.Contains(t, connectors, "google-drive")
	assert.Contains(t, connectors, "gmail")
	assert.Contains(t, connectors, "google-calendar")
}

func TestProviderRegistry_GetConnectorsForProvider_GitHub(t *testing.T) {
	registry := newTestProviderRegistry(nil)

	connectors := registry.GetConnectorsForProvider(domain.ProviderGitHub)

	// GitHub is a single unified connector
	require.NotEmpty(t, connectors)
	assert.Contains(t, connectors, "github")
	assert.Len(t, connectors, 1, "GitHub should have exactly one unified connector")
}

func TestProviderRegistry_GetConnectorsForProvider_Microsoft(t *testing.T) {
	registry := newTestProviderRegistry(nil)

	connectors := registry.GetConnectorsForProvider(domain.ProviderMicrosoft)

	// Microsoft has multiple connectors
	require.NotEmpty(t, connectors)
	assert.Contains(t, connectors, "outlook")
	assert.Contains(t, connectors, "onedrive")
	assert.Contains(t, connectors, "microsoft-calendar")
}

func TestProviderRegistry_GetConnectorsForProvider_Unknown(t *testing.T) {
	registry := newTestProviderRegistry(nil)

	connectors := registry.GetConnectorsForProvider(domain.ProviderType("unknown"))

	assert.Empty(t, connectors)
}

func TestProviderRegistry_GetProviderForConnector_Filesystem(t *testing.T) {
	registry := newTestProviderRegistry(nil)

	provider, err := registry.GetProviderForConnector("filesystem")

	require.NoError(t, err)
	assert.Equal(t, domain.ProviderLocal, provider)
}

func TestProviderRegistry_GetProviderForConnector_GitHub(t *testing.T) {
	registry := newTestProviderRegistry(nil)

	provider, err := registry.GetProviderForConnector("github")

	require.NoError(t, err)
	assert.Equal(t, domain.ProviderGitHub, provider)
}

func TestProviderRegistry_GetProviderForConnector_Outlook(t *testing.T) {
	registry := newTestProviderRegistry(nil)

	provider, err := registry.GetProviderForConnector("outlook")

	require.NoError(t, err)
	assert.Equal(t, domain.ProviderMicrosoft, provider)
}

func TestProviderRegistry_GetProviderForConnector_Unknown(t *testing.T) {
	registry := newTestProviderRegistry(nil)

	provider, err := registry.GetProviderForConnector("unknown")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown connector type")
	assert.Empty(t, provider)
}

func TestProviderRegistry_IsCompatible_Valid(t *testing.T) {
	registry := newTestProviderRegistry(nil)

	tests := []struct {
		provider  domain.ProviderType
		connector string
		expected  bool
	}{
		{domain.ProviderLocal, "filesystem", true},
		{domain.ProviderGoogle, "google-drive", true},
		{domain.ProviderGoogle, "gmail", true},
		{domain.ProviderGitHub, "github", true},
		{domain.ProviderMicrosoft, "outlook", true},
		{domain.ProviderMicrosoft, "onedrive", true},
	}

	for _, tt := range tests {
		t.Run(string(tt.provider)+"_"+tt.connector, func(t *testing.T) {
			result := registry.IsCompatible(tt.provider, tt.connector)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestProviderRegistry_IsCompatible_Invalid(t *testing.T) {
	registry := newTestProviderRegistry(nil)

	tests := []struct {
		provider  domain.ProviderType
		connector string
	}{
		{domain.ProviderLocal, "github"},
		{domain.ProviderGoogle, "filesystem"},
		{domain.ProviderGitHub, "google-drive"},
		{domain.ProviderMicrosoft, "gmail"},
		{domain.ProviderType("unknown"), "filesystem"},
	}

	for _, tt := range tests {
		t.Run(string(tt.provider)+"_"+tt.connector, func(t *testing.T) {
			result := registry.IsCompatible(tt.provider, tt.connector)
			assert.False(t, result)
		})
	}
}

func TestProviderRegistry_GetDefaultAuthMethod(t *testing.T) {
	registry := newTestProviderRegistry(nil)

	tests := []struct {
		provider domain.ProviderType
		expected domain.AuthMethod
	}{
		{domain.ProviderLocal, domain.AuthMethodNone},
		{domain.ProviderGoogle, domain.AuthMethodOAuth},
		{domain.ProviderGitHub, domain.AuthMethodPAT}, // PAT is default for GitHub (simpler)
		{domain.ProviderMicrosoft, domain.AuthMethodOAuth},
		{domain.ProviderType("unknown"), domain.AuthMethodNone},
	}

	for _, tt := range tests {
		t.Run(string(tt.provider), func(t *testing.T) {
			method := registry.GetDefaultAuthMethod(tt.provider)
			assert.Equal(t, tt.expected, method)
		})
	}
}

func TestProviderRegistry_GetAuthCapability(t *testing.T) {
	registry := newTestProviderRegistry(nil)

	tests := []struct {
		provider      domain.ProviderType
		supportsPAT   bool
		supportsOAuth bool
		requiresAuth  bool
	}{
		{domain.ProviderLocal, false, false, false},
		{domain.ProviderGoogle, false, true, true},
		{domain.ProviderGitHub, true, true, true}, // GitHub supports both!
		{domain.ProviderMicrosoft, false, true, true},
		{domain.ProviderType("unknown"), false, false, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.provider), func(t *testing.T) {
			authCap := registry.GetAuthCapability(tt.provider)
			assert.Equal(t, tt.supportsPAT, authCap.SupportsPAT(), "SupportsPAT mismatch")
			assert.Equal(t, tt.supportsOAuth, authCap.SupportsOAuth(), "SupportsOAuth mismatch")
			assert.Equal(t, tt.requiresAuth, authCap.RequiresAuth(), "RequiresAuth mismatch")
		})
	}
}

func TestProviderRegistry_SupportsMultipleAuthMethods(t *testing.T) {
	registry := newTestProviderRegistry(nil)

	tests := []struct {
		provider domain.ProviderType
		expected bool
	}{
		{domain.ProviderLocal, false},
		{domain.ProviderGoogle, false},
		{domain.ProviderGitHub, true}, // GitHub supports both PAT and OAuth
		{domain.ProviderMicrosoft, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.provider), func(t *testing.T) {
			result := registry.SupportsMultipleAuthMethods(tt.provider)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestProviderRegistry_GetSupportedAuthMethods(t *testing.T) {
	registry := newTestProviderRegistry(nil)

	tests := []struct {
		provider domain.ProviderType
		expected []domain.AuthMethod
	}{
		{domain.ProviderLocal, nil},
		{domain.ProviderGoogle, []domain.AuthMethod{domain.AuthMethodOAuth}},
		{domain.ProviderGitHub, []domain.AuthMethod{domain.AuthMethodPAT, domain.AuthMethodOAuth}},
		{domain.ProviderMicrosoft, []domain.AuthMethod{domain.AuthMethodOAuth}},
		{domain.ProviderType("unknown"), nil},
	}

	for _, tt := range tests {
		t.Run(string(tt.provider), func(t *testing.T) {
			methods := registry.GetSupportedAuthMethods(tt.provider)
			if tt.expected == nil {
				assert.Empty(t, methods)
			} else {
				assert.Equal(t, tt.expected, methods)
			}
		})
	}
}

func TestProviderRegistry_HasMultipleConnectors(t *testing.T) {
	registry := newTestProviderRegistry(nil)

	tests := []struct {
		provider domain.ProviderType
		expected bool
	}{
		{domain.ProviderLocal, false},
		{domain.ProviderGoogle, true},    // Drive, Gmail, Calendar
		{domain.ProviderGitHub, false},   // Single connector
		{domain.ProviderMicrosoft, true}, // Outlook, OneDrive, Calendar
		{domain.ProviderType("unknown"), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.provider), func(t *testing.T) {
			result := registry.HasMultipleConnectors(tt.provider)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestProviderRegistry_GetOAuthEndpoints(t *testing.T) {
	// Create mock factory with OAuth defaults
	// Include all connectors for each provider since map iteration order is random
	googleDefaults := &driven.OAuthDefaults{
		AuthURL:  "https://accounts.google.com/o/oauth2/v2/auth",
		TokenURL: "https://oauth2.googleapis.com/token",
		Scopes:   []string{"drive.readonly"},
	}
	microsoftDefaults := &driven.OAuthDefaults{
		AuthURL:  "https://login.microsoftonline.com/common/oauth2/v2.0/authorize",
		TokenURL: "https://login.microsoftonline.com/common/oauth2/v2.0/token",
		Scopes:   []string{"Mail.Read"},
	}
	mockFactory := &mockConnectorFactoryForProvider{
		oauthDefaults: map[string]*driven.OAuthDefaults{
			"github": {
				AuthURL:  "https://github.com/login/oauth/authorize",
				TokenURL: "https://github.com/login/oauth/access_token",
				Scopes:   []string{"repo", "read:user"},
			},
			// Google connectors
			"google-drive":    googleDefaults,
			"gmail":           googleDefaults,
			"google-calendar": googleDefaults,
			// Microsoft connectors
			"outlook":            microsoftDefaults,
			"onedrive":           microsoftDefaults,
			"microsoft-calendar": microsoftDefaults,
			// Dropbox connector
			"dropbox": {
				AuthURL:  "https://www.dropbox.com/oauth2/authorize",
				TokenURL: "https://api.dropboxapi.com/oauth2/token",
				Scopes:   []string{"files.metadata.read", "files.content.read"},
			},
		},
	}
	registry := newTestProviderRegistry(mockFactory)

	t.Run("GitHub", func(t *testing.T) {
		endpoints := registry.GetOAuthEndpoints(domain.ProviderGitHub)
		require.NotNil(t, endpoints)
		assert.Equal(t, "https://github.com/login/oauth/authorize", endpoints.AuthURL)
		assert.Equal(t, "https://github.com/login/oauth/access_token", endpoints.TokenURL)
		assert.Contains(t, endpoints.Scopes, "repo")
	})

	t.Run("Google", func(t *testing.T) {
		endpoints := registry.GetOAuthEndpoints(domain.ProviderGoogle)
		require.NotNil(t, endpoints)
		assert.Equal(t, "https://accounts.google.com/o/oauth2/v2/auth", endpoints.AuthURL)
		assert.Equal(t, "https://oauth2.googleapis.com/token", endpoints.TokenURL)
	})

	t.Run("Microsoft", func(t *testing.T) {
		endpoints := registry.GetOAuthEndpoints(domain.ProviderMicrosoft)
		require.NotNil(t, endpoints)
		assert.Equal(t, "https://login.microsoftonline.com/common/oauth2/v2.0/authorize", endpoints.AuthURL)
		assert.Equal(t, "https://login.microsoftonline.com/common/oauth2/v2.0/token", endpoints.TokenURL)
	})

	t.Run("Local returns nil", func(t *testing.T) {
		endpoints := registry.GetOAuthEndpoints(domain.ProviderLocal)
		assert.Nil(t, endpoints)
	})

	t.Run("Unknown returns nil", func(t *testing.T) {
		endpoints := registry.GetOAuthEndpoints(domain.ProviderType("unknown"))
		assert.Nil(t, endpoints)
	})
}

func TestProviderRegistry_NilDependencies(t *testing.T) {
	// Test with nil dependencies
	registry := NewProviderRegistry(nil, nil)

	t.Run("GetProviders returns nil", func(t *testing.T) {
		assert.Nil(t, registry.GetProviders())
	})

	t.Run("GetConnectorsForProvider returns nil", func(t *testing.T) {
		assert.Nil(t, registry.GetConnectorsForProvider(domain.ProviderGoogle))
	})

	t.Run("GetProviderForConnector returns error", func(t *testing.T) {
		_, err := registry.GetProviderForConnector("github")
		assert.Error(t, err)
	})

	t.Run("IsCompatible returns false", func(t *testing.T) {
		assert.False(t, registry.IsCompatible(domain.ProviderGoogle, "gmail"))
	})

	t.Run("GetAuthCapability returns AuthCapNone", func(t *testing.T) {
		assert.Equal(t, domain.AuthCapNone, registry.GetAuthCapability(domain.ProviderGoogle))
	})

	t.Run("HasMultipleConnectors returns false", func(t *testing.T) {
		assert.False(t, registry.HasMultipleConnectors(domain.ProviderGoogle))
	})

	t.Run("GetOAuthEndpoints returns nil", func(t *testing.T) {
		assert.Nil(t, registry.GetOAuthEndpoints(domain.ProviderGoogle))
	})
}
