package notion

import (
	"strconv"
	"strings"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

// Config holds Notion connector configuration.
type Config struct {
	// IncludeComments enables fetching page comments (additional API calls).
	IncludeComments bool
	// ContentTypes specifies what to sync: "pages", "databases", or both.
	ContentTypes []string
	// MaxBlockDepth limits recursive block fetching (default: 10).
	MaxBlockDepth int
	// PageSize is the number of items per API page (max: 100).
	PageSize int
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		IncludeComments: true,
		ContentTypes:    []string{"pages", "databases"},
		MaxBlockDepth:   10,
		PageSize:        100,
	}
}

// ParseConfig extracts configuration from a Source.
func ParseConfig(source domain.Source) (*Config, error) {
	cfg := DefaultConfig()

	// Parse include_comments
	if val := source.Config["include_comments"]; val != "" {
		cfg.IncludeComments = val == "true" || val == "1"
	}

	// Parse content_types
	if val := source.Config["content_types"]; val != "" {
		cfg.ContentTypes = strings.Split(val, ",")
		for i := range cfg.ContentTypes {
			cfg.ContentTypes[i] = strings.TrimSpace(cfg.ContentTypes[i])
		}
	}

	// Parse max_block_depth
	if val := source.Config["max_block_depth"]; val != "" {
		if n, err := strconv.Atoi(val); err == nil && n > 0 {
			cfg.MaxBlockDepth = n
		}
	}

	// Parse page_size
	if val := source.Config["page_size"]; val != "" {
		if n, err := strconv.Atoi(val); err == nil && n > 0 && n <= 100 {
			cfg.PageSize = n
		}
	}

	return cfg, nil
}

// ShouldSyncPages returns true if pages should be synced.
func (c *Config) ShouldSyncPages() bool {
	for _, ct := range c.ContentTypes {
		if ct == "pages" {
			return true
		}
	}
	return false
}

// ShouldSyncDatabases returns true if databases should be synced.
func (c *Config) ShouldSyncDatabases() bool {
	for _, ct := range c.ContentTypes {
		if ct == "databases" {
			return true
		}
	}
	return false
}
