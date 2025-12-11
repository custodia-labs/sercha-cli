package memory

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

func TestNewSourceStore(t *testing.T) {
	store := NewSourceStore()
	require.NotNil(t, store)
	assert.NotNil(t, store.sources)
}

func TestSourceStore_Save_Success(t *testing.T) {
	store := NewSourceStore()
	ctx := context.Background()

	source := domain.Source{
		ID:              "src-1",
		Type:            "filesystem",
		Name:            "My Documents",
		Config:          map[string]string{"path": "/home/user/docs"},
		AuthorizationID: "auth-1",
	}

	err := store.Save(ctx, source)
	require.NoError(t, err)

	// Verify it was saved
	saved, err := store.Get(ctx, "src-1")
	require.NoError(t, err)
	assert.Equal(t, "src-1", saved.ID)
	assert.Equal(t, "filesystem", saved.Type)
	assert.Equal(t, "My Documents", saved.Name)
	assert.Equal(t, "auth-1", saved.AuthorizationID)
	assert.Equal(t, "/home/user/docs", saved.Config["path"])
}

func TestSourceStore_Save_Update(t *testing.T) {
	store := NewSourceStore()
	ctx := context.Background()

	source1 := domain.Source{
		ID:   "src-1",
		Name: "Original Name",
		Type: "filesystem",
	}
	source2 := domain.Source{
		ID:   "src-1",
		Name: "Updated Name",
		Type: "gmail",
	}

	err := store.Save(ctx, source1)
	require.NoError(t, err)

	err = store.Save(ctx, source2)
	require.NoError(t, err)

	// Should have the updated values
	saved, err := store.Get(ctx, "src-1")
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", saved.Name)
	assert.Equal(t, "gmail", saved.Type)
}

func TestSourceStore_Save_MultipleDistinctSources(t *testing.T) {
	store := NewSourceStore()
	ctx := context.Background()

	sources := []domain.Source{
		{ID: "src-1", Name: "Source 1", Type: "filesystem"},
		{ID: "src-2", Name: "Source 2", Type: "gmail"},
		{ID: "src-3", Name: "Source 3", Type: "slack"},
	}

	for _, source := range sources {
		err := store.Save(ctx, source)
		require.NoError(t, err)
	}

	// Verify all were saved
	list, err := store.List(ctx)
	require.NoError(t, err)
	assert.Len(t, list, 3)
}

func TestSourceStore_Save_WithNilConfig(t *testing.T) {
	store := NewSourceStore()
	ctx := context.Background()

	source := domain.Source{
		ID:     "src-1",
		Name:   "Test Source",
		Type:   "filesystem",
		Config: nil, // nil config should be handled
	}

	err := store.Save(ctx, source)
	require.NoError(t, err)

	saved, err := store.Get(ctx, "src-1")
	require.NoError(t, err)
	assert.Nil(t, saved.Config)
}

func TestSourceStore_Save_WithEmptyConfig(t *testing.T) {
	store := NewSourceStore()
	ctx := context.Background()

	source := domain.Source{
		ID:     "src-1",
		Name:   "Test Source",
		Type:   "filesystem",
		Config: map[string]string{}, // empty config
	}

	err := store.Save(ctx, source)
	require.NoError(t, err)

	saved, err := store.Get(ctx, "src-1")
	require.NoError(t, err)
	assert.NotNil(t, saved.Config)
	assert.Empty(t, saved.Config)
}

func TestSourceStore_Get_NotFound(t *testing.T) {
	store := NewSourceStore()
	ctx := context.Background()

	source, err := store.Get(ctx, "nonexistent")

	assert.ErrorIs(t, err, domain.ErrNotFound)
	assert.Nil(t, source)
}

func TestSourceStore_Get_Success(t *testing.T) {
	store := NewSourceStore()
	ctx := context.Background()

	original := domain.Source{
		ID:              "src-1",
		Type:            "gmail",
		Name:            "Work Email",
		Config:          map[string]string{"email": "user@example.com"},
		AuthorizationID: "auth-oauth-1",
	}

	err := store.Save(ctx, original)
	require.NoError(t, err)

	retrieved, err := store.Get(ctx, "src-1")

	require.NoError(t, err)
	require.NotNil(t, retrieved)
	assert.Equal(t, "src-1", retrieved.ID)
	assert.Equal(t, "gmail", retrieved.Type)
	assert.Equal(t, "Work Email", retrieved.Name)
	assert.Equal(t, "auth-oauth-1", retrieved.AuthorizationID)
	assert.Equal(t, "user@example.com", retrieved.Config["email"])
}

