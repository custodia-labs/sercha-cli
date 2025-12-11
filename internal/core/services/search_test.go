package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/custodia-labs/sercha-cli/internal/adapters/driven/storage/memory"
	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
)

// --- Mock implementations ---

// mockSearchEngine implements driven.SearchEngine for testing.
type mockSearchEngine struct {
	hits      []driven.SearchHit
	searchErr error
	indexErr  error
	deleteErr error
}

func (m *mockSearchEngine) Index(_ context.Context, _ domain.Chunk) error {
	return m.indexErr
}

func (m *mockSearchEngine) Delete(_ context.Context, _ string) error {
	return m.deleteErr
}

func (m *mockSearchEngine) Search(_ context.Context, _ string, limit int) ([]driven.SearchHit, error) {
	if m.searchErr != nil {
		return nil, m.searchErr
	}
	if limit > len(m.hits) {
		return m.hits, nil
	}
	return m.hits[:limit], nil
}

func (m *mockSearchEngine) Close() error {
	return nil
}

// mockVectorIndex implements driven.VectorIndex for testing.
type mockVectorIndex struct {
	hits      []driven.VectorHit
	searchErr error
	addErr    error
	deleteErr error
}

func (m *mockVectorIndex) Add(_ context.Context, _ string, _ []float32) error {
	return m.addErr
}

func (m *mockVectorIndex) Delete(_ context.Context, _ string) error {
	return m.deleteErr
}

func (m *mockVectorIndex) Search(_ context.Context, _ []float32, k int) ([]driven.VectorHit, error) {
	if m.searchErr != nil {
		return nil, m.searchErr
	}
	if k > len(m.hits) {
		return m.hits, nil
	}
	return m.hits[:k], nil
}

func (m *mockVectorIndex) Close() error {
	return nil
}

// mockEmbeddingService implements driven.EmbeddingService for testing.
type mockEmbeddingService struct {
	embedding []float32
	embedErr  error
	dims      int
}

func (m *mockEmbeddingService) Embed(_ context.Context, _ string) ([]float32, error) {
	if m.embedErr != nil {
		return nil, m.embedErr
	}
	return m.embedding, nil
}

func (m *mockEmbeddingService) EmbedBatch(_ context.Context, texts []string) ([][]float32, error) {
	if m.embedErr != nil {
		return nil, m.embedErr
	}
	result := make([][]float32, len(texts))
	for i := range texts {
		result[i] = m.embedding
	}
	return result, nil
}

func (m *mockEmbeddingService) Dimensions() int {
	if m.dims > 0 {
		return m.dims
	}
	return 384
}

func (m *mockEmbeddingService) ModelName() string {
	return "mock-embed"
}

func (m *mockEmbeddingService) Ping(_ context.Context) error {
	return nil
}

func (m *mockEmbeddingService) Close() error {
	return nil
}

// mockLLMService implements driven.LLMService for testing.
type mockLLMService struct {
	rewriteResult string
	rewriteErr    error
}

func (m *mockLLMService) Generate(_ context.Context, _ string, _ driven.GenerateOptions) (string, error) {
	return "", nil
}

func (m *mockLLMService) Chat(_ context.Context, _ []driven.ChatMessage, _ driven.ChatOptions) (string, error) {
	return "", nil
}

func (m *mockLLMService) RewriteQuery(_ context.Context, query string) (string, error) {
	if m.rewriteErr != nil {
		return "", m.rewriteErr
	}
	if m.rewriteResult != "" {
		return m.rewriteResult, nil
	}
	return query + " expanded", nil
}

func (m *mockLLMService) Summarise(_ context.Context, _ string, _ int) (string, error) {
	return "", nil
}

func (m *mockLLMService) ModelName() string {
	return "mock-llm"
}

func (m *mockLLMService) Ping(_ context.Context) error {
	return nil
}

func (m *mockLLMService) Close() error {
	return nil
}

// --- Test helpers ---

