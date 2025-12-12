package outlook

import (
	"strings"
)

// ResolveWebURL converts an Outlook URI to a web URL for the user.
// Returns empty string if the URI cannot be resolved.
func ResolveWebURL(uri string, metadata map[string]any) string {
	// First, check if we have the web_link in metadata (preferred)
	if metadata != nil {
		if webLink, ok := metadata["web_link"].(string); ok && webLink != "" {
			return webLink
		}
	}

	// Fallback: return generic Outlook web app URL
	if strings.HasPrefix(uri, "outlook://messages/") ||
		strings.HasPrefix(uri, "outlook://conversations/") {
		return "https://outlook.office.com/mail/"
	}
	return ""
}
