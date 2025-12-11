// Package sourcedetail provides the source detail view component for the TUI.
package sourcedetail

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

// MenuOption represents an action in the source detail menu.
type MenuOption int

const (
	OptionViewDocuments MenuOption = iota
	OptionSyncNow
	OptionDeleteSource
	OptionBack
)

// View is the source detail view.
type View struct {
	styles           *styles.Styles
	sourceService    driving.SourceService
	syncOrchestrator driving.SyncOrchestrator
	documentService  driving.DocumentService

	source   *domain.Source
	docCount int
	selected MenuOption
	width    int
	height   int
	ready    bool
	err      error
	syncing  bool
	deleting bool
}

// NewView creates a new source detail view.
func NewView(
	s *styles.Styles,
	sourceService driving.SourceService,
	syncOrchestrator driving.SyncOrchestrator,
	documentService driving.DocumentService,
) *View {
	return &View{
		styles:           s,
		sourceService:    sourceService,
		syncOrchestrator: syncOrchestrator,
		documentService:  documentService,
		selected:         OptionViewDocuments,
	}
}

// SetSource sets the source to display details for.
func (v *View) SetSource(source domain.Source) {
	v.source = &source
	v.err = nil
	v.syncing = false
	v.deleting = false
	v.selected = OptionViewDocuments
}

// Init initialises the view.
func (v *View) Init() tea.Cmd {
	return v.loadDocCount()
}

// loadDocCount returns a command that counts documents for the source.
func (v *View) loadDocCount() tea.Cmd {
	return func() tea.Msg {
		if v.source == nil || v.documentService == nil {
			return nil
		}

		docs, err := v.documentService.ListBySource(context.Background(), v.source.ID)
		if err != nil {
			return messages.ErrorOccurred{Err: err}
		}
		v.docCount = len(docs)
		return nil
	}
}

// Update handles messages for the source detail view.
func (v *View) Update(msg tea.Msg) (*View, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
		v.ready = true
		return v, nil

	case tea.KeyMsg:
		return v.handleKeyMsg(msg)

	case messages.SourceRemoved:
		v.deleting = false
		if msg.Err != nil {
			v.err = msg.Err
		} else {
			// Navigate back after deletion
			return v, func() tea.Msg {
				return messages.ViewChanged{View: messages.ViewSources}
			}
		}
		return v, nil

	case messages.ErrorOccurred:
		v.err = msg.Err
		v.syncing = false
		return v, nil
	}

	return v, nil
}

// handleKeyMsg handles key presses.
func (v *View) handleKeyMsg(msg tea.KeyMsg) (*View, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if v.selected > OptionViewDocuments {
			v.selected--
		}
	case "down", "j":
		if v.selected < OptionBack {
			v.selected++
		}
	case "enter":
		return v.handleSelect()
	case "esc":
		return v, func() tea.Msg {
			return messages.ViewChanged{View: messages.ViewSources}
		}
	}

	return v, nil
}

// handleSelect handles selection of a menu option.
func (v *View) handleSelect() (*View, tea.Cmd) {
	switch v.selected {
	case OptionViewDocuments:
		if v.source != nil {
			return v, func() tea.Msg {
				return messages.SourceSelected{Source: *v.source}
			}
		}
	case OptionSyncNow:
		cmd := v.syncSource()
		return v, cmd
	case OptionDeleteSource:
		cmd := v.deleteSource()
		return v, cmd
	case OptionBack:
		return v, func() tea.Msg {
			return messages.ViewChanged{View: messages.ViewSources}
		}
	}
	return v, nil
}

// syncSource returns a command that syncs the source.
func (v *View) syncSource() tea.Cmd {
	return func() tea.Msg {
		if v.source == nil || v.syncOrchestrator == nil {
			return messages.ErrorOccurred{Err: fmt.Errorf("sync not available")}
		}

		v.syncing = true
		err := v.syncOrchestrator.Sync(context.Background(), v.source.ID)
		if err != nil {
			return messages.ErrorOccurred{Err: err}
		}
		v.syncing = false
		return nil
	}
}

// deleteSource returns a command that deletes the source.
func (v *View) deleteSource() tea.Cmd {
	return func() tea.Msg {
		if v.source == nil || v.sourceService == nil {
			return messages.SourceRemoved{Err: fmt.Errorf("source service not available")}
		}

		v.deleting = true
		err := v.sourceService.Remove(context.Background(), v.source.ID)
		return messages.SourceRemoved{ID: v.source.ID, Err: err}
	}
}

// View renders the source detail view.
func (v *View) View() string {
	if v.source == nil {
		return v.styles.Muted.Render("No source selected")
	}

	var b strings.Builder

	// Title
	b.WriteString(v.styles.Title.Render(fmt.Sprintf("Source: %s", v.source.Name)))
	b.WriteString("\n\n")

	// Source info
	b.WriteString(v.styles.Subtitle.Render("Type: "))
	b.WriteString(v.styles.Normal.Render(v.source.Type))
	b.WriteString("\n")

	b.WriteString(v.styles.Subtitle.Render("ID: "))
	b.WriteString(v.styles.Muted.Render(v.source.ID))
	b.WriteString("\n")

	b.WriteString(v.styles.Subtitle.Render("Documents: "))
	b.WriteString(v.styles.Normal.Render(fmt.Sprintf("%d", v.docCount)))
	b.WriteString("\n\n")

	// Error state
	if v.err != nil {
		b.WriteString(v.styles.Error.Render(fmt.Sprintf("Error: %s", v.err.Error())))
		b.WriteString("\n\n")
	}

	// Status
	if v.syncing {
		b.WriteString(v.styles.Muted.Render("Syncing..."))
		b.WriteString("\n\n")
	}
	if v.deleting {
		b.WriteString(v.styles.Muted.Render("Deleting..."))
		b.WriteString("\n\n")
	}

	// Menu separator
	b.WriteString(strings.Repeat("─", minInt(40, v.width-4)))
	b.WriteString("\n\n")

	// Menu options
	options := []struct {
		option MenuOption
		label  string
	}{
		{OptionViewDocuments, "View Documents"},
		{OptionSyncNow, "Sync Now"},
		{OptionDeleteSource, "Delete Source"},
		{OptionBack, "Back"},
	}

	for _, opt := range options {
		indicator := "  "
		if v.selected == opt.option {
			indicator = "> "
			b.WriteString(v.styles.Selected.Render(fmt.Sprintf("%s%s", indicator, opt.label)))
		} else {
			b.WriteString(v.styles.Normal.Render(fmt.Sprintf("%s%s", indicator, opt.label)))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(v.renderHelp())

	return b.String()
}

// renderHelp renders the help footer.
func (v *View) renderHelp() string {
	return v.styles.Help.Render("[↑/↓] navigate  [enter] select  [esc] back")
}

// SetDimensions sets the view dimensions.
func (v *View) SetDimensions(width, height int) {
	v.width = width
	v.height = height
	v.ready = true
}

// Source returns the current source.
func (v *View) Source() *domain.Source {
	return v.source
}

// SelectedOption returns the currently selected menu option.
func (v *View) SelectedOption() MenuOption {
	return v.selected
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
