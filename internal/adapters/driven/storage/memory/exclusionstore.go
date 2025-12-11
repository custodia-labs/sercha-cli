package memory

import (
	"context"
	"sync"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
)

// Ensure ExclusionStore implements the interface.
var _ driven.ExclusionStore = (*ExclusionStore)(nil)

// ExclusionStore is an in-memory implementation of driven.ExclusionStore.
type ExclusionStore struct {
	mu         sync.RWMutex
	exclusions map[string]domain.Exclusion
}

// NewExclusionStore creates a new in-memory exclusion store.
func NewExclusionStore() *ExclusionStore {
	return &ExclusionStore{
		exclusions: make(map[string]domain.Exclusion),
	}
}

// Add creates a new exclusion.
func (s *ExclusionStore) Add(_ context.Context, exclusion *domain.Exclusion) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.exclusions[exclusion.ID] = *exclusion
	return nil
}

// Remove deletes an exclusion by ID.
func (s *ExclusionStore) Remove(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.exclusions, id)
	return nil
}

// GetBySourceID returns all exclusions for a source.
func (s *ExclusionStore) GetBySourceID(_ context.Context, sourceID string) ([]domain.Exclusion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]domain.Exclusion, 0)
	for _, exclusion := range s.exclusions {
		if exclusion.SourceID == sourceID {
			result = append(result, exclusion)
		}
	}
	return result, nil
}

// IsExcluded checks if a URI is excluded for a source.
func (s *ExclusionStore) IsExcluded(_ context.Context, sourceID, uri string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, exclusion := range s.exclusions {
		if exclusion.SourceID == sourceID && exclusion.URI == uri {
			return true, nil
		}
	}
	return false, nil
}

// List returns all exclusions.
func (s *ExclusionStore) List(_ context.Context) ([]domain.Exclusion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]domain.Exclusion, 0, len(s.exclusions))
	for _, exclusion := range s.exclusions {
		result = append(result, exclusion)
	}
	return result, nil
}
