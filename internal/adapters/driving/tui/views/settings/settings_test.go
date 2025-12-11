package settings

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/custodia-labs/sercha-cli/internal/adapters/driving/tui/messages"
	"github.com/custodia-labs/sercha-cli/internal/adapters/driving/tui/styles"
	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

// MockSettingsService is a mock implementation of driving.SettingsService.
type MockSettingsService struct {
	mock.Mock
}

func (m *MockSettingsService) Get() (*domain.AppSettings, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.AppSettings), args.Error(1)
}

func (m *MockSettingsService) Save(settings *domain.AppSettings) error {
	args := m.Called(settings)
	return args.Error(0)
}

func (m *MockSettingsService) SetSearchMode(mode domain.SearchMode) error {
	args := m.Called(mode)
	return args.Error(0)
}

func (m *MockSettingsService) SetEmbeddingProvider(provider domain.AIProvider, model, apiKey string) error {
	args := m.Called(provider, model, apiKey)
	return args.Error(0)
}

func (m *MockSettingsService) SetLLMProvider(provider domain.AIProvider, model, apiKey string) error {
	args := m.Called(provider, model, apiKey)
	return args.Error(0)
}

func (m *MockSettingsService) Validate() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockSettingsService) RequiresEmbedding() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockSettingsService) RequiresLLM() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockSettingsService) GetDefaults() domain.AppSettings {
	args := m.Called()
	return args.Get(0).(domain.AppSettings)
}

func (m *MockSettingsService) ValidateEmbeddingConfig() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockSettingsService) ValidateLLMConfig() error {
	args := m.Called()
	return args.Error(0)
}

// Helper function to create test settings.
func testSettings() *domain.AppSettings {
	return &domain.AppSettings{
		Search: domain.SearchSettings{
			Mode: domain.SearchModeTextOnly,
		},
		Embedding: domain.EmbeddingSettings{
			Provider: domain.AIProviderOllama,
			Model:    "nomic-embed-text",
			BaseURL:  "http://localhost:11434",
		},
		LLM: domain.LLMSettings{
			Provider: domain.AIProviderOllama,
			Model:    "llama3.2",
			BaseURL:  "http://localhost:11434",
		},
	}
}

func TestNewView(t *testing.T) {
	s := styles.DefaultStyles()
	mockService := new(MockSettingsService)

	view := NewView(s, mockService)

	require.NotNil(t, view)
	assert.NotNil(t, view.styles)
	assert.Equal(t, mockService, view.settingsService)
	assert.Equal(t, SectionOverview, view.section)
	assert.Equal(t, 0, view.selected)
	assert.Equal(t, 0, view.focusedField)
	assert.NotNil(t, view.embeddingAPIKeyInput)
	assert.NotNil(t, view.llmAPIKeyInput)
}

func TestNewView_NilStyles(t *testing.T) {
	mockService := new(MockSettingsService)

	view := NewView(nil, mockService)

	require.NotNil(t, view)
	// Should create default styles
	assert.NotNil(t, view.styles)
}

func TestView_Init(t *testing.T) {
	mockService := new(MockSettingsService)
	view := NewView(nil, mockService)

	cmd := view.Init()

	require.NotNil(t, cmd)
	// Init should return loadSettings command
}

func TestView_Init_LoadSettings_Success(t *testing.T) {
	mockService := new(MockSettingsService)
	settings := testSettings()
	mockService.On("Get").Return(settings, nil)

	view := NewView(nil, mockService)
	cmd := view.Init()

	require.NotNil(t, cmd)
	result := cmd()
	loaded, ok := result.(messages.SettingsLoaded)
	require.True(t, ok)
	assert.NoError(t, loaded.Err)
	assert.Equal(t, settings, loaded.Settings)
	mockService.AssertExpectations(t)
}

func TestView_Init_LoadSettings_Error(t *testing.T) {
	mockService := new(MockSettingsService)
	expectedErr := fmt.Errorf("failed to load settings")
	mockService.On("Get").Return((*domain.AppSettings)(nil), expectedErr)

	view := NewView(nil, mockService)
	cmd := view.Init()

	require.NotNil(t, cmd)
	result := cmd()
	loaded, ok := result.(messages.SettingsLoaded)
	require.True(t, ok)
	assert.Equal(t, expectedErr, loaded.Err)
	assert.Nil(t, loaded.Settings)
	mockService.AssertExpectations(t)
}

func TestView_Init_NoService(t *testing.T) {
	view := NewView(nil, nil)
	cmd := view.Init()

	require.NotNil(t, cmd)
	result := cmd()
	loaded, ok := result.(messages.SettingsLoaded)
	require.True(t, ok)
	assert.Error(t, loaded.Err)
	assert.Contains(t, loaded.Err.Error(), "settings service not available")
}

func TestView_Update_WindowSize(t *testing.T) {
	mockService := new(MockSettingsService)
	view := NewView(nil, mockService)

	msg := tea.WindowSizeMsg{Width: 120, Height: 60}
	updated, cmd := view.Update(msg)

	assert.Equal(t, view, updated)
	assert.Nil(t, cmd)
	assert.True(t, view.ready)
	assert.Equal(t, 120, view.width)
	assert.Equal(t, 60, view.height)
}

func TestView_Update_SettingsLoaded_Success(t *testing.T) {
	mockService := new(MockSettingsService)
	view := NewView(nil, mockService)
	settings := testSettings()

	msg := messages.SettingsLoaded{
		Settings: settings,
		Err:      nil,
	}
	updated, cmd := view.Update(msg)

	assert.Equal(t, view, updated)
	assert.Nil(t, cmd)
	assert.Equal(t, settings, view.settings)
	assert.NoError(t, view.err)
}

