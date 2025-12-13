package dropbox

import (
	"context"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// Rate limit configuration for Dropbox API.
// Dropbox has different limits per endpoint but generally allows ~150-200 calls/minute.
// We use conservative limits to avoid hitting quotas.
const (
	// RequestsPerSecond is the sustained rate limit.
	RequestsPerSecond = 5.0
	// BurstSize is the maximum burst size.
	BurstSize = 10
)

// RateLimiter provides rate limiting for Dropbox API requests.
// It uses a token bucket algorithm with optional backoff for 429 responses.
type RateLimiter struct {
	mu      sync.Mutex
	limiter *rate.Limiter
	retryAt time.Time
}

// NewRateLimiter creates a new rate limiter for Dropbox.
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		limiter: rate.NewLimiter(rate.Limit(RequestsPerSecond), BurstSize),
	}
}

// Wait blocks until a request can be made without exceeding the rate limit.
// It also respects any backoff period set by RecordRateLimitError.
func (r *RateLimiter) Wait(ctx context.Context) error {
	// First, check for backoff from previous rate limit errors
	r.mu.Lock()
	retryAt := r.retryAt
	r.mu.Unlock()

	if time.Now().Before(retryAt) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Until(retryAt)):
		}
	}

	// Then wait for the token bucket
	return r.limiter.Wait(ctx)
}

// RecordRateLimitError records a rate limit error and sets a backoff period.
// Call this when receiving a 429 response from Dropbox APIs.
// The retryAfterSeconds parameter should come from the Retry-After header.
func (r *RateLimiter) RecordRateLimitError(retryAfterSeconds int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if retryAfterSeconds <= 0 {
		// Default backoff: 60 seconds
		retryAfterSeconds = 60
	}

	r.retryAt = time.Now().Add(time.Duration(retryAfterSeconds) * time.Second)
}

// Allow checks if a request can be made immediately without blocking.
// Returns true if the request is allowed, false if it would exceed the rate limit.
func (r *RateLimiter) Allow() bool {
	r.mu.Lock()
	retryAt := r.retryAt
	r.mu.Unlock()

	if time.Now().Before(retryAt) {
		return false
	}

	return r.limiter.Allow()
}
