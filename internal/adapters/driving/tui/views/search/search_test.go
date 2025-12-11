package search

import (
	"context"
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/custodia-labs/sercha-cli/internal/adapters/driving/tui/keymap"
	"github.com/custodia-labs/sercha-cli/internal/adapters/driving/tui/messages"
	"github.com/custodia-labs/sercha-cli/internal/adapters/driving/tui/styles"
	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

// MockSearchService implements driving.SearchService for testing.
type MockSearchService struct {
	SearchFunc func(ctx context.Context, query string, opts domain.SearchOptions) ([]domain.SearchResult, error)
}

func (m *MockSearchService) Search(
	ctx context.Context,
	query string,
	opts domain.SearchOptions,
) ([]domain.SearchResult, error) {
	if m.SearchFunc != nil {
		return m.SearchFunc(ctx, query, opts)
	}
	return []domain.SearchResult{}, nil
}

// MockResultActionService implements driving.ResultActionService for testing.
type MockResultActionService struct {
	CopyToClipboardFunc func(ctx context.Context, result *domain.SearchResult) error
	OpenDocumentFunc    func(ctx context.Context, result *domain.SearchResult) error
}

func (m *MockResultActionService) CopyToClipboard(ctx context.Context, result *domain.SearchResult) error {
	if m.CopyToClipboardFunc != nil {
		return m.CopyToClipboardFunc(ctx, result)
	}
	return nil
}

func (m *MockResultActionService) OpenDocument(ctx context.Context, result *domain.SearchResult) error {
	if m.OpenDocumentFunc != nil {
		return m.OpenDocumentFunc(ctx, result)
	}
	return nil
}

// Helper function to create test search results.
func testSearchResults() []domain.SearchResult {
	return []domain.SearchResult{
		{
			Document: domain.Document{
				ID:       "1",
				Title:    "Test Document 1",
				URI:      "/path/to/doc1.txt",
				SourceID: "test-source",
			},
			Score: 0.95,
		},
		{
			Document: domain.Document{
				ID:       "2",
				Title:    "Test Document 2",
				URI:      "/path/to/doc2.txt",
				SourceID: "test-source",
			},
			Score: 0.85,
		},
	}
}

func TestNewView(t *testing.T) {
	s := styles.DefaultStyles()
	km := keymap.DefaultKeyMap()
	mock := &MockSearchService{}

	view := NewView(s, km, mock, nil)

	require.NotNil(t, view)
	assert.False(t, view.Ready())
	assert.Equal(t, "", view.Query())
	assert.True(t, view.InputFocused())
}

func TestNewView_NilStyles(t *testing.T) {
	view := NewView(nil, nil, nil, nil)

	require.NotNil(t, view)
	assert.NotNil(t, view.styles)
	assert.NotNil(t, view.keymap)
}

func TestView_WithContext(t *testing.T) {
	view := NewView(nil, nil, nil, nil)
	type contextKey string
	ctx := context.WithValue(context.Background(), contextKey("key"), "value")

	result := view.WithContext(ctx)

	assert.Equal(t, view, result)
	assert.Equal(t, ctx, view.ctx)
}

func TestView_Init(t *testing.T) {
	view := NewView(nil, nil, nil, nil)

	cmd := view.Init()

	// Blink command from input
	assert.NotNil(t, cmd)
}

func TestView_Update_WindowSize(t *testing.T) {
	view := NewView(nil, nil, nil, nil)

	msg := tea.WindowSizeMsg{Width: 80, Height: 24}
	updated, cmd := view.Update(msg)

	assert.Equal(t, view, updated)
	assert.Nil(t, cmd)
	assert.True(t, view.Ready())
	assert.Equal(t, 80, view.Width())
	assert.Equal(t, 24, view.Height())
}

func TestView_Update_SearchCompleted(t *testing.T) {
	view := NewView(nil, nil, nil, nil)
	view.SetDimensions(80, 24)
	view.focusInput = true

	results := testSearchResults()
	msg := messages.SearchCompleted{Results: results, Err: nil}
	updated, cmd := view.Update(msg)

	assert.Equal(t, view, updated)
	assert.Nil(t, cmd)
	assert.Len(t, view.Results(), 2)
	assert.False(t, view.InputFocused())
}

func TestView_Update_SearchCompleted_WithError(t *testing.T) {
	view := NewView(nil, nil, nil, nil)
	view.SetDimensions(80, 24)

	err := errors.New("search failed")
	msg := messages.SearchCompleted{Results: nil, Err: err}
	updated, cmd := view.Update(msg)

	assert.Equal(t, view, updated)
	assert.Nil(t, cmd)
	assert.Error(t, view.Err())
}

func TestView_Update_ErrorOccurred(t *testing.T) {
	view := NewView(nil, nil, nil, nil)

	err := errors.New("something went wrong")
	msg := messages.ErrorOccurred{Err: err}
	updated, cmd := view.Update(msg)

	assert.Equal(t, view, updated)
	assert.Nil(t, cmd)
	assert.Error(t, view.Err())
}

func TestView_Update_KeyEnter_WithQuery(t *testing.T) {
	searchCalled := false
	mock := &MockSearchService{
		SearchFunc: func(ctx context.Context, query string, opts domain.SearchOptions) ([]domain.SearchResult, error) {
			searchCalled = true
			assert.Equal(t, "test", query)
			return []domain.SearchResult{}, nil
		},
	}
	view := NewView(nil, nil, mock, nil)
	view.SetQuery("test")

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := view.Update(msg)

	require.NotNil(t, cmd)
	result := cmd()
	assert.IsType(t, messages.SearchCompleted{}, result)
	assert.True(t, searchCalled)
	assert.False(t, view.InputFocused())
}

func TestView_Update_KeyEnter_EmptyQuery(t *testing.T) {
	view := NewView(nil, nil, nil, nil)

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := view.Update(msg)

	assert.Nil(t, cmd)
}

func TestView_Update_KeyEsc_BackToMenu(t *testing.T) {
	view := NewView(nil, nil, nil, nil)

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	_, cmd := view.Update(msg)

	require.NotNil(t, cmd)
	result := cmd()
	changed, ok := result.(messages.ViewChanged)
	require.True(t, ok)
	assert.Equal(t, messages.ViewMenu, changed.View)
}

func TestView_Update_KeyN_NewSearch(t *testing.T) {
	view := NewView(nil, nil, nil, nil)
	view.SetDimensions(80, 24)
	view.Update(messages.SearchCompleted{Results: testSearchResults()})
	view.focusInput = false
	view.SetQuery("old query")

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
	view.Update(msg)

	assert.True(t, view.InputFocused())
	assert.Equal(t, "", view.Query())
}

func TestView_Update_KeyEnter_InResultsMode_OpensActionMenu(t *testing.T) {
	view := NewView(nil, nil, nil, nil)
	view.SetDimensions(80, 24)
	view.Update(messages.SearchCompleted{Results: testSearchResults()})
	view.focusInput = false

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	view.Update(msg)

	assert.NotNil(t, view.actionMenu)
	assert.True(t, view.actionMenu.visible)
	assert.Equal(t, 0, view.actionMenu.selected)
	assert.Len(t, view.actionMenu.actions, 3)
}

func TestView_Update_KeyEnter_InResultsMode_NoResults(t *testing.T) {
	view := NewView(nil, nil, nil, nil)
	view.focusInput = false

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	view.Update(msg)

	assert.Nil(t, view.actionMenu)
}

func TestView_Update_KeyUp(t *testing.T) {
	view := NewView(nil, nil, nil, nil)
	view.Update(messages.SearchCompleted{
		Results: testSearchResults(),
	})
	// Simulate being in results mode (after search)
	view.focusInput = false

	// Select second item first
	view.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 1, view.SelectedIndex())

	msg := tea.KeyMsg{Type: tea.KeyUp}
	view.Update(msg)

	assert.Equal(t, 0, view.SelectedIndex())
}

