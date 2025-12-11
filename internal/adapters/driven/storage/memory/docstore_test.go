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

func TestNewDocumentStore(t *testing.T) {
	store := NewDocumentStore()
	require.NotNil(t, store)
	assert.NotNil(t, store.documents)
	assert.NotNil(t, store.chunks)
}

func TestDocumentStore_SaveDocument_Success(t *testing.T) {
	store := NewDocumentStore()
	ctx := context.Background()

	now := time.Now()
	doc := &domain.Document{
		ID:        "doc-1",
		SourceID:  "src-1",
		URI:       "/path/to/document.txt",
		Title:     "Test Document",
		ParentID:  nil,
		Metadata:  map[string]any{"author": "John Doe", "tags": []string{"test"}},
		CreatedAt: now,
		UpdatedAt: now,
	}

	err := store.SaveDocument(ctx, doc)
	require.NoError(t, err)

	// Verify it was saved
	saved, err := store.GetDocument(ctx, "doc-1")
	require.NoError(t, err)
	assert.Equal(t, "doc-1", saved.ID)
	assert.Equal(t, "src-1", saved.SourceID)
	assert.Equal(t, "/path/to/document.txt", saved.URI)
	assert.Equal(t, "Test Document", saved.Title)
	assert.Equal(t, "John Doe", saved.Metadata["author"])
}

func TestDocumentStore_SaveDocument_Update(t *testing.T) {
	store := NewDocumentStore()
	ctx := context.Background()

	doc1 := &domain.Document{
		ID:       "doc-1",
		SourceID: "src-1",
		Title:    "Original Title",
	}
	doc2 := &domain.Document{
		ID:       "doc-1",
		SourceID: "src-1",
		Title:    "Updated Title",
	}

	err := store.SaveDocument(ctx, doc1)
	require.NoError(t, err)

	err = store.SaveDocument(ctx, doc2)
	require.NoError(t, err)

	// Should have the updated values
	saved, err := store.GetDocument(ctx, "doc-1")
	require.NoError(t, err)
	assert.Equal(t, "Updated Title", saved.Title)
}

func TestDocumentStore_SaveDocument_WithParent(t *testing.T) {
	store := NewDocumentStore()
	ctx := context.Background()

	parentID := "doc-parent"
	doc := &domain.Document{
		ID:       "doc-1",
		SourceID: "src-1",
		Title:    "Child Document",
		ParentID: &parentID,
	}

	err := store.SaveDocument(ctx, doc)
	require.NoError(t, err)

	saved, err := store.GetDocument(ctx, "doc-1")
	require.NoError(t, err)
	require.NotNil(t, saved.ParentID)
	assert.Equal(t, "doc-parent", *saved.ParentID)
}

func TestDocumentStore_SaveDocument_NilMetadata(t *testing.T) {
	store := NewDocumentStore()
	ctx := context.Background()

	doc := &domain.Document{
		ID:       "doc-1",
		SourceID: "src-1",
		Title:    "Document",
		Metadata: nil,
	}

	err := store.SaveDocument(ctx, doc)
	require.NoError(t, err)

	saved, err := store.GetDocument(ctx, "doc-1")
	require.NoError(t, err)
	assert.Nil(t, saved.Metadata)
}

func TestDocumentStore_GetDocument_NotFound(t *testing.T) {
	store := NewDocumentStore()
	ctx := context.Background()

	doc, err := store.GetDocument(ctx, "nonexistent")

	assert.ErrorIs(t, err, domain.ErrNotFound)
	assert.Nil(t, doc)
}

func TestDocumentStore_GetDocument_Success(t *testing.T) {
	store := NewDocumentStore()
	ctx := context.Background()

	original := &domain.Document{
		ID:       "doc-1",
		SourceID: "src-1",
		URI:      "https://example.com/doc",
		Title:    "Example Document",
		Metadata: map[string]any{"key": "value"},
	}

	err := store.SaveDocument(ctx, original)
	require.NoError(t, err)

	retrieved, err := store.GetDocument(ctx, "doc-1")

	require.NoError(t, err)
	require.NotNil(t, retrieved)
	assert.Equal(t, "doc-1", retrieved.ID)
	assert.Equal(t, "src-1", retrieved.SourceID)
	assert.Equal(t, "https://example.com/doc", retrieved.URI)
	assert.Equal(t, "Example Document", retrieved.Title)
	assert.Equal(t, "value", retrieved.Metadata["key"])
}

