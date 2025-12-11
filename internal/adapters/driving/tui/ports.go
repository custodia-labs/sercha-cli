// Package tui provides an interactive terminal user interface for sercha.
// It implements a driving adapter following hexagonal architecture principles.
package tui

import (
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driving"
)

// Ports aggregates all driving port interfaces required by the TUI.
// This provides a single injection point for dependency injection.
type Ports struct {
	// Search provides search capabilities.
	Search driving.SearchService

	// Source manages source configurations.
	Source driving.SourceService

	// Sync orchestrates document synchronisation.
	Sync driving.SyncOrchestrator

	// ResultAction provides actions on search results.
	ResultAction driving.ResultActionService

	// Document manages documents within sources.
	Document driving.DocumentService

	// ConnectorRegistry provides available connector types.
	ConnectorRegistry driving.ConnectorRegistry

	// ProviderRegistry provides provider-connector compatibility info.
	ProviderRegistry driving.ProviderRegistry

	// Settings manages application settings.
	Settings driving.SettingsService

	// Credentials manages user credentials (tokens and account identifiers).
	Credentials driving.CredentialsService

	// AuthProvider manages OAuth app configurations (reusable across sources).
	AuthProvider driving.AuthProviderService
}

// NewPorts creates a new Ports aggregate with the given services.
func NewPorts(
	search driving.SearchService,
	source driving.SourceService,
	sync driving.SyncOrchestrator,
	resultAction driving.ResultActionService,
) *Ports {
	return &Ports{
		Search:       search,
		Source:       source,
		Sync:         sync,
		ResultAction: resultAction,
	}
}

// Validate ensures all required ports are set.
// Returns an error if any port is nil.
func (p *Ports) Validate() error {
	if p.Search == nil {
		return ErrMissingSearchService
	}
	if p.Source == nil {
		return ErrMissingSourceService
	}
	if p.Sync == nil {
		return ErrMissingSyncOrchestrator
	}
	return nil
}
