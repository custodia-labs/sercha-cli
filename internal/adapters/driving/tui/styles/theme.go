// Package styles provides colour themes and styling for the TUI.
package styles

import (
	"github.com/charmbracelet/lipgloss"
)

// Theme defines the colour palette and styling for the TUI.
type Theme struct {
	// Primary is the main accent colour.
	Primary lipgloss.Color

	// Secondary is the secondary accent colour.
	Secondary lipgloss.Color

	// Background is the background colour.
	Background lipgloss.Color

	// Foreground is the default text colour.
	Foreground lipgloss.Color

	// Muted is for less important text.
	Muted lipgloss.Color

	// Success indicates positive outcomes.
	Success lipgloss.Color

	// Warning indicates caution.
	Warning lipgloss.Color

	// Error indicates problems.
	Error lipgloss.Color

	// Border is the border colour.
	Border lipgloss.Color
}

// DefaultTheme returns the default colour theme.
func DefaultTheme() *Theme {
	return &Theme{
		Primary:    lipgloss.Color("#7C3AED"), // Purple
		Secondary:  lipgloss.Color("#06B6D4"), // Cyan
		Background: lipgloss.Color("#1E1E2E"), // Dark gray
		Foreground: lipgloss.Color("#CDD6F4"), // Light gray
		Muted:      lipgloss.Color("#6C7086"), // Medium gray
		Success:    lipgloss.Color("#A6E3A1"), // Green
		Warning:    lipgloss.Color("#F9E2AF"), // Yellow
		Error:      lipgloss.Color("#F38BA8"), // Red
		Border:     lipgloss.Color("#45475A"), // Border gray
	}
}

// Styles contains pre-configured lipgloss styles.
type Styles struct {
	theme *Theme

	// Title style for headers.
	Title lipgloss.Style

	// Subtitle style for secondary headers.
	Subtitle lipgloss.Style

	// Normal style for regular text.
	Normal lipgloss.Style

	// Muted style for less important text.
	Muted lipgloss.Style

	// Selected style for highlighted items.
	Selected lipgloss.Style

	// Error style for error messages.
	Error lipgloss.Style

	// Success style for success messages.
	Success lipgloss.Style

	// Warning style for warning messages.
	Warning lipgloss.Style

	// InputField style for input areas.
	InputField lipgloss.Style

	// StatusBar style for the status bar.
	StatusBar lipgloss.Style

	// Help style for help text.
	Help lipgloss.Style

	// Border style for bordered containers.
	Border lipgloss.Style
}

// NewStyles creates styles from a theme.
func NewStyles(theme *Theme) *Styles {
	if theme == nil {
		theme = DefaultTheme()
	}

	return &Styles{
		theme: theme,

		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(theme.Primary),

		Subtitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(theme.Secondary),

		Normal: lipgloss.NewStyle().
			Foreground(theme.Foreground),

		Muted: lipgloss.NewStyle().
			Foreground(theme.Muted),

		Selected: lipgloss.NewStyle().
			Bold(true).
			Foreground(theme.Foreground).
			Background(theme.Primary),

		Error: lipgloss.NewStyle().
			Foreground(theme.Error),

		Success: lipgloss.NewStyle().
			Foreground(theme.Success),

		Warning: lipgloss.NewStyle().
			Foreground(theme.Warning),

		InputField: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(theme.Border).
			Padding(0, 1),

		StatusBar: lipgloss.NewStyle().
			Foreground(theme.Muted).
			Background(lipgloss.Color("#181825")).
			Padding(0, 1),

		Help: lipgloss.NewStyle().
			Foreground(theme.Muted),

		Border: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(theme.Border),
	}
}

// DefaultStyles returns styles with the default theme.
func DefaultStyles() *Styles {
	return NewStyles(DefaultTheme())
}

// Theme returns the theme used by these styles.
func (s *Styles) Theme() *Theme {
	return s.theme
}
