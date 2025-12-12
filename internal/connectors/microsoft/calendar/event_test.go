package calendar

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEventToRawDocument(t *testing.T) {
	event := &Event{
		ID:      "event-123",
		Subject: "Team Meeting",
		Body: &EventBody{
			ContentType: "text",
			Content:     "Discuss project updates",
		},
		Start: &DateTimeZone{
			DateTime: "2024-01-15T10:00:00",
			TimeZone: "UTC",
		},
		End: &DateTimeZone{
			DateTime: "2024-01-15T11:00:00",
			TimeZone: "UTC",
		},
		Location: &Location{
			DisplayName: "Conference Room A",
		},
		Organiser: &EmailAddress{
			EmailAddress: struct {
				Name    string `json:"name"`
				Address string `json:"address"`
			}{
				Name:    "John Doe",
				Address: "john@example.com",
			},
		},
		Attendees: []Attendee{
			{
				EmailAddress: struct {
					Name    string `json:"name"`
					Address string `json:"address"`
				}{
					Name:    "Jane Smith",
					Address: "jane@example.com",
				},
			},
		},
		WebLink:              "https://outlook.office.com/calendar/event/123",
		IsCancelled:          false,
		IsAllDay:             false,
		Importance:           "normal",
		Sensitivity:          "normal",
		ShowAs:               "busy",
		CreatedDateTime:      "2024-01-10T08:00:00Z",
		LastModifiedDateTime: "2024-01-12T09:00:00Z",
	}

	doc := EventToRawDocument(event, "cal-abc", "source-xyz")

	assert.Equal(t, "source-xyz", doc.SourceID)
	assert.Equal(t, "mscal://cal-abc/events/event-123", doc.URI)
	assert.Equal(t, "text/calendar", doc.MIMEType)

	// Check content includes relevant fields
	content := string(doc.Content)
	assert.Contains(t, content, "Team Meeting")
	assert.Contains(t, content, "Discuss project updates")
	assert.Contains(t, content, "Conference Room A")
	assert.Contains(t, content, "Jane Smith")

	// Check metadata
	assert.Equal(t, "event-123", doc.Metadata["event_id"])
	assert.Equal(t, "cal-abc", doc.Metadata["calendar_id"])
	assert.Equal(t, "Team Meeting", doc.Metadata["title"])
	assert.Equal(t, "2024-01-15T10:00:00", doc.Metadata["start_time"])
	assert.Equal(t, "2024-01-15T11:00:00", doc.Metadata["end_time"])
	assert.Equal(t, "Conference Room A", doc.Metadata["location"])
	assert.Equal(t, "john@example.com", doc.Metadata["organiser"])
	assert.Equal(t, "John Doe", doc.Metadata["organiser_name"])
	assert.Equal(t, false, doc.Metadata["is_all_day"])
	assert.Equal(t, false, doc.Metadata["is_cancelled"])
}

func TestEventToRawDocument_WithSeriesMaster(t *testing.T) {
	event := &Event{
		ID:             "event-instance-456",
		Subject:        "Recurring Meeting",
		SeriesMasterID: "event-series-123",
	}

	doc := EventToRawDocument(event, "cal-abc", "source-xyz")

	assert.NotNil(t, doc.ParentURI)
	assert.Equal(t, "mscal://cal-abc/events/event-series-123", *doc.ParentURI)
	assert.Equal(t, "event-series-123", doc.Metadata["series_master_id"])
}

func TestEventToRawDocument_NoParent(t *testing.T) {
	event := &Event{
		ID:      "event-123",
		Subject: "Single Event",
	}

	doc := EventToRawDocument(event, "cal-abc", "source-xyz")

	assert.Nil(t, doc.ParentURI)
}

func TestBuildEventContent(t *testing.T) {
	tests := []struct {
		name     string
		event    *Event
		contains []string
	}{
		{
			name: "full event",
			event: &Event{
				Subject: "Meeting",
				Body: &EventBody{
					ContentType: "text",
					Content:     "Discussion topics",
				},
				Location: &Location{DisplayName: "Room 101"},
				Attendees: []Attendee{
					{
						EmailAddress: struct {
							Name    string `json:"name"`
							Address string `json:"address"`
						}{
							Name: "Alice",
						},
					},
				},
			},
			contains: []string{"Meeting", "Discussion topics", "Room 101", "Alice"},
		},
		{
			name: "event with HTML body",
			event: &Event{
				Subject: "Newsletter",
				Body: &EventBody{
					ContentType: "html",
					Content:     "<p>Hello <b>World</b></p>",
				},
			},
			contains: []string{"Newsletter", "Hello World"},
		},
		{
			name: "minimal event",
			event: &Event{
				Subject: "Quick Sync",
			},
			contains: []string{"Quick Sync"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := buildEventContent(tt.event)

			for _, s := range tt.contains {
				assert.Contains(t, content, s)
			}
		})
	}
}

