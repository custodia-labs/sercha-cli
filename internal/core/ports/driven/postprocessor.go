package driven

import (
	"context"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

// PostProcessor processes document content to produce chunks.
// PostProcessors are chained in a pipeline (e.g., chunking, stemming, expansion).
type PostProcessor interface {
	// Name returns the processor name for logging and configuration.
	Name() string

	// Process takes a document and returns chunks.
	// If the processor modifies chunks (e.g., stemming), it receives and returns chunks.
	// If the processor creates chunks (e.g., chunker), it receives nil and returns new chunks.
	Process(ctx context.Context, doc *domain.Document, chunks []domain.Chunk) ([]domain.Chunk, error)
}

// PostProcessorPipeline chains multiple PostProcessors.
type PostProcessorPipeline interface {
	// Process runs the document through all processors in order.
	// Returns the final chunks after all processing.
	Process(ctx context.Context, doc *domain.Document) ([]domain.Chunk, error)
}
