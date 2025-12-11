package domain

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestErrors_Existence tests that all error variables exist and are not nil
func TestErrors_Existence(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"ErrNotFound", ErrNotFound},
		{"ErrAlreadyExists", ErrAlreadyExists},
		{"ErrInvalidInput", ErrInvalidInput},
		{"ErrNotImplemented", ErrNotImplemented},
		{"ErrUnsupportedType", ErrUnsupportedType},
		{"ErrSyncInProgress", ErrSyncInProgress},
		{"ErrLLMUnavailable", ErrLLMUnavailable},
		{"ErrEmbeddingUnavailable", ErrEmbeddingUnavailable},
		{"ErrSearchUnavailable", ErrSearchUnavailable},
		{"ErrVectorIndexUnavailable", ErrVectorIndexUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotNil(t, tt.err)
			assert.NotEmpty(t, tt.err.Error())
		})
	}
}

// TestErrNotFound tests ErrNotFound error
func TestErrNotFound(t *testing.T) {
	assert.Equal(t, "not found", ErrNotFound.Error())
	assert.True(t, errors.Is(ErrNotFound, ErrNotFound))
	assert.False(t, errors.Is(ErrNotFound, ErrAlreadyExists))
}

// TestErrAlreadyExists tests ErrAlreadyExists error
func TestErrAlreadyExists(t *testing.T) {
	assert.Equal(t, "already exists", ErrAlreadyExists.Error())
	assert.True(t, errors.Is(ErrAlreadyExists, ErrAlreadyExists))
	assert.False(t, errors.Is(ErrAlreadyExists, ErrNotFound))
}

// TestErrInvalidInput tests ErrInvalidInput error
func TestErrInvalidInput(t *testing.T) {
	assert.Equal(t, "invalid input", ErrInvalidInput.Error())
	assert.True(t, errors.Is(ErrInvalidInput, ErrInvalidInput))
	assert.False(t, errors.Is(ErrInvalidInput, ErrNotFound))
}

// TestErrNotImplemented tests ErrNotImplemented error
func TestErrNotImplemented(t *testing.T) {
	assert.Equal(t, "not implemented", ErrNotImplemented.Error())
	assert.True(t, errors.Is(ErrNotImplemented, ErrNotImplemented))
	assert.False(t, errors.Is(ErrNotImplemented, ErrNotFound))
}

// TestErrUnsupportedType tests ErrUnsupportedType error
func TestErrUnsupportedType(t *testing.T) {
	assert.Equal(t, "unsupported type", ErrUnsupportedType.Error())
	assert.True(t, errors.Is(ErrUnsupportedType, ErrUnsupportedType))
	assert.False(t, errors.Is(ErrUnsupportedType, ErrNotFound))
}

// TestErrSyncInProgress tests ErrSyncInProgress error
func TestErrSyncInProgress(t *testing.T) {
	assert.Equal(t, "sync in progress", ErrSyncInProgress.Error())
	assert.True(t, errors.Is(ErrSyncInProgress, ErrSyncInProgress))
	assert.False(t, errors.Is(ErrSyncInProgress, ErrNotFound))
}

// TestErrLLMUnavailable tests ErrLLMUnavailable error
func TestErrLLMUnavailable(t *testing.T) {
	assert.Equal(t, "LLM service unavailable", ErrLLMUnavailable.Error())
	assert.True(t, errors.Is(ErrLLMUnavailable, ErrLLMUnavailable))
	assert.False(t, errors.Is(ErrLLMUnavailable, ErrEmbeddingUnavailable))
}

// TestErrEmbeddingUnavailable tests ErrEmbeddingUnavailable error
func TestErrEmbeddingUnavailable(t *testing.T) {
	assert.Equal(t, "embedding service unavailable", ErrEmbeddingUnavailable.Error())
	assert.True(t, errors.Is(ErrEmbeddingUnavailable, ErrEmbeddingUnavailable))
	assert.False(t, errors.Is(ErrEmbeddingUnavailable, ErrLLMUnavailable))
}

// TestErrSearchUnavailable tests ErrSearchUnavailable error
func TestErrSearchUnavailable(t *testing.T) {
	assert.Equal(t, "search engine unavailable", ErrSearchUnavailable.Error())
	assert.True(t, errors.Is(ErrSearchUnavailable, ErrSearchUnavailable))
	assert.False(t, errors.Is(ErrSearchUnavailable, ErrVectorIndexUnavailable))
}

// TestErrVectorIndexUnavailable tests ErrVectorIndexUnavailable error
func TestErrVectorIndexUnavailable(t *testing.T) {
	assert.Equal(t, "vector index unavailable", ErrVectorIndexUnavailable.Error())
	assert.True(t, errors.Is(ErrVectorIndexUnavailable, ErrVectorIndexUnavailable))
	assert.False(t, errors.Is(ErrVectorIndexUnavailable, ErrSearchUnavailable))
}

