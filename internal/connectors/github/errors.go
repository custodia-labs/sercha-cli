package github

import (
	"errors"
	"fmt"
	"time"
)

// GitHub-specific errors.
var (
	// ErrConfigInvalidContentType indicates an invalid content type was specified.
	ErrConfigInvalidContentType = errors.New("github: invalid content type")

	// ErrRepoNotFound indicates the repository was not found or is not accessible.
	ErrRepoNotFound = errors.New("github: repository not found")

	// ErrBranchNotFound indicates the specified branch was not found.
	ErrBranchNotFound = errors.New("github: branch not found")

	// ErrWikiDisabled indicates the repository's wiki is disabled.
	ErrWikiDisabled = errors.New("github: wiki is disabled for this repository")

	// ErrInvalidCursor indicates the cursor format is invalid.
	ErrInvalidCursor = errors.New("github: invalid cursor format")
)

// RateLimitError represents a rate limit exceeded error with reset time.
type RateLimitError struct {
	ResetAt   time.Time
	Remaining int
	Limit     int
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("github: rate limit exceeded, resets at %s", e.ResetAt.Format(time.RFC3339))
}

// APIError represents a GitHub API error response.
type APIError struct {
	StatusCode int
	Message    string
	URL        string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("github: API error %d: %s (URL: %s)", e.StatusCode, e.Message, e.URL)
}

// IsNotFound checks if the error indicates a resource was not found.
func IsNotFound(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == 404
	}
	return errors.Is(err, ErrRepoNotFound) || errors.Is(err, ErrBranchNotFound)
}

// IsRateLimited checks if the error indicates rate limiting.
func IsRateLimited(err error) bool {
	var rateLimitErr *RateLimitError
	return errors.As(err, &rateLimitErr)
}

// IsUnauthorized checks if the error indicates an authentication failure.
func IsUnauthorized(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == 401
	}
	return false
}

// IsForbidden checks if the error indicates a forbidden resource.
func IsForbidden(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == 403
	}
	return false
}
