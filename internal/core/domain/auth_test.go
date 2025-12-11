package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestOAuthToken_IsExpired_ZeroExpiry tests that zero expiry means never expires
func TestOAuthToken_IsExpired_ZeroExpiry(t *testing.T) {
	token := &OAuthToken{
		AccessToken: "test-token",
		Expiry:      time.Time{}, // Zero value
	}

	assert.False(t, token.IsExpired(), "Token with zero expiry should not be expired")
}

// TestOAuthToken_IsExpired_FutureExpiry tests token not yet expired
func TestOAuthToken_IsExpired_FutureExpiry(t *testing.T) {
	token := &OAuthToken{
		AccessToken: "test-token",
		Expiry:      time.Now().Add(time.Hour),
	}

	assert.False(t, token.IsExpired(), "Token with future expiry should not be expired")
}

// TestOAuthToken_IsExpired_PastExpiry tests expired token
func TestOAuthToken_IsExpired_PastExpiry(t *testing.T) {
	token := &OAuthToken{
		AccessToken: "test-token",
		Expiry:      time.Now().Add(-time.Hour),
	}

	assert.True(t, token.IsExpired(), "Token with past expiry should be expired")
}

// TestOAuthToken_IsExpired_JustExpired tests token that just expired
func TestOAuthToken_IsExpired_JustExpired(t *testing.T) {
	token := &OAuthToken{
		AccessToken: "test-token",
		Expiry:      time.Now().Add(-time.Second),
	}

	assert.True(t, token.IsExpired(), "Token that just expired should be expired")
}

// TestOAuthToken_IsExpired_FarFutureExpiry tests token far in the future
func TestOAuthToken_IsExpired_FarFutureExpiry(t *testing.T) {
	token := &OAuthToken{
		AccessToken: "test-token",
		Expiry:      time.Now().Add(365 * 24 * time.Hour), // 1 year
	}

	assert.False(t, token.IsExpired(), "Token with far future expiry should not be expired")
}

// TestOAuthToken_IsExpired_FarPastExpiry tests token long expired
func TestOAuthToken_IsExpired_FarPastExpiry(t *testing.T) {
	token := &OAuthToken{
		AccessToken: "test-token",
		Expiry:      time.Now().Add(-365 * 24 * time.Hour), // 1 year ago
	}

	assert.True(t, token.IsExpired(), "Token with far past expiry should be expired")
}

// TestOAuthToken_Fields tests OAuthToken structure fields
func TestOAuthToken_Fields(t *testing.T) {
	expiry := time.Now().Add(time.Hour)
	token := &OAuthToken{
		AccessToken:  "access-token-123",
		RefreshToken: "refresh-token-456",
		TokenType:    "Bearer",
		Expiry:       expiry,
	}

	assert.Equal(t, "access-token-123", token.AccessToken)
	assert.Equal(t, "refresh-token-456", token.RefreshToken)
	assert.Equal(t, "Bearer", token.TokenType)
	assert.Equal(t, expiry, token.Expiry)
}

// TestOAuthToken_EmptyToken tests token with empty access token
func TestOAuthToken_EmptyToken(t *testing.T) {
	token := &OAuthToken{
		AccessToken: "",
		Expiry:      time.Now().Add(time.Hour),
	}

	// The token should not be expired even though it's empty
	assert.False(t, token.IsExpired())
}

// TestOAuthToken_WithoutRefreshToken tests token without refresh token
func TestOAuthToken_WithoutRefreshToken(t *testing.T) {
	token := &OAuthToken{
		AccessToken:  "access-token",
		RefreshToken: "",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(time.Hour),
	}

	assert.Equal(t, "access-token", token.AccessToken)
	assert.Empty(t, token.RefreshToken)
	assert.False(t, token.IsExpired())
}

// TestOAuthToken_NilToken tests nil token pointer handling
func TestOAuthToken_NilToken(t *testing.T) {
	var token *OAuthToken

	// This test documents that calling IsExpired on nil will panic
	// In production code, callers should check for nil before calling methods
	assert.Panics(t, func() {
		_ = token.IsExpired()
	}, "Calling IsExpired on nil token should panic")
}

// TestOAuthToken_MultipleTokens tests multiple token instances don't interfere
func TestOAuthToken_MultipleTokens(t *testing.T) {
	token1 := &OAuthToken{
		AccessToken: "token1",
		Expiry:      time.Now().Add(time.Hour),
	}

	token2 := &OAuthToken{
		AccessToken: "token2",
		Expiry:      time.Now().Add(-time.Hour),
	}

	assert.False(t, token1.IsExpired())
	assert.True(t, token2.IsExpired())
}

// TestOAuthToken_ExpiryBoundary tests expiry at the boundary
func TestOAuthToken_ExpiryBoundary(t *testing.T) {
	// Create a token that expires in a very short time
	token := &OAuthToken{
		AccessToken: "test-token",
		Expiry:      time.Now().Add(10 * time.Millisecond),
	}

	// Initially not expired
	assert.False(t, token.IsExpired())

	// Wait for expiry
	time.Sleep(20 * time.Millisecond)

	// Now expired
	assert.True(t, token.IsExpired())
}