func TestDocumentStore_SaveChunks_Success(t *testing.T) {
	store := NewDocumentStore()
	ctx := context.Background()

	chunks := []domain.Chunk{
		{
			ID:         "chunk-1",
			DocumentID: "doc-1",
			Content:    "First chunk content",
			Position:   0,
			Embedding:  []float32{0.1, 0.2, 0.3},
			Metadata:   map[string]any{"section": "intro"},
		},
		{
			ID:         "chunk-2",
			DocumentID: "doc-1",
			Content:    "Second chunk content",
			Position:   1,
			Embedding:  []float32{0.4, 0.5, 0.6},
			Metadata:   map[string]any{"section": "body"},
		},
	}

	err := store.SaveChunks(ctx, chunks)
	require.NoError(t, err)

	// Verify they were saved
	saved, err := store.GetChunks(ctx, "doc-1")
	require.NoError(t, err)
	assert.Len(t, saved, 2)
	assert.Equal(t, "chunk-1", saved[0].ID)
	assert.Equal(t, "chunk-2", saved[1].ID)
}

func TestDocumentStore_SaveChunks_Empty(t *testing.T) {
	store := NewDocumentStore()
	ctx := context.Background()

	err := store.SaveChunks(ctx, []domain.Chunk{})
	require.NoError(t, err)

	// Empty save should not error
}

func TestDocumentStore_SaveChunks_Nil(t *testing.T) {
	store := NewDocumentStore()
	ctx := context.Background()

	err := store.SaveChunks(ctx, nil)
	require.NoError(t, err)

	// Nil save should not error
}

func TestDocumentStore_SaveChunks_Update(t *testing.T) {
	store := NewDocumentStore()
	ctx := context.Background()

	chunks1 := []domain.Chunk{
		{ID: "chunk-1", DocumentID: "doc-1", Content: "Original"},
	}
	chunks2 := []domain.Chunk{
		{ID: "chunk-1-new", DocumentID: "doc-1", Content: "Updated"},
	}

	err := store.SaveChunks(ctx, chunks1)
	require.NoError(t, err)

	err = store.SaveChunks(ctx, chunks2)
	require.NoError(t, err)

	// Should have the new chunks
	saved, err := store.GetChunks(ctx, "doc-1")
	require.NoError(t, err)
	assert.Len(t, saved, 1)
	assert.Equal(t, "chunk-1-new", saved[0].ID)
	assert.Equal(t, "Updated", saved[0].Content)
}

func TestDocumentStore_GetChunks_NotFound(t *testing.T) {
	store := NewDocumentStore()
	ctx := context.Background()

	chunks, err := store.GetChunks(ctx, "nonexistent")

	require.NoError(t, err)
	assert.Nil(t, chunks)
}

func TestDocumentStore_GetChunks_Success(t *testing.T) {
	store := NewDocumentStore()
	ctx := context.Background()

	chunks := []domain.Chunk{
		{ID: "chunk-1", DocumentID: "doc-1", Content: "Content 1", Position: 0},
		{ID: "chunk-2", DocumentID: "doc-1", Content: "Content 2", Position: 1},
		{ID: "chunk-3", DocumentID: "doc-1", Content: "Content 3", Position: 2},
	}

	err := store.SaveChunks(ctx, chunks)
	require.NoError(t, err)

	retrieved, err := store.GetChunks(ctx, "doc-1")

	require.NoError(t, err)
	require.Len(t, retrieved, 3)
	assert.Equal(t, "chunk-1", retrieved[0].ID)
	assert.Equal(t, "chunk-2", retrieved[1].ID)
	assert.Equal(t, "chunk-3", retrieved[2].ID)
}

