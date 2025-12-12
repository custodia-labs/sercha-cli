package onedrive

import (
	"fmt"
	"strings"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

// DriveItem represents a OneDrive file or folder from the Graph API.
type DriveItem struct {
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	Size             int64             `json:"size"`
	WebURL           string            `json:"webUrl"`
	CreatedDateTime  string            `json:"createdDateTime"`
	ModifiedDateTime string            `json:"lastModifiedDateTime"`
	File             *FileInfo         `json:"file,omitempty"`
	Folder           *FolderInfo       `json:"folder,omitempty"`
	ParentReference  *ParentReference  `json:"parentReference,omitempty"`
	Deleted          *DeletedInfo      `json:"deleted,omitempty"`
	Content          *DownloadLocation `json:"@microsoft.graph.downloadUrl,omitempty"`
}

// FileInfo contains file-specific metadata.
type FileInfo struct {
	MIMEType string `json:"mimeType"`
	Hashes   *struct {
		QuickXorHash string `json:"quickXorHash"`
		SHA1Hash     string `json:"sha1Hash"`
	} `json:"hashes,omitempty"`
}

// FolderInfo contains folder-specific metadata.
type FolderInfo struct {
	ChildCount int `json:"childCount"`
}

// ParentReference contains parent folder information.
type ParentReference struct {
	DriveID   string `json:"driveId"`
	DriveType string `json:"driveType"`
	ID        string `json:"id"`
	Name      string `json:"name"`
	Path      string `json:"path"`
}

// DeletedInfo indicates the item was deleted.
type DeletedInfo struct {
	State string `json:"state"`
}

// DownloadLocation is a temporary download URL.
type DownloadLocation struct {
	URL string `json:"@microsoft.graph.downloadUrl"`
}

// DriveItemWithRemoved wraps DriveItem with removal detection.
type DriveItemWithRemoved struct {
	DriveItem
	Deleted *DeletedInfo `json:"deleted,omitempty"`
}

// IsFolder returns true if the item is a folder.
func (d *DriveItem) IsFolder() bool {
	return d.Folder != nil
}

// IsDeleted returns true if the item was deleted.
func (d *DriveItem) IsDeleted() bool {
	return d.Deleted != nil
}

// GetMIMEType returns the file's MIME type.
func (d *DriveItem) GetMIMEType() string {
	if d.File != nil && d.File.MIMEType != "" {
		return d.File.MIMEType
	}
	if d.IsFolder() {
		return "application/vnd.ms-folder"
	}
	return "application/octet-stream"
}

// GetPath returns the file path.
func (d *DriveItem) GetPath() string {
	if d.ParentReference != nil && d.ParentReference.Path != "" {
		return d.ParentReference.Path + "/" + d.Name
	}
	return "/" + d.Name
}

// FileToRawDocument converts a DriveItem to a RawDocument.
func FileToRawDocument(item *DriveItem, content []byte, sourceID string) *domain.RawDocument {
	path := item.GetPath()

	metadata := map[string]any{
		"file_id":       item.ID,
		"title":         item.Name,
		"path":          path,
		"size":          item.Size,
		"web_link":      item.WebURL,
		"modified_time": item.ModifiedDateTime,
		"created_time":  item.CreatedDateTime,
	}

	if item.ParentReference != nil {
		metadata["parent_id"] = item.ParentReference.ID
		metadata["drive_id"] = item.ParentReference.DriveID
		metadata["drive_type"] = item.ParentReference.DriveType
	}

	parentURI := buildParentURI(item)

	return &domain.RawDocument{
		SourceID:  sourceID,
		URI:       fmt.Sprintf("onedrive://files/%s", item.ID),
		MIMEType:  item.GetMIMEType(),
		Content:   content,
		Metadata:  metadata,
		ParentURI: parentURI,
	}
}

// buildParentURI creates the parent URI for hierarchy tracking.
func buildParentURI(item *DriveItem) *string {
	if item.ParentReference == nil || item.ParentReference.ID == "" {
		return nil
	}
	uri := fmt.Sprintf("onedrive://folders/%s", item.ParentReference.ID)
	return &uri
}

// ShouldSyncFile checks if a file should be synced based on config.
func ShouldSyncFile(item *DriveItem, cfg *Config) bool {
	if item == nil {
		return false
	}

	// Skip folders
	if item.IsFolder() {
		return false
	}

	// Skip deleted items (handled separately)
	if item.IsDeleted() {
		return false
	}

	// Check MIME type filter
	if len(cfg.MimeTypeFilter) > 0 {
		mimeType := item.GetMIMEType()
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

// IsItemRemoved checks if a delta response item was removed.
func IsItemRemoved(item *DriveItemWithRemoved) bool {
	return item.Deleted != nil
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
