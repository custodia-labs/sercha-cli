package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
)

// mockConnectorFactory is a minimal mock for testing GetSetupHint delegation.
type mockConnectorFactory struct {
	hints map[string]string
}

func (m *mockConnectorFactory) Create(_ context.Context, _ domain.Source) (driven.Connector, error) {
	return nil, nil
}

func (m *mockConnectorFactory) Register(_ string, _ driven.ConnectorBuilder) {}

func (m *mockConnectorFactory) SupportedTypes() []string {
	return nil
}

func (m *mockConnectorFactory) BuildAuthURL(_ string, _ *domain.AuthProvider, _, _, _ string) (string, error) {
	return "", nil
}

func (m *mockConnectorFactory) ExchangeCode(_ context.Context, _ string, _ *domain.AuthProvider, _, _, _ string) (*domain.OAuthToken, error) {
	return nil, nil
}

func (m *mockConnectorFactory) RefreshToken(_ context.Context, _ string, _ *domain.AuthProvider, _ string) (*domain.OAuthToken, error) {
	return nil, nil
}

func (m *mockConnectorFactory) GetUserInfo(_ context.Context, _ string, _ string) (string, error) {
	return "", nil
}

func (m *mockConnectorFactory) GetDefaultOAuthConfig(_ string) *driven.OAuthDefaults {
	return nil
}

func (m *mockConnectorFactory) SupportsOAuth(_ string) bool {
	return false
}

func (m *mockConnectorFactory) GetSetupHint(connectorType string) string {
	if m.hints != nil {
		return m.hints[connectorType]
	}
	return ""
}

func TestNewConnectorRegistry(t *testing.T) {
	registry := NewConnectorRegistry(nil)

	require.NotNil(t, registry)
	assert.NotNil(t, registry.connectors)
}

func TestConnectorRegistry_List(t *testing.T) {
	registry := NewConnectorRegistry(nil)

	connectors := registry.List()

	// All built-in connectors: filesystem, github, google-drive, gmail, google-calendar,
	// outlook, onedrive, microsoft-calendar, dropbox, notion
	assert.Len(t, connectors, 10)

	// Verify all expected connectors are present
	ids := make(map[string]bool)
	for _, c := range connectors {
		ids[c.ID] = true
	}
	assert.True(t, ids["filesystem"])
	assert.True(t, ids["github"])
	assert.True(t, ids["google-drive"])
	assert.True(t, ids["gmail"])
	assert.True(t, ids["google-calendar"])
	assert.True(t, ids["outlook"])
	assert.True(t, ids["onedrive"])
	assert.True(t, ids["microsoft-calendar"])
	assert.True(t, ids["dropbox"])
	assert.True(t, ids["notion"])
}

func TestConnectorRegistry_Get_Filesystem(t *testing.T) {
	registry := NewConnectorRegistry(nil)

	connector, err := registry.Get("filesystem")

	require.NoError(t, err)
	require.NotNil(t, connector)
	assert.Equal(t, "filesystem", connector.ID)
	assert.Equal(t, "Local Filesystem", connector.Name)
	assert.Equal(t, domain.AuthCapNone, connector.AuthCapability)
	assert.Len(t, connector.ConfigKeys, 2) // path and patterns
}

func TestConnectorRegistry_Get_GitHub(t *testing.T) {
	registry := NewConnectorRegistry(nil)

	connector, err := registry.Get("github")

	require.NoError(t, err)
	require.NotNil(t, connector)
	assert.Equal(t, "github", connector.ID)
	assert.Equal(t, "GitHub", connector.Name)
	// GitHub supports both PAT and OAuth
	assert.True(t, connector.AuthCapability.SupportsPAT())
	assert.True(t, connector.AuthCapability.SupportsOAuth())
	assert.True(t, connector.AuthCapability.SupportsMultipleMethods())
	// No required config keys for GitHub - indexes all accessible repos
	assert.Len(t, connector.ConfigKeys, 2) // content_types, file_patterns
}

func TestConnectorRegistry_Get_NotFound(t *testing.T) {
	registry := NewConnectorRegistry(nil)

	connector, err := registry.Get("nonexistent")

	assert.ErrorIs(t, err, domain.ErrNotFound)
	assert.Nil(t, connector)
}

func TestConnectorRegistry_ValidateConfig_Filesystem_Valid(t *testing.T) {
	registry := NewConnectorRegistry(nil)

	err := registry.ValidateConfig("filesystem", map[string]string{
		"path": "/home/user/docs",
	})

	assert.NoError(t, err)
}

func TestConnectorRegistry_ValidateConfig_Filesystem_MissingRequired(t *testing.T) {
	registry := NewConnectorRegistry(nil)

	err := registry.ValidateConfig("filesystem", map[string]string{})

	assert.ErrorIs(t, err, domain.ErrInvalidInput)
}

func TestConnectorRegistry_ValidateConfig_Filesystem_EmptyRequired(t *testing.T) {
	registry := NewConnectorRegistry(nil)

	err := registry.ValidateConfig("filesystem", map[string]string{
		"path": "",
	})

	assert.ErrorIs(t, err, domain.ErrInvalidInput)
}

