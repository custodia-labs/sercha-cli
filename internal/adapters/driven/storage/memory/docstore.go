package memory

import (
	"context"
	"sync"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
)

// Ensure DocumentStore implements the interface.
var _ driven.DocumentStore = (*DocumentStore)(nil)

// DocumentStore is an in-memory implementation of driven.DocumentStore.
type DocumentStore struct {
	mu        sync.RWMutex
	documents map[string]domain.Document
	chunks    map[string][]domain.Chunk
}

// NewDocumentStore creates a new in-memory document store.
func NewDocumentStore() *DocumentStore {
	return &DocumentStore{
		documents: make(map[string]domain.Document),
		chunks:    make(map[string][]domain.Chunk),
	}
}

// SaveDocument stores or updates a document.
func (s *DocumentStore) SaveDocument(_ context.Context, doc *domain.Document) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.documents[doc.ID] = *doc
	return nil
}

// SaveChunks stores chunks for a document.
func (s *DocumentStore) SaveChunks(_ context.Context, chunks []domain.Chunk) error {
	if len(chunks) == 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	docID := chunks[0].DocumentID
	s.chunks[docID] = chunks
	return nil
}

// GetDocument retrieves a document by ID.
func (s *DocumentStore) GetDocument(_ context.Context, id string) (*domain.Document, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	doc, ok := s.documents[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return &doc, nil
}

// GetChunks retrieves all chunks for a document.
func (s *DocumentStore) GetChunks(_ context.Context, documentID string) ([]domain.Chunk, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	chunks, ok := s.chunks[documentID]
	if !ok {
		return nil, nil
	}
	return chunks, nil
}

// GetChunk retrieves a specific chunk by ID.
func (s *DocumentStore) GetChunk(_ context.Context, id string) (*domain.Chunk, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, chunks := range s.chunks {
		for _, chunk := range chunks {
			if chunk.ID == id {
				return &chunk, nil
			}
		}
	}
	return nil, domain.ErrNotFound
}

// DeleteDocument removes a document and its chunks.
func (s *DocumentStore) DeleteDocument(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.documents, id)
	delete(s.chunks, id)
	return nil
}

// ListDocuments returns documents for a source.
func (s *DocumentStore) ListDocuments(_ context.Context, sourceID string) ([]domain.Document, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []domain.Document
	for id := range s.documents {
		doc := s.documents[id]
		if doc.SourceID == sourceID {
			result = append(result, doc)
		}
	}
	return result, nil
}
