// Package docdetails provides the document details view component for the TUI.
package docdetails

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/custodia-labs/sercha-cli/internal/adapters/driving/tui/messages"
	"github.com/custodia-labs/sercha-cli/internal/adapters/driving/tui/styles"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driving"
)

// View is the document details view.
type View struct {
	styles *styles.Styles

	details      *driving.DocumentDetails
	scrollOffset int
	width        int
	height       int
	ready        bool
	err          error
}

// NewView creates a new document details view.
func NewView(s *styles.Styles) *View {
	return &View{
		styles: s,
	}
}

// SetDetails sets the document details to display.
func (v *View) SetDetails(details *driving.DocumentDetails) {
	v.details = details
	v.scrollOffset = 0
	v.err = nil
}

// SetError sets an error to display.
func (v *View) SetError(err error) {
	v.err = err
}

// Init initialises the view.
func (v *View) Init() tea.Cmd {
	return nil
}

// Update handles messages for the document details view.
func (v *View) Update(msg tea.Msg) (*View, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
		v.ready = true
		return v, nil

	case tea.KeyMsg:
		return v.handleKeyMsg(msg)

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
	case "c":
		// Copy path - stub for now
		return v, nil
	case "esc":
		return v, func() tea.Msg {
			return messages.ViewChanged{View: messages.ViewDocuments}
		}
	}

	return v, nil
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
	lines := v.buildContent()
	maxOffset := len(lines) - v.visibleLines()
	if maxOffset < 0 {
		maxOffset = 0
	}
	return maxOffset
}

// buildContent builds the content lines for display.
func (v *View) buildContent() []string {
	if v.details == nil {
		return nil
	}

	var lines []string

	// Basic info
	lines = append(lines,
		v.formatField("ID", v.details.ID),
		v.formatField("Title", v.details.Title),
		v.formatField("Source", fmt.Sprintf("%s (%s)", v.details.SourceName, v.details.SourceType)),
		v.formatField("URI", v.details.URI),
		v.formatField("Chunks", fmt.Sprintf("%d", v.details.ChunkCount)))

	// Timestamps
	if !v.details.CreatedAt.IsZero() {
		lines = append(lines, v.formatField("Created", v.details.CreatedAt.Format("2006-01-02 15:04:05")))
	}
	if !v.details.UpdatedAt.IsZero() {
		lines = append(lines, v.formatField("Updated", v.details.UpdatedAt.Format("2006-01-02 15:04:05")))
	}

	// Metadata section
	if len(v.details.Metadata) > 0 {
		lines = append(lines, "", "Metadata:")

		// Sort keys for consistent display
		keys := make([]string, 0, len(v.details.Metadata))
		for k := range v.details.Metadata {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, key := range keys {
			value := v.details.Metadata[key]
			// Truncate long values
			if len(value) > 50 {
				value = value[:47] + "..."
			}
			lines = append(lines, fmt.Sprintf("  %s: %s", key, value))
		}
	}

	return lines
}

// formatField formats a field for display.
func (v *View) formatField(label, value string) string {
	return fmt.Sprintf("%-12s %s", label+":", value)
}

// View renders the document details view.
func (v *View) View() string {
	var b strings.Builder

	// Title
	b.WriteString(v.styles.Title.Render("Document Details"))
	b.WriteString("\n")

	// Separator
	b.WriteString(strings.Repeat("─", minInt(v.width-4, 60)))
	b.WriteString("\n\n")

	// Error state
	if v.err != nil {
		b.WriteString(v.styles.Error.Render(fmt.Sprintf("Error: %s", v.err.Error())))
		b.WriteString("\n\n")
		b.WriteString(v.renderHelp())
		return b.String()
	}

	// No details
	if v.details == nil {
		b.WriteString(v.styles.Muted.Render("No document details available"))
		b.WriteString("\n\n")
		b.WriteString(v.renderHelp())
		return b.String()
	}

	// Content
	lines := v.buildContent()
	visibleLines := v.visibleLines()
	for i := v.scrollOffset; i < len(lines) && i < v.scrollOffset+visibleLines; i++ {
		line := lines[i]

		// Style based on content
		//nolint:nestif // View rendering requires nested conditional styling
		if strings.HasPrefix(line, "Metadata:") {
			b.WriteString(v.styles.Subtitle.Render(line))
		} else if strings.HasPrefix(line, "  ") {
			// Metadata key-value
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				b.WriteString(v.styles.Muted.Render(parts[0] + ":"))
				b.WriteString(v.styles.Normal.Render(parts[1]))
			} else {
				b.WriteString(v.styles.Muted.Render(line))
			}
		} else if strings.Contains(line, ":") {
			// Field label-value
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				b.WriteString(v.styles.Subtitle.Render(parts[0] + ":"))
				b.WriteString(v.styles.Normal.Render(parts[1]))
			} else {
				b.WriteString(v.styles.Normal.Render(line))
			}
		} else {
			b.WriteString(v.styles.Normal.Render(line))
		}
		b.WriteString("\n")
	}

	// Scroll indicator
	if len(lines) > visibleLines {
		b.WriteString("\n")
		b.WriteString(v.styles.Muted.Render(fmt.Sprintf("  [Line %d-%d of %d]",
			v.scrollOffset+1,
			minInt(v.scrollOffset+visibleLines, len(lines)),
			len(lines))))
	}

	b.WriteString("\n\n")
	b.WriteString(v.renderHelp())

	return b.String()
}

// renderHelp renders the help footer.
func (v *View) renderHelp() string {
	return v.styles.Help.Render("[↑/↓] scroll  [c] copy path  [esc] back")
}

// SetDimensions sets the view dimensions.
func (v *View) SetDimensions(width, height int) {
	v.width = width
	v.height = height
	v.ready = true
}

// Details returns the current document details.
func (v *View) Details() *driving.DocumentDetails {
	return v.details
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
