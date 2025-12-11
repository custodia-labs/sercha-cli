package domain

import "time"

// OAuthToken represents stored OAuth credentials.
type OAuthToken struct {
	// AccessToken is the bearer token for API access.
	AccessToken string `json:"access_token"`
	// RefreshToken is used to obtain new access tokens.
	RefreshToken string `json:"refresh_token,omitempty"`
	// TokenType is typically "Bearer".
	TokenType string `json:"token_type"`
	// Expiry is when the access token expires.
	Expiry time.Time `json:"expiry,omitempty"`
}

// IsExpired returns true if the token has expired.
func (t *OAuthToken) IsExpired() bool {
	if t.Expiry.IsZero() {
		return false
	}
	return time.Now().After(t.Expiry)
}
