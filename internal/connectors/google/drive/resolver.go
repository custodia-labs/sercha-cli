package drive

import "strings"

// ResolveWebURL converts a Google Drive URI to a web URL.
// Checks metadata for stored web_link first, then falls back to URI conversion.
func ResolveWebURL(uri string, metadata map[string]any) string {
	// Check metadata first (stored during sync from Google API)
	if metadata != nil {
		if webLink, ok := metadata["web_link"].(string); ok && webLink != "" {
			return webLink
		}
	}

	// Fallback: gdrive://files/{id} -> https://drive.google.com/file/d/{id}/view
	if strings.HasPrefix(uri, "gdrive://files/") {
		fileID := strings.TrimPrefix(uri, "gdrive://files/")
		return "https://drive.google.com/file/d/" + fileID + "/view"
	}

	return ""
}
