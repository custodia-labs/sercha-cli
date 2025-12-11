package cli

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/custodia-labs/sercha-cli/internal/core/ports/driving"
)

// mockSyncOrchestrator implements driving.SyncOrchestrator for testing.
type mockSyncOrchestrator struct{}

func (m *mockSyncOrchestrator) Sync(_ context.Context, _ string) error {
	return nil
}

func (m *mockSyncOrchestrator) SyncAll(_ context.Context) error {
	return nil
}

func (m *mockSyncOrchestrator) Status(_ context.Context, _ string) (*driving.SyncStatus, error) {
	return nil, nil
}

func setupSyncTest() func() {
	oldSync := syncOrchestrator
	syncOrchestrator = &mockSyncOrchestrator{}
	return func() {
		syncOrchestrator = oldSync
	}
}

func TestSyncCmd_Use(t *testing.T) {
	assert.Equal(t, "sync [source-id]", syncCmd.Use)
}

func TestSyncCmd_Short(t *testing.T) {
	assert.Equal(t, "Synchronise documents from sources", syncCmd.Short)
}

func TestSyncCmd_Long(t *testing.T) {
	assert.Contains(t, syncCmd.Long, "document synchronisation")
	assert.Contains(t, syncCmd.Long, "source ID")
}

func TestSyncCmd_ExecutesWithoutArgs(t *testing.T) {
	cleanup := setupSyncTest()
	defer cleanup()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"sync"})
	defer func() {
		rootCmd.SetArgs(nil)
	}()

	err := rootCmd.Execute()

	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "Synchronising all sources...")
}

func TestSyncCmd_ExecutesWithSourceID(t *testing.T) {
	cleanup := setupSyncTest()
	defer cleanup()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"sync", "source-456"})
	defer func() {
		rootCmd.SetArgs(nil)
	}()

	err := rootCmd.Execute()

	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "Synchronising source: source-456")
}

func TestSyncCmd_ServiceNotConfigured(t *testing.T) {
	oldSync := syncOrchestrator
	syncOrchestrator = nil
	defer func() {
		syncOrchestrator = oldSync
	}()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"sync"})
	defer func() {
		rootCmd.SetArgs(nil)
	}()

	err := rootCmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "sync service not configured")
}

func TestSyncCmd_ServiceError_SingleSource(t *testing.T) {
	oldSync := syncOrchestrator
	syncOrchestrator = &mockSyncOrchestratorError{}
	defer func() {
		syncOrchestrator = oldSync
	}()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"sync", "src-1"})
	defer func() {
		rootCmd.SetArgs(nil)
	}()

	err := rootCmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "sync failed")
}

func TestSyncCmd_ServiceError_AllSources(t *testing.T) {
	oldSync := syncOrchestrator
	syncOrchestrator = &mockSyncOrchestratorError{}
	defer func() {
		syncOrchestrator = oldSync
	}()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"sync"})
	defer func() {
		rootCmd.SetArgs(nil)
	}()

	err := rootCmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "sync failed")
}
