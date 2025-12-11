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

func TestNewExclusionStore(t *testing.T) {
	store := NewExclusionStore()
	require.NotNil(t, store)
}

func TestExclusionStore_Add(t *testing.T) {
	store := NewExclusionStore()
	ctx := context.Background()

	exclusion := domain.Exclusion{
		ID:         "excl-1",
		SourceID:   "src-1",
		DocumentID: "doc-1",
		URI:        "/path/to/file.txt",
		Reason:     "user excluded",
		ExcludedAt: time.Now(),
	}

	err := store.Add(ctx, &exclusion)
	assert.NoError(t, err)

	// Verify it's stored
	exclusions, err := store.List(ctx)
	require.NoError(t, err)
	assert.Len(t, exclusions, 1)
	assert.Equal(t, "excl-1", exclusions[0].ID)
}

func TestExclusionStore_Remove(t *testing.T) {
	store := NewExclusionStore()
	ctx := context.Background()

	exclusion := domain.Exclusion{
		ID:       "excl-1",
		SourceID: "src-1",
	}
	_ = store.Add(ctx, &exclusion)

	err := store.Remove(ctx, "excl-1")
	assert.NoError(t, err)

	exclusions, _ := store.List(ctx)
	assert.Len(t, exclusions, 0)
}

func TestExclusionStore_GetBySourceID(t *testing.T) {
	store := NewExclusionStore()
	ctx := context.Background()

	// Add exclusions for different sources
	_ = store.Add(ctx, &domain.Exclusion{ID: "excl-1", SourceID: "src-1", URI: "/a"})
	_ = store.Add(ctx, &domain.Exclusion{ID: "excl-2", SourceID: "src-1", URI: "/b"})
	_ = store.Add(ctx, &domain.Exclusion{ID: "excl-3", SourceID: "src-2", URI: "/c"})

	exclusions, err := store.GetBySourceID(ctx, "src-1")
	require.NoError(t, err)
	assert.Len(t, exclusions, 2)
}

func TestExclusionStore_IsExcluded(t *testing.T) {
	store := NewExclusionStore()
	ctx := context.Background()

	_ = store.Add(ctx, &domain.Exclusion{ID: "excl-1", SourceID: "src-1", URI: "/path/to/file.txt"})

	// Test excluded URI
	excluded, err := store.IsExcluded(ctx, "src-1", "/path/to/file.txt")
	require.NoError(t, err)
	assert.True(t, excluded)

	// Test non-excluded URI
	excluded, err = store.IsExcluded(ctx, "src-1", "/other/file.txt")
	require.NoError(t, err)
	assert.False(t, excluded)

	// Test different source
	excluded, err = store.IsExcluded(ctx, "src-2", "/path/to/file.txt")
	require.NoError(t, err)
	assert.False(t, excluded)
}

func TestExclusionStore_List(t *testing.T) {
	store := NewExclusionStore()
	ctx := context.Background()

	_ = store.Add(ctx, &domain.Exclusion{ID: "excl-1"})
	_ = store.Add(ctx, &domain.Exclusion{ID: "excl-2"})

	exclusions, err := store.List(ctx)
	require.NoError(t, err)
	assert.Len(t, exclusions, 2)
}

func TestExclusionStore_List_Empty(t *testing.T) {
	store := NewExclusionStore()
	ctx := context.Background()

	exclusions, err := store.List(ctx)
	require.NoError(t, err)
	assert.Empty(t, exclusions)
	assert.NotNil(t, exclusions)
}

func TestExclusionStore_GetBySourceID_Empty(t *testing.T) {
	store := NewExclusionStore()
	ctx := context.Background()

	exclusions, err := store.GetBySourceID(ctx, "src-1")
	require.NoError(t, err)
	assert.Empty(t, exclusions)
	assert.NotNil(t, exclusions)
}

