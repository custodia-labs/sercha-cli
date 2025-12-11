package plaintext

import (
	"context"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
)

// Ensure Normaliser implements the interface.
var _ driven.Normaliser = (*Normaliser)(nil)

// Normaliser handles plain text documents.
type Normaliser struct{}

// New creates a new plain text normaliser.
func New() *Normaliser {
	return &Normaliser{}
}

// SupportedMIMETypes returns the MIME types this normaliser handles.
func (n *Normaliser) SupportedMIMETypes() []string {
	return []string{
		"text/plain",
		"text/x-go",
		"text/x-python",
		"text/x-rust",
		"text/x-java",
		"text/x-c",
		"text/x-c++",
		"text/x-ruby",
		"text/x-shellscript",
		"text/x-sql",
		"text/csv",
		"text/yaml",
		"text/toml",
		"text/javascript",
		"text/jsx",
		"text/typescript",
		"text/typescript-jsx",
		"text/css",
		"text/html",
		"application/json",
		"application/xml",
		"image/svg+xml",
	}
}

// SupportedConnectorTypes returns connector types for specialised handling.
func (n *Normaliser) SupportedConnectorTypes() []string {
	return nil // All connectors
}

// Priority returns the selection priority.
func (n *Normaliser) Priority() int {
	return 5 // Fallback normaliser
}

// Normalise converts a raw document to a normalised document.
// The Content field contains the full text content.
// Chunking is handled by the PostProcessor pipeline.
func (n *Normaliser) Normalise(_ context.Context, raw *domain.RawDocument) (*driven.NormaliseResult, error) {
	if raw == nil {
		return nil, domain.ErrInvalidInput
	}

	// Extract title from metadata if available, otherwise from URI
	title := extractTitleFromMetadataOrURI(raw)

	// Convert raw bytes to string content
	content := string(raw.Content)

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

	// Add MIME type to metadata for reference
	if doc.Metadata == nil {
		doc.Metadata = make(map[string]any)
	}
	doc.Metadata["mime_type"] = raw.MIMEType

	return &driven.NormaliseResult{
		Document: doc,
	}, nil
}

// extractTitleFromMetadataOrURI checks metadata for title first, then falls back to URI.
// This supports connectors like Google Drive that set Metadata["title"] to the actual file name.
func extractTitleFromMetadataOrURI(raw *domain.RawDocument) string {
	if raw.Metadata != nil {
		if title, ok := raw.Metadata["title"].(string); ok && title != "" {
			return title
		}
	}
	return extractTitle(raw.URI)
}

// extractTitle extracts a human-readable title from a URI.
func extractTitle(uri string) string {
	// Get filename from path
	filename := filepath.Base(uri)

	// Remove common extensions for cleaner title
	ext := filepath.Ext(filename)
	if ext != "" {
		filename = strings.TrimSuffix(filename, ext)
	}

	// Replace underscores and dashes with spaces
	filename = strings.ReplaceAll(filename, "_", " ")
	filename = strings.ReplaceAll(filename, "-", " ")

	return filename
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
