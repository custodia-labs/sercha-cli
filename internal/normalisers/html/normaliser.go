package html

import (
	"context"
	"html"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
)

// Ensure Normaliser implements the interface.
var _ driven.Normaliser = (*Normaliser)(nil)

// Normaliser handles HTML documents.
type Normaliser struct{}

// New creates a new HTML normaliser.
func New() *Normaliser {
	return &Normaliser{}
}

// SupportedMIMETypes returns the MIME types this normaliser handles.
func (n *Normaliser) SupportedMIMETypes() []string {
	return []string{"text/html", "application/xhtml+xml"}
}

// SupportedConnectorTypes returns connector types for specialised handling.
func (n *Normaliser) SupportedConnectorTypes() []string {
	return nil // All connectors
}

// Priority returns the selection priority.
func (n *Normaliser) Priority() int {
	return 50 // Generic MIME normaliser, higher than plaintext
}

// Normalise converts an HTML document to a normalised document.
// The Content field contains the text with HTML tags stripped.
// Chunking is handled by the PostProcessor pipeline.
func (n *Normaliser) Normalise(_ context.Context, raw *domain.RawDocument) (*driven.NormaliseResult, error) {
	if raw == nil {
		return nil, domain.ErrInvalidInput
	}

	rawContent := string(raw.Content)

	// Extract title from <title> tag or filename
	title := extractHTMLTitle(rawContent, raw.URI)

	// Convert HTML to plain text
	content := stripHTML(rawContent)

	// Build document with Content field populated
	doc := domain.Document{
		ID:        uuid.New().String(),
		SourceID:  raw.SourceID,
		URI:       raw.URI,
		Title:     title,
		Content:   content,
		Metadata:  copyMetadata(raw.Metadata),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Add MIME type and format info to metadata
	if doc.Metadata == nil {
		doc.Metadata = make(map[string]any)
	}
	doc.Metadata["mime_type"] = raw.MIMEType
	doc.Metadata["format"] = "html"

	return &driven.NormaliseResult{
		Document: doc,
	}, nil
}

// Pre-compiled regular expressions for HTML parsing performance.
var (
	titleTag          = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
	scriptTag         = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	styleTag          = regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
	noscriptTag       = regexp.MustCompile(`(?is)<noscript[^>]*>.*?</noscript>`)
	headTag           = regexp.MustCompile(`(?is)<head[^>]*>.*?</head>`)
	svgTag            = regexp.MustCompile(`(?is)<svg[^>]*>.*?</svg>`)
	htmlComments      = regexp.MustCompile(`(?s)<!--.*?-->`)
	blockElements     = regexp.MustCompile(`(?i)</(p|div|br|hr|h[1-6]|li|tr|blockquote|pre|table|section|article)>`)
	openBlockElements = regexp.MustCompile(`(?i)<(p|div|h[1-6]|li|tr|blockquote|pre|table|section|article)[^>]*>`)
	brTags            = regexp.MustCompile(`(?i)<br\s*/?>`)
	hrTags            = regexp.MustCompile(`(?i)<hr\s*/?>`)
	allTags           = regexp.MustCompile(`<[^>]+>`)
	multiSpaces       = regexp.MustCompile(`[ \t]+`)
	multiNewlines     = regexp.MustCompile(`\n{3,}`)
)

// extractHTMLTitle extracts a title from the HTML content or falls back to filename.
func extractHTMLTitle(content, uri string) string {
	// Try to find <title> tag
	matches := titleTag.FindStringSubmatch(content)
	if len(matches) > 1 {
		title := strings.TrimSpace(matches[1])
		// Decode HTML entities in title
		title = html.UnescapeString(title)
		if title != "" {
			return title
		}
	}

	// Fall back to filename
	filename := filepath.Base(uri)
	ext := filepath.Ext(filename)
	if ext != "" {
		filename = strings.TrimSuffix(filename, ext)
	}
	filename = strings.ReplaceAll(filename, "_", " ")
	filename = strings.ReplaceAll(filename, "-", " ")
	return filename
}

// stripHTML removes HTML tags and extracts readable text content.
func stripHTML(content string) string {
	// Remove script, style, noscript, head, and svg tags entirely
	content = scriptTag.ReplaceAllString(content, "")
	content = styleTag.ReplaceAllString(content, "")
	content = noscriptTag.ReplaceAllString(content, "")
	content = headTag.ReplaceAllString(content, "")
	content = svgTag.ReplaceAllString(content, "")

	// Remove HTML comments
	content = htmlComments.ReplaceAllString(content, "")

	// Add newlines before block elements for readability
	content = openBlockElements.ReplaceAllString(content, "\n")

	// Add newlines after closing block elements
	content = blockElements.ReplaceAllString(content, "\n")

	// Convert <br> and <hr> to newlines
	content = brTags.ReplaceAllString(content, "\n")
	content = hrTags.ReplaceAllString(content, "\n")

	// Strip all remaining HTML tags
	content = allTags.ReplaceAllString(content, "")

	// Decode HTML entities
	content = html.UnescapeString(content)

	// Collapse multiple spaces (but preserve newlines)
	content = multiSpaces.ReplaceAllString(content, " ")

	// Collapse multiple newlines
	content = multiNewlines.ReplaceAllString(content, "\n\n")

	// Trim each line and remove empty lines
	lines := strings.Split(content, "\n")
	var result []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}

// copyMetadata creates a shallow copy of metadata.
func copyMetadata(src map[string]any) map[string]any {
	if src == nil {
		return nil
	}
	dst := make(map[string]any, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
