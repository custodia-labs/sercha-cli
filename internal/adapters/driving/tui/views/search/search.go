// Package search provides the main search view for the TUI.
package search

import (
	"context"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/custodia-labs/sercha-cli/internal/adapters/driving/tui/components/input"
	"github.com/custodia-labs/sercha-cli/internal/adapters/driving/tui/components/list"
	"github.com/custodia-labs/sercha-cli/internal/adapters/driving/tui/components/status"
	"github.com/custodia-labs/sercha-cli/internal/adapters/driving/tui/keymap"
	"github.com/custodia-labs/sercha-cli/internal/adapters/driving/tui/messages"
	"github.com/custodia-labs/sercha-cli/internal/adapters/driving/tui/styles"
	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driving"
)

// ActionMenu represents a simple action selection overlay.
type ActionMenu struct {
	actions  []string
	selected int
	visible  bool
	result   *domain.SearchResult
}

// View represents the search view with input, results list, and status bar.
type View struct {
	styles    *styles.Styles
	keymap    *keymap.KeyMap
	input     *input.SearchInput
	list      *list.ResultList
	statusbar *status.Bar

	searchService driving.SearchService
	actionService driving.ResultActionService
	ctx           context.Context

	width      int
	height     int
	ready      bool
	err        error
	focusInput bool // true = input mode (typing), false = results mode (navigating)
	actionMenu *ActionMenu
}

// NewView creates a new search view.
func NewView(
	s *styles.Styles,
	km *keymap.KeyMap,
	searchService driving.SearchService,
	actionService driving.ResultActionService,
) *View {
	if s == nil {
		s = styles.DefaultStyles()
	}
	if km == nil {
		km = keymap.DefaultKeyMap()
	}

	return &View{
		styles:        s,
		keymap:        km,
		input:         input.NewSearchInput(s),
		list:          list.NewResultList(s),
		statusbar:     status.NewBar(s, km),
		searchService: searchService,
		actionService: actionService,
		ctx:           context.Background(),
		width:         80,
		height:        24,
		ready:         false,
		focusInput:    true, // Start in input mode
		actionMenu:    nil,
	}
}

// WithContext sets the context for the view.
func (v *View) WithContext(ctx context.Context) *View {
	v.ctx = ctx
	return v
}

// Init initialises the view.
func (v *View) Init() tea.Cmd {
	return v.input.Init()
}

// Update handles messages for the search view.
func (v *View) Update(msg tea.Msg) (*View, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.SetDimensions(msg.Width, msg.Height)
		v.ready = true
		return v, nil

	case tea.KeyMsg:
		return v.handleKeyMsg(msg)

	case messages.SearchCompleted:
		v.handleSearchCompleted(msg)
		return v, nil

	case messages.ErrorOccurred:
		v.err = msg.Err
		v.statusbar.SetState(status.StateError)
		v.statusbar.SetMessage(msg.Err.Error())
		return v, nil
	}

	// Forward to input component
	var inputCmd tea.Cmd
	v.input, inputCmd = v.input.Update(msg)
	if inputCmd != nil {
		cmds = append(cmds, inputCmd)
	}

	// Forward to list component
	var listCmd tea.Cmd
	v.list, listCmd = v.list.Update(msg)
	if listCmd != nil {
		cmds = append(cmds, listCmd)
	}

	return v, tea.Batch(cmds...)
}

