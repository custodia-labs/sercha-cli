package onedrive

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDriveItem_IsFolder(t *testing.T) {
	tests := []struct {
		name     string
		item     *DriveItem
		expected bool
	}{
		{
			name:     "file item",
			item:     &DriveItem{ID: "file-1", File: &FileInfo{MIMEType: "text/plain"}},
			expected: false,
		},
		{
			name:     "folder item",
			item:     &DriveItem{ID: "folder-1", Folder: &FolderInfo{ChildCount: 5}},
			expected: true,
		},
		{
			name:     "neither file nor folder",
			item:     &DriveItem{ID: "item-1"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.item.IsFolder())
		})
	}
}

func TestDriveItem_IsDeleted(t *testing.T) {
	tests := []struct {
		name     string
		item     *DriveItem
		expected bool
	}{
		{
			name:     "not deleted",
			item:     &DriveItem{ID: "file-1"},
			expected: false,
		},
		{
			name:     "deleted",
			item:     &DriveItem{ID: "file-1", Deleted: &DeletedInfo{State: "deleted"}},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.item.IsDeleted())
		})
	}
}

func TestDriveItem_GetMIMEType(t *testing.T) {
	tests := []struct {
		name     string
		item     *DriveItem
		expected string
	}{
		{
			name:     "file with MIME type",
			item:     &DriveItem{File: &FileInfo{MIMEType: "text/plain"}},
			expected: "text/plain",
		},
		{
			name:     "folder",
			item:     &DriveItem{Folder: &FolderInfo{}},
			expected: "application/vnd.ms-folder",
		},
		{
			name:     "file without MIME type",
			item:     &DriveItem{File: &FileInfo{}},
			expected: "application/octet-stream",
		},
		{
			name:     "neither",
			item:     &DriveItem{},
			expected: "application/octet-stream",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.item.GetMIMEType())
		})
	}
}

func TestDriveItem_GetPath(t *testing.T) {
	tests := []struct {
		name     string
		item     *DriveItem
		expected string
	}{
		{
			name:     "root item",
			item:     &DriveItem{Name: "test.txt"},
			expected: "/test.txt",
		},
		{
			name: "nested item",
			item: &DriveItem{
				Name: "test.txt",
				ParentReference: &ParentReference{
					Path: "/drive/root:/Documents",
				},
			},
			expected: "/drive/root:/Documents/test.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.item.GetPath())
		})
	}
}

func TestFileToRawDocument(t *testing.T) {
	item := &DriveItem{
		ID:               "file-123",
		Name:             "document.txt",
		Size:             1024,
		WebURL:           "https://onedrive.live.com/view?id=file-123",
		CreatedDateTime:  "2024-01-15T10:00:00Z",
		ModifiedDateTime: "2024-01-15T12:30:00Z",
		File:             &FileInfo{MIMEType: "text/plain"},
		ParentReference: &ParentReference{
			ID:        "parent-456",
			DriveID:   "drive-789",
			DriveType: "personal",
			Path:      "/drive/root:/Documents",
		},
	}

	content := []byte("Hello, World!")
	doc := FileToRawDocument(item, content, "source-abc")

	assert.Equal(t, "source-abc", doc.SourceID)
	assert.Equal(t, "onedrive://files/file-123", doc.URI)
	assert.Equal(t, "text/plain", doc.MIMEType)
	assert.Equal(t, content, doc.Content)

	// Check metadata
	assert.Equal(t, "file-123", doc.Metadata["file_id"])
	assert.Equal(t, "document.txt", doc.Metadata["title"])
	assert.Equal(t, "/drive/root:/Documents/document.txt", doc.Metadata["path"])
	assert.Equal(t, int64(1024), doc.Metadata["size"])
	assert.Equal(t, "https://onedrive.live.com/view?id=file-123", doc.Metadata["web_link"])
	assert.Equal(t, "2024-01-15T12:30:00Z", doc.Metadata["modified_time"])
	assert.Equal(t, "2024-01-15T10:00:00Z", doc.Metadata["created_time"])
	assert.Equal(t, "parent-456", doc.Metadata["parent_id"])
	assert.Equal(t, "drive-789", doc.Metadata["drive_id"])
	assert.Equal(t, "personal", doc.Metadata["drive_type"])

	// Check parent URI
	assert.NotNil(t, doc.ParentURI)
	assert.Equal(t, "onedrive://folders/parent-456", *doc.ParentURI)
}

