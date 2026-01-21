// Package settings provides the settings configuration view for the TUI.
package settings

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/custodia-labs/sercha-cli/internal/adapters/driving/tui/messages"
	"github.com/custodia-labs/sercha-cli/internal/adapters/driving/tui/styles"
	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driving"
)

// Section tracks which settings section is active.
type Section int

const (
	SectionOverview Section = iota
	SectionSearchMode
	SectionEmbedding
	SectionLLM
)

// Key constants for key handling.
const (
	keyDown  = "down"
	keyEnter = "enter"
	keyTab   = "tab"
)

// View is the settings configuration view.
type View struct {
	styles          *styles.Styles
	settingsService driving.SettingsService

	// Current settings
	settings *domain.AppSettings
	err      error

	// Navigation state
	section      Section
	selected     int // selection within current section
	focusedField int // for text input focus

	// Text inputs for API keys
	embeddingAPIKeyInput textinput.Model
	llmAPIKeyInput       textinput.Model

	// Dimensions
	width  int
	height int
	ready  bool
}

// NewView creates a new settings view.
func NewView(s *styles.Styles, settingsService driving.SettingsService) *View {
	if s == nil {
		s = styles.DefaultStyles()
	}

	embeddingAPIKeyInput := textinput.New()
	embeddingAPIKeyInput.Placeholder = "Enter API key"
	embeddingAPIKeyInput.EchoMode = textinput.EchoPassword
	embeddingAPIKeyInput.CharLimit = 256

	llmAPIKeyInput := textinput.New()
	llmAPIKeyInput.Placeholder = "Enter API key"
	llmAPIKeyInput.EchoMode = textinput.EchoPassword
	llmAPIKeyInput.CharLimit = 256

	return &View{
		styles:               s,
		settingsService:      settingsService,
		section:              SectionOverview,
		embeddingAPIKeyInput: embeddingAPIKeyInput,
		llmAPIKeyInput:       llmAPIKeyInput,
	}
}

// Init initialises the view and loads settings.
func (v *View) Init() tea.Cmd {
	return v.loadSettings()
}

// loadSettings returns a command that loads current settings.
func (v *View) loadSettings() tea.Cmd {
	return func() tea.Msg {
		if v.settingsService == nil {
			return messages.SettingsLoaded{Err: fmt.Errorf("settings service not available")}
		}
		settings, err := v.settingsService.Get()
		return messages.SettingsLoaded{Settings: settings, Err: err}
	}
}

// Update handles messages for the settings view.
func (v *View) Update(msg tea.Msg) (*View, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
		v.ready = true
		return v, nil

	case messages.SettingsLoaded:
		if msg.Err != nil {
			v.err = msg.Err
		} else {
			v.settings = msg.Settings
			v.err = nil
		}
		return v, nil

	case messages.SettingsSaved:
		if msg.Err != nil {
			v.err = msg.Err
		} else {
			v.err = nil
			// Reload settings after save
			cmd := v.loadSettings()
			return v, cmd
		}
		return v, nil

	case tea.KeyMsg:
		return v.handleKeyMsg(msg)
	}

	return v, nil
}

// handleKeyMsg handles key presses based on current section.
//
//nolint:exhaustive // explicit default handling for escape provides better UX
func (v *View) handleKeyMsg(msg tea.KeyMsg) (*View, tea.Cmd) {
	// Global escape to go back
	if msg.String() == "esc" {
		switch v.section {
		case SectionOverview:
			return v, func() tea.Msg {
				return messages.ViewChanged{View: messages.ViewMenu}
			}
		default:
			v.section = SectionOverview
			v.selected = 0
			return v, nil
		}
	}

	switch v.section {
	case SectionOverview:
		return v.handleOverviewKeys(msg)
	case SectionSearchMode:
		return v.handleSearchModeKeys(msg)
	case SectionEmbedding:
		return v.handleEmbeddingKeys(msg)
	case SectionLLM:
		return v.handleLLMKeys(msg)
	}

	return v, nil
}

