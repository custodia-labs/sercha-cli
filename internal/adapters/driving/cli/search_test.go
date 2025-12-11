package cli

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

func TestSearchCmd_Use(t *testing.T) {
	assert.Equal(t, "search [query]", searchCmd.Use)
}

func TestSearchCmd_Short(t *testing.T) {
	assert.Equal(t, "Search indexed documents", searchCmd.Short)
}

func TestSearchCmd_Long(t *testing.T) {
	assert.Contains(t, searchCmd.Long, "hybrid search")
	assert.Contains(t, searchCmd.Long, "BM25")
	assert.Contains(t, searchCmd.Long, "semantic")
}

func TestSearchCmd_RequiresExactlyOneArg(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"search"})
	defer func() {
		rootCmd.SetArgs(nil)
	}()

	err := rootCmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 1 arg(s)")
}

func TestSearchCmd_HasLimitFlag(t *testing.T) {
	flag := searchCmd.Flags().Lookup("limit")
	require.NotNil(t, flag, "limit flag should exist")
	assert.Equal(t, "n", flag.Shorthand)
	assert.Equal(t, "10", flag.DefValue)
}

func TestSearchCmd_ExecutesWithQuery(t *testing.T) {
	cleanup := setupTestServices()
	defer cleanup()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"search", "test query"})
	defer func() {
		rootCmd.SetArgs(nil)
	}()

	err := rootCmd.Execute()

	assert.NoError(t, err)
	// New behaviour: returns mock results
	assert.Contains(t, buf.String(), "Results:")
}

func TestSearchCmd_ExecutesWithLimitFlag(t *testing.T) {
	cleanup := setupTestServices()
	defer cleanup()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"search", "--limit", "25", "test query"})
	defer func() {
		rootCmd.SetArgs(nil)
	}()

	err := rootCmd.Execute()

	assert.NoError(t, err)
	// New behaviour: returns mock results
	assert.Contains(t, buf.String(), "Results:")
}

func TestSearchCmd_ExecutesWithShortLimitFlag(t *testing.T) {
	cleanup := setupTestServices()
	defer cleanup()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"search", "-n", "5", "another query"})
	defer func() {
		rootCmd.SetArgs(nil)
	}()

	err := rootCmd.Execute()

	assert.NoError(t, err)
	// New behaviour: returns mock results
	assert.Contains(t, buf.String(), "Results:")
}

func TestSearchCmd_JSONOutput(t *testing.T) {
	cleanup := setupTestServices()
	defer cleanup()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"search", "--json", "test query"})
	defer func() {
		rootCmd.SetArgs(nil)
		searchJSON = false // Reset flag
	}()

	err := rootCmd.Execute()

	assert.NoError(t, err)
	// JSON uses capitalized field names from struct tags
	assert.Contains(t, buf.String(), "\"ID\"")
	assert.Contains(t, buf.String(), "\"Title\"")
	assert.Contains(t, buf.String(), "\"Score\"")
}

func TestSearchCmd_ServiceNotConfigured(t *testing.T) {
	oldService := searchService
	searchService = nil
	defer func() {
		searchService = oldService
	}()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"search", "test"})
	defer func() {
		rootCmd.SetArgs(nil)
	}()

	err := rootCmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "search service not configured")
}

func TestOutputSearchJSON_EmptyResults(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)

	err := outputSearchJSON(rootCmd, []domain.SearchResult{})

	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "[]")
}

func TestOutputSearchTable_EmptyResults(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)

	err := outputSearchTable(rootCmd, []domain.SearchResult{})

	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "No results found")
}

func TestOutputSearchTable_WithHighlights(t *testing.T) {
	cleanup := setupTestServices()
	defer cleanup()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)

	results := []domain.SearchResult{
		{
			Document:   domain.Document{ID: "doc-1", Title: "Test Document"},
			Score:      0.95,
			Highlights: []string{"This is a highlight snippet"},
		},
	}

	err := outputSearchTable(rootCmd, results)

	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "Test Document")
	assert.Contains(t, buf.String(), "0.95")
	assert.Contains(t, buf.String(), "This is a highlight snippet")
}

func TestOutputSearchTable_WithoutTitle(t *testing.T) {
	cleanup := setupTestServices()
	defer cleanup()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)

	results := []domain.SearchResult{
		{
			Document: domain.Document{ID: "doc-123"},
			Score:    0.75,
		},
	}

	err := outputSearchTable(rootCmd, results)

	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "doc-123")
	assert.Contains(t, buf.String(), "0.75")
}

func TestSearchCmd_ServiceError(t *testing.T) {
	oldService := searchService
	searchService = &mockSearchServiceError{}
	defer func() {
		searchService = oldService
	}()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"search", "test"})
	defer func() {
		rootCmd.SetArgs(nil)
	}()

	err := rootCmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "search failed")
}