func TestExclusionStore_GetBySourceID_NoMatches(t *testing.T) {
	store := NewExclusionStore()
	ctx := context.Background()

	_ = store.Add(ctx, &domain.Exclusion{ID: "excl-1", SourceID: "src-1"})
	_ = store.Add(ctx, &domain.Exclusion{ID: "excl-2", SourceID: "src-2"})

	exclusions, err := store.GetBySourceID(ctx, "src-nonexistent")
	require.NoError(t, err)
	assert.Empty(t, exclusions)
}

func TestExclusionStore_IsExcluded_EmptyStore(t *testing.T) {
	store := NewExclusionStore()
	ctx := context.Background()

	excluded, err := store.IsExcluded(ctx, "src-1", "/any/path")
	require.NoError(t, err)
	assert.False(t, excluded)
}

func TestExclusionStore_IsExcluded_EmptySourceID(t *testing.T) {
	store := NewExclusionStore()
	ctx := context.Background()

	_ = store.Add(ctx, &domain.Exclusion{ID: "excl-1", SourceID: "src-1", URI: "/path"})

	excluded, err := store.IsExcluded(ctx, "", "/path")
	require.NoError(t, err)
	assert.False(t, excluded)
}

func TestExclusionStore_IsExcluded_EmptyURI(t *testing.T) {
	store := NewExclusionStore()
	ctx := context.Background()

	_ = store.Add(ctx, &domain.Exclusion{ID: "excl-1", SourceID: "src-1", URI: ""})

	excluded, err := store.IsExcluded(ctx, "src-1", "")
	require.NoError(t, err)
	assert.True(t, excluded)
}

func TestExclusionStore_Remove_NonExistent(t *testing.T) {
	store := NewExclusionStore()
	ctx := context.Background()

	// Remove non-existent should not error
	err := store.Remove(ctx, "nonexistent")
	assert.NoError(t, err)
}

func TestExclusionStore_Remove_EmptyID(t *testing.T) {
	store := NewExclusionStore()
	ctx := context.Background()

	// Remove with empty ID should not error
	err := store.Remove(ctx, "")
	assert.NoError(t, err)
}

func TestExclusionStore_Add_Update(t *testing.T) {
	store := NewExclusionStore()
	ctx := context.Background()

	exclusion1 := domain.Exclusion{
		ID:       "excl-1",
		SourceID: "src-1",
		URI:      "/original/path",
		Reason:   "original reason",
	}
	exclusion2 := domain.Exclusion{
		ID:       "excl-1",
		SourceID: "src-1",
		URI:      "/updated/path",
		Reason:   "updated reason",
	}

	err := store.Add(ctx, &exclusion1)
	require.NoError(t, err)

	err = store.Add(ctx, &exclusion2)
	require.NoError(t, err)

	// Should have the updated values
	exclusions, err := store.List(ctx)
	require.NoError(t, err)
	assert.Len(t, exclusions, 1)
	assert.Equal(t, "/updated/path", exclusions[0].URI)
	assert.Equal(t, "updated reason", exclusions[0].Reason)
}

func TestExclusionStore_Add_WithDocumentID(t *testing.T) {
	store := NewExclusionStore()
	ctx := context.Background()

	exclusion := domain.Exclusion{
		ID:         "excl-1",
		SourceID:   "src-1",
		DocumentID: "doc-1",
		URI:        "/path/to/doc",
		Reason:     "sensitive content",
		ExcludedAt: time.Now(),
	}

	err := store.Add(ctx, &exclusion)
	require.NoError(t, err)

	exclusions, err := store.List(ctx)
	require.NoError(t, err)
	assert.Len(t, exclusions, 1)
	assert.Equal(t, "doc-1", exclusions[0].DocumentID)
}

func TestExclusionStore_MultipleExclusionsForSameURI(t *testing.T) {
	store := NewExclusionStore()
	ctx := context.Background()

	// Different sources can exclude the same URI
	_ = store.Add(ctx, &domain.Exclusion{ID: "excl-1", SourceID: "src-1", URI: "/path"})
	_ = store.Add(ctx, &domain.Exclusion{ID: "excl-2", SourceID: "src-2", URI: "/path"})

	// Both should be excluded for their respective sources
	excluded1, err := store.IsExcluded(ctx, "src-1", "/path")
	require.NoError(t, err)
	assert.True(t, excluded1)

	excluded2, err := store.IsExcluded(ctx, "src-2", "/path")
	require.NoError(t, err)
	assert.True(t, excluded2)
}