func setupTestDocStore(t *testing.T) *memory.DocumentStore {
	t.Helper()
	store := memory.NewDocumentStore()
	ctx := context.Background()
	now := time.Now()

	// Create test documents and chunks.
	docs := []struct {
		id       string
		sourceID string
		title    string
		content  string
	}{
		{"doc-1", "src-1", "Getting Started with Sercha", "Sercha is a search engine for your files."},
		{"doc-2", "src-1", "Configuration Guide", "Configure Sercha using the settings command."},
		{"doc-3", "src-2", "API Reference", "The API provides search endpoints and document management."},
	}

	for _, d := range docs {
		doc := &domain.Document{
			ID:        d.id,
			SourceID:  d.sourceID,
			URI:       "file://" + d.id,
			Title:     d.title,
			CreatedAt: now,
			UpdatedAt: now,
		}
		require.NoError(t, store.SaveDocument(ctx, doc))

		chunk := domain.Chunk{
			ID:         "chunk-" + d.id,
			DocumentID: d.id,
			Content:    d.content,
			Position:   0,
		}
		require.NoError(t, store.SaveChunks(ctx, []domain.Chunk{chunk}))
	}

	return store
}

func createTestHits() []driven.SearchHit {
	return []driven.SearchHit{
		{ChunkID: "chunk-doc-1", Score: 0.9},
		{ChunkID: "chunk-doc-2", Score: 0.8},
		{ChunkID: "chunk-doc-3", Score: 0.7},
	}
}

func createTestVectorHits() []driven.VectorHit {
	return []driven.VectorHit{
		{ChunkID: "chunk-doc-2", Similarity: 0.95},
		{ChunkID: "chunk-doc-1", Similarity: 0.85},
		{ChunkID: "chunk-doc-3", Similarity: 0.75},
	}
}

// --- Tests ---

func TestNewSearchService(t *testing.T) {
	docStore := memory.NewDocumentStore()
	service := NewSearchService(docStore, nil, nil, nil, nil)

	require.NotNil(t, service)
	assert.NotNil(t, service.docStore)
}

