package domain

import (
	"fmt"
	"strings"
	"time"
)

// Source represents a configured data source.
// Each source produces documents via a connector and belongs to a specific user account.
type Source struct {
	// ID is the unique identifier for the source.
	ID string

	// Type identifies the connector type (e.g., "filesystem", "gmail").
	Type string

	// Name is the human-readable name for this source.
	Name string

	// Config contains connector-specific configuration.
	Config map[string]string

	// AuthorizationID references the Authorization used by this source.
	// Deprecated: Use AuthProviderID and CredentialsID instead.
	// Kept for backward compatibility during migration.
	AuthorizationID string

	// AuthProviderID references the AuthProvider (OAuth app or PAT provider config).
	// Empty string for no-auth connectors (filesystem).
	AuthProviderID string

	// CredentialsID references this source's Credentials (tokens + account info).
	// Empty string for no-auth connectors.
	CredentialsID string

	// CreatedAt is when the source was created.
	CreatedAt time.Time

	// UpdatedAt is when the source was last updated.
	UpdatedAt time.Time
}

// DisplayName returns the source name with account identifier if provided.
// Used for display in CLI/TUI where the account context helps identify the source.
// If the account identifier is already present in the name, it is not appended again.
func (s *Source) DisplayName(accountIdentifier string) string {
	if accountIdentifier != "" && !strings.Contains(s.Name, accountIdentifier) {
		return fmt.Sprintf("%s - %s", s.Name, accountIdentifier)
	}
	return s.Name
}

// SyncState tracks the synchronisation progress for a source.
type SyncState struct {
	// SourceID links to the Source being synced.
	SourceID string

	// Cursor is an opaque token for incremental sync.
	Cursor string

	// LastSync is when the last successful sync completed.
	LastSync time.Time
}
