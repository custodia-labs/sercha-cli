package auth

import (
	"context"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
)

// Ensure NullTokenProvider implements the TokenProvider interface.
var _ driven.TokenProvider = (*NullTokenProvider)(nil)

// NullTokenProvider is for connectors that require no authentication.
// Used by filesystem connector and other local data sources.
type NullTokenProvider struct{}

// NewNullTokenProvider creates a token provider for no-auth connectors.
func NewNullTokenProvider() *NullTokenProvider {
	return &NullTokenProvider{}
}

// GetToken returns an empty string since no authentication is needed.
func (p *NullTokenProvider) GetToken(_ context.Context) (string, error) {
	return "", nil
}

// AuthorizationID returns an empty string since there's no authorization.
func (p *NullTokenProvider) AuthorizationID() string {
	return ""
}

// AuthMethod returns AuthMethodNone.
func (p *NullTokenProvider) AuthMethod() domain.AuthMethod {
	return domain.AuthMethodNone
}

// IsAuthenticated always returns true since no-auth is always "authenticated".
func (p *NullTokenProvider) IsAuthenticated() bool {
	return true
}
