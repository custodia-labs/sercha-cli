package tui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/custodia-labs/sercha-cli/internal/adapters/driving/tui/messages"
	"github.com/custodia-labs/sercha-cli/internal/adapters/driving/tui/styles"
	"github.com/custodia-labs/sercha-cli/internal/adapters/driving/tui/views/addsource"
	"github.com/custodia-labs/sercha-cli/internal/adapters/driving/tui/views/doccontent"
	"github.com/custodia-labs/sercha-cli/internal/adapters/driving/tui/views/docdetails"
	"github.com/custodia-labs/sercha-cli/internal/adapters/driving/tui/views/documents"
	"github.com/custodia-labs/sercha-cli/internal/adapters/driving/tui/views/menu"
	"github.com/custodia-labs/sercha-cli/internal/adapters/driving/tui/views/search"
	"github.com/custodia-labs/sercha-cli/internal/adapters/driving/tui/views/settings"
	"github.com/custodia-labs/sercha-cli/internal/adapters/driving/tui/views/sourcedetail"
	"github.com/custodia-labs/sercha-cli/internal/adapters/driving/tui/views/sources"
	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driving"
)

// App is the main TUI application following the Elm architecture.
// It implements tea.Model for use with Bubbletea.
type App struct {
	// ports provides access to core services via driving ports.
	ports *Ports

	// ctx is the context for cancellation.
	ctx context.Context

	// styles holds the TUI styles.
	styles *styles.Styles

	// menuView is the main navigation menu.
	menuView *menu.View

	// searchView is the styled search view component.
	searchView *search.View

	// sourcesView is the sources management view component.
	sourcesView *sources.View

	// sourceDetailView is the source detail view component.
	sourceDetailView *sourcedetail.View

	// documentsView is the documents list view component.
	documentsView *documents.View

	// docContentView is the document content view component.
	docContentView *doccontent.View

	// docDetailsView is the document details view component.
	docDetailsView *docdetails.View

	// addSourceView is the add source wizard view component.
	addSourceView *addsource.View

	// settingsView is the settings configuration view component.
	settingsView *settings.View

	// selectedSource tracks the currently selected source for navigation.
	selectedSource *domain.Source

	// selectedDocument tracks the currently selected document for navigation.
	selectedDocument *domain.Document

	// currentView tracks which view is active.
	currentView messages.ViewType

	// query is the current search query (kept for accessor compatibility).
	query string

	// results holds the current search results (kept for accessor compatibility).
	results []domain.SearchResult

	// selectedIndex is the currently selected result (kept for accessor compatibility).
	selectedIndex int

	// err holds the last error that occurred.
	err error

	// width and height are terminal dimensions.
	width  int
	height int

	// ready indicates if the app has initialised.
	ready bool
}

// Ensure App implements tea.Model.
var _ tea.Model = (*App)(nil)

// NewApp creates a new TUI application with the given ports.
func NewApp(ports *Ports) (*App, error) {
	if err := ports.Validate(); err != nil {
		return nil, fmt.Errorf("creating app: %w", err)
	}

	s := styles.DefaultStyles()
	menuView := menu.NewView(s)
	searchView := search.NewView(s, nil, ports.Search, ports.ResultAction)
	sourcesView := sources.NewView(s, ports.Source, ports.Credentials)
	sourceDetailView := sourcedetail.NewView(s, ports.Source, ports.Sync, ports.Document)
	documentsView := documents.NewView(s, ports.Document)
	docContentView := doccontent.NewView(s, ports.Document)
	docDetailsView := docdetails.NewView(s)
	addSourceView := addsource.NewView(
		s, ports.Source, ports.ConnectorRegistry, ports.ProviderRegistry,
		ports.AuthProvider, ports.Credentials,
	)
	settingsView := settings.NewView(s, ports.Settings)

	return &App{
		ports:            ports,
		ctx:              context.Background(),
		styles:           s,
		menuView:         menuView,
		searchView:       searchView,
		sourcesView:      sourcesView,
		sourceDetailView: sourceDetailView,
		documentsView:    documentsView,
		docContentView:   docContentView,
		docDetailsView:   docDetailsView,
		addSourceView:    addSourceView,
		settingsView:     settingsView,
		currentView:      messages.ViewMenu, // Start with menu
	}, nil
}

// WithContext sets the context for the app.
func (a *App) WithContext(ctx context.Context) *App {
	a.ctx = ctx
	return a
}

// Init implements tea.Model.
// It runs initial commands when the program starts.
func (a *App) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		tea.SetWindowTitle("sercha - Local Search"),
	)
}