func (v *View) handleOverviewKeys(msg tea.KeyMsg) (*View, tea.Cmd) {
	// Overview menu: Search Mode, Embedding, LLM
	maxItems := 3

	switch msg.String() {
	case "up", "k":
		if v.selected > 0 {
			v.selected--
		}
	case keyDown, "j":
		if v.selected < maxItems-1 {
			v.selected++
		}
	case keyEnter:
		switch v.selected {
		case 0:
			v.section = SectionSearchMode
			v.selected = v.getSearchModeIndex()
		case 1:
			v.section = SectionEmbedding
			v.selected = v.getEmbeddingProviderIndex()
		case 2:
			v.section = SectionLLM
			v.selected = v.getLLMProviderIndex()
		}
	}
	return v, nil
}

func (v *View) handleSearchModeKeys(msg tea.KeyMsg) (*View, tea.Cmd) {
	modes := domain.AllSearchModes()

	switch msg.String() {
	case "up", "k":
		if v.selected > 0 {
			v.selected--
		}
	case keyDown, "j":
		if v.selected < len(modes)-1 {
			v.selected++
		}
	case keyEnter:
		if v.selected >= 0 && v.selected < len(modes) {
			cmd := v.setSearchMode(modes[v.selected])
			return v, cmd
		}
	}
	return v, nil
}

//nolint:dupl,gocognit,gocyclo // duplicate with handleLLMKeys; TUI input complexity
func (v *View) handleEmbeddingKeys(msg tea.KeyMsg) (*View, tea.Cmd) {
	providers := domain.AllEmbeddingProviders()

	// If we're focused on the API key input
	if v.focusedField == 1 {
		switch msg.String() {
		case keyTab, "shift+tab":
			v.focusedField = 0
			v.embeddingAPIKeyInput.Blur()
			return v, nil
		case keyEnter:
			// Save embedding provider
			if v.selected >= 0 && v.selected < len(providers) {
				cmd := v.setEmbeddingProvider(providers[v.selected], v.embeddingAPIKeyInput.Value())
				return v, cmd
			}
		default:
			var cmd tea.Cmd
			v.embeddingAPIKeyInput, cmd = v.embeddingAPIKeyInput.Update(msg)
			return v, cmd
		}
		return v, nil
	}

	switch msg.String() {
	case "up", "k":
		if v.selected > 0 {
			v.selected--
		}
	case keyDown, "j":
		if v.selected < len(providers)-1 {
			v.selected++
		}
	case keyTab:
		// Tab to API key input if provider requires it
		if v.selected >= 0 && v.selected < len(providers) && providers[v.selected].RequiresAPIKey() {
			v.focusedField = 1
			cmd := v.embeddingAPIKeyInput.Focus()
			return v, cmd
		}
	case keyEnter:
		if v.selected >= 0 && v.selected < len(providers) {
			provider := providers[v.selected]
			if provider.RequiresAPIKey() {
				// Need API key - focus on input
				v.focusedField = 1
				cmd := v.embeddingAPIKeyInput.Focus()
				return v, cmd
			}
			// No API key needed - save directly
			cmd := v.setEmbeddingProvider(provider, "")
			return v, cmd
		}
	}
	return v, nil
}

