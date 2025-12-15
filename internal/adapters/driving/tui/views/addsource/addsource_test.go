package addsource

import (
	"context"
	"errors"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/custodia-labs/sercha-cli/internal/adapters/driving/tui/messages"
	"github.com/custodia-labs/sercha-cli/internal/adapters/driving/tui/styles"
	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driving"
)

// MockSourceService implements driving.SourceService for testing.
type MockSourceService struct {
	AddFunc    func(ctx context.Context, source domain.Source) error
	ListFunc   func(ctx context.Context) ([]domain.Source, error)
	RemoveFunc func(ctx context.Context, id string) error
}

func (m *MockSourceService) Add(ctx context.Context, source domain.Source) error {
	if m.AddFunc != nil {
		return m.AddFunc(ctx, source)
	}
	return nil
}

func (m *MockSourceService) Get(ctx context.Context, id string) (*domain.Source, error) {
	return nil, nil
}

func (m *MockSourceService) List(ctx context.Context) ([]domain.Source, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx)
	}
	return []domain.Source{}, nil
}

func (m *MockSourceService) Remove(ctx context.Context, id string) error {
	if m.RemoveFunc != nil {
		return m.RemoveFunc(ctx, id)
	}
	return nil
}

func (m *MockSourceService) Update(ctx context.Context, source domain.Source) error {
	return nil
}

func (m *MockSourceService) ValidateConfig(
	ctx context.Context,
	connectorType string,
	config map[string]string,
) error {
	return nil
}

// MockConnectorRegistry implements driving.ConnectorRegistry for testing.
type MockConnectorRegistry struct {
	ListFunc           func() []domain.ConnectorType
	GetFunc            func(id string) (*domain.ConnectorType, error)
	GetOAuthDefaultsFn func(connectorType string) *driving.OAuthDefaults
	SupportsOAuthFn    func(connectorType string) bool
	BuildAuthURLFn     func(connectorType string, authProvider *domain.AuthProvider, redirectURI, state, codeChallenge string) (string, error)
	GetUserInfoFn      func(ctx context.Context, connectorType string, accessToken string) (string, error)
	hints              map[string]string
}

func (m *MockConnectorRegistry) List() []domain.ConnectorType {
	if m.ListFunc != nil {
		return m.ListFunc()
	}
	return []domain.ConnectorType{
		{
			ID:             "filesystem",
			Name:           "Local Filesystem",
			ProviderType:   domain.ProviderLocal,
			AuthCapability: domain.AuthCapNone,
			AuthMethod:     domain.AuthMethodNone,
			ConfigKeys: []domain.ConfigKey{
				{Key: "path", Label: "Path", Required: true},
			},
		},
		{
			ID:             "github",
			Name:           "GitHub",
			ProviderType:   domain.ProviderGitHub,
			AuthCapability: domain.AuthCapPAT | domain.AuthCapOAuth, // GitHub supports both
			AuthMethod:     domain.AuthMethodPAT,                    // Default for backward compat
			ConfigKeys: []domain.ConfigKey{
				{Key: "content_types", Label: "Content Types", Required: false},
				{Key: "file_patterns", Label: "File Patterns", Required: false},
			},
		},
	}
}

func (m *MockConnectorRegistry) Get(id string) (*domain.ConnectorType, error) {
	if m.GetFunc != nil {
		return m.GetFunc(id)
	}
	connectors := m.List()
	for i := range connectors {
		if connectors[i].ID == id {
			return &connectors[i], nil
		}
	}
	return nil, domain.ErrNotFound
}

func (m *MockConnectorRegistry) ValidateConfig(
	connectorID string,
	config map[string]string,
) error {
	return nil
}

func (m *MockConnectorRegistry) GetOAuthDefaults(connectorType string) *driving.OAuthDefaults {
	if m.GetOAuthDefaultsFn != nil {
		return m.GetOAuthDefaultsFn(connectorType)
	}
	return &driving.OAuthDefaults{
		AuthURL:  "https://example.com/oauth/authorize",
		TokenURL: "https://example.com/oauth/token",
		Scopes:   []string{"read", "write"},
	}
}

func (m *MockConnectorRegistry) SupportsOAuth(connectorType string) bool {
	if m.SupportsOAuthFn != nil {
		return m.SupportsOAuthFn(connectorType)
	}
	return true
}

func (m *MockConnectorRegistry) BuildAuthURL(
	connectorType string,
	authProvider *domain.AuthProvider,
	redirectURI, state, codeChallenge string,
) (string, error) {
	if m.BuildAuthURLFn != nil {
		return m.BuildAuthURLFn(connectorType, authProvider, redirectURI, state, codeChallenge)
	}
	// Default implementation builds a simple URL
	return "https://example.com/oauth/authorize?client_id=" + authProvider.OAuth.ClientID + "&state=" + state, nil
}

func (m *MockConnectorRegistry) GetUserInfo(ctx context.Context, connectorType string, accessToken string) (string, error) {
	if m.GetUserInfoFn != nil {
		return m.GetUserInfoFn(ctx, connectorType, accessToken)
	}
	return "test@example.com", nil
}

func (m *MockConnectorRegistry) GetSetupHint(connectorType string) string {
	if m.hints != nil {
		return m.hints[connectorType]
	}
	return ""
}

func (m *MockConnectorRegistry) GetConnectorsForProvider(provider domain.ProviderType) []domain.ConnectorType {
	var result []domain.ConnectorType
	for _, c := range m.List() {
		if c.ProviderType == provider {
			result = append(result, c)
		}
	}
	return result
}

func (m *MockConnectorRegistry) ExchangeCode(_ context.Context, _ string, _ *domain.AuthProvider, _, _, _ string) (*domain.OAuthToken, error) {
	return nil, nil
}

// MockCredentialsService implements driving.CredentialsService for testing.
type MockCredentialsService struct {
	SaveFunc          func(ctx context.Context, creds domain.Credentials) error
	GetFunc           func(ctx context.Context, id string) (*domain.Credentials, error)
	GetBySourceIDFunc func(ctx context.Context, sourceID string) (*domain.Credentials, error)
	DeleteFunc        func(ctx context.Context, id string) error
}

func (m *MockCredentialsService) Save(ctx context.Context, creds domain.Credentials) error {
	if m.SaveFunc != nil {
		return m.SaveFunc(ctx, creds)
	}
	return nil
}

func (m *MockCredentialsService) Get(ctx context.Context, id string) (*domain.Credentials, error) {
	if m.GetFunc != nil {
		return m.GetFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockCredentialsService) GetBySourceID(ctx context.Context, sourceID string) (*domain.Credentials, error) {
	if m.GetBySourceIDFunc != nil {
		return m.GetBySourceIDFunc(ctx, sourceID)
	}
	return nil, nil
}

func (m *MockCredentialsService) Delete(ctx context.Context, id string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return nil
}

// MockAuthProviderService implements driving.AuthProviderService for testing.
type MockAuthProviderService struct {
	SaveFunc           func(ctx context.Context, provider domain.AuthProvider) error
	GetFunc            func(ctx context.Context, id string) (*domain.AuthProvider, error)
	ListFunc           func(ctx context.Context) ([]domain.AuthProvider, error)
	ListByProviderFunc func(ctx context.Context, provider domain.ProviderType) ([]domain.AuthProvider, error)
	DeleteFunc         func(ctx context.Context, id string) error
}

func (m *MockAuthProviderService) Save(ctx context.Context, provider domain.AuthProvider) error {
	if m.SaveFunc != nil {
		return m.SaveFunc(ctx, provider)
	}
	return nil
}

func (m *MockAuthProviderService) Get(ctx context.Context, id string) (*domain.AuthProvider, error) {
	if m.GetFunc != nil {
		return m.GetFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockAuthProviderService) List(ctx context.Context) ([]domain.AuthProvider, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx)
	}
	return []domain.AuthProvider{}, nil
}

func (m *MockAuthProviderService) ListByProvider(ctx context.Context, provider domain.ProviderType) ([]domain.AuthProvider, error) {
	if m.ListByProviderFunc != nil {
		return m.ListByProviderFunc(ctx, provider)
	}
	return []domain.AuthProvider{}, nil
}

func (m *MockAuthProviderService) Delete(ctx context.Context, id string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return nil
}

// MockProviderRegistry implements driving.ProviderRegistry for testing.
type MockProviderRegistry struct {
	HasMultipleConnectorsFunc func(provider domain.ProviderType) bool
}

func (m *MockProviderRegistry) GetProviders() []domain.ProviderType {
	return []domain.ProviderType{domain.ProviderGoogle, domain.ProviderGitHub}
}

func (m *MockProviderRegistry) GetConnectorsForProvider(provider domain.ProviderType) []string {
	return nil
}

func (m *MockProviderRegistry) GetProviderForConnector(connectorType string) (domain.ProviderType, error) {
	return domain.ProviderLocal, nil
}

func (m *MockProviderRegistry) IsCompatible(provider domain.ProviderType, connectorType string) bool {
	return true
}

