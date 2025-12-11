package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestSource_Fields tests Source structure fields
func TestSource_Fields(t *testing.T) {
	source := Source{
		ID:              "source-123",
		Type:            "filesystem",
		Name:            "My Documents",
		Config:          map[string]string{"root_path": "/home/user/docs"},
		AuthorizationID: "auth-456",
	}

	assert.Equal(t, "source-123", source.ID)
	assert.Equal(t, "filesystem", source.Type)
	assert.Equal(t, "My Documents", source.Name)
	assert.Equal(t, "/home/user/docs", source.Config["root_path"])
	assert.Equal(t, "auth-456", source.AuthorizationID)
}

// TestSource_EmptyConfig tests Source with empty config
func TestSource_EmptyConfig(t *testing.T) {
	source := Source{
		ID:              "source-123",
		Type:            "simple",
		Name:            "Simple Source",
		Config:          map[string]string{},
		AuthorizationID: "auth-456",
	}

	assert.NotNil(t, source.Config)
	assert.Empty(t, source.Config)
}

// TestSource_NilConfig tests Source with nil config
func TestSource_NilConfig(t *testing.T) {
	source := Source{
		ID:              "source-123",
		Type:            "simple",
		Name:            "Simple Source",
		Config:          nil,
		AuthorizationID: "auth-456",
	}

	assert.Nil(t, source.Config)
}

// TestSource_MultipleConfigKeys tests Source with multiple config values
func TestSource_MultipleConfigKeys(t *testing.T) {
	source := Source{
		ID:   "source-123",
		Type: "github",
		Name: "My GitHub Repos",
		Config: map[string]string{
			"repository":     "owner/repo",
			"branch":         "main",
			"include_issues": "true",
			"include_prs":    "false",
		},
		AuthorizationID: "auth-456",
	}

	assert.Len(t, source.Config, 4)
	assert.Equal(t, "owner/repo", source.Config["repository"])
	assert.Equal(t, "main", source.Config["branch"])
	assert.Equal(t, "true", source.Config["include_issues"])
	assert.Equal(t, "false", source.Config["include_prs"])
}

// TestSource_FilesystemExample tests filesystem source configuration
func TestSource_FilesystemExample(t *testing.T) {
	source := Source{
		ID:   "fs-source-1",
		Type: "filesystem",
		Name: "Local Documents",
		Config: map[string]string{
			"root_path":      "/home/user/documents",
			"include_hidden": "false",
			"file_patterns":  "*.txt,*.pdf,*.md",
		},
		AuthorizationID: "local-auth",
	}

	assert.Equal(t, "filesystem", source.Type)
	assert.Equal(t, "/home/user/documents", source.Config["root_path"])
	assert.Contains(t, source.Config, "include_hidden")
	assert.Contains(t, source.Config, "file_patterns")
}

// TestSource_GoogleDriveExample tests Google Drive source configuration
func TestSource_GoogleDriveExample(t *testing.T) {
	source := Source{
		ID:   "drive-source-1",
		Type: "google-drive",
		Name: "My Google Drive",
		Config: map[string]string{
			"folder_id":     "abc123xyz",
			"shared_drives": "true",
			"file_types":    "document,spreadsheet,presentation",
		},
		AuthorizationID: "google-auth-1",
	}

	assert.Equal(t, "google-drive", source.Type)
	assert.Equal(t, "abc123xyz", source.Config["folder_id"])
	assert.Equal(t, "google-auth-1", source.AuthorizationID)
}

// TestSource_EmptyStrings tests Source with empty string values
func TestSource_EmptyStrings(t *testing.T) {
	source := Source{
		ID:              "",
		Type:            "",
		Name:            "",
		Config:          map[string]string{},
		AuthorizationID: "",
	}

	assert.Empty(t, source.ID)
	assert.Empty(t, source.Type)
	assert.Empty(t, source.Name)
	assert.Empty(t, source.AuthorizationID)
}

// TestSyncState_Fields tests SyncState structure fields
func TestSyncState_Fields(t *testing.T) {
	lastSync := time.Now()
	syncState := SyncState{
		SourceID: "source-123",
		Cursor:   "opaque-cursor-token",
		LastSync: lastSync,
	}

	assert.Equal(t, "source-123", syncState.SourceID)
	assert.Equal(t, "opaque-cursor-token", syncState.Cursor)
	assert.Equal(t, lastSync, syncState.LastSync)
}

// TestSyncState_EmptyCursor tests SyncState with empty cursor
func TestSyncState_EmptyCursor(t *testing.T) {
	syncState := SyncState{
		SourceID: "source-123",
		Cursor:   "",
		LastSync: time.Now(),
	}

	assert.Empty(t, syncState.Cursor)
}

// TestSyncState_ZeroTime tests SyncState with zero time (never synced)
func TestSyncState_ZeroTime(t *testing.T) {
	syncState := SyncState{
		SourceID: "source-123",
		Cursor:   "",
		LastSync: time.Time{},
	}

	assert.True(t, syncState.LastSync.IsZero())
}

