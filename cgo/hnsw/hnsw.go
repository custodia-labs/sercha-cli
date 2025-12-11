//go:build cgo

package hnsw

/*
#cgo CXXFLAGS: -std=c++17 -O3 -I${SRCDIR}/../../clib/build/_deps/hnswlib-src
#cgo LDFLAGS: -lstdc++

#include "hnsw_wrapper.h"
#include <stdlib.h>
*/
import "C"

import (
	"context"
	"errors"
	"sync"
	"unsafe"

	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
)

// Ensure Index implements the interface.
var _ driven.VectorIndex = (*Index)(nil)

// Default configuration values
const (
	DefaultMaxElements = 100000
)

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
type Index struct {
	mu        sync.RWMutex
	idx       *C.HnswIndex
	path      string
	dimension int
	precision Precision
}

// New creates or opens an HNSW index with the specified storage precision.
// The precision parameter only affects disk storage; runtime always uses float32.
func New(path string, dimension int, precision Precision) (*Index, error) {
	if path == "" {
		return nil, errors.New("hnsw: path cannot be empty")
	}
	if dimension <= 0 {
		return nil, errors.New("hnsw: dimension must be positive")
	}

	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))

	// Try to open existing index first
	idx := C.hnsw_open(cpath, C.int(dimension))
	if idx == nil {
		// Create new index with specified precision
		idx = C.hnsw_create(cpath, C.int(dimension), C.int(DefaultMaxElements), C.HnswPrecision(precision))
		if idx == nil {
			return nil, errors.New("hnsw: failed to create index")
		}
	}

	return &Index{
		idx:       idx,
		path:      path,
		dimension: dimension,
		precision: precision,
	}, nil
}

// Add inserts a vector for the given chunk ID.
func (idx *Index) Add(_ context.Context, chunkID string, embedding []float32) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	if idx.idx == nil {
		return errors.New("hnsw: index is closed")
	}

	if len(embedding) != idx.dimension {
		return errors.New("hnsw: embedding dimension mismatch")
	}

	cChunkID := C.CString(chunkID)
	defer C.free(unsafe.Pointer(cChunkID))

	result := C.hnsw_add(
		idx.idx,
		cChunkID,
		(*C.float)(unsafe.Pointer(&embedding[0])),
		C.int(idx.dimension),
	)

	if result != 0 {
		return errors.New("hnsw: failed to add vector")
	}

	return nil
}

// Delete removes a vector from the index.
func (idx *Index) Delete(_ context.Context, chunkID string) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	if idx.idx == nil {
		return errors.New("hnsw: index is closed")
	}

	cChunkID := C.CString(chunkID)
	defer C.free(unsafe.Pointer(cChunkID))

	result := C.hnsw_delete(idx.idx, cChunkID)
	if result != 0 {
		return errors.New("hnsw: failed to delete vector")
	}

	return nil
}

// Search finds the k nearest neighbours to the query vector.
func (idx *Index) Search(_ context.Context, query []float32, k int) ([]driven.VectorHit, error) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	if idx.idx == nil {
		return nil, errors.New("hnsw: index is closed")
	}

	if len(query) != idx.dimension {
		return nil, errors.New("hnsw: query dimension mismatch")
	}

	if k <= 0 {
		return nil, nil
	}

	var results *C.HnswSearchResult
	count := C.hnsw_search(
		idx.idx,
		(*C.float)(unsafe.Pointer(&query[0])),
		C.int(idx.dimension),
		C.int(k),
		&results,
	)

	if count < 0 {
		return nil, errors.New("hnsw: search failed")
	}

	if count == 0 || results == nil {
		return nil, nil
	}

	defer C.hnsw_free_results(results, count)

	// Convert C results to Go slice
	hits := make([]driven.VectorHit, int(count))
	cResults := unsafe.Slice(results, int(count))

	for i := 0; i < int(count); i++ {
		hits[i] = driven.VectorHit{
			ChunkID:    C.GoString(cResults[i].chunk_id),
			Similarity: float64(cResults[i].similarity),
		}
	}

	return hits, nil
}

// Close releases resources.
func (idx *Index) Close() error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	if idx.idx != nil {
		C.hnsw_close(idx.idx)
		idx.idx = nil
	}

	return nil
}
