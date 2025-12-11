package cli

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

var settingsCmd = &cobra.Command{
	Use:   "settings",
	Short: "Manage application settings",
	Long: `View and configure search settings, AI providers, and other options.

Use subcommands to configure specific settings or run the interactive wizard.`,
	RunE: runSettingsShow,
}

var settingsShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current settings",
	RunE:  runSettingsShow,
}

var settingsWizardCmd = &cobra.Command{
	Use:   "wizard",
	Short: "Interactive setup wizard",
	Long:  `Run an interactive wizard to configure all settings step by step.`,
	RunE:  runSettingsWizard,
}

var settingsModeCmd = &cobra.Command{
	Use:   "mode",
	Short: "Set search mode",
	Long: `Set the search mode to control how searches are performed.

Available modes:
  text_only    - Keyword search only (fastest, no setup required)
  hybrid       - Text + semantic vector search (requires embedding provider)
  llm_assisted - Text + LLM query expansion (requires LLM provider)
  full         - Text + semantic + LLM (requires both providers)`,
	RunE: runSettingsMode,
}

var settingsEmbeddingCmd = &cobra.Command{
	Use:   "embedding",
	Short: "Configure embedding provider",
	Long:  `Configure the embedding provider for semantic search.`,
	RunE:  runSettingsEmbedding,
}

var settingsLLMCmd = &cobra.Command{
	Use:   "llm",
	Short: "Configure LLM provider",
	Long:  `Configure the LLM provider for query expansion and conversational search.`,
	RunE:  runSettingsLLM,
}

func init() {
	settingsCmd.AddCommand(settingsShowCmd)
	settingsCmd.AddCommand(settingsWizardCmd)
	settingsCmd.AddCommand(settingsModeCmd)
	settingsCmd.AddCommand(settingsEmbeddingCmd)
	settingsCmd.AddCommand(settingsLLMCmd)
	rootCmd.AddCommand(settingsCmd)
}

func runSettingsShow(cmd *cobra.Command, _ []string) error {
	if settingsService == nil {
		return errors.New("settings service not configured")
	}

	settings, err := settingsService.Get()
	if err != nil {
		return fmt.Errorf("failed to get settings: %w", err)
	}

	cmd.Println("Current Settings")
	cmd.Println("================")
	cmd.Println()

	// Search settings
	cmd.Println("[Search]")
	cmd.Printf("  Mode: %s\n", settings.Search.Mode.Description())
	cmd.Println()

	// Embedding settings
	cmd.Println("[Embedding]")
	cmd.Printf("  Provider: %s\n", settings.Embedding.Provider.Description())
	cmd.Printf("  Model: %s\n", settings.Embedding.Model)
	if settings.Embedding.Provider.IsLocal() {
		cmd.Printf("  Base URL: %s\n", settings.Embedding.BaseURL)
	}
	if settings.Embedding.Provider.RequiresAPIKey() {
		if settings.Embedding.APIKey != "" {
			cmd.Printf("  API Key: %s\n", maskAPIKey(settings.Embedding.APIKey))
		} else {
			cmd.Printf("  API Key: (not set)\n")
		}
	}
	status := "configured"
	if !settings.Embedding.IsConfigured() {
		status = "not configured"
	}
	cmd.Printf("  Status: %s\n", status)
	cmd.Println()

	// LLM settings
	cmd.Println("[LLM]")
	cmd.Printf("  Provider: %s\n", settings.LLM.Provider.Description())
	cmd.Printf("  Model: %s\n", settings.LLM.Model)
	if settings.LLM.Provider.IsLocal() {
		cmd.Printf("  Base URL: %s\n", settings.LLM.BaseURL)
	}
	if settings.LLM.Provider.RequiresAPIKey() {
		if settings.LLM.APIKey != "" {
			cmd.Printf("  API Key: %s\n", maskAPIKey(settings.LLM.APIKey))
		} else {
			cmd.Printf("  API Key: (not set)\n")
		}
	}
	status = "configured"
	if !settings.LLM.IsConfigured() {
		status = "not configured"
	}
	cmd.Printf("  Status: %s\n", status)
	cmd.Println()

	// Vector index settings
	cmd.Println("[Vector Index]")
	if settings.VectorIndex.Enabled {
		cmd.Printf("  Enabled: yes\n")
		cmd.Printf("  Dimensions: %d\n", settings.VectorIndex.Dimensions)
	} else {
		cmd.Printf("  Enabled: no\n")
	}
	cmd.Println()

	// Validation
	if err := settingsService.Validate(); err != nil {
		cmd.Printf("Warning: %v\n", err)
		cmd.Println("Run 'sercha settings wizard' to fix configuration issues.")
	} else {
		cmd.Println("Configuration is valid.")
	}

	return nil
}

