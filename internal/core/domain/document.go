package domain

import "time"

// Document represents an indexed document with metadata.
// It is the canonical representation after normalisation.
type Document struct {
	// ID is the unique identifier for the document.
	ID string

	// SourceID links to the Source that produced this document.
	SourceID string

	// URI is the original location (file path, URL, etc).
	URI string

	// Title is the human-readable title.
	Title string

	// Content is the full text content after normalisation.
	// This is the complete document text before chunking.
	Content string

	// ParentID links to a parent document for hierarchical sources.
	ParentID *string

	// Metadata contains arbitrary key-value pairs.
	Metadata map[string]any

	// CreatedAt is when the document was first indexed.
	CreatedAt time.Time

	// UpdatedAt is when the document was last updated.
	UpdatedAt time.Time
}

// Chunk represents a searchable unit within a document.
// Documents are split into chunks for granular search results.
type Chunk struct {
	// ID is the unique identifier for the chunk.
	ID string

	// DocumentID links to the parent Document.
	DocumentID string

	// Content is the text content of this chunk.
	Content string

	// Position is the ordinal position within the document.
	Position int

	// Embedding is the vector representation for semantic search.
	Embedding []float32

	// Metadata contains chunk-specific key-value pairs.
	Metadata map[string]any
}
