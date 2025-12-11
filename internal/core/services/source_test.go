package services

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/custodia-labs/sercha-cli/internal/adapters/driven/storage/memory"
	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

func TestNewSourceService(t *testing.T) {
	sourceStore := memory.NewSourceStore()
	syncStore := memory.NewSyncStateStore()
	docStore := memory.NewDocumentStore()

	service := NewSourceService(sourceStore, syncStore, docStore)

	require.NotNil(t, service)
	assert.NotNil(t, service.sourceStore)
	assert.NotNil(t, service.syncStore)
	assert.NotNil(t, service.docStore)
}

func TestSourceService_Add_Success(t *testing.T) {
	sourceStore := memory.NewSourceStore()
	syncStore := memory.NewSyncStateStore()
	docStore := memory.NewDocumentStore()
	service := NewSourceService(sourceStore, syncStore, docStore)
	ctx := context.Background()

	source := domain.Source{
		ID:   "test-source",
		Name: "Test Source",
		Type: "filesystem",
	}

	err := service.Add(ctx, source)

	require.NoError(t, err)

	// Verify source was added
	retrieved, err := service.Get(ctx, "test-source")
	require.NoError(t, err)
	assert.Equal(t, "Test Source", retrieved.Name)
	assert.Equal(t, "filesystem", retrieved.Type)
}

func TestSourceService_Add_EmptyID(t *testing.T) {
	sourceStore := memory.NewSourceStore()
	syncStore := memory.NewSyncStateStore()
	docStore := memory.NewDocumentStore()
	service := NewSourceService(sourceStore, syncStore, docStore)
	ctx := context.Background()

	source := domain.Source{
		ID:   "", // Empty ID
		Name: "Test Source",
		Type: "filesystem",
	}

	err := service.Add(ctx, source)

	assert.ErrorIs(t, err, domain.ErrInvalidInput)
}

func TestSourceService_Add_AlreadyExists(t *testing.T) {
	sourceStore := memory.NewSourceStore()
	syncStore := memory.NewSyncStateStore()
	docStore := memory.NewDocumentStore()
	service := NewSourceService(sourceStore, syncStore, docStore)
	ctx := context.Background()

	source := domain.Source{
		ID:   "test-source",
		Name: "Test Source",
		Type: "filesystem",
	}

	err := service.Add(ctx, source)
	require.NoError(t, err)

	// Try to add again
	err = service.Add(ctx, source)

	assert.ErrorIs(t, err, domain.ErrAlreadyExists)
}

func TestSourceService_Add_NilStore(t *testing.T) {
	service := NewSourceService(nil, nil, nil)
	ctx := context.Background()

	source := domain.Source{
		ID:   "test-source",
		Name: "Test Source",
	}

	err := service.Add(ctx, source)

	assert.ErrorIs(t, err, domain.ErrNotImplemented)
}

func TestSourceService_Get_Success(t *testing.T) {
	sourceStore := memory.NewSourceStore()
	syncStore := memory.NewSyncStateStore()
	docStore := memory.NewDocumentStore()
	service := NewSourceService(sourceStore, syncStore, docStore)
	ctx := context.Background()

	source := domain.Source{
		ID:   "test-source",
		Name: "Test Source",
		Type: "github",
	}
	err := service.Add(ctx, source)
	require.NoError(t, err)

	retrieved, err := service.Get(ctx, "test-source")

	require.NoError(t, err)
	require.NotNil(t, retrieved)
	assert.Equal(t, "test-source", retrieved.ID)
	assert.Equal(t, "Test Source", retrieved.Name)
	assert.Equal(t, "github", retrieved.Type)
}

func TestSourceService_Get_NotFound(t *testing.T) {
	sourceStore := memory.NewSourceStore()
	syncStore := memory.NewSyncStateStore()
	docStore := memory.NewDocumentStore()
	service := NewSourceService(sourceStore, syncStore, docStore)
	ctx := context.Background()

	retrieved, err := service.Get(ctx, "nonexistent")

	assert.ErrorIs(t, err, domain.ErrNotFound)
	assert.Nil(t, retrieved)
}

func TestSourceService_Get_NilStore(t *testing.T) {
	service := NewSourceService(nil, nil, nil)
	ctx := context.Background()

	_, err := service.Get(ctx, "test-source")

	assert.ErrorIs(t, err, domain.ErrNotImplemented)
}

func TestSourceService_List_Empty(t *testing.T) {
	sourceStore := memory.NewSourceStore()
	syncStore := memory.NewSyncStateStore()
	docStore := memory.NewDocumentStore()
	service := NewSourceService(sourceStore, syncStore, docStore)
	ctx := context.Background()

	sources, err := service.List(ctx)

	require.NoError(t, err)
	// Empty store returns empty list (no placeholder data)
	assert.Empty(t, sources)
}

