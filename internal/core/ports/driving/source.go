package driving

import (
	"context"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

// SourceService manages source configurations.
type SourceService interface {
	// Add creates a new source configuration.
	Add(ctx context.Context, source domain.Source) error

	// Get retrieves a source by ID.
	Get(ctx context.Context, id string) (*domain.Source, error)

	// List returns all configured sources.
	List(ctx context.Context) ([]domain.Source, error)

	// Update modifies an existing source configuration.
	Update(ctx context.Context, source domain.Source) error

	// Remove deletes a source and its indexed data.
	Remove(ctx context.Context, id string) error

	// ValidateConfig validates source configuration for a connector type.
	// Returns an error if required fields are missing or invalid.
	ValidateConfig(ctx context.Context, connectorType string, config map[string]string) error
}
