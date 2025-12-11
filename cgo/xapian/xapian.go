//go:build cgo

package xapian

/*
#cgo pkg-config: xapian-core
#cgo CXXFLAGS: -std=c++17

#include "xapian_wrapper.h"
#include <stdlib.h>
*/
import "C"

import (
	"context"
	"errors"
	"sync"
	"unsafe"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
)

// Ensure Engine implements the interface.
var _ driven.SearchEngine = (*Engine)(nil)

// Engine provides full-text search using Xapian.
type Engine struct {
	mu   sync.RWMutex
	db   C.xapian_db
	path string
}

// New creates a new Xapian search engine.
func New(path string) (*Engine, error) {
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))

	db := C.xapian_open(cpath)
	if db == nil {
		errMsg := C.GoString(C.xapian_get_error())
		return nil, errors.New("xapian: failed to open database: " + errMsg)
	}

	return &Engine{
		db:   db,
		path: path,
	}, nil
}

// Index adds or updates a chunk in the search index.
func (e *Engine) Index(_ context.Context, chunk domain.Chunk) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.db == nil {
		return errors.New("xapian: database is closed")
	}

	cChunkID := C.CString(chunk.ID)
	defer C.free(unsafe.Pointer(cChunkID))

	cDocID := C.CString(chunk.DocumentID)
	defer C.free(unsafe.Pointer(cDocID))

	cContent := C.CString(chunk.Content)
	defer C.free(unsafe.Pointer(cContent))

	result := C.xapian_index(e.db, cChunkID, cDocID, cContent)
	if result != 0 {
		errMsg := C.GoString(C.xapian_get_error())
		return errors.New("xapian: failed to index chunk: " + errMsg)
	}

	return nil
}

// Delete removes a chunk from the search index.
func (e *Engine) Delete(_ context.Context, chunkID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.db == nil {
		return errors.New("xapian: database is closed")
	}

	cChunkID := C.CString(chunkID)
	defer C.free(unsafe.Pointer(cChunkID))

	result := C.xapian_delete(e.db, cChunkID)
	if result != 0 {
		errMsg := C.GoString(C.xapian_get_error())
		return errors.New("xapian: failed to delete chunk: " + errMsg)
	}

	return nil
}

// Search performs a keyword search and returns matching chunk IDs with scores.
func (e *Engine) Search(_ context.Context, query string, limit int) ([]driven.SearchHit, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if e.db == nil {
		return nil, errors.New("xapian: database is closed")
	}

	cQuery := C.CString(query)
	defer C.free(unsafe.Pointer(cQuery))

	results := C.xapian_search(e.db, cQuery, C.int(limit))
	defer C.xapian_free_results(results)

	if results.results == nil {
		// Check if there was an error or just no results
		errMsg := C.GoString(C.xapian_get_error())
		if errMsg != "" {
			return nil, errors.New("xapian: search failed: " + errMsg)
		}
		return nil, nil // No results, but no error
	}

	// Convert C results to Go slice
	hits := make([]driven.SearchHit, int(results.count))

	// Get slice of C results
	cResults := unsafe.Slice(results.results, int(results.count))

	for i := 0; i < int(results.count); i++ {
		hits[i] = driven.SearchHit{
			ChunkID: C.GoString(cResults[i].chunk_id),
			Score:   float64(cResults[i].score),
		}
	}

	return hits, nil
}

// Close releases resources.
func (e *Engine) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.db != nil {
		C.xapian_close(e.db)
		e.db = nil
	}

	return nil
}
