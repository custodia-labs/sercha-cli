package filesystem

import "strings"

// ResolveWebURL converts a filesystem URI to a local path for opening.
// Handles file:// URIs and bare paths.
func ResolveWebURL(uri string, _ map[string]any) string {
	// Strip file:// prefix for local paths
	if strings.HasPrefix(uri, "file://") {
		return strings.TrimPrefix(uri, "file://")
	}
	// Bare paths pass through unchanged
	return uri
}
