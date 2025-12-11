package domain

// SearchOptions configures a search query.
type SearchOptions struct {
	// Limit is the maximum number of results.
	Limit int

	// Offset is the number of results to skip.
	Offset int

	// SourceIDs filters to specific sources.
	SourceIDs []string

	// Semantic enables vector similarity search.
	Semantic bool

	// Hybrid enables combined keyword + semantic search.
	Hybrid bool
}

// SearchResult represents a single search hit.
type SearchResult struct {
	// Document is the matched document.
	Document Document

	// Chunk is the specific chunk that matched.
	Chunk Chunk

	// Score is the relevance score.
	Score float64

	// Highlights contains snippets with matched terms.
	Highlights []string

	// SourceName is the display name of the source (includes account identifier).
	// Example: "Gmail - user@gmail.com" or "GitHub - octocat"
	SourceName string
}
