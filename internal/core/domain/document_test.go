package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDocument_Fields tests Document structure fields
func TestDocument_Fields(t *testing.T) {
	now := time.Now()
	parentID := "parent-123"

	doc := Document{
		ID:        "doc-123",
		SourceID:  "source-456",
		URI:       "file:///path/to/document.pdf",
		Title:     "Test Document",
		ParentID:  &parentID,
		Metadata:  map[string]any{"author": "John Doe", "pages": 42},
		CreatedAt: now,
		UpdatedAt: now,
	}

	assert.Equal(t, "doc-123", doc.ID)
	assert.Equal(t, "source-456", doc.SourceID)
	assert.Equal(t, "file:///path/to/document.pdf", doc.URI)
	assert.Equal(t, "Test Document", doc.Title)
	require.NotNil(t, doc.ParentID)
	assert.Equal(t, "parent-123", *doc.ParentID)
	assert.Equal(t, "John Doe", doc.Metadata["author"])
	assert.Equal(t, 42, doc.Metadata["pages"])
	assert.Equal(t, now, doc.CreatedAt)
	assert.Equal(t, now, doc.UpdatedAt)
}

// TestDocument_WithoutParent tests Document without parent
func TestDocument_WithoutParent(t *testing.T) {
	doc := Document{
		ID:       "doc-123",
		SourceID: "source-456",
		URI:      "file:///standalone.txt",
		Title:    "Standalone Document",
		ParentID: nil,
	}

	assert.Nil(t, doc.ParentID)
}

// TestDocument_EmptyMetadata tests Document with empty metadata
func TestDocument_EmptyMetadata(t *testing.T) {
	doc := Document{
		ID:       "doc-123",
		SourceID: "source-456",
		URI:      "file:///empty.txt",
		Title:    "Empty Metadata",
		Metadata: map[string]any{},
	}

	assert.NotNil(t, doc.Metadata)
	assert.Empty(t, doc.Metadata)
}

// TestDocument_NilMetadata tests Document with nil metadata
func TestDocument_NilMetadata(t *testing.T) {
	doc := Document{
		ID:       "doc-123",
		SourceID: "source-456",
		URI:      "file:///nil.txt",
		Title:    "Nil Metadata",
		Metadata: nil,
	}

	assert.Nil(t, doc.Metadata)
}

// TestDocument_ComplexMetadata tests Document with complex metadata types
func TestDocument_ComplexMetadata(t *testing.T) {
	doc := Document{
		ID:       "doc-123",
		SourceID: "source-456",
		URI:      "file:///complex.txt",
		Title:    "Complex Metadata",
		Metadata: map[string]any{
			"string": "value",
			"int":    42,
			"float":  3.14,
			"bool":   true,
			"array":  []string{"a", "b", "c"},
			"nested": map[string]any{"key": "value"},
			"nil":    nil,
		},
	}

	assert.Equal(t, "value", doc.Metadata["string"])
	assert.Equal(t, 42, doc.Metadata["int"])
	assert.Equal(t, 3.14, doc.Metadata["float"])
	assert.Equal(t, true, doc.Metadata["bool"])
	assert.IsType(t, []string{}, doc.Metadata["array"])
	assert.IsType(t, map[string]any{}, doc.Metadata["nested"])
	assert.Nil(t, doc.Metadata["nil"])
}

// TestDocument_URIFormats tests various URI formats
func TestDocument_URIFormats(t *testing.T) {
	tests := []struct {
		name string
		uri  string
	}{
		{"file path", "file:///path/to/file.txt"},
		{"http url", "https://example.com/document"},
		{"drive url", "drive://file-id-123"},
		{"email", "gmail://message-id-456"},
		{"relative path", "documents/file.pdf"},
		{"empty", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := Document{
				ID:       "doc-123",
				SourceID: "source-456",
				URI:      tt.uri,
				Title:    "Test",
			}
			assert.Equal(t, tt.uri, doc.URI)
		})
	}
}

// TestChunk_Fields tests Chunk structure fields
func TestChunk_Fields(t *testing.T) {
	chunk := Chunk{
		ID:         "chunk-123",
		DocumentID: "doc-456",
		Content:    "This is the chunk content.",
		Position:   0,
		Embedding:  []float32{0.1, 0.2, 0.3},
		Metadata:   map[string]any{"section": "introduction"},
	}

	assert.Equal(t, "chunk-123", chunk.ID)
	assert.Equal(t, "doc-456", chunk.DocumentID)
	assert.Equal(t, "This is the chunk content.", chunk.Content)
	assert.Equal(t, 0, chunk.Position)
	assert.Equal(t, []float32{0.1, 0.2, 0.3}, chunk.Embedding)
	assert.Equal(t, "introduction", chunk.Metadata["section"])
}

// TestChunk_Positions tests various chunk positions
func TestChunk_Positions(t *testing.T) {
	tests := []struct {
		name     string
		position int
	}{
		{"first chunk", 0},
		{"second chunk", 1},
		{"middle chunk", 50},
		{"large position", 9999},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunk := Chunk{
				ID:         "chunk-123",
				DocumentID: "doc-456",
				Position:   tt.position,
			}
			assert.Equal(t, tt.position, chunk.Position)
		})
	}
}

// TestChunk_EmptyContent tests chunk with empty content
func TestChunk_EmptyContent(t *testing.T) {
	chunk := Chunk{
		ID:         "chunk-123",
		DocumentID: "doc-456",
		Content:    "",
		Position:   0,
	}

	assert.Empty(t, chunk.Content)
}

// TestChunk_LongContent tests chunk with long content
func TestChunk_LongContent(t *testing.T) {
	longContent := string(make([]byte, 10000))
	chunk := Chunk{
		ID:         "chunk-123",
		DocumentID: "doc-456",
		Content:    longContent,
		Position:   0,
	}

	assert.Len(t, chunk.Content, 10000)
}