func (m *MockProviderRegistry) GetDefaultAuthMethod(provider domain.ProviderType) domain.AuthMethod {
	return domain.AuthMethodNone
}

func (m *MockProviderRegistry) HasMultipleConnectors(provider domain.ProviderType) bool {
	if m.HasMultipleConnectorsFunc != nil {
		return m.HasMultipleConnectorsFunc(provider)
	}
	return false
}

func (m *MockProviderRegistry) GetOAuthEndpoints(provider domain.ProviderType) *driving.OAuthEndpoints {
	return &driving.OAuthEndpoints{
		AuthURL:   "https://example.com/oauth/authorize",
		TokenURL:  "https://example.com/oauth/token",
		DeviceURL: "",
		Scopes:    []string{"read", "write"},
	}
}

func (m *MockProviderRegistry) GetAuthCapability(provider domain.ProviderType) domain.AuthCapability {
	return 0
}

func (m *MockProviderRegistry) GetSupportedAuthMethods(provider domain.ProviderType) []domain.AuthMethod {
	return []domain.AuthMethod{domain.AuthMethodNone}
}

func (m *MockProviderRegistry) SupportsMultipleAuthMethods(provider domain.ProviderType) bool {
	return false
}

func TestNewView(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewView(s, nil, nil, nil, nil, nil)

	require.NotNil(t, view)
	assert.Equal(t, StepSelectConnector, view.step)
	assert.False(t, view.ready)
	assert.NotNil(t, view.clientIDInput)
	assert.NotNil(t, view.clientSecretInput)
	assert.NotNil(t, view.tokenInput)
}

func TestNewView_NilParams(t *testing.T) {
	view := NewView(nil, nil, nil, nil, nil, nil)

	require.NotNil(t, view)
	assert.Nil(t, view.styles)
}

func TestView_Init(t *testing.T) {
	registry := &MockConnectorRegistry{}
	view := NewView(nil, nil, registry, nil, nil, nil)

	cmd := view.Init()

	require.NotNil(t, cmd)
	msg := cmd()
	loaded, ok := msg.(connectorsLoaded)
	require.True(t, ok)
	assert.Len(t, loaded.connectors, 2)
}

func TestView_Init_NilRegistry(t *testing.T) {
	view := NewView(nil, nil, nil, nil, nil, nil)

	cmd := view.Init()

	require.NotNil(t, cmd)
	msg := cmd()
	errMsg, ok := msg.(messages.ErrorOccurred)
	require.True(t, ok)
	assert.Error(t, errMsg.Err)
}

func TestView_Update_WindowSize(t *testing.T) {
	view := NewView(nil, nil, nil, nil, nil, nil)

	msg := tea.WindowSizeMsg{Width: 80, Height: 24}
	updated, cmd := view.Update(msg)

	assert.Equal(t, view, updated)
	assert.Nil(t, cmd)
	assert.True(t, view.ready)
	assert.Equal(t, 80, view.width)
	assert.Equal(t, 24, view.height)
}

func TestView_Update_ConnectorsLoaded(t *testing.T) {
	view := NewView(nil, nil, nil, nil, nil, nil)

	connectors := []domain.ConnectorType{
		{ID: "filesystem", Name: "Filesystem"},
		{ID: "github", Name: "GitHub"},
	}
	msg := connectorsLoaded{connectors: connectors}
	view.Update(msg)

	assert.Len(t, view.connectors, 2)
}

func TestView_Update_KeyMsg_Escape_FromSelectConnector(t *testing.T) {
	view := NewView(nil, nil, nil, nil, nil, nil)
	view.step = StepSelectConnector

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	_, cmd := view.Update(msg)

	require.NotNil(t, cmd)
	result := cmd()
	changed, ok := result.(messages.ViewChanged)
	require.True(t, ok)
	assert.Equal(t, messages.ViewSources, changed.View)
}

func TestView_Update_KeyMsg_Escape_FromEnterConfig(t *testing.T) {
	view := NewView(nil, nil, nil, nil, nil, nil)
	view.step = StepEnterConfig

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	view.Update(msg)

	assert.Equal(t, StepSelectConnector, view.step)
}

func TestView_Update_KeyMsg_Escape_FromSelectAuth(t *testing.T) {
	view := NewView(nil, nil, nil, nil, nil, nil)
	view.step = StepSelectAuth

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	view.Update(msg)

	assert.Equal(t, StepEnterConfig, view.step)
}

func TestView_Update_KeyMsg_Escape_FromEnterCredentials_CreatingNew(t *testing.T) {
	view := NewView(nil, nil, nil, nil, nil, nil)
	view.step = StepEnterCredentials
	view.creatingNewAuth = true

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	view.Update(msg)

	assert.Equal(t, StepSelectAuth, view.step)
}

func TestView_Update_KeyMsg_Escape_FromEnterCredentials_NotCreatingNew(t *testing.T) {
	view := NewView(nil, nil, nil, nil, nil, nil)
	view.step = StepEnterCredentials
	view.creatingNewAuth = false

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	view.Update(msg)

	assert.Equal(t, StepEnterConfig, view.step)
}

func TestView_Update_KeyMsg_Escape_FromOAuthFlow(t *testing.T) {
	view := NewView(nil, nil, nil, nil, nil, nil)
	view.step = StepOAuthFlow
	view.waitingForAuth = true

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	view.Update(msg)

	assert.Equal(t, StepEnterCredentials, view.step)
	assert.False(t, view.waitingForAuth)
}

func TestView_Update_KeyMsg_Escape_FromComplete(t *testing.T) {
	view := NewView(nil, nil, nil, nil, nil, nil)
	view.step = StepComplete

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	_, cmd := view.Update(msg)

	require.NotNil(t, cmd)
	result := cmd()
	changed, ok := result.(messages.ViewChanged)
	require.True(t, ok)
	assert.Equal(t, messages.ViewSources, changed.View)
}

func TestView_Update_KeyMsg_NavigateConnectors_Down(t *testing.T) {
	view := NewView(nil, nil, nil, nil, nil, nil)
	view.step = StepSelectConnector
	view.connectors = []domain.ConnectorType{
		{ID: "filesystem"},
		{ID: "github"},
	}
	view.selected = 0

	msg := tea.KeyMsg{Type: tea.KeyDown}
	view.Update(msg)

	assert.Equal(t, 1, view.selected)
}

func TestView_Update_KeyMsg_NavigateConnectors_Up(t *testing.T) {
	view := NewView(nil, nil, nil, nil, nil, nil)
	view.step = StepSelectConnector
	view.connectors = []domain.ConnectorType{
		{ID: "filesystem"},
		{ID: "github"},
	}
	view.selected = 1

	msg := tea.KeyMsg{Type: tea.KeyUp}
	view.Update(msg)

	assert.Equal(t, 0, view.selected)
}

func TestView_Update_KeyMsg_NavigateConnectors_J(t *testing.T) {
	view := NewView(nil, nil, nil, nil, nil, nil)
	view.step = StepSelectConnector
	view.connectors = []domain.ConnectorType{
		{ID: "filesystem"},
		{ID: "github"},
	}
	view.selected = 0

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	view.Update(msg)

	assert.Equal(t, 1, view.selected)
}

func TestView_Update_KeyMsg_NavigateConnectors_K(t *testing.T) {
	view := NewView(nil, nil, nil, nil, nil, nil)
	view.step = StepSelectConnector
	view.connectors = []domain.ConnectorType{
		{ID: "filesystem"},
		{ID: "github"},
	}
	view.selected = 1

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	view.Update(msg)

	assert.Equal(t, 0, view.selected)
}

func TestView_Update_KeyMsg_SelectConnector(t *testing.T) {
	view := NewView(nil, nil, nil, nil, nil, nil)
	view.step = StepSelectConnector
	view.connectors = []domain.ConnectorType{
		{
			ID:             "filesystem",
			Name:           "Filesystem",
			AuthCapability: domain.AuthCapNone,
			AuthMethod:     domain.AuthMethodNone,
			ConfigKeys: []domain.ConfigKey{
				{Key: "path", Required: true},
			},
		},
	}
	view.selected = 0

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	view.Update(msg)

	assert.Equal(t, StepEnterConfig, view.step)
	require.NotNil(t, view.connector)
	assert.Equal(t, "filesystem", view.connector.ID)
	assert.Len(t, view.configInputs, 1)
}

func TestView_Update_KeyMsg_EnterComplete(t *testing.T) {
	view := NewView(nil, nil, nil, nil, nil, nil)
	view.step = StepComplete

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := view.Update(msg)

	require.NotNil(t, cmd)
	result := cmd()
	changed, ok := result.(messages.ViewChanged)
	require.True(t, ok)
	assert.Equal(t, messages.ViewSources, changed.View)
}

