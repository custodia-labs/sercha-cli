package onedrive

import (
	"strconv"
	"strings"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

// Config holds OneDrive connector configuration.
type Config struct {
	// FolderIDs limits syncing to specific folders (optional).
	// If empty, syncs from root.
	FolderIDs []string
	// MimeTypeFilter limits syncing to specific MIME types (optional).
	MimeTypeFilter []string
	// MaxResults is the page size for API requests.
	MaxResults int64
	// IncludeSharedWithMe includes files shared with the user.
	IncludeSharedWithMe bool
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		MaxResults:          100,
		IncludeSharedWithMe: false,
	}
}

// ParseConfig extracts configuration from a Source.
func ParseConfig(source domain.Source) (*Config, error) {
	cfg := DefaultConfig()

	// Parse folder_ids
	if val := source.Config["folder_ids"]; val != "" {
		cfg.FolderIDs = strings.Split(val, ",")
		for i := range cfg.FolderIDs {
			cfg.FolderIDs[i] = strings.TrimSpace(cfg.FolderIDs[i])
		}
	}

	// Parse mime_types filter
	if val := source.Config["mime_types"]; val != "" {
		cfg.MimeTypeFilter = strings.Split(val, ",")
		for i := range cfg.MimeTypeFilter {
			cfg.MimeTypeFilter[i] = strings.TrimSpace(cfg.MimeTypeFilter[i])
		}
	}

	// Parse max_results
	if val := source.Config["max_results"]; val != "" {
		if n, err := strconv.ParseInt(val, 10, 64); err == nil && n > 0 {
			cfg.MaxResults = n
		}
	}

	// Parse include_shared
	if val := source.Config["include_shared"]; val != "" {
		cfg.IncludeSharedWithMe = val == "true" || val == "1"
	}

	return cfg, nil
}
