package doccontent

import (
	"context"
	"errors"
	"fmt"
	"strings"
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
	GetContentFunc func(ctx context.Context, documentID string) (string, error)
}

func (m *MockDocumentService) ListBySource(ctx context.Context, sourceID string) ([]domain.Document, error) {
	return nil, nil
}

func (m *MockDocumentService) Get(ctx context.Context, documentID string) (*domain.Document, error) {
	return nil, nil
}

func (m *MockDocumentService) GetContent(ctx context.Context, documentID string) (string, error) {
	if m.GetContentFunc != nil {
		return m.GetContentFunc(ctx, documentID)
	}
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

func TestNewView(t *testing.T) {
	s := styles.DefaultStyles()
	mock := &MockDocumentService{}

	view := NewView(s, mock)

	require.NotNil(t, view)
	assert.False(t, view.ready)
	assert.Empty(t, view.content)
}

func TestNewView_NilParams(t *testing.T) {
	view := NewView(nil, nil)

	require.NotNil(t, view)
	assert.Nil(t, view.styles)
	assert.Nil(t, view.documentService)
}

func TestView_SetDocument(t *testing.T) {
	mock := &MockDocumentService{
		GetContentFunc: func(ctx context.Context, documentID string) (string, error) {
			assert.Equal(t, "doc-1", documentID)
			return "Test content", nil
		},
	}
	view := NewView(nil, mock)

	doc := domain.Document{ID: "doc-1", Title: "Test Doc"}
	cmd := view.SetDocument(&doc)

	require.NotNil(t, cmd)
	assert.Equal(t, "doc-1", view.document.ID)
	assert.Equal(t, 0, view.scrollOffset)

	// Execute command
	result := cmd()
	loaded, ok := result.(messages.DocumentContentLoaded)
	require.True(t, ok)
	assert.Equal(t, "doc-1", loaded.DocumentID)
	assert.Equal(t, "Test content", loaded.Content)
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

func TestView_Update_ContentLoaded(t *testing.T) {
	view := NewView(nil, nil)
	view.width = 80
	view.height = 24
	view.document = &domain.Document{ID: "doc-1"}

	msg := messages.DocumentContentLoaded{
		DocumentID: "doc-1",
		Content:    "Line 1\nLine 2\nLine 3",
		Err:        nil,
	}
	updated, cmd := view.Update(msg)

	assert.Equal(t, view, updated)
	assert.Nil(t, cmd)
	assert.Equal(t, "Line 1\nLine 2\nLine 3", view.content)
	assert.False(t, view.loading)
	assert.NoError(t, view.err)
}

func TestView_Update_ContentLoaded_Error(t *testing.T) {
	view := NewView(nil, nil)

	msg := messages.DocumentContentLoaded{Err: errors.New("failed to load")}
	updated, cmd := view.Update(msg)

	assert.Equal(t, view, updated)
	assert.Nil(t, cmd)
	assert.Error(t, view.err)
}

func TestView_Update_KeyMsg_ScrollDown(t *testing.T) {
	view := NewView(nil, nil)
	view.width = 80
	view.height = 10
	view.content = "Line 1\nLine 2\nLine 3\nLine 4\nLine 5\nLine 6\nLine 7\nLine 8\nLine 9\nLine 10\nLine 11\nLine 12"
	view.wrapContent()

	msg := tea.KeyMsg{Type: tea.KeyDown}
	view.Update(msg)
	assert.Equal(t, 1, view.scrollOffset)

	// Test j key
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	view.Update(msg)
	assert.Equal(t, 2, view.scrollOffset)
}

func TestView_Update_KeyMsg_ScrollUp(t *testing.T) {
	view := NewView(nil, nil)
	view.width = 80
	view.height = 10
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

func TestView_Update_KeyMsg_PageDown(t *testing.T) {
	view := NewView(nil, nil)
	view.width = 80
	view.height = 10
	view.content = "Line 1\nLine 2\nLine 3\nLine 4\nLine 5\nLine 6\nLine 7\nLine 8\n" +
		"Line 9\nLine 10\nLine 11\nLine 12\nLine 13\nLine 14\nLine 15\n" +
		"Line 16\nLine 17\nLine 18\nLine 19\nLine 20"
	view.wrapContent()
	view.scrollOffset = 0

	msg := tea.KeyMsg{Type: tea.KeyPgDown}
	view.Update(msg)
	assert.Greater(t, view.scrollOffset, 0)
}

func TestView_Update_KeyMsg_PageUp(t *testing.T) {
	view := NewView(nil, nil)
	view.width = 80
	view.height = 10
	view.content = "Line 1\nLine 2\nLine 3"
	view.wrapContent()
	view.scrollOffset = 5

	msg := tea.KeyMsg{Type: tea.KeyPgUp}
	view.Update(msg)
	assert.Less(t, view.scrollOffset, 5)
}

func TestView_Update_KeyMsg_Back(t *testing.T) {
	view := NewView(nil, nil)

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	_, cmd := view.Update(msg)

	require.NotNil(t, cmd)
	result := cmd()
	changed, ok := result.(messages.ViewChanged)
	assert.True(t, ok)
	assert.Equal(t, messages.ViewDocuments, changed.View)
}

func TestView_Update_ErrorOccurred(t *testing.T) {
	view := NewView(nil, nil)

	msg := messages.ErrorOccurred{Err: errors.New("test error")}
	view.Update(msg)

	assert.Error(t, view.err)
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

func TestView_View_WithContent(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewView(s, nil)
	view.width = 80
	view.height = 24
	view.ready = true
	view.document = &domain.Document{ID: "doc-1", Title: "Test Document"}
	view.content = "# Test Content\n\nThis is some test content."
	view.wrapContent()

	output := view.View()

	assert.Contains(t, output, "Test Content")
}

func TestView_View_Error(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewView(s, nil)
	view.width = 80
	view.height = 24
	view.ready = true
	view.err = errors.New("failed to load content")

	output := view.View()

	assert.Contains(t, output, "Error")
}

func TestView_WrapContent(t *testing.T) {
	view := NewView(nil, nil)
	view.width = 40
	view.content = "Short line\nThis is a much longer line that should be wrapped to fit within the width"
	view.wrapContent()

	// Should have created wrapped lines
	assert.NotEmpty(t, view.lines)
}

func TestView_LoadContent_NoService(t *testing.T) {
	view := NewView(nil, nil)
	view.document = &domain.Document{ID: "doc-1"}

	cmd := view.loadContent()
	result := cmd()

	loaded, ok := result.(messages.DocumentContentLoaded)
	assert.True(t, ok)
	assert.Error(t, loaded.Err)
}

func TestView_LoadContent_NoDocument(t *testing.T) {
	mock := &MockDocumentService{}
	view := NewView(nil, mock)
	view.document = nil

	cmd := view.loadContent()
	result := cmd()

	loaded, ok := result.(messages.DocumentContentLoaded)
	assert.True(t, ok)
	assert.Error(t, loaded.Err)
}

func TestView_SetDimensions(t *testing.T) {
	view := NewView(nil, nil)

	view.SetDimensions(100, 50)

	assert.Equal(t, 100, view.width)
	assert.Equal(t, 50, view.height)
}

// Additional tests for handleKeyMsg to reach 85%+ coverage

func TestView_Update_KeyMsg_CtrlU(t *testing.T) {
	view := NewView(nil, nil)
	view.width = 80
	view.height = 15
	view.content = generateMultilineContent(30)
	view.wrapContent()
	view.scrollOffset = 10

	// Create a key message that will return "ctrl+u" from String()
	msg := tea.KeyMsg{Type: tea.KeyCtrlU}
	view.Update(msg)

	assert.Less(t, view.scrollOffset, 10, "Ctrl+U should scroll up by visible lines")
}

func TestView_Update_KeyMsg_CtrlU_BoundaryZero(t *testing.T) {
	view := NewView(nil, nil)
	view.width = 80
	view.height = 15
	view.content = "Line 1\nLine 2\nLine 3"
	view.wrapContent()
	view.scrollOffset = 2

	// Create a key message that will return "ctrl+u" from String()
	msg := tea.KeyMsg{Type: tea.KeyCtrlU}
	view.Update(msg)

	// Should not go below 0
	assert.GreaterOrEqual(t, view.scrollOffset, 0)
}

func TestView_Update_KeyMsg_CtrlD(t *testing.T) {
	view := NewView(nil, nil)
	view.width = 80
	view.height = 15
	view.content = generateMultilineContent(30)
	view.wrapContent()
	view.scrollOffset = 5

	// Create a key message that will return "ctrl+d" from String()
	msg := tea.KeyMsg{Type: tea.KeyCtrlD}
	view.Update(msg)

	assert.Greater(t, view.scrollOffset, 5, "Ctrl+D should scroll down by visible lines")
}

func TestView_Update_KeyMsg_CtrlD_BoundaryMax(t *testing.T) {
	view := NewView(nil, nil)
	view.width = 80
	view.height = 10
	view.content = "Line 1\nLine 2\nLine 3\nLine 4\nLine 5"
	view.wrapContent()
	maxOffset := view.maxScrollOffset()
	view.scrollOffset = maxOffset - 1

	// Create a key message that will return "ctrl+d" from String()
	msg := tea.KeyMsg{Type: tea.KeyCtrlD}
	view.Update(msg)

	// Should not exceed max offset
	assert.LessOrEqual(t, view.scrollOffset, maxOffset)
	assert.Equal(t, maxOffset, view.scrollOffset)
}

func TestView_Update_KeyMsg_HomeKey(t *testing.T) {
	view := NewView(nil, nil)
	view.width = 80
	view.height = 10
	view.content = generateMultilineContent(20)
	view.wrapContent()
	view.scrollOffset = 10

	msg := tea.KeyMsg{Type: tea.KeyHome}
	view.Update(msg)

	assert.Equal(t, 0, view.scrollOffset, "Home key should scroll to top")
}

func TestView_Update_KeyMsg_GKey(t *testing.T) {
	view := NewView(nil, nil)
	view.width = 80
	view.height = 10
	view.content = generateMultilineContent(20)
	view.wrapContent()
	view.scrollOffset = 10

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}
	view.Update(msg)

	assert.Equal(t, 0, view.scrollOffset, "g key should scroll to top")
}

func TestView_Update_KeyMsg_EndKey(t *testing.T) {
	view := NewView(nil, nil)
	view.width = 80
	view.height = 10
	view.content = generateMultilineContent(20)
	view.wrapContent()
	view.scrollOffset = 0

	msg := tea.KeyMsg{Type: tea.KeyEnd}
	view.Update(msg)

	assert.Equal(t, view.maxScrollOffset(), view.scrollOffset, "End key should scroll to bottom")
}

func TestView_Update_KeyMsg_ShiftG(t *testing.T) {
	view := NewView(nil, nil)
	view.width = 80
	view.height = 10
	view.content = generateMultilineContent(20)
	view.wrapContent()
	view.scrollOffset = 0

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}}
	view.Update(msg)

	assert.Equal(t, view.maxScrollOffset(), view.scrollOffset, "G key should scroll to bottom")
}

func TestView_Update_KeyMsg_CopyKey(t *testing.T) {
	view := NewView(nil, nil)
	view.width = 80
	view.height = 10
	view.content = "Test content"
	view.wrapContent()

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}}
	updated, cmd := view.Update(msg)

	assert.Equal(t, view, updated)
	assert.Nil(t, cmd, "Copy command is a stub and should return nil")
}

