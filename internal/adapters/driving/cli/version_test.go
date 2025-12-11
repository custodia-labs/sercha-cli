package cli

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersionCmd_Use(t *testing.T) {
	assert.Equal(t, "version", versionCmd.Use)
}

func TestVersionCmd_Short(t *testing.T) {
	assert.Equal(t, "Print the version number", versionCmd.Short)
}

func TestVersionCmd_Executes(t *testing.T) {
	// Save and restore version
	originalVersion := version
	version = "test-version-1.0.0"
	defer func() { version = originalVersion }()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"version"})
	defer func() {
		rootCmd.SetArgs(nil)
	}()

	err := rootCmd.Execute()

	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "sercha version test-version-1.0.0")
}

func TestVersionCmd_DisplaysDevByDefault(t *testing.T) {
	// Save and restore version
	originalVersion := version
	version = "dev"
	defer func() { version = originalVersion }()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"version"})
	defer func() {
		rootCmd.SetArgs(nil)
	}()

	err := rootCmd.Execute()

	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "sercha version dev")
}
