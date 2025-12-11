package memory

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

func TestNewSyncStateStore(t *testing.T) {
	store := NewSyncStateStore()
	require.NotNil(t, store)
	assert.NotNil(t, store.states)
}

func TestSyncStateStore_Save_Success(t *testing.T) {
	store := NewSyncStateStore()
	ctx := context.Background()

	now := time.Now()
	state := domain.SyncState{
		SourceID: "src-1",
		Cursor:   "cursor-token-123",
		LastSync: now,
	}

	err := store.Save(ctx, state)
	require.NoError(t, err)

	// Verify it was saved
	saved, err := store.Get(ctx, "src-1")
	require.NoError(t, err)
	assert.Equal(t, "src-1", saved.SourceID)
	assert.Equal(t, "cursor-token-123", saved.Cursor)
	assert.Equal(t, now.Unix(), saved.LastSync.Unix()) // Compare Unix timestamps to avoid precision issues
}

func TestSyncStateStore_Save_Update(t *testing.T) {
	store := NewSyncStateStore()
	ctx := context.Background()

	time1 := time.Now()
	time2 := time.Now().Add(time.Hour)

	state1 := domain.SyncState{
		SourceID: "src-1",
		Cursor:   "cursor-v1",
		LastSync: time1,
	}
	state2 := domain.SyncState{
		SourceID: "src-1",
		Cursor:   "cursor-v2",
		LastSync: time2,
	}

	err := store.Save(ctx, state1)
	require.NoError(t, err)

	err = store.Save(ctx, state2)
	require.NoError(t, err)

	// Should have the updated values
	saved, err := store.Get(ctx, "src-1")
	require.NoError(t, err)
	assert.Equal(t, "cursor-v2", saved.Cursor)
	assert.Equal(t, time2.Unix(), saved.LastSync.Unix())
}

func TestSyncStateStore_Save_MultipleDistinctStates(t *testing.T) {
	store := NewSyncStateStore()
	ctx := context.Background()

	now := time.Now()
	states := []domain.SyncState{
		{SourceID: "src-1", Cursor: "cursor-1", LastSync: now},
		{SourceID: "src-2", Cursor: "cursor-2", LastSync: now.Add(time.Hour)},
		{SourceID: "src-3", Cursor: "cursor-3", LastSync: now.Add(2 * time.Hour)},
	}

	for _, state := range states {
		err := store.Save(ctx, state)
		require.NoError(t, err)
	}

	// Verify all were saved independently
	for _, state := range states {
		saved, err := store.Get(ctx, state.SourceID)
		require.NoError(t, err)
		assert.Equal(t, state.SourceID, saved.SourceID)
		assert.Equal(t, state.Cursor, saved.Cursor)
	}
}

func TestSyncStateStore_Save_WithEmptyCursor(t *testing.T) {
	store := NewSyncStateStore()
	ctx := context.Background()

	state := domain.SyncState{
		SourceID: "src-1",
		Cursor:   "", // empty cursor
		LastSync: time.Now(),
	}

	err := store.Save(ctx, state)
	require.NoError(t, err)

	saved, err := store.Get(ctx, "src-1")
	require.NoError(t, err)
	assert.Equal(t, "", saved.Cursor)
}

func TestSyncStateStore_Save_WithZeroTime(t *testing.T) {
	store := NewSyncStateStore()
	ctx := context.Background()

	state := domain.SyncState{
		SourceID: "src-1",
		Cursor:   "cursor-123",
		LastSync: time.Time{}, // zero time
	}

	err := store.Save(ctx, state)
	require.NoError(t, err)

	saved, err := store.Get(ctx, "src-1")
	require.NoError(t, err)
	assert.True(t, saved.LastSync.IsZero())
}

func TestSyncStateStore_Get_NotFound(t *testing.T) {
	store := NewSyncStateStore()
	ctx := context.Background()

	state, err := store.Get(ctx, "nonexistent")

	assert.ErrorIs(t, err, domain.ErrNotFound)
	assert.Nil(t, state)
}

