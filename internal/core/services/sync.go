package services

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driving"
	"github.com/custodia-labs/sercha-cli/internal/logger"
)

// Ensure SyncOrchestrator implements the interface.
var _ driving.SyncOrchestrator = (*SyncOrchestrator)(nil)

// SyncOrchestrator coordinates document synchronisation.
type SyncOrchestrator struct {
	sourceStore      driven.SourceStore
	syncStore        driven.SyncStateStore
	docStore         driven.DocumentStore
	exclusionStore   driven.ExclusionStore
	factory          driven.ConnectorFactory
	registry         driven.NormaliserRegistry
	pipeline         driven.PostProcessorPipeline
	searchIndex      driven.SearchEngine
	vectorIndex      driven.VectorIndex
	embeddingService driven.EmbeddingService

	// Status tracking
	mu          sync.RWMutex
	activeSyncs map[string]*driving.SyncStatus
}

// NewSyncOrchestrator creates a new sync orchestrator.
// The searchIndex, vectorIndex and embeddingService are used when creating Indexers for sync.
// VectorIndex and embeddingService are optional - if nil, semantic indexing is disabled.
func NewSyncOrchestrator(
	sourceStore driven.SourceStore,
	syncStore driven.SyncStateStore,
	docStore driven.DocumentStore,
	exclusionStore driven.ExclusionStore,
	factory driven.ConnectorFactory,
	registry driven.NormaliserRegistry,
	pipeline driven.PostProcessorPipeline,
	searchIndex driven.SearchEngine,
	vectorIndex driven.VectorIndex,
	embeddingService driven.EmbeddingService,
) *SyncOrchestrator {
	return &SyncOrchestrator{
		sourceStore:      sourceStore,
		syncStore:        syncStore,
		docStore:         docStore,
		exclusionStore:   exclusionStore,
		factory:          factory,
		registry:         registry,
		pipeline:         pipeline,
		searchIndex:      searchIndex,
		vectorIndex:      vectorIndex,
		embeddingService: embeddingService,
		activeSyncs:      make(map[string]*driving.SyncStatus),
	}
}

// Sync triggers synchronisation for a source.
//
//nolint:gocyclo // Orchestration function with necessary sequential steps
func (o *SyncOrchestrator) Sync(ctx context.Context, sourceID string) error {
	// 1. Get source configuration
	source, err := o.sourceStore.Get(ctx, sourceID)
	if err != nil {
		return fmt.Errorf("get source: %w", err)
	}

	// 2. Create connector from source
	if o.factory == nil {
		return fmt.Errorf("create connector: connector factory not configured")
	}
	connector, err := o.factory.Create(ctx, *source)
	if err != nil {
		return fmt.Errorf("create connector: %w", err)
	}
	defer connector.Close()

	// 3. Validate connector (check auth, configuration, connectivity)
	caps := connector.Capabilities()
	if caps.SupportsValidation {
		if err := connector.Validate(ctx); err != nil {
			return fmt.Errorf("%w: %w", domain.ErrConnectorValidation, err)
		}
	}

	// 4. Get sync state (for incremental sync)
	syncState, err := o.syncStore.Get(ctx, sourceID)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return fmt.Errorf("get sync state: %w", err)
	}

	// 5. Initialise status tracking
	status := &driving.SyncStatus{
		SourceID:           sourceID,
		Running:            true,
		DocumentsProcessed: 0,
		ErrorCount:         0,
	}
	o.setStatus(sourceID, status)
	defer o.clearStatus(sourceID)

	logger.Info("Starting sync for source %s", sourceID)

	// 6. Choose sync strategy based on connector capabilities
	var newCursor string

	if caps.SupportsIncremental && syncState != nil && syncState.Cursor != "" {
		// Incremental sync
		changesCh, errsCh := connector.IncrementalSync(ctx, *syncState)
		newCursor, err = o.processChanges(ctx, source, changesCh, errsCh, status)
	} else {
		// Full sync
		docsCh, errsCh := connector.FullSync(ctx)
		newCursor, err = o.processDocuments(ctx, source, docsCh, errsCh, status)
		// For full sync, fall back to current time if no cursor was returned
		if err == nil && newCursor == "" && caps.SupportsCursorReturn {
			newCursor = fmt.Sprintf("%d", time.Now().UnixNano())
		}
	}

	if err != nil {
		return err
	}

	// 7. Update sync state with new cursor
	newState := domain.SyncState{
		SourceID: sourceID,
		Cursor:   newCursor,
		LastSync: time.Now(),
	}
	if err := o.syncStore.Save(ctx, newState); err != nil {
		return fmt.Errorf("save sync state: %w", err)
	}

	logger.Info("Sync complete: %d documents, %d errors", status.DocumentsProcessed, status.ErrorCount)
	status.Running = false
	return nil
}

