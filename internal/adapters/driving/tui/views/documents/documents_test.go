package documents

import (
	"context"
	"errors"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/custodia-labs/sercha-cli/internal/adapters/driving/tui/messages"
	"github.com/custodia-labs/sercha-cli/internal/adapters/driving/tui/styles"
	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driving"
)

// MockDocumentService implements driving.DocumentService for testing.
type MockDocumentService struct {
	ListBySourceFunc func(ctx context.Context, sourceID string) ([]domain.Document, error)
	GetFunc          func(ctx context.Context, documentID string) (*domain.Document, error)
	GetContentFunc   func(ctx context.Context, documentID string) (string, error)
	GetDetailsFunc   func(ctx context.Context, documentID string) (*driving.DocumentDetails, error)
	ExcludeFunc      func(ctx context.Context, documentID string, reason string) error
	RefreshFunc      func(ctx context.Context, documentID string) error
	OpenFunc         func(ctx context.Context, documentID string) error
}

func (m *MockDocumentService) ListBySource(ctx context.Context, sourceID string) ([]domain.Document, error) {
	if m.ListBySourceFunc != nil {
		return m.ListBySourceFunc(ctx, sourceID)
	}
	return []domain.Document{}, nil
}

func (m *MockDocumentService) Get(ctx context.Context, documentID string) (*domain.Document, error) {
	if m.GetFunc != nil {
		return m.GetFunc(ctx, documentID)
	}
	return nil, nil
}

func (m *MockDocumentService) GetContent(ctx context.Context, documentID string) (string, error) {
	if m.GetContentFunc != nil {
		return m.GetContentFunc(ctx, documentID)
	}
	return "", nil
}

func (m *MockDocumentService) GetDetails(ctx context.Context, documentID string) (*driving.DocumentDetails, error) {
	if m.GetDetailsFunc != nil {
		return m.GetDetailsFunc(ctx, documentID)
	}
	return nil, nil
}

func (m *MockDocumentService) Exclude(ctx context.Context, documentID, reason string) error {
	if m.ExcludeFunc != nil {
		return m.ExcludeFunc(ctx, documentID, reason)
	}
	return nil
}

func (m *MockDocumentService) Refresh(ctx context.Context, documentID string) error {
	if m.RefreshFunc != nil {
		return m.RefreshFunc(ctx, documentID)
	}
	return nil
}

func (m *MockDocumentService) Open(ctx context.Context, documentID string) error {
	if m.OpenFunc != nil {
		return m.OpenFunc(ctx, documentID)
	}
	return nil
}

func TestNewView(t *testing.T) {
	s := styles.DefaultStyles()
	mock := &MockDocumentService{}

	view := NewView(s, mock)

	require.NotNil(t, view)
	assert.False(t, view.ready)
	assert.Empty(t, view.documents)
}

func TestNewView_NilParams(t *testing.T) {
	view := NewView(nil, nil)

	require.NotNil(t, view)
	assert.Nil(t, view.styles)
	assert.Nil(t, view.documentService)
}

func TestView_SetSource(t *testing.T) {
	mock := &MockDocumentService{
		ListBySourceFunc: func(ctx context.Context, sourceID string) ([]domain.Document, error) {
			assert.Equal(t, "src-1", sourceID)
			return []domain.Document{
				{ID: "doc-1", Title: "Doc 1"},
			}, nil
		},
	}
	view := NewView(nil, mock)

	source := domain.Source{ID: "src-1", Name: "Test Source"}
	cmd := view.SetSource(source)

	require.NotNil(t, cmd)
	assert.Equal(t, "src-1", view.source.ID)
	assert.Equal(t, 0, view.selected)
	assert.False(t, view.showingMenu)

	// Execute command
	result := cmd()
	loaded, ok := result.(messages.DocumentsLoaded)
	require.True(t, ok)
	assert.Equal(t, "src-1", loaded.SourceID)
	assert.Len(t, loaded.Documents, 1)
}

func TestView_Init(t *testing.T) {
	view := NewView(nil, nil)

	cmd := view.Init()

	assert.Nil(t, cmd)
}

func TestView_Update_WindowSize(t *testing.T) {
	view := NewView(nil, nil)

	msg := tea.WindowSizeMsg{Width: 80, Height: 24}
	updated, cmd := view.Update(msg)

	assert.Equal(t, view, updated)
	assert.Nil(t, cmd)
	assert.True(t, view.ready)
	assert.Equal(t, 80, view.width)
	assert.Equal(t, 24, view.height)
}

