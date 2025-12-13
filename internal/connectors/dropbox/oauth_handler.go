package dropbox

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

// OAuthHandler implements OAuth operations for Dropbox.
// Handles Dropbox-specific OAuth requirements like token_access_type=offline for refresh tokens.
type OAuthHandler struct{}

// NewOAuthHandler creates a new Dropbox OAuth handler.
func NewOAuthHandler() *OAuthHandler {
	return &OAuthHandler{}
}

// BuildAuthURL constructs the Dropbox OAuth authorization URL.
// Includes token_access_type=offline to ensure refresh tokens are returned.
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
		"state":                 {state},
		"code_challenge":        {codeChallenge},
		"code_challenge_method": {"S256"},
		// Dropbox-specific: required for refresh tokens
		"token_access_type": {"offline"},
	}

	// Dropbox uses space-separated scopes in the scope parameter
	if len(scopes) > 0 {
		params.Set("scope", strings.Join(scopes, " "))
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

	resp, err := refreshDropboxToken(ctx, tokenURL, cfg.ClientID, cfg.ClientSecret, refreshToken)
	if err != nil {
		return nil, err
	}

	// Dropbox may return a new refresh token
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

// GetUserInfo fetches the user's email from Dropbox.
func (h *OAuthHandler) GetUserInfo(ctx context.Context, accessToken string) (string, error) {
	userInfo, err := GetUserInfo(ctx, accessToken)
	if err != nil {
		return "", err
	}
	return userInfo.Email, nil
}

// DefaultConfig returns default OAuth URLs and scopes for Dropbox.
func (h *OAuthHandler) DefaultConfig() driven.OAuthDefaults {
	return driven.OAuthDefaults{
		AuthURL:  defaultAuthURL,
		TokenURL: defaultTokenURL,
		Scopes:   defaultScopes,
	}
}

// SetupHint returns guidance for setting up a Dropbox OAuth app.
func (h *OAuthHandler) SetupHint() string {
	return "Create OAuth app at www.dropbox.com/developers/apps"
}

// Dropbox OAuth constants.
const (
	defaultAuthURL = "https://www.dropbox.com/oauth2/authorize"
	//nolint:gosec // G101: Not credentials, OAuth endpoint URL
	defaultTokenURL = "https://api.dropboxapi.com/oauth2/token"
)

// defaultScopes are the default OAuth scopes for Dropbox.
var defaultScopes = []string{
	"files.metadata.read",
	"files.content.read",
	"account_info.read",
}

// UserInfo represents Dropbox user information.
type UserInfo struct {
	AccountID string `json:"account_id"`
	Email     string `json:"email"`
	Name      struct {
		DisplayName string `json:"display_name"`
	} `json:"name"`
}

// GetUserInfo fetches Dropbox user information.
func GetUserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"https://api.dropboxapi.com/2/users/get_current_account",
		http.NoBody,
	)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("user info request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("user info failed with status %d", resp.StatusCode)
	}

	var userInfo UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("decode user info: %w", err)
	}

	return &userInfo, nil
}

// refreshDropboxToken refreshes a Dropbox OAuth token.
func refreshDropboxToken(
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
