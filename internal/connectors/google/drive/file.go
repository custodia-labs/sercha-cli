package drive

import (
	"context"
	"fmt"
	"io"
	"strings"

	"google.golang.org/api/drive/v3"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

// Google Docs MIME types that can be exported.
const (
	MimeTypeGoogleDoc    = "application/vnd.google-apps.document"
	MimeTypeGoogleSheet  = "application/vnd.google-apps.spreadsheet"
	MimeTypeGoogleSlides = "application/vnd.google-apps.presentation"
	MimeTypeFolder       = "application/vnd.google-apps.folder"
)

// Export formats for Google Workspace files.
const (
	ExportMimeText = "text/plain"
	ExportMimeCSV  = "text/csv"
)

// MaxExportSize is the maximum size for exported content (5MB).
const MaxExportSize = 5 * 1024 * 1024

// FileToRawDocument converts a Drive file to a RawDocument.
func FileToRawDocument(
	ctx context.Context, svc *drive.Service, file *drive.File, sourceID string,
) (*domain.RawDocument, error) {
	// Skip folders
	if file.MimeType == MimeTypeFolder {
		return nil, nil
	}

	content, exportedMime, err := fetchFileContent(ctx, svc, file)
	if err != nil {
		// Log error but continue with metadata only
		content = nil
	}

	// Use exported MIME type if file was converted, otherwise use original
	mimeType := file.MimeType
	if exportedMime != "" {
		mimeType = exportedMime
	}

	// Build path from parents (simplified - just using first parent)
	path := buildFilePath(file)

	return &domain.RawDocument{
		SourceID: sourceID,
		URI:      fmt.Sprintf("gdrive://files/%s", file.Id),
		MIMEType: mimeType,
		Content:  content,
		Metadata: map[string]any{
			"file_id":       file.Id,
			"title":         file.Name,
			"path":          path,
			"size":          file.Size,
			"web_link":      file.WebViewLink,
			"modified_time": file.ModifiedTime,
		},
	}, nil
}

// fetchFileContent retrieves the content of a file.
// Returns (content, exportedMIME, error) where exportedMIME is non-empty if the file was converted.
func fetchFileContent(ctx context.Context, svc *drive.Service, file *drive.File) ([]byte, string, error) {
	// Handle Google Workspace files (Docs, Sheets, etc.)
	switch file.MimeType {
	case MimeTypeGoogleDoc:
		content, err := exportGoogleFile(ctx, svc, file.Id, ExportMimeText)
		return content, ExportMimeText, err
	case MimeTypeGoogleSheet:
		content, err := exportGoogleFile(ctx, svc, file.Id, ExportMimeCSV)
		return content, ExportMimeCSV, err
	case MimeTypeGoogleSlides:
		content, err := exportGoogleFile(ctx, svc, file.Id, ExportMimeText)
		return content, ExportMimeText, err
	}

	// Skip files we can't normalise or files that are too large
	if !shouldDownloadContent(file.MimeType) || file.Size > MaxExportSize {
		return nil, "", nil
	}

	// Download regular file content
	resp, err := svc.Files.Get(file.Id).Context(ctx).Download()
	if err != nil {
		return nil, "", fmt.Errorf("download file: %w", err)
	}
	defer resp.Body.Close()

	// Read with size limit
	limitedReader := io.LimitReader(resp.Body, MaxExportSize)
	data, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, "", fmt.Errorf("read file content: %w", err)
	}

	// Regular files keep their original MIME type (empty exportedMIME)
	return data, "", nil
}

// exportGoogleFile exports a Google Workspace file to the specified format.
func exportGoogleFile(ctx context.Context, svc *drive.Service, fileID, exportMime string) ([]byte, error) {
	resp, err := svc.Files.Export(fileID, exportMime).Context(ctx).Download()
	if err != nil {
		return nil, fmt.Errorf("export file: %w", err)
	}
	defer resp.Body.Close()

	// Read with size limit
	limitedReader := io.LimitReader(resp.Body, MaxExportSize)
	data, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("read export: %w", err)
	}

	return data, nil
}

// buildFilePath constructs a simple path representation.
func buildFilePath(file *drive.File) string {
	if len(file.Parents) == 0 {
		return "/" + file.Name
	}
	// We could resolve parent names, but that would require additional API calls.
	// For now, just use the parent ID as a placeholder.
	return fmt.Sprintf("/%s/%s", file.Parents[0], file.Name)
}

// shouldDownloadContent checks if a MIME type requires content download.
// This includes text files and binary formats that have normalisers (e.g., PDF).
func shouldDownloadContent(mimeType string) bool {
	// Text MIME types
	if strings.HasPrefix(mimeType, "text/") {
		return true
	}

	// Types that need content downloaded for normalisation
	downloadTypes := []string{
		// Text-based formats
		"application/json",
		"application/xml",
		"application/javascript",
		"application/x-yaml",
		"application/x-sh",
		"application/sql",
		// Binary formats with normalisers
		"application/pdf",
	}

	for _, t := range downloadTypes {
		if mimeType == t {
			return true
		}
	}

	return false
}

// ShouldSyncFile checks if a file should be synced based on config.
func ShouldSyncFile(file *drive.File, cfg *Config) bool {
	// Skip folders
	if file.MimeType == MimeTypeFolder {
		return false
	}

	// Skip trashed files
	if file.Trashed {
		return false
	}

	// Check MIME type filter
	if len(cfg.MimeTypeFilter) > 0 {
		found := false
		for _, filter := range cfg.MimeTypeFilter {
			if file.MimeType == filter {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check content type configuration
	switch file.MimeType {
	case MimeTypeGoogleDoc:
		return cfg.HasContentType(ContentDocs)
	case MimeTypeGoogleSheet:
		return cfg.HasContentType(ContentSheets)
	default:
		return cfg.HasContentType(ContentFiles)
	}
}
