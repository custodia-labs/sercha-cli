// Package addsource provides the add source wizard view for the TUI.
package addsource

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"

	drivenoauth "github.com/custodia-labs/sercha-cli/internal/adapters/driven/oauth"
	"github.com/custodia-labs/sercha-cli/internal/adapters/driving/oauth"
	"github.com/custodia-labs/sercha-cli/internal/adapters/driving/tui/messages"
	"github.com/custodia-labs/sercha-cli/internal/adapters/driving/tui/styles"
	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driving"
)

// WizardStep tracks the current step in the wizard.
type WizardStep int

const (
	StepSelectConnector WizardStep = iota
	StepEnterConfig
	StepSelectAuthMethod // Choose PAT vs OAuth for connectors supporting both
	StepSelectAuth       // Only for multi-connector providers with existing auths
	StepEnterCredentials // Inline Client ID/Secret entry
	StepOAuthFlow        // Browser auth + waiting
	StepComplete
)

// Key constants.
const (
	keyEnter = "enter"
	keyDown  = "down"
)

// View is the add source wizard view.
type View struct {
	styles              *styles.Styles
	sourceService       driving.SourceService
	connectorRegistry   driving.ConnectorRegistry
	providerRegistry    driving.ProviderRegistry
	authProviderService driving.AuthProviderService
	credentialsService  driving.CredentialsService

	// Wizard state
	step       WizardStep
	connectors []domain.ConnectorType
	selected   int

	// Selected connector
	connector *domain.ConnectorType

	// Config inputs
	configInputs []textinput.Model
	configKeys   []string
	focusIndex   int

	// Auth method selection (for connectors supporting PAT+OAuth)
	authMethodOptions       []domain.AuthMethod
	selectedAuthMethodIndex int
	chosenAuthMethod        domain.AuthMethod

	// AuthProvider selection (for multi-connector providers with existing OAuth apps)
	authProviders     []domain.AuthProvider
	selectedAuthIndex int
	creatingNewAuth   bool // true if user chose "Add new OAuth app"

	// Inline credential inputs
	clientIDInput     textinput.Model
	clientSecretInput textinput.Model
	tokenInput        textinput.Model // For PAT providers
	credentialFocus   int             // 0 = clientID, 1 = clientSecret, 2 = token

	// OAuth flow state (new AuthProvider + Credentials system)
	oauthState             *driving.OAuthFlowState
	waitingForAuth         bool
	selectedAuthProviderID string                   // ID of selected/created AuthProvider
	pendingOAuthTokens     *domain.OAuthCredentials // Tokens received from OAuth flow
	accountIdentifier      string                   // Account ID fetched after OAuth
	callbackServer         *oauth.CallbackServer

	// Result
	source *domain.Source
	err    error

	width  int
	height int
	ready  bool
}

// NewView creates a new add source wizard view.
func NewView(
	s *styles.Styles,
	sourceService driving.SourceService,
	connectorRegistry driving.ConnectorRegistry,
	providerRegistry driving.ProviderRegistry,
	authProviderService driving.AuthProviderService,
	credentialsService driving.CredentialsService,
) *View {
	// Initialise credential input fields
	clientIDInput := textinput.New()
	clientIDInput.Placeholder = "Client ID from developer console"
	clientIDInput.CharLimit = 256

	clientSecretInput := textinput.New()
	clientSecretInput.Placeholder = "Client Secret from developer console"
	clientSecretInput.EchoMode = textinput.EchoPassword
	clientSecretInput.CharLimit = 256

	tokenInput := textinput.New()
	tokenInput.Placeholder = "Personal Access Token"
	tokenInput.EchoMode = textinput.EchoPassword
	tokenInput.CharLimit = 256

	return &View{
		styles:              s,
		sourceService:       sourceService,
		connectorRegistry:   connectorRegistry,
		providerRegistry:    providerRegistry,
		authProviderService: authProviderService,
		credentialsService:  credentialsService,
		step:                StepSelectConnector,
		clientIDInput:       clientIDInput,
		clientSecretInput:   clientSecretInput,
		tokenInput:          tokenInput,
	}
}

// Init initialises the view and loads connectors.
func (v *View) Init() tea.Cmd {
	return v.loadConnectors()
}

// loadConnectors returns a command that loads available connectors.
func (v *View) loadConnectors() tea.Cmd {
	return func() tea.Msg {
		if v.connectorRegistry == nil {
			return messages.ErrorOccurred{Err: fmt.Errorf("connector registry not available")}
		}
		connectors := v.connectorRegistry.List()
		return connectorsLoaded{connectors: connectors}
	}
}

// connectorsLoaded is a message indicating connectors have been loaded.
type connectorsLoaded struct {
	connectors []domain.ConnectorType
}

