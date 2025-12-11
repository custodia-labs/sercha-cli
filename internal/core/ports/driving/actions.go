package driving

import (
	"context"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

// ResultActionService provides actions on search results for external actors.
// This is used by TUI, CLI, and MCP adapters.
type ResultActionService interface {
	// CopyToClipboard copies the result's content to the system clipboard.
	CopyToClipboard(ctx context.Context, result *domain.SearchResult) error

	// OpenDocument opens the result's document in the default application.
	OpenDocument(ctx context.Context, result *domain.SearchResult) error
}
