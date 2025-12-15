package cli

import (
	"context"
	"time"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driving"
)

// mockSearchService implements driving.SearchService for testing.
type mockSearchService struct{}

func (m *mockSearchService) Search(
	_ context.Context, query string, _ domain.SearchOptions,
) ([]domain.SearchResult, error) {
	if query == "" {
		return []domain.SearchResult{}, nil
	}
	return []domain.SearchResult{
		{
			Document: domain.Document{ID: "doc-1", Title: "Test Doc"},
			Score:    0.95,
		},
	}, nil
}

// mockSourceService implements driving.SourceService for testing.
type mockSourceService struct{}

func (m *mockSourceService) Add(_ context.Context, source domain.Source) error {
	return nil
}

func (m *mockSourceService) Get(_ context.Context, id string) (*domain.Source, error) {
	return &domain.Source{ID: id, Type: "filesystem", Name: "test"}, nil
}

func (m *mockSourceService) List(_ context.Context) ([]domain.Source, error) {
	return []domain.Source{
		{ID: "src-1", Type: "filesystem", Name: "~/Documents"},
	}, nil
}

func (m *mockSourceService) Remove(_ context.Context, id string) error {
	return nil
}

func (m *mockSourceService) Update(_ context.Context, _ domain.Source) error {
	return nil
}

func (m *mockSourceService) ValidateConfig(_ context.Context, _ string, _ map[string]string) error {
	return nil
}

// mockSourceServiceEmpty implements driving.SourceService that returns empty lists.
type mockSourceServiceEmpty struct{}

func (m *mockSourceServiceEmpty) Add(_ context.Context, _ domain.Source) error {
	return nil
}

func (m *mockSourceServiceEmpty) Get(_ context.Context, id string) (*domain.Source, error) {
	return nil, domain.ErrNotFound
}

func (m *mockSourceServiceEmpty) List(_ context.Context) ([]domain.Source, error) {
	return []domain.Source{}, nil
}

func (m *mockSourceServiceEmpty) Remove(_ context.Context, _ string) error {
	return nil
}

func (m *mockSourceServiceEmpty) Update(_ context.Context, _ domain.Source) error {
	return nil
}

func (m *mockSourceServiceEmpty) ValidateConfig(_ context.Context, _ string, _ map[string]string) error {
	return nil
}

// mockSourceServiceWithAuth implements driving.SourceService that returns sources with authorization IDs.
type mockSourceServiceWithAuth struct{}

func (m *mockSourceServiceWithAuth) Add(_ context.Context, _ domain.Source) error {
	return nil
}

func (m *mockSourceServiceWithAuth) Get(_ context.Context, id string) (*domain.Source, error) {
	return &domain.Source{ID: id, Type: "github", Name: "test", AuthorizationID: "auth-1"}, nil
}

func (m *mockSourceServiceWithAuth) List(_ context.Context) ([]domain.Source, error) {
	return []domain.Source{
		{ID: "src-1", Type: "github", Name: "My GitHub Repo", AuthorizationID: "auth-1"},
		{ID: "src-2", Type: "filesystem", Name: "~/Documents", AuthorizationID: ""},
	}, nil
}

func (m *mockSourceServiceWithAuth) Remove(_ context.Context, _ string) error {
	return nil
}

func (m *mockSourceServiceWithAuth) Update(_ context.Context, _ domain.Source) error {
	return nil
}

func (m *mockSourceServiceWithAuth) ValidateConfig(_ context.Context, _ string, _ map[string]string) error {
	return nil
}

// mockSyncOrchestratorFull implements driving.SyncOrchestrator for testing.
type mockSyncOrchestratorFull struct{}

func (m *mockSyncOrchestratorFull) Sync(_ context.Context, _ string) error {
	return nil
}

func (m *mockSyncOrchestratorFull) SyncAll(_ context.Context) error {
	return nil
}

func (m *mockSyncOrchestratorFull) Status(_ context.Context, _ string) (*driving.SyncStatus, error) {
	return nil, nil
}

// mockDocumentService implements driving.DocumentService for testing.
type mockDocumentService struct{}

func (m *mockDocumentService) ListBySource(_ context.Context, sourceID string) ([]domain.Document, error) {
	return []domain.Document{
		{ID: "doc-1", SourceID: sourceID, Title: "Test Document 1", URI: "/path/to/doc1.txt"},
		{ID: "doc-2", SourceID: sourceID, Title: "Test Document 2", URI: "/path/to/doc2.txt"},
	}, nil
}