func TestView_Update_KeyMsg_UnknownKey(t *testing.T) {
	view := NewView(nil, nil)
	view.width = 80
	view.height = 10
	view.content = "Test content"
	view.wrapContent()
	view.scrollOffset = 0

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}}
	view.Update(msg)

	// Unknown key should not change state
	assert.Equal(t, 0, view.scrollOffset)
}

// Additional tests for visibleLines to improve coverage

func TestView_VisibleLines_VerySmallHeight(t *testing.T) {
	view := NewView(nil, nil)
	view.height = 3 // Less than reserved lines

	lines := view.visibleLines()

	assert.Equal(t, 1, lines, "Should return at least 1 visible line")
}

func TestView_VisibleLines_ZeroHeight(t *testing.T) {
	view := NewView(nil, nil)
	view.height = 0

	lines := view.visibleLines()

	assert.Equal(t, 1, lines, "Should return at least 1 visible line even with zero height")
}

func TestView_VisibleLines_NormalHeight(t *testing.T) {
	view := NewView(nil, nil)
	view.height = 24

	lines := view.visibleLines()

	assert.Equal(t, 18, lines, "Should calculate correct visible lines (24 - 6 reserved)")
}

func TestView_VisibleLines_LargeHeight(t *testing.T) {
	view := NewView(nil, nil)
	view.height = 100

	lines := view.visibleLines()

	assert.Equal(t, 94, lines, "Should calculate correct visible lines for large height")
}

