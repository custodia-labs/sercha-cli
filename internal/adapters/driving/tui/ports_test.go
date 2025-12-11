package tui

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driving"
)

// MockSearchService implements driving.SearchService for testing.
type MockSearchService struct {
	SearchFunc func(
		ctx context.Context, query string, opts domain.SearchOptions,
	) ([]domain.SearchResult, error)
}

func (m *MockSearchService) Search(
	ctx context.Context, query string, opts domain.SearchOptions,
) ([]domain.SearchResult, error) {
	if m.SearchFunc != nil {
		return m.SearchFunc(ctx, query, opts)
	}
	return nil, nil
}

// MockSourceService implements driving.SourceService for testing.
type MockSourceService struct {
	AddFunc    func(ctx context.Context, source domain.Source) error
	GetFunc    func(ctx context.Context, id string) (*domain.Source, error)
	ListFunc   func(ctx context.Context) ([]domain.Source, error)
	RemoveFunc func(ctx context.Context, id string) error
}

func (m *MockSourceService) Add(ctx context.Context, source domain.Source) error {
	if m.AddFunc != nil {
		return m.AddFunc(ctx, source)
	}
	return nil
}

func (m *MockSourceService) Get(ctx context.Context, id string) (*domain.Source, error) {
	if m.GetFunc != nil {
		return m.GetFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockSourceService) List(ctx context.Context) ([]domain.Source, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx)
	}
	return nil, nil
}

func (m *MockSourceService) Remove(ctx context.Context, id string) error {
	if m.RemoveFunc != nil {
		return m.RemoveFunc(ctx, id)
	}
	return nil
}

func (m *MockSourceService) Update(ctx context.Context, source domain.Source) error {
	return nil
}

func (m *MockSourceService) ValidateConfig(ctx context.Context, connectorType string, config map[string]string) error {
	return nil
}

// MockSyncOrchestrator implements driving.SyncOrchestrator for testing.
type MockSyncOrchestrator struct {
	SyncFunc    func(ctx context.Context, sourceID string) error
	SyncAllFunc func(ctx context.Context) error
	StatusFunc  func(ctx context.Context, sourceID string) (*driving.SyncStatus, error)
}

func (m *MockSyncOrchestrator) Sync(ctx context.Context, sourceID string) error {
	if m.SyncFunc != nil {
		return m.SyncFunc(ctx, sourceID)
	}
	return nil
}

func (m *MockSyncOrchestrator) SyncAll(ctx context.Context) error {
	if m.SyncAllFunc != nil {
		return m.SyncAllFunc(ctx)
	}
	return nil
}

func (m *MockSyncOrchestrator) Status(ctx context.Context, sourceID string) (*driving.SyncStatus, error) {
	if m.StatusFunc != nil {
		return m.StatusFunc(ctx, sourceID)
	}
	return nil, nil
}

// MockResultActionService implements driving.ResultActionService for testing.
type MockResultActionService struct {
	CopyToClipboardFunc func(ctx context.Context, result *domain.SearchResult) error
	OpenDocumentFunc    func(ctx context.Context, result *domain.SearchResult) error
}

func (m *MockResultActionService) CopyToClipboard(ctx context.Context, result *domain.SearchResult) error {
	if m.CopyToClipboardFunc != nil {
		return m.CopyToClipboardFunc(ctx, result)
	}
	return nil
}

func (m *MockResultActionService) OpenDocument(ctx context.Context, result *domain.SearchResult) error {
	if m.OpenDocumentFunc != nil {
		return m.OpenDocumentFunc(ctx, result)
	}
	return nil
}

func TestNewPorts(t *testing.T) {
	search := &MockSearchService{}
	source := &MockSourceService{}
	sync := &MockSyncOrchestrator{}
	resultAction := &MockResultActionService{}

	ports := NewPorts(search, source, sync, resultAction)

	require.NotNil(t, ports)
	assert.Equal(t, search, ports.Search)
	assert.Equal(t, source, ports.Source)
	assert.Equal(t, sync, ports.Sync)
	assert.Equal(t, resultAction, ports.ResultAction)
}

func TestPorts_Validate_AllSet(t *testing.T) {
	ports := &Ports{
		Search: &MockSearchService{},
		Source: &MockSourceService{},
		Sync:   &MockSyncOrchestrator{},
	}

	err := ports.Validate()

	assert.NoError(t, err)
}

func TestPorts_Validate_MissingSearch(t *testing.T) {
	ports := &Ports{
		Search: nil,
		Source: &MockSourceService{},
		Sync:   &MockSyncOrchestrator{},
	}

	err := ports.Validate()

	assert.ErrorIs(t, err, ErrMissingSearchService)
}

func TestPorts_Validate_MissingSource(t *testing.T) {
	ports := &Ports{
		Search: &MockSearchService{},
		Source: nil,
		Sync:   &MockSyncOrchestrator{},
	}

	err := ports.Validate()

	assert.ErrorIs(t, err, ErrMissingSourceService)
}

func TestPorts_Validate_MissingSync(t *testing.T) {
	ports := &Ports{
		Search: &MockSearchService{},
		Source: &MockSourceService{},
		Sync:   nil,
	}

	err := ports.Validate()

	assert.ErrorIs(t, err, ErrMissingSyncOrchestrator)
}
