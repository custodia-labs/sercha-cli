package microsoft

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

func TestNewOAuthHandler(t *testing.T) {
	handler := NewOAuthHandler()
	require.NotNil(t, handler)
}

func TestOAuthHandler_BuildAuthURL(t *testing.T) {
	handler := NewOAuthHandler()

	authProvider := &domain.AuthProvider{
		OAuth: &domain.OAuthProviderConfig{
			ClientID: "test-client-id",
			Scopes:   []string{"openid", "offline_access", "User.Read"},
		},
	}

	url := handler.BuildAuthURL(authProvider, "http://localhost:8080/callback", "test-state", "test-challenge")

	assert.Contains(t, url, "https://login.microsoftonline.com/common/oauth2/v2.0/authorize")
	assert.Contains(t, url, "client_id=test-client-id")
	assert.Contains(t, url, "redirect_uri=http")
	assert.Contains(t, url, "response_type=code")
	assert.Contains(t, url, "state=test-state")
	assert.Contains(t, url, "code_challenge=test-challenge")
	assert.Contains(t, url, "code_challenge_method=S256")
	assert.Contains(t, url, "response_mode=query")
}

func TestOAuthHandler_BuildAuthURL_CustomAuthURL(t *testing.T) {
	handler := NewOAuthHandler()

	customAuthURL := "https://login.microsoftonline.com/tenant-id/oauth2/v2.0/authorize"
	authProvider := &domain.AuthProvider{
		OAuth: &domain.OAuthProviderConfig{
			ClientID: "test-client-id",
			AuthURL:  customAuthURL,
			Scopes:   []string{"openid"},
		},
	}

	url := handler.BuildAuthURL(authProvider, "http://localhost:8080/callback", "state", "challenge")

	assert.True(t, strings.HasPrefix(url, customAuthURL))
}

func TestOAuthHandler_DefaultConfig(t *testing.T) {
	handler := NewOAuthHandler()

	defaults := handler.DefaultConfig()

	assert.Equal(t, defaultAuthURL, defaults.AuthURL)
	assert.Equal(t, defaultTokenURL, defaults.TokenURL)
	assert.NotEmpty(t, defaults.Scopes)

	// Verify required scopes are present
	assert.Contains(t, defaults.Scopes, "openid")
	assert.Contains(t, defaults.Scopes, "offline_access")
	assert.Contains(t, defaults.Scopes, "User.Read")
	assert.Contains(t, defaults.Scopes, "Mail.Read")
	assert.Contains(t, defaults.Scopes, "Calendars.Read")
	assert.Contains(t, defaults.Scopes, "Files.Read")
}

func TestOAuthHandler_SetupHint(t *testing.T) {
	handler := NewOAuthHandler()

	hint := handler.SetupHint()

	assert.NotEmpty(t, hint)
	assert.Contains(t, hint, "portal.azure.com")
}

func TestDefaultScopes(t *testing.T) {
	// Verify all required scopes are defined
	requiredScopes := []string{
		"openid",
		"offline_access",
		"User.Read",
		"Mail.Read",
		"Calendars.Read",
		"Files.Read",
	}

	for _, scope := range requiredScopes {
		assert.Contains(t, defaultScopes, scope, "missing required scope: %s", scope)
	}
}

func TestDefaultURLs(t *testing.T) {
	assert.Equal(t, "https://login.microsoftonline.com/common/oauth2/v2.0/authorize", defaultAuthURL)
	assert.Equal(t, "https://login.microsoftonline.com/common/oauth2/v2.0/token", defaultTokenURL)
}
