package microsoft

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	drivenoauth "github.com/custodia-labs/sercha-cli/internal/adapters/driven/oauth"
	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
)

// OAuthHandler implements OAuth operations for Microsoft.
// Handles Microsoft-specific OAuth requirements like offline_access for refresh tokens.
type OAuthHandler struct{}

// NewOAuthHandler creates a new Microsoft OAuth handler.
func NewOAuthHandler() *OAuthHandler {
	return &OAuthHandler{}
}

// BuildAuthURL constructs the Microsoft OAuth authorization URL.
// Includes offline_access scope to ensure refresh tokens are returned.
func (h *OAuthHandler) BuildAuthURL(
	authProvider *domain.AuthProvider,
	redirectURI, state, codeChallenge string,
) string {
	cfg := authProvider.OAuth
	authURL := cfg.AuthURL
	if authURL == "" {
		authURL = defaultAuthURL
	}

	// Use default scopes if none configured
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = defaultScopes
	}

	params := url.Values{
		"client_id":             {cfg.ClientID},
		"redirect_uri":          {redirectURI},
		"response_type":         {"code"},
		"scope":                 {strings.Join(scopes, " ")},
		"state":                 {state},
		"code_challenge":        {codeChallenge},
		"code_challenge_method": {"S256"},
		// Microsoft-specific: response_mode=query for easier code extraction
		"response_mode": {"query"},
	}

	return authURL + "?" + params.Encode()
}

// ExchangeCode exchanges an authorization code for tokens.
func (h *OAuthHandler) ExchangeCode(
	ctx context.Context,
	authProvider *domain.AuthProvider,
	code, redirectURI, codeVerifier string,
) (*domain.OAuthToken, error) {
	cfg := authProvider.OAuth
	tokenURL := cfg.TokenURL
	if tokenURL == "" {
		tokenURL = defaultTokenURL
	}

	resp, err := drivenoauth.ExchangeCodeForTokens(
		ctx, tokenURL, cfg.ClientID, cfg.ClientSecret,
		code, redirectURI, codeVerifier,
	)
	if err != nil {
		return nil, err
	}

	return &domain.OAuthToken{
		AccessToken:  resp.AccessToken,
		RefreshToken: resp.RefreshToken,
		TokenType:    resp.TokenType,
		Expiry:       resp.Expiry,
	}, nil
}

// RefreshToken refreshes an expired access token using a refresh token.
func (h *OAuthHandler) RefreshToken(
	ctx context.Context,
	authProvider *domain.AuthProvider,
	refreshToken string,
) (*domain.OAuthToken, error) {
	cfg := authProvider.OAuth
	tokenURL := cfg.TokenURL
	if tokenURL == "" {
		tokenURL = defaultTokenURL
	}

	resp, err := refreshMicrosoftToken(ctx, tokenURL, cfg.ClientID, cfg.ClientSecret, refreshToken)
	if err != nil {
		return nil, err
	}

	// Microsoft may return a new refresh token
	newRefreshToken := resp.RefreshToken
	if newRefreshToken == "" {
		newRefreshToken = refreshToken
	}

	return &domain.OAuthToken{
		AccessToken:  resp.AccessToken,
		RefreshToken: newRefreshToken,
		TokenType:    resp.TokenType,
		Expiry:       resp.Expiry,
	}, nil
}

// GetUserInfo fetches the user's email from Microsoft Graph.
func (h *OAuthHandler) GetUserInfo(ctx context.Context, accessToken string) (string, error) {
	userInfo, err := GetUserInfo(ctx, accessToken)
	if err != nil {
		return "", err
	}
	return userInfo.GetUserEmail(), nil
}

// DefaultConfig returns default OAuth URLs and scopes for Microsoft.
func (h *OAuthHandler) DefaultConfig() driven.OAuthDefaults {
	return driven.OAuthDefaults{
		AuthURL:  defaultAuthURL,
		TokenURL: defaultTokenURL,
		Scopes:   defaultScopes,
	}
}

// SetupHint returns guidance for setting up a Microsoft OAuth app.
func (h *OAuthHandler) SetupHint() string {
	return "Create OAuth app at portal.azure.com > App registrations"
}

// Microsoft OAuth constants.
const (
	defaultAuthURL = "https://login.microsoftonline.com/common/oauth2/v2.0/authorize"
	//nolint:gosec // G101: Not credentials, OAuth endpoint URL
	defaultTokenURL = "https://login.microsoftonline.com/common/oauth2/v2.0/token"
)

// defaultScopes are the default OAuth scopes for Microsoft.
// Includes all scopes upfront to avoid re-authorization.
var defaultScopes = []string{
	"openid",
	"offline_access", // Required for refresh tokens
	"User.Read",      // User profile
	"Mail.Read",      // Outlook mail (read-only)
	"Calendars.Read", // Calendar events (read-only)
	"Files.Read",     // OneDrive files (read-only)
}

// refreshMicrosoftToken refreshes a Microsoft OAuth token.
func refreshMicrosoftToken(
	ctx context.Context,
	tokenURL, clientID, clientSecret, refreshToken string,
) (*drivenoauth.TokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("client_id", clientID)
	if clientSecret != "" {
		data.Set("client_secret", clientSecret)
	}
	data.Set("refresh_token", refreshToken)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token refresh request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token refresh failed with status %d", resp.StatusCode)
	}

	var tokenResp drivenoauth.TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("decode token response: %w", err)
	}

	// Calculate expiry
	if tokenResp.ExpiresIn > 0 {
		tokenResp.Expiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	}

	return &tokenResp, nil
}
