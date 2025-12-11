package github

import (
	"regexp"
	"strings"
)

// linkRegex matches Link header entries: <url>; rel="type".
var linkRegex = regexp.MustCompile(`<([^>]+)>;\s*rel="([^"]+)"`)

// ParseNextLink extracts the "next" URL from a Link header.
// Returns empty string if no next link is found.
func ParseNextLink(linkHeader string) string {
	if linkHeader == "" {
		return ""
	}

	// Split by comma for multiple links
	parts := strings.Split(linkHeader, ",")
	for _, part := range parts {
		matches := linkRegex.FindStringSubmatch(strings.TrimSpace(part))
		if len(matches) == 3 && matches[2] == "next" {
			return matches[1]
		}
	}

	return ""
}

// ParseAllLinks extracts all URLs from a Link header by relationship type.
// Returns a map of rel type to URL.
func ParseAllLinks(linkHeader string) map[string]string {
	links := make(map[string]string)
	if linkHeader == "" {
		return links
	}

	parts := strings.Split(linkHeader, ",")
	for _, part := range parts {
		matches := linkRegex.FindStringSubmatch(strings.TrimSpace(part))
		if len(matches) == 3 {
			links[matches[2]] = matches[1]
		}
	}

	return links
}

// HasNextPage checks if there is a next page available.
func HasNextPage(linkHeader string) bool {
	return ParseNextLink(linkHeader) != ""
}

// GetLastPage extracts the "last" URL from a Link header.
func GetLastPage(linkHeader string) string {
	links := ParseAllLinks(linkHeader)
	return links["last"]
}
