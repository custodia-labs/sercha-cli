package memory

import (
	"context"
	"sync"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
)

// Ensure SourceStore implements the interface.
var _ driven.SourceStore = (*SourceStore)(nil)

// SourceStore is an in-memory implementation of driven.SourceStore.
type SourceStore struct {
	mu      sync.RWMutex
	sources map[string]domain.Source
}

// NewSourceStore creates a new in-memory source store.
func NewSourceStore() *SourceStore {
	return &SourceStore{
		sources: make(map[string]domain.Source),
	}
}

// Save stores or updates a source.
func (s *SourceStore) Save(_ context.Context, source domain.Source) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sources[source.ID] = source
	return nil
}

// Get retrieves a source by ID.
func (s *SourceStore) Get(_ context.Context, id string) (*domain.Source, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	source, ok := s.sources[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return &source, nil
}

// Delete removes a source.
func (s *SourceStore) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sources, id)
	return nil
}

// List returns all configured sources.
func (s *SourceStore) List(_ context.Context) ([]domain.Source, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]domain.Source, 0, len(s.sources))
	for _, source := range s.sources {
		result = append(result, source)
	}
	return result, nil
}
