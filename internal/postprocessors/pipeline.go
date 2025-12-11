// Package postprocessors provides document content processing implementations.
package postprocessors

import (
	"context"
	"fmt"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
)

// Pipeline chains multiple PostProcessors and runs them in order.
// It implements the PostProcessorPipeline interface.
type Pipeline struct {
	processors []driven.PostProcessor
}

// NewPipeline creates a new processing pipeline with the given processors.
// Processors are executed in the order provided.
func NewPipeline(processors ...driven.PostProcessor) *Pipeline {
	return &Pipeline{
		processors: processors,
	}
}

// Process runs the document through all processors in order.
// The first processor receives nil chunks and should create them.
// Subsequent processors receive and may modify the chunks.
func (p *Pipeline) Process(ctx context.Context, doc *domain.Document) ([]domain.Chunk, error) {
	if doc == nil {
		return nil, fmt.Errorf("document is nil")
	}

	var chunks []domain.Chunk

	for _, processor := range p.processors {
		var err error
		chunks, err = processor.Process(ctx, doc, chunks)
		if err != nil {
			return nil, fmt.Errorf("processor %s: %w", processor.Name(), err)
		}
	}

	return chunks, nil
}

// Add appends a processor to the pipeline.
func (p *Pipeline) Add(processor driven.PostProcessor) {
	p.processors = append(p.processors, processor)
}

// Len returns the number of processors in the pipeline.
func (p *Pipeline) Len() int {
	return len(p.processors)
}
