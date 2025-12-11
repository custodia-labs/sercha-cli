package services

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driving"
)

// Ensure DocumentService implements the interface.
var _ driving.DocumentService = (*DocumentService)(nil)

// Sentinel errors for stub implementations.
var ErrRefreshNotImplemented = errors.New("document refresh not yet implemented")

// DocumentService manages documents within sources.
type DocumentService struct {
	docStore          driven.DocumentStore
	sourceStore       driven.SourceStore
	exclusionStore    driven.ExclusionStore
	connectorRegistry driving.ConnectorRegistry
}

// NewDocumentService creates a new document service.
func NewDocumentService(
	docStore driven.DocumentStore,
	sourceStore driven.SourceStore,
	exclusionStore driven.ExclusionStore,
	connectorRegistry driving.ConnectorRegistry,
) *DocumentService {
	return &DocumentService{
		docStore:          docStore,
		sourceStore:       sourceStore,
		exclusionStore:    exclusionStore,
		connectorRegistry: connectorRegistry,
	}
}

// ListBySource returns all documents for a source.
func (s *DocumentService) ListBySource(ctx context.Context, sourceID string) ([]domain.Document, error) {
	if s.docStore == nil {
		return nil, domain.ErrNotImplemented
	}
	return s.docStore.ListDocuments(ctx, sourceID)
}

// Get retrieves a document by ID.
func (s *DocumentService) Get(ctx context.Context, documentID string) (*domain.Document, error) {
	if s.docStore == nil {
		return nil, domain.ErrNotImplemented
	}
	return s.docStore.GetDocument(ctx, documentID)
}

// GetContent returns the concatenated content of all chunks.
func (s *DocumentService) GetContent(ctx context.Context, documentID string) (string, error) {
	if s.docStore == nil {
		return "", domain.ErrNotImplemented
	}

	// Verify document exists
	_, err := s.docStore.GetDocument(ctx, documentID)
	if err != nil {
		return "", err
	}

	// Get all chunks
	chunks, err := s.docStore.GetChunks(ctx, documentID)
	if err != nil {
		return "", err
	}

	// Sort by position
	sort.Slice(chunks, func(i, j int) bool {
		return chunks[i].Position < chunks[j].Position
	})

	// Concatenate content
	var builder strings.Builder
	for i, chunk := range chunks {
		if i > 0 {
			builder.WriteString("\n")
		}
		builder.WriteString(chunk.Content)
	}

	return builder.String(), nil
}

// GetDetails returns connector-agnostic metadata for display.
func (s *DocumentService) GetDetails(ctx context.Context, documentID string) (*driving.DocumentDetails, error) {
	if s.docStore == nil {
		return nil, domain.ErrNotImplemented
	}

	// Get document
	doc, err := s.docStore.GetDocument(ctx, documentID)
	if err != nil {
		return nil, err
	}

	// Get source info
	var sourceName, sourceType string
	if s.sourceStore != nil {
		source, err := s.sourceStore.Get(ctx, doc.SourceID)
		if err == nil && source != nil {
			sourceName = source.Name
			sourceType = source.Type
		}
	}

	// Get chunk count
	chunks, err := s.docStore.GetChunks(ctx, documentID)
	chunkCount := 0
	if err == nil {
		chunkCount = len(chunks)
	}

	// Flatten metadata to string map
	metadata := make(map[string]string)
	for key, value := range doc.Metadata {
		metadata[key] = fmt.Sprintf("%v", value)
	}

	return &driving.DocumentDetails{
		ID:         doc.ID,
		SourceID:   doc.SourceID,
		SourceName: sourceName,
		SourceType: sourceType,
		Title:      doc.Title,
		URI:        doc.URI,
		ChunkCount: chunkCount,
		CreatedAt:  doc.CreatedAt,
		UpdatedAt:  doc.UpdatedAt,
		Metadata:   metadata,
	}, nil
}

