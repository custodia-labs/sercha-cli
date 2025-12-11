package calendar

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.Empty(t, cfg.CalendarIDs)
	assert.Equal(t, int64(250), cfg.MaxResults)
	assert.True(t, cfg.ShowDeleted)
	assert.True(t, cfg.SingleEvents)
}

func TestParseConfig_Defaults(t *testing.T) {
	source := domain.Source{
		Config: map[string]string{},
	}

	cfg, err := ParseConfig(source)

	require.NoError(t, err)
	assert.Empty(t, cfg.CalendarIDs)
	assert.Equal(t, int64(250), cfg.MaxResults)
	assert.True(t, cfg.ShowDeleted)
	assert.True(t, cfg.SingleEvents)
}

func TestParseConfig_CalendarIDs(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected []string
	}{
		{
			name:     "single calendar ID",
			value:    "primary",
			expected: []string{"primary"},
		},
		{
			name:     "multiple calendar IDs",
			value:    "primary,work@example.com,holidays",
			expected: []string{"primary", "work@example.com", "holidays"},
		},
		{
			name:     "calendar IDs with spaces",
			value:    "primary, work@example.com , holidays",
			expected: []string{"primary", "work@example.com", "holidays"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := domain.Source{
				Config: map[string]string{
					"calendar_ids": tt.value,
				},
			}

			cfg, err := ParseConfig(source)

			require.NoError(t, err)
			assert.Equal(t, tt.expected, cfg.CalendarIDs)
		})
	}
}

func TestParseConfig_MaxResults(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected int64
	}{
		{
			name:     "valid number",
			value:    "100",
			expected: 100,
		},
		{
			name:     "large number",
			value:    "500",
			expected: 500,
		},
		{
			name:     "invalid number falls back to default",
			value:    "not-a-number",
			expected: 250,
		},
		{
			name:     "zero falls back to default",
			value:    "0",
			expected: 250,
		},
		{
			name:     "negative falls back to default",
			value:    "-10",
			expected: 250,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := domain.Source{
				Config: map[string]string{
					"max_results": tt.value,
				},
			}

			cfg, err := ParseConfig(source)

			require.NoError(t, err)
			assert.Equal(t, tt.expected, cfg.MaxResults)
		})
	}
}

func TestParseConfig_ShowDeleted(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected bool
	}{
		{
			name:     "default is true",
			value:    "",
			expected: true,
		},
		{
			name:     "false disables",
			value:    "false",
			expected: false,
		},
		{
			name:     "any other value keeps default",
			value:    "no",
			expected: true,
		},
		{
			name:     "true keeps true",
			value:    "true",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := domain.Source{
				Config: map[string]string{
					"show_deleted": tt.value,
				},
			}

			cfg, err := ParseConfig(source)

			require.NoError(t, err)
			assert.Equal(t, tt.expected, cfg.ShowDeleted)
		})
	}
}

func TestParseConfig_SingleEvents(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected bool
	}{
		{
			name:     "default is true",
			value:    "",
			expected: true,
		},
		{
			name:     "false disables",
			value:    "false",
			expected: false,
		},
		{
			name:     "any other value keeps default",
			value:    "no",
			expected: true,
		},
		{
			name:     "true keeps true",
			value:    "true",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := domain.Source{
				Config: map[string]string{
					"single_events": tt.value,
				},
			}

			cfg, err := ParseConfig(source)

			require.NoError(t, err)
			assert.Equal(t, tt.expected, cfg.SingleEvents)
		})
	}
}

func TestParseConfig_AllOptions(t *testing.T) {
	source := domain.Source{
		Config: map[string]string{
			"calendar_ids":  "primary,work",
			"max_results":   "100",
			"show_deleted":  "false",
			"single_events": "false",
		},
	}

	cfg, err := ParseConfig(source)

	require.NoError(t, err)
	assert.Equal(t, []string{"primary", "work"}, cfg.CalendarIDs)
	assert.Equal(t, int64(100), cfg.MaxResults)
	assert.False(t, cfg.ShowDeleted)
	assert.False(t, cfg.SingleEvents)
}