func TestDocumentStore_GetChunk_Success(t *testing.T) {
	store := NewDocumentStore()
	ctx := context.Background()

	chunks := []domain.Chunk{
		{ID: "chunk-1", DocumentID: "doc-1", Content: "Content 1", Position: 0},
		{ID: "chunk-2", DocumentID: "doc-1", Content: "Content 2", Position: 1},
	}

	err := store.SaveChunks(ctx, chunks)
	require.NoError(t, err)

	retrieved, err := store.GetChunk(ctx, "chunk-2")

	require.NoError(t, err)
	require.NotNil(t, retrieved)
	assert.Equal(t, "chunk-2", retrieved.ID)
	assert.Equal(t, "Content 2", retrieved.Content)
	assert.Equal(t, 1, retrieved.Position)
}

func TestDocumentStore_GetChunk_NotFound(t *testing.T) {
	store := NewDocumentStore()
	ctx := context.Background()

	chunk, err := store.GetChunk(ctx, "nonexistent")

	assert.ErrorIs(t, err, domain.ErrNotFound)
	assert.Nil(t, chunk)
}

func TestDocumentStore_GetChunk_FromMultipleDocuments(t *testing.T) {
	store := NewDocumentStore()
	ctx := context.Background()

	chunks1 := []domain.Chunk{
		{ID: "chunk-1", DocumentID: "doc-1", Content: "Doc 1 Content"},
	}
	chunks2 := []domain.Chunk{
		{ID: "chunk-2", DocumentID: "doc-2", Content: "Doc 2 Content"},
	}

	_ = store.SaveChunks(ctx, chunks1)
	_ = store.SaveChunks(ctx, chunks2)

	// Should find chunk from doc-2
	retrieved, err := store.GetChunk(ctx, "chunk-2")
	require.NoError(t, err)
	assert.Equal(t, "doc-2", retrieved.DocumentID)
}

func TestDocumentStore_DeleteDocument_Success(t *testing.T) {
	store := NewDocumentStore()
	ctx := context.Background()

	doc := &domain.Document{
		ID:       "doc-1",
		SourceID: "src-1",
		Title:    "Test Document",
	}
	chunks := []domain.Chunk{
		{ID: "chunk-1", DocumentID: "doc-1", Content: "Content"},
	}

	_ = store.SaveDocument(ctx, doc)
	_ = store.SaveChunks(ctx, chunks)

	err := store.DeleteDocument(ctx, "doc-1")
	require.NoError(t, err)

	// Document should be deleted
	_, err = store.GetDocument(ctx, "doc-1")
	assert.ErrorIs(t, err, domain.ErrNotFound)

	// Chunks should also be deleted
	deletedChunks, err := store.GetChunks(ctx, "doc-1")
	require.NoError(t, err)
	assert.Nil(t, deletedChunks)
}

func TestDocumentStore_DeleteDocument_NonExistent(t *testing.T) {
	store := NewDocumentStore()
	ctx := context.Background()

	// Delete non-existent should not error
	err := store.DeleteDocument(ctx, "nonexistent")
	assert.NoError(t, err)
}

func TestDocumentStore_DeleteDocument_OnlyChunks(t *testing.T) {
	store := NewDocumentStore()
	ctx := context.Background()

	// Save only chunks, no document
	chunks := []domain.Chunk{
		{ID: "chunk-1", DocumentID: "doc-1", Content: "Content"},
	}
	_ = store.SaveChunks(ctx, chunks)

	err := store.DeleteDocument(ctx, "doc-1")
	require.NoError(t, err)

	// Chunks should be deleted
	deletedChunks, err := store.GetChunks(ctx, "doc-1")
	require.NoError(t, err)
	assert.Nil(t, deletedChunks)
}

func TestDocumentStore_ListDocuments_Empty(t *testing.T) {
	store := NewDocumentStore()
	ctx := context.Background()

	docs, err := store.ListDocuments(ctx, "src-1")

	require.NoError(t, err)
	assert.Nil(t, docs)
}

func TestDocumentStore_ListDocuments_Success(t *testing.T) {
	store := NewDocumentStore()
	ctx := context.Background()

	docs := []*domain.Document{
		{ID: "doc-1", SourceID: "src-1", Title: "Doc 1"},
		{ID: "doc-2", SourceID: "src-1", Title: "Doc 2"},
		{ID: "doc-3", SourceID: "src-2", Title: "Doc 3"},
	}

	for _, doc := range docs {
		_ = store.SaveDocument(ctx, doc)
	}

	retrieved, err := store.ListDocuments(ctx, "src-1")

	require.NoError(t, err)
	assert.Len(t, retrieved, 2)

	ids := make(map[string]bool)
	for _, d := range retrieved {
		ids[d.ID] = true
	}
	assert.True(t, ids["doc-1"])
	assert.True(t, ids["doc-2"])
	assert.False(t, ids["doc-3"])
}