// Additional tests for maxScrollOffset to improve coverage

func TestView_MaxScrollOffset_EmptyLines(t *testing.T) {
	view := NewView(nil, nil)
	view.height = 24
	view.lines = []string{}

	maxOffset := view.maxScrollOffset()

	assert.Equal(t, 0, maxOffset, "Empty content should have 0 max offset")
}

func TestView_MaxScrollOffset_ContentFitsScreen(t *testing.T) {
	view := NewView(nil, nil)
	view.height = 24
	view.lines = []string{"Line 1", "Line 2", "Line 3"}

	maxOffset := view.maxScrollOffset()

	assert.Equal(t, 0, maxOffset, "Content that fits on screen should have 0 max offset")
}

func TestView_MaxScrollOffset_ContentExceedsScreen(t *testing.T) {
	view := NewView(nil, nil)
	view.height = 10
	view.lines = make([]string, 30)

	maxOffset := view.maxScrollOffset()

	visible := view.visibleLines()
	expected := 30 - visible
	assert.Equal(t, expected, maxOffset, "Should calculate correct max offset for long content")
	assert.Greater(t, maxOffset, 0)
}

func TestView_MaxScrollOffset_ExactlyFits(t *testing.T) {
	view := NewView(nil, nil)
	view.height = 10
	visible := view.visibleLines()
	view.lines = make([]string, visible)

	maxOffset := view.maxScrollOffset()

	assert.Equal(t, 0, maxOffset, "Content that exactly fits should have 0 max offset")
}