// handleKeyMsg processes keyboard input.
func (v *View) handleKeyMsg(msg tea.KeyMsg) (*View, tea.Cmd) {
	// If action menu is visible, handle its keys
	if v.actionMenu != nil && v.actionMenu.visible {
		return v.handleActionMenuKey(msg)
	}

	// Esc always signals to go back to menu
	if msg.Type == tea.KeyEsc {
		return v, func() tea.Msg {
			return messages.ViewChanged{View: messages.ViewMenu}
		}
	}

	// Enter in input mode submits search
	if msg.Type == tea.KeyEnter && v.focusInput {
		query := v.input.Value()
		if query == "" {
			return v, nil
		}
		v.statusbar.SetState(status.StateSearching)
		v.focusInput = false // Move to results mode after search
		v.input.Blur()
		cmd := v.performSearch(query)
		return v, cmd
	}

	// Input mode: all keys go to input
	if v.focusInput {
		v.input, _ = v.input.Update(msg)
		return v, nil
	}

	// Results mode: handle Enter to open action menu
	if msg.Type == tea.KeyEnter {
		result := v.list.SelectedResult()
		if result != nil {
			v.actionMenu = &ActionMenu{
				actions:  []string{"Copy plain text", "Open Document", "Cancel"},
				selected: 0,
				visible:  true,
				result:   result,
			}
		}
		return v, nil
	}

	// Results mode: handle navigation
	//nolint:exhaustive // handling only relevant key types
	switch msg.Type {
	case tea.KeyUp:
		v.list.MoveUp()
		return v, nil
	case tea.KeyDown:
		v.list.MoveDown()
		return v, nil
	}

	switch msg.String() {
	case "k":
		v.list.MoveUp()
		return v, nil
	case "j":
		v.list.MoveDown()
		return v, nil
	case "n":
		// New search: clear input and focus it
		v.focusInput = true
		v.input.Focus()
		v.input.SetValue("")
		return v, nil
	}

	return v, nil
}

// handleActionMenuKey processes keyboard input when action menu is visible.
func (v *View) handleActionMenuKey(msg tea.KeyMsg) (*View, tea.Cmd) {
	//nolint:exhaustive // handling only relevant key types
	switch msg.Type {
	case tea.KeyUp:
		if v.actionMenu.selected > 0 {
			v.actionMenu.selected--
		}
		return v, nil
	case tea.KeyDown:
		if v.actionMenu.selected < len(v.actionMenu.actions)-1 {
			v.actionMenu.selected++
		}
		return v, nil
	case tea.KeyEnter:
		action := v.actionMenu.actions[v.actionMenu.selected]
		result := v.actionMenu.result
		v.actionMenu = nil // Close menu
		return v.executeAction(action, result)
	case tea.KeyEsc:
		v.actionMenu = nil // Close menu
		return v, nil
	default:
		// Handle other keys
	}

	// Handle vim-style navigation in action menu
	switch msg.String() {
	case "k":
		if v.actionMenu.selected > 0 {
			v.actionMenu.selected--
		}
		return v, nil
	case "j":
		if v.actionMenu.selected < len(v.actionMenu.actions)-1 {
			v.actionMenu.selected++
		}
		return v, nil
	}

	return v, nil
}

// executeAction performs the selected action on a search result.
func (v *View) executeAction(action string, result *domain.SearchResult) (*View, tea.Cmd) {
	if result == nil {
		return v, nil
	}

	switch action {
	case "Copy plain text":
		if v.actionService != nil {
			err := v.actionService.CopyToClipboard(v.ctx, result)
			if err != nil {
				v.statusbar.SetMessage("Copy: " + err.Error())
			} else {
				v.statusbar.SetMessage("Copied to clipboard")
			}
		} else {
			v.statusbar.SetMessage("Copy not available")
		}
	case "Open Document":
		if v.actionService != nil {
			err := v.actionService.OpenDocument(v.ctx, result)
			if err != nil {
				v.statusbar.SetMessage("Open: " + err.Error())
			} else {
				v.statusbar.SetMessage("Opening document...")
			}
		} else {
			v.statusbar.SetMessage("Open not available")
		}
	case "Cancel":
		// Do nothing, menu is already closed
	}

	return v, nil
}

// performSearch executes a search and returns results.
func (v *View) performSearch(query string) tea.Cmd {
	return func() tea.Msg {
		if v.searchService == nil {
			return messages.ErrorOccurred{Err: ErrNoSearchService}
		}

		results, err := v.searchService.Search(v.ctx, query, domain.SearchOptions{})
		if err != nil {
			return messages.SearchCompleted{Results: nil, Err: err}
		}
		return messages.SearchCompleted{Results: results, Err: nil}
	}
}