func TestDocumentStore_ListDocuments_FiltersBySourceID(t *testing.T) {
	store := NewDocumentStore()
	ctx := context.Background()

	docs := []*domain.Document{
		{ID: "doc-1", SourceID: "src-1"},
		{ID: "doc-2", SourceID: "src-2"},
		{ID: "doc-3", SourceID: "src-1"},
		{ID: "doc-4", SourceID: "src-3"},
	}

	for _, doc := range docs {
		_ = store.SaveDocument(ctx, doc)
	}

	// List for src-1
	docs1, err := store.ListDocuments(ctx, "src-1")
	require.NoError(t, err)
	assert.Len(t, docs1, 2)

	// List for src-2
	docs2, err := store.ListDocuments(ctx, "src-2")
	require.NoError(t, err)
	assert.Len(t, docs2, 1)

	// List for src-3
	docs3, err := store.ListDocuments(ctx, "src-3")
	require.NoError(t, err)
	assert.Len(t, docs3, 1)

	// List for nonexistent source
	docs4, err := store.ListDocuments(ctx, "src-nonexistent")
	require.NoError(t, err)
	assert.Nil(t, docs4)
}

func TestDocumentStore_Concurrency_SaveAndGetDocuments(t *testing.T) {
	store := NewDocumentStore()
	ctx := context.Background()

	var wg sync.WaitGroup
	numGoroutines := 50

	// Concurrent saves
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			doc := &domain.Document{
				ID:       "doc-" + string(rune('A'+id)),
				SourceID: "src-1",
				Title:    "Document " + string(rune('A'+id)),
			}
			_ = store.SaveDocument(ctx, doc)
		}(i)
	}
	wg.Wait()

	// Concurrent reads
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			_, _ = store.GetDocument(ctx, "doc-"+string(rune('A'+id)))
		}(i)
	}
	wg.Wait()

	// Verify all saved
	docs, err := store.ListDocuments(ctx, "src-1")
	require.NoError(t, err)
	assert.Len(t, docs, numGoroutines)
}

func TestDocumentStore_Concurrency_SaveAndGetChunks(t *testing.T) {
	store := NewDocumentStore()
	ctx := context.Background()

	var wg sync.WaitGroup
	numGoroutines := 50

	// Concurrent chunk saves
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			chunks := []domain.Chunk{
				{
					ID:         "chunk-" + string(rune('A'+id)),
					DocumentID: "doc-" + string(rune('A'+id)),
					Content:    "Content " + string(rune('A'+id)),
				},
			}
			_ = store.SaveChunks(ctx, chunks)
		}(i)
	}
	wg.Wait()

	// Concurrent chunk reads
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			_, _ = store.GetChunks(ctx, "doc-"+string(rune('A'+id)))
			_, _ = store.GetChunk(ctx, "chunk-"+string(rune('A'+id)))
		}(i)
	}
	wg.Wait()
}

func TestDocumentStore_Concurrency_MixedOperations(t *testing.T) {
	store := NewDocumentStore()
	ctx := context.Background()

	var wg sync.WaitGroup
	numOperations := 100

	// Pre-populate
	for i := 0; i < 10; i++ {
		doc := &domain.Document{
			ID:       "doc-" + string(rune('0'+i)),
			SourceID: "src-1",
		}
		_ = store.SaveDocument(ctx, doc)
	}

	// Run mixed concurrent operations
	wg.Add(numOperations)
	for i := 0; i < numOperations; i++ {
		go func(id int) {
			defer wg.Done()
			switch id % 5 {
			case 0: // Save document
				doc := &domain.Document{
					ID:       "doc-concurrent-" + string(rune('A'+id%26)),
					SourceID: "src-1",
				}
				_ = store.SaveDocument(ctx, doc)
			case 1: // Save chunks
				chunks := []domain.Chunk{
					{ID: "chunk-" + string(rune('A'+id%26)), DocumentID: "doc-concurrent"},
				}
				_ = store.SaveChunks(ctx, chunks)
			case 2: // Get document
				_, _ = store.GetDocument(ctx, "doc-"+string(rune('0'+id%10)))
			case 3: // Get chunks
				_, _ = store.GetChunks(ctx, "doc-"+string(rune('0'+id%10)))
			case 4: // List documents
				_, _ = store.ListDocuments(ctx, "src-1")
			}
		}(i)
	}
	wg.Wait()

	// Should not panic or deadlock
	docs, err := store.ListDocuments(ctx, "src-1")
	require.NoError(t, err)
	assert.NotNil(t, docs)
}