// SyncAll triggers synchronisation for all configured sources.
func (o *SyncOrchestrator) SyncAll(ctx context.Context) error {
	sources, err := o.sourceStore.List(ctx)
	if err != nil {
		return fmt.Errorf("list sources: %w", err)
	}

	var errs []error
	for _, source := range sources {
		if err := o.Sync(ctx, source.ID); err != nil {
			errs = append(errs, fmt.Errorf("sync %s: %w", source.ID, err))
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// Status returns sync status for a source.
func (o *SyncOrchestrator) Status(_ context.Context, sourceID string) (*driving.SyncStatus, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	if status, ok := o.activeSyncs[sourceID]; ok {
		// Return a copy to avoid race conditions
		return &driving.SyncStatus{
			SourceID:           status.SourceID,
			Running:            status.Running,
			DocumentsProcessed: status.DocumentsProcessed,
			ErrorCount:         status.ErrorCount,
		}, nil
	}

	// Not running - return idle status
	return &driving.SyncStatus{
		SourceID: sourceID,
		Running:  false,
	}, nil
}

// processDocuments handles full sync - processes all documents from the connector.
// Returns the new cursor from SyncComplete if the connector provides one.
//
//nolint:gocognit // Orchestration function coordinating multiple async operations
func (o *SyncOrchestrator) processDocuments(
	ctx context.Context,
	source *domain.Source,
	docsCh <-chan domain.RawDocument,
	errsCh <-chan error,
	status *driving.SyncStatus,
) (string, error) {
	var newCursor string

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()

		case err, ok := <-errsCh:
			if !ok {
				errsCh = nil
				continue
			}
			// Check if this is a SyncComplete (successful completion with cursor)
			if sc, isSyncComplete := driven.IsSyncComplete(err); isSyncComplete {
				newCursor = sc.NewCursor
				continue
			}
			if err != nil {
				return "", fmt.Errorf("connector error: %w", err)
			}

		case rawDoc, ok := <-docsCh:
			if !ok {
				return newCursor, nil // Done - channel closed
			}

			logger.Debug("Processing: %s", rawDoc.URI)
			if err := o.processOneDocument(ctx, source, &rawDoc); err != nil {
				status.ErrorCount++
				if errors.Is(err, domain.ErrNotImplemented) {
					logger.Debug("Skipping %s: %v", rawDoc.URI, err)
				} else {
					logger.Debug("Failed to process %s: %v", rawDoc.URI, err)
				}
				continue
			}
			status.DocumentsProcessed++
		}
	}
}

// processChanges handles incremental sync - processes document changes.
// Returns the new cursor from SyncComplete if the connector provides one.
//
//nolint:gocognit // Orchestration function coordinating multiple async operations
func (o *SyncOrchestrator) processChanges(
	ctx context.Context,
	source *domain.Source,
	changesCh <-chan domain.RawDocumentChange,
	errsCh <-chan error,
	status *driving.SyncStatus,
) (string, error) {
	var newCursor string

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()

		case err, ok := <-errsCh:
			if !ok {
				errsCh = nil
				continue
			}
			// Check if this is a SyncComplete (successful completion with cursor)
			if sc, isSyncComplete := driven.IsSyncComplete(err); isSyncComplete {
				newCursor = sc.NewCursor
				continue
			}
			if err != nil {
				return "", fmt.Errorf("connector error: %w", err)
			}

		case change, ok := <-changesCh:
			if !ok {
				return newCursor, nil // Done - channel closed
			}

			switch change.Type {
			case domain.ChangeCreated, domain.ChangeUpdated:
				logger.Debug("Processing: %s", change.Document.URI)
				if err := o.processOneDocument(ctx, source, &change.Document); err != nil {
					status.ErrorCount++
					if errors.Is(err, domain.ErrNotImplemented) {
						logger.Debug("Skipping %s: %v", change.Document.URI, err)
					} else {
						logger.Debug("Failed to process %s: %v", change.Document.URI, err)
					}
					continue
				}

			case domain.ChangeDeleted:
				logger.Debug("Deleting: %s", change.Document.URI)
				if err := o.deleteDocumentByURI(ctx, source.ID, change.Document.URI); err != nil {
					status.ErrorCount++
					logger.Debug("Failed to delete %s: %v", change.Document.URI, err)
					continue
				}
			}
			status.DocumentsProcessed++
		}
	}
}