func TestFileToRawDocument_NoParent(t *testing.T) {
	item := &DriveItem{
		ID:   "file-123",
		Name: "root-file.txt",
		File: &FileInfo{MIMEType: "text/plain"},
	}

	doc := FileToRawDocument(item, nil, "source-abc")

	assert.Nil(t, doc.ParentURI)
}

func TestShouldSyncFile(t *testing.T) {
	cfg := DefaultConfig()

	tests := []struct {
		name     string
		item     *DriveItem
		expected bool
	}{
		{
			name:     "nil item",
			item:     nil,
			expected: false,
		},
		{
			name:     "folder",
			item:     &DriveItem{Folder: &FolderInfo{}},
			expected: false,
		},
		{
			name:     "deleted item",
			item:     &DriveItem{Deleted: &DeletedInfo{State: "deleted"}},
			expected: false,
		},
		{
			name:     "valid file",
			item:     &DriveItem{ID: "file-1", File: &FileInfo{MIMEType: "text/plain"}},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldSyncFile(tt.item, cfg)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestShouldSyncFile_WithMimeFilter(t *testing.T) {
	cfg := &Config{
		MimeTypeFilter: []string{"text/plain", "application/json"},
	}

	tests := []struct {
		name     string
		mimeType string
		expected bool
	}{
		{"matching exact", "text/plain", true},
		{"matching json", "application/json", true},
		{"non-matching", "image/png", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := &DriveItem{
				ID:   "file-1",
				File: &FileInfo{MIMEType: tt.mimeType},
			}
			result := ShouldSyncFile(item, cfg)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsItemRemoved(t *testing.T) {
	tests := []struct {
		name     string
		item     *DriveItemWithRemoved
		expected bool
	}{
		{
			name: "not removed",
			item: &DriveItemWithRemoved{
				DriveItem: DriveItem{ID: "file-1"},
			},
			expected: false,
		},
		{
			name: "removed",
			item: &DriveItemWithRemoved{
				DriveItem: DriveItem{ID: "file-1"},
				Deleted:   &DeletedInfo{State: "deleted"},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsItemRemoved(tt.item)
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
		{"text/plain", "text/plain", true},
		{"text/html", "text/html", true},
		{"application/json", "application/json", true},
		{"application/xml", "application/xml", true},
		{"application/javascript", "application/javascript", true},
		{"application/x-yaml", "application/x-yaml", true},
		{"application/x-sh", "application/x-sh", true},
		{"application/sql", "application/sql", true},
		{"application/pdf", "application/pdf", true},
		{"image/png", "image/png", false},
		{"application/octet-stream", "application/octet-stream", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldDownloadContent(tt.mimeType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildParentURI(t *testing.T) {
	tests := []struct {
		name      string
		item      *DriveItem
		expectNil bool
		expected  string
	}{
		{
			name: "has parent",
			item: &DriveItem{
				ParentReference: &ParentReference{ID: "parent-123"},
			},
			expectNil: false,
			expected:  "onedrive://folders/parent-123",
		},
		{
			name:      "no parent reference",
			item:      &DriveItem{},
			expectNil: true,
		},
		{
			name: "empty parent ID",
			item: &DriveItem{
				ParentReference: &ParentReference{ID: ""},
			},
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildParentURI(tt.item)

			if tt.expectNil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				assert.Equal(t, tt.expected, *result)
			}
		})
	}
}