func TestDocumentStore_Concurrency_DeleteWhileReading(t *testing.T) {
	store := NewDocumentStore()
	ctx := context.Background()

	// Pre-populate
	for i := 0; i < 10; i++ {
		doc := &domain.Document{
			ID:       "doc-" + string(rune('A'+i)),
			SourceID: "src-1",
		}
		_ = store.SaveDocument(ctx, doc)
	}

	var wg sync.WaitGroup
	numOperations := 100

	// Concurrent reads and deletes
	wg.Add(numOperations)
	for i := 0; i < numOperations; i++ {
		go func(id int) {
			defer wg.Done()
			if id%2 == 0 {
				_, _ = store.GetDocument(ctx, "doc-"+string(rune('A'+id%10)))
			} else {
				_ = store.DeleteDocument(ctx, "doc-"+string(rune('A'+id%10)))
			}
		}(i)
	}
	wg.Wait()

	// Should not panic or deadlock
	_, _ = store.ListDocuments(ctx, "src-1")
}

func TestDocumentStore_ContextCancellation(t *testing.T) {
	store := NewDocumentStore()

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	doc := &domain.Document{
		ID:       "doc-1",
		SourceID: "src-1",
		Title:    "Test",
	}
	chunks := []domain.Chunk{
		{ID: "chunk-1", DocumentID: "doc-1", Content: "Content"},
	}

	// Operations should complete even with cancelled context
	err := store.SaveDocument(ctx, doc)
	assert.NoError(t, err)

	err = store.SaveChunks(ctx, chunks)
	assert.NoError(t, err)

	_, err = store.GetDocument(ctx, "doc-1")
	assert.NoError(t, err)

	_, err = store.GetChunks(ctx, "doc-1")
	assert.NoError(t, err)

	_, err = store.GetChunk(ctx, "chunk-1")
	assert.NoError(t, err)

	_, err = store.ListDocuments(ctx, "src-1")
	assert.NoError(t, err)

	err = store.DeleteDocument(ctx, "doc-1")
	assert.NoError(t, err)
}

func TestDocumentStore_DataIsolation_Document(t *testing.T) {
	store := NewDocumentStore()
	ctx := context.Background()

	doc := &domain.Document{
		ID:       "doc-1",
		SourceID: "src-1",
		Title:    "Original Title",
		Metadata: map[string]any{"key": "value"},
	}

	err := store.SaveDocument(ctx, doc)
	require.NoError(t, err)

	// Get the document
	retrieved, err := store.GetDocument(ctx, "doc-1")
	require.NoError(t, err)

	// Modify the retrieved copy - Title is a value type so it's safe
	retrieved.Title = "Modified Title"
	// Metadata is a map (reference type), modifying it would affect the stored copy
	// This is a known limitation of the memory store

	// Verify Title change doesn't affect stored copy (value type)
	original, err := store.GetDocument(ctx, "doc-1")
	require.NoError(t, err)
	assert.Equal(t, "Original Title", original.Title)

	// Note: Metadata map is shared (reference type), so modifications would be visible
	// In practice, callers should not modify retrieved values
}

