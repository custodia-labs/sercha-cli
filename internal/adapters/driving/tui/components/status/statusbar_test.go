package status

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/custodia-labs/sercha-cli/internal/adapters/driving/tui/keymap"
	"github.com/custodia-labs/sercha-cli/internal/adapters/driving/tui/styles"
)

func TestNewBar(t *testing.T) {
	s := styles.DefaultStyles()
	km := keymap.DefaultKeyMap()
	bar := NewBar(s, km)

	require.NotNil(t, bar)
	assert.Equal(t, StateReady, bar.State())
	assert.Equal(t, "", bar.Message())
	assert.Equal(t, 0, bar.ResultCount())
}

func TestNewBar_NilStyles(t *testing.T) {
	bar := NewBar(nil, nil)

	require.NotNil(t, bar)
	assert.NotNil(t, bar.styles)
	assert.NotNil(t, bar.keymap)
}

func TestStatusBar_Init(t *testing.T) {
	bar := NewBar(nil, nil)

	cmd := bar.Init()

	assert.Nil(t, cmd)
}

func TestStatusBar_Update(t *testing.T) {
	bar := NewBar(nil, nil)

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updated, cmd := bar.Update(msg)

	assert.Equal(t, bar, updated)
	assert.Nil(t, cmd)
}

func TestStatusBar_SetState(t *testing.T) {
	bar := NewBar(nil, nil)

	bar.SetState(StateSearching)

	assert.Equal(t, StateSearching, bar.State())
}

func TestStatusBar_State(t *testing.T) {
	bar := NewBar(nil, nil)

	assert.Equal(t, StateReady, bar.State())
}

func TestStatusBar_SetMessage(t *testing.T) {
	bar := NewBar(nil, nil)

	bar.SetMessage("test message")

	assert.Equal(t, "test message", bar.Message())
}

func TestStatusBar_Message(t *testing.T) {
	bar := NewBar(nil, nil)

	assert.Equal(t, "", bar.Message())
}

func TestStatusBar_SetResultCount(t *testing.T) {
	bar := NewBar(nil, nil)

	bar.SetResultCount(42)

	assert.Equal(t, 42, bar.ResultCount())
}

func TestStatusBar_ResultCount(t *testing.T) {
	bar := NewBar(nil, nil)

	assert.Equal(t, 0, bar.ResultCount())
}

func TestStatusBar_SetWidth(t *testing.T) {
	bar := NewBar(nil, nil)

	bar.SetWidth(120)

	assert.Equal(t, 120, bar.Width())
}

func TestStatusBar_Width(t *testing.T) {
	bar := NewBar(nil, nil)

	assert.Equal(t, 80, bar.Width()) // Default
}

func TestStatusBar_Clear(t *testing.T) {
	bar := NewBar(nil, nil)
	bar.SetState(StateError)
	bar.SetMessage("error message")
	bar.SetResultCount(10)

	bar.Clear()

	assert.Equal(t, StateReady, bar.State())
	assert.Equal(t, "", bar.Message())
	assert.Equal(t, 0, bar.ResultCount())
}

func TestStatusBar_View_Ready(t *testing.T) {
	bar := NewBar(nil, nil)

	view := bar.View()

	assert.NotEmpty(t, view)
	assert.Contains(t, view, "Ready")
}

func TestStatusBar_View_Searching(t *testing.T) {
	bar := NewBar(nil, nil)
	bar.SetState(StateSearching)

	view := bar.View()

	assert.Contains(t, view, "Searching")
}

func TestStatusBar_View_Error(t *testing.T) {
	bar := NewBar(nil, nil)
	bar.SetState(StateError)

	view := bar.View()

	assert.Contains(t, view, "Error")
}

func TestStatusBar_View_ErrorWithMessage(t *testing.T) {
	bar := NewBar(nil, nil)
	bar.SetState(StateError)
	bar.SetMessage("connection failed")

	view := bar.View()

	assert.Contains(t, view, "Error")
	assert.Contains(t, view, "connection failed")
}

func TestStatusBar_View_Help(t *testing.T) {
	bar := NewBar(nil, nil)
	bar.SetState(StateHelp)

	view := bar.View()

	assert.Contains(t, view, "Help")
}

func TestStatusBar_View_WithResults(t *testing.T) {
	bar := NewBar(nil, nil)
	bar.SetResultCount(5)

	view := bar.View()

	assert.Contains(t, view, "5 results")
}

func TestStatusBar_View_ShowsKeybindings(t *testing.T) {
	bar := NewBar(nil, nil)

	view := bar.View()

	// Should show quit keybinding
	assert.Contains(t, view, "quit")
}

func TestState_Constants(t *testing.T) {
	assert.Equal(t, State("ready"), StateReady)
	assert.Equal(t, State("searching"), StateSearching)
	assert.Equal(t, State("error"), StateError)
	assert.Equal(t, State("help"), StateHelp)
}