// Update handles messages for the add source wizard.
//
//nolint:gocritic // evalOrder: bubbletea pattern returns cmd from method call
func (v *View) Update(msg tea.Msg) (*View, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
		v.ready = true
		return v, nil

	case connectorsLoaded:
		v.connectors = msg.connectors
		return v, nil

	case messages.ErrorOccurred:
		v.err = msg.Err
		return v, nil

	case tea.KeyMsg:
		return v.handleKeyMsg(msg)

	case messages.SourceAdded:
		if msg.Err != nil {
			v.err = msg.Err
		} else {
			v.source = &msg.Source
			v.step = StepComplete
		}
		return v, nil

	case authProvidersLoaded:
		v.authProviders = msg.authProviders
		v.selectedAuthIndex = 0
		// If there are existing auth providers, show the selection step
		// Otherwise go straight to credentials
		if len(msg.authProviders) > 0 {
			v.step = StepSelectAuth
		} else {
			v.creatingNewAuth = false
			v.initCredentialInputs()
			v.step = StepEnterCredentials
			return v, v.clientIDInput.Focus()
		}
		return v, nil

	case messages.OAuthFlowCompleted:
		v.waitingForAuth = false
		// Stop callback server if running
		if v.callbackServer != nil {
			_ = v.callbackServer.Stop() //nolint:errcheck // best-effort cleanup
			v.callbackServer = nil
		}
		if msg.Err != nil {
			v.err = msg.Err
			v.step = StepEnterCredentials
		} else {
			// OAuth completed, create the source with the new authorization
			return v, v.createSourceWithNewAuthorization()
		}
		return v, nil

	case oauthFlowStarted:
		// Set view state
		v.selectedAuthProviderID = msg.authProviderID
		v.oauthState = msg.flowState
		v.waitingForAuth = true
		v.step = StepOAuthFlow

		// Start callback server
		v.callbackServer = oauth.NewCallbackServer(msg.flowState.RedirectPort, msg.flowState.State)
		if err := v.callbackServer.Start(); err != nil {
			v.err = fmt.Errorf("failed to start callback server: %w", err)
			v.step = StepEnterCredentials
			return v, nil
		}

		// Open browser (failure is ok, URL is shown in UI)
		_ = oauth.OpenBrowser(msg.flowState.AuthURL) //nolint:errcheck // URL shown in UI

		// Return a command that waits for the callback
		return v, v.waitForOAuthCallback(msg.authProviderID, msg.flowState)
	}

	return v, nil
}

// handleKeyMsg handles key presses based on current step.
//
//nolint:gocyclo // central key handler requires complexity for wizard navigation
func (v *View) handleKeyMsg(msg tea.KeyMsg) (*View, tea.Cmd) {
	if msg.String() == "esc" { //nolint:nestif // escape handling requires nested conditionals for step navigation
		// Go back one step or exit
		switch v.step {
		case StepSelectConnector:
			return v, func() tea.Msg {
				return messages.ViewChanged{View: messages.ViewSources}
			}
		case StepEnterConfig:
			v.step = StepSelectConnector
			return v, nil
		case StepSelectAuthMethod:
			v.step = StepEnterConfig
			return v, nil
		case StepSelectAuth:
			// Go back to auth method if we came from there, otherwise config
			if v.connector != nil && v.connector.AuthCapability.SupportsMultipleMethods() {
				v.step = StepSelectAuthMethod
			} else {
				v.step = StepEnterConfig
			}
			return v, nil
		case StepEnterCredentials:
			// Go back to auth selection if we came from there
			if v.creatingNewAuth {
				v.step = StepSelectAuth
			} else if v.connector != nil && v.connector.AuthCapability.SupportsMultipleMethods() {
				v.step = StepSelectAuthMethod
			} else {
				v.step = StepEnterConfig
			}
			return v, nil
		case StepOAuthFlow:
			// Cancel OAuth flow
			v.waitingForAuth = false
			v.step = StepEnterCredentials
			return v, nil
		case StepComplete:
			return v, func() tea.Msg {
				return messages.ViewChanged{View: messages.ViewSources}
			}
		}
	}

	switch v.step {
	case StepSelectConnector:
		return v.handleConnectorSelect(msg)
	case StepEnterConfig:
		return v.handleConfigInput(msg)
	case StepSelectAuthMethod:
		return v.handleAuthMethodSelect(msg)
	case StepSelectAuth:
		return v.handleAuthSelect(msg)
	case StepEnterCredentials:
		return v.handleCredentialsInput(msg)
	case StepOAuthFlow:
		// Waiting for OAuth callback - no key handling needed
		return v, nil
	case StepComplete:
		if msg.String() == keyEnter {
			return v, func() tea.Msg {
				return messages.ViewChanged{View: messages.ViewSources}
			}
		}
	}

	return v, nil
}

func (v *View) handleConnectorSelect(msg tea.KeyMsg) (*View, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if v.selected > 0 {
			v.selected--
		}
	case keyDown, "j":
		if v.selected < len(v.connectors)-1 {
			v.selected++
		}
	case keyEnter:
		if len(v.connectors) > 0 && v.selected < len(v.connectors) {
			v.connector = &v.connectors[v.selected]
			cmd := v.initConfigInputs()
			v.step = StepEnterConfig
			return v, cmd
		}
	}
	return v, nil
}

func (v *View) initConfigInputs() tea.Cmd {
	if v.connector == nil {
		return nil
	}

	v.configInputs = make([]textinput.Model, len(v.connector.ConfigKeys))
	v.configKeys = make([]string, len(v.connector.ConfigKeys))

	for i, key := range v.connector.ConfigKeys {
		ti := textinput.New()
		// Build placeholder with default value if available
		placeholder := key.Description
		if key.Default != "" {
			if placeholder != "" {
				placeholder = fmt.Sprintf("%s (default: %s)", placeholder, key.Default)
			} else {
				placeholder = fmt.Sprintf("default: %s", key.Default)
			}
		}
		ti.Placeholder = placeholder
		if key.Secret {
			ti.EchoMode = textinput.EchoPassword
		}
		// Explicitly set empty value to ensure input starts empty
		ti.SetValue("")
		v.configInputs[i] = ti
		v.configKeys[i] = key.Key
	}
	v.focusIndex = 0

	// Return focus command for first input if any
	if len(v.configInputs) > 0 {
		return v.configInputs[0].Focus()
	}
	return nil
}

