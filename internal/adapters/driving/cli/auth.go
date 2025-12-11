package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driving"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage OAuth app configurations",
	Long: `Add, list, and remove OAuth application configurations.

OAuth apps store credentials (client_id, client_secret, scopes) that enable
OAuth authentication for sources. For example, one Google OAuth app can be
used for Google Drive, Gmail, and Google Calendar sources.

For connectors supporting PAT (Personal Access Token), you can skip auth setup
and use --token directly with 'sercha source add'.

Examples:
  # Add OAuth app for Google
  sercha auth add --provider google

  # Add OAuth app non-interactively
  sercha auth add --provider github --client-id "xxx" --client-secret "yyy"

  # List configured OAuth apps
  sercha auth list

  # Add source using OAuth app
  sercha source add github --auth <auth-id> -c content_types=files

  # Add source using PAT (no auth setup needed)
  sercha source add github --token ghp_xxx -c content_types=files`,
}

var authAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new OAuth app configuration",
	Long: `Add a new OAuth application configuration.

This command configures an OAuth application for a provider. You can run it
interactively (wizard mode) or non-interactively with all flags provided.

Examples:
  sercha auth add                          # Interactive wizard
  sercha auth add --name "My Google App" --provider google

  # Non-interactive (all required flags):
  sercha auth add \
    --name "My Google App" \
    --provider google \
    --client-id "YOUR_CLIENT_ID" \
    --client-secret "YOUR_CLIENT_SECRET"`,
	RunE: runAuthAdd,
}

var authListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured OAuth apps",
	RunE:  runAuthList,
}

var authRemoveCmd = &cobra.Command{
	Use:   "remove [auth-id]",
	Short: "Remove an OAuth app configuration",
	Args:  cobra.ExactArgs(1),
	RunE:  runAuthRemove,
}

// Flags for auth add.
var (
	authAddName         string
	authAddProvider     string
	authAddClientID     string
	authAddClientSecret string
	authAddScopes       string
)

func init() {
	// Auth add flags
	authAddCmd.Flags().StringVar(
		&authAddName, "name", "", "Name for the OAuth app configuration")
	authAddCmd.Flags().StringVar(
		&authAddProvider, "provider", "", "Provider type (google, github)")
	authAddCmd.Flags().StringVar(
		&authAddClientID, "client-id", "", "OAuth client ID (for non-interactive mode)")
	authAddCmd.Flags().StringVar(
		&authAddClientSecret, "client-secret", "", "OAuth client secret (for non-interactive mode)")
	authAddCmd.Flags().StringVar(
		&authAddScopes, "scopes", "", "OAuth scopes (comma-separated, uses defaults if not provided)")

	// Add subcommands
	authCmd.AddCommand(authAddCmd)
	authCmd.AddCommand(authListCmd)
	authCmd.AddCommand(authRemoveCmd)
	rootCmd.AddCommand(authCmd)
}

