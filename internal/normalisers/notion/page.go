package notion

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
)

// MIME types for Notion documents.
const (
	MIMETypeNotionPage   = "application/vnd.notion.page+json"
	MIMETypeNotionDB     = "application/vnd.notion.database+json"
	MIMETypeNotionDBItem = "application/vnd.notion.database-item+json"
)

// Ensure PageNormaliser implements the interface.
var _ driven.Normaliser = (*PageNormaliser)(nil)

// PageNormaliser handles Notion page documents.
type PageNormaliser struct{}

// NewPage creates a new Notion page normaliser.
func NewPage() *PageNormaliser {
	return &PageNormaliser{}
}

// SupportedMIMETypes returns the MIME types this normaliser handles.
func (n *PageNormaliser) SupportedMIMETypes() []string {
	return []string{MIMETypeNotionPage}
}

// SupportedConnectorTypes returns connector types for specialised handling.
func (n *PageNormaliser) SupportedConnectorTypes() []string {
	return []string{"notion"}
}

// Priority returns the selection priority.
func (n *PageNormaliser) Priority() int {
	return 95 // Connector-specific priority
}

// Normalise converts a Notion page document to a normalised document.
func (n *PageNormaliser) Normalise(_ context.Context, raw *domain.RawDocument) (*driven.NormaliseResult, error) {
	if raw == nil {
		return nil, domain.ErrInvalidInput
	}

	// Extract title from metadata
	title := "Untitled"
	if t, ok := raw.Metadata["title"].(string); ok && t != "" {
		title = t
	}

	// Build content with header
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s\n\n", title))

	// Add metadata section if there's useful info
	if icon, ok := raw.Metadata["icon"].(string); ok && icon != "" {
		sb.WriteString(fmt.Sprintf("%s ", icon))
	}

	// Add the page content (already extracted from blocks)
	content := string(raw.Content)
	if content != "" {
		sb.WriteString(content)
	}

	// Add comments if present
	if comments, ok := raw.Metadata["comments"].([]string); ok && len(comments) > 0 {
		sb.WriteString("\n\n---\n\n## Comments\n\n")
		for _, comment := range comments {
			sb.WriteString(fmt.Sprintf("- %s\n", comment))
		}
	}

	// Build document
	doc := domain.Document{
		ID:        uuid.New().String(),
		SourceID:  raw.SourceID,
		URI:       raw.URI,
		Title:     title,
		Content:   sb.String(),
		Metadata:  copyMetadata(raw.Metadata),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Add normaliser info to metadata
	if doc.Metadata == nil {
		doc.Metadata = make(map[string]any)
	}
	doc.Metadata["mime_type"] = raw.MIMEType
	doc.Metadata["format"] = "notion_page"

	return &driven.NormaliseResult{
		Document: doc,
	}, nil
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
