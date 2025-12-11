package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

func TestNewProviderRegistry(t *testing.T) {
	registry := NewProviderRegistry()
	require.NotNil(t, registry)
}

func TestProviderRegistry_GetProviders(t *testing.T) {
	registry := NewProviderRegistry()

	providers := registry.GetProviders()

	assert.Len(t, providers, 3) // local, google, github

	// Verify all expected providers are present
	providerSet := make(map[domain.ProviderType]bool)
	for _, p := range providers {
		providerSet[p] = true
	}
	assert.True(t, providerSet[domain.ProviderLocal])
	assert.True(t, providerSet[domain.ProviderGoogle])
	assert.True(t, providerSet[domain.ProviderGitHub])
}

func TestProviderRegistry_GetConnectorsForProvider_Local(t *testing.T) {
	registry := NewProviderRegistry()

	connectors := registry.GetConnectorsForProvider(domain.ProviderLocal)

	require.NotEmpty(t, connectors)
	assert.Contains(t, connectors, "filesystem")
}

func TestProviderRegistry_GetConnectorsForProvider_Google(t *testing.T) {
	registry := NewProviderRegistry()

	connectors := registry.GetConnectorsForProvider(domain.ProviderGoogle)

	// Google has multiple connectors mapped (even if not yet implemented)
	require.NotEmpty(t, connectors)
	assert.Contains(t, connectors, "google-drive")
	assert.Contains(t, connectors, "gmail")
}

func TestProviderRegistry_GetConnectorsForProvider_GitHub(t *testing.T) {
	registry := NewProviderRegistry()

	connectors := registry.GetConnectorsForProvider(domain.ProviderGitHub)

	// GitHub is now a single unified connector
	require.NotEmpty(t, connectors)
	assert.Contains(t, connectors, "github")
	assert.Len(t, connectors, 1, "GitHub should have exactly one unified connector")
}

func TestProviderRegistry_GetConnectorsForProvider_Unknown(t *testing.T) {
	registry := NewProviderRegistry()

	connectors := registry.GetConnectorsForProvider(domain.ProviderType("unknown"))

	assert.Nil(t, connectors)
}

func TestProviderRegistry_GetConnectorsForProvider_ReturnsACopy(t *testing.T) {
	registry := NewProviderRegistry()

	connectors1 := registry.GetConnectorsForProvider(domain.ProviderGoogle)
	connectors2 := registry.GetConnectorsForProvider(domain.ProviderGoogle)

	// Modify one, ensure the other is unaffected
	if len(connectors1) > 0 {
		connectors1[0] = "modified"
		assert.NotEqual(t, connectors1[0], connectors2[0])
	}
}

func TestProviderRegistry_GetProviderForConnector_Filesystem(t *testing.T) {
	registry := NewProviderRegistry()

	provider, err := registry.GetProviderForConnector("filesystem")

	require.NoError(t, err)
	assert.Equal(t, domain.ProviderLocal, provider)
}

func TestProviderRegistry_GetProviderForConnector_GitHub(t *testing.T) {
	registry := NewProviderRegistry()

	provider, err := registry.GetProviderForConnector("github")

	require.NoError(t, err)
	assert.Equal(t, domain.ProviderGitHub, provider)
}

func TestProviderRegistry_GetProviderForConnector_Unknown(t *testing.T) {
	registry := NewProviderRegistry()

	provider, err := registry.GetProviderForConnector("unknown")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown connector type")
	assert.Empty(t, provider)
}

func TestProviderRegistry_IsCompatible_Valid(t *testing.T) {
	registry := NewProviderRegistry()

	tests := []struct {
		provider  domain.ProviderType
		connector string
		expected  bool
	}{
		{domain.ProviderLocal, "filesystem", true},
		{domain.ProviderGoogle, "google-drive", true},
		{domain.ProviderGoogle, "gmail", true},
		{domain.ProviderGitHub, "github", true},
	}

	for _, tt := range tests {
		t.Run(string(tt.provider)+"_"+tt.connector, func(t *testing.T) {
			result := registry.IsCompatible(tt.provider, tt.connector)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestProviderRegistry_IsCompatible_Invalid(t *testing.T) {
	registry := NewProviderRegistry()

	tests := []struct {
		provider  domain.ProviderType
		connector string
	}{
		{domain.ProviderLocal, "github"},
		{domain.ProviderGoogle, "filesystem"},
		{domain.ProviderGitHub, "google-drive"},
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
	registry := NewProviderRegistry()

	tests := []struct {
		provider domain.ProviderType
		expected domain.AuthMethod
	}{
		{domain.ProviderLocal, domain.AuthMethodNone},
		{domain.ProviderGoogle, domain.AuthMethodOAuth},
		{domain.ProviderGitHub, domain.AuthMethodPAT}, // PAT is default for GitHub (simpler)
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
	registry := NewProviderRegistry()

	tests := []struct {
		provider      domain.ProviderType
		supportsPAT   bool
		supportsOAuth bool
		requiresAuth  bool
	}{
		{domain.ProviderLocal, false, false, false},
		{domain.ProviderGoogle, false, true, true},
		{domain.ProviderGitHub, true, true, true}, // GitHub supports both!
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
	registry := NewProviderRegistry()

	tests := []struct {
		provider domain.ProviderType
		expected bool
	}{
		{domain.ProviderLocal, false},
		{domain.ProviderGoogle, false},
		{domain.ProviderGitHub, true}, // GitHub supports both PAT and OAuth
	}

	for _, tt := range tests {
		t.Run(string(tt.provider), func(t *testing.T) {
			result := registry.SupportsMultipleAuthMethods(tt.provider)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestProviderRegistry_GetSupportedAuthMethods(t *testing.T) {
	registry := NewProviderRegistry()

	tests := []struct {
		provider domain.ProviderType
		expected []domain.AuthMethod
	}{
		{domain.ProviderLocal, nil},
		{domain.ProviderGoogle, []domain.AuthMethod{domain.AuthMethodOAuth}},
		{domain.ProviderGitHub, []domain.AuthMethod{domain.AuthMethodPAT, domain.AuthMethodOAuth}},
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
	registry := NewProviderRegistry()

	tests := []struct {
		provider domain.ProviderType
		expected bool
	}{
		{domain.ProviderLocal, false},
		{domain.ProviderGoogle, true}, // Drive, Gmail, Calendar, Docs
		{domain.ProviderGitHub, false},
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
	registry := NewProviderRegistry()

	t.Run("GitHub", func(t *testing.T) {
		endpoints := registry.GetOAuthEndpoints(domain.ProviderGitHub)
		require.NotNil(t, endpoints)
		assert.Equal(t, "https://github.com/login/oauth/authorize", endpoints.AuthURL)
		assert.Equal(t, "https://github.com/login/oauth/access_token", endpoints.TokenURL)
		assert.Equal(t, "https://github.com/login/device/code", endpoints.DeviceURL)
		assert.Contains(t, endpoints.Scopes, "repo")
	})

	t.Run("Google", func(t *testing.T) {
		endpoints := registry.GetOAuthEndpoints(domain.ProviderGoogle)
		require.NotNil(t, endpoints)
		assert.Equal(t, "https://accounts.google.com/o/oauth2/v2/auth", endpoints.AuthURL)
		assert.Equal(t, "https://oauth2.googleapis.com/token", endpoints.TokenURL)
		assert.Empty(t, endpoints.DeviceURL)
		assert.NotEmpty(t, endpoints.Scopes)
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
