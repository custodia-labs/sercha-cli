package sourcedetail

import (
	"context"
	"errors"
	"testing"

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
}

func (m *MockDocumentService) ListBySource(ctx context.Context, sourceID string) ([]domain.Document, error) {
	if m.ListBySourceFunc != nil {
		return m.ListBySourceFunc(ctx, sourceID)
	}
	return []domain.Document{}, nil
}

func (m *MockDocumentService) Get(ctx context.Context, documentID string) (*domain.Document, error) {
	return nil, nil
}

func (m *MockDocumentService) GetContent(ctx context.Context, documentID string) (string, error) {
	return "", nil
}

func (m *MockDocumentService) GetDetails(ctx context.Context, documentID string) (*driving.DocumentDetails, error) {
	return nil, nil
}

func (m *MockDocumentService) Exclude(ctx context.Context, documentID, reason string) error {
	return nil
}

func (m *MockDocumentService) Refresh(ctx context.Context, documentID string) error {
	return nil
}

func (m *MockDocumentService) Open(ctx context.Context, documentID string) error {
	return nil
}

// MockSourceService implements driving.SourceService for testing.
type MockSourceService struct {
	RemoveFunc func(ctx context.Context, id string) error
}

func (m *MockSourceService) Add(ctx context.Context, source domain.Source) error {
	return nil
}

func (m *MockSourceService) Get(ctx context.Context, id string) (*domain.Source, error) {
	return nil, nil
}

func (m *MockSourceService) List(ctx context.Context) ([]domain.Source, error) {
	return nil, nil
}

func (m *MockSourceService) Remove(ctx context.Context, id string) error {
	if m.RemoveFunc != nil {
		return m.RemoveFunc(ctx, id)
	}
	return nil
}

func (m *MockSourceService) Update(ctx context.Context, source domain.Source) error {
	return nil
}

func (m *MockSourceService) ValidateConfig(ctx context.Context, connectorType string, config map[string]string) error {
	return nil
}

// MockSyncOrchestrator implements driving.SyncOrchestrator for testing.
type MockSyncOrchestrator struct {
	SyncFunc func(ctx context.Context, sourceID string) error
}

func (m *MockSyncOrchestrator) Sync(ctx context.Context, sourceID string) error {
	if m.SyncFunc != nil {
		return m.SyncFunc(ctx, sourceID)
	}
	return nil
}

func (m *MockSyncOrchestrator) SyncAll(ctx context.Context) error {
	return nil
}

func (m *MockSyncOrchestrator) Status(ctx context.Context, sourceID string) (*driving.SyncStatus, error) {
	return nil, nil
}

func TestNewView(t *testing.T) {
	s := styles.DefaultStyles()

	view := NewView(s, nil, nil, nil)

	require.NotNil(t, view)
	assert.False(t, view.ready)
	assert.Equal(t, OptionViewDocuments, view.selected)
}

func TestNewView_NilParams(t *testing.T) {
	view := NewView(nil, nil, nil, nil)

	require.NotNil(t, view)
	assert.Nil(t, view.styles)
}

func TestView_SetSource(t *testing.T) {
	view := NewView(nil, nil, nil, nil)

	source := domain.Source{ID: "src-1", Name: "Test Source", Type: "filesystem"}
	view.SetSource(source)

	require.NotNil(t, view.source)
	assert.Equal(t, "src-1", view.source.ID)
	assert.Equal(t, OptionViewDocuments, view.selected)
	assert.False(t, view.syncing)
	assert.False(t, view.deleting)
}

func TestView_Init(t *testing.T) {
	mock := &MockDocumentService{
		ListBySourceFunc: func(ctx context.Context, sourceID string) ([]domain.Document, error) {
			return []domain.Document{{ID: "doc-1"}, {ID: "doc-2"}}, nil
		},
	}
	view := NewView(nil, nil, nil, mock)
	view.source = &domain.Source{ID: "src-1"}

	cmd := view.Init()

	require.NotNil(t, cmd)
	// The loadDocCount sets docCount directly (returns nil msg)
	cmd()
	assert.Equal(t, 2, view.docCount)
}

func TestView_Update_WindowSize(t *testing.T) {
	view := NewView(nil, nil, nil, nil)

	msg := tea.WindowSizeMsg{Width: 80, Height: 24}
	updated, cmd := view.Update(msg)

	assert.Equal(t, view, updated)
	assert.Nil(t, cmd)
	assert.True(t, view.ready)
	assert.Equal(t, 80, view.width)
	assert.Equal(t, 24, view.height)
}

func TestView_Update_KeyMsg_Navigation(t *testing.T) {
	view := NewView(nil, nil, nil, nil)
	view.source = &domain.Source{ID: "src-1"}
	view.selected = OptionViewDocuments

	// Navigate down
	msg := tea.KeyMsg{Type: tea.KeyDown}
	view.Update(msg)
	assert.Equal(t, OptionSyncNow, view.selected)

	// Navigate with j
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	view.Update(msg)
	assert.Equal(t, OptionDeleteSource, view.selected)

	// Navigate up
	msg = tea.KeyMsg{Type: tea.KeyUp}
	view.Update(msg)
	assert.Equal(t, OptionSyncNow, view.selected)

	// Navigate with k
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	view.Update(msg)
	assert.Equal(t, OptionViewDocuments, view.selected)
}

