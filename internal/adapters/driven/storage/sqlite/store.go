package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite" // SQLite driver

	"github.com/custodia-labs/sercha-cli/internal/adapters/driven/storage/sqlite/migrations"
	"github.com/custodia-labs/sercha-cli/internal/core/domain"
	"github.com/custodia-labs/sercha-cli/internal/core/ports/driven"
)

// jsonNull is the JSON representation of null.
const jsonNull = "null"

// Store is a unified SQLite-based storage that provides access to
// all metadata store interfaces through wrapper types.
type Store struct {
	db   *sql.DB
	path string
}

// NewStore creates a new SQLite store at the specified data directory.
// If dataDir is empty, defaults to ~/.sercha/data/metadata.db.
func NewStore(dataDir string) (*Store, error) {
	if dataDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("getting home directory: %w", err)
		}
		dataDir = filepath.Join(home, ".sercha", "data")
	}

	// Ensure directory exists
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return nil, fmt.Errorf("creating data directory: %w", err)
	}

	dbPath := filepath.Join(dataDir, "metadata.db")

	// Open database with WAL mode for better concurrency
	db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enabling foreign keys: %w", err)
	}

	s := &Store{
		db:   db,
		path: dbPath,
	}

	// Run migrations
	if err := s.migrate(migrations.FS); err != nil {
		db.Close()
		return nil, fmt.Errorf("running migrations: %w", err)
	}

	return s, nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// Path returns the database file path.
func (s *Store) Path() string {
	return s.path
}

// SourceStore returns a SourceStore interface backed by this store.
func (s *Store) SourceStore() driven.SourceStore {
	return &sourceStore{store: s}
}

// DocumentStore returns a DocumentStore interface backed by this store.
func (s *Store) DocumentStore() driven.DocumentStore {
	return &documentStore{store: s}
}

// SyncStateStore returns a SyncStateStore interface backed by this store.
func (s *Store) SyncStateStore() driven.SyncStateStore {
	return &syncStateStore{store: s}
}

// ExclusionStore returns an ExclusionStore interface backed by this store.
func (s *Store) ExclusionStore() driven.ExclusionStore {
	return &exclusionStore{store: s}
}

// SchedulerStore returns a SchedulerStore interface backed by this store.
func (s *Store) SchedulerStore() driven.SchedulerStore {
	return &schedulerStore{store: s}
}

// AuthProviderStore returns an AuthProviderStore interface backed by this store.
func (s *Store) AuthProviderStore() driven.AuthProviderStore {
	return &authProviderStore{store: s}
}

// CredentialsStore returns a CredentialsStore interface backed by this store.
func (s *Store) CredentialsStore() driven.CredentialsStore {
	return &credentialsStore{store: s}
}

// migrate runs all pending migrations.
func (s *Store) migrate(fsys embed.FS) error {
	// Ensure schema_migrations table exists
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("creating schema_migrations table: %w", err)
	}

	// Get current version
	var currentVersion int
	row := s.db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_migrations")
	if err := row.Scan(&currentVersion); err != nil {
		return fmt.Errorf("getting current version: %w", err)
	}

	// Find all up migrations
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return fmt.Errorf("reading migrations directory: %w", err)
	}

	// Sort and run migrations
	var upFiles []string
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasSuffix(name, ".up.sql") {
			upFiles = append(upFiles, name)
		}
	}
	sort.Strings(upFiles)

	for _, name := range upFiles {
		// Extract version number (e.g., "001_initial.up.sql" -> 1)
		var version int
		if _, err := fmt.Sscanf(name, "%d_", &version); err != nil {
			continue // Skip files that don't match pattern
		}

		if version <= currentVersion {
			continue // Already applied
		}

		// Read and execute migration
		content, err := fs.ReadFile(fsys, name)
		if err != nil {
			return fmt.Errorf("reading migration %s: %w", name, err)
		}

		if _, err := s.db.Exec(string(content)); err != nil {
			return fmt.Errorf("executing migration %s: %w", name, err)
		}
	}

	return nil
}

// ==================== Source Store ====================

// sourceStore implements driven.SourceStore.
type sourceStore struct {
	store *Store
}

var _ driven.SourceStore = (*sourceStore)(nil)

