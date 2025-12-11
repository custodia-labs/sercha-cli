package domain

import "time"

// Exclusion represents a document that has been excluded from syncing.
// When a document is excluded, it will not be re-indexed during future syncs.
type Exclusion struct {
	// ID is the unique identifier for the exclusion.
	ID string

	// SourceID links to the Source this exclusion applies to.
	SourceID string

	// DocumentID is the ID of the excluded document.
	DocumentID string

	// URI is the original location for matching on re-sync.
	URI string

	// Reason is an optional explanation for the exclusion.
	Reason string

	// ExcludedAt is when the document was excluded.
	ExcludedAt time.Time
}
