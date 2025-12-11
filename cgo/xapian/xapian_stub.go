//go:build !cgo

package xapian

import (
	"context"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
)

// Ensure Engine implements the interface.
var _ driven.SearchEngine = (*Engine)(nil)

// Engine provides full-text search using Xapian.
// This is a stub for builds without CGO.
type Engine struct {
	path string
}

// New creates a new Xapian search engine.
func New(path string) (*Engine, error) {
	return &Engine{
		path: path,
	}, nil
}

// Index adds or updates a chunk in the search index.
func (e *Engine) Index(_ context.Context, _ domain.Chunk) error {
	return domain.ErrNotImplemented
}

// Delete removes a chunk from the search index.
func (e *Engine) Delete(_ context.Context, _ string) error {
	return domain.ErrNotImplemented
}

// Search performs a keyword search and returns matching chunk IDs with scores.
func (e *Engine) Search(_ context.Context, _ string, _ int) ([]driven.SearchHit, error) {
	return nil, domain.ErrNotImplemented
}

// Close releases resources.
func (e *Engine) Close() error {
	return nil
}
