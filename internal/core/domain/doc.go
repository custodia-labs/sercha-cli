// Package domain defines the core business entities for Sercha.
//
// This package is part of the hexagonal architecture's innermost layer.
// It has NO external dependencies and defines the fundamental types:
//
//   - Document: An indexed document with metadata
//   - Chunk: A searchable unit within a document
//   - Source: A configured data source
//   - RawDocument: Opaque bytes from a connector
//
// # Architectural Position
//
// Domain is at the centre of the hexagon. It may only import
// the Go standard library. All other packages depend on domain,
// never the reverse.
//
// # Import Rules
//
//   - Can Import: Standard library only
//   - Cannot Import: Any internal/ package, any external dependency
package domain
