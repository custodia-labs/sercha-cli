package sources

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
)

// MockSourceService implements driving.SourceService for testing.
type MockSourceService struct {
	ListFunc   func(ctx context.Context) ([]domain.Source, error)
	RemoveFunc func(ctx context.Context, id string) error
}

func (m *MockSourceService) Add(ctx context.Context, source domain.Source) error {
	return nil
}

func (m *MockSourceService) Get(ctx context.Context, id string) (*domain.Source, error) {
	return nil, nil
}

func (m *MockSourceService) List(ctx context.Context) ([]domain.Source, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx)
	}
	return []domain.Source{}, nil
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

func TestNewView(t *testing.T) {
	s := styles.DefaultStyles()
	mock := &MockSourceService{}

	view := NewView(s, mock, nil)

	require.NotNil(t, view)
	assert.False(t, view.ready)
	assert.Empty(t, view.sources)
	assert.Equal(t, 0, view.selected)
}

func TestNewView_NilParams(t *testing.T) {
	view := NewView(nil, nil, nil)

	require.NotNil(t, view)
	assert.Nil(t, view.styles)
	assert.Nil(t, view.sourceService)
}

func TestView_Init(t *testing.T) {
	sources := []domain.Source{
		{ID: "src-1", Name: "Source 1", Type: "filesystem"},
		{ID: "src-2", Name: "Source 2", Type: "notion"},
	}
	mock := &MockSourceService{
		ListFunc: func(ctx context.Context) ([]domain.Source, error) {
			return sources, nil
		},
	}
	view := NewView(nil, mock, nil)

	cmd := view.Init()

	require.NotNil(t, cmd)
	result := cmd()
	loaded, ok := result.(sourcesLoadedMsg)
	require.True(t, ok)
	assert.Len(t, loaded.Sources, 2)
	assert.NoError(t, loaded.Err)
}

func TestView_Init_NilService(t *testing.T) {
	view := NewView(nil, nil, nil)

	cmd := view.Init()

	require.NotNil(t, cmd)
	result := cmd()
	loaded, ok := result.(sourcesLoadedMsg)
	require.True(t, ok)
	assert.Error(t, loaded.Err)
}

func TestView_Update_WindowSize(t *testing.T) {
	view := NewView(nil, nil, nil)

	msg := tea.WindowSizeMsg{Width: 80, Height: 24}
	updated, cmd := view.Update(msg)

	assert.Equal(t, view, updated)
	assert.Nil(t, cmd)
	assert.True(t, view.ready)
	assert.Equal(t, 80, view.width)
	assert.Equal(t, 24, view.height)
}

func TestView_Update_SourcesLoaded(t *testing.T) {
	view := NewView(nil, nil, nil)
	view.loading = true

	sources := []domain.Source{
		{ID: "src-1", Name: "Source 1"},
	}
	msg := messages.SourcesLoaded{Sources: sources, Err: nil}
	updated, cmd := view.Update(msg)

	assert.Equal(t, view, updated)
	assert.Nil(t, cmd)
	assert.False(t, view.loading)
	assert.Len(t, view.sources, 1)
	assert.NoError(t, view.err)
}

func TestView_Update_SourcesLoaded_Error(t *testing.T) {
	view := NewView(nil, nil, nil)
	view.loading = true

	msg := messages.SourcesLoaded{Err: errors.New("failed to load")}
	view.Update(msg)

	assert.False(t, view.loading)
	assert.Error(t, view.err)
}

func TestView_Update_KeyMsg_NavigateDown(t *testing.T) {
	view := NewView(nil, nil, nil)
	view.sources = []domain.Source{
		{ID: "src-1"}, {ID: "src-2"}, {ID: "src-3"},
	}
	view.selected = 0

	// Test down key
	msg := tea.KeyMsg{Type: tea.KeyDown}
	view.Update(msg)
	assert.Equal(t, 1, view.selected)

	// Test j key
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	view.Update(msg)
	assert.Equal(t, 2, view.selected)

	// Test boundary - can't go past last item
	msg = tea.KeyMsg{Type: tea.KeyDown}
	view.Update(msg)
	assert.Equal(t, 2, view.selected)
}

func TestView_Update_KeyMsg_NavigateUp(t *testing.T) {
	view := NewView(nil, nil, nil)
	view.sources = []domain.Source{
		{ID: "src-1"}, {ID: "src-2"}, {ID: "src-3"},
	}
	view.selected = 2

	// Test up key
	msg := tea.KeyMsg{Type: tea.KeyUp}
	view.Update(msg)
	assert.Equal(t, 1, view.selected)

	// Test k key
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	view.Update(msg)
	assert.Equal(t, 0, view.selected)

	// Test boundary - can't go before first item
	msg = tea.KeyMsg{Type: tea.KeyUp}
	view.Update(msg)
	assert.Equal(t, 0, view.selected)
}

func TestView_Update_KeyMsg_Enter(t *testing.T) {
	view := NewView(nil, nil, nil)
	view.sources = []domain.Source{
		{ID: "src-1", Name: "Source 1", Type: "filesystem"},
		{ID: "src-2", Name: "Source 2", Type: "notion"},
	}
	view.selected = 1

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := view.Update(msg)

	require.NotNil(t, cmd)
	result := cmd()
	selected, ok := result.(messages.SourceSelected)
	require.True(t, ok)
	assert.Equal(t, "src-2", selected.Source.ID)
	assert.Equal(t, "Source 2", selected.Source.Name)
}

func TestView_Update_KeyMsg_Enter_EmptyList(t *testing.T) {
	view := NewView(nil, nil, nil)
	view.sources = []domain.Source{}

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := view.Update(msg)

	assert.Nil(t, cmd)
}

