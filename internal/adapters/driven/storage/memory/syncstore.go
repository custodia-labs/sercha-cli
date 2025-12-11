package memory

import (
	"context"
	"sync"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
)

// Ensure SyncStateStore implements the interface.
var _ driven.SyncStateStore = (*SyncStateStore)(nil)

// SyncStateStore is an in-memory implementation of driven.SyncStateStore.
type SyncStateStore struct {
	mu     sync.RWMutex
	states map[string]domain.SyncState
}

// NewSyncStateStore creates a new in-memory sync state store.
func NewSyncStateStore() *SyncStateStore {
	return &SyncStateStore{
		states: make(map[string]domain.SyncState),
	}
}

// Save stores or updates sync state.
func (s *SyncStateStore) Save(_ context.Context, state domain.SyncState) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.states[state.SourceID] = state
	return nil
}

// Get retrieves sync state for a source.
func (s *SyncStateStore) Get(_ context.Context, sourceID string) (*domain.SyncState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, ok := s.states[sourceID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return &state, nil
}

// Delete removes sync state for a source.
func (s *SyncStateStore) Delete(_ context.Context, sourceID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.states, sourceID)
	return nil
}
