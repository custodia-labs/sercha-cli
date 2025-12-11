// Package documents provides the documents list view component for the TUI.
package documents

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

// ActionOption represents a document action.
type ActionOption int

const (
	ActionShowContent ActionOption = iota
	ActionShowDetails
	ActionOpenDocument
	ActionRefresh
	ActionExclude
	ActionCancel
)

// View is the documents list view.
type View struct {
	styles          *styles.Styles
	documentService driving.DocumentService

	source       *domain.Source
	documents    []domain.Document
	selected     int
	width        int
	height       int
	ready        bool
	err          error
	loading      bool
	showingMenu  bool
	menuSelected ActionOption
	scrollOffset int
}

// NewView creates a new documents view.
func NewView(s *styles.Styles, documentService driving.DocumentService) *View {
	return &View{
		styles:          s,
		documentService: documentService,
		documents:       []domain.Document{},
	}
}

// SetSource sets the source and loads its documents.
func (v *View) SetSource(source domain.Source) tea.Cmd {
	v.source = &source
	v.documents = []domain.Document{}
	v.selected = 0
	v.scrollOffset = 0
	v.err = nil
	v.showingMenu = false
	return v.loadDocuments()
}

// Init initialises the view.
func (v *View) Init() tea.Cmd {
	return nil
}

// loadDocuments returns a command that loads documents for the source.
func (v *View) loadDocuments() tea.Cmd {
	return func() tea.Msg {
		if v.source == nil || v.documentService == nil {
			return messages.DocumentsLoaded{Err: fmt.Errorf("document service not available")}
		}

		v.loading = true
		docs, err := v.documentService.ListBySource(context.Background(), v.source.ID)
		return messages.DocumentsLoaded{
			SourceID:  v.source.ID,
			Documents: docs,
			Err:       err,
		}
	}
}

// Update handles messages for the documents view.
func (v *View) Update(msg tea.Msg) (*View, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
		v.ready = true
		return v, nil

	case tea.KeyMsg:
		if v.showingMenu {
			return v.handleMenuKeyMsg(msg)
		}
		return v.handleKeyMsg(msg)

	case messages.DocumentsLoaded:
		v.loading = false
		if msg.Err != nil {
			v.err = msg.Err
		} else {
			v.documents = msg.Documents
			v.err = nil
		}
		return v, nil

	case messages.DocumentExcluded:
		if msg.Err != nil {
			v.err = msg.Err
		} else {
			// Reload documents after exclusion
			cmd := v.loadDocuments()
			return v, cmd
		}
		return v, nil

	case messages.DocumentRefreshed:
		if msg.Err != nil {
			v.err = msg.Err
		}
		return v, nil

	case messages.ErrorOccurred:
		v.err = msg.Err
		return v, nil
	}

	return v, nil
}

// handleKeyMsg handles key presses in list mode.
func (v *View) handleKeyMsg(msg tea.KeyMsg) (*View, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if v.selected > 0 {
			v.selected--
			v.adjustScroll()
		}
	case "down", "j":
		if v.selected < len(v.documents)-1 {
			v.selected++
			v.adjustScroll()
		}
	case "enter":
		if len(v.documents) > 0 {
			v.showingMenu = true
			v.menuSelected = ActionShowContent
		}
	case "esc":
		return v, func() tea.Msg {
			return messages.ViewChanged{View: messages.ViewSourceDetail}
		}
	case "r":
		// Reload documents
		v.loading = true
		cmd := v.loadDocuments()
		return v, cmd
	}

	return v, nil
}

// handleMenuKeyMsg handles key presses in action menu mode.
func (v *View) handleMenuKeyMsg(msg tea.KeyMsg) (*View, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if v.menuSelected > ActionShowContent {
			v.menuSelected--
		}
	case "down", "j":
		if v.menuSelected < ActionCancel {
			v.menuSelected++
		}
	case "enter":
		return v.handleMenuSelect()
	case "esc":
		v.showingMenu = false
	}

	return v, nil
}