func TestSyncStateStore_Get_Success(t *testing.T) {
	store := NewSyncStateStore()
	ctx := context.Background()

	now := time.Now()
	original := domain.SyncState{
		SourceID: "src-1",
		Cursor:   "complex-cursor-token-xyz",
		LastSync: now,
	}

	err := store.Save(ctx, original)
	require.NoError(t, err)

	retrieved, err := store.Get(ctx, "src-1")

	require.NoError(t, err)
	require.NotNil(t, retrieved)
	assert.Equal(t, "src-1", retrieved.SourceID)
	assert.Equal(t, "complex-cursor-token-xyz", retrieved.Cursor)
	assert.Equal(t, now.Unix(), retrieved.LastSync.Unix())
}

func TestSyncStateStore_Get_EmptySourceID(t *testing.T) {
	store := NewSyncStateStore()
	ctx := context.Background()

	state, err := store.Get(ctx, "")

	assert.ErrorIs(t, err, domain.ErrNotFound)
	assert.Nil(t, state)
}

func TestSyncStateStore_Delete_Success(t *testing.T) {
	store := NewSyncStateStore()
	ctx := context.Background()

	state := domain.SyncState{
		SourceID: "src-1",
		Cursor:   "cursor-123",
		LastSync: time.Now(),
	}

	err := store.Save(ctx, state)
	require.NoError(t, err)

	err = store.Delete(ctx, "src-1")
	require.NoError(t, err)

	// Should not be found after deletion
	_, err = store.Get(ctx, "src-1")
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestSyncStateStore_Delete_NonExistent(t *testing.T) {
	store := NewSyncStateStore()
	ctx := context.Background()

	// Delete non-existent should not error
	err := store.Delete(ctx, "nonexistent")
	assert.NoError(t, err)
}

func TestSyncStateStore_Delete_EmptySourceID(t *testing.T) {
	store := NewSyncStateStore()
	ctx := context.Background()

	// Delete with empty ID should not error
	err := store.Delete(ctx, "")
	assert.NoError(t, err)
}

func TestSyncStateStore_Delete_VerifyOthersRemain(t *testing.T) {
	store := NewSyncStateStore()
	ctx := context.Background()

	now := time.Now()
	states := []domain.SyncState{
		{SourceID: "src-1", Cursor: "cursor-1", LastSync: now},
		{SourceID: "src-2", Cursor: "cursor-2", LastSync: now},
		{SourceID: "src-3", Cursor: "cursor-3", LastSync: now},
	}

	for _, state := range states {
		_ = store.Save(ctx, state)
	}

	// Delete one
	err := store.Delete(ctx, "src-2")
	require.NoError(t, err)

	// Verify the deleted one is gone
	_, err = store.Get(ctx, "src-2")
	assert.ErrorIs(t, err, domain.ErrNotFound)

	// Verify others remain
	_, err = store.Get(ctx, "src-1")
	assert.NoError(t, err)
	_, err = store.Get(ctx, "src-3")
	assert.NoError(t, err)
}

func TestSyncStateStore_SaveAfterDelete(t *testing.T) {
	store := NewSyncStateStore()
	ctx := context.Background()

	now := time.Now()
	state := domain.SyncState{
		SourceID: "src-1",
		Cursor:   "cursor-v1",
		LastSync: now,
	}

	// Save, delete, then save again
	err := store.Save(ctx, state)
	require.NoError(t, err)

	err = store.Delete(ctx, "src-1")
	require.NoError(t, err)

	state.Cursor = "cursor-v2"
	err = store.Save(ctx, state)
	require.NoError(t, err)

	// Should have the new state
	saved, err := store.Get(ctx, "src-1")
	require.NoError(t, err)
	assert.Equal(t, "cursor-v2", saved.Cursor)
}

func TestSyncStateStore_Concurrency_SaveAndGet(t *testing.T) {
	store := NewSyncStateStore()
	ctx := context.Background()

	var wg sync.WaitGroup
	numGoroutines := 50

	// Concurrent saves
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			state := domain.SyncState{
				SourceID: "src-" + string(rune('A'+id)),
				Cursor:   "cursor-" + string(rune('A'+id)),
				LastSync: time.Now(),
			}
			_ = store.Save(ctx, state)
		}(i)
	}
	wg.Wait()

	// Concurrent reads
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			_, _ = store.Get(ctx, "src-"+string(rune('A'+id)))
		}(i)
	}
	wg.Wait()

	// Verify all saved
	for i := 0; i < numGoroutines; i++ {
		_, err := store.Get(ctx, "src-"+string(rune('A'+i)))
		assert.NoError(t, err)
	}
}

