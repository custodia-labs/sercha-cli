package services

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
)

// PKCE code verifier length (RFC 7636 recommends 43-128 characters).
const codeVerifierLength = 64

// generateCodeVerifier creates a cryptographically random code verifier for PKCE.
func generateCodeVerifier() (string, error) {
	bytes := make([]byte, codeVerifierLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	// Use base64url encoding without padding
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

// generateCodeChallenge creates a S256 code challenge from the verifier.
func generateCodeChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

// generateState creates a random state parameter for CSRF protection.
func generateState() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}
