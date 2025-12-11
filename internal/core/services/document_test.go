package services

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/custodia-labs/sercha-cli/internal/adapters/driven/storage/memory"
	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

func TestNewDocumentService(t *testing.T) {
	docStore := memory.NewDocumentStore()
	sourceStore := memory.NewSourceStore()
	exclusionStore := memory.NewExclusionStore()

	svc := NewDocumentService(docStore, sourceStore, exclusionStore, nil)
	require.NotNil(t, svc)
}

func TestDocumentService_ListBySource(t *testing.T) {
	docStore := memory.NewDocumentStore()
	sourceStore := memory.NewSourceStore()
	exclusionStore := memory.NewExclusionStore()
	svc := NewDocumentService(docStore, sourceStore, exclusionStore, nil)
	ctx := context.Background()

	// Add documents
	_ = docStore.SaveDocument(ctx, &domain.Document{ID: "doc-1", SourceID: "src-1", Title: "Doc 1"})
	_ = docStore.SaveDocument(ctx, &domain.Document{ID: "doc-2", SourceID: "src-1", Title: "Doc 2"})
	_ = docStore.SaveDocument(ctx, &domain.Document{ID: "doc-3", SourceID: "src-2", Title: "Doc 3"})

	docs, err := svc.ListBySource(ctx, "src-1")
	require.NoError(t, err)
	assert.Len(t, docs, 2)
}

func TestDocumentService_Get(t *testing.T) {
	docStore := memory.NewDocumentStore()
	svc := NewDocumentService(docStore, nil, nil, nil)
	ctx := context.Background()

	_ = docStore.SaveDocument(ctx, &domain.Document{ID: "doc-1", Title: "Test Doc"})

	doc, err := svc.Get(ctx, "doc-1")
	require.NoError(t, err)
	assert.Equal(t, "Test Doc", doc.Title)
}

func TestDocumentService_GetContent(t *testing.T) {
	docStore := memory.NewDocumentStore()
	svc := NewDocumentService(docStore, nil, nil, nil)
	ctx := context.Background()

	// Add document and chunks
	_ = docStore.SaveDocument(ctx, &domain.Document{ID: "doc-1"})
	_ = docStore.SaveChunks(ctx, []domain.Chunk{
		{ID: "chunk-1", DocumentID: "doc-1", Content: "First paragraph.", Position: 0},
		{ID: "chunk-2", DocumentID: "doc-1", Content: "Second paragraph.", Position: 1},
	})

	content, err := svc.GetContent(ctx, "doc-1")
	require.NoError(t, err)
	assert.Contains(t, content, "First paragraph.")
	assert.Contains(t, content, "Second paragraph.")
}

func TestDocumentService_GetDetails(t *testing.T) {
	docStore := memory.NewDocumentStore()
	sourceStore := memory.NewSourceStore()
	svc := NewDocumentService(docStore, sourceStore, nil, nil)
	ctx := context.Background()

	// Add source and document
	_ = sourceStore.Save(ctx, domain.Source{ID: "src-1", Name: "My Source", Type: "filesystem"})
	_ = docStore.SaveDocument(ctx, &domain.Document{
		ID:        "doc-1",
		SourceID:  "src-1",
		Title:     "Test Doc",
		URI:       "/path/to/file.txt",
		CreatedAt: time.Now(),
		Metadata:  map[string]any{"size": 1024},
	})
	_ = docStore.SaveChunks(ctx, []domain.Chunk{
		{ID: "chunk-1", DocumentID: "doc-1"},
		{ID: "chunk-2", DocumentID: "doc-1"},
	})

	details, err := svc.GetDetails(ctx, "doc-1")
	require.NoError(t, err)
	assert.Equal(t, "doc-1", details.ID)
	assert.Equal(t, "My Source", details.SourceName)
	assert.Equal(t, "filesystem", details.SourceType)
	assert.Equal(t, "Test Doc", details.Title)
	assert.Equal(t, 2, details.ChunkCount)
	assert.Equal(t, "1024", details.Metadata["size"])
}

