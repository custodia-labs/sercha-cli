package ics

import (
	"bufio"
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
)

// Ensure Normaliser implements the interface.
var _ driven.Normaliser = (*Normaliser)(nil)

// Normaliser handles ICS (iCalendar) documents.
type Normaliser struct{}

// New creates a new ICS normaliser.
func New() *Normaliser {
	return &Normaliser{}
}

// SupportedMIMETypes returns the MIME types this normaliser handles.
func (n *Normaliser) SupportedMIMETypes() []string {
	return []string{
		"text/calendar",
	}
}

// SupportedConnectorTypes returns connector types for specialised handling.
func (n *Normaliser) SupportedConnectorTypes() []string {
	return nil // All connectors
}

// Priority returns the selection priority.
func (n *Normaliser) Priority() int {
	return 50 // Generic MIME normaliser
}

// event represents a parsed calendar event.
type event struct {
	Summary     string
	Description string
	Location    string
	Start       string
	End         string
	Organiser   string
	Attendees   []string
}

// Normalise converts an ICS document to a normalised document.
func (n *Normaliser) Normalise(_ context.Context, raw *domain.RawDocument) (*driven.NormaliseResult, error) {
	if raw == nil {
		return nil, domain.ErrInvalidInput
	}

	// Parse the iCalendar content
	events, calendarName := parseICS(raw.Content)

	if len(events) == 0 {
		// If no events found, still create a document with raw content
		return n.createDocument(raw, "", string(raw.Content)), nil
	}

	// Build searchable content from all events
	var content strings.Builder
	for i := range events {
		if i > 0 {
			content.WriteString("\n---\n\n")
		}
		content.WriteString(formatEvent(&events[i]))
	}

	// Use first event summary or calendar name as title
	title := ""
	if len(events) > 0 && events[0].Summary != "" {
		title = events[0].Summary
		if len(events) > 1 {
			title += " (and more)"
		}
	} else if calendarName != "" {
		title = calendarName
	}

	return n.createDocument(raw, title, content.String()), nil
}

// createDocument builds the normalised document.
func (n *Normaliser) createDocument(raw *domain.RawDocument, title, content string) *driven.NormaliseResult {
	if title == "" {
		title = extractTitleFromMetadataOrURI(raw)
	}

	doc := domain.Document{
		ID:        uuid.New().String(),
		SourceID:  raw.SourceID,
		URI:       raw.URI,
		Title:     title,
		Content:   strings.TrimSpace(content),
		Metadata:  copyMetadata(raw.Metadata),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if doc.Metadata == nil {
		doc.Metadata = make(map[string]any)
	}
	doc.Metadata["mime_type"] = raw.MIMEType
	doc.Metadata["format"] = "ics"

	return &driven.NormaliseResult{
		Document: doc,
	}
}

// parseState holds the state during ICS parsing.
type parseState struct {
	events       []event
	currentEvent *event
	calendarName string
	currentProp  string
	currentValue strings.Builder
}

// parseICS extracts events from iCalendar content.
func parseICS(content []byte) (events []event, calendarName string) {
	state := &parseState{}

	scanner := bufio.NewScanner(bytes.NewReader(content))
	for scanner.Scan() {
		state.processLine(scanner.Text())
	}

	// Don't forget the last property
	state.flushProperty()

	return state.events, state.calendarName
}

// processLine handles a single line of ICS content.
func (s *parseState) processLine(line string) {
	// Handle line folding (lines starting with space are continuations)
	if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
		if s.currentProp != "" {
			s.currentValue.WriteString(strings.TrimPrefix(strings.TrimPrefix(line, " "), "\t"))
		}
		return
	}

	// Flush previous property if we have one
	s.flushProperty()

	// Handle component markers
	if line == "BEGIN:VEVENT" {
		s.currentEvent = &event{}
		return
	}
	if line == "END:VEVENT" {
		if s.currentEvent != nil {
			s.events = append(s.events, *s.currentEvent)
		}
		s.currentEvent = nil
		return
	}

	// Parse property:value
	prop, value := parsePropValue(line)
	if prop == "" {
		return
	}

	// Calendar-level properties
	if s.currentEvent == nil {
		if prop == "X-WR-CALNAME" {
			s.calendarName = decodeValue(value)
		}
		return
	}

	// Start collecting this property
	s.currentProp = prop
	s.currentValue.WriteString(value)
}

// flushProperty applies the current property to the event and resets state.
func (s *parseState) flushProperty() {
	if s.currentProp != "" && s.currentEvent != nil {
		applyProperty(s.currentEvent, s.currentProp, s.currentValue.String())
	}
	s.currentProp = ""
	s.currentValue.Reset()
}