// Save stores or updates a source.
func (s *sourceStore) Save(ctx context.Context, source domain.Source) error {
	configJSON, err := json.Marshal(source.Config)
	if err != nil {
		return fmt.Errorf("marshalling config: %w", err)
	}

	now := time.Now().UTC()
	if source.CreatedAt.IsZero() {
		source.CreatedAt = now
	}
	source.UpdatedAt = now

	_, err = s.store.db.ExecContext(ctx, `
		INSERT INTO sources (id, type, name, config, auth_provider_id, credentials_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			type = excluded.type,
			name = excluded.name,
			config = excluded.config,
			auth_provider_id = excluded.auth_provider_id,
			credentials_id = excluded.credentials_id,
			updated_at = excluded.updated_at
	`, source.ID, source.Type, source.Name, string(configJSON),
		nullString(source.AuthProviderID), nullString(source.CredentialsID),
		source.CreatedAt, source.UpdatedAt)

	if err != nil {
		return fmt.Errorf("saving source: %w", err)
	}
	return nil
}

// Get retrieves a source by ID.
func (s *sourceStore) Get(ctx context.Context, id string) (*domain.Source, error) {
	row := s.store.db.QueryRowContext(ctx, `
		SELECT id, type, name, config, auth_provider_id, credentials_id, created_at, updated_at
		FROM sources WHERE id = ?
	`, id)

	var source domain.Source
	var configJSON string
	var authProviderID, credentialsID sql.NullString
	var createdAt, updatedAt sql.NullTime
	if err := row.Scan(&source.ID, &source.Type, &source.Name, &configJSON,
		&authProviderID, &credentialsID, &createdAt, &updatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("scanning source: %w", err)
	}

	if err := json.Unmarshal([]byte(configJSON), &source.Config); err != nil {
		return nil, fmt.Errorf("unmarshaling config: %w", err)
	}

	source.AuthProviderID = authProviderID.String
	source.CredentialsID = credentialsID.String
	if createdAt.Valid {
		source.CreatedAt = createdAt.Time
	}
	if updatedAt.Valid {
		source.UpdatedAt = updatedAt.Time
	}

	return &source, nil
}

// Delete removes a source.
func (s *sourceStore) Delete(ctx context.Context, id string) error {
	_, err := s.store.db.ExecContext(ctx, "DELETE FROM sources WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting source: %w", err)
	}
	return nil
}

// List returns all configured sources.
func (s *sourceStore) List(ctx context.Context) ([]domain.Source, error) {
	rows, err := s.store.db.QueryContext(ctx, `
		SELECT id, type, name, config, auth_provider_id, credentials_id, created_at, updated_at
		FROM sources
	`)
	if err != nil {
		return nil, fmt.Errorf("querying sources: %w", err)
	}
	defer rows.Close()

	var sources []domain.Source //nolint:prealloc // size unknown from query
	for rows.Next() {
		var source domain.Source
		var configJSON string
		var authProviderID, credentialsID sql.NullString
		var createdAt, updatedAt sql.NullTime
		if err := rows.Scan(&source.ID, &source.Type, &source.Name, &configJSON,
			&authProviderID, &credentialsID, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scanning source: %w", err)
		}

		if err := json.Unmarshal([]byte(configJSON), &source.Config); err != nil {
			return nil, fmt.Errorf("unmarshaling config: %w", err)
		}

		source.AuthProviderID = authProviderID.String
		source.CredentialsID = credentialsID.String
		if createdAt.Valid {
			source.CreatedAt = createdAt.Time
		}
		if updatedAt.Valid {
			source.UpdatedAt = updatedAt.Time
		}
		sources = append(sources, source)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating sources: %w", err)
	}

	return sources, nil
}

// ==================== Document Store ====================

// documentStore implements driven.DocumentStore.
type documentStore struct {
	store *Store
}

var _ driven.DocumentStore = (*documentStore)(nil)

// SaveDocument stores or updates a document.
func (s *documentStore) SaveDocument(ctx context.Context, doc *domain.Document) error {
	metadataJSON, err := json.Marshal(doc.Metadata)
	if err != nil {
		return fmt.Errorf("marshalling metadata: %w", err)
	}

	_, err = s.store.db.ExecContext(ctx, `
		INSERT INTO documents (id, source_id, uri, title, content, parent_id, metadata, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			source_id = excluded.source_id,
			uri = excluded.uri,
			title = excluded.title,
			content = excluded.content,
			parent_id = excluded.parent_id,
			metadata = excluded.metadata,
			updated_at = excluded.updated_at
	`, doc.ID, doc.SourceID, doc.URI, doc.Title, doc.Content,
		doc.ParentID, string(metadataJSON), doc.CreatedAt, doc.UpdatedAt)

	if err != nil {
		return fmt.Errorf("saving document: %w", err)
	}
	return nil
}