// Update implements tea.Model.
// It handles messages and updates the model state.
//
//nolint:gocognit,gocyclo,funlen // central message handler requires complexity
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.ready = true
		// Forward to all views for proper sizing
		a.menuView.SetDimensions(msg.Width, msg.Height)
		a.searchView.SetDimensions(msg.Width, msg.Height)
		a.sourcesView.SetDimensions(msg.Width, msg.Height)
		a.sourceDetailView.SetDimensions(msg.Width, msg.Height)
		a.documentsView.SetDimensions(msg.Width, msg.Height)
		a.docContentView.SetDimensions(msg.Width, msg.Height)
		a.docDetailsView.SetDimensions(msg.Width, msg.Height)
		a.addSourceView.SetDimensions(msg.Width, msg.Height)
		a.settingsView.SetDimensions(msg.Width, msg.Height)
		return a, nil

	case tea.KeyMsg:
		// Global quit with ctrl+c
		if msg.String() == "ctrl+c" {
			return a, tea.Quit
		}

		// Forward key messages to active view
		switch a.currentView {
		case messages.ViewMenu:
			a.menuView, cmd = a.menuView.Update(msg)
			return a, cmd

		case messages.ViewSearch:
			a.searchView, cmd = a.searchView.Update(msg)
			// Sync state from searchView for accessor compatibility
			a.query = a.searchView.Query()
			a.results = a.searchView.Results()
			a.selectedIndex = a.searchView.SelectedIndex()
			a.err = a.searchView.Err()
			return a, cmd

		case messages.ViewSources:
			// Esc from sources goes to menu
			if msg.Type == tea.KeyEsc {
				a.currentView = messages.ViewMenu
				return a, nil
			}
			a.sourcesView, cmd = a.sourcesView.Update(msg)
			return a, cmd

		case messages.ViewSourceDetail:
			a.sourceDetailView, cmd = a.sourceDetailView.Update(msg)
			return a, cmd

		case messages.ViewDocuments:
			a.documentsView, cmd = a.documentsView.Update(msg)
			return a, cmd

		case messages.ViewDocContent:
			a.docContentView, cmd = a.docContentView.Update(msg)
			return a, cmd

		case messages.ViewDocDetails:
			a.docDetailsView, cmd = a.docDetailsView.Update(msg)
			return a, cmd

		case messages.ViewHelp:
			// Esc from help goes to menu
			if msg.Type == tea.KeyEsc {
				a.currentView = messages.ViewMenu
				return a, nil
			}
			return a, nil

		case messages.ViewAddSource:
			a.addSourceView, cmd = a.addSourceView.Update(msg)
			return a, cmd

		case messages.ViewSettings:
			a.settingsView, cmd = a.settingsView.Update(msg)
			return a, cmd
		}
		return a, nil

	case messages.SearchCompleted:
		// Forward to searchView
		a.searchView, cmd = a.searchView.Update(msg)
		// Sync state
		a.results = a.searchView.Results()
		a.err = a.searchView.Err()
		a.selectedIndex = 0
		return a, cmd

	case messages.ViewChanged:
		a.currentView = msg.View
		// Initialise views when switching to them
		switch msg.View {
		case messages.ViewSearch:
			a.searchView.Reset()
			return a, a.searchView.Init()
		case messages.ViewSources:
			return a, a.sourcesView.Init()
		case messages.ViewSourceDetail:
			return a, a.sourceDetailView.Init()
		case messages.ViewAddSource:
			a.addSourceView.Reset()
			return a, a.addSourceView.Init()
		case messages.ViewSettings:
			a.settingsView.Reset()
			return a, a.settingsView.Init()
		case messages.ViewMenu, messages.ViewHelp,
			messages.ViewDocuments, messages.ViewDocContent, messages.ViewDocDetails:
			// Other views don't need special initialisation
		}
		return a, nil

	case messages.SourceSelected:
		// Navigate from sources to source detail
		a.selectedSource = &msg.Source
		a.sourceDetailView.SetSource(msg.Source)
		// Check if coming from source detail (View Documents)
		if a.currentView == messages.ViewSourceDetail {
			// Go to documents list
			a.currentView = messages.ViewDocuments
			return a, a.documentsView.SetSource(msg.Source)
		}
		// Coming from sources list
		a.currentView = messages.ViewSourceDetail
		return a, a.sourceDetailView.Init()

	case messages.DocumentsLoaded:
		a.documentsView, cmd = a.documentsView.Update(msg)
		return a, cmd

	case messages.DocumentSelected:
		// Navigate to document content
		a.selectedDocument = &msg.Document
		a.currentView = messages.ViewDocContent
		return a, a.docContentView.SetDocument(&msg.Document)

	case messages.DocumentContentLoaded:
		a.docContentView, cmd = a.docContentView.Update(msg)
		return a, cmd

	case messages.DocumentDetailsLoaded:
		if msg.Err != nil {
			a.err = msg.Err
		} else if details, ok := msg.Details.(*driving.DocumentDetails); ok {
			a.docDetailsView.SetDetails(details)
			a.currentView = messages.ViewDocDetails
		}
		return a, nil

	case messages.DocumentExcluded:
		a.documentsView, cmd = a.documentsView.Update(msg)
		return a, cmd

	case messages.DocumentRefreshed:
		a.documentsView, cmd = a.documentsView.Update(msg)
		return a, cmd

	case messages.ErrorOccurred:
		a.err = msg.Err
		// Forward to current view
		switch a.currentView {
		case messages.ViewSearch:
			a.searchView, cmd = a.searchView.Update(msg)
		case messages.ViewDocuments:
			a.documentsView, cmd = a.documentsView.Update(msg)
		case messages.ViewDocContent:
			a.docContentView, cmd = a.docContentView.Update(msg)
		case messages.ViewDocDetails:
			a.docDetailsView, cmd = a.docDetailsView.Update(msg)
		case messages.ViewAddSource:
			a.addSourceView, cmd = a.addSourceView.Update(msg)
		case messages.ViewMenu, messages.ViewSources, messages.ViewHelp,
			messages.ViewSourceDetail, messages.ViewSettings:
			// Other views don't handle error messages
		}
		return a, cmd

	case messages.Quit:
		return a, tea.Quit

	case messages.SourcesLoaded, messages.SourceRemoved:
		// Forward to relevant view
		if a.currentView == messages.ViewSources {
			a.sourcesView, cmd = a.sourcesView.Update(msg)
			return a, cmd
		}
		if a.currentView == messages.ViewSourceDetail {
			a.sourceDetailView, cmd = a.sourceDetailView.Update(msg)
			return a, cmd
		}

	case messages.SourceAdded:
		// Forward to add source view
		if a.currentView == messages.ViewAddSource {
			a.addSourceView, cmd = a.addSourceView.Update(msg)
			return a, cmd
		}

	case messages.SettingsLoaded, messages.SettingsSaved:
		// Forward to settings view
		if a.currentView == messages.ViewSettings {
			a.settingsView, cmd = a.settingsView.Update(msg)
			return a, cmd
		}
	}

	// Forward other messages to active view
	switch a.currentView {
	case messages.ViewMenu:
		a.menuView, cmd = a.menuView.Update(msg)
	case messages.ViewSearch:
		a.searchView, cmd = a.searchView.Update(msg)
	case messages.ViewSources:
		a.sourcesView, cmd = a.sourcesView.Update(msg)
	case messages.ViewSourceDetail:
		a.sourceDetailView, cmd = a.sourceDetailView.Update(msg)
	case messages.ViewDocuments:
		a.documentsView, cmd = a.documentsView.Update(msg)
	case messages.ViewDocContent:
		a.docContentView, cmd = a.docContentView.Update(msg)
	case messages.ViewDocDetails:
		a.docDetailsView, cmd = a.docDetailsView.Update(msg)
	case messages.ViewAddSource:
		a.addSourceView, cmd = a.addSourceView.Update(msg)
	case messages.ViewSettings:
		a.settingsView, cmd = a.settingsView.Update(msg)
	case messages.ViewHelp:
		// Help view doesn't need to handle other messages
	}

	return a, cmd
}

