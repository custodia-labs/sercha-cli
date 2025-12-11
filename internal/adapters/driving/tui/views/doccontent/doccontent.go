// Package doccontent provides the document content view component for the TUI.
package doccontent

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/custodia-labs/sercha-cli/internal/adapters/driving/tui/messages"
	"github.com/custodia-labs/sercha-cli/internal/adapters/driving/tui/styles"
	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driving"
)

// View is the document content view.
type View struct {
	styles          *styles.Styles
	documentService driving.DocumentService

	document     *domain.Document
	content      string
	lines        []string
	scrollOffset int
	width        int
	height       int
	ready        bool
	err          error
	loading      bool
}

// NewView creates a new document content view.
func NewView(s *styles.Styles, documentService driving.DocumentService) *View {
	return &View{
		styles:          s,
		documentService: documentService,
	}
}

// SetDocument sets the document and loads its content.
func (v *View) SetDocument(doc *domain.Document) tea.Cmd {
	v.document = doc
	v.content = ""
	v.lines = nil
	v.scrollOffset = 0
	v.err = nil
	return v.loadContent()
}

// Init initialises the view.
func (v *View) Init() tea.Cmd {
	return nil
}

// loadContent returns a command that loads the document content.
func (v *View) loadContent() tea.Cmd {
	return func() tea.Msg {
		if v.document == nil || v.documentService == nil {
			return messages.DocumentContentLoaded{Err: fmt.Errorf("document service not available")}
		}

		v.loading = true
		content, err := v.documentService.GetContent(context.Background(), v.document.ID)
		return messages.DocumentContentLoaded{
			DocumentID: v.document.ID,
			Content:    content,
			Err:        err,
		}
	}
}

// Update handles messages for the document content view.
func (v *View) Update(msg tea.Msg) (*View, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
		v.ready = true
		v.wrapContent()
		return v, nil

	case tea.KeyMsg:
		return v.handleKeyMsg(msg)

	case messages.DocumentContentLoaded:
		v.loading = false
		if msg.Err != nil {
			v.err = msg.Err
		} else {
			v.content = msg.Content
			v.wrapContent()
			v.err = nil
		}
		return v, nil

	case messages.ErrorOccurred:
		v.err = msg.Err
		return v, nil
	}

	return v, nil
}

// handleKeyMsg handles key presses.
func (v *View) handleKeyMsg(msg tea.KeyMsg) (*View, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if v.scrollOffset > 0 {
			v.scrollOffset--
		}
	case "down", "j":
		maxOffset := v.maxScrollOffset()
		if v.scrollOffset < maxOffset {
			v.scrollOffset++
		}
	case "pgup", "ctrl+u":
		v.scrollOffset -= v.visibleLines()
		if v.scrollOffset < 0 {
			v.scrollOffset = 0
		}
	case "pgdown", "ctrl+d":
		maxOffset := v.maxScrollOffset()
		v.scrollOffset += v.visibleLines()
		if v.scrollOffset > maxOffset {
			v.scrollOffset = maxOffset
		}
	case "home", "g":
		v.scrollOffset = 0
	case "end", "G":
		v.scrollOffset = v.maxScrollOffset()
	case "c":
		// Copy all content - stub for now
		return v, nil
	case "esc":
		return v, func() tea.Msg {
			return messages.ViewChanged{View: messages.ViewDocuments}
		}
	}

	return v, nil
}

// wrapContent wraps the content to fit the view width.
func (v *View) wrapContent() {
	if v.content == "" {
		v.lines = nil
		return
	}

	// Calculate available width (accounting for padding)
	contentWidth := v.width - 4
	if contentWidth < 20 {
		contentWidth = 20
	}

	// Split into lines and wrap long lines
	rawLines := strings.Split(v.content, "\n")
	v.lines = make([]string, 0, len(rawLines))

	for _, line := range rawLines {
		if len(line) <= contentWidth {
			v.lines = append(v.lines, line)
		} else {
			// Wrap long lines
			for len(line) > contentWidth {
				v.lines = append(v.lines, line[:contentWidth])
				line = line[contentWidth:]
			}
			if line != "" {
				v.lines = append(v.lines, line)
			}
		}
	}
}

// visibleLines returns the number of lines that can be displayed.
func (v *View) visibleLines() int {
	// Reserve lines for title, separator, help, and padding
	reserved := 6
	available := v.height - reserved
	if available < 1 {
		available = 1
	}
	return available
}

// maxScrollOffset returns the maximum scroll offset.
func (v *View) maxScrollOffset() int {
	maxOffset := len(v.lines) - v.visibleLines()
	if maxOffset < 0 {
		maxOffset = 0
	}
	return maxOffset
}

// View renders the document content view.
func (v *View) View() string {
	var b strings.Builder

	// Title
	title := "Document Content"
	if v.document != nil {
		docTitle := v.document.Title
		if docTitle == "" {
			docTitle = v.document.ID
		}
		title = docTitle
	}
	b.WriteString(v.styles.Title.Render(title))
	b.WriteString("\n")

	// Separator
	b.WriteString(strings.Repeat("─", minInt(v.width-4, 60)))
	b.WriteString("\n\n")

	// Loading state
	if v.loading {
		b.WriteString(v.styles.Muted.Render("Loading content..."))
		b.WriteString("\n\n")
		b.WriteString(v.renderHelp())
		return b.String()
	}

	// Error state
	if v.err != nil {
		b.WriteString(v.styles.Error.Render(fmt.Sprintf("Error: %s", v.err.Error())))
		b.WriteString("\n\n")
		b.WriteString(v.renderHelp())
		return b.String()
	}

	// Empty content
	if len(v.lines) == 0 {
		b.WriteString(v.styles.Muted.Render("(No content)"))
		b.WriteString("\n\n")
		b.WriteString(v.renderHelp())
		return b.String()
	}

	// Content
	visibleLines := v.visibleLines()
	for i := v.scrollOffset; i < len(v.lines) && i < v.scrollOffset+visibleLines; i++ {
		b.WriteString(v.styles.Normal.Render(v.lines[i]))
		b.WriteString("\n")
	}

	// Scroll position indicator
	if len(v.lines) > visibleLines {
		b.WriteString("\n")
		percentage := 0
		if v.maxScrollOffset() > 0 {
			percentage = v.scrollOffset * 100 / v.maxScrollOffset()
		}
		b.WriteString(v.styles.Muted.Render(fmt.Sprintf("  [%d%%] Line %d-%d of %d",
			percentage,
			v.scrollOffset+1,
			minInt(v.scrollOffset+visibleLines, len(v.lines)),
			len(v.lines))))
	}

	b.WriteString("\n\n")
	b.WriteString(v.renderHelp())

	return b.String()
}

// renderHelp renders the help footer.
func (v *View) renderHelp() string {
	return v.styles.Help.Render("[↑/↓/PgUp/PgDn] scroll  [g/G] top/bottom  [c] copy all  [esc] back")
}

// SetDimensions sets the view dimensions.
func (v *View) SetDimensions(width, height int) {
	v.width = width
	v.height = height
	v.ready = true
	v.wrapContent()
}

// Document returns the current document.
func (v *View) Document() *domain.Document {
	return v.document
}

// Content returns the document content.
func (v *View) Content() string {
	return v.content
}

// Err returns the last error.
func (v *View) Err() error {
	return v.err
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
