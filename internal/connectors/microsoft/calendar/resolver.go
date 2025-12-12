package calendar

import "strings"

// ResolveWebURL converts a Microsoft Calendar URI to a web URL.
// URIs are in format: mscal://{calendarId}/events/{eventId}
func ResolveWebURL(uri string, metadata map[string]any) string {
	if !strings.HasPrefix(uri, "mscal://") {
		return ""
	}

	// Use webLink from metadata if available (stored as html_link)
	if metadata != nil {
		if webLink, ok := metadata["html_link"].(string); ok && webLink != "" {
			return webLink
		}
	}

	// Fallback to generic Outlook Calendar web interface
	return "https://outlook.office.com/calendar/"
}
