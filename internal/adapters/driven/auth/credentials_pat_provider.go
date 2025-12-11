package auth

import (
	"context"
	"fmt"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
)

// Ensure CredentialsPATProvider implements the TokenProvider interface.
var _ driven.TokenProvider = (*CredentialsPATProvider)(nil)

// CredentialsPATProvider provides static Personal Access Tokens.
// Uses the new Credentials store instead of AuthorizationStore.
// PATs don't expire and don't require refresh.
type CredentialsPATProvider struct {
	credentialsID    string
	credentialsStore driven.CredentialsStore
}

// NewCredentialsPATProvider creates a token provider for PAT-based authentication
// using the new Credentials store.
func NewCredentialsPATProvider(
	credentialsID string,
	credentialsStore driven.CredentialsStore,
) *CredentialsPATProvider {
	return &CredentialsPATProvider{
		credentialsID:    credentialsID,
		credentialsStore: credentialsStore,
	}
}

// GetToken returns the PAT token.
// PATs don't expire, so no refresh logic is needed.
func (p *CredentialsPATProvider) GetToken(ctx context.Context) (string, error) {
	creds, err := p.credentialsStore.Get(ctx, p.credentialsID)
	if err != nil {
		return "", fmt.Errorf("get credentials: %w", err)
	}
	if creds.PAT == nil {
		return "", fmt.Errorf("credentials have no PAT token")
	}
	return creds.PAT.Token, nil
}

// AuthorizationID returns the credentials ID (for compatibility).
func (p *CredentialsPATProvider) AuthorizationID() string {
	return p.credentialsID
}

// AuthMethod returns AuthMethodPAT.
func (p *CredentialsPATProvider) AuthMethod() domain.AuthMethod {
	return domain.AuthMethodPAT
}

// IsAuthenticated returns true if valid PAT credentials exist.
func (p *CredentialsPATProvider) IsAuthenticated() bool {
	creds, err := p.credentialsStore.Get(context.Background(), p.credentialsID)
	if err != nil {
		return false
	}
	return creds.PAT != nil && creds.PAT.Token != ""
}