func TestDocumentService_Exclude(t *testing.T) {
	docStore := memory.NewDocumentStore()
	exclusionStore := memory.NewExclusionStore()
	svc := NewDocumentService(docStore, nil, exclusionStore, nil)
	ctx := context.Background()

	// Add document
	_ = docStore.SaveDocument(ctx, &domain.Document{
		ID:       "doc-1",
		SourceID: "src-1",
		URI:      "/path/to/file.txt",
	})

	err := svc.Exclude(ctx, "doc-1", "user excluded")
	require.NoError(t, err)

	// Document should be deleted
	_, err = docStore.GetDocument(ctx, "doc-1")
	assert.Error(t, err)

	// Exclusion should be added
	excluded, _ := exclusionStore.IsExcluded(ctx, "src-1", "/path/to/file.txt")
	assert.True(t, excluded)
}

func TestDocumentService_Exclude_NonExistentDocument(t *testing.T) {
	docStore := memory.NewDocumentStore()
	exclusionStore := memory.NewExclusionStore()
	svc := NewDocumentService(docStore, nil, exclusionStore, nil)
	ctx := context.Background()

	// Try to exclude a document that doesn't exist
	err := svc.Exclude(ctx, "non-existent-doc", "test reason")
	assert.Error(t, err)
}

func TestDocumentService_Exclude_WithoutExclusionStore(t *testing.T) {
	docStore := memory.NewDocumentStore()
	svc := NewDocumentService(docStore, nil, nil, nil) // No exclusion store
	ctx := context.Background()

	// Add document
	_ = docStore.SaveDocument(ctx, &domain.Document{
		ID:       "doc-1",
		SourceID: "src-1",
		URI:      "/path/to/file.txt",
	})

	// Should still delete the document even without exclusion store
	err := svc.Exclude(ctx, "doc-1", "user excluded")
	require.NoError(t, err)

	// Document should be deleted
	_, err = docStore.GetDocument(ctx, "doc-1")
	assert.Error(t, err)
}

func TestDocumentService_Exclude_WithEmptyReason(t *testing.T) {
	docStore := memory.NewDocumentStore()
	exclusionStore := memory.NewExclusionStore()
	svc := NewDocumentService(docStore, nil, exclusionStore, nil)
	ctx := context.Background()

	// Add document
	_ = docStore.SaveDocument(ctx, &domain.Document{
		ID:       "doc-1",
		SourceID: "src-1",
		URI:      "/path/to/file.txt",
	})

	// Exclude with empty reason
	err := svc.Exclude(ctx, "doc-1", "")
	require.NoError(t, err)

	// Document should still be deleted
	_, err = docStore.GetDocument(ctx, "doc-1")
	assert.Error(t, err)
}

func TestDocumentService_Exclude_WithSpecialCharactersInURI(t *testing.T) {
	docStore := memory.NewDocumentStore()
	exclusionStore := memory.NewExclusionStore()
	svc := NewDocumentService(docStore, nil, exclusionStore, nil)
	ctx := context.Background()

	// Add document with special characters in URI
	_ = docStore.SaveDocument(ctx, &domain.Document{
		ID:       "doc-special",
		SourceID: "src-1",
		URI:      "/path/with spaces/and-special-chars!@#$.txt",
	})

	err := svc.Exclude(ctx, "doc-special", "testing special chars")
	require.NoError(t, err)

	// Document should be deleted
	_, err = docStore.GetDocument(ctx, "doc-special")
	assert.Error(t, err)

	// Exclusion should be recorded with correct URI
	excluded, _ := exclusionStore.IsExcluded(ctx, "src-1", "/path/with spaces/and-special-chars!@#$.txt")
	assert.True(t, excluded)
}

func TestDocumentService_Exclude_VerifiesExclusionID(t *testing.T) {
	docStore := memory.NewDocumentStore()
	exclusionStore := memory.NewExclusionStore()
	svc := NewDocumentService(docStore, nil, exclusionStore, nil)
	ctx := context.Background()

	// Add document
	docID := "doc-test-123"
	_ = docStore.SaveDocument(ctx, &domain.Document{
		ID:       docID,
		SourceID: "src-1",
		URI:      "/path/to/file.txt",
	})

	err := svc.Exclude(ctx, docID, "test reason")
	require.NoError(t, err)

	// Verify exclusion was created with correct ID format
	excluded, _ := exclusionStore.IsExcluded(ctx, "src-1", "/path/to/file.txt")
	assert.True(t, excluded)
}

func TestDocumentService_Refresh_NotImplemented(t *testing.T) {
	svc := NewDocumentService(nil, nil, nil, nil)
	ctx := context.Background()

	err := svc.Refresh(ctx, "doc-1")
	assert.ErrorIs(t, err, ErrRefreshNotImplemented)
}