func TestView_Update_SourceAdded_Success(t *testing.T) {
	view := NewView(nil, nil, nil, nil, nil, nil)

	source := domain.Source{ID: "src-1", Name: "Test"}
	msg := messages.SourceAdded{Source: source, Err: nil}
	view.Update(msg)

	require.NotNil(t, view.source)
	assert.Equal(t, "src-1", view.source.ID)
	assert.Equal(t, StepComplete, view.step)
}

func TestView_Update_SourceAdded_Error(t *testing.T) {
	view := NewView(nil, nil, nil, nil, nil, nil)

	msg := messages.SourceAdded{Err: errors.New("failed to add")}
	view.Update(msg)

	assert.Error(t, view.err)
}

func TestView_Update_AuthProvidersLoaded_WithProviders(t *testing.T) {
	view := NewView(nil, nil, nil, nil, nil, nil)
	providers := []domain.AuthProvider{
		{ID: "auth-1", Name: "My Auth"},
	}

	msg := authProvidersLoaded{authProviders: providers}
	view.Update(msg)

	assert.Len(t, view.authProviders, 1)
	assert.Equal(t, StepSelectAuth, view.step)
	assert.Equal(t, 0, view.selectedAuthIndex)
}

func TestView_Update_AuthProvidersLoaded_NoProviders(t *testing.T) {
	view := NewView(nil, nil, nil, nil, nil, nil)
	view.connector = &domain.ConnectorType{
		ID:             "github",
		AuthCapability: domain.AuthCapPAT,
		AuthMethod:     domain.AuthMethodPAT,
	}

	msg := authProvidersLoaded{authProviders: []domain.AuthProvider{}}
	_, cmd := view.Update(msg)

	assert.Equal(t, StepEnterCredentials, view.step)
	assert.False(t, view.creatingNewAuth)
	require.NotNil(t, cmd)
}

func TestView_Update_OAuthFlowCompleted_Success(t *testing.T) {
	sourceService := &MockSourceService{}
	credentialsService := &MockCredentialsService{}
	view := NewView(nil, sourceService, nil, nil, nil, credentialsService)
	view.waitingForAuth = true
	view.selectedAuthProviderID = "auth-provider-1"
	view.pendingOAuthTokens = &domain.OAuthCredentials{
		AccessToken:  "test-token",
		RefreshToken: "test-refresh",
		TokenType:    "Bearer",
	}
	view.connector = &domain.ConnectorType{
		ID:   "google-drive",
		Name: "Google Drive",
	}
	ti := textinput.New()
	ti.SetValue("test-folder")
	view.configInputs = []textinput.Model{ti}
	view.configKeys = []string{"folder"}

	msg := messages.OAuthFlowCompleted{Err: nil}
	_, cmd := view.Update(msg)

	assert.False(t, view.waitingForAuth)
	require.NotNil(t, cmd)
}

func TestView_Update_OAuthFlowCompleted_Error(t *testing.T) {
	view := NewView(nil, nil, nil, nil, nil, nil)
	view.waitingForAuth = true

	msg := messages.OAuthFlowCompleted{Err: errors.New("oauth failed")}
	view.Update(msg)

	assert.False(t, view.waitingForAuth)
	assert.Error(t, view.err)
	assert.Equal(t, StepEnterCredentials, view.step)
}

func TestView_ValidateConfig_Valid(t *testing.T) {
	view := NewView(nil, nil, nil, nil, nil, nil)
	view.connector = &domain.ConnectorType{
		ConfigKeys: []domain.ConfigKey{
			{Key: "path", Required: true},
		},
	}
	ti := textinput.New()
	ti.SetValue("/home/user")
	view.configInputs = []textinput.Model{ti}

	result := view.validateConfig()

	assert.True(t, result)
	assert.Nil(t, view.err)
}

func TestView_ValidateConfig_MissingRequired(t *testing.T) {
	view := NewView(nil, nil, nil, nil, nil, nil)
	view.connector = &domain.ConnectorType{
		ConfigKeys: []domain.ConfigKey{
			{Key: "path", Label: "Path", Required: true},
		},
	}
	ti := textinput.New()
	ti.SetValue("")
	view.configInputs = []textinput.Model{ti}

	result := view.validateConfig()

	assert.False(t, result)
	assert.Error(t, view.err)
}

func TestView_ValidateConfig_NilConnector(t *testing.T) {
	view := NewView(nil, nil, nil, nil, nil, nil)
	view.connector = nil

	result := view.validateConfig()

	assert.False(t, result)
}

func TestView_CreateSource_Success(t *testing.T) {
	sourceService := &MockSourceService{}
	view := NewView(nil, sourceService, nil, nil, nil, nil)
	view.connector = &domain.ConnectorType{
		ID:   "filesystem",
		Name: "Filesystem",
	}
	ti := textinput.New()
	ti.SetValue("/home/user")
	view.configInputs = []textinput.Model{ti}
	view.configKeys = []string{"path"}

	cmd := view.createSource()

	require.NotNil(t, cmd)
	msg := cmd()
	added, ok := msg.(messages.SourceAdded)
	require.True(t, ok)
	assert.NoError(t, added.Err)
	assert.Equal(t, "filesystem", added.Source.Type)
}

func TestView_CreateSource_NilService(t *testing.T) {
	view := NewView(nil, nil, nil, nil, nil, nil)
	view.connector = &domain.ConnectorType{ID: "filesystem"}

	cmd := view.createSource()

	require.NotNil(t, cmd)
	msg := cmd()
	added, ok := msg.(messages.SourceAdded)
	require.True(t, ok)
	assert.Error(t, added.Err)
}

func TestView_CreateSource_WithNameFromPath(t *testing.T) {
	sourceService := &MockSourceService{}
	view := NewView(nil, sourceService, nil, nil, nil, nil)
	view.connector = &domain.ConnectorType{
		ID:   "filesystem",
		Name: "Filesystem",
	}
	ti := textinput.New()
	ti.SetValue("/home/user/docs")
	view.configInputs = []textinput.Model{ti}
	view.configKeys = []string{"path"}

	cmd := view.createSource()

	require.NotNil(t, cmd)
	msg := cmd()
	added, ok := msg.(messages.SourceAdded)
	require.True(t, ok)
	assert.NoError(t, added.Err)
	assert.Equal(t, "/home/user/docs", added.Source.Name)
}

func TestView_CreateSource_WithNameFromOwnerRepo(t *testing.T) {
	sourceService := &MockSourceService{}
	view := NewView(nil, sourceService, nil, nil, nil, nil)
	view.connector = &domain.ConnectorType{
		ID:   "github",
		Name: "GitHub",
	}
	ti1 := textinput.New()
	ti1.SetValue("myorg")
	ti2 := textinput.New()
	ti2.SetValue("myrepo")
	view.configInputs = []textinput.Model{ti1, ti2}
	view.configKeys = []string{"owner", "repo"}

	cmd := view.createSource()

	require.NotNil(t, cmd)
	msg := cmd()
	added, ok := msg.(messages.SourceAdded)
	require.True(t, ok)
	assert.NoError(t, added.Err)
	assert.Equal(t, "myorg/myrepo", added.Source.Name)
}

func TestView_CreateAuthorizationAndSource_Success(t *testing.T) {
	sourceService := &MockSourceService{}
	credentialsService := &MockCredentialsService{}
	view := NewView(nil, sourceService, nil, nil, nil, credentialsService)
	view.connector = &domain.ConnectorType{
		ID:             "github",
		Name:           "GitHub",
		ProviderType:   domain.ProviderGitHub,
		AuthCapability: domain.AuthCapPAT | domain.AuthCapOAuth,
		AuthMethod:     domain.AuthMethodPAT,
	}
	view.configInputs = []textinput.Model{}
	view.configKeys = []string{}
	view.tokenInput.SetValue("ghp_test_token")

	cmd := view.createAuthorizationAndSource()

	require.NotNil(t, cmd)
	msg := cmd()
	added, ok := msg.(messages.SourceAdded)
	require.True(t, ok)
	assert.NoError(t, added.Err)
}

func TestView_CreateAuthorizationAndSource_NilService(t *testing.T) {
	view := NewView(nil, nil, nil, nil, nil, nil)
	view.connector = &domain.ConnectorType{ID: "github"}

	cmd := view.createAuthorizationAndSource()

	require.NotNil(t, cmd)
	msg := cmd()
	added, ok := msg.(messages.SourceAdded)
	require.True(t, ok)
	assert.Error(t, added.Err)
}

func TestView_CreateAuthorizationAndStartOAuth_Success(t *testing.T) {
	authProviderService := &MockAuthProviderService{}
	providerRegistry := &MockProviderRegistry{}
	connectorRegistry := &MockConnectorRegistry{}
	view := NewView(nil, nil, connectorRegistry, providerRegistry, authProviderService, nil)
	view.connector = &domain.ConnectorType{
		ID:             "google-drive",
		Name:           "Google Drive",
		ProviderType:   domain.ProviderGoogle,
		AuthCapability: domain.AuthCapOAuth,
		AuthMethod:     domain.AuthMethodOAuth,
	}
	view.clientIDInput.SetValue("test-client-id")
	view.clientSecretInput.SetValue("test-client-secret")

	cmd := view.createAuthorizationAndStartOAuth()

	require.NotNil(t, cmd)
	msg := cmd()
	started, ok := msg.(oauthFlowStarted)
	require.True(t, ok)
	require.NotNil(t, started.flowState)
	assert.Contains(t, started.flowState.AuthURL, "example.com/oauth")
	// Note: waitingForAuth and step are set in Update() when handling this message,
	// not in the command function itself
}

