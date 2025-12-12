package onedrive

import "strings"

// ResolveWebURL converts a OneDrive URI to a web URL.
// Prefers the web_link from metadata if available.
func ResolveWebURL(uri string, metadata map[string]any) string {
	// First, check if we have the web_link in metadata (preferred)
	if metadata != nil {
		if webLink, ok := metadata["web_link"].(string); ok && webLink != "" {
			return webLink
		}
	}

	// Fallback: return generic OneDrive URL
	if strings.HasPrefix(uri, "onedrive://files/") ||
		strings.HasPrefix(uri, "onedrive://folders/") {
		return "https://onedrive.live.com/"
	}

	return ""
}
