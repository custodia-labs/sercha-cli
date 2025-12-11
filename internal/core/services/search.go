package services

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driving"
	"github.com/custodia-labs/sercha-cli/internal/logger"
)

// Ensure SearchService implements the interface.
var _ driving.SearchService = (*SearchService)(nil)

// scoredChunk holds intermediate search results before hydration.
type scoredChunk struct {
	chunkID string
	score   float64
	source  string // "keyword", "vector", or "merged"
}

// SearchService provides hybrid search functionality.
type SearchService struct {
	docStore         driven.DocumentStore
	searchIndex      driven.SearchEngine
	vectorIndex      driven.VectorIndex
	embeddingService driven.EmbeddingService
	llmService       driven.LLMService
	sourceStore      driven.SourceStore
	credentialsStore driven.CredentialsStore
}

// NewSearchService creates a new search service.
// The embeddingService and llmService parameters are optional (can be nil).
func NewSearchService(
	docStore driven.DocumentStore,
	searchIndex driven.SearchEngine,
	vectorIndex driven.VectorIndex,
	embeddingService driven.EmbeddingService,
	llmService driven.LLMService,
) *SearchService {
	return &SearchService{
		docStore:         docStore,
		searchIndex:      searchIndex,
		vectorIndex:      vectorIndex,
		embeddingService: embeddingService,
		llmService:       llmService,
	}
}

// SetSourceStore sets the source store for SourceName enrichment.
func (s *SearchService) SetSourceStore(store driven.SourceStore) {
	s.sourceStore = store
}

// SetCredentialsStore sets the credentials store for SourceName enrichment.
func (s *SearchService) SetCredentialsStore(store driven.CredentialsStore) {
	s.credentialsStore = store
}

// Search performs hybrid search across all indexed documents.
func (s *SearchService) Search(
	ctx context.Context, query string, opts domain.SearchOptions,
) ([]domain.SearchResult, error) {
	logger.Section("Search Execution")
	logger.Debug("Query: %q", query)

	// Return empty for empty query
	query = strings.TrimSpace(query)
	if query == "" {
		logger.Debug("Empty query, returning no results")
		return []domain.SearchResult{}, nil
	}

	// Determine limit (default to 20)
	limit := opts.Limit
	if limit <= 0 {
		limit = 20
	}
	logger.Debug("Limit: %d, Offset: %d", limit, opts.Offset)

	// Request more results internally to account for filtering
	internalLimit := limit * 2
	if len(opts.SourceIDs) > 0 {
		internalLimit = limit * 3
		logger.Debug("Source filter: %v", opts.SourceIDs)
	}
	logger.Debug("Internal limit: %d", internalLimit)

	// Determine effective search mode based on options and available services
	mode := s.effectiveMode(opts)
	logger.Info("Effective search mode: %s", mode.Description())

	// Log available services
	logger.Debug("Services available: keyword=%t, vector=%t, embedding=%t, llm=%t",
		s.searchIndex != nil,
		s.vectorIndex != nil,
		s.embeddingService != nil,
		s.llmService != nil)

	// Execute search based on mode
	var chunks []scoredChunk
	var err error

	switch mode {
	case domain.SearchModeTextOnly:
		logger.Debug("Executing keyword search")
		chunks, err = s.keywordSearch(ctx, query, internalLimit)

	case domain.SearchModeHybrid:
		logger.Debug("Executing hybrid search (keyword + vector)")
		chunks, err = s.hybridSearch(ctx, query, internalLimit)

	case domain.SearchModeLLMAssisted:
		logger.Debug("Executing LLM-assisted search")
		chunks, err = s.llmAssistedSearch(ctx, query, internalLimit)

	case domain.SearchModeFull:
		logger.Debug("Executing full search (LLM + hybrid)")
		chunks, err = s.fullSearch(ctx, query, internalLimit)

	default:
		logger.Debug("Fallback to keyword search")
		chunks, err = s.keywordSearch(ctx, query, internalLimit)
	}

	if err != nil {
		logger.Warn("Search failed: %v", err)
		return nil, fmt.Errorf("search: %w", err)
	}

	logger.Debug("Raw results: %d chunks", len(chunks))

	// Hydrate results with full document data
	results, err := s.hydrateResults(ctx, chunks, query)
	if err != nil {
		return nil, fmt.Errorf("hydrate results: %w", err)
	}

	logger.Debug("Hydrated results: %d documents", len(results))

	// Filter by source IDs if specified
	if len(opts.SourceIDs) > 0 {
		results = s.filterBySourceIDs(results, opts.SourceIDs)
		logger.Debug("After source filter: %d results", len(results))
	}

	// Apply pagination
	results = s.applyPagination(results, opts.Offset, limit)
	logger.Info("Final results: %d", len(results))

	return results, nil
}