func TestView_Update_KeyDown(t *testing.T) {
	view := NewView(nil, nil, nil, nil)
	view.Update(messages.SearchCompleted{
		Results: testSearchResults(),
	})
	// Simulate being in results mode (after search)
	view.focusInput = false

	msg := tea.KeyMsg{Type: tea.KeyDown}
	view.Update(msg)

	assert.Equal(t, 1, view.SelectedIndex())
}

func TestView_Update_KeyK(t *testing.T) {
	view := NewView(nil, nil, nil, nil)
	view.Update(messages.SearchCompleted{
		Results: testSearchResults(),
	})
	// Simulate being in results mode (after search)
	view.focusInput = false
	view.Update(tea.KeyMsg{Type: tea.KeyDown})

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	view.Update(msg)

	assert.Equal(t, 0, view.SelectedIndex())
}

func TestView_Update_KeyJ(t *testing.T) {
	view := NewView(nil, nil, nil, nil)
	view.Update(messages.SearchCompleted{
		Results: testSearchResults(),
	})
	// Simulate being in results mode (after search)
	view.focusInput = false

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	view.Update(msg)

	assert.Equal(t, 1, view.SelectedIndex())
}

func TestView_Update_CharacterInput(t *testing.T) {
	view := NewView(nil, nil, nil, nil)

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
	view.Update(msg)

	assert.Equal(t, "a", view.Query())
}