func TestStripHTMLTags(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple HTML",
			input:    "<p>Hello</p>",
			expected: "Hello",
		},
		{
			name:     "nested tags",
			input:    "<div><p>Hello <b>World</b></p></div>",
			expected: "Hello World",
		},
		{
			name:     "no HTML",
			input:    "Plain text",
			expected: "Plain text",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripHTMLTags(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatAttendees(t *testing.T) {
	tests := []struct {
		name      string
		attendees []Attendee
		expected  string
	}{
		{
			name:      "empty attendees",
			attendees: []Attendee{},
			expected:  "",
		},
		{
			name: "single attendee with name",
			attendees: []Attendee{
				{
					EmailAddress: struct {
						Name    string `json:"name"`
						Address string `json:"address"`
					}{
						Name:    "Alice",
						Address: "alice@example.com",
					},
				},
			},
			expected: "Attendees: Alice",
		},
		{
			name: "attendee without name",
			attendees: []Attendee{
				{
					EmailAddress: struct {
						Name    string `json:"name"`
						Address string `json:"address"`
					}{
						Address: "bob@example.com",
					},
				},
			},
			expected: "Attendees: bob@example.com",
		},
		{
			name: "multiple attendees",
			attendees: []Attendee{
				{
					EmailAddress: struct {
						Name    string `json:"name"`
						Address string `json:"address"`
					}{Name: "Alice"},
				},
				{
					EmailAddress: struct {
						Name    string `json:"name"`
						Address string `json:"address"`
					}{Address: "bob@example.com"},
				},
			},
			expected: "Attendees: Alice, bob@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatAttendees(tt.attendees)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractEventTimes(t *testing.T) {
	event := &Event{
		Start: &DateTimeZone{DateTime: "2024-01-15T10:00:00"},
		End:   &DateTimeZone{DateTime: "2024-01-15T11:00:00"},
	}

	start, end := extractEventTimes(event)

	assert.Equal(t, "2024-01-15T10:00:00", start)
	assert.Equal(t, "2024-01-15T11:00:00", end)
}

func TestExtractEventTimes_NilTimes(t *testing.T) {
	event := &Event{}

	start, end := extractEventTimes(event)

	assert.Empty(t, start)
	assert.Empty(t, end)
}

func TestBuildSeriesMasterURI(t *testing.T) {
	tests := []struct {
		name      string
		event     *Event
		expectNil bool
		expected  string
	}{
		{
			name: "has series master",
			event: &Event{
				ID:             "instance-123",
				SeriesMasterID: "master-456",
			},
			expectNil: false,
			expected:  "mscal://cal-1/events/master-456",
		},
		{
			name: "no series master",
			event: &Event{
				ID: "event-123",
			},
			expectNil: true,
		},
		{
			name: "series master same as ID",
			event: &Event{
				ID:             "event-123",
				SeriesMasterID: "event-123",
			},
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildSeriesMasterURI(tt.event, "cal-1")

			if tt.expectNil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				assert.Equal(t, tt.expected, *result)
			}
		})
	}
}

func TestShouldSyncEvent(t *testing.T) {
	tests := []struct {
		name     string
		event    *Event
		expected bool
	}{
		{
			name:     "nil event",
			event:    nil,
			expected: false,
		},
		{
			name:     "empty ID",
			event:    &Event{ID: ""},
			expected: false,
		},
		{
			name:     "valid event",
			event:    &Event{ID: "event-123"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldSyncEvent(tt.event)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsEventRemoved(t *testing.T) {
	tests := []struct {
		name     string
		event    *EventWithRemoved
		expected bool
	}{
		{
			name: "not removed",
			event: &EventWithRemoved{
				Event: Event{ID: "event-123"},
			},
			expected: false,
		},
		{
			name: "removed via @removed",
			event: &EventWithRemoved{
				Event: Event{ID: "event-123"},
				Removed: &struct {
					Reason string `json:"reason"`
				}{Reason: "deleted"},
			},
			expected: true,
		},
		{
			name: "cancelled event",
			event: &EventWithRemoved{
				Event: Event{ID: "event-123", IsCancelled: true},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsEventRemoved(tt.event)
			assert.Equal(t, tt.expected, result)
		})
	}
}
