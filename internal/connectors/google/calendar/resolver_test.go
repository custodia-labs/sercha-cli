package calendar

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
			name: "metadata html_link returns web URL",
			uri:  "gcal://events/abc123",
			metadata: map[string]any{
				"html_link": "https://www.google.com/calendar/event?eid=YWJjMTIzX2V2ZW50QGdtYWlsLmNvbQ",
			},
			want: "https://www.google.com/calendar/event?eid=YWJjMTIzX2V2ZW50QGdtYWlsLmNvbQ",
		},
		{
			name: "html_link with complex encoded event ID",
			uri:  "gcal://events/xyz789",
			metadata: map[string]any{
				"html_link": "https://www.google.com/calendar/event?eid=eHl6Nzg5X2V2ZW50X2lkQGdtYWlsLmNvbQ",
			},
			want: "https://www.google.com/calendar/event?eid=eHl6Nzg5X2V2ZW50X2lkQGdtYWlsLmNvbQ",
		},
		{
			name:     "nil metadata returns empty",
			uri:      "gcal://events/abc123",
			metadata: nil,
			want:     "",
		},
		{
			name:     "empty metadata returns empty",
			uri:      "gcal://events/abc123",
			metadata: map[string]any{},
			want:     "",
		},
		{
			name:     "empty html_link returns empty",
			uri:      "gcal://events/abc123",
			metadata: map[string]any{"html_link": ""},
			want:     "",
		},
		{
			name:     "html_link not a string returns empty",
			uri:      "gcal://events/abc123",
			metadata: map[string]any{"html_link": 12345},
			want:     "",
		},
		{
			name: "uri is ignored when html_link present",
			uri:  "",
			metadata: map[string]any{
				"html_link": "https://www.google.com/calendar/event?eid=test",
			},
			want: "https://www.google.com/calendar/event?eid=test",
		},
		{
			name:     "other metadata keys are ignored",
			uri:      "gcal://events/abc123",
			metadata: map[string]any{"web_link": "ignored", "other_key": "also-ignored"},
			want:     "",
		},
		{
			name:     "empty string uri with nil metadata",
			uri:      "",
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
