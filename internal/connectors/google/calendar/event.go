package calendar

import (
	"fmt"
	"strings"

	"google.golang.org/api/calendar/v3"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

// EventToRawDocument converts a Google Calendar event to a RawDocument.
func EventToRawDocument(event *calendar.Event, calendarID, sourceID string) *domain.RawDocument {
	content := buildEventContent(event)
	startTime, endTime := extractEventTimes(event)
	parentURI := buildRecurringParentURI(event, calendarID)

	return &domain.RawDocument{
		SourceID:  sourceID,
		URI:       fmt.Sprintf("gcal://%s/events/%s", calendarID, event.Id),
		MIMEType:  "text/calendar",
		Content:   []byte(content),
		ParentURI: parentURI,
		Metadata: map[string]any{
			"event_id":           event.Id,
			"calendar_id":        calendarID,
			"title":              event.Summary,
			"description":        event.Description,
			"location":           event.Location,
			"start_time":         startTime,
			"end_time":           endTime,
			"status":             event.Status,
			"html_link":          event.HtmlLink,
			"recurring_event_id": event.RecurringEventId,
			"organiser":          getOrganiserEmail(event),
			"created":            event.Created,
			"updated":            event.Updated,
		},
	}
}

// buildEventContent constructs the content string from event details.
func buildEventContent(event *calendar.Event) string {
	var contentParts []string
	if event.Summary != "" {
		contentParts = append(contentParts, event.Summary)
	}
	if event.Description != "" {
		contentParts = append(contentParts, event.Description)
	}
	if event.Location != "" {
		contentParts = append(contentParts, "Location: "+event.Location)
	}

	if attendeeStr := formatAttendees(event.Attendees); attendeeStr != "" {
		contentParts = append(contentParts, attendeeStr)
	}

	return strings.Join(contentParts, "\n\n")
}

// formatAttendees formats the attendee list as a string.
func formatAttendees(attendees []*calendar.EventAttendee) string {
	if len(attendees) == 0 {
		return ""
	}

	var names []string
	for _, a := range attendees {
		if a.DisplayName != "" {
			names = append(names, a.DisplayName)
		} else if a.Email != "" {
			names = append(names, a.Email)
		}
	}

	if len(names) == 0 {
		return ""
	}
	return "Attendees: " + strings.Join(names, ", ")
}

// extractEventTimes extracts start and end times from an event.
func extractEventTimes(event *calendar.Event) (startTime, endTime string) {
	if event.Start != nil {
		if event.Start.DateTime != "" {
			startTime = event.Start.DateTime
		} else {
			startTime = event.Start.Date
		}
	}
	if event.End != nil {
		if event.End.DateTime != "" {
			endTime = event.End.DateTime
		} else {
			endTime = event.End.Date
		}
	}
	return startTime, endTime
}

// buildRecurringParentURI builds a parent URI for recurring event instances.
func buildRecurringParentURI(event *calendar.Event, calendarID string) *string {
	if event.RecurringEventId != "" && event.RecurringEventId != event.Id {
		uri := fmt.Sprintf("gcal://%s/events/%s", calendarID, event.RecurringEventId)
		return &uri
	}
	return nil
}

// getOrganiserEmail extracts the organiser email from an event.
func getOrganiserEmail(event *calendar.Event) string {
	if event.Organizer != nil { //nolint:misspell // Google API field name
		return event.Organizer.Email //nolint:misspell // Google API field name
	}
	return ""
}

// ShouldSyncEvent checks if an event should be synced.
func ShouldSyncEvent(event *calendar.Event) bool {
	// Skip cancelled events unless we want them for deletion tracking
	// (handled by ShowDeleted config)
	return event != nil && event.Id != ""
}
