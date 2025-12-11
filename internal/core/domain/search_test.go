package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSearchOptions_Fields tests SearchOptions structure fields
func TestSearchOptions_Fields(t *testing.T) {
	opts := SearchOptions{
		Limit:     10,
		Offset:    0,
		SourceIDs: []string{"source-1", "source-2"},
		Semantic:  true,
		Hybrid:    false,
	}

	assert.Equal(t, 10, opts.Limit)
	assert.Equal(t, 0, opts.Offset)
	assert.Len(t, opts.SourceIDs, 2)
	assert.True(t, opts.Semantic)
	assert.False(t, opts.Hybrid)
}

// TestSearchOptions_DefaultValues tests SearchOptions with zero values
func TestSearchOptions_DefaultValues(t *testing.T) {
	opts := SearchOptions{}

	assert.Equal(t, 0, opts.Limit)
	assert.Equal(t, 0, opts.Offset)
	assert.Nil(t, opts.SourceIDs)
	assert.False(t, opts.Semantic)
	assert.False(t, opts.Hybrid)
}

// TestSearchOptions_NoSourceFilter tests search without source filtering
func TestSearchOptions_NoSourceFilter(t *testing.T) {
	opts := SearchOptions{
		Limit:     20,
		Offset:    0,
		SourceIDs: nil,
		Semantic:  false,
		Hybrid:    false,
	}

	assert.Nil(t, opts.SourceIDs)
}

// TestSearchOptions_EmptySourceFilter tests search with empty source list
func TestSearchOptions_EmptySourceFilter(t *testing.T) {
	opts := SearchOptions{
		Limit:     20,
		Offset:    0,
		SourceIDs: []string{},
		Semantic:  false,
		Hybrid:    false,
	}

	assert.NotNil(t, opts.SourceIDs)
	assert.Empty(t, opts.SourceIDs)
}

// TestSearchOptions_Pagination tests various pagination scenarios
func TestSearchOptions_Pagination(t *testing.T) {
	tests := []struct {
		name   string
		limit  int
		offset int
	}{
		{"first page", 10, 0},
		{"second page", 10, 10},
		{"third page", 10, 20},
		{"large page", 100, 0},
		{"small page", 5, 0},
		{"offset without limit", 0, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := SearchOptions{
				Limit:  tt.limit,
				Offset: tt.offset,
			}
			assert.Equal(t, tt.limit, opts.Limit)
			assert.Equal(t, tt.offset, opts.Offset)
		})
	}
}

// TestSearchOptions_SemanticOnly tests semantic-only search
func TestSearchOptions_SemanticOnly(t *testing.T) {
	opts := SearchOptions{
		Limit:    10,
		Offset:   0,
		Semantic: true,
		Hybrid:   false,
	}

	assert.True(t, opts.Semantic)
	assert.False(t, opts.Hybrid)
}

// TestSearchOptions_HybridOnly tests hybrid-only search
func TestSearchOptions_HybridOnly(t *testing.T) {
	opts := SearchOptions{
		Limit:    10,
		Offset:   0,
		Semantic: false,
		Hybrid:   true,
	}

	assert.False(t, opts.Semantic)
	assert.True(t, opts.Hybrid)
}

// TestSearchOptions_BothFlags tests both semantic and hybrid flags set
func TestSearchOptions_BothFlags(t *testing.T) {
	opts := SearchOptions{
		Limit:    10,
		Offset:   0,
		Semantic: true,
		Hybrid:   true,
	}

	assert.True(t, opts.Semantic)
	assert.True(t, opts.Hybrid)
}

// TestSearchOptions_TextOnly tests text-only search (no flags)
func TestSearchOptions_TextOnly(t *testing.T) {
	opts := SearchOptions{
		Limit:    10,
		Offset:   0,
		Semantic: false,
		Hybrid:   false,
	}

	assert.False(t, opts.Semantic)
	assert.False(t, opts.Hybrid)
}

// TestSearchOptions_MultipleSourceIDs tests filtering by multiple sources
func TestSearchOptions_MultipleSourceIDs(t *testing.T) {
	opts := SearchOptions{
		Limit:     10,
		Offset:    0,
		SourceIDs: []string{"src-1", "src-2", "src-3", "src-4", "src-5"},
	}

	assert.Len(t, opts.SourceIDs, 5)
	assert.Contains(t, opts.SourceIDs, "src-1")
	assert.Contains(t, opts.SourceIDs, "src-5")
}

// TestSearchResult_Fields tests SearchResult structure fields
func TestSearchResult_Fields(t *testing.T) {
	result := SearchResult{
		Document: Document{
			ID:    "doc-123",
			Title: "Test Document",
		},
		Chunk: Chunk{
			ID:      "chunk-456",
			Content: "Matching content",
		},
		Score:      0.85,
		Highlights: []string{"matching <b>term</b>"},
	}

	assert.Equal(t, "doc-123", result.Document.ID)
	assert.Equal(t, "chunk-456", result.Chunk.ID)
	assert.Equal(t, 0.85, result.Score)
	assert.Len(t, result.Highlights, 1)
}

// TestSearchResult_ScoreValues tests various score values
func TestSearchResult_ScoreValues(t *testing.T) {
	tests := []struct {
		name  string
		score float64
	}{
		{"perfect match", 1.0},
		{"high relevance", 0.9},
		{"medium relevance", 0.5},
		{"low relevance", 0.1},
		{"zero score", 0.0},
		{"negative score", -0.5}, // Edge case
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SearchResult{
				Score: tt.score,
			}
			assert.Equal(t, tt.score, result.Score)
		})
	}
}

