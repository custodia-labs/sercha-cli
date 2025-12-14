package dropbox

import (
	"fmt"
	"net/url"
	"strings"
)

// ResolveWebURL converts a dropbox:// URI to a web URL.
// The metadata should contain the file path for web URL construction.
func ResolveWebURL(uri string, metadata map[string]any) string {
	// Priority 1: Use API-provided preview URL if available (future-proofing)
	// The Dropbox API may provide this via GetTemporaryLink or sharing endpoints
	if previewURL, ok := metadata["preview_url"].(string); ok && previewURL != "" {
		return previewURL
	}

	// Priority 2: Construct URL from path (Dropbox web interface pattern)
	// This URL pattern has been stable since 2015
	if path, ok := metadata["path"].(string); ok && path != "" {
		encodedPath := url.PathEscape(strings.TrimPrefix(path, "/"))
		return fmt.Sprintf("https://www.dropbox.com/home/%s", encodedPath)
	}

	// Priority 3: Use file ID for preview (requires shared link permissions)
	if fileID, ok := metadata["file_id"].(string); ok && fileID != "" {
		id := strings.TrimPrefix(fileID, "id:")
		return fmt.Sprintf("https://www.dropbox.com/preview/%s", id)
	}

	// Fallback to Dropbox home
	return "https://www.dropbox.com/home"
}
