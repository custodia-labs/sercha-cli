package mcp

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

// SearchInput is the input schema for the search tool.
type SearchInput struct {
	Query string `json:"query" jsonschema:"the search query to find documents"`
	Limit int    `json:"limit,omitempty" jsonschema:"maximum number of results to return (default 10)"`
}

// SearchOutput is the output schema for the search tool.
type SearchOutput struct {
	Results []SearchResultOutput `json:"results"`
	Count   int                  `json:"count"`
}

// SearchResultOutput represents a single search result.
type SearchResultOutput struct {
	DocumentID string   `json:"document_id"`
	Title      string   `json:"title"`
	URI        string   `json:"uri"`
	Score      float64  `json:"score"`
	Highlights []string `json:"highlights,omitempty"`
	Content    string   `json:"content,omitempty"`
}

// registerTools registers all tool handlers with the MCP server.
func (s *Server) registerTools() {
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "search",
		Description: "Search across all indexed documents",
	}, s.handleSearch)
}

// handleSearch handles the search tool invocation.
func (s *Server) handleSearch(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input SearchInput,
) (*mcp.CallToolResult, SearchOutput, error) {
	limit := input.Limit
	if limit <= 0 {
		limit = 10
	}

	opts := domain.SearchOptions{Limit: limit}
	results, err := s.ports.Search.Search(ctx, input.Query, opts)
	if err != nil {
		return nil, SearchOutput{}, err
	}

	output := SearchOutput{
		Results: make([]SearchResultOutput, len(results)),
		Count:   len(results),
	}

	for i := range results {
		output.Results[i] = SearchResultOutput{
			DocumentID: results[i].Document.ID,
			Title:      results[i].Document.Title,
			URI:        results[i].Document.URI,
			Score:      results[i].Score,
			Highlights: results[i].Highlights,
			Content:    results[i].Chunk.Content,
		}
	}

	return nil, output, nil
}
