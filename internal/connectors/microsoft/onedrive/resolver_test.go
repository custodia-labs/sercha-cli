package onedrive

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveWebURL(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		expected string
	}{
		{
			name:     "file URI",
			uri:      "onedrive://files/ABC123DEF",
			expected: "https://onedrive.live.com/",
		},
		{
			name:     "folder URI",
			uri:      "onedrive://folders/FOLDER456",
			expected: "https://onedrive.live.com/",
		},
		{
			name:     "unknown URI scheme",
			uri:      "gdrive://files/123",
			expected: "",
		},
		{
			name:     "empty URI",
			uri:      "",
			expected: "",
		},
		{
			name:     "invalid onedrive URI",
			uri:      "onedrive://other/something",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ResolveWebURL(tt.uri, nil)
			assert.Equal(t, tt.expected, result)
		})
	}
}
