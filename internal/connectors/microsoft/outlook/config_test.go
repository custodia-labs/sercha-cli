package outlook

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.Equal(t, "inbox", cfg.FolderID)
	assert.Equal(t, int64(100), cfg.MaxResults)
	assert.False(t, cfg.IncludeSpamTrash)
}

func TestParseConfig_Defaults(t *testing.T) {
	source := domain.Source{
		Config: map[string]string{},
	}

	cfg, err := ParseConfig(source)

	require.NoError(t, err)
	assert.Equal(t, "inbox", cfg.FolderID)
	assert.Equal(t, int64(100), cfg.MaxResults)
	assert.False(t, cfg.IncludeSpamTrash)
}

func TestParseConfig_FolderID(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{
			name:     "inbox",
			value:    "inbox",
			expected: "inbox",
		},
		{
			name:     "sent items",
			value:    "sentitems",
			expected: "sentitems",
		},
		{
			name:     "drafts",
			value:    "drafts",
			expected: "drafts",
		},
		{
			name:     "custom folder ID",
			value:    "AAMkAGI2THVSAAA=",
			expected: "AAMkAGI2THVSAAA=",
		},
		{
			name:     "with spaces",
			value:    "  inbox  ",
			expected: "inbox",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := domain.Source{
				Config: map[string]string{
					"folder_id": tt.value,
				},
			}

			cfg, err := ParseConfig(source)

			require.NoError(t, err)
			assert.Equal(t, tt.expected, cfg.FolderID)
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
			value:    "50",
			expected: 50,
		},
		{
			name:     "max allowed",
			value:    "1000",
			expected: 1000,
		},
		{
			name:     "over max is capped",
			value:    "2000",
			expected: 1000,
		},
		{
			name:     "invalid number falls back to default",
			value:    "not-a-number",
			expected: 100,
		},
		{
			name:     "zero falls back to default",
			value:    "0",
			expected: 100,
		},
		{
			name:     "negative falls back to default",
			value:    "-10",
			expected: 100,
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

func TestParseConfig_IncludeSpamTrash(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected bool
	}{
		{
			name:     "default is false",
			value:    "",
			expected: false,
		},
		{
			name:     "true enables",
			value:    "true",
			expected: true,
		},
		{
			name:     "false keeps default",
			value:    "false",
			expected: false,
		},
		{
			name:     "any other value keeps default",
			value:    "yes",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := domain.Source{
				Config: map[string]string{
					"include_spam_trash": tt.value,
				},
			}

			cfg, err := ParseConfig(source)

			require.NoError(t, err)
			assert.Equal(t, tt.expected, cfg.IncludeSpamTrash)
		})
	}
}

func TestParseConfig_AllOptions(t *testing.T) {
	source := domain.Source{
		Config: map[string]string{
			"folder_id":          "sentitems",
			"max_results":        "500",
			"include_spam_trash": "true",
		},
	}

	cfg, err := ParseConfig(source)

	require.NoError(t, err)
	assert.Equal(t, "sentitems", cfg.FolderID)
	assert.Equal(t, int64(500), cfg.MaxResults)
	assert.True(t, cfg.IncludeSpamTrash)
}
