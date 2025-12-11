// Package sources provides the sources view component for the TUI.
package sources

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

// View is the sources management view.
type View struct {
	styles             *styles.Styles
	sourceService      driving.SourceService
	credentialsService driving.CredentialsService

	sources            []domain.Source
	accountIdentifiers map[string]string // sourceID -> accountIdentifier
	selected           int
	width              int
	height             int
	ready              bool
	err                error
	loading            bool
}

// NewView creates a new sources view.
func NewView(
	s *styles.Styles,
	sourceService driving.SourceService,
	credentialsService driving.CredentialsService,
) *View {
	return &View{
		styles:             s,
		sourceService:      sourceService,
		credentialsService: credentialsService,
		sources:            []domain.Source{},
		accountIdentifiers: make(map[string]string),
	}
}

// Init initialises the view and loads sources.
func (v *View) Init() tea.Cmd {
	return v.loadSources()
}

// sourcesLoadedMsg extends messages.SourcesLoaded with account identifiers.
type sourcesLoadedMsg struct {
	messages.SourcesLoaded
	AccountIdentifiers map[string]string
}

// loadSources returns a command that loads sources from the service.
func (v *View) loadSources() tea.Cmd {
	return func() tea.Msg {
		if v.sourceService == nil {
			return sourcesLoadedMsg{
				SourcesLoaded:      messages.SourcesLoaded{Err: fmt.Errorf("source service not available")},
				AccountIdentifiers: nil,
			}
		}

		ctx := context.Background()
		sources, err := v.sourceService.List(ctx)
		if err != nil {
			return sourcesLoadedMsg{
				SourcesLoaded:      messages.SourcesLoaded{Err: err},
				AccountIdentifiers: nil,
			}
		}

		// Fetch account identifiers for sources with credentials
		accountIDs := v.fetchAccountIdentifiers(ctx, sources)

		return sourcesLoadedMsg{
			SourcesLoaded:      messages.SourcesLoaded{Sources: sources, Err: nil},
			AccountIdentifiers: accountIDs,
		}
	}
}

// fetchAccountIdentifiers retrieves account identifiers for sources with credentials.
func (v *View) fetchAccountIdentifiers(ctx context.Context, sources []domain.Source) map[string]string {
	accountIDs := make(map[string]string)
	if v.credentialsService == nil {
		return accountIDs
	}

	for i := range sources {
		src := &sources[i]
		if src.CredentialsID == "" {
			continue
		}
		creds, err := v.credentialsService.Get(ctx, src.CredentialsID)
		if err != nil || creds == nil {
			continue
		}
		if creds.AccountIdentifier != "" {
			accountIDs[src.ID] = creds.AccountIdentifier
		}
	}
	return accountIDs
}

// Update handles messages for the sources view.
func (v *View) Update(msg tea.Msg) (*View, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
		v.ready = true
		return v, nil

	case tea.KeyMsg:
		return v.handleKeyMsg(msg)

	case sourcesLoadedMsg:
		v.loading = false
		if msg.Err != nil {
			v.err = msg.Err
		} else {
			v.sources = msg.Sources
			v.accountIdentifiers = msg.AccountIdentifiers
			v.err = nil
		}
		return v, nil

	case messages.SourcesLoaded:
		// Also handle the base type for backward compatibility
		v.loading = false
		if msg.Err != nil {
			v.err = msg.Err
		} else {
			v.sources = msg.Sources
			v.err = nil
		}
		return v, nil

	case messages.SourceRemoved:
		if msg.Err != nil {
			v.err = msg.Err
		} else {
			// Reload sources after removal
			cmd := v.loadSources()
			return v, cmd
		}
		return v, nil
	}

	return v, nil
}

// handleKeyMsg handles key presses.
func (v *View) handleKeyMsg(msg tea.KeyMsg) (*View, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if v.selected > 0 {
			v.selected--
		}
	case "down", "j":
		if v.selected < len(v.sources)-1 {
			v.selected++
		}
	case "enter":
		// Navigate to source detail
		if len(v.sources) > 0 && v.selected < len(v.sources) {
			source := v.sources[v.selected]
			return v, func() tea.Msg {
				return messages.SourceSelected{Source: source}
			}
		}
	case "a":
		// Add new source
		return v, func() tea.Msg {
			return messages.ViewChanged{View: messages.ViewAddSource}
		}
	case "d", "delete", "backspace":
		// Delete selected source
		if len(v.sources) > 0 && v.selected < len(v.sources) {
			cmd := v.deleteSource(v.sources[v.selected].ID)
			return v, cmd
		}
	case "r":
		// Reload sources
		v.loading = true
		cmd := v.loadSources()
		return v, cmd
	}

	return v, nil
}

// deleteSource returns a command that deletes a source.
func (v *View) deleteSource(id string) tea.Cmd {
	return func() tea.Msg {
		if v.sourceService == nil {
			return messages.SourceRemoved{ID: id, Err: fmt.Errorf("source service not available")}
		}

		err := v.sourceService.Remove(context.Background(), id)
		return messages.SourceRemoved{ID: id, Err: err}
	}
}

// View renders the sources view.
func (v *View) View() string {
	var b strings.Builder

	// Title
	b.WriteString(v.styles.Title.Render("Sources"))
	b.WriteString("\n\n")

	// Loading state
	if v.loading {
		b.WriteString(v.styles.Muted.Render("Loading sources..."))
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
	if len(v.sources) == 0 {
		b.WriteString(v.styles.Muted.Render("No sources configured."))
		b.WriteString("\n\n")
		b.WriteString(v.renderHelp())
		return b.String()
	}

	// Sources list
	for i := range v.sources {
		line := v.renderSource(i, &v.sources[i])
		b.WriteString(line)
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(v.renderHelp())

	return b.String()
}

// renderSource renders a single source line.
func (v *View) renderSource(index int, source *domain.Source) string {
	indicator := "  "
	if index == v.selected {
		indicator = "> "
	}

	// Format: > [type] name (account)
	typeStr := fmt.Sprintf("[%s]", source.Type)
	name := source.Name
	if name == "" {
		name = source.ID
	}

	// Append account identifier if available
	if accountID, ok := v.accountIdentifiers[source.ID]; ok && accountID != "" {
		name = fmt.Sprintf("%s - %s", name, accountID)
	}

	// Truncate name if needed
	maxNameLen := v.width - len(typeStr) - 12
	if maxNameLen < 10 {
		maxNameLen = 10
	}
	if len(name) > maxNameLen {
		name = name[:maxNameLen-3] + "..."
	}

	var line string
	if index == v.selected {
		line = v.styles.Selected.Render(fmt.Sprintf("%s%-10s %s", indicator, typeStr, name))
	} else {
		line = v.styles.Normal.Render(indicator) +
			v.styles.Subtitle.Render(fmt.Sprintf("%-10s ", typeStr)) +
			v.styles.Normal.Render(name)
	}

	return line
}

// renderHelp renders the help footer.
func (v *View) renderHelp() string {
	return v.styles.Help.Render("[a] add  [enter] details  [d] delete  [r] reload  [esc] back  [q] quit")
}

// SetDimensions sets the view dimensions.
func (v *View) SetDimensions(width, height int) {
	v.width = width
	v.height = height
	v.ready = true
}

// Sources returns the current list of sources.
func (v *View) Sources() []domain.Source {
	return v.sources
}

// SelectedIndex returns the currently selected source index.
func (v *View) SelectedIndex() int {
	return v.selected
}

// Err returns the last error.
func (v *View) Err() error {
	return v.err
}
