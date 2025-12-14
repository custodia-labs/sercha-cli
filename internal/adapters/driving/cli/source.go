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

	"github.com/custodia-labs/sercha-cli/internal/adapters/driving/oauth"
	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

var sourceCmd = &cobra.Command{
	Use:   "source",
	Short: "Manage document sources",
	Long:  `Add, list, or remove document sources (directories, GitHub, Google Drive, etc.).`,
}

var sourceAddCmd = &cobra.Command{
	Use:   "add [connector-type]",
	Short: "Add a new document source",
	Long: `Add a new document source using a connector type.

Available connector types can be listed with 'sercha connector list'.
For OAuth connectors, create an auth provider first with 'sercha auth add'.

Non-interactive mode requires the connector type as an argument and uses -c flags for config.
Interactive mode (no arguments) will prompt for all required information.

Examples:
  # Interactive mode
  sercha source add

  # Non-interactive: filesystem source (no auth required)
  sercha source add filesystem -c path=/Users/me/Documents

  # Non-interactive: GitHub source with PAT token
  sercha source add github --token ghp_xxx -c content_types=files,issues

  # Non-interactive: GitHub source with OAuth
  sercha source add github --auth <auth-id> -c content_types=files,issues

  # Specify auth method explicitly (for connectors supporting both)
  sercha source add github --auth-method token --token ghp_xxx -c content_types=files`,
	Args: cobra.MaximumNArgs(1),
	RunE: runSourceAdd,
}

var sourceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured sources",
	RunE:  runSourceList,
}

var sourceRemoveCmd = &cobra.Command{
	Use:   "remove [source-id]",
	Short: "Remove a document source",
	Args:  cobra.ExactArgs(1),
	RunE:  runSourceRemove,
}

var connectorCmd = &cobra.Command{
	Use:   "connector",
	Short: "Manage connectors",
	Long:  `List available connector types and their configuration.`,
}

var connectorListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available connector types",
	RunE:  runConnectorList,
}

// Flags for source add.
var (
	sourceName       string
	sourceConfig     []string
	sourceAuth       string // --auth flag for AuthProvider ID
	sourceToken      string
	sourceAuthMethod string
)

// authSelectionResult holds the result of auth selection for the new system.
// Credentials are NOT saved yet - they will be saved after the source is created.
type authSelectionResult struct {
	// AuthProviderID is the ID of the OAuth app configuration (for OAuth auth).
	AuthProviderID string
	// AccountIdentifier is the user's email/username from the provider.
	AccountIdentifier string
	// PendingCredentials holds credential data to save AFTER source creation.
	// This is nil for no-auth connectors.
	PendingCredentials *pendingCredentials
}

// pendingCredentials holds credential data before it's saved to the database.
// We need this because credentials have a FK to source_id, so source must exist first.
type pendingCredentials struct {
	OAuth *domain.OAuthCredentials
	PAT   *domain.PATCredentials
}

func init() {
	// Source commands
	sourceAddCmd.Flags().StringVar(&sourceName, "name", "", "Name for the source (defaults to connector type)")
	sourceAddCmd.Flags().StringVar(
		&sourceAuth, "auth", "",
		"Auth provider ID for OAuth authentication (see 'sercha auth list')")
	sourceAddCmd.Flags().StringVar(
		&sourceToken, "token", "",
		"Personal Access Token for PAT authentication (non-interactive)")
	sourceAddCmd.Flags().StringVar(
		&sourceAuthMethod, "auth-method", "",
		"Authentication method: 'token' or 'oauth' (for connectors supporting both)")
	sourceAddCmd.Flags().StringArrayVarP(
		&sourceConfig, "config", "c", nil,
		"Configuration key=value pairs (can be repeated)")
	sourceCmd.AddCommand(sourceAddCmd)
	sourceCmd.AddCommand(sourceListCmd)
	sourceCmd.AddCommand(sourceRemoveCmd)
	rootCmd.AddCommand(sourceCmd)

	// Connector commands
	connectorCmd.AddCommand(connectorListCmd)
	rootCmd.AddCommand(connectorCmd)
}

