// Package keymap defines keybindings for the TUI.
package keymap

import (
	"github.com/charmbracelet/bubbles/key"
)

// KeyMap defines all keybindings for the TUI.
type KeyMap struct {
	// Quit exits the application.
	Quit key.Binding

	// Help shows the help view.
	Help key.Binding

	// Back returns to the previous view.
	Back key.Binding

	// Search triggers a search.
	Search key.Binding

	// Up navigates up in a list.
	Up key.Binding

	// Down navigates down in a list.
	Down key.Binding

	// Select confirms a selection.
	Select key.Binding

	// Cancel cancels the current operation.
	Cancel key.Binding

	// NewSearch starts a new search from results view.
	NewSearch key.Binding

	// Actions opens the action menu on a result.
	Actions key.Binding
}

// DefaultKeyMap returns the default keybindings.
func DefaultKeyMap() *KeyMap {
	return &KeyMap{
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
		Search: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "search"),
		),
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Select: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
		NewSearch: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "new search"),
		),
		Actions: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "actions"),
		),
	}
}

// ShortHelp returns a short list of keybindings for the help view.
func (k *KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Quit, k.Help}
}

// ResultsHelp returns keybindings for the results view.
func (k *KeyMap) ResultsHelp() []key.Binding {
	return []key.Binding{k.NewSearch, k.Up, k.Actions, k.Back}
}

// FullHelp returns the full list of keybindings for the help view.
func (k *KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Select},
		{k.Search, k.Back, k.Cancel},
		{k.Help, k.Quit},
	}
}

// Matches checks if a key string matches a binding.
func Matches(keyStr string, binding key.Binding) bool {
	for _, k := range binding.Keys() {
		if k == keyStr {
			return true
		}
	}
	return false
}
