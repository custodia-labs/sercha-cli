package notion

import (
	"strings"
)

// ResolveWebURL converts a Notion internal URI to a web-accessible URL.
// URI patterns:
//   - notion://pages/{page_id}
//   - notion://databases/{database_id}
//   - notion://blocks/{block_id}
//
// Web URL patterns:
//   - https://www.notion.so/{page_id}
//   - https://www.notion.so/{database_id}
func ResolveWebURL(uri string, metadata map[string]any) string {
	const baseURL = "https://www.notion.so"

	// First check if we have a direct URL in metadata
	if metadata != nil {
		if url, ok := metadata["url"].(string); ok && url != "" {
			return url
		}
	}

	// Parse the URI
	if strings.HasPrefix(uri, "notion://pages/") {
		pageID := strings.TrimPrefix(uri, "notion://pages/")
		return buildNotionURL(baseURL, pageID)
	}

	if strings.HasPrefix(uri, "notion://databases/") {
		dbID := strings.TrimPrefix(uri, "notion://databases/")
		return buildNotionURL(baseURL, dbID)
	}

	if strings.HasPrefix(uri, "notion://blocks/") {
		blockID := strings.TrimPrefix(uri, "notion://blocks/")
		return buildNotionURL(baseURL, blockID)
	}

	// Fallback to base URL
	return baseURL
}

// buildNotionURL constructs a Notion web URL from an ID.
// Notion IDs can be UUIDs with or without hyphens.
func buildNotionURL(baseURL, id string) string {
	// Remove hyphens from UUID for cleaner URL (Notion accepts both)
	cleanID := strings.ReplaceAll(id, "-", "")
	return baseURL + "/" + cleanID
}

// ConfigKeys returns the configuration keys for the Notion connector.
func ConfigKeys() []string {
	return []string{
		"include_comments",
		"content_types",
		"max_block_depth",
		"page_size",
	}
}