func TestExclusionStore_SpecialCharactersInURI(t *testing.T) {
	store := NewExclusionStore()
	ctx := context.Background()

	specialURIs := []string{
		"/path with spaces/file.txt",
		"/path/with/unicode/文件.txt",
		"/path/with/special/chars!@#$%^&*()",
		"/path/with/quotes/\"file\".txt",
		"/path/with/backslash\\file.txt",
	}

	for i, uri := range specialURIs {
		exclusion := domain.Exclusion{
			ID:       "excl-" + string(rune('A'+i)),
			SourceID: "src-1",
			URI:      uri,
		}
		err := store.Add(ctx, &exclusion)
		require.NoError(t, err)
	}

	// Verify all can be checked
	for _, uri := range specialURIs {
		excluded, err := store.IsExcluded(ctx, "src-1", uri)
		require.NoError(t, err)
		assert.True(t, excluded)
	}
}

func TestExclusionStore_Concurrency_AddAndList(t *testing.T) {
	store := NewExclusionStore()
	ctx := context.Background()

	var wg sync.WaitGroup
	numGoroutines := 50

	// Concurrent adds
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			exclusion := domain.Exclusion{
				ID:         "excl-" + string(rune('A'+id)),
				SourceID:   "src-1",
				URI:        "/path/" + string(rune('A'+id)),
				Reason:     "test",
				ExcludedAt: time.Now(),
			}
			_ = store.Add(ctx, &exclusion)
		}(i)
	}
	wg.Wait()

	// Concurrent reads
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			_, _ = store.List(ctx)
		}()
	}
	wg.Wait()

	// Verify all were added
	exclusions, err := store.List(ctx)
	require.NoError(t, err)
	assert.Len(t, exclusions, numGoroutines)
}

func TestExclusionStore_Concurrency_MixedOperations(t *testing.T) {
	store := NewExclusionStore()
	ctx := context.Background()

	var wg sync.WaitGroup
	numOperations := 100

	// Pre-populate
	for i := 0; i < 10; i++ {
		_ = store.Add(ctx, &domain.Exclusion{
			ID:       "excl-" + string(rune('0'+i)),
			SourceID: "src-1",
			URI:      "/path/" + string(rune('0'+i)),
		})
	}

	// Run mixed concurrent operations
	wg.Add(numOperations)
	for i := 0; i < numOperations; i++ {
		go func(id int) {
			defer wg.Done()
			switch id % 5 {
			case 0: // Add
				exclusion := domain.Exclusion{
					ID:       "excl-concurrent-" + string(rune('A'+id%26)),
					SourceID: "src-" + string(rune('1'+id%3)),
					URI:      "/concurrent/path",
				}
				_ = store.Add(ctx, &exclusion)
			case 1: // Remove
				_ = store.Remove(ctx, "excl-"+string(rune('0'+id%10)))
			case 2: // List
				_, _ = store.List(ctx)
			case 3: // GetBySourceID
				_, _ = store.GetBySourceID(ctx, "src-1")
			case 4: // IsExcluded
				_, _ = store.IsExcluded(ctx, "src-1", "/path/"+string(rune('0'+id%10)))
			}
		}(i)
	}
	wg.Wait()

	// Should not panic or deadlock
	_, _ = store.List(ctx)
}

