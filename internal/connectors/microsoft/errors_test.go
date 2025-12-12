package microsoft

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWrapError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		expected   error
	}{
		{
			name:       "unauthorised",
			statusCode: http.StatusUnauthorized,
			expected:   ErrUnauthorised,
		},
		{
			name:       "forbidden",
			statusCode: http.StatusForbidden,
			expected:   ErrForbidden,
		},
		{
			name:       "not found",
			statusCode: http.StatusNotFound,
			expected:   ErrNotFound,
		},
		{
			name:       "gone (delta token expired)",
			statusCode: http.StatusGone,
			expected:   ErrDeltaTokenExpired,
		},
		{
			name:       "rate limited",
			statusCode: http.StatusTooManyRequests,
			expected:   ErrRateLimited,
		},
		{
			name:       "bad request",
			statusCode: http.StatusBadRequest,
			expected:   ErrBadRequest,
		},
		{
			name:       "internal server error",
			statusCode: http.StatusInternalServerError,
			expected:   ErrServerError,
		},
		{
			name:       "service unavailable",
			statusCode: http.StatusServiceUnavailable,
			expected:   ErrServerError,
		},
		{
			name:       "success returns nil",
			statusCode: http.StatusOK,
			expected:   nil,
		},
		{
			name:       "created returns nil",
			statusCode: http.StatusCreated,
			expected:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WrapError(tt.statusCode)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsUnauthorised(t *testing.T) {
	assert.True(t, IsUnauthorised(http.StatusUnauthorized))
	assert.False(t, IsUnauthorised(http.StatusOK))
	assert.False(t, IsUnauthorised(http.StatusForbidden))
}

func TestIsDeltaTokenExpired(t *testing.T) {
	assert.True(t, IsDeltaTokenExpired(http.StatusGone))
	assert.False(t, IsDeltaTokenExpired(http.StatusOK))
	assert.False(t, IsDeltaTokenExpired(http.StatusUnauthorized))
}

func TestIsRateLimited(t *testing.T) {
	assert.True(t, IsRateLimited(http.StatusTooManyRequests))
	assert.False(t, IsRateLimited(http.StatusOK))
	assert.False(t, IsRateLimited(http.StatusUnauthorized))
}

func TestIsNotFound(t *testing.T) {
	assert.True(t, IsNotFound(http.StatusNotFound))
	assert.False(t, IsNotFound(http.StatusOK))
	assert.False(t, IsNotFound(http.StatusUnauthorized))
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		expected   bool
	}{
		{
			name:       "rate limited is retryable",
			statusCode: http.StatusTooManyRequests,
			expected:   true,
		},
		{
			name:       "service unavailable is retryable",
			statusCode: http.StatusServiceUnavailable,
			expected:   true,
		},
		{
			name:       "gateway timeout is retryable",
			statusCode: http.StatusGatewayTimeout,
			expected:   true,
		},
		{
			name:       "unauthorised is not retryable",
			statusCode: http.StatusUnauthorized,
			expected:   false,
		},
		{
			name:       "not found is not retryable",
			statusCode: http.StatusNotFound,
			expected:   false,
		},
		{
			name:       "internal server error is not retryable",
			statusCode: http.StatusInternalServerError,
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRetryable(tt.statusCode)
			assert.Equal(t, tt.expected, result)
		})
	}
}