// handleSearchCompleted processes search results.
func (v *View) handleSearchCompleted(msg messages.SearchCompleted) {
	if msg.Err != nil {
		v.err = msg.Err
		v.statusbar.SetState(status.StateError)
		v.statusbar.SetMessage(msg.Err.Error())
		return
	}

	v.err = nil
	v.list.SetResults(msg.Results)
	v.statusbar.SetState(status.StateResults)
	v.statusbar.SetResultCount(len(msg.Results))

	// Switch to results mode after successful search
	v.focusInput = false
	v.input.Blur()
}

// View renders the search view.
func (v *View) View() string {
	if !v.ready {
		return "Initialising..."
	}

	sections := make([]string, 0, 10)

	// Header
	header := v.styles.Title.Render("Sercha")
	sections = append(sections, header, "")

	// Search input
	inputView := v.input.View()
	sections = append(sections, inputView, "")

	// Error display
	if v.err != nil {
		errView := v.styles.Error.Render("Error: " + v.err.Error())
		sections = append(sections, errView, "")
	}

	// Results list
	listView := v.list.View()
	sections = append(sections, listView)

	// Action menu overlay (if visible)
	if v.actionMenu != nil && v.actionMenu.visible {
		sections = append(sections, "")
		menuView := v.renderActionMenu()
		sections = append(sections, menuView)
	}

	// Status bar at bottom
	sections = append(sections, "")
	statusView := v.statusbar.View()
	sections = append(sections, statusView)

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// renderActionMenu renders the action menu overlay.
func (v *View) renderActionMenu() string {
	if v.actionMenu == nil {
		return ""
	}

	lines := make([]string, 0, len(v.actionMenu.actions))
	for i, action := range v.actionMenu.actions {
		indicator := "  "
		if i == v.actionMenu.selected {
			indicator = "> "
		}

		var line string
		if i == v.actionMenu.selected {
			line = v.styles.Selected.Render(indicator + action)
		} else {
			line = v.styles.Normal.Render(indicator + action)
		}
		lines = append(lines, line)
	}

	content := strings.Join(lines, "\n")

	// Wrap in a bordered box
	menuStyle := v.styles.Border.
		Padding(0, 1)

	return menuStyle.Render(content)
}

// SetDimensions sets the view dimensions.
func (v *View) SetDimensions(width, height int) {
	v.width = width
	v.height = height
	v.ready = true

	// Allocate space to components
	v.input.SetWidth(width)
	v.list.SetDimensions(width, height-10) // Reserve space for header, input, status
	v.statusbar.SetWidth(width)
}

// Width returns the current width.
func (v *View) Width() int {
	return v.width
}

// Height returns the current height.
func (v *View) Height() int {
	return v.height
}

// Ready returns whether the view is ready to render.
func (v *View) Ready() bool {
	return v.ready
}

// Query returns the current search query.
func (v *View) Query() string {
	return v.input.Value()
}

// SetQuery sets the search query.
func (v *View) SetQuery(query string) {
	v.input.SetValue(query)
}

// Results returns the current search results.
func (v *View) Results() []domain.SearchResult {
	return v.list.Results()
}

// SelectedIndex returns the index of the selected result.
func (v *View) SelectedIndex() int {
	return v.list.Selected()
}

// SelectedResult returns the currently selected result.
func (v *View) SelectedResult() *domain.SearchResult {
	return v.list.SelectedResult()
}

// Err returns the current error, if any.
func (v *View) Err() error {
	return v.err
}

// ClearError clears the current error.
func (v *View) ClearError() {
	v.err = nil
	v.statusbar.SetState(status.StateReady)
	v.statusbar.SetMessage("")
}

// Reset resets the view to initial input mode.
func (v *View) Reset() {
	v.focusInput = true
	v.input.Focus()
	v.input.SetValue("")
	v.list.SetResults(nil)
	v.err = nil
	v.statusbar.SetState(status.StateReady)
	v.statusbar.SetMessage("")
}

// InputFocused returns whether the input has focus.
func (v *View) InputFocused() bool {
	return v.focusInput
}
