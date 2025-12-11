package services

import (
	"context"
	"fmt"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driving"
)

// Ensure SourceService implements the interface.
var _ driving.SourceService = (*SourceService)(nil)

// SourceService manages source configurations.
type SourceService struct {
	sourceStore       driven.SourceStore
	syncStore         driven.SyncStateStore
	docStore          driven.DocumentStore
	connectorRegistry driving.ConnectorRegistry
}

// NewSourceService creates a new source service.
func NewSourceService(
	sourceStore driven.SourceStore,
	syncStore driven.SyncStateStore,
	docStore driven.DocumentStore,
) *SourceService {
	return &SourceService{
		sourceStore: sourceStore,
		syncStore:   syncStore,
		docStore:    docStore,
	}
}

// SetConnectorRegistry sets the connector registry for config validation.
func (s *SourceService) SetConnectorRegistry(registry driving.ConnectorRegistry) {
	s.connectorRegistry = registry
}

// Add creates a new source configuration.
func (s *SourceService) Add(ctx context.Context, source domain.Source) error {
	if s.sourceStore == nil {
		return domain.ErrNotImplemented
	}
	if source.ID == "" {
		return domain.ErrInvalidInput
	}
	// Check if already exists
	existing, err := s.sourceStore.Get(ctx, source.ID)
	if err == nil && existing != nil {
		return domain.ErrAlreadyExists
	}
	return s.sourceStore.Save(ctx, source)
}

// Get retrieves a source by ID.
func (s *SourceService) Get(ctx context.Context, id string) (*domain.Source, error) {
	if s.sourceStore == nil {
		return nil, domain.ErrNotImplemented
	}
	return s.sourceStore.Get(ctx, id)
}

// List returns all configured sources.
func (s *SourceService) List(ctx context.Context) ([]domain.Source, error) {
	if s.sourceStore == nil {
		return nil, domain.ErrNotImplemented
	}
	return s.sourceStore.List(ctx)
}

// Update modifies an existing source configuration.
func (s *SourceService) Update(ctx context.Context, source domain.Source) error {
	if s.sourceStore == nil {
		return domain.ErrNotImplemented
	}
	if source.ID == "" {
		return domain.ErrInvalidInput
	}
	// Verify source exists
	_, err := s.sourceStore.Get(ctx, source.ID)
	if err != nil {
		return domain.ErrNotFound
	}
	return s.sourceStore.Save(ctx, source)
}

// Remove deletes a source and its indexed data.
func (s *SourceService) Remove(ctx context.Context, id string) error {
	if s.sourceStore == nil {
		return domain.ErrNotImplemented
	}
	// Cleanup: delete documents, sync state, then source
	if s.docStore != nil {
		docs, err := s.docStore.ListDocuments(ctx, id)
		if err == nil {
			for i := range docs {
				//nolint:errcheck // Intentionally ignore errors to continue cleanup
				_ = s.docStore.DeleteDocument(ctx, docs[i].ID)
			}
		}
	}
	if s.syncStore != nil {
		//nolint:errcheck // Intentionally ignore errors to continue cleanup
		_ = s.syncStore.Delete(ctx, id)
	}
	return s.sourceStore.Delete(ctx, id)
}

// ValidateConfig validates source configuration for a connector type.
func (s *SourceService) ValidateConfig(_ context.Context, connectorType string, config map[string]string) error {
	if s.connectorRegistry == nil {
		return domain.ErrNotImplemented
	}

	// Get connector type definition from registry
	connType, err := s.connectorRegistry.Get(connectorType)
	if err != nil {
		return fmt.Errorf("unknown connector type %q: %w", connectorType, err)
	}

	// Validate required config keys are present
	var missingKeys []string
	for _, key := range connType.ConfigKeys {
		if key.Required {
			value, exists := config[key.Key]
			if !exists || value == "" {
				missingKeys = append(missingKeys, key.Key)
			}
		}
	}

	if len(missingKeys) > 0 {
		return fmt.Errorf("missing required config keys: %v", missingKeys)
	}

	return nil
}