func TestView_Update_SettingsLoaded_Error(t *testing.T) {
	mockService := new(MockSettingsService)
	view := NewView(nil, mockService)
	expectedErr := fmt.Errorf("load failed")

	msg := messages.SettingsLoaded{
		Settings: nil,
		Err:      expectedErr,
	}
	updated, cmd := view.Update(msg)

	assert.Equal(t, view, updated)
	assert.Nil(t, cmd)
	assert.Nil(t, view.settings)
	assert.Equal(t, expectedErr, view.err)
}

func TestView_Update_SettingsSaved_Success(t *testing.T) {
	mockService := new(MockSettingsService)
	settings := testSettings()
	mockService.On("Get").Return(settings, nil)

	view := NewView(nil, mockService)

	msg := messages.SettingsSaved{Err: nil}
	updated, cmd := view.Update(msg)

	assert.Equal(t, view, updated)
	require.NotNil(t, cmd)
	assert.NoError(t, view.err)

	// Should reload settings
	result := cmd()
	loaded, ok := result.(messages.SettingsLoaded)
	require.True(t, ok)
	assert.NoError(t, loaded.Err)
	mockService.AssertExpectations(t)
}

func TestView_Update_SettingsSaved_Error(t *testing.T) {
	mockService := new(MockSettingsService)
	view := NewView(nil, mockService)
	expectedErr := fmt.Errorf("save failed")

	msg := messages.SettingsSaved{Err: expectedErr}
	updated, cmd := view.Update(msg)

	assert.Equal(t, view, updated)
	assert.Nil(t, cmd)
	assert.Equal(t, expectedErr, view.err)
}

func TestView_Update_KeyMsg_Escape_FromOverview(t *testing.T) {
	mockService := new(MockSettingsService)
	view := NewView(nil, mockService)
	view.section = SectionOverview

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	updated, cmd := view.Update(msg)

	assert.Equal(t, view, updated)
	require.NotNil(t, cmd)

	result := cmd()
	changed, ok := result.(messages.ViewChanged)
	require.True(t, ok)
	assert.Equal(t, messages.ViewMenu, changed.View)
}

func TestView_Update_KeyMsg_Escape_FromSubsection(t *testing.T) {
	mockService := new(MockSettingsService)
	view := NewView(nil, mockService)
	view.section = SectionSearchMode
	view.selected = 2

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	updated, cmd := view.Update(msg)

	assert.Equal(t, view, updated)
	assert.Nil(t, cmd)
	assert.Equal(t, SectionOverview, view.section)
	assert.Equal(t, 0, view.selected)
}

func TestView_Update_KeyMsg_Overview_NavigateDown(t *testing.T) {
	mockService := new(MockSettingsService)
	view := NewView(nil, mockService)
	view.section = SectionOverview
	view.selected = 0

	// Test down key
	msg := tea.KeyMsg{Type: tea.KeyDown}
	view.Update(msg)
	assert.Equal(t, 1, view.selected)

	// Test j key
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	view.Update(msg)
	assert.Equal(t, 2, view.selected)

	// Test boundary - can't go past last item (3 items: 0-2)
	view.Update(msg)
	assert.Equal(t, 2, view.selected)
}

func TestView_Update_KeyMsg_Overview_NavigateUp(t *testing.T) {
	mockService := new(MockSettingsService)
	view := NewView(nil, mockService)
	view.section = SectionOverview
	view.selected = 2

	// Test up key
	msg := tea.KeyMsg{Type: tea.KeyUp}
	view.Update(msg)
	assert.Equal(t, 1, view.selected)

	// Test k key
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	view.Update(msg)
	assert.Equal(t, 0, view.selected)

	// Test boundary - can't go before first item
	view.Update(msg)
	assert.Equal(t, 0, view.selected)
}

func TestView_Update_KeyMsg_Overview_Enter_SearchMode(t *testing.T) {
	mockService := new(MockSettingsService)
	view := NewView(nil, mockService)
	view.section = SectionOverview
	view.selected = 0
	view.settings = testSettings()

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updated, cmd := view.Update(msg)

	assert.Equal(t, view, updated)
	assert.Nil(t, cmd)
	assert.Equal(t, SectionSearchMode, view.section)
	assert.Equal(t, 0, view.selected) // Index of current search mode
}

func TestView_Update_KeyMsg_Overview_Enter_Embedding(t *testing.T) {
	mockService := new(MockSettingsService)
	view := NewView(nil, mockService)
	view.section = SectionOverview
	view.selected = 1
	view.settings = testSettings()

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updated, cmd := view.Update(msg)

	assert.Equal(t, view, updated)
	assert.Nil(t, cmd)
	assert.Equal(t, SectionEmbedding, view.section)
}

func TestView_Update_KeyMsg_Overview_Enter_LLM(t *testing.T) {
	mockService := new(MockSettingsService)
	view := NewView(nil, mockService)
	view.section = SectionOverview
	view.selected = 2
	view.settings = testSettings()

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updated, cmd := view.Update(msg)

	assert.Equal(t, view, updated)
	assert.Nil(t, cmd)
	assert.Equal(t, SectionLLM, view.section)
}

func TestView_Update_KeyMsg_SearchMode_Navigate(t *testing.T) {
	mockService := new(MockSettingsService)
	view := NewView(nil, mockService)
	view.section = SectionSearchMode
	view.selected = 0

	// Navigate down
	msg := tea.KeyMsg{Type: tea.KeyDown}
	view.Update(msg)
	assert.Equal(t, 1, view.selected)

	// Navigate up
	msg = tea.KeyMsg{Type: tea.KeyUp}
	view.Update(msg)
	assert.Equal(t, 0, view.selected)
}

