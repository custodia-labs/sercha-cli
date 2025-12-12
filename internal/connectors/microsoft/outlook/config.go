package outlook

import (
	"strconv"
	"strings"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

// FolderFilter identifies which emails to sync.
type FolderFilter string

const (
	// FolderInbox syncs emails from Inbox.
	FolderInbox FolderFilter = "inbox"
	// FolderSentItems syncs sent emails.
	FolderSentItems FolderFilter = "sentitems"
	// FolderDrafts syncs draft emails.
	FolderDrafts FolderFilter = "drafts"
	// FolderAll syncs all emails.
	FolderAll FolderFilter = ""
)

// Config holds Outlook connector configuration.
type Config struct {
	// FolderID limits syncing to a specific folder ID (optional).
	// Common values: "inbox", "sentitems", "drafts", or a specific folder ID.
	// If empty, syncs Inbox by default.
	FolderID string
	// MaxResults is the page size for API requests (default: 100, max: 1000).
	MaxResults int64
	// IncludeSpamTrash includes spam and deleted items if true.
	IncludeSpamTrash bool
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		FolderID:   "inbox",
		MaxResults: 100,
	}
}

// ParseConfig extracts configuration from a Source.
func ParseConfig(source domain.Source) (*Config, error) {
	cfg := DefaultConfig()

	// Parse folder_id
	if val := source.Config["folder_id"]; val != "" {
		cfg.FolderID = strings.TrimSpace(val)
	}

	// Parse max_results
	if val := source.Config["max_results"]; val != "" {
		if n, err := strconv.ParseInt(val, 10, 64); err == nil && n > 0 {
			cfg.MaxResults = n
			// Microsoft Graph max is 1000
			if cfg.MaxResults > 1000 {
				cfg.MaxResults = 1000
			}
		}
	}

	// Parse include_spam_trash
	if val := source.Config["include_spam_trash"]; val == "true" {
		cfg.IncludeSpamTrash = true
	}

	return cfg, nil
}
