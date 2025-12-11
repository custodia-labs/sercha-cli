package gmail

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
			name:     "gmail://messages/ URI converts to web URL",
			uri:      "gmail://messages/18abc123def456",
			metadata: nil,
			want:     "https://mail.google.com/mail/u/0/#all/18abc123def456",
		},
		{
			name:     "gmail message with long ID",
			uri:      "gmail://messages/18f1234567890abcdef",
			metadata: nil,
			want:     "https://mail.google.com/mail/u/0/#all/18f1234567890abcdef",
		},
		{
			name:     "non-gmail URI returns empty",
			uri:      "https://mail.google.com/mail/u/0/#all/123",
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
			name:     "gmail:// prefix without messages returns empty",
			uri:      "gmail://other/123",
			metadata: nil,
			want:     "",
		},
		{
			name:     "gmail://messages/ prefix only",
			uri:      "gmail://messages/",
			metadata: nil,
			want:     "https://mail.google.com/mail/u/0/#all/",
		},
		{
			name:     "metadata is ignored",
			uri:      "gmail://messages/abc123",
			metadata: map[string]any{"web_link": "should-be-ignored"},
			want:     "https://mail.google.com/mail/u/0/#all/abc123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveWebURL(tt.uri, tt.metadata)
			assert.Equal(t, tt.want, got)
		})
	}
}