func TestExclusionStore_Concurrency_UpdateSameExclusion(t *testing.T) {
	store := NewExclusionStore()
	ctx := context.Background()

	// Add initial exclusion
	_ = store.Add(ctx, &domain.Exclusion{
		ID:       "excl-1",
		SourceID: "src-1",
		URI:      "/original",
	})

	var wg sync.WaitGroup
	numGoroutines := 50

	// Concurrent updates to the same exclusion
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			exclusion := domain.Exclusion{
				ID:       "excl-1",
				SourceID: "src-1",
				URI:      "/updated-" + string(rune('A'+id)),
				Reason:   "updated reason " + string(rune('A'+id)),
			}
			_ = store.Add(ctx, &exclusion)
		}(i)
	}
	wg.Wait()

	// Verify exclusion exists and has some update
	exclusions, err := store.List(ctx)
	require.NoError(t, err)
	assert.Len(t, exclusions, 1)
	assert.Equal(t, "excl-1", exclusions[0].ID)
	assert.NotEqual(t, "/original", exclusions[0].URI)
}

func TestExclusionStore_Concurrency_AddRemoveCycle(t *testing.T) {
	store := NewExclusionStore()
	ctx := context.Background()

	var wg sync.WaitGroup
	numCycles := 50

	wg.Add(numCycles * 2)
	for i := 0; i < numCycles; i++ {
		// Add
		go func(id int) {
			defer wg.Done()
			exclusion := domain.Exclusion{
				ID:       "excl-cycle",
				SourceID: "src-1",
				URI:      "/cycle/" + string(rune('A'+id)),
			}
			_ = store.Add(ctx, &exclusion)
		}(i)

		// Remove
		go func() {
			defer wg.Done()
			_ = store.Remove(ctx, "excl-cycle")
		}()
	}
	wg.Wait()

	// Should not panic or deadlock
	// Final state is indeterminate due to race, but operation should be safe
	_, _ = store.List(ctx)
}

func TestExclusionStore_ContextCancellation(t *testing.T) {
	store := NewExclusionStore()

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	exclusion := domain.Exclusion{
		ID:       "excl-1",
		SourceID: "src-1",
		URI:      "/path",
		Reason:   "test",
	}

	// Operations should complete even with cancelled context
	// (memory store doesn't actually use context for cancellation)
	err := store.Add(ctx, &exclusion)
	assert.NoError(t, err)

	_, err = store.List(ctx)
	assert.NoError(t, err)

	_, err = store.GetBySourceID(ctx, "src-1")
	assert.NoError(t, err)

	_, err = store.IsExcluded(ctx, "src-1", "/path")
	assert.NoError(t, err)

	err = store.Remove(ctx, "excl-1")
	assert.NoError(t, err)
}

func TestExclusionStore_DataIsolation(t *testing.T) {
	store := NewExclusionStore()
	ctx := context.Background()

	exclusion := domain.Exclusion{
		ID:       "excl-1",
		SourceID: "src-1",
		URI:      "/original/path",
		Reason:   "original reason",
	}

	err := store.Add(ctx, &exclusion)
	require.NoError(t, err)

	// Get the exclusion
	exclusions, err := store.List(ctx)
	require.NoError(t, err)
	require.Len(t, exclusions, 1)

	// Modify the retrieved copy
	exclusions[0].URI = "/modified/path"
	exclusions[0].Reason = "modified reason"

	// Original in store should be unchanged
	original, err := store.List(ctx)
	require.NoError(t, err)
	assert.Equal(t, "/original/path", original[0].URI)
	assert.Equal(t, "original reason", original[0].Reason)
}

func TestExclusionStore_InterfaceCompliance(t *testing.T) {
	store := NewExclusionStore()
	ctx := context.Background()

	// Verify all interface methods work
	exclusion := domain.Exclusion{
		ID:         "excl-test",
		SourceID:   "src-test",
		DocumentID: "doc-test",
		URI:        "/test/path",
		Reason:     "testing",
		ExcludedAt: time.Now(),
	}

	// Add
	err := store.Add(ctx, &exclusion)
	assert.NoError(t, err)

	// GetBySourceID
	_, err = store.GetBySourceID(ctx, "src-test")
	assert.NoError(t, err)

	// IsExcluded
	_, err = store.IsExcluded(ctx, "src-test", "/test/path")
	assert.NoError(t, err)

	// List
	_, err = store.List(ctx)
	assert.NoError(t, err)

	// Remove
	err = store.Remove(ctx, "excl-test")
	assert.NoError(t, err)
}
