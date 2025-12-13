package dropbox

import (
	"strconv"
	"strings"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

// Config holds Dropbox connector configuration.
type Config struct {
	// FolderPath is the root path to sync (optional, defaults to "" for root).
	FolderPath string
	// MimeTypeFilter limits syncing to specific MIME types (optional).
	MimeTypeFilter []string
	// MaxResults is the page size for API requests.
	MaxResults uint32
	// Recursive includes subfolders (default: true).
	Recursive bool
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		FolderPath: "",
		MaxResults: 100,
		Recursive:  true,
	}
}

// ParseConfig extracts configuration from a Source.
func ParseConfig(source domain.Source) (*Config, error) {
	cfg := DefaultConfig()

	// Parse folder_path
	if val := source.Config["folder_path"]; val != "" {
		// Ensure path starts with / if not empty
		if val != "" && !strings.HasPrefix(val, "/") {
			val = "/" + val
		}
		cfg.FolderPath = val
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
		if n, err := strconv.ParseUint(val, 10, 32); err == nil && n > 0 {
			cfg.MaxResults = uint32(n)
		}
	}

	// Parse recursive
	if val := source.Config["recursive"]; val != "" {
		cfg.Recursive = val == "true" || val == "1"
	}

	return cfg, nil
}
