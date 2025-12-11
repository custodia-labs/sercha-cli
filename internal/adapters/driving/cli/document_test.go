package cli

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Document Command Tests

func TestDocumentCmd_Use(t *testing.T) {
	assert.Equal(t, "document", documentCmd.Use)
}

func TestDocumentCmd_Short(t *testing.T) {
	assert.Equal(t, "Manage indexed documents", documentCmd.Short)
}

func TestDocumentCmd_HasSubcommands(t *testing.T) {
	commands := documentCmd.Commands()
	commandNames := make([]string, 0, len(commands))
	for _, cmd := range commands {
		commandNames = append(commandNames, cmd.Name())
	}

	assert.Contains(t, commandNames, "list")
	assert.Contains(t, commandNames, "get")
	assert.Contains(t, commandNames, "content")
	assert.Contains(t, commandNames, "details")
	assert.Contains(t, commandNames, "exclude")
	assert.Contains(t, commandNames, "refresh")
	assert.Contains(t, commandNames, "open")
}

// Document List Tests

func TestDocumentListCmd_Use(t *testing.T) {
	assert.Equal(t, "list [source-id]", documentListCmd.Use)
}

func TestDocumentListCmd_RequiresExactlyOneArg(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"document", "list"})
	defer func() {
		rootCmd.SetArgs(nil)
	}()

	err := rootCmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 1 arg(s)")
}

func TestDocumentListCmd_ExecutesWithArg(t *testing.T) {
	cleanup := setupTestServices()
	defer cleanup()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"document", "list", "src-1"})
	defer func() {
		rootCmd.SetArgs(nil)
	}()

	err := rootCmd.Execute()

	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "Documents for source")
	assert.Contains(t, buf.String(), "doc-1")
	assert.Contains(t, buf.String(), "Test Document 1")
}

// Document Get Tests

func TestDocumentGetCmd_Use(t *testing.T) {
	assert.Equal(t, "get [doc-id]", documentGetCmd.Use)
}

func TestDocumentGetCmd_RequiresExactlyOneArg(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"document", "get"})
	defer func() {
		rootCmd.SetArgs(nil)
	}()

	err := rootCmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 1 arg(s)")
}

func TestDocumentGetCmd_ExecutesWithArg(t *testing.T) {
	cleanup := setupTestServices()
	defer cleanup()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"document", "get", "doc-1"})
	defer func() {
		rootCmd.SetArgs(nil)
	}()

	err := rootCmd.Execute()

	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "Document:")
	assert.Contains(t, buf.String(), "doc-1")
	assert.Contains(t, buf.String(), "Title:")
	assert.Contains(t, buf.String(), "Source:")
}

// Document Content Tests

func TestDocumentContentCmd_Use(t *testing.T) {
	assert.Equal(t, "content [doc-id]", documentContentCmd.Use)
}

func TestDocumentContentCmd_RequiresExactlyOneArg(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"document", "content"})
	defer func() {
		rootCmd.SetArgs(nil)
	}()

	err := rootCmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 1 arg(s)")
}

func TestDocumentContentCmd_ExecutesWithArg(t *testing.T) {
	cleanup := setupTestServices()
	defer cleanup()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"document", "content", "doc-1"})
	defer func() {
		rootCmd.SetArgs(nil)
	}()

	err := rootCmd.Execute()

	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "This is the content of the test document.")
}

// Document Details Tests

func TestDocumentDetailsCmd_Use(t *testing.T) {
	assert.Equal(t, "details [doc-id]", documentDetailsCmd.Use)
}

func TestDocumentDetailsCmd_RequiresExactlyOneArg(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"document", "details"})
	defer func() {
		rootCmd.SetArgs(nil)
	}()

	err := rootCmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 1 arg(s)")
}

func TestDocumentDetailsCmd_ExecutesWithArg(t *testing.T) {
	cleanup := setupTestServices()
	defer cleanup()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"document", "details", "doc-1"})
	defer func() {
		rootCmd.SetArgs(nil)
	}()

	err := rootCmd.Execute()

	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "Document Details:")
	assert.Contains(t, buf.String(), "Title:")
	assert.Contains(t, buf.String(), "Source:")
	assert.Contains(t, buf.String(), "Chunks:")
}

// Document Exclude Tests

func TestDocumentExcludeCmd_Use(t *testing.T) {
	assert.Equal(t, "exclude [doc-id]", documentExcludeCmd.Use)
}

