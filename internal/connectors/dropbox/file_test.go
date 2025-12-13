package dropbox

import (
	"testing"
	"time"

	"github.com/dropbox/dropbox-sdk-go-unofficial/v6/dropbox/files"
	"github.com/stretchr/testify/assert"
)

// newTestFileMetadata creates a FileMetadata for testing with embedded Metadata fields.
func newTestFileMetadata(id, name, pathDisplay, pathLower string, size uint64, serverMod time.Time) *files.FileMetadata {
	fm := &files.FileMetadata{
		Id:             id,
		Size:           size,
		ServerModified: serverMod,
	}
	fm.Name = name
	fm.PathDisplay = pathDisplay
	fm.PathLower = pathLower
	return fm
}

func TestFileToRawDocument(t *testing.T) {
	modTime := time.Date(2024, 1, 15, 12, 30, 0, 0, time.UTC)
	file := newTestFileMetadata(
		"id:abc123def456",
		"document.pdf",
		"/Documents/Work/document.pdf",
		"/documents/work/document.pdf",
		1024,
		modTime,
	)
	file.Rev = "rev123"
	file.ContentHash = "hash456"

	content := []byte("Hello, World!")
	doc := FileToRawDocument(file, content, "source-abc")

	assert.Equal(t, "source-abc", doc.SourceID)
	assert.Equal(t, "dropbox://files/id:abc123def456", doc.URI)
	assert.Equal(t, "application/pdf", doc.MIMEType)
	assert.Equal(t, content, doc.Content)

	// Check metadata
	assert.Equal(t, "id:abc123def456", doc.Metadata["file_id"])
	assert.Equal(t, "document.pdf", doc.Metadata["title"])
	assert.Equal(t, "/Documents/Work/document.pdf", doc.Metadata["path"])
	assert.Equal(t, uint64(1024), doc.Metadata["size"])
	assert.Equal(t, "2024-01-15T12:30:00Z", doc.Metadata["modified_time"])
	assert.Equal(t, "rev123", doc.Metadata["rev"])
	assert.Equal(t, "hash456", doc.Metadata["content_hash"])

	// Check parent URI
	assert.NotNil(t, doc.ParentURI)
	assert.Equal(t, "dropbox://folders/Documents/Work", *doc.ParentURI)
}

func TestFileToRawDocument_NoParent(t *testing.T) {
	modTime := time.Now()
	file := newTestFileMetadata(
		"id:root-file",
		"root-file.txt",
		"/root-file.txt",
		"/root-file.txt",
		0,
		modTime,
	)

	doc := FileToRawDocument(file, nil, "source-abc")

	// Root level file has no parent
	assert.Nil(t, doc.ParentURI)
}

func TestFileToRawDocument_NilContent(t *testing.T) {
	modTime := time.Now()
	file := newTestFileMetadata(
		"id:large-file",
		"large.zip",
		"/large.zip",
		"/large.zip",
		0,
		modTime,
	)

	doc := FileToRawDocument(file, nil, "source-abc")

	assert.Nil(t, doc.Content)
	assert.Equal(t, "application/zip", doc.MIMEType)
}

func TestBuildParentURI(t *testing.T) {
	tests := []struct {
		name        string
		pathDisplay string
		expectNil   bool
		expected    string
	}{
		{
			name:        "nested file",
			pathDisplay: "/Documents/Work/file.txt",
			expectNil:   false,
			expected:    "dropbox://folders/Documents/Work",
		},
		{
			name:        "single folder depth",
			pathDisplay: "/Documents/file.txt",
			expectNil:   false,
			expected:    "dropbox://folders/Documents",
		},
		{
			name:        "root level file",
			pathDisplay: "/file.txt",
			expectNil:   true,
		},
		{
			name:        "empty path",
			pathDisplay: "",
			expectNil:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := &files.FileMetadata{}
			file.PathDisplay = tt.pathDisplay
			result := buildParentURI(file)

			if tt.expectNil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				assert.Equal(t, tt.expected, *result)
			}
		})
	}
}

func TestShouldSyncFile(t *testing.T) {
	cfg := DefaultConfig()

	t.Run("nil file", func(t *testing.T) {
		result := ShouldSyncFile(nil, cfg)
		assert.False(t, result)
	})

	t.Run("valid file", func(t *testing.T) {
		file := newTestFileMetadata("id:test", "test.txt", "/test.txt", "/test.txt", 0, time.Now())
		result := ShouldSyncFile(file, cfg)
		assert.True(t, result)
	})
}

