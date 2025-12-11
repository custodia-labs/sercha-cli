package calendar

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/api/calendar/v3"
)

func TestEventToRawDocument(t *testing.T) {
	event := &calendar.Event{
		Id:               "event-123",
		Summary:          "Team Meeting",
		Description:      "Weekly sync to discuss project progress",
		Location:         "Conference Room A",
		Status:           "confirmed",
		HtmlLink:         "https://calendar.google.com/event?eid=abc123",
		RecurringEventId: "",
		Created:          "2024-01-15T10:00:00Z",
		Updated:          "2024-01-15T10:30:00Z",
		Start: &calendar.EventDateTime{
			DateTime: "2024-01-20T14:00:00-05:00",
		},
		End: &calendar.EventDateTime{
			DateTime: "2024-01-20T15:00:00-05:00",
		},
		Organizer: &calendar.EventOrganizer{
			Email: "organizer@example.com",
		},
		Attendees: []*calendar.EventAttendee{
			{DisplayName: "Alice", Email: "alice@example.com"},
			{DisplayName: "Bob", Email: "bob@example.com"},
		},
	}

	doc := EventToRawDocument(event, "primary", "source-abc")

	assert.Equal(t, "source-abc", doc.SourceID)
	assert.Equal(t, "gcal://primary/events/event-123", doc.URI)
	assert.Equal(t, "text/calendar", doc.MIMEType)

	// Check content includes relevant fields
	content := string(doc.Content)
	assert.Contains(t, content, "Team Meeting")
	assert.Contains(t, content, "Weekly sync to discuss project progress")
	assert.Contains(t, content, "Conference Room A")
	assert.Contains(t, content, "Attendees:")
	assert.Contains(t, content, "Alice")
	assert.Contains(t, content, "Bob")

	// Check metadata
	assert.Equal(t, "event-123", doc.Metadata["event_id"])
	assert.Equal(t, "primary", doc.Metadata["calendar_id"])
	assert.Equal(t, "Team Meeting", doc.Metadata["title"])
	assert.Equal(t, "Weekly sync to discuss project progress", doc.Metadata["description"])
	assert.Equal(t, "Conference Room A", doc.Metadata["location"])
	assert.Equal(t, "2024-01-20T14:00:00-05:00", doc.Metadata["start_time"])
	assert.Equal(t, "2024-01-20T15:00:00-05:00", doc.Metadata["end_time"])
	assert.Equal(t, "confirmed", doc.Metadata["status"])
	assert.Equal(t, "https://calendar.google.com/event?eid=abc123", doc.Metadata["html_link"])
	assert.Equal(t, "organizer@example.com", doc.Metadata["organiser"])
}

func TestEventToRawDocument_AllDayEvent(t *testing.T) {
	event := &calendar.Event{
		Id:      "event-allday",
		Summary: "Company Holiday",
		Start: &calendar.EventDateTime{
			Date: "2024-12-25",
		},
		End: &calendar.EventDateTime{
			Date: "2024-12-26",
		},
	}

	doc := EventToRawDocument(event, "primary", "source-abc")

	assert.Equal(t, "2024-12-25", doc.Metadata["start_time"])
	assert.Equal(t, "2024-12-26", doc.Metadata["end_time"])
}

func TestEventToRawDocument_RecurringInstance(t *testing.T) {
	event := &calendar.Event{
		Id:               "event-123_20240120T140000Z",
		Summary:          "Weekly Standup",
		RecurringEventId: "event-123", // Different from event ID
		Start: &calendar.EventDateTime{
			DateTime: "2024-01-20T09:00:00Z",
		},
		End: &calendar.EventDateTime{
			DateTime: "2024-01-20T09:15:00Z",
		},
	}

	doc := EventToRawDocument(event, "work@example.com", "source-abc")

	assert.NotNil(t, doc.ParentURI)
	assert.Equal(t, "gcal://work@example.com/events/event-123", *doc.ParentURI)
	assert.Equal(t, "event-123", doc.Metadata["recurring_event_id"])
}

func TestEventToRawDocument_NoRecurringParent(t *testing.T) {
	event := &calendar.Event{
		Id:               "event-123",
		Summary:          "One-time Event",
		RecurringEventId: "", // Not a recurring instance
		Start: &calendar.EventDateTime{
			DateTime: "2024-01-20T14:00:00Z",
		},
		End: &calendar.EventDateTime{
			DateTime: "2024-01-20T15:00:00Z",
		},
	}

	doc := EventToRawDocument(event, "primary", "source-abc")

	assert.Nil(t, doc.ParentURI)
}

func TestBuildEventContent(t *testing.T) {
	tests := []struct {
		name     string
		event    *calendar.Event
		contains []string
		excludes []string
	}{
		{
			name: "full event",
			event: &calendar.Event{
				Summary:     "Meeting",
				Description: "Important discussion",
				Location:    "Room 101",
				Attendees: []*calendar.EventAttendee{
					{DisplayName: "Alice"},
				},
			},
			contains: []string{"Meeting", "Important discussion", "Location: Room 101", "Attendees:", "Alice"},
		},
		{
			name: "event without location",
			event: &calendar.Event{
				Summary:     "Meeting",
				Description: "Discussion",
			},
			contains: []string{"Meeting", "Discussion"},
			excludes: []string{"Location:"},
		},
		{
			name: "event without description",
			event: &calendar.Event{
				Summary:  "Quick Call",
				Location: "Phone",
			},
			contains: []string{"Quick Call", "Location: Phone"},
		},
		{
			name: "event with email-only attendees",
			event: &calendar.Event{
				Summary: "Meeting",
				Attendees: []*calendar.EventAttendee{
					{Email: "user@example.com"},
				},
			},
			contains: []string{"Meeting", "user@example.com"},
		},
		{
			name: "minimal event",
			event: &calendar.Event{
				Summary: "Simple Event",
			},
			contains: []string{"Simple Event"},
			excludes: []string{"Location:", "Attendees:"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := buildEventContent(tt.event)

			for _, s := range tt.contains {
				assert.Contains(t, content, s)
			}
			for _, s := range tt.excludes {
				assert.NotContains(t, content, s)
			}
		})
	}
}