//nolint:gocritic // evalOrder: bubbletea pattern returns cmd from method call
func (v *View) handleConfigInput(msg tea.KeyMsg) (*View, tea.Cmd) {
	switch msg.String() {
	case "tab", keyDown:
		v.focusIndex++
		if v.focusIndex >= len(v.configInputs) {
			v.focusIndex = 0
		}
		return v, v.updateFocus()
	case "shift+tab", "up":
		v.focusIndex--
		if v.focusIndex < 0 {
			v.focusIndex = len(v.configInputs) - 1
		}
		return v, v.updateFocus()
	case keyEnter:
		// Validate required fields
		if v.validateConfig() {
			return v, v.determineNextStepAfterConfig()
		}
		return v, nil
	default:
		// Forward to current input
		if len(v.configInputs) > 0 && v.focusIndex < len(v.configInputs) {
			var cmd tea.Cmd
			v.configInputs[v.focusIndex], cmd = v.configInputs[v.focusIndex].Update(msg)
			return v, cmd
		}
	}
	return v, nil
}

// determineNextStepAfterConfig determines the next wizard step after config based on auth requirements.
func (v *View) determineNextStepAfterConfig() tea.Cmd {
	if v.connector == nil {
		return nil
	}

	// No auth needed - create source directly
	if !v.connector.AuthCapability.RequiresAuth() {
		return v.createSource()
	}

	// If connector supports multiple auth methods (PAT + OAuth), show auth method selection
	if v.connector.AuthCapability.SupportsMultipleMethods() {
		v.authMethodOptions = v.connector.AuthCapability.SupportedMethods()
		v.selectedAuthMethodIndex = 0
		v.step = StepSelectAuthMethod
		return nil
	}

	// Single auth method - proceed based on what's supported
	if v.connector.AuthCapability.SupportsPAT() {
		v.chosenAuthMethod = domain.AuthMethodPAT
		v.creatingNewAuth = false
		v.initCredentialInputs()
		v.step = StepEnterCredentials
		return v.tokenInput.Focus()
	}

	// OAuth only - check if provider has multiple connectors and existing auths
	v.chosenAuthMethod = domain.AuthMethodOAuth
	if v.providerRegistry != nil && v.providerRegistry.HasMultipleConnectors(v.connector.ProviderType) {
		// Load existing authorizations to see if user can reuse one
		v.step = StepSelectAuth
		return v.loadAuthorizations()
	}

	// Single-connector OAuth provider - go straight to credentials
	v.creatingNewAuth = false
	v.initCredentialInputs()
	v.step = StepEnterCredentials
	return v.clientIDInput.Focus()
}

// initCredentialInputs prepares the credential inputs for the current auth method.
// Note: This only clears values. The caller is responsible for returning the Focus() command.
func (v *View) initCredentialInputs() {
	v.credentialFocus = 0
	v.clientIDInput.SetValue("")
	v.clientSecretInput.SetValue("")
	v.tokenInput.SetValue("")
	// Focus() is NOT called here - callers return the appropriate Focus() command
}

func (v *View) updateFocus() tea.Cmd {
	cmds := make([]tea.Cmd, len(v.configInputs))
	for i := range v.configInputs {
		if i == v.focusIndex {
			cmds[i] = v.configInputs[i].Focus()
		} else {
			v.configInputs[i].Blur()
		}
	}
	return tea.Batch(cmds...)
}

func (v *View) validateConfig() bool {
	if v.connector == nil {
		return false
	}

	for i, key := range v.connector.ConfigKeys {
		if key.Required && strings.TrimSpace(v.configInputs[i].Value()) == "" {
			v.err = fmt.Errorf("required field %s is empty", key.Label)
			return false
		}
	}
	v.err = nil
	return true
}

// authProvidersLoaded is a message indicating auth providers have been loaded.
type authProvidersLoaded struct {
	authProviders []domain.AuthProvider
}

// loadAuthorizations returns a command that loads compatible auth providers.
func (v *View) loadAuthorizations() tea.Cmd {
	return func() tea.Msg {
		if v.authProviderService == nil || v.connector == nil {
			return messages.ErrorOccurred{Err: fmt.Errorf("auth provider service not available")}
		}
		providers, err := v.authProviderService.ListByProvider(context.Background(), v.connector.ProviderType)
		if err != nil {
			return messages.ErrorOccurred{Err: fmt.Errorf("failed to load auth providers: %w", err)}
		}
		return authProvidersLoaded{authProviders: providers}
	}
}

// handleAuthMethodSelect handles user selection in the auth method selection step.
// This step is shown when a connector supports both PAT and OAuth.
//
//nolint:gocritic // evalOrder: bubbletea pattern returns cmd from method call
func (v *View) handleAuthMethodSelect(msg tea.KeyMsg) (*View, tea.Cmd) {
	maxIndex := len(v.authMethodOptions) - 1

	switch msg.String() {
	case "up", "k":
		if v.selectedAuthMethodIndex > 0 {
			v.selectedAuthMethodIndex--
		}
	case keyDown, "j":
		if v.selectedAuthMethodIndex < maxIndex {
			v.selectedAuthMethodIndex++
		}
	case keyEnter:
		if v.selectedAuthMethodIndex >= 0 && v.selectedAuthMethodIndex < len(v.authMethodOptions) {
			v.chosenAuthMethod = v.authMethodOptions[v.selectedAuthMethodIndex]

			// Proceed based on the chosen method
			if v.chosenAuthMethod == domain.AuthMethodPAT {
				v.creatingNewAuth = false
				v.initCredentialInputs()
				v.step = StepEnterCredentials
				return v, v.tokenInput.Focus()
			}

			// OAuth - check if provider has multiple connectors and existing auths
			if v.providerRegistry != nil && v.providerRegistry.HasMultipleConnectors(v.connector.ProviderType) {
				return v, v.loadAuthorizations()
			}

			// Single-connector OAuth provider - go straight to credentials
			v.creatingNewAuth = false
			v.initCredentialInputs()
			v.step = StepEnterCredentials
			return v, v.clientIDInput.Focus()
		}
	}
	return v, nil
}