// handleMenuSelect handles selection of an action.
func (v *View) handleMenuSelect() (*View, tea.Cmd) {
	if v.selected >= len(v.documents) {
		v.showingMenu = false
		return v, nil
	}

	doc := v.documents[v.selected]

	switch v.menuSelected {
	case ActionShowContent:
		v.showingMenu = false
		return v, func() tea.Msg {
			return messages.DocumentSelected{Document: doc}
		}
	case ActionShowDetails:
		v.showingMenu = false
		cmd := v.loadDocDetails(doc.ID)
		return v, cmd
	case ActionOpenDocument:
		v.showingMenu = false
		cmd := v.openDocument(doc.ID)
		return v, cmd
	case ActionRefresh:
		v.showingMenu = false
		cmd := v.refreshDocument(doc.ID)
		return v, cmd
	case ActionExclude:
		v.showingMenu = false
		cmd := v.excludeDocument(doc.ID)
		return v, cmd
	case ActionCancel:
		v.showingMenu = false
	}

	return v, nil
}

// loadDocDetails returns a command that loads document details.
func (v *View) loadDocDetails(docID string) tea.Cmd {
	return func() tea.Msg {
		if v.documentService == nil {
			return messages.ErrorOccurred{Err: fmt.Errorf("document service not available")}
		}

		details, err := v.documentService.GetDetails(context.Background(), docID)
		return messages.DocumentDetailsLoaded{
			DocumentID: docID,
			Details:    details,
			Err:        err,
		}
	}
}

// openDocument returns a command that opens the document.
func (v *View) openDocument(docID string) tea.Cmd {
	return func() tea.Msg {
		if v.documentService == nil {
			return messages.ErrorOccurred{Err: fmt.Errorf("document service not available")}
		}

		err := v.documentService.Open(context.Background(), docID)
		if err != nil {
			return messages.ErrorOccurred{Err: err}
		}
		return nil
	}
}

// refreshDocument returns a command that refreshes the document.
func (v *View) refreshDocument(docID string) tea.Cmd {
	return func() tea.Msg {
		if v.documentService == nil {
			return messages.DocumentRefreshed{DocumentID: docID, Err: fmt.Errorf("document service not available")}
		}

		err := v.documentService.Refresh(context.Background(), docID)
		return messages.DocumentRefreshed{DocumentID: docID, Err: err}
	}
}

// excludeDocument returns a command that excludes the document.
func (v *View) excludeDocument(docID string) tea.Cmd {
	return func() tea.Msg {
		if v.documentService == nil {
			return messages.DocumentExcluded{DocumentID: docID, Err: fmt.Errorf("document service not available")}
		}

		err := v.documentService.Exclude(context.Background(), docID, "user excluded")
		return messages.DocumentExcluded{DocumentID: docID, Err: err}
	}
}

// adjustScroll adjusts the scroll offset to keep the selected item visible.
func (v *View) adjustScroll() {
	visibleItems := v.visibleItemCount()
	if v.selected < v.scrollOffset {
		v.scrollOffset = v.selected
	} else if v.selected >= v.scrollOffset+visibleItems {
		v.scrollOffset = v.selected - visibleItems + 1
	}
}

// visibleItemCount returns the number of items that can be displayed.
func (v *View) visibleItemCount() int {
	// Reserve lines for title, separator, help, and padding
	reserved := 8
	available := v.height - reserved
	if available < 1 {
		available = 1
	}
	return available
}