func TestDocumentExcludeCmd_RequiresExactlyOneArg(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"document", "exclude"})
	defer func() {
		rootCmd.SetArgs(nil)
	}()

	err := rootCmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 1 arg(s)")
}

func TestDocumentExcludeCmd_ExecutesWithArg(t *testing.T) {
	cleanup := setupTestServices()
	defer cleanup()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"document", "exclude", "doc-1"})
	defer func() {
		rootCmd.SetArgs(nil)
	}()

	err := rootCmd.Execute()

	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "excluded from index")
}

func TestDocumentExcludeCmd_WithReasonFlag(t *testing.T) {
	cleanup := setupTestServices()
	defer cleanup()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"document", "exclude", "doc-1", "--reason", "outdated content"})
	defer func() {
		rootCmd.SetArgs(nil)
		excludeReason = "" // Reset flag
	}()

	err := rootCmd.Execute()

	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "excluded from index")
}

// Document Refresh Tests

func TestDocumentRefreshCmd_Use(t *testing.T) {
	assert.Equal(t, "refresh [doc-id]", documentRefreshCmd.Use)
}

func TestDocumentRefreshCmd_RequiresExactlyOneArg(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"document", "refresh"})
	defer func() {
		rootCmd.SetArgs(nil)
	}()

	err := rootCmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 1 arg(s)")
}

func TestDocumentRefreshCmd_ExecutesWithArg(t *testing.T) {
	cleanup := setupTestServices()
	defer cleanup()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"document", "refresh", "doc-1"})
	defer func() {
		rootCmd.SetArgs(nil)
	}()

	err := rootCmd.Execute()

	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "refreshed successfully")
}

// Document Open Tests

func TestDocumentOpenCmd_Use(t *testing.T) {
	assert.Equal(t, "open [doc-id]", documentOpenCmd.Use)
}

func TestDocumentOpenCmd_RequiresExactlyOneArg(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"document", "open"})
	defer func() {
		rootCmd.SetArgs(nil)
	}()

	err := rootCmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 1 arg(s)")
}

func TestDocumentOpenCmd_ExecutesWithArg(t *testing.T) {
	cleanup := setupTestServices()
	defer cleanup()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"document", "open", "doc-1"})
	defer func() {
		rootCmd.SetArgs(nil)
	}()

	err := rootCmd.Execute()

	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "Opened document")
}

// Service Not Configured Tests

func TestDocumentListCmd_ServiceNotConfigured(t *testing.T) {
	// Ensure documentService is nil
	oldService := documentService
	documentService = nil
	defer func() {
		documentService = oldService
	}()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"document", "list", "src-1"})
	defer func() {
		rootCmd.SetArgs(nil)
	}()

	err := rootCmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "document service not configured")
}

func TestDocumentGetCmd_ServiceNotConfigured(t *testing.T) {
	oldService := documentService
	documentService = nil
	defer func() {
		documentService = oldService
	}()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"document", "get", "doc-1"})
	defer func() {
		rootCmd.SetArgs(nil)
	}()

	err := rootCmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "document service not configured")
}

func TestDocumentContentCmd_ServiceNotConfigured(t *testing.T) {
	oldService := documentService
	documentService = nil
	defer func() {
		documentService = oldService
	}()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"document", "content", "doc-1"})
	defer func() {
		rootCmd.SetArgs(nil)
	}()

	err := rootCmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "document service not configured")
}

func TestDocumentDetailsCmd_ServiceNotConfigured(t *testing.T) {
	oldService := documentService
	documentService = nil
	defer func() {
		documentService = oldService
	}()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"document", "details", "doc-1"})
	defer func() {
		rootCmd.SetArgs(nil)
	}()

	err := rootCmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "document service not configured")
}

func TestDocumentExcludeCmd_ServiceNotConfigured(t *testing.T) {
	oldService := documentService
	documentService = nil
	defer func() {
		documentService = oldService
	}()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"document", "exclude", "doc-1"})
	defer func() {
		rootCmd.SetArgs(nil)
	}()

	err := rootCmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "document service not configured")
}

func TestDocumentRefreshCmd_ServiceNotConfigured(t *testing.T) {
	oldService := documentService
	documentService = nil
	defer func() {
		documentService = oldService
	}()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"document", "refresh", "doc-1"})
	defer func() {
		rootCmd.SetArgs(nil)
	}()

	err := rootCmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "document service not configured")
}

