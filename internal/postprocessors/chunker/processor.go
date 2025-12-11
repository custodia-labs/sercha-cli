// Package chunker provides a fixed-size text chunking processor.
package chunker

import (
	"context"

	"github.com/google/uuid"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

// DefaultChunkSize is the default number of characters per chunk.
const DefaultChunkSize = 1000

// DefaultChunkOverlap is the default number of overlapping characters.
const DefaultChunkOverlap = 200

// Processor splits document content into fixed-size chunks.
// It implements the PostProcessor interface.
type Processor struct {
	chunkSize int
	overlap   int
}

// Option configures the chunker processor.
type Option func(*Processor)

// WithChunkSize sets the chunk size in characters.
func WithChunkSize(size int) Option {
	return func(p *Processor) {
		if size > 0 {
			p.chunkSize = size
		}
	}
}

// WithOverlap sets the overlap between chunks in characters.
func WithOverlap(overlap int) Option {
	return func(p *Processor) {
		if overlap >= 0 {
			p.overlap = overlap
		}
	}
}

// New creates a new chunker processor with the given options.
func New(opts ...Option) *Processor {
	p := &Processor{
		chunkSize: DefaultChunkSize,
		overlap:   DefaultChunkOverlap,
	}

	for _, opt := range opts {
		opt(p)
	}

	// Ensure overlap doesn't exceed chunk size
	if p.overlap >= p.chunkSize {
		p.overlap = p.chunkSize / 4
	}

	return p
}

// Name returns the processor name.
func (p *Processor) Name() string {
	return "chunker"
}

// Process splits the document content into chunks.
// Input chunks are ignored; this processor creates new chunks from document content.
func (p *Processor) Process(ctx context.Context, doc *domain.Document, _ []domain.Chunk) ([]domain.Chunk, error) {
	if doc.Content == "" {
		// Empty content produces no chunks
		return nil, nil
	}

	content := doc.Content
	contentLen := len(content)

	// Estimate number of chunks
	estimatedChunks := (contentLen / (p.chunkSize - p.overlap)) + 1
	chunks := make([]domain.Chunk, 0, estimatedChunks)

	position := 0
	start := 0

	for start < contentLen {
		end := start + p.chunkSize
		if end > contentLen {
			end = contentLen
		}

		chunkContent := content[start:end]

		chunk := domain.Chunk{
			ID:         uuid.New().String(),
			DocumentID: doc.ID,
			Content:    chunkContent,
			Position:   position,
			Metadata:   make(map[string]any),
		}

		chunks = append(chunks, chunk)
		position++

		// Move start forward by (chunkSize - overlap)
		start += p.chunkSize - p.overlap

		// Avoid infinite loop for edge cases
		if p.chunkSize <= p.overlap {
			break
		}
	}

	return chunks, nil
}