// Exclude removes a document and marks it to skip during re-sync.
func (s *DocumentService) Exclude(ctx context.Context, documentID, reason string) error {
	if s.docStore == nil {
		return domain.ErrNotImplemented
	}

	// Get document first to capture URI
	doc, err := s.docStore.GetDocument(ctx, documentID)
	if err != nil {
		return err
	}

	// Add to exclusion store
	if s.exclusionStore != nil {
		exclusion := &domain.Exclusion{
			ID:         fmt.Sprintf("excl-%s", documentID),
			SourceID:   doc.SourceID,
			DocumentID: documentID,
			URI:        doc.URI,
			Reason:     reason,
			ExcludedAt: time.Now(),
		}
		if err := s.exclusionStore.Add(ctx, exclusion); err != nil {
			return fmt.Errorf("failed to add exclusion: %w", err)
		}
	}

	// Delete the document
	return s.docStore.DeleteDocument(ctx, documentID)
}

// Refresh re-syncs a single document from its source.
// TODO: Implement when sync infrastructure supports single-document refresh.
func (s *DocumentService) Refresh(_ context.Context, _ string) error {
	return ErrRefreshNotImplemented
}

// Open opens the document in the default application.
func (s *DocumentService) Open(ctx context.Context, documentID string) error {
	if s.docStore == nil {
		return domain.ErrNotImplemented
	}

	// Get document to retrieve its URI and metadata
	doc, err := s.docStore.GetDocument(ctx, documentID)
	if err != nil {
		return err
	}

	// Try to resolve using the connector's WebURLResolver
	openableURL := s.resolveWebURL(ctx, doc)

	// Open the resolved URL using the OS-specific command
	return openURL(openableURL)
}

// resolveWebURL converts a document URI to an openable URL using the connector's resolver.
func (s *DocumentService) resolveWebURL(ctx context.Context, doc *domain.Document) string {
	if resolved := s.tryConnectorResolver(ctx, doc); resolved != "" {
		return resolved
	}
	return convertToOpenableURL(doc.URI)
}

// tryConnectorResolver attempts to resolve URL using the connector's WebURLResolver.
func (s *DocumentService) tryConnectorResolver(ctx context.Context, doc *domain.Document) string {
	if s.sourceStore == nil || s.connectorRegistry == nil {
		return ""
	}
	source, err := s.sourceStore.Get(ctx, doc.SourceID)
	if err != nil || source == nil {
		return ""
	}
	connectorType, err := s.connectorRegistry.Get(source.Type)
	if err != nil || connectorType == nil || connectorType.WebURLResolver == nil {
		return ""
	}
	return connectorType.WebURLResolver(doc.URI, doc.Metadata)
}

// openURL opens a URL/path using the system default handler.
func openURL(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	return cmd.Start()
}

// convertToOpenableURL converts internal URIs to browser-openable URLs.
func convertToOpenableURL(uri string) string {
	// GitHub URIs: github://owner/repo/blob/branch/path -> https://github.com/owner/repo/blob/branch/path
	if strings.HasPrefix(uri, "github://") {
		return "https://github.com/" + strings.TrimPrefix(uri, "github://")
	}

	// GitHub issue URIs: github://owner/repo/issues/123 -> https://github.com/owner/repo/issues/123
	// (Already handled by the above rule)

	// GitHub PR URIs: github://owner/repo/pull/123 -> https://github.com/owner/repo/pull/123
	// (Already handled by the above rule)

	// File URIs: file:///path/to/file -> /path/to/file (for local opening)
	if strings.HasPrefix(uri, "file://") {
		return strings.TrimPrefix(uri, "file://")
	}

	// HTTP/HTTPS URLs: pass through as-is
	if strings.HasPrefix(uri, "http://") || strings.HasPrefix(uri, "https://") {
		return uri
	}

	// Local file paths: pass through as-is
	return uri
}
