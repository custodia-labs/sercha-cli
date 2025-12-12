package microsoft

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRateLimiter(t *testing.T) {
	tests := []struct {
		name    string
		service ServiceType
	}{
		{name: "outlook", service: ServiceOutlook},
		{name: "onedrive", service: ServiceOneDrive},
		{name: "calendar", service: ServiceCalendar},
		{name: "unknown service", service: ServiceType("unknown")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rl := NewRateLimiter(tt.service)
			require.NotNil(t, rl)
			assert.NotNil(t, rl.limiter)
		})
	}
}

func TestNewRateLimiterWithConfig(t *testing.T) {
	cfg := RateLimitConfig{
		RequestsPerSecond: 5.0,
		BurstSize:         10,
	}

	rl := NewRateLimiterWithConfig(cfg)

	require.NotNil(t, rl)
	assert.NotNil(t, rl.limiter)
}

func TestRateLimiter_Wait(t *testing.T) {
	rl := NewRateLimiter(ServiceOutlook)

	ctx := context.Background()
	err := rl.Wait(ctx)

	assert.NoError(t, err)
}

func TestRateLimiter_Wait_ContextCancelled(t *testing.T) {
	rl := NewRateLimiter(ServiceOutlook)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := rl.Wait(ctx)

	assert.ErrorIs(t, err, context.Canceled)
}

func TestRateLimiter_Allow(t *testing.T) {
	rl := NewRateLimiter(ServiceOutlook)

	// First few requests should be allowed (burst)
	for i := 0; i < 5; i++ {
		assert.True(t, rl.Allow(), "request %d should be allowed", i)
	}
}

func TestRateLimiter_RecordRateLimitError(t *testing.T) {
	rl := NewRateLimiter(ServiceOutlook)

	// Record a rate limit error with 1 second backoff
	rl.RecordRateLimitError(1)

	// Should not allow immediately
	assert.False(t, rl.Allow())

	// Wait for backoff to expire
	time.Sleep(1100 * time.Millisecond)

	// Should allow after backoff
	assert.True(t, rl.Allow())
}

func TestRateLimiter_RecordRateLimitError_DefaultBackoff(t *testing.T) {
	rl := NewRateLimiter(ServiceOutlook)

	// Record with zero (should default to 60s)
	rl.RecordRateLimitError(0)

	rl.mu.Lock()
	retryAt := rl.retryAt
	rl.mu.Unlock()

	// Should be approximately 60 seconds from now
	expectedRetry := time.Now().Add(60 * time.Second)
	assert.WithinDuration(t, expectedRetry, retryAt, 2*time.Second)
}

func TestRateLimiter_RecordRateLimitError_NegativeBackoff(t *testing.T) {
	rl := NewRateLimiter(ServiceOutlook)

	// Record with negative (should default to 60s)
	rl.RecordRateLimitError(-5)

	rl.mu.Lock()
	retryAt := rl.retryAt
	rl.mu.Unlock()

	// Should be approximately 60 seconds from now
	expectedRetry := time.Now().Add(60 * time.Second)
	assert.WithinDuration(t, expectedRetry, retryAt, 2*time.Second)
}

func TestDefaultRateLimits(t *testing.T) {
	// Verify all service types have defaults
	for _, service := range []ServiceType{ServiceOutlook, ServiceOneDrive, ServiceCalendar} {
		cfg, ok := DefaultRateLimits[service]
		assert.True(t, ok, "missing rate limit config for %s", service)
		assert.Greater(t, cfg.RequestsPerSecond, 0.0)
		assert.Greater(t, cfg.BurstSize, 0)
	}
}