// Additional tests for View() to improve coverage

func TestView_View_NoDocument(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewView(s, nil)
	view.width = 80
	view.height = 24
	view.ready = true
	view.document = nil

	output := view.View()

	assert.Contains(t, output, "Document Content", "Should show default title when no document")
}

func TestView_View_DocumentWithEmptyTitle(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewView(s, nil)
	view.width = 80
	view.height = 24
	view.ready = true
	view.document = &domain.Document{ID: "doc-123", Title: ""}

	output := view.View()

	assert.Contains(t, output, "doc-123", "Should show document ID when title is empty")
}

func TestView_View_WithScrollIndicator(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewView(s, nil)
	view.width = 80
	view.height = 10
	view.ready = true
	view.document = &domain.Document{ID: "doc-1", Title: "Test"}
	view.content = generateMultilineContent(30)
	view.wrapContent()
	view.scrollOffset = 5

	output := view.View()

	assert.Contains(t, output, "Line", "Should show scroll indicator")
	assert.Contains(t, output, "%", "Should show percentage")
}

func TestView_View_ScrollIndicator_ZeroMaxOffset(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewView(s, nil)
	view.width = 80
	view.height = 30
	view.ready = true
	view.document = &domain.Document{ID: "doc-1", Title: "Test"}
	view.content = "Line 1\nLine 2\nLine 3"
	view.wrapContent()
	view.scrollOffset = 0

	output := view.View()

	// With small content that fits on screen, scroll indicator should not be shown
	// since len(v.lines) <= visibleLines
	assert.NotContains(t, output, "[0%]", "Should not show scroll indicator when content fits")
}

