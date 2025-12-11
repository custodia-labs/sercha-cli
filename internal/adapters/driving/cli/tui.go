package cli

import (
	"context"
	"fmt"
	"os"
	"runtime/debug"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/custodia-labs/sercha-cli/internal/adapters/driving/tui"
	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driving"
)

// TUIConfig holds configuration for the TUI command.
type TUIConfig struct {
	SearchService       driving.SearchService
	SourceService       driving.SourceService
	SyncOrchestrator    driving.SyncOrchestrator
	ResultActionService driving.ResultActionService
	DocumentService     driving.DocumentService
	ConnectorRegistry   driving.ConnectorRegistry
	ProviderRegistry    driving.ProviderRegistry
	SettingsService     driving.SettingsService
	CredentialsService  driving.CredentialsService
	AuthProviderService driving.AuthProviderService
	Scheduler           driving.Scheduler
	SchedulerConfig     domain.SchedulerConfig
}

// tuiConfig holds the current TUI configuration.
var tuiConfig *TUIConfig

// tuiCmd represents the tui command.
var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch the interactive terminal UI",
	Long: `Launch the interactive terminal user interface for Sercha.

The TUI provides a visual interface for searching your indexed documents,
managing sources, and viewing search results with keyboard navigation.

Controls:
  ↑/k, ↓/j - Navigate results
  Enter    - Search / Select
  Esc      - Back / Cancel
  ?        - Toggle help
  q        - Quit`,
	RunE: runTUI,
}

// SetTUIConfig sets the configuration for the TUI command.
func SetTUIConfig(config *TUIConfig) {
	tuiConfig = config
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}

func runTUI(cmd *cobra.Command, args []string) error {
	// Add panic recovery to get stack traces
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "Panic in TUI: %v\n", r)
			fmt.Fprintf(os.Stderr, "Stack trace:\n%s\n", debug.Stack())
		}
	}()

	// Start scheduler if enabled (TUI is long-running, needs background tasks)
	if tuiConfig != nil && tuiConfig.SchedulerConfig.Enabled && tuiConfig.Scheduler != nil {
		schedulerCtx, schedulerCancel := context.WithCancel(context.Background())
		defer schedulerCancel()

		go func() {
			if err := tuiConfig.Scheduler.Start(schedulerCtx); err != nil {
				// Log but don't fail - scheduler errors shouldn't block TUI
				fmt.Fprintf(os.Stderr, "scheduler stopped: %v\n", err)
			}
		}()

		defer func() {
			if err := tuiConfig.Scheduler.Stop(); err != nil {
				fmt.Fprintf(os.Stderr, "scheduler stop error: %v\n", err)
			}
		}()
	}

	// Build ports from configuration
	ports := &tui.Ports{}

	if tuiConfig != nil {
		ports.Search = tuiConfig.SearchService
		ports.Source = tuiConfig.SourceService
		ports.Sync = tuiConfig.SyncOrchestrator
		ports.ResultAction = tuiConfig.ResultActionService
		ports.Document = tuiConfig.DocumentService
		ports.ConnectorRegistry = tuiConfig.ConnectorRegistry
		ports.ProviderRegistry = tuiConfig.ProviderRegistry
		ports.Settings = tuiConfig.SettingsService
		ports.Credentials = tuiConfig.CredentialsService
		ports.AuthProvider = tuiConfig.AuthProviderService
	}

	// Create the TUI app
	app, err := tui.NewApp(ports)
	if err != nil {
		return fmt.Errorf("failed to create TUI: %w", err)
	}

	// Set up context from command
	app.WithContext(cmd.Context())

	// Create and run the bubbletea program
	p := tea.NewProgram(app, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	return nil
}
