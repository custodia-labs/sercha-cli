package cli

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driving"
)

// MockTUISearchService implements driving.SearchService for TUI tests.
type MockTUISearchService struct {
	SearchFunc func(
		ctx context.Context, query string, opts domain.SearchOptions,
	) ([]domain.SearchResult, error)
}

func (m *MockTUISearchService) Search(
	ctx context.Context, query string, opts domain.SearchOptions,
) ([]domain.SearchResult, error) {
	if m.SearchFunc != nil {
		return m.SearchFunc(ctx, query, opts)
	}
	return []domain.SearchResult{}, nil
}

// MockTUISourceService implements driving.SourceService for TUI tests.
type MockTUISourceService struct{}

func (m *MockTUISourceService) Add(ctx context.Context, source domain.Source) error {
	return nil
}

func (m *MockTUISourceService) List(ctx context.Context) ([]domain.Source, error) {
	return []domain.Source{}, nil
}

func (m *MockTUISourceService) Remove(ctx context.Context, id string) error {
	return nil
}

func (m *MockTUISourceService) Get(ctx context.Context, id string) (*domain.Source, error) {
	return &domain.Source{}, nil
}

func (m *MockTUISourceService) Update(ctx context.Context, source domain.Source) error {
	return nil
}

func (m *MockTUISourceService) ValidateConfig(
	ctx context.Context,
	connectorType string,
	config map[string]string,
) error {
	return nil
}

// MockTUISyncOrchestrator implements driving.SyncOrchestrator for TUI tests.
type MockTUISyncOrchestrator struct{}

func (m *MockTUISyncOrchestrator) Sync(ctx context.Context, sourceID string) error {
	return nil
}

func (m *MockTUISyncOrchestrator) SyncAll(ctx context.Context) error {
	return nil
}

func (m *MockTUISyncOrchestrator) Status(ctx context.Context, sourceID string) (*driving.SyncStatus, error) {
	return nil, nil
}

func TestTUICmd_Exists(t *testing.T) {
	// Verify the tui command is registered
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "tui" {
			found = true
			break
		}
	}
	assert.True(t, found, "tui command should be registered")
}

func TestTUICmd_ShortDescription(t *testing.T) {
	assert.Equal(t, "Launch the interactive terminal UI", tuiCmd.Short)
}

func TestTUICmd_LongDescription(t *testing.T) {
	assert.Contains(t, tuiCmd.Long, "interactive terminal user interface")
	assert.Contains(t, tuiCmd.Long, "Controls:")
}

func TestSetTUIConfig(t *testing.T) {
	config := &TUIConfig{
		SearchService:    &MockTUISearchService{},
		SourceService:    &MockTUISourceService{},
		SyncOrchestrator: &MockTUISyncOrchestrator{},
	}

	SetTUIConfig(config)

	assert.Equal(t, config, tuiConfig)

	// Cleanup
	tuiConfig = nil
}

func TestTUICmd_HelpOutput(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"tui", "--help"})

	err := rootCmd.Execute()

	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "interactive terminal user interface")
	assert.Contains(t, output, "Controls:")
}

func TestTUIConfig_Fields(t *testing.T) {
	config := &TUIConfig{
		SearchService:    &MockTUISearchService{},
		SourceService:    &MockTUISourceService{},
		SyncOrchestrator: &MockTUISyncOrchestrator{},
	}

	assert.NotNil(t, config.SearchService)
	assert.NotNil(t, config.SourceService)
	assert.NotNil(t, config.SyncOrchestrator)
}