func TestView_Update_KeyMsg_SelectViewDocuments(t *testing.T) {
	view := NewView(nil, nil, nil, nil)
	view.source = &domain.Source{ID: "src-1", Name: "Test"}
	view.selected = OptionViewDocuments

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := view.Update(msg)

	require.NotNil(t, cmd)
	result := cmd()
	selected, ok := result.(messages.SourceSelected)
	assert.True(t, ok)
	assert.Equal(t, "src-1", selected.Source.ID)
}

func TestView_Update_KeyMsg_SelectSyncNow(t *testing.T) {
	syncCalled := false
	syncMock := &MockSyncOrchestrator{
		SyncFunc: func(ctx context.Context, sourceID string) error {
			syncCalled = true
			return nil
		},
	}
	view := NewView(nil, nil, syncMock, nil)
	view.source = &domain.Source{ID: "src-1"}
	view.selected = OptionSyncNow

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := view.Update(msg)

	require.NotNil(t, cmd)
	// Execute command - syncing is set and then cleared within the command
	cmd()
	assert.True(t, syncCalled)
	// syncing is false after cmd completes (set to false in syncSource())
	assert.False(t, view.syncing)
}

func TestView_Update_KeyMsg_SelectDeleteSource(t *testing.T) {
	deleteCalled := false
	sourceMock := &MockSourceService{
		RemoveFunc: func(ctx context.Context, id string) error {
			deleteCalled = true
			assert.Equal(t, "src-1", id)
			return nil
		},
	}
	view := NewView(nil, sourceMock, nil, nil)
	view.source = &domain.Source{ID: "src-1"}
	view.selected = OptionDeleteSource

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := view.Update(msg)

	require.NotNil(t, cmd)
	// Execute command - deleting is set inside the cmd function
	cmd()
	assert.True(t, view.deleting)
	assert.True(t, deleteCalled)
}

func TestView_Update_KeyMsg_SelectBack(t *testing.T) {
	view := NewView(nil, nil, nil, nil)
	view.source = &domain.Source{ID: "src-1"}
	view.selected = OptionBack

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := view.Update(msg)

	require.NotNil(t, cmd)
	result := cmd()
	changed, ok := result.(messages.ViewChanged)
	assert.True(t, ok)
	assert.Equal(t, messages.ViewSources, changed.View)
}

func TestView_Update_KeyMsg_Escape(t *testing.T) {
	view := NewView(nil, nil, nil, nil)
	view.source = &domain.Source{ID: "src-1"}

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	_, cmd := view.Update(msg)

	require.NotNil(t, cmd)
	result := cmd()
	changed, ok := result.(messages.ViewChanged)
	assert.True(t, ok)
	assert.Equal(t, messages.ViewSources, changed.View)
}

func TestView_Update_SourceRemoved(t *testing.T) {
	view := NewView(nil, nil, nil, nil)
	view.source = &domain.Source{ID: "src-1"}
	view.deleting = true

	msg := messages.SourceRemoved{ID: "src-1", Err: nil}
	_, cmd := view.Update(msg)

	assert.False(t, view.deleting)
	require.NotNil(t, cmd)
	result := cmd()
	changed, ok := result.(messages.ViewChanged)
	assert.True(t, ok)
	assert.Equal(t, messages.ViewSources, changed.View)
}

func TestView_Update_SourceRemoved_Error(t *testing.T) {
	view := NewView(nil, nil, nil, nil)
	view.source = &domain.Source{ID: "src-1"}
	view.deleting = true

	msg := messages.SourceRemoved{ID: "src-1", Err: errors.New("failed")}
	view.Update(msg)

	assert.False(t, view.deleting)
	assert.Error(t, view.err)
}

func TestView_Update_ErrorOccurred(t *testing.T) {
	view := NewView(nil, nil, nil, nil)
	view.syncing = true

	msg := messages.ErrorOccurred{Err: errors.New("test error")}
	view.Update(msg)

	assert.Error(t, view.err)
	assert.False(t, view.syncing)
}

func TestView_View(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewView(s, nil, nil, nil)
	view.width = 80
	view.height = 24
	view.ready = true
	view.source = &domain.Source{
		ID:   "src-1",
		Name: "Test Source",
		Type: "filesystem",
	}
	view.docCount = 10

	output := view.View()

	assert.Contains(t, output, "Test Source")
	assert.Contains(t, output, "filesystem")
	assert.Contains(t, output, "View Documents")
}

func TestView_View_Error(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewView(s, nil, nil, nil)
	view.width = 80
	view.height = 24
	view.ready = true
	view.source = &domain.Source{ID: "src-1", Name: "Test"}
	view.err = errors.New("something failed")

	output := view.View()

	assert.Contains(t, output, "Error")
}

func TestView_SetDimensions(t *testing.T) {
	view := NewView(nil, nil, nil, nil)

	view.SetDimensions(100, 50)

	assert.Equal(t, 100, view.width)
	assert.Equal(t, 50, view.height)
}
