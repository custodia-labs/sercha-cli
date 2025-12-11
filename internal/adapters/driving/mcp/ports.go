package mcp

import (
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driving"
)

// Ports aggregates all driving port interfaces required by the MCP server.
// This provides a single injection point for dependency injection.
type Ports struct {
	// Search provides search capabilities.
	Search driving.SearchService

	// Source manages source configurations.
	Source driving.SourceService

	// Document manages documents within sources.
	Document driving.DocumentService
}

// Validate ensures all required ports are set.
// Returns an error if any required port is nil.
func (p *Ports) Validate() error {
	if p.Search == nil {
		return ErrMissingSearchService
	}
	// Source and Document are optional for MVP
	return nil
}
