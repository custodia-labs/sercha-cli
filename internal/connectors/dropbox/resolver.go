package dropbox

import (
	"fmt"
	"net/url"
	"strings"
)

// ResolveWebURL converts a dropbox:// URI to a web URL.
// The metadata should contain the file path for web URL construction.
func ResolveWebURL(uri string, metadata map[string]any) string {
	// Try to use the path from metadata to construct a web URL
	if path, ok := metadata["path"].(string); ok && path != "" {
		// Dropbox web URL format: https://www.dropbox.com/home{path}
		// URL-encode the path for safety
		encodedPath := url.PathEscape(strings.TrimPrefix(path, "/"))
		return fmt.Sprintf("https://www.dropbox.com/home/%s", encodedPath)
	}

	// If we have a file_id, we can try to construct a preview URL
	// but this requires the file to be shared, so it's not always reliable
	if fileID, ok := metadata["file_id"].(string); ok && fileID != "" {
		// Strip the "id:" prefix if present
		id := strings.TrimPrefix(fileID, "id:")
		return fmt.Sprintf("https://www.dropbox.com/preview/%s", id)
	}

	// Fallback to Dropbox home
	return "https://www.dropbox.com/home"
}
