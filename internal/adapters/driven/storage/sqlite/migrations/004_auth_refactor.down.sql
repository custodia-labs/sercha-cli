-- Migration 004 rollback: Revert auth architecture refactor
-- This restores the original schema with authorization_id only

-- SQLite doesn't support DROP COLUMN in older versions, so we recreate the table
-- First, create a backup of sources
CREATE TABLE sources_backup AS SELECT id, type, name, config, authorization_id FROM sources;

-- Drop the modified sources table
DROP TABLE IF EXISTS sources;

-- Recreate sources without the new columns
CREATE TABLE sources (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,
    name TEXT NOT NULL,
    config TEXT NOT NULL,
    authorization_id TEXT NOT NULL,
    FOREIGN KEY (authorization_id) REFERENCES authorizations(id)
);

-- Restore data
INSERT INTO sources (id, type, name, config, authorization_id)
SELECT id, type, name, config, authorization_id FROM sources_backup;

-- Drop backup
DROP TABLE sources_backup;

-- Recreate indexes
CREATE INDEX IF NOT EXISTS idx_sources_type ON sources(type);
CREATE INDEX IF NOT EXISTS idx_sources_auth ON sources(authorization_id);

-- Drop new tables
DROP TABLE IF EXISTS credentials;
DROP TABLE IF EXISTS auth_providers;

-- Remove migration record
DELETE FROM schema_migrations WHERE version = 4;
