package cli

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/custodia-labs/sercha-cli/internal/core/ports/driving"
)

var syncCmd = &cobra.Command{
	Use:   "sync [source-id]",
	Short: "Synchronise documents from sources",
	Long: `Triggers document synchronisation from configured sources.
If a source ID is provided, only that source is synchronised.
Otherwise, all sources are synchronised.`,
	RunE: runSync,
}

func init() {
	rootCmd.AddCommand(syncCmd)
}

func runSync(cmd *cobra.Command, args []string) error {
	if syncOrchestrator == nil {
		return errors.New("sync service not configured")
	}

	ctx := context.Background()

	if len(args) > 0 {
		// Sync specific source
		sourceID := args[0]
		cmd.Printf("Synchronising source: %s...\n", sourceID)

		if err := syncWithProgress(ctx, cmd, syncOrchestrator, sourceID); err != nil {
			return fmt.Errorf("sync failed: %w", err)
		}

		cmd.Printf("Source %s synchronised successfully.\n", sourceID)
	} else {
		// Sync all sources
		cmd.Println("Synchronising all sources...")

		if err := syncOrchestrator.SyncAll(ctx); err != nil {
			return fmt.Errorf("sync failed: %w", err)
		}

		cmd.Println("All sources synchronised successfully.")
	}

	return nil
}

// syncWithProgress runs sync while displaying progress updates.
func syncWithProgress(
	ctx context.Context,
	cmd *cobra.Command,
	syncOrch driving.SyncOrchestrator,
	sourceID string,
) error {
	// Start sync in goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- syncOrch.Sync(ctx, sourceID)
	}()

	// Poll status every 500ms
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	lastCount := 0
	for {
		select {
		case err := <-errCh:
			// Print final status (ignore status error - best effort)
			status, statusErr := syncOrch.Status(ctx, sourceID)
			if statusErr == nil && status != nil && status.DocumentsProcessed > 0 {
				cmd.Printf("\rProcessed %d documents (%d errors)\n",
					status.DocumentsProcessed, status.ErrorCount)
			}
			return err
		case <-ticker.C:
			// Check progress (ignore status error - best effort)
			status, statusErr := syncOrch.Status(ctx, sourceID)
			if statusErr == nil && status != nil && status.DocumentsProcessed > lastCount {
				cmd.Printf("\rProcessing... %d documents", status.DocumentsProcessed)
				lastCount = status.DocumentsProcessed
			}
		}
	}
}
