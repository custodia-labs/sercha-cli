package dropbox

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRateLimiter(t *testing.T) {
	rl := NewRateLimiter()

	require.NotNil(t, rl)
	require.NotNil(t, rl.limiter)
}

func TestRateLimiter_Wait_Success(t *testing.T) {
	rl := NewRateLimiter()
	ctx := context.Background()

	// First few calls should succeed quickly
	for i := 0; i < 3; i++ {
		err := rl.Wait(ctx)
		assert.NoError(t, err)
	}
}

func TestRateLimiter_Wait_ContextCancelled(t *testing.T) {
	rl := NewRateLimiter()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := rl.Wait(ctx)

	assert.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestRateLimiter_Wait_RespectsBackoff(t *testing.T) {
	rl := NewRateLimiter()
	ctx := context.Background()

	// Set a very short backoff for testing
	rl.RecordRateLimitError(1) // 1 second backoff

	start := time.Now()
	err := rl.Wait(ctx)
	elapsed := time.Since(start)

	assert.NoError(t, err)
	// Should have waited at least close to 1 second (with some tolerance)
	assert.True(t, elapsed >= 900*time.Millisecond, "Expected wait of ~1s, got %v", elapsed)
}

func TestRateLimiter_RecordRateLimitError(t *testing.T) {
	rl := NewRateLimiter()

	// Record error with specific retry time
	rl.RecordRateLimitError(60)

	// retryAt should be approximately 60 seconds from now
	expectedRetry := time.Now().Add(60 * time.Second)
	assert.WithinDuration(t, expectedRetry, rl.retryAt, 1*time.Second)
}

func TestRateLimiter_RecordRateLimitError_DefaultBackoff(t *testing.T) {
	rl := NewRateLimiter()

	// Record error with zero/invalid retry time should default to 60s
	rl.RecordRateLimitError(0)

	expectedRetry := time.Now().Add(60 * time.Second)
	assert.WithinDuration(t, expectedRetry, rl.retryAt, 1*time.Second)
}

func TestRateLimiter_RecordRateLimitError_NegativeBackoff(t *testing.T) {
	rl := NewRateLimiter()

	// Negative value should also default to 60s
	rl.RecordRateLimitError(-10)

	expectedRetry := time.Now().Add(60 * time.Second)
	assert.WithinDuration(t, expectedRetry, rl.retryAt, 1*time.Second)
}

func TestRateLimiter_Allow(t *testing.T) {
	rl := NewRateLimiter()

	// Should allow requests up to burst limit
	allowed := 0
	for i := 0; i < BurstSize+5; i++ {
		if rl.Allow() {
			allowed++
		}
	}

	// Should have allowed at least BurstSize requests
	assert.GreaterOrEqual(t, allowed, BurstSize)
}

func TestRateLimiter_Allow_RespectsBackoff(t *testing.T) {
	rl := NewRateLimiter()

	// Set a backoff
	rl.RecordRateLimitError(300) // 5 minutes

	// Should not allow any requests during backoff
	allowed := rl.Allow()
	assert.False(t, allowed)
}

func TestRateLimiter_Allow_AfterBackoff(t *testing.T) {
	rl := NewRateLimiter()

	// Set a very short backoff (already passed)
	rl.mu.Lock()
	rl.retryAt = time.Now().Add(-1 * time.Second)
	rl.mu.Unlock()

	// Should allow requests after backoff expires
	allowed := rl.Allow()
	assert.True(t, allowed)
}

func TestRateLimiterConstants(t *testing.T) {
	// Verify rate limit constants are set appropriately
	assert.Equal(t, 5.0, RequestsPerSecond)
	assert.Equal(t, 10, BurstSize)
}
