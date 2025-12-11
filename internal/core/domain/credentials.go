package domain

import "time"

// Credentials stores user-specific authentication tokens for a Source.
// Each Source has exactly one Credentials (or none for no-auth sources like filesystem).
//
// This separates user tokens from OAuth app credentials (stored in AuthProvider),
// enabling one OAuth app to serve multiple user accounts.
type Credentials struct {
	// ID is the unique identifier (UUID).
	ID string `json:"id"`
	// SourceID links to the Source this credentials belongs to (1:1 relationship).
	SourceID string `json:"source_id"`

	// AccountIdentifier is the user's email or username from the provider.
	// Fetched from the provider's userinfo endpoint after authentication.
	// Examples: "user@gmail.com", "octocat", "user@company.slack.com"
	AccountIdentifier string `json:"account_identifier,omitempty"`

	// OAuth holds OAuth tokens (for OAuth authentication).
	// Nil for PAT authentication.
	OAuth *OAuthCredentials `json:"oauth,omitempty"`

	// PAT holds the Personal Access Token (for PAT authentication).
	// Nil for OAuth authentication.
	PAT *PATCredentials `json:"pat,omitempty"`

	// CreatedAt is when the credentials were created.
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt is when the credentials were last updated.
	UpdatedAt time.Time `json:"updated_at"`
}

// OAuthCredentials stores OAuth tokens for a specific user account.
type OAuthCredentials struct {
	// AccessToken is the bearer token for API access.
	AccessToken string `json:"access_token"`
	// RefreshToken is used to obtain new access tokens.
	RefreshToken string `json:"refresh_token,omitempty"`
	// TokenType is typically "Bearer".
	TokenType string `json:"token_type"`
	// Expiry is when the access token expires.
	Expiry time.Time `json:"expiry,omitempty"`
}

// PATCredentials stores a Personal Access Token.
type PATCredentials struct {
	// Token is the actual personal access token.
	Token string `json:"token"`
}

// IsExpired returns true if the OAuth access token has expired.
func (c *OAuthCredentials) IsExpired() bool {
	if c.Expiry.IsZero() {
		return false
	}
	return time.Now().After(c.Expiry)
}

// IsAuthenticated returns true if the credentials contain valid tokens.
func (c *Credentials) IsAuthenticated() bool {
	if c.OAuth != nil && c.OAuth.AccessToken != "" {
		return true
	}
	if c.PAT != nil && c.PAT.Token != "" {
		return true
	}
	return false
}

// NeedsRefresh returns true if OAuth tokens need refreshing.
func (c *Credentials) NeedsRefresh() bool {
	if c.OAuth == nil {
		return false
	}
	return c.OAuth.IsExpired() && c.OAuth.RefreshToken != ""
}

// GetAccessToken returns the access token (either OAuth or PAT).
func (c *Credentials) GetAccessToken() string {
	if c.OAuth != nil && c.OAuth.AccessToken != "" {
		return c.OAuth.AccessToken
	}
	if c.PAT != nil && c.PAT.Token != "" {
		return c.PAT.Token
	}
	return ""
}

// HasRefreshToken returns true if a refresh token is available.
func (c *Credentials) HasRefreshToken() bool {
	return c.OAuth != nil && c.OAuth.RefreshToken != ""
}