func TestView_Update_Backspace(t *testing.T) {
	view := NewView(nil, nil, nil, nil)
	view.SetQuery("test")

	msg := tea.KeyMsg{Type: tea.KeyBackspace}
	view.Update(msg)

	assert.Equal(t, "tes", view.Query())
}

func TestView_View_NotReady(t *testing.T) {
	view := NewView(nil, nil, nil, nil)

	output := view.View()

	assert.Contains(t, output, "Initialising")
}

func TestView_View_Ready(t *testing.T) {
	view := NewView(nil, nil, nil, nil)
	view.SetDimensions(80, 24)

	output := view.View()

	assert.Contains(t, output, "Sercha")
	assert.Contains(t, output, "Search")
}

func TestView_View_WithError(t *testing.T) {
	view := NewView(nil, nil, nil, nil)
	view.SetDimensions(80, 24)
	view.err = errors.New("test error")

	output := view.View()

	assert.Contains(t, output, "Error")
	assert.Contains(t, output, "test error")
}

func TestView_View_WithResults(t *testing.T) {
	view := NewView(nil, nil, nil, nil)
	view.SetDimensions(80, 24)
	view.Update(messages.SearchCompleted{
		Results: []domain.SearchResult{
			{Document: domain.Document{Title: "Test Doc"}, Score: 0.95},
		},
	})

	output := view.View()

	assert.Contains(t, output, "Test Doc")
}

func TestView_View_WithActionMenu(t *testing.T) {
	view := NewView(nil, nil, nil, nil)
	view.SetDimensions(80, 24)
	view.Update(messages.SearchCompleted{Results: testSearchResults()})
	view.focusInput = false

	// Open action menu
	view.Update(tea.KeyMsg{Type: tea.KeyEnter})

	output := view.View()

	assert.Contains(t, output, "Copy plain text")
	assert.Contains(t, output, "Open Document")
	assert.Contains(t, output, "Cancel")
	assert.Contains(t, output, ">") // Selection indicator
}

func TestView_SetDimensions(t *testing.T) {
	view := NewView(nil, nil, nil, nil)

	view.SetDimensions(100, 50)

	assert.Equal(t, 100, view.Width())
	assert.Equal(t, 50, view.Height())
	assert.True(t, view.Ready())
}

func TestView_Width(t *testing.T) {
	view := NewView(nil, nil, nil, nil)

	assert.Equal(t, 80, view.Width()) // Default
}

func TestView_Height(t *testing.T) {
	view := NewView(nil, nil, nil, nil)

	assert.Equal(t, 24, view.Height()) // Default
}

func TestView_Ready(t *testing.T) {
	view := NewView(nil, nil, nil, nil)

	assert.False(t, view.Ready())

	view.SetDimensions(80, 24)
	assert.True(t, view.Ready())
}

func TestView_Query(t *testing.T) {
	view := NewView(nil, nil, nil, nil)

	assert.Equal(t, "", view.Query())
}

func TestView_SetQuery(t *testing.T) {
	view := NewView(nil, nil, nil, nil)

	view.SetQuery("test query")

	assert.Equal(t, "test query", view.Query())
}

func TestView_Results(t *testing.T) {
	view := NewView(nil, nil, nil, nil)

	assert.Nil(t, view.Results())
}

func TestView_SelectedIndex(t *testing.T) {
	view := NewView(nil, nil, nil, nil)

	assert.Equal(t, 0, view.SelectedIndex())
}

