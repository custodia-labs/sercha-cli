package services

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateCodeVerifier(t *testing.T) {
	t.Run("generates valid code verifier", func(t *testing.T) {
		verifier, err := generateCodeVerifier()

		require.NoError(t, err)
		require.NotEmpty(t, verifier)

		// Should be base64url encoded
		_, err = base64.RawURLEncoding.DecodeString(verifier)
		assert.NoError(t, err, "verifier should be valid base64url")

		// Should have proper length (64 bytes * 4/3 for base64 encoding, approximately)
		// Base64url encoding of 64 bytes results in 86 characters (no padding)
		assert.Greater(t, len(verifier), 80, "verifier should be long enough")
		assert.Less(t, len(verifier), 130, "verifier should not be too long")
	})

	t.Run("generates unique verifiers", func(t *testing.T) {
		verifier1, err1 := generateCodeVerifier()
		verifier2, err2 := generateCodeVerifier()

		require.NoError(t, err1)
		require.NoError(t, err2)

		// Each call should produce a different verifier
		assert.NotEqual(t, verifier1, verifier2, "consecutive calls should produce different verifiers")
	})

	t.Run("uses base64url encoding without padding", func(t *testing.T) {
		verifier, err := generateCodeVerifier()

		require.NoError(t, err)

		// Should not contain padding characters
		assert.False(t, strings.Contains(verifier, "="), "should not contain padding")

		// Should use URL-safe characters (no + or /)
		assert.False(t, strings.Contains(verifier, "+"), "should not contain +")
		assert.False(t, strings.Contains(verifier, "/"), "should not contain /")

		// May contain URL-safe alternatives (- and _)
		// This is valid for base64url encoding
	})

	t.Run("generates cryptographically random data", func(t *testing.T) {
		// Generate multiple verifiers and check they're different
		verifiers := make(map[string]bool)
		iterations := 100

		for i := 0; i < iterations; i++ {
			verifier, err := generateCodeVerifier()
			require.NoError(t, err)

			// Should never generate the same verifier twice
			assert.False(t, verifiers[verifier], "should not generate duplicate verifiers")
			verifiers[verifier] = true
		}

		assert.Len(t, verifiers, iterations, "all verifiers should be unique")
	})

	t.Run("decodes to correct byte length", func(t *testing.T) {
		verifier, err := generateCodeVerifier()
		require.NoError(t, err)

		decoded, err := base64.RawURLEncoding.DecodeString(verifier)
		require.NoError(t, err)

		// Should decode to exactly 64 bytes
		assert.Equal(t, codeVerifierLength, len(decoded), "decoded verifier should be exactly 64 bytes")
	})
}

func TestGenerateCodeChallenge(t *testing.T) {
	t.Run("generates valid S256 code challenge", func(t *testing.T) {
		verifier := "test-verifier-12345"
		challenge := generateCodeChallenge(verifier)

		require.NotEmpty(t, challenge)

		// Should be base64url encoded
		_, err := base64.RawURLEncoding.DecodeString(challenge)
		assert.NoError(t, err, "challenge should be valid base64url")
	})

	t.Run("produces consistent challenge for same verifier", func(t *testing.T) {
		verifier := "test-verifier-12345"

		challenge1 := generateCodeChallenge(verifier)
		challenge2 := generateCodeChallenge(verifier)

		assert.Equal(t, challenge1, challenge2, "same verifier should produce same challenge")
	})

	t.Run("produces different challenges for different verifiers", func(t *testing.T) {
		verifier1 := "test-verifier-1"
		verifier2 := "test-verifier-2"

		challenge1 := generateCodeChallenge(verifier1)
		challenge2 := generateCodeChallenge(verifier2)

		assert.NotEqual(t, challenge1, challenge2, "different verifiers should produce different challenges")
	})

	t.Run("uses SHA256 hashing", func(t *testing.T) {
		verifier := "test-verifier"
		challenge := generateCodeChallenge(verifier)

		decoded, err := base64.RawURLEncoding.DecodeString(challenge)
		require.NoError(t, err)

		// SHA256 produces 32 bytes
		assert.Equal(t, 32, len(decoded), "SHA256 hash should be 32 bytes")
	})

	t.Run("uses base64url encoding without padding", func(t *testing.T) {
		verifier := "test-verifier-12345"
		challenge := generateCodeChallenge(verifier)

		// Should not contain padding characters
		assert.False(t, strings.Contains(challenge, "="), "should not contain padding")

		// Should use URL-safe characters (no + or /)
		assert.False(t, strings.Contains(challenge, "+"), "should not contain +")
		assert.False(t, strings.Contains(challenge, "/"), "should not contain /")
	})

	t.Run("handles empty verifier", func(t *testing.T) {
		verifier := ""
		challenge := generateCodeChallenge(verifier)

		// Should still produce a valid challenge (hash of empty string)
		require.NotEmpty(t, challenge)
		_, err := base64.RawURLEncoding.DecodeString(challenge)
		assert.NoError(t, err)
	})

	t.Run("handles long verifier", func(t *testing.T) {
		// Create a very long verifier
		verifier := strings.Repeat("a", 1000)
		challenge := generateCodeChallenge(verifier)

		// Should still produce valid challenge of consistent length
		require.NotEmpty(t, challenge)
		decoded, err := base64.RawURLEncoding.DecodeString(challenge)
		require.NoError(t, err)
		assert.Equal(t, 32, len(decoded), "SHA256 hash should always be 32 bytes")
	})

	t.Run("integration with generateCodeVerifier", func(t *testing.T) {
		// Generate a real verifier and challenge
		verifier, err := generateCodeVerifier()
		require.NoError(t, err)

		challenge := generateCodeChallenge(verifier)
		require.NotEmpty(t, challenge)

		// Challenge should be valid base64url
		_, err = base64.RawURLEncoding.DecodeString(challenge)
		assert.NoError(t, err)

		// Verify the challenge is deterministic
		challenge2 := generateCodeChallenge(verifier)
		assert.Equal(t, challenge, challenge2)
	})
}

