package docdetails

import (
	"errors"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/custodia-labs/sercha-cli/internal/adapters/driving/tui/messages"
	"github.com/custodia-labs/sercha-cli/internal/adapters/driving/tui/styles"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driving"
)

func TestNewView(t *testing.T) {
	s := styles.DefaultStyles()

	view := NewView(s)

	require.NotNil(t, view)
	assert.False(t, view.ready)
	assert.Nil(t, view.details)
}

func TestNewView_NilStyles(t *testing.T) {
	view := NewView(nil)

	require.NotNil(t, view)
	assert.Nil(t, view.styles)
}

func TestView_SetDetails(t *testing.T) {
	view := NewView(nil)

	details := &driving.DocumentDetails{
		ID:         "doc-1",
		Title:      "Test Document",
		SourceName: "Test Source",
		SourceType: "filesystem",
		ChunkCount: 5,
	}
	view.SetDetails(details)

	assert.Equal(t, "doc-1", view.details.ID)
	assert.Equal(t, "Test Document", view.details.Title)
	assert.Equal(t, 0, view.scrollOffset)
	assert.NoError(t, view.err)
}

func TestView_SetError(t *testing.T) {
	view := NewView(nil)

	err := errors.New("test error")
	view.SetError(err)

	assert.Error(t, view.err)
}

func TestView_Init(t *testing.T) {
	view := NewView(nil)

	cmd := view.Init()

	assert.Nil(t, cmd)
}

func TestView_Update_WindowSize(t *testing.T) {
	view := NewView(nil)

	msg := tea.WindowSizeMsg{Width: 80, Height: 24}
	updated, cmd := view.Update(msg)

	assert.Equal(t, view, updated)
	assert.Nil(t, cmd)
	assert.True(t, view.ready)
	assert.Equal(t, 80, view.width)
	assert.Equal(t, 24, view.height)
}

func TestView_Update_KeyMsg_Back(t *testing.T) {
	view := NewView(nil)

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	_, cmd := view.Update(msg)

	require.NotNil(t, cmd)
	result := cmd()
	changed, ok := result.(messages.ViewChanged)
	assert.True(t, ok)
	assert.Equal(t, messages.ViewDocuments, changed.View)
}

func TestView_Update_KeyMsg_ScrollUp(t *testing.T) {
	view := NewView(nil)
	view.scrollOffset = 5

	msg := tea.KeyMsg{Type: tea.KeyUp}
	view.Update(msg)
	assert.Equal(t, 4, view.scrollOffset)

	// Test k key
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	view.Update(msg)
	assert.Equal(t, 3, view.scrollOffset)

	// Test boundary
	view.scrollOffset = 0
	msg = tea.KeyMsg{Type: tea.KeyUp}
	view.Update(msg)
	assert.Equal(t, 0, view.scrollOffset)
}

func TestView_Update_KeyMsg_ScrollDown(t *testing.T) {
	view := NewView(nil)
	view.height = 10
	view.scrollOffset = 0

	msg := tea.KeyMsg{Type: tea.KeyDown}
	view.Update(msg)

	// Test j key
	view.scrollOffset = 0
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	view.Update(msg)
}

func TestView_Update_ErrorOccurred(t *testing.T) {
	view := NewView(nil)

	msg := messages.ErrorOccurred{Err: errors.New("test error")}
	view.Update(msg)

	assert.Error(t, view.err)
}

func TestView_View_Loading(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewView(s)
	view.width = 80
	view.height = 24
	view.ready = true
	view.details = nil

	output := view.View()

	// Should show loading or empty state
	assert.NotEmpty(t, output)
}

func TestView_View_WithDetails(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewView(s)
	view.width = 80
	view.height = 24
	view.ready = true
	view.details = &driving.DocumentDetails{
		ID:         "doc-1",
		Title:      "Test Document",
		SourceID:   "src-1",
		SourceName: "Test Source",
		SourceType: "filesystem",
		URI:        "/path/to/file.md",
		ChunkCount: 5,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Metadata:   map[string]string{"type": "markdown"},
	}

	output := view.View()

	assert.Contains(t, output, "Test Document")
	assert.Contains(t, output, "Test Source")
}

func TestView_View_Error(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewView(s)
	view.width = 80
	view.height = 24
	view.ready = true
	view.err = errors.New("failed to load details")

	output := view.View()

	assert.Contains(t, output, "Error")
}

func TestView_View_MetadataFormatting(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewView(s)
	view.width = 80
	view.height = 24
	view.ready = true
	view.details = &driving.DocumentDetails{
		ID:       "doc-1",
		Title:    "Test",
		Metadata: map[string]string{"key1": "value1", "key2": "value2"},
	}

	output := view.View()

	// Should render metadata
	assert.NotEmpty(t, output)
}

func TestView_SetDimensions(t *testing.T) {
	view := NewView(nil)

	view.SetDimensions(100, 50)

	assert.Equal(t, 100, view.width)
	assert.Equal(t, 50, view.height)
}