func TestConnectorRegistry_ValidateConfig_GitHub_NoConfigRequired(t *testing.T) {
	registry := NewConnectorRegistry(nil)

	// GitHub no longer requires owner/repo - it indexes ALL accessible repos
	err := registry.ValidateConfig("github", map[string]string{})

	assert.NoError(t, err)
}

func TestConnectorRegistry_ValidateConfig_GitHub_WithOptionalFilters(t *testing.T) {
	registry := NewConnectorRegistry(nil)

	// Optional filtering configuration
	err := registry.ValidateConfig("github", map[string]string{
		"content_types": "files,issues",
		"file_patterns": "*.go,*.md",
	})

	assert.NoError(t, err)
}

func TestConnectorRegistry_ValidateConfig_NonExistent(t *testing.T) {
	registry := NewConnectorRegistry(nil)

	err := registry.ValidateConfig("nonexistent", map[string]string{})

	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestConnectorRegistry_FilesystemConfigKeys(t *testing.T) {
	registry := NewConnectorRegistry(nil)

	connector, err := registry.Get("filesystem")
	require.NoError(t, err)

	// Check path key
	var pathKey, patternsKey *domain.ConfigKey
	for i := range connector.ConfigKeys {
		switch connector.ConfigKeys[i].Key {
		case "path":
			pathKey = &connector.ConfigKeys[i]
		case "patterns":
			patternsKey = &connector.ConfigKeys[i]
		}
	}

	require.NotNil(t, pathKey)
	assert.True(t, pathKey.Required)
	assert.False(t, pathKey.Secret)
	assert.Equal(t, "Directory Path", pathKey.Label)

	require.NotNil(t, patternsKey)
	assert.False(t, patternsKey.Required)
	assert.False(t, patternsKey.Secret)
}

func TestConnectorRegistry_GitHubConfigKeys(t *testing.T) {
	registry := NewConnectorRegistry(nil)

	connector, err := registry.Get("github")
	require.NoError(t, err)

	keys := make(map[string]domain.ConfigKey)
	for _, key := range connector.ConfigKeys {
		keys[key.Key] = key
	}

	// GitHub only has optional filtering keys - no required owner/repo
	assert.False(t, keys["content_types"].Required)
	assert.False(t, keys["file_patterns"].Required)
}

func TestConnectorRegistry_GitHubAuthCapability(t *testing.T) {
	registry := NewConnectorRegistry(nil)

	connector, err := registry.Get("github")
	require.NoError(t, err)

	// GitHub supports both PAT and OAuth authentication
	assert.True(t, connector.AuthCapability.SupportsPAT(), "GitHub should support PAT")
	assert.True(t, connector.AuthCapability.SupportsOAuth(), "GitHub should support OAuth")
	assert.True(t, connector.AuthCapability.RequiresAuth(), "GitHub should require auth")
	assert.True(t, connector.AuthCapability.SupportsMultipleMethods(), "GitHub should support multiple auth methods")

	// Verify SupportedMethods returns both
	methods := connector.AuthCapability.SupportedMethods()
	assert.Len(t, methods, 2)
	assert.Contains(t, methods, domain.AuthMethodPAT)
	assert.Contains(t, methods, domain.AuthMethodOAuth)
}

func TestConnectorRegistry_FilesystemAuthCapability(t *testing.T) {
	registry := NewConnectorRegistry(nil)

	connector, err := registry.Get("filesystem")
	require.NoError(t, err)

	// Filesystem doesn't require authentication
	assert.False(t, connector.AuthCapability.SupportsPAT())
	assert.False(t, connector.AuthCapability.SupportsOAuth())
	assert.False(t, connector.AuthCapability.RequiresAuth())

	// SupportedMethods should return empty for no auth
	methods := connector.AuthCapability.SupportedMethods()
	assert.Empty(t, methods)
}

func TestConnectorRegistry_GetSetupHint_NilFactory(t *testing.T) {
	registry := NewConnectorRegistry(nil)

	hint := registry.GetSetupHint("github")

	// With nil factory, should return empty string
	assert.Equal(t, "", hint)
}

func TestConnectorRegistry_GetSetupHint_UnknownConnector(t *testing.T) {
	registry := NewConnectorRegistry(nil)

	hint := registry.GetSetupHint("nonexistent")

	assert.Equal(t, "", hint)
}

func TestConnectorRegistry_GetSetupHint_WithFactory(t *testing.T) {
	mockFactory := &mockConnectorFactory{
		hints: map[string]string{
			"github": "Create a GitHub PAT at https://github.com/settings/tokens",
		},
	}
	registry := NewConnectorRegistry(mockFactory)

	hint := registry.GetSetupHint("github")

	assert.Equal(t, "Create a GitHub PAT at https://github.com/settings/tokens", hint)
}

func TestConnectorRegistry_GetSetupHint_WithFactory_NoHint(t *testing.T) {
	mockFactory := &mockConnectorFactory{
		hints: map[string]string{},
	}
	registry := NewConnectorRegistry(mockFactory)

	hint := registry.GetSetupHint("unknown")

	assert.Equal(t, "", hint)
}