func TestDocumentStore_DataIsolation_Chunks(t *testing.T) {
	store := NewDocumentStore()
	ctx := context.Background()

	chunks := []domain.Chunk{
		{ID: "chunk-1", DocumentID: "doc-1", Content: "Original Content"},
	}

	err := store.SaveChunks(ctx, chunks)
	require.NoError(t, err)

	// Get the chunks
	retrieved, err := store.GetChunks(ctx, "doc-1")
	require.NoError(t, err)

	// The chunks slice is stored by reference, so the returned slice points to the same array
	// Verify this is the case by showing that modifications affect the stored copy
	retrieved[0].Content = "Modified Content"

	// The stored copy will reflect the change because slices share underlying arrays
	modified, err := store.GetChunks(ctx, "doc-1")
	require.NoError(t, err)
	assert.Equal(t, "Modified Content", modified[0].Content)

	// Note: This is a known limitation of the memory store - it doesn't deep copy
	// In practice, callers should not modify retrieved values
}

func TestDocumentStore_ChunkWithLargeEmbedding(t *testing.T) {
	store := NewDocumentStore()
	ctx := context.Background()

	// Create chunk with large embedding vector
	embedding := make([]float32, 1536) // Common size for embeddings
	for i := range embedding {
		embedding[i] = float32(i) * 0.001
	}

	chunks := []domain.Chunk{
		{
			ID:         "chunk-1",
			DocumentID: "doc-1",
			Content:    "Content",
			Embedding:  embedding,
		},
	}

	err := store.SaveChunks(ctx, chunks)
	require.NoError(t, err)

	retrieved, err := store.GetChunk(ctx, "chunk-1")
	require.NoError(t, err)
	assert.Len(t, retrieved.Embedding, 1536)
	assert.Equal(t, float32(0), retrieved.Embedding[0])
	assert.Equal(t, float32(1)*0.001, retrieved.Embedding[1])
}

func TestDocumentStore_ChunkWithNilEmbedding(t *testing.T) {
	store := NewDocumentStore()
	ctx := context.Background()

	chunks := []domain.Chunk{
		{
			ID:         "chunk-1",
			DocumentID: "doc-1",
			Content:    "Content",
			Embedding:  nil,
		},
	}

	err := store.SaveChunks(ctx, chunks)
	require.NoError(t, err)

	retrieved, err := store.GetChunk(ctx, "chunk-1")
	require.NoError(t, err)
	assert.Nil(t, retrieved.Embedding)
}

func TestDocumentStore_MultipleChunksPerDocument(t *testing.T) {
	store := NewDocumentStore()
	ctx := context.Background()

	// Save 100 chunks for a single document
	chunks := make([]domain.Chunk, 100)
	for i := 0; i < 100; i++ {
		chunks[i] = domain.Chunk{
			ID:         "chunk-" + string(rune('A'+i%26)) + "-" + string(rune('0'+i/26)),
			DocumentID: "doc-1",
			Content:    "Content " + string(rune('0'+i)),
			Position:   i,
		}
	}

	err := store.SaveChunks(ctx, chunks)
	require.NoError(t, err)

	retrieved, err := store.GetChunks(ctx, "doc-1")
	require.NoError(t, err)
	assert.Len(t, retrieved, 100)
}

func TestDocumentStore_InterfaceCompliance(t *testing.T) {
	store := NewDocumentStore()
	ctx := context.Background()

	// Verify all interface methods work
	doc := &domain.Document{
		ID:       "doc-test",
		SourceID: "src-test",
		Title:    "Test Document",
	}
	chunks := []domain.Chunk{
		{ID: "chunk-test", DocumentID: "doc-test", Content: "Test content"},
	}

	// SaveDocument
	err := store.SaveDocument(ctx, doc)
	assert.NoError(t, err)

	// SaveChunks
	err = store.SaveChunks(ctx, chunks)
	assert.NoError(t, err)

	// GetDocument
	_, err = store.GetDocument(ctx, "doc-test")
	assert.NoError(t, err)

	// GetChunks
	_, err = store.GetChunks(ctx, "doc-test")
	assert.NoError(t, err)

	// GetChunk
	_, err = store.GetChunk(ctx, "chunk-test")
	assert.NoError(t, err)

	// ListDocuments
	_, err = store.ListDocuments(ctx, "src-test")
	assert.NoError(t, err)

	// DeleteDocument
	err = store.DeleteDocument(ctx, "doc-test")
	assert.NoError(t, err)
}
