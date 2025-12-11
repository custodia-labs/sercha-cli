package input

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/custodia-labs/sercha-cli/internal/adapters/driving/tui/styles"
)

func TestNewSearchInput(t *testing.T) {
	s := styles.DefaultStyles()
	input := NewSearchInput(s)

	require.NotNil(t, input)
	assert.Equal(t, "", input.Value())
	assert.True(t, input.Focused())
}

func TestNewSearchInput_NilStyles(t *testing.T) {
	input := NewSearchInput(nil)

	require.NotNil(t, input)
	assert.NotNil(t, input.styles)
}

func TestSearchInput_Init(t *testing.T) {
	input := NewSearchInput(nil)

	cmd := input.Init()

	// Blink command should be returned
	assert.NotNil(t, cmd)
}

func TestSearchInput_Update(t *testing.T) {
	input := NewSearchInput(nil)

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
	updated, cmd := input.Update(msg)

	assert.Equal(t, input, updated)
	// textinput returns nil cmd for regular key presses
	_ = cmd
	assert.Equal(t, "a", input.Value())
}

func TestSearchInput_View(t *testing.T) {
	input := NewSearchInput(nil)

	view := input.View()

	assert.NotEmpty(t, view)
	assert.Contains(t, view, "Search")
}

func TestSearchInput_Value(t *testing.T) {
	input := NewSearchInput(nil)

	input.SetValue("test query")

	assert.Equal(t, "test query", input.Value())
}

func TestSearchInput_SetValue(t *testing.T) {
	input := NewSearchInput(nil)

	input.SetValue("hello world")

	assert.Equal(t, "hello world", input.Value())
}

func TestSearchInput_Focus(t *testing.T) {
	input := NewSearchInput(nil)
	input.Blur()

	assert.False(t, input.Focused())

	cmd := input.Focus()

	assert.NotNil(t, cmd)
	assert.True(t, input.Focused())
}

func TestSearchInput_Blur(t *testing.T) {
	input := NewSearchInput(nil)

	assert.True(t, input.Focused())

	input.Blur()

	assert.False(t, input.Focused())
}

func TestSearchInput_Focused(t *testing.T) {
	input := NewSearchInput(nil)

	assert.True(t, input.Focused())

	input.Blur()
	assert.False(t, input.Focused())
}

func TestSearchInput_SetWidth(t *testing.T) {
	input := NewSearchInput(nil)

	input.SetWidth(100)

	assert.Equal(t, 100, input.Width())
}

func TestSearchInput_SetWidth_Minimum(t *testing.T) {
	input := NewSearchInput(nil)

	input.SetWidth(10) // Very small, should use minimum

	assert.Equal(t, 10, input.Width())
	// Internal textinput width should be at least 20
}

func TestSearchInput_Width(t *testing.T) {
	input := NewSearchInput(nil)

	assert.Equal(t, 50, input.Width()) // Default width
}

func TestSearchInput_Reset(t *testing.T) {
	input := NewSearchInput(nil)
	input.SetValue("some text")

	input.Reset()

	assert.Equal(t, "", input.Value())
}

func TestSearchInput_Update_MultipleKeys(t *testing.T) {
	input := NewSearchInput(nil)

	keys := []rune{'h', 'e', 'l', 'l', 'o'}
	for _, k := range keys {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{k}}
		input.Update(msg)
	}

	assert.Equal(t, "hello", input.Value())
}

func TestSearchInput_Update_Backspace(t *testing.T) {
	input := NewSearchInput(nil)
	input.SetValue("test")

	msg := tea.KeyMsg{Type: tea.KeyBackspace}
	input.Update(msg)

	assert.Equal(t, "tes", input.Value())
}
