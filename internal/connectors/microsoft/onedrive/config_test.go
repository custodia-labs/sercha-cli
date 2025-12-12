package onedrive

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.Equal(t, int64(100), cfg.MaxResults)
	assert.Empty(t, cfg.FolderIDs)
	assert.Empty(t, cfg.MimeTypeFilter)
	assert.False(t, cfg.IncludeSharedWithMe)
}

func TestParseConfig_Default(t *testing.T) {
	source := domain.Source{
		Config: make(map[string]string),
	}

	cfg, err := ParseConfig(source)

	require.NoError(t, err)
	assert.Equal(t, int64(100), cfg.MaxResults)
	assert.Empty(t, cfg.FolderIDs)
	assert.Empty(t, cfg.MimeTypeFilter)
	assert.False(t, cfg.IncludeSharedWithMe)
}

func TestParseConfig_WithFolderIDs(t *testing.T) {
	source := domain.Source{
		Config: map[string]string{
			"folder_ids": "folder-1, folder-2, folder-3",
		},
	}

	cfg, err := ParseConfig(source)

	require.NoError(t, err)
	assert.Equal(t, []string{"folder-1", "folder-2", "folder-3"}, cfg.FolderIDs)
}

func TestParseConfig_WithMimeTypes(t *testing.T) {
	source := domain.Source{
		Config: map[string]string{
			"mime_types": "text/plain, application/json",
		},
	}

	cfg, err := ParseConfig(source)

	require.NoError(t, err)
	assert.Equal(t, []string{"text/plain", "application/json"}, cfg.MimeTypeFilter)
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

func TestParseConfig_WithIncludeShared(t *testing.T) {
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := domain.Source{
				Config: map[string]string{
					"include_shared": tt.value,
				},
			}

			cfg, err := ParseConfig(source)

			require.NoError(t, err)
			assert.Equal(t, tt.expected, cfg.IncludeSharedWithMe)
		})
	}
}

func TestParseConfig_AllOptions(t *testing.T) {
	source := domain.Source{
		Config: map[string]string{
			"folder_ids":     "root, documents",
			"mime_types":     "text/plain",
			"max_results":    "25",
			"include_shared": "true",
		},
	}

	cfg, err := ParseConfig(source)

	require.NoError(t, err)
	assert.Equal(t, []string{"root", "documents"}, cfg.FolderIDs)
	assert.Equal(t, []string{"text/plain"}, cfg.MimeTypeFilter)
	assert.Equal(t, int64(25), cfg.MaxResults)
	assert.True(t, cfg.IncludeSharedWithMe)
}
