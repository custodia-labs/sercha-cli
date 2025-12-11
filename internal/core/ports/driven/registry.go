package driven

import (
	"context"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

// NormaliserRegistry selects the appropriate normaliser for a document.
// It maintains a priority-ordered list of normalisers and dispatches
// based on MIME type and connector type.
type NormaliserRegistry interface {
	// Normalise transforms a raw document using the best matching normaliser.
	// Selection priority: connector-specific > MIME-specific > fallback.
	Normalise(ctx context.Context, raw *domain.RawDocument) (*NormaliseResult, error)

	// Register adds a normaliser to the registry.
	Register(normaliser Normaliser)

	// SupportedMIMETypes returns all MIME types that can be normalised.
	SupportedMIMETypes() []string
}