func TestView_Update_KeyMsg_Delete(t *testing.T) {
	deletedID := ""
	mock := &MockSourceService{
		RemoveFunc: func(ctx context.Context, id string) error {
			deletedID = id
			return nil
		},
	}
	view := NewView(nil, mock, nil)
	view.sources = []domain.Source{
		{ID: "src-1"}, {ID: "src-2"},
	}
	view.selected = 0

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}
	_, cmd := view.Update(msg)

	require.NotNil(t, cmd)
	cmd()
	assert.Equal(t, "src-1", deletedID)
}

func TestView_Update_KeyMsg_Reload(t *testing.T) {
	mock := &MockSourceService{
		ListFunc: func(ctx context.Context) ([]domain.Source, error) {
			return []domain.Source{{ID: "reloaded"}}, nil
		},
	}
	view := NewView(nil, mock, nil)

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}
	_, cmd := view.Update(msg)

	assert.True(t, view.loading)
	require.NotNil(t, cmd)
}

func TestView_Update_SourceRemoved(t *testing.T) {
	mock := &MockSourceService{
		ListFunc: func(ctx context.Context) ([]domain.Source, error) {
			return []domain.Source{{ID: "remaining"}}, nil
		},
	}
	view := NewView(nil, mock, nil)

	msg := messages.SourceRemoved{ID: "src-1", Err: nil}
	_, cmd := view.Update(msg)

	require.NotNil(t, cmd) // Should trigger reload
}

func TestView_Update_SourceRemoved_Error(t *testing.T) {
	view := NewView(nil, nil, nil)

	msg := messages.SourceRemoved{ID: "src-1", Err: errors.New("delete failed")}
	_, cmd := view.Update(msg)

	assert.Nil(t, cmd)
	assert.Error(t, view.err)
}

func TestView_View_Loading(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewView(s, nil, nil)
	view.width = 80
	view.height = 24
	view.ready = true
	view.loading = true

	output := view.View()

	assert.Contains(t, output, "Loading")
}

func TestView_View_Error(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewView(s, nil, nil)
	view.width = 80
	view.height = 24
	view.ready = true
	view.err = errors.New("something went wrong")

	output := view.View()

	assert.Contains(t, output, "Error")
	assert.Contains(t, output, "something went wrong")
}

func TestView_View_Empty(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewView(s, nil, nil)
	view.width = 80
	view.height = 24
	view.ready = true
	view.sources = []domain.Source{}

	output := view.View()

	assert.Contains(t, output, "No sources configured")
}

func TestView_View_WithSources(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewView(s, nil, nil)
	view.width = 80
	view.height = 24
	view.ready = true
	view.sources = []domain.Source{
		{ID: "src-1", Name: "My Documents", Type: "filesystem"},
		{ID: "src-2", Name: "Work Notes", Type: "notion"},
	}

	output := view.View()

	assert.Contains(t, output, "Sources")
	assert.Contains(t, output, "filesystem")
	assert.Contains(t, output, "notion")
	assert.Contains(t, output, "My Documents")
	assert.Contains(t, output, "Work Notes")
}

func TestView_SetDimensions(t *testing.T) {
	view := NewView(nil, nil, nil)

	view.SetDimensions(100, 50)

	assert.Equal(t, 100, view.width)
	assert.Equal(t, 50, view.height)
	assert.True(t, view.ready)
}

func TestView_Sources(t *testing.T) {
	view := NewView(nil, nil, nil)
	view.sources = []domain.Source{{ID: "src-1"}, {ID: "src-2"}}

	sources := view.Sources()

	assert.Len(t, sources, 2)
	assert.Equal(t, "src-1", sources[0].ID)
}

func TestView_SelectedIndex(t *testing.T) {
	view := NewView(nil, nil, nil)
	view.selected = 3

	assert.Equal(t, 3, view.SelectedIndex())
}

func TestView_Err(t *testing.T) {
	view := NewView(nil, nil, nil)
	view.err = errors.New("test error")

	assert.Error(t, view.Err())
}

func TestView_RenderSource_Selected(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewView(s, nil, nil)
	view.width = 80
	view.selected = 0

	source := domain.Source{ID: "src-1", Name: "Test Source", Type: "filesystem"}
	output := view.renderSource(0, &source)

	assert.Contains(t, output, "Test Source")
	assert.Contains(t, output, ">")
}

func TestView_RenderSource_NotSelected(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewView(s, nil, nil)
	view.width = 80
	view.selected = 1

	source := domain.Source{ID: "src-1", Name: "Test Source", Type: "filesystem"}
	output := view.renderSource(0, &source)

	assert.Contains(t, output, "Test Source")
}

func TestView_RenderSource_LongName(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewView(s, nil, nil)
	view.width = 40

	source := domain.Source{
		ID:   "src-1",
		Name: "This is a very long source name that should be truncated",
		Type: "filesystem",
	}
	output := view.renderSource(0, &source)

	// Name should be truncated
	assert.Contains(t, output, "...")
}

func TestView_RenderSource_EmptyName(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewView(s, nil, nil)
	view.width = 80

	source := domain.Source{ID: "src-1", Name: "", Type: "filesystem"}
	output := view.renderSource(0, &source)

	// Should fall back to ID
	assert.Contains(t, output, "src-1")
}

func TestView_DeleteSource_NilService(t *testing.T) {
	view := NewView(nil, nil, nil)
	view.sources = []domain.Source{{ID: "src-1"}}

	cmd := view.deleteSource("src-1")
	result := cmd()

	removed, ok := result.(messages.SourceRemoved)
	require.True(t, ok)
	assert.Error(t, removed.Err)
}