func runConnectorList(cmd *cobra.Command, _ []string) error {
	if connectorRegistry == nil {
		return errors.New("connector registry not configured")
	}

	connectors := connectorRegistry.List()
	if len(connectors) == 0 {
		cmd.Println("No connectors available.")
		return nil
	}

	cmd.Println("Available connectors:")
	cmd.Println()
	for _, c := range connectors {
		cmd.Printf("  %s\n", c.ID)
		cmd.Printf("    Name: %s\n", c.Name)
		cmd.Printf("    Description: %s\n", c.Description)
		cmd.Printf("    Provider: %s\n", c.ProviderType)
		// Show auth capability
		authDesc := "none"
		if c.AuthCapability.SupportsMultipleMethods() {
			authDesc = "token/oauth"
		} else if c.AuthCapability.SupportsPAT() {
			authDesc = "token"
		} else if c.AuthCapability.SupportsOAuth() {
			authDesc = "oauth"
		}
		cmd.Printf("    Auth: %s\n", authDesc)
		if len(c.ConfigKeys) > 0 {
			cmd.Println("    Config:")
			for _, key := range c.ConfigKeys {
				req := ""
				if key.Required {
					req = " (required)"
				}
				cmd.Printf("      --%s: %s%s\n", key.Key, key.Description, req)
			}
		}
		cmd.Println()
	}

	return nil
}

//nolint:gocognit,errcheck,gocyclo,funlen // CLI interactive flow with intentional error ignoring for UX
func runSourceAdd(cmd *cobra.Command, args []string) error {
	if sourceService == nil {
		return errors.New("source service not configured")
	}
	if connectorRegistry == nil {
		return errors.New("connector registry not configured")
	}

	ctx := context.Background()

	// Parse config flags into map
	configFromFlags := make(map[string]string)
	for _, kv := range sourceConfig {
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) == 2 {
			configFromFlags[parts[0]] = parts[1]
		} else {
			return fmt.Errorf("invalid config format: %s (expected key=value)", kv)
		}
	}

	// Determine if running non-interactively (connector type provided as arg)
	isNonInteractive := len(args) > 0

	// Get connector type from args or prompt
	var connectorType string
	if len(args) > 0 {
		connectorType = args[0]
	} else {
		// Interactive mode: list connectors and prompt
		connectors := connectorRegistry.List()
		cmd.Println("Available connectors:")
		for i, c := range connectors {
			cmd.Printf("  %d. %s - %s\n", i+1, c.ID, c.Description)
		}
		cmd.Print("\nEnter connector number: ")
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		var idx int
		if _, err := fmt.Sscanf(input, "%d", &idx); err != nil || idx < 1 || idx > len(connectors) {
			return fmt.Errorf("invalid selection: %s", input)
		}
		connectorType = connectors[idx-1].ID
	}

	// Get connector info
	connector, err := connectorRegistry.Get(connectorType)
	if err != nil {
		return fmt.Errorf("unknown connector type: %s", connectorType)
	}

	// Generate source ID early (needed for Credentials.SourceID)
	sourceID := uuid.New().String()

	// Handle authentication using new AuthProvider/Credentials system
	authResult, err := selectAuthWithNewSystem(ctx, cmd, connector, sourceID, isNonInteractive)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	// Collect configuration from flags or prompts
	config := make(map[string]string)
	reader := bufio.NewReader(os.Stdin)

	for _, key := range connector.ConfigKeys {
		// Try to get from --config flags first
		if val, ok := configFromFlags[key.Key]; ok {
			config[key.Key] = val
			continue
		}

		// In non-interactive mode, fail if required field missing
		if isNonInteractive {
			if key.Required {
				return fmt.Errorf("required config missing: -c %s=<value>", key.Key)
			}
			continue
		}

		// Prompt for value (interactive mode)
		prompt := key.Label
		if !key.Required {
			prompt += " (optional)"
		}
		if key.Secret {
			prompt += " [hidden]"
		}
		cmd.Printf("%s: ", prompt)

		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input != "" {
			config[key.Key] = input
		} else if key.Required {
			return fmt.Errorf("required field %s not provided", key.Key)
		}
	}

	// Generate name (use account identifier if available for clarity)
	name := sourceName
	//nolint:nestif // Intentional nesting for name derivation logic
	if name == "" {
		name = connector.Name
		if val, ok := config["path"]; ok {
			name = val
		} else if val, ok := config["owner"]; ok {
			if repo, ok := config["repo"]; ok {
				name = val + "/" + repo
			}
		}
		// Append account identifier for OAuth sources
		if authResult.AccountIdentifier != "" {
			name = fmt.Sprintf("%s (%s)", name, authResult.AccountIdentifier)
		}
	}

	// Create and save source FIRST (without credentials_id)
	// Credentials have FK to source, so source must exist before credentials
	source := domain.Source{
		ID:             sourceID,
		Type:           connectorType,
		Name:           name,
		Config:         config,
		AuthProviderID: authResult.AuthProviderID,
		// CredentialsID will be set after credentials are saved
	}

	if err := sourceService.Add(ctx, source); err != nil {
		return fmt.Errorf("failed to add source: %w", err)
	}

	// Now save credentials (if any) - source exists, FK constraint satisfied
	var credentialsID string
	if authResult.PendingCredentials != nil {
		now := time.Now()
		creds := domain.Credentials{
			ID:                uuid.New().String(),
			SourceID:          sourceID,
			AccountIdentifier: authResult.AccountIdentifier,
			OAuth:             authResult.PendingCredentials.OAuth,
			PAT:               authResult.PendingCredentials.PAT,
			CreatedAt:         now,
			UpdatedAt:         now,
		}
		if err := credentialsService.Save(ctx, creds); err != nil {
			// Rollback source creation
			_ = sourceService.Remove(ctx, sourceID)
			return fmt.Errorf("failed to save credentials: %w", err)
		}
		credentialsID = creds.ID

		// Update source with credentials_id
		source.CredentialsID = credentialsID
		if err := sourceService.Update(ctx, source); err != nil {
			// Best effort - source exists but credentials_id not linked
			cmd.Printf("Warning: failed to link credentials to source: %v\n", err)
		}
	}

	cmd.Printf("Added source: %s (%s)\n", sourceID, connector.Name)
	if authResult.AuthProviderID != "" {
		cmd.Printf("Using OAuth app: %s\n", authResult.AuthProviderID)
	}
	if authResult.AccountIdentifier != "" {
		cmd.Printf("Account: %s\n", authResult.AccountIdentifier)
	}
	return nil
}

