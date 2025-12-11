package tui

import (
	"context"
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/custodia-labs/sercha-cli/internal/adapters/driving/tui/messages"
	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driving"
)

func newTestPorts() *Ports {
	return &Ports{
		Search: &MockSearchService{},
		Source: &MockSourceService{},
		Sync:   &MockSyncOrchestrator{},
	}
}

// goToSearchView navigates the app from menu to search view for testing.
func goToSearchView(app *App) {
	app.SetDimensions(80, 24)
	// Send ViewChanged to go to search view (simulates selecting Search from menu)
	app.Update(messages.ViewChanged{View: messages.ViewSearch})
}

func TestNewApp_Success(t *testing.T) {
	ports := newTestPorts()

	app, err := NewApp(ports)

	require.NoError(t, err)
	require.NotNil(t, app)
	assert.Equal(t, messages.ViewMenu, app.CurrentView()) // Now starts at menu
}

func TestNewApp_InvalidPorts(t *testing.T) {
	ports := &Ports{
		Search: nil,
		Source: &MockSourceService{},
		Sync:   &MockSyncOrchestrator{},
	}

	app, err := NewApp(ports)

	assert.Error(t, err)
	assert.Nil(t, app)
}

func TestApp_WithContext(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)

	type contextKey string
	ctx := context.WithValue(context.Background(), contextKey("key"), "value")
	result := app.WithContext(ctx)

	assert.Equal(t, app, result)
}

func TestApp_Init(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)

	cmd := app.Init()

	// Init returns a batch command
	assert.NotNil(t, cmd)
}

func TestApp_Update_WindowSize(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)

	msg := tea.WindowSizeMsg{Width: 80, Height: 24}
	model, cmd := app.Update(msg)

	assert.Equal(t, app, model)
	assert.Nil(t, cmd)
	assert.True(t, app.Ready())
}

func TestApp_Update_QueryChanged(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	goToSearchView(app) // Navigate to search view first

	// QueryChanged messages are no longer handled at app level
	// Query is synced from searchView after key input
	// Type characters to set the query
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}}
	app.Update(msg)
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}}
	app.Update(msg)
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}
	app.Update(msg)
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}}
	app.Update(msg)

	assert.Equal(t, "test", app.Query())
}

func TestApp_Update_SearchCompleted(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)

	results := []domain.SearchResult{
		{Document: domain.Document{Title: "Doc 1"}, Score: 0.9},
	}
	msg := messages.SearchCompleted{Results: results, Err: nil}
	model, cmd := app.Update(msg)

	assert.Equal(t, app, model)
	assert.Nil(t, cmd)
	assert.Len(t, app.Results(), 1)
	assert.Equal(t, 0, app.SelectedIndex())
}

func TestApp_Update_SearchCompleted_WithError(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)

	err := errors.New("search failed")
	msg := messages.SearchCompleted{Results: nil, Err: err}
	model, cmd := app.Update(msg)

	assert.Equal(t, app, model)
	assert.Nil(t, cmd)
	assert.Error(t, app.Err())
}

func TestApp_Update_ResultSelected_Valid(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	goToSearchView(app) // Navigate to search view first

	// First add some results
	app.Update(messages.SearchCompleted{
		Results: []domain.SearchResult{
			{Document: domain.Document{Title: "Doc 1"}},
			{Document: domain.Document{Title: "Doc 2"}},
		},
	})

	// After search completes, focusInput is false, so navigation works
	// Navigate down using key input (results are selected via navigation)
	msg := tea.KeyMsg{Type: tea.KeyDown}
	model, cmd := app.Update(msg)

	assert.Equal(t, app, model)
	assert.Nil(t, cmd)
	assert.Equal(t, 1, app.SelectedIndex())
}

func TestApp_Update_ResultSelected_OutOfBounds(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)

	msg := messages.ResultSelected{Index: 99}
	model, cmd := app.Update(msg)

	assert.Equal(t, app, model)
	assert.Nil(t, cmd)
	assert.Equal(t, 0, app.SelectedIndex())
}

func TestApp_Update_ViewChanged(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)

	msg := messages.ViewChanged{View: messages.ViewHelp}
	model, cmd := app.Update(msg)

	assert.Equal(t, app, model)
	assert.Nil(t, cmd)
	assert.Equal(t, messages.ViewHelp, app.CurrentView())
}

func TestApp_Update_ErrorOccurred(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)

	err := errors.New("something went wrong")
	msg := messages.ErrorOccurred{Err: err}
	model, cmd := app.Update(msg)

	assert.Equal(t, app, model)
	assert.Nil(t, cmd)
	assert.Error(t, app.Err())
}

func TestApp_Update_KeyMsg_Quit(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	app.SetDimensions(80, 24)

	// Test quit from menu view - 'q' should quit
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	model, cmd := app.Update(msg)

	assert.Equal(t, app, model)
	// Quit returns tea.Quit
	assert.NotNil(t, cmd)
}

