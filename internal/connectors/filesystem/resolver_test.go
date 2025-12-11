package filesystem

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
			name:     "file:// URI is converted to local path",
			uri:      "file:///Users/test/documents/file.txt",
			metadata: nil,
			want:     "/Users/test/documents/file.txt",
		},
		{
			name:     "file:// URI with spaces",
			uri:      "file:///Users/test/my documents/file.txt",
			metadata: nil,
			want:     "/Users/test/my documents/file.txt",
		},
		{
			name:     "bare path passes through unchanged",
			uri:      "/Users/test/documents/file.txt",
			metadata: nil,
			want:     "/Users/test/documents/file.txt",
		},
		{
			name:     "relative path passes through unchanged",
			uri:      "relative/path/to/file.txt",
			metadata: nil,
			want:     "relative/path/to/file.txt",
		},
		{
			name:     "empty string passes through",
			uri:      "",
			metadata: nil,
			want:     "",
		},
		{
			name:     "metadata is ignored",
			uri:      "file:///test/file.txt",
			metadata: map[string]any{"some_key": "some_value"},
			want:     "/test/file.txt",
		},
		{
			name:     "windows-style path passes through",
			uri:      "C:\\Users\\test\\file.txt",
			metadata: nil,
			want:     "C:\\Users\\test\\file.txt",
		},
		{
			name:     "file:// prefix only",
			uri:      "file://",
			metadata: nil,
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveWebURL(tt.uri, tt.metadata)
			assert.Equal(t, tt.want, got)
		})
	}
}