func runSourceList(cmd *cobra.Command, _ []string) error {
	if sourceService == nil {
		return errors.New("source service not configured")
	}

	ctx := context.Background()
	sources, err := sourceService.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list sources: %w", err)
	}

	if len(sources) == 0 {
		cmd.Println("No configured sources.")
		return nil
	}

	cmd.Println("Configured sources:")
	cmd.Println()
	for i := range sources {
		cmd.Printf("  %s\n", sources[i].ID)
		cmd.Printf("    Type: %s\n", sources[i].Type)
		cmd.Printf("    Name: %s\n", sources[i].Name)
		// Show new auth system info
		if sources[i].AuthProviderID != "" && authProviderService != nil {
			if provider, err := authProviderService.Get(ctx, sources[i].AuthProviderID); err == nil {
				cmd.Printf("    OAuth App: %s (%s)\n", provider.Name, provider.ID[:8])
			}
		}
		if sources[i].CredentialsID != "" && credentialsService != nil {
			if creds, err := credentialsService.Get(ctx, sources[i].CredentialsID); err == nil {
				if creds.AccountIdentifier != "" {
					cmd.Printf("    Account: %s\n", creds.AccountIdentifier)
				}
			}
		}
		cmd.Println()
	}

	return nil
}

func runSourceRemove(cmd *cobra.Command, args []string) error {
	if sourceService == nil {
		return errors.New("source service not configured")
	}

	sourceID := args[0]
	ctx := context.Background()

	if err := sourceService.Remove(ctx, sourceID); err != nil {
		return fmt.Errorf("failed to remove source: %w", err)
	}

	cmd.Printf("Removed source: %s\n", sourceID)
	cmd.Println("Note: Associated credentials were not removed.")
	return nil
}

