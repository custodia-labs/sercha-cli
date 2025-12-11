package list

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/custodia-labs/sercha-cli/internal/adapters/driving/tui/styles"
	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

func sampleResults() []domain.SearchResult {
	return []domain.SearchResult{
		{Document: domain.Document{Title: "Document One"}, Score: 0.95},
		{Document: domain.Document{Title: "Document Two"}, Score: 0.85},
		{Document: domain.Document{Title: "Document Three"}, Score: 0.75},
	}
}

func TestNewResultList(t *testing.T) {
	s := styles.DefaultStyles()
	list := NewResultList(s)

	require.NotNil(t, list)
	assert.Equal(t, 0, list.Selected())
	assert.True(t, list.IsEmpty())
}

func TestNewResultList_NilStyles(t *testing.T) {
	list := NewResultList(nil)

	require.NotNil(t, list)
	assert.NotNil(t, list.styles)
}

func TestResultList_Init(t *testing.T) {
	list := NewResultList(nil)

	cmd := list.Init()

	assert.Nil(t, cmd)
}

func TestResultList_SetResults(t *testing.T) {
	list := NewResultList(nil)
	results := sampleResults()

	list.SetResults(results)

	assert.Equal(t, 3, list.Count())
	assert.False(t, list.IsEmpty())
	assert.Equal(t, 0, list.Selected())
}

func TestResultList_Results(t *testing.T) {
	list := NewResultList(nil)
	results := sampleResults()
	list.SetResults(results)

	got := list.Results()

	assert.Equal(t, results, got)
}

func TestResultList_Selected(t *testing.T) {
	list := NewResultList(nil)
	list.SetResults(sampleResults())

	assert.Equal(t, 0, list.Selected())

	list.SetSelected(1)
	assert.Equal(t, 1, list.Selected())
}

func TestResultList_SetSelected_Valid(t *testing.T) {
	list := NewResultList(nil)
	list.SetResults(sampleResults())

	list.SetSelected(2)

	assert.Equal(t, 2, list.Selected())
}

func TestResultList_SetSelected_OutOfBounds(t *testing.T) {
	list := NewResultList(nil)
	list.SetResults(sampleResults())

	list.SetSelected(99)

	assert.Equal(t, 0, list.Selected()) // Unchanged
}

func TestResultList_SetSelected_Negative(t *testing.T) {
	list := NewResultList(nil)
	list.SetResults(sampleResults())

	list.SetSelected(-1)

	assert.Equal(t, 0, list.Selected()) // Unchanged
}

func TestResultList_SelectedResult(t *testing.T) {
	list := NewResultList(nil)
	results := sampleResults()
	list.SetResults(results)

	result := list.SelectedResult()

	require.NotNil(t, result)
	assert.Equal(t, "Document One", result.Document.Title)
}

func TestResultList_SelectedResult_Empty(t *testing.T) {
	list := NewResultList(nil)

	result := list.SelectedResult()

	assert.Nil(t, result)
}

func TestResultList_MoveUp(t *testing.T) {
	list := NewResultList(nil)
	list.SetResults(sampleResults())
	list.SetSelected(1)

	list.MoveUp()

	assert.Equal(t, 0, list.Selected())
}

func TestResultList_MoveUp_AtTop(t *testing.T) {
	list := NewResultList(nil)
	list.SetResults(sampleResults())

	list.MoveUp()

	assert.Equal(t, 0, list.Selected()) // Stays at 0
}

func TestResultList_MoveDown(t *testing.T) {
	list := NewResultList(nil)
	list.SetResults(sampleResults())

	list.MoveDown()

	assert.Equal(t, 1, list.Selected())
}

func TestResultList_MoveDown_AtBottom(t *testing.T) {
	list := NewResultList(nil)
	list.SetResults(sampleResults())
	list.SetSelected(2)

	list.MoveDown()

	assert.Equal(t, 2, list.Selected()) // Stays at 2
}

func TestResultList_Update_KeyUp(t *testing.T) {
	list := NewResultList(nil)
	list.SetResults(sampleResults())
	list.SetSelected(1)

	msg := tea.KeyMsg{Type: tea.KeyUp}
	updated, cmd := list.Update(msg)

	assert.Equal(t, list, updated)
	assert.Nil(t, cmd)
	assert.Equal(t, 0, list.Selected())
}

func TestResultList_Update_KeyDown(t *testing.T) {
	list := NewResultList(nil)
	list.SetResults(sampleResults())

	msg := tea.KeyMsg{Type: tea.KeyDown}
	updated, cmd := list.Update(msg)

	assert.Equal(t, list, updated)
	assert.Nil(t, cmd)
	assert.Equal(t, 1, list.Selected())
}

func TestResultList_Update_KeyK(t *testing.T) {
	list := NewResultList(nil)
	list.SetResults(sampleResults())
	list.SetSelected(1)

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	list.Update(msg)

	assert.Equal(t, 0, list.Selected())
}

func TestResultList_Update_KeyJ(t *testing.T) {
	list := NewResultList(nil)
	list.SetResults(sampleResults())

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	list.Update(msg)

	assert.Equal(t, 1, list.Selected())
}

func TestResultList_View_Empty(t *testing.T) {
	list := NewResultList(nil)

	view := list.View()

	assert.Contains(t, view, "No results")
}

func TestResultList_View_WithResults(t *testing.T) {
	list := NewResultList(nil)
	list.SetResults(sampleResults())

	view := list.View()

	assert.Contains(t, view, "Results (3)")
	assert.Contains(t, view, "Document One")
	assert.Contains(t, view, "0.95")
}

func TestResultList_View_SelectedIndicator(t *testing.T) {
	list := NewResultList(nil)
	list.SetResults(sampleResults())

	view := list.View()

	assert.Contains(t, view, ">") // Selected indicator
}

func TestResultList_SetDimensions(t *testing.T) {
	list := NewResultList(nil)

	list.SetDimensions(100, 20)

	assert.Equal(t, 100, list.Width())
	assert.Equal(t, 20, list.Height())
}

func TestResultList_Width(t *testing.T) {
	list := NewResultList(nil)

	assert.Equal(t, 80, list.Width()) // Default
}

func TestResultList_Height(t *testing.T) {
	list := NewResultList(nil)

	assert.Equal(t, 10, list.Height()) // Default
}

func TestResultList_Count(t *testing.T) {
	list := NewResultList(nil)

	assert.Equal(t, 0, list.Count())

	list.SetResults(sampleResults())
	assert.Equal(t, 3, list.Count())
}

func TestResultList_IsEmpty(t *testing.T) {
	list := NewResultList(nil)

	assert.True(t, list.IsEmpty())

	list.SetResults(sampleResults())
	assert.False(t, list.IsEmpty())
}

func TestResultList_View_UntitledDocument(t *testing.T) {
	list := NewResultList(nil)
	list.SetResults([]domain.SearchResult{
		{Document: domain.Document{Title: ""}, Score: 0.5},
	})

	view := list.View()

	assert.Contains(t, view, "(Untitled)")
}

func TestResultList_View_LongTitle(t *testing.T) {
	list := NewResultList(nil)
	longTitle := "This is a very long document title that should be truncated when displayed in the list view"
	list.SetResults([]domain.SearchResult{
		{Document: domain.Document{Title: longTitle}, Score: 0.5},
	})

	view := list.View()

	// Should be truncated with ellipsis
	assert.Contains(t, view, "...")
}