// effectiveMode determines the search mode based on options and available services.
// It gracefully degrades if required services are unavailable.
func (s *SearchService) effectiveMode(opts domain.SearchOptions) domain.SearchMode {
	// Check what capabilities are available
	canDoVector := s.vectorIndex != nil && s.embeddingService != nil
	canDoLLM := s.llmService != nil

	// If options explicitly request semantic search
	if opts.Semantic && canDoVector {
		return domain.SearchModeHybrid
	}

	// If options explicitly request hybrid search
	if opts.Hybrid {
		if canDoVector {
			return domain.SearchModeHybrid
		}
		// Degrade to text-only if vector search unavailable
		return domain.SearchModeTextOnly
	}

	// Determine best available mode
	if canDoVector && canDoLLM {
		return domain.SearchModeFull
	}
	if canDoVector {
		return domain.SearchModeHybrid
	}
	if canDoLLM {
		return domain.SearchModeLLMAssisted
	}

	return domain.SearchModeTextOnly
}

// keywordSearch performs full-text search using Xapian.
func (s *SearchService) keywordSearch(ctx context.Context, query string, limit int) ([]scoredChunk, error) {
	if s.searchIndex == nil {
		logger.Warn("Keyword search unavailable: search engine is nil")
		return nil, errors.New("search engine unavailable")
	}

	logger.Debug("Keyword search: query=%q, limit=%d", query, limit)

	hits, err := s.searchIndex.Search(ctx, query, limit)
	if err != nil {
		logger.Warn("Keyword search error: %v", err)
		return nil, fmt.Errorf("keyword search: %w", err)
	}

	logger.Debug("Keyword search: %d hits", len(hits))

	results := make([]scoredChunk, len(hits))
	for i, hit := range hits {
		results[i] = scoredChunk{
			chunkID: hit.ChunkID,
			score:   hit.Score,
			source:  "keyword",
		}
	}

	return results, nil
}

// vectorSearch performs semantic similarity search using HNSW.
func (s *SearchService) vectorSearch(ctx context.Context, query string, limit int) ([]scoredChunk, error) {
	if s.vectorIndex == nil {
		logger.Warn("Vector search unavailable: vector index is nil")
		return nil, errors.New("vector index unavailable")
	}
	if s.embeddingService == nil {
		logger.Warn("Vector search unavailable: embedding service is nil")
		return nil, errors.New("embedding service unavailable")
	}

	logger.Debug("Vector search: query=%q, limit=%d", query, limit)

	// Generate query embedding
	logger.Debug("Generating query embedding...")
	embedding, err := s.embeddingService.Embed(ctx, query)
	if err != nil {
		logger.Warn("Query embedding failed: %v", err)
		return nil, fmt.Errorf("generate query embedding: %w", err)
	}
	logger.Debug("Query embedding: %d dimensions", len(embedding))

	// Search vector index
	hits, err := s.vectorIndex.Search(ctx, embedding, limit)
	if err != nil {
		logger.Warn("Vector index search failed: %v", err)
		return nil, fmt.Errorf("vector search: %w", err)
	}

	logger.Debug("Vector search: %d hits", len(hits))

	results := make([]scoredChunk, len(hits))
	for i, hit := range hits {
		results[i] = scoredChunk{
			chunkID: hit.ChunkID,
			score:   hit.Similarity, // Cosine similarity 0-1
			source:  "vector",
		}
	}

	return results, nil
}