//nolint:dupl,gocognit,gocyclo // duplicate with handleEmbeddingKeys; TUI input complexity
func (v *View) handleLLMKeys(msg tea.KeyMsg) (*View, tea.Cmd) {
	providers := domain.AllLLMProviders()

	// If we're focused on the API key input
	if v.focusedField == 1 {
		switch msg.String() {
		case keyTab, "shift+tab":
			v.focusedField = 0
			v.llmAPIKeyInput.Blur()
			return v, nil
		case keyEnter:
			// Save LLM provider
			if v.selected >= 0 && v.selected < len(providers) {
				cmd := v.setLLMProvider(providers[v.selected], v.llmAPIKeyInput.Value())
				return v, cmd
			}
		default:
			var cmd tea.Cmd
			v.llmAPIKeyInput, cmd = v.llmAPIKeyInput.Update(msg)
			return v, cmd
		}
		return v, nil
	}

	switch msg.String() {
	case "up", "k":
		if v.selected > 0 {
			v.selected--
		}
	case keyDown, "j":
		if v.selected < len(providers)-1 {
			v.selected++
		}
	case keyTab:
		// Tab to API key input if provider requires it
		if v.selected >= 0 && v.selected < len(providers) && providers[v.selected].RequiresAPIKey() {
			v.focusedField = 1
			cmd := v.llmAPIKeyInput.Focus()
			return v, cmd
		}
	case keyEnter:
		if v.selected >= 0 && v.selected < len(providers) {
			provider := providers[v.selected]
			if provider.RequiresAPIKey() {
				// Need API key - focus on input
				v.focusedField = 1
				cmd := v.llmAPIKeyInput.Focus()
				return v, cmd
			}
			// No API key needed - save directly
			cmd := v.setLLMProvider(provider, "")
			return v, cmd
		}
	}
	return v, nil
}

// Commands to update settings.

func (v *View) setSearchMode(mode domain.SearchMode) tea.Cmd {
	return func() tea.Msg {
		if v.settingsService == nil {
			return messages.SettingsSaved{Err: fmt.Errorf("settings service not available")}
		}
		err := v.settingsService.SetSearchMode(mode)
		if err == nil {
			v.section = SectionOverview
			v.selected = 0
		}
		return messages.SettingsSaved{Err: err}
	}
}

func (v *View) setEmbeddingProvider(provider domain.AIProvider, apiKey string) tea.Cmd {
	return func() tea.Msg {
		if v.settingsService == nil {
			return messages.SettingsSaved{Err: fmt.Errorf("settings service not available")}
		}
		// Use default model
		defaults := domain.DefaultEmbeddingModels()
		model := defaults[provider]
		err := v.settingsService.SetEmbeddingProvider(provider, model, apiKey)
		if err == nil {
			v.section = SectionOverview
			v.selected = 0
			v.focusedField = 0
			v.embeddingAPIKeyInput.SetValue("")
			v.embeddingAPIKeyInput.Blur()
		}
		return messages.SettingsSaved{Err: err}
	}
}

func (v *View) setLLMProvider(provider domain.AIProvider, apiKey string) tea.Cmd {
	return func() tea.Msg {
		if v.settingsService == nil {
			return messages.SettingsSaved{Err: fmt.Errorf("settings service not available")}
		}
		// Use default model
		defaults := domain.DefaultLLMModels()
		model := defaults[provider]
		err := v.settingsService.SetLLMProvider(provider, model, apiKey)
		if err == nil {
			v.section = SectionOverview
			v.selected = 0
			v.focusedField = 0
			v.llmAPIKeyInput.SetValue("")
			v.llmAPIKeyInput.Blur()
		}
		return messages.SettingsSaved{Err: err}
	}
}

// Helper methods to get current selection indices.

func (v *View) getSearchModeIndex() int {
	if v.settings == nil {
		return 0
	}
	modes := domain.AllSearchModes()
	for i, m := range modes {
		if m == v.settings.Search.Mode {
			return i
		}
	}
	return 0
}

func (v *View) getEmbeddingProviderIndex() int {
	if v.settings == nil {
		return 0
	}
	providers := domain.AllEmbeddingProviders()
	for i, p := range providers {
		if p == v.settings.Embedding.Provider {
			return i
		}
	}
	return 0
}

func (v *View) getLLMProviderIndex() int {
	if v.settings == nil {
		return 0
	}
	providers := domain.AllLLMProviders()
	for i, p := range providers {
		if p == v.settings.LLM.Provider {
			return i
		}
	}
	return 0
}