func TestSyncStateStore_Concurrency_MixedOperations(t *testing.T) {
	store := NewSyncStateStore()
	ctx := context.Background()

	var wg sync.WaitGroup
	numOperations := 100

	// Pre-populate with some data
	for i := 0; i < 10; i++ {
		_ = store.Save(ctx, domain.SyncState{
			SourceID: "src-" + string(rune('0'+i)),
			Cursor:   "cursor-" + string(rune('0'+i)),
			LastSync: time.Now(),
		})
	}

	// Run mixed concurrent operations
	wg.Add(numOperations)
	for i := 0; i < numOperations; i++ {
		go func(id int) {
			defer wg.Done()
			switch id % 3 {
			case 0: // Save
				state := domain.SyncState{
					SourceID: "src-concurrent-" + string(rune('A'+id%26)),
					Cursor:   "cursor-concurrent",
					LastSync: time.Now(),
				}
				_ = store.Save(ctx, state)
			case 1: // Get
				_, _ = store.Get(ctx, "src-"+string(rune('0'+id%10)))
			case 2: // Delete
				_ = store.Delete(ctx, "src-concurrent-"+string(rune('A'+id%26)))
			}
		}(i)
	}
	wg.Wait()

	// Should not panic or deadlock
	_, _ = store.Get(ctx, "src-0")
}

func TestSyncStateStore_Concurrency_UpdateSameState(t *testing.T) {
	store := NewSyncStateStore()
	ctx := context.Background()

	// Save initial state
	_ = store.Save(ctx, domain.SyncState{
		SourceID: "src-1",
		Cursor:   "initial",
		LastSync: time.Now(),
	})

	var wg sync.WaitGroup
	numGoroutines := 50

	// Concurrent updates to the same state
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			state := domain.SyncState{
				SourceID: "src-1",
				Cursor:   "updated-" + string(rune('A'+id)),
				LastSync: time.Now().Add(time.Duration(id) * time.Second),
			}
			_ = store.Save(ctx, state)
		}(i)
	}
	wg.Wait()

	// Verify state exists and has some update
	saved, err := store.Get(ctx, "src-1")
	require.NoError(t, err)
	assert.Equal(t, "src-1", saved.SourceID)
	assert.NotEqual(t, "initial", saved.Cursor) // Should be updated
}

func TestSyncStateStore_Concurrency_SaveDeleteCycle(t *testing.T) {
	store := NewSyncStateStore()
	ctx := context.Background()

	var wg sync.WaitGroup
	numCycles := 50

	wg.Add(numCycles * 2)
	for i := 0; i < numCycles; i++ {
		// Save
		go func(id int) {
			defer wg.Done()
			state := domain.SyncState{
				SourceID: "src-cycle",
				Cursor:   "cursor-" + string(rune('A'+id)),
				LastSync: time.Now(),
			}
			_ = store.Save(ctx, state)
		}(i)

		// Delete
		go func() {
			defer wg.Done()
			_ = store.Delete(ctx, "src-cycle")
		}()
	}
	wg.Wait()

	// Should not panic or deadlock
	// Final state is indeterminate due to race, but operation should be safe
	_, _ = store.Get(ctx, "src-cycle")
}

func TestSyncStateStore_ContextCancellation(t *testing.T) {
	store := NewSyncStateStore()

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	state := domain.SyncState{
		SourceID: "src-1",
		Cursor:   "cursor-123",
		LastSync: time.Now(),
	}

	// Operations should complete even with cancelled context
	// (memory store doesn't actually use context for cancellation)
	err := store.Save(ctx, state)
	assert.NoError(t, err)

	_, err = store.Get(ctx, "src-1")
	assert.NoError(t, err)

	err = store.Delete(ctx, "src-1")
	assert.NoError(t, err)
}