func TestSourceStore_Get_EmptyID(t *testing.T) {
	store := NewSourceStore()
	ctx := context.Background()

	source, err := store.Get(ctx, "")

	assert.ErrorIs(t, err, domain.ErrNotFound)
	assert.Nil(t, source)
}

func TestSourceStore_Delete_Success(t *testing.T) {
	store := NewSourceStore()
	ctx := context.Background()

	source := domain.Source{
		ID:   "src-1",
		Name: "Test Source",
		Type: "filesystem",
	}

	err := store.Save(ctx, source)
	require.NoError(t, err)

	err = store.Delete(ctx, "src-1")
	require.NoError(t, err)

	// Should not be found after deletion
	_, err = store.Get(ctx, "src-1")
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestSourceStore_Delete_NonExistent(t *testing.T) {
	store := NewSourceStore()
	ctx := context.Background()

	// Delete non-existent should not error
	err := store.Delete(ctx, "nonexistent")
	assert.NoError(t, err)
}

func TestSourceStore_Delete_EmptyID(t *testing.T) {
	store := NewSourceStore()
	ctx := context.Background()

	// Delete with empty ID should not error
	err := store.Delete(ctx, "")
	assert.NoError(t, err)
}

func TestSourceStore_Delete_VerifyOthersRemain(t *testing.T) {
	store := NewSourceStore()
	ctx := context.Background()

	sources := []domain.Source{
		{ID: "src-1", Name: "Source 1", Type: "filesystem"},
		{ID: "src-2", Name: "Source 2", Type: "gmail"},
		{ID: "src-3", Name: "Source 3", Type: "slack"},
	}

	for _, source := range sources {
		_ = store.Save(ctx, source)
	}

	// Delete one
	err := store.Delete(ctx, "src-2")
	require.NoError(t, err)

	// Verify the deleted one is gone
	_, err = store.Get(ctx, "src-2")
	assert.ErrorIs(t, err, domain.ErrNotFound)

	// Verify others remain
	remaining, err := store.List(ctx)
	require.NoError(t, err)
	assert.Len(t, remaining, 2)

	ids := make(map[string]bool)
	for _, s := range remaining {
		ids[s.ID] = true
	}
	assert.True(t, ids["src-1"])
	assert.False(t, ids["src-2"])
	assert.True(t, ids["src-3"])
}

func TestSourceStore_List_Empty(t *testing.T) {
	store := NewSourceStore()
	ctx := context.Background()

	sources, err := store.List(ctx)

	require.NoError(t, err)
	assert.Empty(t, sources)
	assert.NotNil(t, sources) // Should be empty slice, not nil
}

func TestSourceStore_List_WithItems(t *testing.T) {
	store := NewSourceStore()
	ctx := context.Background()

	sources := []domain.Source{
		{ID: "src-1", Name: "Source 1", Type: "filesystem"},
		{ID: "src-2", Name: "Source 2", Type: "gmail"},
		{ID: "src-3", Name: "Source 3", Type: "slack"},
	}

	for _, source := range sources {
		_ = store.Save(ctx, source)
	}

	list, err := store.List(ctx)

	require.NoError(t, err)
	assert.Len(t, list, 3)

	// Verify all items are present
	ids := make(map[string]bool)
	for _, s := range list {
		ids[s.ID] = true
	}
	assert.True(t, ids["src-1"])
	assert.True(t, ids["src-2"])
	assert.True(t, ids["src-3"])
}

func TestSourceStore_List_AfterDeleteAll(t *testing.T) {
	store := NewSourceStore()
	ctx := context.Background()

	// Add some sources
	_ = store.Save(ctx, domain.Source{ID: "src-1"})
	_ = store.Save(ctx, domain.Source{ID: "src-2"})

	// Delete all
	_ = store.Delete(ctx, "src-1")
	_ = store.Delete(ctx, "src-2")

	// List should be empty
	list, err := store.List(ctx)
	require.NoError(t, err)
	assert.Empty(t, list)
}

func TestSourceStore_Concurrency_SaveAndGet(t *testing.T) {
	store := NewSourceStore()
	ctx := context.Background()

	var wg sync.WaitGroup
	numGoroutines := 50

	// Concurrent saves
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			source := domain.Source{
				ID:   "src-" + string(rune('A'+id)),
				Name: "Source " + string(rune('A'+id)),
				Type: "filesystem",
			}
			_ = store.Save(ctx, source)
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
	list, err := store.List(ctx)
	require.NoError(t, err)
	assert.Len(t, list, numGoroutines)
}

func TestSourceStore_Concurrency_MixedOperations(t *testing.T) {
	store := NewSourceStore()
	ctx := context.Background()

	var wg sync.WaitGroup
	numOperations := 100

	// Pre-populate with some data
	for i := 0; i < 10; i++ {
		_ = store.Save(ctx, domain.Source{
			ID:   "src-" + string(rune('0'+i)),
			Name: "Source " + string(rune('0'+i)),
		})
	}

	// Run mixed concurrent operations
	wg.Add(numOperations)
	for i := 0; i < numOperations; i++ {
		go func(id int) {
			defer wg.Done()
			switch id % 4 {
			case 0: // Save
				source := domain.Source{
					ID:   "src-concurrent-" + string(rune('A'+id%26)),
					Name: "Concurrent Source",
				}
				_ = store.Save(ctx, source)
			case 1: // Get
				_, _ = store.Get(ctx, "src-"+string(rune('0'+id%10)))
			case 2: // List
				_, _ = store.List(ctx)
			case 3: // Delete
				_ = store.Delete(ctx, "src-concurrent-"+string(rune('A'+id%26)))
			}
		}(i)
	}
	wg.Wait()

	// Should not panic or deadlock
	list, err := store.List(ctx)
	require.NoError(t, err)
	assert.NotNil(t, list)
}

func TestSourceStore_Concurrency_UpdateSameSource(t *testing.T) {
	store := NewSourceStore()
	ctx := context.Background()

	// Save initial source
	_ = store.Save(ctx, domain.Source{
		ID:   "src-1",
		Name: "Original",
	})

	var wg sync.WaitGroup
	numGoroutines := 50

	// Concurrent updates to the same source
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			source := domain.Source{
				ID:   "src-1",
				Name: "Updated " + string(rune('A'+id)),
				Type: "type-" + string(rune('A'+id)),
			}
			_ = store.Save(ctx, source)
		}(i)
	}
	wg.Wait()

	// Verify source exists and has some update
	saved, err := store.Get(ctx, "src-1")
	require.NoError(t, err)
	assert.Equal(t, "src-1", saved.ID)
	assert.NotEqual(t, "Original", saved.Name) // Should be updated
}