// View implements tea.Model.
// It renders the current view as a string.
func (a *App) View() string {
	if !a.ready {
		return "Initialising..."
	}

	switch a.currentView {
	case messages.ViewMenu:
		return a.menuView.View()
	case messages.ViewSearch:
		return a.viewSearch()
	case messages.ViewSources:
		return a.viewSources()
	case messages.ViewSourceDetail:
		return a.sourceDetailView.View()
	case messages.ViewDocuments:
		return a.documentsView.View()
	case messages.ViewDocContent:
		return a.docContentView.View()
	case messages.ViewDocDetails:
		return a.docDetailsView.View()
	case messages.ViewAddSource:
		return a.addSourceView.View()
	case messages.ViewSettings:
		return a.settingsView.View()
	case messages.ViewHelp:
		return a.viewHelp()
	default:
		return a.menuView.View()
	}
}

// viewSearch renders the search view using the styled searchView component.
func (a *App) viewSearch() string {
	return a.searchView.View()
}

// viewSources renders the sources view.
func (a *App) viewSources() string {
	return a.sourcesView.View()
}

// viewHelp renders the help view.
func (a *App) viewHelp() string {
	return `Help

Navigation:
  esc         Back to Menu
  ctrl+c      Quit

Menu:
  j/k, ↑/↓    Navigate options
  enter       Select option
  q           Quit

Search:
  (type)      Enter search query
  enter       Submit search
  esc         Back to Menu

Results:
  j/k, ↑/↓    Navigate results
  esc         Back to Menu

[esc] back to menu`
}

// Run starts the TUI application.
func (a *App) Run() error {
	p := tea.NewProgram(a, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// Query returns the current search query.
func (a *App) Query() string {
	return a.query
}

// Results returns the current search results.
func (a *App) Results() []domain.SearchResult {
	return a.results
}

// SelectedIndex returns the currently selected result index.
func (a *App) SelectedIndex() int {
	return a.selectedIndex
}

// CurrentView returns the current view type.
func (a *App) CurrentView() messages.ViewType {
	return a.currentView
}

// Err returns the last error that occurred.
func (a *App) Err() error {
	return a.err
}

// Ready returns whether the app has been initialised.
func (a *App) Ready() bool {
	return a.ready
}

// SetDimensions sets the terminal dimensions (for testing).
func (a *App) SetDimensions(width, height int) {
	a.width = width
	a.height = height
	a.ready = true
	// Also set searchView dimensions so it renders properly
	a.searchView.SetDimensions(width, height)
}
