package dropbox

import (
	"fmt"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/dropbox/dropbox-sdk-go-unofficial/v6/dropbox/files"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

// FileToRawDocument converts a Dropbox FileMetadata to a RawDocument.
func FileToRawDocument(file *files.FileMetadata, content []byte, sourceID string) *domain.RawDocument {
	metadata := map[string]any{
		"file_id":       file.Id,
		"title":         file.Name,
		"path":          file.PathDisplay,
		"size":          file.Size,
		"modified_time": file.ServerModified.Format(time.RFC3339),
		"rev":           file.Rev,
		"content_hash":  file.ContentHash,
	}

	parentURI := buildParentURI(file)
	mimeType := getMIMETypeWithContent(file.Name, content)

	return &domain.RawDocument{
		SourceID:  sourceID,
		URI:       fmt.Sprintf("dropbox://files/%s", file.Id),
		MIMEType:  mimeType,
		Content:   content,
		Metadata:  metadata,
		ParentURI: parentURI,
	}
}

// buildParentURI creates the parent URI for hierarchy tracking.
func buildParentURI(file *files.FileMetadata) *string {
	if file.PathDisplay == "" {
		return nil
	}

	parentPath := path.Dir(file.PathDisplay)
	if parentPath == "" || parentPath == "." || parentPath == "/" {
		return nil
	}

	// For Dropbox we use the parent path as the folder identifier
	uri := fmt.Sprintf("dropbox://folders%s", parentPath)
	return &uri
}

// ShouldSyncFile checks if a file should be synced based on config.
func ShouldSyncFile(file *files.FileMetadata, cfg *Config) bool {
	if file == nil {
		return false
	}

	// Check MIME type filter
	if len(cfg.MimeTypeFilter) > 0 {
		mimeType := getMIMEType(file.Name)
		found := false
		for _, filter := range cfg.MimeTypeFilter {
			if mimeType == filter || strings.HasPrefix(mimeType, filter) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// shouldDownloadContent checks if a MIME type requires content download.
// This includes text files and binary formats that have normalisers (e.g., PDF).
func shouldDownloadContent(mimeType string) bool {
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

// mimeTypes maps file extensions to MIME types.
var mimeTypes = map[string]string{
	// Text files
	".txt":  "text/plain",
	".md":   "text/markdown",
	".html": "text/html",
	".htm":  "text/html",
	".css":  "text/css",
	".csv":  "text/csv",
	".xml":  "application/xml",

	// Code files
	".js":   "application/javascript",
	".ts":   "application/typescript",
	".json": "application/json",
	".yaml": "application/x-yaml",
	".yml":  "application/x-yaml",
	".py":   "text/x-python",
	".go":   "text/x-go",
	".java": "text/x-java",
	".c":    "text/x-c",
	".cpp":  "text/x-c++",
	".h":    "text/x-c",
	".hpp":  "text/x-c++",
	".rs":   "text/x-rust",
	".rb":   "text/x-ruby",
	".php":  "text/x-php",
	".sql":  "application/sql",
	".sh":   "application/x-sh",

	// Documents
	".pdf":  "application/pdf",
	".doc":  "application/msword",
	".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	".xls":  "application/vnd.ms-excel",
	".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	".ppt":  "application/vnd.ms-powerpoint",
	".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",

	// Images
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".gif":  "image/gif",
	".svg":  "image/svg+xml",
	".webp": "image/webp",

	// Archives
	".zip": "application/zip",
	".tar": "application/x-tar",
	".gz":  "application/gzip",
}

// getMIMEType guesses MIME type from file extension.
// Dropbox API doesn't always provide MIME type so we infer from extension.
// Use getMIMETypeWithContent when content is available for better detection.
func getMIMEType(filename string) string {
	ext := strings.ToLower(path.Ext(filename))
	if mimeType, ok := mimeTypes[ext]; ok {
		return mimeType
	}
	return "application/octet-stream"
}

// getMIMETypeWithContent guesses MIME type, using content detection as fallback.
// This provides better accuracy for files without extensions or with unknown extensions.
func getMIMETypeWithContent(filename string, content []byte) string {
	// First try extension-based lookup
	ext := strings.ToLower(path.Ext(filename))
	if mimeType, ok := mimeTypes[ext]; ok {
		return mimeType
	}

	// No extension match - try content detection if we have content
	if len(content) > 0 {
		detected := http.DetectContentType(content)
		return detected
	}

	return "application/octet-stream"
}

// MaxContentSize is the maximum file size to download (5MB).
const MaxContentSize = 5 * 1024 * 1024
