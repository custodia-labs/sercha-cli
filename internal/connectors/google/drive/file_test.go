package drive

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/api/drive/v3"
)

func TestShouldSyncFile(t *testing.T) {
	tests := []struct {
		name     string
		file     *drive.File
		config   *Config
		expected bool
	}{
		{
			name: "regular file with files content type",
			file: &drive.File{
				MimeType: "text/plain",
				Trashed:  false,
			},
			config: &Config{
				ContentTypes: []ContentType{ContentFiles},
			},
			expected: true,
		},
		{
			name: "folder is skipped",
			file: &drive.File{
				MimeType: MimeTypeFolder,
				Trashed:  false,
			},
			config: &Config{
				ContentTypes: []ContentType{ContentFiles},
			},
			expected: false,
		},
		{
			name: "trashed file is skipped",
			file: &drive.File{
				MimeType: "text/plain",
				Trashed:  true,
			},
			config: &Config{
				ContentTypes: []ContentType{ContentFiles},
			},
			expected: false,
		},
		{
			name: "Google Doc with docs content type",
			file: &drive.File{
				MimeType: MimeTypeGoogleDoc,
				Trashed:  false,
			},
			config: &Config{
				ContentTypes: []ContentType{ContentDocs},
			},
			expected: true,
		},
		{
			name: "Google Doc without docs content type",
			file: &drive.File{
				MimeType: MimeTypeGoogleDoc,
				Trashed:  false,
			},
			config: &Config{
				ContentTypes: []ContentType{ContentFiles},
			},
			expected: false,
		},
		{
			name: "Google Sheet with sheets content type",
			file: &drive.File{
				MimeType: MimeTypeGoogleSheet,
				Trashed:  false,
			},
			config: &Config{
				ContentTypes: []ContentType{ContentSheets},
			},
			expected: true,
		},
		{
			name: "Google Sheet without sheets content type",
			file: &drive.File{
				MimeType: MimeTypeGoogleSheet,
				Trashed:  false,
			},
			config: &Config{
				ContentTypes: []ContentType{ContentFiles, ContentDocs},
			},
			expected: false,
		},
		{
			name: "file matches MIME type filter",
			file: &drive.File{
				MimeType: "application/pdf",
				Trashed:  false,
			},
			config: &Config{
				ContentTypes:   []ContentType{ContentFiles},
				MimeTypeFilter: []string{"application/pdf", "text/plain"},
			},
			expected: true,
		},
		{
			name: "file does not match MIME type filter",
			file: &drive.File{
				MimeType: "application/pdf",
				Trashed:  false,
			},
			config: &Config{
				ContentTypes:   []ContentType{ContentFiles},
				MimeTypeFilter: []string{"text/plain", "text/markdown"},
			},
			expected: false,
		},
		{
			name: "no MIME type filter allows all",
			file: &drive.File{
				MimeType: "application/octet-stream",
				Trashed:  false,
			},
			config: &Config{
				ContentTypes:   []ContentType{ContentFiles},
				MimeTypeFilter: []string{},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldSyncFile(tt.file, tt.config)
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
		{
			name:     "text/plain",
			mimeType: "text/plain",
			expected: true,
		},
		{
			name:     "text/html",
			mimeType: "text/html",
			expected: true,
		},
		{
			name:     "text/markdown",
			mimeType: "text/markdown",
			expected: true,
		},
		{
			name:     "application/json",
			mimeType: "application/json",
			expected: true,
		},
		{
			name:     "application/xml",
			mimeType: "application/xml",
			expected: true,
		},
		{
			name:     "application/javascript",
			mimeType: "application/javascript",
			expected: true,
		},
		{
			name:     "application/x-yaml",
			mimeType: "application/x-yaml",
			expected: true,
		},
		{
			name:     "application/x-sh",
			mimeType: "application/x-sh",
			expected: true,
		},
		{
			name:     "application/sql",
			mimeType: "application/sql",
			expected: true,
		},
		{
			name:     "application/pdf is downloadable",
			mimeType: "application/pdf",
			expected: true,
		},
		{
			name:     "image/png is not downloadable",
			mimeType: "image/png",
			expected: false,
		},
		{
			name:     "application/octet-stream is not downloadable",
			mimeType: "application/octet-stream",
			expected: false,
		},
		{
			name:     "video/mp4 is not downloadable",
			mimeType: "video/mp4",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldDownloadContent(tt.mimeType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildFilePath(t *testing.T) {
	tests := []struct {
		name     string
		file     *drive.File
		expected string
	}{
		{
			name: "file with parent",
			file: &drive.File{
				Name:    "document.txt",
				Parents: []string{"parent-folder-id"},
			},
			expected: "/parent-folder-id/document.txt",
		},
		{
			name: "file without parent",
			file: &drive.File{
				Name:    "document.txt",
				Parents: []string{},
			},
			expected: "/document.txt",
		},
		{
			name: "file with multiple parents uses first",
			file: &drive.File{
				Name:    "document.txt",
				Parents: []string{"parent-1", "parent-2"},
			},
			expected: "/parent-1/document.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildFilePath(tt.file)
			assert.Equal(t, tt.expected, result)
		})
	}
}
