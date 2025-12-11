package driving

import "context"

// SyncOrchestrator coordinates document synchronisation from sources.
type SyncOrchestrator interface {
	// Sync triggers synchronisation for a source.
	Sync(ctx context.Context, sourceID string) error

	// SyncAll triggers synchronisation for all configured sources.
	SyncAll(ctx context.Context) error

	// Status returns sync status for a source.
	Status(ctx context.Context, sourceID string) (*SyncStatus, error)
}

// SyncStatus represents the current state of a sync operation.
type SyncStatus struct {
	// SourceID identifies the source.
	SourceID string

	// Running indicates if sync is currently in progress.
	Running bool

	// DocumentsProcessed is the count of documents processed.
	DocumentsProcessed int

	// ErrorCount is the number of errors encountered.
	ErrorCount int
}