// View renders the settings view.
func (v *View) View() string {
	var b strings.Builder

	b.WriteString(v.styles.Title.Render("Settings"))
	b.WriteString("\n\n")

	// Error display
	if v.err != nil {
		b.WriteString(v.styles.Error.Render(fmt.Sprintf("Error: %s", v.err.Error())))
		b.WriteString("\n\n")
	}

	// Loading state
	if v.settings == nil {
		b.WriteString(v.styles.Muted.Render("Loading settings..."))
		return b.String()
	}

	switch v.section {
	case SectionOverview:
		b.WriteString(v.renderOverview())
	case SectionSearchMode:
		b.WriteString(v.renderSearchModeSelect())
	case SectionEmbedding:
		b.WriteString(v.renderEmbeddingSelect())
	case SectionLLM:
		b.WriteString(v.renderLLMSelect())
	}

	b.WriteString("\n")
	b.WriteString(v.renderHelp())

	return b.String()
}

func (v *View) renderOverview() string {
	var b strings.Builder

	embeddingValue := "Not Set"
	if v.settings.Embedding.Provider != "" {
		embeddingValue = fmt.Sprintf("%s (%s)", v.settings.Embedding.Provider.Description(), v.settings.Embedding.Model)
	}

	llmValue := "Not Set"
	if v.settings.LLM.Provider != "" {
		llmValue = fmt.Sprintf("%s (%s)", v.settings.LLM.Provider.Description(), v.settings.LLM.Model)
	}

	items := []struct {
		label  string
		value  string
		status string
	}{
		{
			label: "Search Mode",
			value: v.settings.Search.Mode.Description(),
		},
		{
			label:  "Embedding Provider",
			value:  embeddingValue,
			status: v.getEmbeddingStatus(),
		},
		{
			label:  "LLM Provider",
			value:  llmValue,
			status: v.getLLMStatus(),
		},
	}

	for i, item := range items {
		indicator := "  "
		if i == v.selected {
			indicator = "> "
		}

		line := fmt.Sprintf("%s%s: %s", indicator, item.label, item.value)
		if item.status != "" {
			line += " " + item.status
		}

		if i == v.selected {
			b.WriteString(v.styles.Selected.Render(line))
		} else {
			b.WriteString(v.styles.Normal.Render(line))
		}
		b.WriteString("\n")
	}

	// Validation status
	b.WriteString("\n")
	if v.settingsService != nil {
		if err := v.settingsService.Validate(); err != nil {
			b.WriteString(v.styles.Warning.Render(fmt.Sprintf("Warning: %s", err.Error())))
		} else {
			b.WriteString(v.styles.Success.Render("Configuration is valid"))
		}
	}

	return b.String()
}

func (v *View) getEmbeddingStatus() string {
	if v.settings.Embedding.IsConfigured() {
		return v.styles.Success.Render("[configured]")
	}
	return v.styles.Warning.Render("[needs API key]")
}

func (v *View) getLLMStatus() string {
	if v.settings.LLM.IsConfigured() {
		return v.styles.Success.Render("[configured]")
	}
	return v.styles.Warning.Render("[needs API key]")
}

func (v *View) renderSearchModeSelect() string {
	var b strings.Builder

	b.WriteString(v.styles.Subtitle.Render("Select Search Mode"))
	b.WriteString("\n\n")

	modes := domain.AllSearchModes()
	for i, mode := range modes {
		indicator := "  "
		if i == v.selected {
			indicator = "> "
		}

		current := ""
		if v.settings != nil && mode == v.settings.Search.Mode {
			current = v.styles.Success.Render(" (current)")
		}

		line := fmt.Sprintf("%s%s%s", indicator, mode.Description(), current)
		if i == v.selected {
			b.WriteString(v.styles.Selected.Render(line))
		} else {
			b.WriteString(v.styles.Normal.Render(line))
		}
		b.WriteString("\n")

		// Show requirements
		if mode.RequiresEmbedding() || mode.RequiresLLM() {
			reqs := []string{}
			if mode.RequiresEmbedding() {
				reqs = append(reqs, "embedding")
			}
			if mode.RequiresLLM() {
				reqs = append(reqs, "LLM")
			}
			b.WriteString(v.styles.Muted.Render(fmt.Sprintf("    Requires: %s", strings.Join(reqs, ", "))))
			b.WriteString("\n")
		}
	}

	return b.String()
}

