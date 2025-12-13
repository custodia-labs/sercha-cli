package dropbox

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

func TestNewOAuthHandler(t *testing.T) {
	handler := NewOAuthHandler()

	require.NotNil(t, handler)
}

func TestOAuthHandler_DefaultConfig(t *testing.T) {
	handler := NewOAuthHandler()

	defaults := handler.DefaultConfig()

	assert.Equal(t, defaultAuthURL, defaults.AuthURL)
	assert.Equal(t, defaultTokenURL, defaults.TokenURL)
	assert.Equal(t, defaultScopes, defaults.Scopes)
}

func TestOAuthHandler_SetupHint(t *testing.T) {
	handler := NewOAuthHandler()

	hint := handler.SetupHint()

	assert.Contains(t, hint, "dropbox.com/developers/apps")
}

func TestOAuthHandler_BuildAuthURL(t *testing.T) {
	handler := NewOAuthHandler()

	authProvider := &domain.AuthProvider{
		OAuth: &domain.OAuthProviderConfig{
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
			AuthURL:      defaultAuthURL,
			TokenURL:     defaultTokenURL,
			Scopes:       defaultScopes,
		},
	}

	authURL := handler.BuildAuthURL(
		authProvider,
		"http://localhost:18080/callback",
		"test-state-123",
		"test-code-challenge",
	)

	// Parse the URL
	parsed, err := url.Parse(authURL)
	require.NoError(t, err)

	// Verify base URL
	assert.Equal(t, "www.dropbox.com", parsed.Host)
	assert.Equal(t, "/oauth2/authorize", parsed.Path)

	// Verify query parameters
	query := parsed.Query()
	assert.Equal(t, "test-client-id", query.Get("client_id"))
	assert.Equal(t, "http://localhost:18080/callback", query.Get("redirect_uri"))
	assert.Equal(t, "code", query.Get("response_type"))
	assert.Equal(t, "test-state-123", query.Get("state"))
	assert.Equal(t, "test-code-challenge", query.Get("code_challenge"))
	assert.Equal(t, "S256", query.Get("code_challenge_method"))
	assert.Equal(t, "offline", query.Get("token_access_type"))

	// Verify scopes (space-separated)
	scopes := query.Get("scope")
	assert.Contains(t, scopes, "files.metadata.read")
	assert.Contains(t, scopes, "files.content.read")
	assert.Contains(t, scopes, "account_info.read")
}

func TestOAuthHandler_BuildAuthURL_CustomScopes(t *testing.T) {
	handler := NewOAuthHandler()

	authProvider := &domain.AuthProvider{
		OAuth: &domain.OAuthProviderConfig{
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
			Scopes:       []string{"custom.scope.one", "custom.scope.two"},
		},
	}

	authURL := handler.BuildAuthURL(
		authProvider,
		"http://localhost:18080/callback",
		"state",
		"challenge",
	)

	parsed, err := url.Parse(authURL)
	require.NoError(t, err)

	query := parsed.Query()
	scopes := query.Get("scope")
	assert.Contains(t, scopes, "custom.scope.one")
	assert.Contains(t, scopes, "custom.scope.two")
}

func TestOAuthHandler_BuildAuthURL_DefaultURLs(t *testing.T) {
	handler := NewOAuthHandler()

	authProvider := &domain.AuthProvider{
		OAuth: &domain.OAuthProviderConfig{
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
			// AuthURL and TokenURL left empty to use defaults
		},
	}

	authURL := handler.BuildAuthURL(
		authProvider,
		"http://localhost:18080/callback",
		"state",
		"challenge",
	)

	// Should use default auth URL
	assert.Contains(t, authURL, "https://www.dropbox.com/oauth2/authorize")
}

func TestOAuthHandler_BuildAuthURL_CustomAuthURL(t *testing.T) {
	handler := NewOAuthHandler()

	customAuthURL := "https://custom.dropbox.com/oauth2/auth"
	authProvider := &domain.AuthProvider{
		OAuth: &domain.OAuthProviderConfig{
			ClientID: "test-client-id",
			AuthURL:  customAuthURL,
		},
	}

	authURL := handler.BuildAuthURL(
		authProvider,
		"http://localhost:18080/callback",
		"state",
		"challenge",
	)

	assert.Contains(t, authURL, customAuthURL)
}

func TestDefaultScopes(t *testing.T) {
	// Verify default scopes are correct for Dropbox
	expectedScopes := []string{
		"files.metadata.read",
		"files.content.read",
		"account_info.read",
	}

	assert.Equal(t, expectedScopes, defaultScopes)
}

func TestDefaultURLs(t *testing.T) {
	// Verify default URLs are correct
	assert.Equal(t, "https://www.dropbox.com/oauth2/authorize", defaultAuthURL)
	assert.Equal(t, "https://api.dropboxapi.com/oauth2/token", defaultTokenURL)
}
