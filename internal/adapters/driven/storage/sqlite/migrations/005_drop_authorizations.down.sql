-- Migration 005 down: Restore authorization system
-- This recreates the authorizations table and authorization_id column

-- Recreate authorizations table
CREATE TABLE IF NOT EXISTS authorizations (
    id TEXT PRIMARY KEY,
    provider_type TEXT NOT NULL,
    auth_method TEXT NOT NULL,
    data TEXT NOT NULL,
    account_identifier TEXT,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_authorizations_provider ON authorizations(provider_type);
CREATE INDEX IF NOT EXISTS idx_authorizations_method ON authorizations(auth_method);

-- Backup current sources
CREATE TABLE sources_backup AS SELECT * FROM sources;

-- Drop current sources table
DROP TABLE IF EXISTS sources;

-- Recreate sources with authorization_id column
CREATE TABLE sources (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,
    name TEXT NOT NULL,
    config TEXT NOT NULL,
    authorization_id TEXT NOT NULL DEFAULT '',
    auth_provider_id TEXT REFERENCES auth_providers(id),
    credentials_id TEXT REFERENCES credentials(id),
    created_at DATETIME,
    updated_at DATETIME,
    FOREIGN KEY (authorization_id) REFERENCES authorizations(id)
);

-- Restore data (authorization_id will be empty string)
INSERT INTO sources (id, type, name, config, authorization_id, auth_provider_id, credentials_id, created_at, updated_at)
SELECT id, type, name, config, '', auth_provider_id, credentials_id, created_at, updated_at FROM sources_backup;

-- Drop backup
DROP TABLE sources_backup;

-- Recreate indices
CREATE INDEX IF NOT EXISTS idx_sources_type ON sources(type);
CREATE INDEX IF NOT EXISTS idx_sources_auth ON sources(authorization_id);
CREATE INDEX IF NOT EXISTS idx_sources_auth_provider ON sources(auth_provider_id);

-- Remove migration record
DELETE FROM schema_migrations WHERE version = 5;
