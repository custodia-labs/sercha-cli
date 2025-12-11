package google

import (
	"context"

	"golang.org/x/oauth2"

	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
)

// TokenSourceAdapter adapts Sercha's TokenProvider interface to oauth2.TokenSource.
// This allows Google API clients to use our existing token management system.
type TokenSourceAdapter struct {
	provider driven.TokenProvider
	ctx      context.Context
}

// NewTokenSource creates an oauth2.TokenSource from a TokenProvider.
// The returned TokenSource can be used with option.WithTokenSource() when
// creating Google API services.
func NewTokenSource(ctx context.Context, provider driven.TokenProvider) oauth2.TokenSource {
	return &TokenSourceAdapter{
		provider: provider,
		ctx:      ctx,
	}
}

// Token implements oauth2.TokenSource interface.
// Called by Google API clients when they need an access token.
func (t *TokenSourceAdapter) Token() (*oauth2.Token, error) {
	accessToken, err := t.provider.GetToken(t.ctx)
	if err != nil {
		return nil, err
	}

	return &oauth2.Token{
		AccessToken: accessToken,
		TokenType:   "Bearer",
	}, nil
}
