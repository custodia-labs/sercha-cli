package gmail

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.Equal(t, []string{"INBOX"}, cfg.LabelIDs)
	assert.Equal(t, int64(100), cfg.MaxResults)
	assert.Empty(t, cfg.Query)
	assert.False(t, cfg.IncludeSpamTrash)
}

func TestParseConfig_Defaults(t *testing.T) {
	source := domain.Source{
		Config: map[string]string{},
	}

	cfg, err := ParseConfig(source)

	require.NoError(t, err)
	assert.Equal(t, []string{"INBOX"}, cfg.LabelIDs)
	assert.Equal(t, int64(100), cfg.MaxResults)
	assert.Empty(t, cfg.Query)
	assert.False(t, cfg.IncludeSpamTrash)
}

func TestParseConfig_LabelIDs(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected []string
	}{
		{
			name:     "single label",
			value:    "SENT",
			expected: []string{"SENT"},
		},
		{
			name:     "multiple labels",
			value:    "INBOX,SENT,IMPORTANT",
			expected: []string{"INBOX", "SENT", "IMPORTANT"},
		},
		{
			name:     "labels with spaces",
			value:    "INBOX, SENT , IMPORTANT",
			expected: []string{"INBOX", "SENT", "IMPORTANT"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := domain.Source{
				Config: map[string]string{
					"label_ids": tt.value,
				},
			}

			cfg, err := ParseConfig(source)

			require.NoError(t, err)
			assert.Equal(t, tt.expected, cfg.LabelIDs)
		})
	}
}

func TestParseConfig_Query(t *testing.T) {
	source := domain.Source{
		Config: map[string]string{
			"query": "from:boss@company.com is:unread",
		},
	}

	cfg, err := ParseConfig(source)

	require.NoError(t, err)
	assert.Equal(t, "from:boss@company.com is:unread", cfg.Query)
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
			name:     "large number",
			value:    "500",
			expected: 500,
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
			name:     "true",
			value:    "true",
			expected: true,
		},
		{
			name:     "false",
			value:    "false",
			expected: false,
		},
		{
			name:     "any other value is false",
			value:    "yes",
			expected: false,
		},
		{
			name:     "empty is false",
			value:    "",
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
			"label_ids":          "INBOX,STARRED",
			"query":              "has:attachment",
			"max_results":        "200",
			"include_spam_trash": "true",
		},
	}

	cfg, err := ParseConfig(source)

	require.NoError(t, err)
	assert.Equal(t, []string{"INBOX", "STARRED"}, cfg.LabelIDs)
	assert.Equal(t, "has:attachment", cfg.Query)
	assert.Equal(t, int64(200), cfg.MaxResults)
	assert.True(t, cfg.IncludeSpamTrash)
}
