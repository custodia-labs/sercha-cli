package github

import (
	"strings"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

// ContentType represents the type of content to index.
type ContentType string

const (
	ContentFiles  ContentType = "files"
	ContentIssues ContentType = "issues"
	ContentPRs    ContentType = "prs"
	ContentWikis  ContentType = "wikis"
)

// AllContentTypes returns all supported content types.
func AllContentTypes() []ContentType {
	return []ContentType{ContentFiles, ContentIssues, ContentPRs, ContentWikis}
}

// Config holds the parsed configuration for a GitHub source.
type Config struct {
	// ContentTypes specifies what content to index.
	// Default: all types (files, issues, prs, wikis)
	ContentTypes []ContentType

	// FilePatterns are glob patterns for file filtering.
	// Default: all files
	FilePatterns []string
}

// ParseConfig parses a source's config map into a Config struct.
// All fields are optional - by default indexes all accessible repos with all content types.
func ParseConfig(source domain.Source) (*Config, error) {
	cfg := &Config{
		ContentTypes: AllContentTypes(), // Default to all content types
		FilePatterns: []string{},        // Empty = all files
	}

	// Parse content_types (optional)
	if contentTypes, ok := source.Config["content_types"]; ok && contentTypes != "" {
		types, err := parseContentTypes(contentTypes)
		if err != nil {
			return nil, err
		}
		cfg.ContentTypes = types
	}

	// Parse file_patterns (optional)
	if patterns, ok := source.Config["file_patterns"]; ok && patterns != "" {
		cfg.FilePatterns = parsePatterns(patterns)
	}

	return cfg, nil
}

// parseContentTypes parses a comma-separated content types string.
func parseContentTypes(s string) ([]ContentType, error) {
	parts := strings.Split(s, ",")
	types := make([]ContentType, 0, len(parts))
	valid := map[string]ContentType{
		"files":  ContentFiles,
		"issues": ContentIssues,
		"prs":    ContentPRs,
		"wikis":  ContentWikis,
	}

	for _, part := range parts {
		part = strings.TrimSpace(strings.ToLower(part))
		if part == "" {
			continue
		}
		ct, ok := valid[part]
		if !ok {
			return nil, ErrConfigInvalidContentType
		}
		types = append(types, ct)
	}

	if len(types) == 0 {
		return AllContentTypes(), nil
	}
	return types, nil
}

// parsePatterns parses a comma-separated glob patterns string.
func parsePatterns(s string) []string {
	parts := strings.Split(s, ",")
	patterns := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			patterns = append(patterns, part)
		}
	}
	return patterns
}

// HasContentType checks if a content type is enabled.
func (c *Config) HasContentType(ct ContentType) bool {
	for _, t := range c.ContentTypes {
		if t == ct {
			return true
		}
	}
	return false
}
