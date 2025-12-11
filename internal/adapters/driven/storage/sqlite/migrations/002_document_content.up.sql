-- Migration 002: Add content field to documents
-- Stores the full text content of the document for display purposes

ALTER TABLE documents ADD COLUMN content TEXT DEFAULT '';

-- Record this migration
INSERT INTO schema_migrations (version) VALUES (2);