func TestView_Update_KeyMsg_SearchMode_Enter_Success(t *testing.T) {
	mockService := new(MockSettingsService)
	mockService.On("SetSearchMode", domain.SearchModeHybrid).Return(nil)

	view := NewView(nil, mockService)
	view.section = SectionSearchMode
	view.selected = 1 // Hybrid mode

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updated, cmd := view.Update(msg)

	assert.Equal(t, view, updated)
	require.NotNil(t, cmd)

	result := cmd()
	saved, ok := result.(messages.SettingsSaved)
	require.True(t, ok)
	assert.NoError(t, saved.Err)
	assert.Equal(t, SectionOverview, view.section)
	assert.Equal(t, 0, view.selected)
	mockService.AssertExpectations(t)
}

func TestView_Update_KeyMsg_SearchMode_Enter_Error(t *testing.T) {
	mockService := new(MockSettingsService)
	expectedErr := fmt.Errorf("failed to set mode")
	mockService.On("SetSearchMode", domain.SearchModeHybrid).Return(expectedErr)

	view := NewView(nil, mockService)
	view.section = SectionSearchMode
	view.selected = 1

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updated, cmd := view.Update(msg)

	assert.Equal(t, view, updated)
	require.NotNil(t, cmd)

	result := cmd()
	saved, ok := result.(messages.SettingsSaved)
	require.True(t, ok)
	assert.Equal(t, expectedErr, saved.Err)
	mockService.AssertExpectations(t)
}

func TestView_Update_KeyMsg_SearchMode_NoService(t *testing.T) {
	view := NewView(nil, nil)
	view.section = SectionSearchMode
	view.selected = 0

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updated, cmd := view.Update(msg)

	assert.Equal(t, view, updated)
	require.NotNil(t, cmd)

	result := cmd()
	saved, ok := result.(messages.SettingsSaved)
	require.True(t, ok)
	assert.Error(t, saved.Err)
	assert.Contains(t, saved.Err.Error(), "settings service not available")
}

func TestView_Update_KeyMsg_Embedding_Navigate(t *testing.T) {
	mockService := new(MockSettingsService)
	view := NewView(nil, mockService)
	view.section = SectionEmbedding
	view.selected = 0

	// Navigate down
	msg := tea.KeyMsg{Type: tea.KeyDown}
	view.Update(msg)
	assert.Equal(t, 1, view.selected)

	// Navigate up
	msg = tea.KeyMsg{Type: tea.KeyUp}
	view.Update(msg)
	assert.Equal(t, 0, view.selected)
}

func TestView_Update_KeyMsg_Embedding_Enter_NoAPIKey_Success(t *testing.T) {
	mockService := new(MockSettingsService)
	// Ollama doesn't require API key
	mockService.On("SetEmbeddingProvider", domain.AIProviderOllama, "nomic-embed-text", "").Return(nil)

	view := NewView(nil, mockService)
	view.section = SectionEmbedding
	view.selected = 0 // Ollama

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updated, cmd := view.Update(msg)

	assert.Equal(t, view, updated)
	require.NotNil(t, cmd)

	result := cmd()
	saved, ok := result.(messages.SettingsSaved)
	require.True(t, ok)
	assert.NoError(t, saved.Err)
	mockService.AssertExpectations(t)
}

func TestView_Update_KeyMsg_Embedding_Enter_RequiresAPIKey_FocusInput(t *testing.T) {
	mockService := new(MockSettingsService)
	view := NewView(nil, mockService)
	view.section = SectionEmbedding
	view.selected = 1 // OpenAI (requires API key)

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updated, cmd := view.Update(msg)

	assert.Equal(t, view, updated)
	require.NotNil(t, cmd) // Focus command
	assert.Equal(t, 1, view.focusedField)
}

func TestView_Update_KeyMsg_Embedding_Tab_ToAPIKeyInput(t *testing.T) {
	mockService := new(MockSettingsService)
	view := NewView(nil, mockService)
	view.section = SectionEmbedding
	view.selected = 1 // OpenAI

	msg := tea.KeyMsg{Type: tea.KeyTab}
	updated, cmd := view.Update(msg)

	assert.Equal(t, view, updated)
	require.NotNil(t, cmd) // Focus command
	assert.Equal(t, 1, view.focusedField)
}

func TestView_Update_KeyMsg_Embedding_Tab_FromAPIKeyInput(t *testing.T) {
	mockService := new(MockSettingsService)
	view := NewView(nil, mockService)
	view.section = SectionEmbedding
	view.selected = 1
	view.focusedField = 1

	msg := tea.KeyMsg{Type: tea.KeyTab}
	updated, cmd := view.Update(msg)

	assert.Equal(t, view, updated)
	assert.Nil(t, cmd)
	assert.Equal(t, 0, view.focusedField)
}

func TestView_Update_KeyMsg_Embedding_APIKeyInput_Enter_Success(t *testing.T) {
	mockService := new(MockSettingsService)
	mockService.On("SetEmbeddingProvider", domain.AIProviderOpenAI, "text-embedding-3-small", "test-key").Return(nil)

	view := NewView(nil, mockService)
	view.section = SectionEmbedding
	view.selected = 1 // OpenAI
	view.focusedField = 1
	view.embeddingAPIKeyInput.SetValue("test-key")

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updated, cmd := view.Update(msg)

	assert.Equal(t, view, updated)
	require.NotNil(t, cmd)

	result := cmd()
	saved, ok := result.(messages.SettingsSaved)
	require.True(t, ok)
	assert.NoError(t, saved.Err)
	assert.Equal(t, SectionOverview, view.section)
	assert.Equal(t, 0, view.focusedField)
	assert.Equal(t, "", view.embeddingAPIKeyInput.Value())
	mockService.AssertExpectations(t)
}

