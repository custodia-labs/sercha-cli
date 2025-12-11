package ics

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
)

func TestNew(t *testing.T) {
	normaliser := New()
	require.NotNil(t, normaliser)
	assert.IsType(t, &Normaliser{}, normaliser)
}

func TestSupportedMIMETypes(t *testing.T) {
	normaliser := New()
	mimeTypes := normaliser.SupportedMIMETypes()

	require.NotEmpty(t, mimeTypes)
	assert.Contains(t, mimeTypes, "text/calendar")
	assert.Len(t, mimeTypes, 1)
}

func TestSupportedConnectorTypes(t *testing.T) {
	normaliser := New()
	assert.Nil(t, normaliser.SupportedConnectorTypes())
}

func TestPriority(t *testing.T) {
	normaliser := New()
	assert.Equal(t, 50, normaliser.Priority())
}

func TestNormalise_NilDocument(t *testing.T) {
	normaliser := New()
	ctx := context.Background()

	result, err := normaliser.Normalise(ctx, nil)
	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrInvalidInput)
	assert.Nil(t, result)
}

func TestNormalise_SimpleEvent(t *testing.T) {
	normaliser := New()
	ctx := context.Background()

	icsContent := `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
SUMMARY:Team Meeting
DESCRIPTION:Weekly sync with the team
LOCATION:Conference Room A
DTSTART:20240115T100000Z
DTEND:20240115T110000Z
END:VEVENT
END:VCALENDAR`

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "/path/to/calendar.ics",
		MIMEType: "text/calendar",
		Content:  []byte(icsContent),
	}

	result, err := normaliser.Normalise(ctx, raw)
	require.NoError(t, err)
	require.NotNil(t, result)

	doc := result.Document
	assert.NotEmpty(t, doc.ID)
	assert.Equal(t, "test-source", doc.SourceID)
	assert.Equal(t, "/path/to/calendar.ics", doc.URI)
	assert.Equal(t, "Team Meeting", doc.Title)
	assert.Contains(t, doc.Content, "Team Meeting")
	assert.Contains(t, doc.Content, "Weekly sync with the team")
	assert.Contains(t, doc.Content, "Conference Room A")
	assert.Equal(t, "text/calendar", doc.Metadata["mime_type"])
	assert.Equal(t, "ics", doc.Metadata["format"])
}

func TestNormalise_MultipleEvents(t *testing.T) {
	normaliser := New()
	ctx := context.Background()

	icsContent := `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
SUMMARY:Morning Standup
DTSTART:20240115T090000Z
END:VEVENT
BEGIN:VEVENT
SUMMARY:Lunch Meeting
DTSTART:20240115T120000Z
END:VEVENT
END:VCALENDAR`

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "/path/to/calendar.ics",
		MIMEType: "text/calendar",
		Content:  []byte(icsContent),
	}

	result, err := normaliser.Normalise(ctx, raw)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Title should indicate multiple events
	assert.Equal(t, "Morning Standup (and more)", result.Document.Title)
	assert.Contains(t, result.Document.Content, "Morning Standup")
	assert.Contains(t, result.Document.Content, "Lunch Meeting")
}

func TestNormalise_WithOrganizer(t *testing.T) {
	normaliser := New()
	ctx := context.Background()

	icsContent := `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
SUMMARY:Project Review
ORGANIZER:mailto:boss@example.com
ATTENDEE:mailto:dev1@example.com
ATTENDEE:mailto:dev2@example.com
DTSTART:20240115T140000Z
END:VEVENT
END:VCALENDAR`

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "/path/to/calendar.ics",
		MIMEType: "text/calendar",
		Content:  []byte(icsContent),
	}

	result, err := normaliser.Normalise(ctx, raw)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Contains(t, result.Document.Content, "boss@example.com")
	assert.Contains(t, result.Document.Content, "dev1@example.com")
	assert.Contains(t, result.Document.Content, "dev2@example.com")
}

func TestNormalise_DateOnly(t *testing.T) {
	normaliser := New()
	ctx := context.Background()

	icsContent := `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
SUMMARY:All Day Event
DTSTART;VALUE=DATE:20240115
DTEND;VALUE=DATE:20240116
END:VEVENT
END:VCALENDAR`

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "/path/to/calendar.ics",
		MIMEType: "text/calendar",
		Content:  []byte(icsContent),
	}

	result, err := normaliser.Normalise(ctx, raw)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Should format dates nicely
	assert.Contains(t, result.Document.Content, "January 15, 2024")
}

func TestNormalise_EscapedCharacters(t *testing.T) {
	normaliser := New()
	ctx := context.Background()

	icsContent := `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
SUMMARY:Meeting with John\, Jane
DESCRIPTION:Discussion about:\n- Topic 1\n- Topic 2
DTSTART:20240115T100000Z
END:VEVENT
END:VCALENDAR`

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "/path/to/calendar.ics",
		MIMEType: "text/calendar",
		Content:  []byte(icsContent),
	}

	result, err := normaliser.Normalise(ctx, raw)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Escaped characters should be decoded
	assert.Contains(t, result.Document.Content, "Meeting with John, Jane")
	assert.Contains(t, result.Document.Content, "Topic 1")
}

