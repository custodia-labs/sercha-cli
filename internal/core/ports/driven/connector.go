package driven

import (
	"context"
	"errors"

	"github.com/custodia-labs/sercha-cli/internal/core/domain"
)

// Connector fetches documents from a data source.
// Each connector type (filesystem, gmail, github, etc.) implements this interface.
type Connector interface {
	// Type returns the connector type identifier.
	Type() string

	// SourceID returns the configured source ID.
	SourceID() string

	// Capabilities returns what this connector supports.
	Capabilities() ConnectorCapabilities

	// Validate checks if the connector is properly configured and authenticated.
	// Performs a lightweight check to verify the connector is ready to sync.
	// For API connectors, this typically makes a test API call.
	// For filesystem, this checks the path exists and is readable.
	// Returns nil if ready to sync, error describing the problem otherwise.
	Validate(ctx context.Context) error

	// FullSync fetches all documents from the source.
	// Returns channels for documents and errors.
	FullSync(ctx context.Context) (<-chan domain.RawDocument, <-chan error)

	// IncrementalSync fetches only changes since the last sync.
	// Only available if SupportsIncremental is true.
	// Connectors that support cursor return should send SyncComplete on the
	// error channel upon successful completion.
	IncrementalSync(ctx context.Context, state domain.SyncState) (<-chan domain.RawDocumentChange, <-chan error)

	// Watch listens for real-time changes.
	// Only available if SupportsWatch is true.
	Watch(ctx context.Context) (<-chan domain.RawDocumentChange, error)

	// GetAccountIdentifier fetches the user's email or username from the provider.
	// Called after OAuth completion to identify the account for display.
	// Returns the account identifier (e.g., "user@gmail.com", "octocat") or empty string.
	// Returns empty string for no-auth connectors (filesystem).
	GetAccountIdentifier(ctx context.Context, accessToken string) (string, error)

	// Close releases resources.
	Close() error
}

// ConnectorCapabilities describes what a connector supports.
type ConnectorCapabilities struct {
	// === Core Sync Capabilities ===

	// SupportsIncremental indicates the connector can fetch only changes.
	SupportsIncremental bool

	// SupportsWatch indicates the connector can push real-time events.
	SupportsWatch bool

	// SupportsHierarchy indicates the source has nested structure.
	SupportsHierarchy bool

	// SupportsBinary indicates the connector handles binary content.
	SupportsBinary bool

	// === Authentication ===

	// RequiresAuth indicates the connector needs authentication.
	// False for local connectors like filesystem.
	RequiresAuth bool

	// === Validation & Health ===

	// SupportsValidation indicates Validate() performs actual validation.
	// When true, Validate() makes a real check (e.g., API call, path check).
	SupportsValidation bool

	// === Sync Behaviour ===

	// SupportsCursorReturn indicates IncrementalSync can return an updated cursor
	// via the SyncComplete sentinel on the error channel.
	SupportsCursorReturn bool

	// SupportsPartialSync indicates the connector can resume interrupted syncs.
	// When true, sync state should be saved incrementally.
	SupportsPartialSync bool

	// === API Characteristics (informational) ===

	// SupportsRateLimiting indicates the connector handles rate limiting internally.
	// Helps the orchestrator understand connector behaviour.
	SupportsRateLimiting bool

	// SupportsPagination indicates the connector handles paginated APIs.
	// Connectors handle pagination internally; this is informational.
	SupportsPagination bool
}

// SyncComplete is sent on the error channel when sync completes successfully.
// Carries the new cursor state for incremental sync.
type SyncComplete struct {
	NewCursor string
}

// Error implements the error interface.
// This allows SyncComplete to be sent on the error channel.
func (SyncComplete) Error() string {
	return "sync complete"
}

// IsSyncComplete checks if an error is actually a successful completion.
// Returns the SyncComplete and true if it is, nil and false otherwise.
func IsSyncComplete(err error) (*SyncComplete, bool) {
	var sc *SyncComplete
	if errors.As(err, &sc) {
		return sc, true
	}
	return nil, false
}