func TestView_CreateAuthorizationAndStartOAuth_SaveAuthProviderError(t *testing.T) {
	authProviderService := &MockAuthProviderService{
		SaveFunc: func(ctx context.Context, provider domain.AuthProvider) error {
			return errors.New("save failed")
		},
	}
	providerRegistry := &MockProviderRegistry{}
	view := NewView(nil, nil, nil, providerRegistry, authProviderService, nil)
	view.connector = &domain.ConnectorType{
		ID:             "google-drive",
		ProviderType:   domain.ProviderGoogle,
		AuthCapability: domain.AuthCapOAuth,
		AuthMethod:     domain.AuthMethodOAuth,
	}
	view.clientIDInput.SetValue("test-client-id")
	view.clientSecretInput.SetValue("test-client-secret")

	cmd := view.createAuthorizationAndStartOAuth()

	require.NotNil(t, cmd)
	msg := cmd()
	errMsg, ok := msg.(messages.ErrorOccurred)
	require.True(t, ok)
	assert.Error(t, errMsg.Err)
}

func TestView_CreateAuthorizationAndStartOAuth_NilProviderRegistry(t *testing.T) {
	// When providerRegistry is nil, the function still works but without OAuth endpoint hints
	// connectorRegistry is required for buildOAuthFlowState
	authProviderService := &MockAuthProviderService{}
	connectorRegistry := &MockConnectorRegistry{}
	view := NewView(nil, nil, connectorRegistry, nil, authProviderService, nil)
	view.connector = &domain.ConnectorType{
		ID:             "google-drive",
		ProviderType:   domain.ProviderGoogle,
		AuthCapability: domain.AuthCapOAuth,
		AuthMethod:     domain.AuthMethodOAuth,
	}
	view.clientIDInput.SetValue("test-client-id")
	view.clientSecretInput.SetValue("test-client-secret")

	cmd := view.createAuthorizationAndStartOAuth()

	require.NotNil(t, cmd)
	msg := cmd()
	// Should still succeed - providerRegistry is optional for OAuth flow
	started, ok := msg.(oauthFlowStarted)
	require.True(t, ok)
	require.NotNil(t, started.flowState)
}

func TestView_CreateAuthorizationAndStartOAuth_NilService(t *testing.T) {
	view := NewView(nil, nil, nil, nil, nil, nil)
	view.connector = &domain.ConnectorType{ID: "google-drive"}

	cmd := view.createAuthorizationAndStartOAuth()

	require.NotNil(t, cmd)
	msg := cmd()
	errMsg, ok := msg.(messages.ErrorOccurred)
	require.True(t, ok)
	assert.Error(t, errMsg.Err)
}

func TestView_CreateSourceWithNewAuthorization_Success(t *testing.T) {
	sourceService := &MockSourceService{}
	credentialsService := &MockCredentialsService{}
	view := NewView(nil, sourceService, nil, nil, nil, credentialsService)
	view.connector = &domain.ConnectorType{
		ID:           "google-drive",
		Name:         "Google Drive",
		ProviderType: domain.ProviderGoogle,
	}
	view.selectedAuthProviderID = "auth-provider-1"
	view.pendingOAuthTokens = &domain.OAuthCredentials{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
		TokenType:    "Bearer",
	}
	view.accountIdentifier = "test@example.com"
	ti := textinput.New()
	ti.SetValue("My Folder")
	view.configInputs = []textinput.Model{ti}
	view.configKeys = []string{"folder"}

	cmd := view.createSourceWithNewAuthorization()

	require.NotNil(t, cmd)
	msg := cmd()
	added, ok := msg.(messages.SourceAdded)
	require.True(t, ok)
	assert.NoError(t, added.Err)
	assert.Equal(t, "auth-provider-1", added.Source.AuthProviderID)
}

func TestView_CreateSourceWithNewAuthorization_NilService(t *testing.T) {
	view := NewView(nil, nil, nil, nil, nil, nil)
	view.connector = &domain.ConnectorType{ID: "google-drive", ProviderType: domain.ProviderGoogle}
	view.selectedAuthProviderID = "auth-provider-1"
	view.pendingOAuthTokens = &domain.OAuthCredentials{
		AccessToken: "test-token",
	}

	cmd := view.createSourceWithNewAuthorization()

	require.NotNil(t, cmd)
	msg := cmd()
	added, ok := msg.(messages.SourceAdded)
	require.True(t, ok)
	assert.Error(t, added.Err)
}

func TestView_CreateSourceWithNewAuthorization_EmptyAuthProviderID(t *testing.T) {
	sourceService := &MockSourceService{}
	credentialsService := &MockCredentialsService{}
	view := NewView(nil, sourceService, nil, nil, nil, credentialsService)
	view.connector = &domain.ConnectorType{ID: "google-drive", ProviderType: domain.ProviderGoogle}
	view.selectedAuthProviderID = "" // Empty auth provider ID
	view.pendingOAuthTokens = &domain.OAuthCredentials{
		AccessToken: "test-token",
	}

	cmd := view.createSourceWithNewAuthorization()

	require.NotNil(t, cmd)
	msg := cmd()
	added, ok := msg.(messages.SourceAdded)
	require.True(t, ok)
	assert.Error(t, added.Err)
}

func TestView_LoadAuthorizations_Success(t *testing.T) {
	authProviderService := &MockAuthProviderService{
		ListByProviderFunc: func(ctx context.Context, provider domain.ProviderType) ([]domain.AuthProvider, error) {
			return []domain.AuthProvider{
				{ID: "auth-provider-1", Name: "My Auth Provider"},
			}, nil
		},
	}
	view := NewView(nil, nil, nil, nil, authProviderService, nil)
	view.connector = &domain.ConnectorType{ID: "github", ProviderType: domain.ProviderGitHub}

	cmd := view.loadAuthorizations()

	require.NotNil(t, cmd)
	msg := cmd()
	loaded, ok := msg.(authProvidersLoaded)
	require.True(t, ok)
	assert.Len(t, loaded.authProviders, 1)
}

func TestView_LoadAuthorizations_Error(t *testing.T) {
	authProviderService := &MockAuthProviderService{
		ListByProviderFunc: func(ctx context.Context, provider domain.ProviderType) ([]domain.AuthProvider, error) {
			return nil, errors.New("list failed")
		},
	}
	view := NewView(nil, nil, nil, nil, authProviderService, nil)
	view.connector = &domain.ConnectorType{ID: "github", ProviderType: domain.ProviderGitHub}

	cmd := view.loadAuthorizations()

	require.NotNil(t, cmd)
	msg := cmd()
	errMsg, ok := msg.(messages.ErrorOccurred)
	require.True(t, ok)
	assert.Error(t, errMsg.Err)
}

func TestView_LoadAuthorizations_NilService(t *testing.T) {
	view := NewView(nil, nil, nil, nil, nil, nil)
	view.connector = &domain.ConnectorType{ID: "github", ProviderType: domain.ProviderGitHub}

	cmd := view.loadAuthorizations()

	require.NotNil(t, cmd)
	msg := cmd()
	errMsg, ok := msg.(messages.ErrorOccurred)
	require.True(t, ok)
	assert.Error(t, errMsg.Err)
}

func TestView_DetermineNextStepAfterConfig_NoAuth(t *testing.T) {
	sourceService := &MockSourceService{}
	view := NewView(nil, sourceService, nil, nil, nil, nil)
	view.connector = &domain.ConnectorType{
		ID:             "filesystem",
		AuthCapability: domain.AuthCapNone,
		AuthMethod:     domain.AuthMethodNone,
	}
	view.configInputs = []textinput.Model{}
	view.configKeys = []string{}

	cmd := view.determineNextStepAfterConfig()

	require.NotNil(t, cmd)
	msg := cmd()
	added, ok := msg.(messages.SourceAdded)
	require.True(t, ok)
	assert.NoError(t, added.Err)
}

func TestView_DetermineNextStepAfterConfig_PAT(t *testing.T) {
	view := NewView(nil, nil, nil, nil, nil, nil)
	view.connector = &domain.ConnectorType{
		ID:             "github",
		AuthCapability: domain.AuthCapPAT,
		AuthMethod:     domain.AuthMethodPAT,
	}

	cmd := view.determineNextStepAfterConfig()

	require.NotNil(t, cmd)
	// For single auth method, goes to StepSelectAuthMethod which then proceeds
	assert.True(t, view.step == StepSelectAuthMethod || view.step == StepEnterCredentials)
}