func TestView_Update_KeyMsg_LLM_Navigate(t *testing.T) {
	mockService := new(MockSettingsService)
	view := NewView(nil, mockService)
	view.section = SectionLLM
	view.selected = 0

	// Navigate down
	msg := tea.KeyMsg{Type: tea.KeyDown}
	view.Update(msg)
	assert.Equal(t, 1, view.selected)

	// Navigate up
	msg = tea.KeyMsg{Type: tea.KeyUp}
	view.Update(msg)
	assert.Equal(t, 0, view.selected)
}

func TestView_Update_KeyMsg_LLM_Enter_NoAPIKey_Success(t *testing.T) {
	mockService := new(MockSettingsService)
	mockService.On("SetLLMProvider", domain.AIProviderOllama, "llama3.2", "").Return(nil)

	view := NewView(nil, mockService)
	view.section = SectionLLM
	view.selected = 0 // Ollama

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updated, cmd := view.Update(msg)

	assert.Equal(t, view, updated)
	require.NotNil(t, cmd)

	result := cmd()
	saved, ok := result.(messages.SettingsSaved)
	require.True(t, ok)
	assert.NoError(t, saved.Err)
	mockService.AssertExpectations(t)
}

func TestView_Update_KeyMsg_LLM_Enter_RequiresAPIKey_FocusInput(t *testing.T) {
	mockService := new(MockSettingsService)
	view := NewView(nil, mockService)
	view.section = SectionLLM
	view.selected = 1 // OpenAI

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updated, cmd := view.Update(msg)

	assert.Equal(t, view, updated)
	require.NotNil(t, cmd) // Focus command
	assert.Equal(t, 1, view.focusedField)
}

func TestView_Update_KeyMsg_LLM_APIKeyInput_Enter_Success(t *testing.T) {
	mockService := new(MockSettingsService)
	mockService.On("SetLLMProvider", domain.AIProviderOpenAI, "gpt-4o-mini", "test-llm-key").Return(nil)

	view := NewView(nil, mockService)
	view.section = SectionLLM
	view.selected = 1 // OpenAI
	view.focusedField = 1
	view.llmAPIKeyInput.SetValue("test-llm-key")

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updated, cmd := view.Update(msg)

	assert.Equal(t, view, updated)
	require.NotNil(t, cmd)

	result := cmd()
	saved, ok := result.(messages.SettingsSaved)
	require.True(t, ok)
	assert.NoError(t, saved.Err)
	assert.Equal(t, SectionOverview, view.section)
	assert.Equal(t, 0, view.focusedField)
	assert.Equal(t, "", view.llmAPIKeyInput.Value())
	mockService.AssertExpectations(t)
}

func TestView_View_NoSettings_LoadingState(t *testing.T) {
	mockService := new(MockSettingsService)
	view := NewView(nil, mockService)
	view.settings = nil

	output := view.View()

	assert.Contains(t, output, "Settings")
	assert.Contains(t, output, "Loading settings...")
}

func TestView_View_WithError(t *testing.T) {
	mockService := new(MockSettingsService)
	mockService.On("Validate").Return(nil)

	view := NewView(nil, mockService)
	view.err = fmt.Errorf("test error")
	view.settings = testSettings()

	output := view.View()

	assert.Contains(t, output, "Error: test error")
	mockService.AssertExpectations(t)
}

func TestView_View_Overview(t *testing.T) {
	mockService := new(MockSettingsService)
	mockService.On("Validate").Return(nil)

	view := NewView(nil, mockService)
	view.section = SectionOverview
	view.settings = testSettings()
	view.ready = true

	output := view.View()

	assert.Contains(t, output, "Settings")
	assert.Contains(t, output, "Search Mode")
	assert.Contains(t, output, "Embedding Provider")
	assert.Contains(t, output, "LLM Provider")
	assert.Contains(t, output, "Configuration is valid")
	assert.Contains(t, output, "[j/k] navigate")
	mockService.AssertExpectations(t)
}

func TestView_View_Overview_ValidationError(t *testing.T) {
	mockService := new(MockSettingsService)
	mockService.On("Validate").Return(fmt.Errorf("invalid configuration"))

	view := NewView(nil, mockService)
	view.section = SectionOverview
	view.settings = testSettings()

	output := view.View()

	assert.Contains(t, output, "Warning: invalid configuration")
	mockService.AssertExpectations(t)
}

func TestView_View_SearchModeSelect(t *testing.T) {
	mockService := new(MockSettingsService)
	view := NewView(nil, mockService)
	view.section = SectionSearchMode
	view.settings = testSettings()
	view.selected = 1

	output := view.View()

	assert.Contains(t, output, "Select Search Mode")
	assert.Contains(t, output, "Text Only")
	assert.Contains(t, output, "Hybrid")
	assert.Contains(t, output, "(current)") // Current mode
	assert.Contains(t, output, "[j/k] navigate")
	assert.Contains(t, output, "[enter] select")
}

func TestView_View_EmbeddingSelect(t *testing.T) {
	mockService := new(MockSettingsService)
	view := NewView(nil, mockService)
	view.section = SectionEmbedding
	view.settings = testSettings()
	view.selected = 0

	output := view.View()

	assert.Contains(t, output, "Select Embedding Provider")
	assert.Contains(t, output, "Ollama")
	assert.Contains(t, output, "OpenAI")
	assert.Contains(t, output, "Model:")
	assert.Contains(t, output, "[j/k] navigate")
}

