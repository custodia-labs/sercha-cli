package notion

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
	"github.com/custodia-labs/sercha-cli/internal/logger"
)

// OAuthHandler implements OAuth operations for Notion.
// Notion uses HTTP Basic Auth for token exchange, unlike most OAuth providers.
type OAuthHandler struct{}

// NewOAuthHandler creates a new Notion OAuth handler.
func NewOAuthHandler() *OAuthHandler {
	return &OAuthHandler{}
}

// BuildAuthURL constructs the Notion OAuth authorization URL.
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
		"client_id":    {cfg.ClientID},
		"redirect_uri": {redirectURI},
		// Notion requires "code" response type
		"response_type": {"code"},
		"state":         {state},
		// Notion uses "owner=user" for user-level integrations
		"owner": {"user"},
	}

	// Note: Notion does not support PKCE (code_challenge) as of API version 2022-06-28

	return authURL + "?" + params.Encode()
}

// ExchangeCode exchanges an authorization code for tokens.
// Notion requires HTTP Basic Auth with base64(CLIENT_ID:CLIENT_SECRET).
func (h *OAuthHandler) ExchangeCode(
	ctx context.Context,
	authProvider *domain.AuthProvider,
	code, redirectURI, _ string, // codeVerifier unused - Notion doesn't support PKCE
) (*domain.OAuthToken, error) {
	cfg := authProvider.OAuth
	tokenURL := cfg.TokenURL
	if tokenURL == "" {
		tokenURL = defaultTokenURL
	}

	resp, err := exchangeNotionCode(ctx, tokenURL, cfg.ClientID, cfg.ClientSecret, code, redirectURI)
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
// Note: Notion access tokens are long-lived but support refresh tokens.
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

	resp, err := refreshNotionToken(ctx, tokenURL, cfg.ClientID, cfg.ClientSecret, refreshToken)
	if err != nil {
		return nil, err
	}

	// Preserve refresh token if not returned
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

// GetUserInfo fetches the user's email from Notion.
func (h *OAuthHandler) GetUserInfo(ctx context.Context, accessToken string) (string, error) {
	userInfo, err := GetUserInfo(ctx, accessToken)
	if err != nil {
		return "", err
	}
	return userInfo.Email, nil
}

// DefaultConfig returns default OAuth URLs for Notion.
func (h *OAuthHandler) DefaultConfig() driven.OAuthDefaults {
	return driven.OAuthDefaults{
		AuthURL:  defaultAuthURL,
		TokenURL: defaultTokenURL,
		Scopes:   nil, // Notion doesn't use traditional OAuth scopes
	}
}

// SetupHint returns guidance for setting up a Notion OAuth app.
func (h *OAuthHandler) SetupHint() string {
	return "Create an integration at www.notion.so/my-integrations"
}

// Notion OAuth constants.
const (
	defaultAuthURL = "https://api.notion.com/v1/oauth/authorize"
	//nolint:gosec // G101: Not credentials, OAuth endpoint URL
	defaultTokenURL = "https://api.notion.com/v1/oauth/token"
)

// notionTokenResponse represents Notion's OAuth token response.
type notionTokenResponse struct {
	AccessToken          string `json:"access_token"`
	RefreshToken         string `json:"refresh_token,omitempty"`
	TokenType            string `json:"token_type"`
	ExpiresIn            int    `json:"expires_in,omitempty"`
	BotID                string `json:"bot_id"`
	WorkspaceID          string `json:"workspace_id"`
	WorkspaceName        string `json:"workspace_name"`
	WorkspaceIcon        string `json:"workspace_icon,omitempty"`
	DuplicatedTemplateID string `json:"duplicated_template_id,omitempty"`
	Owner                struct {
		Type string `json:"type"`
		User struct {
			Object    string `json:"object"`
			ID        string `json:"id"`
			Name      string `json:"name,omitempty"`
			AvatarURL string `json:"avatar_url,omitempty"`
			Type      string `json:"type,omitempty"`
			Person    struct {
				Email string `json:"email"`
			} `json:"person,omitempty"`
		} `json:"user,omitempty"`
	} `json:"owner,omitempty"`
	Expiry time.Time `json:"-"` // Calculated field
}

// UserInfo represents Notion user information.
type UserInfo struct {
	Object    string `json:"object"`
	ID        string `json:"id"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url,omitempty"`
	Type      string `json:"type"`
	Email     string `json:"-"` // Extracted from person.email
}

// GetUserInfo fetches Notion user information using the bot API.
func GetUserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
	// Notion doesn't have a dedicated "get current user" endpoint for OAuth
	// We use the /v1/users/me endpoint to get the bot info
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		"https://api.notion.com/v1/users/me",
		http.NoBody,
	)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Notion-Version", notionAPIVersion)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("user info request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("user info failed with status %d", resp.StatusCode)
	}

	var botInfo struct {
		Object string `json:"object"`
		ID     string `json:"id"`
		Name   string `json:"name"`
		Bot    struct {
			Owner struct {
				Type string `json:"type"`
				User struct {
					ID     string `json:"id"`
					Name   string `json:"name"`
					Person struct {
						Email string `json:"email"`
					} `json:"person"`
				} `json:"user"`
			} `json:"owner"`
		} `json:"bot"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&botInfo); err != nil {
		return nil, fmt.Errorf("decode user info: %w", err)
	}

	// Extract email from bot owner (user)
	email := botInfo.Bot.Owner.User.Person.Email
	if email == "" {
		// Fallback to using the workspace name or bot ID as identifier
		email = botInfo.Name
	}

	return &UserInfo{
		Object: botInfo.Object,
		ID:     botInfo.ID,
		Name:   botInfo.Name,
		Email:  email,
	}, nil
}

// tokenExchangeRequest represents the JSON body for Notion token exchange.
type tokenExchangeRequest struct {
	GrantType   string `json:"grant_type"`
	Code        string `json:"code,omitempty"`
	RedirectURI string `json:"redirect_uri,omitempty"`
}

// exchangeNotionCode exchanges an authorization code for tokens.
// Notion requires HTTP Basic Auth: base64(CLIENT_ID:CLIENT_SECRET).
// Unlike most OAuth providers, Notion uses JSON body instead of form-encoded.
func exchangeNotionCode(
	ctx context.Context,
	tokenURL, clientID, clientSecret, code, redirectURI string,
) (*notionTokenResponse, error) {
	reqBody := tokenExchangeRequest{
		GrantType:   "authorization_code",
		Code:        code,
		RedirectURI: redirectURI,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request body: %w", err)
	}

	logger.Debug("Notion token exchange: POST %s", tokenURL)
	logger.Debug("Notion token exchange body: %s", string(jsonBody))
	logger.Debug("Notion token exchange client_id: %s", clientID)
	logger.Debug("Notion token exchange redirect_uri: %s", redirectURI)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Notion uses HTTP Basic Auth for token exchange
	auth := base64.StdEncoding.EncodeToString([]byte(clientID + ":" + clientSecret))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Notion-Version", notionAPIVersion)

	logger.Debug("Notion token exchange headers: Content-Type=application/json, Notion-Version=%s", notionAPIVersion)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token exchange request: %w", err)
	}
	defer resp.Body.Close()

	logger.Debug("Notion token exchange response status: %d", resp.StatusCode)

	// Read response body for logging
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}
	logger.Debug("Notion token exchange response body: %s", string(body))

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error   string `json:"error"`
			Message string `json:"message"`
		}
		if decErr := json.Unmarshal(body, &errResp); decErr == nil && errResp.Message != "" {
			return nil, fmt.Errorf("token exchange failed: %s (error: %s)", errResp.Message, errResp.Error)
		}
		return nil, fmt.Errorf("token exchange failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp notionTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("decode token response: %w", err)
	}

	// Calculate expiry if provided
	if tokenResp.ExpiresIn > 0 {
		tokenResp.Expiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	}

	logger.Debug("Notion token exchange success: workspace=%s", tokenResp.WorkspaceName)

	return &tokenResp, nil
}

// refreshTokenRequest represents the JSON body for Notion token refresh.
type refreshTokenRequest struct {
	GrantType    string `json:"grant_type"`
	RefreshToken string `json:"refresh_token"`
}

// refreshNotionToken refreshes a Notion OAuth token.
// Like token exchange, Notion uses JSON body instead of form-encoded.
func refreshNotionToken(
	ctx context.Context,
	tokenURL, clientID, clientSecret, refreshToken string,
) (*notionTokenResponse, error) {
	reqBody := refreshTokenRequest{
		GrantType:    "refresh_token",
		RefreshToken: refreshToken,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Notion uses HTTP Basic Auth
	auth := base64.StdEncoding.EncodeToString([]byte(clientID + ":" + clientSecret))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Notion-Version", notionAPIVersion)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token refresh request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token refresh failed with status %d", resp.StatusCode)
	}

	var tokenResp notionTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("decode token response: %w", err)
	}

	// Calculate expiry
	if tokenResp.ExpiresIn > 0 {
		tokenResp.Expiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	}

	return &tokenResp, nil
}

// notionAPIVersion is the Notion API version to use.
const notionAPIVersion = "2022-06-28"
