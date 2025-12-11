package calendar

import (
	"strconv"
	"strings"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

// Config holds Google Calendar connector configuration.
type Config struct {
	// CalendarIDs limits syncing to specific calendars (optional).
	// If empty, syncs all calendars the user can access.
	CalendarIDs []string
	// MaxResults is the page size for API requests.
	MaxResults int64
	// ShowDeleted includes deleted events if true.
	ShowDeleted bool
	// SingleEvents expands recurring events into instances.
	SingleEvents bool
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		MaxResults:   250,
		ShowDeleted:  true, // Need this for incremental sync to detect deletions
		SingleEvents: true, // Expand recurring events for easier indexing
	}
}

// ParseConfig extracts configuration from a Source.
func ParseConfig(source domain.Source) (*Config, error) {
	cfg := DefaultConfig()

	// Parse calendar_ids
	if val := source.Config["calendar_ids"]; val != "" {
		cfg.CalendarIDs = strings.Split(val, ",")
		for i := range cfg.CalendarIDs {
			cfg.CalendarIDs[i] = strings.TrimSpace(cfg.CalendarIDs[i])
		}
	}

	// Parse max_results
	if val := source.Config["max_results"]; val != "" {
		if n, err := strconv.ParseInt(val, 10, 64); err == nil && n > 0 {
			cfg.MaxResults = n
		}
	}

	// Parse show_deleted
	if val := source.Config["show_deleted"]; val == "false" {
		cfg.ShowDeleted = false
	}

	// Parse single_events
	if val := source.Config["single_events"]; val == "false" {
		cfg.SingleEvents = false
	}

	return cfg, nil
}
