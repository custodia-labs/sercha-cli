package driven

import (
	"context"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

// TokenProvider provides access tokens for authenticated API calls.
// Implementations handle token refresh transparently.
//
// This interface is designed to work alongside the Scheduler's proactive refresh:
//   - Scheduler: Proactive refresh every 45min (prevents refresh token expiry)
//   - TokenProvider: Reactive refresh if token expired when connector needs it
type TokenProvider interface {
	// GetToken returns a valid access token.
	// If the current token is expired, it will be refreshed automatically.
	// Returns empty string for no-auth connectors.
	GetToken(ctx context.Context) (string, error)

	// AuthorizationID returns the authorization ID being used.
	// Returns empty string for no-auth connectors.
	AuthorizationID() string

	// AuthMethod returns the authentication method (oauth, pat, none).
	AuthMethod() domain.AuthMethod

	// IsAuthenticated returns true if valid authentication is available.
	// Always true for no-auth connectors (NullTokenProvider).
	IsAuthenticated() bool
}