// handleAuthSelect handles user selection in the auth selection step.
// This step shows existing auth providers plus "Create new OAuth app" option.
//
//nolint:gocritic // evalOrder: bubbletea pattern returns cmd from method call
func (v *View) handleAuthSelect(msg tea.KeyMsg) (*View, tea.Cmd) {
	// Options: existing auth providers + "Create new OAuth app" at the end
	maxIndex := len(v.authProviders) // last index is "create new"

	switch msg.String() {
	case "up", "k":
		if v.selectedAuthIndex > 0 {
			v.selectedAuthIndex--
		}
	case keyDown, "j":
		if v.selectedAuthIndex < maxIndex {
			v.selectedAuthIndex++
		}
	case "n", "a":
		// Shortcut to add new OAuth app
		v.creatingNewAuth = true
		v.initCredentialInputs()
		v.step = StepEnterCredentials
		return v, v.clientIDInput.Focus()
	case keyEnter:
		if v.selectedAuthIndex == len(v.authProviders) {
			// "Create new OAuth app" selected
			v.creatingNewAuth = true
			v.initCredentialInputs()
			v.step = StepEnterCredentials
			return v, v.clientIDInput.Focus()
		}
		// Use existing auth provider - start OAuth flow with its credentials
		if v.selectedAuthIndex >= 0 && v.selectedAuthIndex < len(v.authProviders) {
			return v, v.startOAuthWithExistingProvider()
		}
		v.err = fmt.Errorf("no OAuth app selected")
	}
	return v, nil
}

// handleCredentialsInput handles inline credential entry (OAuth Client ID/Secret or PAT).
//
//nolint:gocritic,gocognit // evalOrder: bubbletea pattern; credential handling requires complexity
func (v *View) handleCredentialsInput(msg tea.KeyMsg) (*View, tea.Cmd) {
	if v.connector == nil {
		return v, nil
	}

	isPAT := v.chosenAuthMethod == domain.AuthMethodPAT

	switch msg.String() {
	case "tab", keyDown:
		if isPAT {
			// PAT only has one field
			return v, nil
		}
		v.credentialFocus++
		if v.credentialFocus > 1 {
			v.credentialFocus = 0
		}
		return v, v.updateCredentialFocus()
	case "shift+tab", "up":
		if isPAT {
			return v, nil
		}
		v.credentialFocus--
		if v.credentialFocus < 0 {
			v.credentialFocus = 1
		}
		return v, v.updateCredentialFocus()
	case keyEnter:
		if isPAT {
			// Validate token
			if strings.TrimSpace(v.tokenInput.Value()) == "" {
				v.err = fmt.Errorf("personal access token is required")
				return v, nil
			}
			// Create authorization with PAT and then create source
			return v, v.createAuthorizationAndSource()
		}
		// OAuth - validate credentials
		if strings.TrimSpace(v.clientIDInput.Value()) == "" {
			v.err = fmt.Errorf("client ID is required")
			return v, nil
		}
		if strings.TrimSpace(v.clientSecretInput.Value()) == "" {
			v.err = fmt.Errorf("client secret is required")
			return v, nil
		}
		v.err = nil
		// Create authorization and start OAuth flow
		return v, v.createAuthorizationAndStartOAuth()
	default:
		// Forward to appropriate input
		var cmd tea.Cmd
		if isPAT {
			v.tokenInput, cmd = v.tokenInput.Update(msg)
		} else if v.credentialFocus == 0 {
			v.clientIDInput, cmd = v.clientIDInput.Update(msg)
		} else {
			v.clientSecretInput, cmd = v.clientSecretInput.Update(msg)
		}
		return v, cmd
	}
}

func (v *View) updateCredentialFocus() tea.Cmd {
	if v.credentialFocus == 0 {
		v.clientSecretInput.Blur()
		return v.clientIDInput.Focus()
	}
	v.clientIDInput.Blur()
	return v.clientSecretInput.Focus()
}

// createAuthorizationAndSource creates a source with PAT credentials (new system).
func (v *View) createAuthorizationAndSource() tea.Cmd {
	return func() tea.Msg {
		if v.sourceService == nil || v.credentialsService == nil || v.connector == nil {
			return messages.SourceAdded{Err: fmt.Errorf("service not available")}
		}

		ctx := context.Background()

		// Build source config
		config := make(map[string]string)
		for i, key := range v.configKeys {
			config[key] = v.configInputs[i].Value()
		}

		name := v.connector.Name
		if val, ok := config["path"]; ok && val != "" {
			name = val
		} else if val, ok := config["owner"]; ok {
			if repo, ok := config["repo"]; ok {
				name = val + "/" + repo
			}
		}

		// Create source first (credentials have FK to source)
		sourceID := uuid.New().String()
		source := domain.Source{
			ID:     sourceID,
			Type:   v.connector.ID,
			Name:   name,
			Config: config,
			// No AuthProviderID for PAT auth
		}

		if err := v.sourceService.Add(ctx, source); err != nil {
			return messages.SourceAdded{Err: fmt.Errorf("failed to add source: %w", err)}
		}

		// Create credentials with PAT
		now := time.Now()
		creds := domain.Credentials{
			ID:       uuid.New().String(),
			SourceID: sourceID,
			PAT: &domain.PATCredentials{
				Token: strings.TrimSpace(v.tokenInput.Value()),
			},
			CreatedAt: now,
			UpdatedAt: now,
		}

		if err := v.credentialsService.Save(ctx, creds); err != nil {
			// Rollback source creation - ignore error as this is best-effort cleanup
			//nolint:errcheck // Best-effort cleanup on failure
			v.sourceService.Remove(ctx, sourceID)
			return messages.SourceAdded{Err: fmt.Errorf("failed to save credentials: %w", err)}
		}

		// Update source with credentials_id
		source.CredentialsID = creds.ID
		//nolint:errcheck // Best effort - source exists but credentials_id not linked
		v.sourceService.Update(ctx, source)

		return messages.SourceAdded{Source: source, Err: nil}
	}
}

