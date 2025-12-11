package github

import "strings"

// ResolveWebURL converts a GitHub URI to a web URL.
// github://owner/repo/blob/branch/path -> https://github.com/owner/repo/blob/branch/path
func ResolveWebURL(uri string, _ map[string]any) string {
	if strings.HasPrefix(uri, "github://") {
		return "https://github.com/" + strings.TrimPrefix(uri, "github://")
	}
	return ""
}
