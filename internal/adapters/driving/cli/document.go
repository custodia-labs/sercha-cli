package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

var documentCmd = &cobra.Command{
	Use:   "document",
	Short: "Manage indexed documents",
	Long:  `List, view, exclude, or refresh indexed documents.`,
}

var documentListCmd = &cobra.Command{
	Use:   "list [source-id]",
	Short: "List documents for a source",
	Args:  cobra.ExactArgs(1),
	RunE:  runDocumentList,
}

var documentGetCmd = &cobra.Command{
	Use:   "get [doc-id]",
	Short: "Show document info",
	Args:  cobra.ExactArgs(1),
	RunE:  runDocumentGet,
}

var documentContentCmd = &cobra.Command{
	Use:   "content [doc-id]",
	Short: "Print document content",
	Args:  cobra.ExactArgs(1),
	RunE:  runDocumentContent,
}

var documentDetailsCmd = &cobra.Command{
	Use:   "details [doc-id]",
	Short: "Show document metadata",
	Args:  cobra.ExactArgs(1),
	RunE:  runDocumentDetails,
}

var documentExcludeCmd = &cobra.Command{
	Use:   "exclude [doc-id]",
	Short: "Exclude document from index",
	Long:  `Removes a document from the index and marks it to be skipped during future syncs.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runDocumentExclude,
}

var documentRefreshCmd = &cobra.Command{
	Use:   "refresh [doc-id]",
	Short: "Resync a single document",
	Args:  cobra.ExactArgs(1),
	RunE:  runDocumentRefresh,
}

var documentOpenCmd = &cobra.Command{
	Use:   "open [doc-id]",
	Short: "Open document in default application",
	Args:  cobra.ExactArgs(1),
	RunE:  runDocumentOpen,
}

// excludeReason is a flag for the exclude command.
var excludeReason string

func init() {
	documentExcludeCmd.Flags().StringVarP(&excludeReason, "reason", "r", "", "Reason for excluding the document")

	documentCmd.AddCommand(documentListCmd)
	documentCmd.AddCommand(documentGetCmd)
	documentCmd.AddCommand(documentContentCmd)
	documentCmd.AddCommand(documentDetailsCmd)
	documentCmd.AddCommand(documentExcludeCmd)
	documentCmd.AddCommand(documentRefreshCmd)
	documentCmd.AddCommand(documentOpenCmd)
	rootCmd.AddCommand(documentCmd)
}

func runDocumentList(cmd *cobra.Command, args []string) error {
	if documentService == nil {
		return errors.New("document service not configured")
	}

	sourceID := args[0]
	ctx := context.Background()

	docs, err := documentService.ListBySource(ctx, sourceID)
	if err != nil {
		return fmt.Errorf("failed to list documents: %w", err)
	}

	if len(docs) == 0 {
		cmd.Printf("No documents found for source: %s\n", sourceID)
		return nil
	}

	cmd.Printf("Documents for source %s:\n\n", sourceID)
	for i := range docs {
		cmd.Printf("  %s\n", docs[i].ID)
		cmd.Printf("    Title: %s\n", docs[i].Title)
		if docs[i].URI != "" {
			cmd.Printf("    URI: %s\n", docs[i].URI)
		}
		cmd.Println()
	}

	cmd.Printf("Total: %d documents\n", len(docs))
	return nil
}

func runDocumentGet(cmd *cobra.Command, args []string) error {
	if documentService == nil {
		return errors.New("document service not configured")
	}

	docID := args[0]
	ctx := context.Background()

	doc, err := documentService.Get(ctx, docID)
	if err != nil {
		return fmt.Errorf("failed to get document: %w", err)
	}

	cmd.Printf("Document: %s\n\n", doc.ID)
	cmd.Printf("  Title:    %s\n", doc.Title)
	cmd.Printf("  Source:   %s\n", doc.SourceID)
	cmd.Printf("  URI:      %s\n", doc.URI)
	cmd.Printf("  Created:  %s\n", doc.CreatedAt.Format("2006-01-02 15:04:05"))
	cmd.Printf("  Updated:  %s\n", doc.UpdatedAt.Format("2006-01-02 15:04:05"))

	if len(doc.Metadata) > 0 {
		cmd.Println("\n  Metadata:")
		for k, v := range doc.Metadata {
			cmd.Printf("    %s: %v\n", k, v)
		}
	}

	return nil
}

func runDocumentContent(cmd *cobra.Command, args []string) error {
	if documentService == nil {
		return errors.New("document service not configured")
	}

	docID := args[0]
	ctx := context.Background()

	content, err := documentService.GetContent(ctx, docID)
	if err != nil {
		return fmt.Errorf("failed to get document content: %w", err)
	}

	cmd.Println(content)
	return nil
}

func runDocumentDetails(cmd *cobra.Command, args []string) error {
	if documentService == nil {
		return errors.New("document service not configured")
	}

	docID := args[0]
	ctx := context.Background()

	details, err := documentService.GetDetails(ctx, docID)
	if err != nil {
		return fmt.Errorf("failed to get document details: %w", err)
	}

	cmd.Printf("Document Details: %s\n\n", details.ID)
	cmd.Printf("  Title:       %s\n", details.Title)
	cmd.Printf("  Source:      %s (%s)\n", details.SourceName, details.SourceType)
	cmd.Printf("  Source ID:   %s\n", details.SourceID)
	cmd.Printf("  URI:         %s\n", details.URI)
	cmd.Printf("  Chunks:      %d\n", details.ChunkCount)
	cmd.Printf("  Created:     %s\n", details.CreatedAt.Format("2006-01-02 15:04:05"))
	cmd.Printf("  Updated:     %s\n", details.UpdatedAt.Format("2006-01-02 15:04:05"))

	if len(details.Metadata) > 0 {
		cmd.Println("\n  Metadata:")
		for k, v := range details.Metadata {
			cmd.Printf("    %s: %s\n", k, v)
		}
	}

	return nil
}

func runDocumentExclude(cmd *cobra.Command, args []string) error {
	if documentService == nil {
		return errors.New("document service not configured")
	}

	docID := args[0]
	ctx := context.Background()

	reason := excludeReason
	if reason == "" {
		reason = "excluded via CLI"
	}

	if err := documentService.Exclude(ctx, docID, reason); err != nil {
		return fmt.Errorf("failed to exclude document: %w", err)
	}

	cmd.Printf("Document %s excluded from index.\n", docID)
	return nil
}

func runDocumentRefresh(cmd *cobra.Command, args []string) error {
	if documentService == nil {
		return errors.New("document service not configured")
	}

	docID := args[0]
	ctx := context.Background()

	cmd.Printf("Refreshing document %s...\n", docID)

	if err := documentService.Refresh(ctx, docID); err != nil {
		return fmt.Errorf("failed to refresh document: %w", err)
	}

	cmd.Printf("Document %s refreshed successfully.\n", docID)
	return nil
}

func runDocumentOpen(cmd *cobra.Command, args []string) error {
	if documentService == nil {
		return errors.New("document service not configured")
	}

	docID := args[0]
	ctx := context.Background()

	if err := documentService.Open(ctx, docID); err != nil {
		return fmt.Errorf("failed to open document: %w", err)
	}

	cmd.Printf("Opened document %s in default application.\n", docID)
	return nil
}
