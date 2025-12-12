package microsoft

import (
	"context"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// ServiceType identifies a Microsoft Graph API service for rate limiting purposes.
type ServiceType string

const (
	// ServiceOutlook is the Outlook Mail API service.
	ServiceOutlook ServiceType = "outlook"
	// ServiceOneDrive is the OneDrive API service.
	ServiceOneDrive ServiceType = "onedrive"
	// ServiceCalendar is the Microsoft Calendar API service.
	ServiceCalendar ServiceType = "calendar"
)

// RateLimitConfig holds rate limiting configuration for a service.
type RateLimitConfig struct {
	// RequestsPerSecond is the sustained rate limit.
	RequestsPerSecond float64
	// BurstSize is the maximum burst size.
	BurstSize int
}

// DefaultRateLimits provides conservative defaults for each Microsoft service.
// Microsoft Graph allows ~10,000 requests per 10 minutes (~16.67/sec).
// We use conservative limits to avoid hitting quotas.
var DefaultRateLimits = map[ServiceType]RateLimitConfig{
	ServiceOutlook:  {RequestsPerSecond: 10.0, BurstSize: 15}, // Conservative for mail operations
	ServiceOneDrive: {RequestsPerSecond: 10.0, BurstSize: 15}, // Conservative for file operations
	ServiceCalendar: {RequestsPerSecond: 10.0, BurstSize: 15}, // Conservative for calendar operations
}

// RateLimiter provides rate limiting for Microsoft Graph API requests.
// It uses a token bucket algorithm with optional backoff for 429 responses.
type RateLimiter struct {
	mu      sync.Mutex
	limiter *rate.Limiter
	retryAt time.Time
	service ServiceType
}

// NewRateLimiter creates a new rate limiter for the specified service.
func NewRateLimiter(service ServiceType) *RateLimiter {
	cfg, ok := DefaultRateLimits[service]
	if !ok {
		// Default fallback
		cfg = RateLimitConfig{RequestsPerSecond: 10.0, BurstSize: 15}
	}

	return &RateLimiter{
		limiter: rate.NewLimiter(rate.Limit(cfg.RequestsPerSecond), cfg.BurstSize),
		service: service,
	}
}

// NewRateLimiterWithConfig creates a rate limiter with custom configuration.
func NewRateLimiterWithConfig(cfg RateLimitConfig) *RateLimiter {
	return &RateLimiter{
		limiter: rate.NewLimiter(rate.Limit(cfg.RequestsPerSecond), cfg.BurstSize),
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
// Call this when receiving a 429 response from Microsoft Graph APIs.
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