func TestDocumentOpenCmd_ServiceNotConfigured(t *testing.T) {
	oldService := documentService
	documentService = nil
	defer func() {
		documentService = oldService
	}()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"document", "open", "doc-1"})
	defer func() {
		rootCmd.SetArgs(nil)
	}()

	err := rootCmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "document service not configured")
}

// Error Case Tests

func TestDocumentListCmd_EmptyList(t *testing.T) {
	// Create a mock that returns empty list
	oldService := documentService
	documentService = &mockDocumentServiceEmpty{}
	defer func() {
		documentService = oldService
	}()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"document", "list", "src-1"})
	defer func() {
		rootCmd.SetArgs(nil)
	}()

	err := rootCmd.Execute()

	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "No documents found")
}

func TestDocumentGetCmd_WithoutMetadata(t *testing.T) {
	// Create a mock that returns document without metadata
	oldService := documentService
	documentService = &mockDocumentServiceNoMetadata{}
	defer func() {
		documentService = oldService
	}()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"document", "get", "doc-1"})
	defer func() {
		rootCmd.SetArgs(nil)
	}()

	err := rootCmd.Execute()

	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "Document:")
	assert.NotContains(t, buf.String(), "Metadata:")
}

func TestDocumentDetailsCmd_WithoutMetadata(t *testing.T) {
	// Create a mock that returns details without metadata
	oldService := documentService
	documentService = &mockDocumentServiceNoMetadata{}
	defer func() {
		documentService = oldService
	}()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"document", "details", "doc-1"})
	defer func() {
		rootCmd.SetArgs(nil)
	}()

	err := rootCmd.Execute()

	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "Document Details:")
	assert.NotContains(t, buf.String(), "Metadata:")
}

func TestDocumentListCmd_WithoutURI(t *testing.T) {
	// Create a mock that returns documents without URI
	oldService := documentService
	documentService = &mockDocumentServiceNoURI{}
	defer func() {
		documentService = oldService
	}()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"document", "list", "src-1"})
	defer func() {
		rootCmd.SetArgs(nil)
	}()

	err := rootCmd.Execute()

	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "Documents for source")
	assert.NotContains(t, buf.String(), "URI:")
}

// Service Error Tests

func TestDocumentListCmd_ServiceError(t *testing.T) {
	oldService := documentService
	documentService = &mockDocumentServiceError{}
	defer func() {
		documentService = oldService
	}()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"document", "list", "src-1"})
	defer func() {
		rootCmd.SetArgs(nil)
	}()

	err := rootCmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list documents")
}

func TestDocumentGetCmd_ServiceError(t *testing.T) {
	oldService := documentService
	documentService = &mockDocumentServiceError{}
	defer func() {
		documentService = oldService
	}()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"document", "get", "doc-1"})
	defer func() {
		rootCmd.SetArgs(nil)
	}()

	err := rootCmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get document")
}

func TestDocumentContentCmd_ServiceError(t *testing.T) {
	oldService := documentService
	documentService = &mockDocumentServiceError{}
	defer func() {
		documentService = oldService
	}()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"document", "content", "doc-1"})
	defer func() {
		rootCmd.SetArgs(nil)
	}()

	err := rootCmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get document content")
}

func TestDocumentDetailsCmd_ServiceError(t *testing.T) {
	oldService := documentService
	documentService = &mockDocumentServiceError{}
	defer func() {
		documentService = oldService
	}()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"document", "details", "doc-1"})
	defer func() {
		rootCmd.SetArgs(nil)
	}()

	err := rootCmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get document details")
}

func TestDocumentExcludeCmd_ServiceError(t *testing.T) {
	oldService := documentService
	documentService = &mockDocumentServiceError{}
	defer func() {
		documentService = oldService
	}()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"document", "exclude", "doc-1"})
	defer func() {
		rootCmd.SetArgs(nil)
	}()

	err := rootCmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to exclude document")
}

func TestDocumentRefreshCmd_ServiceError(t *testing.T) {
	oldService := documentService
	documentService = &mockDocumentServiceError{}
	defer func() {
		documentService = oldService
	}()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"document", "refresh", "doc-1"})
	defer func() {
		rootCmd.SetArgs(nil)
	}()

	err := rootCmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to refresh document")
}

func TestDocumentOpenCmd_ServiceError(t *testing.T) {
	oldService := documentService
	documentService = &mockDocumentServiceError{}
	defer func() {
		documentService = oldService
	}()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"document", "open", "doc-1"})
	defer func() {
		rootCmd.SetArgs(nil)
	}()

	err := rootCmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open document")
}