func (m *mockDocumentService) Get(_ context.Context, documentID string) (*domain.Document, error) {
	return &domain.Document{
		ID:        documentID,
		SourceID:  "src-1",
		Title:     "Test Document",
		URI:       "/path/to/document.txt",
		CreatedAt: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		UpdatedAt: time.Date(2024, 1, 16, 14, 0, 0, 0, time.UTC),
		Metadata:  map[string]any{"author": "test"},
	}, nil
}

func (m *mockDocumentService) GetContent(_ context.Context, _ string) (string, error) {
	return "This is the content of the test document.", nil
}

func (m *mockDocumentService) GetDetails(_ context.Context, documentID string) (*driving.DocumentDetails, error) {
	return &driving.DocumentDetails{
		ID:         documentID,
		SourceID:   "src-1",
		SourceName: "Test Source",
		SourceType: "filesystem",
		Title:      "Test Document",
		URI:        "/path/to/document.txt",
		ChunkCount: 5,
		CreatedAt:  time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		UpdatedAt:  time.Date(2024, 1, 16, 14, 0, 0, 0, time.UTC),
		Metadata:   map[string]string{"author": "test"},
	}, nil
}

func (m *mockDocumentService) Exclude(_ context.Context, _, _ string) error {
	return nil
}

func (m *mockDocumentService) Refresh(_ context.Context, _ string) error {
	return nil
}

func (m *mockDocumentService) Open(_ context.Context, _ string) error {
	return nil
}

// mockDocumentServiceEmpty returns empty lists for testing edge cases.
type mockDocumentServiceEmpty struct{}

func (m *mockDocumentServiceEmpty) ListBySource(_ context.Context, _ string) ([]domain.Document, error) {
	return []domain.Document{}, nil
}

func (m *mockDocumentServiceEmpty) Get(_ context.Context, documentID string) (*domain.Document, error) {
	return &domain.Document{ID: documentID}, nil
}

func (m *mockDocumentServiceEmpty) GetContent(_ context.Context, _ string) (string, error) {
	return "", nil
}

func (m *mockDocumentServiceEmpty) GetDetails(_ context.Context, documentID string) (*driving.DocumentDetails, error) {
	return &driving.DocumentDetails{ID: documentID}, nil
}

func (m *mockDocumentServiceEmpty) Exclude(_ context.Context, _, _ string) error {
	return nil
}

func (m *mockDocumentServiceEmpty) Refresh(_ context.Context, _ string) error {
	return nil
}

func (m *mockDocumentServiceEmpty) Open(_ context.Context, _ string) error {
	return nil
}

// mockDocumentServiceNoMetadata returns documents without metadata for testing edge cases.
type mockDocumentServiceNoMetadata struct{}

func (m *mockDocumentServiceNoMetadata) ListBySource(_ context.Context, sourceID string) ([]domain.Document, error) {
	return []domain.Document{
		{ID: "doc-1", SourceID: sourceID, Title: "Test Document 1"},
	}, nil
}

func (m *mockDocumentServiceNoMetadata) Get(_ context.Context, documentID string) (*domain.Document, error) {
	return &domain.Document{
		ID:        documentID,
		SourceID:  "src-1",
		Title:     "Test Document",
		URI:       "/path/to/document.txt",
		CreatedAt: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		UpdatedAt: time.Date(2024, 1, 16, 14, 0, 0, 0, time.UTC),
		Metadata:  map[string]any{}, // Empty metadata
	}, nil
}

func (m *mockDocumentServiceNoMetadata) GetContent(_ context.Context, _ string) (string, error) {
	return "content", nil
}

func (m *mockDocumentServiceNoMetadata) GetDetails(_ context.Context, documentID string) (*driving.DocumentDetails, error) {
	return &driving.DocumentDetails{
		ID:         documentID,
		SourceID:   "src-1",
		SourceName: "Test Source",
		SourceType: "filesystem",
		Title:      "Test Document",
		URI:        "/path/to/document.txt",
		ChunkCount: 5,
		CreatedAt:  time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		UpdatedAt:  time.Date(2024, 1, 16, 14, 0, 0, 0, time.UTC),
		Metadata:   map[string]string{}, // Empty metadata
	}, nil
}

func (m *mockDocumentServiceNoMetadata) Exclude(_ context.Context, _, _ string) error {
	return nil
}

func (m *mockDocumentServiceNoMetadata) Refresh(_ context.Context, _ string) error {
	return nil
}

func (m *mockDocumentServiceNoMetadata) Open(_ context.Context, _ string) error {
	return nil
}

