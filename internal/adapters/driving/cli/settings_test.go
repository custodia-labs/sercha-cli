package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test helper functions in settings.go

func TestMaskAPIKey(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Short key",
			input:    "abc123",
			expected: "****",
		},
		{
			name:     "Exactly 8 chars",
			input:    "12345678",
			expected: "****",
		},
		{
			name:     "Long key",
			input:    "sk-1234567890abcdef",
			expected: "sk-1...cdef",
		},
		{
			name:     "Very long key",
			input:    "sk-proj-1234567890abcdefghijklmnop",
			expected: "sk-p...mnop",
		},
		{
			name:     "Empty key",
			input:    "",
			expected: "****",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maskAPIKey(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseChoice(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		maxVal     int
		defaultVal int
		expected   int
	}{
		{
			name:       "Empty input returns default",
			input:      "",
			maxVal:     5,
			defaultVal: 1,
			expected:   1,
		},
		{
			name:       "Valid choice within range",
			input:      "3",
			maxVal:     5,
			defaultVal: 1,
			expected:   3,
		},
		{
			name:       "Choice below minimum returns default",
			input:      "0",
			maxVal:     5,
			defaultVal: 1,
			expected:   1,
		},
		{
			name:       "Choice above maximum returns default",
			input:      "6",
			maxVal:     5,
			defaultVal: 1,
			expected:   1,
		},
		{
			name:       "Invalid input returns default",
			input:      "abc",
			maxVal:     5,
			defaultVal: 2,
			expected:   2,
		},
		{
			name:       "Negative number returns default",
			input:      "-1",
			maxVal:     5,
			defaultVal: 1,
			expected:   1,
		},
		{
			name:       "Whitespace returns default",
			input:      "   ",
			maxVal:     5,
			defaultVal: 1,
			expected:   1,
		},
		{
			name:       "Maximum value is valid",
			input:      "5",
			maxVal:     5,
			defaultVal: 1,
			expected:   5,
		},
		{
			name:       "Minimum value is valid",
			input:      "1",
			maxVal:     5,
			defaultVal: 3,
			expected:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseChoice(tt.input, tt.maxVal, tt.defaultVal)
			assert.Equal(t, tt.expected, result)
		})
	}
}