func TestSyncStateStore_DataIsolation(t *testing.T) {
	store := NewSyncStateStore()
	ctx := context.Background()

	now := time.Now()
	state := domain.SyncState{
		SourceID: "src-1",
		Cursor:   "original-cursor",
		LastSync: now,
	}

	err := store.Save(ctx, state)
	require.NoError(t, err)

	// Get the state
	retrieved, err := store.Get(ctx, "src-1")
	require.NoError(t, err)

	// Modify the retrieved copy
	retrieved.Cursor = "modified-cursor"
	retrieved.LastSync = now.Add(time.Hour)

	// Original in store should be unchanged
	original, err := store.Get(ctx, "src-1")
	require.NoError(t, err)
	assert.Equal(t, "original-cursor", original.Cursor)
	assert.Equal(t, now.Unix(), original.LastSync.Unix())
}

func TestSyncStateStore_InterfaceCompliance(t *testing.T) {
	store := NewSyncStateStore()
	ctx := context.Background()

	// Verify all interface methods work
	state := domain.SyncState{
		SourceID: "src-test",
		Cursor:   "test-cursor",
		LastSync: time.Now(),
	}

	// Save
	err := store.Save(ctx, state)
	assert.NoError(t, err)

	// Get
	_, err = store.Get(ctx, "src-test")
	assert.NoError(t, err)

	// Delete
	err = store.Delete(ctx, "src-test")
	assert.NoError(t, err)
}

func TestSyncStateStore_TimePrecision(t *testing.T) {
	store := NewSyncStateStore()
	ctx := context.Background()

	// Use a specific time with nanosecond precision
	specificTime := time.Date(2024, 1, 15, 14, 30, 45, 123456789, time.UTC)

	state := domain.SyncState{
		SourceID: "src-1",
		Cursor:   "cursor-123",
		LastSync: specificTime,
	}

	err := store.Save(ctx, state)
	require.NoError(t, err)

	retrieved, err := store.Get(ctx, "src-1")
	require.NoError(t, err)

	// Times should be equal with nanosecond precision
	assert.True(t, specificTime.Equal(retrieved.LastSync))
}

func TestSyncStateStore_LargeCursor(t *testing.T) {
	store := NewSyncStateStore()
	ctx := context.Background()

	// Test with a very large cursor string
	largeCursor := string(make([]byte, 10000))
	for i := range largeCursor {
		largeCursor = string(rune('A' + i%26))
	}

	state := domain.SyncState{
		SourceID: "src-1",
		Cursor:   largeCursor,
		LastSync: time.Now(),
	}

	err := store.Save(ctx, state)
	require.NoError(t, err)

	retrieved, err := store.Get(ctx, "src-1")
	require.NoError(t, err)
	assert.Len(t, retrieved.Cursor, len(largeCursor))
}

func TestSyncStateStore_SpecialCharactersInCursor(t *testing.T) {
	store := NewSyncStateStore()
	ctx := context.Background()

	specialCursors := []string{
		"cursor with spaces",
		"cursor\nwith\nnewlines",
		"cursor\twith\ttabs",
		"cursor/with/slashes",
		"cursor\\with\\backslashes",
		"cursor\"with\"quotes",
		"cursor'with'apostrophes",
		"cursor{with}braces",
		"cursor[with]brackets",
		"cursor<with>angles",
		"cursor|with|pipes",
		"",
	}

	for i, cursor := range specialCursors {
		sourceID := "src-" + string(rune('A'+i))
		state := domain.SyncState{
			SourceID: sourceID,
			Cursor:   cursor,
			LastSync: time.Now(),
		}

		err := store.Save(ctx, state)
		require.NoError(t, err)

		retrieved, err := store.Get(ctx, sourceID)
		require.NoError(t, err)
		assert.Equal(t, cursor, retrieved.Cursor)
	}
}
