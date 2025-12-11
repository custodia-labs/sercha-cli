package cli

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSourceCmd_Use(t *testing.T) {
	assert.Equal(t, "source", sourceCmd.Use)
}

func TestSourceCmd_Short(t *testing.T) {
	assert.Equal(t, "Manage document sources", sourceCmd.Short)
}

func TestSourceCmd_HasSubcommands(t *testing.T) {
	commands := sourceCmd.Commands()
	commandNames := make([]string, 0, len(commands))
	for _, cmd := range commands {
		commandNames = append(commandNames, cmd.Name())
	}

	assert.Contains(t, commandNames, "add")
	assert.Contains(t, commandNames, "list")
	assert.Contains(t, commandNames, "remove")
}

// Source Add Tests

func TestSourceAddCmd_Use(t *testing.T) {
	assert.Equal(t, "add [connector-type]", sourceAddCmd.Use)
}

func TestSourceAddCmd_AcceptsMaxOneArg(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"source", "add", "filesystem", "extra-arg"})
	defer func() {
		rootCmd.SetArgs(nil)
	}()

	err := rootCmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "accepts at most 1 arg(s)")
}

func TestSourceAddCmd_ErrorsWithoutServices(t *testing.T) {
	// Reset services to nil
	oldSourceService := sourceService
	oldConnectorRegistry := connectorRegistry
	sourceService = nil
	connectorRegistry = nil
	defer func() {
		sourceService = oldSourceService
		connectorRegistry = oldConnectorRegistry
	}()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"source", "add", "filesystem"})
	defer func() {
		rootCmd.SetArgs(nil)
	}()

	err := rootCmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not configured")
}

// Source List Tests

func TestSourceListCmd_Use(t *testing.T) {
	assert.Equal(t, "list", sourceListCmd.Use)
}

func TestSourceListCmd_Executes(t *testing.T) {
	cleanup := setupTestServices()
	defer cleanup()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"source", "list"})
	defer func() {
		rootCmd.SetArgs(nil)
	}()

	err := rootCmd.Execute()

	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "Configured sources:")
}

// Source Remove Tests

func TestSourceRemoveCmd_Use(t *testing.T) {
	assert.Equal(t, "remove [source-id]", sourceRemoveCmd.Use)
}

func TestSourceRemoveCmd_RequiresExactlyOneArg(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"source", "remove"})
	defer func() {
		rootCmd.SetArgs(nil)
	}()

	err := rootCmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 1 arg(s)")
}

func TestSourceRemoveCmd_ExecutesWithArg(t *testing.T) {
	cleanup := setupTestServices()
	defer cleanup()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"source", "remove", "source-123"})
	defer func() {
		rootCmd.SetArgs(nil)
	}()

	err := rootCmd.Execute()

	assert.NoError(t, err)
	// New behaviour: returns "Removed source: source-123"
	assert.Contains(t, buf.String(), "Removed source:")
}

// Connector List Tests

func TestConnectorCmd_Use(t *testing.T) {
	assert.Equal(t, "connector", connectorCmd.Use)
}

func TestConnectorListCmd_Use(t *testing.T) {
	assert.Equal(t, "list", connectorListCmd.Use)
}

func TestConnectorListCmd_Executes(t *testing.T) {
	oldRegistry := connectorRegistry
	connectorRegistry = &mockConnectorRegistry{}
	defer func() {
		connectorRegistry = oldRegistry
	}()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"connector", "list"})
	defer func() {
		rootCmd.SetArgs(nil)
	}()

	err := rootCmd.Execute()

	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "Available connectors:")
	assert.Contains(t, buf.String(), "filesystem")
	assert.Contains(t, buf.String(), "github")
	assert.Contains(t, buf.String(), "Local Filesystem")
	assert.Contains(t, buf.String(), "Config:")
	assert.Contains(t, buf.String(), "--path")
}

func TestConnectorListCmd_EmptyList(t *testing.T) {
	oldRegistry := connectorRegistry
	connectorRegistry = &mockConnectorRegistryEmpty{}
	defer func() {
		connectorRegistry = oldRegistry
	}()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"connector", "list"})
	defer func() {
		rootCmd.SetArgs(nil)
	}()

	err := rootCmd.Execute()

	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "No connectors available")
}

func TestConnectorListCmd_ServiceNotConfigured(t *testing.T) {
	oldRegistry := connectorRegistry
	connectorRegistry = nil
	defer func() {
		connectorRegistry = oldRegistry
	}()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"connector", "list"})
	defer func() {
		rootCmd.SetArgs(nil)
	}()

	err := rootCmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "connector registry not configured")
}

// Source List Empty Tests

func TestSourceListCmd_EmptyList(t *testing.T) {
	oldService := sourceService
	sourceService = &mockSourceServiceEmpty{}
	defer func() {
		sourceService = oldService
	}()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"source", "list"})
	defer func() {
		rootCmd.SetArgs(nil)
	}()

	err := rootCmd.Execute()

	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "No configured sources")
}

func TestSourceListCmd_ServiceNotConfigured(t *testing.T) {
	oldService := sourceService
	sourceService = nil
	defer func() {
		sourceService = oldService
	}()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"source", "list"})
	defer func() {
		rootCmd.SetArgs(nil)
	}()

	err := rootCmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "source service not configured")
}

func TestSourceRemoveCmd_ServiceNotConfigured(t *testing.T) {
	oldService := sourceService
	sourceService = nil
	defer func() {
		sourceService = oldService
	}()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"source", "remove", "src-1"})
	defer func() {
		rootCmd.SetArgs(nil)
	}()

	err := rootCmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "source service not configured")
}

// Service Error Tests

func TestSourceListCmd_ServiceError(t *testing.T) {
	oldService := sourceService
	sourceService = &mockSourceServiceError{}
	defer func() {
		sourceService = oldService
	}()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"source", "list"})
	defer func() {
		rootCmd.SetArgs(nil)
	}()

	err := rootCmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list sources")
}

func TestSourceRemoveCmd_ServiceError(t *testing.T) {
	oldService := sourceService
	sourceService = &mockSourceServiceError{}
	defer func() {
		sourceService = oldService
	}()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"source", "remove", "src-1"})
	defer func() {
		rootCmd.SetArgs(nil)
	}()

	err := rootCmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to remove source")
}
