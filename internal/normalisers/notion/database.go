package notion

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
)

// Ensure DatabaseNormaliser implements the interface.
var _ driven.Normaliser = (*DatabaseNormaliser)(nil)

// DatabaseNormaliser handles Notion database documents.
type DatabaseNormaliser struct{}

// NewDatabase creates a new Notion database normaliser.
func NewDatabase() *DatabaseNormaliser {
	return &DatabaseNormaliser{}
}

// SupportedMIMETypes returns the MIME types this normaliser handles.
func (n *DatabaseNormaliser) SupportedMIMETypes() []string {
	return []string{MIMETypeNotionDB}
}

// SupportedConnectorTypes returns connector types for specialised handling.
func (n *DatabaseNormaliser) SupportedConnectorTypes() []string {
	return []string{"notion"}
}

// Priority returns the selection priority.
func (n *DatabaseNormaliser) Priority() int {
	return 95 // Connector-specific priority
}

// Normalise converts a Notion database document to a normalised document.
func (n *DatabaseNormaliser) Normalise(_ context.Context, raw *domain.RawDocument) (*driven.NormaliseResult, error) {
	if raw == nil {
		return nil, domain.ErrInvalidInput
	}

	// Extract title from metadata
	title := "Untitled Database"
	if t, ok := raw.Metadata["title"].(string); ok && t != "" {
		title = t
	}

	// Content is already formatted in the connector (database.go)
	// It includes title, description, and property schema
	content := string(raw.Content)

	// Build document
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

	// Add normaliser info to metadata
	if doc.Metadata == nil {
		doc.Metadata = make(map[string]any)
	}
	doc.Metadata["mime_type"] = raw.MIMEType
	doc.Metadata["format"] = "notion_database"

	return &driven.NormaliseResult{
		Document: doc,
	}, nil
}