func TestView_SelectedResult_Empty(t *testing.T) {
	view := NewView(nil, nil, nil, nil)

	assert.Nil(t, view.SelectedResult())
}

func TestView_SelectedResult_WithResults(t *testing.T) {
	view := NewView(nil, nil, nil, nil)
	view.Update(messages.SearchCompleted{
		Results: testSearchResults(),
	})

	result := view.SelectedResult()

	require.NotNil(t, result)
	assert.Equal(t, "Test Document 1", result.Document.Title)
}

func TestView_Err(t *testing.T) {
	view := NewView(nil, nil, nil, nil)

	assert.Nil(t, view.Err())
}

func TestView_ClearError(t *testing.T) {
	view := NewView(nil, nil, nil, nil)
	view.err = errors.New("some error")

	view.ClearError()

	assert.Nil(t, view.Err())
}

func TestView_Reset(t *testing.T) {
	view := NewView(nil, nil, nil, nil)
	view.SetDimensions(80, 24)
	view.SetQuery("test query")
	view.Update(messages.SearchCompleted{Results: testSearchResults()})
	view.focusInput = false
	view.err = errors.New("test error")

	view.Reset()

	assert.True(t, view.InputFocused())
	assert.Equal(t, "", view.Query())
	assert.Nil(t, view.Results())
	assert.Nil(t, view.Err())
}

func TestView_InputFocused(t *testing.T) {
	view := NewView(nil, nil, nil, nil)

	assert.True(t, view.InputFocused())

	view.focusInput = false
	assert.False(t, view.InputFocused())
}

func TestView_PerformSearch_NoService(t *testing.T) {
	view := NewView(nil, nil, nil, nil)
	view.SetQuery("test")

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := view.Update(msg)

	require.NotNil(t, cmd)
	result := cmd()

	assert.IsType(t, messages.ErrorOccurred{}, result)
	errMsg := result.(messages.ErrorOccurred)
	assert.Equal(t, ErrNoSearchService, errMsg.Err)
}

func TestView_PerformSearch_ServiceError(t *testing.T) {
	expectedErr := errors.New("search service error")
	mock := &MockSearchService{
		SearchFunc: func(ctx context.Context, query string, opts domain.SearchOptions) ([]domain.SearchResult, error) {
			return nil, expectedErr
		},
	}
	view := NewView(nil, nil, mock, nil)
	view.SetQuery("test")

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := view.Update(msg)

	require.NotNil(t, cmd)
	result := cmd()

	assert.IsType(t, messages.SearchCompleted{}, result)
	completed := result.(messages.SearchCompleted)
	assert.Error(t, completed.Err)
}

// Action Menu Tests

func TestView_ActionMenu_NavigateDown(t *testing.T) {
	view := NewView(nil, nil, nil, nil)
	view.SetDimensions(80, 24)
	view.Update(messages.SearchCompleted{Results: testSearchResults()})
	view.focusInput = false

	// Open action menu
	view.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Equal(t, 0, view.actionMenu.selected)

	// Navigate down
	view.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 1, view.actionMenu.selected)

	view.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 2, view.actionMenu.selected)

	// Try to go past last item
	view.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 2, view.actionMenu.selected)
}

func TestView_ActionMenu_NavigateUp(t *testing.T) {
	view := NewView(nil, nil, nil, nil)
	view.SetDimensions(80, 24)
	view.Update(messages.SearchCompleted{Results: testSearchResults()})
	view.focusInput = false

	// Open action menu
	view.Update(tea.KeyMsg{Type: tea.KeyEnter})
	view.actionMenu.selected = 2

	// Navigate up
	view.Update(tea.KeyMsg{Type: tea.KeyUp})
	assert.Equal(t, 1, view.actionMenu.selected)

	view.Update(tea.KeyMsg{Type: tea.KeyUp})
	assert.Equal(t, 0, view.actionMenu.selected)

	// Try to go before first item
	view.Update(tea.KeyMsg{Type: tea.KeyUp})
	assert.Equal(t, 0, view.actionMenu.selected)
}

func TestView_ActionMenu_NavigateWithVimKeys(t *testing.T) {
	view := NewView(nil, nil, nil, nil)
	view.SetDimensions(80, 24)
	view.Update(messages.SearchCompleted{Results: testSearchResults()})
	view.focusInput = false

	// Open action menu
	view.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Equal(t, 0, view.actionMenu.selected)

	// Navigate down with j
	view.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	assert.Equal(t, 1, view.actionMenu.selected)

	// Navigate up with k
	view.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	assert.Equal(t, 0, view.actionMenu.selected)
}