// hybridSearch combines keyword and vector search using RRF.
func (s *SearchService) hybridSearch(ctx context.Context, query string, limit int) ([]scoredChunk, error) {
	logger.Debug("Hybrid search: running keyword and vector searches in parallel")

	// Run keyword and vector searches in parallel
	var keywordResults, vectorResults []scoredChunk
	var keywordErr, vectorErr error

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		keywordResults, keywordErr = s.keywordSearch(ctx, query, limit)
	}()

	go func() {
		defer wg.Done()
		vectorResults, vectorErr = s.vectorSearch(ctx, query, limit)
	}()

	wg.Wait()

	// Handle errors gracefully - degrade if one search fails
	if keywordErr != nil && vectorErr != nil {
		logger.Warn("Hybrid search: both keyword and vector searches failed")
		return nil, fmt.Errorf("hybrid search: keyword=%w, vector=%w", keywordErr, vectorErr)
	}

	if keywordErr != nil {
		logger.Warn("Hybrid search: keyword search failed, using vector results only")
		return vectorResults, nil
	}

	if vectorErr != nil {
		logger.Warn("Hybrid search: vector search failed, using keyword results only")
		return keywordResults, nil
	}

	// Merge using Reciprocal Rank Fusion
	logger.Debug("Hybrid search: merging %d keyword + %d vector results with RRF",
		len(keywordResults), len(vectorResults))
	merged := s.reciprocalRankFusion(keywordResults, vectorResults, 60)
	logger.Debug("Hybrid search: merged to %d results", len(merged))

	return merged, nil
}

// llmAssistedSearch uses LLM to expand the query before keyword search.
func (s *SearchService) llmAssistedSearch(ctx context.Context, query string, limit int) ([]scoredChunk, error) {
	// Expand query using LLM if available
	expandedQuery := query
	if s.llmService != nil {
		logger.Debug("LLM query rewrite: original=%q", query)
		expanded, err := s.llmService.RewriteQuery(ctx, query)
		if err == nil && expanded != "" {
			expandedQuery = expanded
			logger.Info("LLM query rewrite: expanded=%q", expanded)
		} else if err != nil {
			logger.Warn("LLM query rewrite failed: %v (using original query)", err)
		}
	} else {
		logger.Debug("LLM service not available, using original query")
	}

	// Perform keyword search with expanded query
	return s.keywordSearch(ctx, expandedQuery, limit)
}

// fullSearch combines LLM query expansion with hybrid search.
func (s *SearchService) fullSearch(ctx context.Context, query string, limit int) ([]scoredChunk, error) {
	// Expand query using LLM if available
	expandedQuery := query
	if s.llmService != nil {
		logger.Debug("Full search: LLM query rewrite for %q", query)
		expanded, err := s.llmService.RewriteQuery(ctx, query)
		if err == nil && expanded != "" {
			expandedQuery = expanded
			logger.Info("Full search: expanded query=%q", expanded)
		} else if err != nil {
			logger.Warn("Full search: LLM rewrite failed: %v", err)
		}
	}

	// Run hybrid search with the expanded query
	return s.hybridSearch(ctx, expandedQuery, limit)
}