// processOneDocument handles the 7-step document processing pipeline.
//
//nolint:gocognit,gocyclo // Pipeline orchestration with sequential steps
func (o *SyncOrchestrator) processOneDocument(
	ctx context.Context,
	source *domain.Source,
	raw *domain.RawDocument,
) error {
	// 1. CHECK EXCLUSION
	excluded, err := o.exclusionStore.IsExcluded(ctx, source.ID, raw.URI)
	if err != nil {
		return fmt.Errorf("check exclusion: %w", err)
	}
	if excluded {
		return nil // Skip silently
	}

	// 2. NORMALISE (produces Document with Content)
	result, err := o.registry.Normalise(ctx, raw)
	if err != nil {
		return fmt.Errorf("normalise: %w", err)
	}

	// 3. RUN POST-PROCESSOR PIPELINE (produces Chunks)
	chunks, err := o.pipeline.Process(ctx, &result.Document)
	if err != nil {
		return fmt.Errorf("post-process: %w", err)
	}

	// 4. GENERATE EMBEDDINGS (if service available)
	if o.embeddingService != nil {
		for i := range chunks {
			embedding, err := o.embeddingService.Embed(ctx, chunks[i].Content)
			if err != nil {
				return fmt.Errorf("embed chunk: %w", err)
			}
			chunks[i].Embedding = embedding
		}
	}

	// 5. SAVE TO DOCUMENT STORE
	if err := o.docStore.SaveDocument(ctx, &result.Document); err != nil {
		return fmt.Errorf("save document: %w", err)
	}
	if err := o.docStore.SaveChunks(ctx, chunks); err != nil {
		return fmt.Errorf("save chunks: %w", err)
	}

	// 6. INDEX FOR KEYWORD SEARCH
	for _, chunk := range chunks {
		if err := o.searchIndex.Index(ctx, chunk); err != nil {
			return fmt.Errorf("index chunk: %w", err)
		}
	}

	// 7. INDEX FOR VECTOR SEARCH (if available)
	if o.vectorIndex != nil && o.embeddingService != nil {
		for _, chunk := range chunks {
			if chunk.Embedding != nil {
				if err := o.vectorIndex.Add(ctx, chunk.ID, chunk.Embedding); err != nil {
					return fmt.Errorf("add vector: %w", err)
				}
			}
		}
	}

	return nil
}

// deleteDocumentByURI removes a document and its indexes by URI.
func (o *SyncOrchestrator) deleteDocumentByURI(ctx context.Context, sourceID, uri string) error {
	// Find document by URI - iterate through source documents
	docs, err := o.docStore.ListDocuments(ctx, sourceID)
	if err != nil {
		return fmt.Errorf("list documents: %w", err)
	}

	var docToDelete *domain.Document
	for i := range docs {
		if docs[i].URI == uri {
			docToDelete = &docs[i]
			break
		}
	}

	if docToDelete == nil {
		// Document not found - might have been deleted already
		return nil
	}

	// Get chunks before deleting
	chunks, err := o.docStore.GetChunks(ctx, docToDelete.ID)
	if err != nil {
		return fmt.Errorf("get chunks: %w", err)
	}

	// Delete from vector index
	if o.vectorIndex != nil {
		for _, chunk := range chunks {
			if err := o.vectorIndex.Delete(ctx, chunk.ID); err != nil {
				logger.Debug("Failed to delete vector %s: %v", chunk.ID, err)
			}
		}
	}

	// Delete from search index
	for _, chunk := range chunks {
		if err := o.searchIndex.Delete(ctx, chunk.ID); err != nil {
			logger.Debug("Failed to delete search index %s: %v", chunk.ID, err)
		}
	}

	// Delete document and chunks from store
	if err := o.docStore.DeleteDocument(ctx, docToDelete.ID); err != nil {
		return fmt.Errorf("delete document: %w", err)
	}

	return nil
}

// setStatus sets the sync status for a source.
func (o *SyncOrchestrator) setStatus(sourceID string, status *driving.SyncStatus) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.activeSyncs[sourceID] = status
}

// clearStatus removes the sync status for a source.
func (o *SyncOrchestrator) clearStatus(sourceID string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	delete(o.activeSyncs, sourceID)
}
