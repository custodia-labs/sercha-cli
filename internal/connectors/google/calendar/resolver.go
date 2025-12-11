package calendar

// ResolveWebURL converts a Google Calendar URI to a web URL.
// Calendar events store the html_link in metadata (complex URL with encoded event ID).
func ResolveWebURL(_ string, metadata map[string]any) string {
	// Calendar stores html_link in metadata from Google API
	if metadata != nil {
		if htmlLink, ok := metadata["html_link"].(string); ok && htmlLink != "" {
			return htmlLink
		}
	}
	return ""
}