// Merges two ranked lists using Reciprocal Rank Fusion (RRF).
// k is the constant (typically 60) to prevent high ranks from dominating.
//
//nolint:godot // Private method - no exported name to start with.
func (s *SearchService) reciprocalRankFusion(list1, list2 []scoredChunk, k int) []scoredChunk {
	scores := make(map[string]float64)
	seen := make(map[string]bool)

	// Calculate RRF scores for list1
	for rank, chunk := range list1 {
		rrf := 1.0 / float64(k+rank+1)
		scores[chunk.chunkID] += rrf
		seen[chunk.chunkID] = true
	}

	// Add RRF scores for list2
	for rank, chunk := range list2 {
		rrf := 1.0 / float64(k+rank+1)
		scores[chunk.chunkID] += rrf
		seen[chunk.chunkID] = true
	}

	// Convert to slice and sort by combined score
	results := make([]scoredChunk, 0, len(seen))
	for id := range seen {
		results = append(results, scoredChunk{
			chunkID: id,
			score:   scores[id],
			source:  "merged",
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	return results
}

// hydrateResults converts chunk IDs to full SearchResult objects.
func (s *SearchService) hydrateResults(
	ctx context.Context, chunks []scoredChunk, query string,
) ([]domain.SearchResult, error) {
	if s.docStore == nil {
		return nil, errors.New("document store unavailable")
	}

	results := make([]domain.SearchResult, 0, len(chunks))

	for _, sc := range chunks {
		// Get chunk from document store
		chunk, err := s.docStore.GetChunk(ctx, sc.chunkID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				// Chunk was deleted, skip it
				continue
			}
			return nil, fmt.Errorf("get chunk %s: %w", sc.chunkID, err)
		}

		// Get parent document
		doc, err := s.docStore.GetDocument(ctx, chunk.DocumentID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				// Document was deleted, skip it
				continue
			}
			return nil, fmt.Errorf("get document %s: %w", chunk.DocumentID, err)
		}

		// Generate highlights
		highlights := s.generateHighlights(chunk.Content, query)

		// Build SourceName from source and credentials
		sourceName := s.getSourceName(ctx, doc.SourceID)

		results = append(results, domain.SearchResult{
			Document:   *doc,
			Chunk:      *chunk,
			Score:      sc.score,
			Highlights: highlights,
			SourceName: sourceName,
		})
	}

	return results, nil
}

// generateHighlights creates text snippets with matched terms.
func (s *SearchService) generateHighlights(content, query string) []string {
	queryTerms := strings.Fields(strings.ToLower(query))
	if len(queryTerms) == 0 {
		return nil
	}

	// Split content into sentences (simple approach)
	sentences := splitSentences(content)
	contentLower := strings.ToLower(content)
	_ = contentLower // Used for future enhancement

	var highlights []string

	for _, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if sentence == "" {
			continue
		}

		sentenceLower := strings.ToLower(sentence)
		for _, term := range queryTerms {
			if strings.Contains(sentenceLower, term) {
				// Create highlight snippet
				highlight := sentence
				if len(highlight) > 200 {
					highlight = highlight[:200] + "..."
				}
				highlights = append(highlights, highlight)
				break
			}
		}

		if len(highlights) >= 3 {
			break // Limit to 3 highlights
		}
	}

	return highlights
}

// splitSentences splits content into sentences.
func splitSentences(content string) []string {
	// Simple sentence splitting by common terminators
	var sentences []string
	var current strings.Builder

	for _, r := range content {
		current.WriteRune(r)
		if r == '.' || r == '!' || r == '?' || r == '\n' {
			s := strings.TrimSpace(current.String())
			if s != "" {
				sentences = append(sentences, s)
			}
			current.Reset()
		}
	}

	// Don't forget the last sentence
	if s := strings.TrimSpace(current.String()); s != "" {
		sentences = append(sentences, s)
	}

	return sentences
}

// filterBySourceIDs filters results to only include specified sources.
func (s *SearchService) filterBySourceIDs(results []domain.SearchResult, sourceIDs []string) []domain.SearchResult {
	sourceSet := make(map[string]bool)
	for _, id := range sourceIDs {
		sourceSet[id] = true
	}

	filtered := make([]domain.SearchResult, 0)
	for i := range results {
		if sourceSet[results[i].Document.SourceID] {
			filtered = append(filtered, results[i])
		}
	}

	return filtered
}

// applyPagination applies offset and limit to results.
func (s *SearchService) applyPagination(results []domain.SearchResult, offset, limit int) []domain.SearchResult {
	if offset >= len(results) {
		return []domain.SearchResult{}
	}

	end := offset + limit
	if end > len(results) {
		end = len(results)
	}

	return results[offset:end]
}

// getSourceName builds a display name for a source by combining source name with account identifier.
// For example: "Gmail - user@gmail.com" or "GitHub - octocat".
// Falls back to just the source name if credentials are not available.
func (s *SearchService) getSourceName(ctx context.Context, sourceID string) string {
	if s.sourceStore == nil {
		return ""
	}

	source, err := s.sourceStore.Get(ctx, sourceID)
	if err != nil || source == nil {
		return ""
	}

	// Get account identifier from credentials if available
	var accountIdentifier string
	if s.credentialsStore != nil && source.CredentialsID != "" {
		creds, err := s.credentialsStore.Get(ctx, source.CredentialsID)
		if err == nil && creds != nil {
			accountIdentifier = creds.AccountIdentifier
		}
	}

	return source.DisplayName(accountIdentifier)
}
