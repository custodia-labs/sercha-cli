package github

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveWebURL(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		metadata map[string]any
		want     string
	}{
		{
			name:     "github:// blob URI converts to web URL",
			uri:      "github://owner/repo/blob/main/path/to/file.go",
			metadata: nil,
			want:     "https://github.com/owner/repo/blob/main/path/to/file.go",
		},
		{
			name:     "github:// issue URI converts to web URL",
			uri:      "github://owner/repo/issues/123",
			metadata: nil,
			want:     "https://github.com/owner/repo/issues/123",
		},
		{
			name:     "github:// PR URI converts to web URL",
			uri:      "github://owner/repo/pull/456",
			metadata: nil,
			want:     "https://github.com/owner/repo/pull/456",
		},
		{
			name:     "github:// wiki URI converts to web URL",
			uri:      "github://owner/repo/wiki/Page-Name",
			metadata: nil,
			want:     "https://github.com/owner/repo/wiki/Page-Name",
		},
		{
			name:     "github:// root repo URI",
			uri:      "github://owner/repo",
			metadata: nil,
			want:     "https://github.com/owner/repo",
		},
		{
			name:     "non-github URI returns empty",
			uri:      "https://github.com/owner/repo",
			metadata: nil,
			want:     "",
		},
		{
			name:     "file:// URI returns empty",
			uri:      "file:///path/to/file",
			metadata: nil,
			want:     "",
		},
		{
			name:     "empty URI returns empty",
			uri:      "",
			metadata: nil,
			want:     "",
		},
		{
			name:     "metadata is ignored",
			uri:      "github://owner/repo/blob/main/file.go",
			metadata: map[string]any{"web_link": "should-be-ignored"},
			want:     "https://github.com/owner/repo/blob/main/file.go",
		},
		{
			name:     "github:// prefix only",
			uri:      "github://",
			metadata: nil,
			want:     "https://github.com/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveWebURL(tt.uri, tt.metadata)
			assert.Equal(t, tt.want, got)
		})
	}
}