func TestView_ActionMenu_Escape_ClosesMenu(t *testing.T) {
	view := NewView(nil, nil, nil, nil)
	view.SetDimensions(80, 24)
	view.Update(messages.SearchCompleted{Results: testSearchResults()})
	view.focusInput = false

	// Open action menu
	view.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.NotNil(t, view.actionMenu)

	// Press Escape
	view.Update(tea.KeyMsg{Type: tea.KeyEsc})
	assert.Nil(t, view.actionMenu)
}

func TestView_ActionMenu_SelectCancel(t *testing.T) {
	view := NewView(nil, nil, nil, nil)
	view.SetDimensions(80, 24)
	view.Update(messages.SearchCompleted{Results: testSearchResults()})
	view.focusInput = false

	// Open action menu
	view.Update(tea.KeyMsg{Type: tea.KeyEnter})
	view.actionMenu.selected = 2 // Cancel

	// Press Enter
	view.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Nil(t, view.actionMenu)
}

func TestView_ActionMenu_CopyToClipboard_Success(t *testing.T) {
	copyCalled := false
	mockAction := &MockResultActionService{
		CopyToClipboardFunc: func(ctx context.Context, result *domain.SearchResult) error {
			copyCalled = true
			assert.Equal(t, "Test Document 1", result.Document.Title)
			return nil
		},
	}

	view := NewView(nil, nil, nil, mockAction)
	view.SetDimensions(80, 24)
	view.Update(messages.SearchCompleted{Results: testSearchResults()})
	view.focusInput = false

	// Open action menu
	view.Update(tea.KeyMsg{Type: tea.KeyEnter})
	view.actionMenu.selected = 0 // Copy plain text

	// Press Enter
	view.Update(tea.KeyMsg{Type: tea.KeyEnter})

	assert.Nil(t, view.actionMenu)
	assert.True(t, copyCalled)
}

func TestView_ActionMenu_CopyToClipboard_Error(t *testing.T) {
	expectedErr := errors.New("copy failed")
	mockAction := &MockResultActionService{
		CopyToClipboardFunc: func(ctx context.Context, result *domain.SearchResult) error {
			return expectedErr
		},
	}

	view := NewView(nil, nil, nil, mockAction)
	view.SetDimensions(80, 24)
	view.Update(messages.SearchCompleted{Results: testSearchResults()})
	view.focusInput = false

	// Open action menu
	view.Update(tea.KeyMsg{Type: tea.KeyEnter})
	view.actionMenu.selected = 0 // Copy plain text

	// Press Enter
	view.Update(tea.KeyMsg{Type: tea.KeyEnter})

	assert.Nil(t, view.actionMenu)
}

func TestView_ActionMenu_CopyToClipboard_NoService(t *testing.T) {
	view := NewView(nil, nil, nil, nil)
	view.SetDimensions(80, 24)
	view.Update(messages.SearchCompleted{Results: testSearchResults()})
	view.focusInput = false

	// Open action menu
	view.Update(tea.KeyMsg{Type: tea.KeyEnter})
	view.actionMenu.selected = 0 // Copy plain text

	// Press Enter
	view.Update(tea.KeyMsg{Type: tea.KeyEnter})

	assert.Nil(t, view.actionMenu)
}

func TestView_ActionMenu_OpenDocument_Success(t *testing.T) {
	openCalled := false
	mockAction := &MockResultActionService{
		OpenDocumentFunc: func(ctx context.Context, result *domain.SearchResult) error {
			openCalled = true
			assert.Equal(t, "Test Document 1", result.Document.Title)
			return nil
		},
	}

	view := NewView(nil, nil, nil, mockAction)
	view.SetDimensions(80, 24)
	view.Update(messages.SearchCompleted{Results: testSearchResults()})
	view.focusInput = false

	// Open action menu
	view.Update(tea.KeyMsg{Type: tea.KeyEnter})
	view.actionMenu.selected = 1 // Open Document

	// Press Enter
	view.Update(tea.KeyMsg{Type: tea.KeyEnter})

	assert.Nil(t, view.actionMenu)
	assert.True(t, openCalled)
}

