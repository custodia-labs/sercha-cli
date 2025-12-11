package gmail

import (
	"strconv"
	"strings"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

// LabelFilter identifies which emails to sync.
type LabelFilter string

const (
	// LabelInbox syncs emails from INBOX.
	LabelInbox LabelFilter = "INBOX"
	// LabelSent syncs sent emails.
	LabelSent LabelFilter = "SENT"
	// LabelAll syncs all emails.
	LabelAll LabelFilter = ""
)

// Config holds Gmail connector configuration.
type Config struct {
	// LabelIDs limits syncing to specific label IDs (optional).
	// If empty, syncs INBOX by default.
	LabelIDs []string
	// Query is a Gmail search query (optional).
	Query string
	// MaxResults is the page size for API requests.
	MaxResults int64
	// IncludeSpamTrash includes spam and trash if true.
	IncludeSpamTrash bool
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		LabelIDs:   []string{"INBOX"},
		MaxResults: 100,
	}
}

// ParseConfig extracts configuration from a Source.
func ParseConfig(source domain.Source) (*Config, error) {
	cfg := DefaultConfig()

	// Parse label_ids
	if val := source.Config["label_ids"]; val != "" {
		cfg.LabelIDs = strings.Split(val, ",")
		for i := range cfg.LabelIDs {
			cfg.LabelIDs[i] = strings.TrimSpace(cfg.LabelIDs[i])
		}
	}

	// Parse query
	if val := source.Config["query"]; val != "" {
		cfg.Query = val
	}

	// Parse max_results
	if val := source.Config["max_results"]; val != "" {
		if n, err := strconv.ParseInt(val, 10, 64); err == nil && n > 0 {
			cfg.MaxResults = n
		}
	}

	// Parse include_spam_trash
	if val := source.Config["include_spam_trash"]; val == "true" {
		cfg.IncludeSpamTrash = true
	}

	return cfg, nil
}