//nolint:gocognit,errcheck,gocyclo,nestif // CLI interactive flow
func runAuthAdd(cmd *cobra.Command, _ []string) error {
	if authProviderService == nil {
		return errors.New("auth provider service not configured")
	}
	if providerRegistry == nil {
		return errors.New("provider registry not configured")
	}

	ctx := context.Background()

	// Check if we have enough flags for non-interactive mode
	nonInteractive := authAddProvider != "" && authAddClientID != "" && authAddClientSecret != ""
	if nonInteractive {
		return runAuthAddNonInteractive(ctx, cmd)
	}

	// Interactive mode
	reader := bufio.NewReader(os.Stdin)

	// Select provider
	var provider domain.ProviderType
	if authAddProvider != "" {
		provider = domain.ProviderType(authAddProvider)
		// Validate provider supports OAuth
		validProviders := providerRegistry.GetProviders()
		validProvider := false
		for _, p := range validProviders {
			if p == provider && p != domain.ProviderLocal {
				validProvider = true
				break
			}
		}
		if !validProvider {
			return fmt.Errorf("invalid provider for OAuth: %s", authAddProvider)
		}
	} else {
		providers := providerRegistry.GetProviders()
		cmd.Println("Available providers for OAuth apps:")
		idx := 1
		var oauthProviders []domain.ProviderType
		for _, p := range providers {
			if p != domain.ProviderLocal {
				connectors := providerRegistry.GetConnectorsForProvider(p)
				cmd.Printf("  %d. %s (connectors: %s)\n", idx, p, strings.Join(connectors, ", "))
				oauthProviders = append(oauthProviders, p)
				idx++
			}
		}
		cmd.Print("\nSelect provider number: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		var selectedIdx int
		if _, err := fmt.Sscanf(input, "%d", &selectedIdx); err != nil ||
			selectedIdx < 1 || selectedIdx > len(oauthProviders) {
			return fmt.Errorf("invalid selection: %s", input)
		}
		provider = oauthProviders[selectedIdx-1]
	}

	// Get name
	name := authAddName
	if name == "" {
		cmd.Printf("Enter a name for this OAuth app [%s OAuth App]: ", provider)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input != "" {
			name = input
		} else {
			name = fmt.Sprintf("%s OAuth App", provider)
		}
	}

	// Collect OAuth config
	oauth, err := collectOAuthAppConfig(cmd, reader, provider)
	if err != nil {
		return err
	}

	// Build auth provider
	authProvider := domain.AuthProvider{
		ID:           uuid.New().String(),
		Name:         name,
		ProviderType: provider,
		OAuth:        oauth,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Save
	if err := authProviderService.Save(ctx, authProvider); err != nil {
		return fmt.Errorf("failed to create OAuth app: %w", err)
	}

	cmd.Printf("\nOAuth app configuration created: %s\n", authProvider.ID)
	cmd.Println("You can now use this when adding sources with: sercha source add <connector> --auth <id>")

	return nil
}

// runAuthAddNonInteractive creates an auth provider using CLI flags only.
func runAuthAddNonInteractive(ctx context.Context, cmd *cobra.Command) error {
	provider := domain.ProviderType(authAddProvider)

	// Validate provider supports OAuth
	validProviders := providerRegistry.GetProviders()
	validProvider := false
	for _, p := range validProviders {
		if p == provider && p != domain.ProviderLocal {
			validProvider = true
			break
		}
	}
	if !validProvider {
		return fmt.Errorf("invalid provider for OAuth: %s", authAddProvider)
	}

	// Get name (use default if not provided)
	name := authAddName
	if name == "" {
		name = fmt.Sprintf("%s OAuth App", provider)
	}

	// Get OAuth defaults from connector registry
	defaults := getOAuthDefaultsForProvider(provider)

	// Build OAuth config
	oauth := &domain.OAuthProviderConfig{
		ClientID:     authAddClientID,
		ClientSecret: authAddClientSecret,
		AuthURL:      defaults.AuthURL,
		TokenURL:     defaults.TokenURL,
	}

	// Parse scopes (use defaults if not provided)
	if authAddScopes != "" {
		scopes := strings.Split(authAddScopes, ",")
		for i := range scopes {
			scopes[i] = strings.TrimSpace(scopes[i])
		}
		oauth.Scopes = scopes
	} else {
		oauth.Scopes = defaults.Scopes
	}

	// Build auth provider
	authProvider := domain.AuthProvider{
		ID:           uuid.New().String(),
		Name:         name,
		ProviderType: provider,
		OAuth:        oauth,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Save
	if err := authProviderService.Save(ctx, authProvider); err != nil {
		return fmt.Errorf("failed to create OAuth app: %w", err)
	}

	cmd.Printf("OAuth app configuration created: %s\n", authProvider.ID)
	return nil
}

//nolint:errcheck // CLI interactive flow
func collectOAuthAppConfig(
	cmd *cobra.Command,
	reader *bufio.Reader,
	provider domain.ProviderType,
) (*domain.OAuthProviderConfig, error) {
	cmd.Println("\nOAuth Application Configuration")
	cmd.Println("-------------------------------")
	cmd.Println("You need to create an OAuth application with your provider")
	cmd.Println("and enter the credentials below.")
	cmd.Println()

	// Get OAuth defaults from connector registry
	defaults := getOAuthDefaultsForProvider(provider)

	// Provide setup hint
	hint := getProviderSetupHint(provider)
	if hint != "" {
		cmd.Println(hint)
		cmd.Println()
	}

	oauth := &domain.OAuthProviderConfig{}

	// Client ID
	cmd.Print("Client ID: ")
	input, _ := reader.ReadString('\n')
	oauth.ClientID = strings.TrimSpace(input)
	if oauth.ClientID == "" {
		return nil, errors.New("client ID is required")
	}

	// Client Secret
	cmd.Print("Client Secret: ")
	input, _ = reader.ReadString('\n')
	oauth.ClientSecret = strings.TrimSpace(input)
	if oauth.ClientSecret == "" {
		return nil, errors.New("client secret is required")
	}

	// Auth URL
	if defaults.AuthURL != "" {
		cmd.Printf("Authorization URL [%s]: ", defaults.AuthURL)
	} else {
		cmd.Print("Authorization URL: ")
	}
	input, _ = reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input != "" {
		oauth.AuthURL = input
	} else if defaults.AuthURL != "" {
		oauth.AuthURL = defaults.AuthURL
	} else {
		return nil, errors.New("authorization URL is required")
	}

	// Token URL
	if defaults.TokenURL != "" {
		cmd.Printf("Token URL [%s]: ", defaults.TokenURL)
	} else {
		cmd.Print("Token URL: ")
	}
	input, _ = reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input != "" {
		oauth.TokenURL = input
	} else if defaults.TokenURL != "" {
		oauth.TokenURL = defaults.TokenURL
	} else {
		return nil, errors.New("token URL is required")
	}

	// Scopes
	if len(defaults.Scopes) > 0 {
		cmd.Printf("Scopes (comma-separated) [%s]: ", strings.Join(defaults.Scopes, ","))
	} else {
		cmd.Print("Scopes (comma-separated): ")
	}
	input, _ = reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input != "" {
		oauth.Scopes = strings.Split(input, ",")
		for i := range oauth.Scopes {
			oauth.Scopes[i] = strings.TrimSpace(oauth.Scopes[i])
		}
	} else if len(defaults.Scopes) > 0 {
		oauth.Scopes = defaults.Scopes
	}

	return oauth, nil
}

func runAuthList(cmd *cobra.Command, _ []string) error {
	if authProviderService == nil {
		return errors.New("auth provider service not configured")
	}

	ctx := context.Background()
	providers, err := authProviderService.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list OAuth apps: %w", err)
	}

	if len(providers) == 0 {
		cmd.Println("No configured OAuth apps.")
		cmd.Println("Add one with: sercha auth add")
		return nil
	}

	cmd.Println("Configured OAuth apps:")
	cmd.Println()
	for i := range providers {
		cmd.Printf("  %s\n", providers[i].ID)
		cmd.Printf("    Name: %s\n", providers[i].Name)
		cmd.Printf("    Provider: %s\n", providers[i].ProviderType)
		if providers[i].OAuth != nil {
			cmd.Printf("    Client ID: %s...\n", truncate(providers[i].OAuth.ClientID, 20))
			cmd.Printf("    Scopes: %s\n", strings.Join(providers[i].OAuth.Scopes, ", "))
		}
		cmd.Printf("    Created: %s\n", providers[i].CreatedAt.Format(time.RFC3339))
		cmd.Println()
	}

	return nil
}

func runAuthRemove(cmd *cobra.Command, args []string) error {
	if authProviderService == nil {
		return errors.New("auth provider service not configured")
	}

	authID := args[0]
	ctx := context.Background()

	// Verify it exists
	provider, err := authProviderService.Get(ctx, authID)
	if err != nil {
		return fmt.Errorf("OAuth app not found: %w", err)
	}

	if err := authProviderService.Delete(ctx, authID); err != nil {
		if errors.Is(err, domain.ErrAuthProviderInUse) {
			return fmt.Errorf("cannot remove: OAuth app is in use by one or more sources")
		}
		return fmt.Errorf("failed to remove OAuth app: %w", err)
	}

	cmd.Printf("Removed OAuth app: %s (%s)\n", provider.Name, authID)
	return nil
}

// truncate truncates a string to the specified length.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

// getOAuthDefaultsForProvider returns OAuth defaults for a provider by looking up
// the first connector that supports OAuth for that provider.
func getOAuthDefaultsForProvider(provider domain.ProviderType) *driving.OAuthDefaults {
	if providerRegistry == nil || connectorRegistry == nil {
		return &driving.OAuthDefaults{}
	}

	// Get connectors for this provider
	connectors := providerRegistry.GetConnectorsForProvider(provider)
	if len(connectors) == 0 {
		return &driving.OAuthDefaults{}
	}

	// Find first connector with OAuth support
	for _, connectorType := range connectors {
		if connectorRegistry.SupportsOAuth(connectorType) {
			defaults := connectorRegistry.GetOAuthDefaults(connectorType)
			if defaults != nil {
				return defaults
			}
		}
	}

	return &driving.OAuthDefaults{}
}

// getProviderSetupHint returns guidance text for setting up OAuth with a provider.
// Uses providerRegistry to get connectors, then connectorRegistry to get the hint.
func getProviderSetupHint(provider domain.ProviderType) string {
	if providerRegistry == nil || connectorRegistry == nil {
		return ""
	}

	// Get connectors for this provider
	connectors := providerRegistry.GetConnectorsForProvider(provider)
	if len(connectors) == 0 {
		return ""
	}

	// Get hint from registry using the first connector that supports OAuth
	for _, connectorID := range connectors {
		if hint := connectorRegistry.GetSetupHint(connectorID); hint != "" {
			return hint
		}
	}
	return ""
}
