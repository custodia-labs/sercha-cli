package domain

// RawDocument represents opaque bytes fetched by a connector.
// It is the connector's output before normalisation.
type RawDocument struct {
	// SourceID links to the Source that produced this document.
	SourceID string

	// URI is the original location (file path, URL, etc).
	URI string

	// MIMEType is the content type (e.g., "application/pdf").
	MIMEType string

	// Content is the raw bytes.
	Content []byte

	// ParentURI links to a parent for hierarchical sources.
	ParentURI *string

	// Metadata contains connector-specific key-value pairs.
	Metadata map[string]any
}

// ChangeType represents the type of document change.
type ChangeType int

const (
	// ChangeCreated indicates a new document.
	ChangeCreated ChangeType = iota

	// ChangeUpdated indicates a modified document.
	ChangeUpdated

	// ChangeDeleted indicates a removed document.
	ChangeDeleted
)

// RawDocumentChange represents a change event from a connector.
// Used for incremental sync and watch operations.
type RawDocumentChange struct {
	// Type is the kind of change.
	Type ChangeType

	// Document is the affected document.
	Document RawDocument
}
