package keymap

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultKeyMap(t *testing.T) {
	km := DefaultKeyMap()

	require.NotNil(t, km)
}

func TestDefaultKeyMap_QuitBinding(t *testing.T) {
	km := DefaultKeyMap()

	keys := km.Quit.Keys()
	assert.Contains(t, keys, "q")
	assert.Contains(t, keys, "ctrl+c")
}

func TestDefaultKeyMap_HelpBinding(t *testing.T) {
	km := DefaultKeyMap()

	keys := km.Help.Keys()
	assert.Contains(t, keys, "?")
}

func TestDefaultKeyMap_BackBinding(t *testing.T) {
	km := DefaultKeyMap()

	keys := km.Back.Keys()
	assert.Contains(t, keys, "esc")
}

func TestDefaultKeyMap_SearchBinding(t *testing.T) {
	km := DefaultKeyMap()

	keys := km.Search.Keys()
	assert.Contains(t, keys, "enter")
}

func TestDefaultKeyMap_UpBinding(t *testing.T) {
	km := DefaultKeyMap()

	keys := km.Up.Keys()
	assert.Contains(t, keys, "up")
	assert.Contains(t, keys, "k")
}

func TestDefaultKeyMap_DownBinding(t *testing.T) {
	km := DefaultKeyMap()

	keys := km.Down.Keys()
	assert.Contains(t, keys, "down")
	assert.Contains(t, keys, "j")
}

func TestDefaultKeyMap_SelectBinding(t *testing.T) {
	km := DefaultKeyMap()

	keys := km.Select.Keys()
	assert.Contains(t, keys, "enter")
}

func TestDefaultKeyMap_CancelBinding(t *testing.T) {
	km := DefaultKeyMap()

	keys := km.Cancel.Keys()
	assert.Contains(t, keys, "esc")
}

func TestShortHelp(t *testing.T) {
	km := DefaultKeyMap()

	bindings := km.ShortHelp()

	assert.Len(t, bindings, 2)
	assert.Equal(t, km.Quit, bindings[0])
	assert.Equal(t, km.Help, bindings[1])
}

func TestFullHelp(t *testing.T) {
	km := DefaultKeyMap()

	bindings := km.FullHelp()

	assert.Len(t, bindings, 3)    // 3 groups
	assert.Len(t, bindings[0], 3) // Up, Down, Select
	assert.Len(t, bindings[1], 3) // Search, Back, Cancel
	assert.Len(t, bindings[2], 2) // Help, Quit
}

func TestMatches_True(t *testing.T) {
	km := DefaultKeyMap()

	assert.True(t, Matches("q", km.Quit))
	assert.True(t, Matches("ctrl+c", km.Quit))
	assert.True(t, Matches("?", km.Help))
	assert.True(t, Matches("up", km.Up))
	assert.True(t, Matches("k", km.Up))
}

func TestMatches_False(t *testing.T) {
	km := DefaultKeyMap()

	assert.False(t, Matches("x", km.Quit))
	assert.False(t, Matches("a", km.Help))
	assert.False(t, Matches("down", km.Up))
}

func TestBindings_HaveHelp(t *testing.T) {
	km := DefaultKeyMap()

	testCases := []struct {
		name    string
		binding key.Binding
	}{
		{"Quit", km.Quit},
		{"Help", km.Help},
		{"Back", km.Back},
		{"Search", km.Search},
		{"Up", km.Up},
		{"Down", km.Down},
		{"Select", km.Select},
		{"Cancel", km.Cancel},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			help := tc.binding.Help()
			assert.NotEmpty(t, help.Key, "binding should have help key")
		})
	}
}