func TestSourceService_List_WithSources(t *testing.T) {
	sourceStore := memory.NewSourceStore()
	syncStore := memory.NewSyncStateStore()
	docStore := memory.NewDocumentStore()
	service := NewSourceService(sourceStore, syncStore, docStore)
	ctx := context.Background()

	// Add sources
	_ = service.Add(ctx, domain.Source{ID: "src-1", Name: "Source 1", Type: "filesystem"})
	_ = service.Add(ctx, domain.Source{ID: "src-2", Name: "Source 2", Type: "github"})
	_ = service.Add(ctx, domain.Source{ID: "src-3", Name: "Source 3", Type: "notion"})

	sources, err := service.List(ctx)

	require.NoError(t, err)
	assert.Len(t, sources, 3)
}

func TestSourceService_List_NilStore(t *testing.T) {
	service := NewSourceService(nil, nil, nil)
	ctx := context.Background()

	_, err := service.List(ctx)

	assert.ErrorIs(t, err, domain.ErrNotImplemented)
}

func TestSourceService_Remove_Success(t *testing.T) {
	sourceStore := memory.NewSourceStore()
	syncStore := memory.NewSyncStateStore()
	docStore := memory.NewDocumentStore()
	service := NewSourceService(sourceStore, syncStore, docStore)
	ctx := context.Background()

	source := domain.Source{
		ID:   "test-source",
		Name: "Test Source",
	}
	err := service.Add(ctx, source)
	require.NoError(t, err)

	err = service.Remove(ctx, "test-source")
	require.NoError(t, err)

	// Verify source was removed
	_, err = service.Get(ctx, "test-source")
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestSourceService_Remove_WithDocuments(t *testing.T) {
	sourceStore := memory.NewSourceStore()
	syncStore := memory.NewSyncStateStore()
	docStore := memory.NewDocumentStore()
	service := NewSourceService(sourceStore, syncStore, docStore)
	ctx := context.Background()

	// Add source
	source := domain.Source{
		ID:   "test-source",
		Name: "Test Source",
	}
	err := service.Add(ctx, source)
	require.NoError(t, err)

	// Add documents for this source
	doc1 := &domain.Document{
		ID:       "doc-1",
		SourceID: "test-source",
		Title:    "Doc 1",
	}
	doc2 := &domain.Document{
		ID:       "doc-2",
		SourceID: "test-source",
		Title:    "Doc 2",
	}
	_ = docStore.SaveDocument(ctx, doc1)
	_ = docStore.SaveDocument(ctx, doc2)

	// Remove source (should cleanup documents)
	err = service.Remove(ctx, "test-source")
	require.NoError(t, err)

	// Verify source was removed
	_, err = service.Get(ctx, "test-source")
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestSourceService_Remove_WithSyncState(t *testing.T) {
	sourceStore := memory.NewSourceStore()
	syncStore := memory.NewSyncStateStore()
	docStore := memory.NewDocumentStore()
	service := NewSourceService(sourceStore, syncStore, docStore)
	ctx := context.Background()

	// Add source
	source := domain.Source{
		ID:   "test-source",
		Name: "Test Source",
	}
	err := service.Add(ctx, source)
	require.NoError(t, err)

	// Add sync state
	syncState := domain.SyncState{
		SourceID: "test-source",
	}
	_ = syncStore.Save(ctx, syncState)

	// Remove source (should cleanup sync state)
	err = service.Remove(ctx, "test-source")
	require.NoError(t, err)

	// Verify sync state was removed
	_, err = syncStore.Get(ctx, "test-source")
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestSourceService_Remove_NilStore(t *testing.T) {
	service := NewSourceService(nil, nil, nil)
	ctx := context.Background()

	err := service.Remove(ctx, "test-source")

	assert.ErrorIs(t, err, domain.ErrNotImplemented)
}

func TestSourceService_Remove_NilDocStore(t *testing.T) {
	sourceStore := memory.NewSourceStore()
	syncStore := memory.NewSyncStateStore()
	service := NewSourceService(sourceStore, syncStore, nil)
	ctx := context.Background()

	source := domain.Source{
		ID:   "test-source",
		Name: "Test Source",
	}
	err := service.Add(ctx, source)
	require.NoError(t, err)

	// Should still work without doc store
	err = service.Remove(ctx, "test-source")
	require.NoError(t, err)
}

func TestSourceService_Remove_NilSyncStore(t *testing.T) {
	sourceStore := memory.NewSourceStore()
	docStore := memory.NewDocumentStore()
	service := NewSourceService(sourceStore, nil, docStore)
	ctx := context.Background()

	source := domain.Source{
		ID:   "test-source",
		Name: "Test Source",
	}
	err := service.Add(ctx, source)
	require.NoError(t, err)

	// Should still work without sync store
	err = service.Remove(ctx, "test-source")
	require.NoError(t, err)
}

func TestSourceService_ValidateConfig_NotImplemented(t *testing.T) {
	sourceStore := memory.NewSourceStore()
	syncStore := memory.NewSyncStateStore()
	docStore := memory.NewDocumentStore()
	service := NewSourceService(sourceStore, syncStore, docStore)
	ctx := context.Background()

	config := map[string]string{
		"path": "/some/path",
	}

	err := service.ValidateConfig(ctx, "filesystem", config)

	assert.ErrorIs(t, err, domain.ErrNotImplemented)
}

func TestSourceService_Add_DifferentTypes(t *testing.T) {
	sourceStore := memory.NewSourceStore()
	syncStore := memory.NewSyncStateStore()
	docStore := memory.NewDocumentStore()
	service := NewSourceService(sourceStore, syncStore, docStore)
	ctx := context.Background()

	sourceTypes := []string{
		"filesystem",
		"github",
		"gitlab",
		"notion",
		"confluence",
		"google-drive",
		"slack",
	}

	for _, sourceType := range sourceTypes {
		source := domain.Source{
			ID:   sourceType + "-src",
			Name: sourceType + " Source",
			Type: sourceType,
		}
		err := service.Add(ctx, source)
		require.NoError(t, err, "failed to add source type: %s", sourceType)
	}

	// Verify all were added
	sources, err := service.List(ctx)
	require.NoError(t, err)
	assert.Len(t, sources, len(sourceTypes))
}

func TestSourceService_Remove_MultipleDocuments(t *testing.T) {
	sourceStore := memory.NewSourceStore()
	syncStore := memory.NewSyncStateStore()
	docStore := memory.NewDocumentStore()
	service := NewSourceService(sourceStore, syncStore, docStore)
	ctx := context.Background()

	// Add source
	source := domain.Source{
		ID:   "test-source",
		Name: "Test Source",
	}
	err := service.Add(ctx, source)
	require.NoError(t, err)

	// Add multiple documents
	for i := 0; i < 10; i++ {
		doc := &domain.Document{
			ID:       fmt.Sprintf("doc-%d", i),
			SourceID: "test-source",
			Title:    fmt.Sprintf("Document %d", i),
		}
		_ = docStore.SaveDocument(ctx, doc)
	}

	// Remove source
	err = service.Remove(ctx, "test-source")
	require.NoError(t, err)

	// Verify all documents were removed
	docs, err := docStore.ListDocuments(ctx, "test-source")
	require.NoError(t, err)
	assert.Empty(t, docs)
}

func TestSourceService_List_ReturnsEmptyWhenNoSources(t *testing.T) {
	sourceStore := memory.NewSourceStore()
	syncStore := memory.NewSyncStateStore()
	docStore := memory.NewDocumentStore()
	service := NewSourceService(sourceStore, syncStore, docStore)
	ctx := context.Background()

	sources, err := service.List(ctx)

	require.NoError(t, err)
	// Should return empty list (no placeholder sources)
	assert.Empty(t, sources)
}

func TestSourceService_Add_WithConfig(t *testing.T) {
	sourceStore := memory.NewSourceStore()
	syncStore := memory.NewSyncStateStore()
	docStore := memory.NewDocumentStore()
	service := NewSourceService(sourceStore, syncStore, docStore)
	ctx := context.Background()

	source := domain.Source{
		ID:   "test-source",
		Name: "Test Source",
		Type: "github",
		Config: map[string]string{
			"owner": "test-org",
			"repo":  "test-repo",
		},
	}

	err := service.Add(ctx, source)
	require.NoError(t, err)

	retrieved, err := service.Get(ctx, "test-source")
	require.NoError(t, err)
	assert.Equal(t, "test-org", retrieved.Config["owner"])
	assert.Equal(t, "test-repo", retrieved.Config["repo"])
}

func TestSourceService_Remove_NonexistentSource(t *testing.T) {
	sourceStore := memory.NewSourceStore()
	syncStore := memory.NewSyncStateStore()
	docStore := memory.NewDocumentStore()
	service := NewSourceService(sourceStore, syncStore, docStore)
	ctx := context.Background()

	// Removing nonexistent source should not error (idempotent)
	err := service.Remove(ctx, "nonexistent")

	assert.NoError(t, err)
}

func TestSourceService_ContextCancellation(t *testing.T) {
	sourceStore := memory.NewSourceStore()
	syncStore := memory.NewSyncStateStore()
	docStore := memory.NewDocumentStore()
	service := NewSourceService(sourceStore, syncStore, docStore)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	source := domain.Source{
		ID:   "test-source",
		Name: "Test Source",
	}

	// Current implementation doesn't check context
	err := service.Add(ctx, source)
	assert.NoError(t, err)
}
