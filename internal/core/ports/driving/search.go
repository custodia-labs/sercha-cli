package driving

import (
	"context"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

// SearchService provides search capabilities to external actors.
type SearchService interface {
	// Search performs hybrid search across all indexed documents.
	Search(ctx context.Context, query string, opts domain.SearchOptions) ([]domain.SearchResult, error)
}
