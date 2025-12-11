// Package list provides list display components for the TUI.
package list

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/custodia-labs/sercha-cli/internal/adapters/driving/tui/styles"
	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

// ResultList displays search results in a navigable list.
type ResultList struct {
	results  []domain.SearchResult
	selected int
	styles   *styles.Styles
	width    int
	height   int
}

// NewResultList creates a new result list component.
func NewResultList(s *styles.Styles) *ResultList {
	if s == nil {
		s = styles.DefaultStyles()
	}

	return &ResultList{
		results:  nil,
		selected: 0,
		styles:   s,
		width:    80,
		height:   10,
	}
}

// Init initialises the result list.
func (r *ResultList) Init() tea.Cmd {
	return nil
}

// Update handles list navigation messages.
func (r *ResultList) Update(msg tea.Msg) (*ResultList, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		//nolint:exhaustive // handling only relevant key types
		switch msg.Type {
		case tea.KeyUp:
			r.MoveUp()
		case tea.KeyDown:
			r.MoveDown()
		default:
			// Handle other keys
		}
		switch msg.String() {
		case "k":
			r.MoveUp()
		case "j":
			r.MoveDown()
		}
	}
	return r, nil
}

// View renders the result list.
func (r *ResultList) View() string {
	if len(r.results) == 0 {
		return r.styles.Muted.Render("No results")
	}

	lines := make([]string, 0, len(r.results)*2+2)

	// Header
	header := r.styles.Subtitle.Render(fmt.Sprintf("Results (%d)", len(r.results)))
	lines = append(lines, header, "")

	// Calculate visible range based on height
	// Each result takes 2-3 lines (title + optional source + preview), so divide by 3 for safety
	visibleCount := (r.height - 4) / 3
	if visibleCount < 1 {
		visibleCount = 1
	}

	start := 0
	if r.selected >= visibleCount {
		start = r.selected - visibleCount + 1
	}
	end := start + visibleCount
	if end > len(r.results) {
		end = len(r.results)
	}

	for i := start; i < end; i++ {
		line := r.renderResult(i, &r.results[i])
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

// renderResult formats a single search result with preview text.
func (r *ResultList) renderResult(index int, result *domain.SearchResult) string {
	// Indicator for selected item
	indicator := "  "
	if index == r.selected {
		indicator = "> "
	}

	// Title with score
	title := result.Document.Title
	if title == "" {
		title = "(Untitled)"
	}

	// Truncate title if too long
	maxTitleLen := r.width - 20
	if maxTitleLen < 10 {
		maxTitleLen = 10
	}
	if len(title) > maxTitleLen {
		title = title[:maxTitleLen-3] + "..."
	}

	score := fmt.Sprintf("%.2f", result.Score)

	var titleLine string
	if index == r.selected {
		titleLine = r.styles.Selected.Render(fmt.Sprintf("%s%-*s  %s", indicator, maxTitleLen, title, score))
	} else {
		titleLine = r.styles.Normal.Render(fmt.Sprintf("%s%-*s  ", indicator, maxTitleLen, title)) +
			r.styles.Muted.Render(score)
	}

	// Preview text (first highlight or chunk content)
	preview := ""
	if len(result.Highlights) > 0 {
		preview = result.Highlights[0]
	} else if result.Chunk.Content != "" {
		preview = result.Chunk.Content
	}

	// Truncate preview to fit width
	maxPreviewLen := r.width - 6
	if maxPreviewLen < 20 {
		maxPreviewLen = 20
	}
	if len(preview) > maxPreviewLen {
		preview = preview[:maxPreviewLen-3] + "..."
	}

	previewLine := r.styles.Muted.Render("    " + preview)

	// Source name line (if available)
	var sourceLine string
	if result.SourceName != "" {
		sourceLine = "\n" + r.styles.Subtitle.Render("    "+result.SourceName)
	}

	return titleLine + sourceLine + "\n" + previewLine
}

// SetResults updates the result list.
func (r *ResultList) SetResults(results []domain.SearchResult) {
	r.results = results
	r.selected = 0
}

// Results returns the current results.
func (r *ResultList) Results() []domain.SearchResult {
	return r.results
}

// Selected returns the index of the selected result.
func (r *ResultList) Selected() int {
	return r.selected
}

// SetSelected sets the selected index.
func (r *ResultList) SetSelected(index int) {
	if index >= 0 && index < len(r.results) {
		r.selected = index
	}
}

// SelectedResult returns the currently selected result, or nil if none.
func (r *ResultList) SelectedResult() *domain.SearchResult {
	if len(r.results) == 0 || r.selected < 0 || r.selected >= len(r.results) {
		return nil
	}
	return &r.results[r.selected]
}

// MoveUp moves selection up.
func (r *ResultList) MoveUp() {
	if r.selected > 0 {
		r.selected--
	}
}

// MoveDown moves selection down.
func (r *ResultList) MoveDown() {
	if r.selected < len(r.results)-1 {
		r.selected++
	}
}

// SetDimensions sets the component dimensions.
func (r *ResultList) SetDimensions(width, height int) {
	r.width = width
	r.height = height
}

// Width returns the current width.
func (r *ResultList) Width() int {
	return r.width
}

// Height returns the current height.
func (r *ResultList) Height() int {
	return r.height
}

// Count returns the number of results.
func (r *ResultList) Count() int {
	return len(r.results)
}

// IsEmpty returns whether the list is empty.
func (r *ResultList) IsEmpty() bool {
	return len(r.results) == 0
}
