package tui

import "errors"

// ErrMissingSearchService is returned when the search service is not provided.
var ErrMissingSearchService = errors.New("tui: search service is required")

// ErrMissingSourceService is returned when the source service is not provided.
var ErrMissingSourceService = errors.New("tui: source service is required")

// ErrMissingSyncOrchestrator is returned when the sync orchestrator is not provided.
var ErrMissingSyncOrchestrator = errors.New("tui: sync orchestrator is required")

// ErrInvalidPorts is returned when ports validation fails.
var ErrInvalidPorts = errors.New("tui: invalid ports configuration")
