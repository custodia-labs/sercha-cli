// Package input provides text input components for the TUI.
package input

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/custodia-labs/sercha-cli/internal/adapters/driving/tui/styles"
)

// SearchInput wraps a bubbles textinput with search-specific styling.
type SearchInput struct {
	textinput textinput.Model
	styles    *styles.Styles
	width     int
}

// NewSearchInput creates a new search input component.
func NewSearchInput(s *styles.Styles) *SearchInput {
	if s == nil {
		s = styles.DefaultStyles()
	}

	ti := textinput.New()
	ti.Placeholder = "Enter search query..."
	ti.Focus()
	ti.CharLimit = 256
	ti.Width = 50

	return &SearchInput{
		textinput: ti,
		styles:    s,
		width:     50,
	}
}

// Init initialises the search input.
func (s *SearchInput) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles input messages.
func (s *SearchInput) Update(msg tea.Msg) (*SearchInput, tea.Cmd) {
	var cmd tea.Cmd
	s.textinput, cmd = s.textinput.Update(msg)
	return s, cmd
}

// View renders the search input.
func (s *SearchInput) View() string {
	label := s.styles.Title.Render("Search: ")
	input := s.styles.InputField.Render(s.textinput.View())
	//nolint:misspell // lipgloss.Center is the correct constant from the library
	return lipgloss.JoinHorizontal(lipgloss.Center, label, input)
}

// Value returns the current input value.
func (s *SearchInput) Value() string {
	return s.textinput.Value()
}

// SetValue sets the input value.
func (s *SearchInput) SetValue(value string) {
	s.textinput.SetValue(value)
}

// Focus sets focus on the input.
func (s *SearchInput) Focus() tea.Cmd {
	return s.textinput.Focus()
}

// Blur removes focus from the input.
func (s *SearchInput) Blur() {
	s.textinput.Blur()
}

// Focused returns whether the input is focused.
func (s *SearchInput) Focused() bool {
	return s.textinput.Focused()
}

// SetWidth sets the width of the input.
func (s *SearchInput) SetWidth(width int) {
	s.width = width
	// Account for label and padding
	inputWidth := width - 10
	if inputWidth < 20 {
		inputWidth = 20
	}
	s.textinput.Width = inputWidth
}

// Width returns the current width.
func (s *SearchInput) Width() int {
	return s.width
}

// Reset clears the input.
func (s *SearchInput) Reset() {
	s.textinput.Reset()
}
