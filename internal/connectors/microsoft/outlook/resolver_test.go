package outlook

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
			name:     "web_link from metadata",
			uri:      "outlook://messages/AAMkAGI2ABC123",
			metadata: map[string]any{"web_link": "https://outlook.office.com/mail/id/AAMkAGI2ABC123"},
			expected: "https://outlook.office.com/mail/id/AAMkAGI2ABC123",
		},
		{
			name:     "message URI without metadata",
			uri:      "outlook://messages/AAMkAGI2ABC123",
			metadata: nil,
			expected: "https://outlook.office.com/mail/",
		},
		{
			name:     "message URI with empty web_link",
			uri:      "outlook://messages/AAMkAGI2ABC123",
			metadata: map[string]any{"web_link": ""},
			expected: "https://outlook.office.com/mail/",
		},
		{
			name:     "conversation URI",
			uri:      "outlook://conversations/AAQkAGI2CONV123",
			metadata: nil,
			expected: "https://outlook.office.com/mail/",
		},
		{
			name:     "unknown URI scheme",
			uri:      "other://something",
			metadata: nil,
			expected: "",
		},
		{
			name:     "empty URI",
			uri:      "",
			metadata: nil,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ResolveWebURL(tt.uri, tt.metadata)
			assert.Equal(t, tt.expected, result)
		})
	}
}