// mockDocumentServiceNoURI returns documents without URI for testing edge cases.
type mockDocumentServiceNoURI struct{}

func (m *mockDocumentServiceNoURI) ListBySource(_ context.Context, sourceID string) ([]domain.Document, error) {
	return []domain.Document{
		{ID: "doc-1", SourceID: sourceID, Title: "Test Document 1", URI: ""},
		{ID: "doc-2", SourceID: sourceID, Title: "Test Document 2", URI: ""},
	}, nil
}

func (m *mockDocumentServiceNoURI) Get(_ context.Context, documentID string) (*domain.Document, error) {
	return &domain.Document{ID: documentID}, nil
}

func (m *mockDocumentServiceNoURI) GetContent(_ context.Context, _ string) (string, error) {
	return "", nil
}

func (m *mockDocumentServiceNoURI) GetDetails(_ context.Context, documentID string) (*driving.DocumentDetails, error) {
	return &driving.DocumentDetails{ID: documentID}, nil
}

func (m *mockDocumentServiceNoURI) Exclude(_ context.Context, _, _ string) error {
	return nil
}

func (m *mockDocumentServiceNoURI) Refresh(_ context.Context, _ string) error {
	return nil
}

func (m *mockDocumentServiceNoURI) Open(_ context.Context, _ string) error {
	return nil
}

// mockConnectorRegistry implements driving.ConnectorRegistry for testing.
type mockConnectorRegistry struct{}

func (m *mockConnectorRegistry) List() []domain.ConnectorType {
	return []domain.ConnectorType{
		{
			ID:           "filesystem",
			Name:         "Local Filesystem",
			Description:  "Index local files and directories",
			ProviderType: domain.ProviderLocal,
			AuthMethod:   domain.AuthMethodNone,
			ConfigKeys: []domain.ConfigKey{
				{
					Key:         "path",
					Label:       "Directory path",
					Description: "Path to the directory to index",
					Required:    true,
				},
			},
		},
		{
			ID:           "github",
			Name:         "GitHub",
			Description:  "Index GitHub repositories",
			ProviderType: domain.ProviderGitHub,
			AuthMethod:   domain.AuthMethodOAuth,
			ConfigKeys: []domain.ConfigKey{
				{
					Key:         "owner",
					Label:       "Repository owner",
					Description: "GitHub username or organization",
					Required:    true,
				},
				{
					Key:         "repo",
					Label:       "Repository name",
					Description: "Name of the repository",
					Required:    true,
				},
			},
		},
	}
}

