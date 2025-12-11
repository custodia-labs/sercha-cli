-- Migration 005: Drop deprecated authorization system
-- Removes authorization_id column from sources and drops authorizations table
-- Sources now use auth_provider_id and credentials_id (added in migration 004)

-- SQLite doesn't support DROP COLUMN directly, so we need to recreate the table
-- First backup the data
CREATE TABLE sources_backup AS SELECT id, type, name, config, auth_provider_id, credentials_id, created_at, updated_at FROM sources;

-- Drop the old sources table
DROP TABLE IF EXISTS sources;

-- Recreate sources without authorization_id and without FK to authorizations
CREATE TABLE sources (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,
    name TEXT NOT NULL,
    config TEXT NOT NULL,
    auth_provider_id TEXT REFERENCES auth_providers(id),
    credentials_id TEXT REFERENCES credentials(id),
    created_at DATETIME,
    updated_at DATETIME
);

-- Restore the data
INSERT INTO sources (id, type, name, config, auth_provider_id, credentials_id, created_at, updated_at)
SELECT id, type, name, config, auth_provider_id, credentials_id, created_at, updated_at FROM sources_backup;

-- Drop backup
DROP TABLE sources_backup;

-- Recreate indices
CREATE INDEX IF NOT EXISTS idx_sources_type ON sources(type);
CREATE INDEX IF NOT EXISTS idx_sources_auth_provider ON sources(auth_provider_id);

-- Now drop the old authorizations table (no longer needed)
DROP TABLE IF EXISTS authorizations;

-- Record this migration
INSERT INTO schema_migrations (version) VALUES (5);