// createAuthorizationAndStartOAuth creates an AuthProvider and starts the OAuth flow (new system).
func (v *View) createAuthorizationAndStartOAuth() tea.Cmd {
	return func() tea.Msg {
		if v.authProviderService == nil || v.connector == nil {
			return messages.ErrorOccurred{Err: fmt.Errorf("service not available")}
		}

		ctx := context.Background()

		// Get OAuth endpoints from provider registry
		var authURL, tokenURL string
		var scopes []string
		if v.providerRegistry != nil {
			if endpoints := v.providerRegistry.GetOAuthEndpoints(v.connector.ProviderType); endpoints != nil {
				authURL = endpoints.AuthURL
				tokenURL = endpoints.TokenURL
				scopes = endpoints.Scopes
			}
		}

		// Create AuthProvider with OAuth config
		now := time.Now()
		authProviderID := uuid.New().String()
		authProvider := domain.AuthProvider{
			ID:           authProviderID,
			Name:         fmt.Sprintf("%s OAuth App", v.connector.Name),
			ProviderType: v.connector.ProviderType,
			OAuth: &domain.OAuthProviderConfig{
				ClientID:     strings.TrimSpace(v.clientIDInput.Value()),
				ClientSecret: strings.TrimSpace(v.clientSecretInput.Value()),
				AuthURL:      authURL,
				TokenURL:     tokenURL,
				Scopes:       scopes,
			},
			CreatedAt: now,
			UpdatedAt: now,
		}

		if err := v.authProviderService.Save(ctx, authProvider); err != nil {
			return messages.ErrorOccurred{Err: fmt.Errorf("failed to create auth provider: %w", err)}
		}

		// Build OAuth flow state using the connector registry
		flowState, err := v.buildOAuthFlowState(&authProvider)
		if err != nil {
			return messages.ErrorOccurred{Err: err}
		}

		// Return message with all needed info - state changes happen in Update()
		return oauthFlowStarted{authProviderID: authProviderID, flowState: flowState}
	}
}

// buildOAuthFlowState constructs the OAuth flow state using the connector registry.
// This method delegates provider-specific URL construction to the connector factory.
func (v *View) buildOAuthFlowState(authProvider *domain.AuthProvider) (*driving.OAuthFlowState, error) {
	if v.connectorRegistry == nil || v.connector == nil {
		return nil, fmt.Errorf("connector registry not available")
	}

	// Generate PKCE verifier and challenge
	codeVerifier := oauth.GenerateCodeVerifier()
	codeChallenge := oauth.GenerateCodeChallenge(codeVerifier)

	// Generate state for CSRF protection
	state := oauth.GenerateCodeVerifier() // Reuse verifier generation for state

	// Use fixed port 18080 for callback
	const redirectPort = 18080
	redirectURI := fmt.Sprintf("http://localhost:%d/callback", redirectPort)

	// Build the authorization URL using connector registry
	// This handles provider-specific parameters (e.g., access_type=offline for Google)
	authURL, err := v.connectorRegistry.BuildAuthURL(
		v.connector.ID,
		authProvider,
		redirectURI,
		state,
		codeChallenge,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build auth URL: %w", err)
	}

	return &driving.OAuthFlowState{
		AuthURL:      authURL,
		CodeVerifier: codeVerifier,
		State:        state,
		RedirectURI:  redirectURI,
		RedirectPort: redirectPort,
	}, nil
}

// oauthFlowStarted indicates the OAuth flow has been initiated.
type oauthFlowStarted struct {
	authProviderID string
	flowState      *driving.OAuthFlowState
}

// waitForOAuthCallback returns a command that waits for the OAuth callback.
// It exchanges the code for tokens and fetches the account identifier.
func (v *View) waitForOAuthCallback(authProviderID string, flowState *driving.OAuthFlowState) tea.Cmd {
	return func() tea.Msg {
		if v.callbackServer == nil {
			return messages.OAuthFlowCompleted{Err: fmt.Errorf("callback server not running")}
		}

		ctx := context.Background()

		// Wait for callback (5 minute timeout)
		code, err := v.callbackServer.WaitForCode(5 * time.Minute)
		if err != nil {
			return messages.OAuthFlowCompleted{Err: fmt.Errorf("authorization failed: %w", err)}
		}

		// Get the AuthProvider to exchange tokens
		if v.authProviderService == nil {
			return messages.OAuthFlowCompleted{Err: fmt.Errorf("auth provider service not available")}
		}

		authProvider, err := v.authProviderService.Get(ctx, authProviderID)
		if err != nil {
			return messages.OAuthFlowCompleted{Err: fmt.Errorf("failed to get auth provider: %w", err)}
		}

		if authProvider.OAuth == nil {
			return messages.OAuthFlowCompleted{Err: fmt.Errorf("auth provider has no OAuth configuration")}
		}

		oauthCfg := authProvider.OAuth
		redirectURI := v.callbackServer.RedirectURI()

		// Exchange code for tokens
		tokens, err := drivenoauth.ExchangeCodeForTokens(
			ctx,
			oauthCfg.TokenURL,
			oauthCfg.ClientID,
			oauthCfg.ClientSecret,
			code,
			redirectURI,
			flowState.CodeVerifier,
		)
		if err != nil {
			return messages.OAuthFlowCompleted{Err: fmt.Errorf("failed to exchange code for tokens: %w", err)}
		}

		// Store tokens in view state for later credential creation
		v.pendingOAuthTokens = &domain.OAuthCredentials{
			AccessToken:  tokens.AccessToken,
			RefreshToken: tokens.RefreshToken,
			TokenType:    tokens.TokenType,
			Expiry:       tokens.Expiry,
		}

		// Fetch account identifier using connector registry
		if v.connector != nil && v.connectorRegistry != nil {
			accountID, err := v.connectorRegistry.GetUserInfo(ctx, v.connector.ID, tokens.AccessToken)
			if err == nil && accountID != "" {
				v.accountIdentifier = accountID
			}
		}

		return messages.OAuthFlowCompleted{Err: nil}
	}
}