func TestView_View_EmbeddingSelect_WithAPIKeyInput(t *testing.T) {
	mockService := new(MockSettingsService)
	view := NewView(nil, mockService)
	view.section = SectionEmbedding
	view.settings = testSettings()
	view.selected = 1 // OpenAI (requires API key)

	output := view.View()

	assert.Contains(t, output, "Select Embedding Provider")
	assert.Contains(t, output, "API Key:")
	assert.Contains(t, output, "[tab] API key")
}

func TestView_View_LLMSelect(t *testing.T) {
	mockService := new(MockSettingsService)
	view := NewView(nil, mockService)
	view.section = SectionLLM
	view.settings = testSettings()
	view.selected = 0

	output := view.View()

	assert.Contains(t, output, "Select LLM Provider")
	assert.Contains(t, output, "Ollama")
	assert.Contains(t, output, "OpenAI")
	assert.Contains(t, output, "Anthropic")
	assert.Contains(t, output, "Model:")
}

func TestView_View_LLMSelect_WithAPIKeyInput(t *testing.T) {
	mockService := new(MockSettingsService)
	view := NewView(nil, mockService)
	view.section = SectionLLM
	view.settings = testSettings()
	view.selected = 1 // OpenAI (requires API key)

	output := view.View()

	assert.Contains(t, output, "Select LLM Provider")
	assert.Contains(t, output, "API Key:")
}

func TestView_SetDimensions(t *testing.T) {
	mockService := new(MockSettingsService)
	view := NewView(nil, mockService)
	view.ready = false

	view.SetDimensions(120, 60)

	assert.Equal(t, 120, view.width)
	assert.Equal(t, 60, view.height)
	assert.True(t, view.ready)
}

func TestView_Reset(t *testing.T) {
	mockService := new(MockSettingsService)
	view := NewView(nil, mockService)

	// Set some state
	view.section = SectionSearchMode
	view.selected = 2
	view.focusedField = 1
	view.err = fmt.Errorf("test error")
	view.embeddingAPIKeyInput.SetValue("test-key")
	view.llmAPIKeyInput.SetValue("test-llm-key")

	view.Reset()

	assert.Equal(t, SectionOverview, view.section)
	assert.Equal(t, 0, view.selected)
	assert.Equal(t, 0, view.focusedField)
	assert.NoError(t, view.err)
	assert.Equal(t, "", view.embeddingAPIKeyInput.Value())
	assert.Equal(t, "", view.llmAPIKeyInput.Value())
}

func TestView_GetSearchModeIndex(t *testing.T) {
	mockService := new(MockSettingsService)
	view := NewView(nil, mockService)
	view.settings = testSettings()
	view.settings.Search.Mode = domain.SearchModeHybrid

	index := view.getSearchModeIndex()

	assert.Equal(t, 1, index) // Hybrid is second in list
}

func TestView_GetSearchModeIndex_NilSettings(t *testing.T) {
	mockService := new(MockSettingsService)
	view := NewView(nil, mockService)
	view.settings = nil

	index := view.getSearchModeIndex()

	assert.Equal(t, 0, index)
}

func TestView_GetEmbeddingProviderIndex(t *testing.T) {
	mockService := new(MockSettingsService)
	view := NewView(nil, mockService)
	view.settings = testSettings()
	view.settings.Embedding.Provider = domain.AIProviderOpenAI

	index := view.getEmbeddingProviderIndex()

	assert.Equal(t, 1, index)
}

func TestView_GetLLMProviderIndex(t *testing.T) {
	mockService := new(MockSettingsService)
	view := NewView(nil, mockService)
	view.settings = testSettings()
	view.settings.LLM.Provider = domain.AIProviderAnthropic

	index := view.getLLMProviderIndex()

	assert.Equal(t, 2, index)
}

func TestView_GetEmbeddingStatus_Configured(t *testing.T) {
	mockService := new(MockSettingsService)
	view := NewView(nil, mockService)
	view.settings = testSettings()
	// Ollama is configured by default (no API key needed)

	status := view.getEmbeddingStatus()

	assert.Contains(t, status, "configured")
}

func TestView_GetEmbeddingStatus_NeedsAPIKey(t *testing.T) {
	mockService := new(MockSettingsService)
	view := NewView(nil, mockService)
	view.settings = testSettings()
	view.settings.Embedding.Provider = domain.AIProviderOpenAI
	view.settings.Embedding.APIKey = "" // Missing API key

	status := view.getEmbeddingStatus()

	assert.Contains(t, status, "needs API key")
}

func TestView_GetLLMStatus_Configured(t *testing.T) {
	mockService := new(MockSettingsService)
	view := NewView(nil, mockService)
	view.settings = testSettings()

	status := view.getLLMStatus()

	assert.Contains(t, status, "configured")
}

func TestView_GetLLMStatus_NeedsAPIKey(t *testing.T) {
	mockService := new(MockSettingsService)
	view := NewView(nil, mockService)
	view.settings = testSettings()
	view.settings.LLM.Provider = domain.AIProviderOpenAI
	view.settings.LLM.APIKey = "" // Missing API key

	status := view.getLLMStatus()

	assert.Contains(t, status, "needs API key")
}

func TestView_RenderHelp_Overview(t *testing.T) {
	mockService := new(MockSettingsService)
	view := NewView(nil, mockService)
	view.section = SectionOverview

	help := view.renderHelp()

	assert.Contains(t, help, "[j/k] navigate")
	assert.Contains(t, help, "[enter] edit")
	assert.Contains(t, help, "[esc] back")
}

func TestView_RenderHelp_SearchMode(t *testing.T) {
	mockService := new(MockSettingsService)
	view := NewView(nil, mockService)
	view.section = SectionSearchMode

	help := view.renderHelp()

	assert.Contains(t, help, "[j/k] navigate")
	assert.Contains(t, help, "[enter] select")
	assert.Contains(t, help, "[esc] back")
}

