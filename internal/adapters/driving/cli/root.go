package cli

import (
	"github.com/spf13/cobra"

	"github.com/custodia-labs/sercha-cli/internal/core/ports/driving"
	"github.com/custodia-labs/sercha-cli/internal/logger"
)

var (
	// Version is set by goreleaser ldflags.
	version = "dev"

	// Verbose enables debug logging.
	verbose bool

	// Services holds injected service implementations for CLI commands.
	searchService       driving.SearchService
	sourceService       driving.SourceService
	syncOrchestrator    driving.SyncOrchestrator
	documentService     driving.DocumentService
	connectorRegistry   driving.ConnectorRegistry
	providerRegistry    driving.ProviderRegistry
	settingsService     driving.SettingsService
	authProviderService driving.AuthProviderService
	credentialsService  driving.CredentialsService
)

// Services holds configuration for CLI commands.
type Services struct {
	Search            driving.SearchService
	Source            driving.SourceService
	Sync              driving.SyncOrchestrator
	Document          driving.DocumentService
	ConnectorRegistry driving.ConnectorRegistry
	ProviderRegistry  driving.ProviderRegistry
	Settings          driving.SettingsService
	AuthProvider      driving.AuthProviderService
	Credentials       driving.CredentialsService
}

// SetServices injects service implementations for CLI commands.
func SetServices(s *Services) {
	if s == nil {
		return
	}
	searchService = s.Search
	sourceService = s.Source
	syncOrchestrator = s.Sync
	documentService = s.Document
	connectorRegistry = s.ConnectorRegistry
	providerRegistry = s.ProviderRegistry
	settingsService = s.Settings
	authProviderService = s.AuthProvider
	credentialsService = s.Credentials
}

// rootCmd is the base command.
var rootCmd = &cobra.Command{
	Use:   "sercha",
	Short: "Local-first semantic search for your documents",
	Long: `Sercha is a local-first semantic search engine that indexes your documents
and provides hybrid keyword + vector search capabilities.

All data stays on your machine. No cloud required.`,
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

// SetVersion sets the version string for the CLI.
func SetVersion(v string) {
	version = v
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose debug output")

	// Use PersistentPreRunE to set verbose mode before any command executes
	rootCmd.PersistentPreRunE = func(_ *cobra.Command, _ []string) error {
		logger.SetVerbose(verbose)
		return nil
	}
}
