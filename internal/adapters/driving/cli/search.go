package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

var (
	searchLimit int
	searchJSON  bool
)

var searchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search indexed documents",
	Long: `Performs hybrid search across all indexed documents.
Combines keyword (BM25) and semantic (vector) search for best results.`,
	Args: cobra.ExactArgs(1),
	RunE: runSearch,
}

func init() {
	searchCmd.Flags().IntVarP(&searchLimit, "limit", "n", 10, "maximum number of results")
	searchCmd.Flags().BoolVar(&searchJSON, "json", false, "output results as JSON")
	rootCmd.AddCommand(searchCmd)
}

func runSearch(cmd *cobra.Command, args []string) error {
	query := args[0]

	if searchService == nil {
		return errors.New("search service not configured")
	}

	ctx := context.Background()
	opts := domain.SearchOptions{
		Limit: searchLimit,
	}

	results, err := searchService.Search(ctx, query, opts)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	if searchJSON {
		return outputSearchJSON(cmd, results)
	}

	return outputSearchTable(cmd, results)
}

func outputSearchJSON(cmd *cobra.Command, results []domain.SearchResult) error {
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal results: %w", err)
	}
	cmd.Println(string(data))
	return nil
}

func outputSearchTable(cmd *cobra.Command, results []domain.SearchResult) error {
	if len(results) == 0 {
		cmd.Println("No results found.")
		return nil
	}

	cmd.Println("Results:")
	cmd.Println()
	for i := range results {
		// Format: [N] Title - Snippet (Score)
		title := results[i].Document.Title
		if title == "" {
			title = results[i].Document.ID
		}

		snippet := ""
		if len(results[i].Highlights) > 0 {
			snippet = results[i].Highlights[0]
		}

		cmd.Printf("  [%d] %s (%.2f)\n", i+1, title, results[i].Score)
		if results[i].SourceName != "" {
			cmd.Printf("      Source: %s\n", results[i].SourceName)
		}
		if snippet != "" {
			cmd.Printf("      %s\n", snippet)
		}
		cmd.Println()
	}

	// TODO: Add interactive result selection when needed
	// cmd.Print("Select result [1-N, q to quit]: ")

	return nil
}