func TestView_DetermineNextStepAfterConfig_OAuth_MultipleConnectors(t *testing.T) {
	authProviderService := &MockAuthProviderService{}
	providerRegistry := &MockProviderRegistry{
		HasMultipleConnectorsFunc: func(provider domain.ProviderType) bool {
			return true
		},
	}
	view := NewView(nil, nil, nil, providerRegistry, authProviderService, nil)
	view.connector = &domain.ConnectorType{
		ID:             "google-drive",
		ProviderType:   domain.ProviderGoogle,
		AuthCapability: domain.AuthCapOAuth,
		AuthMethod:     domain.AuthMethodOAuth,
	}

	cmd := view.determineNextStepAfterConfig()

	require.NotNil(t, cmd)
}

func TestView_DetermineNextStepAfterConfig_OAuth_SingleConnector(t *testing.T) {
	providerRegistry := &MockProviderRegistry{
		HasMultipleConnectorsFunc: func(provider domain.ProviderType) bool {
			return false
		},
	}
	view := NewView(nil, nil, nil, providerRegistry, nil, nil)
	view.connector = &domain.ConnectorType{
		ID:             "github",
		ProviderType:   domain.ProviderGitHub,
		AuthCapability: domain.AuthCapOAuth,
		AuthMethod:     domain.AuthMethodOAuth,
	}

	cmd := view.determineNextStepAfterConfig()

	require.NotNil(t, cmd)
	// For single auth method, goes to StepSelectAuthMethod which then proceeds
	assert.True(t, view.step == StepSelectAuthMethod || view.step == StepEnterCredentials)
}

func TestView_DetermineNextStepAfterConfig_NilConnector(t *testing.T) {
	view := NewView(nil, nil, nil, nil, nil, nil)
	view.connector = nil

	cmd := view.determineNextStepAfterConfig()

	assert.Nil(t, cmd)
}

func TestView_HandleAuthSelect_NavigateDown(t *testing.T) {
	view := NewView(nil, nil, nil, nil, nil, nil)
	view.step = StepSelectAuth
	view.authProviders = []domain.AuthProvider{
		{ID: "auth-1"},
		{ID: "auth-2"},
	}
	view.selectedAuthIndex = 0

	msg := tea.KeyMsg{Type: tea.KeyDown}
	view.handleAuthSelect(msg)

	assert.Equal(t, 1, view.selectedAuthIndex)
}

func TestView_HandleAuthSelect_NavigateUp(t *testing.T) {
	view := NewView(nil, nil, nil, nil, nil, nil)
	view.step = StepSelectAuth
	view.authProviders = []domain.AuthProvider{
		{ID: "auth-1"},
		{ID: "auth-2"},
	}
	view.selectedAuthIndex = 1

	msg := tea.KeyMsg{Type: tea.KeyUp}
	view.handleAuthSelect(msg)

	assert.Equal(t, 0, view.selectedAuthIndex)
}

func TestView_HandleAuthSelect_ShortcutN(t *testing.T) {
	view := NewView(nil, nil, nil, nil, nil, nil)
	view.step = StepSelectAuth
	view.connector = &domain.ConnectorType{
		ID:             "google-drive",
		AuthCapability: domain.AuthCapOAuth,
		AuthMethod:     domain.AuthMethodOAuth,
	}

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
	_, cmd := view.handleAuthSelect(msg)

	assert.True(t, view.creatingNewAuth)
	assert.Equal(t, StepEnterCredentials, view.step)
	require.NotNil(t, cmd)
}

func TestView_HandleAuthSelect_ShortcutA(t *testing.T) {
	view := NewView(nil, nil, nil, nil, nil, nil)
	view.step = StepSelectAuth
	view.connector = &domain.ConnectorType{
		ID:             "google-drive",
		AuthCapability: domain.AuthCapOAuth,
		AuthMethod:     domain.AuthMethodOAuth,
	}

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
	_, cmd := view.handleAuthSelect(msg)

	assert.True(t, view.creatingNewAuth)
	assert.Equal(t, StepEnterCredentials, view.step)
	require.NotNil(t, cmd)
}

func TestView_HandleAuthSelect_EnterAddNew(t *testing.T) {
	view := NewView(nil, nil, nil, nil, nil, nil)
	view.step = StepSelectAuth
	view.connector = &domain.ConnectorType{
		ID:             "google-drive",
		AuthCapability: domain.AuthCapOAuth,
		AuthMethod:     domain.AuthMethodOAuth,
	}
	view.authProviders = []domain.AuthProvider{
		{ID: "auth-1"},
	}
	view.selectedAuthIndex = 1 // "Add new" is at index len(authorizations)

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := view.handleAuthSelect(msg)

	assert.True(t, view.creatingNewAuth)
	assert.Equal(t, StepEnterCredentials, view.step)
	require.NotNil(t, cmd)
}

func TestView_HandleAuthSelect_EnterExisting(t *testing.T) {
	sourceService := &MockSourceService{}
	view := NewView(nil, sourceService, nil, nil, nil, nil)
	view.step = StepSelectAuth
	view.connector = &domain.ConnectorType{ID: "github", Name: "GitHub"}
	view.authProviders = []domain.AuthProvider{
		{ID: "auth-1", Name: "My Auth"},
	}
	view.selectedAuthIndex = 0
	view.configInputs = []textinput.Model{}
	view.configKeys = []string{}

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := view.handleAuthSelect(msg)

	require.NotNil(t, cmd)
}

func TestView_HandleAuthSelect_EnterNoAuth(t *testing.T) {
	view := NewView(nil, nil, nil, nil, nil, nil)
	view.step = StepSelectAuth
	view.connector = &domain.ConnectorType{
		ID:             "google-drive",
		AuthCapability: domain.AuthCapOAuth,
		AuthMethod:     domain.AuthMethodOAuth,
	}
	view.authProviders = []domain.AuthProvider{}
	// selectedAuthIndex is 0, but there are no authorizations (index 0 would be "Add new" in this case)
	// This means index 0 = len(authorizations), which triggers "Add new" flow
	view.selectedAuthIndex = 0

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := view.handleAuthSelect(msg)

	// When there are no authorizations, index 0 == len(authorizations), so it should trigger "Add new"
	assert.True(t, view.creatingNewAuth)
	assert.Equal(t, StepEnterCredentials, view.step)
	require.NotNil(t, cmd)
}

func TestView_HandleCredentialsInput_PAT_Enter_Success(t *testing.T) {
	sourceService := &MockSourceService{}
	credentialsService := &MockCredentialsService{}
	view := NewView(nil, sourceService, nil, nil, nil, credentialsService)
	view.step = StepEnterCredentials
	view.connector = &domain.ConnectorType{
		ID:             "github",
		ProviderType:   domain.ProviderGitHub,
		AuthCapability: domain.AuthCapPAT | domain.AuthCapOAuth,
		AuthMethod:     domain.AuthMethodPAT,
	}
	view.chosenAuthMethod = domain.AuthMethodPAT // Set the chosen auth method
	view.tokenInput.SetValue("test-token")
	view.configInputs = []textinput.Model{}
	view.configKeys = []string{}

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := view.handleCredentialsInput(msg)

	require.NotNil(t, cmd)
}

func TestView_HandleCredentialsInput_PAT_Enter_EmptyToken(t *testing.T) {
	view := NewView(nil, nil, nil, nil, nil, nil)
	view.step = StepEnterCredentials
	view.connector = &domain.ConnectorType{
		AuthCapability: domain.AuthCapPAT,
		AuthMethod:     domain.AuthMethodPAT,
	}
	view.chosenAuthMethod = domain.AuthMethodPAT // Set the chosen auth method
	view.tokenInput.SetValue("")

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	view.handleCredentialsInput(msg)

	assert.Error(t, view.err)
}

func TestView_HandleCredentialsInput_OAuth_Enter_EmptyClientID(t *testing.T) {
	view := NewView(nil, nil, nil, nil, nil, nil)
	view.step = StepEnterCredentials
	view.connector = &domain.ConnectorType{
		AuthCapability: domain.AuthCapOAuth,
		AuthMethod:     domain.AuthMethodOAuth,
	}
	view.chosenAuthMethod = domain.AuthMethodOAuth
	view.clientIDInput.SetValue("")
	view.clientSecretInput.SetValue("secret")

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	view.handleCredentialsInput(msg)

	assert.Error(t, view.err)
}

func TestView_HandleCredentialsInput_OAuth_Enter_EmptyClientSecret(t *testing.T) {
	view := NewView(nil, nil, nil, nil, nil, nil)
	view.step = StepEnterCredentials
	view.connector = &domain.ConnectorType{
		AuthCapability: domain.AuthCapOAuth,
		AuthMethod:     domain.AuthMethodOAuth,
	}
	view.chosenAuthMethod = domain.AuthMethodOAuth
	view.clientIDInput.SetValue("client-id")
	view.clientSecretInput.SetValue("")

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	view.handleCredentialsInput(msg)

	assert.Error(t, view.err)
}

