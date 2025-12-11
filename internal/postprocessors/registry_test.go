package postprocessors

import (
	"context"
	"testing"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
)

// registryMockProcessor is a simple mock for testing registry functionality.
type registryMockProcessor struct {
	name string
}

func (m *registryMockProcessor) Name() string { return m.name }
func (m *registryMockProcessor) Process(_ context.Context, _ *domain.Document, chunks []domain.Chunk) ([]domain.Chunk, error) {
	return chunks, nil
}

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	if r == nil {
		t.Fatal("NewRegistry returned nil")
	}
	if len(r.builders) != 0 {
		t.Errorf("expected empty builders, got %d", len(r.builders))
	}
}

func TestRegistry_Register(t *testing.T) {
	r := NewRegistry()

	builder := func(_ map[string]any) (driven.PostProcessor, error) {
		return &registryMockProcessor{name: "test"}, nil
	}

	r.Register("test", builder)

	if !r.Has("test") {
		t.Error("expected 'test' to be registered")
	}
}

func TestRegistry_Build_Success(t *testing.T) {
	r := NewRegistry()

	builder := func(cfg map[string]any) (driven.PostProcessor, error) {
		name := "default"
		if n, ok := cfg["name"].(string); ok {
			name = n
		}
		return &registryMockProcessor{name: name}, nil
	}

	r.Register("test", builder)

	proc, err := r.Build("test", map[string]any{"name": "custom"})
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if proc.Name() != "custom" {
		t.Errorf("expected name 'custom', got %q", proc.Name())
	}
}

func TestRegistry_Build_UnknownProcessor(t *testing.T) {
	r := NewRegistry()

	_, err := r.Build("unknown", nil)
	if err == nil {
		t.Error("expected error for unknown processor")
	}
}

func TestRegistry_Has(t *testing.T) {
	r := NewRegistry()

	if r.Has("nonexistent") {
		t.Error("expected Has to return false for nonexistent processor")
	}

	r.Register("exists", func(_ map[string]any) (driven.PostProcessor, error) {
		return &registryMockProcessor{name: "exists"}, nil
	})

	if !r.Has("exists") {
		t.Error("expected Has to return true for registered processor")
	}
}

func TestRegistry_Names(t *testing.T) {
	r := NewRegistry()

	names := r.Names()
	if len(names) != 0 {
		t.Errorf("expected 0 names, got %d", len(names))
	}

	r.Register("alpha", func(_ map[string]any) (driven.PostProcessor, error) {
		return &registryMockProcessor{name: "alpha"}, nil
	})
	r.Register("beta", func(_ map[string]any) (driven.PostProcessor, error) {
		return &registryMockProcessor{name: "beta"}, nil
	})

	names = r.Names()
	if len(names) != 2 {
		t.Errorf("expected 2 names, got %d", len(names))
	}

	// Check both names are present (order may vary)
	nameSet := make(map[string]bool)
	for _, n := range names {
		nameSet[n] = true
	}
	if !nameSet["alpha"] || !nameSet["beta"] {
		t.Errorf("expected names alpha and beta, got %v", names)
	}
}

func TestRegisterDefaults(t *testing.T) {
	r := NewRegistry()
	RegisterDefaults(r)

	if !r.Has("chunker") {
		t.Error("expected 'chunker' to be registered after RegisterDefaults")
	}
}

func TestBuildChunker_WithConfig(t *testing.T) {
	r := NewRegistry()
	RegisterDefaults(r)

	cfg := map[string]any{
		"chunk_size": 500,
		"overlap":    100,
	}

	proc, err := r.Build("chunker", cfg)
	if err != nil {
		t.Fatalf("Build chunker failed: %v", err)
	}

	if proc.Name() != "chunker" {
		t.Errorf("expected name 'chunker', got %q", proc.Name())
	}
}

func TestBuildChunker_WithNilConfig(t *testing.T) {
	r := NewRegistry()
	RegisterDefaults(r)

	proc, err := r.Build("chunker", nil)
	if err != nil {
		t.Fatalf("Build chunker with nil config failed: %v", err)
	}

	if proc.Name() != "chunker" {
		t.Errorf("expected name 'chunker', got %q", proc.Name())
	}
}

func TestGetIntFromConfig(t *testing.T) {
	tests := []struct {
		name     string
		cfg      map[string]any
		key      string
		expected int
	}{
		{"int value", map[string]any{"size": 100}, "size", 100},
		{"int64 value", map[string]any{"size": int64(200)}, "size", 200},
		{"float64 value", map[string]any{"size": float64(300)}, "size", 300},
		{"string value", map[string]any{"size": "400"}, "size", 0},
		{"missing key", map[string]any{"other": 100}, "size", 0},
		{"nil config", nil, "size", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getIntFromConfig(tt.cfg, tt.key)
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}
