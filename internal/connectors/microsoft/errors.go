package microsoft

import (
	"errors"
	"net/http"
)

// Error types for Microsoft Graph API responses.
var (
	// ErrUnauthorised indicates the access token is invalid or expired.
	ErrUnauthorised = errors.New("microsoft: unauthorised")

	// ErrForbidden indicates the user lacks permission for the requested resource.
	ErrForbidden = errors.New("microsoft: forbidden")

	// ErrNotFound indicates the requested resource does not exist.
	ErrNotFound = errors.New("microsoft: not found")

	// ErrRateLimited indicates the request was throttled by Microsoft Graph.
	ErrRateLimited = errors.New("microsoft: rate limited")

	// ErrDeltaTokenExpired indicates the delta sync token has expired.
	// A full sync is required when this error occurs.
	ErrDeltaTokenExpired = errors.New("microsoft: delta token expired, full sync required")

	// ErrBadRequest indicates the request was malformed.
	ErrBadRequest = errors.New("microsoft: bad request")

	// ErrServerError indicates a server-side error from Microsoft Graph.
	ErrServerError = errors.New("microsoft: server error")
)

// WrapError converts an HTTP status code to an appropriate error.
func WrapError(statusCode int) error {
	switch statusCode {
	case http.StatusUnauthorized:
		return ErrUnauthorised
	case http.StatusForbidden:
		return ErrForbidden
	case http.StatusNotFound:
		return ErrNotFound
	case http.StatusGone:
		return ErrDeltaTokenExpired
	case http.StatusTooManyRequests:
		return ErrRateLimited
	case http.StatusBadRequest:
		return ErrBadRequest
	default:
		if statusCode >= 500 {
			return ErrServerError
		}
		return nil
	}
}

// IsUnauthorised checks if the status code indicates an authentication failure.
func IsUnauthorised(statusCode int) bool {
	return statusCode == http.StatusUnauthorized
}

// IsDeltaTokenExpired checks if the status code indicates an expired delta token.
// Microsoft Graph returns 410 Gone when the delta token has expired.
func IsDeltaTokenExpired(statusCode int) bool {
	return statusCode == http.StatusGone
}

// IsRateLimited checks if the status code indicates rate limiting.
func IsRateLimited(statusCode int) bool {
	return statusCode == http.StatusTooManyRequests
}

// IsNotFound checks if the status code indicates a missing resource.
func IsNotFound(statusCode int) bool {
	return statusCode == http.StatusNotFound
}

// IsRetryable checks if the error is potentially transient and can be retried.
func IsRetryable(statusCode int) bool {
	return statusCode == http.StatusTooManyRequests ||
		statusCode == http.StatusServiceUnavailable ||
		statusCode == http.StatusGatewayTimeout
}
