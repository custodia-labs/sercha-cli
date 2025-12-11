package mcp

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

func TestServer_handleSearch(t *testing.T) {
	ctx := context.Background()

	t.Run("returns search results", func(t *testing.T) {
		mockSearch := &mockSearchService{
			results: []domain.SearchResult{
				{
					Document: domain.Document{
						ID:    "doc-1",
						Title: "Test Doc",
						URI:   "/path/to/doc",
					},
					Chunk: domain.Chunk{
						Content: "This is the content",
					},
					Score:      0.95,
					Highlights: []string{"matched text"},
				},
			},
		}

		ports := &Ports{Search: mockSearch}
		server, err := NewServer(ports)
		require.NoError(t, err)

		input := SearchInput{Query: "test", Limit: 10}
		_, output, err := server.handleSearch(ctx, nil, input)

		require.NoError(t, err)
		assert.Equal(t, 1, output.Count)
		assert.Len(t, output.Results, 1)
		assert.Equal(t, "doc-1", output.Results[0].DocumentID)
		assert.Equal(t, "Test Doc", output.Results[0].Title)
		assert.Equal(t, "/path/to/doc", output.Results[0].URI)
		assert.Equal(t, 0.95, output.Results[0].Score)
		assert.Equal(t, "This is the content", output.Results[0].Content)
	})

	t.Run("default limit is 10", func(t *testing.T) {
		mockSearch := &mockSearchService{}
		ports := &Ports{Search: mockSearch}
		server, err := NewServer(ports)
		require.NoError(t, err)

		input := SearchInput{Query: "test", Limit: 0}
		_, output, err := server.handleSearch(ctx, nil, input)

		require.NoError(t, err)
		assert.Equal(t, 0, output.Count)
	})

	t.Run("returns error on search failure", func(t *testing.T) {
		mockSearch := &mockSearchService{
			err: errors.New("search failed"),
		}

		ports := &Ports{Search: mockSearch}
		server, err := NewServer(ports)
		require.NoError(t, err)

		input := SearchInput{Query: "test"}
		_, _, err = server.handleSearch(ctx, nil, input)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "search failed")
	})
}
