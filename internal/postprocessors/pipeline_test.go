package postprocessors

import (
	"context"
	"errors"
	"testing"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

// mockProcessor is a test processor that returns predefined chunks.
type mockProcessor struct {
	name   string
	chunks []domain.Chunk
	err    error
}

func (m *mockProcessor) Name() string {
	return m.name
}

func (m *mockProcessor) Process(_ context.Context, _ *domain.Document, chunks []domain.Chunk) ([]domain.Chunk, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.chunks != nil {
		return m.chunks, nil
	}
	return chunks, nil
}

func TestNewPipeline(t *testing.T) {
	p := NewPipeline()
	if p == nil {
		t.Fatal("expected non-nil pipeline")
	}
	if p.Len() != 0 {
		t.Errorf("expected 0 processors, got %d", p.Len())
	}
}

func TestPipeline_Add(t *testing.T) {
	p := NewPipeline()
	p.Add(&mockProcessor{name: "test"})

	if p.Len() != 1 {
		t.Errorf("expected 1 processor, got %d", p.Len())
	}
}

func TestPipeline_Process_NilDocument(t *testing.T) {
	p := NewPipeline()

	_, err := p.Process(context.Background(), nil)
	if err == nil {
		t.Error("expected error for nil document")
	}
}

func TestPipeline_Process_EmptyPipeline(t *testing.T) {
	p := NewPipeline()
	doc := &domain.Document{
		ID:      "test-doc",
		Content: "test content",
	}

	chunks, err := p.Process(context.Background(), doc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if chunks != nil {
		t.Errorf("expected nil chunks from empty pipeline, got %v", chunks)
	}
}

func TestPipeline_Process_SingleProcessor(t *testing.T) {
	expectedChunks := []domain.Chunk{
		{ID: "chunk-1", Content: "test"},
	}

	p := NewPipeline(&mockProcessor{
		name:   "chunker",
		chunks: expectedChunks,
	})

	doc := &domain.Document{
		ID:      "test-doc",
		Content: "test content",
	}

	chunks, err := p.Process(context.Background(), doc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chunks) != len(expectedChunks) {
		t.Errorf("expected %d chunks, got %d", len(expectedChunks), len(chunks))
	}
}

func TestPipeline_Process_MultipleProcessors(t *testing.T) {
	firstChunks := []domain.Chunk{
		{ID: "chunk-1", Content: "first"},
	}
	secondChunks := []domain.Chunk{
		{ID: "chunk-1", Content: "modified"},
		{ID: "chunk-2", Content: "added"},
	}

	p := NewPipeline(
		&mockProcessor{name: "first", chunks: firstChunks},
		&mockProcessor{name: "second", chunks: secondChunks},
	)

	doc := &domain.Document{
		ID:      "test-doc",
		Content: "test content",
	}

	chunks, err := p.Process(context.Background(), doc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chunks) != len(secondChunks) {
		t.Errorf("expected %d chunks, got %d", len(secondChunks), len(chunks))
	}
}

func TestPipeline_Process_ProcessorError(t *testing.T) {
	expectedErr := errors.New("processor failed")

	p := NewPipeline(&mockProcessor{
		name: "failing",
		err:  expectedErr,
	})

	doc := &domain.Document{
		ID:      "test-doc",
		Content: "test content",
	}

	_, err := p.Process(context.Background(), doc)
	if err == nil {
		t.Error("expected error from failing processor")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected wrapped error, got: %v", err)
	}
}

func TestPipeline_Process_PassthroughProcessor(t *testing.T) {
	initialChunks := []domain.Chunk{
		{ID: "chunk-1", Content: "test"},
	}

	p := NewPipeline(
		&mockProcessor{name: "chunker", chunks: initialChunks},
		&mockProcessor{name: "passthrough"}, // Returns received chunks unchanged
	)

	doc := &domain.Document{
		ID:      "test-doc",
		Content: "test content",
	}

	chunks, err := p.Process(context.Background(), doc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chunks) != len(initialChunks) {
		t.Errorf("expected %d chunks, got %d", len(initialChunks), len(chunks))
	}
}
