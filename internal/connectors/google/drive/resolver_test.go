package drive

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
			name:     "metadata web_link takes precedence",
			uri:      "gdrive://files/1abc123",
			metadata: map[string]any{"web_link": "https://docs.google.com/document/d/1abc123/edit"},
			want:     "https://docs.google.com/document/d/1abc123/edit",
		},
		{
			name:     "metadata web_link for spreadsheet",
			uri:      "gdrive://files/2xyz789",
			metadata: map[string]any{"web_link": "https://docs.google.com/spreadsheets/d/2xyz789/edit"},
			want:     "https://docs.google.com/spreadsheets/d/2xyz789/edit",
		},
		{
			name:     "fallback to URI conversion when no metadata",
			uri:      "gdrive://files/1abc123def456",
			metadata: nil,
			want:     "https://drive.google.com/file/d/1abc123def456/view",
		},
		{
			name:     "fallback to URI conversion when metadata empty",
			uri:      "gdrive://files/xyz789",
			metadata: map[string]any{},
			want:     "https://drive.google.com/file/d/xyz789/view",
		},
		{
			name:     "fallback when web_link is empty string",
			uri:      "gdrive://files/abc",
			metadata: map[string]any{"web_link": ""},
			want:     "https://drive.google.com/file/d/abc/view",
		},
		{
			name:     "fallback when web_link is not a string",
			uri:      "gdrive://files/def",
			metadata: map[string]any{"web_link": 12345},
			want:     "https://drive.google.com/file/d/def/view",
		},
		{
			name:     "non-gdrive URI with metadata returns metadata",
			uri:      "https://something-else.com",
			metadata: map[string]any{"web_link": "https://docs.google.com/document/d/1abc/edit"},
			want:     "https://docs.google.com/document/d/1abc/edit",
		},
		{
			name:     "non-gdrive URI without metadata returns empty",
			uri:      "https://something-else.com",
			metadata: nil,
			want:     "",
		},
		{
			name:     "empty URI without metadata returns empty",
			uri:      "",
			metadata: nil,
			want:     "",
		},
		{
			name:     "gdrive://files/ prefix only",
			uri:      "gdrive://files/",
			metadata: nil,
			want:     "https://drive.google.com/file/d//view",
		},
		{
			name:     "other metadata keys are ignored",
			uri:      "gdrive://files/test123",
			metadata: map[string]any{"other_key": "value", "html_link": "ignored"},
			want:     "https://drive.google.com/file/d/test123/view",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveWebURL(tt.uri, tt.metadata)
			assert.Equal(t, tt.want, got)
		})
	}
}
