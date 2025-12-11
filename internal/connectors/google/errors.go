package google

import (
	"errors"
	"net/http"

	"google.golang.org/api/googleapi"
)

// Common Google API errors.
var (
	// ErrUnauthorized indicates invalid or expired credentials.
	ErrUnauthorized = errors.New("google: unauthorised (invalid credentials)")

	// ErrForbidden indicates insufficient permissions.
	ErrForbidden = errors.New("google: forbidden (insufficient permissions)")

	// ErrNotFound indicates the requested resource was not found.
	ErrNotFound = errors.New("google: resource not found")

	// ErrRateLimited indicates the API rate limit was exceeded.
	ErrRateLimited = errors.New("google: rate limit exceeded")

	// ErrQuotaExceeded indicates the API quota was exceeded.
	ErrQuotaExceeded = errors.New("google: quota exceeded")

	// ErrSyncTokenExpired indicates the sync token has expired (410 GONE).
	// The client should perform a full resync.
	ErrSyncTokenExpired = errors.New("google: sync token expired, full resync required")

	// ErrHistoryIDExpired indicates the Gmail historyId is no longer valid.
	// The client should perform a full resync.
	ErrHistoryIDExpired = errors.New("google: history ID expired, full resync required")
)

// IsUnauthorized returns true if the error indicates invalid credentials.
func IsUnauthorized(err error) bool {
	if errors.Is(err, ErrUnauthorized) {
		return true
	}
	var gerr *googleapi.Error
	if errors.As(err, &gerr) {
		return gerr.Code == http.StatusUnauthorized
	}
	return false
}

// IsForbidden returns true if the error indicates insufficient permissions.
func IsForbidden(err error) bool {
	if errors.Is(err, ErrForbidden) {
		return true
	}
	var gerr *googleapi.Error
	if errors.As(err, &gerr) {
		return gerr.Code == http.StatusForbidden
	}
	return false
}

// IsNotFound returns true if the error indicates a missing resource.
func IsNotFound(err error) bool {
	if errors.Is(err, ErrNotFound) {
		return true
	}
	var gerr *googleapi.Error
	if errors.As(err, &gerr) {
		return gerr.Code == http.StatusNotFound
	}
	return false
}

// IsRateLimited returns true if the error indicates rate limiting.
func IsRateLimited(err error) bool {
	if errors.Is(err, ErrRateLimited) {
		return true
	}
	var gerr *googleapi.Error
	if errors.As(err, &gerr) {
		return gerr.Code == http.StatusTooManyRequests
	}
	return false
}

// IsSyncTokenExpired returns true if the error indicates an expired sync token (410 GONE).
// This is used by Calendar and sometimes Drive to indicate the client needs to resync.
func IsSyncTokenExpired(err error) bool {
	if errors.Is(err, ErrSyncTokenExpired) {
		return true
	}
	var gerr *googleapi.Error
	if errors.As(err, &gerr) {
		return gerr.Code == http.StatusGone
	}
	return false
}

// IsHistoryIDExpired returns true if the error indicates an expired Gmail history ID.
// Gmail returns 404 with a specific error when the historyId is no longer valid.
func IsHistoryIDExpired(err error) bool {
	if errors.Is(err, ErrHistoryIDExpired) {
		return true
	}
	var gerr *googleapi.Error
	if errors.As(err, &gerr) {
		// Gmail returns 404 when historyId is too old
		if gerr.Code == http.StatusNotFound {
			return true
		}
	}
	return false
}

// WrapError converts a Google API error to a more specific error type.
func WrapError(err error) error {
	if err == nil {
		return nil
	}

	var gerr *googleapi.Error
	if !errors.As(err, &gerr) {
		return err
	}

	switch gerr.Code {
	case http.StatusUnauthorized:
		return ErrUnauthorized
	case http.StatusForbidden:
		return ErrForbidden
	case http.StatusNotFound:
		return ErrNotFound
	case http.StatusTooManyRequests:
		return ErrRateLimited
	case http.StatusGone:
		return ErrSyncTokenExpired
	default:
		return err
	}
}