func TestView_RenderHelp_Embedding_ProviderFocused(t *testing.T) {
	mockService := new(MockSettingsService)
	view := NewView(nil, mockService)
	view.section = SectionEmbedding
	view.focusedField = 0

	help := view.renderHelp()

	assert.Contains(t, help, "[j/k] navigate")
	assert.Contains(t, help, "[tab] API key")
	assert.Contains(t, help, "[enter] select")
}

func TestView_RenderHelp_Embedding_APIKeyFocused(t *testing.T) {
	mockService := new(MockSettingsService)
	view := NewView(nil, mockService)
	view.section = SectionEmbedding
	view.focusedField = 1

	help := view.renderHelp()

	assert.Contains(t, help, "[tab] back to list")
	assert.Contains(t, help, "[enter] save")
	assert.Contains(t, help, "[esc] back")
}

// Test edge cases for navigation boundaries.
func TestView_Update_KeyMsg_SearchMode_Navigate_Boundaries(t *testing.T) {
	mockService := new(MockSettingsService)
	view := NewView(nil, mockService)
	view.section = SectionSearchMode
	modes := domain.AllSearchModes()

	// Navigate to last item
	view.selected = len(modes) - 1
	msg := tea.KeyMsg{Type: tea.KeyDown}
	view.Update(msg)
	assert.Equal(t, len(modes)-1, view.selected) // Can't go past last

	// Navigate to first item
	view.selected = 0
	msg = tea.KeyMsg{Type: tea.KeyUp}
	view.Update(msg)
	assert.Equal(t, 0, view.selected) // Can't go before first
}

// Test that text input updates are forwarded to the textinput model.
func TestView_Update_KeyMsg_Embedding_APIKeyInput_TextInput(t *testing.T) {
	mockService := new(MockSettingsService)
	view := NewView(nil, mockService)
	view.section = SectionEmbedding
	view.selected = 1
	view.focusedField = 1

	// Simulate typing a character
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
	updated, _ := view.Update(msg)

	// Should update the view (command may or may not be nil depending on textinput behavior)
	assert.Equal(t, view, updated)
	// The textinput should have processed the input
	assert.Equal(t, 1, view.focusedField)
}

// Test that shift+tab works the same as tab when in API key input.
func TestView_Update_KeyMsg_Embedding_ShiftTab_FromAPIKeyInput(t *testing.T) {
	mockService := new(MockSettingsService)
	view := NewView(nil, mockService)
	view.section = SectionEmbedding
	view.selected = 1
	view.focusedField = 1

	msg := tea.KeyMsg{Type: tea.KeyShiftTab}
	updated, cmd := view.Update(msg)

	assert.Equal(t, view, updated)
	assert.Nil(t, cmd)
	assert.Equal(t, 0, view.focusedField)
}

// Test section constants.
func TestSectionConstants(t *testing.T) {
	assert.Equal(t, Section(0), SectionOverview)
	assert.Equal(t, Section(1), SectionSearchMode)
	assert.Equal(t, Section(2), SectionEmbedding)
	assert.Equal(t, Section(3), SectionLLM)
}

// Test that tab on provider not requiring API key does nothing.
func TestView_Update_KeyMsg_Embedding_Tab_NoAPIKeyNeeded(t *testing.T) {
	mockService := new(MockSettingsService)
	view := NewView(nil, mockService)
	view.section = SectionEmbedding
	view.selected = 0 // Ollama (no API key needed)

	msg := tea.KeyMsg{Type: tea.KeyTab}
	updated, cmd := view.Update(msg)

	assert.Equal(t, view, updated)
	assert.Nil(t, cmd)
	assert.Equal(t, 0, view.focusedField) // Should remain on provider
}

// Test enter with invalid selection index (edge case).
func TestView_Update_KeyMsg_SearchMode_Enter_InvalidIndex(t *testing.T) {
	mockService := new(MockSettingsService)
	view := NewView(nil, mockService)
	view.section = SectionSearchMode
	view.selected = 999 // Invalid index

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updated, cmd := view.Update(msg)

	assert.Equal(t, view, updated)
	assert.Nil(t, cmd) // Should not generate a command
}

// Test all sections properly handle unknown messages.
func TestView_Update_KeyMsg_UnknownKey(t *testing.T) {
	mockService := new(MockSettingsService)
	view := NewView(nil, mockService)

	sections := []Section{
		SectionOverview,
		SectionSearchMode,
		SectionEmbedding,
		SectionLLM,
	}

	for _, section := range sections {
		view.section = section
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}} // Unknown key
		updated, cmd := view.Update(msg)

		assert.Equal(t, view, updated)
		assert.Nil(t, cmd)
	}
}

// Test LLM section with same patterns as Embedding.
func TestView_Update_KeyMsg_LLM_Tab_ToAPIKeyInput(t *testing.T) {
	mockService := new(MockSettingsService)
	view := NewView(nil, mockService)
	view.section = SectionLLM
	view.selected = 1 // OpenAI

	msg := tea.KeyMsg{Type: tea.KeyTab}
	updated, cmd := view.Update(msg)

	assert.Equal(t, view, updated)
	require.NotNil(t, cmd) // Focus command
	assert.Equal(t, 1, view.focusedField)
}

func TestView_Update_KeyMsg_LLM_Tab_FromAPIKeyInput(t *testing.T) {
	mockService := new(MockSettingsService)
	view := NewView(nil, mockService)
	view.section = SectionLLM
	view.selected = 1
	view.focusedField = 1

	msg := tea.KeyMsg{Type: tea.KeyTab}
	updated, cmd := view.Update(msg)

	assert.Equal(t, view, updated)
	assert.Nil(t, cmd)
	assert.Equal(t, 0, view.focusedField)
}