func TestView_ActionMenu_OpenDocument_Error(t *testing.T) {
	expectedErr := errors.New("open failed")
	mockAction := &MockResultActionService{
		OpenDocumentFunc: func(ctx context.Context, result *domain.SearchResult) error {
			return expectedErr
		},
	}

	view := NewView(nil, nil, nil, mockAction)
	view.SetDimensions(80, 24)
	view.Update(messages.SearchCompleted{Results: testSearchResults()})
	view.focusInput = false

	// Open action menu
	view.Update(tea.KeyMsg{Type: tea.KeyEnter})
	view.actionMenu.selected = 1 // Open Document

	// Press Enter
	view.Update(tea.KeyMsg{Type: tea.KeyEnter})

	assert.Nil(t, view.actionMenu)
}

func TestView_ActionMenu_OpenDocument_NoService(t *testing.T) {
	view := NewView(nil, nil, nil, nil)
	view.SetDimensions(80, 24)
	view.Update(messages.SearchCompleted{Results: testSearchResults()})
	view.focusInput = false

	// Open action menu
	view.Update(tea.KeyMsg{Type: tea.KeyEnter})
	view.actionMenu.selected = 1 // Open Document

	// Press Enter
	view.Update(tea.KeyMsg{Type: tea.KeyEnter})

	assert.Nil(t, view.actionMenu)
}

func TestView_ActionMenu_ExecuteAction_NilResult(t *testing.T) {
	view := NewView(nil, nil, nil, nil)
	view.SetDimensions(80, 24)

	// Manually create action menu with nil result
	view.actionMenu = &ActionMenu{
		actions:  []string{"Copy plain text", "Open Document", "Cancel"},
		selected: 0,
		visible:  true,
		result:   nil,
	}

	// Press Enter
	view.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Should close menu and do nothing
	assert.Nil(t, view.actionMenu)
}

func TestView_RenderActionMenu_NilMenu(t *testing.T) {
	view := NewView(nil, nil, nil, nil)

	output := view.renderActionMenu()

	assert.Equal(t, "", output)
}

func TestView_RenderActionMenu_WithSelection(t *testing.T) {
	view := NewView(nil, nil, nil, nil)
	view.SetDimensions(80, 24)
	view.Update(messages.SearchCompleted{Results: testSearchResults()})
	view.focusInput = false

	// Open action menu
	view.Update(tea.KeyMsg{Type: tea.KeyEnter})
	view.actionMenu.selected = 1

	output := view.renderActionMenu()

	assert.Contains(t, output, "Copy plain text")
	assert.Contains(t, output, "Open Document")
	assert.Contains(t, output, "Cancel")
}

// Edge cases and integration tests

func TestView_Update_ForwardsToComponents(t *testing.T) {
	view := NewView(nil, nil, nil, nil)
	view.SetDimensions(80, 24)

	// Generic message that should be forwarded to components
	type customMsg struct{}
	msg := customMsg{}

	updated, _ := view.Update(msg)

	assert.Equal(t, view, updated)
	// Message is forwarded to input and list components
}

func TestView_Update_KeyEnter_SwitchesToResultsMode(t *testing.T) {
	mock := &MockSearchService{
		SearchFunc: func(ctx context.Context, query string, opts domain.SearchOptions) ([]domain.SearchResult, error) {
			return testSearchResults(), nil
		},
	}
	view := NewView(nil, nil, mock, nil)
	view.SetQuery("test")
	assert.True(t, view.InputFocused())

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	view.Update(msg)

	assert.False(t, view.InputFocused())
}

func TestView_Update_SearchCompleted_ClearsError(t *testing.T) {
	view := NewView(nil, nil, nil, nil)
	view.SetDimensions(80, 24)
	view.err = errors.New("previous error")

	msg := messages.SearchCompleted{Results: testSearchResults(), Err: nil}
	view.Update(msg)

	assert.Nil(t, view.Err())
}

func TestView_ActionMenu_UnknownKey_DoesNothing(t *testing.T) {
	view := NewView(nil, nil, nil, nil)
	view.SetDimensions(80, 24)
	view.Update(messages.SearchCompleted{Results: testSearchResults()})
	view.focusInput = false

	// Open action menu
	view.Update(tea.KeyMsg{Type: tea.KeyEnter})
	initialSelection := view.actionMenu.selected

	// Press unknown key
	view.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})

	// Selection should not change
	assert.Equal(t, initialSelection, view.actionMenu.selected)
	assert.NotNil(t, view.actionMenu)
}