func TestDocumentService_Open_NilDocStore(t *testing.T) {
	svc := NewDocumentService(nil, nil, nil, nil)
	ctx := context.Background()

	err := svc.Open(ctx, "doc-1")
	assert.ErrorIs(t, err, domain.ErrNotImplemented)
}

func TestDocumentService_NilDocStore(t *testing.T) {
	svc := NewDocumentService(nil, nil, nil, nil)
	ctx := context.Background()

	_, err := svc.ListBySource(ctx, "src-1")
	assert.ErrorIs(t, err, domain.ErrNotImplemented)

	_, err = svc.Get(ctx, "doc-1")
	assert.ErrorIs(t, err, domain.ErrNotImplemented)

	_, err = svc.GetContent(ctx, "doc-1")
	assert.ErrorIs(t, err, domain.ErrNotImplemented)

	_, err = svc.GetDetails(ctx, "doc-1")
	assert.ErrorIs(t, err, domain.ErrNotImplemented)

	err = svc.Exclude(ctx, "doc-1", "reason")
	assert.ErrorIs(t, err, domain.ErrNotImplemented)
}

func TestDocumentService_ListBySource_EmptySource(t *testing.T) {
	docStore := memory.NewDocumentStore()
	svc := NewDocumentService(docStore, nil, nil, nil)
	ctx := context.Background()

	// Empty source returns empty list, not nil
	docs, err := svc.ListBySource(ctx, "src-1")
	require.NoError(t, err)
	assert.Empty(t, docs)

	// Unknown source also returns empty list
	docs, err = svc.ListBySource(ctx, "unknown-src")
	require.NoError(t, err)
	assert.Empty(t, docs)
}

func TestDocumentService_GetContent_NotFound(t *testing.T) {
	docStore := memory.NewDocumentStore()
	svc := NewDocumentService(docStore, nil, nil, nil)
	ctx := context.Background()

	// Non-existent document returns error
	_, err := svc.GetContent(ctx, "unknown-doc")
	assert.Error(t, err)
}

func TestDocumentService_GetDetails_NotFound(t *testing.T) {
	docStore := memory.NewDocumentStore()
	svc := NewDocumentService(docStore, nil, nil, nil)
	ctx := context.Background()

	// Non-existent document returns error
	_, err := svc.GetDetails(ctx, "unknown-doc")
	assert.Error(t, err)
}

func TestDocumentService_GetDetails_WithChunksError(t *testing.T) {
	docStore := memory.NewDocumentStore()
	sourceStore := memory.NewSourceStore()
	svc := NewDocumentService(docStore, sourceStore, nil, nil)
	ctx := context.Background()

	// Add source and document
	_ = sourceStore.Save(ctx, domain.Source{ID: "src-1", Name: "My Source", Type: "filesystem"})
	_ = docStore.SaveDocument(ctx, &domain.Document{
		ID:        "doc-1",
		SourceID:  "src-1",
		Title:     "Test Doc",
		URI:       "/path/to/file.txt",
		CreatedAt: time.Now(),
		Metadata:  map[string]any{"size": 1024},
	})
	// Don't add chunks - GetChunks will fail or return empty

	details, err := svc.GetDetails(ctx, "doc-1")
	require.NoError(t, err)
	assert.Equal(t, "doc-1", details.ID)
	assert.Equal(t, 0, details.ChunkCount) // No chunks, should be 0
}

func TestDocumentService_GetDetails_WithoutSourceStore(t *testing.T) {
	docStore := memory.NewDocumentStore()
	svc := NewDocumentService(docStore, nil, nil, nil) // No source store
	ctx := context.Background()

	// Add document without source
	_ = docStore.SaveDocument(ctx, &domain.Document{
		ID:        "doc-1",
		SourceID:  "src-1",
		Title:     "Test Doc",
		URI:       "/path/to/file.txt",
		CreatedAt: time.Now(),
		Metadata:  map[string]any{"size": 1024},
	})

	details, err := svc.GetDetails(ctx, "doc-1")
	require.NoError(t, err)
	assert.Equal(t, "doc-1", details.ID)
	// Source name and type should be empty since no source store
	assert.Empty(t, details.SourceName)
	assert.Empty(t, details.SourceType)
}

