package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrors_AreDistinct(t *testing.T) {
	errors := []error{
		ErrMissingSearchService,
		ErrMissingSourceService,
		ErrMissingSyncOrchestrator,
		ErrInvalidPorts,
	}

	// Ensure all errors are unique
	seen := make(map[string]bool)
	for _, err := range errors {
		msg := err.Error()
		assert.False(t, seen[msg], "duplicate error message: %s", msg)
		seen[msg] = true
	}
}

func TestErrMissingSearchService_Message(t *testing.T) {
	assert.Contains(t, ErrMissingSearchService.Error(), "search service")
}

func TestErrMissingSourceService_Message(t *testing.T) {
	assert.Contains(t, ErrMissingSourceService.Error(), "source service")
}

func TestErrMissingSyncOrchestrator_Message(t *testing.T) {
	assert.Contains(t, ErrMissingSyncOrchestrator.Error(), "sync orchestrator")
}

func TestErrInvalidPorts_Message(t *testing.T) {
	assert.Contains(t, ErrInvalidPorts.Error(), "invalid ports")
}