func TestView_Navigation_OnlyWorksInResultsMode(t *testing.T) {
	view := NewView(nil, nil, nil, nil)
	view.Update(messages.SearchCompleted{Results: testSearchResults()})
	view.focusInput = true // In input mode
	initialIndex := view.SelectedIndex()

	// Try to navigate with j/k - should not navigate
	view.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	// Selection should not change in input mode
	assert.Equal(t, initialIndex, view.SelectedIndex())
}

func TestView_MultipleSearches(t *testing.T) {
	mock := &MockSearchService{
		SearchFunc: func(ctx context.Context, query string, opts domain.SearchOptions) ([]domain.SearchResult, error) {
			return testSearchResults(), nil
		},
	}
	view := NewView(nil, nil, mock, nil)
	view.SetDimensions(80, 24)

	// First search
	view.SetQuery("first")
	view.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.False(t, view.InputFocused())

	// Start new search
	view.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	assert.True(t, view.InputFocused())
	assert.Equal(t, "", view.Query())

	// Second search
	view.SetQuery("second")
	view.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.False(t, view.InputFocused())
}

func TestView_WindowSizeMsg_SetsReady(t *testing.T) {
	view := NewView(nil, nil, nil, nil)
	assert.False(t, view.Ready())

	msg := tea.WindowSizeMsg{Width: 100, Height: 50}
	view.Update(msg)

	assert.True(t, view.Ready())
	assert.Equal(t, 100, view.Width())
	assert.Equal(t, 50, view.Height())
}

func TestView_ActionMenu_EnsuresCorrectBehavior(t *testing.T) {
	view := NewView(nil, nil, nil, nil)
	view.SetDimensions(80, 24)
	view.Update(messages.SearchCompleted{Results: testSearchResults()})
	view.focusInput = false

	// Open action menu
	view.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Verify action menu state
	require.NotNil(t, view.actionMenu)
	assert.True(t, view.actionMenu.visible)
	assert.NotNil(t, view.actionMenu.result)
	assert.Equal(t, "Test Document 1", view.actionMenu.result.Document.Title)
	assert.Len(t, view.actionMenu.actions, 3)
	assert.Equal(t, "Copy plain text", view.actionMenu.actions[0])
	assert.Equal(t, "Open Document", view.actionMenu.actions[1])
	assert.Equal(t, "Cancel", view.actionMenu.actions[2])
}

func TestView_ContextPropagation(t *testing.T) {
	type contextKey string
	ctx := context.WithValue(context.Background(), contextKey("test"), "value")

	searchCalled := false
	mock := &MockSearchService{
		SearchFunc: func(receivedCtx context.Context, query string, opts domain.SearchOptions) ([]domain.SearchResult, error) {
			searchCalled = true
			// Verify context is passed through
			val := receivedCtx.Value(contextKey("test"))
			assert.Equal(t, "value", val)
			return testSearchResults(), nil
		},
	}

	view := NewView(nil, nil, mock, nil).WithContext(ctx)
	view.SetQuery("test")

	_, cmd := view.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)
	cmd() // Execute the search command

	assert.True(t, searchCalled)
}

func TestView_ActionMenu_ContextPropagation(t *testing.T) {
	type contextKey string
	ctx := context.WithValue(context.Background(), contextKey("test"), "value")

	copyCalled := false
	mockAction := &MockResultActionService{
		CopyToClipboardFunc: func(receivedCtx context.Context, result *domain.SearchResult) error {
			copyCalled = true
			// Verify context is passed through
			val := receivedCtx.Value(contextKey("test"))
			assert.Equal(t, "value", val)
			return nil
		},
	}

	view := NewView(nil, nil, nil, mockAction).WithContext(ctx)
	view.SetDimensions(80, 24)
	view.Update(messages.SearchCompleted{Results: testSearchResults()})
	view.focusInput = false

	// Open action menu and select copy
	view.Update(tea.KeyMsg{Type: tea.KeyEnter})
	view.actionMenu.selected = 0
	view.Update(tea.KeyMsg{Type: tea.KeyEnter})

	assert.True(t, copyCalled)
}