// View renders the documents view.
func (v *View) View() string {
	var b strings.Builder

	// Title
	sourceName := "Unknown"
	if v.source != nil {
		sourceName = v.source.Name
	}
	title := fmt.Sprintf("Documents - %s (%d)", sourceName, len(v.documents))
	b.WriteString(v.styles.Title.Render(title))
	b.WriteString("\n\n")

	// Loading state
	if v.loading {
		b.WriteString(v.styles.Muted.Render("Loading documents..."))
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

	// Empty state
	if len(v.documents) == 0 {
		b.WriteString(v.styles.Muted.Render("No documents indexed for this source."))
		b.WriteString("\n\n")
		b.WriteString(v.renderHelp())
		return b.String()
	}

	// Action menu overlay
	if v.showingMenu {
		b.WriteString(v.renderActionMenu())
		return b.String()
	}

	// Documents list
	visibleItems := v.visibleItemCount()
	for i := v.scrollOffset; i < len(v.documents) && i < v.scrollOffset+visibleItems; i++ {
		line := v.renderDocument(i, &v.documents[i])
		b.WriteString(line)
		b.WriteString("\n")
	}

	// Scroll indicator
	if len(v.documents) > visibleItems {
		b.WriteString("\n")
		b.WriteString(v.styles.Muted.Render(fmt.Sprintf("  [%d-%d of %d]",
			v.scrollOffset+1,
			min(v.scrollOffset+visibleItems, len(v.documents)),
			len(v.documents))))
	}

	b.WriteString("\n\n")
	b.WriteString(v.renderHelp())

	return b.String()
}

// renderDocument renders a single document line.
func (v *View) renderDocument(index int, doc *domain.Document) string {
	indicator := "  "
	if index == v.selected {
		indicator = "> "
	}

	title := doc.Title
	if title == "" {
		title = doc.ID
	}

	// Truncate title if needed
	maxTitleLen := v.width/2 - 4
	if maxTitleLen < 10 {
		maxTitleLen = 10
	}
	if len(title) > maxTitleLen {
		title = title[:maxTitleLen-3] + "..."
	}

	// Truncate URI if needed
	uri := doc.URI
	maxURILen := v.width/2 - 4
	if maxURILen < 10 {
		maxURILen = 10
	}
	if len(uri) > maxURILen {
		uri = "..." + uri[len(uri)-maxURILen+3:]
	}

	if index == v.selected {
		return v.styles.Selected.Render(fmt.Sprintf("%s%-*s  %s", indicator, maxTitleLen, title, uri))
	}

	return v.styles.Normal.Render(indicator) +
		v.styles.Normal.Render(fmt.Sprintf("%-*s  ", maxTitleLen, title)) +
		v.styles.Muted.Render(uri)
}

// renderActionMenu renders the action menu overlay.
func (v *View) renderActionMenu() string {
	var b strings.Builder

	// Show selected document context
	if v.selected < len(v.documents) {
		doc := v.documents[v.selected]
		title := doc.Title
		if title == "" {
			title = doc.ID
		}
		b.WriteString(v.styles.Subtitle.Render(fmt.Sprintf("Actions for: %s", title)))
		b.WriteString("\n\n")
	}

	// Menu options
	options := []struct {
		action ActionOption
		label  string
	}{
		{ActionShowContent, "Show Content"},
		{ActionShowDetails, "Show Details"},
		{ActionOpenDocument, "Open Document"},
		{ActionRefresh, "Refresh"},
		{ActionExclude, "Remove (Exclude)"},
		{ActionCancel, "Cancel"},
	}

	for _, opt := range options {
		indicator := "  "
		if v.menuSelected == opt.action {
			indicator = "> "
			b.WriteString(v.styles.Selected.Render(fmt.Sprintf("%s%s", indicator, opt.label)))
		} else {
			b.WriteString(v.styles.Normal.Render(fmt.Sprintf("%s%s", indicator, opt.label)))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(v.styles.Help.Render("[↑/↓] navigate  [enter] select  [esc] cancel"))

	return b.String()
}

// renderHelp renders the help footer.
func (v *View) renderHelp() string {
	return v.styles.Help.Render("[↑/↓] navigate  [enter] actions  [r] reload  [esc] back")
}

// SetDimensions sets the view dimensions.
func (v *View) SetDimensions(width, height int) {
	v.width = width
	v.height = height
	v.ready = true
}

// Documents returns the current list of documents.
func (v *View) Documents() []domain.Document {
	return v.documents
}

// SelectedIndex returns the currently selected document index.
func (v *View) SelectedIndex() int {
	return v.selected
}

// SelectedDocument returns the currently selected document.
func (v *View) SelectedDocument() *domain.Document {
	if v.selected < len(v.documents) {
		return &v.documents[v.selected]
	}
	return nil
}

// IsShowingMenu returns true if the action menu is visible.
func (v *View) IsShowingMenu() bool {
	return v.showingMenu
}

// Err returns the last error.
func (v *View) Err() error {
	return v.err
}
