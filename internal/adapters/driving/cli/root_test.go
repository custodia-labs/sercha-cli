package cli

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetVersion(t *testing.T) {
	// Given
	originalVersion := version
	defer func() { version = originalVersion }()

	// When
	SetVersion("1.2.3")

	// Then
	assert.Equal(t, "1.2.3", version)
}

func TestRootCmd_Use(t *testing.T) {
	assert.Equal(t, "sercha", rootCmd.Use)
}

func TestRootCmd_Short(t *testing.T) {
	assert.Equal(t, "Local-first semantic search for your documents", rootCmd.Short)
}

func TestRootCmd_Long(t *testing.T) {
	assert.Contains(t, rootCmd.Long, "local-first semantic search engine")
	assert.Contains(t, rootCmd.Long, "All data stays on your machine")
}

func TestRootCmd_HasSubcommands(t *testing.T) {
	commands := rootCmd.Commands()

	// Verify expected subcommands exist
	commandNames := make([]string, 0, len(commands))
	for _, cmd := range commands {
		commandNames = append(commandNames, cmd.Name())
	}

	assert.Contains(t, commandNames, "search", "should have search command")
	assert.Contains(t, commandNames, "source", "should have source command")
	assert.Contains(t, commandNames, "sync", "should have sync command")
	assert.Contains(t, commandNames, "version", "should have version command")
}

func TestExecute_ReturnsNoErrorWithHelp(t *testing.T) {
	// Save and restore stdout
	oldOut := rootCmd.OutOrStdout()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"--help"})
	defer func() {
		rootCmd.SetOut(oldOut)
		rootCmd.SetArgs(nil)
	}()

	// When
	err := Execute()

	// Then
	assert.NoError(t, err)
}

func TestSetServices_WithNilServices(t *testing.T) {
	// Save current state
	oldSearch := searchService
	oldSource := sourceService
	oldSync := syncOrchestrator
	oldDocument := documentService
	oldConnector := connectorRegistry
	oldProvider := providerRegistry
	oldSettings := settingsService

	defer func() {
		// Restore state
		searchService = oldSearch
		sourceService = oldSource
		syncOrchestrator = oldSync
		documentService = oldDocument
		connectorRegistry = oldConnector
		providerRegistry = oldProvider
		settingsService = oldSettings
	}()

	// Set some values first
	searchService = &mockSearchService{}
	sourceService = &mockSourceService{}

	// Call with nil should not panic and should not change values
	SetServices(nil)

	// Services should remain unchanged
	assert.NotNil(t, searchService)
	assert.NotNil(t, sourceService)
}

func TestSetServices_WithValidServices(t *testing.T) {
	// Save current state
	oldSearch := searchService
	oldSource := sourceService
	oldSync := syncOrchestrator
	oldDocument := documentService

	defer func() {
		// Restore state
		searchService = oldSearch
		sourceService = oldSource
		syncOrchestrator = oldSync
		documentService = oldDocument
	}()

	// Clear services
	searchService = nil
	sourceService = nil
	syncOrchestrator = nil
	documentService = nil

	// Create mock services
	mockSearch := &mockSearchService{}
	mockSource := &mockSourceService{}
	mockSync := &mockSyncOrchestratorFull{}
	mockDoc := &mockDocumentService{}

	services := &Services{
		Search:   mockSearch,
		Source:   mockSource,
		Sync:     mockSync,
		Document: mockDoc,
	}

	// Set services
	SetServices(services)

	// Verify services were set
	assert.NotNil(t, searchService)
	assert.NotNil(t, sourceService)
	assert.NotNil(t, syncOrchestrator)
	assert.NotNil(t, documentService)
}
