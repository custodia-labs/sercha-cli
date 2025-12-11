-- Migration 001: Initial schema
-- This schema is derived from domain models in internal/core/domain/

-- Schema version tracking
CREATE TABLE IF NOT EXISTS schema_migrations (
    version INTEGER PRIMARY KEY,
    applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Authorizations table (domain.Authorization)
-- Stores OAuth/PAT credentials for connectors
CREATE TABLE IF NOT EXISTS authorizations (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    provider_type TEXT NOT NULL,  -- 'local', 'google', 'github', 'slack', 'notion'
    auth_method TEXT NOT NULL,    -- 'none', 'oauth', 'pat'
    oauth_app TEXT,               -- JSON: OAuthAppConfig (client_id, client_secret, etc.)
    pat TEXT,                     -- JSON: PATConfig (token)
    tokens TEXT,                  -- JSON: OAuthToken (access_token, refresh_token, etc.)
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_authorizations_provider ON authorizations(provider_type);

-- Sources table (domain.Source)
-- Stores configured data sources
CREATE TABLE IF NOT EXISTS sources (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,           -- Connector type: 'filesystem', 'gmail', etc.
    name TEXT NOT NULL,           -- Human-readable name
    config TEXT NOT NULL,         -- JSON: map[string]string connector config
    authorization_id TEXT NOT NULL,
    FOREIGN KEY (authorization_id) REFERENCES authorizations(id)
);

CREATE INDEX IF NOT EXISTS idx_sources_type ON sources(type);
CREATE INDEX IF NOT EXISTS idx_sources_auth ON sources(authorization_id);

-- Sync state table (domain.SyncState)
-- Tracks synchronization progress for each source
CREATE TABLE IF NOT EXISTS sync_states (
    source_id TEXT PRIMARY KEY,
    cursor TEXT,                  -- Opaque token for incremental sync
    last_sync DATETIME,           -- When last successful sync completed
    FOREIGN KEY (source_id) REFERENCES sources(id) ON DELETE CASCADE
);

-- Documents table (domain.Document)
-- Stores indexed document metadata
CREATE TABLE IF NOT EXISTS documents (
    id TEXT PRIMARY KEY,
    source_id TEXT NOT NULL,
    uri TEXT NOT NULL,            -- Original location (file path, URL, etc.)
    title TEXT NOT NULL,
    parent_id TEXT,               -- For hierarchical sources
    metadata TEXT,                -- JSON: map[string]any arbitrary metadata
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    FOREIGN KEY (source_id) REFERENCES sources(id) ON DELETE CASCADE,
    FOREIGN KEY (parent_id) REFERENCES documents(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_documents_source ON documents(source_id);
CREATE INDEX IF NOT EXISTS idx_documents_uri ON documents(uri);
CREATE INDEX IF NOT EXISTS idx_documents_parent ON documents(parent_id);

-- Chunks table (domain.Chunk)
-- Stores searchable units within documents
CREATE TABLE IF NOT EXISTS chunks (
    id TEXT PRIMARY KEY,
    document_id TEXT NOT NULL,
    content TEXT NOT NULL,        -- Text content of the chunk
    position INTEGER NOT NULL,    -- Ordinal position within document
    embedding BLOB,               -- Vector representation ([]float32 serialized)
    metadata TEXT,                -- JSON: map[string]any chunk-specific metadata
    FOREIGN KEY (document_id) REFERENCES documents(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_chunks_document ON chunks(document_id);
CREATE INDEX IF NOT EXISTS idx_chunks_position ON chunks(document_id, position);

-- Exclusions table (domain.Exclusion)
-- Documents excluded from syncing
CREATE TABLE IF NOT EXISTS exclusions (
    id TEXT PRIMARY KEY,
    source_id TEXT NOT NULL,
    document_id TEXT NOT NULL,
    uri TEXT NOT NULL,            -- For matching on re-sync
    reason TEXT,                  -- Optional explanation
    excluded_at DATETIME NOT NULL,
    FOREIGN KEY (source_id) REFERENCES sources(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_exclusions_source ON exclusions(source_id);
CREATE INDEX IF NOT EXISTS idx_exclusions_uri ON exclusions(source_id, uri);

-- Record this migration
INSERT INTO schema_migrations (version) VALUES (1);