func TestGenerateState(t *testing.T) {
	t.Run("generates valid state parameter", func(t *testing.T) {
		state, err := generateState()

		require.NoError(t, err)
		require.NotEmpty(t, state)

		// Should be base64url encoded
		_, err = base64.RawURLEncoding.DecodeString(state)
		assert.NoError(t, err, "state should be valid base64url")
	})

	t.Run("generates unique states", func(t *testing.T) {
		state1, err1 := generateState()
		state2, err2 := generateState()

		require.NoError(t, err1)
		require.NoError(t, err2)

		// Each call should produce a different state
		assert.NotEqual(t, state1, state2, "consecutive calls should produce different states")
	})

	t.Run("uses base64url encoding without padding", func(t *testing.T) {
		state, err := generateState()

		require.NoError(t, err)

		// Should not contain padding characters
		assert.False(t, strings.Contains(state, "="), "should not contain padding")

		// Should use URL-safe characters (no + or /)
		assert.False(t, strings.Contains(state, "+"), "should not contain +")
		assert.False(t, strings.Contains(state, "/"), "should not contain /")
	})

	t.Run("generates cryptographically random data", func(t *testing.T) {
		// Generate multiple states and check they're different
		states := make(map[string]bool)
		iterations := 100

		for i := 0; i < iterations; i++ {
			state, err := generateState()
			require.NoError(t, err)

			// Should never generate the same state twice
			assert.False(t, states[state], "should not generate duplicate states")
			states[state] = true
		}

		assert.Len(t, states, iterations, "all states should be unique")
	})

	t.Run("decodes to 32 bytes", func(t *testing.T) {
		state, err := generateState()
		require.NoError(t, err)

		decoded, err := base64.RawURLEncoding.DecodeString(state)
		require.NoError(t, err)

		// Should decode to exactly 32 bytes
		assert.Equal(t, 32, len(decoded), "decoded state should be exactly 32 bytes")
	})

	t.Run("has sufficient entropy for CSRF protection", func(t *testing.T) {
		state, err := generateState()
		require.NoError(t, err)

		// 32 bytes = 256 bits of entropy, which is more than sufficient for CSRF protection
		decoded, err := base64.RawURLEncoding.DecodeString(state)
		require.NoError(t, err)

		// Verify all bytes are present and not all zeros
		hasNonZero := false
		for _, b := range decoded {
			if b != 0 {
				hasNonZero = true
				break
			}
		}
		assert.True(t, hasNonZero, "state should contain non-zero bytes")
	})

	t.Run("proper length for state parameter", func(t *testing.T) {
		state, err := generateState()
		require.NoError(t, err)

		// Base64url encoding of 32 bytes results in 43 characters (no padding)
		// 32 bytes * 8 bits/byte = 256 bits
		// 256 bits / 6 bits per base64 char = 42.67 chars -> rounds to 43
		assert.Equal(t, 43, len(state), "base64url encoded 32 bytes should be 43 characters")
	})
}

func TestPKCEFlow(t *testing.T) {
	t.Run("complete PKCE flow", func(t *testing.T) {
		// Step 1: Generate code verifier
		verifier, err := generateCodeVerifier()
		require.NoError(t, err)
		require.NotEmpty(t, verifier)

		// Step 2: Generate code challenge from verifier
		challenge := generateCodeChallenge(verifier)
		require.NotEmpty(t, challenge)

		// Step 3: Generate state for CSRF protection
		state, err := generateState()
		require.NoError(t, err)
		require.NotEmpty(t, state)

		// Verify all are unique
		assert.NotEqual(t, verifier, challenge, "verifier and challenge should differ")
		assert.NotEqual(t, verifier, state, "verifier and state should differ")
		assert.NotEqual(t, challenge, state, "challenge and state should differ")

		// Verify verifier and challenge are related (deterministic)
		challenge2 := generateCodeChallenge(verifier)
		assert.Equal(t, challenge, challenge2, "challenge should be reproducible from verifier")
	})
}