// SaveChunks stores chunks for a document.
func (s *documentStore) SaveChunks(ctx context.Context, chunks []domain.Chunk) error {
	tx, err := s.store.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO chunks (id, document_id, content, position, embedding, metadata)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			document_id = excluded.document_id,
			content = excluded.content,
			position = excluded.position,
			embedding = excluded.embedding,
			metadata = excluded.metadata
	`)
	if err != nil {
		return fmt.Errorf("preparing statement: %w", err)
	}
	defer stmt.Close()

	for _, chunk := range chunks {
		metadataJSON, err := json.Marshal(chunk.Metadata)
		if err != nil {
			return fmt.Errorf("marshalling chunk metadata: %w", err)
		}

		embeddingBlob := float32SliceToBytes(chunk.Embedding)

		if _, err := stmt.ExecContext(ctx, chunk.ID, chunk.DocumentID, chunk.Content,
			chunk.Position, embeddingBlob, string(metadataJSON)); err != nil {
			return fmt.Errorf("saving chunk: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}
	return nil
}

// GetDocument retrieves a document by ID.
func (s *documentStore) GetDocument(ctx context.Context, id string) (*domain.Document, error) {
	row := s.store.db.QueryRowContext(ctx, `
		SELECT id, source_id, uri, title, content, parent_id, metadata, created_at, updated_at
		FROM documents WHERE id = ?
	`, id)

	return scanDocument(row)
}

// GetChunks retrieves all chunks for a document.
func (s *documentStore) GetChunks(ctx context.Context, documentID string) ([]domain.Chunk, error) {
	rows, err := s.store.db.QueryContext(ctx, `
		SELECT id, document_id, content, position, embedding, metadata
		FROM chunks WHERE document_id = ?
		ORDER BY position
	`, documentID)
	if err != nil {
		return nil, fmt.Errorf("querying chunks: %w", err)
	}
	defer rows.Close()

	var chunks []domain.Chunk //nolint:prealloc // size unknown from query
	for rows.Next() {
		chunk, err := scanChunk(rows)
		if err != nil {
			return nil, err
		}
		chunks = append(chunks, *chunk)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating chunks: %w", err)
	}

	return chunks, nil
}

// GetChunk retrieves a specific chunk by ID.
func (s *documentStore) GetChunk(ctx context.Context, id string) (*domain.Chunk, error) {
	row := s.store.db.QueryRowContext(ctx, `
		SELECT id, document_id, content, position, embedding, metadata
		FROM chunks WHERE id = ?
	`, id)

	return scanChunkRow(row)
}

// DeleteDocument removes a document and its chunks.
func (s *documentStore) DeleteDocument(ctx context.Context, id string) error {
	_, err := s.store.db.ExecContext(ctx, "DELETE FROM documents WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting document: %w", err)
	}
	return nil
}

// ListDocuments returns documents for a source.
func (s *documentStore) ListDocuments(ctx context.Context, sourceID string) ([]domain.Document, error) {
	rows, err := s.store.db.QueryContext(ctx, `
		SELECT id, source_id, uri, title, content, parent_id, metadata, created_at, updated_at
		FROM documents WHERE source_id = ?
	`, sourceID)
	if err != nil {
		return nil, fmt.Errorf("querying documents: %w", err)
	}
	defer rows.Close()

	var docs []domain.Document //nolint:prealloc // size unknown from query
	for rows.Next() {
		doc, err := scanDocumentRows(rows)
		if err != nil {
			return nil, err
		}
		docs = append(docs, *doc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating documents: %w", err)
	}

	return docs, nil
}

// ==================== Sync State Store ====================

// syncStateStore implements driven.SyncStateStore.
type syncStateStore struct {
	store *Store
}

var _ driven.SyncStateStore = (*syncStateStore)(nil)

// Save stores or updates sync state.
func (s *syncStateStore) Save(ctx context.Context, state domain.SyncState) error {
	_, err := s.store.db.ExecContext(ctx, `
		INSERT INTO sync_states (source_id, cursor, last_sync)
		VALUES (?, ?, ?)
		ON CONFLICT(source_id) DO UPDATE SET
			cursor = excluded.cursor,
			last_sync = excluded.last_sync
	`, state.SourceID, state.Cursor, state.LastSync)

	if err != nil {
		return fmt.Errorf("saving sync state: %w", err)
	}
	return nil
}

// Get retrieves sync state for a source.
func (s *syncStateStore) Get(ctx context.Context, sourceID string) (*domain.SyncState, error) {
	row := s.store.db.QueryRowContext(ctx, `
		SELECT source_id, cursor, last_sync
		FROM sync_states WHERE source_id = ?
	`, sourceID)

	var state domain.SyncState
	var lastSync sql.NullTime
	if err := row.Scan(&state.SourceID, &state.Cursor, &lastSync); err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("scanning sync state: %w", err)
	}

	if lastSync.Valid {
		state.LastSync = lastSync.Time
	}

	return &state, nil
}

// Delete removes sync state for a source.
func (s *syncStateStore) Delete(ctx context.Context, sourceID string) error {
	_, err := s.store.db.ExecContext(ctx, "DELETE FROM sync_states WHERE source_id = ?", sourceID)
	if err != nil {
		return fmt.Errorf("deleting sync state: %w", err)
	}
	return nil
}

// ==================== Exclusion Store ====================

// exclusionStore implements driven.ExclusionStore.
type exclusionStore struct {
	store *Store
}

var _ driven.ExclusionStore = (*exclusionStore)(nil)

// Add creates a new exclusion.
func (s *exclusionStore) Add(ctx context.Context, exclusion *domain.Exclusion) error {
	_, err := s.store.db.ExecContext(ctx, `
		INSERT INTO exclusions (id, source_id, document_id, uri, reason, excluded_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, exclusion.ID, exclusion.SourceID, exclusion.DocumentID, exclusion.URI, exclusion.Reason, exclusion.ExcludedAt)

	if err != nil {
		return fmt.Errorf("adding exclusion: %w", err)
	}
	return nil
}