func TestView_Update_DocumentsLoaded(t *testing.T) {
	view := NewView(nil, nil)
	view.source = &domain.Source{ID: "src-1"}

	docs := []domain.Document{
		{ID: "doc-1", Title: "Doc 1"},
		{ID: "doc-2", Title: "Doc 2"},
	}
	msg := messages.DocumentsLoaded{SourceID: "src-1", Documents: docs, Err: nil}
	updated, cmd := view.Update(msg)

	assert.Equal(t, view, updated)
	assert.Nil(t, cmd)
	assert.Len(t, view.documents, 2)
	assert.False(t, view.loading)
}

func TestView_Update_DocumentsLoaded_Error(t *testing.T) {
	view := NewView(nil, nil)
	view.source = &domain.Source{ID: "src-1"}

	msg := messages.DocumentsLoaded{SourceID: "src-1", Documents: nil, Err: errors.New("failed")}
	updated, cmd := view.Update(msg)

	assert.Equal(t, view, updated)
	assert.Nil(t, cmd)
	assert.Error(t, view.err)
}

func TestView_Update_DocumentsLoaded_DifferentSource(t *testing.T) {
	view := NewView(nil, nil)
	view.source = &domain.Source{ID: "src-1"}
	view.documents = []domain.Document{{ID: "old-doc"}}

	// Note: The current implementation doesn't validate source ID match,
	// it will update documents regardless. This tests actual behaviour.
	msg := messages.DocumentsLoaded{SourceID: "src-2", Documents: []domain.Document{{ID: "new-doc"}}}
	view.Update(msg)

	// Current behaviour: documents are updated regardless of source ID
	assert.Len(t, view.documents, 1)
	assert.Equal(t, "new-doc", view.documents[0].ID)
}

func TestView_Update_KeyMsg_Navigation(t *testing.T) {
	view := NewView(nil, nil)
	view.width = 80
	view.height = 24
	view.ready = true
	view.documents = []domain.Document{
		{ID: "doc-1", Title: "Doc 1"},
		{ID: "doc-2", Title: "Doc 2"},
		{ID: "doc-3", Title: "Doc 3"},
	}

	// Test down navigation
	msg := tea.KeyMsg{Type: tea.KeyDown}
	view.Update(msg)
	assert.Equal(t, 1, view.selected)

	// Test j navigation
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	view.Update(msg)
	assert.Equal(t, 2, view.selected)

	// Test boundary (should not go past last)
	msg = tea.KeyMsg{Type: tea.KeyDown}
	view.Update(msg)
	assert.Equal(t, 2, view.selected)

	// Test up navigation
	msg = tea.KeyMsg{Type: tea.KeyUp}
	view.Update(msg)
	assert.Equal(t, 1, view.selected)

	// Test k navigation
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	view.Update(msg)
	assert.Equal(t, 0, view.selected)

	// Test boundary (should not go below 0)
	msg = tea.KeyMsg{Type: tea.KeyUp}
	view.Update(msg)
	assert.Equal(t, 0, view.selected)
}

func TestView_Update_KeyMsg_OpenMenu(t *testing.T) {
	view := NewView(nil, nil)
	view.documents = []domain.Document{
		{ID: "doc-1", Title: "Doc 1"},
	}

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	view.Update(msg)

	assert.True(t, view.showingMenu)
	assert.Equal(t, ActionShowContent, view.menuSelected)
}

func TestView_Update_KeyMsg_Back(t *testing.T) {
	view := NewView(nil, nil)
	view.documents = []domain.Document{{ID: "doc-1"}}

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	_, cmd := view.Update(msg)

	require.NotNil(t, cmd)
	result := cmd()
	changed, ok := result.(messages.ViewChanged)
	assert.True(t, ok)
	assert.Equal(t, messages.ViewSourceDetail, changed.View)
}

func TestView_HandleMenuKeyMsg_Navigation(t *testing.T) {
	view := NewView(nil, nil)
	view.documents = []domain.Document{{ID: "doc-1"}}
	view.showingMenu = true
	view.menuSelected = ActionShowContent

	// Navigate down
	msg := tea.KeyMsg{Type: tea.KeyDown}
	view.Update(msg)
	assert.Equal(t, ActionShowDetails, view.menuSelected)

	// Navigate up
	msg = tea.KeyMsg{Type: tea.KeyUp}
	view.Update(msg)
	assert.Equal(t, ActionShowContent, view.menuSelected)
}

