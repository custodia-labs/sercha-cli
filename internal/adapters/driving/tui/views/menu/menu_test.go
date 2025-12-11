package menu

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/custodia-labs/sercha-cli/internal/adapters/driving/tui/messages"
	"github.com/custodia-labs/sercha-cli/internal/adapters/driving/tui/styles"
)

func TestNewView(t *testing.T) {
	s := styles.DefaultStyles()

	view := NewView(s)

	require.NotNil(t, view)
	assert.NotNil(t, view.styles)
	assert.Len(t, view.items, 5)
	assert.Equal(t, 0, view.selected)
	assert.Equal(t, 80, view.width)
	assert.Equal(t, 24, view.height)
}

func TestNewView_NilStyles(t *testing.T) {
	view := NewView(nil)

	require.NotNil(t, view)
	// Should create default styles
	assert.NotNil(t, view.styles)
}

func TestView_Init(t *testing.T) {
	view := NewView(nil)

	cmd := view.Init()

	assert.Nil(t, cmd)
}

func TestView_Update_WindowSize(t *testing.T) {
	view := NewView(nil)

	msg := tea.WindowSizeMsg{Width: 100, Height: 50}
	updated, cmd := view.Update(msg)

	assert.Equal(t, view, updated)
	assert.Nil(t, cmd)
	assert.True(t, view.ready)
	assert.Equal(t, 100, view.width)
	assert.Equal(t, 50, view.height)
}

func TestView_Update_KeyMsg_NavigateDown(t *testing.T) {
	view := NewView(nil)
	view.selected = 0

	// Test down key
	msg := tea.KeyMsg{Type: tea.KeyDown}
	view.Update(msg)
	assert.Equal(t, 1, view.selected)

	// Test j key
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	view.Update(msg)
	assert.Equal(t, 2, view.selected)

	// Navigate to last item (5 items: Search, Sources, Settings, Help, Quit)
	view.Update(msg)
	assert.Equal(t, 3, view.selected)
	view.Update(msg)
	assert.Equal(t, 4, view.selected)

	// Test boundary - can't go past last item
	view.Update(msg)
	assert.Equal(t, 4, view.selected)
}

func TestView_Update_KeyMsg_NavigateUp(t *testing.T) {
	view := NewView(nil)
	view.selected = 3

	// Test up key
	msg := tea.KeyMsg{Type: tea.KeyUp}
	view.Update(msg)
	assert.Equal(t, 2, view.selected)

	// Test k key
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	view.Update(msg)
	assert.Equal(t, 1, view.selected)

	view.Update(msg)
	assert.Equal(t, 0, view.selected)

	// Test boundary - can't go before first item
	view.Update(msg)
	assert.Equal(t, 0, view.selected)
}

func TestView_Update_KeyMsg_Enter_ViewChange(t *testing.T) {
	view := NewView(nil)
	view.selected = 0 // Search

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := view.Update(msg)

	require.NotNil(t, cmd)
	result := cmd()
	changed, ok := result.(messages.ViewChanged)
	require.True(t, ok)
	assert.Equal(t, messages.ViewSearch, changed.View)
}

func TestView_Update_KeyMsg_Enter_Sources(t *testing.T) {
	view := NewView(nil)
	view.selected = 1 // Sources

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := view.Update(msg)

	require.NotNil(t, cmd)
	result := cmd()
	changed, ok := result.(messages.ViewChanged)
	require.True(t, ok)
	assert.Equal(t, messages.ViewSources, changed.View)
}

func TestView_Update_KeyMsg_Enter_Help(t *testing.T) {
	view := NewView(nil)
	view.selected = 3 // Help

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := view.Update(msg)

	require.NotNil(t, cmd)
	result := cmd()
	changed, ok := result.(messages.ViewChanged)
	require.True(t, ok)
	assert.Equal(t, messages.ViewHelp, changed.View)
}

func TestView_Update_KeyMsg_Enter_Quit(t *testing.T) {
	view := NewView(nil)
	view.selected = 4 // Quit

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := view.Update(msg)

	// Should return tea.Quit
	require.NotNil(t, cmd)
}

func TestView_Update_KeyMsg_Q(t *testing.T) {
	view := NewView(nil)

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	_, cmd := view.Update(msg)

	// Should return tea.Quit
	require.NotNil(t, cmd)
}

func TestView_View_NotReady(t *testing.T) {
	view := NewView(nil)
	view.ready = false

	output := view.View()

	assert.Contains(t, output, "Initialising")
}

func TestView_View_Ready(t *testing.T) {
	view := NewView(nil)
	view.width = 80
	view.height = 24
	view.ready = true

	output := view.View()

	assert.Contains(t, output, "Sercha")
	assert.Contains(t, output, "Local Document Search")
	assert.Contains(t, output, "Search")
	assert.Contains(t, output, "Sources")
	assert.Contains(t, output, "Help")
	assert.Contains(t, output, "Quit")
	assert.Contains(t, output, ">") // Selection indicator
}

func TestView_View_MenuItems(t *testing.T) {
	view := NewView(nil)
	view.ready = true

	// Default selection is Search (index 0)
	output := view.View()
	assert.Contains(t, output, "Search")
	assert.Contains(t, output, "Sources")
}

func TestView_SetDimensions(t *testing.T) {
	view := NewView(nil)
	view.ready = false

	view.SetDimensions(120, 60)

	assert.Equal(t, 120, view.width)
	assert.Equal(t, 60, view.height)
	assert.True(t, view.ready)
}

func TestView_Selected(t *testing.T) {
	view := NewView(nil)
	view.selected = 2

	assert.Equal(t, 2, view.Selected())
}

func TestMenuItem_Properties(t *testing.T) {
	view := NewView(nil)

	// Search item
	assert.Equal(t, "Search", view.items[0].Label)
	assert.Equal(t, messages.ViewSearch, view.items[0].View)
	assert.False(t, view.items[0].Quit)

	// Sources item
	assert.Equal(t, "Sources", view.items[1].Label)
	assert.Equal(t, messages.ViewSources, view.items[1].View)
	assert.False(t, view.items[1].Quit)

	// Settings item
	assert.Equal(t, "Settings", view.items[2].Label)
	assert.Equal(t, messages.ViewSettings, view.items[2].View)
	assert.False(t, view.items[2].Quit)

	// Help item
	assert.Equal(t, "Help", view.items[3].Label)
	assert.Equal(t, messages.ViewHelp, view.items[3].View)
	assert.False(t, view.items[3].Quit)

	// Quit item
	assert.Equal(t, "Quit", view.items[4].Label)
	assert.True(t, view.items[4].Quit)
}