func TestView_View_ScrollIndicator_AtMaxOffset(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewView(s, nil)
	view.width = 80
	view.height = 10
	view.ready = true
	view.document = &domain.Document{ID: "doc-1", Title: "Test"}
	view.content = generateMultilineContent(30)
	view.wrapContent()
	view.scrollOffset = view.maxScrollOffset()

	output := view.View()

	assert.Contains(t, output, "100%", "Should show 100% at bottom")
}

func TestView_View_ScrollIndicator_MidScroll(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewView(s, nil)
	view.width = 80
	view.height = 10
	view.ready = true
	view.document = &domain.Document{ID: "doc-1", Title: "Test"}
	view.content = generateMultilineContent(30)
	view.wrapContent()
	maxOffset := view.maxScrollOffset()
	view.scrollOffset = maxOffset / 2

	output := view.View()

	// Should show a percentage between 0 and 100
	assert.Contains(t, output, "%")
}

func TestView_View_EmptyContentRendering(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewView(s, nil)
	view.width = 80
	view.height = 24
	view.ready = true
	view.document = &domain.Document{ID: "doc-1", Title: "Test"}
	view.content = ""
	view.wrapContent()

	output := view.View()

	assert.Contains(t, output, "(No content)", "Should show no content message")
}

func TestView_View_NarrowWidth(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewView(s, nil)
	view.width = 20
	view.height = 24
	view.ready = true
	view.document = &domain.Document{ID: "doc-1", Title: "Test"}
	view.content = "Test content"
	view.wrapContent()

	output := view.View()

	assert.NotEmpty(t, output, "Should render even with narrow width")
}

func TestView_View_WideWidth(t *testing.T) {
	s := styles.DefaultStyles()
	view := NewView(s, nil)
	view.width = 200
	view.height = 24
	view.ready = true
	view.document = &domain.Document{ID: "doc-1", Title: "Test"}
	view.content = "Test content"
	view.wrapContent()

	output := view.View()

	assert.NotEmpty(t, output, "Should render with wide width")
}

// Tests for getter methods

func TestView_Document_Getter(t *testing.T) {
	view := NewView(nil, nil)
	doc := &domain.Document{ID: "doc-1", Title: "Test Document"}
	view.document = doc

	result := view.Document()

	assert.Equal(t, doc, result)
	assert.Equal(t, "doc-1", result.ID)
}

func TestView_Document_Getter_Nil(t *testing.T) {
	view := NewView(nil, nil)
	view.document = nil

	result := view.Document()

	assert.Nil(t, result)
}

func TestView_Content_Getter(t *testing.T) {
	view := NewView(nil, nil)
	view.content = "Test content here"

	result := view.Content()

	assert.Equal(t, "Test content here", result)
}

func TestView_Content_Getter_Empty(t *testing.T) {
	view := NewView(nil, nil)
	view.content = ""

	result := view.Content()

	assert.Equal(t, "", result)
}

func TestView_Err_Getter(t *testing.T) {
	view := NewView(nil, nil)
	testErr := errors.New("test error")
	view.err = testErr

	result := view.Err()

	assert.Equal(t, testErr, result)
	assert.Error(t, result)
}

func TestView_Err_Getter_Nil(t *testing.T) {
	view := NewView(nil, nil)
	view.err = nil

	result := view.Err()

	assert.Nil(t, result)
}

// Tests for minInt helper

func TestMinInt_FirstSmaller(t *testing.T) {
	result := minInt(5, 10)
	assert.Equal(t, 5, result)
}

func TestMinInt_SecondSmaller(t *testing.T) {
	result := minInt(20, 15)
	assert.Equal(t, 15, result)
}

func TestMinInt_Equal(t *testing.T) {
	result := minInt(10, 10)
	assert.Equal(t, 10, result)
}

func TestMinInt_NegativeNumbers(t *testing.T) {
	result := minInt(-5, -10)
	assert.Equal(t, -10, result)
}

func TestMinInt_ZeroAndPositive(t *testing.T) {
	result := minInt(0, 5)
	assert.Equal(t, 0, result)
}