func TestView_HandleCredentialsInput_OAuth_Enter_Success(t *testing.T) {
	authProviderService := &MockAuthProviderService{}
	providerRegistry := &MockProviderRegistry{}
	view := NewView(nil, nil, nil, providerRegistry, authProviderService, nil)
	view.step = StepEnterCredentials
	view.connector = &domain.ConnectorType{
		ID:             "google-drive",
		ProviderType:   domain.ProviderGoogle,
		AuthCapability: domain.AuthCapOAuth,
		AuthMethod:     domain.AuthMethodOAuth,
	}
	view.chosenAuthMethod = domain.AuthMethodOAuth
	view.clientIDInput.SetValue("client-id")
	view.clientSecretInput.SetValue("client-secret")

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := view.handleCredentialsInput(msg)

	require.NotNil(t, cmd)
	assert.Nil(t, view.err)
}

func TestView_HandleCredentialsInput_Tab_OAuth(t *testing.T) {
	view := NewView(nil, nil, nil, nil, nil, nil)
	view.connector = &domain.ConnectorType{
		AuthCapability: domain.AuthCapOAuth,
		AuthMethod:     domain.AuthMethodOAuth,
	}
	view.chosenAuthMethod = domain.AuthMethodOAuth
	view.credentialFocus = 0

	msg := tea.KeyMsg{Type: tea.KeyTab}
	_, cmd := view.handleCredentialsInput(msg)

	assert.Equal(t, 1, view.credentialFocus)
	require.NotNil(t, cmd)
}

func TestView_HandleCredentialsInput_ShiftTab_OAuth(t *testing.T) {
	view := NewView(nil, nil, nil, nil, nil, nil)
	view.connector = &domain.ConnectorType{
		AuthCapability: domain.AuthCapOAuth,
		AuthMethod:     domain.AuthMethodOAuth,
	}
	view.chosenAuthMethod = domain.AuthMethodOAuth
	view.credentialFocus = 1

	msg := tea.KeyMsg{Type: tea.KeyShiftTab}
	_, cmd := view.handleCredentialsInput(msg)

	assert.Equal(t, 0, view.credentialFocus)
	require.NotNil(t, cmd)
}

func TestView_HandleCredentialsInput_Tab_PAT(t *testing.T) {
	view := NewView(nil, nil, nil, nil, nil, nil)
	view.connector = &domain.ConnectorType{
		AuthCapability: domain.AuthCapPAT,
		AuthMethod:     domain.AuthMethodPAT,
	}
	view.chosenAuthMethod = domain.AuthMethodPAT

	msg := tea.KeyMsg{Type: tea.KeyTab}
	_, cmd := view.handleCredentialsInput(msg)

	assert.Nil(t, cmd) // PAT only has one field
}

func TestView_HandleCredentialsInput_NilConnector(t *testing.T) {
	view := NewView(nil, nil, nil, nil, nil, nil)
	view.connector = nil

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := view.handleCredentialsInput(msg)

	assert.Nil(t, cmd)
}

func TestView_HandleCredentialsInput_TextInput_ClientID(t *testing.T) {
	view := NewView(nil, nil, nil, nil, nil, nil)
	view.connector = &domain.ConnectorType{
		AuthCapability: domain.AuthCapOAuth,
		AuthMethod:     domain.AuthMethodOAuth,
	}
	view.chosenAuthMethod = domain.AuthMethodOAuth
	view.credentialFocus = 0

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
	view.handleCredentialsInput(msg)

	// Should forward to client ID input
	assert.Equal(t, 0, view.credentialFocus)
}

func TestView_HandleCredentialsInput_TextInput_ClientSecret(t *testing.T) {
	view := NewView(nil, nil, nil, nil, nil, nil)
	view.connector = &domain.ConnectorType{
		AuthCapability: domain.AuthCapOAuth,
		AuthMethod:     domain.AuthMethodOAuth,
	}
	view.chosenAuthMethod = domain.AuthMethodOAuth
	view.credentialFocus = 1

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}}
	view.handleCredentialsInput(msg)

	// Should forward to client secret input
	assert.Equal(t, 1, view.credentialFocus)
}

func TestView_HandleCredentialsInput_TextInput_Token(t *testing.T) {
	view := NewView(nil, nil, nil, nil, nil, nil)
	view.connector = &domain.ConnectorType{
		AuthCapability: domain.AuthCapPAT,
		AuthMethod:     domain.AuthMethodPAT,
	}
	view.chosenAuthMethod = domain.AuthMethodPAT

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}}
	view.handleCredentialsInput(msg)

	// Should forward to token input
}

func TestView_HandleConfigInput_Enter_Valid(t *testing.T) {
	sourceService := &MockSourceService{}
	view := NewView(nil, sourceService, nil, nil, nil, nil)
	view.connector = &domain.ConnectorType{
		ID:             "filesystem",
		AuthCapability: domain.AuthCapNone,
		AuthMethod:     domain.AuthMethodNone,
		ConfigKeys: []domain.ConfigKey{
			{Key: "path", Required: true},
		},
	}
	ti := textinput.New()
	ti.SetValue("/home/user")
	view.configInputs = []textinput.Model{ti}
	view.configKeys = []string{"path"}

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := view.handleConfigInput(msg)

	require.NotNil(t, cmd)
}

func TestView_HandleConfigInput_Enter_Invalid(t *testing.T) {
	view := NewView(nil, nil, nil, nil, nil, nil)
	view.connector = &domain.ConnectorType{
		ConfigKeys: []domain.ConfigKey{
			{Key: "path", Label: "Path", Required: true},
		},
	}
	ti := textinput.New()
	ti.SetValue("")
	view.configInputs = []textinput.Model{ti}

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	view.handleConfigInput(msg)

	assert.Error(t, view.err)
}

func TestView_HandleConfigInput_Down(t *testing.T) {
	view := NewView(nil, nil, nil, nil, nil, nil)
	view.connector = &domain.ConnectorType{
		ConfigKeys: []domain.ConfigKey{
			{Key: "field1"},
			{Key: "field2"},
		},
	}
	view.initConfigInputs()
	view.focusIndex = 0

	msg := tea.KeyMsg{Type: tea.KeyDown}
	view.handleConfigInput(msg)

	assert.Equal(t, 1, view.focusIndex)
}

func TestView_HandleConfigInput_UpKey(t *testing.T) {
	view := NewView(nil, nil, nil, nil, nil, nil)
	view.connector = &domain.ConnectorType{
		ConfigKeys: []domain.ConfigKey{
			{Key: "field1"},
			{Key: "field2"},
		},
	}
	view.initConfigInputs()
	view.focusIndex = 1

	msg := tea.KeyMsg{Type: tea.KeyUp}
	view.handleConfigInput(msg)

	assert.Equal(t, 0, view.focusIndex)
}

func TestView_HandleConfigInput_Tab_Wrap(t *testing.T) {
	view := NewView(nil, nil, nil, nil, nil, nil)
	view.connector = &domain.ConnectorType{
		ConfigKeys: []domain.ConfigKey{
			{Key: "field1"},
			{Key: "field2"},
		},
	}
	view.initConfigInputs()
	view.focusIndex = 1

	msg := tea.KeyMsg{Type: tea.KeyTab}
	view.handleConfigInput(msg)

	assert.Equal(t, 0, view.focusIndex)
}

func TestView_HandleConfigInput_ShiftTab_Wrap(t *testing.T) {
	view := NewView(nil, nil, nil, nil, nil, nil)
	view.connector = &domain.ConnectorType{
		ConfigKeys: []domain.ConfigKey{
			{Key: "field1"},
			{Key: "field2"},
		},
	}
	view.initConfigInputs()
	view.focusIndex = 0

	msg := tea.KeyMsg{Type: tea.KeyShiftTab}
	view.handleConfigInput(msg)

	assert.Equal(t, 1, view.focusIndex)
}

func TestView_HandleConfigInput_TextInput(t *testing.T) {
	view := NewView(nil, nil, nil, nil, nil, nil)
	view.connector = &domain.ConnectorType{
		ConfigKeys: []domain.ConfigKey{
			{Key: "path"},
		},
	}
	view.initConfigInputs()

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
	view.handleConfigInput(msg)

	// Should forward to current input
	assert.Equal(t, 0, view.focusIndex)
}

func TestView_InitCredentialInputs(t *testing.T) {
	view := NewView(nil, nil, nil, nil, nil, nil)
	view.connector = &domain.ConnectorType{
		AuthCapability: domain.AuthCapOAuth,
		AuthMethod:     domain.AuthMethodOAuth,
	}
	view.clientIDInput.SetValue("old-value")

	view.initCredentialInputs()

	assert.Equal(t, 0, view.credentialFocus)
	assert.Equal(t, "", view.clientIDInput.Value())
	assert.Equal(t, "", view.clientSecretInput.Value())
	assert.Equal(t, "", view.tokenInput.Value())
}