// TestChunk_NoEmbedding tests chunk without embedding
func TestChunk_NoEmbedding(t *testing.T) {
	chunk := Chunk{
		ID:         "chunk-123",
		DocumentID: "doc-456",
		Content:    "Content without embedding",
		Position:   0,
		Embedding:  nil,
	}

	assert.Nil(t, chunk.Embedding)
}

// TestChunk_EmptyEmbedding tests chunk with empty embedding slice
func TestChunk_EmptyEmbedding(t *testing.T) {
	chunk := Chunk{
		ID:         "chunk-123",
		DocumentID: "doc-456",
		Content:    "Content with empty embedding",
		Position:   0,
		Embedding:  []float32{},
	}

	assert.NotNil(t, chunk.Embedding)
	assert.Empty(t, chunk.Embedding)
}

// TestChunk_LargeEmbedding tests chunk with large embedding vector
func TestChunk_LargeEmbedding(t *testing.T) {
	// 1536 dimensions (OpenAI text-embedding-3-small size)
	embedding := make([]float32, 1536)
	for i := range embedding {
		embedding[i] = float32(i) * 0.001
	}

	chunk := Chunk{
		ID:         "chunk-123",
		DocumentID: "doc-456",
		Content:    "Content with large embedding",
		Position:   0,
		Embedding:  embedding,
	}

	assert.Len(t, chunk.Embedding, 1536)
	assert.Equal(t, float32(0.0), chunk.Embedding[0])
	// Use InDelta for floating point comparison
	assert.InDelta(t, 1.535, chunk.Embedding[1535], 0.0001)
}

// TestChunk_MetadataTypes tests various metadata types in chunks
func TestChunk_MetadataTypes(t *testing.T) {
	chunk := Chunk{
		ID:         "chunk-123",
		DocumentID: "doc-456",
		Content:    "Test content",
		Position:   0,
		Metadata: map[string]any{
			"section":    "introduction",
			"page":       1,
			"confidence": 0.95,
			"tags":       []string{"important", "technical"},
		},
	}

	assert.Equal(t, "introduction", chunk.Metadata["section"])
	assert.Equal(t, 1, chunk.Metadata["page"])
	assert.Equal(t, 0.95, chunk.Metadata["confidence"])
	assert.IsType(t, []string{}, chunk.Metadata["tags"])
}

// TestDocument_TimeFields tests document time fields
func TestDocument_TimeFields(t *testing.T) {
	created := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	updated := time.Date(2024, 1, 2, 14, 30, 0, 0, time.UTC)

	doc := Document{
		ID:        "doc-123",
		SourceID:  "source-456",
		URI:       "file:///test.txt",
		Title:     "Test",
		CreatedAt: created,
		UpdatedAt: updated,
	}

	assert.Equal(t, created, doc.CreatedAt)
	assert.Equal(t, updated, doc.UpdatedAt)
	assert.True(t, doc.UpdatedAt.After(doc.CreatedAt))
}

// TestDocument_ZeroTimeFields tests document with zero time values
func TestDocument_ZeroTimeFields(t *testing.T) {
	doc := Document{
		ID:        "doc-123",
		SourceID:  "source-456",
		URI:       "file:///test.txt",
		Title:     "Test",
		CreatedAt: time.Time{},
		UpdatedAt: time.Time{},
	}

	assert.True(t, doc.CreatedAt.IsZero())
	assert.True(t, doc.UpdatedAt.IsZero())
}

// TestChunk_NilMetadata tests chunk with nil metadata
func TestChunk_NilMetadata(t *testing.T) {
	chunk := Chunk{
		ID:         "chunk-123",
		DocumentID: "doc-456",
		Content:    "Test content",
		Position:   0,
		Metadata:   nil,
	}

	assert.Nil(t, chunk.Metadata)
}

// TestDocument_MultipleChunks tests relationship between document and multiple chunks
func TestDocument_MultipleChunks(t *testing.T) {
	docID := "doc-123"

	chunks := []Chunk{
		{ID: "chunk-1", DocumentID: docID, Content: "First chunk", Position: 0},
		{ID: "chunk-2", DocumentID: docID, Content: "Second chunk", Position: 1},
		{ID: "chunk-3", DocumentID: docID, Content: "Third chunk", Position: 2},
	}

	// Verify all chunks reference the same document
	for _, chunk := range chunks {
		assert.Equal(t, docID, chunk.DocumentID)
	}

	// Verify positions are sequential
	for i, chunk := range chunks {
		assert.Equal(t, i, chunk.Position)
	}
}

// TestDocument_HierarchicalStructure tests parent-child document relationships
func TestDocument_HierarchicalStructure(t *testing.T) {
	parentID := "parent-doc"

	parent := Document{
		ID:       parentID,
		SourceID: "source-123",
		URI:      "file:///parent.pdf",
		Title:    "Parent Document",
		ParentID: nil,
	}

	child1ID := parentID
	child1 := Document{
		ID:       "child-1",
		SourceID: "source-123",
		URI:      "file:///parent.pdf#section1",
		Title:    "Child 1",
		ParentID: &child1ID,
	}

	child2ID := parentID
	child2 := Document{
		ID:       "child-2",
		SourceID: "source-123",
		URI:      "file:///parent.pdf#section2",
		Title:    "Child 2",
		ParentID: &child2ID,
	}

	assert.Nil(t, parent.ParentID)
	require.NotNil(t, child1.ParentID)
	require.NotNil(t, child2.ParentID)
	assert.Equal(t, parentID, *child1.ParentID)
	assert.Equal(t, parentID, *child2.ParentID)
}