func TestSourceStore_Concurrency_SaveDeleteCycle(t *testing.T) {
	store := NewSourceStore()
	ctx := context.Background()

	var wg sync.WaitGroup
	numCycles := 50

	wg.Add(numCycles * 2)
	for i := 0; i < numCycles; i++ {
		// Save
		go func(id int) {
			defer wg.Done()
			source := domain.Source{
				ID:   "src-cycle",
				Name: "Cycle " + string(rune('A'+id)),
			}
			_ = store.Save(ctx, source)
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

func TestSourceStore_ContextCancellation(t *testing.T) {
	store := NewSourceStore()

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	source := domain.Source{
		ID:   "src-1",
		Name: "Test",
	}

	// Operations should complete even with cancelled context
	// (memory store doesn't actually use context for cancellation)
	err := store.Save(ctx, source)
	assert.NoError(t, err)

	_, err = store.Get(ctx, "src-1")
	assert.NoError(t, err)

	_, err = store.List(ctx)
	assert.NoError(t, err)

	err = store.Delete(ctx, "src-1")
	assert.NoError(t, err)
}

func TestSourceStore_DataIsolation(t *testing.T) {
	store := NewSourceStore()
	ctx := context.Background()

	source := domain.Source{
		ID:     "src-1",
		Name:   "Original",
		Type:   "filesystem",
		Config: map[string]string{"key": "value"},
	}

	err := store.Save(ctx, source)
	require.NoError(t, err)

	// Get the source
	retrieved, err := store.Get(ctx, "src-1")
	require.NoError(t, err)

	// Modify the retrieved copy - Name is a value type so it's safe
	retrieved.Name = "Modified"
	// Config is a map (reference type), modifying it would affect the stored copy
	// This is a known limitation of the memory store

	// Verify Name change doesn't affect stored copy (value type)
	original, err := store.Get(ctx, "src-1")
	require.NoError(t, err)
	assert.Equal(t, "Original", original.Name)

	// Note: Config map is shared (reference type), so modifications would be visible
	// In practice, callers should not modify retrieved values
}

func TestSourceStore_InterfaceCompliance(t *testing.T) {
	store := NewSourceStore()
	ctx := context.Background()

	// Verify all interface methods work
	source := domain.Source{
		ID:              "src-test",
		Type:            "test",
		Name:            "Test Source",
		Config:          map[string]string{"test": "value"},
		AuthorizationID: "auth-test",
	}

	// Save
	err := store.Save(ctx, source)
	assert.NoError(t, err)

	// Get
	_, err = store.Get(ctx, "src-test")
	assert.NoError(t, err)

	// List
	_, err = store.List(ctx)
	assert.NoError(t, err)

	// Delete
	err = store.Delete(ctx, "src-test")
	assert.NoError(t, err)
}