func TestShouldSyncFile_WithMimeFilter(t *testing.T) {
	cfg := &Config{
		MimeTypeFilter: []string{"text/plain", "application/pdf"},
	}

	tests := []struct {
		name     string
		filename string
		expected bool
	}{
		{"matching text file", "test.txt", true},
		{"matching pdf", "document.pdf", true},
		{"non-matching image", "photo.png", false},
		{"non-matching archive", "data.zip", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := newTestFileMetadata("id:test", tt.filename, "/"+tt.filename, "/"+tt.filename, 0, time.Now())
			result := ShouldSyncFile(file, cfg)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestShouldSyncFile_MimeFilterPrefix(t *testing.T) {
	cfg := &Config{
		MimeTypeFilter: []string{"text/"}, // Should match all text/* types
	}

	tests := []struct {
		name     string
		filename string
		expected bool
	}{
		{"plain text", "test.txt", true},
		{"markdown", "readme.md", true},
		{"html", "page.html", true},
		{"css", "styles.css", true},
		{"pdf", "document.pdf", false},
		{"image", "photo.png", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := newTestFileMetadata("id:test", tt.filename, "/"+tt.filename, "/"+tt.filename, 0, time.Now())
			result := ShouldSyncFile(file, cfg)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestShouldDownloadContent(t *testing.T) {
	tests := []struct {
		name     string
		mimeType string
		expected bool
	}{
		// Text types (should download)
		{"text/plain", "text/plain", true},
		{"text/html", "text/html", true},
		{"text/css", "text/css", true},
		{"text/csv", "text/csv", true},
		{"text/markdown", "text/markdown", true},
		{"text/x-python", "text/x-python", true},

		// Application types that should download
		{"application/json", "application/json", true},
		{"application/xml", "application/xml", true},
		{"application/javascript", "application/javascript", true},
		{"application/x-yaml", "application/x-yaml", true},
		{"application/x-sh", "application/x-sh", true},
		{"application/sql", "application/sql", true},
		{"application/pdf", "application/pdf", true},

		// Binary types (should not download)
		{"image/png", "image/png", false},
		{"image/jpeg", "image/jpeg", false},
		{"application/zip", "application/zip", false},
		{"application/octet-stream", "application/octet-stream", false},
		{"application/vnd.ms-excel", "application/vnd.ms-excel", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldDownloadContent(tt.mimeType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetMIMEType(t *testing.T) {
	tests := []struct {
		filename string
		expected string
	}{
		// Text files
		{"document.txt", "text/plain"},
		{"readme.md", "text/markdown"},
		{"page.html", "text/html"},
		{"page.htm", "text/html"},
		{"styles.css", "text/css"},
		{"data.csv", "text/csv"},
		{"config.xml", "application/xml"},

		// Code files
		{"script.js", "application/javascript"},
		{"app.ts", "application/typescript"},
		{"config.json", "application/json"},
		{"settings.yaml", "application/x-yaml"},
		{"settings.yml", "application/x-yaml"},
		{"main.py", "text/x-python"},
		{"main.go", "text/x-go"},
		{"Main.java", "text/x-java"},
		{"main.c", "text/x-c"},
		{"main.cpp", "text/x-c++"},
		{"header.h", "text/x-c"},
		{"header.hpp", "text/x-c++"},
		{"main.rs", "text/x-rust"},
		{"app.rb", "text/x-ruby"},
		{"index.php", "text/x-php"},
		{"query.sql", "application/sql"},
		{"script.sh", "application/x-sh"},

		// Documents
		{"document.pdf", "application/pdf"},
		{"document.doc", "application/msword"},
		{"document.docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
		{"spreadsheet.xls", "application/vnd.ms-excel"},
		{"spreadsheet.xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"},
		{"presentation.ppt", "application/vnd.ms-powerpoint"},
		{"presentation.pptx", "application/vnd.openxmlformats-officedocument.presentationml.presentation"},

		// Images
		{"photo.jpg", "image/jpeg"},
		{"photo.jpeg", "image/jpeg"},
		{"image.png", "image/png"},
		{"animation.gif", "image/gif"},
		{"icon.svg", "image/svg+xml"},
		{"photo.webp", "image/webp"},

		// Archives
		{"archive.zip", "application/zip"},
		{"archive.tar", "application/x-tar"},
		{"archive.gz", "application/gzip"},

		// Unknown
		{"file.unknown", "application/octet-stream"},
		{"noextension", "application/octet-stream"},
		{"FILE.TXT", "text/plain"}, // Case insensitive
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := getMIMEType(tt.filename)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMaxContentSize(t *testing.T) {
	// Verify the constant is set correctly (5MB)
	assert.Equal(t, 5*1024*1024, MaxContentSize)
}
