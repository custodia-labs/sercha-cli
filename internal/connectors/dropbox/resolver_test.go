package dropbox

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveWebURL(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		metadata map[string]any
		expected string
	}{
		{
			name: "preview_url takes highest priority",
			uri:  "dropbox://files/id:abc123",
			metadata: map[string]any{
				"preview_url": "https://www.dropbox.com/scl/fi/xyz/preview.pdf?dl=0",
				"path":        "/Documents/report.pdf",
				"file_id":     "id:abc123",
			},
			expected: "https://www.dropbox.com/scl/fi/xyz/preview.pdf?dl=0",
		},
		{
			name: "empty preview_url falls back to path",
			uri:  "dropbox://files/id:abc123",
			metadata: map[string]any{
				"preview_url": "",
				"path":        "/Documents/report.pdf",
			},
			expected: "https://www.dropbox.com/home/Documents%2Freport.pdf",
		},
		{
			name: "with path metadata",
			uri:  "dropbox://files/id:abc123",
			metadata: map[string]any{
				"path": "/Documents/report.pdf",
			},
			expected: "https://www.dropbox.com/home/Documents%2Freport.pdf",
		},
		{
			name: "with path starting with slash",
			uri:  "dropbox://files/id:xyz789",
			metadata: map[string]any{
				"path": "/Work/Projects/design.psd",
			},
			expected: "https://www.dropbox.com/home/Work%2FProjects%2Fdesign.psd",
		},
		{
			name: "with file_id only (no path)",
			uri:  "dropbox://files/id:abc123",
			metadata: map[string]any{
				"file_id": "id:abc123",
			},
			expected: "https://www.dropbox.com/preview/abc123",
		},
		{
			name: "with file_id with prefix",
			uri:  "dropbox://files/id:xyz789",
			metadata: map[string]any{
				"file_id": "id:xyz789",
			},
			expected: "https://www.dropbox.com/preview/xyz789",
		},
		{
			name:     "empty metadata",
			uri:      "dropbox://files/id:test",
			metadata: map[string]any{},
			expected: "https://www.dropbox.com/home",
		},
		{
			name:     "nil metadata",
			uri:      "dropbox://files/id:test",
			metadata: nil,
			expected: "https://www.dropbox.com/home",
		},
		{
			name: "path takes precedence over file_id",
			uri:  "dropbox://files/id:abc123",
			metadata: map[string]any{
				"path":    "/Documents/file.txt",
				"file_id": "id:abc123",
			},
			expected: "https://www.dropbox.com/home/Documents%2Ffile.txt",
		},
		{
			name: "empty path falls back to file_id",
			uri:  "dropbox://files/id:abc123",
			metadata: map[string]any{
				"path":    "",
				"file_id": "id:abc123",
			},
			expected: "https://www.dropbox.com/preview/abc123",
		},
		{
			name: "special characters in path",
			uri:  "dropbox://files/id:special",
			metadata: map[string]any{
				"path": "/My Files/Report (2024).pdf",
			},
			expected: "https://www.dropbox.com/home/My%20Files%2FReport%20%282024%29.pdf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ResolveWebURL(tt.uri, tt.metadata)
			assert.Equal(t, tt.expected, result)
		})
	}
}
