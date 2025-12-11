-- Migration 002 rollback: Remove content field from documents
-- SQLite doesn't support DROP COLUMN directly, so we recreate the table

-- Create new table without content column
CREATE TABLE documents_new (
    id TEXT PRIMARY KEY,
    source_id TEXT NOT NULL,
    uri TEXT NOT NULL,
    title TEXT NOT NULL,
    parent_id TEXT,
    metadata TEXT,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    FOREIGN KEY (source_id) REFERENCES sources(id) ON DELETE CASCADE,
    FOREIGN KEY (parent_id) REFERENCES documents(id) ON DELETE SET NULL
);

-- Copy data
INSERT INTO documents_new SELECT id, source_id, uri, title, parent_id, metadata, created_at, updated_at FROM documents;

-- Drop old table and rename
DROP TABLE documents;
ALTER TABLE documents_new RENAME TO documents;

-- Recreate indexes
CREATE INDEX IF NOT EXISTS idx_documents_source ON documents(source_id);
CREATE INDEX IF NOT EXISTS idx_documents_uri ON documents(uri);
CREATE INDEX IF NOT EXISTS idx_documents_parent ON documents(parent_id);

-- Remove migration record
DELETE FROM schema_migrations WHERE version = 2;