// Test overview with nil service for validation.
func TestView_View_Overview_NilService(t *testing.T) {
	view := NewView(nil, nil)
	view.section = SectionOverview
	view.settings = testSettings()

	output := view.View()

	// Should not crash, but won't show validation status
	assert.Contains(t, output, "Settings")
	assert.Contains(t, output, "Search Mode")
}

// Test that textinput models have correct configuration.
func TestNewView_TextInputConfiguration(t *testing.T) {
	mockService := new(MockSettingsService)
	view := NewView(nil, mockService)

	// Check embedding API key input
	assert.Equal(t, "Enter API key", view.embeddingAPIKeyInput.Placeholder)
	assert.Equal(t, 256, view.embeddingAPIKeyInput.CharLimit)

	// Check LLM API key input
	assert.Equal(t, "Enter API key", view.llmAPIKeyInput.Placeholder)
	assert.Equal(t, 256, view.llmAPIKeyInput.CharLimit)
}

// Test Embedding navigation boundaries.
func TestView_Update_KeyMsg_Embedding_Navigate_Boundaries(t *testing.T) {
	mockService := new(MockSettingsService)
	view := NewView(nil, mockService)
	view.section = SectionEmbedding
	providers := domain.AllEmbeddingProviders()

	// Navigate to last item
	view.selected = len(providers) - 1
	msg := tea.KeyMsg{Type: tea.KeyDown}
	view.Update(msg)
	assert.Equal(t, len(providers)-1, view.selected)

	// Navigate to first item
	view.selected = 0
	msg = tea.KeyMsg{Type: tea.KeyUp}
	view.Update(msg)
	assert.Equal(t, 0, view.selected)
}

// Test Embedding enter with error.
func TestView_Update_KeyMsg_Embedding_Enter_Error(t *testing.T) {
	mockService := new(MockSettingsService)
	expectedErr := fmt.Errorf("failed to set embedding provider")
	mockService.On("SetEmbeddingProvider", domain.AIProviderOllama, "nomic-embed-text", "").Return(expectedErr)

	view := NewView(nil, mockService)
	view.section = SectionEmbedding
	view.selected = 0

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updated, cmd := view.Update(msg)

	assert.Equal(t, view, updated)
	require.NotNil(t, cmd)

	result := cmd()
	saved, ok := result.(messages.SettingsSaved)
	require.True(t, ok)
	assert.Equal(t, expectedErr, saved.Err)
	mockService.AssertExpectations(t)
}

// Test Embedding with invalid selection index.
func TestView_Update_KeyMsg_Embedding_Enter_InvalidIndex(t *testing.T) {
	mockService := new(MockSettingsService)
	view := NewView(nil, mockService)
	view.section = SectionEmbedding
	view.selected = 999

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updated, cmd := view.Update(msg)

	assert.Equal(t, view, updated)
	assert.Nil(t, cmd)
}

// Test LLM navigation boundaries.
func TestView_Update_KeyMsg_LLM_Navigate_Boundaries(t *testing.T) {
	mockService := new(MockSettingsService)
	view := NewView(nil, mockService)
	view.section = SectionLLM
	providers := domain.AllLLMProviders()

	// Navigate to last item
	view.selected = len(providers) - 1
	msg := tea.KeyMsg{Type: tea.KeyDown}
	view.Update(msg)
	assert.Equal(t, len(providers)-1, view.selected)

	// Navigate to first item
	view.selected = 0
	msg = tea.KeyMsg{Type: tea.KeyUp}
	view.Update(msg)
	assert.Equal(t, 0, view.selected)
}

// Test LLM enter with error.
func TestView_Update_KeyMsg_LLM_Enter_Error(t *testing.T) {
	mockService := new(MockSettingsService)
	expectedErr := fmt.Errorf("failed to set LLM provider")
	mockService.On("SetLLMProvider", domain.AIProviderOllama, "llama3.2", "").Return(expectedErr)

	view := NewView(nil, mockService)
	view.section = SectionLLM
	view.selected = 0

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updated, cmd := view.Update(msg)

	assert.Equal(t, view, updated)
	require.NotNil(t, cmd)

	result := cmd()
	saved, ok := result.(messages.SettingsSaved)
	require.True(t, ok)
	assert.Equal(t, expectedErr, saved.Err)
	mockService.AssertExpectations(t)
}

// Test LLM with invalid selection index.
func TestView_Update_KeyMsg_LLM_Enter_InvalidIndex(t *testing.T) {
	mockService := new(MockSettingsService)
	view := NewView(nil, mockService)
	view.section = SectionLLM
	view.selected = 999

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updated, cmd := view.Update(msg)

	assert.Equal(t, view, updated)
	assert.Nil(t, cmd)
}

// Test LLM tab on provider not requiring API key.
func TestView_Update_KeyMsg_LLM_Tab_NoAPIKeyNeeded(t *testing.T) {
	mockService := new(MockSettingsService)
	view := NewView(nil, mockService)
	view.section = SectionLLM
	view.selected = 0 // Ollama

	msg := tea.KeyMsg{Type: tea.KeyTab}
	updated, cmd := view.Update(msg)

	assert.Equal(t, view, updated)
	assert.Nil(t, cmd)
	assert.Equal(t, 0, view.focusedField)
}

