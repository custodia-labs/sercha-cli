package google

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

// OAuthHandler implements OAuth operations for Google.
// Handles Google-specific OAuth requirements like access_type=offline for refresh tokens.
type OAuthHandler struct{}

// NewOAuthHandler creates a new Google OAuth handler.
func NewOAuthHandler() *OAuthHandler {
	return &OAuthHandler{}
}

// BuildAuthURL constructs the Google OAuth authorization URL.
// Includes access_type=offline and prompt=consent to ensure refresh tokens are returned.
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
		"response_type":         {"code"},
		"scope":                 {strings.Join(cfg.Scopes, " ")},
		"state":                 {state},
		"code_challenge":        {codeChallenge},
		"code_challenge_method": {"S256"},
		// Google-specific: required for refresh tokens
		"access_type": {"offline"},
		"prompt":      {"consent"},
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

	resp, err := refreshGoogleToken(ctx, tokenURL, cfg.ClientID, cfg.ClientSecret, refreshToken)
	if err != nil {
		return nil, err
	}

	return &domain.OAuthToken{
		AccessToken:  resp.AccessToken,
		RefreshToken: refreshToken, // Google doesn't always return a new refresh token
		TokenType:    resp.TokenType,
		Expiry:       resp.Expiry,
	}, nil
}

// GetUserInfo fetches the user's email from Google.
func (h *OAuthHandler) GetUserInfo(ctx context.Context, accessToken string) (string, error) {
	userInfo, err := GetUserInfo(ctx, accessToken)
	if err != nil {
		return "", err
	}
	return userInfo.Email, nil
}

// DefaultConfig returns default OAuth URLs and scopes for Google.
func (h *OAuthHandler) DefaultConfig() driven.OAuthDefaults {
	return driven.OAuthDefaults{
		AuthURL:  defaultAuthURL,
		TokenURL: defaultTokenURL,
		Scopes:   defaultScopes,
	}
}

// SetupHint returns guidance for setting up a Google OAuth app.
func (h *OAuthHandler) SetupHint() string {
	return "Create OAuth app at console.cloud.google.com/apis/credentials"
}

// Google OAuth constants.
const (
	defaultAuthURL  = "https://accounts.google.com/o/oauth2/v2/auth"
	defaultTokenURL = "https://oauth2.googleapis.com/token"
)

// defaultScopes are the default OAuth scopes for Google.
// Includes all scopes upfront to avoid re-authorization.
var defaultScopes = []string{
	"openid",
	"https://www.googleapis.com/auth/userinfo.email",
	"https://www.googleapis.com/auth/userinfo.profile",
	"https://www.googleapis.com/auth/drive.readonly",
	"https://www.googleapis.com/auth/gmail.readonly",
	"https://www.googleapis.com/auth/calendar.readonly",
}

// refreshGoogleToken refreshes a Google OAuth token.
func refreshGoogleToken(
	ctx context.Context,
	tokenURL, clientID, clientSecret, refreshToken string,
) (*drivenoauth.TokenResponse, error) {
	return refreshOAuthToken(ctx, tokenURL, clientID, clientSecret, refreshToken)
}

// refreshOAuthToken performs a standard OAuth2 token refresh.
func refreshOAuthToken(
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