//nolint:dupl // intentional duplicate structure with renderLLMSelect for maintainability
func (v *View) renderEmbeddingSelect() string {
	var b strings.Builder

	b.WriteString(v.styles.Subtitle.Render("Select Embedding Provider"))
	b.WriteString("\n\n")

	providers := domain.AllEmbeddingProviders()
	for i, provider := range providers {
		indicator := "  "
		if i == v.selected && v.focusedField == 0 {
			indicator = "> "
		}

		current := ""
		if v.settings != nil && provider == v.settings.Embedding.Provider {
			current = v.styles.Success.Render(" (current)")
		}

		line := fmt.Sprintf("%s%s%s", indicator, provider.Description(), current)
		if i == v.selected && v.focusedField == 0 {
			b.WriteString(v.styles.Selected.Render(line))
		} else {
			b.WriteString(v.styles.Normal.Render(line))
		}
		b.WriteString("\n")

		// Show default model
		defaults := domain.DefaultEmbeddingModels()
		if model, ok := defaults[provider]; ok {
			b.WriteString(v.styles.Muted.Render(fmt.Sprintf("    Model: %s", model)))
			b.WriteString("\n")
		}
	}

	// API key input (if selected provider requires it)
	if v.selected >= 0 && v.selected < len(providers) && providers[v.selected].RequiresAPIKey() {
		b.WriteString("\n")
		b.WriteString(v.styles.Normal.Render("API Key:"))
		b.WriteString("\n")
		b.WriteString(v.embeddingAPIKeyInput.View())
		b.WriteString("\n")
	}

	return b.String()
}

//nolint:dupl // intentional duplicate structure with renderEmbeddingSelect for maintainability
func (v *View) renderLLMSelect() string {
	var b strings.Builder

	b.WriteString(v.styles.Subtitle.Render("Select LLM Provider"))
	b.WriteString("\n\n")

	providers := domain.AllLLMProviders()
	for i, provider := range providers {
		indicator := "  "
		if i == v.selected && v.focusedField == 0 {
			indicator = "> "
		}

		current := ""
		if v.settings != nil && provider == v.settings.LLM.Provider {
			current = v.styles.Success.Render(" (current)")
		}

		line := fmt.Sprintf("%s%s%s", indicator, provider.Description(), current)
		if i == v.selected && v.focusedField == 0 {
			b.WriteString(v.styles.Selected.Render(line))
		} else {
			b.WriteString(v.styles.Normal.Render(line))
		}
		b.WriteString("\n")

		// Show default model
		defaults := domain.DefaultLLMModels()
		if model, ok := defaults[provider]; ok {
			b.WriteString(v.styles.Muted.Render(fmt.Sprintf("    Model: %s", model)))
			b.WriteString("\n")
		}
	}

	// API key input (if selected provider requires it)
	if v.selected >= 0 && v.selected < len(providers) && providers[v.selected].RequiresAPIKey() {
		b.WriteString("\n")
		b.WriteString(v.styles.Normal.Render("API Key:"))
		b.WriteString("\n")
		b.WriteString(v.llmAPIKeyInput.View())
		b.WriteString("\n")
	}

	return b.String()
}

func (v *View) renderHelp() string {
	switch v.section {
	case SectionOverview:
		return v.styles.Help.Render("[j/k] navigate  [enter] edit  [esc] back")
	case SectionSearchMode:
		return v.styles.Help.Render("[j/k] navigate  [enter] select  [esc] back")
	case SectionEmbedding, SectionLLM:
		if v.focusedField == 1 {
			return v.styles.Help.Render("[tab] back to list  [enter] save  [esc] back")
		}
		return v.styles.Help.Render("[j/k] navigate  [tab] API key  [enter] select  [esc] back")
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

// Reset resets the view to initial state.
func (v *View) Reset() {
	v.section = SectionOverview
	v.selected = 0
	v.focusedField = 0
	v.err = nil
	v.embeddingAPIKeyInput.SetValue("")
	v.embeddingAPIKeyInput.Blur()
	v.llmAPIKeyInput.SetValue("")
	v.llmAPIKeyInput.Blur()
}