// Test Embedding API key input with invalid index.
func TestView_Update_KeyMsg_Embedding_APIKeyInput_Enter_InvalidIndex(t *testing.T) {
	mockService := new(MockSettingsService)
	view := NewView(nil, mockService)
	view.section = SectionEmbedding
	view.selected = 999
	view.focusedField = 1

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updated, cmd := view.Update(msg)

	assert.Equal(t, view, updated)
	assert.Nil(t, cmd)
}

// Test LLM API key input with invalid index.
func TestView_Update_KeyMsg_LLM_APIKeyInput_Enter_InvalidIndex(t *testing.T) {
	mockService := new(MockSettingsService)
	view := NewView(nil, mockService)
	view.section = SectionLLM
	view.selected = 999
	view.focusedField = 1

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updated, cmd := view.Update(msg)

	assert.Equal(t, view, updated)
	assert.Nil(t, cmd)
}

// Test renderHelp for LLM section with provider focused.
func TestView_RenderHelp_LLM_ProviderFocused(t *testing.T) {
	mockService := new(MockSettingsService)
	view := NewView(nil, mockService)
	view.section = SectionLLM
	view.focusedField = 0

	help := view.renderHelp()

	assert.Contains(t, help, "[j/k] navigate")
	assert.Contains(t, help, "[tab] API key")
	assert.Contains(t, help, "[enter] select")
}

// Test renderHelp for LLM section with API key focused.
func TestView_RenderHelp_LLM_APIKeyFocused(t *testing.T) {
	mockService := new(MockSettingsService)
	view := NewView(nil, mockService)
	view.section = SectionLLM
	view.focusedField = 1

	help := view.renderHelp()

	assert.Contains(t, help, "[tab] back to list")
	assert.Contains(t, help, "[enter] save")
	assert.Contains(t, help, "[esc] back")
}

// Test getSearchModeIndex with unknown mode.
func TestView_GetSearchModeIndex_UnknownMode(t *testing.T) {
	mockService := new(MockSettingsService)
	view := NewView(nil, mockService)
	view.settings = testSettings()
	view.settings.Search.Mode = domain.SearchMode("unknown")

	index := view.getSearchModeIndex()

	assert.Equal(t, 0, index) // Should default to 0 for unknown
}

// Test getEmbeddingProviderIndex with nil settings.
func TestView_GetEmbeddingProviderIndex_NilSettings(t *testing.T) {
	mockService := new(MockSettingsService)
	view := NewView(nil, mockService)
	view.settings = nil

	index := view.getEmbeddingProviderIndex()

	assert.Equal(t, 0, index)
}

// Test getEmbeddingProviderIndex with unknown provider.
func TestView_GetEmbeddingProviderIndex_UnknownProvider(t *testing.T) {
	mockService := new(MockSettingsService)
	view := NewView(nil, mockService)
	view.settings = testSettings()
	view.settings.Embedding.Provider = domain.AIProvider("unknown")

	index := view.getEmbeddingProviderIndex()

	assert.Equal(t, 0, index)
}

// Test getLLMProviderIndex with nil settings.
func TestView_GetLLMProviderIndex_NilSettings(t *testing.T) {
	mockService := new(MockSettingsService)
	view := NewView(nil, mockService)
	view.settings = nil

	index := view.getLLMProviderIndex()

	assert.Equal(t, 0, index)
}

// Test getLLMProviderIndex with unknown provider.
func TestView_GetLLMProviderIndex_UnknownProvider(t *testing.T) {
	mockService := new(MockSettingsService)
	view := NewView(nil, mockService)
	view.settings = testSettings()
	view.settings.LLM.Provider = domain.AIProvider("unknown")

	index := view.getLLMProviderIndex()

	assert.Equal(t, 0, index)
}

// Test LLM text input handling.
func TestView_Update_KeyMsg_LLM_APIKeyInput_TextInput(t *testing.T) {
	mockService := new(MockSettingsService)
	view := NewView(nil, mockService)
	view.section = SectionLLM
	view.selected = 1
	view.focusedField = 1

	// Simulate typing a character
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}}
	updated, _ := view.Update(msg)

	assert.Equal(t, view, updated)
	assert.Equal(t, 1, view.focusedField)
}

// Test LLM shift+tab from API key input.
func TestView_Update_KeyMsg_LLM_ShiftTab_FromAPIKeyInput(t *testing.T) {
	mockService := new(MockSettingsService)
	view := NewView(nil, mockService)
	view.section = SectionLLM
	view.selected = 1
	view.focusedField = 1

	msg := tea.KeyMsg{Type: tea.KeyShiftTab}
	updated, cmd := view.Update(msg)

	assert.Equal(t, view, updated)
	assert.Nil(t, cmd)
	assert.Equal(t, 0, view.focusedField)
}

// Test Embedding no service.
func TestView_Update_KeyMsg_Embedding_NoService(t *testing.T) {
	view := NewView(nil, nil)
	view.section = SectionEmbedding
	view.selected = 0

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updated, cmd := view.Update(msg)

	assert.Equal(t, view, updated)
	require.NotNil(t, cmd)

	result := cmd()
	saved, ok := result.(messages.SettingsSaved)
	require.True(t, ok)
	assert.Error(t, saved.Err)
	assert.Contains(t, saved.Err.Error(), "settings service not available")
}

// Test LLM no service.
func TestView_Update_KeyMsg_LLM_NoService(t *testing.T) {
	view := NewView(nil, nil)
	view.section = SectionLLM
	view.selected = 0

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updated, cmd := view.Update(msg)

	assert.Equal(t, view, updated)
	require.NotNil(t, cmd)

	result := cmd()
	saved, ok := result.(messages.SettingsSaved)
	require.True(t, ok)
	assert.Error(t, saved.Err)
	assert.Contains(t, saved.Err.Error(), "settings service not available")
}