func TestSearchService_Search_EmptyQuery(t *testing.T) {
	docStore := setupTestDocStore(t)
	searchEngine := &mockSearchEngine{hits: createTestHits()}
	service := NewSearchService(docStore, searchEngine, nil, nil, nil)
	ctx := context.Background()

	results, err := service.Search(ctx, "", domain.SearchOptions{})

	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestSearchService_Search_WhitespaceQuery(t *testing.T) {
	docStore := setupTestDocStore(t)
	searchEngine := &mockSearchEngine{hits: createTestHits()}
	service := NewSearchService(docStore, searchEngine, nil, nil, nil)
	ctx := context.Background()

	results, err := service.Search(ctx, "   \t\n  ", domain.SearchOptions{})

	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestSearchService_Search_KeywordOnly(t *testing.T) {
	docStore := setupTestDocStore(t)
	searchEngine := &mockSearchEngine{hits: createTestHits()}
	service := NewSearchService(docStore, searchEngine, nil, nil, nil)
	ctx := context.Background()

	results, err := service.Search(ctx, "sercha", domain.SearchOptions{})

	require.NoError(t, err)
	assert.Len(t, results, 3)

	// Verify results have correct structure.
	for _, r := range results {
		assert.NotEmpty(t, r.Document.ID)
		assert.NotEmpty(t, r.Document.Title)
		assert.Greater(t, r.Score, 0.0)
	}
}

func TestSearchService_Search_HybridMode(t *testing.T) {
	docStore := setupTestDocStore(t)
	searchEngine := &mockSearchEngine{hits: createTestHits()}
	vectorIndex := &mockVectorIndex{hits: createTestVectorHits()}
	embedService := &mockEmbeddingService{embedding: make([]float32, 384)}
	service := NewSearchService(docStore, searchEngine, vectorIndex, embedService, nil)
	ctx := context.Background()

	results, err := service.Search(ctx, "sercha configuration", domain.SearchOptions{
		Hybrid: true,
	})

	require.NoError(t, err)
	assert.NotEmpty(t, results)
}

func TestSearchService_Search_SemanticMode(t *testing.T) {
	docStore := setupTestDocStore(t)
	searchEngine := &mockSearchEngine{hits: createTestHits()}
	vectorIndex := &mockVectorIndex{hits: createTestVectorHits()}
	embedService := &mockEmbeddingService{embedding: make([]float32, 384)}
	service := NewSearchService(docStore, searchEngine, vectorIndex, embedService, nil)
	ctx := context.Background()

	results, err := service.Search(ctx, "how to configure", domain.SearchOptions{
		Semantic: true,
	})

	require.NoError(t, err)
	assert.NotEmpty(t, results)
}

func TestSearchService_Search_FullMode(t *testing.T) {
	docStore := setupTestDocStore(t)
	searchEngine := &mockSearchEngine{hits: createTestHits()}
	vectorIndex := &mockVectorIndex{hits: createTestVectorHits()}
	embedService := &mockEmbeddingService{embedding: make([]float32, 384)}
	llmService := &mockLLMService{rewriteResult: "sercha configuration guide setup"}
	service := NewSearchService(docStore, searchEngine, vectorIndex, embedService, llmService)
	ctx := context.Background()

	results, err := service.Search(ctx, "setup", domain.SearchOptions{})

	require.NoError(t, err)
	assert.NotEmpty(t, results)
}

func TestSearchService_Search_LLMAssistedMode(t *testing.T) {
	docStore := setupTestDocStore(t)
	searchEngine := &mockSearchEngine{hits: createTestHits()}
	llmService := &mockLLMService{rewriteResult: "sercha getting started guide"}
	service := NewSearchService(docStore, searchEngine, nil, nil, llmService)
	ctx := context.Background()

	results, err := service.Search(ctx, "start", domain.SearchOptions{})

	require.NoError(t, err)
	assert.NotEmpty(t, results)
}

func TestSearchService_Search_LimitOption(t *testing.T) {
	docStore := setupTestDocStore(t)
	searchEngine := &mockSearchEngine{hits: createTestHits()}
	service := NewSearchService(docStore, searchEngine, nil, nil, nil)
	ctx := context.Background()

	results, err := service.Search(ctx, "test", domain.SearchOptions{
		Limit: 1,
	})

	require.NoError(t, err)
	assert.Len(t, results, 1)
}

func TestSearchService_Search_OffsetOption(t *testing.T) {
	docStore := setupTestDocStore(t)
	searchEngine := &mockSearchEngine{hits: createTestHits()}
	service := NewSearchService(docStore, searchEngine, nil, nil, nil)
	ctx := context.Background()

	// Get all results first.
	allResults, err := service.Search(ctx, "test", domain.SearchOptions{})
	require.NoError(t, err)
	require.Len(t, allResults, 3)

	// Get with offset.
	offsetResults, err := service.Search(ctx, "test", domain.SearchOptions{
		Offset: 1,
	})
	require.NoError(t, err)
	assert.Len(t, offsetResults, 2)
}

func TestSearchService_Search_SourceIDFilter(t *testing.T) {
	docStore := setupTestDocStore(t)
	searchEngine := &mockSearchEngine{hits: createTestHits()}
	service := NewSearchService(docStore, searchEngine, nil, nil, nil)
	ctx := context.Background()

	results, err := service.Search(ctx, "test", domain.SearchOptions{
		SourceIDs: []string{"src-1"},
	})

	require.NoError(t, err)
	// Only docs from src-1 should be returned.
	for _, r := range results {
		assert.Equal(t, "src-1", r.Document.SourceID)
	}
}

func TestSearchService_Search_NoSearchEngine(t *testing.T) {
	docStore := setupTestDocStore(t)
	service := NewSearchService(docStore, nil, nil, nil, nil)
	ctx := context.Background()

	_, err := service.Search(ctx, "test", domain.SearchOptions{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "search engine unavailable")
}

func TestSearchService_Search_SearchEngineError(t *testing.T) {
	docStore := setupTestDocStore(t)
	searchEngine := &mockSearchEngine{searchErr: errors.New("search failed")}
	service := NewSearchService(docStore, searchEngine, nil, nil, nil)
	ctx := context.Background()

	_, err := service.Search(ctx, "test", domain.SearchOptions{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "search failed")
}

func TestSearchService_Search_VectorSearchError_Degrades(t *testing.T) {
	docStore := setupTestDocStore(t)
	searchEngine := &mockSearchEngine{hits: createTestHits()}
	vectorIndex := &mockVectorIndex{searchErr: errors.New("vector failed")}
	embedService := &mockEmbeddingService{embedding: make([]float32, 384)}
	service := NewSearchService(docStore, searchEngine, vectorIndex, embedService, nil)
	ctx := context.Background()

	// Hybrid should degrade to keyword-only when vector fails.
	results, err := service.Search(ctx, "test", domain.SearchOptions{
		Hybrid: true,
	})

	require.NoError(t, err)
	assert.NotEmpty(t, results) // Falls back to keyword results.
}

func TestSearchService_Search_EmbeddingError_Degrades(t *testing.T) {
	docStore := setupTestDocStore(t)
	searchEngine := &mockSearchEngine{hits: createTestHits()}
	vectorIndex := &mockVectorIndex{hits: createTestVectorHits()}
	embedService := &mockEmbeddingService{embedErr: errors.New("embed failed")}
	service := NewSearchService(docStore, searchEngine, vectorIndex, embedService, nil)
	ctx := context.Background()

	// Hybrid should degrade to keyword-only when embedding fails.
	results, err := service.Search(ctx, "test", domain.SearchOptions{
		Hybrid: true,
	})

	require.NoError(t, err)
	assert.NotEmpty(t, results)
}

func TestSearchService_Search_LLMError_Degrades(t *testing.T) {
	docStore := setupTestDocStore(t)
	searchEngine := &mockSearchEngine{hits: createTestHits()}
	llmService := &mockLLMService{rewriteErr: errors.New("llm failed")}
	service := NewSearchService(docStore, searchEngine, nil, nil, llmService)
	ctx := context.Background()

	// Should fall back to original query when LLM fails.
	results, err := service.Search(ctx, "test", domain.SearchOptions{})

	require.NoError(t, err)
	assert.NotEmpty(t, results)
}

func TestSearchService_Search_MissingChunk_Skipped(t *testing.T) {
	docStore := setupTestDocStore(t)
	// Include a non-existent chunk ID.
	hits := []driven.SearchHit{
		{ChunkID: "chunk-doc-1", Score: 0.9},
		{ChunkID: "non-existent-chunk", Score: 0.85},
		{ChunkID: "chunk-doc-2", Score: 0.8},
	}
	searchEngine := &mockSearchEngine{hits: hits}
	service := NewSearchService(docStore, searchEngine, nil, nil, nil)
	ctx := context.Background()

	results, err := service.Search(ctx, "test", domain.SearchOptions{})

	require.NoError(t, err)
	assert.Len(t, results, 2) // Missing chunk should be skipped.
}

func TestSearchService_Search_MissingDocument_Skipped(t *testing.T) {
	docStore := memory.NewDocumentStore()
	ctx := context.Background()
	now := time.Now()

	// Save chunk without document.
	chunk := domain.Chunk{
		ID:         "orphan-chunk",
		DocumentID: "missing-doc",
		Content:    "orphan content",
		Position:   0,
	}
	require.NoError(t, docStore.SaveChunks(ctx, []domain.Chunk{chunk}))

	// Save normal document and chunk.
	doc := &domain.Document{
		ID:        "doc-1",
		SourceID:  "src-1",
		Title:     "Test",
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, docStore.SaveDocument(ctx, doc))
	require.NoError(t, docStore.SaveChunks(ctx, []domain.Chunk{
		{ID: "chunk-1", DocumentID: "doc-1", Content: "test content"},
	}))

	hits := []driven.SearchHit{
		{ChunkID: "orphan-chunk", Score: 0.9},
		{ChunkID: "chunk-1", Score: 0.8},
	}
	searchEngine := &mockSearchEngine{hits: hits}
	service := NewSearchService(docStore, searchEngine, nil, nil, nil)

	results, err := service.Search(ctx, "test", domain.SearchOptions{})

	require.NoError(t, err)
	assert.Len(t, results, 1) // Orphan chunk skipped.
}

func TestSearchService_Search_Highlights(t *testing.T) {
	docStore := setupTestDocStore(t)
	searchEngine := &mockSearchEngine{hits: createTestHits()}
	service := NewSearchService(docStore, searchEngine, nil, nil, nil)
	ctx := context.Background()

	results, err := service.Search(ctx, "sercha", domain.SearchOptions{})

	require.NoError(t, err)
	require.NotEmpty(t, results)

	// First result should have highlights containing the query term.
	foundHighlight := false
	for _, r := range results {
		for _, h := range r.Highlights {
			if len(h) > 0 {
				foundHighlight = true
				break
			}
		}
	}
	assert.True(t, foundHighlight, "should have generated highlights")
}

func TestSearchService_effectiveMode(t *testing.T) {
	tests := []struct {
		name         string
		hasVector    bool
		hasEmbedding bool
		hasLLM       bool
		opts         domain.SearchOptions
		expectedMode domain.SearchMode
	}{
		{
			name:         "text only when nothing available",
			expectedMode: domain.SearchModeTextOnly,
		},
		{
			name:         "hybrid when vector available",
			hasVector:    true,
			hasEmbedding: true,
			expectedMode: domain.SearchModeHybrid,
		},
		{
			name:         "llm assisted when only llm available",
			hasLLM:       true,
			expectedMode: domain.SearchModeLLMAssisted,
		},
		{
			name:         "full when all available",
			hasVector:    true,
			hasEmbedding: true,
			hasLLM:       true,
			expectedMode: domain.SearchModeFull,
		},
		{
			name:         "semantic option forces hybrid",
			hasVector:    true,
			hasEmbedding: true,
			hasLLM:       true,
			opts:         domain.SearchOptions{Semantic: true},
			expectedMode: domain.SearchModeHybrid,
		},
		{
			name:         "hybrid option degraded when no vector",
			opts:         domain.SearchOptions{Hybrid: true},
			expectedMode: domain.SearchModeTextOnly,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var vectorIndex driven.VectorIndex
			var embedService driven.EmbeddingService
			var llmService driven.LLMService

			if tt.hasVector {
				vectorIndex = &mockVectorIndex{}
			}
			if tt.hasEmbedding {
				embedService = &mockEmbeddingService{}
			}
			if tt.hasLLM {
				llmService = &mockLLMService{}
			}

			service := NewSearchService(nil, nil, vectorIndex, embedService, llmService)
			mode := service.effectiveMode(tt.opts)

			assert.Equal(t, tt.expectedMode, mode)
		})
	}
}

func TestSearchService_reciprocalRankFusion(t *testing.T) {
	service := &SearchService{}

	list1 := []scoredChunk{
		{chunkID: "a", score: 1.0},
		{chunkID: "b", score: 0.9},
		{chunkID: "c", score: 0.8},
	}
	list2 := []scoredChunk{
		{chunkID: "b", score: 1.0},
		{chunkID: "d", score: 0.9},
		{chunkID: "a", score: 0.8},
	}

	merged := service.reciprocalRankFusion(list1, list2, 60)

	require.NotEmpty(t, merged)
	// "b" should be at top (appears in both lists with good ranks).
	assert.Equal(t, "b", merged[0].chunkID)
	// All unique IDs should be present.
	ids := make(map[string]bool)
	for _, c := range merged {
		ids[c.chunkID] = true
	}
	assert.True(t, ids["a"])
	assert.True(t, ids["b"])
	assert.True(t, ids["c"])
	assert.True(t, ids["d"])
}

func TestSearchService_splitSentences(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected int
	}{
		{
			name:     "single sentence",
			content:  "This is a sentence.",
			expected: 1,
		},
		{
			name:     "multiple sentences",
			content:  "First sentence. Second sentence! Third sentence?",
			expected: 3,
		},
		{
			name:     "with newlines",
			content:  "Line one\nLine two\nLine three",
			expected: 3,
		},
		{
			name:     "empty content",
			content:  "",
			expected: 0,
		},
		{
			name:     "trailing content",
			content:  "Sentence one. Trailing content without terminator",
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sentences := splitSentences(tt.content)
			assert.Len(t, sentences, tt.expected)
		})
	}
}

func TestSearchService_generateHighlights(t *testing.T) {
	service := &SearchService{}

	tests := []struct {
		name        string
		content     string
		query       string
		expectEmpty bool
	}{
		{
			name:        "matching query",
			content:     "Sercha is a search engine. It provides fast search.",
			query:       "sercha",
			expectEmpty: false,
		},
		{
			name:        "no match",
			content:     "This content has nothing relevant.",
			query:       "banana",
			expectEmpty: true,
		},
		{
			name:        "empty query",
			content:     "Some content here.",
			query:       "",
			expectEmpty: true,
		},
		{
			name:        "case insensitive",
			content:     "SERCHA is great. Sercha rocks!",
			query:       "sercha",
			expectEmpty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			highlights := service.generateHighlights(tt.content, tt.query)
			if tt.expectEmpty {
				assert.Empty(t, highlights)
			} else {
				assert.NotEmpty(t, highlights)
			}
		})
	}
}

