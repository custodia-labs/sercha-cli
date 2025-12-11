// Package status provides status bar components for the TUI.
package status

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/custodia-labs/sercha-cli/internal/adapters/driving/tui/keymap"
	"github.com/custodia-labs/sercha-cli/internal/adapters/driving/tui/styles"
)

// State represents the current application state for display.
type State string

const (
	StateReady     State = "ready"
	StateSearching State = "searching"
	StateError     State = "error"
	StateHelp      State = "help"
	StateResults   State = "results"
)

// Bar displays application status and keybinding hints.
type Bar struct {
	styles      *styles.Styles
	keymap      *keymap.KeyMap
	state       State
	message     string
	resultCount int
	width       int
}

// NewBar creates a new status bar component.
func NewBar(s *styles.Styles, km *keymap.KeyMap) *Bar {
	if s == nil {
		s = styles.DefaultStyles()
	}
	if km == nil {
		km = keymap.DefaultKeyMap()
	}

	return &Bar{
		styles:      s,
		keymap:      km,
		state:       StateReady,
		message:     "",
		resultCount: 0,
		width:       80,
	}
}

// Init initialises the status bar.
func (s *Bar) Init() tea.Cmd {
	return nil
}

// Update handles status bar messages.
func (s *Bar) Update(msg tea.Msg) (*Bar, tea.Cmd) {
	// Bar is mostly passive, updated via Set methods
	return s, nil
}

// View renders the status bar.
func (s *Bar) View() string {
	// Left side: state/message
	left := s.renderLeft()

	// Right side: keybinding hints
	right := s.renderRight()

	// Calculate padding
	leftLen := lipgloss.Width(left)
	rightLen := lipgloss.Width(right)
	padding := s.width - leftLen - rightLen
	if padding < 1 {
		padding = 1
	}

	bar := s.styles.StatusBar.Width(s.width).Render(
		left + strings.Repeat(" ", padding) + right,
	)

	return bar
}

// renderLeft renders the left side of the status bar.
func (s *Bar) renderLeft() string {
	switch s.state {
	case StateSearching:
		return s.styles.Muted.Render("Searching...")
	case StateError:
		if s.message != "" {
			return s.styles.Error.Render(fmt.Sprintf("Error: %s", s.message))
		}
		return s.styles.Error.Render("Error")
	case StateHelp:
		return s.styles.Normal.Render("Help")
	case StateReady, StateResults:
		if s.resultCount > 0 {
			return s.styles.Normal.Render(fmt.Sprintf("%d results", s.resultCount))
		}
		return s.styles.Muted.Render("Ready")
	}
	return s.styles.Muted.Render("Ready")
}

// renderRight renders keybinding hints.
func (s *Bar) renderRight() string {
	var bindings []key.Binding

	// Show different hints based on state
	if s.state == StateResults && s.resultCount > 0 {
		bindings = s.keymap.ResultsHelp()
	} else {
		bindings = s.keymap.ShortHelp()
	}

	hints := make([]string, 0, len(bindings))
	for _, b := range bindings {
		h := b.Help()
		hint := fmt.Sprintf("%s: %s", h.Key, h.Desc)
		hints = append(hints, hint)
	}
	return s.styles.Muted.Render(strings.Join(hints, " | "))
}

// SetState sets the current state.
func (s *Bar) SetState(state State) {
	s.state = state
}

// State returns the current state.
func (s *Bar) State() State {
	return s.state
}

// SetMessage sets a custom message.
func (s *Bar) SetMessage(message string) {
	s.message = message
}

// Message returns the current message.
func (s *Bar) Message() string {
	return s.message
}

// SetResultCount sets the result count.
func (s *Bar) SetResultCount(count int) {
	s.resultCount = count
}

// ResultCount returns the current result count.
func (s *Bar) ResultCount() int {
	return s.resultCount
}

// SetWidth sets the status bar width.
func (s *Bar) SetWidth(width int) {
	s.width = width
}

// Width returns the current width.
func (s *Bar) Width() int {
	return s.width
}

// Clear resets the status bar to default state.
func (s *Bar) Clear() {
	s.state = StateReady
	s.message = ""
	s.resultCount = 0
}