// createSourceWithNewAuthorization creates the source with AuthProviderID and Credentials (new system).
func (v *View) createSourceWithNewAuthorization() tea.Cmd {
	return func() tea.Msg {
		if v.sourceService == nil || v.credentialsService == nil ||
			v.connector == nil || v.selectedAuthProviderID == "" {
			return messages.SourceAdded{Err: fmt.Errorf("service not available")}
		}

		if v.pendingOAuthTokens == nil {
			return messages.SourceAdded{Err: fmt.Errorf("no OAuth tokens available")}
		}

		ctx := context.Background()

		config := make(map[string]string)
		for i, key := range v.configKeys {
			config[key] = v.configInputs[i].Value()
		}

		name := v.connector.Name
		if val, ok := config["path"]; ok && val != "" {
			name = val
		} else if val, ok := config["owner"]; ok {
			if repo, ok := config["repo"]; ok {
				name = val + "/" + repo
			}
		}

		// Append account identifier for OAuth sources (like CLI does)
		if v.accountIdentifier != "" {
			name = fmt.Sprintf("%s (%s)", name, v.accountIdentifier)
		}

		// Create source first (credentials have FK to source)
		sourceID := uuid.New().String()
		source := domain.Source{
			ID:             sourceID,
			Type:           v.connector.ID,
			Name:           name,
			Config:         config,
			AuthProviderID: v.selectedAuthProviderID,
		}

		if err := v.sourceService.Add(ctx, source); err != nil {
			return messages.SourceAdded{Err: fmt.Errorf("failed to add source: %w", err)}
		}

		// Create credentials with OAuth tokens and account identifier
		now := time.Now()
		creds := domain.Credentials{
			ID:                uuid.New().String(),
			SourceID:          sourceID,
			AccountIdentifier: v.accountIdentifier,
			OAuth:             v.pendingOAuthTokens,
			CreatedAt:         now,
			UpdatedAt:         now,
		}

		if err := v.credentialsService.Save(ctx, creds); err != nil {
			// Rollback source creation - ignore error as this is best-effort cleanup
			//nolint:errcheck // Best-effort cleanup on failure
			v.sourceService.Remove(ctx, sourceID)
			return messages.SourceAdded{Err: fmt.Errorf("failed to save credentials: %w", err)}
		}

		// Update source with credentials_id
		source.CredentialsID = creds.ID
		//nolint:errcheck // Best effort - source exists but credentials_id not linked
		v.sourceService.Update(ctx, source)

		return messages.SourceAdded{Source: source, Err: nil}
	}
}

func (v *View) createSource() tea.Cmd {
	return func() tea.Msg {
		if v.sourceService == nil || v.connector == nil {
			return messages.SourceAdded{Err: fmt.Errorf("service not available")}
		}

		config := make(map[string]string)
		for i, key := range v.configKeys {
			config[key] = v.configInputs[i].Value()
		}

		name := v.connector.Name
		if val, ok := config["path"]; ok && val != "" {
			name = val
		} else if val, ok := config["owner"]; ok {
			if repo, ok := config["repo"]; ok {
				name = val + "/" + repo
			}
		}

		source := domain.Source{
			ID:     uuid.New().String(),
			Type:   v.connector.ID,
			Name:   name,
			Config: config,
		}

		err := v.sourceService.Add(context.Background(), source)
		return messages.SourceAdded{Source: source, Err: err}
	}
}

// startOAuthWithExistingProvider starts the OAuth flow using an existing AuthProvider (new system).
func (v *View) startOAuthWithExistingProvider() tea.Cmd {
	return func() tea.Msg {
		if v.connector == nil {
			return messages.ErrorOccurred{Err: fmt.Errorf("service not available")}
		}

		// Get the selected auth provider
		if len(v.authProviders) == 0 || v.selectedAuthIndex < 0 || v.selectedAuthIndex >= len(v.authProviders) {
			return messages.ErrorOccurred{Err: fmt.Errorf("no OAuth app selected")}
		}

		selectedProvider := v.authProviders[v.selectedAuthIndex]

		if selectedProvider.OAuth == nil {
			return messages.ErrorOccurred{Err: fmt.Errorf("selected provider has no OAuth configuration")}
		}

		// Build OAuth flow state using the connector registry
		flowState, err := v.buildOAuthFlowState(&selectedProvider)
		if err != nil {
			return messages.ErrorOccurred{Err: err}
		}

		// Return message with all needed info - state changes happen in Update()
		return oauthFlowStarted{authProviderID: selectedProvider.ID, flowState: flowState}
	}
}

// View renders the add source wizard.
func (v *View) View() string {
	var b strings.Builder

	b.WriteString(v.styles.Title.Render("Add Source"))
	b.WriteString("\n\n")

	// Progress indicator - simplified to show current step
	b.WriteString(v.renderProgress())
	b.WriteString("\n\n")

	// Error display
	if v.err != nil {
		b.WriteString(v.styles.Error.Render(fmt.Sprintf("Error: %s", v.err.Error())))
		b.WriteString("\n\n")
	}

	// Step content
	switch v.step {
	case StepSelectConnector:
		b.WriteString(v.renderConnectorSelect())
	case StepEnterConfig:
		b.WriteString(v.renderConfigInput())
	case StepSelectAuthMethod:
		b.WriteString(v.renderAuthMethodSelect())
	case StepSelectAuth:
		b.WriteString(v.renderAuthSelect())
	case StepEnterCredentials:
		b.WriteString(v.renderCredentialsInput())
	case StepOAuthFlow:
		b.WriteString(v.renderOAuthFlow())
	case StepComplete:
		b.WriteString(v.renderComplete())
	}

	b.WriteString("\n")
	b.WriteString(v.renderHelp())

	return b.String()
}

