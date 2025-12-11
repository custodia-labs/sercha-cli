-- Migration 001: Rollback initial schema

DROP INDEX IF EXISTS idx_exclusions_uri;
DROP INDEX IF EXISTS idx_exclusions_source;
DROP TABLE IF EXISTS exclusions;

DROP INDEX IF EXISTS idx_chunks_position;
DROP INDEX IF EXISTS idx_chunks_document;
DROP TABLE IF EXISTS chunks;

DROP INDEX IF EXISTS idx_documents_parent;
DROP INDEX IF EXISTS idx_documents_uri;
DROP INDEX IF EXISTS idx_documents_source;
DROP TABLE IF EXISTS documents;

DROP TABLE IF EXISTS sync_states;

DROP INDEX IF EXISTS idx_sources_auth;
DROP INDEX IF EXISTS idx_sources_type;
DROP TABLE IF EXISTS sources;

DROP INDEX IF EXISTS idx_authorizations_provider;
DROP TABLE IF EXISTS authorizations;

DELETE FROM schema_migrations WHERE version = 1;