// parsePropValue extracts property name and value from a line.
func parsePropValue(line string) (prop, value string) {
	colonIdx := strings.Index(line, ":")
	if colonIdx == -1 {
		return "", ""
	}

	prop = line[:colonIdx]
	value = line[colonIdx+1:]

	// Handle properties with parameters (e.g., DTSTART;VALUE=DATE:20240101)
	if semicolonIdx := strings.Index(prop, ";"); semicolonIdx != -1 {
		prop = prop[:semicolonIdx]
	}

	return prop, value
}

// applyProperty sets the property value on the event.
func applyProperty(evt *event, prop, value string) {
	decoded := decodeValue(value)
	switch prop {
	case "SUMMARY":
		evt.Summary = decoded
	case "DESCRIPTION":
		evt.Description = decoded
	case "LOCATION":
		evt.Location = decoded
	case "DTSTART":
		evt.Start = formatDateTime(value)
	case "DTEND":
		evt.End = formatDateTime(value)
	case "ORGANIZER": //nolint:misspell // iCalendar standard uses American spelling
		evt.Organiser = extractEmail(decoded)
	case "ATTENDEE":
		if email := extractEmail(decoded); email != "" {
			evt.Attendees = append(evt.Attendees, email)
		}
	}
}

// decodeValue handles iCalendar escape sequences.
func decodeValue(value string) string {
	// Handle common escape sequences
	value = strings.ReplaceAll(value, "\\n", "\n")
	value = strings.ReplaceAll(value, "\\N", "\n")
	value = strings.ReplaceAll(value, "\\,", ",")
	value = strings.ReplaceAll(value, "\\;", ";")
	value = strings.ReplaceAll(value, "\\\\", "\\")
	return value
}

// formatDateTime converts iCalendar date/time to readable format.
func formatDateTime(value string) string {
	// Handle basic date format: 20240115
	if len(value) == 8 {
		if t, err := time.Parse("20060102", value); err == nil {
			return t.Format("January 2, 2006")
		}
	}

	// Handle date-time format: 20240115T100000 or 20240115T100000Z
	value = strings.TrimSuffix(value, "Z")
	if len(value) == 15 {
		if t, err := time.Parse("20060102T150405", value); err == nil {
			return t.Format("January 2, 2006 at 3:04 PM")
		}
	}

	return value
}

// extractEmail extracts email from organiser/attendee values.
func extractEmail(value string) string {
	// Common formats: "mailto:email@example.com" or just "email@example.com"
	value = strings.TrimPrefix(value, "mailto:")
	value = strings.TrimPrefix(value, "MAILTO:")
	if strings.Contains(value, "@") {
		return value
	}
	return ""
}

// formatEvent creates readable text from an event.
func formatEvent(evt *event) string {
	var result strings.Builder

	if evt.Summary != "" {
		result.WriteString("Event: ")
		result.WriteString(evt.Summary)
		result.WriteString("\n")
	}

	if evt.Start != "" {
		result.WriteString("When: ")
		result.WriteString(evt.Start)
		if evt.End != "" && evt.End != evt.Start {
			result.WriteString(" to ")
			result.WriteString(evt.End)
		}
		result.WriteString("\n")
	}

	if evt.Location != "" {
		result.WriteString("Where: ")
		result.WriteString(evt.Location)
		result.WriteString("\n")
	}

	if evt.Organiser != "" {
		result.WriteString("Organiser: ")
		result.WriteString(evt.Organiser)
		result.WriteString("\n")
	}

	if len(evt.Attendees) > 0 {
		result.WriteString("Attendees: ")
		result.WriteString(strings.Join(evt.Attendees, ", "))
		result.WriteString("\n")
	}

	if evt.Description != "" {
		result.WriteString("\n")
		result.WriteString(evt.Description)
		result.WriteString("\n")
	}

	return result.String()
}

// extractTitleFromMetadataOrURI checks metadata for title first, then falls back to URI.
func extractTitleFromMetadataOrURI(raw *domain.RawDocument) string {
	if raw.Metadata != nil {
		if title, ok := raw.Metadata["title"].(string); ok && title != "" {
			return title
		}
	}
	return extractTitleFromURI(raw.URI)
}

// extractTitleFromURI extracts a title from the file URI.
func extractTitleFromURI(uri string) string {
	filename := filepath.Base(uri)
	ext := filepath.Ext(filename)
	if ext != "" {
		filename = strings.TrimSuffix(filename, ext)
	}
	filename = strings.ReplaceAll(filename, "_", " ")
	filename = strings.ReplaceAll(filename, "-", " ")
	return filename
}

// copyMetadata creates a shallow copy of metadata.
func copyMetadata(src map[string]any) map[string]any {
	if src == nil {
		return nil
	}
	dst := make(map[string]any, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