func (v *View) renderProgress() string {
	// Map current step to display step index
	stepNames := []string{"Connector", "Configure", "Authenticate", "Done"}
	currentIdx := 0
	switch v.step {
	case StepSelectConnector:
		currentIdx = 0
	case StepEnterConfig:
		currentIdx = 1
	case StepSelectAuthMethod, StepSelectAuth, StepEnterCredentials, StepOAuthFlow:
		currentIdx = 2
	case StepComplete:
		currentIdx = 3
	}

	progress := ""
	for i, name := range stepNames {
		if i > 0 {
			progress += " > "
		}
		if i == currentIdx {
			progress += v.styles.Selected.Render(name)
		} else if i < currentIdx {
			progress += v.styles.Success.Render(name)
		} else {
			progress += v.styles.Muted.Render(name)
		}
	}
	return progress
}

func (v *View) renderConnectorSelect() string {
	var b strings.Builder

	b.WriteString(v.styles.Subtitle.Render("Select a connector type:"))
	b.WriteString("\n\n")

	if len(v.connectors) == 0 {
		b.WriteString(v.styles.Muted.Render("No connectors available."))
		return b.String()
	}

	for i, c := range v.connectors {
		indicator := "  "
		if i == v.selected {
			indicator = "> "
		}

		// Show auth capability badge
		authBadge := ""
		if !c.AuthCapability.RequiresAuth() {
			authBadge = "[no auth]"
		} else if c.AuthCapability.SupportsMultipleMethods() {
			authBadge = "[token/oauth]"
		} else if c.AuthCapability.SupportsPAT() {
			authBadge = "[token]"
		} else if c.AuthCapability.SupportsOAuth() {
			authBadge = "[oauth]"
		}

		line := fmt.Sprintf("%s%s - %s %s", indicator, c.Name, c.Description, authBadge)
		if i == v.selected {
			b.WriteString(v.styles.Selected.Render(line))
		} else {
			b.WriteString(v.styles.Normal.Render(line))
		}
		b.WriteString("\n")
	}

	return b.String()
}

func (v *View) renderConfigInput() string {
	var b strings.Builder

	if v.connector == nil {
		return ""
	}

	b.WriteString(v.styles.Subtitle.Render(fmt.Sprintf("Configure %s:", v.connector.Name)))
	b.WriteString("\n\n")

	if len(v.connector.ConfigKeys) == 0 {
		b.WriteString(v.styles.Muted.Render("No configuration needed."))
		b.WriteString("\n")
		b.WriteString(v.styles.Help.Render("Press enter to continue."))
		return b.String()
	}

	for i, key := range v.connector.ConfigKeys {
		label := key.Label
		if key.Required {
			label += " *"
		}
		b.WriteString(v.styles.Normal.Render(label + ":"))
		b.WriteString("\n")
		b.WriteString(v.configInputs[i].View())
		b.WriteString("\n\n")
	}

	return b.String()
}

func (v *View) renderAuthMethodSelect() string {
	var b strings.Builder

	if v.connector == nil {
		return ""
	}

	b.WriteString(v.styles.Subtitle.Render(fmt.Sprintf("Select authentication method for %s:", v.connector.Name)))
	b.WriteString("\n\n")

	b.WriteString(v.styles.Muted.Render("This connector supports multiple authentication methods."))
	b.WriteString("\n\n")

	for i, method := range v.authMethodOptions {
		indicator := "  "
		if i == v.selectedAuthMethodIndex {
			indicator = "> "
		}

		// Format method name and description
		var methodName, methodDesc string
		switch method {
		case domain.AuthMethodPAT:
			methodName = "Personal Access Token"
			methodDesc = "Use a token from your account settings"
		case domain.AuthMethodOAuth:
			methodName = "OAuth App"
			methodDesc = "Authenticate via browser with OAuth"
		case domain.AuthMethodNone:
			methodName = "None"
			methodDesc = "No authentication required"
		}

		line := fmt.Sprintf("%s%s - %s", indicator, methodName, methodDesc)
		if i == v.selectedAuthMethodIndex {
			b.WriteString(v.styles.Selected.Render(line))
		} else {
			b.WriteString(v.styles.Normal.Render(line))
		}
		b.WriteString("\n")
	}

	return b.String()
}

func (v *View) renderAuthSelect() string {
	var b strings.Builder

	if v.connector == nil {
		return ""
	}

	b.WriteString(v.styles.Subtitle.Render("Available OAuth app configurations:"))
	b.WriteString("\n\n")

	b.WriteString(v.styles.Muted.Render("Select an existing OAuth app or create a new one."))
	b.WriteString("\n\n")

	// Show existing auth providers
	for i := range v.authProviders {
		provider := &v.authProviders[i]
		indicator := "  "
		if i == v.selectedAuthIndex {
			indicator = "> "
		}

		// Show short ID for identification
		shortID := provider.ID
		if len(shortID) > 8 {
			shortID = shortID[:8]
		}

		line := fmt.Sprintf("%s%d. %s (%s)", indicator, i+1, provider.Name, shortID)
		if i == v.selectedAuthIndex {
			b.WriteString(v.styles.Selected.Render(line))
		} else {
			b.WriteString(v.styles.Normal.Render(line))
		}
		b.WriteString("\n")
	}

	// "Create new OAuth app configuration" option at the end
	indicator := "  "
	if v.selectedAuthIndex == len(v.authProviders) {
		indicator = "> "
	}
	line := fmt.Sprintf("%s%d. Create new OAuth app configuration", indicator, len(v.authProviders)+1)
	if v.selectedAuthIndex == len(v.authProviders) {
		b.WriteString(v.styles.Selected.Render(line))
	} else {
		b.WriteString(v.styles.Normal.Render(line))
	}
	b.WriteString("\n")

	return b.String()
}