func (m *mockConnectorRegistry) Get(id string) (*domain.ConnectorType, error) {
	for _, c := range m.List() {
		if c.ID == id {
			return &c, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (m *mockConnectorRegistry) GetOAuthDefaults(_ string) *driving.OAuthDefaults {
	return nil
}

func (m *mockConnectorRegistry) SupportsOAuth(_ string) bool {
	return false
}

func (m *mockConnectorRegistry) BuildAuthURL(_ string, authProvider *domain.AuthProvider, _, state, _ string) (string, error) {
	return "https://example.com/oauth/authorize?client_id=" + authProvider.OAuth.ClientID + "&state=" + state, nil
}

func (m *mockConnectorRegistry) GetUserInfo(_ context.Context, _ string, _ string) (string, error) {
	return "test@example.com", nil
}

func (m *mockConnectorRegistry) GetSetupHint(_ string) string {
	return ""
}

func (m *mockConnectorRegistry) GetConnectorsForProvider(provider domain.ProviderType) []domain.ConnectorType {
	var result []domain.ConnectorType
	for _, c := range m.List() {
		if c.ProviderType == provider {
			result = append(result, c)
		}
	}
	return result
}

func (m *mockConnectorRegistry) ValidateConfig(_ string, _ map[string]string) error {
	return nil
}

func (m *mockConnectorRegistry) ExchangeCode(_ context.Context, _ string, _ *domain.AuthProvider, _, _, _ string) (*domain.OAuthToken, error) {
	return nil, nil
}

// mockConnectorRegistryEmpty implements driving.ConnectorRegistry that returns empty list.
type mockConnectorRegistryEmpty struct{}

func (m *mockConnectorRegistryEmpty) List() []domain.ConnectorType {
	return []domain.ConnectorType{}
}

func (m *mockConnectorRegistryEmpty) Get(_ string) (*domain.ConnectorType, error) {
	return nil, domain.ErrNotFound
}

func (m *mockConnectorRegistryEmpty) GetOAuthDefaults(_ string) *driving.OAuthDefaults {
	return nil
}

func (m *mockConnectorRegistryEmpty) SupportsOAuth(_ string) bool {
	return false
}

func (m *mockConnectorRegistryEmpty) BuildAuthURL(_ string, _ *domain.AuthProvider, _, _, _ string) (string, error) {
	return "", domain.ErrNotFound
}

func (m *mockConnectorRegistryEmpty) GetUserInfo(_ context.Context, _ string, _ string) (string, error) {
	return "", domain.ErrNotFound
}

func (m *mockConnectorRegistryEmpty) GetSetupHint(_ string) string {
	return ""
}

func (m *mockConnectorRegistryEmpty) GetConnectorsForProvider(_ domain.ProviderType) []domain.ConnectorType {
	return nil
}

func (m *mockConnectorRegistryEmpty) ValidateConfig(_ string, _ map[string]string) error {
	return domain.ErrNotFound
}

func (m *mockConnectorRegistryEmpty) ExchangeCode(_ context.Context, _ string, _ *domain.AuthProvider, _, _, _ string) (*domain.OAuthToken, error) {
	return nil, domain.ErrNotFound
}

// mockSearchServiceError implements driving.SearchService that returns errors.
type mockSearchServiceError struct{}

func (m *mockSearchServiceError) Search(_ context.Context, _ string, _ domain.SearchOptions) ([]domain.SearchResult, error) {
	return nil, domain.ErrNotFound
}

// mockSourceServiceError implements driving.SourceService that returns errors.
type mockSourceServiceError struct{}

func (m *mockSourceServiceError) Add(_ context.Context, _ domain.Source) error {
	return domain.ErrNotFound
}

func (m *mockSourceServiceError) Get(_ context.Context, _ string) (*domain.Source, error) {
	return nil, domain.ErrNotFound
}

func (m *mockSourceServiceError) List(_ context.Context) ([]domain.Source, error) {
	return nil, domain.ErrNotFound
}

func (m *mockSourceServiceError) Remove(_ context.Context, _ string) error {
	return domain.ErrNotFound
}

func (m *mockSourceServiceError) Update(_ context.Context, _ domain.Source) error {
	return domain.ErrNotFound
}

func (m *mockSourceServiceError) ValidateConfig(_ context.Context, _ string, _ map[string]string) error {
	return domain.ErrNotFound
}

// mockDocumentServiceError implements driving.DocumentService that returns errors.
type mockDocumentServiceError struct{}

func (m *mockDocumentServiceError) ListBySource(_ context.Context, _ string) ([]domain.Document, error) {
	return nil, domain.ErrNotFound
}

func (m *mockDocumentServiceError) Get(_ context.Context, _ string) (*domain.Document, error) {
	return nil, domain.ErrNotFound
}

func (m *mockDocumentServiceError) GetContent(_ context.Context, _ string) (string, error) {
	return "", domain.ErrNotFound
}

func (m *mockDocumentServiceError) GetDetails(_ context.Context, _ string) (*driving.DocumentDetails, error) {
	return nil, domain.ErrNotFound
}

func (m *mockDocumentServiceError) Exclude(_ context.Context, _, _ string) error {
	return domain.ErrNotFound
}

func (m *mockDocumentServiceError) Refresh(_ context.Context, _ string) error {
	return domain.ErrNotFound
}

func (m *mockDocumentServiceError) Open(_ context.Context, _ string) error {
	return domain.ErrNotFound
}

// mockSyncOrchestratorError implements driving.SyncOrchestrator that returns errors.
type mockSyncOrchestratorError struct{}

func (m *mockSyncOrchestratorError) Sync(_ context.Context, _ string) error {
	return domain.ErrNotFound
}

func (m *mockSyncOrchestratorError) SyncAll(_ context.Context) error {
	return domain.ErrNotFound
}

func (m *mockSyncOrchestratorError) Status(_ context.Context, _ string) (*driving.SyncStatus, error) {
	return nil, domain.ErrNotFound
}

// setupTestServices injects mock services for testing and returns a cleanup func.
func setupTestServices() func() {
	oldSearch := searchService
	oldSource := sourceService
	oldSync := syncOrchestrator
	oldDocument := documentService

	searchService = &mockSearchService{}
	sourceService = &mockSourceService{}
	syncOrchestrator = &mockSyncOrchestratorFull{}
	documentService = &mockDocumentService{}

	return func() {
		searchService = oldSearch
		sourceService = oldSource
		syncOrchestrator = oldSync
		documentService = oldDocument
	}
}