func TestMinInt_NegativeAndPositive(t *testing.T) {
	result := minInt(-5, 5)
	assert.Equal(t, -5, result)
}

// Additional tests for wrapContent edge cases

func TestView_WrapContent_EmptyContent(t *testing.T) {
	view := NewView(nil, nil)
	view.width = 80
	view.content = ""

	view.wrapContent()

	assert.Nil(t, view.lines)
}

func TestView_WrapContent_VeryNarrowWidth(t *testing.T) {
	view := NewView(nil, nil)
	view.width = 10 // Less than minimum + padding
	view.content = "This is a test line that will need wrapping"

	view.wrapContent()

	// Should use minimum width of 20
	assert.NotEmpty(t, view.lines)
	assert.Greater(t, len(view.lines), 1, "Long line should be wrapped")
}

func TestView_WrapContent_ExactWidthLine(t *testing.T) {
	view := NewView(nil, nil)
	view.width = 30
	contentWidth := 26 // width - 4
	view.content = strings.Repeat("x", contentWidth)

	view.wrapContent()

	assert.Len(t, view.lines, 1, "Line that exactly fits should not be wrapped")
}

func TestView_WrapContent_OneCharOverWidth(t *testing.T) {
	view := NewView(nil, nil)
	view.width = 30
	contentWidth := 26 // width - 4
	view.content = strings.Repeat("x", contentWidth+1)

	view.wrapContent()

	assert.Greater(t, len(view.lines), 1, "Line one char over should be wrapped")
}

func TestView_WrapContent_MultipleNewlines(t *testing.T) {
	view := NewView(nil, nil)
	view.width = 80
	view.content = "Line 1\n\n\nLine 2"

	view.wrapContent()

	assert.Len(t, view.lines, 4, "Should preserve empty lines")
}

func TestView_WrapContent_TrailingNewline(t *testing.T) {
	view := NewView(nil, nil)
	view.width = 80
	view.content = "Line 1\nLine 2\n"

	view.wrapContent()

	// Trailing newline creates an empty line
	assert.Len(t, view.lines, 3)
}

// Test for scroll boundary with page up at offset 0
func TestView_Update_KeyMsg_PageUp_AtZero(t *testing.T) {
	view := NewView(nil, nil)
	view.width = 80
	view.height = 10
	view.content = generateMultilineContent(20)
	view.wrapContent()
	view.scrollOffset = 0

	msg := tea.KeyMsg{Type: tea.KeyPgUp}
	view.Update(msg)

	assert.Equal(t, 0, view.scrollOffset, "Should stay at 0 when already at top")
}

// Test for scroll boundary with page down at max
func TestView_Update_KeyMsg_PageDown_AtMax(t *testing.T) {
	view := NewView(nil, nil)
	view.width = 80
	view.height = 10
	view.content = generateMultilineContent(20)
	view.wrapContent()
	maxOffset := view.maxScrollOffset()
	view.scrollOffset = maxOffset

	msg := tea.KeyMsg{Type: tea.KeyPgDown}
	view.Update(msg)

	assert.Equal(t, maxOffset, view.scrollOffset, "Should stay at max when already at bottom")
}

// Test for scroll down when at max already
func TestView_Update_KeyMsg_ScrollDown_AtMax(t *testing.T) {
	view := NewView(nil, nil)
	view.width = 80
	view.height = 10
	view.content = generateMultilineContent(20)
	view.wrapContent()
	maxOffset := view.maxScrollOffset()
	view.scrollOffset = maxOffset

	msg := tea.KeyMsg{Type: tea.KeyDown}
	view.Update(msg)

	assert.Equal(t, maxOffset, view.scrollOffset, "Should not exceed max offset")
}

// Helper function to generate multiline content for testing
func generateMultilineContent(lines int) string {
	var content strings.Builder
	for i := 1; i <= lines; i++ {
		if i > 1 {
			content.WriteString("\n")
		}
		content.WriteString(fmt.Sprintf("This is line number %d with some content", i))
	}
	return content.String()
}