func (v *View) renderCredentialsInput() string {
	var b strings.Builder

	if v.connector == nil {
		return ""
	}

	isPAT := v.chosenAuthMethod == domain.AuthMethodPAT

	if isPAT {
		b.WriteString(v.styles.Subtitle.Render(fmt.Sprintf("Enter Personal Access Token for %s:", v.connector.Name)))
		b.WriteString("\n\n")
		b.WriteString(v.styles.Muted.Render("Create a token in your account settings."))
		b.WriteString("\n\n")
		b.WriteString(v.styles.Normal.Render("Token:"))
		b.WriteString("\n")
		b.WriteString(v.tokenInput.View())
		b.WriteString("\n")
	} else {
		b.WriteString(v.styles.Subtitle.Render(fmt.Sprintf("Enter OAuth App Credentials for %s:", v.connector.Name)))
		b.WriteString("\n\n")
		b.WriteString(v.styles.Muted.Render(v.getProviderHint()))
		b.WriteString("\n\n")
		b.WriteString(v.styles.Normal.Render("Client ID:"))
		b.WriteString("\n")
		b.WriteString(v.clientIDInput.View())
		b.WriteString("\n\n")
		b.WriteString(v.styles.Normal.Render("Client Secret:"))
		b.WriteString("\n")
		b.WriteString(v.clientSecretInput.View())
		b.WriteString("\n")
	}

	return b.String()
}

func (v *View) getProviderHint() string {
	if v.connector == nil || v.connectorRegistry == nil {
		return ""
	}
	// Get hint from connector registry - returns empty for local providers
	return v.connectorRegistry.GetSetupHint(v.connector.ID)
}

func (v *View) renderOAuthFlow() string {
	var b strings.Builder

	b.WriteString(v.styles.Subtitle.Render("Authenticating..."))
	b.WriteString("\n\n")

	if v.waitingForAuth {
		b.WriteString(v.styles.Normal.Render("A browser window should have opened for authentication."))
		b.WriteString("\n\n")
		b.WriteString(v.styles.Muted.Render("Complete the authorization in your browser."))
		b.WriteString("\n")
		b.WriteString(v.styles.Muted.Render("This window will update automatically when done."))
		b.WriteString("\n\n")
		if v.oauthState != nil {
			b.WriteString(v.styles.Help.Render(fmt.Sprintf("Auth URL: %s", v.oauthState.AuthURL)))
		}
	} else {
		b.WriteString(v.styles.Normal.Render("Starting authentication flow..."))
	}

	return b.String()
}

func (v *View) renderComplete() string {
	var b strings.Builder

	b.WriteString(v.styles.Subtitle.Render("Source Added Successfully!"))
	b.WriteString("\n\n")

	if v.source != nil {
		b.WriteString(fmt.Sprintf("ID: %s\n", v.source.ID))
		b.WriteString(fmt.Sprintf("Type: %s\n", v.source.Type))
		b.WriteString(fmt.Sprintf("Name: %s\n", v.source.Name))
	}

	return b.String()
}

func (v *View) renderHelp() string {
	switch v.step {
	case StepSelectConnector:
		return v.styles.Help.Render("[j/k] navigate  [enter] select  [esc] cancel")
	case StepEnterConfig:
		return v.styles.Help.Render("[tab] next field  [enter] continue  [esc] back")
	case StepSelectAuthMethod:
		return v.styles.Help.Render("[j/k] navigate  [enter] select  [esc] back")
	case StepSelectAuth:
		return v.styles.Help.Render("[j/k] navigate  [enter] select  [n] new app  [esc] back")
	case StepEnterCredentials:
		return v.styles.Help.Render("[tab] next field  [enter] continue  [esc] back")
	case StepOAuthFlow:
		return v.styles.Help.Render("[esc] cancel")
	case StepComplete:
		return v.styles.Help.Render("[enter] done  [esc] back to sources")
	default:
		return ""
	}
}

// SetDimensions sets the view dimensions.
func (v *View) SetDimensions(width, height int) {
	v.width = width
	v.height = height
	v.ready = true
}

// Reset resets the wizard to initial state.
func (v *View) Reset() {
	// Stop callback server if running
	if v.callbackServer != nil {
		_ = v.callbackServer.Stop() //nolint:errcheck // best-effort cleanup
		v.callbackServer = nil
	}
	v.step = StepSelectConnector
	v.selected = 0
	v.connector = nil
	v.configInputs = nil
	v.configKeys = nil
	v.focusIndex = 0
	v.authMethodOptions = nil
	v.selectedAuthMethodIndex = 0
	v.chosenAuthMethod = domain.AuthMethodNone
	v.authProviders = nil
	v.selectedAuthIndex = 0
	v.creatingNewAuth = false
	v.credentialFocus = 0
	v.clientIDInput.SetValue("")
	v.clientSecretInput.SetValue("")
	v.tokenInput.SetValue("")
	v.oauthState = nil
	v.waitingForAuth = false
	v.selectedAuthProviderID = ""
	v.pendingOAuthTokens = nil
	v.accountIdentifier = ""
	v.source = nil
	v.err = nil
}