func TestView_InitCredentialInputs_PAT(t *testing.T) {
	view := NewView(nil, nil, nil, nil, nil, nil)
	view.connector = &domain.ConnectorType{
		AuthCapability: domain.AuthCapPAT,
		AuthMethod:     domain.AuthMethodPAT,
	}

	view.initCredentialInputs()

	assert.Equal(t, 0, view.credentialFocus)
}

func TestView_UpdateCredentialFocus_ToClientID(t *testing.T) {
	view := NewView(nil, nil, nil, nil, nil, nil)
	view.credentialFocus = 0

	cmd := view.updateCredentialFocus()

	require.NotNil(t, cmd)
}

func TestView_UpdateCredentialFocus_ToClientSecret(t *testing.T) {
	view := NewView(nil, nil, nil, nil, nil, nil)
	view.credentialFocus = 1

	cmd := view.updateCredentialFocus()

	require.NotNil(t, cmd)
}

func TestView_View_SelectConnector(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewView(s, nil, nil, nil, nil, nil)
	view.ready = true
	view.step = StepSelectConnector
	view.connectors = []domain.ConnectorType{
		{ID: "filesystem", Name: "Filesystem", Description: "Local files"},
	}

	output := view.View()

	assert.Contains(t, output, "Add Source")
	assert.Contains(t, output, "Select a connector type")
	assert.Contains(t, output, "Filesystem")
}

func TestView_View_EnterConfig(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewView(s, nil, nil, nil, nil, nil)
	view.ready = true
	view.step = StepEnterConfig
	view.connector = &domain.ConnectorType{
		Name: "Filesystem",
		ConfigKeys: []domain.ConfigKey{
			{Key: "path", Label: "Path", Required: true},
		},
	}
	ti := textinput.New()
	view.configInputs = []textinput.Model{ti}

	output := view.View()

	assert.Contains(t, output, "Configure")
	assert.Contains(t, output, "Filesystem")
}

func TestView_View_SelectAuth(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewView(s, nil, nil, nil, nil, nil)
	view.ready = true
	view.step = StepSelectAuth
	view.connector = &domain.ConnectorType{
		Name: "Google Drive",
	}
	view.authProviders = []domain.AuthProvider{
		{ID: "auth-1", Name: "My Google Auth"},
	}

	output := view.View()

	assert.Contains(t, output, "OAuth app")
	assert.Contains(t, output, "My Google Auth")
	assert.Contains(t, output, "Create new OAuth app")
}

func TestView_View_EnterCredentials_PAT(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewView(s, nil, nil, nil, nil, nil)
	view.ready = true
	view.step = StepEnterCredentials
	view.connector = &domain.ConnectorType{
		Name:           "GitHub",
		AuthCapability: domain.AuthCapPAT | domain.AuthCapOAuth,
		AuthMethod:     domain.AuthMethodPAT,
	}
	view.chosenAuthMethod = domain.AuthMethodPAT

	output := view.View()

	assert.Contains(t, output, "Enter Personal Access Token")
	assert.Contains(t, output, "Token:")
}

func TestView_View_EnterCredentials_OAuth(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewView(s, nil, nil, nil, nil, nil)
	view.ready = true
	view.step = StepEnterCredentials
	view.connector = &domain.ConnectorType{
		Name:           "Google Drive",
		ProviderType:   domain.ProviderGoogle,
		AuthCapability: domain.AuthCapOAuth,
		AuthMethod:     domain.AuthMethodOAuth,
	}
	view.chosenAuthMethod = domain.AuthMethodOAuth

	output := view.View()

	assert.Contains(t, output, "Enter OAuth App Credentials")
	assert.Contains(t, output, "Client ID:")
	assert.Contains(t, output, "Client Secret:")
}

func TestView_View_OAuthFlow(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewView(s, nil, nil, nil, nil, nil)
	view.ready = true
	view.step = StepOAuthFlow
	view.waitingForAuth = true
	view.oauthState = &driving.OAuthFlowState{
		AuthURL: "https://example.com/oauth",
	}

	output := view.View()

	assert.Contains(t, output, "Authenticating")
	assert.Contains(t, output, "browser window")
	assert.Contains(t, output, "https://example.com/oauth")
}

func TestView_View_OAuthFlow_NotWaiting(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewView(s, nil, nil, nil, nil, nil)
	view.ready = true
	view.step = StepOAuthFlow
	view.waitingForAuth = false

	output := view.View()

	assert.Contains(t, output, "Starting authentication flow")
}

func TestView_View_Complete(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewView(s, nil, nil, nil, nil, nil)
	view.ready = true
	view.step = StepComplete
	view.source = &domain.Source{
		ID:   "src-1",
		Type: "filesystem",
		Name: "My Source",
	}

	output := view.View()

	assert.Contains(t, output, "Source Added Successfully")
	assert.Contains(t, output, "src-1")
}

func TestView_View_Error(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewView(s, nil, nil, nil, nil, nil)
	view.ready = true
	view.step = StepSelectConnector
	view.err = errors.New("something went wrong")

	output := view.View()

	assert.Contains(t, output, "Error")
	assert.Contains(t, output, "something went wrong")
}

func TestView_SetDimensions(t *testing.T) {
	view := NewView(nil, nil, nil, nil, nil, nil)

	view.SetDimensions(100, 50)

	assert.Equal(t, 100, view.width)
	assert.Equal(t, 50, view.height)
	assert.True(t, view.ready)
}

func TestView_Reset(t *testing.T) {
	view := NewView(nil, nil, nil, nil, nil, nil)
	view.step = StepComplete
	view.selected = 2
	view.connector = &domain.ConnectorType{ID: "test"}
	view.source = &domain.Source{ID: "src-1"}
	view.err = errors.New("error")
	view.clientIDInput.SetValue("client-id")
	view.clientSecretInput.SetValue("secret")
	view.tokenInput.SetValue("token")

	view.Reset()

	assert.Equal(t, StepSelectConnector, view.step)
	assert.Equal(t, 0, view.selected)
	assert.Nil(t, view.connector)
	assert.Nil(t, view.source)
	assert.Nil(t, view.err)
	assert.Equal(t, "", view.clientIDInput.Value())
	assert.Equal(t, "", view.clientSecretInput.Value())
	assert.Equal(t, "", view.tokenInput.Value())
}

func TestView_InitConfigInputs(t *testing.T) {
	view := NewView(nil, nil, nil, nil, nil, nil)
	view.connector = &domain.ConnectorType{
		ConfigKeys: []domain.ConfigKey{
			{Key: "path", Label: "Path", Required: true, Secret: false},
			{Key: "token", Label: "Token", Required: false, Secret: true},
		},
	}

	view.initConfigInputs()

	assert.Len(t, view.configInputs, 2)
	assert.Len(t, view.configKeys, 2)
	assert.Equal(t, "path", view.configKeys[0])
	assert.Equal(t, "token", view.configKeys[1])
	assert.Equal(t, 0, view.focusIndex)
}

func TestView_InitConfigInputs_NilConnector(t *testing.T) {
	view := NewView(nil, nil, nil, nil, nil, nil)
	view.connector = nil

	// Should not panic
	view.initConfigInputs()
}

func TestView_HandleConnectorSelect_Boundary(t *testing.T) {
	view := NewView(nil, nil, nil, nil, nil, nil)
	view.connectors = []domain.ConnectorType{
		{ID: "filesystem"},
	}
	view.selected = 0

	// Try to go up at the top
	msg := tea.KeyMsg{Type: tea.KeyUp}
	view.handleConnectorSelect(msg)
	assert.Equal(t, 0, view.selected)

	// Try to go down at the bottom
	msg = tea.KeyMsg{Type: tea.KeyDown}
	view.handleConnectorSelect(msg)
	assert.Equal(t, 0, view.selected) // Should stay at 0, only 1 item
}

func TestView_HandleConnectorSelect_EmptyConnectors(t *testing.T) {
	view := NewView(nil, nil, nil, nil, nil, nil)
	view.connectors = []domain.ConnectorType{}
	view.selected = 0

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	view.handleConnectorSelect(msg)

	// Should not crash
	assert.Nil(t, view.connector)
}

func TestView_HandleConnectorSelect_InvalidIndex(t *testing.T) {
	view := NewView(nil, nil, nil, nil, nil, nil)
	view.connectors = []domain.ConnectorType{
		{ID: "filesystem"},
	}
	view.selected = 10 // Invalid

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	view.handleConnectorSelect(msg)

	// Should not crash
	assert.Nil(t, view.connector)
}