// TestSearchResult_NoHighlights tests result without highlights
func TestSearchResult_NoHighlights(t *testing.T) {
	result := SearchResult{
		Document:   Document{ID: "doc-123"},
		Chunk:      Chunk{ID: "chunk-456"},
		Score:      0.8,
		Highlights: nil,
	}

	assert.Nil(t, result.Highlights)
}

// TestSearchResult_EmptyHighlights tests result with empty highlights slice
func TestSearchResult_EmptyHighlights(t *testing.T) {
	result := SearchResult{
		Document:   Document{ID: "doc-123"},
		Chunk:      Chunk{ID: "chunk-456"},
		Score:      0.8,
		Highlights: []string{},
	}

	assert.NotNil(t, result.Highlights)
	assert.Empty(t, result.Highlights)
}

// TestSearchResult_MultipleHighlights tests result with multiple highlights
func TestSearchResult_MultipleHighlights(t *testing.T) {
	result := SearchResult{
		Document: Document{ID: "doc-123"},
		Chunk:    Chunk{ID: "chunk-456"},
		Score:    0.9,
		Highlights: []string{
			"First <b>highlight</b> snippet",
			"Second <b>highlight</b> snippet",
			"Third <b>highlight</b> snippet",
		},
	}

	assert.Len(t, result.Highlights, 3)
	for _, highlight := range result.Highlights {
		assert.Contains(t, highlight, "<b>highlight</b>")
	}
}

// TestSearchResult_LongHighlight tests result with long highlight text
func TestSearchResult_LongHighlight(t *testing.T) {
	longHighlight := string(make([]byte, 1000))
	result := SearchResult{
		Document:   Document{ID: "doc-123"},
		Chunk:      Chunk{ID: "chunk-456"},
		Score:      0.7,
		Highlights: []string{longHighlight},
	}

	assert.Len(t, result.Highlights[0], 1000)
}

// TestSearchResult_CompleteDocument tests result with complete document data
func TestSearchResult_CompleteDocument(t *testing.T) {
	result := SearchResult{
		Document: Document{
			ID:       "doc-123",
			SourceID: "source-456",
			URI:      "file:///test.txt",
			Title:    "Test Document",
			Metadata: map[string]any{"author": "Test Author"},
		},
		Chunk: Chunk{
			ID:         "chunk-789",
			DocumentID: "doc-123",
			Content:    "This is the matching chunk content",
			Position:   0,
		},
		Score:      0.95,
		Highlights: []string{"matching <b>chunk</b> content"},
	}

	assert.Equal(t, "doc-123", result.Document.ID)
	assert.Equal(t, "doc-123", result.Chunk.DocumentID)
	assert.Equal(t, "Test Author", result.Document.Metadata["author"])
}

// TestSearchOptions_NegativeValues tests edge cases with negative values
func TestSearchOptions_NegativeValues(t *testing.T) {
	opts := SearchOptions{
		Limit:  -1,
		Offset: -10,
	}

	// Document edge case behavior
	assert.Equal(t, -1, opts.Limit)
	assert.Equal(t, -10, opts.Offset)
}

// TestSearchOptions_LargeValues tests very large pagination values
func TestSearchOptions_LargeValues(t *testing.T) {
	opts := SearchOptions{
		Limit:  1000000,
		Offset: 5000000,
	}

	assert.Equal(t, 1000000, opts.Limit)
	assert.Equal(t, 5000000, opts.Offset)
}

// TestSearchResult_ZeroScore tests result with zero score
func TestSearchResult_ZeroScore(t *testing.T) {
	result := SearchResult{
		Document:   Document{ID: "doc-123"},
		Chunk:      Chunk{ID: "chunk-456"},
		Score:      0.0,
		Highlights: []string{"some highlight"},
	}

	assert.Equal(t, 0.0, result.Score)
}

// TestSearchOptions_SingleSource tests filtering by single source
func TestSearchOptions_SingleSource(t *testing.T) {
	opts := SearchOptions{
		Limit:     10,
		Offset:    0,
		SourceIDs: []string{"single-source"},
	}

	assert.Len(t, opts.SourceIDs, 1)
	assert.Equal(t, "single-source", opts.SourceIDs[0])
}

// TestSearchResult_HTMLHighlights tests highlights with HTML tags
func TestSearchResult_HTMLHighlights(t *testing.T) {
	result := SearchResult{
		Document: Document{ID: "doc-123"},
		Chunk:    Chunk{ID: "chunk-456"},
		Score:    0.8,
		Highlights: []string{
			"Text with <b>bold</b> highlight",
			"Text with <em>emphasis</em> highlight",
			"Text with <mark>marked</mark> highlight",
		},
	}

	assert.Len(t, result.Highlights, 3)
	assert.Contains(t, result.Highlights[0], "<b>")
	assert.Contains(t, result.Highlights[1], "<em>")
	assert.Contains(t, result.Highlights[2], "<mark>")
}

// TestSearchResult_ChunkPosition tests that chunk position is preserved
func TestSearchResult_ChunkPosition(t *testing.T) {
	result := SearchResult{
		Document: Document{ID: "doc-123"},
		Chunk: Chunk{
			ID:       "chunk-456",
			Position: 42,
			Content:  "Test content",
		},
		Score: 0.8,
	}

	assert.Equal(t, 42, result.Chunk.Position)
}

// TestSearchOptions_DuplicateSourceIDs tests duplicate source IDs
func TestSearchOptions_DuplicateSourceIDs(t *testing.T) {
	opts := SearchOptions{
		Limit:     10,
		Offset:    0,
		SourceIDs: []string{"src-1", "src-2", "src-1", "src-3", "src-2"},
	}

	// Duplicates are preserved in the slice (filtering is application logic)
	assert.Len(t, opts.SourceIDs, 5)
}
