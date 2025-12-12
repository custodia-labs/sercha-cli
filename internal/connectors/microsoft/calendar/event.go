package calendar

import (
	"fmt"
	"strings"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

// Event represents a Microsoft Calendar event from the Graph API.
type Event struct {
	ID                   string        `json:"id"`
	Subject              string        `json:"subject"`
	Body                 *EventBody    `json:"body,omitempty"`
	Start                *DateTimeZone `json:"start,omitempty"`
	End                  *DateTimeZone `json:"end,omitempty"`
	Location             *Location     `json:"location,omitempty"`
	Organiser            *EmailAddress `json:"organizer,omitempty"` //nolint:misspell // Microsoft API field name
	Attendees            []Attendee    `json:"attendees,omitempty"`
	WebLink              string        `json:"webLink"`
	IsCancelled          bool          `json:"isCancelled"`
	IsAllDay             bool          `json:"isAllDay"`
	Importance           string        `json:"importance"`
	Sensitivity          string        `json:"sensitivity"`
	ShowAs               string        `json:"showAs"`
	Categories           []string      `json:"categories,omitempty"`
	SeriesMasterID       string        `json:"seriesMasterId,omitempty"`
	Recurrence           *Recurrence   `json:"recurrence,omitempty"`
	CreatedDateTime      string        `json:"createdDateTime"`
	LastModifiedDateTime string        `json:"lastModifiedDateTime"`
}

// EventBody contains the event body content.
type EventBody struct {
	ContentType string `json:"contentType"`
	Content     string `json:"content"`
}

// DateTimeZone contains a date-time with time zone.
type DateTimeZone struct {
	DateTime string `json:"dateTime"`
	TimeZone string `json:"timeZone"`
}

// Location contains location information.
type Location struct {
	DisplayName string `json:"displayName"`
	Address     *struct {
		Street     string `json:"street"`
		City       string `json:"city"`
		State      string `json:"state"`
		PostalCode string `json:"postalCode"`
		Country    string `json:"countryOrRegion"`
	} `json:"address,omitempty"`
}

// EmailAddress contains email address information.
type EmailAddress struct {
	EmailAddress struct {
		Name    string `json:"name"`
		Address string `json:"address"`
	} `json:"emailAddress"`
}

// Attendee represents an event attendee.
type Attendee struct {
	Type   string `json:"type"`
	Status *struct {
		Response string `json:"response"`
		Time     string `json:"time"`
	} `json:"status,omitempty"`
	EmailAddress struct {
		Name    string `json:"name"`
		Address string `json:"address"`
	} `json:"emailAddress"`
}

// Recurrence contains recurrence pattern information.
type Recurrence struct {
	Pattern *struct {
		Type       string   `json:"type"`
		Interval   int      `json:"interval"`
		DaysOfWeek []string `json:"daysOfWeek,omitempty"`
	} `json:"pattern,omitempty"`
	Range *struct {
		Type      string `json:"type"`
		StartDate string `json:"startDate"`
		EndDate   string `json:"endDate,omitempty"`
	} `json:"range,omitempty"`
}

// EventWithRemoved wraps Event with removal detection for delta queries.
type EventWithRemoved struct {
	Event
	Removed *struct {
		Reason string `json:"reason"`
	} `json:"@removed,omitempty"`
}

// EventToRawDocument converts a Microsoft Calendar event to a RawDocument.
func EventToRawDocument(event *Event, calendarID, sourceID string) *domain.RawDocument {
	content := buildEventContent(event)
	startTime, endTime := extractEventTimes(event)
	parentURI := buildSeriesMasterURI(event, calendarID)

	metadata := map[string]any{
		"event_id":     event.ID,
		"calendar_id":  calendarID,
		"title":        event.Subject,
		"start_time":   startTime,
		"end_time":     endTime,
		"is_all_day":   event.IsAllDay,
		"is_cancelled": event.IsCancelled,
		"importance":   event.Importance,
		"sensitivity":  event.Sensitivity,
		"show_as":      event.ShowAs,
		"html_link":    event.WebLink,
		"created":      event.CreatedDateTime,
		"updated":      event.LastModifiedDateTime,
	}

	if event.Location != nil && event.Location.DisplayName != "" {
		metadata["location"] = event.Location.DisplayName
	}

	if event.Organiser != nil {
		metadata["organiser"] = event.Organiser.EmailAddress.Address
		metadata["organiser_name"] = event.Organiser.EmailAddress.Name
	}

	if event.SeriesMasterID != "" {
		metadata["series_master_id"] = event.SeriesMasterID
	}

	if len(event.Categories) > 0 {
		metadata["categories"] = event.Categories
	}

	return &domain.RawDocument{
		SourceID:  sourceID,
		URI:       fmt.Sprintf("mscal://%s/events/%s", calendarID, event.ID),
		MIMEType:  "text/calendar",
		Content:   []byte(content),
		ParentURI: parentURI,
		Metadata:  metadata,
	}
}

// buildEventContent constructs the content string from event details.
func buildEventContent(event *Event) string {
	var contentParts []string

	if event.Subject != "" {
		contentParts = append(contentParts, event.Subject)
	}

	if event.Body != nil && event.Body.Content != "" {
		// Strip HTML if content type is HTML
		content := event.Body.Content
		if event.Body.ContentType == "html" {
			content = stripHTMLTags(content)
		}
		if content != "" {
			contentParts = append(contentParts, content)
		}
	}

	if event.Location != nil && event.Location.DisplayName != "" {
		contentParts = append(contentParts, "Location: "+event.Location.DisplayName)
	}

	if attendeeStr := formatAttendees(event.Attendees); attendeeStr != "" {
		contentParts = append(contentParts, attendeeStr)
	}

	return strings.Join(contentParts, "\n\n")
}

// stripHTMLTags removes HTML tags from a string (simple implementation).
func stripHTMLTags(s string) string {
	var result strings.Builder
	var inTag bool

	for _, r := range s {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			result.WriteRune(r)
		}
	}

	return strings.TrimSpace(result.String())
}

// formatAttendees formats the attendee list as a string.
func formatAttendees(attendees []Attendee) string {
	if len(attendees) == 0 {
		return ""
	}

	var names []string
	for _, a := range attendees {
		if a.EmailAddress.Name != "" {
			names = append(names, a.EmailAddress.Name)
		} else if a.EmailAddress.Address != "" {
			names = append(names, a.EmailAddress.Address)
		}
	}

	if len(names) == 0 {
		return ""
	}
	return "Attendees: " + strings.Join(names, ", ")
}

// extractEventTimes extracts start and end times from an event.
func extractEventTimes(event *Event) (startTime, endTime string) {
	if event.Start != nil {
		startTime = event.Start.DateTime
	}
	if event.End != nil {
		endTime = event.End.DateTime
	}
	return startTime, endTime
}

// buildSeriesMasterURI builds a parent URI for recurring event instances.
func buildSeriesMasterURI(event *Event, calendarID string) *string {
	if event.SeriesMasterID != "" && event.SeriesMasterID != event.ID {
		uri := fmt.Sprintf("mscal://%s/events/%s", calendarID, event.SeriesMasterID)
		return &uri
	}
	return nil
}

// ShouldSyncEvent checks if an event should be synced.
func ShouldSyncEvent(event *Event) bool {
	return event != nil && event.ID != ""
}

// IsEventRemoved checks if a delta response event was removed.
func IsEventRemoved(event *EventWithRemoved) bool {
	return event.Removed != nil || event.IsCancelled
}