func TestView_RenderHelp(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewView(s, nil, nil, nil, nil, nil)

	// Test each step
	view.step = StepSelectConnector
	help := view.renderHelp()
	assert.Contains(t, help, "navigate")

	view.step = StepEnterConfig
	help = view.renderHelp()
	assert.Contains(t, help, "tab")

	view.step = StepSelectAuth
	help = view.renderHelp()
	assert.Contains(t, help, "navigate")

	view.step = StepEnterCredentials
	help = view.renderHelp()
	assert.Contains(t, help, "tab")

	view.step = StepOAuthFlow
	help = view.renderHelp()
	assert.Contains(t, help, "cancel")

	view.step = StepComplete
	help = view.renderHelp()
	assert.Contains(t, help, "done")
}

func TestView_RenderConnectorSelect_Empty(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewView(s, nil, nil, nil, nil, nil)
	view.connectors = []domain.ConnectorType{}

	output := view.renderConnectorSelect()

	assert.Contains(t, output, "No connectors available")
}

func TestView_RenderConnectorSelect_WithConnectors(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewView(s, nil, nil, nil, nil, nil)
	view.connectors = []domain.ConnectorType{
		{ID: "filesystem", Name: "Filesystem", AuthCapability: domain.AuthCapNone, AuthMethod: domain.AuthMethodNone},
		{ID: "github", Name: "GitHub", AuthCapability: domain.AuthCapPAT | domain.AuthCapOAuth, AuthMethod: domain.AuthMethodPAT},
		{ID: "google-drive", Name: "Google Drive", AuthCapability: domain.AuthCapOAuth, AuthMethod: domain.AuthMethodOAuth},
	}
	view.selected = 0

	output := view.renderConnectorSelect()

	assert.Contains(t, output, "Filesystem")
	assert.Contains(t, output, "GitHub")
	assert.Contains(t, output, "Google Drive")
	// Updated to expect capability-based auth display
	assert.Contains(t, output, "[no auth]")
}

func TestView_RenderConfigInput_NoConfig(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewView(s, nil, nil, nil, nil, nil)
	view.connector = &domain.ConnectorType{
		Name:       "Test",
		ConfigKeys: []domain.ConfigKey{},
	}

	output := view.renderConfigInput()

	assert.Contains(t, output, "No configuration needed")
}

func TestView_RenderConfigInput_NilConnector(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewView(s, nil, nil, nil, nil, nil)
	view.connector = nil

	output := view.renderConfigInput()

	assert.Equal(t, "", output)
}

func TestView_RenderAuthSelect_NoAuthorizations(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewView(s, nil, nil, nil, nil, nil)
	view.connector = &domain.ConnectorType{
		Name:           "Google Drive",
		AuthCapability: domain.AuthCapOAuth,
		AuthMethod:     domain.AuthMethodOAuth,
	}
	view.authProviders = []domain.AuthProvider{}

	output := view.renderAuthSelect()

	assert.Contains(t, output, "OAuth app")
	assert.Contains(t, output, "Create new OAuth app")
}

func TestView_RenderAuthSelect_WithAuthorizations(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewView(s, nil, nil, nil, nil, nil)
	view.connector = &domain.ConnectorType{
		Name:           "Google Drive",
		AuthCapability: domain.AuthCapOAuth,
		AuthMethod:     domain.AuthMethodOAuth,
	}
	view.authProviders = []domain.AuthProvider{
		{ID: "auth-1", Name: "My Google Auth"},
		{ID: "auth-2", Name: "Work Google Auth"},
	}
	view.selectedAuthIndex = 0

	output := view.renderAuthSelect()

	assert.Contains(t, output, "OAuth app")
	assert.Contains(t, output, "My Google Auth")
	assert.Contains(t, output, "Work Google Auth")
}

func TestView_RenderAuthSelect_NilConnector(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewView(s, nil, nil, nil, nil, nil)
	view.connector = nil

	output := view.renderAuthSelect()

	assert.Equal(t, "", output)
}

func TestView_RenderCredentialsInput_NilConnector(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewView(s, nil, nil, nil, nil, nil)
	view.connector = nil

	output := view.renderCredentialsInput()

	assert.Equal(t, "", output)
}

func TestView_GetProviderHint_NilRegistry(t *testing.T) {
	// When registry is nil, should return empty string
	view := NewView(nil, nil, nil, nil, nil, nil)
	view.connector = &domain.ConnectorType{
		ID:           "google-drive",
		ProviderType: domain.ProviderGoogle,
	}

	hint := view.getProviderHint()

	assert.Equal(t, "", hint)
}

func TestView_GetProviderHint_WithRegistry(t *testing.T) {
	// When registry returns a hint, it should be returned
	mockRegistry := &MockConnectorRegistry{
		hints: map[string]string{
			"google-drive": "Visit console.cloud.google.com for OAuth credentials",
		},
	}
	view := NewView(nil, nil, mockRegistry, nil, nil, nil)
	view.connector = &domain.ConnectorType{
		ID:           "google-drive",
		ProviderType: domain.ProviderGoogle,
	}

	hint := view.getProviderHint()

	assert.Contains(t, hint, "console.cloud.google.com")
}

func TestView_GetProviderHint_RegistryReturnsEmpty(t *testing.T) {
	// When registry returns empty, should return empty string
	mockRegistry := &MockConnectorRegistry{}
	view := NewView(nil, nil, mockRegistry, nil, nil, nil)
	view.connector = &domain.ConnectorType{
		ID:           "local-fs",
		ProviderType: domain.ProviderLocal,
	}

	hint := view.getProviderHint()

	assert.Equal(t, "", hint)
}

func TestView_GetProviderHint_NilConnector(t *testing.T) {
	// When connector is nil, should return empty string
	view := NewView(nil, nil, nil, nil, nil, nil)
	view.connector = nil

	hint := view.getProviderHint()

	assert.Equal(t, "", hint)
}

func TestView_RenderProgress(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewView(s, nil, nil, nil, nil, nil)

	view.step = StepSelectConnector
	output := view.renderProgress()
	assert.Contains(t, output, "Connector")

	view.step = StepEnterConfig
	output = view.renderProgress()
	assert.Contains(t, output, "Configure")

	view.step = StepSelectAuth
	output = view.renderProgress()
	assert.Contains(t, output, "Authenticate")

	view.step = StepEnterCredentials
	output = view.renderProgress()
	assert.Contains(t, output, "Authenticate")

	view.step = StepOAuthFlow
	output = view.renderProgress()
	assert.Contains(t, output, "Authenticate")

	view.step = StepComplete
	output = view.renderProgress()
	assert.Contains(t, output, "Done")
}

func TestView_HandleAuthSelect_Navigate_Boundaries(t *testing.T) {
	view := NewView(nil, nil, nil, nil, nil, nil)
	view.authProviders = []domain.AuthProvider{
		{ID: "auth-1"},
	}
	view.selectedAuthIndex = 0

	// Try to go up at top
	msg := tea.KeyMsg{Type: tea.KeyUp}
	view.handleAuthSelect(msg)
	assert.Equal(t, 0, view.selectedAuthIndex)

	// Navigate to "Add new" option
	view.selectedAuthIndex = 1
	msg = tea.KeyMsg{Type: tea.KeyDown}
	view.handleAuthSelect(msg)
	assert.Equal(t, 1, view.selectedAuthIndex) // Can't go past "Add new"
}

func TestView_UpdateFocus(t *testing.T) {
	view := NewView(nil, nil, nil, nil, nil, nil)
	view.connector = &domain.ConnectorType{
		ConfigKeys: []domain.ConfigKey{
			{Key: "field1"},
			{Key: "field2"},
		},
	}
	view.initConfigInputs()
	view.focusIndex = 0

	cmd := view.updateFocus()

	require.NotNil(t, cmd)
}

func TestView_HandleOAuthFlowState_NoKeyHandling(t *testing.T) {
	view := NewView(nil, nil, nil, nil, nil, nil)
	view.step = StepOAuthFlow

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}
	_, cmd := view.Update(msg)

	assert.Nil(t, cmd) // OAuth flow doesn't handle keys except Esc
}

func TestStepConstants(t *testing.T) {
	assert.Equal(t, WizardStep(0), StepSelectConnector)
	assert.Equal(t, WizardStep(1), StepEnterConfig)
	assert.Equal(t, WizardStep(2), StepSelectAuthMethod) // Choose PAT vs OAuth
	assert.Equal(t, WizardStep(3), StepSelectAuth)       // Select existing auth or add new
	assert.Equal(t, WizardStep(4), StepEnterCredentials)
	assert.Equal(t, WizardStep(5), StepOAuthFlow)
	assert.Equal(t, WizardStep(6), StepComplete)
}

func TestView_HandleConfigInput_NoInputs(t *testing.T) {
	view := NewView(nil, nil, nil, nil, nil, nil)
	view.configInputs = []textinput.Model{}

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
	view.handleConfigInput(msg)

	// Should not crash
}

func TestView_HandleConfigInput_InvalidIndex(t *testing.T) {
	view := NewView(nil, nil, nil, nil, nil, nil)
	ti := textinput.New()
	view.configInputs = []textinput.Model{ti}
	view.focusIndex = 10 // Invalid

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
	view.handleConfigInput(msg)

	// Should not crash
}
