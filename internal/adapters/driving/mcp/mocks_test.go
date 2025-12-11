package mcp

import (
	"context"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driving"
)

// mockSearchService is a mock implementation of driving.SearchService.
type mockSearchService struct {
	results []domain.SearchResult
	err     error
}

func (m *mockSearchService) Search(
	_ context.Context,
	_ string,
	_ domain.SearchOptions,
) ([]domain.SearchResult, error) {
	return m.results, m.err
}

// mockSourceService is a mock implementation of driving.SourceService.
type mockSourceService struct {
	sources []domain.Source
	source  *domain.Source
	err     error
}

func (m *mockSourceService) Add(_ context.Context, _ domain.Source) error {
	return m.err
}

func (m *mockSourceService) Get(_ context.Context, _ string) (*domain.Source, error) {
	return m.source, m.err
}

func (m *mockSourceService) List(_ context.Context) ([]domain.Source, error) {
	return m.sources, m.err
}

func (m *mockSourceService) Remove(_ context.Context, _ string) error {
	return m.err
}

func (m *mockSourceService) Update(_ context.Context, _ domain.Source) error {
	return m.err
}

func (m *mockSourceService) ValidateConfig(_ context.Context, _ string, _ map[string]string) error {
	return m.err
}

// mockDocumentService is a mock implementation of driving.DocumentService.
type mockDocumentService struct {
	documents []domain.Document
	document  *domain.Document
	content   string
	details   *driving.DocumentDetails
	err       error
}

func (m *mockDocumentService) ListBySource(_ context.Context, _ string) ([]domain.Document, error) {
	return m.documents, m.err
}

func (m *mockDocumentService) Get(_ context.Context, _ string) (*domain.Document, error) {
	return m.document, m.err
}

func (m *mockDocumentService) GetContent(_ context.Context, _ string) (string, error) {
	return m.content, m.err
}

func (m *mockDocumentService) GetDetails(_ context.Context, _ string) (*driving.DocumentDetails, error) {
	return m.details, m.err
}

func (m *mockDocumentService) Exclude(_ context.Context, _, _ string) error {
	return m.err
}

func (m *mockDocumentService) Refresh(_ context.Context, _ string) error {
	return m.err
}

func (m *mockDocumentService) Open(_ context.Context, _ string) error {
	return m.err
}
