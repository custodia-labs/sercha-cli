//go:build !cgo

package hnsw

import (
	"context"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
)

// Ensure Index implements the interface.
var _ driven.VectorIndex = (*Index)(nil)

// Precision defines the storage precision for vectors.
// Runtime operations always use float32; this only affects disk storage.
type Precision int

const (
	// PrecisionFloat32 stores vectors at full precision (no compression).
	PrecisionFloat32 Precision = 0
	// PrecisionFloat16 stores vectors at half precision (50% storage savings).
	PrecisionFloat16 Precision = 1
	// PrecisionInt8 stores vectors at 8-bit precision (75% storage savings).
	PrecisionInt8 Precision = 2
)

// Index provides vector similarity search using HNSWlib.
// This is a stub for builds without CGO.
type Index struct {
	path      string
	dimension int
	precision Precision
}

// New creates or opens an HNSW index with the specified storage precision.
// This is a stub for builds without CGO.
func New(path string, dimension int, precision Precision) (*Index, error) {
	return &Index{
		path:      path,
		dimension: dimension,
		precision: precision,
	}, nil
}

// Add inserts a vector for the given chunk ID.
func (idx *Index) Add(_ context.Context, _ string, _ []float32) error {
	return domain.ErrNotImplemented
}

// Delete removes a vector from the index.
func (idx *Index) Delete(_ context.Context, _ string) error {
	return domain.ErrNotImplemented
}

// Search finds the k nearest neighbors to the query vector.
func (idx *Index) Search(_ context.Context, _ []float32, _ int) ([]driven.VectorHit, error) {
	return nil, domain.ErrNotImplemented
}

// Close releases resources.
func (idx *Index) Close() error {
	return nil
}