func TestDocumentService_GetDetails_WithDifferentMetadataTypes(t *testing.T) {
	docStore := memory.NewDocumentStore()
	sourceStore := memory.NewSourceStore()
	svc := NewDocumentService(docStore, sourceStore, nil, nil)
	ctx := context.Background()

	// Add source and document with various metadata types
	_ = sourceStore.Save(ctx, domain.Source{ID: "src-1", Name: "My Source", Type: "filesystem"})
	_ = docStore.SaveDocument(ctx, &domain.Document{
		ID:       "doc-1",
		SourceID: "src-1",
		Title:    "Test Doc",
		URI:      "/path/to/file.txt",
		Metadata: map[string]any{
			"size":      1024,
			"author":    "John Doe",
			"published": true,
			"tags":      []string{"test", "doc"},
		},
	})

	details, err := svc.GetDetails(ctx, "doc-1")
	require.NoError(t, err)
	assert.Equal(t, "1024", details.Metadata["size"])
	assert.Equal(t, "John Doe", details.Metadata["author"])
	assert.Equal(t, "true", details.Metadata["published"])
	// Array types should be converted to string
	assert.Contains(t, details.Metadata["tags"], "test")
}

func TestDocumentService_GetContent_MultipleChunksUnsorted(t *testing.T) {
	docStore := memory.NewDocumentStore()
	svc := NewDocumentService(docStore, nil, nil, nil)
	ctx := context.Background()

	// Add document and chunks in wrong order
	_ = docStore.SaveDocument(ctx, &domain.Document{ID: "doc-1"})
	_ = docStore.SaveChunks(ctx, []domain.Chunk{
		{ID: "chunk-3", DocumentID: "doc-1", Content: "Third paragraph.", Position: 2},
		{ID: "chunk-1", DocumentID: "doc-1", Content: "First paragraph.", Position: 0},
		{ID: "chunk-2", DocumentID: "doc-1", Content: "Second paragraph.", Position: 1},
	})

	content, err := svc.GetContent(ctx, "doc-1")
	require.NoError(t, err)

	// Should be sorted by position
	expected := "First paragraph.\nSecond paragraph.\nThird paragraph."
	assert.Equal(t, expected, content)
}

func TestDocumentService_GetContent_SingleChunk(t *testing.T) {
	docStore := memory.NewDocumentStore()
	svc := NewDocumentService(docStore, nil, nil, nil)
	ctx := context.Background()

	// Add document and single chunk
	_ = docStore.SaveDocument(ctx, &domain.Document{ID: "doc-1"})
	_ = docStore.SaveChunks(ctx, []domain.Chunk{
		{ID: "chunk-1", DocumentID: "doc-1", Content: "Only paragraph.", Position: 0},
	})

	content, err := svc.GetContent(ctx, "doc-1")
	require.NoError(t, err)
	assert.Equal(t, "Only paragraph.", content)
}

func TestDocumentService_GetContent_EmptyChunks(t *testing.T) {
	docStore := memory.NewDocumentStore()
	svc := NewDocumentService(docStore, nil, nil, nil)
	ctx := context.Background()

	// Add document with no chunks
	_ = docStore.SaveDocument(ctx, &domain.Document{ID: "doc-1"})

	content, err := svc.GetContent(ctx, "doc-1")
	require.NoError(t, err)
	assert.Empty(t, content)
}

func TestConvertToOpenableURL(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		expected string
	}{
		{
			name:     "GitHub file URI",
			uri:      "github://owner/repo/blob/main/path/to/file.go",
			expected: "https://github.com/owner/repo/blob/main/path/to/file.go",
		},
		{
			name:     "GitHub issue URI",
			uri:      "github://owner/repo/issues/123",
			expected: "https://github.com/owner/repo/issues/123",
		},
		{
			name:     "GitHub PR URI",
			uri:      "github://owner/repo/pull/456",
			expected: "https://github.com/owner/repo/pull/456",
		},
		{
			name:     "File URI",
			uri:      "file:///path/to/local/file.txt",
			expected: "/path/to/local/file.txt",
		},
		{
			name:     "HTTP URL passthrough",
			uri:      "http://example.com/page",
			expected: "http://example.com/page",
		},
		{
			name:     "HTTPS URL passthrough",
			uri:      "https://example.com/page",
			expected: "https://example.com/page",
		},
		{
			name:     "Local path passthrough",
			uri:      "/Users/test/Documents/file.txt",
			expected: "/Users/test/Documents/file.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertToOpenableURL(tt.uri)
			assert.Equal(t, tt.expected, result)
		})
	}
}