func runSettingsWizard(cmd *cobra.Command, _ []string) error {
	if settingsService == nil {
		return errors.New("settings service not configured")
	}

	cmd.Println("Sercha Settings Wizard")
	cmd.Println("======================")
	cmd.Println()

	reader := bufio.NewReader(os.Stdin)

	// Step 1: Search Mode
	cmd.Println("Step 1: Select Search Mode")
	cmd.Println("--------------------------")
	modes := domain.AllSearchModes()
	for i, mode := range modes {
		cmd.Printf("  %d. %s\n", i+1, mode.Description())
	}
	cmd.Print("\nEnter choice [1]: ")
	input := readLine(reader)
	modeIdx := parseChoice(input, len(modes), 1)
	selectedMode := modes[modeIdx-1]

	if err := settingsService.SetSearchMode(selectedMode); err != nil {
		return fmt.Errorf("failed to set search mode: %w", err)
	}
	cmd.Printf("Set search mode to: %s\n\n", selectedMode.Description())

	// Step 2: Configure Embedding Provider (if needed)
	if settingsService.RequiresEmbedding() {
		cmd.Println("Step 2: Configure Embedding Provider")
		cmd.Println("------------------------------------")
		cmd.Println("Your search mode requires semantic search. Please configure an embedding provider.")
		cmd.Println()

		if err := configureEmbeddingProvider(cmd, reader); err != nil {
			return err
		}
	} else {
		cmd.Println("Step 2: Embedding Provider (skipped)")
		cmd.Println("------------------------------------")
		cmd.Println("Not required for text-only search mode.")
	}

	// Step 3: Configure LLM Provider (if needed)
	if settingsService.RequiresLLM() {
		cmd.Println("Step 3: Configure LLM Provider")
		cmd.Println("------------------------------")
		cmd.Println("Your search mode requires an LLM. Please configure an LLM provider.")
		cmd.Println()

		if err := configureLLMProvider(cmd, reader); err != nil {
			return err
		}
	} else {
		cmd.Println("Step 3: LLM Provider (skipped)")
		cmd.Println("------------------------------")
		cmd.Println("Not required for current search mode.")
	}

	// Final validation
	cmd.Println("Configuration Complete!")
	cmd.Println("=======================")
	if err := settingsService.Validate(); err != nil {
		cmd.Printf("Warning: %v\n", err)
	} else {
		cmd.Println("All settings are valid and saved.")
	}

	return nil
}

func runSettingsMode(cmd *cobra.Command, _ []string) error {
	if settingsService == nil {
		return errors.New("settings service not configured")
	}

	reader := bufio.NewReader(os.Stdin)

	cmd.Println("Select Search Mode")
	cmd.Println("------------------")
	modes := domain.AllSearchModes()
	for i, mode := range modes {
		cmd.Printf("  %d. %s\n", i+1, mode.Description())
	}
	cmd.Print("\nEnter choice: ")
	input := readLine(reader)
	idx := parseChoice(input, len(modes), 0)
	if idx == 0 {
		return errors.New("invalid selection")
	}

	selectedMode := modes[idx-1]
	if err := settingsService.SetSearchMode(selectedMode); err != nil {
		return fmt.Errorf("failed to set search mode: %w", err)
	}

	cmd.Printf("Search mode set to: %s\n", selectedMode.Description())

	// Check if additional configuration is needed
	if selectedMode.RequiresEmbedding() {
		settings, _ := settingsService.Get() //nolint:errcheck // Best-effort check
		if settings != nil && !settings.Embedding.IsConfigured() {
			cmd.Println("\nNote: This mode requires an embedding provider.")
			cmd.Println("Run 'sercha settings embedding' to configure.")
		}
	}
	if selectedMode.RequiresLLM() {
		settings, _ := settingsService.Get() //nolint:errcheck // Best-effort check
		if settings != nil && !settings.LLM.IsConfigured() {
			cmd.Println("\nNote: This mode requires an LLM provider.")
			cmd.Println("Run 'sercha settings llm' to configure.")
		}
	}

	return nil
}

func runSettingsEmbedding(cmd *cobra.Command, _ []string) error {
	if settingsService == nil {
		return errors.New("settings service not configured")
	}

	reader := bufio.NewReader(os.Stdin)
	return configureEmbeddingProvider(cmd, reader)
}

