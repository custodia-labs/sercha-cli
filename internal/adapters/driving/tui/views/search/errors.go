package search

import "errors"

// Error definitions for the search view.
var (
	// ErrNoSearchService indicates that no search service was provided.
	ErrNoSearchService = errors.New("search service is required")
)
