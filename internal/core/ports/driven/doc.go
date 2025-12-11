// Package driven defines the interfaces that core calls OUT to infrastructure.
//
// These are the "driven" or "secondary" ports in hexagonal architecture.
// Core services depend on these interfaces, and infrastructure adapters
// implement them.
//
// # Required Interfaces
//
// These must be provided for the application to function:
//
//   - Connector: Fetches documents from a data source
//   - ConnectorFactory: Creates connectors from configuration
//   - Normaliser: Transforms raw documents into indexed form
//   - NormaliserRegistry: Selects appropriate normaliser
//   - DocumentStore: Document persistence
//   - SourceStore: Source configuration persistence
//   - SyncStateStore: Sync progress persistence
//   - ExclusionStore: Document exclusion persistence
//   - AuthorizationStore: Authorization/credentials persistence
//   - ConfigStore: Application configuration
//   - SearchEngine: Full-text search (Xapian). BM25 keyword search is always required.
//
// # Optional Interfaces
//
// These can be nil - the application degrades gracefully:
//
//   - VectorIndex: Vector storage/search (HNSWlib). Only enabled when EmbeddingService is configured.
//   - EmbeddingService: Generates vector embeddings. Without it, VectorIndex is also disabled.
//   - LLMService: Language model operations. Without it, query rewriting/summarisation is disabled.
//
// # Import Rules
//
//   - Can Import: domain package only
//   - Cannot Import: Any adapter, connector, or normaliser package
package driven