// selectAuthWithNewSystem handles authentication using the new AuthProvider/Credentials architecture.
// For OAuth connectors: selects/creates AuthProvider, runs OAuth flow, creates Credentials.
// For PAT connectors: prompts for PAT, creates Credentials.
// For no-auth connectors: returns empty result.
//
//nolint:errcheck,gocyclo,gocognit,nestif // CLI interactive flow
func selectAuthWithNewSystem(
	ctx context.Context,
	cmd *cobra.Command,
	connector *domain.ConnectorType,
	sourceID string,
	isNonInteractive bool,
) (*authSelectionResult, error) {
	result := &authSelectionResult{}

	// No auth needed for this connector
	if !connector.AuthCapability.RequiresAuth() {
		return result, nil
	}

	// Check required services
	if authProviderService == nil {
		return nil, errors.New("auth provider service not configured")
	}
	if credentialsService == nil {
		return nil, errors.New("credentials service not configured")
	}

	reader := bufio.NewReader(os.Stdin)

	// Determine auth method (if connector supports multiple)
	var chosenMethod domain.AuthMethod

	// Non-interactive mode: determine auth method from flags
	if isNonInteractive {
		// Check if auth method was explicitly specified
		if sourceAuthMethod != "" {
			switch strings.ToLower(sourceAuthMethod) {
			case "token", "pat":
				if !connector.AuthCapability.SupportsPAT() {
					return nil, fmt.Errorf("connector %s does not support PAT authentication", connector.ID)
				}
				chosenMethod = domain.AuthMethodPAT
			case "oauth":
				if !connector.AuthCapability.SupportsOAuth() {
					return nil, fmt.Errorf("connector %s does not support OAuth authentication", connector.ID)
				}
				chosenMethod = domain.AuthMethodOAuth
			default:
				return nil, fmt.Errorf("invalid --auth-method: %s (use 'token' or 'oauth')", sourceAuthMethod)
			}
		} else if sourceToken != "" {
			// Token provided, use PAT auth
			if !connector.AuthCapability.SupportsPAT() {
				return nil, fmt.Errorf("connector %s does not support PAT authentication", connector.ID)
			}
			chosenMethod = domain.AuthMethodPAT
		} else if sourceAuth != "" {
			// Auth provider ID provided, use OAuth
			if !connector.AuthCapability.SupportsOAuth() {
				return nil, fmt.Errorf("connector %s does not support OAuth authentication", connector.ID)
			}
			chosenMethod = domain.AuthMethodOAuth
		} else {
			// No auth flags, check if connector requires auth
			if connector.AuthCapability.SupportsPAT() {
				return nil, fmt.Errorf(
					"connector %s requires authentication. Use --token for PAT or --auth for OAuth",
					connector.ID)
			} else if connector.AuthCapability.SupportsOAuth() {
				return nil, fmt.Errorf(
					"connector %s requires OAuth authentication. Use --auth flag",
					connector.ID)
			}
		}
	} else if connector.AuthCapability.SupportsMultipleMethods() {
		// Interactive mode: prompt for auth method
		methods := connector.AuthCapability.SupportedMethods()
		cmd.Printf("\n%s supports multiple authentication methods:\n", connector.Name)
		for i, method := range methods {
			desc := ""
			switch method {
			case domain.AuthMethodPAT:
				desc = "Personal Access Token - Use a token from your account settings"
			case domain.AuthMethodOAuth:
				desc = "OAuth App - Authenticate via browser with OAuth"
			case domain.AuthMethodNone:
				desc = "No authentication required"
			}
			cmd.Printf("  %d. %s\n", i+1, desc)
		}
		cmd.Print("\nChoose auth method: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		var idx int
		if _, err := fmt.Sscanf(input, "%d", &idx); err != nil || idx < 1 || idx > len(methods) {
			return nil, fmt.Errorf("invalid selection: %s", input)
		}
		chosenMethod = methods[idx-1]
	} else if connector.AuthCapability.SupportsPAT() {
		chosenMethod = domain.AuthMethodPAT
	} else {
		chosenMethod = domain.AuthMethodOAuth
	}

	// Handle based on chosen method
	//nolint:exhaustive // AuthMethodNone is explicitly handled above
	switch chosenMethod {
	case domain.AuthMethodPAT:
		return handlePATAuth(ctx, cmd, reader, connector, sourceID, isNonInteractive)
	case domain.AuthMethodOAuth:
		return handleOAuthAuth(ctx, cmd, reader, connector, sourceID, isNonInteractive)
	default:
		return result, nil
	}
}

// handlePATAuth handles PAT authentication flow.
//
//nolint:errcheck,nestif // CLI interactive flow
func handlePATAuth(
	_ context.Context,
	cmd *cobra.Command,
	reader *bufio.Reader,
	connector *domain.ConnectorType,
	_ string, // sourceID - unused, credentials are pending until source is created
	isNonInteractive bool,
) (*authSelectionResult, error) {
	result := &authSelectionResult{}

	var token string
	var accountID string

	// Non-interactive mode: use --token flag
	if isNonInteractive {
		if sourceToken == "" {
			return nil, errors.New("--token flag required for PAT authentication in non-interactive mode")
		}
		token = sourceToken
	} else {
		// Interactive mode: prompt for token
		cmd.Println("\nPersonal Access Token Configuration")
		cmd.Println("-----------------------------------")

		// Provide hint from connector registry
		if hint := connectorRegistry.GetSetupHint(connector.ID); hint != "" {
			cmd.Println(hint)
		}
		cmd.Println()

		// Get token
		cmd.Print("Enter your personal access token: ")
		var err error
		token, err = reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("failed to read token: %w", err)
		}
		token = strings.TrimSpace(token)
		if token == "" {
			return nil, errors.New("token is required")
		}

		// Get account identifier (optional for PAT)
		cmd.Print("Enter account identifier (email/username, optional): ")
		accountID, _ = reader.ReadString('\n')
		accountID = strings.TrimSpace(accountID)
	}

	result.AccountIdentifier = accountID

	// Store credentials as pending (will be saved AFTER source is created)
	// This avoids FK constraint violation since credentials.source_id must reference existing source
	result.PendingCredentials = &pendingCredentials{
		PAT: &domain.PATCredentials{
			Token: token,
		},
	}

	if !isNonInteractive {
		cmd.Println("\nCredentials prepared.")
	}
	return result, nil
}