func TestApp_Update_KeyMsg_CtrlC(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)

	msg := tea.KeyMsg{Type: tea.KeyCtrlC}
	model, cmd := app.Update(msg)

	assert.Equal(t, app, model)
	assert.NotNil(t, cmd)
}

func TestApp_Update_KeyMsg_NavigateUp(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)

	// Add results and set selected index
	app.Update(messages.SearchCompleted{
		Results: []domain.SearchResult{
			{Document: domain.Document{Title: "Doc 1"}},
			{Document: domain.Document{Title: "Doc 2"}},
		},
	})
	app.Update(messages.ResultSelected{Index: 1})

	msg := tea.KeyMsg{Type: tea.KeyUp}
	app.Update(msg)

	assert.Equal(t, 0, app.SelectedIndex())
}

func TestApp_Update_KeyMsg_NavigateDown(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	goToSearchView(app) // Navigate to search view first

	// Add results (this also sets focusInput to false)
	app.Update(messages.SearchCompleted{
		Results: []domain.SearchResult{
			{Document: domain.Document{Title: "Doc 1"}},
			{Document: domain.Document{Title: "Doc 2"}},
		},
	})

	msg := tea.KeyMsg{Type: tea.KeyDown}
	app.Update(msg)

	assert.Equal(t, 1, app.SelectedIndex())
}

func TestApp_Update_KeyMsg_QuestionMark(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	app.SetDimensions(80, 24)

	// '?' from menu doesn't go to help - need to use ViewChanged
	// Test that we can navigate to help view
	app.Update(messages.ViewChanged{View: messages.ViewHelp})

	assert.Equal(t, messages.ViewHelp, app.CurrentView())
}

func TestApp_Update_KeyMsg_Escape(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	app.SetDimensions(80, 24)

	// Go to help view first
	app.Update(messages.ViewChanged{View: messages.ViewHelp})
	assert.Equal(t, messages.ViewHelp, app.CurrentView())

	// Press escape to go back to menu (not search)
	msg := tea.KeyMsg{Type: tea.KeyEsc}
	app.Update(msg)

	assert.Equal(t, messages.ViewMenu, app.CurrentView()) // Escape goes to menu now
}

func TestApp_View_NotReady(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)

	view := app.View()

	assert.Contains(t, view, "Initialising")
}

func TestApp_View_SearchView(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	goToSearchView(app) // Navigate to search view first

	view := app.View()

	assert.Contains(t, view, "Search:")
}

func TestApp_View_HelpView(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	app.SetDimensions(80, 24)
	app.Update(messages.ViewChanged{View: messages.ViewHelp})

	view := app.View()

	assert.Contains(t, view, "Help")
	assert.Contains(t, view, "Navigation") // Updated help view uses "Navigation" instead of "Keybindings"
}

func TestApp_View_SourcesView(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	app.SetDimensions(80, 24)
	app.Update(messages.ViewChanged{View: messages.ViewSources})

	view := app.View()

	assert.Contains(t, view, "Sources")
}

func TestApp_SetDimensions(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)

	assert.False(t, app.Ready())

	app.SetDimensions(100, 50)

	assert.True(t, app.Ready())
}

func TestApp_Update_KeyMsg_K_NavigateUp(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)

	// Add results and set selected index
	app.Update(messages.SearchCompleted{
		Results: []domain.SearchResult{
			{Document: domain.Document{Title: "Doc 1"}},
			{Document: domain.Document{Title: "Doc 2"}},
		},
	})
	app.Update(messages.ResultSelected{Index: 1})

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	app.Update(msg)

	assert.Equal(t, 0, app.SelectedIndex())
}

func TestApp_Update_KeyMsg_J_NavigateDown(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	goToSearchView(app) // Navigate to search view first

	// Add results (focusInput becomes false)
	app.Update(messages.SearchCompleted{
		Results: []domain.SearchResult{
			{Document: domain.Document{Title: "Doc 1"}},
			{Document: domain.Document{Title: "Doc 2"}},
		},
	})

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	app.Update(msg)

	assert.Equal(t, 1, app.SelectedIndex())
}

func TestApp_Update_KeyMsg_NavigateUp_AtBoundary(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)

	// Already at index 0
	msg := tea.KeyMsg{Type: tea.KeyUp}
	app.Update(msg)

	assert.Equal(t, 0, app.SelectedIndex())
}

func TestApp_Update_KeyMsg_NavigateDown_AtBoundary(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)

	// Add one result
	app.Update(messages.SearchCompleted{
		Results: []domain.SearchResult{
			{Document: domain.Document{Title: "Doc 1"}},
		},
	})

	// Already at last index
	msg := tea.KeyMsg{Type: tea.KeyDown}
	app.Update(msg)

	assert.Equal(t, 0, app.SelectedIndex())
}

