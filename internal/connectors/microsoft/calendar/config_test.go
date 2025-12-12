package calendar

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.Equal(t, int64(100), cfg.MaxResults)
	assert.Empty(t, cfg.CalendarIDs)
	assert.False(t, cfg.ShowCancelled)
	assert.True(t, cfg.SingleEvents)
}

func TestParseConfig_Default(t *testing.T) {
	source := domain.Source{
		Config: make(map[string]string),
	}

	cfg, err := ParseConfig(source)

	require.NoError(t, err)
	assert.Equal(t, int64(100), cfg.MaxResults)
	assert.Empty(t, cfg.CalendarIDs)
	assert.False(t, cfg.ShowCancelled)
	assert.True(t, cfg.SingleEvents)
}

func TestParseConfig_WithCalendarIDs(t *testing.T) {
	source := domain.Source{
		Config: map[string]string{
			"calendar_ids": "cal-1, cal-2, cal-3",
		},
	}

	cfg, err := ParseConfig(source)

	require.NoError(t, err)
	assert.Equal(t, []string{"cal-1", "cal-2", "cal-3"}, cfg.CalendarIDs)
}

func TestParseConfig_WithMaxResults(t *testing.T) {
	source := domain.Source{
		Config: map[string]string{
			"max_results": "50",
		},
	}

	cfg, err := ParseConfig(source)

	require.NoError(t, err)
	assert.Equal(t, int64(50), cfg.MaxResults)
}

func TestParseConfig_WithInvalidMaxResults(t *testing.T) {
	source := domain.Source{
		Config: map[string]string{
			"max_results": "not-a-number",
		},
	}

	cfg, err := ParseConfig(source)

	require.NoError(t, err)
	assert.Equal(t, int64(100), cfg.MaxResults) // Should use default
}

func TestParseConfig_WithShowCancelled(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected bool
	}{
		{"true string", "true", true},
		{"1 string", "1", true},
		{"false string", "false", false},
		{"0 string", "0", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := domain.Source{
				Config: map[string]string{
					"show_cancelled": tt.value,
				},
			}

			cfg, err := ParseConfig(source)

			require.NoError(t, err)
			assert.Equal(t, tt.expected, cfg.ShowCancelled)
		})
	}
}

func TestParseConfig_WithSingleEvents(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected bool
	}{
		{"true string", "true", true},
		{"1 string", "1", true},
		{"false string", "false", false},
		{"0 string", "0", false},
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
			"calendar_ids":   "primary, work",
			"max_results":    "25",
			"show_cancelled": "true",
			"single_events":  "false",
		},
	}

	cfg, err := ParseConfig(source)

	require.NoError(t, err)
	assert.Equal(t, []string{"primary", "work"}, cfg.CalendarIDs)
	assert.Equal(t, int64(25), cfg.MaxResults)
	assert.True(t, cfg.ShowCancelled)
	assert.False(t, cfg.SingleEvents)
}
