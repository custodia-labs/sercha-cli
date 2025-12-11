// Package menu provides the main navigation menu view for the TUI.
package menu

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/custodia-labs/sercha-cli/internal/adapters/driving/tui/messages"
	"github.com/custodia-labs/sercha-cli/internal/adapters/driving/tui/styles"
)

// Item represents a single menu option.
type Item struct {
	Label string
	View  messages.ViewType
	Quit  bool // If true, selecting this item quits the app
}

// View represents the main menu view.
type View struct {
	styles   *styles.Styles
	items    []Item
	selected int
	width    int
	height   int
	ready    bool
}

// NewView creates a new menu view.
func NewView(s *styles.Styles) *View {
	if s == nil {
		s = styles.DefaultStyles()
	}

	return &View{
		styles: s,
		items: []Item{
			{Label: "Search", View: messages.ViewSearch},
			{Label: "Sources", View: messages.ViewSources},
			{Label: "Settings", View: messages.ViewSettings},
			{Label: "Help", View: messages.ViewHelp},
			{Label: "Quit", Quit: true},
		},
		selected: 0,
		width:    80,
		height:   24,
	}
}

// Init initialises the menu view.
func (v *View) Init() tea.Cmd {
	return nil
}

// Update handles messages for the menu view.
func (v *View) Update(msg tea.Msg) (*View, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
		v.ready = true
		return v, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if v.selected > 0 {
				v.selected--
			}
			return v, nil

		case "down", "j":
			if v.selected < len(v.items)-1 {
				v.selected++
			}
			return v, nil

		case "enter":
			item := v.items[v.selected]
			if item.Quit {
				return v, tea.Quit
			}
			return v, func() tea.Msg {
				return messages.ViewChanged{View: item.View}
			}

		case "q":
			return v, tea.Quit
		}
	}

	return v, nil
}

// View renders the menu.
func (v *View) View() string {
	if !v.ready {
		return "Initialising..."
	}

	var b strings.Builder

	// Title
	title := v.styles.Title.Render("Sercha")
	b.WriteString(title)
	b.WriteString("\n\n")

	// Subtitle
	subtitle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Render("Local Document Search")
	b.WriteString(subtitle)
	b.WriteString("\n\n")

	// Menu items
	for i, item := range v.items {
		cursor := "  "
		style := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))

		if i == v.selected {
			cursor = "> "
			style = lipgloss.NewStyle().
				Foreground(lipgloss.Color("86")).
				Bold(true)
		}

		line := cursor + style.Render(item.Label)
		b.WriteString(line)
		b.WriteString("\n")
	}

	// Footer with keybindings
	b.WriteString("\n")
	footer := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Render("[j/k] Navigate  [Enter] Select  [q] Quit")
	b.WriteString(footer)

	return b.String()
}

// SetDimensions sets the view dimensions.
func (v *View) SetDimensions(width, height int) {
	v.width = width
	v.height = height
	v.ready = true
}

// Selected returns the currently selected index.
func (v *View) Selected() int {
	return v.selected
}