func TestView_HandleMenuKeyMsg_Cancel(t *testing.T) {
	view := NewView(nil, nil)
	view.documents = []domain.Document{{ID: "doc-1"}}
	view.showingMenu = true

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	view.Update(msg)

	assert.False(t, view.showingMenu)
}

func TestView_HandleMenuSelect_ShowContent(t *testing.T) {
	view := NewView(nil, nil)
	view.documents = []domain.Document{{ID: "doc-1", Title: "Test Doc"}}
	view.showingMenu = true
	view.menuSelected = ActionShowContent

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := view.Update(msg)

	assert.False(t, view.showingMenu)
	require.NotNil(t, cmd)

	result := cmd()
	selected, ok := result.(messages.DocumentSelected)
	assert.True(t, ok)
	assert.Equal(t, "doc-1", selected.Document.ID)
}

func TestView_HandleMenuSelect_ShowDetails(t *testing.T) {
	detailsCalled := false
	mock := &MockDocumentService{
		GetDetailsFunc: func(ctx context.Context, documentID string) (*driving.DocumentDetails, error) {
			detailsCalled = true
			assert.Equal(t, "doc-1", documentID)
			return &driving.DocumentDetails{ID: "doc-1", Title: "Test"}, nil
		},
	}
	view := NewView(nil, mock)
	view.documents = []domain.Document{{ID: "doc-1"}}
	view.showingMenu = true
	view.menuSelected = ActionShowDetails

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := view.Update(msg)

	assert.False(t, view.showingMenu)
	require.NotNil(t, cmd)

	result := cmd()
	loaded, ok := result.(messages.DocumentDetailsLoaded)
	assert.True(t, ok)
	assert.True(t, detailsCalled)
	assert.Equal(t, "doc-1", loaded.DocumentID)
}

func TestView_HandleMenuSelect_OpenDocument(t *testing.T) {
	openCalled := false
	mock := &MockDocumentService{
		OpenFunc: func(ctx context.Context, documentID string) error {
			openCalled = true
			return nil
		},
	}
	view := NewView(nil, mock)
	view.documents = []domain.Document{{ID: "doc-1"}}
	view.showingMenu = true
	view.menuSelected = ActionOpenDocument

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := view.Update(msg)

	require.NotNil(t, cmd)
	cmd()
	assert.True(t, openCalled)
}

func TestView_HandleMenuSelect_Refresh(t *testing.T) {
	refreshCalled := false
	mock := &MockDocumentService{
		RefreshFunc: func(ctx context.Context, documentID string) error {
			refreshCalled = true
			return nil
		},
	}
	view := NewView(nil, mock)
	view.documents = []domain.Document{{ID: "doc-1"}}
	view.showingMenu = true
	view.menuSelected = ActionRefresh

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := view.Update(msg)

	require.NotNil(t, cmd)
	cmd()
	assert.True(t, refreshCalled)
}

func TestView_HandleMenuSelect_Exclude(t *testing.T) {
	excludeCalled := false
	mock := &MockDocumentService{
		ExcludeFunc: func(ctx context.Context, documentID string, reason string) error {
			excludeCalled = true
			assert.Equal(t, "doc-1", documentID)
			return nil
		},
	}
	view := NewView(nil, mock)
	view.documents = []domain.Document{{ID: "doc-1"}}
	view.showingMenu = true
	view.menuSelected = ActionExclude

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := view.Update(msg)

	require.NotNil(t, cmd)
	cmd()
	assert.True(t, excludeCalled)
}

func TestView_HandleMenuSelect_Cancel(t *testing.T) {
	view := NewView(nil, nil)
	view.documents = []domain.Document{{ID: "doc-1"}}
	view.showingMenu = true
	view.menuSelected = ActionCancel

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	view.Update(msg)

	assert.False(t, view.showingMenu)
}

func TestView_View_EmptyState(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewView(s, nil)
	view.width = 80
	view.height = 24
	view.ready = true
	view.source = &domain.Source{ID: "src-1", Name: "Test"}
	view.documents = []domain.Document{}

	output := view.View()

	assert.Contains(t, output, "No documents")
}

