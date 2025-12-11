package driven

import (
	"context"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

// Normaliser transforms raw documents into indexed form.
// Each normaliser handles specific MIME types (e.g., PDF, Markdown).
type Normaliser interface {
	// SupportedMIMETypes returns the MIME types this normaliser handles.
	SupportedMIMETypes() []string

	// SupportedConnectorTypes returns connector types for specialised handling.
	// Empty slice means all connectors.
	SupportedConnectorTypes() []string

	// Priority returns the selection priority (higher = preferred).
	// Connector-specific normalisers should return 90-100.
	// Generic MIME normalisers should return 50-89.
	// Fallback normalisers should return 1-9.
	Priority() int

	// Normalise transforms a raw document into a document and chunks.
	Normalise(ctx context.Context, raw *domain.RawDocument) (*NormaliseResult, error)
}

// NormaliseResult contains the output of normalisation.
// Note: Normalisation only produces a Document with Content.
// Chunking is handled by the PostProcessor pipeline.
type NormaliseResult struct {
	// Document is the normalised document with Content field populated.
	Document domain.Document
}
