-- Migration 004: Auth architecture refactor
-- Separates OAuth app credentials (AuthProvider) from user tokens (Credentials)
-- This enables multiple accounts per OAuth app

-- Auth providers table (domain.AuthProvider)
-- Stores reusable OAuth app or PAT provider configurations
CREATE TABLE IF NOT EXISTS auth_providers (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    provider_type TEXT NOT NULL,   -- 'local', 'google', 'github', 'slack', 'notion'
    auth_method TEXT NOT NULL,     -- 'none', 'oauth', 'pat'
    oauth TEXT,                    -- JSON: OAuthProviderConfig (client_id, client_secret, scopes)
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_auth_providers_provider ON auth_providers(provider_type);
CREATE INDEX IF NOT EXISTS idx_auth_providers_method ON auth_providers(auth_method);

-- Credentials table (domain.Credentials)
-- Stores user-specific tokens for each source
CREATE TABLE IF NOT EXISTS credentials (
    id TEXT PRIMARY KEY,
    source_id TEXT NOT NULL UNIQUE,
    account_identifier TEXT,       -- User email/username (e.g., "user@gmail.com")
    oauth TEXT,                    -- JSON: OAuthCredentials (access_token, refresh_token, expiry)
    pat TEXT,                      -- JSON: PATCredentials (token)
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    FOREIGN KEY (source_id) REFERENCES sources(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_credentials_source ON credentials(source_id);

-- Add new columns to sources table
-- SQLite doesn't support adding constraints with ALTER TABLE, so we add nullable columns
ALTER TABLE sources ADD COLUMN auth_provider_id TEXT REFERENCES auth_providers(id);
ALTER TABLE sources ADD COLUMN credentials_id TEXT REFERENCES credentials(id);
ALTER TABLE sources ADD COLUMN created_at DATETIME;
ALTER TABLE sources ADD COLUMN updated_at DATETIME;

-- Create index on new columns
CREATE INDEX IF NOT EXISTS idx_sources_auth_provider ON sources(auth_provider_id);

-- Note: The old authorization_id column is kept for backward compatibility
-- It will be removed in a future migration after all code is updated
-- Old authorizations table is also kept for now (will be dropped later)

-- Record this migration
INSERT INTO schema_migrations (version) VALUES (4);