// TestSyncState_RecentSync tests SyncState with recent sync
func TestSyncState_RecentSync(t *testing.T) {
	recentTime := time.Now().Add(-5 * time.Minute)
	syncState := SyncState{
		SourceID: "source-123",
		Cursor:   "cursor-123",
		LastSync: recentTime,
	}

	assert.True(t, time.Since(syncState.LastSync) < 10*time.Minute)
}

// TestSyncState_OldSync tests SyncState with old sync
func TestSyncState_OldSync(t *testing.T) {
	oldTime := time.Now().Add(-24 * time.Hour)
	syncState := SyncState{
		SourceID: "source-123",
		Cursor:   "cursor-123",
		LastSync: oldTime,
	}

	assert.True(t, time.Since(syncState.LastSync) > 1*time.Hour)
}

// TestSyncState_LongCursor tests SyncState with long cursor string
func TestSyncState_LongCursor(t *testing.T) {
	longCursor := string(make([]byte, 1000))
	syncState := SyncState{
		SourceID: "source-123",
		Cursor:   longCursor,
		LastSync: time.Now(),
	}

	assert.Len(t, syncState.Cursor, 1000)
}

// TestSource_SpecialCharacters tests Source with special characters in config
func TestSource_SpecialCharacters(t *testing.T) {
	source := Source{
		ID:   "source-123",
		Type: "custom",
		Name: "Source with Special Chars: @#$%",
		Config: map[string]string{
			"url":     "https://example.com?query=test&foo=bar",
			"pattern": "*.{txt,md}",
			"exclude": "[cache]|[tmp]",
		},
		AuthorizationID: "auth-456",
	}

	assert.Contains(t, source.Name, "@#$%")
	assert.Contains(t, source.Config["url"], "?")
	assert.Contains(t, source.Config["pattern"], "{")
	assert.Contains(t, source.Config["exclude"], "|")
}

// TestSource_UnicodeInName tests Source with Unicode characters
func TestSource_UnicodeInName(t *testing.T) {
	source := Source{
		ID:              "source-123",
		Type:            "filesystem",
		Name:            "文档目录",
		Config:          map[string]string{"root_path": "/docs"},
		AuthorizationID: "auth-456",
	}

	assert.Equal(t, "文档目录", source.Name)
}

// TestSyncState_MultipleSources tests different sync states for different sources
func TestSyncState_MultipleSources(t *testing.T) {
	states := []SyncState{
		{
			SourceID: "source-1",
			Cursor:   "cursor-1",
			LastSync: time.Now().Add(-1 * time.Hour),
		},
		{
			SourceID: "source-2",
			Cursor:   "cursor-2",
			LastSync: time.Now().Add(-2 * time.Hour),
		},
		{
			SourceID: "source-3",
			Cursor:   "",
			LastSync: time.Time{}, // Never synced
		},
	}

	assert.Len(t, states, 3)
	assert.NotEqual(t, states[0].SourceID, states[1].SourceID)
	assert.True(t, states[2].LastSync.IsZero())
}

// TestSource_RequiredFields tests what fields are typically required
func TestSource_RequiredFields(t *testing.T) {
	// Minimal valid source
	source := Source{
		ID:              "source-123",
		Type:            "filesystem",
		Name:            "Test Source",
		AuthorizationID: "auth-456",
	}

	assert.NotEmpty(t, source.ID)
	assert.NotEmpty(t, source.Type)
	assert.NotEmpty(t, source.Name)
	assert.NotEmpty(t, source.AuthorizationID)
}

// TestSource_ConfigStringValues tests that Config only stores strings
func TestSource_ConfigStringValues(t *testing.T) {
	source := Source{
		ID:   "source-123",
		Type: "custom",
		Name: "Test",
		Config: map[string]string{
			"string_val": "text",
			"bool_val":   "true", // Stored as string
			"int_val":    "42",   // Stored as string
			"float_val":  "3.14", // Stored as string
		},
		AuthorizationID: "auth-456",
	}

	// All values should be strings
	for _, v := range source.Config {
		assert.IsType(t, "", v)
	}
}

// TestSyncState_CursorFormats tests various cursor formats
func TestSyncState_CursorFormats(t *testing.T) {
	tests := []struct {
		name   string
		cursor string
	}{
		{"simple token", "abc123"},
		{"base64", "YWJjMTIzNDU2Nzg5"},
		{"json", `{"offset": 100, "timestamp": "2024-01-01T00:00:00Z"}`},
		{"url encoded", "page%3D10%26limit%3D20"},
		{"empty", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := SyncState{
				SourceID: "source-123",
				Cursor:   tt.cursor,
				LastSync: time.Now(),
			}
			assert.Equal(t, tt.cursor, state.Cursor)
		})
	}
}
