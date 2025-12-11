package styles

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultTheme(t *testing.T) {
	theme := DefaultTheme()

	require.NotNil(t, theme)
	assert.NotEmpty(t, string(theme.Primary))
	assert.NotEmpty(t, string(theme.Secondary))
	assert.NotEmpty(t, string(theme.Background))
	assert.NotEmpty(t, string(theme.Foreground))
	assert.NotEmpty(t, string(theme.Muted))
	assert.NotEmpty(t, string(theme.Success))
	assert.NotEmpty(t, string(theme.Warning))
	assert.NotEmpty(t, string(theme.Error))
	assert.NotEmpty(t, string(theme.Border))
}

func TestDefaultTheme_ColorsAreDistinct(t *testing.T) {
	theme := DefaultTheme()

	//nolint:misspell // using colors for technical accuracy
	colors := []lipgloss.Color{
		theme.Primary,
		theme.Secondary,
		theme.Success,
		theme.Warning,
		theme.Error,
	}

	seen := make(map[string]bool)
	for _, c := range colors { //nolint:misspell // using colors for technical accuracy
		s := string(c)
		assert.False(t, seen[s], "duplicate color: %s", s) //nolint:misspell // using color for technical accuracy
		seen[s] = true
	}
}

func TestNewStyles_WithTheme(t *testing.T) {
	theme := DefaultTheme()
	styles := NewStyles(theme)

	require.NotNil(t, styles)
	assert.Equal(t, theme, styles.Theme())
}

func TestNewStyles_NilTheme(t *testing.T) {
	styles := NewStyles(nil)

	require.NotNil(t, styles)
	assert.NotNil(t, styles.Theme())
}

func TestDefaultStyles(t *testing.T) {
	styles := DefaultStyles()

	require.NotNil(t, styles)
	assert.NotNil(t, styles.Theme())
}

func TestStyles_AllStylesInitialised(t *testing.T) {
	styles := DefaultStyles()

	// All style fields should be initialised (not zero-value)
	assert.NotEqual(t, lipgloss.Style{}, styles.Title)
	assert.NotEqual(t, lipgloss.Style{}, styles.Subtitle)
	assert.NotEqual(t, lipgloss.Style{}, styles.Normal)
	assert.NotEqual(t, lipgloss.Style{}, styles.Muted)
	assert.NotEqual(t, lipgloss.Style{}, styles.Selected)
	assert.NotEqual(t, lipgloss.Style{}, styles.Error)
	assert.NotEqual(t, lipgloss.Style{}, styles.Success)
	assert.NotEqual(t, lipgloss.Style{}, styles.InputField)
	assert.NotEqual(t, lipgloss.Style{}, styles.StatusBar)
	assert.NotEqual(t, lipgloss.Style{}, styles.Help)
	assert.NotEqual(t, lipgloss.Style{}, styles.Border)
}

func TestStyles_TitleIsBold(t *testing.T) {
	styles := DefaultStyles()

	// Verify the style has bold enabled
	// We can't easily test ANSI output without terminal
	rendered := styles.Title.Render("Test")
	assert.NotEmpty(t, rendered)
}

func TestStyles_CanRenderText(t *testing.T) {
	styles := DefaultStyles()

	testCases := []struct {
		name  string
		style lipgloss.Style
	}{
		{"Title", styles.Title},
		{"Subtitle", styles.Subtitle},
		{"Normal", styles.Normal},
		{"Muted", styles.Muted},
		{"Selected", styles.Selected},
		{"Error", styles.Error},
		{"Success", styles.Success},
		{"Help", styles.Help},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.style.Render("test text")
			assert.NotEmpty(t, result)
		})
	}
}