func TestFormatAttendees(t *testing.T) {
	tests := []struct {
		name      string
		attendees []*calendar.EventAttendee
		expected  string
	}{
		{
			name:      "nil attendees",
			attendees: nil,
			expected:  "",
		},
		{
			name:      "empty attendees",
			attendees: []*calendar.EventAttendee{},
			expected:  "",
		},
		{
			name: "single attendee with display name",
			attendees: []*calendar.EventAttendee{
				{DisplayName: "Alice", Email: "alice@example.com"},
			},
			expected: "Attendees: Alice",
		},
		{
			name: "single attendee with email only",
			attendees: []*calendar.EventAttendee{
				{Email: "alice@example.com"},
			},
			expected: "Attendees: alice@example.com",
		},
		{
			name: "multiple attendees",
			attendees: []*calendar.EventAttendee{
				{DisplayName: "Alice"},
				{DisplayName: "Bob"},
				{Email: "charlie@example.com"},
			},
			expected: "Attendees: Alice, Bob, charlie@example.com",
		},
		{
			name: "attendee with no name or email",
			attendees: []*calendar.EventAttendee{
				{},
			},
			expected: "",
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
	tests := []struct {
		name          string
		event         *calendar.Event
		expectedStart string
		expectedEnd   string
	}{
		{
			name: "datetime event",
			event: &calendar.Event{
				Start: &calendar.EventDateTime{DateTime: "2024-01-20T14:00:00Z"},
				End:   &calendar.EventDateTime{DateTime: "2024-01-20T15:00:00Z"},
			},
			expectedStart: "2024-01-20T14:00:00Z",
			expectedEnd:   "2024-01-20T15:00:00Z",
		},
		{
			name: "all-day event",
			event: &calendar.Event{
				Start: &calendar.EventDateTime{Date: "2024-01-20"},
				End:   &calendar.EventDateTime{Date: "2024-01-21"},
			},
			expectedStart: "2024-01-20",
			expectedEnd:   "2024-01-21",
		},
		{
			name: "nil start and end",
			event: &calendar.Event{
				Start: nil,
				End:   nil,
			},
			expectedStart: "",
			expectedEnd:   "",
		},
		{
			name: "datetime takes precedence over date",
			event: &calendar.Event{
				Start: &calendar.EventDateTime{
					DateTime: "2024-01-20T14:00:00Z",
					Date:     "2024-01-20",
				},
				End: &calendar.EventDateTime{
					DateTime: "2024-01-20T15:00:00Z",
					Date:     "2024-01-20",
				},
			},
			expectedStart: "2024-01-20T14:00:00Z",
			expectedEnd:   "2024-01-20T15:00:00Z",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end := extractEventTimes(tt.event)
			assert.Equal(t, tt.expectedStart, start)
			assert.Equal(t, tt.expectedEnd, end)
		})
	}
}

func TestBuildRecurringParentURI(t *testing.T) {
	tests := []struct {
		name       string
		event      *calendar.Event
		calendarID string
		expectNil  bool
		expected   string
	}{
		{
			name: "recurring instance",
			event: &calendar.Event{
				Id:               "event-123_20240120",
				RecurringEventId: "event-123",
			},
			calendarID: "primary",
			expectNil:  false,
			expected:   "gcal://primary/events/event-123",
		},
		{
			name: "non-recurring event",
			event: &calendar.Event{
				Id:               "event-456",
				RecurringEventId: "",
			},
			calendarID: "primary",
			expectNil:  true,
		},
		{
			name: "recurring event ID same as event ID",
			event: &calendar.Event{
				Id:               "event-123",
				RecurringEventId: "event-123",
			},
			calendarID: "primary",
			expectNil:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildRecurringParentURI(tt.event, tt.calendarID)

			if tt.expectNil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				assert.Equal(t, tt.expected, *result)
			}
		})
	}
}

func TestGetOrganiserEmail(t *testing.T) {
	tests := []struct {
		name     string
		event    *calendar.Event
		expected string
	}{
		{
			name: "has organizer",
			event: &calendar.Event{
				Organizer: &calendar.EventOrganizer{
					Email: "organizer@example.com",
				},
			},
			expected: "organizer@example.com",
		},
		{
			name: "no organizer",
			event: &calendar.Event{
				Organizer: nil,
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getOrganiserEmail(tt.event)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestShouldSyncEvent(t *testing.T) {
	tests := []struct {
		name     string
		event    *calendar.Event
		expected bool
	}{
		{
			name:     "valid event",
			event:    &calendar.Event{Id: "event-123"},
			expected: true,
		},
		{
			name:     "nil event",
			event:    nil,
			expected: false,
		},
		{
			name:     "empty event ID",
			event:    &calendar.Event{Id: ""},
			expected: false,
		},
		{
			name: "cancelled event still syncable",
			event: &calendar.Event{
				Id:     "event-123",
				Status: "cancelled",
			},
			expected: true, // Cancelled events are filtered elsewhere
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldSyncEvent(tt.event)
			assert.Equal(t, tt.expected, result)
		})
	}
}