// handleOAuthAuth handles OAuth authentication flow.
//
//nolint:errcheck,gocyclo,gocognit,funlen,nestif // CLI interactive flow
func handleOAuthAuth(
	ctx context.Context,
	cmd *cobra.Command,
	reader *bufio.Reader,
	connector *domain.ConnectorType,
	_ string, // sourceID - unused, credentials are pending until source is created
	isNonInteractive bool,
) (*authSelectionResult, error) {
	result := &authSelectionResult{}

	// Check for --auth flag
	var authProvider *domain.AuthProvider
	if sourceAuth != "" {
		provider, err := authProviderService.Get(ctx, sourceAuth)
		if err != nil {
			return nil, fmt.Errorf("auth provider not found: %s", sourceAuth)
		}
		if provider.ProviderType != connector.ProviderType {
			return nil, fmt.Errorf(
				"auth provider %s is for %s, but connector requires %s",
				sourceAuth, provider.ProviderType, connector.ProviderType)
		}
		authProvider = provider
		result.AuthProviderID = provider.ID
	} else if isNonInteractive {
		return nil, fmt.Errorf("--auth flag required for OAuth connector %s", connector.ID)
	}

	// Interactive: select or create AuthProvider
	if authProvider == nil {
		providers, err := authProviderService.ListByProvider(ctx, connector.ProviderType)
		if err != nil {
			return nil, fmt.Errorf("failed to list auth providers: %w", err)
		}

		if len(providers) == 0 {
			cmd.Printf("\nNo OAuth app configurations found for %s.\n", connector.ProviderType)
			cmd.Print("Would you like to create one now? [Y/n]: ")
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(strings.ToLower(input))
			if input != "" && input != "y" && input != "yes" {
				return nil, errors.New("OAuth app configuration required for this connector")
			}

			// Create AuthProvider inline
			newProvider, err := createAuthProviderInline(ctx, cmd, reader, connector.ProviderType)
			if err != nil {
				return nil, err
			}
			authProvider = newProvider
			result.AuthProviderID = newProvider.ID
		} else {
			// Select existing AuthProvider
			cmd.Println("\nAvailable OAuth app configurations:")
			for i := range providers {
				cmd.Printf("  %d. %s (%s)\n", i+1, providers[i].Name, providers[i].ID[:8])
			}
			cmd.Printf("  %d. Create new OAuth app configuration\n", len(providers)+1)
			cmd.Print("\nSelect number: ")
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(input)
			var idx int
			if _, err := fmt.Sscanf(input, "%d", &idx); err != nil || idx < 1 || idx > len(providers)+1 {
				return nil, fmt.Errorf("invalid selection: %s", input)
			}

			if idx <= len(providers) {
				authProvider = &providers[idx-1]
				result.AuthProviderID = authProvider.ID
			} else {
				// Create new AuthProvider
				newProvider, err := createAuthProviderInline(ctx, cmd, reader, connector.ProviderType)
				if err != nil {
					return nil, err
				}
				authProvider = newProvider
				result.AuthProviderID = newProvider.ID
			}
		}
	}

	// Run OAuth flow to get tokens
	cmd.Println("\nStarting OAuth authentication...")

	// Verify OAuth configuration exists
	if authProvider.OAuth == nil {
		return nil, errors.New("auth provider has no OAuth configuration")
	}

	// Generate PKCE verifier and challenge
	state := uuid.New().String()
	codeVerifier := oauth.GenerateCodeVerifier()
	codeChallenge := oauth.GenerateCodeChallenge(codeVerifier)

	// Start callback server on fixed port (must match OAuth redirect URI in provider settings)
	// Register http://localhost:18080/callback in your OAuth app's redirect URIs
	const oauthCallbackPort = 18080
	callbackServer := oauth.NewCallbackServer(oauthCallbackPort, state)
	if err := callbackServer.Start(); err != nil {
		return nil, fmt.Errorf("failed to start callback server: %w", err)
	}
	defer callbackServer.Stop()

	// Build auth URL via connector registry (includes provider-specific params)
	authURL, err := connectorRegistry.BuildAuthURL(
		connector.ID, authProvider, callbackServer.RedirectURI(), state, codeChallenge)
	if err != nil {
		return nil, fmt.Errorf("failed to build auth URL: %w", err)
	}

	cmd.Println("\nOpening browser for authentication...")
	cmd.Printf("If the browser doesn't open, visit:\n%s\n", authURL)

	if err := oauth.OpenBrowser(authURL); err != nil {
		cmd.Printf("Failed to open browser: %v\n", err)
	}

	cmd.Println("\nWaiting for authorization...")

	// Wait for callback
	code, err := callbackServer.WaitForCode(5 * time.Minute)
	if err != nil {
		return nil, fmt.Errorf("authorization failed: %w", err)
	}

	// Exchange code for tokens via connector-specific handler
	// This allows connectors like Notion to use their custom token exchange
	cmd.Println("Exchanging authorization code for tokens...")
	tokens, err := connectorRegistry.ExchangeCode(
		ctx, connector.ID, authProvider, code, callbackServer.RedirectURI(), codeVerifier,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code for tokens: %w", err)
	}

	// Get account identifier from provider via connector registry
	accountID, err := connectorRegistry.GetUserInfo(ctx, connector.ID, tokens.AccessToken)
	if err != nil {
		cmd.Printf("Warning: could not fetch account identifier: %v\n", err)
	}
	result.AccountIdentifier = accountID

	// Store credentials as pending (will be saved AFTER source is created)
	// This avoids FK constraint violation since credentials.source_id must reference existing source
	result.PendingCredentials = &pendingCredentials{
		OAuth: &domain.OAuthCredentials{
			AccessToken:  tokens.AccessToken,
			RefreshToken: tokens.RefreshToken,
			TokenType:    tokens.TokenType,
			Expiry:       tokens.Expiry,
		},
	}

	cmd.Println("Authentication successful!")
	if accountID != "" {
		cmd.Printf("Authenticated as: %s\n", accountID)
	}

	return result, nil
}

// createAuthProviderInline creates an AuthProvider during source add flow.
//
//nolint:errcheck // CLI interactive flow
func createAuthProviderInline(
	ctx context.Context,
	cmd *cobra.Command,
	reader *bufio.Reader,
	providerType domain.ProviderType,
) (*domain.AuthProvider, error) {
	cmd.Printf("\nCreating OAuth app configuration for %s\n", providerType)

	// Get name
	cmd.Printf("Enter a name for this OAuth app [%s OAuth App]: ", providerType)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	name := input
	if name == "" {
		name = fmt.Sprintf("%s OAuth App", providerType)
	}

	// Collect OAuth config
	oauth, err := collectOAuthAppConfig(cmd, reader, providerType)
	if err != nil {
		return nil, err
	}

	// Build auth provider
	now := time.Now()
	authProvider := domain.AuthProvider{
		ID:           uuid.New().String(),
		Name:         name,
		ProviderType: providerType,
		OAuth:        oauth,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	// Save
	if err := authProviderService.Save(ctx, authProvider); err != nil {
		return nil, fmt.Errorf("failed to create auth provider: %w", err)
	}

	cmd.Printf("\nOAuth app configuration created: %s\n", authProvider.ID)
	return &authProvider, nil
}