func runSettingsLLM(cmd *cobra.Command, _ []string) error {
	if settingsService == nil {
		return errors.New("settings service not configured")
	}

	reader := bufio.NewReader(os.Stdin)
	return configureLLMProvider(cmd, reader)
}

//nolint:dupl // Similar to configureLLMProvider but for embeddings - intentional for CLI flow clarity
func configureEmbeddingProvider(cmd *cobra.Command, reader *bufio.Reader) error {
	cmd.Println("Select Embedding Provider")
	providers := domain.AllEmbeddingProviders()
	for i, p := range providers {
		cmd.Printf("  %d. %s\n", i+1, p.Description())
	}
	cmd.Print("\nEnter choice [1]: ")
	input := readLine(reader)
	idx := parseChoice(input, len(providers), 1)
	selectedProvider := providers[idx-1]

	// Get model
	defaults := domain.DefaultEmbeddingModels()
	defaultModel := defaults[selectedProvider]
	cmd.Printf("Enter model name [%s]: ", defaultModel)
	model := readLine(reader)
	if model == "" {
		model = defaultModel
	}

	// Get API key if needed
	var apiKey string
	if selectedProvider.RequiresAPIKey() {
		cmd.Print("Enter API key: ")
		apiKey = readPassword()
		cmd.Println()
		if apiKey == "" {
			return errors.New("API key is required for this provider")
		}
	}

	if err := settingsService.SetEmbeddingProvider(selectedProvider, model, apiKey); err != nil {
		return fmt.Errorf("failed to configure embedding provider: %w", err)
	}

	// Validate the configuration by pinging the service
	cmd.Print("Validating configuration... ")
	if err := settingsService.ValidateEmbeddingConfig(); err != nil {
		cmd.Printf("FAILED: %v\n", err)
		return fmt.Errorf("embedding configuration validation failed: %w", err)
	}
	cmd.Println("OK")

	cmd.Printf("Embedding provider configured: %s (%s)\n\n", selectedProvider.Description(), model)
	return nil
}

//nolint:dupl // Similar to configureEmbeddingProvider but for LLM - intentional for CLI flow clarity
func configureLLMProvider(cmd *cobra.Command, reader *bufio.Reader) error {
	cmd.Println("Select LLM Provider")
	providers := domain.AllLLMProviders()
	for i, p := range providers {
		cmd.Printf("  %d. %s\n", i+1, p.Description())
	}
	cmd.Print("\nEnter choice [1]: ")
	input := readLine(reader)
	idx := parseChoice(input, len(providers), 1)
	selectedProvider := providers[idx-1]

	// Get model
	defaults := domain.DefaultLLMModels()
	defaultModel := defaults[selectedProvider]
	cmd.Printf("Enter model name [%s]: ", defaultModel)
	model := readLine(reader)
	if model == "" {
		model = defaultModel
	}

	// Get API key if needed
	var apiKey string
	if selectedProvider.RequiresAPIKey() {
		cmd.Print("Enter API key: ")
		apiKey = readPassword()
		cmd.Println()
		if apiKey == "" {
			return errors.New("API key is required for this provider")
		}
	}

	if err := settingsService.SetLLMProvider(selectedProvider, model, apiKey); err != nil {
		return fmt.Errorf("failed to configure LLM provider: %w", err)
	}

	// Validate the configuration by pinging the service
	cmd.Print("Validating configuration... ")
	if err := settingsService.ValidateLLMConfig(); err != nil {
		cmd.Printf("FAILED: %v\n", err)
		return fmt.Errorf("LLM configuration validation failed: %w", err)
	}
	cmd.Println("OK")

	cmd.Printf("LLM provider configured: %s (%s)\n\n", selectedProvider.Description(), model)
	return nil
}

// Helper functions.

//nolint:errcheck // CLI helper, error ignored for UX
func readLine(reader *bufio.Reader) string {
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

func parseChoice(input string, maxVal, defaultVal int) int {
	if input == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(input)
	if err != nil || val < 1 || val > maxVal {
		return defaultVal
	}
	return val
}

//nolint:errcheck // CLI helper, error ignored for UX
func readPassword() string {
	// Try to read password without echo
	if term.IsTerminal(int(os.Stdin.Fd())) {
		password, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err == nil {
			return string(password)
		}
	}
	// Fallback to regular input
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "..." + key[len(key)-4:]
}