// Remove deletes an exclusion by ID.
func (s *exclusionStore) Remove(ctx context.Context, id string) error {
	_, err := s.store.db.ExecContext(ctx, "DELETE FROM exclusions WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("removing exclusion: %w", err)
	}
	return nil
}

// GetBySourceID returns all exclusions for a source.
func (s *exclusionStore) GetBySourceID(ctx context.Context, sourceID string) ([]domain.Exclusion, error) {
	rows, err := s.store.db.QueryContext(ctx, `
		SELECT id, source_id, document_id, uri, reason, excluded_at
		FROM exclusions WHERE source_id = ?
	`, sourceID)
	if err != nil {
		return nil, fmt.Errorf("querying exclusions: %w", err)
	}
	defer rows.Close()

	return scanExclusions(rows)
}

// IsExcluded checks if a URI is excluded for a source.
func (s *exclusionStore) IsExcluded(ctx context.Context, sourceID, uri string) (bool, error) {
	var count int
	err := s.store.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM exclusions WHERE source_id = ? AND uri = ?
	`, sourceID, uri).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("checking exclusion: %w", err)
	}
	return count > 0, nil
}

// List returns all exclusions.
func (s *exclusionStore) List(ctx context.Context) ([]domain.Exclusion, error) {
	rows, err := s.store.db.QueryContext(ctx, `
		SELECT id, source_id, document_id, uri, reason, excluded_at
		FROM exclusions
	`)
	if err != nil {
		return nil, fmt.Errorf("querying exclusions: %w", err)
	}
	defer rows.Close()

	return scanExclusions(rows)
}

// ==================== Helper Functions ====================

// float32SliceToBytes converts a []float32 to a byte slice for storage.
func float32SliceToBytes(floats []float32) []byte {
	if len(floats) == 0 {
		return nil
	}
	buf := make([]byte, len(floats)*4)
	for i, f := range floats {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(f))
	}
	return buf
}

// bytesToFloat32Slice converts a byte slice back to []float32.
func bytesToFloat32Slice(data []byte) []float32 {
	if len(data) == 0 {
		return nil
	}
	floats := make([]float32, len(data)/4)
	for i := range floats {
		floats[i] = math.Float32frombits(binary.LittleEndian.Uint32(data[i*4:]))
	}
	return floats
}

// scanDocument scans a single document row.
func scanDocument(row *sql.Row) (*domain.Document, error) {
	var doc domain.Document
	var parentID sql.NullString
	var metadataJSON string

	if err := row.Scan(&doc.ID, &doc.SourceID, &doc.URI, &doc.Title, &doc.Content,
		&parentID, &metadataJSON, &doc.CreatedAt, &doc.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("scanning document: %w", err)
	}

	if parentID.Valid {
		doc.ParentID = &parentID.String
	}

	if metadataJSON != "" {
		if err := json.Unmarshal([]byte(metadataJSON), &doc.Metadata); err != nil {
			return nil, fmt.Errorf("unmarshaling metadata: %w", err)
		}
	}

	return &doc, nil
}

// scanDocumentRows scans a document from *sql.Rows.
func scanDocumentRows(rows *sql.Rows) (*domain.Document, error) {
	var doc domain.Document
	var parentID sql.NullString
	var metadataJSON string

	if err := rows.Scan(&doc.ID, &doc.SourceID, &doc.URI, &doc.Title, &doc.Content,
		&parentID, &metadataJSON, &doc.CreatedAt, &doc.UpdatedAt); err != nil {
		return nil, fmt.Errorf("scanning document: %w", err)
	}

	if parentID.Valid {
		doc.ParentID = &parentID.String
	}

	if metadataJSON != "" {
		if err := json.Unmarshal([]byte(metadataJSON), &doc.Metadata); err != nil {
			return nil, fmt.Errorf("unmarshaling metadata: %w", err)
		}
	}

	return &doc, nil
}

// scanChunk scans a chunk from *sql.Rows.
func scanChunk(rows *sql.Rows) (*domain.Chunk, error) {
	var chunk domain.Chunk
	var embeddingBlob []byte
	var metadataJSON string

	if err := rows.Scan(&chunk.ID, &chunk.DocumentID, &chunk.Content,
		&chunk.Position, &embeddingBlob, &metadataJSON); err != nil {
		return nil, fmt.Errorf("scanning chunk: %w", err)
	}

	chunk.Embedding = bytesToFloat32Slice(embeddingBlob)

	if metadataJSON != "" {
		if err := json.Unmarshal([]byte(metadataJSON), &chunk.Metadata); err != nil {
			return nil, fmt.Errorf("unmarshaling chunk metadata: %w", err)
		}
	}

	return &chunk, nil
}

// scanChunkRow scans a chunk from *sql.Row.
func scanChunkRow(row *sql.Row) (*domain.Chunk, error) {
	var chunk domain.Chunk
	var embeddingBlob []byte
	var metadataJSON string

	if err := row.Scan(&chunk.ID, &chunk.DocumentID, &chunk.Content,
		&chunk.Position, &embeddingBlob, &metadataJSON); err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("scanning chunk: %w", err)
	}

	chunk.Embedding = bytesToFloat32Slice(embeddingBlob)

	if metadataJSON != "" {
		if err := json.Unmarshal([]byte(metadataJSON), &chunk.Metadata); err != nil {
			return nil, fmt.Errorf("unmarshaling chunk metadata: %w", err)
		}
	}

	return &chunk, nil
}

// scanExclusions scans multiple exclusion rows.
func scanExclusions(rows *sql.Rows) ([]domain.Exclusion, error) {
	var exclusions []domain.Exclusion //nolint:prealloc // size unknown from query
	for rows.Next() {
		var e domain.Exclusion
		if err := rows.Scan(&e.ID, &e.SourceID, &e.DocumentID, &e.URI, &e.Reason, &e.ExcludedAt); err != nil {
			return nil, fmt.Errorf("scanning exclusion: %w", err)
		}
		exclusions = append(exclusions, e)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating exclusions: %w", err)
	}

	return exclusions, nil
}

// =============================================================================
// AuthProviderStore Implementation
// =============================================================================

type authProviderStore struct {
	store *Store
}

var _ driven.AuthProviderStore = (*authProviderStore)(nil)

// Save stores or updates an auth provider.
func (s *authProviderStore) Save(ctx context.Context, provider domain.AuthProvider) error {
	if provider.ID == "" {
		return domain.ErrInvalidInput
	}

	oauthJSON, err := json.Marshal(provider.OAuth)
	if err != nil {
		return fmt.Errorf("marshalling oauth config: %w", err)
	}

	_, err = s.store.db.ExecContext(ctx, `
		INSERT INTO auth_providers
			(id, name, provider_type, auth_method, oauth, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			provider_type = excluded.provider_type,
			auth_method = excluded.auth_method,
			oauth = excluded.oauth,
			updated_at = excluded.updated_at
	`, provider.ID, provider.Name, string(provider.ProviderType), string(provider.AuthMethod),
		string(oauthJSON), provider.CreatedAt, provider.UpdatedAt)

	if err != nil {
		return fmt.Errorf("saving auth provider: %w", err)
	}
	return nil
}

// Get retrieves an auth provider by ID.
func (s *authProviderStore) Get(ctx context.Context, id string) (*domain.AuthProvider, error) {
	row := s.store.db.QueryRowContext(ctx, `
		SELECT id, name, provider_type, auth_method, oauth, created_at, updated_at
		FROM auth_providers WHERE id = ?
	`, id)

	return scanAuthProvider(row)
}

// List returns all auth providers.
func (s *authProviderStore) List(ctx context.Context) ([]domain.AuthProvider, error) {
	rows, err := s.store.db.QueryContext(ctx, `
		SELECT id, name, provider_type, auth_method, oauth, created_at, updated_at
		FROM auth_providers
	`)
	if err != nil {
		return nil, fmt.Errorf("querying auth providers: %w", err)
	}
	defer rows.Close()

	return scanAuthProviderRows(rows)
}

// ListByProvider returns all auth providers for a specific provider type.
func (s *authProviderStore) ListByProvider(
	ctx context.Context,
	providerType domain.ProviderType,
) ([]domain.AuthProvider, error) {
	rows, err := s.store.db.QueryContext(ctx, `
		SELECT id, name, provider_type, auth_method, oauth, created_at, updated_at
		FROM auth_providers WHERE provider_type = ?
	`, string(providerType))
	if err != nil {
		return nil, fmt.Errorf("querying auth providers by provider: %w", err)
	}
	defer rows.Close()

	return scanAuthProviderRows(rows)
}

// Delete removes an auth provider by ID.
func (s *authProviderStore) Delete(ctx context.Context, id string) error {
	// Check if any sources are using this provider
	var count int
	err := s.store.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM sources WHERE auth_provider_id = ?", id).Scan(&count)
	if err != nil {
		return fmt.Errorf("checking provider usage: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("cannot delete auth provider: still in use by %d source(s)", count)
	}

	_, err = s.store.db.ExecContext(ctx, "DELETE FROM auth_providers WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting auth provider: %w", err)
	}
	return nil
}

// scanAuthProvider scans a single auth provider row.
func scanAuthProvider(row *sql.Row) (*domain.AuthProvider, error) {
	var provider domain.AuthProvider
	var providerType, authMethod string
	var oauthJSON sql.NullString

	if err := row.Scan(&provider.ID, &provider.Name, &providerType, &authMethod,
		&oauthJSON, &provider.CreatedAt, &provider.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("scanning auth provider: %w", err)
	}

	provider.ProviderType = domain.ProviderType(providerType)
	provider.AuthMethod = domain.AuthMethod(authMethod)

	if oauthJSON.Valid && oauthJSON.String != jsonNull {
		var oauth domain.OAuthProviderConfig
		if err := json.Unmarshal([]byte(oauthJSON.String), &oauth); err != nil {
			return nil, fmt.Errorf("unmarshalling oauth config: %w", err)
		}
		provider.OAuth = &oauth
	}

	return &provider, nil
}

// scanAuthProviderRows scans multiple auth provider rows.
func scanAuthProviderRows(rows *sql.Rows) ([]domain.AuthProvider, error) {
	var providers []domain.AuthProvider //nolint:prealloc // size unknown from query
	for rows.Next() {
		var provider domain.AuthProvider
		var providerType, authMethod string
		var oauthJSON sql.NullString

		if err := rows.Scan(&provider.ID, &provider.Name, &providerType, &authMethod,
			&oauthJSON, &provider.CreatedAt, &provider.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning auth provider: %w", err)
		}

		provider.ProviderType = domain.ProviderType(providerType)
		provider.AuthMethod = domain.AuthMethod(authMethod)

		if oauthJSON.Valid && oauthJSON.String != jsonNull {
			var oauth domain.OAuthProviderConfig
			if err := json.Unmarshal([]byte(oauthJSON.String), &oauth); err != nil {
				return nil, fmt.Errorf("unmarshalling oauth config: %w", err)
			}
			provider.OAuth = &oauth
		}

		providers = append(providers, provider)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating auth providers: %w", err)
	}

	return providers, nil
}

// =============================================================================
// CredentialsStore Implementation
// =============================================================================

type credentialsStore struct {
	store *Store
}

var _ driven.CredentialsStore = (*credentialsStore)(nil)

// Save stores or updates credentials.
func (s *credentialsStore) Save(ctx context.Context, creds domain.Credentials) error {
	if creds.ID == "" || creds.SourceID == "" {
		return domain.ErrInvalidInput
	}

	oauthJSON, err := json.Marshal(creds.OAuth)
	if err != nil {
		return fmt.Errorf("marshalling oauth credentials: %w", err)
	}

	patJSON, err := json.Marshal(creds.PAT)
	if err != nil {
		return fmt.Errorf("marshalling pat credentials: %w", err)
	}

	_, err = s.store.db.ExecContext(ctx, `
		INSERT INTO credentials
			(id, source_id, account_identifier, oauth, pat, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			source_id = excluded.source_id,
			account_identifier = excluded.account_identifier,
			oauth = excluded.oauth,
			pat = excluded.pat,
			updated_at = excluded.updated_at
	`, creds.ID, creds.SourceID, creds.AccountIdentifier,
		string(oauthJSON), string(patJSON), creds.CreatedAt, creds.UpdatedAt)

	if err != nil {
		return fmt.Errorf("saving credentials: %w", err)
	}
	return nil
}

// Get retrieves credentials by ID.
func (s *credentialsStore) Get(ctx context.Context, id string) (*domain.Credentials, error) {
	row := s.store.db.QueryRowContext(ctx, `
		SELECT id, source_id, account_identifier, oauth, pat, created_at, updated_at
		FROM credentials WHERE id = ?
	`, id)

	return scanCredentials(row)
}

// GetBySourceID retrieves credentials for a specific source.
func (s *credentialsStore) GetBySourceID(ctx context.Context, sourceID string) (*domain.Credentials, error) {
	row := s.store.db.QueryRowContext(ctx, `
		SELECT id, source_id, account_identifier, oauth, pat, created_at, updated_at
		FROM credentials WHERE source_id = ?
	`, sourceID)

	creds, err := scanCredentials(row)
	if errors.Is(err, domain.ErrNotFound) {
		return nil, nil // No credentials for this source is valid
	}
	return creds, err
}

// Delete removes credentials by ID.
func (s *credentialsStore) Delete(ctx context.Context, id string) error {
	_, err := s.store.db.ExecContext(ctx, "DELETE FROM credentials WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting credentials: %w", err)
	}
	return nil
}

// scanCredentials scans a single credentials row.
func scanCredentials(row *sql.Row) (*domain.Credentials, error) {
	var creds domain.Credentials
	var oauthJSON, patJSON sql.NullString

	if err := row.Scan(&creds.ID, &creds.SourceID, &creds.AccountIdentifier,
		&oauthJSON, &patJSON, &creds.CreatedAt, &creds.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("scanning credentials: %w", err)
	}

	if oauthJSON.Valid && oauthJSON.String != jsonNull {
		var oauth domain.OAuthCredentials
		if err := json.Unmarshal([]byte(oauthJSON.String), &oauth); err != nil {
			return nil, fmt.Errorf("unmarshalling oauth credentials: %w", err)
		}
		creds.OAuth = &oauth
	}

	if patJSON.Valid && patJSON.String != jsonNull {
		var pat domain.PATCredentials
		if err := json.Unmarshal([]byte(patJSON.String), &pat); err != nil {
			return nil, fmt.Errorf("unmarshalling pat credentials: %w", err)
		}
		creds.PAT = &pat
	}

	return &creds, nil
}