func TestSearchService_filterBySourceIDs(t *testing.T) {
	service := &SearchService{}

	results := []domain.SearchResult{
		{Document: domain.Document{SourceID: "src-1"}},
		{Document: domain.Document{SourceID: "src-2"}},
		{Document: domain.Document{SourceID: "src-1"}},
		{Document: domain.Document{SourceID: "src-3"}},
	}

	filtered := service.filterBySourceIDs(results, []string{"src-1", "src-3"})

	assert.Len(t, filtered, 3)
	for _, r := range filtered {
		assert.True(t, r.Document.SourceID == "src-1" || r.Document.SourceID == "src-3")
	}
}

func TestSearchService_applyPagination(t *testing.T) {
	service := &SearchService{}

	results := make([]domain.SearchResult, 10)
	for i := range results {
		results[i] = domain.SearchResult{Score: float64(10 - i)}
	}

	tests := []struct {
		name     string
		offset   int
		limit    int
		expected int
	}{
		{"no pagination", 0, 20, 10},
		{"limit only", 0, 5, 5},
		{"offset only", 3, 20, 7},
		{"offset and limit", 2, 3, 3},
		{"offset beyond length", 15, 5, 0},
		{"partial end", 8, 5, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paginated := service.applyPagination(results, tt.offset, tt.limit)
			assert.Len(t, paginated, tt.expected)
		})
	}
}

func TestSearchService_Search_BothSearchesFail(t *testing.T) {
	docStore := setupTestDocStore(t)
	searchEngine := &mockSearchEngine{searchErr: errors.New("keyword failed")}
	vectorIndex := &mockVectorIndex{searchErr: errors.New("vector failed")}
	embedService := &mockEmbeddingService{embedding: make([]float32, 384)}
	service := NewSearchService(docStore, searchEngine, vectorIndex, embedService, nil)
	ctx := context.Background()

	_, err := service.Search(ctx, "test", domain.SearchOptions{Hybrid: true})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "keyword")
	assert.Contains(t, err.Error(), "vector")
}

func TestSearchService_Search_NilDocStore(t *testing.T) {
	searchEngine := &mockSearchEngine{hits: createTestHits()}
	service := NewSearchService(nil, searchEngine, nil, nil, nil)
	ctx := context.Background()

	_, err := service.Search(ctx, "test", domain.SearchOptions{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "document store unavailable")
}
