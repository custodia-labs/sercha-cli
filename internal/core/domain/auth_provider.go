package domain

import "time"

// AuthProvider represents a reusable authentication provider configuration.
// For OAuth: stores client credentials (can be shared across multiple sources/accounts).
// For PAT: stores provider info (each source has its own PAT in Credentials).
//
// Example: One Google OAuth app can be used by multiple Gmail, Drive, and Calendar sources.
type AuthProvider struct {
	// ID is the unique identifier (UUID).
	ID string `json:"id"`
	// Name is the user-friendly name (e.g., "My Google App", "Work GitHub").
	Name string `json:"name"`
	// ProviderType identifies the provider (google, github, slack, etc.).
	ProviderType ProviderType `json:"provider_type"`
	// AuthMethod indicates how sources using this provider authenticate (oauth, pat, none).
	AuthMethod AuthMethod `json:"auth_method"`

	// OAuth holds OAuth application credentials (for AuthMethodOAuth).
	// Nil for PAT or no-auth providers.
	OAuth *OAuthProviderConfig `json:"oauth,omitempty"`

	// CreatedAt is when the provider was created.
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt is when the provider was last updated.
	UpdatedAt time.Time `json:"updated_at"`
}

// OAuthProviderConfig stores OAuth application credentials.
// These are the client credentials from the OAuth provider's developer console.
type OAuthProviderConfig struct {
	// ClientID is the OAuth client ID from the developer console.
	ClientID string `json:"client_id"`
	// ClientSecret is the OAuth client secret from the developer console.
	ClientSecret string `json:"client_secret"`
	// Scopes are the OAuth scopes to request.
	Scopes []string `json:"scopes"`
	// AuthURL is the authorization endpoint (optional override for custom OAuth servers).
	AuthURL string `json:"auth_url,omitempty"`
	// TokenURL is the token exchange endpoint (optional override for custom OAuth servers).
	TokenURL string `json:"token_url,omitempty"`
	// RedirectURI is the callback URI (default: http://localhost:PORT/callback).
	RedirectURI string `json:"redirect_uri,omitempty"`
}

// IsOAuth returns true if this provider uses OAuth authentication.
func (p *AuthProvider) IsOAuth() bool {
	return p.AuthMethod == AuthMethodOAuth && p.OAuth != nil
}

// IsPAT returns true if this provider uses PAT authentication.
func (p *AuthProvider) IsPAT() bool {
	return p.AuthMethod == AuthMethodPAT
}

// RequiresCredentials returns true if sources using this provider need credentials.
func (p *AuthProvider) RequiresCredentials() bool {
	return p.AuthMethod != AuthMethodNone
}
