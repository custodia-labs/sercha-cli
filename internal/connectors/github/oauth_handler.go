package github

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

// OAuthHandler implements OAuth operations for GitHub.
type OAuthHandler struct{}

// NewOAuthHandler creates a new GitHub OAuth handler.
func NewOAuthHandler() *OAuthHandler {
	return &OAuthHandler{}
}

// BuildAuthURL constructs the GitHub OAuth authorization URL.
// GitHub doesn't require access_type=offline like Google does.
func (h *OAuthHandler) BuildAuthURL(
	authProvider *domain.AuthProvider,
	redirectURI, state, codeChallenge string,
) string {
	cfg := authProvider.OAuth
	authURL := cfg.AuthURL
	if authURL == "" {
		authURL = defaultAuthURL
	}

	params := url.Values{
		"client_id":             {cfg.ClientID},
		"redirect_uri":          {redirectURI},
		"scope":                 {strings.Join(cfg.Scopes, " ")},
		"state":                 {state},
		"code_challenge":        {codeChallenge},
		"code_challenge_method": {"S256"},
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
// Note: GitHub OAuth apps don't typically use refresh tokens.
// GitHub Apps use installation tokens which expire, but OAuth apps have long-lived tokens.
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

	resp, err := refreshGitHubToken(ctx, tokenURL, cfg.ClientID, cfg.ClientSecret, refreshToken)
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

// GetUserInfo fetches the user's login from GitHub.
func (h *OAuthHandler) GetUserInfo(ctx context.Context, accessToken string) (string, error) {
	req, err := http.NewRequestWithContext(
		ctx, http.MethodGet, "https://api.github.com/user", http.NoBody)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("user info request failed with status %d", resp.StatusCode)
	}

	var userInfo struct {
		Login string `json:"login"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return "", fmt.Errorf("decode user info: %w", err)
	}

	return userInfo.Login, nil
}

// DefaultConfig returns default OAuth URLs and scopes for GitHub.
func (h *OAuthHandler) DefaultConfig() driven.OAuthDefaults {
	return driven.OAuthDefaults{
		AuthURL:  defaultAuthURL,
		TokenURL: defaultTokenURL,
		Scopes:   defaultScopes,
	}
}

// SetupHint returns guidance for setting up a GitHub OAuth app.
func (h *OAuthHandler) SetupHint() string {
	return "Create OAuth app at github.com/settings/developers"
}

// GitHub OAuth constants.
const (
	defaultAuthURL = "https://github.com/login/oauth/authorize"
	//nolint:gosec // G101: Not credentials, OAuth endpoint URL
	defaultTokenURL = "https://github.com/login/oauth/access_token"
)

// defaultScopes are the default OAuth scopes for GitHub.
var defaultScopes = []string{"repo", "read:user"}

// refreshGitHubToken refreshes a GitHub OAuth token.
func refreshGitHubToken(
	ctx context.Context,
	tokenURL, clientID, clientSecret, refreshToken string,
) (*drivenoauth.TokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
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