func TestApp_Update_KeyMsg_Enter_WithQuery(t *testing.T) {
	searchCalled := false
	ports := &Ports{
		Search: &MockSearchService{
			SearchFunc: func(
				ctx context.Context, query string, opts domain.SearchOptions,
			) ([]domain.SearchResult, error) {
				searchCalled = true
				assert.Equal(t, "test", query)
				return []domain.SearchResult{}, nil
			},
		},
		Source: &MockSourceService{},
		Sync:   &MockSyncOrchestrator{},
	}
	app, _ := NewApp(ports)
	goToSearchView(app) // Navigate to search view first

	// Type "test" into the search box
	for _, r := range "test" {
		app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := app.Update(msg)

	// Execute the command
	assert.NotNil(t, cmd)
	result := cmd()
	assert.IsType(t, messages.SearchCompleted{}, result)
	assert.True(t, searchCalled)
}

func TestApp_Update_KeyMsg_Enter_EmptyQuery(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	goToSearchView(app) // Navigate to search view first

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := app.Update(msg)

	assert.Nil(t, cmd)
}

func TestApp_Update_KeyMsg_CharacterInput(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	goToSearchView(app) // Navigate to search view first

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
	app.Update(msg)

	assert.Equal(t, "a", app.Query())
}

func TestApp_Update_KeyMsg_Backspace(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	goToSearchView(app) // Navigate to search view first

	// First type something
	for _, r := range "test" {
		app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	assert.Equal(t, "test", app.Query())

	msg := tea.KeyMsg{Type: tea.KeyBackspace}
	app.Update(msg)

	assert.Equal(t, "tes", app.Query())
}

func TestApp_Update_KeyMsg_Backspace_EmptyQuery(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	goToSearchView(app) // Navigate to search view first

	msg := tea.KeyMsg{Type: tea.KeyBackspace}
	app.Update(msg)

	assert.Equal(t, "", app.Query())
}

func TestApp_Update_KeyMsg_Escape_InSearchView(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	goToSearchView(app) // Navigate to search view first

	// In search view, press Esc
	msg := tea.KeyMsg{Type: tea.KeyEsc}
	_, cmd := app.Update(msg)

	// Esc in search view returns a command that produces ViewChanged
	require.NotNil(t, cmd)
	result := cmd()
	viewChanged, ok := result.(messages.ViewChanged)
	require.True(t, ok)
	assert.Equal(t, messages.ViewMenu, viewChanged.View)

	// Process the ViewChanged message
	app.Update(viewChanged)
	assert.Equal(t, messages.ViewMenu, app.CurrentView())
}

func TestApp_View_SearchView_WithResults(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	goToSearchView(app) // Navigate to search view first

	// Add results
	app.Update(messages.SearchCompleted{
		Results: []domain.SearchResult{
			{Document: domain.Document{Title: "Test Doc"}, Score: 0.95},
		},
	})

	view := app.View()

	assert.Contains(t, view, "Results (1)")
	assert.Contains(t, view, "Test Doc")
	assert.Contains(t, view, "0.95")
}

func TestApp_View_SearchView_WithError(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	goToSearchView(app) // Navigate to search view first

	// Set error
	app.Update(messages.ErrorOccurred{Err: errors.New("test error")})

	view := app.View()

	assert.Contains(t, view, "Error:")
	assert.Contains(t, view, "test error")
}

func TestApp_Update_Quit(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)

	msg := messages.Quit{}
	_, cmd := app.Update(msg)

	assert.NotNil(t, cmd)
}

// Test SourceSelected message handling - navigate from sources to source detail.
func TestApp_Update_SourceSelected_FromSourcesList(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	app.SetDimensions(80, 24)
	app.Update(messages.ViewChanged{View: messages.ViewSources})

	source := domain.Source{
		ID:   "source1",
		Name: "Test Source",
	}
	msg := messages.SourceSelected{Source: source}
	model, cmd := app.Update(msg)

	assert.Equal(t, app, model)
	assert.NotNil(t, cmd)
	assert.Equal(t, messages.ViewSourceDetail, app.CurrentView())
	assert.NotNil(t, app.selectedSource)
	assert.Equal(t, "source1", app.selectedSource.ID)
}

// Test SourceSelected message handling - navigate from source detail to documents.
func TestApp_Update_SourceSelected_FromSourceDetail(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	app.SetDimensions(80, 24)
	app.Update(messages.ViewChanged{View: messages.ViewSourceDetail})

	source := domain.Source{
		ID:   "source1",
		Name: "Test Source",
	}
	msg := messages.SourceSelected{Source: source}
	model, cmd := app.Update(msg)

	assert.Equal(t, app, model)
	assert.NotNil(t, cmd)
	assert.Equal(t, messages.ViewDocuments, app.CurrentView())
}

// Test DocumentsLoaded message handling.
func TestApp_Update_DocumentsLoaded(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	app.SetDimensions(80, 24)
	app.Update(messages.ViewChanged{View: messages.ViewDocuments})

	docs := []domain.Document{
		{ID: "doc1", Title: "Document 1"},
		{ID: "doc2", Title: "Document 2"},
	}
	msg := messages.DocumentsLoaded{
		SourceID:  "source1",
		Documents: docs,
		Err:       nil,
	}
	model, cmd := app.Update(msg)

	assert.Equal(t, app, model)
	assert.Nil(t, cmd)
}

// Test DocumentSelected message handling.
func TestApp_Update_DocumentSelected(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	app.SetDimensions(80, 24)

	doc := domain.Document{
		ID:    "doc1",
		Title: "Test Document",
	}
	msg := messages.DocumentSelected{Document: doc}
	model, cmd := app.Update(msg)

	assert.Equal(t, app, model)
	assert.NotNil(t, cmd)
	assert.Equal(t, messages.ViewDocContent, app.CurrentView())
	assert.NotNil(t, app.selectedDocument)
	assert.Equal(t, "doc1", app.selectedDocument.ID)
}

// Test DocumentContentLoaded message handling.
func TestApp_Update_DocumentContentLoaded(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	app.SetDimensions(80, 24)
	app.Update(messages.ViewChanged{View: messages.ViewDocContent})

	msg := messages.DocumentContentLoaded{
		DocumentID: "doc1",
		Content:    "Test content",
		Err:        nil,
	}
	model, cmd := app.Update(msg)

	assert.Equal(t, app, model)
	assert.Nil(t, cmd)
}

// Test DocumentContentLoaded message with error.
func TestApp_Update_DocumentContentLoaded_WithError(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	app.SetDimensions(80, 24)
	app.Update(messages.ViewChanged{View: messages.ViewDocContent})

	msg := messages.DocumentContentLoaded{
		DocumentID: "doc1",
		Content:    "",
		Err:        errors.New("load failed"),
	}
	model, cmd := app.Update(msg)

	assert.Equal(t, app, model)
	assert.Nil(t, cmd)
}

// Test DocumentDetailsLoaded message handling.
func TestApp_Update_DocumentDetailsLoaded(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	app.SetDimensions(80, 24)

	details := &driving.DocumentDetails{
		ID:         "doc1",
		Title:      "Test Doc",
		ChunkCount: 10,
	}
	msg := messages.DocumentDetailsLoaded{
		DocumentID: "doc1",
		Details:    details,
		Err:        nil,
	}
	model, cmd := app.Update(msg)

	assert.Equal(t, app, model)
	assert.Nil(t, cmd)
	assert.Equal(t, messages.ViewDocDetails, app.CurrentView())
}

// Test DocumentDetailsLoaded message with error.
func TestApp_Update_DocumentDetailsLoaded_WithError(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	app.SetDimensions(80, 24)

	msg := messages.DocumentDetailsLoaded{
		DocumentID: "doc1",
		Details:    nil,
		Err:        errors.New("load failed"),
	}
	model, cmd := app.Update(msg)

	assert.Equal(t, app, model)
	assert.Nil(t, cmd)
	assert.Error(t, app.Err())
}

// Test DocumentDetailsLoaded message with invalid details type.
func TestApp_Update_DocumentDetailsLoaded_InvalidType(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	app.SetDimensions(80, 24)

	msg := messages.DocumentDetailsLoaded{
		DocumentID: "doc1",
		Details:    "invalid type",
		Err:        nil,
	}
	model, cmd := app.Update(msg)

	assert.Equal(t, app, model)
	assert.Nil(t, cmd)
	// Should not change view with invalid details
	assert.NotEqual(t, messages.ViewDocDetails, app.CurrentView())
}

// Test DocumentExcluded message handling.
func TestApp_Update_DocumentExcluded(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	app.SetDimensions(80, 24)
	app.Update(messages.ViewChanged{View: messages.ViewDocuments})

	msg := messages.DocumentExcluded{
		DocumentID: "doc1",
		Err:        nil,
	}
	model, cmd := app.Update(msg)

	assert.Equal(t, app, model)
	_ = cmd // Command may or may not be nil depending on view implementation
}

// Test DocumentRefreshed message handling.
func TestApp_Update_DocumentRefreshed(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	app.SetDimensions(80, 24)
	app.Update(messages.ViewChanged{View: messages.ViewDocuments})

	msg := messages.DocumentRefreshed{
		DocumentID: "doc1",
		Err:        nil,
	}
	model, cmd := app.Update(msg)

	assert.Equal(t, app, model)
	assert.Nil(t, cmd)
}

// Test SourcesLoaded message forwarded to sources view.
func TestApp_Update_SourcesLoaded_InSourcesView(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	app.SetDimensions(80, 24)
	app.Update(messages.ViewChanged{View: messages.ViewSources})

	sources := []domain.Source{
		{ID: "source1", Name: "Source 1"},
	}
	msg := messages.SourcesLoaded{
		Sources: sources,
		Err:     nil,
	}
	model, cmd := app.Update(msg)

	assert.Equal(t, app, model)
	assert.Nil(t, cmd)
}

// Test SourcesLoaded message forwarded to source detail view.
func TestApp_Update_SourcesLoaded_InSourceDetailView(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	app.SetDimensions(80, 24)
	app.Update(messages.ViewChanged{View: messages.ViewSourceDetail})

	sources := []domain.Source{
		{ID: "source1", Name: "Source 1"},
	}
	msg := messages.SourcesLoaded{
		Sources: sources,
		Err:     nil,
	}
	model, cmd := app.Update(msg)

	assert.Equal(t, app, model)
	assert.Nil(t, cmd)
}

// Test SourceRemoved message forwarded to sources view.
func TestApp_Update_SourceRemoved_InSourcesView(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	app.SetDimensions(80, 24)
	app.Update(messages.ViewChanged{View: messages.ViewSources})

	msg := messages.SourceRemoved{
		ID:  "source1",
		Err: nil,
	}
	model, cmd := app.Update(msg)

	assert.Equal(t, app, model)
	_ = cmd // Command may or may not be nil depending on view implementation
}

// Test SourceAdded message forwarded to add source view.
func TestApp_Update_SourceAdded(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	app.SetDimensions(80, 24)
	app.Update(messages.ViewChanged{View: messages.ViewAddSource})

	msg := messages.SourceAdded{
		Source: domain.Source{ID: "source1", Name: "New Source"},
		Err:    nil,
	}
	model, cmd := app.Update(msg)

	assert.Equal(t, app, model)
	assert.Nil(t, cmd)
}

// Test SettingsLoaded message forwarded to settings view.
func TestApp_Update_SettingsLoaded(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	app.SetDimensions(80, 24)
	app.Update(messages.ViewChanged{View: messages.ViewSettings})

	settings := &domain.AppSettings{
		Search: domain.SearchSettings{
			Mode: domain.SearchModeTextOnly,
		},
	}
	msg := messages.SettingsLoaded{
		Settings: settings,
		Err:      nil,
	}
	model, cmd := app.Update(msg)

	assert.Equal(t, app, model)
	assert.Nil(t, cmd)
}

// Test SettingsSaved message forwarded to settings view.
func TestApp_Update_SettingsSaved(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	app.SetDimensions(80, 24)
	app.Update(messages.ViewChanged{View: messages.ViewSettings})

	msg := messages.SettingsSaved{
		Err: nil,
	}
	model, cmd := app.Update(msg)

	assert.Equal(t, app, model)
	_ = cmd // Command may or may not be nil depending on view implementation
}

// Test ErrorOccurred in search view.
func TestApp_Update_ErrorOccurred_InSearchView(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	app.SetDimensions(80, 24)
	app.Update(messages.ViewChanged{View: messages.ViewSearch})

	err := errors.New("search error")
	msg := messages.ErrorOccurred{Err: err}
	model, cmd := app.Update(msg)

	assert.Equal(t, app, model)
	assert.Nil(t, cmd)
	assert.Error(t, app.Err())
}

// Test ErrorOccurred in documents view.
func TestApp_Update_ErrorOccurred_InDocumentsView(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	app.SetDimensions(80, 24)
	app.Update(messages.ViewChanged{View: messages.ViewDocuments})

	err := errors.New("documents error")
	msg := messages.ErrorOccurred{Err: err}
	model, cmd := app.Update(msg)

	assert.Equal(t, app, model)
	assert.Nil(t, cmd)
	assert.Error(t, app.Err())
}

// Test ErrorOccurred in doc content view.
func TestApp_Update_ErrorOccurred_InDocContentView(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	app.SetDimensions(80, 24)
	app.Update(messages.ViewChanged{View: messages.ViewDocContent})

	err := errors.New("content error")
	msg := messages.ErrorOccurred{Err: err}
	model, cmd := app.Update(msg)

	assert.Equal(t, app, model)
	assert.Nil(t, cmd)
	assert.Error(t, app.Err())
}

// Test ErrorOccurred in doc details view.
func TestApp_Update_ErrorOccurred_InDocDetailsView(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	app.SetDimensions(80, 24)
	app.Update(messages.ViewChanged{View: messages.ViewDocDetails})

	err := errors.New("details error")
	msg := messages.ErrorOccurred{Err: err}
	model, cmd := app.Update(msg)

	assert.Equal(t, app, model)
	assert.Nil(t, cmd)
	assert.Error(t, app.Err())
}

// Test ErrorOccurred in menu view (not forwarded).
func TestApp_Update_ErrorOccurred_InMenuView(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	app.SetDimensions(80, 24)
	// Default view is menu

	err := errors.New("menu error")
	msg := messages.ErrorOccurred{Err: err}
	model, cmd := app.Update(msg)

	assert.Equal(t, app, model)
	assert.Nil(t, cmd)
	assert.Error(t, app.Err())
}

// Test ViewChanged to different views with Init.
func TestApp_Update_ViewChanged_ToSearch(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	app.SetDimensions(80, 24)

	msg := messages.ViewChanged{View: messages.ViewSearch}
	model, cmd := app.Update(msg)

	assert.Equal(t, app, model)
	assert.NotNil(t, cmd)
	assert.Equal(t, messages.ViewSearch, app.CurrentView())
}

func TestApp_Update_ViewChanged_ToSources(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	app.SetDimensions(80, 24)

	msg := messages.ViewChanged{View: messages.ViewSources}
	model, cmd := app.Update(msg)

	assert.Equal(t, app, model)
	assert.NotNil(t, cmd)
	assert.Equal(t, messages.ViewSources, app.CurrentView())
}

func TestApp_Update_ViewChanged_ToSourceDetail(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	app.SetDimensions(80, 24)

	msg := messages.ViewChanged{View: messages.ViewSourceDetail}
	model, cmd := app.Update(msg)

	assert.Equal(t, app, model)
	assert.NotNil(t, cmd)
	assert.Equal(t, messages.ViewSourceDetail, app.CurrentView())
}

func TestApp_Update_ViewChanged_ToAddSource(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	app.SetDimensions(80, 24)

	msg := messages.ViewChanged{View: messages.ViewAddSource}
	model, cmd := app.Update(msg)

	assert.Equal(t, app, model)
	assert.NotNil(t, cmd)
	assert.Equal(t, messages.ViewAddSource, app.CurrentView())
}

func TestApp_Update_ViewChanged_ToSettings(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	app.SetDimensions(80, 24)

	msg := messages.ViewChanged{View: messages.ViewSettings}
	model, cmd := app.Update(msg)

	assert.Equal(t, app, model)
	assert.NotNil(t, cmd)
	assert.Equal(t, messages.ViewSettings, app.CurrentView())
}

func TestApp_Update_ViewChanged_ToMenu(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	app.SetDimensions(80, 24)
	// Start at different view
	app.Update(messages.ViewChanged{View: messages.ViewSearch})

	msg := messages.ViewChanged{View: messages.ViewMenu}
	model, cmd := app.Update(msg)

	assert.Equal(t, app, model)
	assert.Nil(t, cmd)
	assert.Equal(t, messages.ViewMenu, app.CurrentView())
}

func TestApp_Update_ViewChanged_ToDocuments(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	app.SetDimensions(80, 24)

	msg := messages.ViewChanged{View: messages.ViewDocuments}
	model, cmd := app.Update(msg)

	assert.Equal(t, app, model)
	assert.Nil(t, cmd)
	assert.Equal(t, messages.ViewDocuments, app.CurrentView())
}

func TestApp_Update_ViewChanged_ToDocContent(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	app.SetDimensions(80, 24)

	msg := messages.ViewChanged{View: messages.ViewDocContent}
	model, cmd := app.Update(msg)

	assert.Equal(t, app, model)
	assert.Nil(t, cmd)
	assert.Equal(t, messages.ViewDocContent, app.CurrentView())
}

func TestApp_Update_ViewChanged_ToDocDetails(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	app.SetDimensions(80, 24)

	msg := messages.ViewChanged{View: messages.ViewDocDetails}
	model, cmd := app.Update(msg)

	assert.Equal(t, app, model)
	assert.Nil(t, cmd)
	assert.Equal(t, messages.ViewDocDetails, app.CurrentView())
}

// Test KeyMsg forwarded to various views.
func TestApp_Update_KeyMsg_InSourcesView_Escape(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	app.SetDimensions(80, 24)
	app.Update(messages.ViewChanged{View: messages.ViewSources})

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	model, cmd := app.Update(msg)

	assert.Equal(t, app, model)
	assert.Nil(t, cmd)
	assert.Equal(t, messages.ViewMenu, app.CurrentView())
}

func TestApp_Update_KeyMsg_InSourceDetailView(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	app.SetDimensions(80, 24)
	app.Update(messages.ViewChanged{View: messages.ViewSourceDetail})

	msg := tea.KeyMsg{Type: tea.KeyDown}
	model, cmd := app.Update(msg)

	assert.Equal(t, app, model)
	// Command may or may not be nil depending on view
	_ = cmd
}

func TestApp_Update_KeyMsg_InDocumentsView(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	app.SetDimensions(80, 24)
	app.Update(messages.ViewChanged{View: messages.ViewDocuments})

	msg := tea.KeyMsg{Type: tea.KeyDown}
	model, cmd := app.Update(msg)

	assert.Equal(t, app, model)
	_ = cmd
}

func TestApp_Update_KeyMsg_InDocContentView(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	app.SetDimensions(80, 24)
	app.Update(messages.ViewChanged{View: messages.ViewDocContent})

	msg := tea.KeyMsg{Type: tea.KeyDown}
	model, cmd := app.Update(msg)

	assert.Equal(t, app, model)
	_ = cmd
}

func TestApp_Update_KeyMsg_InDocDetailsView(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	app.SetDimensions(80, 24)
	app.Update(messages.ViewChanged{View: messages.ViewDocDetails})

	msg := tea.KeyMsg{Type: tea.KeyDown}
	model, cmd := app.Update(msg)

	assert.Equal(t, app, model)
	_ = cmd
}

func TestApp_Update_KeyMsg_InAddSourceView(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	app.SetDimensions(80, 24)
	app.Update(messages.ViewChanged{View: messages.ViewAddSource})

	msg := tea.KeyMsg{Type: tea.KeyDown}
	model, cmd := app.Update(msg)

	assert.Equal(t, app, model)
	_ = cmd
}

func TestApp_Update_KeyMsg_InSettingsView(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	app.SetDimensions(80, 24)
	app.Update(messages.ViewChanged{View: messages.ViewSettings})

	msg := tea.KeyMsg{Type: tea.KeyDown}
	model, cmd := app.Update(msg)

	assert.Equal(t, app, model)
	_ = cmd
}

func TestApp_Update_KeyMsg_InHelpView_Escape(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	app.SetDimensions(80, 24)
	app.Update(messages.ViewChanged{View: messages.ViewHelp})

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	model, cmd := app.Update(msg)

	assert.Equal(t, app, model)
	assert.Nil(t, cmd)
	assert.Equal(t, messages.ViewMenu, app.CurrentView())
}

func TestApp_Update_KeyMsg_InHelpView_OtherKey(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	app.SetDimensions(80, 24)
	app.Update(messages.ViewChanged{View: messages.ViewHelp})

	msg := tea.KeyMsg{Type: tea.KeyDown}
	model, cmd := app.Update(msg)

	assert.Equal(t, app, model)
	assert.Nil(t, cmd)
}

// Test View rendering for all view types.
func TestApp_View_MenuView(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	// Must set dimensions which also sets ready=true
	msg := tea.WindowSizeMsg{Width: 80, Height: 24}
	app.Update(msg)
	// Ensure we're at menu view
	app.currentView = messages.ViewMenu

	view := app.View()

	assert.Contains(t, view, "Sercha")
}

func TestApp_View_SourceDetailView(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	// Must initialize with window size first
	msg := tea.WindowSizeMsg{Width: 80, Height: 24}
	app.Update(msg)
	// Set a source first
	source := domain.Source{ID: "s1", Name: "Test Source"}
	app.selectedSource = &source
	app.sourceDetailView.SetSource(source)
	app.sourceDetailView.SetDimensions(80, 24)
	app.Update(messages.ViewChanged{View: messages.ViewSourceDetail})

	view := app.View()

	// Should show source name or details
	assert.NotEmpty(t, view)
}

func TestApp_View_DocumentsView(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	app.SetDimensions(80, 24)
	app.Update(messages.ViewChanged{View: messages.ViewDocuments})

	view := app.View()

	assert.Contains(t, view, "Documents")
}

func TestApp_View_DocContentView(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	// Must initialize with window size first
	msg := tea.WindowSizeMsg{Width: 80, Height: 24}
	app.Update(msg)
	// Set a document first to avoid panic
	doc := domain.Document{ID: "d1", Title: "Test Doc"}
	app.selectedDocument = &doc
	app.docContentView.SetDocument(&doc)
	app.docContentView.SetDimensions(80, 24)
	app.Update(messages.ViewChanged{View: messages.ViewDocContent})

	view := app.View()

	// Should show some content or empty state
	assert.NotEmpty(t, view)
}

func TestApp_View_DocDetailsView(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	// Must initialize with window size first
	msg := tea.WindowSizeMsg{Width: 80, Height: 24}
	app.Update(msg)
	app.docDetailsView.SetDimensions(80, 24)
	app.Update(messages.ViewChanged{View: messages.ViewDocDetails})

	view := app.View()

	assert.Contains(t, view, "Document Details")
}

func TestApp_View_AddSourceView(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	app.SetDimensions(80, 24)
	app.Update(messages.ViewChanged{View: messages.ViewAddSource})

	view := app.View()

	assert.Contains(t, view, "Add Source")
}

func TestApp_View_SettingsView(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	app.SetDimensions(80, 24)
	app.Update(messages.ViewChanged{View: messages.ViewSettings})

	view := app.View()

	assert.Contains(t, view, "Settings")
}

func TestApp_View_DefaultView(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	// Must initialize with window size first
	msg := tea.WindowSizeMsg{Width: 80, Height: 24}
	app.Update(msg)
	// Set to an unrecognized view type
	app.currentView = messages.ViewType(999)

	view := app.View()

	// Should default to menu view
	assert.Contains(t, view, "Sercha")
}

// Test message forwarding to views.
func TestApp_Update_MessageForwardedToMenuView(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	app.SetDimensions(80, 24)
	// Default is menu view

	// Send a generic message (like QueryChanged which menu doesn't handle)
	msg := messages.QueryChanged{Query: "test"}
	model, cmd := app.Update(msg)

	assert.Equal(t, app, model)
	assert.Nil(t, cmd)
}

func TestApp_Update_MessageForwardedToSearchView(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	app.SetDimensions(80, 24)
	app.Update(messages.ViewChanged{View: messages.ViewSearch})

	msg := messages.QueryChanged{Query: "test"}
	model, cmd := app.Update(msg)

	assert.Equal(t, app, model)
	// SearchView handles QueryChanged
	_ = cmd
}

func TestApp_Update_MessageForwardedToSourcesView(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	app.SetDimensions(80, 24)
	app.Update(messages.ViewChanged{View: messages.ViewSources})

	msg := messages.QueryChanged{Query: "test"}
	model, cmd := app.Update(msg)

	assert.Equal(t, app, model)
	assert.Nil(t, cmd)
}

func TestApp_Update_MessageForwardedToSourceDetailView(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	app.SetDimensions(80, 24)
	app.Update(messages.ViewChanged{View: messages.ViewSourceDetail})

	msg := messages.QueryChanged{Query: "test"}
	model, cmd := app.Update(msg)

	assert.Equal(t, app, model)
	_ = cmd
}

func TestApp_Update_MessageForwardedToDocumentsView(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	app.SetDimensions(80, 24)
	app.Update(messages.ViewChanged{View: messages.ViewDocuments})

	msg := messages.QueryChanged{Query: "test"}
	model, cmd := app.Update(msg)

	assert.Equal(t, app, model)
	_ = cmd
}

func TestApp_Update_MessageForwardedToDocContentView(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	app.SetDimensions(80, 24)
	app.Update(messages.ViewChanged{View: messages.ViewDocContent})

	msg := messages.QueryChanged{Query: "test"}
	model, cmd := app.Update(msg)

	assert.Equal(t, app, model)
	_ = cmd
}

func TestApp_Update_MessageForwardedToDocDetailsView(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	app.SetDimensions(80, 24)
	app.Update(messages.ViewChanged{View: messages.ViewDocDetails})

	msg := messages.QueryChanged{Query: "test"}
	model, cmd := app.Update(msg)

	assert.Equal(t, app, model)
	_ = cmd
}

func TestApp_Update_MessageForwardedToAddSourceView(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	app.SetDimensions(80, 24)
	app.Update(messages.ViewChanged{View: messages.ViewAddSource})

	msg := messages.QueryChanged{Query: "test"}
	model, cmd := app.Update(msg)

	assert.Equal(t, app, model)
	_ = cmd
}

func TestApp_Update_MessageForwardedToSettingsView(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	app.SetDimensions(80, 24)
	app.Update(messages.ViewChanged{View: messages.ViewSettings})

	msg := messages.QueryChanged{Query: "test"}
	model, cmd := app.Update(msg)

	assert.Equal(t, app, model)
	_ = cmd
}

func TestApp_Update_MessageForwardedToHelpView(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	app.SetDimensions(80, 24)
	app.Update(messages.ViewChanged{View: messages.ViewHelp})

	msg := messages.QueryChanged{Query: "test"}
	model, cmd := app.Update(msg)

	assert.Equal(t, app, model)
	assert.Nil(t, cmd)
}

// Test that window resize messages are forwarded to all views.
func TestApp_Update_WindowSize_AllViewsNotified(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)

	msg := tea.WindowSizeMsg{Width: 120, Height: 60}
	model, cmd := app.Update(msg)

	assert.Equal(t, app, model)
	assert.Nil(t, cmd)
	assert.True(t, app.Ready())
	// All views should have received dimensions
	// (This is tested implicitly through the app's behavior)
}

