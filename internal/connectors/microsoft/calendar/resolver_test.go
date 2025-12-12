package calendar

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
			name:     "event URI",
			uri:      "mscal://cal-123/events/event-456",
			expected: "https://outlook.office.com/calendar/",
		},
		{
			name:     "unknown URI scheme",
			uri:      "gcal://cal-123/events/event-456",
			expected: "",
		},
		{
			name:     "empty URI",
			uri:      "",
			expected: "",
		},
		{
			name:     "other scheme",
			uri:      "outlook://messages/123",
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
