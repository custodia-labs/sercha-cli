package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
)

// Ensure CredentialsOAuthProvider implements the TokenProvider interface.
var _ driven.TokenProvider = (*CredentialsOAuthProvider)(nil)

// CredentialsOAuthProvider provides OAuth access tokens with automatic refresh.
// Uses the new Credentials and AuthProvider stores instead of AuthorizationStore.
type CredentialsOAuthProvider struct {
	credentialsID     string
	credentialsStore  driven.CredentialsStore
	authProviderID    string
	authProviderStore driven.AuthProviderStore

	mu            sync.RWMutex
	cachedToken   string
	cacheExpiry   time.Time
	refreshBuffer time.Duration
}

// NewCredentialsOAuthProvider creates a token provider for OAuth-based authentication
// using the new Credentials and AuthProvider stores.
func NewCredentialsOAuthProvider(
	credentialsID string,
	credentialsStore driven.CredentialsStore,
	authProviderID string,
	authProviderStore driven.AuthProviderStore,
) *CredentialsOAuthProvider {
	return &CredentialsOAuthProvider{
		credentialsID:     credentialsID,
		credentialsStore:  credentialsStore,
		authProviderID:    authProviderID,
		authProviderStore: authProviderStore,
		refreshBuffer:     5 * time.Minute,
	}
}

// GetToken returns a valid access token, refreshing if necessary.
//
//nolint:gocognit,nestif,gocyclo // Token refresh with necessary concurrency checks
func (p *CredentialsOAuthProvider) GetToken(ctx context.Context) (string, error) {
	// Fast path: check cache with read lock
	p.mu.RLock()
	if p.cachedToken != "" && time.Now().Before(p.cacheExpiry) {
		token := p.cachedToken
		p.mu.RUnlock()
		return token, nil
	}
	p.mu.RUnlock()

	// Slow path: need refresh, acquire write lock
	p.mu.Lock()
	defer p.mu.Unlock()

	// Double-check after acquiring write lock
	if p.cachedToken != "" && time.Now().Before(p.cacheExpiry) {
		return p.cachedToken, nil
	}

	// Get current credentials
	creds, err := p.credentialsStore.Get(ctx, p.credentialsID)
	if err != nil {
		return "", fmt.Errorf("get credentials: %w", err)
	}
	if creds.OAuth == nil {
		return "", fmt.Errorf("credentials have no OAuth tokens")
	}

	// Check if we need to refresh
	needsRefresh := creds.OAuth.IsExpired()
	if !creds.OAuth.Expiry.IsZero() {
		needsRefresh = needsRefresh || time.Until(creds.OAuth.Expiry) < p.refreshBuffer
	}

	if needsRefresh && creds.OAuth.RefreshToken != "" {
		// Get auth provider for token URL
		provider, err := p.authProviderStore.Get(ctx, p.authProviderID)
		if err != nil {
			return "", fmt.Errorf("get auth provider: %w", err)
		}
		if provider.OAuth == nil {
			return "", fmt.Errorf("auth provider has no OAuth config")
		}

		// Refresh the token
		newTokens, err := p.refreshToken(ctx, creds.OAuth.RefreshToken, provider.OAuth)
		if err != nil {
			return "", fmt.Errorf("refresh token: %w", err)
		}

		// Update credentials with new tokens
		creds.OAuth.AccessToken = newTokens.AccessToken
		if newTokens.RefreshToken != "" {
			creds.OAuth.RefreshToken = newTokens.RefreshToken
		}
		creds.OAuth.Expiry = newTokens.Expiry
		creds.OAuth.TokenType = newTokens.TokenType
		creds.UpdatedAt = time.Now()

		if err := p.credentialsStore.Save(ctx, *creds); err != nil {
			return "", fmt.Errorf("save refreshed credentials: %w", err)
		}
	}

	// Cache the token
	p.cachedToken = creds.OAuth.AccessToken

	// Set cache expiry
	if !creds.OAuth.Expiry.IsZero() {
		p.cacheExpiry = creds.OAuth.Expiry.Add(-p.refreshBuffer)
	} else {
		p.cacheExpiry = time.Now().Add(1 * time.Hour)
	}

	return p.cachedToken, nil
}

// refreshToken performs the OAuth2 token refresh.
func (p *CredentialsOAuthProvider) refreshToken(
	ctx context.Context,
	refreshToken string,
	oauthConfig *domain.OAuthProviderConfig,
) (*domain.OAuthCredentials, error) {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)
	data.Set("client_id", oauthConfig.ClientID)
	data.Set("client_secret", oauthConfig.ClientSecret)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, oauthConfig.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token refresh failed with status %d", resp.StatusCode)
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
		RefreshToken string `json:"refresh_token"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("decode token response: %w", err)
	}

	return &domain.OAuthCredentials{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		TokenType:    tokenResp.TokenType,
		Expiry:       time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
	}, nil
}

// AuthorizationID returns the credentials ID (for compatibility).
func (p *CredentialsOAuthProvider) AuthorizationID() string {
	return p.credentialsID
}

// AuthMethod returns AuthMethodOAuth.
func (p *CredentialsOAuthProvider) AuthMethod() domain.AuthMethod {
	return domain.AuthMethodOAuth
}

// IsAuthenticated returns true if the credentials have valid tokens.
func (p *CredentialsOAuthProvider) IsAuthenticated() bool {
	p.mu.RLock()
	if p.cachedToken != "" && time.Now().Before(p.cacheExpiry) {
		p.mu.RUnlock()
		return true
	}
	p.mu.RUnlock()

	creds, err := p.credentialsStore.Get(context.Background(), p.credentialsID)
	if err != nil {
		return false
	}
	return creds.OAuth != nil && creds.OAuth.AccessToken != ""
}

// InvalidateCache clears the cached token.
func (p *CredentialsOAuthProvider) InvalidateCache() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.cachedToken = ""
	p.cacheExpiry = time.Time{}
}