// TestErrors_Uniqueness tests that all errors are distinct
func TestErrors_Uniqueness(t *testing.T) {
	allErrors := []error{
		ErrNotFound,
		ErrAlreadyExists,
		ErrInvalidInput,
		ErrNotImplemented,
		ErrUnsupportedType,
		ErrSyncInProgress,
		ErrLLMUnavailable,
		ErrEmbeddingUnavailable,
		ErrSearchUnavailable,
		ErrVectorIndexUnavailable,
	}

	// Check that each error is unique
	for i, err1 := range allErrors {
		for j, err2 := range allErrors {
			if i != j {
				assert.False(t, errors.Is(err1, err2),
					"Error %v should not match error %v", err1, err2)
			}
		}
	}
}

// TestErrors_WithWrapping tests error wrapping behavior
func TestErrors_WithWrapping(t *testing.T) {
	// Wrap ErrNotFound
	wrappedErr := errors.Join(ErrNotFound, errors.New("additional context"))

	// Should still be identifiable as ErrNotFound
	assert.True(t, errors.Is(wrappedErr, ErrNotFound))
	assert.Contains(t, wrappedErr.Error(), "not found")
}

// TestErrors_ErrorMessages tests that error messages are descriptive
func TestErrors_ErrorMessages(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		shouldHave []string
	}{
		{
			name:       "ErrNotFound message",
			err:        ErrNotFound,
			shouldHave: []string{"not", "found"},
		},
		{
			name:       "ErrAlreadyExists message",
			err:        ErrAlreadyExists,
			shouldHave: []string{"already", "exists"},
		},
		{
			name:       "ErrInvalidInput message",
			err:        ErrInvalidInput,
			shouldHave: []string{"invalid", "input"},
		},
		{
			name:       "ErrLLMUnavailable message",
			err:        ErrLLMUnavailable,
			shouldHave: []string{"LLM", "unavailable"},
		},
		{
			name:       "ErrEmbeddingUnavailable message",
			err:        ErrEmbeddingUnavailable,
			shouldHave: []string{"embedding", "unavailable"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := tt.err.Error()
			for _, word := range tt.shouldHave {
				assert.Contains(t, msg, word)
			}
		})
	}
}

// TestErrors_InSwitchStatement tests using errors in switch statements
func TestErrors_InSwitchStatement(t *testing.T) {
	testErr := ErrNotFound

	var result string
	switch {
	case errors.Is(testErr, ErrNotFound):
		result = "not found"
	case errors.Is(testErr, ErrAlreadyExists):
		result = "already exists"
	default:
		result = "unknown"
	}

	assert.Equal(t, "not found", result)
}

// TestErrors_ComparingWithIs tests errors.Is comparison
func TestErrors_ComparingWithIs(t *testing.T) {
	// Test direct comparison
	assert.True(t, errors.Is(ErrNotFound, ErrNotFound))

	// Test with wrapped error
	wrapped := errors.Join(errors.New("context"), ErrInvalidInput)
	assert.True(t, errors.Is(wrapped, ErrInvalidInput))

	// Test negative case
	assert.False(t, errors.Is(ErrNotFound, ErrAlreadyExists))
}

// TestErrors_DirectComparison tests that domain errors can be compared directly
func TestErrors_DirectComparison(t *testing.T) {
	// These are simple errors, not custom types
	// They can be compared directly
	assert.Equal(t, ErrNotFound, ErrNotFound)
	assert.NotEqual(t, ErrNotFound, ErrAlreadyExists)
}

// TestErrors_ServiceErrors tests service-related errors
func TestErrors_ServiceErrors(t *testing.T) {
	serviceErrors := []error{
		ErrLLMUnavailable,
		ErrEmbeddingUnavailable,
		ErrSearchUnavailable,
		ErrVectorIndexUnavailable,
	}

	// All should contain "unavailable" in their message
	for _, err := range serviceErrors {
		assert.Contains(t, err.Error(), "unavailable",
			"Service error %v should mention unavailable", err)
	}
}

// TestErrors_DataErrors tests data-related errors
func TestErrors_DataErrors(t *testing.T) {
	dataErrors := map[string]error{
		"not found":      ErrNotFound,
		"already exists": ErrAlreadyExists,
		"invalid input":  ErrInvalidInput,
	}

	for expectedMsg, err := range dataErrors {
		assert.Equal(t, expectedMsg, err.Error())
	}
}

// TestErrors_OperationErrors tests operation-related errors
func TestErrors_OperationErrors(t *testing.T) {
	operationErrors := []error{
		ErrNotImplemented,
		ErrUnsupportedType,
		ErrSyncInProgress,
	}

	// All should be non-nil and have messages
	for _, err := range operationErrors {
		assert.NotNil(t, err)
		assert.NotEmpty(t, err.Error())
	}
}
