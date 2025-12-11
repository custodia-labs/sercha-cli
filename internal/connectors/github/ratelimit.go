package github

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

const (
	// GitHubRateLimit is the authenticated rate limit (5000/hour).
	GitHubRateLimit = 5000

	// ProactiveRate is the proactive throttle rate (~1.2 req/sec = 4320/hr).
	ProactiveRate = 1.2

	// MinBuffer is the minimum remaining requests before waiting for reset.
	MinBuffer = 100

	// HeaderRateLimit is the rate limit header.
	HeaderRateLimit = "X-RateLimit-Limit"

	// HeaderRateRemaining is the remaining requests header.
	HeaderRateRemaining = "X-RateLimit-Remaining"

	// HeaderRateReset is the reset timestamp header (Unix seconds).
	HeaderRateReset = "X-RateLimit-Reset"

	// HeaderRetryAfter is the retry-after header (seconds).
	HeaderRetryAfter = "Retry-After"
)

// RateLimiter implements dual-strategy rate limiting for GitHub API.
type RateLimiter struct {
	mu        sync.Mutex
	remaining int           // From API header
	limit     int           // From API header
	resetTime time.Time     // From API header
	bucket    *rate.Limiter // Proactive throttling
	minBuffer int           // Reserve requests
}

// NewRateLimiter creates a new rate limiter with proactive throttling.
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		remaining: GitHubRateLimit, // Assume full quota initially
		limit:     GitHubRateLimit,
		bucket:    rate.NewLimiter(rate.Limit(ProactiveRate), 1),
		minBuffer: MinBuffer,
	}
}

// Wait blocks until it's safe to make a request.
// It uses both proactive throttling and reactive API limit checking.
func (r *RateLimiter) Wait(ctx context.Context) error {
	// 1. Check token bucket (proactive throttling)
	if err := r.bucket.Wait(ctx); err != nil {
		return err
	}

	// 2. Check API limit (reactive)
	r.mu.Lock()
	remaining := r.remaining
	resetTime := r.resetTime
	r.mu.Unlock()

	if remaining < r.minBuffer && time.Now().Before(resetTime) {
		waitDuration := time.Until(resetTime)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitDuration):
		}
	}

	return nil
}

// UpdateFromResponse updates rate limit state from response headers.
func (r *RateLimiter) UpdateFromResponse(resp *http.Response) {
	if resp == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Parse X-RateLimit-Remaining
	if remaining := resp.Header.Get(HeaderRateRemaining); remaining != "" {
		if val, err := strconv.Atoi(remaining); err == nil {
			r.remaining = val
		}
	}

	// Parse X-RateLimit-Limit
	if limit := resp.Header.Get(HeaderRateLimit); limit != "" {
		if val, err := strconv.Atoi(limit); err == nil {
			r.limit = val
		}
	}

	// Parse X-RateLimit-Reset (Unix timestamp)
	if reset := resp.Header.Get(HeaderRateReset); reset != "" {
		if val, err := strconv.ParseInt(reset, 10, 64); err == nil {
			r.resetTime = time.Unix(val, 0)
		}
	}
}

// CheckRateLimit checks if the response indicates rate limiting.
// Returns a RateLimitError if rate limited, nil otherwise.
func (r *RateLimiter) CheckRateLimit(resp *http.Response) error {
	if resp == nil {
		return nil
	}

	// Update state from headers
	r.UpdateFromResponse(resp)

	// Check for rate limit responses (403 with rate limit or 429)
	if resp.StatusCode == 429 || (resp.StatusCode == 403 && r.remaining == 0) {
		r.mu.Lock()
		resetTime := r.resetTime
		remaining := r.remaining
		limit := r.limit
		r.mu.Unlock()

		// Check Retry-After header
		if retryAfter := resp.Header.Get(HeaderRetryAfter); retryAfter != "" {
			if seconds, err := strconv.Atoi(retryAfter); err == nil {
				resetTime = time.Now().Add(time.Duration(seconds) * time.Second)
			}
		}

		return &RateLimitError{
			ResetAt:   resetTime,
			Remaining: remaining,
			Limit:     limit,
		}
	}

	return nil
}

// Remaining returns the current remaining requests.
func (r *RateLimiter) Remaining() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.remaining
}

// Limit returns the rate limit.
func (r *RateLimiter) Limit() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.limit
}

// ResetTime returns the rate limit reset time.
func (r *RateLimiter) ResetTime() time.Time {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.resetTime
}

// WaitForReset waits until the rate limit resets.
func (r *RateLimiter) WaitForReset(ctx context.Context) error {
	r.mu.Lock()
	resetTime := r.resetTime
	r.mu.Unlock()

	if time.Now().After(resetTime) {
		return nil // Already reset
	}

	waitDuration := time.Until(resetTime)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(waitDuration):
		return nil
	}
}
