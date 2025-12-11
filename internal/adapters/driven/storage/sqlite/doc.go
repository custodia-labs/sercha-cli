// Package sqlite provides a unified SQLite-based implementation of driven port interfaces.
//
// This adapter uses modernc.org/sqlite, a pure Go SQLite implementation that requires
// no CGO, enabling easy cross-compilation. It implements multiple store interfaces
// through a single database connection:
//
//   - SourceStore: Source configuration persistence
//   - DocumentStore: Document and chunk persistence
//   - SyncStateStore: Sync progress persistence
//   - ExclusionStore: Document exclusion persistence
//   - AuthorizationStore: OAuth credentials persistence
//
// # Schema
//
// The database schema is managed through versioned migrations stored in the
// migrations/ directory. Each migration is a pair of .up.sql and .down.sql files.
//
// # Data Location
//
// By default, the database is stored at ~/.sercha/data/metadata.db
//
// # Thread Safety
//
// All operations are thread-safe. The store uses database-level locking provided
// by SQLite in WAL mode.
package sqlite