func TestView_View_WithDocuments(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewView(s, nil)
	view.width = 80
	view.height = 24
	view.ready = true
	view.source = &domain.Source{ID: "src-1", Name: "Test"}
	view.documents = []domain.Document{
		{ID: "doc-1", Title: "Document One", URI: "/path/to/doc1.md"},
		{ID: "doc-2", Title: "Document Two", URI: "/path/to/doc2.md"},
	}

	output := view.View()

	assert.Contains(t, output, "Document One")
	assert.Contains(t, output, "Document Two")
}

func TestView_View_Loading(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewView(s, nil)
	view.width = 80
	view.height = 24
	view.ready = true
	view.loading = true

	output := view.View()

	assert.Contains(t, output, "Loading")
}

func TestView_View_Error(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewView(s, nil)
	view.width = 80
	view.height = 24
	view.ready = true
	view.err = errors.New("something failed")

	output := view.View()

	assert.Contains(t, output, "Error")
}

func TestView_View_WithMenu(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewView(s, nil)
	view.width = 80
	view.height = 24
	view.ready = true
	view.source = &domain.Source{ID: "src-1", Name: "Test"}
	view.documents = []domain.Document{{ID: "doc-1", Title: "Test"}}
	view.showingMenu = true

	output := view.View()

	assert.Contains(t, output, "Show Content")
	assert.Contains(t, output, "Show Details")
}

func TestView_SetDimensions(t *testing.T) {
	view := NewView(nil, nil)

	view.SetDimensions(100, 50)

	assert.Equal(t, 100, view.width)
	assert.Equal(t, 50, view.height)
}

func TestView_AdjustScroll(t *testing.T) {
	view := NewView(nil, nil)
	view.height = 10
	view.documents = make([]domain.Document, 20)

	// Select item beyond visible area
	view.selected = 15
	view.adjustScroll()

	assert.Greater(t, view.scrollOffset, 0)
}

func TestView_RenderDocument_Truncation(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewView(s, nil)
	view.width = 40
	view.height = 24
	view.ready = true
	view.source = &domain.Source{ID: "src-1", Name: "Test"}

	// Long title and URI that should be truncated
	view.documents = []domain.Document{
		{
			ID:    "doc-1",
			Title: "This is a very long document title that should be truncated",
			URI:   "/very/long/path/to/some/deeply/nested/document.md",
		},
	}

	output := view.View()
	// Should render without panic even with truncation
	assert.NotEmpty(t, output)
}

func TestView_Update_ErrorOccurred(t *testing.T) {
	view := NewView(nil, nil)

	msg := messages.ErrorOccurred{Err: errors.New("test error")}
	view.Update(msg)

	assert.Error(t, view.err)
}

func TestView_LoadDocuments_NoService(t *testing.T) {
	view := NewView(nil, nil)
	view.source = &domain.Source{ID: "src-1"}

	cmd := view.loadDocuments()
	result := cmd()

	loaded, ok := result.(messages.DocumentsLoaded)
	assert.True(t, ok)
	assert.Error(t, loaded.Err)
}

func TestView_LoadDocuments_NoSource(t *testing.T) {
	mock := &MockDocumentService{}
	view := NewView(nil, mock)
	view.source = nil

	cmd := view.loadDocuments()
	result := cmd()

	loaded, ok := result.(messages.DocumentsLoaded)
	assert.True(t, ok)
	assert.Error(t, loaded.Err)
}

func TestView_Documents_Getter(t *testing.T) {
	view := NewView(nil, nil)
	now := time.Now()
	view.documents = []domain.Document{
		{ID: "doc-1", Title: "Test", CreatedAt: now},
	}

	docs := view.Documents()

	assert.Len(t, docs, 1)
	assert.Equal(t, "doc-1", docs[0].ID)
}

func TestView_SelectedIndex_Getter(t *testing.T) {
	view := NewView(nil, nil)
	view.selected = 5

	assert.Equal(t, 5, view.SelectedIndex())
}

func TestView_SelectedDocument_Getter(t *testing.T) {
	view := NewView(nil, nil)
	view.documents = []domain.Document{
		{ID: "doc-1", Title: "First"},
		{ID: "doc-2", Title: "Second"},
	}
	view.selected = 1

	doc := view.SelectedDocument()
	require.NotNil(t, doc)
	assert.Equal(t, "doc-2", doc.ID)
}

func TestView_SelectedDocument_Empty(t *testing.T) {
	view := NewView(nil, nil)
	view.documents = []domain.Document{}

	doc := view.SelectedDocument()
	assert.Nil(t, doc)
}