func TestNormalise_LineFolding(t *testing.T) {
	normaliser := New()
	ctx := context.Background()

	// Long lines are folded with leading space
	icsContent := `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
SUMMARY:This is a very long summary that would normally be folded
 across multiple lines in the ICS format
DTSTART:20240115T100000Z
END:VEVENT
END:VCALENDAR`

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "/path/to/calendar.ics",
		MIMEType: "text/calendar",
		Content:  []byte(icsContent),
	}

	result, err := normaliser.Normalise(ctx, raw)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Folded lines should be joined
	assert.Contains(t, result.Document.Title, "This is a very long summary")
}

func TestNormalise_CalendarName(t *testing.T) {
	normaliser := New()
	ctx := context.Background()

	icsContent := `BEGIN:VCALENDAR
VERSION:2.0
X-WR-CALNAME:Work Calendar
BEGIN:VEVENT
DTSTART:20240115T100000Z
END:VEVENT
END:VCALENDAR`

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "/path/to/calendar.ics",
		MIMEType: "text/calendar",
		Content:  []byte(icsContent),
	}

	result, err := normaliser.Normalise(ctx, raw)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Should use calendar name when event has no summary
	assert.Equal(t, "Work Calendar", result.Document.Title)
}

func TestNormalise_NoEvents(t *testing.T) {
	normaliser := New()
	ctx := context.Background()

	icsContent := `BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Test//Test//EN
END:VCALENDAR`

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "/path/to/empty_calendar.ics",
		MIMEType: "text/calendar",
		Content:  []byte(icsContent),
	}

	result, err := normaliser.Normalise(ctx, raw)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Should fall back to filename
	assert.Equal(t, "empty calendar", result.Document.Title)
}

func TestNormalise_WithMetadata(t *testing.T) {
	normaliser := New()
	ctx := context.Background()

	icsContent := `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
SUMMARY:Test Event
DTSTART:20240115T100000Z
END:VEVENT
END:VCALENDAR`

	raw := &domain.RawDocument{
		SourceID: "test-source",
		URI:      "/path/to/calendar.ics",
		MIMEType: "text/calendar",
		Content:  []byte(icsContent),
		Metadata: map[string]any{
			"source": "google-calendar",
		},
	}

	result, err := normaliser.Normalise(ctx, raw)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Should preserve original metadata
	assert.Equal(t, "google-calendar", result.Document.Metadata["source"])
	// And add format metadata
	assert.Equal(t, "ics", result.Document.Metadata["format"])
}

func TestDecodeValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "newline lowercase",
			input:    "Line 1\\nLine 2",
			expected: "Line 1\nLine 2",
		},
		{
			name:     "newline uppercase",
			input:    "Line 1\\NLine 2",
			expected: "Line 1\nLine 2",
		},
		{
			name:     "escaped comma",
			input:    "Item 1\\, Item 2",
			expected: "Item 1, Item 2",
		},
		{
			name:     "escaped semicolon",
			input:    "Part 1\\; Part 2",
			expected: "Part 1; Part 2",
		},
		{
			name:     "escaped backslash",
			input:    "Path\\\\file",
			expected: "Path\\file",
		},
		{
			name:     "no escapes",
			input:    "Plain text",
			expected: "Plain text",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := decodeValue(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestFormatDateTime(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "date only",
			input:    "20240115",
			expected: "January 15, 2024",
		},
		{
			name:     "datetime with Z",
			input:    "20240115T100000Z",
			expected: "January 15, 2024 at 10:00 AM",
		},
		{
			name:     "datetime without Z",
			input:    "20240115T143000",
			expected: "January 15, 2024 at 2:30 PM",
		},
		{
			name:     "invalid format",
			input:    "invalid",
			expected: "invalid",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := formatDateTime(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestExtractEmail(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "mailto prefix",
			input:    "mailto:user@example.com",
			expected: "user@example.com",
		},
		{
			name:     "MAILTO prefix",
			input:    "MAILTO:user@example.com",
			expected: "user@example.com",
		},
		{
			name:     "plain email",
			input:    "user@example.com",
			expected: "user@example.com",
		},
		{
			name:     "no email",
			input:    "not an email",
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := extractEmail(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestExtractTitleFromURI(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		expected string
	}{
		{
			name:     "simple filename",
			uri:      "/path/to/calendar.ics",
			expected: "calendar",
		},
		{
			name:     "with underscores",
			uri:      "/path/to/my_calendar_file.ics",
			expected: "my calendar file",
		},
		{
			name:     "with dashes",
			uri:      "/path/to/my-calendar-file.ics",
			expected: "my calendar file",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := extractTitleFromURI(tc.uri)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestCopyMetadata(t *testing.T) {
	tests := []struct {
		name string
		src  map[string]any
	}{
		{
			name: "nil map",
			src:  nil,
		},
		{
			name: "empty map",
			src:  map[string]any{},
		},
		{
			name: "with values",
			src: map[string]any{
				"key1": "value1",
				"key2": 42,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := copyMetadata(tc.src)
			if tc.src == nil {
				assert.Nil(t, result)
			} else {
				assert.Equal(t, len(tc.src), len(result))
				for k, v := range tc.src {
					assert.Equal(t, v, result[k])
				}
			}
		})
	}
}

func TestInterfaceCompliance(t *testing.T) {
	var _ driven.Normaliser = (*Normaliser)(nil)
}
