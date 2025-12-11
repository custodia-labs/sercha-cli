package markdown

import (
	"context"
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

// Normaliser handles Markdown documents.
type Normaliser struct{}

// New creates a new Markdown normaliser.
func New() *Normaliser {
	return &Normaliser{}
}

// SupportedMIMETypes returns the MIME types this normaliser handles.
func (n *Normaliser) SupportedMIMETypes() []string {
	return []string{"text/markdown", "text/x-markdown"}
}

// SupportedConnectorTypes returns connector types for specialised handling.
func (n *Normaliser) SupportedConnectorTypes() []string {
	return nil // All connectors
}

// Priority returns the selection priority.
func (n *Normaliser) Priority() int {
	return 50 // Generic MIME normaliser, higher than plaintext
}

// Normalise converts a markdown document to a normalised document.
// The Content field contains the text with markdown formatting simplified.
// Chunking is handled by the PostProcessor pipeline.
func (n *Normaliser) Normalise(_ context.Context, raw *domain.RawDocument) (*driven.NormaliseResult, error) {
	if raw == nil {
		return nil, domain.ErrInvalidInput
	}

	rawContent := string(raw.Content)

	// Extract title from first heading or filename
	title := extractMarkdownTitle(rawContent, raw.URI)

	// Convert markdown to plain text (simplified)
	content := stripMarkdown(rawContent)

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
	doc.Metadata["format"] = "markdown"

	return &driven.NormaliseResult{
		Document: doc,
	}, nil
}

// extractMarkdownTitle extracts a title from the markdown content or falls back to filename.
func extractMarkdownTitle(content, uri string) string {
	// Try to find first H1 heading (# Title)
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "#"))
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

// stripMarkdown removes common markdown formatting for plain text content.
// This is a simplified implementation that handles common cases.
func stripMarkdown(content string) string {
	// Remove code blocks (```...```)
	codeBlock := regexp.MustCompile("(?s)```[^`]*```")
	content = codeBlock.ReplaceAllString(content, "")

	// Remove inline code (`code`)
	inlineCode := regexp.MustCompile("`[^`]+`")
	content = inlineCode.ReplaceAllString(content, "")

	// Remove images ![alt](url)
	images := regexp.MustCompile(`!\[[^\]]*\]\([^)]+\)`)
	content = images.ReplaceAllString(content, "")

	// Convert links [text](url) to just text
	links := regexp.MustCompile(`\[([^\]]+)\]\([^)]+\)`)
	content = links.ReplaceAllString(content, "$1")

	// Remove heading markers (# ## ### etc)
	headings := regexp.MustCompile(`(?m)^#{1,6}\s+`)
	content = headings.ReplaceAllString(content, "")

	// Remove bold/italic markers
	content = strings.ReplaceAll(content, "**", "")
	content = strings.ReplaceAll(content, "__", "")
	content = strings.ReplaceAll(content, "*", "")
	content = strings.ReplaceAll(content, "_", " ")

	// Remove blockquote markers
	blockquote := regexp.MustCompile(`(?m)^>\s*`)
	content = blockquote.ReplaceAllString(content, "")

	// Remove horizontal rules
	hr := regexp.MustCompile(`(?m)^[-*_]{3,}\s*$`)
	content = hr.ReplaceAllString(content, "")

	// Remove list markers (- * + and numbered)
	listMarkers := regexp.MustCompile(`(?m)^\s*[-*+]\s+`)
	content = listMarkers.ReplaceAllString(content, "")
	numberedList := regexp.MustCompile(`(?m)^\s*\d+\.\s+`)
	content = numberedList.ReplaceAllString(content, "")

	// Collapse multiple newlines
	multiNewlines := regexp.MustCompile(`\n{3,}`)
	content = multiNewlines.ReplaceAllString(content, "\n\n")

	return strings.TrimSpace(content)
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