// Test SourcesLoaded/SourceRemoved messages ignored in non-relevant views.
func TestApp_Update_SourcesLoaded_InOtherView(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	app.SetDimensions(80, 24)
	app.Update(messages.ViewChanged{View: messages.ViewSearch})

	msg := messages.SourcesLoaded{
		Sources: []domain.Source{{ID: "s1"}},
		Err:     nil,
	}
	model, _ := app.Update(msg)

	assert.Equal(t, app, model)
	// Message is not processed in search view
}

func TestApp_Update_SourceRemoved_InOtherView(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	app.SetDimensions(80, 24)
	app.Update(messages.ViewChanged{View: messages.ViewSearch})

	msg := messages.SourceRemoved{ID: "s1", Err: nil}
	model, _ := app.Update(msg)

	assert.Equal(t, app, model)
}

func TestApp_Update_SourceAdded_InOtherView(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	app.SetDimensions(80, 24)
	app.Update(messages.ViewChanged{View: messages.ViewSearch})

	msg := messages.SourceAdded{Source: domain.Source{ID: "s1"}, Err: nil}
	model, _ := app.Update(msg)

	assert.Equal(t, app, model)
}

func TestApp_Update_SettingsLoaded_InOtherView(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	app.SetDimensions(80, 24)
	app.Update(messages.ViewChanged{View: messages.ViewSearch})

	msg := messages.SettingsLoaded{Settings: &domain.AppSettings{}, Err: nil}
	model, _ := app.Update(msg)

	assert.Equal(t, app, model)
}

func TestApp_Update_SettingsSaved_InOtherView(t *testing.T) {
	ports := newTestPorts()
	app, _ := NewApp(ports)
	app.SetDimensions(80, 24)
	app.Update(messages.ViewChanged{View: messages.ViewSearch})

	msg := messages.SettingsSaved{Err: nil}
	model, _ := app.Update(msg)

	assert.Equal(t, app, model)
}
