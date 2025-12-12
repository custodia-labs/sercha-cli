package calendar

import (
	"strconv"
	"strings"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

// Config holds Microsoft Calendar connector configuration.
type Config struct {
	// CalendarIDs limits syncing to specific calendars (optional).
	// If empty, syncs from the default calendar.
	CalendarIDs []string
	// MaxResults is the page size for API requests.
	MaxResults int64
	// ShowCancelled includes cancelled events.
	ShowCancelled bool
	// SingleEvents expands recurring events into instances.
	SingleEvents bool
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		MaxResults:    100,
		ShowCancelled: false,
		SingleEvents:  true,
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

	// Parse show_cancelled
	if val := source.Config["show_cancelled"]; val != "" {
		cfg.ShowCancelled = val == "true" || val == "1"
	}

	// Parse single_events
	if val := source.Config["single_events"]; val != "" {
		cfg.SingleEvents = val == "true" || val == "1"
	}

	return cfg, nil
}
