package dropbox

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.Equal(t, "", cfg.FolderPath)
	assert.Equal(t, uint32(100), cfg.MaxResults)
	assert.True(t, cfg.Recursive)
	assert.Empty(t, cfg.MimeTypeFilter)
}

func TestParseConfig_Default(t *testing.T) {
	source := domain.Source{
		Config: make(map[string]string),
	}

	cfg, err := ParseConfig(source)

	require.NoError(t, err)
	assert.Equal(t, "", cfg.FolderPath)
	assert.Equal(t, uint32(100), cfg.MaxResults)
	assert.True(t, cfg.Recursive)
	assert.Empty(t, cfg.MimeTypeFilter)
}

func TestParseConfig_WithFolderPath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"with leading slash", "/Documents", "/Documents"},
		{"without leading slash", "Documents", "/Documents"},
		{"nested path", "Documents/Work", "/Documents/Work"},
		{"nested with slash", "/Documents/Work", "/Documents/Work"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := domain.Source{
				Config: map[string]string{
					"folder_path": tt.input,
				},
			}

			cfg, err := ParseConfig(source)

			require.NoError(t, err)
			assert.Equal(t, tt.expected, cfg.FolderPath)
		})
	}
}

func TestParseConfig_WithMimeTypes(t *testing.T) {
	source := domain.Source{
		Config: map[string]string{
			"mime_types": "text/plain, application/pdf, text/markdown",
		},
	}

	cfg, err := ParseConfig(source)

	require.NoError(t, err)
	assert.Equal(t, []string{"text/plain", "application/pdf", "text/markdown"}, cfg.MimeTypeFilter)
}

func TestParseConfig_WithMaxResults(t *testing.T) {
	source := domain.Source{
		Config: map[string]string{
			"max_results": "50",
		},
	}

	cfg, err := ParseConfig(source)

	require.NoError(t, err)
	assert.Equal(t, uint32(50), cfg.MaxResults)
}

func TestParseConfig_WithInvalidMaxResults(t *testing.T) {
	source := domain.Source{
		Config: map[string]string{
			"max_results": "not-a-number",
		},
	}

	cfg, err := ParseConfig(source)

	require.NoError(t, err)
	assert.Equal(t, uint32(100), cfg.MaxResults) // Should use default
}

func TestParseConfig_WithZeroMaxResults(t *testing.T) {
	source := domain.Source{
		Config: map[string]string{
			"max_results": "0",
		},
	}

	cfg, err := ParseConfig(source)

	require.NoError(t, err)
	assert.Equal(t, uint32(100), cfg.MaxResults) // Should use default for zero
}

func TestParseConfig_WithRecursive(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected bool
	}{
		{"true string", "true", true},
		{"1 string", "1", true},
		{"false string", "false", false},
		{"0 string", "0", false},
		{"other string", "yes", false},
		{"empty string", "", true}, // default is true
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := domain.Source{
				Config: map[string]string{
					"recursive": tt.value,
				},
			}

			cfg, err := ParseConfig(source)

			require.NoError(t, err)
			assert.Equal(t, tt.expected, cfg.Recursive)
		})
	}
}

func TestParseConfig_AllOptions(t *testing.T) {
	source := domain.Source{
		Config: map[string]string{
			"folder_path": "/Documents/Work",
			"mime_types":  "text/plain,application/pdf",
			"max_results": "25",
			"recursive":   "false",
		},
	}

	cfg, err := ParseConfig(source)

	require.NoError(t, err)
	assert.Equal(t, "/Documents/Work", cfg.FolderPath)
	assert.Equal(t, []string{"text/plain", "application/pdf"}, cfg.MimeTypeFilter)
	assert.Equal(t, uint32(25), cfg.MaxResults)
	assert.False(t, cfg.Recursive)
}
