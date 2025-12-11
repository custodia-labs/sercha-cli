package drive

import (
	"strconv"
	"strings"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

// ContentType identifies what content to sync from Google Drive.
type ContentType string

const (
	// ContentFiles syncs regular files.
	ContentFiles ContentType = "files"
	// ContentDocs syncs Google Docs (exported to text).
	ContentDocs ContentType = "docs"
	// ContentSheets syncs Google Sheets (exported to CSV text).
	ContentSheets ContentType = "sheets"
)

// DefaultContentTypes are the content types synced by default.
var DefaultContentTypes = []ContentType{ContentFiles, ContentDocs, ContentSheets}

// Config holds Google Drive connector configuration.
type Config struct {
	// ContentTypes specifies what types of content to sync.
	ContentTypes []ContentType
	// MimeTypeFilter limits syncing to specific MIME types (optional).
	MimeTypeFilter []string
	// FolderIDs limits syncing to specific folders (optional).
	FolderIDs []string
	// MaxResults is the page size for API requests.
	MaxResults int64
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		ContentTypes: DefaultContentTypes,
		MaxResults:   100,
	}
}

// ParseConfig extracts configuration from a Source.
func ParseConfig(source domain.Source) (*Config, error) {
	cfg := DefaultConfig()

	// Parse content_types from source config
	if val := source.Config["content_types"]; val != "" {
		types := strings.Split(val, ",")
		cfg.ContentTypes = make([]ContentType, 0, len(types))
		for _, t := range types {
			ct := ContentType(strings.TrimSpace(t))
			if isValidContentType(ct) {
				cfg.ContentTypes = append(cfg.ContentTypes, ct)
			}
		}
	}

	// Parse mime_types filter
	if val := source.Config["mime_types"]; val != "" {
		cfg.MimeTypeFilter = strings.Split(val, ",")
		for i := range cfg.MimeTypeFilter {
			cfg.MimeTypeFilter[i] = strings.TrimSpace(cfg.MimeTypeFilter[i])
		}
	}

	// Parse folder_ids
	if val := source.Config["folder_ids"]; val != "" {
		cfg.FolderIDs = strings.Split(val, ",")
		for i := range cfg.FolderIDs {
			cfg.FolderIDs[i] = strings.TrimSpace(cfg.FolderIDs[i])
		}
	}

	// Parse max_results
	if val := source.Config["max_results"]; val != "" {
		if n, err := strconv.ParseInt(val, 10, 64); err == nil && n > 0 {
			cfg.MaxResults = n
		}
	}

	return cfg, nil
}

// HasContentType checks if a content type is enabled.
func (c *Config) HasContentType(ct ContentType) bool {
	for _, t := range c.ContentTypes {
		if t == ct {
			return true
		}
	}
	return false
}

func isValidContentType(ct ContentType) bool {
	switch ct {
	case ContentFiles, ContentDocs, ContentSheets:
		return true
	default:
		return false
	}
}
