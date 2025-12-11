package drive

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.Equal(t, DefaultContentTypes, cfg.ContentTypes)
	assert.Equal(t, int64(100), cfg.MaxResults)
	assert.Empty(t, cfg.MimeTypeFilter)
	assert.Empty(t, cfg.FolderIDs)
}

func TestParseConfig_Defaults(t *testing.T) {
	source := domain.Source{
		Config: map[string]string{},
	}

	cfg, err := ParseConfig(source)

	require.NoError(t, err)
	assert.Equal(t, DefaultContentTypes, cfg.ContentTypes)
	assert.Equal(t, int64(100), cfg.MaxResults)
	assert.Empty(t, cfg.MimeTypeFilter)
	assert.Empty(t, cfg.FolderIDs)
}

func TestParseConfig_ContentTypes(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected []ContentType
	}{
		{
			name:     "single content type",
			value:    "files",
			expected: []ContentType{ContentFiles},
		},
		{
			name:     "multiple content types",
			value:    "files,docs,sheets",
			expected: []ContentType{ContentFiles, ContentDocs, ContentSheets},
		},
		{
			name:     "content types with spaces",
			value:    "files, docs , sheets",
			expected: []ContentType{ContentFiles, ContentDocs, ContentSheets},
		},
		{
			name:     "invalid content types filtered out",
			value:    "files,invalid,docs",
			expected: []ContentType{ContentFiles, ContentDocs},
		},
		{
			name:     "only invalid content types",
			value:    "invalid,unknown",
			expected: []ContentType{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := domain.Source{
				Config: map[string]string{
					"content_types": tt.value,
				},
			}

			cfg, err := ParseConfig(source)

			require.NoError(t, err)
			assert.Equal(t, tt.expected, cfg.ContentTypes)
		})
	}
}

func TestParseConfig_MimeTypes(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected []string
	}{
		{
			name:     "single mime type",
			value:    "application/pdf",
			expected: []string{"application/pdf"},
		},
		{
			name:     "multiple mime types",
			value:    "application/pdf,text/plain,text/markdown",
			expected: []string{"application/pdf", "text/plain", "text/markdown"},
		},
		{
			name:     "mime types with spaces",
			value:    "application/pdf, text/plain , text/markdown",
			expected: []string{"application/pdf", "text/plain", "text/markdown"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := domain.Source{
				Config: map[string]string{
					"mime_types": tt.value,
				},
			}

			cfg, err := ParseConfig(source)

			require.NoError(t, err)
			assert.Equal(t, tt.expected, cfg.MimeTypeFilter)
		})
	}
}

func TestParseConfig_FolderIDs(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected []string
	}{
		{
			name:     "single folder ID",
			value:    "folder-123",
			expected: []string{"folder-123"},
		},
		{
			name:     "multiple folder IDs",
			value:    "folder-1,folder-2,folder-3",
			expected: []string{"folder-1", "folder-2", "folder-3"},
		},
		{
			name:     "folder IDs with spaces",
			value:    "folder-1, folder-2 , folder-3",
			expected: []string{"folder-1", "folder-2", "folder-3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := domain.Source{
				Config: map[string]string{
					"folder_ids": tt.value,
				},
			}

			cfg, err := ParseConfig(source)

			require.NoError(t, err)
			assert.Equal(t, tt.expected, cfg.FolderIDs)
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

func TestParseConfig_AllOptions(t *testing.T) {
	source := domain.Source{
		Config: map[string]string{
			"content_types": "docs,sheets",
			"mime_types":    "application/pdf",
			"folder_ids":    "folder-1,folder-2",
			"max_results":   "200",
		},
	}

	cfg, err := ParseConfig(source)

	require.NoError(t, err)
	assert.Equal(t, []ContentType{ContentDocs, ContentSheets}, cfg.ContentTypes)
	assert.Equal(t, []string{"application/pdf"}, cfg.MimeTypeFilter)
	assert.Equal(t, []string{"folder-1", "folder-2"}, cfg.FolderIDs)
	assert.Equal(t, int64(200), cfg.MaxResults)
}

func TestConfig_HasContentType(t *testing.T) {
	tests := []struct {
		name         string
		contentTypes []ContentType
		check        ContentType
		expected     bool
	}{
		{
			name:         "has content type",
			contentTypes: []ContentType{ContentFiles, ContentDocs},
			check:        ContentFiles,
			expected:     true,
		},
		{
			name:         "does not have content type",
			contentTypes: []ContentType{ContentFiles, ContentDocs},
			check:        ContentSheets,
			expected:     false,
		},
		{
			name:         "empty content types",
			contentTypes: []ContentType{},
			check:        ContentFiles,
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{ContentTypes: tt.contentTypes}
			result := cfg.HasContentType(tt.check)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsValidContentType(t *testing.T) {
	tests := []struct {
		name     string
		ct       ContentType
		expected bool
	}{
		{
			name:     "files is valid",
			ct:       ContentFiles,
			expected: true,
		},
		{
			name:     "docs is valid",
			ct:       ContentDocs,
			expected: true,
		},
		{
			name:     "sheets is valid",
			ct:       ContentSheets,
			expected: true,
		},
		{
			name:     "unknown is invalid",
			ct:       ContentType("unknown"),
			expected: false,
		},
		{
			name:     "empty is invalid",
			ct:       ContentType(""),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidContentType(tt.ct)
			assert.Equal(t, tt.expected, result)
		})
	}
}
