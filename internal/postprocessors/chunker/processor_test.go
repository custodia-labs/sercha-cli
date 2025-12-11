package chunker

import (
	"context"
	"strings"
	"testing"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

func TestNew(t *testing.T) {
	t.Run("default values", func(t *testing.T) {
		p := New()
		if p.chunkSize != DefaultChunkSize {
			t.Errorf("expected chunkSize %d, got %d", DefaultChunkSize, p.chunkSize)
		}
		if p.overlap != DefaultChunkOverlap {
			t.Errorf("expected overlap %d, got %d", DefaultChunkOverlap, p.overlap)
		}
	})

	t.Run("custom chunk size", func(t *testing.T) {
		p := New(WithChunkSize(500))
		if p.chunkSize != 500 {
			t.Errorf("expected chunkSize 500, got %d", p.chunkSize)
		}
	})

	t.Run("custom overlap", func(t *testing.T) {
		p := New(WithOverlap(100))
		if p.overlap != 100 {
			t.Errorf("expected overlap 100, got %d", p.overlap)
		}
	})

	t.Run("overlap exceeds chunk size", func(t *testing.T) {
		p := New(WithChunkSize(100), WithOverlap(150))
		if p.overlap >= p.chunkSize {
			t.Error("overlap should be reduced when it exceeds chunk size")
		}
	})

	t.Run("zero values ignored", func(t *testing.T) {
		p := New(WithChunkSize(0), WithOverlap(-1))
		if p.chunkSize != DefaultChunkSize {
			t.Errorf("expected default chunkSize, got %d", p.chunkSize)
		}
		if p.overlap != DefaultChunkOverlap {
			t.Errorf("expected default overlap, got %d", p.overlap)
		}
	})
}

func TestProcessor_Name(t *testing.T) {
	p := New()
	if p.Name() != "chunker" {
		t.Errorf("expected name 'chunker', got '%s'", p.Name())
	}
}

func TestProcessor_Process_EmptyContent(t *testing.T) {
	p := New()
	doc := &domain.Document{
		ID:      "test-doc",
		Content: "",
	}

	chunks, err := p.Process(context.Background(), doc, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chunks) != 0 {
		t.Errorf("expected 0 chunks for empty content, got %d", len(chunks))
	}
}

func TestProcessor_Process_SmallContent(t *testing.T) {
	p := New(WithChunkSize(100), WithOverlap(20))
	doc := &domain.Document{
		ID:      "test-doc",
		Content: "This is a small piece of content.",
	}

	chunks, err := p.Process(context.Background(), doc, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chunks) != 1 {
		t.Errorf("expected 1 chunk for small content, got %d", len(chunks))
	}

	if chunks[0].DocumentID != doc.ID {
		t.Errorf("expected DocumentID '%s', got '%s'", doc.ID, chunks[0].DocumentID)
	}
	if chunks[0].Content != doc.Content {
		t.Errorf("expected content to match document content")
	}
	if chunks[0].Position != 0 {
		t.Errorf("expected position 0, got %d", chunks[0].Position)
	}
}

func TestProcessor_Process_LargeContent(t *testing.T) {
	p := New(WithChunkSize(100), WithOverlap(20))

	// Create content that spans multiple chunks
	content := strings.Repeat("x", 250) // Should create 3-4 chunks with overlap
	doc := &domain.Document{
		ID:      "test-doc",
		Content: content,
	}

	chunks, err := p.Process(context.Background(), doc, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chunks) < 2 {
		t.Errorf("expected multiple chunks, got %d", len(chunks))
	}

	// Verify chunk IDs are unique
	seenIDs := make(map[string]bool)
	for _, chunk := range chunks {
		if seenIDs[chunk.ID] {
			t.Errorf("duplicate chunk ID: %s", chunk.ID)
		}
		seenIDs[chunk.ID] = true
	}

	// Verify positions are sequential
	for i, chunk := range chunks {
		if chunk.Position != i {
			t.Errorf("expected position %d, got %d", i, chunk.Position)
		}
	}

	// Verify all chunks have DocumentID set
	for _, chunk := range chunks {
		if chunk.DocumentID != doc.ID {
			t.Errorf("expected DocumentID '%s', got '%s'", doc.ID, chunk.DocumentID)
		}
	}

	// Verify first chunk is full size
	if len(chunks[0].Content) != 100 {
		t.Errorf("expected first chunk size 100, got %d", len(chunks[0].Content))
	}
}

func TestProcessor_Process_ExactChunkSize(t *testing.T) {
	p := New(WithChunkSize(50), WithOverlap(0))

	content := strings.Repeat("a", 100) // Exactly 2 chunks
	doc := &domain.Document{
		ID:      "test-doc",
		Content: content,
	}

	chunks, err := p.Process(context.Background(), doc, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chunks) != 2 {
		t.Errorf("expected 2 chunks, got %d", len(chunks))
	}
}

func TestProcessor_Process_OverlapContent(t *testing.T) {
	p := New(WithChunkSize(10), WithOverlap(3))

	content := "0123456789ABCDEFGHIJ" // 20 chars
	doc := &domain.Document{
		ID:      "test-doc",
		Content: content,
	}

	chunks, err := p.Process(context.Background(), doc, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// With size 10 and overlap 3, step is 7
	// Chunks should be: 0-9, 7-16, 14-20
	if len(chunks) < 2 {
		t.Errorf("expected at least 2 chunks with overlap, got %d", len(chunks))
	}

	// First chunk should be 10 chars
	if len(chunks[0].Content) != 10 {
		t.Errorf("expected first chunk length 10, got %d", len(chunks[0].Content))
	}
}

func TestProcessor_Process_IgnoresInputChunks(t *testing.T) {
	p := New(WithChunkSize(100))

	existingChunks := []domain.Chunk{
		{ID: "existing", Content: "should be ignored"},
	}

	doc := &domain.Document{
		ID:      "test-doc",
		Content: "New content to chunk",
	}

	chunks, err := p.Process(context.Background(), doc, existingChunks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should create new chunks, not return existing ones
	for _, chunk := range chunks {
		if chunk.ID == "existing" {
			t.Error("existing chunks should be ignored")
		}
	}
}

func TestProcessor_Process_MetadataInitialized(t *testing.T) {
	p := New(WithChunkSize(100))

	doc := &domain.Document{
		ID:      "test-doc",
		Content: "Test content",
	}

	chunks, err := p.Process(context.Background(), doc, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, chunk := range chunks {
		if chunk.Metadata == nil {
			t.Error("expected chunk Metadata to be initialized")
		}
	}
}
