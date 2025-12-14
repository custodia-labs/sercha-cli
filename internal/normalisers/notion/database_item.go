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

// Ensure DatabaseItemNormaliser implements the interface.
var _ driven.Normaliser = (*DatabaseItemNormaliser)(nil)

// DatabaseItemNormaliser handles Notion database item (page in database) documents.
type DatabaseItemNormaliser struct{}

// NewDatabaseItem creates a new Notion database item normaliser.
func NewDatabaseItem() *DatabaseItemNormaliser {
	return &DatabaseItemNormaliser{}
}

// SupportedMIMETypes returns the MIME types this normaliser handles.
func (n *DatabaseItemNormaliser) SupportedMIMETypes() []string {
	return []string{MIMETypeNotionDBItem}
}

// SupportedConnectorTypes returns connector types for specialised handling.
func (n *DatabaseItemNormaliser) SupportedConnectorTypes() []string {
	return []string{"notion"}
}

// Priority returns the selection priority.
func (n *DatabaseItemNormaliser) Priority() int {
	return 95 // Connector-specific priority
}

// Normalise converts a Notion database item to a normalised document.
func (n *DatabaseItemNormaliser) Normalise(
	_ context.Context, raw *domain.RawDocument,
) (*driven.NormaliseResult, error) {
	if raw == nil {
		return nil, domain.ErrInvalidInput
	}

	// Extract title from metadata
	title := "Untitled"
	if t, ok := raw.Metadata["title"].(string); ok && t != "" {
		title = t
	}

	// Build content with header and properties
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s\n\n", title))

	// Add icon if present
	if icon, ok := raw.Metadata["icon"].(string); ok && icon != "" {
		sb.WriteString(fmt.Sprintf("%s ", icon))
	}

	// Add properties section (prefixed with "prop_" in metadata)
	var props []string
	for k, v := range raw.Metadata {
		if strings.HasPrefix(k, "prop_") {
			propName := strings.TrimPrefix(k, "prop_")
			props = append(props, fmt.Sprintf("- **%s:** %v", propName, v))
		}
	}
	if len(props) > 0 {
		sb.WriteString("## Properties\n\n")
		for _, p := range props {
			sb.WriteString(p + "\n")
		}
		sb.WriteString("\n")
	}

	// Add the page content (already extracted from blocks)
	content := string(raw.Content)
	if content != "" {
		sb.WriteString("## Content\n\n")
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
	doc.Metadata["format"] = "notion_database_item"

	return &driven.NormaliseResult{
		Document: doc,
	}, nil
}
